package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &githubRepositoryEnvironmentResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryEnvironmentResource{}
	_ resource.ResourceWithImportState = &githubRepositoryEnvironmentResource{}
)

func NewGithubRepositoryEnvironmentResource() resource.Resource {
	return &githubRepositoryEnvironmentResource{}
}

type githubRepositoryEnvironmentResource struct {
	client *Owner
}

type githubRepositoryEnvironmentResourceModel struct {
	Repository             types.String `tfsdk:"repository"`
	Environment            types.String `tfsdk:"environment"`
	CanAdminsBypass        types.Bool   `tfsdk:"can_admins_bypass"`
	PreventSelfReview      types.Bool   `tfsdk:"prevent_self_review"`
	WaitTimer              types.Int64  `tfsdk:"wait_timer"`
	Reviewers              types.List   `tfsdk:"reviewers"`
	DeploymentBranchPolicy types.List   `tfsdk:"deployment_branch_policy"`

	// Computed
	ID types.String `tfsdk:"id"`
}

type githubRepositoryEnvironmentReviewersModel struct {
	Teams types.Set `tfsdk:"teams"`
	Users types.Set `tfsdk:"users"`
}

type githubRepositoryEnvironmentDeploymentBranchPolicyModel struct {
	ProtectedBranches    types.Bool `tfsdk:"protected_branches"`
	CustomBranchPolicies types.Bool `tfsdk:"custom_branch_policies"`
}

func (r *githubRepositoryEnvironmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_environment"
}

func (r *githubRepositoryEnvironmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages environments for GitHub repositories",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the resource in the format 'repository:environment'",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The repository of the environment.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"environment": schema.StringAttribute{
				Description: "The name of the environment.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"can_admins_bypass": schema.BoolAttribute{
				Description: "Can Admins bypass deployment protections",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"prevent_self_review": schema.BoolAttribute{
				Description: "Prevent users from approving workflows runs that they triggered.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"wait_timer": schema.Int64Attribute{
				Description: "Amount of time to delay a job after the job is initially triggered.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 43200),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"reviewers": schema.ListAttribute{
				Description: "The environment reviewers configuration.",
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListNull(types.ObjectType{AttrTypes: reviewersAttrTypes()})),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				ElementType: types.ObjectType{
					AttrTypes: reviewersAttrTypes(),
				},
			},
			"deployment_branch_policy": schema.ListAttribute{
				Description: "The deployment branch policy configuration",
				Optional:    true,
				Computed:    true,
				Default:     listdefault.StaticValue(types.ListNull(types.ObjectType{AttrTypes: deploymentBranchPolicyAttrTypes()})),
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				ElementType: types.ObjectType{
					AttrTypes: deploymentBranchPolicyAttrTypes(),
				},
			},
		},
	}
}

func reviewersAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"teams": types.SetType{ElemType: types.Int64Type},
		"users": types.SetType{ElemType: types.Int64Type},
	}
}

func deploymentBranchPolicyAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"protected_branches":     types.BoolType,
		"custom_branch_policies": types.BoolType,
	}
}

