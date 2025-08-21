package github

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/shurcooL/githubv4"
)

var (
	_ resource.Resource                = &githubTeamMembersResource{}
	_ resource.ResourceWithConfigure   = &githubTeamMembersResource{}
	_ resource.ResourceWithImportState = &githubTeamMembersResource{}
)

type githubTeamMembersResource struct {
	client *Owner
}

type githubTeamMembersResourceModel struct {
	ID      types.String `tfsdk:"id"`
	TeamID  types.String `tfsdk:"team_id"`
	Members types.Set    `tfsdk:"members"`
}

type teamMemberModel struct {
	Username types.String `tfsdk:"username"`
	Role     types.String `tfsdk:"role"`
}

type MemberChange struct {
	Old, New map[string]any
}

func NewGithubTeamMembersResource() resource.Resource {
	return &githubTeamMembersResource{}
}

func (r *githubTeamMembersResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_members"
}

func (r *githubTeamMembersResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides an authoritative GitHub team members resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The team ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_id": schema.StringAttribute{
				Description: "The GitHub team id or slug",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"members": schema.SetNestedAttribute{
				Description: "List of team members.",
				Required:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							Description: "The user to add to the team.",
							Required:    true,
						},
						"role": schema.StringAttribute{
							Description: "The role of the user within the team. Must be one of 'member' or 'maintainer'.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("member"),
							Validators: []validator.String{
								stringvalidator.OneOf("member", "maintainer"),
							},
						},
					},
				},
			},
		},
	}
}

func (r *githubTeamMembersResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubTeamMembersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubTeamMembersResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamIdString := plan.TeamID.ValueString()
	teamId, err := r.getTeamID(teamIdString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Getting Team ID",
			fmt.Sprintf("Could not get team ID for %s: %s", teamIdString, err.Error()),
		)
		return
	}

	// Get the members from the plan
	var members []teamMemberModel
	resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &members, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	orgId := r.client.ID()

	for _, member := range members {
		username := member.Username.ValueString()
		role := member.Role.ValueString()

		tflog.Debug(ctx, "Creating team membership", map[string]any{
			"team_id":  teamIdString,
			"username": username,
			"role":     role,
		})

		_, _, err = client.Teams.AddTeamMembershipByID(ctx,
			orgId,
			teamId,
			username,
			&github.TeamAddTeamMembershipOptions{
				Role: role,
			},
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating Team Membership",
				fmt.Sprintf("Could not create team membership for %s/%s: %s", teamIdString, username, err.Error()),
			)
			return
		}
	}

	// Set the ID to the team ID
	plan.ID = types.StringValue(teamIdString)
	plan.TeamID = types.StringValue(teamIdString)

	// Read the current state
	state := plan
	r.read(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *githubTeamMembersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubTeamMembersResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.read(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *githubTeamMembersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state githubTeamMembersResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamIdString := plan.TeamID.ValueString()
	teamId, err := r.getTeamID(teamIdString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Getting Team ID",
			fmt.Sprintf("Could not get team ID for %s: %s", teamIdString, err.Error()),
		)
		return
	}

	// Get old and new members
	var oldMembers, newMembers []teamMemberModel
	resp.Diagnostics.Append(state.Members.ElementsAs(ctx, &oldMembers, false)...)
	resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &newMembers, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build change map
	vals := make(map[string]*MemberChange)
	for _, member := range oldMembers {
		username := member.Username.ValueString()
		vals[username] = &MemberChange{
			Old: map[string]any{
				"username": username,
				"role":     member.Role.ValueString(),
			},
		}
	}
	for _, member := range newMembers {
		username := member.Username.ValueString()
		if _, ok := vals[username]; !ok {
			vals[username] = &MemberChange{}
		}
		vals[username].New = map[string]any{
			"username": username,
			"role":     member.Role.ValueString(),
		}
	}

	client := r.client.V3Client()
	orgId := r.client.ID()

	for username, change := range vals {
		var create, delete bool

		switch {
		// create a new one if old is nil
		case change.Old == nil:
			create = true
		// delete existing if new is nil
		case change.New == nil:
			delete = true
		// no change
		case reflect.DeepEqual(change.Old, change.New):
			continue
		// recreate - role changed
		default:
			delete = true
			create = true
		}

		if delete {
			tflog.Debug(ctx, "Deleting team membership", map[string]any{
				"team_id":  teamIdString,
				"username": username,
			})

			_, err = client.Teams.RemoveTeamMembershipByID(ctx, orgId, teamId, username)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Deleting Team Membership",
					fmt.Sprintf("Could not delete team membership for %s/%s: %s", teamIdString, username, err.Error()),
				)
				return
			}
		}

		if create {
			role := change.New["role"].(string)

			tflog.Debug(ctx, "Creating team membership", map[string]any{
				"team_id":  teamIdString,
				"username": username,
				"role":     role,
			})

			_, _, err = client.Teams.AddTeamMembershipByID(ctx,
				orgId,
				teamId,
				username,
				&github.TeamAddTeamMembershipOptions{
					Role: role,
				},
			)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Creating Team Membership",
					fmt.Sprintf("Could not create team membership for %s/%s: %s", teamIdString, username, err.Error()),
				)
				return
			}
		}
	}

	// Read the updated state
	newState := plan
	r.read(ctx, &newState, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *githubTeamMembersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubTeamMembersResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	teamIdString := state.TeamID.ValueString()
	teamId, err := r.getTeamID(teamIdString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Getting Team ID",
			fmt.Sprintf("Could not get team ID for %s: %s", teamIdString, err.Error()),
		)
		return
	}

	// Get the members from the state
	var members []teamMemberModel
	resp.Diagnostics.Append(state.Members.ElementsAs(ctx, &members, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	orgId := r.client.ID()

	for _, member := range members {
		username := member.Username.ValueString()

		tflog.Debug(ctx, "Deleting team membership", map[string]any{
			"team_id":  teamIdString,
			"username": username,
		})

		_, err = client.Teams.RemoveTeamMembershipByID(ctx, orgId, teamId, username)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Deleting Team Membership",
				fmt.Sprintf("Could not delete team membership for %s/%s: %s", teamIdString, username, err.Error()),
			)
			return
		}
	}
}

