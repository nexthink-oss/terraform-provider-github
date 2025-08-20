package framework

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ resource.Resource                = &githubActionsOrganizationPermissionsResource{}
	_ resource.ResourceWithConfigure   = &githubActionsOrganizationPermissionsResource{}
	_ resource.ResourceWithImportState = &githubActionsOrganizationPermissionsResource{}
)

func NewGithubActionsOrganizationPermissionsResource() resource.Resource {
	return &githubActionsOrganizationPermissionsResource{}
}

type githubActionsOrganizationPermissionsResource struct {
	client *githubpkg.Owner
}

type githubActionsOrganizationPermissionsResourceModel struct {
	// Required attributes
	EnabledRepositories types.String `tfsdk:"enabled_repositories"`

	// Optional attributes
	AllowedActions types.String `tfsdk:"allowed_actions"`

	// Nested configuration blocks
	AllowedActionsConfig      types.List `tfsdk:"allowed_actions_config"`
	EnabledRepositoriesConfig types.List `tfsdk:"enabled_repositories_config"`

	// Computed attributes
	ID types.String `tfsdk:"id"`
}

type allowedActionsConfigModel struct {
	GithubOwnedAllowed types.Bool `tfsdk:"github_owned_allowed"`
	PatternsAllowed    types.Set  `tfsdk:"patterns_allowed"`
	VerifiedAllowed    types.Bool `tfsdk:"verified_allowed"`
}

type enabledRepositoriesConfigModel struct {
	RepositoryIds types.Set `tfsdk:"repository_ids"`
}

func (r *githubActionsOrganizationPermissionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_organization_permissions"
}

func (r *githubActionsOrganizationPermissionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages Actions permissions within a GitHub organization",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enabled_repositories": schema.StringAttribute{
				Description: "The policy that controls the repositories in the organization that are allowed to run GitHub Actions. Can be one of: 'all', 'none', or 'selected'.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("all", "none", "selected"),
				},
			},
			"allowed_actions": schema.StringAttribute{
				Description: "The permissions policy that controls the actions that are allowed to run. Can be one of: 'all', 'local_only', or 'selected'.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("all", "local_only", "selected"),
				},
			},
			"allowed_actions_config": schema.ListAttribute{
				Description: "Sets the actions that are allowed in an organization. Only available when 'allowed_actions' = 'selected'",
				Optional:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"github_owned_allowed": types.BoolType,
						"patterns_allowed":     types.SetType{ElemType: types.StringType},
						"verified_allowed":     types.BoolType,
					},
				},
			},
			"enabled_repositories_config": schema.ListAttribute{
				Description: "Sets the list of selected repositories that are enabled for GitHub Actions in an organization. Only available when 'enabled_repositories' = 'selected'.",
				Optional:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"repository_ids": types.SetType{ElemType: types.Int64Type},
					},
				},
			},
		},
	}
}

func (r *githubActionsOrganizationPermissionsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*githubpkg.Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *githubpkg.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *githubActionsOrganizationPermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubActionsOrganizationPermissionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check if we're working with an organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization",
		)
		return
	}

	orgName := r.client.Name()
	client := r.client.V3Client()

	// Set up the basic permissions
	allowedActions := plan.AllowedActions.ValueString()
	enabledRepositories := plan.EnabledRepositories.ValueString()

	_, _, err := client.Actions.EditActionsPermissions(ctx,
		orgName,
		github.ActionsPermissions{
			AllowedActions:      &allowedActions,
			EnabledRepositories: &enabledRepositories,
		})
	if err != nil {
		resp.Diagnostics.AddError("Error setting organization actions permissions", err.Error())
		return
	}

	// Handle allowed actions config if allowed_actions is "selected"
	if allowedActions == "selected" {
		if !plan.AllowedActionsConfig.IsNull() && len(plan.AllowedActionsConfig.Elements()) > 0 {
			actionsAllowed, diags := r.buildActionsAllowedFromPlan(ctx, plan.AllowedActionsConfig)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if actionsAllowed != nil {
				log.Printf("[DEBUG] Allowed actions config is set")
				_, _, err = client.Actions.EditActionsAllowed(ctx,
					orgName,
					*actionsAllowed)
				if err != nil {
					resp.Diagnostics.AddError("Error setting allowed actions config", err.Error())
					return
				}
			}
		}
	}

	// Handle enabled repositories config if enabled_repositories is "selected"
	if enabledRepositories == "selected" {
		if !plan.EnabledRepositoriesConfig.IsNull() && len(plan.EnabledRepositoriesConfig.Elements()) > 0 {
			enabledRepos, diags := r.buildEnabledRepositoriesFromPlan(ctx, plan.EnabledRepositoriesConfig)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if enabledRepos != nil {
				_, err = client.Actions.SetEnabledReposInOrg(ctx,
					orgName,
					enabledRepos)
				if err != nil {
					resp.Diagnostics.AddError("Error setting enabled repositories config", err.Error())
					return
				}
			}
		} else {
			resp.Diagnostics.AddError(
				"Configuration Error",
				"enabled_repositories_config must be specified when enabled_repositories is 'selected'",
			)
			return
		}
	}

	plan.ID = types.StringValue(orgName)

	// Read the resource to populate all fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubActionsOrganizationPermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubActionsOrganizationPermissionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	r.readResource(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubActionsOrganizationPermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubActionsOrganizationPermissionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check if we're working with an organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization",
		)
		return
	}

	orgName := r.client.Name()
	client := r.client.V3Client()

	// Set up the basic permissions
	allowedActions := plan.AllowedActions.ValueString()
	enabledRepositories := plan.EnabledRepositories.ValueString()

	_, _, err := client.Actions.EditActionsPermissions(ctx,
		orgName,
		github.ActionsPermissions{
			AllowedActions:      &allowedActions,
			EnabledRepositories: &enabledRepositories,
		})
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization actions permissions", err.Error())
		return
	}

	// Handle allowed actions config if allowed_actions is "selected"
	if allowedActions == "selected" {
		if !plan.AllowedActionsConfig.IsNull() && len(plan.AllowedActionsConfig.Elements()) > 0 {
			actionsAllowed, diags := r.buildActionsAllowedFromPlan(ctx, plan.AllowedActionsConfig)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if actionsAllowed != nil {
				log.Printf("[DEBUG] Allowed actions config is set")
				_, _, err = client.Actions.EditActionsAllowed(ctx,
					orgName,
					*actionsAllowed)
				if err != nil {
					resp.Diagnostics.AddError("Error updating allowed actions config", err.Error())
					return
				}
			}
		}
	}

	// Handle enabled repositories config if enabled_repositories is "selected"
	if enabledRepositories == "selected" {
		if !plan.EnabledRepositoriesConfig.IsNull() && len(plan.EnabledRepositoriesConfig.Elements()) > 0 {
			enabledRepos, diags := r.buildEnabledRepositoriesFromPlan(ctx, plan.EnabledRepositoriesConfig)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if enabledRepos != nil {
				_, err = client.Actions.SetEnabledReposInOrg(ctx,
					orgName,
					enabledRepos)
				if err != nil {
					resp.Diagnostics.AddError("Error updating enabled repositories config", err.Error())
					return
				}
			}
		} else {
			resp.Diagnostics.AddError(
				"Configuration Error",
				"enabled_repositories_config must be specified when enabled_repositories is 'selected'",
			)
			return
		}
	}

	// Read the resource to populate all fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubActionsOrganizationPermissionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubActionsOrganizationPermissionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check if we're working with an organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization",
		)
		return
	}

	orgName := r.client.Name()
	client := r.client.V3Client()

	// Reset to default permissions (all allowed actions, all repositories)
	_, _, err := client.Actions.EditActionsPermissions(ctx,
		orgName,
		github.ActionsPermissions{
			AllowedActions:      github.Ptr("all"),
			EnabledRepositories: github.Ptr("all"),
		})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting organization actions permissions", err.Error())
		return
	}
}

func (r *githubActionsOrganizationPermissionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper methods

func (r *githubActionsOrganizationPermissionsResource) readResource(ctx context.Context, model *githubActionsOrganizationPermissionsResourceModel, diagnostics *diag.Diagnostics) {
	if !r.client.IsOrganization {
		diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization",
		)
		return
	}

	orgName := model.ID.ValueString()
	client := r.client.V3Client()

	actionsPermissions, _, err := client.Actions.GetActionsPermissions(ctx, orgName)
	if err != nil {
		diagnostics.AddError("Error reading organization actions permissions", err.Error())
		return
	}

	// Set basic attributes
	model.AllowedActions = types.StringValue(actionsPermissions.GetAllowedActions())
	model.EnabledRepositories = types.StringValue(actionsPermissions.GetEnabledRepositories())

	// Handle allowed actions config
	allowedActions := model.AllowedActions.ValueString()
	currentAllowedActionsConfig := model.AllowedActionsConfig

	serverHasAllowedActionsConfig := actionsPermissions.GetAllowedActions() == "selected"
	userWantsAllowedActionsConfig := (allowedActions == "selected" && !currentAllowedActionsConfig.IsNull() && len(currentAllowedActionsConfig.Elements()) > 0) || allowedActions == ""

	if serverHasAllowedActionsConfig && userWantsAllowedActionsConfig {
		actionsAllowed, _, err := client.Actions.GetActionsAllowed(ctx, orgName)
		if err != nil {
			diagnostics.AddError("Error reading allowed actions config", err.Error())
			return
		}

		if actionsAllowed != nil {
			allowedActionsConfigValue, diags := r.buildAllowedActionsConfigFromAPI(ctx, actionsAllowed)
			diagnostics.Append(diags...)
			if diagnostics.HasError() {
				return
			}
			model.AllowedActionsConfig = allowedActionsConfigValue
		} else {
			model.AllowedActionsConfig = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"github_owned_allowed": types.BoolType,
					"patterns_allowed":     types.SetType{ElemType: types.StringType},
					"verified_allowed":     types.BoolType,
				},
			})
		}
	} else {
		model.AllowedActionsConfig = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"github_owned_allowed": types.BoolType,
				"patterns_allowed":     types.SetType{ElemType: types.StringType},
				"verified_allowed":     types.BoolType,
			},
		})
	}

	// Handle enabled repositories config
	if actionsPermissions.GetEnabledRepositories() == "selected" {
		opts := &github.ListOptions{PerPage: 10, Page: 1}
		var repoList []int64
		var allRepos []*github.Repository

		for {
			enabledRepos, resp, err := client.Actions.ListEnabledReposInOrg(ctx, orgName, opts)
			if err != nil {
				diagnostics.AddError("Error reading enabled repositories", err.Error())
				return
			}
			allRepos = append(allRepos, enabledRepos.Repositories...)

			opts.Page = resp.NextPage
			if resp.NextPage == 0 {
				break
			}
		}

		for _, repo := range allRepos {
			repoList = append(repoList, *repo.ID)
		}

		if len(allRepos) > 0 {
			enabledReposConfigValue, diags := r.buildEnabledRepositoriesConfigFromAPI(ctx, repoList)
			diagnostics.Append(diags...)
			if diagnostics.HasError() {
				return
			}
			model.EnabledRepositoriesConfig = enabledReposConfigValue
		} else {
			model.EnabledRepositoriesConfig = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"repository_ids": types.SetType{ElemType: types.Int64Type},
				},
			})
		}
	} else {
		model.EnabledRepositoriesConfig = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"repository_ids": types.SetType{ElemType: types.Int64Type},
			},
		})
	}
}