func (r *githubRepositoryEnvironmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubRepositoryEnvironmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubRepositoryEnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := plan.Repository.ValueString()
	envName := plan.Environment.ValueString()
	escapedEnvName := url.PathEscape(envName)

	updateData, diags := r.createUpdateEnvironmentData(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.V3Client().Repositories.CreateUpdateEnvironment(ctx, owner, repoName, escapedEnvName, &updateData)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating repository environment",
			fmt.Sprintf("Could not create repository environment %s for repository %s: %s", envName, repoName, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", repoName, envName))

	// Read the environment to get the current state
	r.readEnvironment(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubRepositoryEnvironmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubRepositoryEnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readEnvironment(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubRepositoryEnvironmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubRepositoryEnvironmentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := plan.Repository.ValueString()
	envName := plan.Environment.ValueString()
	escapedEnvName := url.PathEscape(envName)

	updateData, diags := r.createUpdateEnvironmentData(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resultKey, _, err := r.client.V3Client().Repositories.CreateUpdateEnvironment(ctx, owner, repoName, escapedEnvName, &updateData)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating repository environment",
			fmt.Sprintf("Could not update repository environment %s for repository %s: %s", envName, repoName, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", repoName, resultKey.GetName()))

	// Read the environment to get the current state
	r.readEnvironment(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubRepositoryEnvironmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubRepositoryEnvironmentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	id := state.ID.ValueString()
	repoName, envName, err := r.parseTwoPartID(id, "repository", "environment")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error parsing resource ID",
			fmt.Sprintf("Could not parse resource ID %s: %s", id, err.Error()),
		)
		return
	}

	escapedEnvName := url.PathEscape(envName)

	_, err = r.client.V3Client().Repositories.DeleteEnvironment(ctx, owner, repoName, escapedEnvName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting repository environment",
			fmt.Sprintf("Could not delete repository environment %s for repository %s: %s", envName, repoName, err.Error()),
		)
		return
	}
}

func (r *githubRepositoryEnvironmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	repoName, envName, err := r.parseTwoPartID(req.ID, "repository", "environment")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing repository environment",
			fmt.Sprintf("Could not parse import ID %s: %s", req.ID, err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository"), repoName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment"), envName)...)
}

func (r *githubRepositoryEnvironmentResource) readEnvironment(ctx context.Context, state *githubRepositoryEnvironmentResourceModel, diags *diag.Diagnostics) {
	owner := r.client.Name()
	id := state.ID.ValueString()
	repoName, envName, err := r.parseTwoPartID(id, "repository", "environment")
	if err != nil {
		diags.AddError(
			"Error parsing resource ID",
			fmt.Sprintf("Could not parse resource ID %s: %s", id, err.Error()),
		)
		return
	}

	escapedEnvName := url.PathEscape(envName)

	env, _, err := r.client.V3Client().Repositories.GetEnvironment(ctx, owner, repoName, escapedEnvName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing repository environment %s from state because it no longer exists in GitHub", id)
				state.ID = types.StringValue("")
				return
			}
		}
		diags.AddError(
			"Error reading repository environment",
			fmt.Sprintf("Could not read repository environment %s for repository %s: %s", envName, repoName, err.Error()),
		)
		return
	}

	state.Repository = types.StringValue(repoName)
	state.Environment = types.StringValue(envName)
	state.CanAdminsBypass = types.BoolValue(env.GetCanAdminsBypass())

	// Reset wait_timer and reviewers - they'll be set below if found in protection rules
	state.WaitTimer = types.Int64Null()
	state.Reviewers = types.ListNull(types.ObjectType{AttrTypes: reviewersAttrTypes()})

	for _, pr := range env.ProtectionRules {
		switch pr.GetType() {
		case "wait_timer":
			if pr.WaitTimer != nil {
				state.WaitTimer = types.Int64Value(int64(*pr.WaitTimer))
			}

		case "required_reviewers":
			teams := make([]attr.Value, 0)
			users := make([]attr.Value, 0)

			for _, r := range pr.Reviewers {
				switch r.GetType() {
				case "Team":
					if team, ok := r.Reviewer.(*github.Team); ok && team.ID != nil {
						teams = append(teams, types.Int64Value(*team.ID))
					}
				case "User":
					if user, ok := r.Reviewer.(*github.User); ok && user.ID != nil {
						users = append(users, types.Int64Value(*user.ID))
					}
				}
			}

			teamsSet, setDiags := types.SetValue(types.Int64Type, teams)
			diags.Append(setDiags...)
			if diags.HasError() {
				return
			}

			usersSet, setDiags := types.SetValue(types.Int64Type, users)
			diags.Append(setDiags...)
			if diags.HasError() {
				return
			}

			reviewersObj, objDiags := types.ObjectValue(reviewersAttrTypes(), map[string]attr.Value{
				"teams": teamsSet,
				"users": usersSet,
			})
			diags.Append(objDiags...)
			if diags.HasError() {
				return
			}

			reviewersList, listDiags := types.ListValue(types.ObjectType{AttrTypes: reviewersAttrTypes()}, []attr.Value{reviewersObj})
			diags.Append(listDiags...)
			if diags.HasError() {
				return
			}

			state.Reviewers = reviewersList

			if pr.PreventSelfReview != nil {
				state.PreventSelfReview = types.BoolValue(*pr.PreventSelfReview)
			} else {
				state.PreventSelfReview = types.BoolValue(false)
			}
		}
	}

	if env.DeploymentBranchPolicy != nil {
		policyObj, objDiags := types.ObjectValue(deploymentBranchPolicyAttrTypes(), map[string]attr.Value{
			"protected_branches":     types.BoolValue(env.DeploymentBranchPolicy.GetProtectedBranches()),
			"custom_branch_policies": types.BoolValue(env.DeploymentBranchPolicy.GetCustomBranchPolicies()),
		})
		diags.Append(objDiags...)
		if diags.HasError() {
			return
		}

		policyList, listDiags := types.ListValue(types.ObjectType{AttrTypes: deploymentBranchPolicyAttrTypes()}, []attr.Value{policyObj})
		diags.Append(listDiags...)
		if diags.HasError() {
			return
		}

		state.DeploymentBranchPolicy = policyList
	} else {
		state.DeploymentBranchPolicy = types.ListNull(types.ObjectType{AttrTypes: deploymentBranchPolicyAttrTypes()})
	}
}

