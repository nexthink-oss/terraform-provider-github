package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ resource.Resource                = &githubAppInstallationRepositoryResource{}
	_ resource.ResourceWithConfigure   = &githubAppInstallationRepositoryResource{}
	_ resource.ResourceWithImportState = &githubAppInstallationRepositoryResource{}
)

type githubAppInstallationRepositoryResource struct {
	client *Owner
}

type githubAppInstallationRepositoryResourceModel struct {
	ID             types.String `tfsdk:"id"`
	InstallationID types.String `tfsdk:"installation_id"`
	Repository     types.String `tfsdk:"repository"`
	RepoID         types.Int64  `tfsdk:"repo_id"`
}

func NewGithubAppInstallationRepositoryResource() resource.Resource {
	return &githubAppInstallationRepositoryResource{}
}

func (r *githubAppInstallationRepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_installation_repository"
}

func (r *githubAppInstallationRepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the associations between app installations and repositories.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the app installation repository association (installation_id:repository).",
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
			"repository": schema.StringAttribute{
				Description: "The repository to install the app on.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"repo_id": schema.Int64Attribute{
				Description: "The GitHub repository ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubAppInstallationRepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubAppInstallationRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubAppInstallationRepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Resource Not Available for User Accounts",
			"This resource can only be used in the context of an organization.",
		)
		return
	}

	installationIDString := data.InstallationID.ValueString()
	installationID, err := strconv.ParseInt(installationIDString, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Installation ID",
			fmt.Sprintf("Installation ID must be a valid integer, got: %s", installationIDString),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	ctx = context.WithValue(ctx, CtxId, data.Repository.ValueString())
	repoName := data.Repository.ValueString()

	repo, _, err := client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get Repository",
			fmt.Sprintf("An unexpected error occurred when getting repository %s: %s", repoName, err.Error()),
		)
		return
	}
	repoID := repo.GetID()

	_, _, err = client.Apps.AddRepository(ctx, installationID, repoID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Add Repository to Installation",
			fmt.Sprintf("An unexpected error occurred when adding repository %s to installation %s: %s", repoName, installationIDString, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(r.buildTwoPartID(installationIDString, repoName))

	// Read the created resource to populate computed fields
	r.readGithubAppInstallationRepository(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubAppInstallationRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubAppInstallationRepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubAppInstallationRepository(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubAppInstallationRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This resource doesn't support updates since all attributes require replacement
	resp.Diagnostics.AddError(
		"Resource Update Not Supported",
		"The github_app_installation_repository resource does not support updates. "+
			"All attributes require resource replacement.",
	)
}

func (r *githubAppInstallationRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubAppInstallationRepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Resource Not Available for User Accounts",
			"This resource can only be used in the context of an organization.",
		)
		return
	}

	installationIDString := data.InstallationID.ValueString()
	installationID, err := strconv.ParseInt(installationIDString, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Installation ID",
			fmt.Sprintf("Installation ID must be a valid integer, got: %s", installationIDString),
		)
		return
	}

	client := r.client.V3Client()
	ctx = context.WithValue(ctx, CtxId, data.ID.ValueString())

	repoID := data.RepoID.ValueInt64()

	_, err = client.Apps.RemoveRepository(ctx, installationID, repoID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Remove Repository from Installation",
			fmt.Sprintf("An unexpected error occurred when removing repository from installation %s: %s", installationIDString, err.Error()),
		)
		return
	}
}

func (r *githubAppInstallationRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	installationIDString, repoName, err := r.parseTwoPartID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: installation_id:repository. Got: %q", req.ID),
		)
		return
	}

	// Verify we can parse the installation ID
	_, err = strconv.ParseInt(installationIDString, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Installation ID",
			fmt.Sprintf("Installation ID must be a valid integer, got: %s", installationIDString),
		)
		return
	}

	data := &githubAppInstallationRepositoryResourceModel{
		ID:             types.StringValue(req.ID),
		InstallationID: types.StringValue(installationIDString),
		Repository:     types.StringValue(repoName),
	}

	// Read the current state to populate computed fields
	r.readGithubAppInstallationRepository(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "imported GitHub app installation repository", map[string]interface{}{
		"id":              data.ID.ValueString(),
		"installation_id": installationIDString,
		"repository":      repoName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubAppInstallationRepositoryResource) buildTwoPartID(a, b string) string {
	return fmt.Sprintf("%s:%s", a, b)
}

func (r *githubAppInstallationRepositoryResource) parseTwoPartID(id string) (string, string, error) {
	// Split on the first colon to handle repository names that might contain colons
	colonIndex := -1
	for i, char := range id {
		if char == ':' {
			colonIndex = i
			break
		}
	}

	if colonIndex == -1 {
		return "", "", fmt.Errorf("unexpected ID format (%q); expected installation_id:repository", id)
	}

	parts := []string{id[:colonIndex], id[colonIndex+1:]}

	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected ID format (%q); expected installation_id:repository", id)
	}
	return parts[0], parts[1], nil
}

func (r *githubAppInstallationRepositoryResource) readGithubAppInstallationRepository(ctx context.Context, data *githubAppInstallationRepositoryResourceModel, diags *diag.Diagnostics) {
	if !r.client.IsOrganization {
		diags.AddError(
			"Resource Not Available for User Accounts",
			"This resource can only be used in the context of an organization.",
		)
		return
	}

	client := r.client.V3Client()
	installationIDString := data.InstallationID.ValueString()
	repoName := data.Repository.ValueString()

	installationID, err := strconv.ParseInt(installationIDString, 10, 64)
	if err != nil {
		diags.AddError(
			"Invalid Installation ID",
			fmt.Sprintf("Installation ID must be a valid integer, got: %s", installationIDString),
		)
		return
	}

	ctx = context.WithValue(ctx, CtxId, data.ID.ValueString())
	opt := &github.ListOptions{PerPage: 100} // maxPerPage from util.go

	for {
		repos, resp, err := client.Apps.ListUserRepos(ctx, installationID, opt)
		if err != nil {
			diags.AddError(
				"Unable to List App Installation Repositories",
				fmt.Sprintf("An unexpected error occurred when listing repositories for installation %s: %s", installationIDString, err.Error()),
			)
			return
		}

		for _, r := range repos.Repositories {
			if r.GetName() == repoName {
				data.InstallationID = types.StringValue(installationIDString)
				data.Repository = types.StringValue(repoName)
				data.RepoID = types.Int64Value(r.GetID())

				tflog.Debug(ctx, "successfully read GitHub app installation repository", map[string]interface{}{
					"id":              data.ID.ValueString(),
					"installation_id": installationIDString,
					"repository":      repoName,
					"repo_id":         r.GetID(),
				})
				return
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	tflog.Info(ctx, "removing app installation repository association from state because it no longer exists in GitHub", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
	data.ID = types.StringNull()
}
