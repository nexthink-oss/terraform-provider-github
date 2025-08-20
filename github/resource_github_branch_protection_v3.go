package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &githubBranchProtectionV3Resource{}
	_ resource.ResourceWithConfigure   = &githubBranchProtectionV3Resource{}
	_ resource.ResourceWithImportState = &githubBranchProtectionV3Resource{}
)

func NewGithubBranchProtectionV3Resource() resource.Resource {
	return &githubBranchProtectionV3Resource{}
}

type githubBranchProtectionV3Resource struct {
	client *Owner
}

type githubBranchProtectionV3ResourceModel struct {
	// Required attributes
	Repository types.String `tfsdk:"repository"`
	Branch     types.String `tfsdk:"branch"`

	// Optional boolean attributes
	EnforceAdmins                 types.Bool `tfsdk:"enforce_admins"`
	RequireSignedCommits          types.Bool `tfsdk:"require_signed_commits"`
	RequireConversationResolution types.Bool `tfsdk:"require_conversation_resolution"`

	// Complex nested attributes
	RequiredStatusChecks       types.List `tfsdk:"required_status_checks"`
	RequiredPullRequestReviews types.List `tfsdk:"required_pull_request_reviews"`
	Restrictions               types.List `tfsdk:"restrictions"`

	// Computed attributes
	ID   types.String `tfsdk:"id"`
	Etag types.String `tfsdk:"etag"`
}

type requiredStatusChecksV3Model struct {
	Strict   types.Bool `tfsdk:"strict"`
	Contexts types.Set  `tfsdk:"contexts"`
	Checks   types.Set  `tfsdk:"checks"`
}

type requiredPullRequestReviewsV3Model struct {
	DismissStaleReviews          types.Bool  `tfsdk:"dismiss_stale_reviews"`
	DismissalUsers               types.Set   `tfsdk:"dismissal_users"`
	DismissalTeams               types.Set   `tfsdk:"dismissal_teams"`
	DismissalApps                types.Set   `tfsdk:"dismissal_apps"`
	RequireCodeOwnerReviews      types.Bool  `tfsdk:"require_code_owner_reviews"`
	RequiredApprovingReviewCount types.Int64 `tfsdk:"required_approving_review_count"`
	RequireLastPushApproval      types.Bool  `tfsdk:"require_last_push_approval"`
	BypassPullRequestAllowances  types.List  `tfsdk:"bypass_pull_request_allowances"`
}

type bypassPullRequestAllowancesV3Model struct {
	Users types.Set `tfsdk:"users"`
	Teams types.Set `tfsdk:"teams"`
	Apps  types.Set `tfsdk:"apps"`
}

type restrictionsV3Model struct {
	Users types.Set `tfsdk:"users"`
	Teams types.Set `tfsdk:"teams"`
	Apps  types.Set `tfsdk:"apps"`
}

func (r *githubBranchProtectionV3Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_protection_v3"
}

