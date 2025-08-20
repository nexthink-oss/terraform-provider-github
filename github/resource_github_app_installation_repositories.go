package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ resource.Resource                = &githubAppInstallationRepositoriesResource{}
	_ resource.ResourceWithConfigure   = &githubAppInstallationRepositoriesResource{}
	_ resource.ResourceWithImportState = &githubAppInstallationRepositoriesResource{}
)

type githubAppInstallationRepositoriesResource struct {
	client *Owner
}

type githubAppInstallationRepositoriesResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	InstallationID       types.String `tfsdk:"installation_id"`
	SelectedRepositories types.Set    `tfsdk:"selected_repositories"`
}

func NewGithubAppInstallationRepositoriesResource() resource.Resource {
	return &githubAppInstallationRepositoriesResource{}
}

func (r *githubAppInstallationRepositoriesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_installation_repositories"
}

func (r *githubAppInstallationRepositoriesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the associations between app installations and repositories.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the app installation repositories resource (installation_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"installation_id": schema.StringAttribute{
				Description: "The GitHub app installation id.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"selected_repositories": schema.SetAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "A list of repository names to install the app on.",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubAppInstallationRepositoriesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *githubAppInstallationRepositoriesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubAppInstallationRepositoriesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	installationIDString := data.InstallationID.ValueString()
	var selectedRepositoryNames []string
	resp.Diagnostics.Append(data.SelectedRepositories.ElementsAs(ctx, &selectedRepositoryNames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	ctx = context.WithValue(ctx, CtxId, installationIDString)

	currentReposNameIDs, instID, err := r.getAllAccessibleRepos(ctx, installationIDString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get Accessible Repositories",
			fmt.Sprintf("An unexpected error occurred when getting accessible repositories: %s", err.Error()),
		)
		return
	}

	// Add repos that are not in the current state on GitHub
	for _, repoName := range selectedRepositoryNames {
		if _, ok := currentReposNameIDs[repoName]; ok {
			// If it already exists, remove it from the map so we can delete all that are left at the end
			delete(currentReposNameIDs, repoName)
		} else {
			repo, _, err := client.Repositories.Get(ctx, owner, repoName)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Get Repository",
					fmt.Sprintf("An unexpected error occurred when getting repository %s: %s", repoName, err.Error()),
				)
				return
			}
			repoID := repo.GetID()
			tflog.Debug(ctx, "adding repository to app installation", map[string]interface{}{
				"repository_name": repoName,
				"repository_id":   repoID,
				"installation_id": instID,
			})
			_, _, err = client.Apps.AddRepository(ctx, instID, repoID)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Add Repository to Installation",
					fmt.Sprintf("An unexpected error occurred when adding repository %s to installation: %s", repoName, err.Error()),
				)
				return
			}
		}
	}

	// Remove repositories that existed on GitHub but not selectedRepositories
	// There is a github limitation that means we can't remove the last repository from an installation.
	// Therefore, we skip the first and delete the rest. The app will then need to be uninstalled via the GUI
	// as there is no current API endpoint for [un]installation. Ensure there is at least one repository remaining.
	if len(selectedRepositoryNames) >= 1 {
		for repoName, repoID := range currentReposNameIDs {
			tflog.Debug(ctx, "removing repository from app installation", map[string]interface{}{
				"repository_name": repoName,
				"repository_id":   repoID,
				"installation_id": instID,
			})
			_, err = client.Apps.RemoveRepository(ctx, instID, repoID)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Remove Repository from Installation",
					fmt.Sprintf("An unexpected error occurred when removing repository %s from installation: %s", repoName, err.Error()),
				)
				return
			}
		}
	}

	data.ID = types.StringValue(installationIDString)

	// Read the created resource to populate computed fields
	r.readGithubAppInstallationRepositories(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubAppInstallationRepositoriesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubAppInstallationRepositoriesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubAppInstallationRepositories(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubAppInstallationRepositoriesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubAppInstallationRepositoriesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	installationIDString := data.InstallationID.ValueString()
	var selectedRepositoryNames []string
	resp.Diagnostics.Append(data.SelectedRepositories.ElementsAs(ctx, &selectedRepositoryNames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	ctx = context.WithValue(ctx, CtxId, installationIDString)

	currentReposNameIDs, instID, err := r.getAllAccessibleRepos(ctx, installationIDString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get Accessible Repositories",
			fmt.Sprintf("An unexpected error occurred when getting accessible repositories: %s", err.Error()),
		)
		return
	}

	// Add repos that are not in the current state on GitHub
	for _, repoName := range selectedRepositoryNames {
		if _, ok := currentReposNameIDs[repoName]; ok {
			// If it already exists, remove it from the map so we can delete all that are left at the end
			delete(currentReposNameIDs, repoName)
		} else {
			repo, _, err := client.Repositories.Get(ctx, owner, repoName)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Get Repository",
					fmt.Sprintf("An unexpected error occurred when getting repository %s: %s", repoName, err.Error()),
				)
				return
			}
			repoID := repo.GetID()
			tflog.Debug(ctx, "adding repository to app installation", map[string]interface{}{
				"repository_name": repoName,
				"repository_id":   repoID,
				"installation_id": instID,
			})
			_, _, err = client.Apps.AddRepository(ctx, instID, repoID)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Add Repository to Installation",
					fmt.Sprintf("An unexpected error occurred when adding repository %s to installation: %s", repoName, err.Error()),
				)
				return
			}
		}
	}

	// Remove repositories that existed on GitHub but not selectedRepositories
	// There is a github limitation that means we can't remove the last repository from an installation.
	// Therefore, we skip the first and delete the rest. The app will then need to be uninstalled via the GUI
	// as there is no current API endpoint for [un]installation. Ensure there is at least one repository remaining.
	if len(selectedRepositoryNames) >= 1 {
		for repoName, repoID := range currentReposNameIDs {
			tflog.Debug(ctx, "removing repository from app installation", map[string]interface{}{
				"repository_name": repoName,
				"repository_id":   repoID,
				"installation_id": instID,
			})
			_, err = client.Apps.RemoveRepository(ctx, instID, repoID)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Remove Repository from Installation",
					fmt.Sprintf("An unexpected error occurred when removing repository %s from installation: %s", repoName, err.Error()),
				)
				return
			}
		}
	}

	// Read the updated resource to populate computed fields
	r.readGithubAppInstallationRepositories(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubAppInstallationRepositoriesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubAppInstallationRepositoriesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	installationIDString := data.InstallationID.ValueString()

	reposNameIDs, instID, err := r.getAllAccessibleRepos(ctx, installationIDString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get Accessible Repositories",
			fmt.Sprintf("An unexpected error occurred when getting accessible repositories: %s", err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	ctx = context.WithValue(ctx, CtxId, installationIDString)

	// There is a github limitation that means we can't remove the last repository from an installation.
	// Therefore, we skip the first and delete the rest. The app will then need to be uninstalled via the GUI
	// as there is no current API endpoint for [un]installation.
	first := true
	for repoName, repoID := range reposNameIDs {
		if first {
			first = false
			tflog.Warn(ctx, "cannot remove last repository from app installation", map[string]interface{}{
				"repository_name": repoName,
				"repository_id":   repoID,
				"installation_id": instID,
				"message":         "Cannot remove last repository from app installation due to API limitations. Manually uninstall the app to remove.",
			})
			continue
		} else {
			_, err = client.Apps.RemoveRepository(ctx, instID, repoID)
			tflog.Debug(ctx, "removing repository from app installation", map[string]interface{}{
				"repository_name": repoName,
				"repository_id":   repoID,
				"installation_id": instID,
			})
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Remove Repository from Installation",
					fmt.Sprintf("An unexpected error occurred when removing repository %s from installation: %s", repoName, err.Error()),
				)
				return
			}
		}
	}
}

func (r *githubAppInstallationRepositoriesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	installationIDString := req.ID

	// Verify we can parse the installation ID
	_, err := strconv.ParseInt(installationIDString, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Installation ID must be a numeric string. Got: %q", installationIDString),
		)
		return
	}

	// Verify the installation exists by trying to get its repositories
	_, _, err = r.getAllAccessibleRepos(ctx, installationIDString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import App Installation Repositories",
			fmt.Sprintf("Unable to read app installation repositories for import: %s", err.Error()),
		)
		return
	}

	data := &githubAppInstallationRepositoriesResourceModel{
		ID:             types.StringValue(installationIDString),
		InstallationID: types.StringValue(installationIDString),
	}

	// Read the current state to populate the selected repositories
	r.readGithubAppInstallationRepositories(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "imported GitHub app installation repositories", map[string]interface{}{
		"id":              data.ID.ValueString(),
		"installation_id": installationIDString,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubAppInstallationRepositoriesResource) getAllAccessibleRepos(ctx context.Context, idString string) (map[string]int64, int64, error) {
	if !r.client.IsOrganization {
		return nil, 0, fmt.Errorf("this resource can only be used in the context of an organization, %q is a user", r.client.Name())
	}

	installationID, err := strconv.ParseInt(idString, 10, 64)
	if err != nil {
		return nil, 0, fmt.Errorf("unexpected ID format (%q), expected numerical ID: %s", idString, err.Error())
	}

	ctx = context.WithValue(ctx, CtxId, idString)
	opt := &github.ListOptions{PerPage: 100} // maxPerPage from util.go
	client := r.client.V3Client()

	allRepos := make(map[string]int64)

	for {
		repos, resp, err := client.Apps.ListUserRepos(ctx, installationID, opt)
		if err != nil {
			return nil, 0, err
		}
		for _, repo := range repos.Repositories {
			allRepos[repo.GetName()] = repo.GetID()
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allRepos, installationID, nil
}

func (r *githubAppInstallationRepositoriesResource) readGithubAppInstallationRepositories(ctx context.Context, data *githubAppInstallationRepositoriesResourceModel, diags *diag.Diagnostics) {
	installationIDString := data.ID.ValueString()

	reposNameIDs, _, err := r.getAllAccessibleRepos(ctx, installationIDString)
	if err != nil {
		diags.AddError(
			"Unable to Read App Installation Repositories",
			fmt.Sprintf("An unexpected error occurred when reading the app installation repositories: %s", err.Error()),
		)
		return
	}

	repoNames := []string{}
	for name := range reposNameIDs {
		repoNames = append(repoNames, name)
	}

	if len(reposNameIDs) > 0 {
		data.InstallationID = types.StringValue(installationIDString)

		repoSet, diag := types.SetValueFrom(ctx, types.StringType, repoNames)
		diags.Append(diag...)
		if diags.HasError() {
			return
		}
		data.SelectedRepositories = repoSet

		tflog.Debug(ctx, "successfully read GitHub app installation repositories", map[string]interface{}{
			"id":                    data.ID.ValueString(),
			"installation_id":       installationIDString,
			"selected_repositories": len(repoNames),
		})
		return
	}

	tflog.Info(ctx, "removing app installation repository association from state because it no longer exists in GitHub", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
	data.ID = types.StringNull()
}
