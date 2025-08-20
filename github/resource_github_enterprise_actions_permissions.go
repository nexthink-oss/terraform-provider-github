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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &githubEnterpriseActionsPermissionsResource{}
	_ resource.ResourceWithConfigure   = &githubEnterpriseActionsPermissionsResource{}
	_ resource.ResourceWithImportState = &githubEnterpriseActionsPermissionsResource{}
)

func NewGithubEnterpriseActionsPermissionsResource() resource.Resource {
	return &githubEnterpriseActionsPermissionsResource{}
}

type githubEnterpriseActionsPermissionsResource struct {
	client *Owner
}

type githubEnterpriseActionsPermissionsResourceModel struct {
	// Required attributes
	EnterpriseSlug       types.String `tfsdk:"enterprise_slug"`
	EnabledOrganizations types.String `tfsdk:"enabled_organizations"`

	// Optional attributes
	AllowedActions types.String `tfsdk:"allowed_actions"`

	// Nested configuration blocks
	AllowedActionsConfig       types.List `tfsdk:"allowed_actions_config"`
	EnabledOrganizationsConfig types.List `tfsdk:"enabled_organizations_config"`

	// Computed attributes
	ID types.String `tfsdk:"id"`
}

type enterpriseAllowedActionsConfigModel struct {
	GithubOwnedAllowed types.Bool `tfsdk:"github_owned_allowed"`
	PatternsAllowed    types.Set  `tfsdk:"patterns_allowed"`
	VerifiedAllowed    types.Bool `tfsdk:"verified_allowed"`
}

type enterpriseEnabledOrganizationsConfigModel struct {
	OrganizationIds types.Set `tfsdk:"organization_ids"`
}

func (r *githubEnterpriseActionsPermissionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_enterprise_actions_permissions"
}

func (r *githubEnterpriseActionsPermissionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages Actions permissions within a GitHub enterprise",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the enterprise.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enterprise_slug": schema.StringAttribute{
				Description: "The slug of the enterprise.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled_organizations": schema.StringAttribute{
				Description: "The policy that controls the organizations in the enterprise that are allowed to run GitHub Actions. Can be one of: 'all', 'none', or 'selected'.",
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
				Description: "Sets the actions that are allowed in an enterprise. Only available when 'allowed_actions' = 'selected'",
				Optional:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"github_owned_allowed": types.BoolType,
						"patterns_allowed":     types.SetType{ElemType: types.StringType},
						"verified_allowed":     types.BoolType,
					},
				},
			},
			"enabled_organizations_config": schema.ListAttribute{
				Description: "Sets the list of selected organizations that are enabled for GitHub Actions in an enterprise. Only available when 'enabled_organizations' = 'selected'.",
				Optional:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"organization_ids": types.SetType{ElemType: types.Int64Type},
					},
				},
			},
		},
	}
}

func (r *githubEnterpriseActionsPermissionsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubEnterpriseActionsPermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubEnterpriseActionsPermissionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	enterpriseSlug := plan.EnterpriseSlug.ValueString()
	client := r.client.V3Client()

	// Set up the basic permissions
	allowedActions := plan.AllowedActions.ValueString()
	enabledOrganizations := plan.EnabledOrganizations.ValueString()

	_, _, err := client.Actions.EditActionsPermissionsInEnterprise(ctx,
		enterpriseSlug,
		github.ActionsPermissionsEnterprise{
			AllowedActions:       &allowedActions,
			EnabledOrganizations: &enabledOrganizations,
		})
	if err != nil {
		resp.Diagnostics.AddError("Error setting enterprise actions permissions", err.Error())
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
				_, _, err = client.Actions.EditActionsAllowedInEnterprise(ctx,
					enterpriseSlug,
					*actionsAllowed)
				if err != nil {
					resp.Diagnostics.AddError("Error setting allowed actions config", err.Error())
					return
				}
			}
		}
	}

	// Handle enabled organizations config if enabled_organizations is "selected"
	if enabledOrganizations == "selected" {
		if !plan.EnabledOrganizationsConfig.IsNull() && len(plan.EnabledOrganizationsConfig.Elements()) > 0 {
			enabledOrgs, diags := r.buildEnabledOrganizationsFromPlan(ctx, plan.EnabledOrganizationsConfig)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if enabledOrgs != nil {
				_, err = client.Actions.SetEnabledOrgsInEnterprise(ctx,
					enterpriseSlug,
					enabledOrgs)
				if err != nil {
					resp.Diagnostics.AddError("Error setting enabled organizations config", err.Error())
					return
				}
			}
		} else {
			resp.Diagnostics.AddError(
				"Configuration Error",
				"enabled_organizations_config must be specified when enabled_organizations is 'selected'",
			)
			return
		}
	}

	plan.ID = types.StringValue(enterpriseSlug)

	// Read the resource to populate all fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubEnterpriseActionsPermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubEnterpriseActionsPermissionsResourceModel
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

func (r *githubEnterpriseActionsPermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubEnterpriseActionsPermissionsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	enterpriseSlug := plan.EnterpriseSlug.ValueString()
	client := r.client.V3Client()

	// Set up the basic permissions
	allowedActions := plan.AllowedActions.ValueString()
	enabledOrganizations := plan.EnabledOrganizations.ValueString()

	_, _, err := client.Actions.EditActionsPermissionsInEnterprise(ctx,
		enterpriseSlug,
		github.ActionsPermissionsEnterprise{
			AllowedActions:       &allowedActions,
			EnabledOrganizations: &enabledOrganizations,
		})
	if err != nil {
		resp.Diagnostics.AddError("Error updating enterprise actions permissions", err.Error())
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
				_, _, err = client.Actions.EditActionsAllowedInEnterprise(ctx,
					enterpriseSlug,
					*actionsAllowed)
				if err != nil {
					resp.Diagnostics.AddError("Error updating allowed actions config", err.Error())
					return
				}
			}
		}
	}

	// Handle enabled organizations config if enabled_organizations is "selected"
	if enabledOrganizations == "selected" {
		if !plan.EnabledOrganizationsConfig.IsNull() && len(plan.EnabledOrganizationsConfig.Elements()) > 0 {
			enabledOrgs, diags := r.buildEnabledOrganizationsFromPlan(ctx, plan.EnabledOrganizationsConfig)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			if enabledOrgs != nil {
				_, err = client.Actions.SetEnabledOrgsInEnterprise(ctx,
					enterpriseSlug,
					enabledOrgs)
				if err != nil {
					resp.Diagnostics.AddError("Error updating enabled organizations config", err.Error())
					return
				}
			}
		} else {
			resp.Diagnostics.AddError(
				"Configuration Error",
				"enabled_organizations_config must be specified when enabled_organizations is 'selected'",
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

func (r *githubEnterpriseActionsPermissionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubEnterpriseActionsPermissionsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	enterpriseSlug := state.EnterpriseSlug.ValueString()
	client := r.client.V3Client()

	// Reset to default permissions (all allowed actions, all organizations)
	_, _, err := client.Actions.EditActionsPermissionsInEnterprise(ctx,
		enterpriseSlug,
		github.ActionsPermissionsEnterprise{
			AllowedActions:       github.Ptr("all"),
			EnabledOrganizations: github.Ptr("all"),
		})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting enterprise actions permissions", err.Error())
		return
	}
}

func (r *githubEnterpriseActionsPermissionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Helper methods

func (r *githubEnterpriseActionsPermissionsResource) readResource(ctx context.Context, model *githubEnterpriseActionsPermissionsResourceModel, diagnostics *diag.Diagnostics) {
	enterpriseSlug := model.ID.ValueString()
	client := r.client.V3Client()

	actionsPermissions, _, err := client.Actions.GetActionsPermissionsInEnterprise(ctx, enterpriseSlug)
	if err != nil {
		diagnostics.AddError("Error reading enterprise actions permissions", err.Error())
		return
	}

	// Set basic attributes
	model.AllowedActions = types.StringValue(actionsPermissions.GetAllowedActions())
	model.EnabledOrganizations = types.StringValue(actionsPermissions.GetEnabledOrganizations())
	model.EnterpriseSlug = types.StringValue(enterpriseSlug)

	// Handle allowed actions config
	allowedActions := model.AllowedActions.ValueString()
	currentAllowedActionsConfig := model.AllowedActionsConfig

	serverHasAllowedActionsConfig := actionsPermissions.GetAllowedActions() == "selected"
	userWantsAllowedActionsConfig := (allowedActions == "selected" && !currentAllowedActionsConfig.IsNull() && len(currentAllowedActionsConfig.Elements()) > 0) || allowedActions == ""

	if serverHasAllowedActionsConfig && userWantsAllowedActionsConfig {
		actionsAllowed, _, err := client.Actions.GetActionsAllowedInEnterprise(ctx, enterpriseSlug)
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

	// Handle enabled organizations config
	if actionsPermissions.GetEnabledOrganizations() == "selected" {
		opts := &github.ListOptions{PerPage: 10, Page: 1}
		var orgList []int64
		var allOrgs []*github.Organization

		for {
			enabledOrgs, resp, err := client.Actions.ListEnabledOrgsInEnterprise(ctx, enterpriseSlug, opts)
			if err != nil {
				diagnostics.AddError("Error reading enabled organizations", err.Error())
				return
			}
			allOrgs = append(allOrgs, enabledOrgs.Organizations...)

			opts.Page = resp.NextPage
			if resp.NextPage == 0 {
				break
			}
		}

		for _, org := range allOrgs {
			orgList = append(orgList, *org.ID)
		}

		if len(allOrgs) > 0 {
			enabledOrgsConfigValue, diags := r.buildEnabledOrganizationsConfigFromAPI(ctx, orgList)
			diagnostics.Append(diags...)
			if diagnostics.HasError() {
				return
			}
			model.EnabledOrganizationsConfig = enabledOrgsConfigValue
		} else {
			model.EnabledOrganizationsConfig = types.ListNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"organization_ids": types.SetType{ElemType: types.Int64Type},
				},
			})
		}
	} else {
		model.EnabledOrganizationsConfig = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"organization_ids": types.SetType{ElemType: types.Int64Type},
			},
		})
	}
}