func (r *githubBranchProtectionV3Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Protects a GitHub branch using the v3 / REST implementation. The `github_branch_protection` resource has moved to the GraphQL API, while this resource will continue to leverage the REST API",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the branch protection rule.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The GitHub repository name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"branch": schema.StringAttribute{
				Description: "The Git branch to protect.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enforce_admins": schema.BoolAttribute{
				Description: "Setting this to 'true' enforces status checks for repository administrators.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"require_signed_commits": schema.BoolAttribute{
				Description: "Setting this to 'true' requires all commits to be signed with GPG.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"require_conversation_resolution": schema.BoolAttribute{
				Description: "Setting this to 'true' requires all conversations on code must be resolved before a pull request can be merged.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"etag": schema.StringAttribute{
				Description: "The ETag of the branch protection rule.",
				Computed:    true,
			},
			"required_status_checks": schema.ListNestedAttribute{
				Description: "Enforce restrictions for required status checks.",
				Optional:    true,
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"strict": schema.BoolAttribute{
							Description: "Require branches to be up to date before merging.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"contexts": schema.SetAttribute{
							Description:        "The list of status checks to require in order to merge into this branch. No status checks are required by default.",
							Optional:           true,
							ElementType:        types.StringType,
							DeprecationMessage: "GitHub is deprecating the use of `contexts`. Use a `checks` array instead.",
						},
						"checks": schema.SetAttribute{
							Description: "The list of status checks to require in order to merge into this branch. No status checks are required by default. Checks should be strings containing the 'context' and 'app_id' like so 'context:app_id'",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"required_pull_request_reviews": schema.ListNestedAttribute{
				Description: "Enforce restrictions for pull request reviews.",
				Optional:    true,
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"dismiss_stale_reviews": schema.BoolAttribute{
							Description: "Dismiss approved reviews automatically when a new commit is pushed.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"dismissal_users": schema.SetAttribute{
							Description: "The list of user logins with dismissal access.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"dismissal_teams": schema.SetAttribute{
							Description: "The list of team slugs with dismissal access. Always use slug of the team, not its name. Each team already has to have access to the repository.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"dismissal_apps": schema.SetAttribute{
							Description: "The list of apps slugs with dismissal access. Always use slug of the app, not its name. Each app already has to have access to the repository.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"require_code_owner_reviews": schema.BoolAttribute{
							Description: "Require an approved review in pull requests including files with a designated code owner.",
							Optional:    true,
						},
						"required_approving_review_count": schema.Int64Attribute{
							Description: "Require 'x' number of approvals to satisfy branch protection requirements. If this is specified it must be a number between 0-6.",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(1),
							Validators: []validator.Int64{
								int64validator.Between(0, 6),
							},
						},
						"require_last_push_approval": schema.BoolAttribute{
							Description: "Require that the most recent push must be approved by someone other than the last pusher.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"bypass_pull_request_allowances": schema.ListNestedAttribute{
							Description: "Allow specific users, teams, or apps to bypass pull request requirements.",
							Optional:    true,
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"users": schema.SetAttribute{
										Description: "The list of user logins allowed to bypass pull request requirements.",
										Optional:    true,
										ElementType: types.StringType,
									},
									"teams": schema.SetAttribute{
										Description: "The list of team slugs allowed to bypass pull request requirements.",
										Optional:    true,
										ElementType: types.StringType,
									},
									"apps": schema.SetAttribute{
										Description: "The list of app slugs allowed to bypass pull request requirements.",
										Optional:    true,
										ElementType: types.StringType,
									},
								},
							},
						},
					},
				},
			},
			"restrictions": schema.ListNestedAttribute{
				Description: "Enforce restrictions for the users and teams that may push to the branch.",
				Optional:    true,
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"users": schema.SetAttribute{
							Description: "The list of user logins with push access.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"teams": schema.SetAttribute{
							Description: "The list of team slugs with push access. Always use slug of the team, not its name. Each team already has to have access to the repository.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"apps": schema.SetAttribute{
							Description: "The list of app slugs with push access.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *githubBranchProtectionV3Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubBranchProtectionV3Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubBranchProtectionV3ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	client := r.client.V3Client()
	orgName := r.client.Name()
	repoName := plan.Repository.ValueString()
	branch := plan.Branch.ValueString()

	protectionRequest, diags := r.buildProtectionRequest(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	protection, _, err := client.Repositories.UpdateBranchProtection(ctx,
		orgName,
		repoName,
		branch,
		protectionRequest,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating branch protection",
			fmt.Sprintf("Could not create branch protection for %s/%s:%s, unexpected error: %s", orgName, repoName, branch, err),
		)
		return
	}

	if err := r.checkBranchRestrictionsUsers(protection.GetRestrictions(), protectionRequest.GetRestrictions()); err != nil {
		resp.Diagnostics.AddError(
			"Error validating branch restrictions",
			fmt.Sprintf("Branch protection created but restrictions validation failed: %s", err),
		)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", repoName, branch))

	if err := r.requireSignedCommitsUpdate(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error setting signed commits requirement",
			fmt.Sprintf("Could not update signed commits requirement: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(r.readResourceData(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *githubBranchProtectionV3Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubBranchProtectionV3ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.readResourceData(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *githubBranchProtectionV3Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubBranchProtectionV3ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	client := r.client.V3Client()
	repoName := plan.Repository.ValueString()
	branch := plan.Branch.ValueString()

	protectionRequest, diags := r.buildProtectionRequest(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgName := r.client.Name()

	protection, _, err := client.Repositories.UpdateBranchProtection(ctx,
		orgName,
		repoName,
		branch,
		protectionRequest,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating branch protection",
			fmt.Sprintf("Could not update branch protection for %s/%s:%s, unexpected error: %s", orgName, repoName, branch, err),
		)
		return
	}

	if err := r.checkBranchRestrictionsUsers(protection.GetRestrictions(), protectionRequest.GetRestrictions()); err != nil {
		resp.Diagnostics.AddError(
			"Error validating branch restrictions",
			fmt.Sprintf("Branch protection updated but restrictions validation failed: %s", err),
		)
		return
	}

	if protectionRequest.RequiredPullRequestReviews == nil {
		_, err = client.Repositories.RemovePullRequestReviewEnforcement(ctx,
			orgName,
			repoName,
			branch,
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error removing pull request review enforcement",
				fmt.Sprintf("Could not remove pull request review enforcement: %s", err),
			)
			return
		}
	}

	if err := r.requireSignedCommitsUpdate(ctx, &plan); err != nil {
		resp.Diagnostics.AddError(
			"Error updating signed commits requirement",
			fmt.Sprintf("Could not update signed commits requirement: %s", err),
		)
		return
	}

	resp.Diagnostics.Append(r.readResourceData(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *githubBranchProtectionV3Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubBranchProtectionV3ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	client := r.client.V3Client()
	repoName := state.Repository.ValueString()
	branch := state.Branch.ValueString()
	orgName := r.client.Name()

	_, err := client.Repositories.RemoveBranchProtection(ctx, orgName, repoName, branch)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting branch protection",
			fmt.Sprintf("Could not delete branch protection for %s/%s:%s, unexpected error: %s", orgName, repoName, branch, err),
		)
		return
	}
}

func (r *githubBranchProtectionV3Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: repository:branch. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("branch"), idParts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// Helper methods for resource operations

func (r *githubBranchProtectionV3Resource) readResourceData(ctx context.Context, data *githubBranchProtectionV3ResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if !r.client.IsOrganization {
		diags.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return diags
	}

	client := r.client.V3Client()
	repoName := data.Repository.ValueString()
	branch := data.Branch.ValueString()
	orgName := r.client.Name()

	githubProtection, resp, err := client.Repositories.GetBranchProtection(ctx,
		orgName, repoName, branch)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				if err := r.requireSignedCommitsRead(ctx, data); err != nil {
					diags.AddError(
						"Error reading signed commit restriction",
						fmt.Sprintf("Could not read signed commit restriction: %s", err),
					)
					return diags
				}
				return diags
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing branch protection %s/%s (%s) from state because it no longer exists in GitHub",
					orgName, repoName, branch)
				data.ID = types.StringValue("")
				return diags
			}
		}

		diags.AddError(
			"Error reading branch protection",
			fmt.Sprintf("Could not read branch protection for %s/%s:%s, unexpected error: %s", orgName, repoName, branch, err),
		)
		return diags
	}

	data.Etag = types.StringValue(resp.Header.Get("ETag"))
	data.Repository = types.StringValue(repoName)
	data.Branch = types.StringValue(branch)
	data.EnforceAdmins = types.BoolValue(githubProtection.GetEnforceAdmins().Enabled)

	if rcr := githubProtection.GetRequiredConversationResolution(); rcr != nil {
		data.RequireConversationResolution = types.BoolValue(rcr.Enabled)
	} else {
		data.RequireConversationResolution = types.BoolValue(false)
	}

	if err := r.flattenAndSetRequiredStatusChecks(ctx, data, githubProtection); err != nil {
		diags.AddError(
			"Error setting required_status_checks",
			fmt.Sprintf("Could not set required_status_checks: %s", err),
		)
		return diags
	}

	if err := r.flattenAndSetRequiredPullRequestReviews(ctx, data, githubProtection); err != nil {
		diags.AddError(
			"Error setting required_pull_request_reviews",
			fmt.Sprintf("Could not set required_pull_request_reviews: %s", err),
		)
		return diags
	}

	if err := r.flattenAndSetRestrictions(ctx, data, githubProtection); err != nil {
		diags.AddError(
			"Error setting restrictions",
			fmt.Sprintf("Could not set restrictions: %s", err),
		)
		return diags
	}

	if err := r.requireSignedCommitsRead(ctx, data); err != nil {
		diags.AddError(
			"Error reading signed commit restriction",
			fmt.Sprintf("Could not read signed commit restriction: %s", err),
		)
		return diags
	}

	return diags
}

