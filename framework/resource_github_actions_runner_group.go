package framework

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ resource.Resource                = &githubActionsRunnerGroupResource{}
	_ resource.ResourceWithConfigure   = &githubActionsRunnerGroupResource{}
	_ resource.ResourceWithImportState = &githubActionsRunnerGroupResource{}
)

func NewGithubActionsRunnerGroupResource() resource.Resource {
	return &githubActionsRunnerGroupResource{}
}

type githubActionsRunnerGroupResource struct {
	client *githubpkg.Owner
}

type githubActionsRunnerGroupResourceModel struct {
	// Required attributes
	Name       types.String `tfsdk:"name"`
	Visibility types.String `tfsdk:"visibility"`

	// Optional attributes
	AllowsPublicRepositories types.Bool `tfsdk:"allows_public_repositories"`
	RestrictedToWorkflows    types.Bool `tfsdk:"restricted_to_workflows"`
	SelectedWorkflows        types.List `tfsdk:"selected_workflows"`
	SelectedRepositoryIds    types.Set  `tfsdk:"selected_repository_ids"`

	// Computed attributes
	ID                      types.String `tfsdk:"id"`
	Default                 types.Bool   `tfsdk:"default"`
	Etag                    types.String `tfsdk:"etag"`
	Inherited               types.Bool   `tfsdk:"inherited"`
	RunnersUrl              types.String `tfsdk:"runners_url"`
	SelectedRepositoriesUrl types.String `tfsdk:"selected_repositories_url"`
}

func (r *githubActionsRunnerGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_runner_group"
}

func (r *githubActionsRunnerGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages an Actions Runner Group within a GitHub organization",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the runner group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Name of the runner group.",
				Required:    true,
			},
			"visibility": schema.StringAttribute{
				Description: "The visibility of the runner group.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("all", "selected", "private"),
				},
			},
			"allows_public_repositories": schema.BoolAttribute{
				Description: "Whether public repositories can be added to the runner group.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"restricted_to_workflows": schema.BoolAttribute{
				Description: "If 'true', the runner group will be restricted to running only the workflows specified in the 'selected_workflows' array. Defaults to 'false'.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"selected_workflows": schema.ListAttribute{
				Description: "List of workflows the runner group should be allowed to run. This setting will be ignored unless restricted_to_workflows is set to 'true'.",
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"selected_repository_ids": schema.SetAttribute{
				Description: "List of repository IDs that can access the runner group.",
				Optional:    true,
				ElementType: types.Int64Type,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"default": schema.BoolAttribute{
				Description: "Whether this is the default runner group.",
				Computed:    true,
			},
			"etag": schema.StringAttribute{
				Description: "An etag representing the runner group object",
				Computed:    true,
			},
			"inherited": schema.BoolAttribute{
				Description: "Whether the runner group is inherited from the enterprise level",
				Computed:    true,
			},
			"runners_url": schema.StringAttribute{
				Description: "The GitHub API URL for the runner group's runners.",
				Computed:    true,
			},
			"selected_repositories_url": schema.StringAttribute{
				Description: "GitHub API URL for the runner group's repositories.",
				Computed:    true,
			},
		},
	}
}

