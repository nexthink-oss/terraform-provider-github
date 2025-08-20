package framework

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &githubActionsOrganizationSecretRepositoriesResource{}
	_ resource.ResourceWithConfigure   = &githubActionsOrganizationSecretRepositoriesResource{}
	_ resource.ResourceWithImportState = &githubActionsOrganizationSecretRepositoriesResource{}
)

type githubActionsOrganizationSecretRepositoriesResource struct {
	client *githubpkg.Owner
}

type githubActionsOrganizationSecretRepositoriesResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	SecretName            types.String `tfsdk:"secret_name"`
	SelectedRepositoryIDs types.Set    `tfsdk:"selected_repository_ids"`
}

func NewGithubActionsOrganizationSecretRepositoriesResource() resource.Resource {
	return &githubActionsOrganizationSecretRepositoriesResource{}
}

func (r *githubActionsOrganizationSecretRepositoriesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_organization_secret_repositories"
}

func (r *githubActionsOrganizationSecretRepositoriesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages repository allow list for an Action Secret within a GitHub organization",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the actions organization secret repositories (same as secret_name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_name": schema.StringAttribute{
				Description: "Name of the existing secret.",
				Required:    true,
				Validators: []validator.String{
					&secretNameValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"selected_repository_ids": schema.SetAttribute{
				Description: "An array of repository ids that can access the organization secret.",
				Required:    true,
				ElementType: types.Int64Type,
			},
		},
	}
}

func (r *githubActionsOrganizationSecretRepositoriesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*githubpkg.Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *githubActionsOrganizationSecretRepositoriesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubActionsOrganizationSecretRepositoriesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate this is an organization
	err := r.checkOrganization()
	if err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts: %s", err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	secretName := data.SecretName.ValueString()

	// Convert selected repository IDs
	var selectedRepositoryIDs []int64
	for _, elem := range data.SelectedRepositoryIDs.Elements() {
		if intVal, ok := elem.(types.Int64); ok && !intVal.IsNull() && !intVal.IsUnknown() {
			selectedRepositoryIDs = append(selectedRepositoryIDs, intVal.ValueInt64())
		}
	}

	_, err = client.Actions.SetSelectedReposForOrgSecret(ctx, owner, secretName, selectedRepositoryIDs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Set Selected Repositories for Organization Secret",
			fmt.Sprintf("An unexpected error occurred when setting selected repositories for the organization secret: %s", err.Error()),
		)
		return
	}

	// Set the ID
	data.ID = types.StringValue(secretName)

	tflog.Debug(ctx, "set selected repositories for GitHub actions organization secret", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"owner":       owner,
		"secret_name": secretName,
		"repo_count":  len(selectedRepositoryIDs),
	})

	// Read the updated resource to populate any computed fields
	r.readGithubActionsOrganizationSecretRepositories(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsOrganizationSecretRepositoriesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubActionsOrganizationSecretRepositoriesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubActionsOrganizationSecretRepositories(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsOrganizationSecretRepositoriesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubActionsOrganizationSecretRepositoriesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate this is an organization
	err := r.checkOrganization()
	if err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts: %s", err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	secretName := data.SecretName.ValueString()

	// Convert selected repository IDs
	var selectedRepositoryIDs []int64
	for _, elem := range data.SelectedRepositoryIDs.Elements() {
		if intVal, ok := elem.(types.Int64); ok && !intVal.IsNull() && !intVal.IsUnknown() {
			selectedRepositoryIDs = append(selectedRepositoryIDs, intVal.ValueInt64())
		}
	}

	_, err = client.Actions.SetSelectedReposForOrgSecret(ctx, owner, secretName, selectedRepositoryIDs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Selected Repositories for Organization Secret",
			fmt.Sprintf("An unexpected error occurred when updating selected repositories for the organization secret: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "updated selected repositories for GitHub actions organization secret", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"owner":       owner,
		"secret_name": secretName,
		"repo_count":  len(selectedRepositoryIDs),
	})

	// Read the updated resource to populate any computed fields
	r.readGithubActionsOrganizationSecretRepositories(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsOrganizationSecretRepositoriesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubActionsOrganizationSecretRepositoriesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate this is an organization
	err := r.checkOrganization()
	if err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts: %s", err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	secretName := data.ID.ValueString()

	// Clear selected repositories by setting an empty array
	selectedRepositoryIDs := []int64{}
	_, err = client.Actions.SetSelectedReposForOrgSecret(ctx, owner, secretName, selectedRepositoryIDs)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Clear Selected Repositories for Organization Secret",
			fmt.Sprintf("An unexpected error occurred when clearing selected repositories for the organization secret: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "cleared selected repositories for GitHub actions organization secret", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"owner":       owner,
		"secret_name": secretName,
	})
}

func (r *githubActionsOrganizationSecretRepositoriesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	secretName := req.ID

	// Validate this is an organization
	err := r.checkOrganization()
	if err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts: %s", err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	// Verify the secret exists by trying to list its repositories
	_, _, err = client.Actions.ListSelectedReposForOrgSecret(ctx, owner, secretName, &github.ListOptions{PerPage: 1})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import Actions Organization Secret Repositories",
			fmt.Sprintf("Unable to read actions organization secret repositories for import: %s", err.Error()),
		)
		return
	}

	data := &githubActionsOrganizationSecretRepositoriesResourceModel{
		ID:         types.StringValue(secretName),
		SecretName: types.StringValue(secretName),
	}

	// Read the selected repositories
	r.readGithubActionsOrganizationSecretRepositories(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "imported GitHub actions organization secret repositories", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"owner":       owner,
		"secret_name": secretName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubActionsOrganizationSecretRepositoriesResource) checkOrganization() error {
	if !r.client.IsOrganization {
		return fmt.Errorf("this resource can only be used with organization accounts")
	}
	return nil
}

