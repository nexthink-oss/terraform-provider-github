package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                 = &githubRepositoryResource{}
	_ resource.ResourceWithConfigure    = &githubRepositoryResource{}
	_ resource.ResourceWithImportState  = &githubRepositoryResource{}
	_ resource.ResourceWithUpgradeState = &githubRepositoryResource{}
)

type githubRepositoryResource struct {
	client *Owner
}

type githubRepositoryResourceModel struct {
	// Computed attributes
	ID types.String `tfsdk:"id"`

	// Required/Basic attributes
	Name types.String `tfsdk:"name"`

	// Optional configuration attributes
	Description    types.String `tfsdk:"description"`
	HomepageURL    types.String `tfsdk:"homepage_url"`
	Private        types.Bool   `tfsdk:"private"`
	Visibility     types.String `tfsdk:"visibility"`
	HasIssues      types.Bool   `tfsdk:"has_issues"`
	HasDiscussions types.Bool   `tfsdk:"has_discussions"`
	HasProjects    types.Bool   `tfsdk:"has_projects"`
	HasDownloads   types.Bool   `tfsdk:"has_downloads"`
	HasWiki        types.Bool   `tfsdk:"has_wiki"`
	IsTemplate     types.Bool   `tfsdk:"is_template"`

	// Merge strategy attributes
	AllowMergeCommit         types.Bool   `tfsdk:"allow_merge_commit"`
	AllowSquashMerge         types.Bool   `tfsdk:"allow_squash_merge"`
	AllowRebaseMerge         types.Bool   `tfsdk:"allow_rebase_merge"`
	AllowAutoMerge           types.Bool   `tfsdk:"allow_auto_merge"`
	SquashMergeCommitTitle   types.String `tfsdk:"squash_merge_commit_title"`
	SquashMergeCommitMessage types.String `tfsdk:"squash_merge_commit_message"`
	MergeCommitTitle         types.String `tfsdk:"merge_commit_title"`
	MergeCommitMessage       types.String `tfsdk:"merge_commit_message"`
	DeleteBranchOnMerge      types.Bool   `tfsdk:"delete_branch_on_merge"`
	WebCommitSignoffRequired types.Bool   `tfsdk:"web_commit_signoff_required"`

	// Repository initialization
	AutoInit          types.Bool   `tfsdk:"auto_init"`
	DefaultBranch     types.String `tfsdk:"default_branch"`
	LicenseTemplate   types.String `tfsdk:"license_template"`
	GitignoreTemplate types.String `tfsdk:"gitignore_template"`

	// Archive settings
	Archived         types.Bool `tfsdk:"archived"`
	ArchiveOnDestroy types.Bool `tfsdk:"archive_on_destroy"`

	// Security settings
	VulnerabilityAlerts                 types.Bool `tfsdk:"vulnerability_alerts"`
	IgnoreVulnerabilityAlertsDuringRead types.Bool `tfsdk:"ignore_vulnerability_alerts_during_read"`
	AllowUpdateBranch                   types.Bool `tfsdk:"allow_update_branch"`

	// Topics and template
	Topics types.Set `tfsdk:"topics"`

	// Complex nested attributes
	SecurityAndAnalysis types.List `tfsdk:"security_and_analysis"`
	Pages               types.List `tfsdk:"pages"`
	Template            types.List `tfsdk:"template"`

	// Computed attributes
	FullName        types.String `tfsdk:"full_name"`
	HTMLURL         types.String `tfsdk:"html_url"`
	SSHCloneURL     types.String `tfsdk:"ssh_clone_url"`
	SVNURL          types.String `tfsdk:"svn_url"`
	GitCloneURL     types.String `tfsdk:"git_clone_url"`
	HTTPCloneURL    types.String `tfsdk:"http_clone_url"`
	ETag            types.String `tfsdk:"etag"`
	PrimaryLanguage types.String `tfsdk:"primary_language"`
	NodeID          types.String `tfsdk:"node_id"`
	RepoID          types.Int64  `tfsdk:"repo_id"`
}

type securityAndAnalysisModel struct {
	AdvancedSecurity             types.List `tfsdk:"advanced_security"`
	SecretScanning               types.List `tfsdk:"secret_scanning"`
	SecretScanningPushProtection types.List `tfsdk:"secret_scanning_push_protection"`
}

type securityFeatureModel struct {
	Status types.String `tfsdk:"status"`
}

type pagesModel struct {
	Source    types.List   `tfsdk:"source"`
	BuildType types.String `tfsdk:"build_type"`
	CNAME     types.String `tfsdk:"cname"`
	Custom404 types.Bool   `tfsdk:"custom_404"`
	HTMLURL   types.String `tfsdk:"html_url"`
	Status    types.String `tfsdk:"status"`
	URL       types.String `tfsdk:"url"`
}

type pagesSourceModel struct {
	Branch types.String `tfsdk:"branch"`
	Path   types.String `tfsdk:"path"`
}

type templateModel struct {
	IncludeAllBranches types.Bool   `tfsdk:"include_all_branches"`
	Owner              types.String `tfsdk:"owner"`
	Repository         types.String `tfsdk:"repository"`
}

func NewGithubRepositoryResource() resource.Resource {
	return &githubRepositoryResource{}
}

