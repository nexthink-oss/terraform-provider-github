package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/shurcooL/githubv4"
)

// UpdateTeamReviewAssignmentInput represents the input for updating team review assignment settings
type UpdateTeamReviewAssignmentInput struct {
	ClientMutationID                 string `json:"clientMutationId,omitempty"`
	TeamID                           string `graphql:"id" json:"id"`
	ReviewRequestDelegation          bool   `graphql:"enabled" json:"enabled"`
	ReviewRequestDelegationAlgorithm string `graphql:"algorithm" json:"algorithm"`
	ReviewRequestDelegationCount     int    `graphql:"teamMemberCount" json:"teamMemberCount"`
	ReviewRequestDelegationNotifyAll bool   `graphql:"notifyTeam" json:"notifyTeam"`
}

// QueryTeamSettings represents the GraphQL query for team settings
type QueryTeamSettings struct {
	Organization struct {
		Team struct {
			Name                             string `graphql:"name"`
			Slug                             string `graphql:"slug"`
			ID                               string `graphql:"id"`
			ReviewRequestDelegation          bool   `graphql:"reviewRequestDelegationEnabled"`
			ReviewRequestDelegationAlgorithm string `graphql:"reviewRequestDelegationAlgorithm"`
			ReviewRequestDelegationCount     int    `graphql:"reviewRequestDelegationMemberCount"`
			ReviewRequestDelegationNotifyAll bool   `graphql:"reviewRequestDelegationNotifyTeam"`
		} `graphql:"team(slug:$slug)"`
	} `graphql:"organization(login:$login)"`
}

var (
	_ resource.Resource                = &githubTeamSettingsResource{}
	_ resource.ResourceWithConfigure   = &githubTeamSettingsResource{}
	_ resource.ResourceWithImportState = &githubTeamSettingsResource{}
)

func NewGithubTeamSettingsResource() resource.Resource {
	return &githubTeamSettingsResource{}
}

type githubTeamSettingsResource struct {
	client *Owner
}

type githubTeamSettingsResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	TeamID                  types.String `tfsdk:"team_id"`
	TeamSlug                types.String `tfsdk:"team_slug"`
	TeamUID                 types.String `tfsdk:"team_uid"`
	ReviewRequestDelegation types.List   `tfsdk:"review_request_delegation"`
}

type githubTeamSettingsReviewRequestDelegationModel struct {
	Algorithm   types.String `tfsdk:"algorithm"`
	MemberCount types.Int64  `tfsdk:"member_count"`
	Notify      types.Bool   `tfsdk:"notify"`
}

func (r *githubTeamSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_settings"
}

func (r *githubTeamSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the team settings (in particular the request review delegation settings)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the team settings resource.",
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
			"team_slug": schema.StringAttribute{
				Description: "The slug of the Team within the Organization.",
				Computed:    true,
			},
			"team_uid": schema.StringAttribute{
				Description: "The unique ID of the Team on GitHub. Corresponds to the ID of the 'github_team_settings' resource.",
				Computed:    true,
			},
			"review_request_delegation": schema.ListNestedAttribute{
				Description: "The settings for delegating code reviews to individuals on behalf of the team. If this block is present, even without any fields, then review request delegation will be enabled for the team.",
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListNull(types.ObjectType{AttrTypes: reviewRequestDelegationAttrTypes()})),
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"algorithm": schema.StringAttribute{
							Description: "The algorithm to use when assigning pull requests to team members. Supported values are 'ROUND_ROBIN' and 'LOAD_BALANCE'.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("ROUND_ROBIN"),
							Validators: []validator.String{
								&teamSettingsAlgorithmValidator{},
							},
						},
						"member_count": schema.Int64Attribute{
							Description: "The number of team members to assign to a pull request.",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(1),
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
						"notify": schema.BoolAttribute{
							Description: "whether to notify the entire team when at least one member is also assigned to the pull request.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
					},
				},
			},
		},
	}
}

func reviewRequestDelegationAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"algorithm":    types.StringType,
		"member_count": types.Int64Type,
		"notify":       types.BoolType,
	}
}

