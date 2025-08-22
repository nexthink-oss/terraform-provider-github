package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
)

var (
	_ resource.Resource                = &githubTeamMembershipResource{}
	_ resource.ResourceWithConfigure   = &githubTeamMembershipResource{}
	_ resource.ResourceWithImportState = &githubTeamMembershipResource{}
)

func NewGithubTeamMembershipResource() resource.Resource {
	return &githubTeamMembershipResource{}
}

type githubTeamMembershipResource struct {
	client *Owner
}

type githubTeamMembershipResourceModel struct {
	ID       types.String `tfsdk:"id"`
	TeamID   types.String `tfsdk:"team_id"`
	Username types.String `tfsdk:"username"`
	Role     types.String `tfsdk:"role"`
	Etag     types.String `tfsdk:"etag"`
}

func (r *githubTeamMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_membership"
}

func (r *githubTeamMembershipResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub team membership resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the team membership.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_id": schema.StringAttribute{
				Description: "The GitHub team id or the GitHub team slug.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description: "The user to add to the team.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					&caseInsensitiveStringPlanModifier{},
				},
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
			"etag": schema.StringAttribute{
				Description: "The etag for the team membership.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubTeamMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *githubTeamMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubTeamMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	orgID := r.client.ID()

	teamIDString := plan.TeamID.ValueString()
	username := plan.Username.ValueString()
	role := plan.Role.ValueString()

	teamID, err := r.getTeamID(teamIDString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get team ID",
			fmt.Sprintf("Could not resolve team ID %s: %s", teamIDString, err.Error()),
		)
		return
	}

	_, _, err = client.Teams.AddTeamMembershipByID(ctx,
		orgID,
		teamID,
		username,
		&github.TeamAddTeamMembershipOptions{
			Role: role,
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create team membership",
			fmt.Sprintf("Could not create team membership for user %s on team %s: %s", username, teamIDString, err.Error()),
		)
		return
	}

	// Set ID using the same format as SDKv2 version
	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", teamIDString, username))

	tflog.Trace(ctx, "Created team membership", map[string]any{
		"team_id":  teamIDString,
		"username": username,
		"role":     role,
	})

	// Read the created resource to populate computed fields
	r.read(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubTeamMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubTeamMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.read(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubTeamMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubTeamMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	orgID := r.client.ID()

	teamIDString := plan.TeamID.ValueString()
	username := plan.Username.ValueString()
	role := plan.Role.ValueString()

	teamID, err := r.getTeamID(teamIDString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get team ID",
			fmt.Sprintf("Could not resolve team ID %s: %s", teamIDString, err.Error()),
		)
		return
	}

	_, _, err = client.Teams.AddTeamMembershipByID(ctx,
		orgID,
		teamID,
		username,
		&github.TeamAddTeamMembershipOptions{
			Role: role,
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update team membership",
			fmt.Sprintf("Could not update team membership for user %s on team %s: %s", username, teamIDString, err.Error()),
		)
		return
	}

	tflog.Trace(ctx, "Updated team membership", map[string]any{
		"team_id":  teamIDString,
		"username": username,
		"role":     role,
	})

	// Read the updated resource to populate computed fields
	r.read(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubTeamMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubTeamMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	orgID := r.client.ID()

	teamIDString := state.TeamID.ValueString()
	username := state.Username.ValueString()

	teamID, err := r.getTeamID(teamIDString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get team ID",
			fmt.Sprintf("Could not resolve team ID %s: %s", teamIDString, err.Error()),
		)
		return
	}

	_, err = client.Teams.RemoveTeamMembershipByID(ctx, orgID, teamID, username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete team membership",
			fmt.Sprintf("Could not delete team membership for user %s on team %s: %s", username, teamIDString, err.Error()),
		)
		return
	}

	tflog.Trace(ctx, "Deleted team membership", map[string]any{
		"team_id":  teamIDString,
		"username": username,
	})
}

func (r *githubTeamMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamIDString, username, err := r.parseTwoPartID(req.ID, "team_id", "username")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Could not parse import ID %s: %s", req.ID, err.Error()),
		)
		return
	}

	teamID, err := r.getTeamID(teamIDString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get team ID",
			fmt.Sprintf("Could not resolve team ID %s: %s", teamIDString, err.Error()),
		)
		return
	}

	// Set the ID using the resolved team ID (as integer string)
	importID := fmt.Sprintf("%d:%s", teamID, username)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), importID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), teamIDString)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), username)...)
}

// Helper methods

func (r *githubTeamMembershipResource) read(ctx context.Context, state *githubTeamMembershipResourceModel, diags *diag.Diagnostics) {
	teamIDString, username, err := r.parseTwoPartID(state.ID.ValueString(), "team_id", "username")
	if err != nil {
		diags.AddError(
			"Invalid state ID",
			fmt.Sprintf("Could not parse state ID %s: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	teamID, err := r.getTeamID(teamIDString)
	if err != nil {
		diags.AddError(
			"Failed to get team ID",
			fmt.Sprintf("Could not resolve team ID %s: %s", teamIDString, err.Error()),
		)
		return
	}

	// Set these early to allow reconciliation from upstream bugs
	state.TeamID = types.StringValue(teamIDString)
	state.Username = types.StringValue(username)

	client := r.client.V3Client()
	orgID := r.client.ID()

	// Add etag context for conditional requests
	requestCtx := context.WithValue(ctx, CtxId, state.ID.ValueString())
	if !state.Etag.IsNull() && !state.Etag.IsUnknown() {
		requestCtx = context.WithValue(requestCtx, CtxEtag, state.Etag.ValueString())
	}

	membership, resp, err := client.Teams.GetTeamMembershipByID(requestCtx, orgID, teamID, username)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "Team membership not found, removing from state", map[string]any{
					"team_id":  teamIDString,
					"username": username,
				})
				state.ID = types.StringValue("")
				return
			}
		}
		diags.AddError(
			"Failed to read team membership",
			fmt.Sprintf("Could not read team membership for user %s on team %s: %s", username, teamIDString, err.Error()),
		)
		return
	}

	if resp != nil && resp.Header != nil {
		state.Etag = types.StringValue(resp.Header.Get("ETag"))
	} else {
		state.Etag = types.StringNull()
	}

	if membership.Role != nil {
		state.Role = types.StringValue(*membership.Role)
	} else {
		state.Role = types.StringValue("member")
	}

	tflog.Trace(ctx, "Read team membership", map[string]any{
		"team_id":  teamIDString,
		"username": username,
		"role":     state.Role.ValueString(),
	})
}

func (r *githubTeamMembershipResource) parseTwoPartID(id, left, right string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected ID format (%q); expected %s:%s", id, left, right)
	}
	return parts[0], parts[1], nil
}

func (r *githubTeamMembershipResource) getTeamID(teamIDString string) (int64, error) {
	// Given a string that is either a team id or team slug, return the
	// id of the team it is referring to.
	client := r.client.V3Client()
	orgName := r.client.Name()

	teamID, parseIntErr := strconv.ParseInt(teamIDString, 10, 64)
	if parseIntErr == nil {
		return teamID, nil
	}

	// teamIDString is not an integer, so we assume it's a team slug
	team, _, err := client.Teams.GetTeamBySlug(context.Background(), orgName, teamIDString)
	if err != nil {
		return 0, err
	}

	return team.GetID(), nil
}
