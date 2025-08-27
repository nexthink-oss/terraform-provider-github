package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/shurcooL/githubv4"
)

var (
	_ resource.Resource                = &githubTeamResource{}
	_ resource.ResourceWithConfigure   = &githubTeamResource{}
	_ resource.ResourceWithImportState = &githubTeamResource{}
)

type githubTeamResource struct {
	client *Owner
}

type githubTeamResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	Privacy                 types.String `tfsdk:"privacy"`
	ParentTeamID            types.String `tfsdk:"parent_team_id"`
	ParentTeamReadID        types.String `tfsdk:"parent_team_read_id"`
	ParentTeamReadSlug      types.String `tfsdk:"parent_team_read_slug"`
	LdapDN                  types.String `tfsdk:"ldap_dn"`
	CreateDefaultMaintainer types.Bool   `tfsdk:"create_default_maintainer"`
	Slug                    types.String `tfsdk:"slug"`
	Etag                    types.String `tfsdk:"etag"`
	NodeID                  types.String `tfsdk:"node_id"`
	MembersCount            types.Int64  `tfsdk:"members_count"`
}

func NewGithubTeamResource() resource.Resource {
	return &githubTeamResource{}
}

func (r *githubTeamResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (r *githubTeamResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub team resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the team.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the team.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A description of the team.",
				Optional:    true,
				Computed:    true,
			},
			"privacy": schema.StringAttribute{
				Description: "The level of privacy for the team. Must be one of 'secret' or 'closed'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("secret"),
				Validators: []validator.String{
					stringvalidator.OneOf("secret", "closed"),
				},
			},
			"parent_team_id": schema.StringAttribute{
				Description: "The ID or slug of the parent team, if this is a nested team.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				PlanModifiers: []planmodifier.String{
					&parentTeamIDPlanModifier{},
				},
			},
			"parent_team_read_id": schema.StringAttribute{
				Description: "The ID of the parent team read in Github.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"parent_team_read_slug": schema.StringAttribute{
				Description: "The slug of the parent team read in Github.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ldap_dn": schema.StringAttribute{
				Description: "The LDAP Distinguished Name of the group where membership will be synchronized. Only available in GitHub Enterprise Server.",
				Optional:    true,
			},
			"create_default_maintainer": schema.BoolAttribute{
				Description: "Adds a default maintainer to the team. Adds the creating user to the team when 'true'.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"slug": schema.StringAttribute{
				Description: "The slug of the created team.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					&slugComputedWhenNameChanges{},
				},
			},
			"etag": schema.StringAttribute{
				Description: "The ETag of the team.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"node_id": schema.StringAttribute{
				Description: "The Node ID of the created team.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"members_count": schema.Int64Attribute{
				Description: "The number of members in the team.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubTeamResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubTeamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubTeamResourceModel
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

	client := r.client.V3Client()
	ownerName := r.client.Name()
	name := plan.Name.ValueString()

	newTeam := github.NewTeam{
		Name:    name,
		Privacy: github.Ptr(plan.Privacy.ValueString()),
	}

	// Only set description if it was configured by the user
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		newTeam.Description = github.Ptr(plan.Description.ValueString())
	}

	if !plan.LdapDN.IsNull() && !plan.LdapDN.IsUnknown() && plan.LdapDN.ValueString() != "" {
		ldapDN := plan.LdapDN.ValueString()
		newTeam.LDAPDN = &ldapDN
	}

	if !plan.ParentTeamID.IsNull() && !plan.ParentTeamID.IsUnknown() && plan.ParentTeamID.ValueString() != "" {
		teamId, err := r.getTeamID(plan.ParentTeamID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to get parent team ID", err.Error())
			return
		}
		newTeam.ParentTeamID = &teamId
	}

	githubTeam, _, err := client.Teams.CreateTeam(ctx, ownerName, newTeam)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create team", err.Error())
		return
	}

	// Handle parent team setting for GitHub App authentication
	if newTeam.ParentTeamID != nil && githubTeam.Parent == nil {
		_, _, err := client.Teams.EditTeamByID(ctx,
			*githubTeam.Organization.ID,
			*githubTeam.ID,
			newTeam,
			false)
		if err != nil {
			resp.Diagnostics.AddError("Failed to set parent team", err.Error())
			return
		}
	}

	// Handle default maintainer removal
	createDefaultMaintainer := plan.CreateDefaultMaintainer.ValueBool()
	if !createDefaultMaintainer {
		tflog.Debug(ctx, "Removing default maintainer from team", map[string]any{
			"team_name": name,
			"owner":     ownerName,
		})
		if err := r.removeDefaultMaintainer(*githubTeam.Slug); err != nil {
			resp.Diagnostics.AddError("Failed to remove default maintainer", err.Error())
			return
		}
	}

	plan.ID = types.StringValue(strconv.FormatInt(githubTeam.GetID(), 10))

	// Read the team back to get the current state
	r.readTeam(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *githubTeamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubTeamResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readTeam(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *githubTeamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state githubTeamResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	client := r.client.V3Client()
	orgId := r.client.ID()
	var removeParentTeam bool

	editedTeam := github.NewTeam{
		Name:    plan.Name.ValueString(),
		Privacy: github.Ptr(plan.Privacy.ValueString()),
	}

	// Only set description if it was configured by the user
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		editedTeam.Description = github.Ptr(plan.Description.ValueString())
	}

	if !plan.ParentTeamID.IsNull() && !plan.ParentTeamID.IsUnknown() && plan.ParentTeamID.ValueString() != "" {
		teamId, err := r.getTeamID(plan.ParentTeamID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to get parent team ID", err.Error())
			return
		}
		editedTeam.ParentTeamID = &teamId
		removeParentTeam = false
	} else {
		removeParentTeam = true
	}

	teamId, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid team ID", fmt.Sprintf("Could not parse team ID %q: %s", plan.ID.ValueString(), err.Error()))
		return
	}

	team, _, err := client.Teams.EditTeamByID(ctx, orgId, teamId, editedTeam, removeParentTeam)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update team", err.Error())
		return
	}

	// Handle LDAP DN updates
	if !plan.LdapDN.Equal(state.LdapDN) {
		planLdapDN := plan.LdapDN.ValueString()
		stateLdapDN := state.LdapDN.ValueString()

		// Only attempt LDAP update if there's actually an LDAP DN to set or remove
		if planLdapDN != "" || stateLdapDN != "" {
			mapping := &github.TeamLDAPMapping{
				LDAPDN: github.Ptr(planLdapDN),
			}
			_, _, err = client.Admin.UpdateTeamLDAPMapping(ctx, team.GetID(), mapping)
			if err != nil {
				resp.Diagnostics.AddError("Failed to update LDAP mapping", err.Error())
				return
			}
		}
	}

	plan.ID = types.StringValue(strconv.FormatInt(team.GetID(), 10))

	// Read the team back to get the current state
	r.readTeam(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *githubTeamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubTeamResourceModel
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

	client := r.client.V3Client()
	orgId := r.client.ID()

	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid team ID", fmt.Sprintf("Could not parse team ID %q: %s", state.ID.ValueString(), err.Error()))
		return
	}

	_, err = client.Teams.DeleteTeamByID(ctx, orgId, id)
	// Handle potential parallel deletion scenario
	if err != nil {
		// Fetch the team to check if it still exists
		_, _, checkErr := client.Teams.GetTeamByID(ctx, orgId, id)
		if checkErr != nil {
			if ghErr, ok := checkErr.(*github.ErrorResponse); ok {
				if ghErr.Response.StatusCode == http.StatusNotFound {
					// Team already deleted, remove from state
					tflog.Warn(ctx, "Team no longer exists, removing from state", map[string]any{"team_id": state.ID.ValueString()})
					return
				}
			}
		}
		resp.Diagnostics.AddError("Failed to delete team", err.Error())
		return
	}
}