func (r *githubBranchProtectionV3Resource) buildProtectionRequest(ctx context.Context, data *githubBranchProtectionV3ResourceModel) (*github.ProtectionRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	req := &github.ProtectionRequest{
		EnforceAdmins:                  data.EnforceAdmins.ValueBool(),
		RequiredConversationResolution: github.Ptr(data.RequireConversationResolution.ValueBool()),
	}

	rsc, err := r.expandRequiredStatusChecks(ctx, data)
	if err != nil {
		diags.AddError(
			"Error expanding required status checks",
			fmt.Sprintf("Could not expand required status checks: %s", err),
		)
		return nil, diags
	}
	req.RequiredStatusChecks = rsc

	rprr, err := r.expandRequiredPullRequestReviews(ctx, data)
	if err != nil {
		diags.AddError(
			"Error expanding required pull request reviews",
			fmt.Sprintf("Could not expand required pull request reviews: %s", err),
		)
		return nil, diags
	}
	req.RequiredPullRequestReviews = rprr

	res, err := r.expandRestrictions(ctx, data)
	if err != nil {
		diags.AddError(
			"Error expanding restrictions",
			fmt.Sprintf("Could not expand restrictions: %s", err),
		)
		return nil, diags
	}
	req.Restrictions = res

	return req, diags
}

