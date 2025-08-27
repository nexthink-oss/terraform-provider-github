package github

import (
	"context"
	"fmt"
	"log"
	"strings"

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
	"github.com/shurcooL/githubv4"
)

var (
	_ resource.Resource                 = &githubBranchProtectionResource{}
	_ resource.ResourceWithConfigure    = &githubBranchProtectionResource{}
	_ resource.ResourceWithImportState  = &githubBranchProtectionResource{}
	_ resource.ResourceWithUpgradeState = &githubBranchProtectionResource{}
)

func NewGithubBranchProtectionResource() resource.Resource {
	return &githubBranchProtectionResource{}
}

type githubBranchProtectionResource struct {
	client *Owner
}

type githubBranchProtectionResourceModel struct {
	// Required attributes
	RepositoryID types.String `tfsdk:"repository_id"`
	Pattern      types.String `tfsdk:"pattern"`

	// Optional boolean attributes
	AllowsDeletions               types.Bool `tfsdk:"allows_deletions"`
	AllowsForcePushes             types.Bool `tfsdk:"allows_force_pushes"`
	EnforceAdmins                 types.Bool `tfsdk:"enforce_admins"`
	RequireSignedCommits          types.Bool `tfsdk:"require_signed_commits"`
	RequiredLinearHistory         types.Bool `tfsdk:"required_linear_history"`
	RequireConversationResolution types.Bool `tfsdk:"require_conversation_resolution"`
	LockBranch                    types.Bool `tfsdk:"lock_branch"`

	// Complex nested attributes
	RequiredPullRequestReviews types.List `tfsdk:"required_pull_request_reviews"`
	RequiredStatusChecks       types.List `tfsdk:"required_status_checks"`
	RestrictPushes             types.List `tfsdk:"restrict_pushes"`

	// Set attributes for actors
	ForcePushBypassers types.Set `tfsdk:"force_push_bypassers"`

	// Computed attributes (ID)
	ID types.String `tfsdk:"id"`
}

type requiredPullRequestReviewsModel struct {
	RequiredApprovingReviewCount types.Int64 `tfsdk:"required_approving_review_count"`
	RequireCodeOwnerReviews      types.Bool  `tfsdk:"require_code_owner_reviews"`
	DismissStaleReviews          types.Bool  `tfsdk:"dismiss_stale_reviews"`
	RestrictDismissals           types.Bool  `tfsdk:"restrict_dismissals"`
	DismissalRestrictions        types.Set   `tfsdk:"dismissal_restrictions"`
	PullRequestBypassers         types.Set   `tfsdk:"pull_request_bypassers"`
	RequireLastPushApproval      types.Bool  `tfsdk:"require_last_push_approval"`
}

type requiredStatusChecksModel struct {
	Strict   types.Bool `tfsdk:"strict"`
	Contexts types.Set  `tfsdk:"contexts"`
}

type restrictPushesModel struct {
	BlocksCreations types.Bool `tfsdk:"blocks_creations"`
	PushAllowances  types.Set  `tfsdk:"push_allowances"`
}

func (r *githubBranchProtectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_protection"
}

func (r *githubBranchProtectionResource) SchemaVersion() int64 {
	return 2
}

func (r *githubBranchProtectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Protects a GitHub branch.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the branch protection rule.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository_id": schema.StringAttribute{
				Description: "The name or node ID of the repository associated with this branch protection rule.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"pattern": schema.StringAttribute{
				Description: "Identifies the protection rule pattern.",
				Required:    true,
			},
			"allows_deletions": schema.BoolAttribute{
				Description: "Setting this to 'true' to allow the branch to be deleted.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"allows_force_pushes": schema.BoolAttribute{
				Description: "Setting this to 'true' to allow force pushes on the branch.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
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
			"required_linear_history": schema.BoolAttribute{
				Description: "Setting this to 'true' enforces a linear commit Git history, which prevents anyone from pushing merge commits to a branch.",
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
			"lock_branch": schema.BoolAttribute{
				Description: "Setting this to 'true' will make the branch read-only and preventing any pushes to it.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"force_push_bypassers": schema.SetAttribute{
				Description: "The list of actor Names/IDs that are allowed to bypass force push restrictions. Actor names must either begin with a '/' for users or the organization name followed by a '/' for teams.",
				Optional:    true,
				ElementType: types.StringType,
			},
		},
		Blocks: map[string]schema.Block{
			"required_pull_request_reviews": schema.ListNestedBlock{
				Description: "Enforce restrictions for pull request reviews.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"required_approving_review_count": schema.Int64Attribute{
							Description: "Require 'x' number of approvals to satisfy branch protection requirements. If this is specified it must be a number between 0-6.",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(1),
							Validators: []validator.Int64{
								int64validator.Between(0, 6),
							},
						},
						"require_code_owner_reviews": schema.BoolAttribute{
							Description: "Require an approved review in pull requests including files with a designated code owner.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"dismiss_stale_reviews": schema.BoolAttribute{
							Description: "Dismiss approved reviews automatically when a new commit is pushed.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"restrict_dismissals": schema.BoolAttribute{
							Description: "Restrict pull request review dismissals.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"dismissal_restrictions": schema.SetAttribute{
							Description: "The list of actor Names/IDs with dismissal access. If not empty, 'restrict_dismissals' is ignored. Actor names must either begin with a '/' for users or the organization name followed by a '/' for teams.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"pull_request_bypassers": schema.SetAttribute{
							Description: "The list of actor Names/IDs that are allowed to bypass pull request requirements. Actor names must either begin with a '/' for users or the organization name followed by a '/' for teams.",
							Optional:    true,
							ElementType: types.StringType,
						},
						"require_last_push_approval": schema.BoolAttribute{
							Description: "Require that The most recent push must be approved by someone other than the last pusher.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
					},
				},
			},
			"required_status_checks": schema.ListNestedBlock{
				Description: "Enforce restrictions for required status checks.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"strict": schema.BoolAttribute{
							Description: "Require branches to be up to date before merging.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"contexts": schema.SetAttribute{
							Description: "The list of status checks to require in order to merge into this branch. No status checks are required by default.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"restrict_pushes": schema.ListNestedBlock{
				Description: "Restrict who can push to matching branches.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"blocks_creations": schema.BoolAttribute{
							Description: "Restrict pushes that create matching branches.",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(true),
						},
						"push_allowances": schema.SetAttribute{
							Description: "The list of actor Names/IDs that may push to the branch. Actor names must either begin with a '/' for users or the organization name followed by a '/' for teams.",
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *githubBranchProtectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubBranchProtectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubBranchProtectionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert plan to GraphQL input
	data, err := r.planToBranchProtectionData(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Plan to Branch Protection Data",
			"Could not convert plan to branch protection data: "+err.Error(),
		)
		return
	}

	// Resolve actor IDs
	if err := r.resolveActorIDs(ctx, &data); err != nil {
		resp.Diagnostics.AddError(
			"Error Resolving Actor IDs",
			"Could not resolve actor IDs: "+err.Error(),
		)
		return
	}

	// Create the branch protection rule via GraphQL
	var mutate struct {
		CreateBranchProtectionRule struct {
			BranchProtectionRule struct {
				ID githubv4.ID
			}
		} `graphql:"createBranchProtectionRule(input: $input)"`
	}

	input := r.dataToCreateInput(data)

	client := r.client.V4Client()
	err = client.Mutate(ctx, &mutate, input, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Branch Protection Rule",
			"Could not create branch protection rule: "+err.Error(),
		)
		return
	}

	// Set the ID
	plan.ID = types.StringValue(mutate.CreateBranchProtectionRule.BranchProtectionRule.ID.(string))

	// Read the created resource to get computed values
	r.readBranchProtection(ctx, &plan, &resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *githubBranchProtectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubBranchProtectionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readBranchProtection(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *githubBranchProtectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubBranchProtectionResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert plan to GraphQL input
	data, err := r.planToBranchProtectionData(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Plan to Branch Protection Data",
			"Could not convert plan to branch protection data: "+err.Error(),
		)
		return
	}

	// Resolve actor IDs
	if err := r.resolveActorIDs(ctx, &data); err != nil {
		resp.Diagnostics.AddError(
			"Error Resolving Actor IDs",
			"Could not resolve actor IDs: "+err.Error(),
		)
		return
	}

	// Update the branch protection rule via GraphQL
	var mutate struct {
		UpdateBranchProtectionRule struct {
			BranchProtectionRule struct {
				ID githubv4.ID
			}
		} `graphql:"updateBranchProtectionRule(input: $input)"`
	}

	input := r.dataToUpdateInput(data, plan.ID.ValueString())

	client := r.client.V4Client()
	err = client.Mutate(ctx, &mutate, input, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Branch Protection Rule",
			"Could not update branch protection rule: "+err.Error(),
		)
		return
	}

	// Read the updated resource to get computed values
	r.readBranchProtection(ctx, &plan, &resp.Diagnostics)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *githubBranchProtectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubBranchProtectionResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var mutate struct {
		DeleteBranchProtectionRule struct {
			ClientMutationID githubv4.ID
		} `graphql:"deleteBranchProtectionRule(input: $input)"`
	}

	input := githubv4.DeleteBranchProtectionRuleInput{
		BranchProtectionRuleID: state.ID.ValueString(),
	}

	client := r.client.V4Client()
	err := client.Mutate(ctx, &mutate, input, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Branch Protection Rule",
			"Could not delete branch protection rule: "+err.Error(),
		)
		return
	}
}

