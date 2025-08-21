package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                     = &githubRepositoryEnvironmentDeploymentPolicyResource{}
	_ resource.ResourceWithConfigure        = &githubRepositoryEnvironmentDeploymentPolicyResource{}
	_ resource.ResourceWithImportState      = &githubRepositoryEnvironmentDeploymentPolicyResource{}
	_ resource.ResourceWithConfigValidators = &githubRepositoryEnvironmentDeploymentPolicyResource{}
)

func NewGithubRepositoryEnvironmentDeploymentPolicyResource() resource.Resource {
	return &githubRepositoryEnvironmentDeploymentPolicyResource{}
}

type githubRepositoryEnvironmentDeploymentPolicyResource struct {
	client *Owner
}

type githubRepositoryEnvironmentDeploymentPolicyResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Repository    types.String `tfsdk:"repository"`
	Environment   types.String `tfsdk:"environment"`
	BranchPattern types.String `tfsdk:"branch_pattern"`
	TagPattern    types.String `tfsdk:"tag_pattern"`
}

func (r *githubRepositoryEnvironmentDeploymentPolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_environment_deployment_policy"
}

func (r *githubRepositoryEnvironmentDeploymentPolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages environment deployment policies for GitHub repositories.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the deployment policy.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository. The name is not case sensitive.",
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
			"branch_pattern": schema.StringAttribute{
				Description: "The name pattern that branches must match in order to deploy to the environment.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					&patternTypeSwitchPlanModifier{},
				},
			},
			"tag_pattern": schema.StringAttribute{
				Description: "The name pattern that tags must match in order to deploy to the environment.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					&patternTypeSwitchPlanModifier{},
				},
			},
		},
	}
}

func (r *githubRepositoryEnvironmentDeploymentPolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubRepositoryEnvironmentDeploymentPolicyResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		// Ensure exactly one of branch_pattern or tag_pattern is specified
		&mutuallyExclusiveValidator{
			Attributes: []string{"branch_pattern", "tag_pattern"},
		},
	}
}

func (r *githubRepositoryEnvironmentDeploymentPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubRepositoryEnvironmentDeploymentPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repository := data.Repository.ValueString()
	environment := data.Environment.ValueString()
	escapedEnvName := url.PathEscape(environment)

	var createData github.DeploymentBranchPolicyRequest
	if !data.BranchPattern.IsNull() && !data.BranchPattern.IsUnknown() {
		createData = github.DeploymentBranchPolicyRequest{
			Name: github.Ptr(data.BranchPattern.ValueString()),
			Type: github.Ptr("branch"),
		}
	} else if !data.TagPattern.IsNull() && !data.TagPattern.IsUnknown() {
		createData = github.DeploymentBranchPolicyRequest{
			Name: github.Ptr(data.TagPattern.ValueString()),
			Type: github.Ptr("tag"),
		}
	} else {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Exactly one of branch_pattern and tag_pattern must be specified",
		)
		return
	}

	policy, _, err := client.Repositories.CreateDeploymentBranchPolicy(ctx, owner, repository, escapedEnvName, &createData)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating deployment policy",
			fmt.Sprintf("Could not create deployment policy for environment %s in repository %s: %s", environment, repository, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(buildThreePartID(repository, escapedEnvName, strconv.FormatInt(policy.GetID(), 10)))

	tflog.Debug(ctx, "created GitHub repository environment deployment policy", map[string]any{
		"id":          data.ID.ValueString(),
		"repository":  repository,
		"environment": environment,
		"policy_id":   policy.GetID(),
	})

	// Read the created resource to populate all computed fields
	r.readGithubRepositoryEnvironmentDeploymentPolicy(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryEnvironmentDeploymentPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubRepositoryEnvironmentDeploymentPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubRepositoryEnvironmentDeploymentPolicy(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryEnvironmentDeploymentPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubRepositoryEnvironmentDeploymentPolicyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repository, envName, branchPolicyIdString, err := parseThreePartID(data.ID.ValueString(), "repository", "environment", "branchPolicyId")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID Format",
			fmt.Sprintf("Unable to parse deployment policy ID: %s", err.Error()),
		)
		return
	}

	branchPolicyId, err := strconv.ParseInt(branchPolicyIdString, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID Format",
			fmt.Sprintf("Unable to convert deployment policy ID to integer: %s", err.Error()),
		)
		return
	}

	var pattern string
	if !data.BranchPattern.IsNull() && !data.BranchPattern.IsUnknown() {
		pattern = data.BranchPattern.ValueString()
	} else if !data.TagPattern.IsNull() && !data.TagPattern.IsUnknown() {
		pattern = data.TagPattern.ValueString()
	} else {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Exactly one of branch_pattern and tag_pattern must be specified",
		)
		return
	}

	updateData := github.DeploymentBranchPolicyRequest{
		Name: github.Ptr(pattern),
	}

	resultKey, _, err := client.Repositories.UpdateDeploymentBranchPolicy(ctx, owner, repository, envName, branchPolicyId, &updateData)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating deployment policy",
			fmt.Sprintf("Could not update deployment policy %d for environment %s in repository %s: %s", branchPolicyId, envName, repository, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(buildThreePartID(repository, envName, strconv.FormatInt(resultKey.GetID(), 10)))

	tflog.Debug(ctx, "updated GitHub repository environment deployment policy", map[string]any{
		"id":          data.ID.ValueString(),
		"repository":  repository,
		"environment": envName,
		"policy_id":   resultKey.GetID(),
	})

	// Read the updated resource to populate all computed fields
	r.readGithubRepositoryEnvironmentDeploymentPolicy(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryEnvironmentDeploymentPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubRepositoryEnvironmentDeploymentPolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repository, envName, branchPolicyIdString, err := parseThreePartID(data.ID.ValueString(), "repository", "environment", "branchPolicyId")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID Format",
			fmt.Sprintf("Unable to parse deployment policy ID: %s", err.Error()),
		)
		return
	}

	branchPolicyId, err := strconv.ParseInt(branchPolicyIdString, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID Format",
			fmt.Sprintf("Unable to convert deployment policy ID to integer: %s", err.Error()),
		)
		return
	}

	_, err = client.Repositories.DeleteDeploymentBranchPolicy(ctx, owner, repository, envName, branchPolicyId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting deployment policy",
			fmt.Sprintf("Could not delete deployment policy %d for environment %s in repository %s: %s", branchPolicyId, envName, repository, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub repository environment deployment policy", map[string]any{
		"id":          data.ID.ValueString(),
		"repository":  repository,
		"environment": envName,
		"policy_id":   branchPolicyId,
	})
}

func (r *githubRepositoryEnvironmentDeploymentPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "repository:environment:id"
	parts := strings.Split(req.ID, ":")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID in the format 'repository:environment:id', got: %s", req.ID),
		)
		return
	}

	repository := parts[0]
	environment := parts[1]
	id := parts[2]

	data := githubRepositoryEnvironmentDeploymentPolicyResourceModel{
		ID:          types.StringValue(buildThreePartID(repository, environment, id)),
		Repository:  types.StringValue(repository),
		Environment: types.StringValue(environment),
	}

	r.readGithubRepositoryEnvironmentDeploymentPolicy(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Helper function to read the deployment policy
func (r *githubRepositoryEnvironmentDeploymentPolicyResource) readGithubRepositoryEnvironmentDeploymentPolicy(ctx context.Context, data *githubRepositoryEnvironmentDeploymentPolicyResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	repository, envName, branchPolicyIdString, err := parseThreePartID(data.ID.ValueString(), "repository", "environment", "branchPolicyId")
	if err != nil {
		diags.AddError(
			"Invalid ID Format",
			fmt.Sprintf("Unable to parse deployment policy ID: %s", err.Error()),
		)
		return
	}

	branchPolicyId, err := strconv.ParseInt(branchPolicyIdString, 10, 64)
	if err != nil {
		diags.AddError(
			"Invalid ID Format",
			fmt.Sprintf("Unable to convert deployment policy ID to integer: %s", err.Error()),
		)
		return
	}

	// Set up context with ID for caching
	ctx = context.WithValue(ctx, CtxId, data.ID.ValueString())

	branchPolicy, _, err := client.Repositories.GetDeploymentBranchPolicy(ctx, owner, repository, envName, branchPolicyId)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing deployment policy from state because it no longer exists in GitHub", map[string]any{
					"repository":  repository,
					"environment": envName,
					"policy_id":   branchPolicyId,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Error reading deployment policy",
			fmt.Sprintf("Could not read deployment policy %d for environment %s in repository %s: %s", branchPolicyId, envName, repository, err.Error()),
		)
		return
	}

	data.Repository = types.StringValue(repository)
	data.Environment = types.StringValue(envName)

	if branchPolicy.GetType() == "branch" {
		data.BranchPattern = types.StringValue(branchPolicy.GetName())
		data.TagPattern = types.StringNull()
	} else {
		data.TagPattern = types.StringValue(branchPolicy.GetName())
		data.BranchPattern = types.StringNull()
	}

	tflog.Debug(ctx, "successfully read GitHub repository environment deployment policy", map[string]any{
		"id":          data.ID.ValueString(),
		"repository":  repository,
		"environment": envName,
		"policy_id":   branchPolicyId,
		"type":        branchPolicy.GetType(),
		"name":        branchPolicy.GetName(),
	})
}

