package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/google/go-github/v74/github"
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

type rulesetContextKey string

const rulesetEtagContextKey rulesetContextKey = "etag"

var (
	_ resource.Resource                 = &githubOrganizationRulesetResource{}
	_ resource.ResourceWithConfigure    = &githubOrganizationRulesetResource{}
	_ resource.ResourceWithImportState  = &githubOrganizationRulesetResource{}
	_ resource.ResourceWithUpgradeState = &githubOrganizationRulesetResource{}
)

type githubOrganizationRulesetResource struct {
	client *Owner
}

// Organization Ruleset Resource Model
type githubOrganizationRulesetResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Target       types.String `tfsdk:"target"`
	Enforcement  types.String `tfsdk:"enforcement"`
	BypassActors types.List   `tfsdk:"bypass_actors"`
	NodeID       types.String `tfsdk:"node_id"`
	RulesetID    types.Int64  `tfsdk:"ruleset_id"`
	Conditions   types.List   `tfsdk:"conditions"`
	Rules        types.List   `tfsdk:"rules"`
	ETag         types.String `tfsdk:"etag"`
}

// Organization conditions model - different from repository conditions
type organizationConditionsModel struct {
	RefName        types.List `tfsdk:"ref_name"`
	RepositoryName types.List `tfsdk:"repository_name"`
	RepositoryID   types.List `tfsdk:"repository_id"`
}

type repositoryNameModel struct {
	Include   types.List `tfsdk:"include"`
	Exclude   types.List `tfsdk:"exclude"`
	Protected types.Bool `tfsdk:"protected"`
}

func NewGithubOrganizationRulesetResource() resource.Resource {
	return &githubOrganizationRulesetResource{}
}

func (r *githubOrganizationRulesetResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_ruleset"
}