func (r *githubBranchProtectionResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"repository": schema.StringAttribute{
						Required: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"branch": schema.StringAttribute{
						Required: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var oldState struct {
					Repository types.String `tfsdk:"repository"`
					Branch     types.String `tfsdk:"branch"`
				}

				resp.Diagnostics.Append(req.State.Get(ctx, &oldState)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// Get repository ID from name
				repoName := oldState.Repository.ValueString()
				repoID, err := getRepositoryID(repoName, r.client)
				if err != nil {
					resp.Diagnostics.AddError(
						"Error Getting Repository ID During Migration",
						"Could not get repository ID during state migration: "+err.Error(),
					)
					return
				}

				// Get branch protection rule ID
				branch := oldState.Branch.ValueString()
				protectionID, err := r.getBranchProtectionID(ctx, repoID.(string), branch)
				if err != nil {
					resp.Diagnostics.AddError(
						"Error Getting Branch Protection ID During Migration",
						"Could not get branch protection rule ID during state migration: "+err.Error(),
					)
					return
				}

				// Create new state structure
				newState := githubBranchProtectionResourceModel{
					ID:                            types.StringValue(protectionID),
					RepositoryID:                  types.StringValue(repoID.(string)),
					Pattern:                       types.StringValue(branch),
					AllowsDeletions:               types.BoolValue(false),
					AllowsForcePushes:             types.BoolValue(false),
					EnforceAdmins:                 types.BoolValue(false),
					RequireSignedCommits:          types.BoolValue(false),
					RequiredLinearHistory:         types.BoolValue(false),
					RequireConversationResolution: types.BoolValue(false),
					LockBranch:                    types.BoolValue(false),
					RequiredPullRequestReviews: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{
						"required_approving_review_count": types.Int64Type,
						"require_code_owner_reviews":      types.BoolType,
						"dismiss_stale_reviews":           types.BoolType,
						"restrict_dismissals":             types.BoolType,
						"dismissal_restrictions":          types.SetType{ElemType: types.StringType},
						"pull_request_bypassers":          types.SetType{ElemType: types.StringType},
						"require_last_push_approval":      types.BoolType,
					}}),
					RequiredStatusChecks: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{
						"strict":   types.BoolType,
						"contexts": types.SetType{ElemType: types.StringType},
					}}),
					RestrictPushes: types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{
						"blocks_creations": types.BoolType,
						"push_allowances":  types.SetType{ElemType: types.StringType},
					}}),
					ForcePushBypassers: types.SetNull(types.StringType),
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
			},
		},
		1: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed: true,
					},
					"repository_id": schema.StringAttribute{
						Required: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"pattern": schema.StringAttribute{
						Required: true,
					},
					"push_restrictions": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"blocks_creations": schema.BoolAttribute{
						Optional: true,
						Computed: true,
						Default:  booldefault.StaticBool(false),
					},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var oldState struct {
					ID               types.String `tfsdk:"id"`
					RepositoryID     types.String `tfsdk:"repository_id"`
					Pattern          types.String `tfsdk:"pattern"`
					PushRestrictions types.Set    `tfsdk:"push_restrictions"`
					BlocksCreations  types.Bool   `tfsdk:"blocks_creations"`
					// Include other attributes that may exist
					AllowsDeletions               types.Bool `tfsdk:"allows_deletions"`
					AllowsForcePushes             types.Bool `tfsdk:"allows_force_pushes"`
					EnforceAdmins                 types.Bool `tfsdk:"enforce_admins"`
					RequireSignedCommits          types.Bool `tfsdk:"require_signed_commits"`
					RequiredLinearHistory         types.Bool `tfsdk:"required_linear_history"`
					RequireConversationResolution types.Bool `tfsdk:"require_conversation_resolution"`
					LockBranch                    types.Bool `tfsdk:"lock_branch"`
					RequiredPullRequestReviews    types.List `tfsdk:"required_pull_request_reviews"`
					RequiredStatusChecks          types.List `tfsdk:"required_status_checks"`
					ForcePushBypassers            types.Set  `tfsdk:"force_push_bypassers"`
				}

				resp.Diagnostics.Append(req.State.Get(ctx, &oldState)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// Migrate push_restrictions and blocks_creations to restrict_pushes block
				var restrictPushes types.List
				if !oldState.PushRestrictions.IsNull() && !oldState.PushRestrictions.IsUnknown() {
					// Convert push_restrictions set to push_allowances
					var pushRestrictions []string
					resp.Diagnostics.Append(oldState.PushRestrictions.ElementsAs(ctx, &pushRestrictions, false)...)
					if resp.Diagnostics.HasError() {
						return
					}

					pushAllowancesSet, diags := types.SetValueFrom(ctx, types.StringType, pushRestrictions)
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}

					blocksCreations := false
					if !oldState.BlocksCreations.IsNull() && !oldState.BlocksCreations.IsUnknown() {
						blocksCreations = oldState.BlocksCreations.ValueBool()
					}

					// Create restrict_pushes block
					restrictPushObj, diags := types.ObjectValue(
						map[string]attr.Type{
							"blocks_creations": types.BoolType,
							"push_allowances":  types.SetType{ElemType: types.StringType},
						},
						map[string]attr.Value{
							"blocks_creations": types.BoolValue(blocksCreations),
							"push_allowances":  pushAllowancesSet,
						},
					)
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}

					restrictPushes, diags = types.ListValue(
						types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"blocks_creations": types.BoolType,
								"push_allowances":  types.SetType{ElemType: types.StringType},
							},
						},
						[]attr.Value{restrictPushObj},
					)
					resp.Diagnostics.Append(diags...)
					if resp.Diagnostics.HasError() {
						return
					}
				} else {
					// No push restrictions, set to null
					restrictPushes = types.ListNull(types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"blocks_creations": types.BoolType,
							"push_allowances":  types.SetType{ElemType: types.StringType},
						},
					})
				}

				// Preserve other attributes, set null/default values as appropriate
				newState := githubBranchProtectionResourceModel{
					ID:             oldState.ID,
					RepositoryID:   oldState.RepositoryID,
					Pattern:        oldState.Pattern,
					RestrictPushes: restrictPushes,

					// Preserve existing attributes or set defaults
					AllowsDeletions:               oldState.AllowsDeletions,
					AllowsForcePushes:             oldState.AllowsForcePushes,
					EnforceAdmins:                 oldState.EnforceAdmins,
					RequireSignedCommits:          oldState.RequireSignedCommits,
					RequiredLinearHistory:         oldState.RequiredLinearHistory,
					RequireConversationResolution: oldState.RequireConversationResolution,
					LockBranch:                    oldState.LockBranch,
					RequiredPullRequestReviews:    oldState.RequiredPullRequestReviews,
					RequiredStatusChecks:          oldState.RequiredStatusChecks,
					ForcePushBypassers:            oldState.ForcePushBypassers,
				}

				// Set null values for any unset attributes
				if newState.AllowsDeletions.IsNull() {
					newState.AllowsDeletions = types.BoolValue(false)
				}
				if newState.AllowsForcePushes.IsNull() {
					newState.AllowsForcePushes = types.BoolValue(false)
				}
				if newState.EnforceAdmins.IsNull() {
					newState.EnforceAdmins = types.BoolValue(false)
				}
				if newState.RequireSignedCommits.IsNull() {
					newState.RequireSignedCommits = types.BoolValue(false)
				}
				if newState.RequiredLinearHistory.IsNull() {
					newState.RequiredLinearHistory = types.BoolValue(false)
				}
				if newState.RequireConversationResolution.IsNull() {
					newState.RequireConversationResolution = types.BoolValue(false)
				}
				if newState.LockBranch.IsNull() {
					newState.LockBranch = types.BoolValue(false)
				}

				// Set null list values if not set
				if newState.RequiredPullRequestReviews.IsNull() {
					newState.RequiredPullRequestReviews = types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{
						"required_approving_review_count": types.Int64Type,
						"require_code_owner_reviews":      types.BoolType,
						"dismiss_stale_reviews":           types.BoolType,
						"restrict_dismissals":             types.BoolType,
						"dismissal_restrictions":          types.SetType{ElemType: types.StringType},
						"pull_request_bypassers":          types.SetType{ElemType: types.StringType},
						"require_last_push_approval":      types.BoolType,
					}})
				}
				if newState.RequiredStatusChecks.IsNull() {
					newState.RequiredStatusChecks = types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{
						"strict":   types.BoolType,
						"contexts": types.SetType{ElemType: types.StringType},
					}})
				}
				if newState.ForcePushBypassers.IsNull() {
					newState.ForcePushBypassers = types.SetNull(types.StringType)
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
			},
		},
	}
}

