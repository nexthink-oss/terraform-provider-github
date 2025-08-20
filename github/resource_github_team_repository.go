package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubTeamRepositoryResource{}
	_ resource.ResourceWithConfigure   = &githubTeamRepositoryResource{}
	_ resource.ResourceWithImportState = &githubTeamRepositoryResource{}
)

type githubTeamRepositoryResource struct {
	client *Owner
}

type githubTeamRepositoryResourceModel struct {
	ID         types.String `tfsdk:"id"`
	TeamID     types.String `tfsdk:"team_id"`
	Repository types.String `tfsdk:"repository"`
	Permission types.String `tfsdk:"permission"`
	Etag       types.String `tfsdk:"etag"`
}

func NewGithubTeamRepositoryResource() resource.Resource {
	return &githubTeamRepositoryResource{}
}

func (r *githubTeamRepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_repository"
}

func (r *githubTeamRepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the associations between teams and repositories.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the team repository association (team_id:repository).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"team_id": schema.StringAttribute{
				Description: "ID or slug of team",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The repository to add to the team.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission": schema.StringAttribute{
				Description: "The permissions of team members regarding the repository. Must be one of 'pull', 'triage', 'push', 'maintain', 'admin' or the name of an existing custom repository role within the organisation.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("pull"),
			},
			"etag": schema.StringAttribute{
				Description: "The ETag of the team repository association.",
				Computed:    true,
			},
		},
	}
}

func (r *githubTeamRepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubTeamRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubTeamRepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used in the context of an organization",
		)
		return
	}

	client := r.client.V3Client()

	// The given team id could be an id or a slug
	givenTeamId := plan.TeamID.ValueString()
	teamSlug, err := r.getTeamSlug(ctx, givenTeamId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get team slug",
			fmt.Sprintf("Could not resolve team slug for %q: %s", givenTeamId, err.Error()),
		)
		return
	}

	orgName := r.client.Name()
	repoName := plan.Repository.ValueString()
	permission := plan.Permission.ValueString()

	_, err = client.Teams.AddTeamRepoBySlug(ctx,
		orgName,
		teamSlug,
		orgName,
		repoName,
		&github.TeamAddTeamRepoOptions{
			Permission: permission,
		},
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create team repository association",
			fmt.Sprintf("Could not add repository %q to team %q: %s", repoName, givenTeamId, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(r.buildTwoPartID(givenTeamId, repoName))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the resource to populate computed attributes
	r.read(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *githubTeamRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubTeamRepositoryResourceModel

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

func (r *githubTeamRepositoryResource) read(ctx context.Context, state *githubTeamRepositoryResourceModel, diagnostics *diag.Diagnostics) {
	if !r.client.IsOrganization {
		diagnostics.AddError(
			"Organization Required",
			"This resource can only be used in the context of an organization",
		)
		return
	}

	client := r.client.V3Client()

	teamIdString, repoName, err := r.parseTwoPartID(state.ID.ValueString(), "team_id", "repository")
	if err != nil {
		diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Could not parse ID %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	teamSlug, err := r.getTeamSlug(ctx, teamIdString)
	if err != nil {
		diagnostics.AddError(
			"Unable to get team slug",
			fmt.Sprintf("Could not resolve team slug for %q: %s", teamIdString, err.Error()),
		)
		return
	}

	orgName := r.client.Name()

	// Add context with ETag for conditional requests
	requestCtx := ctx
	if !state.Etag.IsNull() && !state.Etag.IsUnknown() {
		requestCtx = context.WithValue(ctx, CtxEtag, state.Etag.ValueString())
	}

	repo, resp, repoErr := client.Teams.IsTeamRepoBySlug(requestCtx, orgName, teamSlug, orgName, repoName)
	if repoErr != nil {
		if ghErr, ok := repoErr.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "Removing team repository association from state because it no longer exists in GitHub", map[string]interface{}{
					"id": state.ID.ValueString(),
				})
				state.ID = types.StringNull()
				return
			}
		}
		diagnostics.AddError(
			"Unable to read team repository association",
			fmt.Sprintf("Could not read team repository association %q: %s", state.ID.ValueString(), repoErr.Error()),
		)
		return
	}

	state.Etag = types.StringValue(resp.Header.Get("ETag"))
	if state.TeamID.IsNull() || state.TeamID.IsUnknown() {
		// If team_id is empty, that means we are importing the resource.
		// Set the team_id to be the id of the team.
		state.TeamID = types.StringValue(teamIdString)
	}
	state.Repository = types.StringValue(repo.GetName())
	state.Permission = types.StringValue(r.getPermission(repo.GetRoleName()))
}

func (r *githubTeamRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubTeamRepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used in the context of an organization",
		)
		return
	}

	client := r.client.V3Client()

	teamIdString, repoName, err := r.parseTwoPartID(plan.ID.ValueString(), "team_id", "repository")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Could not parse ID %q: %s", plan.ID.ValueString(), err.Error()),
		)
		return
	}

	teamSlug, err := r.getTeamSlug(ctx, teamIdString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get team slug",
			fmt.Sprintf("Could not resolve team slug for %q: %s", teamIdString, err.Error()),
		)
		return
	}

	orgName := r.client.Name()
	permission := plan.Permission.ValueString()

	// the go-github library's AddTeamRepo method uses the add/update endpoint from GitHub API
	_, err = client.Teams.AddTeamRepoBySlug(ctx,
		orgName,
		teamSlug,
		orgName,
		repoName,
		&github.TeamAddTeamRepoOptions{
			Permission: permission,
		},
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update team repository association",
			fmt.Sprintf("Could not update repository %q permissions for team %q: %s", repoName, teamIdString, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(r.buildTwoPartID(teamIdString, repoName))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read the resource to populate computed attributes
	r.read(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *githubTeamRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubTeamRepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used in the context of an organization",
		)
		return
	}

	client := r.client.V3Client()

	teamIdString, repoName, err := r.parseTwoPartID(state.ID.ValueString(), "team_id", "repository")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID format",
			fmt.Sprintf("Could not parse ID %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	teamSlug, err := r.getTeamSlug(ctx, teamIdString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get team slug",
			fmt.Sprintf("Could not resolve team slug for %q: %s", teamIdString, err.Error()),
		)
		return
	}

	orgName := r.client.Name()

	deleteResp, err := client.Teams.RemoveTeamRepoBySlug(ctx, orgName, teamSlug, orgName, repoName)

	if deleteResp != nil && deleteResp.StatusCode == 404 {
		tflog.Debug(ctx, "Failed to find team to delete for repo", map[string]interface{}{
			"team_id":    teamIdString,
			"repository": repoName,
		})
		// Try to get the current repo name in case it was renamed
		repo, _, repoErr := client.Repositories.Get(ctx, orgName, repoName)
		if repoErr != nil {
			resp.Diagnostics.AddError(
				"Unable to get repository",
				fmt.Sprintf("Could not get repository %q: %s", repoName, repoErr.Error()),
			)
			return
		}
		newRepoName := repo.GetName()
		if newRepoName != repoName {
			tflog.Info(ctx, "Repository name has changed, trying delete again", map[string]interface{}{
				"old_name": repoName,
				"new_name": newRepoName,
			})
			_, err = client.Teams.RemoveTeamRepoBySlug(ctx, orgName, teamSlug, orgName, newRepoName)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to delete team repository association",
					fmt.Sprintf("Could not remove repository %q from team %q: %s", newRepoName, teamIdString, err.Error()),
				)
				return
			}
		}
		// If we reach here, the resource was already deleted (404) so we can return successfully
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete team repository association",
			fmt.Sprintf("Could not remove repository %q from team %q: %s", repoName, teamIdString, err.Error()),
		)
		return
	}
}

