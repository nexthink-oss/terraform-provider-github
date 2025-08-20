package framework

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ resource.Resource                = &githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource{}
	_ resource.ResourceWithConfigure   = &githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource{}
	_ resource.ResourceWithImportState = &githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource{}
)

type githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource struct {
	client *githubpkg.Owner
}

type githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Repository       types.String `tfsdk:"repository"`
	UseDefault       types.Bool   `tfsdk:"use_default"`
	IncludeClaimKeys types.List   `tfsdk:"include_claim_keys"`
}

func NewGithubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource() resource.Resource {
	return &githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource{}
}

func (r *githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_repository_oidc_subject_claim_customization_template"
}

func (r *githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages an OpenID Connect subject claim customization template for a repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this resource.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"use_default": schema.BoolAttribute{
				Description: "Whether to use the default template or not. If 'true', 'include_claim_keys' must not be set.",
				Required:    true,
			},
			"include_claim_keys": schema.ListAttribute{
				Description: "A list of OpenID Connect claims.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*githubpkg.Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *githubpkg.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repository := data.Repository.ValueString()
	owner := r.client.Name()
	useDefault := data.UseDefault.ValueBool()

	tflog.Debug(ctx, "Creating GitHub Actions repository OIDC subject claim customization template", map[string]interface{}{
		"owner":       owner,
		"repository":  repository,
		"use_default": useDefault,
	})

	// Validate use_default and include_claim_keys combination
	if useDefault && !data.IncludeClaimKeys.IsNull() && !data.IncludeClaimKeys.IsUnknown() {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"include_claim_keys cannot be set when use_default is true",
		)
		return
	}

	customOIDCSubjectClaimTemplate := &github.OIDCSubjectClaimCustomTemplate{
		UseDefault: &useDefault,
	}

	if !data.IncludeClaimKeys.IsNull() && !data.IncludeClaimKeys.IsUnknown() {
		var includeClaimKeys []string
		resp.Diagnostics.Append(data.IncludeClaimKeys.ElementsAs(ctx, &includeClaimKeys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		customOIDCSubjectClaimTemplate.IncludeClaimKeys = includeClaimKeys
	}

	_, err := r.client.V3Client().Actions.SetRepoOIDCSubjectClaimCustomTemplate(ctx, owner, repository, customOIDCSubjectClaimTemplate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create GitHub Actions Repository OIDC Subject Claim Customization Template",
			fmt.Sprintf("Error creating OIDC subject claim customization template for repository %s/%s: %s", owner, repository, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(repository)

	tflog.Debug(ctx, "Successfully created GitHub Actions repository OIDC subject claim customization template", map[string]interface{}{
		"owner":      owner,
		"repository": repository,
	})

	// Save data into Terraform state and call Read to populate computed attributes
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call Read to populate the full state
	readReq := resource.ReadRequest{State: resp.State}
	readResp := resource.ReadResponse{State: resp.State}
	r.Read(ctx, readReq, &readResp)

	resp.Diagnostics.Append(readResp.Diagnostics...)
	resp.State = readResp.State
}

func (r *githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repository := data.ID.ValueString()
	owner := r.client.Name()

	tflog.Debug(ctx, "Reading GitHub Actions repository OIDC subject claim customization template", map[string]interface{}{
		"owner":      owner,
		"repository": repository,
	})

	template, response, err := r.client.V3Client().Actions.GetRepoOIDCSubjectClaimCustomTemplate(ctx, owner, repository)
	if err != nil {
		if response != nil && response.StatusCode == http.StatusNotFound {
			tflog.Debug(ctx, "GitHub Actions repository OIDC subject claim customization template not found, removing from state", map[string]interface{}{
				"owner":      owner,
				"repository": repository,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Actions Repository OIDC Subject Claim Customization Template",
			fmt.Sprintf("Error reading OIDC subject claim customization template for repository %s/%s: %s", owner, repository, err.Error()),
		)
		return
	}

	// Set the attributes
	data.Repository = types.StringValue(repository)
	data.UseDefault = types.BoolValue(*template.UseDefault)

	// Convert include_claim_keys to framework list
	if template.IncludeClaimKeys != nil && len(template.IncludeClaimKeys) > 0 {
		includeClaimKeysList, diags := types.ListValueFrom(ctx, types.StringType, template.IncludeClaimKeys)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.IncludeClaimKeys = includeClaimKeysList
	} else {
		data.IncludeClaimKeys = types.ListNull(types.StringType)
	}

	tflog.Debug(ctx, "Successfully read GitHub Actions repository OIDC subject claim customization template", map[string]interface{}{
		"owner":              owner,
		"repository":         repository,
		"use_default":        template.UseDefault,
		"include_claim_keys": template.IncludeClaimKeys,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repository := data.Repository.ValueString()
	owner := r.client.Name()
	useDefault := data.UseDefault.ValueBool()

	tflog.Debug(ctx, "Updating GitHub Actions repository OIDC subject claim customization template", map[string]interface{}{
		"owner":       owner,
		"repository":  repository,
		"use_default": useDefault,
	})

	// Validate use_default and include_claim_keys combination
	if useDefault && !data.IncludeClaimKeys.IsNull() && !data.IncludeClaimKeys.IsUnknown() {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"include_claim_keys cannot be set when use_default is true",
		)
		return
	}

	customOIDCSubjectClaimTemplate := &github.OIDCSubjectClaimCustomTemplate{
		UseDefault: &useDefault,
	}

	if !data.IncludeClaimKeys.IsNull() && !data.IncludeClaimKeys.IsUnknown() {
		var includeClaimKeys []string
		resp.Diagnostics.Append(data.IncludeClaimKeys.ElementsAs(ctx, &includeClaimKeys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		customOIDCSubjectClaimTemplate.IncludeClaimKeys = includeClaimKeys
	}

	_, err := r.client.V3Client().Actions.SetRepoOIDCSubjectClaimCustomTemplate(ctx, owner, repository, customOIDCSubjectClaimTemplate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update GitHub Actions Repository OIDC Subject Claim Customization Template",
			fmt.Sprintf("Error updating OIDC subject claim customization template for repository %s/%s: %s", owner, repository, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Successfully updated GitHub Actions repository OIDC subject claim customization template", map[string]interface{}{
		"owner":      owner,
		"repository": repository,
	})

	// Save data into Terraform state and call Read to populate computed attributes
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call Read to populate the full state
	readReq := resource.ReadRequest{State: resp.State}
	readResp := resource.ReadResponse{State: resp.State}
	r.Read(ctx, readReq, &readResp)

	resp.Diagnostics.Append(readResp.Diagnostics...)
	resp.State = readResp.State
}

func (r *githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repository := data.Repository.ValueString()
	owner := r.client.Name()

	tflog.Debug(ctx, "Deleting GitHub Actions repository OIDC subject claim customization template", map[string]interface{}{
		"owner":      owner,
		"repository": repository,
	})

	// Reset the repository to use the default claims
	// https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect#using-the-default-subject-claims
	customOIDCSubjectClaimTemplate := &github.OIDCSubjectClaimCustomTemplate{
		UseDefault: github.Ptr(true),
	}

	_, err := r.client.V3Client().Actions.SetRepoOIDCSubjectClaimCustomTemplate(ctx, owner, repository, customOIDCSubjectClaimTemplate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete GitHub Actions Repository OIDC Subject Claim Customization Template",
			fmt.Sprintf("Error resetting OIDC subject claim customization template for repository %s/%s to default: %s", owner, repository, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Successfully deleted GitHub Actions repository OIDC subject claim customization template", map[string]interface{}{
		"owner":      owner,
		"repository": repository,
	})
}

func (r *githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
