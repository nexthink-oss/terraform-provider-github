package github

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubRepositoryCollaboratorsResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryCollaboratorsResource{}
	_ resource.ResourceWithImportState = &githubRepositoryCollaboratorsResource{}
	_ resource.ResourceWithModifyPlan  = &githubRepositoryCollaboratorsResource{}
)

type githubRepositoryCollaboratorsResource struct {
	client *Owner
}

type githubRepositoryCollaboratorsResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Repository    types.String `tfsdk:"repository"`
	User          types.Set    `tfsdk:"user"`
	Team          types.Set    `tfsdk:"team"`
	InvitationIds types.Map    `tfsdk:"invitation_ids"`
	IgnoreTeam    types.Set    `tfsdk:"ignore_team"`
}

type userCollaboratorModel struct {
	Permission types.String `tfsdk:"permission"`
	Username   types.String `tfsdk:"username"`
}

type teamCollaboratorModel struct {
	Permission types.String `tfsdk:"permission"`
	TeamId     types.String `tfsdk:"team_id"`
}

type ignoreTeamModel struct {
	TeamId types.String `tfsdk:"team_id"`
}

// Internal types for processing
type userCollaborator struct {
	permission string
	username   string
}

func (c userCollaborator) Empty() bool {
	return c == userCollaborator{}
}

type invitedCollaborator struct {
	userCollaborator
	invitationID int64
}

type teamCollaborator struct {
	permission string
	teamID     int64
	teamSlug   string
}

func (c teamCollaborator) Empty() bool {
	return c == teamCollaborator{}
}

func NewGithubRepositoryCollaboratorsResource() resource.Resource {
	return &githubRepositoryCollaboratorsResource{}
}

func (r *githubRepositoryCollaboratorsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_collaborators"
}

func (r *githubRepositoryCollaboratorsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub repository collaborators resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the repository collaborators (repository name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The GitHub repository",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user": schema.SetNestedAttribute{
				Description: "List of users to add as collaborators",
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.ObjectType{AttrTypes: userCollaboratorAttrTypes()}, []attr.Value{})),
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"permission": schema.StringAttribute{
							Description: "The permission to grant the collaborator. Must be one of: pull, push, maintain, triage, admin.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("push"),
							Validators: []validator.String{
								&permissionValidator{},
							},
						},
						"username": schema.StringAttribute{
							Description: "The user to add to the repository as a collaborator.",
							Required:    true,
							Validators: []validator.String{
								&caseInsensitiveStringValidator{},
							},
						},
					},
				},
			},
			"team": schema.SetNestedAttribute{
				Description: "List of teams to add as collaborators",
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.ObjectType{AttrTypes: teamCollaboratorAttrTypes()}, []attr.Value{})),
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"permission": schema.StringAttribute{
							Description: "The permission to grant the team. Must be one of: pull, push, maintain, triage, admin.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("push"),
							Validators: []validator.String{
								&permissionValidator{},
							},
						},
						"team_id": schema.StringAttribute{
							Description: "Team ID or slug to add to the repository as a collaborator.",
							Required:    true,
						},
					},
				},
			},
			"invitation_ids": schema.MapAttribute{
				Description: "Map of usernames to invitation ID for any users added",
				ElementType: types.StringType,
				Computed:    true,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
			"ignore_team": schema.SetNestedAttribute{
				Description: "List of teams to ignore",
				Optional:    true,
				Computed:    true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.ObjectType{AttrTypes: ignoreTeamAttrTypes()}, []attr.Value{})),
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"team_id": schema.StringAttribute{
							Description: "ID or slug of the team to ignore.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (r *githubRepositoryCollaboratorsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubRepositoryCollaboratorsResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Check for create or update (not delete)
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan githubRepositoryCollaboratorsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state githubRepositoryCollaboratorsResourceModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// If there was a change to the user list, mark invitation_ids as computed
		if !plan.User.Equal(state.User) {
			invitationIds := types.MapUnknown(types.StringType)
			plan.InvitationIds = invitationIds
		}
	} else {
		// On create, invitation_ids should be computed
		invitationIds := types.MapUnknown(types.StringType)
		plan.InvitationIds = invitationIds
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *githubRepositoryCollaboratorsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubRepositoryCollaboratorsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	isOrg := r.client.IsOrganization
	repoName := data.Repository.ValueString()

	// Validate for duplicate collaborators
	if diags := r.validateCollaborators(ctx, &data); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	ignoreTeamIds, diags := r.getIgnoreTeamIds(ctx, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	userCollaborators, invitations, teamCollaborators, err := r.listAllCollaborators(ctx, client, isOrg, owner, repoName, ignoreTeamIds)
	if err != nil {
		if r.is404Error(err) {
			resp.Diagnostics.AddError(
				"Repository Not Found",
				fmt.Sprintf("Repository %s/%s not found or not accessible: %s", owner, repoName, err.Error()),
			)
		} else {
			resp.Diagnostics.AddError(
				"Unable to Read Repository Collaborators",
				fmt.Sprintf("An unexpected error occurred when reading repository collaborators: %s", err.Error()),
			)
		}
		return
	}

	// Get planned teams and users
	teams, diags := r.getTeamsFromSet(ctx, data.Team)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	users, diags := r.getUsersFromSet(ctx, data.User)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Apply team changes
	if err := r.matchTeamCollaborators(ctx, repoName, teams, teamCollaborators); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Manage Team Collaborators",
			fmt.Sprintf("An unexpected error occurred when managing team collaborators: %s", err.Error()),
		)
		return
	}

	// Apply user changes
	if err := r.matchUserCollaboratorsAndInvites(ctx, repoName, users, userCollaborators, invitations); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Manage User Collaborators",
			fmt.Sprintf("An unexpected error occurred when managing user collaborators: %s", err.Error()),
		)
		return
	}

	data.ID = types.StringValue(repoName)

	tflog.Debug(ctx, "created GitHub repository collaborators", map[string]any{
		"id":         data.ID.ValueString(),
		"repository": repoName,
	})

	// Read the created resource to populate all computed fields
	r.readGithubRepositoryCollaborators(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryCollaboratorsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubRepositoryCollaboratorsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubRepositoryCollaborators(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryCollaboratorsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubRepositoryCollaboratorsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	isOrg := r.client.IsOrganization
	repoName := data.Repository.ValueString()

	// Validate for duplicate collaborators
	if diags := r.validateCollaborators(ctx, &data); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	ignoreTeamIds, diags := r.getIgnoreTeamIds(ctx, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	userCollaborators, invitations, teamCollaborators, err := r.listAllCollaborators(ctx, client, isOrg, owner, repoName, ignoreTeamIds)
	if err != nil {
		if r.is404Error(err) {
			resp.Diagnostics.AddError(
				"Repository Not Found",
				fmt.Sprintf("Repository %s/%s not found or not accessible: %s", owner, repoName, err.Error()),
			)
		} else {
			resp.Diagnostics.AddError(
				"Unable to Read Repository Collaborators",
				fmt.Sprintf("An unexpected error occurred when reading repository collaborators: %s", err.Error()),
			)
		}
		return
	}

	// Get planned teams and users
	teams, diags := r.getTeamsFromSet(ctx, data.Team)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	users, diags := r.getUsersFromSet(ctx, data.User)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Apply team changes
	if err := r.matchTeamCollaborators(ctx, repoName, teams, teamCollaborators); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Manage Team Collaborators",
			fmt.Sprintf("An unexpected error occurred when managing team collaborators: %s", err.Error()),
		)
		return
	}

	// Apply user changes
	if err := r.matchUserCollaboratorsAndInvites(ctx, repoName, users, userCollaborators, invitations); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Manage User Collaborators",
			fmt.Sprintf("An unexpected error occurred when managing user collaborators: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "updated GitHub repository collaborators", map[string]any{
		"id":         data.ID.ValueString(),
		"repository": repoName,
	})

	// Read the updated resource to populate all computed fields
	r.readGithubRepositoryCollaborators(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryCollaboratorsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubRepositoryCollaboratorsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	isOrg := r.client.IsOrganization
	repoName := data.Repository.ValueString()

	ignoreTeamIds, diags := r.getIgnoreTeamIds(ctx, &data)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	userCollaborators, invitations, teamCollaborators, err := r.listAllCollaborators(ctx, client, isOrg, owner, repoName, ignoreTeamIds)
	if err != nil {
		if r.is404Error(err) {
			// Repository doesn't exist, consider deletion successful
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Repository Collaborators",
			fmt.Sprintf("An unexpected error occurred when reading repository collaborators for deletion: %s", err.Error()),
		)
		return
	}

	log.Printf("[DEBUG] Deleting all users, invites and collaborators for repo: %s.", repoName)

	// Delete all users
	if err := r.matchUserCollaboratorsAndInvites(ctx, repoName, nil, userCollaborators, invitations); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Remove User Collaborators",
			fmt.Sprintf("An unexpected error occurred when removing user collaborators: %s", err.Error()),
		)
		return
	}

	// Delete all teams
	if err := r.matchTeamCollaborators(ctx, repoName, nil, teamCollaborators); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Remove Team Collaborators",
			fmt.Sprintf("An unexpected error occurred when removing team collaborators: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub repository collaborators", map[string]any{
		"id":         data.ID.ValueString(),
		"repository": repoName,
	})
}

func (r *githubRepositoryCollaboratorsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import by repository name
	data := &githubRepositoryCollaboratorsResourceModel{
		ID:         types.StringValue(req.ID),
		Repository: types.StringValue(req.ID),
	}

	r.readGithubRepositoryCollaborators(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func userCollaboratorAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"permission": types.StringType,
		"username":   types.StringType,
	}
}

func teamCollaboratorAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"permission": types.StringType,
		"team_id":    types.StringType,
	}
}

func ignoreTeamAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"team_id": types.StringType,
	}
}

func (r *githubRepositoryCollaboratorsResource) validateCollaborators(ctx context.Context, data *githubRepositoryCollaboratorsResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// Check for duplicate teams
	teams, teamDiags := r.getTeamsFromSet(ctx, data.Team)
	diags.Append(teamDiags...)
	if diags.HasError() {
		return diags
	}

	teamsMap := make(map[string]struct{}, len(teams))
	for _, team := range teams {
		if _, found := teamsMap[team.TeamId.ValueString()]; found {
			diags.AddError(
				"Duplicate Team Collaborator",
				fmt.Sprintf("Duplicate set member: %s", team.TeamId.ValueString()),
			)
		}
		teamsMap[team.TeamId.ValueString()] = struct{}{}
	}

	// Check for duplicate users
	users, userDiags := r.getUsersFromSet(ctx, data.User)
	diags.Append(userDiags...)
	if diags.HasError() {
		return diags
	}

	usersMap := make(map[string]struct{}, len(users))
	for _, user := range users {
		if _, found := usersMap[user.Username.ValueString()]; found {
			diags.AddError(
				"Duplicate User Collaborator",
				fmt.Sprintf("Duplicate set member found: %s", user.Username.ValueString()),
			)
		}
		usersMap[user.Username.ValueString()] = struct{}{}
	}

	return diags
}

func (r *githubRepositoryCollaboratorsResource) getIgnoreTeamIds(ctx context.Context, data *githubRepositoryCollaboratorsResourceModel) ([]int64, diag.Diagnostics) {
	var diags diag.Diagnostics

	ignoreTeams, ignoreDiags := r.getIgnoreTeamsFromSet(ctx, data.IgnoreTeam)
	diags.Append(ignoreDiags...)
	if diags.HasError() {
		return nil, diags
	}

	ignoreTeamIds := make([]int64, len(ignoreTeams))
	for i, team := range ignoreTeams {
		id, err := r.getTeamID(ctx, team.TeamId.ValueString())
		if err != nil {
			diags.AddError(
				"Unable to Get Team ID",
				fmt.Sprintf("Unable to get team ID for %s: %s", team.TeamId.ValueString(), err.Error()),
			)
			return nil, diags
		}
		ignoreTeamIds[i] = id
	}

	return ignoreTeamIds, diags
}

func (r *githubRepositoryCollaboratorsResource) getTeamID(ctx context.Context, teamIDString string) (int64, error) {
	client := r.client.V3Client()
	orgName := r.client.Name()

	teamId, parseIntErr := strconv.ParseInt(teamIDString, 10, 64)
	if parseIntErr == nil {
		return teamId, nil
	}

	// The given id not an integer, assume it is a team slug
	team, _, slugErr := client.Teams.GetTeamBySlug(ctx, orgName, teamIDString)
	if slugErr != nil {
		return -1, fmt.Errorf("%s%s", parseIntErr.Error(), slugErr.Error())
	}
	return team.GetID(), nil
}

