package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                 = &githubOrganizationWebhookResource{}
	_ resource.ResourceWithConfigure    = &githubOrganizationWebhookResource{}
	_ resource.ResourceWithImportState  = &githubOrganizationWebhookResource{}
	_ resource.ResourceWithUpgradeState = &githubOrganizationWebhookResource{}
)

type githubOrganizationWebhookResource struct {
	client *Owner
}

type githubOrganizationWebhookResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Events        types.Set    `tfsdk:"events"`
	Configuration types.List   `tfsdk:"configuration"`
	URL           types.String `tfsdk:"url"`
	Active        types.Bool   `tfsdk:"active"`
	ETag          types.String `tfsdk:"etag"`
}

type githubOrganizationWebhookConfigurationModel struct {
	URL         types.String `tfsdk:"url"`
	ContentType types.String `tfsdk:"content_type"`
	Secret      types.String `tfsdk:"secret"`
	InsecureSSL types.Bool   `tfsdk:"insecure_ssl"`
}

func NewGithubOrganizationWebhookResource() resource.Resource {
	return &githubOrganizationWebhookResource{}
}

func (r *githubOrganizationWebhookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_webhook"
}

func (r *githubOrganizationWebhookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages webhooks for GitHub organizations",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the webhook.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"events": schema.SetAttribute{
				Description: "A list of events which should trigger the webhook.",
				Required:    true,
				ElementType: types.StringType,
			},
			"configuration": schema.ListAttribute{
				Description: "Configuration for the webhook.",
				Optional:    true,
				Sensitive:   true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"url":          types.StringType,
						"content_type": types.StringType,
						"secret":       types.StringType,
						"insecure_ssl": types.BoolType,
					},
				},
				Validators: []validator.List{
					&maxItemsValidator{max: 1},
				},
			},
			"url": schema.StringAttribute{
				Description: "URL of the webhook.",
				Computed:    true,
			},
			"active": schema.BoolAttribute{
				Description: "Indicate if the webhook should receive events.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"etag": schema.StringAttribute{
				Description: "An etag representing the webhook object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubOrganizationWebhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *githubOrganizationWebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubOrganizationWebhookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	hook, diags := r.buildGithubHookFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgName := r.client.Name()
	client := r.client.V3Client()

	createdHook, _, err := client.Organizations.CreateHook(ctx, orgName, hook)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating organization webhook",
			"Could not create organization webhook, unexpected error: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(strconv.FormatInt(createdHook.GetID(), 10))

	// GitHub returns the secret as a string of 8 asterisks "********"
	// We would prefer to store the real secret in state, so we'll
	// write the configuration secret in state from our request to GitHub
	if createdHook.Config.Secret != nil {
		createdHook.Config.Secret = hook.Config.Secret
	}

	// Update state with created webhook data
	diags = r.updateStateFromGithubHook(ctx, &plan, createdHook)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubOrganizationWebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubOrganizationWebhookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	orgName := r.client.Name()
	client := r.client.V3Client()
	hookID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing webhook ID",
			"Could not parse webhook ID: "+err.Error(),
		)
		return
	}

	// Use etag for conditional requests if available
	requestCtx := context.WithValue(ctx, CtxId, state.ID.ValueString())
	if !state.ETag.IsNull() && !state.ETag.IsUnknown() {
		requestCtx = context.WithValue(requestCtx, CtxEtag, state.ETag.ValueString())
	}

	hook, resp_github, err := client.Organizations.GetHook(requestCtx, orgName, hookID)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing organization webhook %s from state because it no longer exists in GitHub", state.ID.ValueString())
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error reading organization webhook",
			"Could not read organization webhook: "+err.Error(),
		)
		return
	}

	// GitHub returns the secret as a string of 8 asterisks "********"
	// We would prefer to store the real secret in state, so we'll
	// write the configuration secret in state from what we get from
	// ResourceData
	if !state.Configuration.IsNull() && len(state.Configuration.Elements()) > 0 {
		currentConfig, diags := r.extractConfigurationFromState(ctx, state.Configuration)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if hook.Config.Secret != nil && !currentConfig.Secret.IsNull() {
			hook.Config.Secret = github.Ptr(currentConfig.Secret.ValueString())
		}
	}

	// Update ETag from response
	if resp_github != nil {
		state.ETag = types.StringValue(resp_github.Header.Get("ETag"))
	}

	// Update state with webhook data
	diags := r.updateStateFromGithubHook(ctx, &state, hook)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubOrganizationWebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubOrganizationWebhookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	hook, diags := r.buildGithubHookFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgName := r.client.Name()
	client := r.client.V3Client()
	hookID, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing webhook ID",
			"Could not parse webhook ID: "+err.Error(),
		)
		return
	}

	requestCtx := context.WithValue(ctx, CtxId, plan.ID.ValueString())

	_, _, err = client.Organizations.EditHook(requestCtx, orgName, hookID, hook)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating organization webhook",
			"Could not update organization webhook: "+err.Error(),
		)
		return
	}

	// Re-read the updated webhook
	updatedHook, _, err := client.Organizations.GetHook(requestCtx, orgName, hookID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading updated organization webhook",
			"Could not read updated organization webhook: "+err.Error(),
		)
		return
	}

	// GitHub returns the secret as a string of 8 asterisks "********"
	// We would prefer to store the real secret in state, so we'll
	// write the configuration secret in state from our request to GitHub
	if updatedHook.Config.Secret != nil {
		updatedHook.Config.Secret = hook.Config.Secret
	}

	// Update state with updated webhook data
	diags = r.updateStateFromGithubHook(ctx, &plan, updatedHook)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubOrganizationWebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubOrganizationWebhookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	orgName := r.client.Name()
	client := r.client.V3Client()
	hookID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing webhook ID",
			"Could not parse webhook ID: "+err.Error(),
		)
		return
	}

	requestCtx := context.WithValue(ctx, CtxId, state.ID.ValueString())

	_, err = client.Organizations.DeleteHook(requestCtx, orgName, hookID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting organization webhook",
			"Could not delete organization webhook: "+err.Error(),
		)
		return
	}
}

