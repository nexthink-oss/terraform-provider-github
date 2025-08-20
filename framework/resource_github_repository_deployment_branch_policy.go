package framework

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ resource.Resource                = &githubRepositoryDeploymentBranchPolicyResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryDeploymentBranchPolicyResource{}
	_ resource.ResourceWithImportState = &githubRepositoryDeploymentBranchPolicyResource{}
)

func NewGithubRepositoryDeploymentBranchPolicyResource() resource.Resource {
	return &githubRepositoryDeploymentBranchPolicyResource{}
}

type githubRepositoryDeploymentBranchPolicyResource struct {
	client *githubpkg.Owner
}

type githubRepositoryDeploymentBranchPolicyResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Repository      types.String `tfsdk:"repository"`
	EnvironmentName types.String `tfsdk:"environment_name"`
	Name            types.String `tfsdk:"name"`
	Etag            types.String `tfsdk:"etag"`
}

func (r *githubRepositoryDeploymentBranchPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_deployment_branch_policy"
}

func (r *githubRepositoryDeploymentBranchPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages deployment branch policies for GitHub repository environments",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the deployment branch policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The GitHub repository name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment_name": schema.StringAttribute{
				Description: "The target environment name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the branch",
				Required:    true,
			},
			"etag": schema.StringAttribute{
				Description: "An etag representing the branch policy object.",
				Computed:    true,
			},
		},
	}
}

func (r *githubRepositoryDeploymentBranchPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubRepositoryDeploymentBranchPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubRepositoryDeploymentBranchPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repository := data.Repository.ValueString()
	environmentName := data.EnvironmentName.ValueString()
	name := data.Name.ValueString()

	policy, _, err := client.Repositories.CreateDeploymentBranchPolicy(ctx, owner, repository, environmentName, &github.DeploymentBranchPolicyRequest{
		Name: &name,
		Type: github.Ptr("branch"),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating deployment branch policy",
			fmt.Sprintf("Could not create deployment branch policy %s for environment %s in repository %s: %s", name, environmentName, repository, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(*policy.ID, 10))

	tflog.Debug(ctx, "created GitHub deployment branch policy", map[string]interface{}{
		"id":               data.ID.ValueString(),
		"repository":       repository,
		"environment_name": environmentName,
		"name":             name,
	})

	// Read the created resource to populate all computed fields
	r.readGithubRepositoryDeploymentBranchPolicy(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryDeploymentBranchPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubRepositoryDeploymentBranchPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubRepositoryDeploymentBranchPolicy(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryDeploymentBranchPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubRepositoryDeploymentBranchPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repository := data.Repository.ValueString()
	environmentName := data.EnvironmentName.ValueString()
	name := data.Name.ValueString()

	id, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID Format",
			fmt.Sprintf("Unable to convert deployment branch policy ID to integer: %s", err.Error()),
		)
		return
	}

	_, _, err = client.Repositories.UpdateDeploymentBranchPolicy(ctx, owner, repository, environmentName, int64(id), &github.DeploymentBranchPolicyRequest{
		Name: &name,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating deployment branch policy",
			fmt.Sprintf("Could not update deployment branch policy %d for environment %s in repository %s: %s", id, environmentName, repository, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "updated GitHub deployment branch policy", map[string]interface{}{
		"id":               data.ID.ValueString(),
		"repository":       repository,
		"environment_name": environmentName,
		"name":             name,
	})

	// Read the updated resource to populate all computed fields
	r.readGithubRepositoryDeploymentBranchPolicy(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryDeploymentBranchPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubRepositoryDeploymentBranchPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repository := data.Repository.ValueString()
	environmentName := data.EnvironmentName.ValueString()

	id, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID Format",
			fmt.Sprintf("Unable to convert deployment branch policy ID to integer: %s", err.Error()),
		)
		return
	}

	_, err = client.Repositories.DeleteDeploymentBranchPolicy(ctx, owner, repository, environmentName, int64(id))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting deployment branch policy",
			fmt.Sprintf("Could not delete deployment branch policy %d for environment %s in repository %s: %s", id, environmentName, repository, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub deployment branch policy", map[string]interface{}{
		"id":               data.ID.ValueString(),
		"repository":       repository,
		"environment_name": environmentName,
	})
}

func (r *githubRepositoryDeploymentBranchPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "repository:environment_name:id"
	parts := strings.Split(req.ID, ":")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in the format 'repository:environment_name:id', got: %s", req.ID),
		)
		return
	}

	repository := parts[0]
	environmentName := parts[1]
	id := parts[2]

	data := githubRepositoryDeploymentBranchPolicyResourceModel{
		ID:              types.StringValue(id),
		Repository:      types.StringValue(repository),
		EnvironmentName: types.StringValue(environmentName),
	}

	r.readGithubRepositoryDeploymentBranchPolicy(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Helper function to read the deployment branch policy
func (r *githubRepositoryDeploymentBranchPolicyResource) readGithubRepositoryDeploymentBranchPolicy(ctx context.Context, data *githubRepositoryDeploymentBranchPolicyResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	repository := data.Repository.ValueString()
	environmentName := data.EnvironmentName.ValueString()

	id, err := strconv.Atoi(data.ID.ValueString())
	if err != nil {
		diags.AddError(
			"Invalid ID Format",
			fmt.Sprintf("Unable to convert deployment branch policy ID to integer: %s", err.Error()),
		)
		return
	}

	// Set up context with ID for caching
	ctx = context.WithValue(ctx, githubpkg.CtxId, data.ID.ValueString())

	// Add etag to context for conditional requests if we have one
	if !data.Etag.IsNull() && !data.Etag.IsUnknown() {
		ctx = context.WithValue(ctx, githubpkg.CtxEtag, data.Etag.ValueString())
	}

	policy, resp, err := client.Repositories.GetDeploymentBranchPolicy(ctx, owner, repository, environmentName, int64(id))
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing deployment branch policy from state because it no longer exists in GitHub", map[string]interface{}{
					"repository":       repository,
					"environment_name": environmentName,
					"id":               data.ID.ValueString(),
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Error reading deployment branch policy",
			fmt.Sprintf("Could not read deployment branch policy %d for environment %s in repository %s: %s", id, environmentName, repository, err.Error()),
		)
		return
	}

	data.Repository = types.StringValue(repository)
	data.EnvironmentName = types.StringValue(environmentName)
	data.Name = types.StringValue(policy.GetName())
	data.Etag = types.StringValue(resp.Header.Get("ETag"))

	tflog.Debug(ctx, "successfully read GitHub deployment branch policy", map[string]interface{}{
		"id":               data.ID.ValueString(),
		"repository":       repository,
		"environment_name": environmentName,
		"name":             data.Name.ValueString(),
	})
}