func (r *githubActionsOrganizationSecretRepositoriesResource) readGithubActionsOrganizationSecretRepositories(ctx context.Context, data *githubActionsOrganizationSecretRepositoriesResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()
	secretName := data.ID.ValueString()

	selectedRepositoryIDs := []int64{}
	opt := &github.ListOptions{
		PerPage: 30,
	}

	for {
		results, githubResp, err := client.Actions.ListSelectedReposForOrgSecret(ctx, owner, secretName, opt)
		if err != nil {
			if ghErr, ok := err.(*github.ErrorResponse); ok {
				if ghErr.Response.StatusCode == http.StatusNotFound {
					tflog.Info(ctx, "removing actions organization secret repositories from state because it no longer exists in GitHub", map[string]interface{}{
						"owner":       owner,
						"secret_name": secretName,
					})
					data.ID = types.StringNull()
					return
				}
			}
			diags.AddError(
				"Unable to Read Selected Repositories for Organization Secret",
				fmt.Sprintf("An unexpected error occurred when reading selected repositories for the organization secret: %s", err.Error()),
			)
			return
		}

		for _, repo := range results.Repositories {
			selectedRepositoryIDs = append(selectedRepositoryIDs, repo.GetID())
		}

		if githubResp.NextPage == 0 {
			break
		}
		opt.Page = githubResp.NextPage
	}

	data.SecretName = types.StringValue(secretName)

	selectedRepositoryIDAttrs := []attr.Value{}
	for _, id := range selectedRepositoryIDs {
		selectedRepositoryIDAttrs = append(selectedRepositoryIDAttrs, types.Int64Value(id))
	}
	data.SelectedRepositoryIDs = types.SetValueMust(types.Int64Type, selectedRepositoryIDAttrs)

	tflog.Debug(ctx, "successfully read GitHub actions organization secret repositories", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"owner":       owner,
		"secret_name": secretName,
		"repo_count":  len(selectedRepositoryIDs),
	})
}
