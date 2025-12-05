package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	githubprovider "github.com/nexthink-oss/terraform-provider-github/v7/github"
	"golang.org/x/oauth2"
)

// Owner is imported from the github package to access the configured client.
// This allows us to use the rate-limited client from the SDKv2 provider.
type Owner = githubprovider.Owner

// ctxEtag is a context key for ETag header handling
type ctxKey string

const ctxEtag ctxKey = "etag"

var (
	_ resource.Resource                 = &Resource{}
	_ resource.ResourceWithImportState  = &Resource{}
	_ resource.ResourceWithConfigure    = &Resource{}
	_ resource.ResourceWithUpgradeState = &Resource{}
)

// Resource implements the github_repository resource.
type Resource struct {
	client *github.Client
	owner  string
	// isOrganization tracks whether the owner is an organization (vs user account)
	isOrganization bool
}

// NewResource returns a new repository resource.
func NewResource() resource.Resource {
	return &Resource{}
}

func (r *Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (r *Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// Version must match SDKv2 SchemaVersion for state compatibility
		Version:     1,
		Description: "Creates and manages repositories within GitHub organizations or personal accounts",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the repository.",
				PlanModifiers: []planmodifier.String{
					NameBasedUnknownModifier(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the repository.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[-a-zA-Z0-9_.]{1,100}$`),
						"must include only alphanumeric characters, underscores or hyphens and consist of 100 characters or less",
					),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A description of the repository.",
			},
			"homepage_url": schema.StringAttribute{
				Optional:    true,
				Description: "URL of a page describing the project.",
			},
			"private": schema.BoolAttribute{
				Optional:           true,
				Computed:           true,
				Description:        "Set to true to create a private repository. Repositories are created as public (e.g. open source) by default.",
				DeprecationMessage: "use visibility instead",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"visibility": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Can be 'public' or 'private'. If your organization is associated with an enterprise account using GitHub Enterprise Cloud or GitHub Enterprise Server 2.20+, visibility can also be 'internal'.",
				Validators: []validator.String{
					stringvalidator.OneOf("public", "private", "internal"),
				},
			},
			"has_issues": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to 'true' to enable the GitHub Issues features on the repository",
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"has_discussions": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to 'true' to enable GitHub Discussions on the repository. Defaults to 'false'.",
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"has_projects": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to 'true' to enable the GitHub Projects features on the repository. Per the GitHub documentation when in an organization that has disabled repository projects it will default to 'false' and will otherwise default to 'true'. If you specify 'true' when it has been disabled it will return an error.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"has_downloads": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to 'true' to enable the (deprecated) downloads features on the repository.",
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"has_wiki": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to 'true' to enable the GitHub Wiki features on the repository.",
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_template": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to 'true' to tell GitHub that this is a template repository.",
				Default:     booldefault.StaticBool(false),
			},
			"allow_merge_commit": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to 'false' to disable merge commits on the repository.",
				Default:     booldefault.StaticBool(true),
			},
			"allow_squash_merge": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to 'false' to disable squash merges on the repository.",
				Default:     booldefault.StaticBool(true),
			},
			"allow_rebase_merge": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to 'false' to disable rebase merges on the repository.",
				Default:     booldefault.StaticBool(true),
			},
			"allow_auto_merge": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to 'true' to allow auto-merging pull requests on the repository.",
				Default:     booldefault.StaticBool(false),
			},
			"squash_merge_commit_title": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Can be 'PR_TITLE' or 'COMMIT_OR_PR_TITLE' for a default squash merge commit title.",
				Default:     stringdefault.StaticString("COMMIT_OR_PR_TITLE"),
			},
			"squash_merge_commit_message": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Can be 'PR_BODY', 'COMMIT_MESSAGES', or 'BLANK' for a default squash merge commit message.",
				Default:     stringdefault.StaticString("COMMIT_MESSAGES"),
			},
			"merge_commit_title": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Can be 'PR_TITLE' or 'MERGE_MESSAGE' for a default merge commit title.",
				Default:     stringdefault.StaticString("MERGE_MESSAGE"),
			},
			"merge_commit_message": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Can be 'PR_BODY', 'PR_TITLE', or 'BLANK' for a default merge commit message.",
				Default:     stringdefault.StaticString("PR_TITLE"),
			},
			"delete_branch_on_merge": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Automatically delete head branch after a pull request is merged. Defaults to 'false'.",
				Default:     booldefault.StaticBool(false),
			},
			"web_commit_signoff_required": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Require contributors to sign off on web-based commits. Defaults to 'false'.",
				Default:     booldefault.StaticBool(false),
			},
			"auto_init": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to 'true' to produce an initial commit in the repository.",
				Default:     booldefault.StaticBool(false),
			},
			"default_branch": schema.StringAttribute{
				Optional:           true,
				Computed:           true,
				Description:        "Can only be set after initial repository creation, and only if the target branch exists",
				DeprecationMessage: "Use the github_branch_default resource instead",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"license_template": schema.StringAttribute{
				Optional:    true,
				Description: "Use the name of the template without the extension. For example, 'mit' or 'mpl-2.0'.",
			},
			"gitignore_template": schema.StringAttribute{
				Optional:    true,
				Description: "Use the name of the template without the extension. For example, 'Haskell'.",
			},
			"archived": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Specifies if the repository should be archived. Defaults to 'false'. NOTE Currently, the API does not support unarchiving.",
				Default:     booldefault.StaticBool(false),
			},
			"archive_on_destroy": schema.BoolAttribute{
				Optional:    true,
				Description: "Set to 'true' to archive the repository instead of deleting on destroy.",
			},
			"vulnerability_alerts": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Set to 'true' to enable security alerts for vulnerable dependencies. Enabling requires alerts to be enabled on the owner level. (Note for importing: GitHub enables the alerts on public repos but disables them on private repos by default). Note that vulnerability alerts have not been successfully tested on any GitHub Enterprise instance and may be unavailable in those settings.",
			},
			"ignore_vulnerability_alerts_during_read": schema.BoolAttribute{
				Optional:    true,
				Description: "Set to true to not call the vulnerability alerts endpoint so the resource can also be used without admin permissions during read.",
			},
			"allow_update_branch": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: " Set to 'true' to always suggest updating pull request branches.",
				Default:     booldefault.StaticBool(false),
			},
			"exclusive_custom_properties": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this resource exclusively manages all custom properties. Defaults to 'true'; if set to 'false', only properties defined in custom_property blocks will be managed by this resource, allowing collaborative management of custom property settings.",
				Default:     booldefault.StaticBool(true),
			},
			// Computed attributes
			"full_name": schema.StringAttribute{
				Computed:    true,
				Description: "A string of the form 'orgname/reponame'.",
				PlanModifiers: []planmodifier.String{
					FullNamePlanModifier(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"html_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL to the repository on the web.",
				PlanModifiers: []planmodifier.String{
					NameBasedUnknownModifier(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ssh_clone_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL that can be provided to 'git clone' to clone the repository via SSH.",
				PlanModifiers: []planmodifier.String{
					NameBasedUnknownModifier(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"svn_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL that can be provided to 'svn checkout' to check out the repository via GitHub's Subversion protocol emulation.",
				PlanModifiers: []planmodifier.String{
					NameBasedUnknownModifier(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"git_clone_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL that can be provided to 'git clone' to clone the repository anonymously via the git protocol.",
				PlanModifiers: []planmodifier.String{
					NameBasedUnknownModifier(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"http_clone_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL that can be provided to 'git clone' to clone the repository via HTTPS.",
				PlanModifiers: []planmodifier.String{
					NameBasedUnknownModifier(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"etag": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"primary_language": schema.StringAttribute{
				Computed: true,
				// Dynamic field - changes based on repository content
			},
			"node_id": schema.StringAttribute{
				Computed:    true,
				Description: "GraphQL global node id for use with v4 API.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repo_id": schema.Int64Attribute{
				Computed:    true,
				Description: "GitHub ID for the repository.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"all_custom_properties": schema.MapAttribute{
				Computed:    true,
				Description: "All custom properties set on the repository, including those not managed by this resource.",
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"topics": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The list of topics of the repository.",
				ElementType: types.StringType,
			},
		},

		Blocks: map[string]schema.Block{
			"security_and_analysis": schema.ListNestedBlock{
				Description: "Security and analysis settings for the repository. To use this parameter you must have admin permissions for the repository or be an owner or security manager for the organization that owns the repository. Only one block is permitted.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"advanced_security": schema.ListNestedBlock{
							Description: "The advanced security configuration for the repository. If a repository's visibility is 'public', advanced security is always enabled and cannot be changed, so this setting cannot be supplied. Only one block is permitted.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"status": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "Set to 'enabled' to enable advanced security features on the repository. Can be 'enabled' or 'disabled'.",
										Validators: []validator.String{
											stringvalidator.OneOf("enabled", "disabled"),
										},
									},
								},
							},
						},
						"secret_scanning": schema.ListNestedBlock{
							Description: "The secret scanning configuration for the repository. Only one block is permitted.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"status": schema.StringAttribute{
										Computed:    true,
										Optional:    true,
										Description: "Set to 'enabled' to enable secret scanning on the repository. Can be 'enabled' or 'disabled'. If set to 'enabled', the repository's visibility must be 'public' or 'security_and_analysis[0].advanced_security[0].status' must also be set to 'enabled'.",
										Validators: []validator.String{
											stringvalidator.OneOf("enabled", "disabled"),
										},
									},
								},
							},
						},
						"secret_scanning_push_protection": schema.ListNestedBlock{
							Description: "The secret scanning push protection configuration for the repository. Only one block is permitted.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"status": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "Set to 'enabled' to enable secret scanning push protection on the repository. Can be 'enabled' or 'disabled'. If set to 'enabled', the repository's visibility must be 'public' or 'security_and_analysis[0].advanced_security[0].status' must also be set to 'enabled'.",
										Validators: []validator.String{
											stringvalidator.OneOf("enabled", "disabled"),
										},
									},
								},
							},
						},
					},
				},
			},
			"pages": schema.ListNestedBlock{
				Description: "The repository's GitHub Pages configuration. Only one block is permitted.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"source": schema.ListNestedBlock{
							Description: "The source branch and directory for the rendered Pages site. Only one block is permitted.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"branch": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "The repository branch used to publish the site's source files. (i.e. 'main' or 'gh-pages')",
									},
									"path": schema.StringAttribute{
										Optional:    true,
										Computed:    true,
										Description: "The repository directory from which the site publishes (Default: '/')",
										Default:     stringdefault.StaticString("/"),
									},
								},
							},
						},
					},
					Attributes: map[string]schema.Attribute{
						"build_type": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The type the page should be sourced.",
							Default:     stringdefault.StaticString("legacy"),
							Validators: []validator.String{
								stringvalidator.OneOf("legacy", "workflow"),
							},
						},
						"cname": schema.StringAttribute{
							Optional:    true,
							Description: "The custom domain for the repository. This can only be set after the repository has been created.",
						},
						"custom_404": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the rendered GitHub Pages site has a custom 404 page",
						},
						"html_url": schema.StringAttribute{
							Computed:    true,
							Description: "URL to the repository on the web.",
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "The GitHub Pages site's build status e.g. building or built.",
						},
						"url": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
			"template": schema.ListNestedBlock{
				Description: "Use a template repository to create this resource. Only one block is permitted.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"include_all_branches": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Whether the new repository should include all the branches from the template repository (defaults to 'false', which includes only the default branch from the template).",
							Default:     booldefault.StaticBool(false),
						},
						"owner": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The GitHub organization or user the template repository is owned by.",
						},
						"repository": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "The name of the template repository.",
						},
					},
				},
			},
			"custom_property": schema.SetNestedBlock{
				Description: "Custom properties for the repository.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "The name of the custom property.",
						},
						"value": schema.ListAttribute{
							Required:    true,
							Description: "The value(s) of the custom property. For single-value properties, provide a list with one element. For multi-select properties, provide multiple elements.",
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

// UpgradeState handles state migration from prior schema versions.
// This matches the SDKv2 resourceGithubRepositoryMigrateState function.
func (r *Resource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// State upgrade from version 0 to version 1
		// Mirrors SDKv2 migrateGithubRepositoryStateV0toV1: removes deprecated "branches.*" attributes
		0: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				// Parse raw JSON state
				var rawState map[string]json.RawMessage
				if err := json.Unmarshal(req.RawState.JSON, &rawState); err != nil {
					resp.Diagnostics.AddError(
						"Unable to Unmarshal Prior State",
						fmt.Sprintf("Error parsing v0 state: %s", err.Error()),
					)
					return
				}

				// Remove deprecated "branches.*" attributes (exact SDKv2 behavior)
				for key := range rawState {
					if strings.HasPrefix(key, "branches.") {
						delete(rawState, key)
					}
				}

				// Re-marshal the cleaned state
				upgradedStateJSON, err := json.Marshal(rawState)
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to Marshal Upgraded State",
						fmt.Sprintf("Error serializing v1 state: %s", err.Error()),
					)
					return
				}

				// Set the upgraded state
				resp.DynamicValue = &tfprotov6.DynamicValue{
					JSON: upgradedStateJSON,
				}
			},
		},
	}
}

// Configure sets up the GitHub client for this resource
func (r *Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// In a muxed setup, ProviderData comes from the SDKv2 provider's meta (*Owner)
	// which contains the rate-limited client configured with all provider settings
	if req.ProviderData != nil {
		owner, ok := req.ProviderData.(*Owner)
		if ok {
			// Use the rate-limited client from the SDKv2 provider
			// This client is configured with rate limiting, retries, delays, etc.
			r.client = owner.V3Client()
			r.owner = owner.Name()
			r.isOrganization = owner.IsOrganization
			return
		}
	}

	// Fallback: Create our own client from environment variables
	// This is used when ProviderData is nil or not the expected type
	// (e.g., during testing or if muxer doesn't pass ProviderData)
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		resp.Diagnostics.AddError(
			"Missing GitHub Token",
			"GITHUB_TOKEN environment variable must be set",
		)
		return
	}

	// Get owner from environment (GITHUB_OWNER or GITHUB_ORGANIZATION)
	ownerName := os.Getenv("GITHUB_OWNER")
	if ownerName == "" {
		ownerName = os.Getenv("GITHUB_ORGANIZATION")
	}

	// Create OAuth client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(ctx, ts)

	// Create GitHub client
	baseURL := os.Getenv("GITHUB_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.github.com/"
	}

	client, err := github.NewClient(httpClient).WithEnterpriseURLs(baseURL, "")
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create GitHub Client",
			fmt.Sprintf("Error creating GitHub client: %s", err.Error()),
		)
		return
	}

	r.client = client
	r.owner = ownerName

	// If owner is not set, try to get authenticated user
	if r.owner == "" {
		user, _, err := client.Users.Get(ctx, "")
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Determine Owner",
				fmt.Sprintf("GITHUB_OWNER not set and unable to fetch authenticated user: %s", err.Error()),
			)
			return
		}
		r.owner = user.GetLogin()
	}
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RepositoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate default_branch
	if !plan.DefaultBranch.IsNull() && !plan.DefaultBranch.IsUnknown() && plan.DefaultBranch.ValueString() != "main" {
		resp.Diagnostics.AddError(
			"Invalid Default Branch",
			"Cannot set the default branch on a new repository to something other than 'main'",
		)
		return
	}

	// Build repository request
	repoReq, diagsConvert := r.planToGithubRepository(ctx, &plan, true)
	resp.Diagnostics.Append(diagsConvert...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoName := plan.Name.ValueString()

	var repo *github.Repository
	var err error

	// Check if using template (template is now a list with max 1 element)
	if !plan.Template.IsNull() && !plan.Template.IsUnknown() && len(plan.Template.Elements()) > 0 {
		var templateModels []TemplateModel
		resp.Diagnostics.Append(plan.Template.ElementsAs(ctx, &templateModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(templateModels) > 0 {
			templateModel := templateModels[0]

			templateRepoReq := &github.TemplateRepoRequest{
				Name:               github.Ptr(repoName),
				Owner:              github.Ptr(r.owner),
				Description:        repoReq.Description,
				Private:            repoReq.Private,
				IncludeAllBranches: github.Ptr(templateModel.IncludeAllBranches.ValueBool()),
			}

			repo, _, err = r.client.Repositories.CreateFromTemplate(ctx,
				templateModel.Owner.ValueString(),
				templateModel.Repository.ValueString(),
				templateRepoReq,
			)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Creating Repository from Template",
					fmt.Sprintf("Could not create repository %s from template: %s", repoName, err.Error()),
				)
				return
			}
		}
	}

	// Create regular repository if not using template
	if repo == nil {
		// Determine if this is an organization or user repository
		// Try to check if owner is an organization
		org, _, err := r.client.Organizations.Get(ctx, r.owner)
		if err == nil && org != nil {
			// It's an organization
			repo, _, err = r.client.Repositories.Create(ctx, r.owner, repoReq)
		} else {
			// It's a user account
			repo, _, err = r.client.Repositories.Create(ctx, "", repoReq)
		}

		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating Repository",
				fmt.Sprintf("Could not create repository %s: %s", repoName, err.Error()),
			)
			return
		}
	}

	// Set ID
	plan.ID = types.StringValue(repo.GetName())

	// If auto_init was true, wait for repository to be initialized
	// GitHub initializes repos asynchronously, so we need to wait
	if !plan.AutoInit.IsNull() && plan.AutoInit.ValueBool() {
		// Poll until the repository has a default branch (indicating initialization is complete)
		maxRetries := 10
		for i := 0; i < maxRetries; i++ {
			time.Sleep(time.Second)
			checkRepo, _, err := r.client.Repositories.Get(ctx, r.owner, repoName)
			if err == nil && checkRepo.GetDefaultBranch() != "" {
				repo = checkRepo
				break
			}
		}
	}

	// Check if we need to update repository settings that aren't supported during create
	needsUpdate := false
	updateReq := &github.Repository{}

	// web_commit_signoff_required might not be respected during create
	if !plan.WebCommitSignoffRequired.IsNull() && !plan.WebCommitSignoffRequired.IsUnknown() {
		if plan.WebCommitSignoffRequired.ValueBool() != repo.GetWebCommitSignoffRequired() {
			needsUpdate = true
			updateReq.WebCommitSignoffRequired = github.Ptr(plan.WebCommitSignoffRequired.ValueBool())
		}
	}

	if needsUpdate {
		_, _, err = r.client.Repositories.Edit(ctx, r.owner, repoName, updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Repository Settings",
				fmt.Sprintf("Could not update settings for repository %s: %s", repoName, err.Error()),
			)
			return
		}
	}

	// Set topics if provided
	if !plan.Topics.IsNull() && !plan.Topics.IsUnknown() {
		var topics []string
		resp.Diagnostics.Append(plan.Topics.ElementsAs(ctx, &topics, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(topics) > 0 {
			_, _, err := r.client.Repositories.ReplaceAllTopics(ctx, r.owner, repoName, topics)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Setting Repository Topics",
					fmt.Sprintf("Could not set topics for repository %s: %s", repoName, err.Error()),
				)
				return
			}
		}
	}

	// Enable GitHub Pages if configured (pages is now a list with max 1 element)
	if !plan.Pages.IsNull() && !plan.Pages.IsUnknown() && len(plan.Pages.Elements()) > 0 {
		var pagesModels []PagesModel
		resp.Diagnostics.Append(plan.Pages.ElementsAs(ctx, &pagesModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(pagesModels) > 0 {
			pages, diagsPages := r.expandPages(ctx, &pagesModels[0])
			resp.Diagnostics.Append(diagsPages...)
			if resp.Diagnostics.HasError() {
				return
			}

			if pages != nil {
				_, _, err := r.client.Repositories.EnablePages(ctx, r.owner, repoName, pages)
				if err != nil {
					resp.Diagnostics.AddError(
						"Error Enabling GitHub Pages",
						fmt.Sprintf("Could not enable GitHub Pages for repository %s: %s", repoName, err.Error()),
					)
					return
				}
			}
		}
	}

	// Set custom properties if provided
	if !plan.CustomProperty.IsNull() && !plan.CustomProperty.IsUnknown() {
		var customProps []CustomPropertyModel
		resp.Diagnostics.Append(plan.CustomProperty.ElementsAs(ctx, &customProps, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(customProps) > 0 {
			exclusiveMode := plan.ExclusiveCustomProperties.ValueBool()
			diagsProps := r.setRepositoryCustomProperties(ctx, repoName, plan.CustomProperty, exclusiveMode, nil)
			resp.Diagnostics.Append(diagsProps...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// Fetch the repository again to get the ETag
	finalRepo, httpResp, err := r.client.Repositories.Get(ctx, r.owner, repoName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Repository After Create",
			fmt.Sprintf("Could not read repository %s after creation: %s", repoName, err.Error()),
		)
		return
	}

	// Populate computed fields from the final repository state
	plan.Etag = types.StringValue(httpResp.Header.Get("ETag"))
	plan.FullName = types.StringValue(finalRepo.GetFullName())
	plan.NodeID = types.StringValue(finalRepo.GetNodeID())
	plan.RepoID = types.Int64Value(finalRepo.GetID())
	plan.HTMLURL = types.StringValue(finalRepo.GetHTMLURL())
	plan.SSHCloneURL = types.StringValue(finalRepo.GetSSHURL())
	plan.SVNURL = types.StringValue(finalRepo.GetSVNURL())
	plan.GitCloneURL = types.StringValue(finalRepo.GetGitURL())
	plan.HTTPCloneURL = types.StringValue(finalRepo.GetCloneURL())
	plan.PrimaryLanguage = types.StringPointerValue(finalRepo.Language)
	plan.DefaultBranch = types.StringValue(finalRepo.GetDefaultBranch())

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RepositoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoName := state.ID.ValueString()

	// Handle etag for conditional requests
	ctxWithEtag := ctx
	if !state.Etag.IsNull() && !state.Etag.IsUnknown() {
		ctxWithEtag = context.WithValue(ctx, ctxEtag, state.Etag.ValueString())
	}

	repo, _, err := r.client.Repositories.Get(ctxWithEtag, r.owner, repoName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				// Not modified, return existing state
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing repository %s/%s from state because it no longer exists in GitHub",
					r.owner, repoName)
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Error Reading Repository",
			fmt.Sprintf("Could not read repository %s: %s", repoName, err.Error()),
		)
		return
	}

	// Update state from API response
	state.Name = types.StringValue(repo.GetName())
	state.FullName = types.StringValue(repo.GetFullName())
	state.Description = types.StringPointerValue(repo.Description)
	state.HomepageURL = types.StringPointerValue(repo.Homepage)
	state.Private = types.BoolValue(repo.GetPrivate())
	state.Visibility = types.StringValue(repo.GetVisibility())
	state.HasIssues = types.BoolValue(repo.GetHasIssues())
	state.HasDiscussions = types.BoolValue(repo.GetHasDiscussions())
	state.HasProjects = types.BoolValue(repo.GetHasProjects())
	state.HasWiki = types.BoolValue(repo.GetHasWiki())
	state.IsTemplate = types.BoolValue(repo.GetIsTemplate())
	state.DefaultBranch = types.StringValue(repo.GetDefaultBranch())
	state.Archived = types.BoolValue(repo.GetArchived())
	state.NodeID = types.StringValue(repo.GetNodeID())
	state.RepoID = types.Int64Value(repo.GetID())

	// URLs
	state.HTMLURL = types.StringValue(repo.GetHTMLURL())
	state.SSHCloneURL = types.StringValue(repo.GetSSHURL())
	state.SVNURL = types.StringValue(repo.GetSVNURL())
	state.GitCloneURL = types.StringValue(repo.GetGitURL())
	state.HTTPCloneURL = types.StringValue(repo.GetCloneURL())

	// Primary language (dynamic field, no UseStateForUnknown)
	state.PrimaryLanguage = types.StringPointerValue(repo.Language)

	// Topics
	if len(repo.Topics) > 0 {
		topicElements := make([]attr.Value, len(repo.Topics))
		for i, topic := range repo.Topics {
			topicElements[i] = types.StringValue(topic)
		}
		topicsSet, diags := types.SetValue(types.StringType, topicElements)
		resp.Diagnostics.Append(diags...)
		state.Topics = topicsSet
	} else {
		state.Topics = types.SetNull(types.StringType)
	}

	// GitHub API doesn't respond with these parameters when repository is archived
	if !repo.GetArchived() {
		state.AllowAutoMerge = types.BoolValue(repo.GetAllowAutoMerge())
		state.AllowMergeCommit = types.BoolValue(repo.GetAllowMergeCommit())
		state.AllowRebaseMerge = types.BoolValue(repo.GetAllowRebaseMerge())
		state.AllowSquashMerge = types.BoolValue(repo.GetAllowSquashMerge())
		state.AllowUpdateBranch = types.BoolPointerValue(repo.AllowUpdateBranch)
		state.DeleteBranchOnMerge = types.BoolValue(repo.GetDeleteBranchOnMerge())
		state.WebCommitSignoffRequired = types.BoolValue(repo.GetWebCommitSignoffRequired())
		state.HasDownloads = types.BoolValue(repo.GetHasDownloads())
		state.MergeCommitMessage = types.StringPointerValue(repo.MergeCommitMessage)
		state.MergeCommitTitle = types.StringPointerValue(repo.MergeCommitTitle)
		state.SquashMergeCommitMessage = types.StringPointerValue(repo.SquashMergeCommitMessage)
		state.SquashMergeCommitTitle = types.StringPointerValue(repo.SquashMergeCommitTitle)
	}

	// Pages - only populate if already in state
	if !state.Pages.IsNull() && repo.GetHasPages() {
		pages, _, err := r.client.Repositories.GetPagesInfo(ctx, r.owner, repoName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading GitHub Pages",
				fmt.Sprintf("Could not read GitHub Pages info for repository %s: %s", repoName, err.Error()),
			)
			return
		}

		pagesObj, diagsPages := r.flattenPages(ctx, pages)
		resp.Diagnostics.Append(diagsPages...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Pages = pagesObj
	}

	// Template (now a list with max 1 element for state compatibility) - only populate if already in state
	if !state.Template.IsNull() && repo.TemplateRepository != nil {
		templateAttrTypes := map[string]attr.Type{
			"owner":                types.StringType,
			"repository":           types.StringType,
			"include_all_branches": types.BoolType,
		}
		templateObj, diagsTemplate := types.ObjectValue(
			templateAttrTypes,
			map[string]attr.Value{
				"owner":                types.StringPointerValue(repo.TemplateRepository.Owner.Login),
				"repository":           types.StringPointerValue(repo.TemplateRepository.Name),
				"include_all_branches": types.BoolValue(false), // API doesn't return this
			},
		)
		resp.Diagnostics.Append(diagsTemplate...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Wrap in a list with single element
		templateList, listDiags := types.ListValue(types.ObjectType{AttrTypes: templateAttrTypes}, []attr.Value{templateObj})
		resp.Diagnostics.Append(listDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Template = templateList
	}

	// Vulnerability alerts
	if !state.IgnoreVulnerabilityAlertsDuringRead.ValueBool() {
		vulnerabilityAlerts, _, err := r.client.Repositories.GetVulnerabilityAlerts(ctx, r.owner, repoName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Vulnerability Alerts",
				fmt.Sprintf("Could not read vulnerability alerts for repository %s: %s", repoName, err.Error()),
			)
			return
		}
		state.VulnerabilityAlerts = types.BoolValue(vulnerabilityAlerts)
	}

	// Security and analysis - only populate if already in state
	if !state.SecurityAndAnalysis.IsNull() && repo.SecurityAndAnalysis != nil {
		securityObj, diagsSecurity := r.flattenSecurityAndAnalysis(ctx, repo.SecurityAndAnalysis)
		resp.Diagnostics.Append(diagsSecurity...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.SecurityAndAnalysis = securityObj
	}

	// Custom properties - only fetch if configured
	if !state.CustomProperty.IsNull() && !state.CustomProperty.IsUnknown() {
		allCustomPropsMap, diagsAllProps := r.getAllCustomPropertiesAsMap(ctx, repoName)
		resp.Diagnostics.Append(diagsAllProps...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Convert map to types.Map for all_custom_properties
		allCustomPropsMapValue, diagsMap := types.MapValue(types.StringType, allCustomPropsMap)
		resp.Diagnostics.Append(diagsMap...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.AllCustomProperties = allCustomPropsMapValue

		// Filter to managed custom properties
		filteredProps, diagsFiltered := r.filterCustomPropertiesFromSet(ctx, allCustomPropsMap, state.CustomProperty)
		resp.Diagnostics.Append(diagsFiltered...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.CustomProperty = filteredProps
	}

	// Set default values for attributes that have defaults but might be null during import
	if state.AutoInit.IsNull() {
		state.AutoInit = types.BoolValue(false)
	}
	if state.ExclusiveCustomProperties.IsNull() {
		state.ExclusiveCustomProperties = types.BoolValue(true)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state RepositoryResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if repository is archived and not being unarchived (which is not supported)
	if state.Archived.ValueBool() && plan.Archived.ValueBool() {
		log.Printf("[INFO] Skipping update of archived repository")
		// Just run Read to refresh state
		var readReq resource.ReadRequest
		var readResp resource.ReadResponse
		readReq.State = req.State
		readResp.State = resp.State
		readResp.Diagnostics = resp.Diagnostics

		r.Read(ctx, readReq, &readResp)

		resp.Diagnostics = readResp.Diagnostics
		resp.State = readResp.State
		return
	}

	repoName := state.ID.ValueString()

	// Build repository update request
	repoReq, diagsConvert := r.planToGithubRepository(ctx, &plan, false)
	resp.Diagnostics.Append(diagsConvert...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Don't update visibility in main request (handled separately)
	repoReq.Visibility = nil

	// Handle default branch specially - only set if it changed
	// (GitHub rejects updates to default_branch on empty repositories)
	// Don't set it at all if both values are effectively the same
	planBranch := plan.DefaultBranch.ValueString()
	stateBranch := state.DefaultBranch.ValueString()
	if planBranch != stateBranch && planBranch != "" {
		repoReq.DefaultBranch = github.Ptr(planBranch)
	}

	// Update repository
	repo, _, err := r.client.Repositories.Edit(ctx, r.owner, repoName, repoReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Repository",
			fmt.Sprintf("Could not update repository %s: %s", repoName, err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(repo.GetName())

	// Handle pages updates (pages is now a list with max 1 element)
	if !plan.Pages.Equal(state.Pages) {
		if !plan.Pages.IsNull() && !plan.Pages.IsUnknown() && len(plan.Pages.Elements()) > 0 {
			var pagesModels []PagesModel
			resp.Diagnostics.Append(plan.Pages.ElementsAs(ctx, &pagesModels, false)...)
			if resp.Diagnostics.HasError() {
				return
			}

			if len(pagesModels) > 0 {
				pagesModel := pagesModels[0]

				// Check if pages currently exists
				existingPages, httpResp, err := r.client.Repositories.GetPagesInfo(ctx, r.owner, repoName)

				if httpResp != nil && httpResp.StatusCode == http.StatusNotFound {
					// Pages doesn't exist, enable it
					pages, diagsPages := r.expandPages(ctx, &pagesModel)
					resp.Diagnostics.Append(diagsPages...)
					if resp.Diagnostics.HasError() {
						return
					}

					_, _, err = r.client.Repositories.EnablePages(ctx, r.owner, repoName, pages)
					if err != nil {
						resp.Diagnostics.AddError(
							"Error Enabling GitHub Pages",
							fmt.Sprintf("Could not enable GitHub Pages for repository %s: %s", repoName, err.Error()),
						)
						return
					}
				} else if err == nil && existingPages != nil {
					// Pages exists, update it
					pagesUpdate, diagsPages := r.expandPagesUpdate(ctx, &pagesModel)
					resp.Diagnostics.Append(diagsPages...)
					if resp.Diagnostics.HasError() {
						return
					}

					_, err = r.client.Repositories.UpdatePages(ctx, r.owner, repoName, pagesUpdate)
					if err != nil {
						resp.Diagnostics.AddError(
							"Error Updating GitHub Pages",
							fmt.Sprintf("Could not update GitHub Pages for repository %s: %s", repoName, err.Error()),
						)
						return
					}
				} else if err != nil {
					resp.Diagnostics.AddError(
						"Error Reading GitHub Pages",
						fmt.Sprintf("Could not read GitHub Pages info for repository %s: %s", repoName, err.Error()),
					)
					return
				}
			}
		} else if !state.Pages.IsNull() && len(state.Pages.Elements()) > 0 {
			// Pages was removed, disable it
			_, err := r.client.Repositories.DisablePages(ctx, r.owner, repoName)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Disabling GitHub Pages",
					fmt.Sprintf("Could not disable GitHub Pages for repository %s: %s", repoName, err.Error()),
				)
				return
			}
		}
	}

	// Handle topics updates
	if !plan.Topics.Equal(state.Topics) {
		var topics []string
		if !plan.Topics.IsNull() && !plan.Topics.IsUnknown() {
			resp.Diagnostics.Append(plan.Topics.ElementsAs(ctx, &topics, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		_, _, err = r.client.Repositories.ReplaceAllTopics(ctx, r.owner, repoName, topics)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Repository Topics",
				fmt.Sprintf("Could not update topics for repository %s: %s", repoName, err.Error()),
			)
			return
		}
	}

	// Handle vulnerability alerts updates
	if !plan.VulnerabilityAlerts.Equal(state.VulnerabilityAlerts) {
		if plan.VulnerabilityAlerts.ValueBool() {
			_, err = r.client.Repositories.EnableVulnerabilityAlerts(ctx, r.owner, repoName)
		} else {
			_, err = r.client.Repositories.DisableVulnerabilityAlerts(ctx, r.owner, repoName)
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating Vulnerability Alerts",
				fmt.Sprintf("Could not update vulnerability alerts for repository %s: %s", repoName, err.Error()),
			)
			return
		}
	}

	// Handle visibility updates separately
	if !plan.Visibility.Equal(state.Visibility) && !plan.Visibility.IsNull() && !plan.Visibility.IsUnknown() {
		visibilityReq := &github.Repository{
			Visibility: github.Ptr(plan.Visibility.ValueString()),
		}
		_, httpResp, err := r.client.Repositories.Edit(ctx, r.owner, repoName, visibilityReq)
		if err != nil {
			// Ignore error if visibility is already set
			if httpResp == nil || httpResp.StatusCode != 422 {
				resp.Diagnostics.AddError(
					"Error Updating Repository Visibility",
					fmt.Sprintf("Could not update visibility for repository %s: %s", repoName, err.Error()),
				)
				return
			}
		}
	}

	// Handle custom properties updates
	if !plan.CustomProperty.Equal(state.CustomProperty) || !plan.ExclusiveCustomProperties.Equal(state.ExclusiveCustomProperties) {
		exclusiveMode := plan.ExclusiveCustomProperties.ValueBool()

		// Get current properties from state as map[string]attr.Value
		var currentPropsMap map[string]attr.Value
		if !state.AllCustomProperties.IsNull() && !state.AllCustomProperties.IsUnknown() {
			currentPropsMap = state.AllCustomProperties.Elements()
		}

		diagsProps := r.setRepositoryCustomProperties(ctx, repoName, plan.CustomProperty, exclusiveMode, currentPropsMap)
		resp.Diagnostics.Append(diagsProps...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Fetch the repository again to get fresh ETag and state
	finalRepo, finalHttpResp, err := r.client.Repositories.Get(ctx, r.owner, repoName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Repository After Update",
			fmt.Sprintf("Could not read repository %s after update: %s", repoName, err.Error()),
		)
		return
	}

	// Populate computed fields from the final repository state
	plan.Etag = types.StringValue(finalHttpResp.Header.Get("ETag"))
	plan.FullName = types.StringValue(finalRepo.GetFullName())
	plan.HTMLURL = types.StringValue(finalRepo.GetHTMLURL())
	plan.SSHCloneURL = types.StringValue(finalRepo.GetSSHURL())
	plan.SVNURL = types.StringValue(finalRepo.GetSVNURL())
	plan.GitCloneURL = types.StringValue(finalRepo.GetGitURL())
	plan.HTTPCloneURL = types.StringValue(finalRepo.GetCloneURL())
	plan.PrimaryLanguage = types.StringPointerValue(finalRepo.Language)
	plan.DefaultBranch = types.StringValue(finalRepo.GetDefaultBranch())

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RepositoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoName := state.ID.ValueString()

	// Check if we should archive instead of delete
	if state.ArchiveOnDestroy.ValueBool() {
		if state.Archived.ValueBool() {
			log.Printf("[DEBUG] Repository already archived, nothing to do on delete: %s/%s", r.owner, repoName)
			return
		}

		// Archive the repository
		state.Archived = types.BoolValue(true)
		repoReq, diagsConvert := r.planToGithubRepository(ctx, &state, false)
		resp.Diagnostics.Append(diagsConvert...)
		if resp.Diagnostics.HasError() {
			return
		}

		log.Printf("[DEBUG] Archiving repository on delete: %s/%s", r.owner, repoName)
		_, _, err := r.client.Repositories.Edit(ctx, r.owner, repoName, repoReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Archiving Repository",
				fmt.Sprintf("Could not archive repository %s: %s", repoName, err.Error()),
			)
			return
		}
		return
	}

	// Delete the repository
	log.Printf("[DEBUG] Deleting repository: %s/%s", r.owner, repoName)
	_, err := r.client.Repositories.Delete(ctx, r.owner, repoName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Repository",
			fmt.Sprintf("Could not delete repository %s: %s", repoName, err.Error()),
		)
		return
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	// SDKv2 behavior: Set auto_init to false on import
	resp.Diagnostics.AddWarning(
		"Import Behavior",
		"The auto_init attribute is set to false during import to match SDKv2 behavior.",
	)
}
