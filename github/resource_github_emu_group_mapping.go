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
)

var (
	_ resource.Resource                = &githubEmuGroupMappingResource{}
	_ resource.ResourceWithConfigure   = &githubEmuGroupMappingResource{}
	_ resource.ResourceWithImportState = &githubEmuGroupMappingResource{}
)

func NewGithubEmuGroupMappingResource() resource.Resource {
	return &githubEmuGroupMappingResource{}
}

type githubEmuGroupMappingResource struct {
	client *Owner
}

type githubEmuGroupMappingResourceModel struct {
	ID       types.String `tfsdk:"id"`
	TeamSlug types.String `tfsdk:"team_slug"`
	GroupID  types.Int64  `tfsdk:"group_id"`
	Etag     types.String `tfsdk:"etag"`
}

func (r *githubEmuGroupMappingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_emu_group_mapping"
}

func (r *githubEmuGroupMappingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages mappings between external groups for enterprise managed users.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the EMU group mapping.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_slug": schema.StringAttribute{
				Description: "Slug of the GitHub team.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.Int64Attribute{
				Description: "Integer corresponding to the external group ID to be linked.",
				Required:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"etag": schema.StringAttribute{
				Description: "The etag of the external group.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubEmuGroupMappingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubEmuGroupMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubEmuGroupMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateEmuGroupMapping(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubEmuGroupMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubEmuGroupMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readEmuGroupMapping(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubEmuGroupMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubEmuGroupMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateEmuGroupMapping(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubEmuGroupMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubEmuGroupMappingResourceModel

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
	teamSlug := data.TeamSlug.ValueString()

	_, err = client.Teams.RemoveConnectedExternalGroup(ctx, owner, teamSlug)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Removing EMU Group Mapping",
			"Could not remove EMU group mapping for team "+teamSlug+": "+err.Error(),
		)
		return
	}
}

func (r *githubEmuGroupMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the group ID from the import ID
	groupID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing EMU Group Mapping",
			"Could not parse group ID: "+err.Error(),
		)
		return
	}

	// Validate this is an organization
	err = r.checkOrganization()
	if err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts: %s", err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	// Get the external group to find the associated team
	group, _, err := client.Teams.GetExternalGroup(ctx, owner, groupID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing EMU Group Mapping",
			"Could not get external group: "+err.Error(),
		)
		return
	}

	if len(group.Teams) != 1 {
		resp.Diagnostics.AddError(
			"Error Importing EMU Group Mapping",
			fmt.Sprintf("Expected exactly one team associated with group, found %d", len(group.Teams)),
		)
		return
	}

	teamSlug := group.Teams[0].TeamName
	id := fmt.Sprintf("teams/%s/external-groups", *teamSlug)

	data := githubEmuGroupMappingResourceModel{
		ID:       types.StringValue(id),
		TeamSlug: types.StringValue(*teamSlug),
		GroupID:  types.Int64Value(groupID),
	}

	r.readEmuGroupMapping(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Helper methods

func (r *githubEmuGroupMappingResource) checkOrganization() error {
	if r.client.Name() == "" {
		return fmt.Errorf("owner is required")
	}
	return nil
}

func (r *githubEmuGroupMappingResource) readEmuGroupMapping(ctx context.Context, data *githubEmuGroupMappingResourceModel, diags *diag.Diagnostics) {
	// Validate this is an organization
	err := r.checkOrganization()
	if err != nil {
		diags.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts: %s", err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	groupID := data.GroupID.ValueInt64()

	group, resp, err := client.Teams.GetExternalGroup(ctx, owner, groupID)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			// If the group is not found, remove it from state
			data.ID = types.StringValue("")
			return
		}
		diags.AddError(
			"Error Reading EMU Group Mapping",
			"Could not read EMU group mapping: "+err.Error(),
		)
		return
	}

	if len(group.Teams) < 1 {
		// If there's no team linked, the mapping was removed outside of Terraform
		data.ID = types.StringValue("")
		return
	}

	// Update computed attributes
	data.Etag = types.StringValue(resp.Header.Get("ETag"))
	data.GroupID = types.Int64Value(group.GetGroupID())

	// Ensure ID is set correctly
	if data.ID.IsNull() || data.ID.ValueString() == "" {
		teamSlug := data.TeamSlug.ValueString()
		data.ID = types.StringValue(fmt.Sprintf("teams/%s/external-groups", teamSlug))
	}
}

func (r *githubEmuGroupMappingResource) updateEmuGroupMapping(ctx context.Context, data *githubEmuGroupMappingResourceModel, diags *diag.Diagnostics) {
	// Validate this is an organization
	err := r.checkOrganization()
	if err != nil {
		diags.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts: %s", err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	teamSlug := data.TeamSlug.ValueString()
	groupID := data.GroupID.ValueInt64()

	externalGroup := &github.ExternalGroup{
		GroupID: &groupID,
	}

	_, _, err = client.Teams.UpdateConnectedExternalGroup(ctx, owner, teamSlug, externalGroup)
	if err != nil {
		diags.AddError(
			"Error Updating EMU Group Mapping",
			"Could not update EMU group mapping for team "+teamSlug+": "+err.Error(),
		)
		return
	}

	// Set the ID
	data.ID = types.StringValue(fmt.Sprintf("teams/%s/external-groups", teamSlug))

	// Read the updated state
	r.readEmuGroupMapping(ctx, data, diags)
}