func (r *githubBranchProtectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Expected format: repository_name:pattern
	idParts := strings.Split(req.ID, ":")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: repository_name:pattern. Got: %q", req.ID),
		)
		return
	}

	repoName := idParts[0]
	pattern := idParts[1]

	// Get repository ID
	repoID, err := getRepositoryID(repoName, r.client)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Getting Repository ID",
			"Could not get repository ID: "+err.Error(),
		)
		return
	}

	// Get branch protection rule ID
	id, err := r.getBranchProtectionID(ctx, repoID.(string), pattern)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Getting Branch Protection Rule ID",
			"Could not get branch protection rule ID: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository_id"), repoID.(string))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// Helper functions

func (r *githubBranchProtectionResource) readBranchProtection(ctx context.Context, state *githubBranchProtectionResourceModel, diagnostics *diag.Diagnostics) {
	var query struct {
		Node struct {
			Node BranchProtectionRule `graphql:"... on BranchProtectionRule"`
		} `graphql:"node(id: $id)"`
	}

	variables := map[string]any{
		"id": state.ID.ValueString(),
	}

	client := r.client.V4Client()
	err := client.Query(ctx, &query, variables)
	if err != nil {
		if strings.Contains(err.Error(), "Could not resolve to a node with the global id") {
			log.Printf("[INFO] Removing branch protection (%s) from state because it no longer exists in GitHub", state.ID.ValueString())
			state.ID = types.StringValue("")
			return
		}
		diagnostics.AddError(
			"Error Reading Branch Protection Rule",
			"Could not read branch protection rule: "+err.Error(),
		)
		return
	}

	protection := query.Node.Node

	// Map the response back to the state
	state.Pattern = types.StringValue(string(protection.Pattern))
	state.AllowsDeletions = types.BoolValue(bool(protection.AllowsDeletions))
	state.AllowsForcePushes = types.BoolValue(bool(protection.AllowsForcePushes))
	state.EnforceAdmins = types.BoolValue(bool(protection.IsAdminEnforced))
	state.RequireSignedCommits = types.BoolValue(bool(protection.RequiresCommitSignatures))
	state.RequiredLinearHistory = types.BoolValue(bool(protection.RequiresLinearHistory))
	state.RequireConversationResolution = types.BoolValue(bool(protection.RequiresConversationResolution))
	state.LockBranch = types.BoolValue(bool(protection.LockBranch))

	// Set complex nested attributes
	r.setApprovingReviews(ctx, state, protection, diagnostics)
	r.setStatusChecks(ctx, state, protection, diagnostics)
	r.setPushes(ctx, state, protection, diagnostics)
	r.setForcePushBypassers(ctx, state, protection, diagnostics)
}