func (r *githubTeamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamId, err := r.getTeamID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get team ID", err.Error())
		return
	}

	var state githubTeamResourceModel
	state.ID = types.StringValue(strconv.FormatInt(teamId, 10))
	state.CreateDefaultMaintainer = types.BoolValue(false)

	// Read the team to populate all attributes
	r.readTeam(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Helper methods

func (r *githubTeamResource) readTeam(ctx context.Context, model *githubTeamResourceModel, diags *diag.Diagnostics) {
	if !r.client.IsOrganization {
		diags.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	client := r.client.V3Client()
	orgId := r.client.ID()

	id, err := strconv.ParseInt(model.ID.ValueString(), 10, 64)
	if err != nil {
		diags.AddError("Invalid team ID", fmt.Sprintf("Could not parse team ID %q: %s", model.ID.ValueString(), err.Error()))
		return
	}

	// Add context with ID and etag for conditional requests
	requestCtx := context.WithValue(ctx, CtxId, model.ID.ValueString())
	if !model.Etag.IsNull() && !model.Etag.IsUnknown() {
		requestCtx = context.WithValue(requestCtx, CtxEtag, model.Etag.ValueString())
	}

	team, resp, err := client.Teams.GetTeamByID(requestCtx, orgId, id)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				return // No changes
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "Team no longer exists, removing from state", map[string]any{"team_id": model.ID.ValueString()})
				model.ID = types.StringNull()
				return
			}
		}
		diags.AddError("Failed to read team", err.Error())
		return
	}

	// Update all attributes
	model.Etag = types.StringValue(resp.Header.Get("ETag"))
	model.Description = types.StringValue(team.GetDescription())
	model.Name = types.StringValue(team.GetName())
	model.Privacy = types.StringValue(team.GetPrivacy())

	// CRITICAL: Only update the read-only computed attributes, preserve user's parent_team_id input
	if parent := team.Parent; parent != nil {
		model.ParentTeamReadID = types.StringValue(strconv.FormatInt(parent.GetID(), 10))
		model.ParentTeamReadSlug = types.StringValue(parent.GetSlug())
	} else {
		model.ParentTeamReadID = types.StringValue("")
		model.ParentTeamReadSlug = types.StringValue("")
		// Only clear parent_team_id if there's no parent AND it was previously set
		if !model.ParentTeamID.IsNull() && !model.ParentTeamID.IsUnknown() && model.ParentTeamID.ValueString() != "" {
			model.ParentTeamID = types.StringValue("")
		}
	}

	if ldapDN := team.GetLDAPDN(); ldapDN != "" {
		model.LdapDN = types.StringValue(ldapDN)
	} else {
		model.LdapDN = types.StringNull()
	}
	model.Slug = types.StringValue(team.GetSlug())
	model.NodeID = types.StringValue(team.GetNodeID())
	model.MembersCount = types.Int64Value(int64(team.GetMembersCount()))
}