func (r *githubRepositoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (r *githubRepositoryResource) SchemaVersion(ctx context.Context) int64 {
	return 1
}

func (r *githubRepositoryResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				// Get the raw state as a dynamic value and convert to map
				rawStateValue := make(map[string]any)
				err := req.State.Raw.As(&rawStateValue)
				if err != nil {
					resp.Diagnostics.AddError(
						"Error Parsing State",
						"Could not parse state during migration: "+err.Error(),
					)
					return
				}

				// Remove any "branches." prefixed attributes (this matches the SDKv2 migration)
				// These were removed in schema version 0 -> 1 migration
				for key := range rawStateValue {
					if strings.HasPrefix(key, "branches.") {
						delete(rawStateValue, key)
					}
				}

				// Create a new model with the migrated data
				var upgradedState githubRepositoryResourceModel

				// Populate the model from the raw state
				if id, ok := rawStateValue["id"]; ok && id != nil {
					upgradedState.ID = types.StringValue(id.(string))
				}
				if name, ok := rawStateValue["name"]; ok && name != nil {
					upgradedState.Name = types.StringValue(name.(string))
				}
				if description, ok := rawStateValue["description"]; ok && description != nil {
					upgradedState.Description = types.StringValue(description.(string))
				}
				if homepageURL, ok := rawStateValue["homepage_url"]; ok && homepageURL != nil {
					upgradedState.HomepageURL = types.StringValue(homepageURL.(string))
				}
				if private, ok := rawStateValue["private"]; ok && private != nil {
					upgradedState.Private = types.BoolValue(private.(bool))
				}
				if visibility, ok := rawStateValue["visibility"]; ok && visibility != nil {
					upgradedState.Visibility = types.StringValue(visibility.(string))
				}
				if hasIssues, ok := rawStateValue["has_issues"]; ok && hasIssues != nil {
					upgradedState.HasIssues = types.BoolValue(hasIssues.(bool))
				}
				if hasDiscussions, ok := rawStateValue["has_discussions"]; ok && hasDiscussions != nil {
					upgradedState.HasDiscussions = types.BoolValue(hasDiscussions.(bool))
				}
				if hasProjects, ok := rawStateValue["has_projects"]; ok && hasProjects != nil {
					upgradedState.HasProjects = types.BoolValue(hasProjects.(bool))
				}
				if hasDownloads, ok := rawStateValue["has_downloads"]; ok && hasDownloads != nil {
					upgradedState.HasDownloads = types.BoolValue(hasDownloads.(bool))
				}
				if hasWiki, ok := rawStateValue["has_wiki"]; ok && hasWiki != nil {
					upgradedState.HasWiki = types.BoolValue(hasWiki.(bool))
				}
				if isTemplate, ok := rawStateValue["is_template"]; ok && isTemplate != nil {
					upgradedState.IsTemplate = types.BoolValue(isTemplate.(bool))
				}
				if allowMergeCommit, ok := rawStateValue["allow_merge_commit"]; ok && allowMergeCommit != nil {
					upgradedState.AllowMergeCommit = types.BoolValue(allowMergeCommit.(bool))
				}
				if allowSquashMerge, ok := rawStateValue["allow_squash_merge"]; ok && allowSquashMerge != nil {
					upgradedState.AllowSquashMerge = types.BoolValue(allowSquashMerge.(bool))
				}
				if allowRebaseMerge, ok := rawStateValue["allow_rebase_merge"]; ok && allowRebaseMerge != nil {
					upgradedState.AllowRebaseMerge = types.BoolValue(allowRebaseMerge.(bool))
				}
				if allowAutoMerge, ok := rawStateValue["allow_auto_merge"]; ok && allowAutoMerge != nil {
					upgradedState.AllowAutoMerge = types.BoolValue(allowAutoMerge.(bool))
				}
				if squashMergeCommitTitle, ok := rawStateValue["squash_merge_commit_title"]; ok && squashMergeCommitTitle != nil {
					upgradedState.SquashMergeCommitTitle = types.StringValue(squashMergeCommitTitle.(string))
				}
				if squashMergeCommitMessage, ok := rawStateValue["squash_merge_commit_message"]; ok && squashMergeCommitMessage != nil {
					upgradedState.SquashMergeCommitMessage = types.StringValue(squashMergeCommitMessage.(string))
				}
				if mergeCommitTitle, ok := rawStateValue["merge_commit_title"]; ok && mergeCommitTitle != nil {
					upgradedState.MergeCommitTitle = types.StringValue(mergeCommitTitle.(string))
				}
				if mergeCommitMessage, ok := rawStateValue["merge_commit_message"]; ok && mergeCommitMessage != nil {
					upgradedState.MergeCommitMessage = types.StringValue(mergeCommitMessage.(string))
				}
				if deleteBranchOnMerge, ok := rawStateValue["delete_branch_on_merge"]; ok && deleteBranchOnMerge != nil {
					upgradedState.DeleteBranchOnMerge = types.BoolValue(deleteBranchOnMerge.(bool))
				}
				if webCommitSignoffRequired, ok := rawStateValue["web_commit_signoff_required"]; ok && webCommitSignoffRequired != nil {
					upgradedState.WebCommitSignoffRequired = types.BoolValue(webCommitSignoffRequired.(bool))
				}
				if autoInit, ok := rawStateValue["auto_init"]; ok && autoInit != nil {
					upgradedState.AutoInit = types.BoolValue(autoInit.(bool))
				}
				if defaultBranch, ok := rawStateValue["default_branch"]; ok && defaultBranch != nil {
					upgradedState.DefaultBranch = types.StringValue(defaultBranch.(string))
				}
				if licenseTemplate, ok := rawStateValue["license_template"]; ok && licenseTemplate != nil {
					upgradedState.LicenseTemplate = types.StringValue(licenseTemplate.(string))
				}
				if gitignoreTemplate, ok := rawStateValue["gitignore_template"]; ok && gitignoreTemplate != nil {
					upgradedState.GitignoreTemplate = types.StringValue(gitignoreTemplate.(string))
				}
				if archived, ok := rawStateValue["archived"]; ok && archived != nil {
					upgradedState.Archived = types.BoolValue(archived.(bool))
				}
				if archiveOnDestroy, ok := rawStateValue["archive_on_destroy"]; ok && archiveOnDestroy != nil {
					upgradedState.ArchiveOnDestroy = types.BoolValue(archiveOnDestroy.(bool))
				}
				if topics, ok := rawStateValue["topics"]; ok && topics != nil {
					if topicsList, ok := topics.([]any); ok {
						topicValues := make([]attr.Value, 0, len(topicsList))
						for _, topic := range topicsList {
							if topicStr, ok := topic.(string); ok {
								topicValues = append(topicValues, types.StringValue(topicStr))
							}
						}
						upgradedState.Topics = types.SetValueMust(types.StringType, topicValues)
					}
				}
				if vulnerabilityAlerts, ok := rawStateValue["vulnerability_alerts"]; ok && vulnerabilityAlerts != nil {
					upgradedState.VulnerabilityAlerts = types.BoolValue(vulnerabilityAlerts.(bool))
				}
				if ignoreVulnerabilityAlertsDuringRead, ok := rawStateValue["ignore_vulnerability_alerts_during_read"]; ok && ignoreVulnerabilityAlertsDuringRead != nil {
					upgradedState.IgnoreVulnerabilityAlertsDuringRead = types.BoolValue(ignoreVulnerabilityAlertsDuringRead.(bool))
				}
				if allowUpdateBranch, ok := rawStateValue["allow_update_branch"]; ok && allowUpdateBranch != nil {
					upgradedState.AllowUpdateBranch = types.BoolValue(allowUpdateBranch.(bool))
				}
				// Computed attributes
				if fullName, ok := rawStateValue["full_name"]; ok && fullName != nil {
					upgradedState.FullName = types.StringValue(fullName.(string))
				}
				if htmlURL, ok := rawStateValue["html_url"]; ok && htmlURL != nil {
					upgradedState.HTMLURL = types.StringValue(htmlURL.(string))
				}
				if sshCloneURL, ok := rawStateValue["ssh_clone_url"]; ok && sshCloneURL != nil {
					upgradedState.SSHCloneURL = types.StringValue(sshCloneURL.(string))
				}
				if svnURL, ok := rawStateValue["svn_url"]; ok && svnURL != nil {
					upgradedState.SVNURL = types.StringValue(svnURL.(string))
				}
				if gitCloneURL, ok := rawStateValue["git_clone_url"]; ok && gitCloneURL != nil {
					upgradedState.GitCloneURL = types.StringValue(gitCloneURL.(string))
				}
				if httpCloneURL, ok := rawStateValue["http_clone_url"]; ok && httpCloneURL != nil {
					upgradedState.HTTPCloneURL = types.StringValue(httpCloneURL.(string))
				}
				if etag, ok := rawStateValue["etag"]; ok && etag != nil {
					upgradedState.ETag = types.StringValue(etag.(string))
				}
				if primaryLanguage, ok := rawStateValue["primary_language"]; ok && primaryLanguage != nil {
					upgradedState.PrimaryLanguage = types.StringValue(primaryLanguage.(string))
				}
				if nodeID, ok := rawStateValue["node_id"]; ok && nodeID != nil {
					upgradedState.NodeID = types.StringValue(nodeID.(string))
				}
				if repoID, ok := rawStateValue["repo_id"]; ok && repoID != nil {
					if repoIDInt, ok := repoID.(int); ok {
						upgradedState.RepoID = types.Int64Value(int64(repoIDInt))
					} else if repoIDFloat, ok := repoID.(float64); ok {
						upgradedState.RepoID = types.Int64Value(int64(repoIDFloat))
					}
				}

				// Handle complex nested blocks - set to null initially
				// These would need more complex migration logic if they existed in v0
				securityFeatureObjType := types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"status": types.StringType,
					},
				}
				securityAnalysisObjType := types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"advanced_security":               types.ListType{ElemType: securityFeatureObjType},
						"secret_scanning":                 types.ListType{ElemType: securityFeatureObjType},
						"secret_scanning_push_protection": types.ListType{ElemType: securityFeatureObjType},
					},
				}
				upgradedState.SecurityAndAnalysis = types.ListNull(securityAnalysisObjType)

				sourceObjType := types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"branch": types.StringType,
						"path":   types.StringType,
					},
				}
				pagesObjType := types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"source":     types.ListType{ElemType: sourceObjType},
						"build_type": types.StringType,
						"cname":      types.StringType,
						"custom_404": types.BoolType,
						"html_url":   types.StringType,
						"status":     types.StringType,
						"url":        types.StringType,
					},
				}
				upgradedState.Pages = types.ListNull(pagesObjType)

				templateObjType := types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"include_all_branches": types.BoolType,
						"owner":                types.StringType,
						"repository":           types.StringType,
					},
				}
				upgradedState.Template = types.ListNull(templateObjType)

				// Set the upgraded state
				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedState)...)
			},
		},
	}
}