func (r *githubActionsOrganizationPermissionsResource) buildActionsAllowedFromPlan(ctx context.Context, planList types.List) (*github.ActionsAllowed, diag.Diagnostics) {
	var diags diag.Diagnostics

	if planList.IsNull() || len(planList.Elements()) == 0 {
		return nil, diags
	}

	element := planList.Elements()[0]
	var configObj allowedActionsConfigModel

	objVal, ok := element.(types.Object)
	if !ok {
		diags.AddError("Type assertion failed", "Expected types.Object")
		return nil, diags
	}

	diags.Append(objVal.As(ctx, &configObj, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	allowed := &github.ActionsAllowed{}

	// Set github owned allowed
	if !configObj.GithubOwnedAllowed.IsNull() {
		githubOwnedAllowed := configObj.GithubOwnedAllowed.ValueBool()
		allowed.GithubOwnedAllowed = &githubOwnedAllowed
	}

	// Set verified allowed
	if !configObj.VerifiedAllowed.IsNull() {
		verifiedAllowed := configObj.VerifiedAllowed.ValueBool()
		allowed.VerifiedAllowed = &verifiedAllowed
	}

	// Set patterns allowed
	if !configObj.PatternsAllowed.IsNull() && len(configObj.PatternsAllowed.Elements()) > 0 {
		var patterns []string
		diags.Append(configObj.PatternsAllowed.ElementsAs(ctx, &patterns, false)...)
		if diags.HasError() {
			return nil, diags
		}
		allowed.PatternsAllowed = patterns
	}

	return allowed, diags
}

func (r *githubActionsOrganizationPermissionsResource) buildEnabledRepositoriesFromPlan(ctx context.Context, planList types.List) ([]int64, diag.Diagnostics) {
	var diags diag.Diagnostics

	if planList.IsNull() || len(planList.Elements()) == 0 {
		diags.AddError("Configuration Error", "enabled_repositories_config must be specified when enabled_repositories is 'selected'")
		return nil, diags
	}

	element := planList.Elements()[0]
	var configObj enabledRepositoriesConfigModel

	objVal, ok := element.(types.Object)
	if !ok {
		diags.AddError("Type assertion failed", "Expected types.Object")
		return nil, diags
	}

	diags.Append(objVal.As(ctx, &configObj, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	if configObj.RepositoryIds.IsNull() || len(configObj.RepositoryIds.Elements()) == 0 {
		diags.AddError("Configuration Error", "repository_ids must be specified in enabled_repositories_config when enabled_repositories is 'selected'")
		return nil, diags
	}

	var repoIds []int64
	diags.Append(configObj.RepositoryIds.ElementsAs(ctx, &repoIds, false)...)
	if diags.HasError() {
		return nil, diags
	}

	return repoIds, diags
}

func (r *githubActionsOrganizationPermissionsResource) buildAllowedActionsConfigFromAPI(ctx context.Context, actionsAllowed *github.ActionsAllowed) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	var patternsAllowed types.Set
	if len(actionsAllowed.PatternsAllowed) > 0 {
		elements := make([]attr.Value, len(actionsAllowed.PatternsAllowed))
		for i, pattern := range actionsAllowed.PatternsAllowed {
			elements[i] = types.StringValue(pattern)
		}
		var setDiags diag.Diagnostics
		patternsAllowed, setDiags = types.SetValue(types.StringType, elements)
		diags.Append(setDiags...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"github_owned_allowed": types.BoolType,
					"patterns_allowed":     types.SetType{ElemType: types.StringType},
					"verified_allowed":     types.BoolType,
				},
			}), diags
		}
	} else {
		patternsAllowed = types.SetNull(types.StringType)
	}

	configValue := map[string]attr.Value{
		"github_owned_allowed": types.BoolValue(actionsAllowed.GetGithubOwnedAllowed()),
		"patterns_allowed":     patternsAllowed,
		"verified_allowed":     types.BoolValue(actionsAllowed.GetVerifiedAllowed()),
	}

	objValue, objDiags := types.ObjectValue(map[string]attr.Type{
		"github_owned_allowed": types.BoolType,
		"patterns_allowed":     types.SetType{ElemType: types.StringType},
		"verified_allowed":     types.BoolType,
	}, configValue)
	diags.Append(objDiags...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"github_owned_allowed": types.BoolType,
				"patterns_allowed":     types.SetType{ElemType: types.StringType},
				"verified_allowed":     types.BoolType,
			},
		}), diags
	}

	listValue, listDiags := types.ListValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"github_owned_allowed": types.BoolType,
			"patterns_allowed":     types.SetType{ElemType: types.StringType},
			"verified_allowed":     types.BoolType,
		},
	}, []attr.Value{objValue})
	diags.Append(listDiags...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"github_owned_allowed": types.BoolType,
				"patterns_allowed":     types.SetType{ElemType: types.StringType},
				"verified_allowed":     types.BoolType,
			},
		}), diags
	}

	return listValue, diags
}

func (r *githubActionsOrganizationPermissionsResource) buildEnabledRepositoriesConfigFromAPI(ctx context.Context, repoIds []int64) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	elements := make([]attr.Value, len(repoIds))
	for i, id := range repoIds {
		elements[i] = types.Int64Value(id)
	}

	repoIdsSet, setDiags := types.SetValue(types.Int64Type, elements)
	diags.Append(setDiags...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"repository_ids": types.SetType{ElemType: types.Int64Type},
			},
		}), diags
	}

	configValue := map[string]attr.Value{
		"repository_ids": repoIdsSet,
	}

	objValue, objDiags := types.ObjectValue(map[string]attr.Type{
		"repository_ids": types.SetType{ElemType: types.Int64Type},
	}, configValue)
	diags.Append(objDiags...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"repository_ids": types.SetType{ElemType: types.Int64Type},
			},
		}), diags
	}

	listValue, listDiags := types.ListValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"repository_ids": types.SetType{ElemType: types.Int64Type},
		},
	}, []attr.Value{objValue})
	diags.Append(listDiags...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"repository_ids": types.SetType{ElemType: types.Int64Type},
			},
		}), diags
	}

	return listValue, diags
}