// Helper functions from the SDKv2 version - we need to implement these or reference them

// Custom validator for mutually exclusive attributes
type mutuallyExclusiveValidator struct {
	Attributes []string
}

func (v *mutuallyExclusiveValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("Exactly one of %v must be specified", v.Attributes)
}

func (v *mutuallyExclusiveValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *mutuallyExclusiveValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var branchPattern, tagPattern types.String

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("branch_pattern"), &branchPattern)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("tag_pattern"), &tagPattern)...)

	if resp.Diagnostics.HasError() {
		return
	}

	branchPatternSet := !branchPattern.IsNull() && !branchPattern.IsUnknown()
	tagPatternSet := !tagPattern.IsNull() && !tagPattern.IsUnknown()

	if !branchPatternSet && !tagPatternSet {
		resp.Diagnostics.AddError(
			"Missing Required Configuration",
			"Exactly one of branch_pattern and tag_pattern must be specified",
		)
	} else if branchPatternSet && tagPatternSet {
		resp.Diagnostics.AddError(
			"Conflicting Configuration",
			"Only one of branch_pattern and tag_pattern may be specified",
		)
	}
}

// Custom plan modifier to force replacement when switching between pattern types
type patternTypeSwitchPlanModifier struct{}

func (m *patternTypeSwitchPlanModifier) Description(ctx context.Context) string {
	return "Forces replacement when switching between branch_pattern and tag_pattern"
}

func (m *patternTypeSwitchPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m *patternTypeSwitchPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If this is a create operation, don't do anything special
	if req.State.Raw.IsNull() {
		return
	}

	// Get the other pattern attribute to check for pattern type switching
	var otherPatternState, otherPatternPlan types.String
	var otherPatternPath path.Path

	// Determine which pattern we're modifying and which is the "other"
	if req.Path.String() == "branch_pattern" {
		otherPatternPath = path.Root("tag_pattern")
	} else {
		otherPatternPath = path.Root("branch_pattern")
	}

	// Get the other pattern's state and plan values
	req.State.GetAttribute(ctx, otherPatternPath, &otherPatternState)
	req.Plan.GetAttribute(ctx, otherPatternPath, &otherPatternPlan)

	// Check if we're switching from one pattern type to another
	currentPatternInState := !req.StateValue.IsNull()
	otherPatternInState := !otherPatternState.IsNull()
	currentPatternInPlan := !req.PlanValue.IsNull() && !req.PlanValue.IsUnknown()
	otherPatternInPlan := !otherPatternPlan.IsNull() && !otherPatternPlan.IsUnknown()

	// Force replacement if:
	// 1. We had one pattern type in state and now we're switching to the other type
	// 2. We had this pattern in state but now it's being removed for the other pattern
	if (currentPatternInState && otherPatternInPlan) ||
		(otherPatternInState && currentPatternInPlan) {
		resp.RequiresReplace = true
	}
}