func (r *githubRepositoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages repositories within GitHub organizations or personal accounts",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The repository name.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[-a-zA-Z0-9_.]{1,100}$`),
						"must include only alphanumeric characters, underscores or hyphens and consist of 100 characters or less",
					),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of the repository.",
				Optional:    true,
				Computed:    true,
			},
			"homepage_url": schema.StringAttribute{
				Description: "URL of a page describing the project.",
				Optional:    true,
				Computed:    true,
			},
			"private": schema.BoolAttribute{
				Description:        "Set to 'true' to create a private repository. This is deprecated, use 'visibility' instead.",
				Optional:           true,
				Computed:           true,
				DeprecationMessage: "use visibility instead",
			},
			"visibility": schema.StringAttribute{
				Description: "Can be 'public' or 'private'. If your organization is associated with an enterprise account using GitHub Enterprise Cloud or GitHub Enterprise Server 2.20+, visibility can also be 'internal'.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("public", "private", "internal"),
				},
			},
			"has_issues": schema.BoolAttribute{
				Description: "Set to 'true' to enable the GitHub Issues features on the repository",
				Optional:    true,
				Computed:    true,
			},
			"has_discussions": schema.BoolAttribute{
				Description: "Set to 'true' to enable GitHub Discussions on the repository. Defaults to 'false'.",
				Optional:    true,
				Computed:    true,
			},
			"has_projects": schema.BoolAttribute{
				Description: "Set to 'true' to enable the GitHub Projects features on the repository. Per the GitHub documentation when in an organization that has disabled repository projects it will default to 'false' and will otherwise default to 'true'. If you specify 'true' when it has been disabled it will return an error.",
				Optional:    true,
				Computed:    true,
			},
			"has_downloads": schema.BoolAttribute{
				Description: "Set to 'true' to enable the (deprecated) downloads features on the repository.",
				Optional:    true,
				Computed:    true,
			},
			"has_wiki": schema.BoolAttribute{
				Description: "Set to 'true' to enable the GitHub Wiki features on the repository.",
				Optional:    true,
				Computed:    true,
			},
			"is_template": schema.BoolAttribute{
				Description: "Set to 'true' to tell GitHub that this is a template repository.",
				Optional:    true,
				Computed:    true,
			},
			"allow_merge_commit": schema.BoolAttribute{
				Description: "Set to 'false' to disable merge commits on the repository.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"allow_squash_merge": schema.BoolAttribute{
				Description: "Set to 'false' to disable squash merges on the repository.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"allow_rebase_merge": schema.BoolAttribute{
				Description: "Set to 'false' to disable rebase merges on the repository.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"allow_auto_merge": schema.BoolAttribute{
				Description: "Set to 'true' to allow auto-merging pull requests on the repository.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"squash_merge_commit_title": schema.StringAttribute{
				Description: "Can be 'PR_TITLE' or 'COMMIT_OR_PR_TITLE' for a default squash merge commit title.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("COMMIT_OR_PR_TITLE"),
			},
			"squash_merge_commit_message": schema.StringAttribute{
				Description: "Can be 'PR_BODY', 'COMMIT_MESSAGES', or 'BLANK' for a default squash merge commit message.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("COMMIT_MESSAGES"),
			},
			"merge_commit_title": schema.StringAttribute{
				Description: "Can be 'PR_TITLE' or 'MERGE_MESSAGE' for a default merge commit title.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("MERGE_MESSAGE"),
			},
			"merge_commit_message": schema.StringAttribute{
				Description: "Can be 'PR_BODY', 'PR_TITLE', or 'BLANK' for a default merge commit message.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("PR_TITLE"),
			},
			"delete_branch_on_merge": schema.BoolAttribute{
				Description: "Automatically delete head branch after a pull request is merged. Defaults to 'false'.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"web_commit_signoff_required": schema.BoolAttribute{
				Description: "Require contributors to sign off on web-based commits. Defaults to 'false'.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"auto_init": schema.BoolAttribute{
				Description: "Set to 'true' to produce an initial commit in the repository.",
				Optional:    true,
			},
			"default_branch": schema.StringAttribute{
				Description:        "Can only be set after initial repository creation, and only if the target branch exists",
				Optional:           true,
				Computed:           true,
				DeprecationMessage: "Use the github_branch_default resource instead",
			},
			"license_template": schema.StringAttribute{
				Description: "Use the name of the template without the extension. For example, 'mit' or 'mpl-2.0'.",
				Optional:    true,
			},
			"gitignore_template": schema.StringAttribute{
				Description: "Use the name of the template without the extension. For example, 'Haskell'.",
				Optional:    true,
			},
			"archived": schema.BoolAttribute{
				Description: "Specifies if the repository should be archived. Defaults to 'false'. NOTE Currently, the API does not support unarchiving.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"archive_on_destroy": schema.BoolAttribute{
				Description: "Set to 'true' to archive the repository instead of deleting on destroy.",
				Optional:    true,
			},
			"topics": schema.SetAttribute{
				Description: "The list of topics of the repository.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,49}$`),
						"must include only lowercase alphanumeric characters or hyphens and cannot start with a hyphen and consist of 50 characters or less",
					)),
				},
			},
			"vulnerability_alerts": schema.BoolAttribute{
				Description: "Set to 'true' to enable security alerts for vulnerable dependencies. Enabling requires alerts to be enabled on the owner level. (Note for importing: GitHub enables the alerts on public repos but disables them on private repos by default). Note that vulnerability alerts have not been successfully tested on any GitHub Enterprise instance and may be unavailable in those settings.",
				Optional:    true,
				Computed:    true,
			},
			"ignore_vulnerability_alerts_during_read": schema.BoolAttribute{
				Description: "Set to true to not call the vulnerability alerts endpoint so the resource can also be used without admin permissions during read.",
				Optional:    true,
			},
			"allow_update_branch": schema.BoolAttribute{
				Description: " Set to 'true' to always suggest updating pull request branches.",
				Optional:    true,
				Computed:    true,
			},
			// Computed attributes
			"full_name": schema.StringAttribute{
				Description: "A string of the form 'orgname/reponame'.",
				Computed:    true,
			},
			"html_url": schema.StringAttribute{
				Description: "URL to the repository on the web.",
				Computed:    true,
			},
			"ssh_clone_url": schema.StringAttribute{
				Description: "URL that can be provided to 'git clone' to clone the repository via SSH.",
				Computed:    true,
			},
			"svn_url": schema.StringAttribute{
				Description: "URL that can be provided to 'svn checkout' to check out the repository via GitHub's Subversion protocol emulation.",
				Computed:    true,
			},
			"git_clone_url": schema.StringAttribute{
				Description: "URL that can be provided to 'git clone' to clone the repository anonymously via the git protocol.",
				Computed:    true,
			},
			"http_clone_url": schema.StringAttribute{
				Description: "URL that can be provided to 'git clone' to clone the repository via HTTPS.",
				Computed:    true,
			},
			"etag": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"primary_language": schema.StringAttribute{
				Computed: true,
			},
			"node_id": schema.StringAttribute{
				Description: "GraphQL global node id for use with v4 API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repo_id": schema.Int64Attribute{
				Description: "GitHub ID for the repository.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"security_and_analysis": schema.ListNestedBlock{
				Description: "Security and analysis settings for the repository. To use this parameter you must have admin permissions for the repository or be an owner or security manager for the organization that owns the repository.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Blocks: map[string]schema.Block{
						"advanced_security": schema.ListNestedBlock{
							Description: "The advanced security configuration for the repository. If a repository's visibility is 'public', advanced security is always enabled and cannot be changed, so this setting cannot be supplied.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"status": schema.StringAttribute{
										Description: "Set to 'enabled' to enable advanced security features on the repository. Can be 'enabled' or 'disabled'.",
										Required:    true,
										Validators: []validator.String{
											stringvalidator.OneOf("enabled", "disabled"),
										},
									},
								},
							},
						},
						"secret_scanning": schema.ListNestedBlock{
							Description: "The secret scanning configuration for the repository.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"status": schema.StringAttribute{
										Description: "Set to 'enabled' to enable secret scanning on the repository. Can be 'enabled' or 'disabled'. If set to 'enabled', the repository's visibility must be 'public' or 'security_and_analysis[0].advanced_security[0].status' must also be set to 'enabled'.",
										Required:    true,
										Validators: []validator.String{
											stringvalidator.OneOf("enabled", "disabled"),
										},
									},
								},
							},
						},
						"secret_scanning_push_protection": schema.ListNestedBlock{
							Description: "The secret scanning push protection configuration for the repository.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"status": schema.StringAttribute{
										Description: "Set to 'enabled' to enable secret scanning push protection on the repository. Can be 'enabled' or 'disabled'. If set to 'enabled', the repository's visibility must be 'public' or 'security_and_analysis[0].advanced_security[0].status' must also be set to 'enabled'.",
										Required:    true,
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
				Description: "The repository's GitHub Pages configuration",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"build_type": schema.StringAttribute{
							Description: "The type the page should be sourced.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("legacy"),
							Validators: []validator.String{
								stringvalidator.OneOf("legacy", "workflow"),
							},
						},
						"cname": schema.StringAttribute{
							Description: "The custom domain for the repository. This can only be set after the repository has been created.",
							Optional:    true,
							Computed:    true,
						},
						"custom_404": schema.BoolAttribute{
							Description: "Whether the rendered GitHub Pages site has a custom 404 page",
							Computed:    true,
						},
						"html_url": schema.StringAttribute{
							Description: "URL to the repository on the web.",
							Computed:    true,
						},
						"status": schema.StringAttribute{
							Description: "The GitHub Pages site's build status e.g. building or built.",
							Computed:    true,
						},
						"url": schema.StringAttribute{
							Computed: true,
						},
					},
					Blocks: map[string]schema.Block{
						"source": schema.ListNestedBlock{
							Description: "The source branch and directory for the rendered Pages site.",
							Validators: []validator.List{
								listvalidator.SizeAtMost(1),
							},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"branch": schema.StringAttribute{
										Description: "The repository branch used to publish the site's source files. (i.e. 'main' or 'gh-pages')",
										Required:    true,
									},
									"path": schema.StringAttribute{
										Description: "The repository directory from which the site publishes (Default: '/')",
										Optional:    true,
										Computed:    true,
										Default:     stringdefault.StaticString("/"),
									},
								},
							},
						},
					},
				},
			},
			"template": schema.ListNestedBlock{
				Description: "Use a template repository to create this resource.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"include_all_branches": schema.BoolAttribute{
							Description: "Whether the new repository should include all the branches from the template repository (defaults to 'false', which includes only the default branch from the template).",
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
						},
						"owner": schema.StringAttribute{
							Description: "The GitHub organization or user the template repository is owned by.",
							Required:    true,
						},
						"repository": schema.StringAttribute{
							Description: "The name of the template repository.",
							Required:    true,
						},
					},
				},
			},
		},
	}
}

