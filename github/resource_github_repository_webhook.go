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
	_ resource.Resource                = &githubRepositoryWebhookResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryWebhookResource{}
	_ resource.ResourceWithImportState = &githubRepositoryWebhookResource{}
)

type githubRepositoryWebhookResource struct {
	client *Owner
}

type githubRepositoryWebhookResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Repository    types.String `tfsdk:"repository"`
	Events        types.Set    `tfsdk:"events"`
	Configuration types.List   `tfsdk:"configuration"`
	URL           types.String `tfsdk:"url"`
	Active        types.Bool   `tfsdk:"active"`
	ETag          types.String `tfsdk:"etag"`
}

type githubRepositoryWebhookConfigurationModel struct {
	URL         types.String `tfsdk:"url"`
	ContentType types.String `tfsdk:"content_type"`
	Secret      types.String `tfsdk:"secret"`
	InsecureSSL types.Bool   `tfsdk:"insecure_ssl"`
}

func NewGithubRepositoryWebhookResource() resource.Resource {
	return &githubRepositoryWebhookResource{}
}

func (r *githubRepositoryWebhookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_webhook"
}

func (r *githubRepositoryWebhookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages repository webhooks within GitHub organizations or personal accounts",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the webhook.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The repository of the webhook.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"events": schema.SetAttribute{
				Description: "A list of events which should trigger the webhook",
				Required:    true,
				ElementType: types.StringType,
			},
			"configuration": schema.ListAttribute{
				Description: "Configuration for the webhook.",
				Optional:    true,
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
				Description: "URL of the webhook",
				Computed:    true,
			},
			"active": schema.BoolAttribute{
				Description: "Indicate if the webhook should receive events. Defaults to 'true'.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"etag": schema.StringAttribute{
				Description: "An etag representing the webhook object.",
				Computed:    true,
			},
		},
	}
}

func (r *githubRepositoryWebhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubRepositoryWebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubRepositoryWebhookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hook, diags := r.buildGithubHookFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := plan.Repository.ValueString()
	client := r.client.V3Client()

	createdHook, _, err := client.Repositories.CreateHook(ctx, owner, repoName, hook)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating repository webhook",
			"Could not create repository webhook, unexpected error: "+err.Error(),
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

func (r *githubRepositoryWebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubRepositoryWebhookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := state.Repository.ValueString()
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

	hook, resp_github, err := client.Repositories.GetHook(requestCtx, owner, repoName, hookID)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing repository webhook %s from state because it no longer exists in GitHub", state.ID.ValueString())
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error reading repository webhook",
			"Could not read repository webhook: "+err.Error(),
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

func (r *githubRepositoryWebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubRepositoryWebhookResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hook, diags := r.buildGithubHookFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := plan.Repository.ValueString()
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

	_, _, err = client.Repositories.EditHook(requestCtx, owner, repoName, hookID, hook)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating repository webhook",
			"Could not update repository webhook: "+err.Error(),
		)
		return
	}

	// Re-read the updated webhook
	updatedHook, _, err := client.Repositories.GetHook(requestCtx, owner, repoName, hookID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading updated repository webhook",
			"Could not read updated repository webhook: "+err.Error(),
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

func (r *githubRepositoryWebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubRepositoryWebhookResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := state.Repository.ValueString()
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

	_, err = client.Repositories.DeleteHook(requestCtx, owner, repoName, hookID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting repository webhook",
			"Could not delete repository webhook: "+err.Error(),
		)
		return
	}
}

func (r *githubRepositoryWebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format repository/webhook_id",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// Helper functions

func (r *githubRepositoryWebhookResource) buildGithubHookFromPlan(ctx context.Context, plan githubRepositoryWebhookResourceModel) (*github.Hook, diag.Diagnostics) {
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

		hookConfig := &github.HookConfig{}
		if !config.URL.IsNull() {
			hookConfig.URL = github.Ptr(config.URL.ValueString())
		}
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

func (r *githubRepositoryWebhookResource) updateStateFromGithubHook(ctx context.Context, state *githubRepositoryWebhookResourceModel, hook *github.Hook) diag.Diagnostics {
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

func (r *githubRepositoryWebhookResource) extractConfigurationFromState(ctx context.Context, configList types.List) (*githubRepositoryWebhookConfigurationModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var config githubRepositoryWebhookConfigurationModel

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

// Custom validator for max items in list
type maxItemsValidator struct {
	max int
}

func (v *maxItemsValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("list must contain at most %d items", v.max)
}

func (v *maxItemsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *maxItemsValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if len(req.ConfigValue.Elements()) > v.max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Too Many List Items",
			fmt.Sprintf("This attribute supports a maximum of %d items, got %d items.",
				v.max, len(req.ConfigValue.Elements())),
		)
		return
	}
}