func (r *githubBranchProtectionV3Resource) requireSignedCommitsRead(ctx context.Context, data *githubBranchProtectionV3ResourceModel) error {
	client := r.client.V3Client()
	repoName := data.Repository.ValueString()
	branch := data.Branch.ValueString()
	orgName := r.client.Name()

	signedCommitStatus, _, err := client.Repositories.GetSignaturesProtectedBranch(ctx,
		orgName, repoName, branch)
	if err != nil {
		log.Printf("[INFO] Not able to read signature protection: %s/%s (%s)", orgName, repoName, branch)
		data.RequireSignedCommits = types.BoolValue(false)
		return nil
	}

	data.RequireSignedCommits = types.BoolValue(signedCommitStatus.GetEnabled())
	return nil
}

func (r *githubBranchProtectionV3Resource) requireSignedCommitsUpdate(ctx context.Context, data *githubBranchProtectionV3ResourceModel) error {
	requiredSignedCommit := data.RequireSignedCommits.ValueBool()
	client := r.client.V3Client()
	repoName := data.Repository.ValueString()
	branch := data.Branch.ValueString()
	orgName := r.client.Name()

	if requiredSignedCommit {
		_, _, err := client.Repositories.RequireSignaturesOnProtectedBranch(ctx, orgName, repoName, branch)
		return err
	} else {
		_, err := client.Repositories.OptionalSignaturesOnProtectedBranch(ctx, orgName, repoName, branch)
		return err
	}
}

func (r *githubBranchProtectionV3Resource) checkBranchRestrictionsUsers(actual *github.BranchRestrictions, expected *github.BranchRestrictionsRequest) error {
	if expected == nil {
		return nil
	}

	expectedUsers := expected.Users

	if actual == nil {
		return fmt.Errorf("unable to add users in restrictions: %s", strings.Join(expectedUsers, ", "))
	}

	actualLoopUp := make(map[string]struct{}, len(actual.Users))
	for _, a := range actual.Users {
		actualLoopUp[a.GetLogin()] = struct{}{}
	}

	notFounds := make([]string, 0, len(actual.Users))

	for _, e := range expectedUsers {
		if _, ok := actualLoopUp[e]; !ok {
			notFounds = append(notFounds, e)
		}
	}

	if len(notFounds) == 0 {
		return nil
	}

	return fmt.Errorf("unable to add users in restrictions: %s", strings.Join(notFounds, ", "))
}