func (r *githubTeamRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	teamIdString, repoName, err := r.parseTwoPartID(req.ID, "team_id", "repository")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("Could not parse import ID %q. Expected format: team_id:repository or team_slug:repository", req.ID),
		)
		return
	}

	teamId, err := r.getTeamID(ctx, teamIdString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get team ID",
			fmt.Sprintf("Could not resolve team ID %q: %s", teamIdString, err.Error()),
		)
		return
	}

	// Set the ID using the numeric team ID for consistency
	importId := r.buildTwoPartID(strconv.FormatInt(teamId, 10), repoName)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), importId)...)
}

// Helper functions

func (r *githubTeamRepositoryResource) parseTwoPartID(id, left, right string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected ID format (%q); expected %s:%s", id, left, right)
	}
	return parts[0], parts[1], nil
}

func (r *githubTeamRepositoryResource) buildTwoPartID(a, b string) string {
	return fmt.Sprintf("%s:%s", a, b)
}

func (r *githubTeamRepositoryResource) getTeamID(ctx context.Context, teamIDString string) (int64, error) {
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

func (r *githubTeamRepositoryResource) getTeamSlug(ctx context.Context, teamIDString string) (string, error) {
	client := r.client.V3Client()
	orgId := r.client.ID()

	teamId, parseIntErr := strconv.ParseInt(teamIDString, 10, 64)
	if parseIntErr == nil {
		// It's an ID, get the team to find the slug
		// Note: This still uses GetTeamByID as it's the only way to get slug from numeric ID
		// This call is minimized by caching and the migration to slug-based APIs elsewhere
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

func (r *githubTeamRepositoryResource) getPermission(permission string) string {
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