func (r *githubRepositoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubRepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if default_branch is set to something other than "main" during creation
	if !plan.DefaultBranch.IsNull() && !plan.DefaultBranch.IsUnknown() {
		if plan.DefaultBranch.ValueString() != "main" {
			resp.Diagnostics.AddError(
				"Invalid Default Branch",
				"cannot set the default branch on a new repository to something other than 'main'",
			)
			return
		}
	}

	// Build repository object
	repository, diags := r.buildGithubRepositoryFromPlan(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoName := plan.Name.ValueString()
	owner := r.client.Name()

	// Handle template repository creation
	if !plan.Template.IsNull() && !plan.Template.IsUnknown() {
		var templateModels []templateModel
		resp.Diagnostics.Append(plan.Template.ElementsAs(ctx, &templateModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(templateModels) > 0 {
			templateModel := templateModels[0]

			// Determine if repository should be private
			isPrivate := r.calculatePrivateFromPlan(plan)

			templateRepoReq := github.TemplateRepoRequest{
				Name:               &repoName,
				Owner:              &owner,
				Private:            github.Ptr(isPrivate),
				IncludeAllBranches: github.Ptr(templateModel.IncludeAllBranches.ValueBool()),
			}

			// Only set description if it was configured by the user
			if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
				templateRepoReq.Description = github.Ptr(plan.Description.ValueString())
			}

			repo, _, err := r.client.V3Client().Repositories.CreateFromTemplate(ctx,
				templateModel.Owner.ValueString(),
				templateModel.Repository.ValueString(),
				&templateRepoReq,
			)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Creating Repository From Template",
					"Could not create repository from template, unexpected error: "+err.Error(),
				)
				return
			}

			plan.Name = types.StringValue(*repo.Name)
		}
	} else {
		// Create repository without template
		var repo *github.Repository
		var err error
		if r.client.IsOrganization {
			repo, _, err = r.client.V3Client().Repositories.Create(ctx, owner, repository)
		} else {
			// Create repository within authenticated user's account
			repo, _, err = r.client.V3Client().Repositories.Create(ctx, "", repository)
		}
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Creating Repository",
				"Could not create repository, unexpected error: "+err.Error(),
			)
			return
		}
		plan.Name = types.StringValue(repo.GetName())
	}

	// Handle topics if specified
	if !plan.Topics.IsNull() && !plan.Topics.IsUnknown() {
		var topics []string
		resp.Diagnostics.Append(plan.Topics.ElementsAs(ctx, &topics, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(topics) > 0 {
			_, _, err := r.client.V3Client().Repositories.ReplaceAllTopics(ctx, owner, repoName, topics)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Setting Repository Topics",
					"Could not set repository topics, unexpected error: "+err.Error(),
				)
				return
			}
		}
	}

	// Handle pages configuration if specified
	if !plan.Pages.IsNull() && !plan.Pages.IsUnknown() {
		var pagesModels []pagesModel
		resp.Diagnostics.Append(plan.Pages.ElementsAs(ctx, &pagesModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(pagesModels) > 0 {
			pages := r.expandPages(ctx, pagesModels[0], resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}

			if pages != nil {
				_, _, err := r.client.V3Client().Repositories.EnablePages(ctx, owner, repoName, pages)
				if err != nil {
					resp.Diagnostics.AddError(
						"Error Enabling Repository Pages",
						"Could not enable repository pages, unexpected error: "+err.Error(),
					)
					return
				}
			}
		}
	}

	// Set the resource ID and name
	plan.ID = types.StringValue(repoName)
	plan.Name = types.StringValue(repoName)

	// Perform an update to apply remaining settings
	r.performUpdate(ctx, plan, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save whether security_and_analysis was configured by the user
	securityWasConfigured := !plan.SecurityAndAnalysis.IsNull()

	// Read the repository to get computed values
	r.readRepository(ctx, &plan, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve null state for security_and_analysis if user didn't configure it
	if !securityWasConfigured && !plan.SecurityAndAnalysis.IsNull() {
		// Define the nested object types for security and analysis
		securityFeatureObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"status": types.StringType,
			},
		}
		securityAnalysisObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"advanced_security":               types.ListType{ElemType: securityFeatureObjType},
				"secret_scanning":                 types.ListType{ElemType: securityFeatureObjType},
				"secret_scanning_push_protection": types.ListType{ElemType: securityFeatureObjType},
			},
		}
		plan.SecurityAndAnalysis = types.ListNull(securityAnalysisObjType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *githubRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubRepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save whether security_and_analysis was null in the existing state
	securityWasNull := state.SecurityAndAnalysis.IsNull()

	r.readRepository(ctx, &state, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve null state for security_and_analysis if it was null before
	if securityWasNull && !state.SecurityAndAnalysis.IsNull() {
		// Define the nested object types for security and analysis
		securityFeatureObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"status": types.StringType,
			},
		}
		securityAnalysisObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"advanced_security":               types.ListType{ElemType: securityFeatureObjType},
				"secret_scanning":                 types.ListType{ElemType: securityFeatureObjType},
				"secret_scanning_push_protection": types.ListType{ElemType: securityFeatureObjType},
			},
		}
		state.SecurityAndAnalysis = types.ListNull(securityAnalysisObjType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *githubRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubRepositoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if repository is archived and not being unarchived (which is not supported)
	var state githubRepositoryResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if (!state.Archived.IsNull() && state.Archived.ValueBool()) &&
		(!plan.Archived.IsNull() && plan.Archived.ValueBool()) {
		log.Printf("[INFO] Skipping update of archived repository")
		return
	}

	r.performUpdate(ctx, plan, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save whether security_and_analysis was configured by the user
	securityWasConfigured := !plan.SecurityAndAnalysis.IsNull()

	// Read the repository to get updated computed values
	r.readRepository(ctx, &plan, resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve null state for security_and_analysis if user didn't configure it
	if !securityWasConfigured && !plan.SecurityAndAnalysis.IsNull() {
		// Define the nested object types for security and analysis
		securityFeatureObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"status": types.StringType,
			},
		}
		securityAnalysisObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"advanced_security":               types.ListType{ElemType: securityFeatureObjType},
				"secret_scanning":                 types.ListType{ElemType: securityFeatureObjType},
				"secret_scanning_push_protection": types.ListType{ElemType: securityFeatureObjType},
			},
		}
		plan.SecurityAndAnalysis = types.ListNull(securityAnalysisObjType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *githubRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubRepositoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoName := state.Name.ValueString()
	owner := r.client.Name()

	// Check if we should archive instead of delete
	if !state.ArchiveOnDestroy.IsNull() && state.ArchiveOnDestroy.ValueBool() {
		if !state.Archived.IsNull() && state.Archived.ValueBool() {
			log.Printf("[DEBUG] Repository already archived, nothing to do on delete: %s/%s", owner, repoName)
			return
		} else {
			// Archive the repository
			state.Archived = types.BoolValue(true)
			repository, diags := r.buildGithubRepositoryFromPlan(ctx, state)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			log.Printf("[DEBUG] Archiving repository on delete: %s/%s", owner, repoName)
			_, _, err := r.client.V3Client().Repositories.Edit(ctx, owner, repoName, repository)
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Archiving Repository",
					"Could not archive repository on delete, unexpected error: "+err.Error(),
				)
			}
			return
		}
	}

	// Delete the repository
	log.Printf("[DEBUG] Deleting repository: %s/%s", owner, repoName)
	_, err := r.client.V3Client().Repositories.Delete(ctx, owner, repoName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Repository",
			"Could not delete repository, unexpected error: "+err.Error(),
		)
	}
}