// Helper functions for converting between Framework types and GitHub API types

func (r *githubBranchProtectionV3Resource) expandNestedSet(ctx context.Context, input types.Set) []string {
	if input.IsNull() || input.IsUnknown() {
		return nil
	}

	var result []string
	for _, elem := range input.Elements() {
		if str, ok := elem.(types.String); ok {
			result = append(result, str.ValueString())
		}
	}
	return result
}

func (r *githubBranchProtectionV3Resource) expandRequiredStatusChecks(ctx context.Context, data *githubBranchProtectionV3ResourceModel) (*github.RequiredStatusChecks, error) {
	if data.RequiredStatusChecks.IsNull() || data.RequiredStatusChecks.IsUnknown() {
		return nil, nil
	}

	var rscList []requiredStatusChecksV3Model
	diags := data.RequiredStatusChecks.ElementsAs(ctx, &rscList, false)
	if diags.HasError() {
		return nil, errors.New("could not parse required_status_checks")
	}

	if len(rscList) == 0 {
		return nil, nil
	}

	rsc := &github.RequiredStatusChecks{
		Strict: rscList[0].Strict.ValueBool(),
	}

	// Initialise empty literal to ensure an empty array is passed mitigating schema errors
	rscChecks := []*github.RequiredStatusCheck{}

	// TODO: Remove once contexts is deprecated
	// Iterate and parse contexts into checks using -1 as default to allow checks from all apps.
	contexts := r.expandNestedSet(ctx, rscList[0].Contexts)
	for _, c := range contexts {
		appID := int64(-1) // Default
		rscChecks = append(rscChecks, &github.RequiredStatusCheck{
			Context: c,
			AppID:   &appID,
		})
	}

	// Iterate and parse checks
	checks := r.expandNestedSet(ctx, rscList[0].Checks)
	for _, c := range checks {
		// Expect a string of "context:app_id", allowing for the absence of "app_id"
		index := strings.LastIndex(c, ":")
		var cContext, cAppId string
		if index <= 0 {
			// If there is no ":" or it's in the first position, there is no app_id.
			cContext, cAppId = c, ""
		} else {
			cContext, cAppId = c[:index], c[index+1:]
		}

		var rscCheck *github.RequiredStatusCheck
		if cAppId != "" {
			// If we have a valid app_id, include it in the RSC
			rscAppId, err := strconv.Atoi(cAppId)
			if err != nil {
				return nil, fmt.Errorf("could not parse %v as valid app_id", cAppId)
			}
			rscAppId64 := int64(rscAppId)
			rscCheck = &github.RequiredStatusCheck{Context: cContext, AppID: &rscAppId64}
		} else {
			// Else simply provide the context
			rscCheck = &github.RequiredStatusCheck{Context: cContext}
		}

		// Append
		rscChecks = append(rscChecks, rscCheck)
	}
	// Assign after looping both checks and contexts
	rsc.Checks = &rscChecks

	return rsc, nil
}