func (r *githubOrganizationWebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Organization webhooks can be imported using just the webhook ID
	// since they're always associated with the configured organization
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *githubOrganizationWebhookResource) SchemaVersion(ctx context.Context) int64 {
	return 1
}

func (r *githubOrganizationWebhookResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema.Schema{
				Description: "Creates and manages webhooks for GitHub organizations",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "The ID of the webhook.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"events": schema.SetAttribute{
						Description: "A list of events which should trigger the webhook.",
						Required:    true,
						ElementType: types.StringType,
					},
					"configuration": schema.MapAttribute{
						Description: "Configuration for the webhook.",
						Optional:    true,
						Sensitive:   true,
						ElementType: types.StringType,
					},
					"url": schema.StringAttribute{
						Description: "URL of the webhook.",
						Computed:    true,
					},
					"active": schema.BoolAttribute{
						Description: "Indicate if the webhook should receive events.",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
					"etag": schema.StringAttribute{
						Description: "An etag representing the webhook object.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			StateUpgrader: r.upgradeStateV0toV1,
		},
	}
}

func (r *githubOrganizationWebhookResource) upgradeStateV0toV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	log.Printf("[DEBUG] Migrating github_organization_webhook from schema version 0 to 1")

	// Define the v0 state model
	type v0StateModel struct {
		ID            types.String `tfsdk:"id"`
		Events        types.Set    `tfsdk:"events"`
		Configuration types.Map    `tfsdk:"configuration"`
		URL           types.String `tfsdk:"url"`
		Active        types.Bool   `tfsdk:"active"`
		ETag          types.String `tfsdk:"etag"`
	}

	var priorStateData v0StateModel

	// Get the prior state
	resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the upgraded state
	upgradedStateData := githubOrganizationWebhookResourceModel{
		ID:     priorStateData.ID,
		Events: priorStateData.Events,
		URL:    priorStateData.URL,
		Active: priorStateData.Active,
		ETag:   priorStateData.ETag,
	}

	// Handle configuration migration from map to list
	if !priorStateData.Configuration.IsNull() && !priorStateData.Configuration.IsUnknown() {
		var configMap map[string]string
		resp.Diagnostics.Append(priorStateData.Configuration.ElementsAs(ctx, &configMap, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Convert map to list structure
		configAttrs := map[string]attr.Value{
			"url":          types.StringNull(),
			"content_type": types.StringNull(),
			"secret":       types.StringNull(),
			"insecure_ssl": types.BoolNull(),
		}

		if url, ok := configMap["url"]; ok && url != "" {
			configAttrs["url"] = types.StringValue(url)
		}
		if contentType, ok := configMap["content_type"]; ok && contentType != "" {
			configAttrs["content_type"] = types.StringValue(contentType)
		}
		if secret, ok := configMap["secret"]; ok && secret != "" {
			configAttrs["secret"] = types.StringValue(secret)
		}
		if insecureSSL, ok := configMap["insecure_ssl"]; ok {
			if insecureSSL == "1" || strings.ToLower(insecureSSL) == "true" {
				configAttrs["insecure_ssl"] = types.BoolValue(true)
			} else {
				configAttrs["insecure_ssl"] = types.BoolValue(false)
			}
		}

		configObj, diags := types.ObjectValue(map[string]attr.Type{
			"url":          types.StringType,
			"content_type": types.StringType,
			"secret":       types.StringType,
			"insecure_ssl": types.BoolType,
		}, configAttrs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		configList, diags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"url":          types.StringType,
				"content_type": types.StringType,
				"secret":       types.StringType,
				"insecure_ssl": types.BoolType,
			},
		}, []attr.Value{configObj})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		upgradedStateData.Configuration = configList
	} else {
		upgradedStateData.Configuration = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"url":          types.StringType,
				"content_type": types.StringType,
				"secret":       types.StringType,
				"insecure_ssl": types.BoolType,
			},
		})
	}

	// Set the upgraded state
	resp.Diagnostics.Append(resp.State.Set(ctx, upgradedStateData)...)

	log.Printf("[DEBUG] Completed migration of github_organization_webhook from schema version 0 to 1")
}

