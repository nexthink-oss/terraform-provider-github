package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubRepositoryCollaboratorResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryCollaboratorResource{}
	_ resource.ResourceWithImportState = &githubRepositoryCollaboratorResource{}
)

type githubRepositoryCollaboratorResource struct {
	client *Owner
}

type githubRepositoryCollaboratorResourceModel struct {
	ID                        types.String `tfsdk:"id"`
	Username                  types.String `tfsdk:"username"`
	Repository                types.String `tfsdk:"repository"`
	Permission                types.String `tfsdk:"permission"`
	PermissionDiffSuppression types.Bool   `tfsdk:"permission_diff_suppression"`
	InvitationID              types.String `tfsdk:"invitation_id"`
}

func NewGithubRepositoryCollaboratorResource() resource.Resource {
	return &githubRepositoryCollaboratorResource{}
}

func (r *githubRepositoryCollaboratorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_collaborator"
}

func (r *githubRepositoryCollaboratorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub repository collaborator resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the repository collaborator (repository:username).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Description: "The user to add to the repository as a collaborator.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					&caseInsensitiveStringPlanModifier{},
				},
			},
			"repository": schema.StringAttribute{
				Description: "The GitHub repository",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permission": schema.StringAttribute{
				Description: "The permission of the outside collaborator for the repository. Must be one of 'pull', 'push', 'maintain', 'triage' or 'admin' or the name of an existing custom repository role within the organization for organization-owned repositories. Must be 'push' for personal repositories. Defaults to 'push'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("push"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					&permissionDiffSuppressionPlanModifier{},
				},
			},
			"permission_diff_suppression": schema.BoolAttribute{
				Description: "Suppress plan diffs for triage and maintain. Defaults to 'false'.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"invitation_id": schema.StringAttribute{
				Description: "ID of the invitation to be used in 'github_user_invitation_accepter'",
				Computed:    true,
			},
		},
	}
}

func (r *githubRepositoryCollaboratorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubRepositoryCollaboratorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubRepositoryCollaboratorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	username := data.Username.ValueString()
	repoName := data.Repository.ValueString()

	owner, repoNameWithoutOwner := r.parseRepoName(repoName, r.client.Name())

	_, _, err := client.Repositories.AddCollaborator(ctx,
		owner,
		repoNameWithoutOwner,
		username,
		&github.RepositoryAddCollaboratorOptions{
			Permission: data.Permission.ValueString(),
		})

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Add Repository Collaborator",
			fmt.Sprintf("An unexpected error occurred when adding the repository collaborator: %s", err.Error()),
		)
		return
	}

	data.ID = types.StringValue(r.buildTwoPartID(repoName, username))

	tflog.Debug(ctx, "created GitHub repository collaborator", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repoName,
		"username":   username,
	})

	// Read the created resource to populate all computed fields
	r.readGithubRepositoryCollaborator(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryCollaboratorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubRepositoryCollaboratorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubRepositoryCollaborator(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryCollaboratorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubRepositoryCollaboratorResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Since all attributes are ForceNew, this should only be called for computed field updates
	r.readGithubRepositoryCollaborator(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryCollaboratorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubRepositoryCollaboratorResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	username := data.Username.ValueString()
	repoName := data.Repository.ValueString()

	owner, repoNameWithoutOwner := r.parseRepoName(repoName, r.client.Name())

	// Delete any pending invitations
	invitation, err := r.findRepoInvitation(ctx, client, owner, repoNameWithoutOwner, username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Find Repository Invitation",
			fmt.Sprintf("An unexpected error occurred when finding repository invitation: %s", err.Error()),
		)
		return
	} else if invitation != nil {
		_, err = client.Repositories.DeleteInvitation(ctx, owner, repoNameWithoutOwner, invitation.GetID())
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Delete Repository Invitation",
				fmt.Sprintf("An unexpected error occurred when deleting repository invitation: %s", err.Error()),
			)
			return
		}
	}

	_, err = client.Repositories.RemoveCollaborator(ctx, owner, repoNameWithoutOwner, username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Remove Repository Collaborator",
			fmt.Sprintf("An unexpected error occurred when removing the repository collaborator: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub repository collaborator", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repoName,
		"username":   username,
	})
}

func (r *githubRepositoryCollaboratorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	repoName, username, err := r.parseTwoPartID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse import ID: %s", err.Error()),
		)
		return
	}

	data := &githubRepositoryCollaboratorResourceModel{
		ID:                        types.StringValue(req.ID),
		Repository:                types.StringValue(repoName),
		Username:                  types.StringValue(username),
		PermissionDiffSuppression: types.BoolValue(false),
	}

	r.readGithubRepositoryCollaborator(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubRepositoryCollaboratorResource) buildTwoPartID(a, b string) string {
	return fmt.Sprintf("%s:%s", a, b)
}

func (r *githubRepositoryCollaboratorResource) parseTwoPartID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected ID format (%q), expected repository:username", id)
	}

	return parts[0], parts[1], nil
}