func (r *githubRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Set the resource ID and name to the repository name
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), types.StringValue(req.ID))...)

	// Set auto_init to false for imported repositories (matches SDKv2 behavior)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("auto_init"), types.BoolValue(false))...)
}

// Helper functions

func (r *githubRepositoryResource) calculatePrivateFromPlan(plan githubRepositoryResourceModel) bool {
	// Prefer visibility over private flag since private flag is deprecated
	if !plan.Visibility.IsNull() && !plan.Visibility.IsUnknown() {
		visibility := plan.Visibility.ValueString()
		return visibility == "private" || visibility == "internal"
	}

	if !plan.Private.IsNull() && !plan.Private.IsUnknown() {
		return plan.Private.ValueBool()
	}

	return false
}

func (r *githubRepositoryResource) calculateVisibilityFromPlan(plan githubRepositoryResourceModel) string {
	if !plan.Visibility.IsNull() && !plan.Visibility.IsUnknown() {
		return plan.Visibility.ValueString()
	}

	if !plan.Private.IsNull() && !plan.Private.IsUnknown() {
		if plan.Private.ValueBool() {
			return "private"
		} else {
			return "public"
		}
	}

	return "public"
}

func (r *githubRepositoryResource) buildGithubRepositoryFromPlan(ctx context.Context, plan githubRepositoryResourceModel) (*github.Repository, diag.Diagnostics) {
	var diags diag.Diagnostics

	repository := &github.Repository{
		Name:       github.Ptr(plan.Name.ValueString()),
		Visibility: github.Ptr(r.calculateVisibilityFromPlan(plan)),
	}

	// Only set description if it was configured by the user
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		repository.Description = github.Ptr(plan.Description.ValueString())
	}

	// Only set homepage if it's not null and not empty
	if !plan.HomepageURL.IsNull() && !plan.HomepageURL.IsUnknown() && plan.HomepageURL.ValueString() != "" {
		repository.Homepage = github.Ptr(plan.HomepageURL.ValueString())
	}

	// Only set optional boolean attributes if they're not null
	if !plan.HasDownloads.IsNull() && !plan.HasDownloads.IsUnknown() {
		repository.HasDownloads = github.Ptr(plan.HasDownloads.ValueBool())
	}
	if !plan.HasIssues.IsNull() && !plan.HasIssues.IsUnknown() {
		repository.HasIssues = github.Ptr(plan.HasIssues.ValueBool())
	}
	if !plan.HasDiscussions.IsNull() && !plan.HasDiscussions.IsUnknown() {
		repository.HasDiscussions = github.Ptr(plan.HasDiscussions.ValueBool())
	}
	if !plan.HasProjects.IsNull() && !plan.HasProjects.IsUnknown() {
		repository.HasProjects = github.Ptr(plan.HasProjects.ValueBool())
	}
	if !plan.HasWiki.IsNull() && !plan.HasWiki.IsUnknown() {
		repository.HasWiki = github.Ptr(plan.HasWiki.ValueBool())
	}
	if !plan.IsTemplate.IsNull() && !plan.IsTemplate.IsUnknown() {
		repository.IsTemplate = github.Ptr(plan.IsTemplate.ValueBool())
	}
	if !plan.AutoInit.IsNull() && !plan.AutoInit.IsUnknown() {
		repository.AutoInit = github.Ptr(plan.AutoInit.ValueBool())
	}
	if !plan.AllowUpdateBranch.IsNull() && !plan.AllowUpdateBranch.IsUnknown() {
		repository.AllowUpdateBranch = github.Ptr(plan.AllowUpdateBranch.ValueBool())
	}

	// These attributes have defaults and should always be set
	repository.AllowMergeCommit = github.Ptr(plan.AllowMergeCommit.ValueBool())
	repository.AllowSquashMerge = github.Ptr(plan.AllowSquashMerge.ValueBool())
	repository.AllowRebaseMerge = github.Ptr(plan.AllowRebaseMerge.ValueBool())
	repository.AllowAutoMerge = github.Ptr(plan.AllowAutoMerge.ValueBool())
	repository.DeleteBranchOnMerge = github.Ptr(plan.DeleteBranchOnMerge.ValueBool())
	repository.WebCommitSignoffRequired = github.Ptr(plan.WebCommitSignoffRequired.ValueBool())
	repository.Archived = github.Ptr(plan.Archived.ValueBool())

	// Only set license and gitignore templates if not null and not empty
	if !plan.LicenseTemplate.IsNull() && !plan.LicenseTemplate.IsUnknown() && plan.LicenseTemplate.ValueString() != "" {
		repository.LicenseTemplate = github.Ptr(plan.LicenseTemplate.ValueString())
	}
	if !plan.GitignoreTemplate.IsNull() && !plan.GitignoreTemplate.IsUnknown() && plan.GitignoreTemplate.ValueString() != "" {
		repository.GitignoreTemplate = github.Ptr(plan.GitignoreTemplate.ValueString())
	}

	// Handle topics
	if !plan.Topics.IsNull() && !plan.Topics.IsUnknown() {
		var topics []string
		diags.Append(plan.Topics.ElementsAs(ctx, &topics, false)...)
		if diags.HasError() {
			return nil, diags
		}
		repository.Topics = topics
	}

	// Handle merge commit settings only if merge commits are allowed
	if !plan.AllowMergeCommit.IsNull() && plan.AllowMergeCommit.ValueBool() {
		repository.MergeCommitTitle = github.Ptr(plan.MergeCommitTitle.ValueString())
		repository.MergeCommitMessage = github.Ptr(plan.MergeCommitMessage.ValueString())
	}

	// Handle squash commit settings only if squash merges are allowed
	if !plan.AllowSquashMerge.IsNull() && plan.AllowSquashMerge.ValueBool() {
		repository.SquashMergeCommitTitle = github.Ptr(plan.SquashMergeCommitTitle.ValueString())
		repository.SquashMergeCommitMessage = github.Ptr(plan.SquashMergeCommitMessage.ValueString())
	}

	// Handle security and analysis
	if !plan.SecurityAndAnalysis.IsNull() && !plan.SecurityAndAnalysis.IsUnknown() {
		var securityModels []securityAndAnalysisModel
		diags.Append(plan.SecurityAndAnalysis.ElementsAs(ctx, &securityModels, false)...)
		if diags.HasError() {
			return nil, diags
		}

		if len(securityModels) > 0 {
			securityAnalysis := r.expandSecurityAndAnalysis(ctx, securityModels[0], diags)
			if diags.HasError() {
				return nil, diags
			}
			repository.SecurityAndAnalysis = securityAnalysis
		}
	}

	return repository, diags
}