func (r *githubTeamResource) getTeamID(identifier string) (int64, error) {
	// Given a string that is either a team id or team slug, return the
	// id of the team it is referring to.
	ctx := context.Background()
	client := r.client.V3Client()
	orgName := r.client.Name()

	teamId, parseIntErr := strconv.ParseInt(identifier, 10, 64)
	if parseIntErr == nil {
		return teamId, nil
	}

	// The given id not an integer, assume it is a team slug
	team, _, slugErr := client.Teams.GetTeamBySlug(ctx, orgName, identifier)
	if slugErr != nil {
		return -1, errors.New(parseIntErr.Error() + slugErr.Error())
	}
	return team.GetID(), nil
}

func (r *githubTeamResource) removeDefaultMaintainer(teamSlug string) error {
	client := r.client.V3Client()
	orgName := r.client.Name()
	v4client := r.client.V4Client()

	type User struct {
		Login githubv4.String
	}

	var query struct {
		Organization struct {
			Team struct {
				Members struct {
					Nodes []User
				}
			} `graphql:"team(slug:$slug)"`
		} `graphql:"organization(login:$login)"`
	}
	variables := map[string]any{
		"slug":  githubv4.String(teamSlug),
		"login": githubv4.String(orgName),
	}

	err := v4client.Query(r.client.StopContext, &query, variables)
	if err != nil {
		return err
	}

	for _, user := range query.Organization.Team.Members.Nodes {
		_, err := client.Teams.RemoveTeamMembershipBySlug(r.client.StopContext, orgName, teamSlug, string(user.Login))
		if err != nil {
			return err
		}
	}

	return nil
}

// slugComputedWhenNameChanges is a plan modifier that marks the slug as computed when the name changes
type slugComputedWhenNameChanges struct{}

func (m *slugComputedWhenNameChanges) Description(ctx context.Context) string {
	return "Marks slug as computed when team name changes"
}

func (m *slugComputedWhenNameChanges) MarkdownDescription(ctx context.Context) string {
	return "Marks slug as computed when team name changes"
}

func (m *slugComputedWhenNameChanges) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If we're creating the resource, use UseStateForUnknown behavior
	if req.State.Raw.IsNull() {
		if !req.PlanValue.IsUnknown() {
			resp.PlanValue = types.StringUnknown()
		}
		return
	}

	// Check if the name has changed
	var stateName, planName types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("name"), &stateName)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("name"), &planName)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If the name has changed, mark slug as unknown so it will be computed
	if !stateName.Equal(planName) {
		resp.PlanValue = types.StringUnknown()
	} else {
		// Name hasn't changed, preserve the state value
		resp.PlanValue = req.StateValue
	}
}

// parentTeamIDPlanModifier implements diff suppression for parent_team_id when it matches parent_team_read_id or parent_team_read_slug
type parentTeamIDPlanModifier struct{}

func (m *parentTeamIDPlanModifier) Description(ctx context.Context) string {
	return "Suppress diff for parent_team_id when it matches parent_team_read_id or parent_team_read_slug"
}

func (m *parentTeamIDPlanModifier) MarkdownDescription(ctx context.Context) string {
	return "Suppress diff for parent_team_id when it matches parent_team_read_id or parent_team_read_slug"
}

func (m *parentTeamIDPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Only apply during updates where we have both state and plan
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	// Get parent team read values from the state
	var parentTeamReadID, parentTeamReadSlug types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("parent_team_read_id"), &parentTeamReadID)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("parent_team_read_slug"), &parentTeamReadSlug)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If the planned parent_team_id matches either the read ID or slug,
	// it means we're referring to the same team - suppress the diff
	if req.PlanValue.Equal(parentTeamReadID) || req.PlanValue.Equal(parentTeamReadSlug) {
		resp.PlanValue = req.StateValue
	}
}