// Additional helper methods would go here...
// Due to the complexity and length, I'll provide the core structure
// and the key CRUD operations. The remaining helper methods follow
// similar patterns to the SDKv2 implementation but adapted for Framework.

type branchProtectionData struct {
	AllowsDeletions                bool
	AllowsForcePushes              bool
	BlocksCreations                bool
	BypassForcePushActorIDs        []string
	BypassPullRequestActorIDs      []string
	DismissesStaleReviews          bool
	IsAdminEnforced                bool
	Pattern                        string
	PushActorIDs                   []string
	RepositoryID                   string
	RequiredApprovingReviewCount   int
	RequiredStatusCheckContexts    []string
	RequiresApprovingReviews       bool
	RequiresCodeOwnerReviews       bool
	RequiresCommitSignatures       bool
	RequiresLinearHistory          bool
	RequiresConversationResolution bool
	RequiresStatusChecks           bool
	RequiresStrictStatusChecks     bool
	RestrictsPushes                bool
	RestrictsReviewDismissals      bool
	ReviewDismissalActorIDs        []string
	RequireLastPushApproval        bool
	LockBranch                     bool
}

func (r *githubBranchProtectionResource) planToBranchProtectionData(ctx context.Context, plan githubBranchProtectionResourceModel) (branchProtectionData, error) {
	data := branchProtectionData{}

	// Set repository ID (resolve if needed)
	repoID, err := getRepositoryID(plan.RepositoryID.ValueString(), r.client)
	if err != nil {
		return data, err
	}
	data.RepositoryID = repoID.(string)

	// Set basic attributes
	data.Pattern = plan.Pattern.ValueString()
	data.AllowsDeletions = plan.AllowsDeletions.ValueBool()
	data.AllowsForcePushes = plan.AllowsForcePushes.ValueBool()
	data.IsAdminEnforced = plan.EnforceAdmins.ValueBool()
	data.RequiresCommitSignatures = plan.RequireSignedCommits.ValueBool()
	data.RequiresLinearHistory = plan.RequiredLinearHistory.ValueBool()
	data.RequiresConversationResolution = plan.RequireConversationResolution.ValueBool()
	data.LockBranch = plan.LockBranch.ValueBool()

	// Extract complex nested attributes
	if err := r.extractApprovingReviews(ctx, plan, &data); err != nil {
		return data, err
	}
	if err := r.extractStatusChecks(ctx, plan, &data); err != nil {
		return data, err
	}
	if err := r.extractPushRestrictions(ctx, plan, &data); err != nil {
		return data, err
	}
	if err := r.extractForcePushBypassers(ctx, plan, &data); err != nil {
		return data, err
	}

	return data, nil
}

func (r *githubBranchProtectionResource) dataToCreateInput(data branchProtectionData) githubv4.CreateBranchProtectionRuleInput {
	return githubv4.CreateBranchProtectionRuleInput{
		AllowsDeletions:                githubv4.NewBoolean(githubv4.Boolean(data.AllowsDeletions)),
		AllowsForcePushes:              githubv4.NewBoolean(githubv4.Boolean(data.AllowsForcePushes)),
		BlocksCreations:                githubv4.NewBoolean(githubv4.Boolean(data.BlocksCreations)),
		BypassForcePushActorIDs:        r.githubv4NewIDSlice(r.githubv4IDSliceEmpty(data.BypassForcePushActorIDs)),
		BypassPullRequestActorIDs:      r.githubv4NewIDSlice(r.githubv4IDSliceEmpty(data.BypassPullRequestActorIDs)),
		DismissesStaleReviews:          githubv4.NewBoolean(githubv4.Boolean(data.DismissesStaleReviews)),
		IsAdminEnforced:                githubv4.NewBoolean(githubv4.Boolean(data.IsAdminEnforced)),
		Pattern:                        githubv4.String(data.Pattern),
		PushActorIDs:                   r.githubv4NewIDSlice(r.githubv4IDSlice(data.PushActorIDs)),
		RepositoryID:                   githubv4.NewID(githubv4.ID(data.RepositoryID)),
		RequiredApprovingReviewCount:   githubv4.NewInt(githubv4.Int(data.RequiredApprovingReviewCount)),
		RequiredStatusCheckContexts:    r.githubv4NewStringSlice(r.githubv4StringSliceEmpty(data.RequiredStatusCheckContexts)),
		RequiresApprovingReviews:       githubv4.NewBoolean(githubv4.Boolean(data.RequiresApprovingReviews)),
		RequiresCodeOwnerReviews:       githubv4.NewBoolean(githubv4.Boolean(data.RequiresCodeOwnerReviews)),
		RequiresCommitSignatures:       githubv4.NewBoolean(githubv4.Boolean(data.RequiresCommitSignatures)),
		RequiresConversationResolution: githubv4.NewBoolean(githubv4.Boolean(data.RequiresConversationResolution)),
		RequiresLinearHistory:          githubv4.NewBoolean(githubv4.Boolean(data.RequiresLinearHistory)),
		RequiresStatusChecks:           githubv4.NewBoolean(githubv4.Boolean(data.RequiresStatusChecks)),
		RequiresStrictStatusChecks:     githubv4.NewBoolean(githubv4.Boolean(data.RequiresStrictStatusChecks)),
		RestrictsPushes:                githubv4.NewBoolean(githubv4.Boolean(data.RestrictsPushes)),
		RestrictsReviewDismissals:      githubv4.NewBoolean(githubv4.Boolean(data.RestrictsReviewDismissals)),
		ReviewDismissalActorIDs:        r.githubv4NewIDSlice(r.githubv4IDSlice(data.ReviewDismissalActorIDs)),
		LockBranch:                     githubv4.NewBoolean(githubv4.Boolean(data.LockBranch)),
		RequireLastPushApproval:        githubv4.NewBoolean(githubv4.Boolean(data.RequireLastPushApproval)),
	}
}