func (r *githubEnterpriseActionsPermissionsResource) buildActionsAllowedFromPlan(ctx context.Context, planList types.List) (*github.ActionsAllowed, diag.Diagnostics) {
	var diags diag.Diagnostics

	if planList.IsNull() || len(planList.Elements()) == 0 {
		return nil, diags
	}

	element := planList.Elements()[0]
	var configObj enterpriseAllowedActionsConfigModel

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

func (r *githubEnterpriseActionsPermissionsResource) buildEnabledOrganizationsFromPlan(ctx context.Context, planList types.List) ([]int64, diag.Diagnostics) {
	var diags diag.Diagnostics

	if planList.IsNull() || len(planList.Elements()) == 0 {
		diags.AddError("Configuration Error", "enabled_organizations_config must be specified when enabled_organizations is 'selected'")
		return nil, diags
	}

	element := planList.Elements()[0]
	var configObj enterpriseEnabledOrganizationsConfigModel

	objVal, ok := element.(types.Object)
	if !ok {
		diags.AddError("Type assertion failed", "Expected types.Object")
		return nil, diags
	}

	diags.Append(objVal.As(ctx, &configObj, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}

	if configObj.OrganizationIds.IsNull() || len(configObj.OrganizationIds.Elements()) == 0 {
		diags.AddError("Configuration Error", "organization_ids must be specified in enabled_organizations_config when enabled_organizations is 'selected'")
		return nil, diags
	}

	var orgIds []int64
	diags.Append(configObj.OrganizationIds.ElementsAs(ctx, &orgIds, false)...)
	if diags.HasError() {
		return nil, diags
	}

	return orgIds, diags
}

func (r *githubEnterpriseActionsPermissionsResource) buildAllowedActionsConfigFromAPI(ctx context.Context, actionsAllowed *github.ActionsAllowed) (types.List, diag.Diagnostics) {
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

func (r *githubEnterpriseActionsPermissionsResource) buildEnabledOrganizationsConfigFromAPI(ctx context.Context, orgIds []int64) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	elements := make([]attr.Value, len(orgIds))
	for i, id := range orgIds {
		elements[i] = types.Int64Value(id)
	}

	orgIdsSet, setDiags := types.SetValue(types.Int64Type, elements)
	diags.Append(setDiags...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"organization_ids": types.SetType{ElemType: types.Int64Type},
			},
		}), diags
	}

	configValue := map[string]attr.Value{
		"organization_ids": orgIdsSet,
	}

	objValue, objDiags := types.ObjectValue(map[string]attr.Type{
		"organization_ids": types.SetType{ElemType: types.Int64Type},
	}, configValue)
	diags.Append(objDiags...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"organization_ids": types.SetType{ElemType: types.Int64Type},
			},
		}), diags
	}

	listValue, listDiags := types.ListValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"organization_ids": types.SetType{ElemType: types.Int64Type},
		},
	}, []attr.Value{objValue})
	diags.Append(listDiags...)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"organization_ids": types.SetType{ElemType: types.Int64Type},
			},
		}), diags
	}

	return listValue, diags
}