func (r *githubRepositoryCollaboratorResource) parseRepoName(repoName string, defaultOwner string) (string, string) {
	// GitHub replaces '/' with '-' for a repo name, so it is safe to assume that if repo name contains '/'
	// then first part will be the owner name and second part will be the repo name
	if strings.Contains(repoName, "/") {
		parts := strings.Split(repoName, "/")
		return parts[0], parts[1]
	} else {
		return defaultOwner, repoName
	}
}

func (r *githubRepositoryCollaboratorResource) findRepoInvitation(ctx context.Context, client *github.Client, owner, repo, collaborator string) (*github.RepositoryInvitation, error) {
	opt := &github.ListOptions{PerPage: maxPerPage}
	for {
		invitations, resp, err := client.Repositories.ListInvitations(ctx, owner, repo, opt)
		if err != nil {
			return nil, err
		}

		for _, i := range invitations {
			if strings.EqualFold(i.GetInvitee().GetLogin(), collaborator) {
				return i, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return nil, nil
}

func (r *githubRepositoryCollaboratorResource) getPermission(permission string) string {
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

func (r *githubRepositoryCollaboratorResource) readGithubRepositoryCollaborator(ctx context.Context, data *githubRepositoryCollaboratorResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()

	repoName, username, err := r.parseTwoPartID(data.ID.ValueString())
	if err != nil {
		diags.AddError(
			"Invalid Resource ID",
			fmt.Sprintf("Unable to parse resource ID: %s", err.Error()),
		)
		return
	}

	owner, repoNameWithoutOwner := r.parseRepoName(repoName, r.client.Name())

	// First, check if the user has been invited but has not yet accepted
	invitation, err := r.findRepoInvitation(ctx, client, owner, repoNameWithoutOwner, username)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				// this short circuits the rest of the code because if the
				// repo is 404, no reason to try to list existing collaborators
				tflog.Info(ctx, "removing repository collaborator from state because it no longer exists in GitHub", map[string]interface{}{
					"owner":      owner,
					"repository": repoName,
					"username":   username,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Find Repository Invitation",
			fmt.Sprintf("An unexpected error occurred when finding repository invitation: %s", err.Error()),
		)
		return
	}

	if invitation != nil {
		username = invitation.GetInvitee().GetLogin()
		permissionName := r.getPermission(invitation.GetPermissions())

		data.Repository = types.StringValue(repoName)
		data.Username = types.StringValue(username)
		data.Permission = types.StringValue(permissionName)
		data.InvitationID = types.StringValue(fmt.Sprintf("%d", invitation.GetID()))
		return
	}

	// Next, check if the user has accepted the invite and is a full collaborator
	opt := &github.ListCollaboratorsOptions{ListOptions: github.ListOptions{
		PerPage: maxPerPage,
	}}

	for {
		collaborators, resp, err := client.Repositories.ListCollaborators(ctx,
			owner, repoNameWithoutOwner, opt)
		if err != nil {
			diags.AddError(
				"Unable to List Repository Collaborators",
				fmt.Sprintf("An unexpected error occurred when listing repository collaborators: %s", err.Error()),
			)
			return
		}

		for _, c := range collaborators {
			if strings.EqualFold(c.GetLogin(), username) {
				data.Repository = types.StringValue(repoName)
				data.Username = types.StringValue(c.GetLogin())
				data.Permission = types.StringValue(r.getPermission(c.GetRoleName()))
				data.InvitationID = types.StringNull()
				return
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	// The user is neither invited nor a collaborator
	tflog.Info(ctx, "removing repository collaborator from state because it no longer exists in GitHub", map[string]interface{}{
		"username":   username,
		"owner":      owner,
		"repository": repoName,
	})
	data.ID = types.StringNull()
}

// Custom plan modifier for case-insensitive username comparison
type caseInsensitiveStringPlanModifier struct{}

func (m caseInsensitiveStringPlanModifier) Description(ctx context.Context) string {
	return "Ignores case differences for username values"
}

func (m caseInsensitiveStringPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m caseInsensitiveStringPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Don't modify plan if we're in a destroy operation or either value is unknown/null
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() ||
		req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	// If strings are equal when compared case-insensitively, keep the state value
	if strings.EqualFold(req.PlanValue.ValueString(), req.StateValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}

// Custom plan modifier for permission diff suppression
type permissionDiffSuppressionPlanModifier struct{}

func (m permissionDiffSuppressionPlanModifier) Description(ctx context.Context) string {
	return "Conditionally suppresses diffs for triage and maintain permissions"
}

func (m permissionDiffSuppressionPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m permissionDiffSuppressionPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Don't modify plan if we're in a destroy operation or either value is unknown/null
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() ||
		req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}

	// Get the permission_diff_suppression value from the plan
	var permissionDiffSuppression types.Bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, req.Path.ParentPath().AtName("permission_diff_suppression"), &permissionDiffSuppression)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply suppression logic if enabled
	if !permissionDiffSuppression.IsNull() && !permissionDiffSuppression.IsUnknown() && permissionDiffSuppression.ValueBool() {
		newValue := req.PlanValue.ValueString()
		if newValue == "triage" || newValue == "maintain" {
			resp.PlanValue = req.StateValue
		}
	}
}