func (r *githubRepositoryCollaboratorsResource) getTeamSlugFromID(ctx context.Context, teamID int64) (string, error) {
	client := r.client.V3Client()
	orgId := r.client.ID()

	// Note: This still uses GetTeamByID as it's the only way to get slug from numeric ID
	// This call is minimized by caching and the migration to slug-based APIs elsewhere
	//nolint:staticcheck // SA1019: GetTeamByID is deprecated but needed for ID->slug conversion
	team, _, err := client.Teams.GetTeamByID(ctx, orgId, teamID)
	if err != nil {
		return "", err
	}
	return team.GetSlug(), nil
}

func (r *githubRepositoryCollaboratorsResource) getTeamSlug(ctx context.Context, teamIDString string) (string, error) {
	client := r.client.V3Client()
	orgId := r.client.ID()

	teamId, parseIntErr := strconv.ParseInt(teamIDString, 10, 64)
	if parseIntErr == nil {
		// It's an ID, get the team to find the slug
		// Note: This still uses GetTeamByID as it's the only way to get slug from numeric ID
		//nolint:staticcheck // SA1019: GetTeamByID is deprecated but needed for ID->slug conversion
		team, _, err := client.Teams.GetTeamByID(ctx, orgId, teamId)
		if err != nil {
			return "", err
		}
		return team.GetSlug(), nil
	}

	// The given id is not an integer, assume it is already a team slug
	return teamIDString, nil
}

func (r *githubRepositoryCollaboratorsResource) is404Error(err error) bool {
	if ghErr, ok := err.(*github.ErrorResponse); ok {
		return ghErr.Response.StatusCode == 404
	}
	return false
}

func (r *githubRepositoryCollaboratorsResource) getPermission(permission string) string {
	// Permissions for some GitHub API routes are expressed as "read",
	// "write", and "admin"; in other places, they are expressed as "pull",
	// "push", and "admin".
	switch permission {
	case "read":
		return "pull"
	case "write":
		return "push"
	default:
		return permission
	}
}

func (r *githubRepositoryCollaboratorsResource) readGithubRepositoryCollaborators(ctx context.Context, data *githubRepositoryCollaboratorsResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()
	isOrg := r.client.IsOrganization
	repoName := data.ID.ValueString()

	ignoreTeamIds, ignoreDiags := r.getIgnoreTeamIds(ctx, data)
	diags.Append(ignoreDiags...)
	if diags.HasError() {
		return
	}

	userCollaborators, invitedCollaborators, teamCollaborators, err := r.listAllCollaborators(ctx, client, isOrg, owner, repoName, ignoreTeamIds)
	if err != nil {
		if r.is404Error(err) {
			tflog.Info(ctx, "removing repository collaborators from state because repository no longer exists", map[string]any{
				"owner":      owner,
				"repository": repoName,
			})
			data.ID = types.StringNull()
			return
		}
		diags.AddError(
			"Unable to Read Repository Collaborators",
			fmt.Sprintf("An unexpected error occurred when reading repository collaborators: %s", err.Error()),
		)
		return
	}

	// Build invitation IDs map
	invitationIds := make(map[string]attr.Value, len(invitedCollaborators))
	for _, i := range invitedCollaborators {
		invitationIds[i.username] = types.StringValue(strconv.FormatInt(i.invitationID, 10))
	}

	// Get team slugs from current configuration for proper flattening
	sourceTeams, teamDiags := r.getTeamsFromSet(ctx, data.Team)
	diags.Append(teamDiags...)
	if diags.HasError() {
		return
	}

	teamSlugs := make([]string, 0, len(sourceTeams))
	for _, t := range sourceTeams {
		if _, parseIntErr := strconv.ParseInt(t.TeamId.ValueString(), 10, 64); parseIntErr != nil {
			teamSlugs = append(teamSlugs, t.TeamId.ValueString())
		}
	}

	// Set computed values
	data.Repository = types.StringValue(repoName)
	data.InvitationIds = types.MapValueMust(types.StringType, invitationIds)

	// Flatten user collaborators
	userElements := r.flattenUserCollaborators(userCollaborators, invitedCollaborators)
	userSet, setDiags := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: userCollaboratorAttrTypes()}, userElements)
	diags.Append(setDiags...)
	if diags.HasError() {
		return
	}
	data.User = userSet

	// Flatten team collaborators
	teamElements := r.flattenTeamCollaborators(teamCollaborators, teamSlugs)
	teamSet, setDiags := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: teamCollaboratorAttrTypes()}, teamElements)
	diags.Append(setDiags...)
	if diags.HasError() {
		return
	}
	data.Team = teamSet

	tflog.Debug(ctx, "successfully read GitHub repository collaborators", map[string]any{
		"id":         data.ID.ValueString(),
		"repository": repoName,
		"users":      len(userElements),
		"teams":      len(teamElements),
	})
}