func (r *githubBranchProtectionV3Resource) expandRequiredPullRequestReviews(ctx context.Context, data *githubBranchProtectionV3ResourceModel) (*github.PullRequestReviewsEnforcementRequest, error) {
	if data.RequiredPullRequestReviews.IsNull() || data.RequiredPullRequestReviews.IsUnknown() {
		return nil, nil
	}

	var rprrList []requiredPullRequestReviewsV3Model
	diags := data.RequiredPullRequestReviews.ElementsAs(ctx, &rprrList, false)
	if diags.HasError() {
		return nil, errors.New("could not parse required_pull_request_reviews")
	}

	if len(rprrList) == 0 {
		return nil, nil
	}

	rprr := &github.PullRequestReviewsEnforcementRequest{}
	drr := &github.DismissalRestrictionsRequest{}

	users := r.expandNestedSet(ctx, rprrList[0].DismissalUsers)
	if len(users) > 0 {
		drr.Users = &users
	}
	teams := r.expandNestedSet(ctx, rprrList[0].DismissalTeams)
	if len(teams) > 0 {
		drr.Teams = &teams
	}
	apps := r.expandNestedSet(ctx, rprrList[0].DismissalApps)
	if len(apps) > 0 {
		drr.Apps = &apps
	}

	bpra, err := r.expandBypassPullRequestAllowances(ctx, rprrList[0].BypassPullRequestAllowances)
	if err != nil {
		return nil, err
	}

	rprr.DismissalRestrictionsRequest = drr
	rprr.DismissStaleReviews = rprrList[0].DismissStaleReviews.ValueBool()
	rprr.RequireCodeOwnerReviews = rprrList[0].RequireCodeOwnerReviews.ValueBool()
	rprr.RequiredApprovingReviewCount = int(rprrList[0].RequiredApprovingReviewCount.ValueInt64())
	requireLastPushApproval := rprrList[0].RequireLastPushApproval.ValueBool()
	rprr.RequireLastPushApproval = &requireLastPushApproval
	rprr.BypassPullRequestAllowancesRequest = bpra

	return rprr, nil
}

func (r *githubBranchProtectionV3Resource) expandRestrictions(ctx context.Context, data *githubBranchProtectionV3ResourceModel) (*github.BranchRestrictionsRequest, error) {
	if data.Restrictions.IsNull() || data.Restrictions.IsUnknown() {
		return nil, nil
	}

	var restrictionsList []restrictionsV3Model
	diags := data.Restrictions.ElementsAs(ctx, &restrictionsList, false)
	if diags.HasError() {
		return nil, errors.New("could not parse restrictions")
	}

	if len(restrictionsList) == 0 {
		return nil, nil
	}

	restrictions := &github.BranchRestrictionsRequest{}

	if restrictionsList[0].Users.IsNull() && restrictionsList[0].Teams.IsNull() && restrictionsList[0].Apps.IsNull() {
		// Restrictions only have set attributes nested, need to return nil values for these.
		// The API won't initialize these as nil
		restrictions.Users = []string{}
		restrictions.Teams = []string{}
		restrictions.Apps = []string{}
		return restrictions, nil
	}

	users := r.expandNestedSet(ctx, restrictionsList[0].Users)
	restrictions.Users = users
	teams := r.expandNestedSet(ctx, restrictionsList[0].Teams)
	restrictions.Teams = teams
	apps := r.expandNestedSet(ctx, restrictionsList[0].Apps)
	restrictions.Apps = apps

	return restrictions, nil
}

func (r *githubBranchProtectionV3Resource) expandBypassPullRequestAllowances(ctx context.Context, input types.List) (*github.BypassPullRequestAllowancesRequest, error) {
	if input.IsNull() || input.IsUnknown() {
		return nil, nil
	}

	var bpraList []bypassPullRequestAllowancesV3Model
	diags := input.ElementsAs(ctx, &bpraList, false)
	if diags.HasError() {
		return nil, errors.New("could not parse bypass_pull_request_allowances")
	}

	if len(bpraList) == 0 {
		return nil, nil
	}

	bpra := &github.BypassPullRequestAllowancesRequest{}

	users := r.expandNestedSet(ctx, bpraList[0].Users)
	bpra.Users = users
	teams := r.expandNestedSet(ctx, bpraList[0].Teams)
	bpra.Teams = teams
	apps := r.expandNestedSet(ctx, bpraList[0].Apps)
	bpra.Apps = apps

	return bpra, nil
}