func (r *githubBranchProtectionResource) dataToUpdateInput(data branchProtectionData, id string) githubv4.UpdateBranchProtectionRuleInput {
	return githubv4.UpdateBranchProtectionRuleInput{
		BranchProtectionRuleID:         id,
		AllowsDeletions:                githubv4.NewBoolean(githubv4.Boolean(data.AllowsDeletions)),
		AllowsForcePushes:              githubv4.NewBoolean(githubv4.Boolean(data.AllowsForcePushes)),
		BlocksCreations:                githubv4.NewBoolean(githubv4.Boolean(data.BlocksCreations)),
		BypassForcePushActorIDs:        r.githubv4NewIDSlice(r.githubv4IDSliceEmpty(data.BypassForcePushActorIDs)),
		BypassPullRequestActorIDs:      r.githubv4NewIDSlice(r.githubv4IDSliceEmpty(data.BypassPullRequestActorIDs)),
		DismissesStaleReviews:          githubv4.NewBoolean(githubv4.Boolean(data.DismissesStaleReviews)),
		IsAdminEnforced:                githubv4.NewBoolean(githubv4.Boolean(data.IsAdminEnforced)),
		Pattern:                        githubv4.NewString(githubv4.String(data.Pattern)),
		PushActorIDs:                   r.githubv4NewIDSlice(r.githubv4IDSlice(data.PushActorIDs)),
		RequiredApprovingReviewCount:   githubv4.NewInt(githubv4.Int(data.RequiredApprovingReviewCount)),
		RequiredStatusCheckContexts:    r.githubv4NewStringSlice(r.githubv4StringSliceEmpty(data.RequiredStatusCheckContexts)),
		RequiresApprovingReviews:       githubv4.NewBoolean(githubv4.Boolean(data.RequiresApprovingReviews)),
		RequiresCodeOwnerReviews:       githubv4.NewBoolean(githubv4.Boolean(data.RequiresCodeOwnerReviews)),
		RequiresCommitSignatures:       githubv4.NewBoolean(githubv4.Boolean(data.RequiresCommitSignatures)),
		RequiresConversationResolution: githubv4.NewBoolean(githubv4.Boolean(data.RequiresConversationResolution)),
		RequiresLinearHistory:          githubv4.NewBoolean(githubv4.Boolean(data.RequiresLinearHistory)),
		RequiresStatusChecks:           githubv4.NewBoolean(githubv4.Boolean(data.RequiresStatusChecks)),
		RequiresStrictStatusChecks:     githubv4.NewBoolean(githubv4.Boolean(data.RequiresStrictStatusChecks)),
		RestrictsPushes:                githubv4.NewBoolean(githubv4.Boolean(data.RestrictsPushes)),
		RestrictsReviewDismissals:      githubv4.NewBoolean(githubv4.Boolean(data.RestrictsReviewDismissals)),
		ReviewDismissalActorIDs:        r.githubv4NewIDSlice(r.githubv4IDSlice(data.ReviewDismissalActorIDs)),
		LockBranch:                     githubv4.NewBoolean(githubv4.Boolean(data.LockBranch)),
		RequireLastPushApproval:        githubv4.NewBoolean(githubv4.Boolean(data.RequireLastPushApproval)),
	}
}