func (r *githubTeamMembersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamId, err := r.getTeamID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Team Members",
			fmt.Sprintf("Could not get team ID for %s: %s", req.ID, err.Error()),
		)
		return
	}

	// Set the ID to the team ID string
	teamIdString := strconv.FormatInt(teamId, 10)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(teamIdString))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), types.StringValue(teamIdString))...)
}

func (r *githubTeamMembersResource) read(ctx context.Context, state *githubTeamMembersResourceModel, diags *diag.Diagnostics) {
	teamIdString := state.TeamID.ValueString()
	if teamIdString == "" {
		diags.AddError("Missing Team ID", "Team ID is required but was empty")
		return
	}

	teamSlug, err := r.getTeamSlug(teamIdString)
	if err != nil {
		diags.AddError(
			"Error Getting Team Slug",
			fmt.Sprintf("Could not get team slug for %s: %s", teamIdString, err.Error()),
		)
		return
	}

	client := r.client.V4Client()
	orgName := r.client.Name()

	tflog.Debug(ctx, "Reading team members", map[string]any{
		"team_id": teamIdString,
	})

	var q struct {
		Organization struct {
			Team struct {
				Members struct {
					Edges []struct {
						Node struct {
							Login string
						}
						Role string
					}
					PageInfo struct {
						EndCursor   githubv4.String
						HasNextPage bool
					}
				} `graphql:"members(membership:IMMEDIATE, first:100, after: $after)"`
			} `graphql:"team(slug:$teamSlug)"`
		} `graphql:"organization(login:$orgName)"`
	}

	variables := map[string]any{
		"teamSlug": githubv4.String(teamSlug),
		"orgName":  githubv4.String(orgName),
		"after":    (*githubv4.String)(nil),
	}

	var teamMembersAndMaintainers []teamMemberModel
	for {
		if err := client.Query(ctx, &q, variables); err != nil {
			diags.AddError(
				"Error Reading Team Members",
				fmt.Sprintf("Could not read team members for %s: %s", teamIdString, err.Error()),
			)
			return
		}

		// Add all members to the list
		for _, member := range q.Organization.Team.Members.Edges {
			teamMembersAndMaintainers = append(teamMembersAndMaintainers, teamMemberModel{
				Username: types.StringValue(member.Node.Login),
				Role:     types.StringValue(strings.ToLower(member.Role)),
			})
		}
		if !q.Organization.Team.Members.PageInfo.HasNextPage {
			break
		}
		variables["after"] = githubv4.NewString(q.Organization.Team.Members.PageInfo.EndCursor)
	}

	// Convert to set
	memberObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"username": types.StringType,
			"role":     types.StringType,
		},
	}

	var memberValues []attr.Value
	for _, member := range teamMembersAndMaintainers {
		memberValue, memberDiags := types.ObjectValue(memberObjectType.AttrTypes, map[string]attr.Value{
			"username": member.Username,
			"role":     member.Role,
		})
		diags.Append(memberDiags...)
		if diags.HasError() {
			return
		}
		memberValues = append(memberValues, memberValue)
	}

	membersSet, setDiags := types.SetValue(memberObjectType, memberValues)
	diags.Append(setDiags...)
	if diags.HasError() {
		return
	}

	state.ID = types.StringValue(teamIdString)
	state.Members = membersSet
}

// Helper functions
func (r *githubTeamMembersResource) getTeamID(teamIDString string) (int64, error) {
	// Given a string that is either a team id or team slug, return the
	// id of the team it is referring to.
	teamId, parseIntErr := strconv.ParseInt(teamIDString, 10, 64)
	if parseIntErr == nil {
		return teamId, nil
	}

	// The given id not an integer, assume it is a team slug
	team, _, slugErr := r.client.V3Client().Teams.GetTeamBySlug(context.Background(), r.client.Name(), teamIDString)
	if slugErr != nil {
		return -1, fmt.Errorf("%s%s", parseIntErr.Error(), slugErr.Error())
	}
	return team.GetID(), nil
}

func (r *githubTeamMembersResource) getTeamSlug(teamIDString string) (string, error) {
	// Given a string that is either a team id or team slug, return the
	// team slug it is referring to.
	teamId, parseIntErr := strconv.ParseInt(teamIDString, 10, 64)
	if parseIntErr == nil {
		// The given id is an integer, so we need to get the slug from the API
		//nolint:staticcheck // SA1019: GetTeamByID is deprecated but needed for ID->slug conversion
		team, _, err := r.client.V3Client().Teams.GetTeamByID(context.Background(), r.client.ID(), teamId)
		if err != nil {
			return "", err
		}
		return team.GetSlug(), nil
	}

	// The given id is not an integer, assume it is a team slug
	return teamIDString, nil
}