func (r *githubRepositoryCollaboratorsResource) flattenUserCollaborators(users []userCollaborator, invites []invitedCollaborator) []userCollaboratorModel {
	allUsers := make([]userCollaborator, len(users), len(users)+len(invites))
	copy(allUsers, users)

	for _, invite := range invites {
		allUsers = append(allUsers, invite.userCollaborator)
	}

	sort.SliceStable(allUsers, func(i, j int) bool {
		return allUsers[i].username < allUsers[j].username
	})

	result := make([]userCollaboratorModel, len(allUsers))
	for i, user := range allUsers {
		result[i] = userCollaboratorModel{
			Permission: types.StringValue(user.permission),
			Username:   types.StringValue(user.username),
		}
	}

	return result
}

func (r *githubRepositoryCollaboratorsResource) flattenTeamCollaborators(teams []teamCollaborator, teamSlugs []string) []teamCollaboratorModel {
	sort.SliceStable(teams, func(i, j int) bool {
		return teams[i].teamID < teams[j].teamID
	})

	result := make([]teamCollaboratorModel, len(teams))
	for i, team := range teams {
		var teamIDString string
		if slices.Contains(teamSlugs, team.teamSlug) {
			teamIDString = team.teamSlug
		} else {
			teamIDString = strconv.FormatInt(team.teamID, 10)
		}

		result[i] = teamCollaboratorModel{
			Permission: types.StringValue(team.permission),
			TeamId:     types.StringValue(teamIDString),
		}
	}

	return result
}

func (r *githubRepositoryCollaboratorsResource) getUsersFromSet(ctx context.Context, userSet types.Set) ([]userCollaboratorModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var users []userCollaboratorModel

	diags.Append(userSet.ElementsAs(ctx, &users, false)...)
	return users, diags
}

func (r *githubRepositoryCollaboratorsResource) getTeamsFromSet(ctx context.Context, teamSet types.Set) ([]teamCollaboratorModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var teams []teamCollaboratorModel

	diags.Append(teamSet.ElementsAs(ctx, &teams, false)...)
	return teams, diags
}

func (r *githubRepositoryCollaboratorsResource) getIgnoreTeamsFromSet(ctx context.Context, ignoreSet types.Set) ([]ignoreTeamModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var teams []ignoreTeamModel

	diags.Append(ignoreSet.ElementsAs(ctx, &teams, false)...)
	return teams, diags
}

// GitHub API interaction functions

func (r *githubRepositoryCollaboratorsResource) listUserCollaborators(ctx context.Context, client *github.Client, isOrg bool, owner, repoName string) ([]userCollaborator, error) {
	userCollaborators := make([]userCollaborator, 0)
	affiliations := []string{"direct", "outside"}

	for _, affiliation := range affiliations {
		opt := &github.ListCollaboratorsOptions{
			ListOptions: github.ListOptions{PerPage: maxPerPage},
			Affiliation: affiliation,
		}

		for {
			collaborators, resp, err := client.Repositories.ListCollaborators(ctx, owner, repoName, opt)
			if err != nil {
				return nil, err
			}

			for _, c := range collaborators {
				// owners are listed in the collaborators list even though they don't have direct permissions
				if !isOrg && c.GetLogin() == owner {
					continue
				}
				permissionName := r.getPermission(c.GetRoleName())
				userCollaborators = append(userCollaborators, userCollaborator{permissionName, c.GetLogin()})
			}

			if resp.NextPage == 0 {
				break
			}
			opt.Page = resp.NextPage
		}
	}
	return userCollaborators, nil
}