func (r *githubActionsRunnerGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubActionsRunnerGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubActionsRunnerGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

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

	orgName := r.client.Name()
	name := plan.Name.ValueString()
	visibility := plan.Visibility.ValueString()
	allowsPublicRepositories := plan.AllowsPublicRepositories.ValueBool()
	restrictedToWorkflows := plan.RestrictedToWorkflows.ValueBool()

	client := r.client.V3Client()

	// Extract selected workflows
	var selectedWorkflows []string
	if !plan.SelectedWorkflows.IsNull() && !plan.SelectedWorkflows.IsUnknown() {
		resp.Diagnostics.Append(plan.SelectedWorkflows.ElementsAs(ctx, &selectedWorkflows, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Extract selected repository IDs
	var selectedRepositoryIDs []int64
	hasSelectedRepositories := !plan.SelectedRepositoryIds.IsNull() && !plan.SelectedRepositoryIds.IsUnknown()
	if hasSelectedRepositories {
		resp.Diagnostics.Append(plan.SelectedRepositoryIds.ElementsAs(ctx, &selectedRepositoryIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Validate visibility and selected repositories
	if visibility != "selected" && hasSelectedRepositories {
		resp.Diagnostics.AddError(
			"Configuration Error",
			"cannot use selected_repository_ids without visibility being set to selected",
		)
		return
	}

	// Create the runner group
	runnerGroup, apiResp, err := client.Actions.CreateOrganizationRunnerGroup(ctx,
		orgName,
		github.CreateRunnerGroupRequest{
			Name:                     &name,
			Visibility:               &visibility,
			RestrictedToWorkflows:    &restrictedToWorkflows,
			SelectedRepositoryIDs:    selectedRepositoryIDs,
			SelectedWorkflows:        selectedWorkflows,
			AllowsPublicRepositories: &allowsPublicRepositories,
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating actions runner group", err.Error())
		return
	}

	// Set the ID
	plan.ID = types.StringValue(strconv.FormatInt(runnerGroup.GetID(), 10))

	// Set computed and response attributes
	if apiResp != nil {
		plan.Etag = types.StringValue(apiResp.Header.Get("ETag"))
	}

	plan.Default = types.BoolValue(runnerGroup.GetDefault())
	plan.Inherited = types.BoolValue(runnerGroup.GetInherited())
	plan.RunnersUrl = types.StringValue(runnerGroup.GetRunnersURL())
	plan.SelectedRepositoriesUrl = types.StringValue(runnerGroup.GetSelectedRepositoriesURL())

	// Read the resource to populate all fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubActionsRunnerGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubActionsRunnerGroupResourceModel
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

func (r *githubActionsRunnerGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubActionsRunnerGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

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

	orgName := r.client.Name()
	name := plan.Name.ValueString()
	visibility := plan.Visibility.ValueString()
	allowsPublicRepositories := plan.AllowsPublicRepositories.ValueBool()
	restrictedToWorkflows := plan.RestrictedToWorkflows.ValueBool()

	client := r.client.V3Client()

	// Extract selected workflows
	var selectedWorkflows []string
	if !plan.SelectedWorkflows.IsNull() && !plan.SelectedWorkflows.IsUnknown() {
		resp.Diagnostics.Append(plan.SelectedWorkflows.ElementsAs(ctx, &selectedWorkflows, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Parse runner group ID
	runnerGroupID, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing runner group ID", err.Error())
		return
	}

	// Update the basic runner group properties
	updateOptions := github.UpdateRunnerGroupRequest{
		Name:                     &name,
		Visibility:               &visibility,
		RestrictedToWorkflows:    &restrictedToWorkflows,
		SelectedWorkflows:        selectedWorkflows,
		AllowsPublicRepositories: &allowsPublicRepositories,
	}

	_, _, err = client.Actions.UpdateOrganizationRunnerGroup(ctx, orgName, runnerGroupID, updateOptions)
	if err != nil {
		resp.Diagnostics.AddError("Error updating actions runner group", err.Error())
		return
	}

	// Handle selected repository IDs
	var selectedRepositoryIDs []int64
	hasSelectedRepositories := !plan.SelectedRepositoryIds.IsNull() && !plan.SelectedRepositoryIds.IsUnknown()
	if hasSelectedRepositories {
		resp.Diagnostics.Append(plan.SelectedRepositoryIds.ElementsAs(ctx, &selectedRepositoryIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Update repository access
	reposOptions := github.SetRepoAccessRunnerGroupRequest{SelectedRepositoryIDs: selectedRepositoryIDs}
	_, err = client.Actions.SetRepositoryAccessRunnerGroup(ctx, orgName, runnerGroupID, reposOptions)
	if err != nil {
		resp.Diagnostics.AddError("Error setting repository access for runner group", err.Error())
		return
	}

	// Read the resource to populate all fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubActionsRunnerGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubActionsRunnerGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

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

	orgName := r.client.Name()
	client := r.client.V3Client()

	runnerGroupID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing runner group ID", err.Error())
		return
	}

	log.Printf("[INFO] Deleting organization runner group: %s/%s (%s)", orgName, state.Name.ValueString(), state.ID.ValueString())
	_, err = client.Actions.DeleteOrganizationRunnerGroup(ctx, orgName, runnerGroupID)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting actions runner group", err.Error())
		return
	}
}

func (r *githubActionsRunnerGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID is just the runner group ID for organization-level resources
	runnerGroupID := req.ID

	// Validate that the ID can be parsed as an integer
	_, err := strconv.ParseInt(runnerGroupID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Import ID must be a valid runner group ID (integer), got: %s", runnerGroupID),
		)
		return
	}

	// Set the ID in state - Terraform will call Read to populate the rest
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), runnerGroupID)...)
}

// Helper methods

func (r *githubActionsRunnerGroupResource) readResource(ctx context.Context, model *githubActionsRunnerGroupResourceModel, diagnostics *diag.Diagnostics) {
	// Check if we're working with an organization
	if !r.client.IsOrganization {
		diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization",
		)
		return
	}

	orgName := r.client.Name()
	client := r.client.V3Client()

	runnerGroupID, err := strconv.ParseInt(model.ID.ValueString(), 10, 64)
	if err != nil {
		diagnostics.AddError("Error parsing runner group ID", err.Error())
		return
	}

	// Create context with ETag if available for conditional requests
	readCtx := ctx
	if !model.Etag.IsNull() && !model.Etag.IsUnknown() {
		// Note: The SDKv2 version uses custom context keys for ETag handling
		// For now, we'll use the basic context and rely on standard GitHub API behavior
	}

	runnerGroup, resp, err := r.getOrganizationRunnerGroup(client, readCtx, orgName, runnerGroupID)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing organization runner group %s/%s from state because it no longer exists in GitHub",
					orgName, model.ID.ValueString())
				// Clear the ID to mark for removal from state
				model.ID = types.StringValue("")
				return
			}
		}
		diagnostics.AddError("Error reading actions runner group", err.Error())
		return
	}

	// If runner group is nil (typically not modified) we can return early
	if runnerGroup == nil {
		return
	}

	// Set ETag
	if resp != nil {
		model.Etag = types.StringValue(resp.Header.Get("ETag"))
	}

	// Set basic attributes
	model.Name = types.StringValue(runnerGroup.GetName())
	model.Visibility = types.StringValue(runnerGroup.GetVisibility())
	model.AllowsPublicRepositories = types.BoolValue(runnerGroup.GetAllowsPublicRepositories())
	model.RestrictedToWorkflows = types.BoolValue(runnerGroup.GetRestrictedToWorkflows())
	model.Default = types.BoolValue(runnerGroup.GetDefault())
	model.Inherited = types.BoolValue(runnerGroup.GetInherited())
	model.RunnersUrl = types.StringValue(runnerGroup.GetRunnersURL())
	model.SelectedRepositoriesUrl = types.StringValue(runnerGroup.GetSelectedRepositoriesURL())

	// Set selected workflows
	if len(runnerGroup.SelectedWorkflows) > 0 {
		workflowElements := make([]attr.Value, len(runnerGroup.SelectedWorkflows))
		for i, workflow := range runnerGroup.SelectedWorkflows {
			workflowElements[i] = types.StringValue(workflow)
		}
		listValue, listDiags := types.ListValue(types.StringType, workflowElements)
		diagnostics.Append(listDiags...)
		if diagnostics.HasError() {
			return
		}
		model.SelectedWorkflows = listValue
	} else {
		model.SelectedWorkflows = types.ListNull(types.StringType)
	}

	// Get selected repository IDs by listing repository access
	selectedRepositoryIDs := []int64{}
	options := github.ListOptions{
		PerPage: 100, // maxPerPage equivalent
	}

	for {
		runnerGroupRepositories, repoResp, err := client.Actions.ListRepositoryAccessRunnerGroup(ctx, orgName, runnerGroupID, &options)
		if err != nil {
			diagnostics.AddError("Error reading repository access for runner group", err.Error())
			return
		}

		for _, repo := range runnerGroupRepositories.Repositories {
			selectedRepositoryIDs = append(selectedRepositoryIDs, *repo.ID)
		}

		if repoResp.NextPage == 0 {
			break
		}

		options.Page = repoResp.NextPage
	}

	// Set selected repository IDs
	if len(selectedRepositoryIDs) > 0 {
		repoElements := make([]attr.Value, len(selectedRepositoryIDs))
		for i, repoID := range selectedRepositoryIDs {
			repoElements[i] = types.Int64Value(repoID)
		}
		setValue, setDiags := types.SetValue(types.Int64Type, repoElements)
		diagnostics.Append(setDiags...)
		if diagnostics.HasError() {
			return
		}
		model.SelectedRepositoryIds = setValue
	} else {
		model.SelectedRepositoryIds = types.SetNull(types.Int64Type)
	}
}

func (r *githubActionsRunnerGroupResource) getOrganizationRunnerGroup(client *github.Client, ctx context.Context, orgName string, groupID int64) (*github.RunnerGroup, *github.Response, error) {
	runnerGroup, resp, err := client.Actions.GetOrganizationRunnerGroup(ctx, orgName, groupID)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok && ghErr.Response.StatusCode == http.StatusNotModified {
			// ignore error StatusNotModified
			return runnerGroup, resp, nil
		}
	}
	return runnerGroup, resp, err
}