func (r *githubRepositoryResource) expandSecurityAndAnalysis(ctx context.Context, model securityAndAnalysisModel, diags diag.Diagnostics) *github.SecurityAndAnalysis {
	var securityAnalysis github.SecurityAndAnalysis

	// Advanced Security
	if !model.AdvancedSecurity.IsNull() && !model.AdvancedSecurity.IsUnknown() {
		var advSecModels []securityFeatureModel
		diags.Append(model.AdvancedSecurity.ElementsAs(ctx, &advSecModels, false)...)
		if diags.HasError() {
			return nil
		}
		if len(advSecModels) > 0 {
			securityAnalysis.AdvancedSecurity = &github.AdvancedSecurity{
				Status: github.Ptr(advSecModels[0].Status.ValueString()),
			}
		}
	}

	// Secret Scanning
	if !model.SecretScanning.IsNull() && !model.SecretScanning.IsUnknown() {
		var secretScanModels []securityFeatureModel
		diags.Append(model.SecretScanning.ElementsAs(ctx, &secretScanModels, false)...)
		if diags.HasError() {
			return nil
		}
		if len(secretScanModels) > 0 {
			securityAnalysis.SecretScanning = &github.SecretScanning{
				Status: github.Ptr(secretScanModels[0].Status.ValueString()),
			}
		}
	}

	// Secret Scanning Push Protection
	if !model.SecretScanningPushProtection.IsNull() && !model.SecretScanningPushProtection.IsUnknown() {
		var secretScanPPModels []securityFeatureModel
		diags.Append(model.SecretScanningPushProtection.ElementsAs(ctx, &secretScanPPModels, false)...)
		if diags.HasError() {
			return nil
		}
		if len(secretScanPPModels) > 0 {
			securityAnalysis.SecretScanningPushProtection = &github.SecretScanningPushProtection{
				Status: github.Ptr(secretScanPPModels[0].Status.ValueString()),
			}
		}
	}

	return &securityAnalysis
}

func (r *githubRepositoryResource) expandPages(ctx context.Context, model pagesModel, diags diag.Diagnostics) *github.Pages {
	source := &github.PagesSource{
		Branch: github.Ptr("main"),
	}

	if !model.Source.IsNull() && !model.Source.IsUnknown() {
		var sourceModels []pagesSourceModel
		diags.Append(model.Source.ElementsAs(ctx, &sourceModels, false)...)
		if diags.HasError() {
			return nil
		}

		if len(sourceModels) > 0 {
			sourceModel := sourceModels[0]
			source.Branch = github.Ptr(sourceModel.Branch.ValueString())

			// To set to the root directory "/", leave source.Path unset
			path := sourceModel.Path.ValueString()
			if path != "" && path != "/" {
				source.Path = github.Ptr(path)
			}
		}
	}

	var buildType *string
	if !model.BuildType.IsNull() && !model.BuildType.IsUnknown() {
		buildType = github.Ptr(model.BuildType.ValueString())
	}

	return &github.Pages{Source: source, BuildType: buildType}
}

