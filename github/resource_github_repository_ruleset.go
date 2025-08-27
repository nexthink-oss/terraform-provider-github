package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                 = &githubRepositoryRulesetResource{}
	_ resource.ResourceWithConfigure    = &githubRepositoryRulesetResource{}
	_ resource.ResourceWithImportState  = &githubRepositoryRulesetResource{}
	_ resource.ResourceWithUpgradeState = &githubRepositoryRulesetResource{}
)

type githubRepositoryRulesetResource struct {
	client *Owner
}

// Main resource model
type githubRepositoryRulesetResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Target       types.String `tfsdk:"target"`
	Repository   types.String `tfsdk:"repository"`
	Enforcement  types.String `tfsdk:"enforcement"`
	BypassActors types.List   `tfsdk:"bypass_actors"`
	NodeID       types.String `tfsdk:"node_id"`
	RulesetID    types.Int64  `tfsdk:"ruleset_id"`
	Conditions   types.List   `tfsdk:"conditions"`
	Rules        types.List   `tfsdk:"rules"`
	ETag         types.String `tfsdk:"etag"`
}

// Bypass Actor model
type bypassActorModel struct {
	ActorID    types.Int64  `tfsdk:"actor_id"`
	ActorType  types.String `tfsdk:"actor_type"`
	BypassMode types.String `tfsdk:"bypass_mode"`
}

// Conditions model
type conditionsModel struct {
	RefName types.List `tfsdk:"ref_name"`
}

type refNameModel struct {
	Include types.List `tfsdk:"include"`
	Exclude types.List `tfsdk:"exclude"`
}

// Rules model
type rulesModel struct {
	Creation                  types.Bool `tfsdk:"creation"`
	Update                    types.Bool `tfsdk:"update"`
	UpdateAllowsFetchAndMerge types.Bool `tfsdk:"update_allows_fetch_and_merge"`
	Deletion                  types.Bool `tfsdk:"deletion"`
	RequiredLinearHistory     types.Bool `tfsdk:"required_linear_history"`
	RequiredDeployments       types.List `tfsdk:"required_deployments"`
	RequiredSignatures        types.Bool `tfsdk:"required_signatures"`
	PullRequest               types.List `tfsdk:"pull_request"`
	RequiredStatusChecks      types.List `tfsdk:"required_status_checks"`
	MergeQueue                types.List `tfsdk:"merge_queue"`
	NonFastForward            types.Bool `tfsdk:"non_fast_forward"`
	CommitMessagePattern      types.List `tfsdk:"commit_message_pattern"`
	CommitAuthorEmailPattern  types.List `tfsdk:"commit_author_email_pattern"`
	CommitterEmailPattern     types.List `tfsdk:"committer_email_pattern"`
	BranchNamePattern         types.List `tfsdk:"branch_name_pattern"`
	TagNamePattern            types.List `tfsdk:"tag_name_pattern"`
	RequiredCodeScanning      types.List `tfsdk:"required_code_scanning"`
}

// Required deployments model
type requiredDeploymentsModel struct {
	RequiredDeploymentEnvironments types.List `tfsdk:"required_deployment_environments"`
}

// Pull request model
type pullRequestModel struct {
	DismissStaleReviewsOnPush         types.Bool  `tfsdk:"dismiss_stale_reviews_on_push"`
	RequireCodeOwnerReview            types.Bool  `tfsdk:"require_code_owner_review"`
	RequireLastPushApproval           types.Bool  `tfsdk:"require_last_push_approval"`
	RequiredApprovingReviewCount      types.Int64 `tfsdk:"required_approving_review_count"`
	RequiredReviewThreadResolution    types.Bool  `tfsdk:"required_review_thread_resolution"`
	AllowMergeCommit                  types.Bool  `tfsdk:"allow_merge_commit"`
	AllowSquashMerge                  types.Bool  `tfsdk:"allow_squash_merge"`
	AllowRebaseMerge                  types.Bool  `tfsdk:"allow_rebase_merge"`
	AutomaticCopilotCodeReviewEnabled types.Bool  `tfsdk:"automatic_copilot_code_review_enabled"`
}

// Required status checks model for ruleset
type rulesetRequiredStatusChecksModel struct {
	RequiredCheck                    types.Set  `tfsdk:"required_check"`
	StrictRequiredStatusChecksPolicy types.Bool `tfsdk:"strict_required_status_checks_policy"`
	DoNotEnforceOnCreate             types.Bool `tfsdk:"do_not_enforce_on_create"`
}

type requiredCheckModel struct {
	Context       types.String `tfsdk:"context"`
	IntegrationID types.Int64  `tfsdk:"integration_id"`
}

// Merge queue model
type mergeQueueModel struct {
	CheckResponseTimeoutMinutes  types.Int64  `tfsdk:"check_response_timeout_minutes"`
	GroupingStrategy             types.String `tfsdk:"grouping_strategy"`
	MaxEntriesToBuild            types.Int64  `tfsdk:"max_entries_to_build"`
	MaxEntriesToMerge            types.Int64  `tfsdk:"max_entries_to_merge"`
	MergeMethod                  types.String `tfsdk:"merge_method"`
	MinEntriesToMerge            types.Int64  `tfsdk:"min_entries_to_merge"`
	MinEntriesToMergeWaitMinutes types.Int64  `tfsdk:"min_entries_to_merge_wait_minutes"`
}