func (r *githubBranchProtectionV3Resource) flattenAndSetRequiredStatusChecks(ctx context.Context, data *githubBranchProtectionV3ResourceModel, protection *github.Protection) error {
	rsc := protection.GetRequiredStatusChecks()

	if rsc != nil {
		// Contexts and Checks arrays to flatten into
		var contexts []attr.Value
		var checks []attr.Value

		// TODO: Remove once contexts is fully deprecated.
		// Flatten contexts
		for _, c := range *rsc.Contexts {
			contexts = append(contexts, types.StringValue(c))
		}

		// Flatten checks
		for _, chk := range *rsc.Checks {
			if chk.AppID != nil {
				checks = append(checks, types.StringValue(fmt.Sprintf("%s:%d", chk.Context, *chk.AppID)))
			} else {
				checks = append(checks, types.StringValue(chk.Context))
			}
		}

		contextsSet, _ := types.SetValue(types.StringType, contexts)
		checksSet, _ := types.SetValue(types.StringType, checks)

		rscModel := requiredStatusChecksV3Model{
			Strict:   types.BoolValue(rsc.Strict),
			Contexts: contextsSet,
			Checks:   checksSet,
		}

		listValue, _ := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"strict":   types.BoolType,
				"contexts": types.SetType{ElemType: types.StringType},
				"checks":   types.SetType{ElemType: types.StringType},
			},
		}, []requiredStatusChecksV3Model{rscModel})

		data.RequiredStatusChecks = listValue
	} else {
		emptyList, _ := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"strict":   types.BoolType,
				"contexts": types.SetType{ElemType: types.StringType},
				"checks":   types.SetType{ElemType: types.StringType},
			},
		}, []attr.Value{})
		data.RequiredStatusChecks = emptyList
	}

	return nil
}

func (r *githubBranchProtectionV3Resource) flattenAndSetRequiredPullRequestReviews(ctx context.Context, data *githubBranchProtectionV3ResourceModel, protection *github.Protection) error {
	rprr := protection.GetRequiredPullRequestReviews()
	if rprr != nil {
		var users, teams, apps []attr.Value
		restrictions := rprr.GetDismissalRestrictions()

		if restrictions != nil {
			for _, u := range restrictions.Users {
				if u.Login != nil {
					users = append(users, types.StringValue(*u.Login))
				}
			}
			for _, t := range restrictions.Teams {
				if t.Slug != nil {
					teams = append(teams, types.StringValue(*t.Slug))
				}
			}
			for _, t := range restrictions.Apps {
				if t.Slug != nil {
					apps = append(apps, types.StringValue(*t.Slug))
				}
			}
		}

		usersSet, _ := types.SetValue(types.StringType, users)
		teamsSet, _ := types.SetValue(types.StringType, teams)
		appsSet, _ := types.SetValue(types.StringType, apps)

		bpra := r.flattenBypassPullRequestAllowances(ctx, rprr.GetBypassPullRequestAllowances())

		rprrModel := requiredPullRequestReviewsV3Model{
			DismissStaleReviews:          types.BoolValue(rprr.DismissStaleReviews),
			DismissalUsers:               usersSet,
			DismissalTeams:               teamsSet,
			DismissalApps:                appsSet,
			RequireCodeOwnerReviews:      types.BoolValue(rprr.RequireCodeOwnerReviews),
			RequireLastPushApproval:      types.BoolValue(rprr.RequireLastPushApproval),
			RequiredApprovingReviewCount: types.Int64Value(int64(rprr.RequiredApprovingReviewCount)),
			BypassPullRequestAllowances:  bpra,
		}

		listValue, _ := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"dismiss_stale_reviews":           types.BoolType,
				"dismissal_users":                 types.SetType{ElemType: types.StringType},
				"dismissal_teams":                 types.SetType{ElemType: types.StringType},
				"dismissal_apps":                  types.SetType{ElemType: types.StringType},
				"require_code_owner_reviews":      types.BoolType,
				"require_last_push_approval":      types.BoolType,
				"required_approving_review_count": types.Int64Type,
				"bypass_pull_request_allowances": types.ListType{ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"users": types.SetType{ElemType: types.StringType},
						"teams": types.SetType{ElemType: types.StringType},
						"apps":  types.SetType{ElemType: types.StringType},
					},
				}},
			},
		}, []requiredPullRequestReviewsV3Model{rprrModel})

		data.RequiredPullRequestReviews = listValue
	} else {
		emptyList, _ := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"dismiss_stale_reviews":           types.BoolType,
				"dismissal_users":                 types.SetType{ElemType: types.StringType},
				"dismissal_teams":                 types.SetType{ElemType: types.StringType},
				"dismissal_apps":                  types.SetType{ElemType: types.StringType},
				"require_code_owner_reviews":      types.BoolType,
				"require_last_push_approval":      types.BoolType,
				"required_approving_review_count": types.Int64Type,
				"bypass_pull_request_allowances": types.ListType{ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"users": types.SetType{ElemType: types.StringType},
						"teams": types.SetType{ElemType: types.StringType},
						"apps":  types.SetType{ElemType: types.StringType},
					},
				}},
			},
		}, []attr.Value{})
		data.RequiredPullRequestReviews = emptyList
	}

	return nil
}

