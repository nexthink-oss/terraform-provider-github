package repository

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// RepositoryResourceModel describes the resource data model for github_repository.
type RepositoryResourceModel struct {
	// Resource identifier
	ID types.String `tfsdk:"id"`

	// Core attributes
	Name        types.String `tfsdk:"name"`
	FullName    types.String `tfsdk:"full_name"`
	Description types.String `tfsdk:"description"`
	HomepageURL types.String `tfsdk:"homepage_url"`

	// Visibility (private is deprecated)
	Private    types.Bool   `tfsdk:"private"`
	Visibility types.String `tfsdk:"visibility"`

	// Feature flags
	HasIssues      types.Bool `tfsdk:"has_issues"`
	HasDiscussions types.Bool `tfsdk:"has_discussions"`
	HasProjects    types.Bool `tfsdk:"has_projects"`
	HasWiki        types.Bool `tfsdk:"has_wiki"`
	HasDownloads   types.Bool `tfsdk:"has_downloads"`
	IsTemplate     types.Bool `tfsdk:"is_template"`

	// Merge settings
	AllowMergeCommit  types.Bool `tfsdk:"allow_merge_commit"`
	AllowSquashMerge  types.Bool `tfsdk:"allow_squash_merge"`
	AllowRebaseMerge  types.Bool `tfsdk:"allow_rebase_merge"`
	AllowAutoMerge    types.Bool `tfsdk:"allow_auto_merge"`
	AllowUpdateBranch types.Bool `tfsdk:"allow_update_branch"`

	// Merge commit configuration
	SquashMergeCommitTitle   types.String `tfsdk:"squash_merge_commit_title"`
	SquashMergeCommitMessage types.String `tfsdk:"squash_merge_commit_message"`
	MergeCommitTitle         types.String `tfsdk:"merge_commit_title"`
	MergeCommitMessage       types.String `tfsdk:"merge_commit_message"`
	DeleteBranchOnMerge      types.Bool   `tfsdk:"delete_branch_on_merge"`
	WebCommitSignoffRequired types.Bool   `tfsdk:"web_commit_signoff_required"`

	// Initialization settings
	AutoInit          types.Bool   `tfsdk:"auto_init"`
	DefaultBranch     types.String `tfsdk:"default_branch"`
	LicenseTemplate   types.String `tfsdk:"license_template"`
	GitignoreTemplate types.String `tfsdk:"gitignore_template"`

	// Archive settings
	Archived         types.Bool `tfsdk:"archived"`
	ArchiveOnDestroy types.Bool `tfsdk:"archive_on_destroy"`

	// Topics
	Topics types.Set `tfsdk:"topics"`

	// Security settings
	VulnerabilityAlerts                 types.Bool `tfsdk:"vulnerability_alerts"`
	IgnoreVulnerabilityAlertsDuringRead types.Bool `tfsdk:"ignore_vulnerability_alerts_during_read"`

	// Computed attributes
	HTMLURL         types.String `tfsdk:"html_url"`
	SSHCloneURL     types.String `tfsdk:"ssh_clone_url"`
	SVNURL          types.String `tfsdk:"svn_url"`
	GitCloneURL     types.String `tfsdk:"git_clone_url"`
	HTTPCloneURL    types.String `tfsdk:"http_clone_url"`
	Etag            types.String `tfsdk:"etag"`
	PrimaryLanguage types.String `tfsdk:"primary_language"`
	NodeID          types.String `tfsdk:"node_id"`
	RepoID          types.Int64  `tfsdk:"repo_id"`

	// Nested blocks (use types.List for ListNestedBlock with SizeAtMost(1), types.Set for SetNestedBlock)
	SecurityAndAnalysis types.List `tfsdk:"security_and_analysis"`
	Pages               types.List `tfsdk:"pages"`
	Template            types.List `tfsdk:"template"`

	// Custom properties (SetNestedBlock)
	CustomProperty            types.Set  `tfsdk:"custom_property"`
	ExclusiveCustomProperties types.Bool `tfsdk:"exclusive_custom_properties"`
	AllCustomProperties       types.Map  `tfsdk:"all_custom_properties"`
}

// SecurityAndAnalysisModel describes the security_and_analysis nested block.
type SecurityAndAnalysisModel struct {
	AdvancedSecurity             types.List `tfsdk:"advanced_security"`
	SecretScanning               types.List `tfsdk:"secret_scanning"`
	SecretScanningPushProtection types.List `tfsdk:"secret_scanning_push_protection"`
}

// AdvancedSecurityModel describes the advanced_security nested block.
type AdvancedSecurityModel struct {
	Status types.String `tfsdk:"status"`
}

// SecretScanningModel describes the secret_scanning nested block.
type SecretScanningModel struct {
	Status types.String `tfsdk:"status"`
}

// SecretScanningPushProtectionModel describes the secret_scanning_push_protection nested block.
type SecretScanningPushProtectionModel struct {
	Status types.String `tfsdk:"status"`
}

// PagesModel describes the pages nested block.
type PagesModel struct {
	Source    types.List   `tfsdk:"source"`
	BuildType types.String `tfsdk:"build_type"`
	CNAME     types.String `tfsdk:"cname"`

	// Computed attributes
	Custom404 types.Bool   `tfsdk:"custom_404"`
	HTMLURL   types.String `tfsdk:"html_url"`
	Status    types.String `tfsdk:"status"`
	URL       types.String `tfsdk:"url"`
}

// PagesSourceModel describes the pages.source nested block.
type PagesSourceModel struct {
	Branch types.String `tfsdk:"branch"`
	Path   types.String `tfsdk:"path"`
}

// TemplateModel describes the template nested block.
type TemplateModel struct {
	Owner              types.String `tfsdk:"owner"`
	Repository         types.String `tfsdk:"repository"`
	IncludeAllBranches types.Bool   `tfsdk:"include_all_branches"`
}

// CustomPropertyModel describes the custom_property nested block.
type CustomPropertyModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.List   `tfsdk:"value"`
}