func (r *githubBranchProtectionResource) resolveActorIDs(ctx context.Context, data *branchProtectionData) error {
	var err error

	if len(data.ReviewDismissalActorIDs) > 0 {
		data.ReviewDismissalActorIDs, err = r.getActorIds(data.ReviewDismissalActorIDs)
		if err != nil {
			return err
		}
	}

	if len(data.PushActorIDs) > 0 {
		data.PushActorIDs, err = r.getActorIds(data.PushActorIDs)
		if err != nil {
			return err
		}
	}

	if len(data.BypassForcePushActorIDs) > 0 {
		data.BypassForcePushActorIDs, err = r.getActorIds(data.BypassForcePushActorIDs)
		if err != nil {
			return err
		}
	}

	if len(data.BypassPullRequestActorIDs) > 0 {
		data.BypassPullRequestActorIDs, err = r.getActorIds(data.BypassPullRequestActorIDs)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *githubBranchProtectionResource) extractApprovingReviews(ctx context.Context, plan githubBranchProtectionResourceModel, data *branchProtectionData) error {
	if plan.RequiredPullRequestReviews.IsNull() || len(plan.RequiredPullRequestReviews.Elements()) == 0 {
		return nil
	}

	data.RequiresApprovingReviews = true

	// Get the first (and only) element
	var reviews []requiredPullRequestReviewsModel
	diags := plan.RequiredPullRequestReviews.ElementsAs(ctx, &reviews, false)
	if diags.HasError() {
		return fmt.Errorf("error extracting required pull request reviews: %v", diags)
	}

	if len(reviews) > 0 {
		review := reviews[0]
		data.RequiredApprovingReviewCount = int(review.RequiredApprovingReviewCount.ValueInt64())
		data.RequiresCodeOwnerReviews = review.RequireCodeOwnerReviews.ValueBool()
		data.DismissesStaleReviews = review.DismissStaleReviews.ValueBool()
		data.RestrictsReviewDismissals = review.RestrictDismissals.ValueBool()
		data.RequireLastPushApproval = review.RequireLastPushApproval.ValueBool()

		// Extract dismissal restrictions
		if !review.DismissalRestrictions.IsNull() {
			var dismissalRestrictions []string
			diags := review.DismissalRestrictions.ElementsAs(ctx, &dismissalRestrictions, false)
			if diags.HasError() {
				return fmt.Errorf("error extracting dismissal restrictions: %v", diags)
			}
			if len(dismissalRestrictions) > 0 {
				data.ReviewDismissalActorIDs = dismissalRestrictions
				data.RestrictsReviewDismissals = true
			}
		}

		// Extract pull request bypassers
		if !review.PullRequestBypassers.IsNull() {
			var bypassers []string
			diags := review.PullRequestBypassers.ElementsAs(ctx, &bypassers, false)
			if diags.HasError() {
				return fmt.Errorf("error extracting pull request bypassers: %v", diags)
			}
			data.BypassPullRequestActorIDs = bypassers
		}
	}

	return nil
}

func (r *githubBranchProtectionResource) extractStatusChecks(ctx context.Context, plan githubBranchProtectionResourceModel, data *branchProtectionData) error {
	if plan.RequiredStatusChecks.IsNull() || len(plan.RequiredStatusChecks.Elements()) == 0 {
		return nil
	}

	data.RequiresStatusChecks = true

	// Get the first (and only) element
	var statusChecks []requiredStatusChecksModel
	diags := plan.RequiredStatusChecks.ElementsAs(ctx, &statusChecks, false)
	if diags.HasError() {
		return fmt.Errorf("error extracting required status checks: %v", diags)
	}

	if len(statusChecks) > 0 {
		check := statusChecks[0]
		data.RequiresStrictStatusChecks = check.Strict.ValueBool()

		// Extract contexts
		if !check.Contexts.IsNull() {
			var contexts []string
			diags := check.Contexts.ElementsAs(ctx, &contexts, false)
			if diags.HasError() {
				return fmt.Errorf("error extracting status check contexts: %v", diags)
			}
			data.RequiredStatusCheckContexts = contexts
		}
	}

	return nil
}

func (r *githubBranchProtectionResource) extractPushRestrictions(ctx context.Context, plan githubBranchProtectionResourceModel, data *branchProtectionData) error {
	if plan.RestrictPushes.IsNull() || len(plan.RestrictPushes.Elements()) == 0 {
		return nil
	}

	data.RestrictsPushes = true

	// Get the first (and only) element
	var pushRestrictions []restrictPushesModel
	diags := plan.RestrictPushes.ElementsAs(ctx, &pushRestrictions, false)
	if diags.HasError() {
		return fmt.Errorf("error extracting push restrictions: %v", diags)
	}

	if len(pushRestrictions) > 0 {
		restriction := pushRestrictions[0]
		data.BlocksCreations = restriction.BlocksCreations.ValueBool()

		// Extract push allowances
		if !restriction.PushAllowances.IsNull() {
			var allowances []string
			diags := restriction.PushAllowances.ElementsAs(ctx, &allowances, false)
			if diags.HasError() {
				return fmt.Errorf("error extracting push allowances: %v", diags)
			}
			data.PushActorIDs = allowances
		}
	}

	return nil
}

func (r *githubBranchProtectionResource) extractForcePushBypassers(ctx context.Context, plan githubBranchProtectionResourceModel, data *branchProtectionData) error {
	if plan.ForcePushBypassers.IsNull() {
		return nil
	}

	var bypassers []string
	diags := plan.ForcePushBypassers.ElementsAs(ctx, &bypassers, false)
	if diags.HasError() {
		return fmt.Errorf("error extracting force push bypassers: %v", diags)
	}

	if len(bypassers) > 0 {
		data.BypassForcePushActorIDs = bypassers
		data.AllowsForcePushes = false
	}

	return nil
}

func (r *githubBranchProtectionResource) setApprovingReviews(ctx context.Context, state *githubBranchProtectionResourceModel, protection BranchProtectionRule, diagnostics *diag.Diagnostics) {
	if !protection.RequiresApprovingReviews {
		state.RequiredPullRequestReviews = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"required_approving_review_count": types.Int64Type,
				"require_code_owner_reviews":      types.BoolType,
				"dismiss_stale_reviews":           types.BoolType,
				"restrict_dismissals":             types.BoolType,
				"dismissal_restrictions":          types.SetType{ElemType: types.StringType},
				"pull_request_bypassers":          types.SetType{ElemType: types.StringType},
				"require_last_push_approval":      types.BoolType,
			},
		})
		return
	}

	// Set dismissal actors and pull request bypassers using helper functions from SDKv2
	data := BranchProtectionResourceData{
		ReviewDismissalActorIDs:   make([]string, 0),
		BypassPullRequestActorIDs: make([]string, 0),
	}

	dismissalActors := r.setDismissalActorIDs(protection.ReviewDismissalAllowances.Nodes, data)
	bypassPullRequestActors := r.setBypassPullRequestActorIDs(protection.BypassPullRequestAllowances.Nodes, data)

	dismissalSet, diags := types.SetValueFrom(ctx, types.StringType, dismissalActors)
	diagnostics.Append(diags...)

	bypassSet, diags := types.SetValueFrom(ctx, types.StringType, bypassPullRequestActors)
	diagnostics.Append(diags...)

	reviewObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"required_approving_review_count": types.Int64Type,
			"require_code_owner_reviews":      types.BoolType,
			"dismiss_stale_reviews":           types.BoolType,
			"restrict_dismissals":             types.BoolType,
			"dismissal_restrictions":          types.SetType{ElemType: types.StringType},
			"pull_request_bypassers":          types.SetType{ElemType: types.StringType},
			"require_last_push_approval":      types.BoolType,
		},
		map[string]attr.Value{
			"required_approving_review_count": types.Int64Value(int64(protection.RequiredApprovingReviewCount)),
			"require_code_owner_reviews":      types.BoolValue(bool(protection.RequiresCodeOwnerReviews)),
			"dismiss_stale_reviews":           types.BoolValue(bool(protection.DismissesStaleReviews)),
			"restrict_dismissals":             types.BoolValue(bool(protection.RestrictsReviewDismissals)),
			"dismissal_restrictions":          dismissalSet,
			"pull_request_bypassers":          bypassSet,
			"require_last_push_approval":      types.BoolValue(bool(protection.RequireLastPushApproval)),
		},
	)
	diagnostics.Append(diags...)

	reviewList, diags := types.ListValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"required_approving_review_count": types.Int64Type,
			"require_code_owner_reviews":      types.BoolType,
			"dismiss_stale_reviews":           types.BoolType,
			"restrict_dismissals":             types.BoolType,
			"dismissal_restrictions":          types.SetType{ElemType: types.StringType},
			"pull_request_bypassers":          types.SetType{ElemType: types.StringType},
			"require_last_push_approval":      types.BoolType,
		},
	}, []attr.Value{reviewObj})
	diagnostics.Append(diags...)

	state.RequiredPullRequestReviews = reviewList
}