func (r *githubOrganizationRulesetResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates a GitHub organization ruleset.",

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
				Description: "Possible values are `branch`, `tag` and `push`. Note: The `push` target is in beta and is subject to change.",
				Validators: []validator.String{
					stringvalidator.OneOf("branch", "tag", "push"),
				},
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
				Description: "Parameters for an organization ruleset condition. `ref_name` is required alongside one of `repository_name` or `repository_id`.",
				Validators: []validator.List{
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

						"repository_name": schema.ListNestedBlock{
							Description: "Parameters for repository name condition.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"include": schema.ListAttribute{
										Required:    true,
										ElementType: types.StringType,
										Description: "Array of repository names or patterns to include. One of these patterns must match for the condition to pass. Also accepts `~ALL` to include all repositories.",
									},
									"exclude": schema.ListAttribute{
										Required:    true,
										ElementType: types.StringType,
										Description: "Array of repository names or patterns to exclude. The condition will not pass if any of these patterns match.",
									},
									"protected": schema.BoolAttribute{
										Optional:    true,
										Computed:    true,
										Default:     booldefault.StaticBool(false),
										Description: "Whether renaming of target repositories is prevented.",
									},
								},
							},
						},
					},

					Attributes: map[string]schema.Attribute{
						"repository_id": schema.ListAttribute{
							Optional:    true,
							ElementType: types.Int64Type,
							Description: "The repository IDs that the ruleset applies to. One of these IDs must match for the condition to pass.",
						},
					},
				},
			},

			// Reuse the same rules schema as repository ruleset
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
						"deletion": schema.BoolAttribute{
							Optional:    true,
							Description: "Only allow users with bypass permissions to delete matching refs.",
						},
						"required_linear_history": schema.BoolAttribute{
							Optional:    true,
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

						"commit_message_pattern": schema.ListNestedBlock{
							Description: "Parameters to be used for the commit_message_pattern rule.",
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
							Description: "Parameters to be used for the commit_author_email_pattern rule.",
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
							Description: "Parameters to be used for the committer_email_pattern rule.",
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

						"required_workflows": schema.ListNestedBlock{
							Description: "Choose which Actions workflows must pass before branches can be merged into a branch that matches this rule.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Blocks: map[string]schema.Block{
									"required_workflow": schema.SetNestedBlock{
										Description: "Actions workflows that are required. Several can be defined.",
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"repository_id": schema.Int64Attribute{
													Required:    true,
													Description: "The repository in which the workflow is defined.",
												},
												"path": schema.StringAttribute{
													Required:    true,
													Description: "The path to the workflow YAML definition file.",
												},
												"ref": schema.StringAttribute{
													Optional:    true,
													Computed:    true,
													Default:     stringdefault.StaticString("master"),
													Description: "The ref (branch or tag) of the workflow file to use.",
												},
											},
										},
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
													Description: "The name of a code scanning tool.",
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

func (r *githubOrganizationRulesetResource) SchemaVersion() int64 {
	return 1
}

func (r *githubOrganizationRulesetResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// State upgrade from version 0 (initial Plugin Framework migration) to version 1
		0: {
			PriorSchema: &schema.Schema{
				Description: "Creates a GitHub organization ruleset.",
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
						Description: "Possible values are `branch`, `tag` and `push`. Note: The `push` target is in beta and is subject to change.",
						Validators: []validator.String{
							stringvalidator.OneOf("branch", "tag", "push"),
						},
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
						Description: "Parameters for an organization ruleset condition. `ref_name` is required alongside one of `repository_name` or `repository_id`.",
						Validators: []validator.List{
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
								"repository_name": schema.ListNestedBlock{
									Description: "Parameters for repository name condition.",
									Validators: []validator.List{
										listvalidator.SizeAtMost(1),
									},
									NestedObject: schema.NestedBlockObject{
										Attributes: map[string]schema.Attribute{
											"include": schema.ListAttribute{
												Required:    true,
												ElementType: types.StringType,
												Description: "Array of repository names or patterns to include. One of these patterns must match for the condition to pass. Also accepts `~ALL` to include all repositories.",
											},
											"exclude": schema.ListAttribute{
												Required:    true,
												ElementType: types.StringType,
												Description: "Array of repository names or patterns to exclude. The condition will not pass if any of these patterns match.",
											},
											"protected": schema.BoolAttribute{
												Optional:    true,
												Computed:    true,
												Default:     booldefault.StaticBool(false),
												Description: "Whether renaming of target repositories is prevented.",
											},
										},
									},
								},
							},
							Attributes: map[string]schema.Attribute{
								"repository_id": schema.ListAttribute{
									Optional:    true,
									ElementType: types.Int64Type,
									Description: "The repository IDs that the ruleset applies to. One of these IDs must match for the condition to pass.",
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
								"deletion": schema.BoolAttribute{
									Optional:    true,
									Description: "Only allow users with bypass permissions to delete matching refs.",
								},
								"required_linear_history": schema.BoolAttribute{
									Optional:    true,
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
								"commit_message_pattern": schema.ListNestedBlock{
									Description: "Parameters to be used for the commit_message_pattern rule.",
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
									Description: "Parameters to be used for the commit_author_email_pattern rule.",
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
									Description: "Parameters to be used for the committer_email_pattern rule.",
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
								"required_workflows": schema.ListNestedBlock{
									Description: "Choose which Actions workflows must pass before branches can be merged into a branch that matches this rule.",
									Validators: []validator.List{
										listvalidator.SizeAtMost(1),
									},
									NestedObject: schema.NestedBlockObject{
										Blocks: map[string]schema.Block{
											"required_workflow": schema.SetNestedBlock{
												Description: "Actions workflows that are required. Several can be defined.",
												NestedObject: schema.NestedBlockObject{
													Attributes: map[string]schema.Attribute{
														"repository_id": schema.Int64Attribute{
															Required:    true,
															Description: "The repository in which the workflow is defined.",
														},
														"path": schema.StringAttribute{
															Required:    true,
															Description: "The path to the workflow YAML definition file.",
														},
														"ref": schema.StringAttribute{
															Optional:    true,
															Computed:    true,
															Default:     stringdefault.StaticString("master"),
															Description: "The ref (branch or tag) of the workflow file to use.",
														},
													},
												},
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
															Description: "The name of a code scanning tool.",
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
				// For version 0->1 upgrade, the schema is identical, so we can copy state directly
				// This handles migration from initial Plugin Framework implementation to versioned implementation
				resp.State.Raw = req.State.Raw.Copy()
			},
		},
	}
}

func (r *githubOrganizationRulesetResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubOrganizationRulesetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubOrganizationRulesetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Framework model to GitHub API object
	ruleset, diags := r.frameworkToAPIRuleset(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[DEBUG] Creating organization ruleset: %s", plan.Name.ValueString())

	createdRuleset, _, err := r.client.V3Client().Organizations.CreateRepositoryRuleset(ctx, r.client.Name(), *ruleset)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating organization ruleset",
			"Could not create organization ruleset, unexpected error: "+err.Error(),
		)
		return
	}

	plan.RulesetID = types.Int64Value(*createdRuleset.ID)
	plan.ID = types.StringValue(strconv.FormatInt(*createdRuleset.ID, 10))
	plan.NodeID = types.StringValue(createdRuleset.GetNodeID())

	// Read the ruleset back to get all computed values
	r.readRuleset(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubOrganizationRulesetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubOrganizationRulesetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readRuleset(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubOrganizationRulesetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubOrganizationRulesetResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Framework model to GitHub API object
	ruleset, diags := r.frameworkToAPIRuleset(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rulesetID := plan.RulesetID.ValueInt64()

	log.Printf("[DEBUG] Updating organization ruleset: %d", rulesetID)

	updatedRuleset, _, err := r.client.V3Client().Organizations.UpdateRepositoryRuleset(ctx, r.client.Name(), rulesetID, *ruleset)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating organization ruleset",
			"Could not update organization ruleset, unexpected error: "+err.Error(),
		)
		return
	}

	plan.RulesetID = types.Int64Value(*updatedRuleset.ID)
	plan.ID = types.StringValue(strconv.FormatInt(*updatedRuleset.ID, 10))
	plan.NodeID = types.StringValue(updatedRuleset.GetNodeID())

	// Read the ruleset back to get all computed values
	r.readRuleset(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubOrganizationRulesetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubOrganizationRulesetResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rulesetID := state.RulesetID.ValueInt64()

	log.Printf("[DEBUG] Deleting organization ruleset: %d", rulesetID)
	_, err := r.client.V3Client().Organizations.DeleteRepositoryRuleset(ctx, r.client.Name(), rulesetID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting organization ruleset",
			"Could not delete organization ruleset, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *githubOrganizationRulesetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rulesetID, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing organization ruleset",
			"Could not parse ruleset ID: "+err.Error(),
		)
		return
	}

	if rulesetID == 0 {
		resp.Diagnostics.AddError(
			"Error importing organization ruleset",
			"`ruleset_id` must be present",
		)
		return
	}

	log.Printf("[DEBUG] Importing organization ruleset with ID: %d", rulesetID)

	var state githubOrganizationRulesetResourceModel
	state.RulesetID = types.Int64Value(rulesetID)
	state.ID = types.StringValue(strconv.FormatInt(rulesetID, 10))

	r.readRuleset(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the ID attribute for ImportState
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(strconv.FormatInt(rulesetID, 10)))...)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// readRuleset fetches the ruleset from GitHub API and updates the state
func (r *githubOrganizationRulesetResource) readRuleset(ctx context.Context, state *githubOrganizationRulesetResourceModel, diags *diag.Diagnostics) {
	rulesetID := state.RulesetID.ValueInt64()

	apiCtx := ctx
	if !state.ETag.IsNull() && !state.ETag.IsUnknown() {
		apiCtx = context.WithValue(ctx, rulesetEtagContextKey, state.ETag.ValueString())
	}

	ruleset, response, err := r.client.V3Client().Organizations.GetRepositoryRuleset(apiCtx, r.client.Name(), rulesetID)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Organization ruleset %d not found, removing from state", rulesetID)
				*state = githubOrganizationRulesetResourceModel{} // Reset state to trigger removal
				return
			}
		}
		diags.AddError(
			"Error reading organization ruleset",
			"Could not read organization ruleset: "+err.Error(),
		)
		return
	}

	// Update ETag
	if response != nil && response.Header.Get("ETag") != "" {
		state.ETag = types.StringValue(response.Header.Get("ETag"))
	}

	// Set ID to the ruleset ID
	state.ID = types.StringValue(strconv.FormatInt(rulesetID, 10))

	// Convert API response to Framework model
	r.apiToFrameworkRuleset(ctx, ruleset, state, diags)
}