func (r *githubRepositoryResource) readRepository(ctx context.Context, state *githubRepositoryResourceModel, diags diag.Diagnostics) {
	owner := r.client.Name()
	repoName := state.Name.ValueString()

	// When the user has not authenticated the provider, use explicit owner if available
	if explicitOwner, _, ok := r.parseFullName(state); ok && owner == "" {
		owner = explicitOwner
	}

	// Set context values for etag handling
	requestCtx := context.WithValue(ctx, CtxId, repoName)
	if !state.ETag.IsNull() && !state.ETag.IsUnknown() {
		requestCtx = context.WithValue(requestCtx, CtxEtag, state.ETag.ValueString())
	}

	repo, resp, err := r.client.V3Client().Repositories.Get(requestCtx, owner, repoName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing repository %s/%s from state because it no longer exists in GitHub", owner, repoName)
				diags.AddWarning(
					"Repository Not Found",
					fmt.Sprintf("Repository %s/%s was not found and will be removed from state", owner, repoName),
				)
				return
			}
		}
		diags.AddError(
			"Error Reading Repository",
			"Could not read repository, unexpected error: "+err.Error(),
		)
		return
	}

	// Update state with repository data
	state.ETag = types.StringValue(resp.Header.Get("ETag"))
	state.ID = types.StringValue(repoName)
	state.Name = types.StringValue(repoName)
	state.Description = types.StringValue(repo.GetDescription())
	state.PrimaryLanguage = types.StringValue(repo.GetLanguage())
	state.HomepageURL = types.StringValue(repo.GetHomepage())
	state.Private = types.BoolValue(repo.GetPrivate())
	state.Visibility = types.StringValue(repo.GetVisibility())
	state.HasIssues = types.BoolValue(repo.GetHasIssues())
	state.HasDiscussions = types.BoolValue(repo.GetHasDiscussions())
	state.HasProjects = types.BoolValue(repo.GetHasProjects())
	state.HasWiki = types.BoolValue(repo.GetHasWiki())
	state.IsTemplate = types.BoolValue(repo.GetIsTemplate())
	state.FullName = types.StringValue(repo.GetFullName())
	state.DefaultBranch = types.StringValue(repo.GetDefaultBranch())
	state.HTMLURL = types.StringValue(repo.GetHTMLURL())
	state.SSHCloneURL = types.StringValue(repo.GetSSHURL())
	state.SVNURL = types.StringValue(repo.GetSVNURL())
	state.GitCloneURL = types.StringValue(repo.GetGitURL())
	state.HTTPCloneURL = types.StringValue(repo.GetCloneURL())
	state.Archived = types.BoolValue(repo.GetArchived())
	state.NodeID = types.StringValue(repo.GetNodeID())
	state.RepoID = types.Int64Value(repo.GetID())

	// Set topics
	if len(repo.Topics) > 0 {
		topicValues := make([]attr.Value, 0, len(repo.Topics))
		for _, topic := range repo.Topics {
			topicValues = append(topicValues, types.StringValue(topic))
		}
		state.Topics = types.SetValueMust(types.StringType, topicValues)
	} else {
		state.Topics = types.SetValueMust(types.StringType, []attr.Value{})
	}

	// GitHub API doesn't respond with following parameters when repository is archived
	if !repo.GetArchived() {
		state.AllowAutoMerge = types.BoolValue(repo.GetAllowAutoMerge())
		state.AllowMergeCommit = types.BoolValue(repo.GetAllowMergeCommit())
		state.AllowRebaseMerge = types.BoolValue(repo.GetAllowRebaseMerge())
		state.AllowSquashMerge = types.BoolValue(repo.GetAllowSquashMerge())
		state.AllowUpdateBranch = types.BoolValue(repo.GetAllowUpdateBranch())
		state.DeleteBranchOnMerge = types.BoolValue(repo.GetDeleteBranchOnMerge())
		state.WebCommitSignoffRequired = types.BoolValue(repo.GetWebCommitSignoffRequired())
		state.HasDownloads = types.BoolValue(repo.GetHasDownloads())
		state.MergeCommitMessage = types.StringValue(repo.GetMergeCommitMessage())
		state.MergeCommitTitle = types.StringValue(repo.GetMergeCommitTitle())
		state.SquashMergeCommitMessage = types.StringValue(repo.GetSquashMergeCommitMessage())
		state.SquashMergeCommitTitle = types.StringValue(repo.GetSquashMergeCommitTitle())
	}

	// Handle GitHub Pages
	if repo.GetHasPages() {
		pages, _, err := r.client.V3Client().Repositories.GetPagesInfo(requestCtx, owner, repoName)
		if err != nil {
			diags.AddError(
				"Error Reading Repository Pages",
				"Could not read repository pages, unexpected error: "+err.Error(),
			)
			return
		}
		state.Pages = r.flattenPages(ctx, pages, diags)
	} else {
		// Define the nested object types for pages
		sourceObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"branch": types.StringType,
				"path":   types.StringType,
			},
		}
		pagesObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"source":     types.ListType{ElemType: sourceObjType},
				"build_type": types.StringType,
				"url":        types.StringType,
				"status":     types.StringType,
				"cname":      types.StringType,
				"custom_404": types.BoolType,
				"html_url":   types.StringType,
			},
		}
		state.Pages = types.ListNull(pagesObjType)
	}

	// Handle template repository information
	if repo.TemplateRepository != nil {
		templateObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"owner":                types.StringType,
				"repository":           types.StringType,
				"include_all_branches": types.BoolType,
			},
		}
		templateObj := types.ObjectValueMust(
			templateObjType.AttrTypes,
			map[string]attr.Value{
				"owner":                types.StringValue(repo.TemplateRepository.Owner.GetLogin()),
				"repository":           types.StringValue(repo.TemplateRepository.GetName()),
				"include_all_branches": types.BoolValue(false), // This isn't returned by API, default to false
			},
		)
		state.Template = types.ListValueMust(templateObjType, []attr.Value{templateObj})
	} else {
		templateObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"owner":                types.StringType,
				"repository":           types.StringType,
				"include_all_branches": types.BoolType,
			},
		}
		state.Template = types.ListNull(templateObjType)
	}

	// Handle vulnerability alerts
	if state.IgnoreVulnerabilityAlertsDuringRead.IsNull() || !state.IgnoreVulnerabilityAlertsDuringRead.ValueBool() {
		vulnerabilityAlerts, _, err := r.client.V3Client().Repositories.GetVulnerabilityAlerts(requestCtx, owner, repoName)
		if err != nil {
			diags.AddError(
				"Error Reading Repository Vulnerability Alerts",
				"Could not read repository vulnerability alerts, unexpected error: "+err.Error(),
			)
			return
		}
		state.VulnerabilityAlerts = types.BoolValue(vulnerabilityAlerts)
	}

	// Handle security and analysis
	if repo.GetSecurityAndAnalysis() != nil {
		state.SecurityAndAnalysis = r.flattenSecurityAndAnalysis(ctx, repo.GetSecurityAndAnalysis(), diags)
	} else {
		// Define the nested object types for security and analysis
		securityFeatureObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"status": types.StringType,
			},
		}
		securityAnalysisObjType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"advanced_security":               types.ListType{ElemType: securityFeatureObjType},
				"secret_scanning":                 types.ListType{ElemType: securityFeatureObjType},
				"secret_scanning_push_protection": types.ListType{ElemType: securityFeatureObjType},
			},
		}
		state.SecurityAndAnalysis = types.ListNull(securityAnalysisObjType)
	}
}

func (r *githubRepositoryResource) performUpdate(ctx context.Context, plan githubRepositoryResourceModel, diags diag.Diagnostics) {
	owner := r.client.Name()
	repoName := plan.Name.ValueString()
	requestCtx := context.WithValue(ctx, CtxId, repoName)

	// Build repository object
	repository, buildDiags := r.buildGithubRepositoryFromPlan(ctx, plan)
	diags.Append(buildDiags...)
	if diags.HasError() {
		return
	}

	// Handle visibility updates separately from other fields (matches SDKv2 behavior)
	repository.Visibility = nil

	// The documentation for `default_branch` states: "This can only be set
	// after a repository has already been created". Only set it if it has changed.
	// For now, we'll skip this check in the framework version as we don't have
	// easy access to the old state in the update method.

	repo, _, err := r.client.V3Client().Repositories.Edit(requestCtx, owner, repoName, repository)
	if err != nil {
		diags.AddError(
			"Error Updating Repository",
			"Could not update repository, unexpected error: "+err.Error(),
		)
		return
	}

	// Update topics if specified
	if !plan.Topics.IsNull() && !plan.Topics.IsUnknown() {
		var topics []string
		diags.Append(plan.Topics.ElementsAs(ctx, &topics, false)...)
		if diags.HasError() {
			return
		}

		_, _, err := r.client.V3Client().Repositories.ReplaceAllTopics(requestCtx, owner, repo.GetName(), topics)
		if err != nil {
			diags.AddError(
				"Error Updating Repository Topics",
				"Could not update repository topics, unexpected error: "+err.Error(),
			)
			return
		}
	}

	// Handle visibility update
	if !plan.Visibility.IsNull() && !plan.Visibility.IsUnknown() {
		visibilityRepo := &github.Repository{
			Visibility: github.Ptr(plan.Visibility.ValueString()),
		}
		log.Printf("[DEBUG] Updating repository visibility to %s", plan.Visibility.ValueString())
		_, resp, err := r.client.V3Client().Repositories.Edit(requestCtx, owner, repoName, visibilityRepo)
		if err != nil {
			// Check if it's just saying visibility is already set
			if resp.StatusCode != 422 || !strings.Contains(err.Error(), fmt.Sprintf("Visibility is already %s", plan.Visibility.ValueString())) {
				diags.AddError(
					"Error Updating Repository Visibility",
					"Could not update repository visibility, unexpected error: "+err.Error(),
				)
				return
			}
		}
	}

	// Handle private flag update (deprecated but still supported)
	if !plan.Private.IsNull() && !plan.Private.IsUnknown() {
		privateRepo := &github.Repository{
			Private: github.Ptr(plan.Private.ValueBool()),
		}
		log.Printf("[DEBUG] Updating repository privacy to %v", plan.Private.ValueBool())
		_, _, err := r.client.V3Client().Repositories.Edit(requestCtx, owner, repoName, privateRepo)
		if err != nil {
			if !strings.Contains(err.Error(), "422 Privacy is already set") {
				diags.AddError(
					"Error Updating Repository Privacy",
					"Could not update repository privacy, unexpected error: "+err.Error(),
				)
				return
			}
		}
	}

	// Handle vulnerability alerts
	if !plan.VulnerabilityAlerts.IsNull() && !plan.VulnerabilityAlerts.IsUnknown() {
		if plan.VulnerabilityAlerts.ValueBool() {
			_, err = r.client.V3Client().Repositories.EnableVulnerabilityAlerts(requestCtx, owner, repoName)
		} else {
			_, err = r.client.V3Client().Repositories.DisableVulnerabilityAlerts(requestCtx, owner, repoName)
		}
		if err != nil {
			diags.AddError(
				"Error Updating Repository Vulnerability Alerts",
				"Could not update repository vulnerability alerts, unexpected error: "+err.Error(),
			)
			return
		}
	}

	// Handle pages updates
	if !plan.Pages.IsNull() && !plan.Pages.IsUnknown() {
		var pagesModels []pagesModel
		diags.Append(plan.Pages.ElementsAs(ctx, &pagesModels, false)...)
		if diags.HasError() {
			return
		}

		if len(pagesModels) > 0 {
			pagesUpdate := r.expandPagesUpdate(ctx, pagesModels[0], diags)
			if diags.HasError() {
				return
			}

			if pagesUpdate != nil {
				existingPages, resp, err := r.client.V3Client().Repositories.GetPagesInfo(requestCtx, owner, repoName)
				if resp.StatusCode != http.StatusNotFound && err != nil {
					diags.AddError(
						"Error Reading Existing Repository Pages",
						"Could not read existing repository pages, unexpected error: "+err.Error(),
					)
					return
				}

				if existingPages == nil {
					// Enable pages
					pages := &github.Pages{
						Source:    pagesUpdate.Source,
						BuildType: pagesUpdate.BuildType,
					}
					_, _, err = r.client.V3Client().Repositories.EnablePages(requestCtx, owner, repoName, pages)
				} else {
					// Update pages
					_, err = r.client.V3Client().Repositories.UpdatePages(requestCtx, owner, repoName, pagesUpdate)
				}
				if err != nil {
					diags.AddError(
						"Error Updating Repository Pages",
						"Could not update repository pages, unexpected error: "+err.Error(),
					)
					return
				}
			}
		} else {
			// Disable pages
			_, err := r.client.V3Client().Repositories.DisablePages(requestCtx, owner, repoName)
			if err != nil {
				diags.AddError(
					"Error Disabling Repository Pages",
					"Could not disable repository pages, unexpected error: "+err.Error(),
				)
				return
			}
		}
	}
}