func (r *githubBranchProtectionResource) setStatusChecks(ctx context.Context, state *githubBranchProtectionResourceModel, protection BranchProtectionRule, diagnostics *diag.Diagnostics) {
	if !protection.RequiresStatusChecks {
		state.RequiredStatusChecks = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"strict":   types.BoolType,
				"contexts": types.SetType{ElemType: types.StringType},
			},
		})
		return
	}

	contextsSlice := make([]string, len(protection.RequiredStatusCheckContexts))
	for i, context := range protection.RequiredStatusCheckContexts {
		contextsSlice[i] = string(context)
	}

	contextsSet, diags := types.SetValueFrom(ctx, types.StringType, contextsSlice)
	diagnostics.Append(diags...)

	statusObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"strict":   types.BoolType,
			"contexts": types.SetType{ElemType: types.StringType},
		},
		map[string]attr.Value{
			"strict":   types.BoolValue(bool(protection.RequiresStrictStatusChecks)),
			"contexts": contextsSet,
		},
	)
	diagnostics.Append(diags...)

	statusList, diags := types.ListValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"strict":   types.BoolType,
			"contexts": types.SetType{ElemType: types.StringType},
		},
	}, []attr.Value{statusObj})
	diagnostics.Append(diags...)

	state.RequiredStatusChecks = statusList
}

func (r *githubBranchProtectionResource) setPushes(ctx context.Context, state *githubBranchProtectionResourceModel, protection BranchProtectionRule, diagnostics *diag.Diagnostics) {
	if !protection.RestrictsPushes {
		state.RestrictPushes = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"blocks_creations": types.BoolType,
				"push_allowances":  types.SetType{ElemType: types.StringType},
			},
		})
		return
	}

	data := BranchProtectionResourceData{
		PushActorIDs: make([]string, 0),
	}

	pushActors := r.setPushActorIDs(protection.PushAllowances.Nodes, data)

	allowancesSet, diags := types.SetValueFrom(ctx, types.StringType, pushActors)
	diagnostics.Append(diags...)

	pushObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"blocks_creations": types.BoolType,
			"push_allowances":  types.SetType{ElemType: types.StringType},
		},
		map[string]attr.Value{
			"blocks_creations": types.BoolValue(bool(protection.BlocksCreations)),
			"push_allowances":  allowancesSet,
		},
	)
	diagnostics.Append(diags...)

	pushList, diags := types.ListValue(types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"blocks_creations": types.BoolType,
			"push_allowances":  types.SetType{ElemType: types.StringType},
		},
	}, []attr.Value{pushObj})
	diagnostics.Append(diags...)

	state.RestrictPushes = pushList
}

func (r *githubBranchProtectionResource) setForcePushBypassers(ctx context.Context, state *githubBranchProtectionResourceModel, protection BranchProtectionRule, diagnostics *diag.Diagnostics) {
	if protection.AllowsForcePushes {
		state.ForcePushBypassers = types.SetNull(types.StringType)
		return
	}

	data := BranchProtectionResourceData{
		BypassForcePushActorIDs: make([]string, 0),
	}

	bypassForcePushActors := r.setBypassForcePushActorIDs(protection.BypassForcePushAllowances.Nodes, data)

	bypassSet, diags := types.SetValueFrom(ctx, types.StringType, bypassForcePushActors)
	diagnostics.Append(diags...)

	state.ForcePushBypassers = bypassSet
}

func (r *githubBranchProtectionResource) getBranchProtectionID(ctx context.Context, repoID, pattern string) (string, error) {
	id, err := r.getBranchProtectionIDInternal(githubv4.ID(repoID), pattern)
	if err != nil {
		return "", err
	}
	return id.(string), nil
}

// Helper methods copied from SDKv2 implementation and adapted for Framework

func (r *githubBranchProtectionResource) getActorIds(data []string) ([]string, error) {
	var actors []string
	for _, v := range data {
		id, err := r.getNodeIDv4(v)
		if err != nil {
			return []string{}, err
		}
		log.Printf("[DEBUG] Retrieved node ID for user/team : %s - node ID : %s", v, id)
		actors = append(actors, id)
	}
	return actors, nil
}

func (r *githubBranchProtectionResource) getNodeIDv4(userOrSlug string) (string, error) {
	orgName := r.client.Name()
	ctx := context.Background()
	client := r.client.V4Client()

	if strings.HasPrefix(userOrSlug, orgName+"/") {
		var queryTeam struct {
			Organization struct {
				Team struct {
					ID string
				} `graphql:"team(slug: $slug)"`
			} `graphql:"organization(login: $organization)"`
		}
		teamName := strings.TrimPrefix(userOrSlug, orgName+"/")
		variablesTeam := map[string]any{
			"slug":         githubv4.String(teamName),
			"organization": githubv4.String(orgName),
		}

		err := client.Query(ctx, &queryTeam, variablesTeam)
		if err != nil {
			return "", err
		}
		log.Printf("[DEBUG] Retrieved node ID for team %s. ID is %s", userOrSlug, queryTeam.Organization.Team.ID)
		return queryTeam.Organization.Team.ID, nil
	} else if strings.HasPrefix(userOrSlug, "/") {
		// The "/" prefix indicates a username
		var queryUser struct {
			User struct {
				ID string
			} `graphql:"user(login: $user)"`
		}
		userName := strings.TrimPrefix(userOrSlug, "/")
		variablesUser := map[string]any{
			"user": githubv4.String(userName),
		}

		err := client.Query(ctx, &queryUser, variablesUser)
		if err != nil {
			return "", err
		}
		log.Printf("[DEBUG] Retrieved node ID for user %s. ID is %s", userOrSlug, queryUser.User.ID)
		return queryUser.User.ID, nil
	} else {
		// If userOrSlug does not contain the team or username prefix, assume it is a node ID
		return userOrSlug, nil
	}
}

func (r *githubBranchProtectionResource) getBranchProtectionIDInternal(repoID githubv4.ID, pattern string) (githubv4.ID, error) {
	var query struct {
		Node struct {
			Repository struct {
				BranchProtectionRules struct {
					Nodes []struct {
						ID      string
						Pattern string
					}
					PageInfo PageInfo
				} `graphql:"branchProtectionRules(first: $first, after: $cursor)"`
				ID string
			} `graphql:"... on Repository"`
		} `graphql:"node(id: $id)"`
	}
	variables := map[string]any{
		"id":     repoID,
		"first":  githubv4.Int(100),
		"cursor": (*githubv4.String)(nil),
	}

	ctx := context.Background()
	client := r.client.V4Client()

	var allRules []struct {
		ID      string
		Pattern string
	}
	for {
		err := client.Query(ctx, &query, variables)
		if err != nil {
			return nil, err
		}

		allRules = append(allRules, query.Node.Repository.BranchProtectionRules.Nodes...)

		if !query.Node.Repository.BranchProtectionRules.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(query.Node.Repository.BranchProtectionRules.PageInfo.EndCursor)
	}

	for i := range allRules {
		if allRules[i].Pattern == pattern {
			return allRules[i].ID, nil
		}
	}

	return nil, fmt.Errorf("could not find a branch protection rule with the pattern '%s'", pattern)
}