// Helper functions

func (r *githubOrganizationWebhookResource) buildGithubHookFromPlan(ctx context.Context, plan githubOrganizationWebhookResourceModel) (*github.Hook, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Build events list
	var events []string
	diags.Append(plan.Events.ElementsAs(ctx, &events, false)...)
	if diags.HasError() {
		return nil, diags
	}

	active := plan.Active.ValueBool()
	hook := &github.Hook{
		Events: events,
		Active: &active,
	}

	// Build configuration if provided
	if !plan.Configuration.IsNull() && len(plan.Configuration.Elements()) > 0 {
		config, configDiags := r.extractConfigurationFromState(ctx, plan.Configuration)
		diags.Append(configDiags...)
		if diags.HasError() {
			return nil, diags
		}

		// Validate required URL field
		if config.URL.IsNull() || config.URL.ValueString() == "" {
			diags.AddError(
				"Missing required configuration",
				"The 'url' field is required when configuration is specified.",
			)
			return nil, diags
		}

		hookConfig := &github.HookConfig{}
		hookConfig.URL = github.Ptr(config.URL.ValueString())

		if !config.ContentType.IsNull() {
			hookConfig.ContentType = github.Ptr(config.ContentType.ValueString())
		}
		if !config.Secret.IsNull() {
			hookConfig.Secret = github.Ptr(config.Secret.ValueString())
		}
		if !config.InsecureSSL.IsNull() {
			if config.InsecureSSL.ValueBool() {
				hookConfig.InsecureSSL = github.Ptr("1")
			} else {
				hookConfig.InsecureSSL = github.Ptr("0")
			}
		}

		hook.Config = hookConfig
	}

	return hook, diags
}

func (r *githubOrganizationWebhookResource) updateStateFromGithubHook(ctx context.Context, state *githubOrganizationWebhookResourceModel, hook *github.Hook) diag.Diagnostics {
	var diags diag.Diagnostics

	if hook.GetURL() != "" {
		state.URL = types.StringValue(hook.GetURL())
	}

	state.Active = types.BoolValue(hook.GetActive())

	// Convert events to Set
	eventElements := make([]attr.Value, len(hook.Events))
	for i, event := range hook.Events {
		eventElements[i] = types.StringValue(event)
	}
	eventsSet, setDiags := types.SetValue(types.StringType, eventElements)
	diags.Append(setDiags...)
	if diags.HasError() {
		return diags
	}
	state.Events = eventsSet

	// Build configuration object
	if hook.Config != nil {
		configAttrs := map[string]attr.Value{
			"url":          types.StringNull(),
			"content_type": types.StringNull(),
			"secret":       types.StringNull(),
			"insecure_ssl": types.BoolNull(),
		}

		if hook.Config.URL != nil {
			configAttrs["url"] = types.StringValue(*hook.Config.URL)
		}
		if hook.Config.ContentType != nil {
			configAttrs["content_type"] = types.StringValue(*hook.Config.ContentType)
		}
		if hook.Config.Secret != nil {
			configAttrs["secret"] = types.StringValue(*hook.Config.Secret)
		}
		if hook.Config.InsecureSSL != nil {
			configAttrs["insecure_ssl"] = types.BoolValue(*hook.Config.InsecureSSL == "1")
		}

		configObj, objDiags := types.ObjectValue(map[string]attr.Type{
			"url":          types.StringType,
			"content_type": types.StringType,
			"secret":       types.StringType,
			"insecure_ssl": types.BoolType,
		}, configAttrs)
		diags.Append(objDiags...)
		if diags.HasError() {
			return diags
		}

		configList, listDiags := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"url":          types.StringType,
				"content_type": types.StringType,
				"secret":       types.StringType,
				"insecure_ssl": types.BoolType,
			},
		}, []attr.Value{configObj})
		diags.Append(listDiags...)
		if diags.HasError() {
			return diags
		}

		state.Configuration = configList
	}

	return diags
}

func (r *githubOrganizationWebhookResource) extractConfigurationFromState(ctx context.Context, configList types.List) (*githubOrganizationWebhookConfigurationModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var config githubOrganizationWebhookConfigurationModel

	if configList.IsNull() || len(configList.Elements()) == 0 {
		return &config, diags
	}

	elements := configList.Elements()
	if len(elements) == 0 {
		return &config, diags
	}

	// Get the first (and only) configuration object
	configObj := elements[0].(types.Object)
	configAttrs := configObj.Attributes()

	if url, ok := configAttrs["url"]; ok {
		config.URL = url.(types.String)
	}
	if contentType, ok := configAttrs["content_type"]; ok {
		config.ContentType = contentType.(types.String)
	}
	if secret, ok := configAttrs["secret"]; ok {
		config.Secret = secret.(types.String)
	}
	if insecureSSL, ok := configAttrs["insecure_ssl"]; ok {
		config.InsecureSSL = insecureSSL.(types.Bool)
	}

	return &config, diags
}
