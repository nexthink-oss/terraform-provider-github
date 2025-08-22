package github

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &githubActionsRepositoryPermissionsResource{}
	_ resource.ResourceWithConfigure   = &githubActionsRepositoryPermissionsResource{}
	_ resource.ResourceWithImportState = &githubActionsRepositoryPermissionsResource{}
)

func NewGithubActionsRepositoryPermissionsResource() resource.Resource {
	return &githubActionsRepositoryPermissionsResource{}
}

type githubActionsRepositoryPermissionsResource struct {
	client *Owner
}

type githubActionsRepositoryPermissionsResourceModel struct {
	// Required attributes
	Repository types.String `tfsdk:"repository"`

	// Optional attributes
	AllowedActions types.String `tfsdk:"allowed_actions"`
	Enabled        types.Bool   `tfsdk:"enabled"`

	// Nested configuration blocks
	AllowedActionsConfig types.List `tfsdk:"allowed_actions_config"`

	// Computed attributes - ID is the repository name
	ID types.String `tfsdk:"id"`
}

type repositoryAllowedActionsConfigModel struct {
	GithubOwnedAllowed types.Bool `tfsdk:"github_owned_allowed"`
	PatternsAllowed    types.Set  `tfsdk:"patterns_allowed"`
	VerifiedAllowed    types.Bool `tfsdk:"verified_allowed"`
}

func (r *githubActionsRepositoryPermissionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_repository_permissions"
}

func (r *githubActionsRepositoryPermissionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Enables and manages Actions permissions for a GitHub repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The name of the repository.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The GitHub repository.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"allowed_actions": schema.StringAttribute{
				Description: "The permissions policy that controls the actions that are allowed to run. Can be one of: 'all', 'local_only', or 'selected'.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("all", "local_only", "selected"),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "Should GitHub actions be enabled on this repository.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
		Blocks: map[string]schema.Block{
			"allowed_actions_config": schema.ListNestedBlock{
				Description: "Sets the actions that are allowed in a repository. Only available when 'allowed_actions' = 'selected'.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"github_owned_allowed": schema.BoolAttribute{
							Description: "Whether GitHub-owned actions are allowed",
							Required:    true,
						},
						"patterns_allowed": schema.SetAttribute{
							Description: "Specifies a list of string-matching patterns to allow specific action(s)",
							ElementType: types.StringType,
							Optional:    true,
						},
						"verified_allowed": schema.BoolAttribute{
							Description: "Whether actions from GitHub Marketplace verified creators are allowed",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (r *githubActionsRepositoryPermissionsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubActionsRepositoryPermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubActionsRepositoryPermissionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := plan.Repository.ValueString()
	client := r.client.V3Client()

	allowedActions := plan.AllowedActions.ValueString()
	enabled := plan.Enabled.ValueBool()
	log.Printf("[DEBUG] Actions enabled: %t", enabled)

	repoActionPermissions := github.ActionsPermissionsRepository{
		Enabled: &enabled,
	}

	// Only specify `allowed_actions` if actions are enabled
	if enabled {
		repoActionPermissions.AllowedActions = &allowedActions
	}

	_, _, err := client.Repositories.EditActionsPermissions(ctx,
		owner,
		repoName,
		repoActionPermissions,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error setting repository actions permissions", err.Error())
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
				_, _, err = client.Repositories.EditActionsAllowed(ctx,
					owner,
					repoName,
					*actionsAllowed)
				if err != nil {
					resp.Diagnostics.AddError("Error setting allowed actions config", err.Error())
					return
				}
			} else {
				log.Printf("[DEBUG] Allowed actions config not set, skipping")
			}
		}
	}

	plan.ID = types.StringValue(repoName)

	// Read the resource to populate all fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubActionsRepositoryPermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubActionsRepositoryPermissionsResourceModel
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

func (r *githubActionsRepositoryPermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubActionsRepositoryPermissionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := plan.Repository.ValueString()
	client := r.client.V3Client()

	allowedActions := plan.AllowedActions.ValueString()
	enabled := plan.Enabled.ValueBool()
	log.Printf("[DEBUG] Actions enabled: %t", enabled)

	repoActionPermissions := github.ActionsPermissionsRepository{
		Enabled: &enabled,
	}

	// Only specify `allowed_actions` if actions are enabled
	if enabled {
		repoActionPermissions.AllowedActions = &allowedActions
	}

	_, _, err := client.Repositories.EditActionsPermissions(ctx,
		owner,
		repoName,
		repoActionPermissions,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error updating repository actions permissions", err.Error())
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
				_, _, err = client.Repositories.EditActionsAllowed(ctx,
					owner,
					repoName,
					*actionsAllowed)
				if err != nil {
					resp.Diagnostics.AddError("Error updating allowed actions config", err.Error())
					return
				}
			} else {
				log.Printf("[DEBUG] Allowed actions config not set, skipping")
			}
		}
	}

	// Read the resource to populate all fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubActionsRepositoryPermissionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubActionsRepositoryPermissionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := state.ID.ValueString()
	client := r.client.V3Client()

	// Reset the repo to "default" settings
	repoActionPermissions := github.ActionsPermissionsRepository{
		AllowedActions: github.Ptr("all"),
		Enabled:        github.Ptr(true),
	}

	_, _, err := client.Repositories.EditActionsPermissions(ctx,
		owner,
		repoName,
		repoActionPermissions,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting repository actions permissions", err.Error())
		return
	}
}

func (r *githubActionsRepositoryPermissionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// Set repository to the same value as id for consistency
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository"), req.ID)...)
}

// Helper methods

func (r *githubActionsRepositoryPermissionsResource) readResource(ctx context.Context, model *githubActionsRepositoryPermissionsResourceModel, diagnostics *diag.Diagnostics) {
	owner := r.client.Name()
	repoName := model.ID.ValueString()
	client := r.client.V3Client()

	actionsPermissions, _, err := client.Repositories.GetActionsPermissions(ctx, owner, repoName)
	if err != nil {
		diagnostics.AddError("Error reading repository actions permissions", err.Error())
		return
	}

	// Set basic attributes
	model.AllowedActions = types.StringValue(actionsPermissions.GetAllowedActions())
	model.Enabled = types.BoolValue(actionsPermissions.GetEnabled())
	model.Repository = types.StringValue(repoName)

	// Handle allowed actions config
	// This logic follows the same pattern as the SDKv2 version for compatibility
	allowedActions := model.AllowedActions.ValueString()
	currentAllowedActionsConfig := model.AllowedActionsConfig

	serverHasAllowedActionsConfig := actionsPermissions.GetAllowedActions() == "selected" && actionsPermissions.GetEnabled()
	userWantsAllowedActionsConfig := (allowedActions == "selected" && !currentAllowedActionsConfig.IsNull() && len(currentAllowedActionsConfig.Elements()) > 0) || allowedActions == ""

	if serverHasAllowedActionsConfig && userWantsAllowedActionsConfig {
		actionsAllowed, _, err := client.Repositories.GetActionsAllowed(ctx, owner, repoName)
		if err != nil {
			diagnostics.AddError("Error reading allowed actions config", err.Error())
			return
		}

		// If actionsAllowed set to local/all by removing all actions config settings, the response will be empty
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
}

func (r *githubActionsRepositoryPermissionsResource) buildActionsAllowedFromPlan(ctx context.Context, planList types.List) (*github.ActionsAllowed, diag.Diagnostics) {
	var diags diag.Diagnostics

	if planList.IsNull() || len(planList.Elements()) == 0 {
		return nil, diags
	}

	element := planList.Elements()[0]
	var configObj repositoryAllowedActionsConfigModel

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

func (r *githubActionsRepositoryPermissionsResource) buildAllowedActionsConfigFromAPI(ctx context.Context, actionsAllowed *github.ActionsAllowed) (types.List, diag.Diagnostics) {
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