func (r *githubBranchProtectionResource) setDismissalActorIDs(actors []DismissalActorTypes, data BranchProtectionResourceData) []string {
	dismissalActors := make([]string, 0, len(actors))
	orgName := r.client.Name()

	for _, a := range actors {
		IsID := false
		for _, v := range data.ReviewDismissalActorIDs {
			if (a.Actor.Team.ID != nil && a.Actor.Team.ID.(string) == v) || (a.Actor.User.ID != nil && a.Actor.User.ID.(string) == v) || (a.Actor.App.ID != nil && a.Actor.App.ID.(string) == v) {
				dismissalActors = append(dismissalActors, v)
				IsID = true
				break
			}
		}
		if !IsID {
			if a.Actor.Team.Slug != "" {
				dismissalActors = append(dismissalActors, orgName+"/"+string(a.Actor.Team.Slug))
				continue
			}
			if a.Actor.User.Login != "" {
				dismissalActors = append(dismissalActors, "/"+string(a.Actor.User.Login))
				continue
			}
			if a.Actor.App != (Actor{}) {
				dismissalActors = append(dismissalActors, a.Actor.App.ID.(string))
			}
		}
	}
	return dismissalActors
}

func (r *githubBranchProtectionResource) setBypassPullRequestActorIDs(actors []BypassPullRequestActorTypes, data BranchProtectionResourceData) []string {
	bypassActors := make([]string, 0, len(actors))
	orgName := r.client.Name()

	for _, a := range actors {
		IsID := false
		for _, v := range data.BypassPullRequestActorIDs {
			if (a.Actor.Team.ID != nil && a.Actor.Team.ID.(string) == v) || (a.Actor.User.ID != nil && a.Actor.User.ID.(string) == v) || (a.Actor.App.ID != nil && a.Actor.App.ID.(string) == v) {
				bypassActors = append(bypassActors, v)
				IsID = true
				break
			}
		}
		if !IsID {
			if a.Actor.Team.Slug != "" {
				bypassActors = append(bypassActors, orgName+"/"+string(a.Actor.Team.Slug))
				continue
			}
			if a.Actor.User.Login != "" {
				bypassActors = append(bypassActors, "/"+string(a.Actor.User.Login))
				continue
			}
			if a.Actor.App != (Actor{}) {
				bypassActors = append(bypassActors, a.Actor.App.ID.(string))
			}
		}
	}
	return bypassActors
}

func (r *githubBranchProtectionResource) setBypassForcePushActorIDs(actors []BypassForcePushActorTypes, data BranchProtectionResourceData) []string {
	bypassActors := make([]string, 0, len(actors))
	orgName := r.client.Name()

	for _, a := range actors {
		IsID := false
		for _, v := range data.BypassForcePushActorIDs {
			if (a.Actor.Team.ID != nil && a.Actor.Team.ID.(string) == v) || (a.Actor.User.ID != nil && a.Actor.User.ID.(string) == v) || (a.Actor.App.ID != nil && a.Actor.App.ID.(string) == v) {
				bypassActors = append(bypassActors, v)
				IsID = true
				break
			}
		}
		if !IsID {
			if a.Actor.Team.Slug != "" {
				bypassActors = append(bypassActors, orgName+"/"+string(a.Actor.Team.Slug))
				continue
			}
			if a.Actor.User.Login != "" {
				bypassActors = append(bypassActors, "/"+string(a.Actor.User.Login))
				continue
			}
			if a.Actor.App != (Actor{}) {
				bypassActors = append(bypassActors, a.Actor.App.ID.(string))
			}
		}
	}
	return bypassActors
}

func (r *githubBranchProtectionResource) setPushActorIDs(actors []PushActorTypes, data BranchProtectionResourceData) []string {
	pushActors := make([]string, 0, len(actors))
	orgName := r.client.Name()

	for _, a := range actors {
		IsID := false
		for _, v := range data.PushActorIDs {
			if (a.Actor.Team.ID != nil && a.Actor.Team.ID.(string) == v) || (a.Actor.User.ID != nil && a.Actor.User.ID.(string) == v) || (a.Actor.App.ID != nil && a.Actor.App.ID.(string) == v) {
				pushActors = append(pushActors, v)
				IsID = true
				break
			}
		}
		if !IsID {
			if a.Actor.Team.Slug != "" {
				pushActors = append(pushActors, orgName+"/"+string(a.Actor.Team.Slug))
				continue
			}
			if a.Actor.User.Login != "" {
				pushActors = append(pushActors, "/"+string(a.Actor.User.Login))
				continue
			}
			if a.Actor.App != (Actor{}) {
				pushActors = append(pushActors, a.Actor.App.ID.(string))
			}
		}
	}
	return pushActors
}

// GitHubv4 helper functions adapted from SDKv2 util_v4.go
func (r *githubBranchProtectionResource) githubv4StringSliceEmpty(ss []string) []githubv4.String {
	vGh4 := make([]githubv4.String, 0)
	for _, s := range ss {
		vGh4 = append(vGh4, githubv4.String(s))
	}
	return vGh4
}

func (r *githubBranchProtectionResource) githubv4IDSlice(ss []string) []githubv4.ID {
	var vGh4 []githubv4.ID
	for _, s := range ss {
		vGh4 = append(vGh4, githubv4.ID(s))
	}
	return vGh4
}

func (r *githubBranchProtectionResource) githubv4IDSliceEmpty(ss []string) []githubv4.ID {
	vGh4 := make([]githubv4.ID, 0)
	for _, s := range ss {
		vGh4 = append(vGh4, githubv4.ID(s))
	}
	return vGh4
}

func (r *githubBranchProtectionResource) githubv4NewStringSlice(v []githubv4.String) *[]githubv4.String {
	return &v
}

func (r *githubBranchProtectionResource) githubv4NewIDSlice(v []githubv4.ID) *[]githubv4.ID {
	return &v
}
