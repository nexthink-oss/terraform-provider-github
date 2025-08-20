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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ resource.Resource                = &githubTeamSyncGroupMappingResource{}
	_ resource.ResourceWithConfigure   = &githubTeamSyncGroupMappingResource{}
	_ resource.ResourceWithImportState = &githubTeamSyncGroupMappingResource{}
)

func NewGithubTeamSyncGroupMappingResource() resource.Resource {
	return &githubTeamSyncGroupMappingResource{}
}

type githubTeamSyncGroupMappingResource struct {
	client *Owner
}

type githubTeamSyncGroupMappingResourceModel struct {
	ID       types.String `tfsdk:"id"`
	TeamSlug types.String `tfsdk:"team_slug"`
	Group    types.Set    `tfsdk:"group"`
	Etag     types.String `tfsdk:"etag"`
}

type githubTeamSyncGroupModel struct {
	GroupID          types.String `tfsdk:"group_id"`
	GroupName        types.String `tfsdk:"group_name"`
	GroupDescription types.String `tfsdk:"group_description"`
}

func (r *githubTeamSyncGroupMappingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_sync_group_mapping"
}

func (r *githubTeamSyncGroupMappingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages the connections between a team and its IdP group(s).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the team sync group mapping.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_slug": schema.StringAttribute{
				Description: "Slug of the team.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group": schema.SetNestedAttribute{
				Description: "An Array of GitHub Identity Provider Groups (or empty []).",
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(groupElementType(), []attr.Value{})),
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_id": schema.StringAttribute{
							Description: "The ID of the IdP group.",
							Required:    true,
						},
						"group_name": schema.StringAttribute{
							Description: "The name of the IdP group.",
							Required:    true,
						},
						"group_description": schema.StringAttribute{
							Description: "The description of the IdP group.",
							Required:    true,
						},
					},
				},
			},
			"etag": schema.StringAttribute{
				Description: "The etag of the team sync group mapping.",
				Computed:    true,
			},
		},
	}
}

func groupElementType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"group_id":          types.StringType,
			"group_name":        types.StringType,
			"group_description": types.StringType,
		},
	}
}

func (r *githubTeamSyncGroupMappingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubTeamSyncGroupMappingResource) checkOrganization() error {
	if !r.client.IsOrganization {
		return fmt.Errorf("this resource can only be used with organization accounts")
	}
	return nil
}

func (r *githubTeamSyncGroupMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubTeamSyncGroupMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate organization
	if err := r.checkOrganization(); err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts: %s", err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	orgName := r.client.Name()
	teamSlug := plan.TeamSlug.ValueString()

	// Convert plan groups to GitHub API format
	idpGroupList, diags := r.expandTeamSyncGroups(ctx, plan.Group)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create or update the team sync group mapping
	_, _, err := client.Teams.CreateOrUpdateIDPGroupConnectionsBySlug(ctx, orgName, teamSlug, *idpGroupList)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Team Sync Group Mapping",
			fmt.Sprintf("Could not create team sync group mapping for team %s: %s", teamSlug, err.Error()),
		)
		return
	}

	// Set the ID
	plan.ID = types.StringValue(fmt.Sprintf("teams/%s/team-sync/group-mappings", teamSlug))

	// Read the resource to get the current state
	r.readTeamSyncGroupMapping(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubTeamSyncGroupMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubTeamSyncGroupMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate organization
	if err := r.checkOrganization(); err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts: %s", err.Error()),
		)
		return
	}

	r.readTeamSyncGroupMapping(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubTeamSyncGroupMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubTeamSyncGroupMappingResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate organization
	if err := r.checkOrganization(); err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts: %s", err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	orgName := r.client.Name()
	teamSlug := plan.TeamSlug.ValueString()

	// Convert plan groups to GitHub API format
	idpGroupList, diags := r.expandTeamSyncGroups(ctx, plan.Group)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update the team sync group mapping
	_, _, err := client.Teams.CreateOrUpdateIDPGroupConnectionsBySlug(ctx, orgName, teamSlug, *idpGroupList)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Team Sync Group Mapping",
			fmt.Sprintf("Could not update team sync group mapping for team %s: %s", teamSlug, err.Error()),
		)
		return
	}

	// Read the resource to get the current state
	r.readTeamSyncGroupMapping(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubTeamSyncGroupMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubTeamSyncGroupMappingResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate organization
	if err := r.checkOrganization(); err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts: %s", err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	orgName := r.client.Name()
	teamSlug := state.TeamSlug.ValueString()

	// Create empty group list to remove all mappings
	groups := make([]*github.IDPGroup, 0)
	emptyGroupList := github.IDPGroupList{Groups: groups}

	_, _, err := client.Teams.CreateOrUpdateIDPGroupConnectionsBySlug(ctx, orgName, teamSlug, emptyGroupList)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Team Sync Group Mapping",
			fmt.Sprintf("Could not delete team sync group mapping for team %s: %s", teamSlug, err.Error()),
		)
		return
	}

	tflog.Info(ctx, "Team sync group mapping deleted", map[string]interface{}{
		"team_slug": teamSlug,
	})
}

