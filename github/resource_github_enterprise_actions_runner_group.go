package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

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

)

var (
	_ resource.Resource                = &githubEnterpriseActionsRunnerGroupResource{}
	_ resource.ResourceWithConfigure   = &githubEnterpriseActionsRunnerGroupResource{}
	_ resource.ResourceWithImportState = &githubEnterpriseActionsRunnerGroupResource{}
)

func NewGithubEnterpriseActionsRunnerGroupResource() resource.Resource {
	return &githubEnterpriseActionsRunnerGroupResource{}
}

type githubEnterpriseActionsRunnerGroupResource struct {
	client *Owner
}

type githubEnterpriseActionsRunnerGroupResourceModel struct {
	// Required attributes
	EnterpriseSlug types.String `tfsdk:"enterprise_slug"`
	Name           types.String `tfsdk:"name"`
	Visibility     types.String `tfsdk:"visibility"`

	// Optional attributes
	AllowsPublicRepositories types.Bool `tfsdk:"allows_public_repositories"`
	RestrictedToWorkflows    types.Bool `tfsdk:"restricted_to_workflows"`
	SelectedWorkflows        types.List `tfsdk:"selected_workflows"`
	SelectedOrganizationIds  types.Set  `tfsdk:"selected_organization_ids"`

	// Computed attributes
	ID                       types.String `tfsdk:"id"`
	Default                  types.Bool   `tfsdk:"default"`
	Etag                     types.String `tfsdk:"etag"`
	RunnersUrl               types.String `tfsdk:"runners_url"`
	SelectedOrganizationsUrl types.String `tfsdk:"selected_organizations_url"`
}

func (r *githubEnterpriseActionsRunnerGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_enterprise_actions_runner_group"
}

func (r *githubEnterpriseActionsRunnerGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages an Actions Runner Group within a GitHub enterprise.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the runner group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enterprise_slug": schema.StringAttribute{
				Description: "The slug of the enterprise.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
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
					stringvalidator.OneOf("all", "selected"),
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
			"selected_organization_ids": schema.SetAttribute{
				Description: "List of organization IDs that can access the runner group.",
				Optional:    true,
				ElementType: types.Int64Type,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Set{},
			},
			"default": schema.BoolAttribute{
				Description: "Whether this is the default runner group.",
				Computed:    true,
			},
			"etag": schema.StringAttribute{
				Description: "An etag representing the runner group object",
				Computed:    true,
			},
			"runners_url": schema.StringAttribute{
				Description: "The GitHub API URL for the runner group's runners.",
				Computed:    true,
			},
			"selected_organizations_url": schema.StringAttribute{
				Description: "GitHub API URL for the runner group's organizations.",
				Computed:    true,
			},
		},
	}
}

func (r *githubEnterpriseActionsRunnerGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubEnterpriseActionsRunnerGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubEnterpriseActionsRunnerGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	enterpriseSlug := plan.EnterpriseSlug.ValueString()
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

	// Extract selected organization IDs
	var selectedOrganizationIDs []int64
	hasSelectedOrganizations := !plan.SelectedOrganizationIds.IsNull() && !plan.SelectedOrganizationIds.IsUnknown()
	if hasSelectedOrganizations {
		resp.Diagnostics.Append(plan.SelectedOrganizationIds.ElementsAs(ctx, &selectedOrganizationIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Validate visibility and selected organizations
	if visibility != "selected" && hasSelectedOrganizations {
		resp.Diagnostics.AddError(
			"Configuration Error",
			"cannot use selected_organization_ids without visibility being set to selected",
		)
		return
	}

	// Create the runner group
	enterpriseRunnerGroup, apiResp, err := client.Enterprise.CreateEnterpriseRunnerGroup(ctx,
		enterpriseSlug,
		github.CreateEnterpriseRunnerGroupRequest{
			Name:                     &name,
			Visibility:               &visibility,
			SelectedOrganizationIDs:  selectedOrganizationIDs,
			AllowsPublicRepositories: &allowsPublicRepositories,
			RestrictedToWorkflows:    &restrictedToWorkflows,
			SelectedWorkflows:        selectedWorkflows,
		},
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating enterprise actions runner group", err.Error())
		return
	}

	// Set the ID
	plan.ID = types.StringValue(strconv.FormatInt(enterpriseRunnerGroup.GetID(), 10))

	// Set computed and response attributes
	if apiResp != nil {
		plan.Etag = types.StringValue(apiResp.Header.Get("ETag"))
	}

	plan.Default = types.BoolValue(enterpriseRunnerGroup.GetDefault())
	plan.RunnersUrl = types.StringValue(enterpriseRunnerGroup.GetRunnersURL())
	plan.SelectedOrganizationsUrl = types.StringValue(enterpriseRunnerGroup.GetSelectedOrganizationsURL())

	// Read the resource to populate all fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubEnterpriseActionsRunnerGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubEnterpriseActionsRunnerGroupResourceModel
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

func (r *githubEnterpriseActionsRunnerGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubEnterpriseActionsRunnerGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	enterpriseSlug := plan.EnterpriseSlug.ValueString()
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
	updateOptions := github.UpdateEnterpriseRunnerGroupRequest{
		Name:                     &name,
		Visibility:               &visibility,
		RestrictedToWorkflows:    &restrictedToWorkflows,
		SelectedWorkflows:        selectedWorkflows,
		AllowsPublicRepositories: &allowsPublicRepositories,
	}

	_, _, err = client.Enterprise.UpdateEnterpriseRunnerGroup(ctx, enterpriseSlug, runnerGroupID, updateOptions)
	if err != nil {
		resp.Diagnostics.AddError("Error updating enterprise actions runner group", err.Error())
		return
	}

	// Handle selected organization IDs
	var selectedOrganizationIDs []int64
	hasSelectedOrganizations := !plan.SelectedOrganizationIds.IsNull() && !plan.SelectedOrganizationIds.IsUnknown()
	if hasSelectedOrganizations {
		resp.Diagnostics.Append(plan.SelectedOrganizationIds.ElementsAs(ctx, &selectedOrganizationIDs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Update organization access
	orgOptions := github.SetOrgAccessRunnerGroupRequest{SelectedOrganizationIDs: selectedOrganizationIDs}
	_, err = client.Enterprise.SetOrganizationAccessRunnerGroup(ctx, enterpriseSlug, runnerGroupID, orgOptions)
	if err != nil {
		resp.Diagnostics.AddError("Error setting organization access for runner group", err.Error())
		return
	}

	// Read the resource to populate all fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubEnterpriseActionsRunnerGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubEnterpriseActionsRunnerGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	enterpriseSlug := state.EnterpriseSlug.ValueString()
	client := r.client.V3Client()

	runnerGroupID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing runner group ID", err.Error())
		return
	}

	log.Printf("[INFO] Deleting enterprise runner group: %s/%s (%s)", enterpriseSlug, state.Name.ValueString(), state.ID.ValueString())
	_, err = client.Enterprise.DeleteEnterpriseRunnerGroup(ctx, enterpriseSlug, runnerGroupID)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting enterprise actions runner group", err.Error())
		return
	}
}

func (r *githubEnterpriseActionsRunnerGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Import ID must be in the format: <enterprise_slug>/<runner_group_id>",
		)
		return
	}

	enterpriseSlug, runnerGroupID := parts[0], parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), runnerGroupID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("enterprise_slug"), enterpriseSlug)...)
}

// Helper methods

