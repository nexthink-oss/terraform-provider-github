package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

)

var (
	_ resource.Resource                = &githubActionsRepositoryAccessLevelResource{}
	_ resource.ResourceWithConfigure   = &githubActionsRepositoryAccessLevelResource{}
	_ resource.ResourceWithImportState = &githubActionsRepositoryAccessLevelResource{}
)

func NewGithubActionsRepositoryAccessLevelResource() resource.Resource {
	return &githubActionsRepositoryAccessLevelResource{}
}

type githubActionsRepositoryAccessLevelResource struct {
	client *Owner
}

type githubActionsRepositoryAccessLevelResourceModel struct {
	// Required attributes
	Repository  types.String `tfsdk:"repository"`
	AccessLevel types.String `tfsdk:"access_level"`

	// Computed attributes - ID is the repository name
	ID types.String `tfsdk:"id"`
}

func (r *githubActionsRepositoryAccessLevelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_repository_access_level"
}

func (r *githubActionsRepositoryAccessLevelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Actions and Reusable Workflow access for a GitHub repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The name of the repository.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The GitHub repository.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"access_level": schema.StringAttribute{
				Description: "Where the actions or reusable workflows of the repository may be used. Possible values are 'none', 'user', 'organization', or 'enterprise'.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("none", "user", "organization", "enterprise"),
				},
			},
		},
	}
}

func (r *githubActionsRepositoryAccessLevelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubActionsRepositoryAccessLevelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubActionsRepositoryAccessLevelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := plan.Repository.ValueString()
	accessLevel := plan.AccessLevel.ValueString()
	client := r.client.V3Client()

	actionAccessLevel := github.RepositoryActionsAccessLevel{
		AccessLevel: github.Ptr(accessLevel),
	}

	_, err := client.Repositories.EditActionsAccessLevel(ctx, owner, repoName, actionAccessLevel)
	if err != nil {
		resp.Diagnostics.AddError("Error setting repository actions access level", err.Error())
		return
	}

	plan.ID = types.StringValue(repoName)

	// Read the resource to populate all fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubActionsRepositoryAccessLevelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubActionsRepositoryAccessLevelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	r.readResource(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubActionsRepositoryAccessLevelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubActionsRepositoryAccessLevelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := plan.Repository.ValueString()
	accessLevel := plan.AccessLevel.ValueString()
	client := r.client.V3Client()

	actionAccessLevel := github.RepositoryActionsAccessLevel{
		AccessLevel: github.Ptr(accessLevel),
	}

	_, err := client.Repositories.EditActionsAccessLevel(ctx, owner, repoName, actionAccessLevel)
	if err != nil {
		resp.Diagnostics.AddError("Error updating repository actions access level", err.Error())
		return
	}

	// Read the resource to populate all fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubActionsRepositoryAccessLevelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubActionsRepositoryAccessLevelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := state.ID.ValueString()
	client := r.client.V3Client()

	// Reset to "none" on delete (same as SDKv2 implementation)
	actionAccessLevel := github.RepositoryActionsAccessLevel{
		AccessLevel: github.Ptr("none"),
	}

	_, err := client.Repositories.EditActionsAccessLevel(ctx, owner, repoName, actionAccessLevel)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting repository actions access level", err.Error())
		return
	}
}

func (r *githubActionsRepositoryAccessLevelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// Set repository to the same value as id for consistency
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository"), req.ID)...)
}

// Helper methods

func (r *githubActionsRepositoryAccessLevelResource) readResource(ctx context.Context, model *githubActionsRepositoryAccessLevelResourceModel, diagnostics *diag.Diagnostics) {
	owner := r.client.Name()
	repoName := model.ID.ValueString()
	client := r.client.V3Client()

	actionAccessLevel, _, err := client.Repositories.GetActionsAccessLevel(ctx, owner, repoName)
	if err != nil {
		diagnostics.AddError("Error reading repository actions access level", err.Error())
		return
	}

	// Set attributes from API response
	model.AccessLevel = types.StringValue(actionAccessLevel.GetAccessLevel())
	model.Repository = types.StringValue(repoName)
}
