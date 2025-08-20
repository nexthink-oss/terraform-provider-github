package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

)

var (
	_ resource.Resource                = &githubOrganizationSecurityManagerResource{}
	_ resource.ResourceWithConfigure   = &githubOrganizationSecurityManagerResource{}
	_ resource.ResourceWithImportState = &githubOrganizationSecurityManagerResource{}
)

type githubOrganizationSecurityManagerResource struct {
	client *Owner
}

type githubOrganizationSecurityManagerResourceModel struct {
	ID       types.String `tfsdk:"id"`
	TeamSlug types.String `tfsdk:"team_slug"`
}

func NewGithubOrganizationSecurityManagerResource() resource.Resource {
	return &githubOrganizationSecurityManagerResource{}
}

func (r *githubOrganizationSecurityManagerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_security_manager"
}

func (r *githubOrganizationSecurityManagerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the Security manager teams for a GitHub Organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the security manager team.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_slug": schema.StringAttribute{
				Description: "The slug of the team to manage.",
				Required:    true,
			},
		},
	}
}

func (r *githubOrganizationSecurityManagerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Helper function to get the security manager role
func (r *githubOrganizationSecurityManagerResource) getSecurityManagerRole(ctx context.Context, orgName string) (*github.CustomOrgRoles, error) {
	client := r.client.V3Client()

	roles, _, err := client.Organizations.ListRoles(ctx, orgName)
	if err != nil {
		return nil, err
	}

	for _, role := range roles.CustomRepoRoles {
		if role.GetName() == "security_manager" {
			return role, nil
		}
	}

	return nil, errors.New("security manager role not found")
}

func (r *githubOrganizationSecurityManagerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubOrganizationSecurityManagerResourceModel

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
	teamSlug := plan.TeamSlug.ValueString()

	team, _, err := client.Teams.GetTeamBySlug(ctx, orgName, teamSlug)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting team",
			fmt.Sprintf("Could not get team %s: %s", teamSlug, err),
		)
		return
	}

	smRole, err := r.getSecurityManagerRole(ctx, orgName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting security manager role",
			fmt.Sprintf("Could not get security manager role: %s", err),
		)
		return
	}

	_, err = client.Organizations.AssignOrgRoleToTeam(ctx, orgName, teamSlug, smRole.GetID())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning security manager role to team",
			fmt.Sprintf("Could not assign security manager role to team %s: %s", teamSlug, err),
		)
		return
	}

	plan.ID = types.StringValue(strconv.FormatInt(team.GetID(), 10))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubOrganizationSecurityManagerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubOrganizationSecurityManagerResourceModel

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

	teamID, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing team ID",
			fmt.Sprintf("Could not parse team ID %s: %s", state.ID.ValueString(), err),
		)
		return
	}

	smRole, err := r.getSecurityManagerRole(ctx, orgName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting security manager role",
			fmt.Sprintf("Could not get security manager role: %s", err),
		)
		return
	}

	// There is no endpoint for getting a single security manager team, so get the list and filter.
	options := &github.ListOptions{PerPage: 100}
	var smTeam *github.Team = nil

	for {
		smTeams, githubResp, err := client.Organizations.ListTeamsAssignedToOrgRole(ctx, orgName, smRole.GetID(), options)
		if err != nil {
			if ghErr, ok := err.(*github.ErrorResponse); ok {
				if ghErr.Response.StatusCode == http.StatusNotFound {
					log.Printf("[INFO] Removing organization security manager team %s from state because it no longer exists in GitHub", state.ID.ValueString())
					resp.State.RemoveResource(ctx)
					return
				}
			}
			resp.Diagnostics.AddError(
				"Error reading security manager teams",
				fmt.Sprintf("Could not read security manager teams: %s", err),
			)
			return
		}

		for _, t := range smTeams {
			if t.GetID() == teamID {
				smTeam = t
				break
			}
		}

		// Break when we've found the team or there are no more pages.
		if smTeam != nil || githubResp.NextPage == 0 {
			break
		}

		options.Page = githubResp.NextPage
	}

	if smTeam == nil {
		log.Printf("[INFO] Removing organization security manager team %s from state because it no longer exists in GitHub", state.ID.ValueString())
		resp.State.RemoveResource(ctx)
		return
	}

	state.TeamSlug = types.StringValue(smTeam.GetSlug())

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubOrganizationSecurityManagerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubOrganizationSecurityManagerResourceModel

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
	teamSlug := plan.TeamSlug.ValueString()

	team, _, err := client.Teams.GetTeamBySlug(ctx, orgName, teamSlug)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting team by slug",
			fmt.Sprintf("Could not get team by slug %s: %s", teamSlug, err),
		)
		return
	}

	smRole, err := r.getSecurityManagerRole(ctx, orgName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting security manager role",
			fmt.Sprintf("Could not get security manager role: %s", err),
		)
		return
	}

	// Adding the same team is a no-op.
	_, err = client.Organizations.AssignOrgRoleToTeam(ctx, orgName, team.GetSlug(), smRole.GetID())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error assigning security manager role to team",
			fmt.Sprintf("Could not assign security manager role to team %s: %s", team.GetSlug(), err),
		)
		return
	}

	// Update the team slug in the plan to reflect any changes
	plan.TeamSlug = types.StringValue(team.GetSlug())

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubOrganizationSecurityManagerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubOrganizationSecurityManagerResourceModel

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
	teamSlug := state.TeamSlug.ValueString()

	smRole, err := r.getSecurityManagerRole(ctx, orgName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting security manager role",
			fmt.Sprintf("Could not get security manager role: %s", err),
		)
		return
	}

	_, err = client.Organizations.RemoveOrgRoleFromTeam(ctx, orgName, teamSlug, smRole.GetID())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error removing security manager role from team",
			fmt.Sprintf("Could not remove security manager role from team %s: %s", teamSlug, err),
		)
		return
	}
}

func (r *githubOrganizationSecurityManagerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Organization security manager resources can be imported using just the team ID
	// since they're always associated with the configured organization
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
