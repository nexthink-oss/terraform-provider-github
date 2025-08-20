package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource{}
	_ resource.ResourceWithConfigure   = &githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource{}
	_ resource.ResourceWithImportState = &githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource{}
)

type githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource struct {
	client *Owner
}

type githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResourceModel struct {
	ID               types.String `tfsdk:"id"`
	IncludeClaimKeys types.List   `tfsdk:"include_claim_keys"`
}

func NewGithubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource() resource.Resource {
	return &githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource{}
}

func (r *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_organization_oidc_subject_claim_customization_template"
}

func (r *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages an OpenID Connect subject claim customization template for an organization",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"include_claim_keys": schema.ListAttribute{
				Description: "A list of OpenID Connect claims.",
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if we're working with an organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization",
		)
		return
	}

	client := r.client.V3Client()
	orgName := r.client.Name()

	// Convert the terraform list to a slice of strings
	var includeClaimKeys []string
	resp.Diagnostics.Append(data.IncludeClaimKeys.ElementsAs(ctx, &includeClaimKeys, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create/update the OIDC subject claim customization template
	_, err := client.Actions.SetOrgOIDCSubjectClaimCustomTemplate(ctx, orgName, &github.OIDCSubjectClaimCustomTemplate{
		IncludeClaimKeys: includeClaimKeys,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Organization OIDC Subject Claim Customization Template",
			fmt.Sprintf("An unexpected error occurred when creating the organization OIDC subject claim customization template: %s", err.Error()),
		)
		return
	}

	// Set the ID
	data.ID = types.StringValue(orgName)

	tflog.Debug(ctx, "created GitHub actions organization OIDC subject claim customization template", map[string]interface{}{
		"id":                 data.ID.ValueString(),
		"owner":              orgName,
		"include_claim_keys": includeClaimKeys,
	})

	// Read the created resource to populate computed fields
	r.readGithubActionsOrganizationOIDCSubjectClaimCustomizationTemplate(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubActionsOrganizationOIDCSubjectClaimCustomizationTemplate(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if we're working with an organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization",
		)
		return
	}

	client := r.client.V3Client()
	orgName := r.client.Name()

	// Convert the terraform list to a slice of strings
	var includeClaimKeys []string
	resp.Diagnostics.Append(data.IncludeClaimKeys.ElementsAs(ctx, &includeClaimKeys, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the OIDC subject claim customization template
	_, err := client.Actions.SetOrgOIDCSubjectClaimCustomTemplate(ctx, orgName, &github.OIDCSubjectClaimCustomTemplate{
		IncludeClaimKeys: includeClaimKeys,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Organization OIDC Subject Claim Customization Template",
			fmt.Sprintf("An unexpected error occurred when updating the organization OIDC subject claim customization template: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "updated GitHub actions organization OIDC subject claim customization template", map[string]interface{}{
		"id":                 data.ID.ValueString(),
		"owner":              orgName,
		"include_claim_keys": includeClaimKeys,
	})

	// Read the updated resource to populate computed fields
	r.readGithubActionsOrganizationOIDCSubjectClaimCustomizationTemplate(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if we're working with an organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization",
		)
		return
	}

	client := r.client.V3Client()
	orgName := r.client.Name()

	// Sets include_claim_keys back to GitHub's defaults
	// https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect#resetting-your-customizations
	_, err := client.Actions.SetOrgOIDCSubjectClaimCustomTemplate(ctx, orgName, &github.OIDCSubjectClaimCustomTemplate{
		IncludeClaimKeys: []string{"repo", "context"},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Organization OIDC Subject Claim Customization Template",
			fmt.Sprintf("An unexpected error occurred when deleting the organization OIDC subject claim customization template: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub actions organization OIDC subject claim customization template", map[string]interface{}{
		"id":    data.ID.ValueString(),
		"owner": orgName,
	})
}

func (r *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by organization name - import ID is passed through as the resource ID
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper function to read the resource and populate the model
func (r *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource) readGithubActionsOrganizationOIDCSubjectClaimCustomizationTemplate(ctx context.Context, data *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	orgName := data.ID.ValueString()

	template, _, err := client.Actions.GetOrgOIDCSubjectClaimCustomTemplate(ctx, orgName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing organization OIDC subject claim customization template from state because it no longer exists in GitHub", map[string]interface{}{
					"owner": orgName,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Organization OIDC Subject Claim Customization Template",
			fmt.Sprintf("An unexpected error occurred when reading the organization OIDC subject claim customization template: %s", err.Error()),
		)
		return
	}

	// Convert the slice of strings to a terraform list
	includeClaimKeysAttrs := make([]attr.Value, len(template.IncludeClaimKeys))
	for i, key := range template.IncludeClaimKeys {
		includeClaimKeysAttrs[i] = types.StringValue(key)
	}

	includeClaimKeysList, listDiags := types.ListValue(types.StringType, includeClaimKeysAttrs)
	diags.Append(listDiags...)
	if diags.HasError() {
		return
	}

	data.IncludeClaimKeys = includeClaimKeysList

	tflog.Debug(ctx, "successfully read GitHub actions organization OIDC subject claim customization template", map[string]interface{}{
		"id":                 data.ID.ValueString(),
		"owner":              orgName,
		"include_claim_keys": template.IncludeClaimKeys,
	})
}