func (r *githubRepositoryResource) expandPagesUpdate(ctx context.Context, model pagesModel, diags diag.Diagnostics) *github.PagesUpdate {
	update := &github.PagesUpdate{}

	// Only set the github.PagesUpdate CNAME field if the value is a non-empty string.
	// Leaving the CNAME field unset will remove the custom domain.
	if !model.CNAME.IsNull() && !model.CNAME.IsUnknown() && model.CNAME.ValueString() != "" {
		update.CNAME = github.Ptr(model.CNAME.ValueString())
	}

	// Only set the github.PagesUpdate BuildType field if the value is a non-empty string.
	if !model.BuildType.IsNull() && !model.BuildType.IsUnknown() && model.BuildType.ValueString() != "" {
		update.BuildType = github.Ptr(model.BuildType.ValueString())
	}

	// To update the GitHub Pages source, the github.PagesUpdate Source field
	// must include the branch name and optionally the subdirectory /docs.
	// This is only necessary if the BuildType is "legacy".
	if update.BuildType == nil || *update.BuildType == "legacy" {
		if !model.Source.IsNull() && !model.Source.IsUnknown() {
			var sourceModels []pagesSourceModel
			diags.Append(model.Source.ElementsAs(ctx, &sourceModels, false)...)
			if diags.HasError() {
				return nil
			}

			if len(sourceModels) > 0 {
				sourceModel := sourceModels[0]
				sourceBranch := sourceModel.Branch.ValueString()
				sourcePath := ""
				if !sourceModel.Path.IsNull() && !sourceModel.Path.IsUnknown() && sourceModel.Path.ValueString() != "" {
					sourcePath = sourceModel.Path.ValueString()
				}
				update.Source = &github.PagesSource{Branch: &sourceBranch, Path: &sourcePath}
			}
		}
	}

	return update
}

func (r *githubRepositoryResource) flattenPages(ctx context.Context, pages *github.Pages, diags diag.Diagnostics) types.List {
	sourceObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"branch": types.StringType,
			"path":   types.StringType,
		},
	}
	pagesObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"source":     types.ListType{ElemType: sourceObjType},
			"build_type": types.StringType,
			"url":        types.StringType,
			"status":     types.StringType,
			"cname":      types.StringType,
			"custom_404": types.BoolType,
			"html_url":   types.StringType,
		},
	}

	if pages == nil {
		return types.ListNull(pagesObjType)
	}

	sourceObj := types.ObjectValueMust(
		sourceObjType.AttrTypes,
		map[string]attr.Value{
			"branch": types.StringValue(pages.GetSource().GetBranch()),
			"path":   types.StringValue(pages.GetSource().GetPath()),
		},
	)

	pagesObj := types.ObjectValueMust(
		pagesObjType.AttrTypes,
		map[string]attr.Value{
			"source":     types.ListValueMust(sourceObjType, []attr.Value{sourceObj}),
			"build_type": types.StringValue(pages.GetBuildType()),
			"url":        types.StringValue(pages.GetURL()),
			"status":     types.StringValue(pages.GetStatus()),
			"cname":      types.StringValue(pages.GetCNAME()),
			"custom_404": types.BoolValue(pages.GetCustom404()),
			"html_url":   types.StringValue(pages.GetHTMLURL()),
		},
	)

	return types.ListValueMust(pagesObjType, []attr.Value{pagesObj})
}

func (r *githubRepositoryResource) flattenSecurityAndAnalysis(ctx context.Context, securityAnalysis *github.SecurityAndAnalysis, diags diag.Diagnostics) types.List {
	securityFeatureObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"status": types.StringType,
		},
	}

	securityAnalysisObjType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"advanced_security":               types.ListType{ElemType: securityFeatureObjType},
			"secret_scanning":                 types.ListType{ElemType: securityFeatureObjType},
			"secret_scanning_push_protection": types.ListType{ElemType: securityFeatureObjType},
		},
	}

	if securityAnalysis == nil {
		return types.ListNull(securityAnalysisObjType)
	}

	securityMap := make(map[string]attr.Value)

	// Advanced Security
	advancedSecurity := securityAnalysis.GetAdvancedSecurity()
	if advancedSecurity != nil {
		advSecObj := types.ObjectValueMust(
			securityFeatureObjType.AttrTypes,
			map[string]attr.Value{
				"status": types.StringValue(advancedSecurity.GetStatus()),
			},
		)
		securityMap["advanced_security"] = types.ListValueMust(securityFeatureObjType, []attr.Value{advSecObj})
	} else {
		securityMap["advanced_security"] = types.ListNull(securityFeatureObjType)
	}

	// Secret Scanning
	secretScanning := securityAnalysis.GetSecretScanning()
	if secretScanning != nil {
		secretScanObj := types.ObjectValueMust(
			securityFeatureObjType.AttrTypes,
			map[string]attr.Value{
				"status": types.StringValue(secretScanning.GetStatus()),
			},
		)
		securityMap["secret_scanning"] = types.ListValueMust(securityFeatureObjType, []attr.Value{secretScanObj})
	} else {
		securityMap["secret_scanning"] = types.ListNull(securityFeatureObjType)
	}

	// Secret Scanning Push Protection
	secretScanningPP := securityAnalysis.GetSecretScanningPushProtection()
	if secretScanningPP != nil {
		secretScanPPObj := types.ObjectValueMust(
			securityFeatureObjType.AttrTypes,
			map[string]attr.Value{
				"status": types.StringValue(secretScanningPP.GetStatus()),
			},
		)
		securityMap["secret_scanning_push_protection"] = types.ListValueMust(securityFeatureObjType, []attr.Value{secretScanPPObj})
	} else {
		securityMap["secret_scanning_push_protection"] = types.ListNull(securityFeatureObjType)
	}

	securityObj := types.ObjectValueMust(
		securityAnalysisObjType.AttrTypes,
		securityMap,
	)

	return types.ListValueMust(securityAnalysisObjType, []attr.Value{securityObj})
}

// parseFullName parses full_name into owner and repository name
func (r *githubRepositoryResource) parseFullName(state *githubRepositoryResourceModel) (string, string, bool) {
	if state.FullName.IsNull() || state.FullName.IsUnknown() {
		return "", "", false
	}

	fullName := state.FullName.ValueString()
	if fullName == "" {
		return "", "", false
	}

	parts := strings.Split(fullName, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	return parts[0], parts[1], true
}