// Custom validator for algorithm field
type teamSettingsAlgorithmValidator struct{}

func (v *teamSettingsAlgorithmValidator) Description(_ context.Context) string {
	return "algorithm must be one of 'ROUND_ROBIN' or 'LOAD_BALANCE'"
}

func (v *teamSettingsAlgorithmValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *teamSettingsAlgorithmValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if value != "ROUND_ROBIN" && value != "LOAD_BALANCE" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Algorithm",
			"review request delegation algorithm must be one of ['ROUND_ROBIN', 'LOAD_BALANCE']",
		)
	}
}

func (r *githubTeamSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubTeamSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubTeamSettingsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts, %q is a user", r.client.Name()),
		)
		return
	}

	teamIDString := plan.TeamID.ValueString()

	nodeId, slug, err := r.resolveTeamIDs(ctx, teamIDString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Resolving Team",
			fmt.Sprintf("Could not resolve team ID or slug %s: %s", teamIDString, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(nodeId)
	plan.TeamSlug = types.StringValue(slug)
	plan.TeamUID = types.StringValue(nodeId)

	// Update the team settings
	r.updateTeamSettings(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back the current state
	r.readTeamSettings(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubTeamSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubTeamSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used with organization accounts, %q is a user", r.client.Name()),
		)
		return
	}

	r.readTeamSettings(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubTeamSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubTeamSettingsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.updateTeamSettings(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back the current state
	r.readTeamSettings(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubTeamSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubTeamSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	graphql := r.client.V4Client()
	teamID := state.ID.ValueString()

	var mutation struct {
		UpdateTeamReviewAssignment struct {
			ClientMutationId githubv4.ID `graphql:"clientMutationId"`
		} `graphql:"updateTeamReviewAssignment(input:$input)"`
	}

	// Reset to default settings (disables review request delegation)
	defaultSettings := UpdateTeamReviewAssignmentInput{
		TeamID:                           teamID,
		ReviewRequestDelegation:          false,
		ReviewRequestDelegationAlgorithm: "ROUND_ROBIN",
		ReviewRequestDelegationCount:     1,
		ReviewRequestDelegationNotifyAll: true,
	}

	err := graphql.Mutate(ctx, &mutation, defaultSettings, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Team Settings",
			fmt.Sprintf("Could not delete team settings for team %s: %s", teamID, err.Error()),
		)
		return
	}

	tflog.Info(ctx, "Team settings deleted", map[string]any{
		"team_id": teamID,
	})
}

func (r *githubTeamSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	nodeId, slug, err := r.resolveTeamIDs(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing Team Settings",
			fmt.Sprintf("Could not resolve team ID or slug %s: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), nodeId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_slug"), slug)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("team_uid"), nodeId)...)
}

func (r *githubTeamSettingsResource) resolveTeamIDs(ctx context.Context, idOrSlug string) (nodeId string, slug string, err error) {
	client := r.client.V3Client()
	orgName := r.client.Name()
	orgId := r.client.ID()

	teamId, parseIntErr := strconv.ParseInt(idOrSlug, 10, 64)
	if parseIntErr != nil {
		// The given id not an integer, assume it is a team slug
		team, _, slugErr := client.Teams.GetTeamBySlug(ctx, orgName, idOrSlug)
		if slugErr != nil {
			return "", "", fmt.Errorf("failed to get team by slug: %v, failed to parse as int: %v", slugErr, parseIntErr)
		}
		return team.GetNodeID(), team.GetSlug(), nil
	} else {
		// The given id is an integer, assume it is a team id
		//nolint:staticcheck // SA1019: GetTeamByID is deprecated but needed for ID->slug conversion
		team, _, teamIdErr := client.Teams.GetTeamByID(ctx, orgId, teamId)
		if teamIdErr != nil {
			// There isn't a team with the given ID, assume it is a teamslug
			team, _, slugErr := client.Teams.GetTeamBySlug(ctx, orgName, idOrSlug)
			if slugErr != nil {
				return "", "", fmt.Errorf("failed to get team by ID: %v, failed to get team by slug: %v", teamIdErr, slugErr)
			}

			return team.GetNodeID(), team.GetSlug(), nil
		}

		return team.GetNodeID(), team.GetSlug(), nil
	}
}

func (r *githubTeamSettingsResource) readTeamSettings(ctx context.Context, state *githubTeamSettingsResourceModel, diags *diag.Diagnostics) {
	graphql := r.client.V4Client()
	orgName := r.client.Name()
	teamSlug := state.TeamSlug.ValueString()

	var query QueryTeamSettings
	variables := map[string]any{
		"slug":  githubv4.String(teamSlug),
		"login": githubv4.String(orgName),
	}

	err := graphql.Query(ctx, &query, variables)
	if err != nil {
		diags.AddError(
			"Error Reading Team Settings",
			fmt.Sprintf("Could not read team settings for team %s: %s", teamSlug, err.Error()),
		)
		return
	}

	if query.Organization.Team.ReviewRequestDelegation {
		reviewRequestDelegationObj := types.ObjectValueMust(
			reviewRequestDelegationAttrTypes(),
			map[string]attr.Value{
				"algorithm":    types.StringValue(query.Organization.Team.ReviewRequestDelegationAlgorithm),
				"member_count": types.Int64Value(int64(query.Organization.Team.ReviewRequestDelegationCount)),
				"notify":       types.BoolValue(query.Organization.Team.ReviewRequestDelegationNotifyAll),
			},
		)
		state.ReviewRequestDelegation = types.ListValueMust(
			types.ObjectType{AttrTypes: reviewRequestDelegationAttrTypes()},
			[]attr.Value{reviewRequestDelegationObj},
		)
	} else {
		state.ReviewRequestDelegation = types.ListNull(types.ObjectType{AttrTypes: reviewRequestDelegationAttrTypes()})
	}
}

func (r *githubTeamSettingsResource) updateTeamSettings(ctx context.Context, plan *githubTeamSettingsResourceModel, diags *diag.Diagnostics) {
	graphql := r.client.V4Client()
	teamID := plan.ID.ValueString()

	var mutation struct {
		UpdateTeamReviewAssignment struct {
			ClientMutationId githubv4.ID `graphql:"clientMutationId"`
		} `graphql:"updateTeamReviewAssignment(input:$input)"`
	}

	if plan.ReviewRequestDelegation.IsNull() || len(plan.ReviewRequestDelegation.Elements()) == 0 {
		// Disable review request delegation
		defaultSettings := UpdateTeamReviewAssignmentInput{
			TeamID:                           teamID,
			ReviewRequestDelegation:          false,
			ReviewRequestDelegationAlgorithm: "ROUND_ROBIN",
			ReviewRequestDelegationCount:     1,
			ReviewRequestDelegationNotifyAll: true,
		}

		err := graphql.Mutate(ctx, &mutation, defaultSettings, nil)
		if err != nil {
			diags.AddError(
				"Error Updating Team Settings",
				fmt.Sprintf("Could not update team settings for team %s: %s", teamID, err.Error()),
			)
		}
	} else {
		// Enable review request delegation with specified settings
		var delegationSettings []githubTeamSettingsReviewRequestDelegationModel
		diags.Append(plan.ReviewRequestDelegation.ElementsAs(ctx, &delegationSettings, false)...)
		if diags.HasError() {
			return
		}

		settings := delegationSettings[0]
		updateSettings := UpdateTeamReviewAssignmentInput{
			TeamID:                           teamID,
			ReviewRequestDelegation:          true,
			ReviewRequestDelegationAlgorithm: settings.Algorithm.ValueString(),
			ReviewRequestDelegationCount:     int(settings.MemberCount.ValueInt64()),
			ReviewRequestDelegationNotifyAll: settings.Notify.ValueBool(),
		}

		err := graphql.Mutate(ctx, &mutation, updateSettings, nil)
		if err != nil {
			diags.AddError(
				"Error Updating Team Settings",
				fmt.Sprintf("Could not update team settings for team %s: %s", teamID, err.Error()),
			)
		}
	}
}
