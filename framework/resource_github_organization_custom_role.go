package framework

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ resource.Resource                = &githubOrganizationCustomRoleResource{}
	_ resource.ResourceWithConfigure   = &githubOrganizationCustomRoleResource{}
	_ resource.ResourceWithImportState = &githubOrganizationCustomRoleResource{}
)

type githubOrganizationCustomRoleResource struct {
	client *githubpkg.Owner
}

type githubOrganizationCustomRoleResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	BaseRole    types.String `tfsdk:"base_role"`
	Permissions types.Set    `tfsdk:"permissions"`
	Description types.String `tfsdk:"description"`
}

func NewGithubOrganizationCustomRoleResource() resource.Resource {
	return &githubOrganizationCustomRoleResource{}
}

func (r *githubOrganizationCustomRoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_custom_role"
}

func (r *githubOrganizationCustomRoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a custom role in a GitHub Organization for use in repositories.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the custom repository role.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The organization custom repository role to create.",
				Required:    true,
			},
			"base_role": schema.StringAttribute{
				Description: "The base role for the custom repository role.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("read", "triage", "write", "maintain"),
				},
			},
			"permissions": schema.SetAttribute{
				Description: "The permissions for the custom repository role.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "The description of the custom repository role.",
				Optional:    true,
			},
		},
	}
}

func (r *githubOrganizationCustomRoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubOrganizationCustomRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubOrganizationCustomRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	orgName := r.client.Name()
	client := r.client.V3Client()

	// Convert permissions set to string slice
	var permissions []string
	resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createOptions := &github.CreateOrUpdateCustomRepoRoleOptions{
		Name:        github.Ptr(plan.Name.ValueString()),
		BaseRole:    github.Ptr(plan.BaseRole.ValueString()),
		Permissions: permissions,
	}

	if !plan.Description.IsNull() {
		createOptions.Description = github.Ptr(plan.Description.ValueString())
	}

	role, _, err := client.Organizations.CreateCustomRepoRole(ctx, orgName, createOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating organization custom role",
			fmt.Sprintf("Could not create organization custom role %s: %s", plan.Name.ValueString(), err),
		)
		return
	}

	plan.ID = types.StringValue(strconv.FormatInt(role.GetID(), 10))

	// Read the created role to update state
	r.updateStateFromRole(ctx, &plan, role)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubOrganizationCustomRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubOrganizationCustomRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	orgName := r.client.Name()
	client := r.client.V3Client()

	// ListCustomRepoRoles returns a list of all custom repository roles for an organization.
	// There is an API endpoint for getting a single custom repository role, but is not
	// implemented in the go-github library.
	roleList, _, err := client.Organizations.ListCustomRepoRoles(ctx, orgName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing organization custom role %s from state because it no longer exists in GitHub", state.ID.ValueString())
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error reading organization custom roles",
			fmt.Sprintf("Could not read organization custom roles: %s", err),
		)
		return
	}

	var role *github.CustomRepoRoles
	roleID := state.ID.ValueString()
	for _, r := range roleList.CustomRepoRoles {
		if strconv.FormatInt(r.GetID(), 10) == roleID {
			role = r
			break
		}
	}

	if role == nil {
		log.Printf("[INFO] Removing organization custom role %s from state because it no longer exists in GitHub", state.ID.ValueString())
		resp.State.RemoveResource(ctx)
		return
	}

	// Update state with role data
	r.updateStateFromRole(ctx, &state, role)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubOrganizationCustomRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubOrganizationCustomRoleResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	orgName := r.client.Name()
	client := r.client.V3Client()

	roleID, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing role ID",
			fmt.Sprintf("Could not parse role ID %s: %s", plan.ID.ValueString(), err),
		)
		return
	}

	// Convert permissions set to string slice
	var permissions []string
	resp.Diagnostics.Append(plan.Permissions.ElementsAs(ctx, &permissions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateOptions := &github.CreateOrUpdateCustomRepoRoleOptions{
		Name:        github.Ptr(plan.Name.ValueString()),
		BaseRole:    github.Ptr(plan.BaseRole.ValueString()),
		Permissions: permissions,
	}

	if !plan.Description.IsNull() {
		updateOptions.Description = github.Ptr(plan.Description.ValueString())
	}

	_, _, err = client.Organizations.UpdateCustomRepoRole(ctx, orgName, roleID, updateOptions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating organization custom role",
			fmt.Sprintf("Could not update organization custom role %s: %s", plan.ID.ValueString(), err),
		)
		return
	}

	// Re-read the updated role
	roleList, _, err := client.Organizations.ListCustomRepoRoles(ctx, orgName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading updated organization custom role",
			fmt.Sprintf("Could not read updated organization custom role: %s", err),
		)
		return
	}

	var updatedRole *github.CustomRepoRoles
	roleIDStr := plan.ID.ValueString()
	for _, r := range roleList.CustomRepoRoles {
		if strconv.FormatInt(r.GetID(), 10) == roleIDStr {
			updatedRole = r
			break
		}
	}

	if updatedRole == nil {
		resp.Diagnostics.AddError(
			"Updated role not found",
			fmt.Sprintf("Could not find updated role with ID %s", plan.ID.ValueString()),
		)
		return
	}

	// Update state with updated role data
	r.updateStateFromRole(ctx, &plan, updatedRole)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubOrganizationCustomRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubOrganizationCustomRoleResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	orgName := r.client.Name()
	client := r.client.V3Client()

	roleID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing role ID",
			fmt.Sprintf("Could not parse role ID %s: %s", state.ID.ValueString(), err),
		)
		return
	}

	_, err = client.Organizations.DeleteCustomRepoRole(ctx, orgName, roleID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting organization custom role",
			fmt.Sprintf("Could not delete organization custom role %s: %s", state.ID.ValueString(), err),
		)
		return
	}
}

func (r *githubOrganizationCustomRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Organization custom roles can be imported using just the role ID
	// since they're always associated with the configured organization
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// Helper function to update state from GitHub API response
func (r *githubOrganizationCustomRoleResource) updateStateFromRole(ctx context.Context, state *githubOrganizationCustomRoleResourceModel, role *github.CustomRepoRoles) {
	state.Name = types.StringValue(role.GetName())
	state.BaseRole = types.StringValue(role.GetBaseRole())

	if role.GetDescription() != "" {
		state.Description = types.StringValue(role.GetDescription())
	} else {
		state.Description = types.StringNull()
	}

	// Convert permissions slice to Set
	permissionElements := make([]attr.Value, len(role.Permissions))
	for i, permission := range role.Permissions {
		permissionElements[i] = types.StringValue(permission)
	}

	permissionsSet, diags := types.SetValue(types.StringType, permissionElements)
	if diags.HasError() {
		// This should not happen in practice, but handle it gracefully
		log.Printf("[ERROR] Failed to convert permissions to set: %v", diags.Errors())
		return
	}
	state.Permissions = permissionsSet
}