func (r *githubTeamSyncGroupMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import using team slug
	teamSlug := req.ID

	// Set the team_slug attribute
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_slug"), teamSlug)...)
	
	// Set the ID
	id := fmt.Sprintf("teams/%s/team-sync/group-mappings", teamSlug)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func (r *githubTeamSyncGroupMappingResource) readTeamSyncGroupMapping(ctx context.Context, state *githubTeamSyncGroupMappingResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	orgName := r.client.Name()
	teamSlug := state.TeamSlug.ValueString()

	// Add context values for etag handling if available
	requestCtx := ctx
	if !state.Etag.IsNull() && !state.Etag.IsUnknown() {
		requestCtx = context.WithValue(ctx, CtxEtag, state.Etag.ValueString())
	}
	requestCtx = context.WithValue(requestCtx, CtxId, state.ID.ValueString())

	idpGroupList, resp, err := client.Teams.ListIDPGroupsForTeamBySlug(requestCtx, orgName, teamSlug)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "Team sync group mapping not found, removing from state", map[string]interface{}{
					"team_slug": teamSlug,
				})
				state.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Error Reading Team Sync Group Mapping",
			fmt.Sprintf("Could not read team sync group mapping for team %s: %s", teamSlug, err.Error()),
		)
		return
	}

	// Convert API response to framework types
	groups, conversionDiags := r.flattenGithubIDPGroupList(ctx, idpGroupList)
	diags.Append(conversionDiags...)
	if diags.HasError() {
		return
	}

	state.Group = groups
	if resp != nil && resp.Header.Get("ETag") != "" {
		state.Etag = types.StringValue(resp.Header.Get("ETag"))
	} else {
		state.Etag = types.StringNull()
	}
}

func (r *githubTeamSyncGroupMappingResource) expandTeamSyncGroups(ctx context.Context, groupsSet types.Set) (*github.IDPGroupList, diag.Diagnostics) {
	var diags diag.Diagnostics
	groups := make([]*github.IDPGroup, 0)

	if groupsSet.IsNull() || len(groupsSet.Elements()) == 0 {
		return &github.IDPGroupList{Groups: groups}, diags
	}

	var groupModels []githubTeamSyncGroupModel
	diags.Append(groupsSet.ElementsAs(ctx, &groupModels, false)...)
	if diags.HasError() {
		return nil, diags
	}

	for _, groupModel := range groupModels {
		group := &github.IDPGroup{
			GroupID:          github.Ptr(groupModel.GroupID.ValueString()),
			GroupName:        github.Ptr(groupModel.GroupName.ValueString()),
			GroupDescription: github.Ptr(groupModel.GroupDescription.ValueString()),
		}
		groups = append(groups, group)
	}

	return &github.IDPGroupList{Groups: groups}, diags
}

func (r *githubTeamSyncGroupMappingResource) flattenGithubIDPGroupList(ctx context.Context, idpGroupList *github.IDPGroupList) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	if idpGroupList == nil || len(idpGroupList.Groups) == 0 {
		return types.SetValueMust(groupElementType(), []attr.Value{}), diags
	}

	groupValues := make([]attr.Value, 0, len(idpGroupList.Groups))
	for _, group := range idpGroupList.Groups {
		groupObj := types.ObjectValueMust(
			groupElementType().AttrTypes,
			map[string]attr.Value{
				"group_id":          types.StringValue(group.GetGroupID()),
				"group_name":        types.StringValue(group.GetGroupName()),
				"group_description": types.StringValue(group.GetGroupDescription()),
			},
		)
		groupValues = append(groupValues, groupObj)
	}

	groupsSet, setDiags := types.SetValue(groupElementType(), groupValues)
	diags.Append(setDiags...)

	return groupsSet, diags
}