func (r *githubEnterpriseActionsRunnerGroupResource) readResource(ctx context.Context, model *githubEnterpriseActionsRunnerGroupResourceModel, diagnostics *diag.Diagnostics) {
	enterpriseSlug := model.EnterpriseSlug.ValueString()
	client := r.client.V3Client()

	runnerGroupID, err := strconv.ParseInt(model.ID.ValueString(), 10, 64)
	if err != nil {
		diagnostics.AddError("Error parsing runner group ID", err.Error())
		return
	}

	// Create context with ETag if available for conditional requests
	readCtx := ctx
	if !model.Etag.IsNull() && !model.Etag.IsUnknown() {
		// Note: The SDKv2 version uses ctxEtag context key, we'll use standard headers
		readCtx = context.WithValue(ctx, "etag", model.Etag.ValueString())
	}

	enterpriseRunnerGroup, resp, err := r.getEnterpriseRunnerGroup(client, readCtx, enterpriseSlug, runnerGroupID)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing enterprise runner group %s/%s from state because it no longer exists in GitHub",
					enterpriseSlug, model.ID.ValueString())
				// Clear the ID to mark for removal from state
				model.ID = types.StringValue("")
				return
			}
		}
		diagnostics.AddError("Error reading enterprise actions runner group", err.Error())
		return
	}

	// If runner group is nil (typically not modified) we can return early
	if enterpriseRunnerGroup == nil {
		return
	}

	// Set ETag
	if resp != nil {
		model.Etag = types.StringValue(resp.Header.Get("ETag"))
	}

	// Set basic attributes
	model.Name = types.StringValue(enterpriseRunnerGroup.GetName())
	model.Visibility = types.StringValue(enterpriseRunnerGroup.GetVisibility())
	model.AllowsPublicRepositories = types.BoolValue(enterpriseRunnerGroup.GetAllowsPublicRepositories())
	model.RestrictedToWorkflows = types.BoolValue(enterpriseRunnerGroup.GetRestrictedToWorkflows())
	model.Default = types.BoolValue(enterpriseRunnerGroup.GetDefault())
	model.RunnersUrl = types.StringValue(enterpriseRunnerGroup.GetRunnersURL())
	model.SelectedOrganizationsUrl = types.StringValue(enterpriseRunnerGroup.GetSelectedOrganizationsURL())

	// Set selected workflows
	if len(enterpriseRunnerGroup.SelectedWorkflows) > 0 {
		workflowElements := make([]attr.Value, len(enterpriseRunnerGroup.SelectedWorkflows))
		for i, workflow := range enterpriseRunnerGroup.SelectedWorkflows {
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

	// Get selected organization IDs by listing organization access
	selectedOrganizationIDs := []int64{}
	optionsOrgs := github.ListOptions{
		PerPage: 100, // maxPerPage equivalent
	}

	for {
		enterpriseRunnerGroupOrganizations, orgResp, err := client.Enterprise.ListOrganizationAccessRunnerGroup(ctx, enterpriseSlug, runnerGroupID, &optionsOrgs)
		if err != nil {
			diagnostics.AddError("Error reading organization access for runner group", err.Error())
			return
		}

		for _, org := range enterpriseRunnerGroupOrganizations.Organizations {
			selectedOrganizationIDs = append(selectedOrganizationIDs, *org.ID)
		}

		if orgResp.NextPage == 0 {
			break
		}

		optionsOrgs.Page = orgResp.NextPage
	}

	// Set selected organization IDs
	if len(selectedOrganizationIDs) > 0 {
		orgElements := make([]attr.Value, len(selectedOrganizationIDs))
		for i, orgID := range selectedOrganizationIDs {
			orgElements[i] = types.Int64Value(orgID)
		}
		setValue, setDiags := types.SetValue(types.Int64Type, orgElements)
		diagnostics.Append(setDiags...)
		if diagnostics.HasError() {
			return
		}
		model.SelectedOrganizationIds = setValue
	} else {
		model.SelectedOrganizationIds = types.SetNull(types.Int64Type)
	}
}

func (r *githubEnterpriseActionsRunnerGroupResource) getEnterpriseRunnerGroup(client *github.Client, ctx context.Context, enterpriseSlug string, groupID int64) (*github.EnterpriseRunnerGroup, *github.Response, error) {
	enterpriseRunnerGroup, resp, err := client.Enterprise.GetEnterpriseRunnerGroup(ctx, enterpriseSlug, groupID)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok && ghErr.Response.StatusCode == http.StatusNotModified {
			// ignore error StatusNotModified
			return enterpriseRunnerGroup, resp, nil
		}
	}
	return enterpriseRunnerGroup, resp, err
}