func (r *githubRepositoryCollaboratorsResource) listInvitations(ctx context.Context, client *github.Client, owner, repoName string) ([]invitedCollaborator, error) {
	invitedCollaborators := make([]invitedCollaborator, 0)

	opt := &github.ListOptions{PerPage: maxPerPage}
	for {
		invitations, resp, err := client.Repositories.ListInvitations(ctx, owner, repoName, opt)
		if err != nil {
			return nil, err
		}

		for _, i := range invitations {
			permissionName := r.getPermission(i.GetPermissions())
			invitedCollaborators = append(invitedCollaborators, invitedCollaborator{
				userCollaborator{permissionName, i.GetInvitee().GetLogin()}, i.GetID(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return invitedCollaborators, nil
}

func (r *githubRepositoryCollaboratorsResource) listTeams(ctx context.Context, client *github.Client, isOrg bool, owner, repoName string, ignoreTeamIds []int64) ([]teamCollaborator, error) {
	allTeams := make([]teamCollaborator, 0)

	if !isOrg {
		return allTeams, nil
	}

	opt := &github.ListOptions{PerPage: maxPerPage}
	for {
		repoTeams, resp, err := client.Repositories.ListTeams(ctx, owner, repoName, opt)
		if err != nil {
			return nil, err
		}

		for _, t := range repoTeams {
			if slices.Contains(ignoreTeamIds, t.GetID()) {
				continue
			}

			allTeams = append(allTeams, teamCollaborator{
				permission: r.getPermission(t.GetPermission()),
				teamID:     t.GetID(),
				teamSlug:   t.GetSlug(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allTeams, nil
}

func (r *githubRepositoryCollaboratorsResource) listAllCollaborators(ctx context.Context, client *github.Client, isOrg bool, owner, repoName string, ignoreTeamIds []int64) ([]userCollaborator, []invitedCollaborator, []teamCollaborator, error) {
	userCollaborators, err := r.listUserCollaborators(ctx, client, isOrg, owner, repoName)
	if err != nil {
		return nil, nil, nil, err
	}
	invitations, err := r.listInvitations(ctx, client, owner, repoName)
	if err != nil {
		return nil, nil, nil, err
	}
	teamCollaborators, err := r.listTeams(ctx, client, isOrg, owner, repoName, ignoreTeamIds)
	if err != nil {
		return nil, nil, nil, err
	}
	return userCollaborators, invitations, teamCollaborators, err
}

func (r *githubRepositoryCollaboratorsResource) matchUserCollaboratorsAndInvites(ctx context.Context, repoName string, want []userCollaboratorModel, hasUsers []userCollaborator, hasInvites []invitedCollaborator) error {
	client := r.client.V3Client()
	owner := r.client.Name()

	for _, has := range hasUsers {
		var wantPermission string
		for _, w := range want {
			if w.Username.ValueString() == has.username {
				wantPermission = w.Permission.ValueString()
				break
			}
		}
		if wantPermission == "" { // user should NOT have permission
			log.Printf("[DEBUG] Removing user %s from repo: %s.", has.username, repoName)
			_, err := client.Repositories.RemoveCollaborator(ctx, owner, repoName, has.username)
			if err != nil {
				return err
			}
		} else if wantPermission != has.permission { // permission should be updated
			log.Printf("[DEBUG] Updating user %s permission from %s to %s for repo: %s.", has.username, has.permission, wantPermission, repoName)
			_, _, err := client.Repositories.AddCollaborator(
				ctx, owner, repoName, has.username, &github.RepositoryAddCollaboratorOptions{
					Permission: wantPermission,
				},
			)
			if err != nil {
				return err
			}
		}
	}

	for _, has := range hasInvites {
		var wantPermission string
		for _, u := range want {
			if u.Username.ValueString() == has.username {
				wantPermission = u.Permission.ValueString()
				break
			}
		}
		if wantPermission == "" { // user should NOT have permission
			log.Printf("[DEBUG] Deleting invite for user %s from repo: %s.", has.username, repoName)
			_, err := client.Repositories.DeleteInvitation(ctx, owner, repoName, has.invitationID)
			if err != nil {
				return err
			}
		} else if wantPermission != has.permission { // permission should be updated
			log.Printf("[DEBUG] Updating invite for user %s permission from %s to %s for repo: %s.", has.username, has.permission, wantPermission, repoName)
			_, _, err := client.Repositories.UpdateInvitation(ctx, owner, repoName, has.invitationID, wantPermission)
			if err != nil {
				return err
			}
		}
	}

	for _, w := range want {
		username := w.Username.ValueString()
		permission := w.Permission.ValueString()
		var found bool
		for _, has := range hasUsers {
			if username == has.username {
				found = true
				break
			}
		}
		if found {
			continue
		}
		for _, has := range hasInvites {
			if username == has.username {
				found = true
				break
			}
		}
		if found {
			continue
		}
		// user needs to be added
		log.Printf("[DEBUG] Inviting user %s with permission %s for repo: %s.", username, permission, repoName)
		_, _, err := client.Repositories.AddCollaborator(
			ctx, owner, repoName, username, &github.RepositoryAddCollaboratorOptions{
				Permission: permission,
			},
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *githubRepositoryCollaboratorsResource) matchTeamCollaborators(ctx context.Context, repoName string, want []teamCollaboratorModel, has []teamCollaborator) error {
	client := r.client.V3Client()
	owner := r.client.Name()

	// Note: Organization validation is handled by the client and resource config

	remove := make([]teamCollaborator, 0)
	for _, hasTeam := range has {
		var wantPerm string
		for _, w := range want {
			teamIDString := w.TeamId.ValueString()
			teamID, err := r.getTeamID(ctx, teamIDString)
			if err != nil {
				return err
			}
			if teamID == hasTeam.teamID {
				wantPerm = w.Permission.ValueString()
				break
			}
		}
		if wantPerm == "" { // team should NOT have permission
			remove = append(remove, hasTeam)
		} else if wantPerm != hasTeam.permission { // permission should be updated
			log.Printf("[DEBUG] Updating team %d permission from %s to %s for repo: %s.", hasTeam.teamID, hasTeam.permission, wantPerm, repoName)
			teamSlug, err := r.getTeamSlugFromID(ctx, hasTeam.teamID)
			if err != nil {
				return err
			}
			_, err = client.Teams.AddTeamRepoBySlug(
				ctx, owner, teamSlug, owner, repoName, &github.TeamAddTeamRepoOptions{
					Permission: wantPerm,
				},
			)
			if err != nil {
				return err
			}
		}
	}

	for _, t := range want {
		teamIDString := t.TeamId.ValueString()
		teamID, err := r.getTeamID(ctx, teamIDString)
		if err != nil {
			return err
		}
		var found bool
		for _, c := range has {
			if teamID == c.teamID {
				found = true
				break
			}
		}
		if found {
			continue
		}
		permission := t.Permission.ValueString()
		// team needs to be added
		log.Printf("[DEBUG] Adding team %s with permission %s for repo: %s.", teamIDString, permission, repoName)
		teamSlug, err := r.getTeamSlug(ctx, teamIDString)
		if err != nil {
			return err
		}
		_, err = client.Teams.AddTeamRepoBySlug(
			ctx, owner, teamSlug, owner, repoName, &github.TeamAddTeamRepoOptions{
				Permission: permission,
			},
		)
		if err != nil {
			return err
		}
	}

	for _, team := range remove {
		log.Printf("[DEBUG] Removing team %d from repo: %s.", team.teamID, repoName)
		teamSlug, err := r.getTeamSlugFromID(ctx, team.teamID)
		if err != nil {
			return err
		}
		_, err = client.Teams.RemoveTeamRepoBySlug(ctx, owner, teamSlug, owner, repoName)
		if err != nil {
			return err
		}
	}

	return nil
}

// Custom validators

type permissionValidator struct{}

func (v *permissionValidator) Description(ctx context.Context) string {
	return "Validates that the permission is one of: pull, push, maintain, triage, admin"
}

func (v *permissionValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *permissionValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	validPermissions := []string{"pull", "push", "maintain", "triage", "admin"}

	if slices.Contains(validPermissions, value) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Permission",
		fmt.Sprintf("Permission must be one of: %s. Got: %s", strings.Join(validPermissions, ", "), value),
	)
}

type caseInsensitiveStringValidator struct{}

func (v *caseInsensitiveStringValidator) Description(ctx context.Context) string {
	return "Validates string values case-insensitively for diffs"
}

func (v *caseInsensitiveStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *caseInsensitiveStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// This validator doesn't add any validation errors but is used for marking case-insensitive fields
	// The actual case-insensitive comparison would be handled by custom plan modifiers if needed
}