func (r *githubRepositoryEnvironmentResource) createUpdateEnvironmentData(ctx context.Context, plan githubRepositoryEnvironmentResourceModel) (github.CreateUpdateEnvironment, diag.Diagnostics) {
	var diags diag.Diagnostics
	data := github.CreateUpdateEnvironment{}

	if !plan.WaitTimer.IsNull() && !plan.WaitTimer.IsUnknown() {
		waitTimer := int(plan.WaitTimer.ValueInt64())
		data.WaitTimer = &waitTimer
	}

	data.CanAdminsBypass = github.Ptr(plan.CanAdminsBypass.ValueBool())
	data.PreventSelfReview = github.Ptr(plan.PreventSelfReview.ValueBool())

	if !plan.Reviewers.IsNull() && !plan.Reviewers.IsUnknown() {
		var reviewers []githubRepositoryEnvironmentReviewersModel
		diags.Append(plan.Reviewers.ElementsAs(ctx, &reviewers, false)...)
		if diags.HasError() {
			return data, diags
		}

		if len(reviewers) > 0 {
			envReviewers := make([]*github.EnvReviewers, 0)
			reviewer := reviewers[0]

			if !reviewer.Teams.IsNull() && !reviewer.Teams.IsUnknown() {
				var teams []types.Int64
				diags.Append(reviewer.Teams.ElementsAs(ctx, &teams, false)...)
				if diags.HasError() {
					return data, diags
				}

				for _, team := range teams {
					envReviewers = append(envReviewers, &github.EnvReviewers{
						Type: github.Ptr("Team"),
						ID:   github.Ptr(team.ValueInt64()),
					})
				}
			}

			if !reviewer.Users.IsNull() && !reviewer.Users.IsUnknown() {
				var users []types.Int64
				diags.Append(reviewer.Users.ElementsAs(ctx, &users, false)...)
				if diags.HasError() {
					return data, diags
				}

				for _, user := range users {
					envReviewers = append(envReviewers, &github.EnvReviewers{
						Type: github.Ptr("User"),
						ID:   github.Ptr(user.ValueInt64()),
					})
				}
			}

			data.Reviewers = envReviewers
		}
	}

	if !plan.DeploymentBranchPolicy.IsNull() && !plan.DeploymentBranchPolicy.IsUnknown() {
		var policies []githubRepositoryEnvironmentDeploymentBranchPolicyModel
		diags.Append(plan.DeploymentBranchPolicy.ElementsAs(ctx, &policies, false)...)
		if diags.HasError() {
			return data, diags
		}

		if len(policies) > 0 {
			policy := policies[0]
			data.DeploymentBranchPolicy = &github.BranchPolicy{
				ProtectedBranches:    github.Ptr(policy.ProtectedBranches.ValueBool()),
				CustomBranchPolicies: github.Ptr(policy.CustomBranchPolicies.ValueBool()),
			}
		}
	}

	return data, diags
}

func (r *githubRepositoryEnvironmentResource) parseTwoPartID(id, part1, part2 string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected ID in the format '%s:%s', got: %s", part1, part2, id)
	}
	return parts[0], parts[1], nil
}