func (r *githubBranchProtectionV3Resource) flattenAndSetRestrictions(ctx context.Context, data *githubBranchProtectionV3ResourceModel, protection *github.Protection) error {
	restrictions := protection.GetRestrictions()
	if restrictions != nil {
		var users, teams, apps []attr.Value

		for _, u := range restrictions.Users {
			if u.Login != nil {
				users = append(users, types.StringValue(*u.Login))
			}
		}

		for _, t := range restrictions.Teams {
			if t.Slug != nil {
				teams = append(teams, types.StringValue(*t.Slug))
			}
		}

		for _, t := range restrictions.Apps {
			if t.Slug != nil {
				apps = append(apps, types.StringValue(*t.Slug))
			}
		}

		usersSet, _ := types.SetValue(types.StringType, users)
		teamsSet, _ := types.SetValue(types.StringType, teams)
		appsSet, _ := types.SetValue(types.StringType, apps)

		restrictionsModel := restrictionsV3Model{
			Users: usersSet,
			Teams: teamsSet,
			Apps:  appsSet,
		}

		listValue, _ := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"users": types.SetType{ElemType: types.StringType},
				"teams": types.SetType{ElemType: types.StringType},
				"apps":  types.SetType{ElemType: types.StringType},
			},
		}, []restrictionsV3Model{restrictionsModel})

		data.Restrictions = listValue
	} else {
		emptyList, _ := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"users": types.SetType{ElemType: types.StringType},
				"teams": types.SetType{ElemType: types.StringType},
				"apps":  types.SetType{ElemType: types.StringType},
			},
		}, []attr.Value{})
		data.Restrictions = emptyList
	}

	return nil
}

func (r *githubBranchProtectionV3Resource) flattenBypassPullRequestAllowances(ctx context.Context, bpra *github.BypassPullRequestAllowances) types.List {
	if bpra == nil {
		emptyList, _ := types.ListValue(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"users": types.SetType{ElemType: types.StringType},
				"teams": types.SetType{ElemType: types.StringType},
				"apps":  types.SetType{ElemType: types.StringType},
			},
		}, []attr.Value{})
		return emptyList
	}

	var users, teams, apps []attr.Value

	for _, u := range bpra.Users {
		if u.Login != nil {
			users = append(users, types.StringValue(*u.Login))
		}
	}

	for _, t := range bpra.Teams {
		if t.Slug != nil {
			teams = append(teams, types.StringValue(*t.Slug))
		}
	}

	for _, t := range bpra.Apps {
		if t.Slug != nil {
			apps = append(apps, types.StringValue(*t.Slug))
		}
	}

	usersSet, _ := types.SetValue(types.StringType, users)
	teamsSet, _ := types.SetValue(types.StringType, teams)
	appsSet, _ := types.SetValue(types.StringType, apps)

	bpraModel := bypassPullRequestAllowancesV3Model{
		Users: usersSet,
		Teams: teamsSet,
		Apps:  appsSet,
	}

	listValue, _ := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"users": types.SetType{ElemType: types.StringType},
			"teams": types.SetType{ElemType: types.StringType},
			"apps":  types.SetType{ElemType: types.StringType},
		},
	}, []bypassPullRequestAllowancesV3Model{bpraModel})

	return listValue
}