// Pattern rule model (for commit message, email, branch/tag patterns)
type patternRuleModel struct {
	Name     types.String `tfsdk:"name"`
	Negate   types.Bool   `tfsdk:"negate"`
	Operator types.String `tfsdk:"operator"`
	Pattern  types.String `tfsdk:"pattern"`
}

// Required code scanning model
type requiredCodeScanningModel struct {
	RequiredCodeScanningTool types.Set `tfsdk:"required_code_scanning_tool"`
}

type requiredCodeScanningToolModel struct {
	AlertsThreshold         types.String `tfsdk:"alerts_threshold"`
	SecurityAlertsThreshold types.String `tfsdk:"security_alerts_threshold"`
	Tool                    types.String `tfsdk:"tool"`
}

func NewGithubRepositoryRulesetResource() resource.Resource {
	return &githubRepositoryRulesetResource{}
}

func (r *githubRepositoryRulesetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_ruleset"
}

func (r *githubRepositoryRulesetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a GitHub repository ruleset.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ruleset ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the ruleset.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 100),
				},
			},
			"target": schema.StringAttribute{
				Required:    true,
				Description: "Possible values are `branch` and `tag`.",
				Validators: []validator.String{
					stringvalidator.OneOf("branch", "tag"),
				},
			},
			"repository": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the repository to apply rulset to.",
			},
			"enforcement": schema.StringAttribute{
				Required:    true,
				Description: "Possible values for Enforcement are `disabled`, `active`, `evaluate`. Note: `evaluate` is currently only supported for owners of type `organization`.",
				Validators: []validator.String{
					stringvalidator.OneOf("disabled", "active", "evaluate"),
				},
			},
			"node_id": schema.StringAttribute{
				Computed:    true,
				Description: "GraphQL global node id for use with v4 API.",
			},
			"ruleset_id": schema.Int64Attribute{
				Computed:    true,
				Description: "GitHub ID for the ruleset.",
			},
			"etag": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},

		Blocks: map[string]schema.Block{
			"bypass_actors": schema.ListNestedBlock{
				Description: "The actors that can bypass the rules in this ruleset.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"actor_id": schema.Int64Attribute{
							Required:    true,
							Description: "The ID of the actor that can bypass a ruleset. When `actor_type` is `OrganizationAdmin`, this should be set to `1`.",
						},
						"actor_type": schema.StringAttribute{
							Required:    true,
							Description: "The type of actor that can bypass a ruleset. Can be one of: `RepositoryRole`, `Team`, `Integration`, `OrganizationAdmin`, `DeployKey`.",
							Validators: []validator.String{
								stringvalidator.OneOf("RepositoryRole", "Team", "Integration", "OrganizationAdmin", "DeployKey"),
							},
						},
						"bypass_mode": schema.StringAttribute{
							Required:    true,
							Description: "When the specified actor can bypass the ruleset. pull_request means that an actor can only bypass rules on pull requests. Can be one of: `always`, `pull_request`.",
							Validators: []validator.String{
								stringvalidator.OneOf("always", "pull_request"),
							},
						},
					},
				},
			},

			"conditions": schema.ListNestedBlock{
				Description: "Parameters for a repository ruleset ref name condition.",
				Validators: []validator.List{
					// MaxItems: 1 equivalent
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"ref_name": schema.ListNestedBlock{
							Description: "Parameters for a repository ruleset ref name condition.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"include": schema.ListAttribute{
										Required:    true,
										ElementType: types.StringType,
										Description: "Array of ref names or patterns to include. One of these patterns must match for the condition to pass. Also accepts `~DEFAULT_BRANCH` to include the default branch or `~ALL` to include all branches.",
									},
									"exclude": schema.ListAttribute{
										Required:    true,
										ElementType: types.StringType,
										Description: "Array of ref names or patterns to exclude. The condition will not pass if any of these patterns match.",
									},
								},
							},
						},
					},
				},
			},

			"rules": schema.ListNestedBlock{
				Description: "Rules within the ruleset.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"creation": schema.BoolAttribute{
							Optional:    true,
							Description: "Only allow users with bypass permission to create matching refs.",
						},
						"update": schema.BoolAttribute{
							Optional:    true,
							Description: "Only allow users with bypass permission to update matching refs.",
						},
						"update_allows_fetch_and_merge": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "Branch can pull changes from its upstream repository. This is only applicable to forked repositories. Requires `update` to be set to `true`.",
						},
						"deletion": schema.BoolAttribute{
							Optional:    true,
							Description: "Only allow users with bypass permissions to delete matching refs.",
						},
						"required_linear_history": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Prevent merge commits from being pushed to matching branches.",
						},
						"required_signatures": schema.BoolAttribute{
							Optional:    true,
							Description: "Commits pushed to matching branches must have verified signatures.",
						},
						"non_fast_forward": schema.BoolAttribute{
							Optional:    true,
							Description: "Prevent users with push access from force pushing to branches.",
						},
					},

					Blocks: map[string]schema.Block{
						"required_deployments": schema.ListNestedBlock{
							Description: "Choose which environments must be successfully deployed to before branches can be merged into a branch that matches this rule.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"required_deployment_environments": schema.ListAttribute{
										Required:    true,
										ElementType: types.StringType,
										Description: "The environments that must be successfully deployed to before branches can be merged.",
									},
								},
							},
						},

						"pull_request": schema.ListNestedBlock{
							Description: "Require all commits be made to a non-target branch and submitted via a pull request before they can be merged.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"dismiss_stale_reviews_on_push": schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(false),
										Description: "New, reviewable commits pushed will dismiss previous pull request review approvals. Defaults to `false`.",
									},
									"require_code_owner_review": schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(false),
										Description: "Require an approving review in pull requests that modify files that have a designated code owner. Defaults to `false`.",
									},
									"require_last_push_approval": schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(false),
										Description: "Whether the most recent reviewable push must be approved by someone other than the person who pushed it. Defaults to `false`.",
									},
									"required_approving_review_count": schema.Int64Attribute{
										Optional:    true,
										Computed:    true,
										Default:     int64default.StaticInt64(0),
										Description: "The number of approving reviews that are required before a pull request can be merged. Defaults to `0`.",
									},
									"required_review_thread_resolution": schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(false),
										Description: "All conversations on code must be resolved before a pull request can be merged. Defaults to `false`.",
									},
									"allow_merge_commit": schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(true),
										Description: "Whether users can use the web UI to merge pull requests with a merge commit. Defaults to `true`.",
									},
									"allow_squash_merge": schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(true),
										Description: "Whether users can use the web UI to squash merge pull requests. Defaults to `true`.",
									},
									"allow_rebase_merge": schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(true),
										Description: "Whether users can use the web UI to rebase merge pull requests. Defaults to `true`.",
									},
									"automatic_copilot_code_review_enabled": schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(false),
										Description: "Enable GitHub Copilot code review automation. Defaults to `false`.",
									},
								},
							},
						},

						"required_status_checks": schema.ListNestedBlock{
							Description: "Choose which status checks must pass before branches can be merged into a branch that matches this rule. When enabled, commits must first be pushed to another branch, then merged or pushed directly to a branch that matches this rule after status checks have passed.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"strict_required_status_checks_policy": schema.BoolAttribute{
										Optional:    true,
										Description: "Whether pull requests targeting a matching branch must be tested with the latest code. This setting will not take effect unless at least one status check is enabled. Defaults to `false`.",
									},
									"do_not_enforce_on_create": schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(false),
										Description: "Allow repositories and branches to be created if a check would otherwise prohibit it.",
									},
								},
								Blocks: map[string]schema.Block{
									"required_check": schema.SetNestedBlock{
										Description: "Status checks that are required. Several can be defined.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"context": schema.StringAttribute{
													Required:    true,
													Description: "The status check context name that must be present on the commit.",
												},
												"integration_id": schema.Int64Attribute{
													Optional:    true,
													Computed:    true,
													Default:     int64default.StaticInt64(0),
													Description: "The optional integration ID that this status check must originate from.",
												},
											},
										},
									},
								},
							},
						},

						"merge_queue": schema.ListNestedBlock{
							Description: "Merges must be performed via a merge queue.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"check_response_timeout_minutes": schema.Int64Attribute{
										Optional:    true,
										Computed:    true,
										Default:     int64default.StaticInt64(60),
										Description: "Maximum time for a required status check to report a conclusion. After this much time has elapsed, checks that have not reported a conclusion will be assumed to have failed. Defaults to `60`.",
										Validators: []validator.Int64{
											int64validator.Between(0, 360),
										},
									},
									"grouping_strategy": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Default:     stringdefault.StaticString("ALLGREEN"),
										Description: "When set to ALLGREEN, the merge commit created by merge queue for each PR in the group must pass all required checks to merge. When set to HEADGREEN, only the commit at the head of the merge group, i.e. the commit containing changes from all of the PRs in the group, must pass its required checks to merge. Can be one of: ALLGREEN, HEADGREEN. Defaults to `ALLGREEN`.",
										Validators: []validator.String{
											stringvalidator.OneOf("ALLGREEN", "HEADGREEN"),
										},
									},
									"max_entries_to_build": schema.Int64Attribute{
										Optional:    true,
										Computed:    true,
										Default:     int64default.StaticInt64(5),
										Description: "Limit the number of queued pull requests requesting checks and workflow runs at the same time. Defaults to `5`.",
										Validators: []validator.Int64{
											int64validator.Between(0, 100),
										},
									},
									"max_entries_to_merge": schema.Int64Attribute{
										Optional:    true,
										Computed:    true,
										Default:     int64default.StaticInt64(5),
										Description: "The maximum number of PRs that will be merged together in a group. Defaults to `5`.",
										Validators: []validator.Int64{
											int64validator.Between(0, 100),
										},
									},
									"merge_method": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Default:     stringdefault.StaticString("MERGE"),
										Description: "Method to use when merging changes from queued pull requests. Can be one of: MERGE, SQUASH, REBASE. Defaults to `MERGE`.",
										Validators: []validator.String{
											stringvalidator.OneOf("MERGE", "SQUASH", "REBASE"),
										},
									},
									"min_entries_to_merge": schema.Int64Attribute{
										Optional:    true,
										Computed:    true,
										Default:     int64default.StaticInt64(1),
										Description: "The minimum number of PRs that will be merged together in a group. Defaults to `1`.",
										Validators: []validator.Int64{
											int64validator.Between(0, 100),
										},
									},
									"min_entries_to_merge_wait_minutes": schema.Int64Attribute{
										Optional:    true,
										Computed:    true,
										Default:     int64default.StaticInt64(5),
										Description: "The time merge queue should wait after the first PR is added to the queue for the minimum group size to be met. After this time has elapsed, the minimum group size will be ignored and a smaller group will be merged. Defaults to `5`.",
										Validators: []validator.Int64{
											int64validator.Between(0, 360),
										},
									},
								},
							},
						},

						"commit_message_pattern": schema.ListNestedBlock{
							Description: "Parameters to be used for the commit_message_pattern rule. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Optional:    true,
										Description: "How this rule will appear to users.",
									},
									"negate": schema.BoolAttribute{
										Optional:    true,
										Description: "If true, the rule will fail if the pattern matches.",
									},
									"operator": schema.StringAttribute{
										Required:    true,
										Description: "The operator to use for matching. Can be one of: `starts_with`, `ends_with`, `contains`, `regex`.",
									},
									"pattern": schema.StringAttribute{
										Required:    true,
										Description: "The pattern to match with.",
									},
								},
							},
						},

						"commit_author_email_pattern": schema.ListNestedBlock{
							Description: "Parameters to be used for the commit_author_email_pattern rule. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Optional:    true,
										Description: "How this rule will appear to users.",
									},
									"negate": schema.BoolAttribute{
										Optional:    true,
										Description: "If true, the rule will fail if the pattern matches.",
									},
									"operator": schema.StringAttribute{
										Required:    true,
										Description: "The operator to use for matching. Can be one of: `starts_with`, `ends_with`, `contains`, `regex`.",
									},
									"pattern": schema.StringAttribute{
										Required:    true,
										Description: "The pattern to match with.",
									},
								},
							},
						},

						"committer_email_pattern": schema.ListNestedBlock{
							Description: "Parameters to be used for the committer_email_pattern rule. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Optional:    true,
										Description: "How this rule will appear to users.",
									},
									"negate": schema.BoolAttribute{
										Optional:    true,
										Description: "If true, the rule will fail if the pattern matches.",
									},
									"operator": schema.StringAttribute{
										Required:    true,
										Description: "The operator to use for matching. Can be one of: `starts_with`, `ends_with`, `contains`, `regex`.",
									},
									"pattern": schema.StringAttribute{
										Required:    true,
										Description: "The pattern to match with.",
									},
								},
							},
						},

						"branch_name_pattern": schema.ListNestedBlock{
							Description: "Parameters to be used for the branch_name_pattern rule. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations. Conflicts with `tag_name_pattern` as it only applies to rulesets with target `branch`.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Optional:    true,
										Description: "How this rule will appear to users.",
									},
									"negate": schema.BoolAttribute{
										Optional:    true,
										Description: "If true, the rule will fail if the pattern matches.",
									},
									"operator": schema.StringAttribute{
										Required:    true,
										Description: "The operator to use for matching. Can be one of: `starts_with`, `ends_with`, `contains`, `regex`.",
									},
									"pattern": schema.StringAttribute{
										Required:    true,
										Description: "The pattern to match with.",
									},
								},
							},
						},

						"tag_name_pattern": schema.ListNestedBlock{
							Description: "Parameters to be used for the tag_name_pattern rule. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations. Conflicts with `branch_name_pattern` as it only applies to rulesets with target `tag`.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Optional:    true,
										Description: "How this rule will appear to users.",
									},
									"negate": schema.BoolAttribute{
										Optional:    true,
										Description: "If true, the rule will fail if the pattern matches.",
									},
									"operator": schema.StringAttribute{
										Required:    true,
										Description: "The operator to use for matching. Can be one of: `starts_with`, `ends_with`, `contains`, `regex`.",
									},
									"pattern": schema.StringAttribute{
										Required:    true,
										Description: "The pattern to match with.",
									},
								},
							},
						},

						"required_code_scanning": schema.ListNestedBlock{
							Description: "Choose which tools must provide code scanning results before the reference is updated. When configured, code scanning must be enabled and have results for both the commit and the reference being updated.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Blocks: map[string]schema.Block{
									"required_code_scanning_tool": schema.SetNestedBlock{
										Description: "Tools that must provide code scanning results for this rule to pass.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"alerts_threshold": schema.StringAttribute{
													Required:    true,
													Description: "The severity level at which code scanning results that raise alerts block a reference update. Can be one of: `none`, `errors`, `errors_and_warnings`, `all`.",
												},
												"security_alerts_threshold": schema.StringAttribute{
													Required:    true,
													Description: "The severity level at which code scanning results that raise security alerts block a reference update. Can be one of: `none`, `critical`, `high_or_higher`, `medium_or_higher`, `all`.",
												},
												"tool": schema.StringAttribute{
													Required:    true,
													Description: "The name of a code scanning tool",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *githubRepositoryRulesetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubRepositoryRulesetResource) SchemaVersion(ctx context.Context) int64 {
	return 1
}

func (r *githubRepositoryRulesetResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &schema.Schema{
				Description: "Creates a GitHub repository ruleset.",
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Description: "The ruleset ID.",
						Computed:    true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						Required:    true,
						Description: "The name of the ruleset.",
					},
					"target": schema.StringAttribute{
						Required:    true,
						Description: "Possible values are `branch` and `tag`.",
					},
					"repository": schema.StringAttribute{
						Optional:    true,
						Description: "Name of the repository to apply rulset to.",
					},
					"enforcement": schema.StringAttribute{
						Required:    true,
						Description: "Possible values for Enforcement are `disabled`, `active`, `evaluate`. Note: `evaluate` is currently only supported for owners of type `organization`.",
					},
					"node_id": schema.StringAttribute{
						Computed:    true,
						Description: "GraphQL global node id for use with v4 API.",
					},
					"ruleset_id": schema.Int64Attribute{
						Computed:    true,
						Description: "GitHub ID for the ruleset.",
					},
					"etag": schema.StringAttribute{
						Computed: true,
					},
				},
				Blocks: map[string]schema.Block{
					"bypass_actors": schema.ListNestedBlock{
						Description: "The actors that can bypass the rules in this ruleset.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"actor_id": schema.Int64Attribute{
									Required:    true,
									Description: "The ID of the actor that can bypass a ruleset. When `actor_type` is `OrganizationAdmin`, this should be set to `1`.",
								},
								"actor_type": schema.StringAttribute{
									Required:    true,
									Description: "The type of actor that can bypass a ruleset. Can be one of: `RepositoryRole`, `Team`, `Integration`, `OrganizationAdmin`, `DeployKey`.",
								},
								"bypass_mode": schema.StringAttribute{
									Required:    true,
									Description: "When the specified actor can bypass the ruleset. pull_request means that an actor can only bypass rules on pull requests. Can be one of: `always`, `pull_request`.",
								},
							},
						},
					},
					"conditions": schema.ListNestedBlock{
						Description: "Parameters for a repository ruleset ref name condition.",
						NestedObject: schema.NestedBlockObject{
							Blocks: map[string]schema.Block{
								"ref_name": schema.ListNestedBlock{
									Description: "Parameters for a repository ruleset ref name condition.",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"include": schema.ListAttribute{
												Required:    true,
												ElementType: types.StringType,
												Description: "Array of ref names or patterns to include. One of these patterns must match for the condition to pass. Also accepts `~DEFAULT_BRANCH` to include the default branch or `~ALL` to include all branches.",
											},
											"exclude": schema.ListAttribute{
												Required:    true,
												ElementType: types.StringType,
												Description: "Array of ref names or patterns to exclude. The condition will not pass if any of these patterns match.",
											},
										},
									},
								},
							},
						},
					},
					"rules": schema.ListNestedBlock{
						Description: "Rules within the ruleset.",
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"creation": schema.BoolAttribute{
									Optional:    true,
									Description: "Only allow users with bypass permission to create matching refs.",
								},
								"update": schema.BoolAttribute{
									Optional:    true,
									Description: "Only allow users with bypass permission to update matching refs.",
								},
								"update_allows_fetch_and_merge": schema.BoolAttribute{
									Optional:    true,
									Computed:    true,
									Description: "Branch can pull changes from its upstream repository. This is only applicable to forked repositories. Requires `update` to be set to `true`.",
								},
								"deletion": schema.BoolAttribute{
									Optional:    true,
									Description: "Only allow users with bypass permissions to delete matching refs.",
								},
								"required_linear_history": schema.BoolAttribute{
									Optional:    true,
									Computed:    true,
									Description: "Prevent merge commits from being pushed to matching branches.",
								},
								"required_signatures": schema.BoolAttribute{
									Optional:    true,
									Description: "Commits pushed to matching branches must have verified signatures.",
								},
								"non_fast_forward": schema.BoolAttribute{
									Optional:    true,
									Description: "Prevent users with push access from force pushing to branches.",
								},
							},
							Blocks: map[string]schema.Block{
								"required_deployments": schema.ListNestedBlock{
									Description: "Choose which environments must be successfully deployed to before branches can be merged into a branch that matches this rule.",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"required_deployment_environments": schema.ListAttribute{
												Required:    true,
												ElementType: types.StringType,
												Description: "The environments that must be successfully deployed to before branches can be merged.",
											},
										},
									},
								},
								"pull_request": schema.ListNestedBlock{
									Description: "Require all commits be made to a non-target branch and submitted via a pull request before they can be merged.",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"dismiss_stale_reviews_on_push": schema.BoolAttribute{
												Optional:    true,
												Computed:    true,
												Description: "New, reviewable commits pushed will dismiss previous pull request review approvals. Defaults to `false`.",
											},
											"require_code_owner_review": schema.BoolAttribute{
												Optional:    true,
												Computed:    true,
												Description: "Require an approving review in pull requests that modify files that have a designated code owner. Defaults to `false`.",
											},
											"require_last_push_approval": schema.BoolAttribute{
												Optional:    true,
												Computed:    true,
												Description: "Whether the most recent reviewable push must be approved by someone other than the person who pushed it. Defaults to `false`.",
											},
											"required_approving_review_count": schema.Int64Attribute{
												Optional:    true,
												Computed:    true,
												Description: "The number of approving reviews that are required before a pull request can be merged. Defaults to `0`.",
											},
											"required_review_thread_resolution": schema.BoolAttribute{
												Optional:    true,
												Computed:    true,
												Description: "All conversations on code must be resolved before a pull request can be merged. Defaults to `false`.",
											},
											"allow_merge_commit": schema.BoolAttribute{
												Optional:    true,
												Computed:    true,
												Description: "Whether users can use the web UI to merge pull requests with a merge commit. Defaults to `true`.",
											},
											"allow_squash_merge": schema.BoolAttribute{
												Optional:    true,
												Computed:    true,
												Description: "Whether users can use the web UI to squash merge pull requests. Defaults to `true`.",
											},
											"allow_rebase_merge": schema.BoolAttribute{
												Optional:    true,
												Computed:    true,
												Description: "Whether users can use the web UI to rebase merge pull requests. Defaults to `true`.",
											},
											"automatic_copilot_code_review_enabled": schema.BoolAttribute{
												Optional:    true,
												Computed:    true,
												Description: "Enable GitHub Copilot code review automation. Defaults to `false`.",
											},
										},
									},
								},
								"required_status_checks": schema.ListNestedBlock{
									Description: "Choose which status checks must pass before branches can be merged into a branch that matches this rule. When enabled, commits must first be pushed to another branch, then merged or pushed directly to a branch that matches this rule after status checks have passed.",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"strict_required_status_checks_policy": schema.BoolAttribute{
												Optional:    true,
												Description: "Whether pull requests targeting a matching branch must be tested with the latest code. This setting will not take effect unless at least one status check is enabled. Defaults to `false`.",
											},
											"do_not_enforce_on_create": schema.BoolAttribute{
												Optional:    true,
												Computed:    true,
												Description: "Allow repositories and branches to be created if a check would otherwise prohibit it.",
											},
										},
										Blocks: map[string]schema.Block{
											"required_check": schema.SetNestedBlock{
												Description: "Status checks that are required. Several can be defined.",
												NestedObject: schema.NestedBlockObject{
													Attributes: map[string]schema.Attribute{
														"context": schema.StringAttribute{
															Required:    true,
															Description: "The status check context name that must be present on the commit.",
														},
														"integration_id": schema.Int64Attribute{
															Optional:    true,
															Computed:    true,
															Description: "The optional integration ID that this status check must originate from.",
														},
													},
												},
											},
										},
									},
								},
								"merge_queue": schema.ListNestedBlock{
									Description: "Merges must be performed via a merge queue.",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"check_response_timeout_minutes": schema.Int64Attribute{
												Optional:    true,
												Computed:    true,
												Description: "Maximum time for a required status check to report a conclusion. After this much time has elapsed, checks that have not reported a conclusion will be assumed to have failed. Defaults to `60`.",
											},
											"grouping_strategy": schema.StringAttribute{
												Optional:    true,
												Computed:    true,
												Description: "When set to ALLGREEN, the merge commit created by merge queue for each PR in the group must pass all required checks to merge. When set to HEADGREEN, only the commit at the head of the merge group, i.e. the commit containing changes from all of the PRs in the group, must pass its required checks to merge. Can be one of: ALLGREEN, HEADGREEN. Defaults to `ALLGREEN`.",
											},
											"max_entries_to_build": schema.Int64Attribute{
												Optional:    true,
												Computed:    true,
												Description: "Limit the number of queued pull requests requesting checks and workflow runs at the same time. Defaults to `5`.",
											},
											"max_entries_to_merge": schema.Int64Attribute{
												Optional:    true,
												Computed:    true,
												Description: "The maximum number of PRs that will be merged together in a group. Defaults to `5`.",
											},
											"merge_method": schema.StringAttribute{
												Optional:    true,
												Computed:    true,
												Description: "Method to use when merging changes from queued pull requests. Can be one of: MERGE, SQUASH, REBASE. Defaults to `MERGE`.",
											},
											"min_entries_to_merge": schema.Int64Attribute{
												Optional:    true,
												Computed:    true,
												Description: "The minimum number of PRs that will be merged together in a group. Defaults to `1`.",
											},
											"min_entries_to_merge_wait_minutes": schema.Int64Attribute{
												Optional:    true,
												Computed:    true,
												Description: "The time merge queue should wait after the first PR is added to the queue for the minimum group size to be met. After this time has elapsed, the minimum group size will be ignored and a smaller group will be merged. Defaults to `5`.",
											},
										},
									},
								},
								"commit_message_pattern": schema.ListNestedBlock{
									Description: "Parameters to be used for the commit_message_pattern rule. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations.",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"name": schema.StringAttribute{
												Optional:    true,
												Description: "How this rule will appear to users.",
											},
											"negate": schema.BoolAttribute{
												Optional:    true,
												Description: "If true, the rule will fail if the pattern matches.",
											},
											"operator": schema.StringAttribute{
												Required:    true,
												Description: "The operator to use for matching. Can be one of: `starts_with`, `ends_with`, `contains`, `regex`.",
											},
											"pattern": schema.StringAttribute{
												Required:    true,
												Description: "The pattern to match with.",
											},
										},
									},
								},
								"commit_author_email_pattern": schema.ListNestedBlock{
									Description: "Parameters to be used for the commit_author_email_pattern rule. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations.",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"name": schema.StringAttribute{
												Optional:    true,
												Description: "How this rule will appear to users.",
											},
											"negate": schema.BoolAttribute{
												Optional:    true,
												Description: "If true, the rule will fail if the pattern matches.",
											},
											"operator": schema.StringAttribute{
												Required:    true,
												Description: "The operator to use for matching. Can be one of: `starts_with`, `ends_with`, `contains`, `regex`.",
											},
											"pattern": schema.StringAttribute{
												Required:    true,
												Description: "The pattern to match with.",
											},
										},
									},
								},
								"committer_email_pattern": schema.ListNestedBlock{
									Description: "Parameters to be used for the committer_email_pattern rule. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations.",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"name": schema.StringAttribute{
												Optional:    true,
												Description: "How this rule will appear to users.",
											},
											"negate": schema.BoolAttribute{
												Optional:    true,
												Description: "If true, the rule will fail if the pattern matches.",
											},
											"operator": schema.StringAttribute{
												Required:    true,
												Description: "The operator to use for matching. Can be one of: `starts_with`, `ends_with`, `contains`, `regex`.",
											},
											"pattern": schema.StringAttribute{
												Required:    true,
												Description: "The pattern to match with.",
											},
										},
									},
								},
								"branch_name_pattern": schema.ListNestedBlock{
									Description: "Parameters to be used for the branch_name_pattern rule. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations. Conflicts with `tag_name_pattern` as it only applies to rulesets with target `branch`.",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"name": schema.StringAttribute{
												Optional:    true,
												Description: "How this rule will appear to users.",
											},
											"negate": schema.BoolAttribute{
												Optional:    true,
												Description: "If true, the rule will fail if the pattern matches.",
											},
											"operator": schema.StringAttribute{
												Required:    true,
												Description: "The operator to use for matching. Can be one of: `starts_with`, `ends_with`, `contains`, `regex`.",
											},
											"pattern": schema.StringAttribute{
												Required:    true,
												Description: "The pattern to match with.",
											},
										},
									},
								},
								"tag_name_pattern": schema.ListNestedBlock{
									Description: "Parameters to be used for the tag_name_pattern rule. This rule only applies to repositories within an enterprise, it cannot be applied to repositories owned by individuals or regular organizations. Conflicts with `branch_name_pattern` as it only applies to rulesets with target `tag`.",
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"name": schema.StringAttribute{
												Optional:    true,
												Description: "How this rule will appear to users.",
											},
											"negate": schema.BoolAttribute{
												Optional:    true,
												Description: "If true, the rule will fail if the pattern matches.",
											},
											"operator": schema.StringAttribute{
												Required:    true,
												Description: "The operator to use for matching. Can be one of: `starts_with`, `ends_with`, `contains`, `regex`.",
											},
											"pattern": schema.StringAttribute{
												Required:    true,
												Description: "The pattern to match with.",
											},
										},
									},
								},
								"required_code_scanning": schema.ListNestedBlock{
									Description: "Choose which tools must provide code scanning results before the reference is updated. When configured, code scanning must be enabled and have results for both the commit and the reference being updated.",
									NestedObject: schema.NestedBlockObject{
										Blocks: map[string]schema.Block{
											"required_code_scanning_tool": schema.SetNestedBlock{
												Description: "Tools that must provide code scanning results for this rule to pass.",
												NestedObject: schema.NestedBlockObject{
													Attributes: map[string]schema.Attribute{
														"alerts_threshold": schema.StringAttribute{
															Required:    true,
															Description: "The severity level at which code scanning results that raise alerts block a reference update. Can be one of: `none`, `errors`, `errors_and_warnings`, `all`.",
														},
														"security_alerts_threshold": schema.StringAttribute{
															Required:    true,
															Description: "The severity level at which code scanning results that raise security alerts block a reference update. Can be one of: `none`, `critical`, `high_or_higher`, `medium_or_higher`, `all`.",
														},
														"tool": schema.StringAttribute{
															Required:    true,
															Description: "The name of a code scanning tool",
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				// State migration from version 0 to version 1
				// Since the schema is identical between v0 and v1, we can use the PriorSchema
				// to automatically handle the migration
				var priorStateData githubRepositoryRulesetResourceModel

				resp.Diagnostics.Append(req.State.Get(ctx, &priorStateData)...)
				if resp.Diagnostics.HasError() {
					return
				}

				// The state structure is identical, so we can directly set the upgraded state
				resp.Diagnostics.Append(resp.State.Set(ctx, priorStateData)...)
			},
		},
	}
}

func (r *githubRepositoryRulesetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubRepositoryRulesetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Framework model to GitHub API object
	rulesetReq, diags := r.frameworkToAPIRuleset(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := plan.Repository.ValueString()

	ruleset, _, err := r.client.V3Client().Repositories.CreateRuleset(ctx, owner, repoName, *rulesetReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating repository ruleset",
			"Could not create repository ruleset: "+err.Error(),
		)
		return
	}

	plan.RulesetID = types.Int64Value(*ruleset.ID)
	plan.ID = types.StringValue(strconv.FormatInt(*ruleset.ID, 10))

	// Read the created resource to get all computed values
	state, diags := r.readRuleset(ctx, plan.Repository.ValueString(), *ruleset.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubRepositoryRulesetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubRepositoryRulesetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rulesetID := state.RulesetID.ValueInt64()
	repoName := state.Repository.ValueString()

	newState, diags := r.readRuleset(ctx, repoName, rulesetID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if newState == nil {
		// Resource has been deleted
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

func (r *githubRepositoryRulesetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubRepositoryRulesetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Framework model to GitHub API object
	rulesetReq, diags := r.frameworkToAPIRuleset(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := plan.Repository.ValueString()
	rulesetID := plan.RulesetID.ValueInt64()

	ruleset, _, err := r.client.V3Client().Repositories.UpdateRuleset(ctx, owner, repoName, rulesetID, *rulesetReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating repository ruleset",
			"Could not update repository ruleset: "+err.Error(),
		)
		return
	}

	plan.RulesetID = types.Int64Value(*ruleset.ID)
	plan.ID = types.StringValue(strconv.FormatInt(*ruleset.ID, 10))

	// Read the updated resource to get all computed values
	state, diags := r.readRuleset(ctx, plan.Repository.ValueString(), *ruleset.ID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubRepositoryRulesetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubRepositoryRulesetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := state.Repository.ValueString()
	rulesetID := state.RulesetID.ValueInt64()

	log.Printf("[DEBUG] Deleting repository ruleset: %s/%s: %d", owner, repoName, rulesetID)
	_, err := r.client.V3Client().Repositories.DeleteRuleset(ctx, owner, repoName, rulesetID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting repository ruleset",
			"Could not delete repository ruleset: "+err.Error(),
		)
		return
	}
}

func (r *githubRepositoryRulesetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse import ID: repository/ruleset_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: repository/ruleset_id. Got: %q", req.ID),
		)
		return
	}

	repoName := parts[0]
	rulesetIDStr := parts[1]

	rulesetID, err := strconv.ParseInt(rulesetIDStr, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Ruleset ID",
			fmt.Sprintf("Could not parse ruleset ID %q: %s", rulesetIDStr, err.Error()),
		)
		return
	}

	if rulesetID == 0 {
		resp.Diagnostics.AddError(
			"Invalid Ruleset ID",
			"`ruleset_id` must be present",
		)
		return
	}

	log.Printf("[DEBUG] Importing repository ruleset with ID: %d, for repository: %s", rulesetID, repoName)

	// Verify repository exists
	owner := r.client.Name()
	repository, _, err := r.client.V3Client().Repositories.Get(ctx, owner, repoName)
	if repository == nil || err != nil {
		resp.Diagnostics.AddError(
			"Repository Not Found",
			fmt.Sprintf("Could not find repository %s/%s: %s", owner, repoName, err.Error()),
		)
		return
	}

	// Verify ruleset exists and read it
	state, diags := r.readRuleset(ctx, *repository.Name, rulesetID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state == nil {
		resp.Diagnostics.AddError(
			"Ruleset Not Found",
			fmt.Sprintf("Could not find ruleset %d in repository %s/%s", rulesetID, owner, *repository.Name),
		)
		return
	}

	// Set the ID attribute for ImportState
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(strconv.FormatInt(rulesetID, 10)))...)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Helper function to read a ruleset and convert to Framework model
func (r *githubRepositoryRulesetResource) readRuleset(ctx context.Context, repoName string, rulesetID int64) (*githubRepositoryRulesetResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	owner := r.client.Name()

	ruleset, resp, err := r.client.V3Client().Repositories.GetRuleset(ctx, owner, repoName, rulesetID, false)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				// Resource has been deleted
				return nil, diags
			}
		}
		diags.AddError(
			"Error reading repository ruleset",
			"Could not read repository ruleset: "+err.Error(),
		)
		return nil, diags
	}

	// Convert API response to Framework model
	state, convertDiags := r.apiToFrameworkRuleset(ctx, ruleset, repoName)
	diags.Append(convertDiags...)
	if diags.HasError() {
		return nil, diags
	}

	// Set ID to the ruleset ID
	state.ID = types.StringValue(strconv.FormatInt(rulesetID, 10))

	// Set ETag
	if resp != nil {
		state.ETag = types.StringValue(resp.Header.Get("ETag"))
	}

	return state, diags
}

// Continue with conversion helper functions...
