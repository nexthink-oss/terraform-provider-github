package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubRepositoryDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryDataSource{}
)

func NewGithubRepositoryDataSource() datasource.DataSource {
	return &githubRepositoryDataSource{}
}

type githubRepositoryDataSource struct {
	client *Owner
}

type githubRepositoryDataSourceModel struct {
	ID                       types.String `tfsdk:"id"`
	FullName                 types.String `tfsdk:"full_name"`
	Name                     types.String `tfsdk:"name"`
	Description              types.String `tfsdk:"description"`
	HomepageURL              types.String `tfsdk:"homepage_url"`
	Private                  types.Bool   `tfsdk:"private"`
	Visibility               types.String `tfsdk:"visibility"`
	HasIssues                types.Bool   `tfsdk:"has_issues"`
	HasDiscussions           types.Bool   `tfsdk:"has_discussions"`
	HasProjects              types.Bool   `tfsdk:"has_projects"`
	HasDownloads             types.Bool   `tfsdk:"has_downloads"`
	HasWiki                  types.Bool   `tfsdk:"has_wiki"`
	IsTemplate               types.Bool   `tfsdk:"is_template"`
	Fork                     types.Bool   `tfsdk:"fork"`
	AllowMergeCommit         types.Bool   `tfsdk:"allow_merge_commit"`
	AllowSquashMerge         types.Bool   `tfsdk:"allow_squash_merge"`
	AllowRebaseMerge         types.Bool   `tfsdk:"allow_rebase_merge"`
	AllowAutoMerge           types.Bool   `tfsdk:"allow_auto_merge"`
	AllowUpdateBranch        types.Bool   `tfsdk:"allow_update_branch"`
	SquashMergeCommitTitle   types.String `tfsdk:"squash_merge_commit_title"`
	SquashMergeCommitMessage types.String `tfsdk:"squash_merge_commit_message"`
	MergeCommitTitle         types.String `tfsdk:"merge_commit_title"`
	MergeCommitMessage       types.String `tfsdk:"merge_commit_message"`
	DefaultBranch            types.String `tfsdk:"default_branch"`
	PrimaryLanguage          types.String `tfsdk:"primary_language"`
	Archived                 types.Bool   `tfsdk:"archived"`
	RepositoryLicense        types.List   `tfsdk:"repository_license"`
	Pages                    types.List   `tfsdk:"pages"`
	Topics                   types.List   `tfsdk:"topics"`
	HTMLURL                  types.String `tfsdk:"html_url"`
	SSHCloneURL              types.String `tfsdk:"ssh_clone_url"`
	SVNURL                   types.String `tfsdk:"svn_url"`
	GitCloneURL              types.String `tfsdk:"git_clone_url"`
	HTTPCloneURL             types.String `tfsdk:"http_clone_url"`
	Template                 types.List   `tfsdk:"template"`
	NodeID                   types.String `tfsdk:"node_id"`
	RepoID                   types.Int64  `tfsdk:"repo_id"`
	DeleteBranchOnMerge      types.Bool   `tfsdk:"delete_branch_on_merge"`
}

type repositoryLicenseModel struct {
	Name        types.String `tfsdk:"name"`
	Path        types.String `tfsdk:"path"`
	License     types.List   `tfsdk:"license"`
	SHA         types.String `tfsdk:"sha"`
	Size        types.Int64  `tfsdk:"size"`
	URL         types.String `tfsdk:"url"`
	HTMLURL     types.String `tfsdk:"html_url"`
	GitURL      types.String `tfsdk:"git_url"`
	DownloadURL types.String `tfsdk:"download_url"`
	Type        types.String `tfsdk:"type"`
	Content     types.String `tfsdk:"content"`
	Encoding    types.String `tfsdk:"encoding"`
}

type licenseModel struct {
	Key            types.String `tfsdk:"key"`
	Name           types.String `tfsdk:"name"`
	URL            types.String `tfsdk:"url"`
	SPDXID         types.String `tfsdk:"spdx_id"`
	HTMLURL        types.String `tfsdk:"html_url"`
	Featured       types.Bool   `tfsdk:"featured"`
	Description    types.String `tfsdk:"description"`
	Implementation types.String `tfsdk:"implementation"`
	Permissions    types.Set    `tfsdk:"permissions"`
	Conditions     types.Set    `tfsdk:"conditions"`
	Limitations    types.Set    `tfsdk:"limitations"`
	Body           types.String `tfsdk:"body"`
}

type repositoryPagesModel struct {
	Source    types.List   `tfsdk:"source"`
	BuildType types.String `tfsdk:"build_type"`
	CNAME     types.String `tfsdk:"cname"`
	Custom404 types.Bool   `tfsdk:"custom_404"`
	HTMLURL   types.String `tfsdk:"html_url"`
	Status    types.String `tfsdk:"status"`
	URL       types.String `tfsdk:"url"`
}

type repositoryPagesSourceModel struct {
	Branch types.String `tfsdk:"branch"`
	Path   types.String `tfsdk:"path"`
}

type repositoryTemplateModel struct {
	Owner      types.String `tfsdk:"owner"`
	Repository types.String `tfsdk:"repository"`
}

func (d *githubRepositoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (d *githubRepositoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get details about a GitHub repository.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"full_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Full name of the repository (in `owner/name` format).",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the repository.",
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
				Computed:    true,
				Description: "Whether the repository is private.",
			},
			"visibility": schema.StringAttribute{
				Computed:    true,
				Description: "Can be 'public' or 'private'.",
			},
			"has_issues": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository has GitHub Issues enabled.",
			},
			"has_discussions": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository has GitHub Discussions enabled.",
			},
			"has_projects": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository has GitHub Projects enabled.",
			},
			"has_downloads": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository has GitHub Downloads enabled.",
			},
			"has_wiki": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository has GitHub Wiki enabled.",
			},
			"is_template": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository is a template.",
			},
			"fork": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository is a fork.",
			},
			"allow_merge_commit": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository allows merge commits.",
			},
			"allow_squash_merge": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository allows squash merges.",
			},
			"allow_rebase_merge": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository allows rebase merges.",
			},
			"allow_auto_merge": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository allows auto-merge for pull requests.",
			},
			"allow_update_branch": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository allows updating branch for pull requests.",
			},
			"squash_merge_commit_title": schema.StringAttribute{
				Computed:    true,
				Description: "The default value for a squash merge commit title.",
			},
			"squash_merge_commit_message": schema.StringAttribute{
				Computed:    true,
				Description: "The default value for a squash merge commit message.",
			},
			"merge_commit_title": schema.StringAttribute{
				Computed:    true,
				Description: "The default value for a merge commit title.",
			},
			"merge_commit_message": schema.StringAttribute{
				Computed:    true,
				Description: "The default value for a merge commit message.",
			},
			"default_branch": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the default branch of the repository.",
			},
			"primary_language": schema.StringAttribute{
				Computed:    true,
				Description: "The primary language of the repository.",
			},
			"archived": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository is archived.",
			},
			"html_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL to the repository on the web.",
			},
			"ssh_clone_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL that can be provided to `git clone` to clone the repository via SSH.",
			},
			"svn_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL that can be provided to `svn checkout` to check out the repository via GNU Subversion.",
			},
			"git_clone_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL that can be provided to `git clone` to clone the repository anonymously via the git protocol.",
			},
			"http_clone_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL that can be provided to `git clone` to clone the repository via HTTPS.",
			},
			"node_id": schema.StringAttribute{
				Computed:    true,
				Description: "GraphQL global node id for use with v4 API.",
			},
			"repo_id": schema.Int64Attribute{
				Computed:    true,
				Description: "GitHub ID for the repository.",
			},
			"delete_branch_on_merge": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the repository allows automatically deleting head branches when pull requests are merged.",
			},
			"repository_license": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The license of the repository.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "The license file name.",
						},
						"path": schema.StringAttribute{
							Computed:    true,
							Description: "The license file path.",
						},
						"sha": schema.StringAttribute{
							Computed:    true,
							Description: "The license file SHA.",
						},
						"size": schema.Int64Attribute{
							Computed:    true,
							Description: "The license file size in bytes.",
						},
						"url": schema.StringAttribute{
							Computed:    true,
							Description: "The license file URL.",
						},
						"html_url": schema.StringAttribute{
							Computed:    true,
							Description: "The license file HTML URL.",
						},
						"git_url": schema.StringAttribute{
							Computed:    true,
							Description: "The license file Git URL.",
						},
						"download_url": schema.StringAttribute{
							Computed:    true,
							Description: "The license file download URL.",
						},
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The license file type.",
						},
						"content": schema.StringAttribute{
							Computed:    true,
							Description: "The license file content (base64 encoded).",
						},
						"encoding": schema.StringAttribute{
							Computed:    true,
							Description: "The license file encoding.",
						},
						"license": schema.ListNestedAttribute{
							Computed:    true,
							Description: "The license information.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"key": schema.StringAttribute{
										Computed:    true,
										Description: "The license key.",
									},
									"name": schema.StringAttribute{
										Computed:    true,
										Description: "The license name.",
									},
									"url": schema.StringAttribute{
										Computed:    true,
										Description: "The license URL.",
									},
									"spdx_id": schema.StringAttribute{
										Computed:    true,
										Description: "The license SPDX ID.",
									},
									"html_url": schema.StringAttribute{
										Computed:    true,
										Description: "The license HTML URL.",
									},
									"featured": schema.BoolAttribute{
										Computed:    true,
										Description: "Whether the license is featured.",
									},
									"description": schema.StringAttribute{
										Computed:    true,
										Description: "The license description.",
									},
									"implementation": schema.StringAttribute{
										Computed:    true,
										Description: "The license implementation.",
									},
									"permissions": schema.SetAttribute{
										Computed:    true,
										ElementType: types.StringType,
										Description: "The license permissions.",
									},
									"conditions": schema.SetAttribute{
										Computed:    true,
										ElementType: types.StringType,
										Description: "The license conditions.",
									},
									"limitations": schema.SetAttribute{
										Computed:    true,
										ElementType: types.StringType,
										Description: "The license limitations.",
									},
									"body": schema.StringAttribute{
										Computed:    true,
										Description: "The license body.",
									},
								},
							},
						},
					},
				},
			},
			"pages": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The GitHub Pages configuration for the repository.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"build_type": schema.StringAttribute{
							Computed:    true,
							Description: "The GitHub Pages build type.",
						},
						"cname": schema.StringAttribute{
							Computed:    true,
							Description: "The custom domain for the repository.",
						},
						"custom_404": schema.BoolAttribute{
							Computed:    true,
							Description: "Whether the repository has a custom 404 page.",
						},
						"html_url": schema.StringAttribute{
							Computed:    true,
							Description: "The GitHub Pages URL.",
						},
						"status": schema.StringAttribute{
							Computed:    true,
							Description: "The GitHub Pages status.",
						},
						"url": schema.StringAttribute{
							Computed:    true,
							Description: "The GitHub Pages API URL.",
						},
						"source": schema.ListNestedAttribute{
							Computed:    true,
							Description: "The source configuration for GitHub Pages.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"branch": schema.StringAttribute{
										Computed:    true,
										Description: "The branch name.",
									},
									"path": schema.StringAttribute{
										Computed:    true,
										Description: "The path from which to publish.",
									},
								},
							},
						},
					},
				},
			},
			"topics": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "The list of topics of the repository.",
			},
			"template": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The template repository configuration.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"owner": schema.StringAttribute{
							Computed:    true,
							Description: "The template repository owner.",
						},
						"repository": schema.StringAttribute{
							Computed:    true,
							Description: "The template repository name.",
						},
					},
				},
			},
		},
	}
}

func (d *githubRepositoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *githubRepositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate ConflictsWith behavior for full_name and name
	hasFullName := !data.FullName.IsNull() && !data.FullName.IsUnknown()
	hasName := !data.Name.IsNull() && !data.Name.IsUnknown()

	if hasFullName && hasName {
		resp.Diagnostics.AddError(
			"Conflicting Configuration",
			"Cannot specify both 'full_name' and 'name' attributes. Please use only one.",
		)
		return
	}

	if !hasFullName && !hasName {
		resp.Diagnostics.AddError(
			"Missing Required Configuration",
			"Must specify either 'full_name' or 'name' attribute.",
		)
		return
	}

	var owner, repoName string

	if hasFullName {
		var err error
		owner, repoName, err = splitRepoFullName(data.FullName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Repository Full Name",
				fmt.Sprintf("Error parsing full_name: %s", err.Error()),
			)
			return
		}
	}

	if hasName {
		repoName = data.Name.ValueString()
		owner = d.client.Name()
	}

	if repoName == "" {
		resp.Diagnostics.AddError(
			"Missing Repository Information",
			"One of 'full_name' or 'name' must be provided",
		)
		return
	}

	if owner == "" {
		owner = d.client.Name()
	}

	tflog.Debug(ctx, "Reading GitHub repository", map[string]any{
		"owner": owner,
		"name":  repoName,
	})

	repo, _, err := d.client.V3Client().Repositories.Get(ctx, owner, repoName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				resp.Diagnostics.AddError(
					"Repository Not Found",
					fmt.Sprintf("GitHub repository %s/%s not found", owner, repoName),
				)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Repository",
			fmt.Sprintf("Error reading repository %s/%s: %s", owner, repoName, err.Error()),
		)
		return
	}

	// Set basic attributes
	data.ID = types.StringValue(repoName)
	data.Name = types.StringValue(repo.GetName())
	data.FullName = types.StringValue(repo.GetFullName())
	data.Description = types.StringValue(repo.GetDescription())
	data.HomepageURL = types.StringValue(repo.GetHomepage())
	data.Private = types.BoolValue(repo.GetPrivate())
	data.Visibility = types.StringValue(repo.GetVisibility())
	data.HasIssues = types.BoolValue(repo.GetHasIssues())
	data.HasDiscussions = types.BoolValue(repo.GetHasDiscussions())
	data.HasWiki = types.BoolValue(repo.GetHasWiki())
	data.IsTemplate = types.BoolValue(repo.GetIsTemplate())
	data.Fork = types.BoolValue(repo.GetFork())
	data.AllowMergeCommit = types.BoolValue(repo.GetAllowMergeCommit())
	data.AllowSquashMerge = types.BoolValue(repo.GetAllowSquashMerge())
	data.AllowRebaseMerge = types.BoolValue(repo.GetAllowRebaseMerge())
	data.AllowAutoMerge = types.BoolValue(repo.GetAllowAutoMerge())
	data.SquashMergeCommitTitle = types.StringValue(repo.GetSquashMergeCommitTitle())
	data.SquashMergeCommitMessage = types.StringValue(repo.GetSquashMergeCommitMessage())
	data.MergeCommitTitle = types.StringValue(repo.GetMergeCommitTitle())
	data.MergeCommitMessage = types.StringValue(repo.GetMergeCommitMessage())
	data.HasDownloads = types.BoolValue(repo.GetHasDownloads())
	data.DefaultBranch = types.StringValue(repo.GetDefaultBranch())
	data.PrimaryLanguage = types.StringValue(repo.GetLanguage())
	data.HTMLURL = types.StringValue(repo.GetHTMLURL())
	data.SSHCloneURL = types.StringValue(repo.GetSSHURL())
	data.SVNURL = types.StringValue(repo.GetSVNURL())
	data.GitCloneURL = types.StringValue(repo.GetGitURL())
	data.HTTPCloneURL = types.StringValue(repo.GetCloneURL())
	data.Archived = types.BoolValue(repo.GetArchived())
	data.NodeID = types.StringValue(repo.GetNodeID())
	data.RepoID = types.Int64Value(repo.GetID())
	data.HasProjects = types.BoolValue(repo.GetHasProjects())
	data.DeleteBranchOnMerge = types.BoolValue(repo.GetDeleteBranchOnMerge())
	data.AllowUpdateBranch = types.BoolValue(repo.GetAllowUpdateBranch())

	// Handle GitHub Pages
	if repo.GetHasPages() {
		pages, _, err := d.client.V3Client().Repositories.GetPagesInfo(ctx, owner, repoName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read GitHub Pages Information",
				fmt.Sprintf("Error reading pages info for repository %s/%s: %s", owner, repoName, err.Error()),
			)
			return
		}
		data.Pages = d.flattenPages(ctx, pages)
	} else {
		data.Pages = d.flattenPages(ctx, nil)
	}

	// Handle repository license
	if repo.License != nil {
		repositoryLicense, _, err := d.client.V3Client().Repositories.License(ctx, owner, repoName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read Repository License",
				fmt.Sprintf("Error reading license for repository %s/%s: %s", owner, repoName, err.Error()),
			)
			return
		}
		data.RepositoryLicense = d.flattenRepositoryLicense(ctx, repositoryLicense)
	} else {
		data.RepositoryLicense = d.flattenRepositoryLicense(ctx, nil)
	}

	// Handle template repository
	if repo.TemplateRepository != nil {
		templateValue := []repositoryTemplateModel{
			{
				Owner:      types.StringValue(repo.TemplateRepository.Owner.GetLogin()),
				Repository: types.StringValue(repo.TemplateRepository.GetName()),
			},
		}
		templateListValue, diag := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"owner":      types.StringType,
				"repository": types.StringType,
			},
		}, templateValue)
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Template = templateListValue
	} else {
		emptyTemplateList, diag := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"owner":      types.StringType,
				"repository": types.StringType,
			},
		}, []repositoryTemplateModel{})
		resp.Diagnostics.Append(diag...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Template = emptyTemplateList
	}

	// Handle topics
	topicsListValue, diag := types.ListValueFrom(ctx, types.StringType, repo.Topics)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Topics = topicsListValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Helper function to split full repository name into owner and name
func splitRepoFullName(fullName string) (string, string, error) {
	parts := strings.Split(fullName, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected full name format (%q), expected owner/repo_name", fullName)
	}
	return parts[0], parts[1], nil
}

// Helper function to flatten GitHub Pages information
func (d *githubRepositoryDataSource) flattenPages(ctx context.Context, pages *github.Pages) types.List {
	if pages == nil {
		emptyList, _ := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"source":     types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"branch": types.StringType, "path": types.StringType}}},
				"build_type": types.StringType,
				"cname":      types.StringType,
				"custom_404": types.BoolType,
				"html_url":   types.StringType,
				"status":     types.StringType,
				"url":        types.StringType,
			},
		}, []repositoryPagesModel{})
		return emptyList
	}

	sourceValue := []repositoryPagesSourceModel{
		{
			Branch: types.StringValue(pages.GetSource().GetBranch()),
			Path:   types.StringValue(pages.GetSource().GetPath()),
		},
	}
	sourceListValue, _ := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"branch": types.StringType,
			"path":   types.StringType,
		},
	}, sourceValue)

	pagesValue := []repositoryPagesModel{
		{
			Source:    sourceListValue,
			BuildType: types.StringValue(pages.GetBuildType()),
			CNAME:     types.StringValue(pages.GetCNAME()),
			Custom404: types.BoolValue(pages.GetCustom404()),
			HTMLURL:   types.StringValue(pages.GetHTMLURL()),
			Status:    types.StringValue(pages.GetStatus()),
			URL:       types.StringValue(pages.GetURL()),
		},
	}

	pagesListValue, _ := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"source":     types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"branch": types.StringType, "path": types.StringType}}},
			"build_type": types.StringType,
			"cname":      types.StringType,
			"custom_404": types.BoolType,
			"html_url":   types.StringType,
			"status":     types.StringType,
			"url":        types.StringType,
		},
	}, pagesValue)

	return pagesListValue
}

// Helper function to flatten repository license information
func (d *githubRepositoryDataSource) flattenRepositoryLicense(ctx context.Context, repositoryLicense *github.RepositoryLicense) types.List {
	if repositoryLicense == nil {
		emptyList, _ := types.ListValueFrom(ctx, types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":         types.StringType,
				"path":         types.StringType,
				"license":      types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"key": types.StringType, "name": types.StringType, "url": types.StringType, "spdx_id": types.StringType, "html_url": types.StringType, "featured": types.BoolType, "description": types.StringType, "implementation": types.StringType, "permissions": types.SetType{ElemType: types.StringType}, "conditions": types.SetType{ElemType: types.StringType}, "limitations": types.SetType{ElemType: types.StringType}, "body": types.StringType}}},
				"sha":          types.StringType,
				"size":         types.Int64Type,
				"url":          types.StringType,
				"html_url":     types.StringType,
				"git_url":      types.StringType,
				"download_url": types.StringType,
				"type":         types.StringType,
				"content":      types.StringType,
				"encoding":     types.StringType,
			},
		}, []repositoryLicenseModel{})
		return emptyList
	}

	// Handle license permissions, conditions, and limitations
	var permissions, conditions, limitations []string
	if repositoryLicense.License != nil {
		if repositoryLicense.License.Permissions != nil {
			permissions = *repositoryLicense.License.Permissions
		}
		if repositoryLicense.License.Conditions != nil {
			conditions = *repositoryLicense.License.Conditions
		}
		if repositoryLicense.License.Limitations != nil {
			limitations = *repositoryLicense.License.Limitations
		}
	}

	permissionsSetValue, _ := types.SetValueFrom(ctx, types.StringType, permissions)
	conditionsSetValue, _ := types.SetValueFrom(ctx, types.StringType, conditions)
	limitationsSetValue, _ := types.SetValueFrom(ctx, types.StringType, limitations)

	licenseValue := []licenseModel{
		{
			Key:            types.StringValue(repositoryLicense.GetLicense().GetKey()),
			Name:           types.StringValue(repositoryLicense.GetLicense().GetName()),
			URL:            types.StringValue(repositoryLicense.GetLicense().GetURL()),
			SPDXID:         types.StringValue(repositoryLicense.GetLicense().GetSPDXID()),
			HTMLURL:        types.StringValue(repositoryLicense.GetLicense().GetHTMLURL()),
			Featured:       types.BoolValue(repositoryLicense.GetLicense().GetFeatured()),
			Description:    types.StringValue(repositoryLicense.GetLicense().GetDescription()),
			Implementation: types.StringValue(repositoryLicense.GetLicense().GetImplementation()),
			Permissions:    permissionsSetValue,
			Conditions:     conditionsSetValue,
			Limitations:    limitationsSetValue,
			Body:           types.StringValue(repositoryLicense.GetLicense().GetBody()),
		},
	}

	licenseListValue, _ := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"key":            types.StringType,
			"name":           types.StringType,
			"url":            types.StringType,
			"spdx_id":        types.StringType,
			"html_url":       types.StringType,
			"featured":       types.BoolType,
			"description":    types.StringType,
			"implementation": types.StringType,
			"permissions":    types.SetType{ElemType: types.StringType},
			"conditions":     types.SetType{ElemType: types.StringType},
			"limitations":    types.SetType{ElemType: types.StringType},
			"body":           types.StringType,
		},
	}, licenseValue)

	repositoryLicenseValue := []repositoryLicenseModel{
		{
			Name:        types.StringValue(repositoryLicense.GetName()),
			Path:        types.StringValue(repositoryLicense.GetPath()),
			License:     licenseListValue,
			SHA:         types.StringValue(repositoryLicense.GetSHA()),
			Size:        types.Int64Value(int64(repositoryLicense.GetSize())),
			URL:         types.StringValue(repositoryLicense.GetURL()),
			HTMLURL:     types.StringValue(repositoryLicense.GetHTMLURL()),
			GitURL:      types.StringValue(repositoryLicense.GetGitURL()),
			DownloadURL: types.StringValue(repositoryLicense.GetDownloadURL()),
			Type:        types.StringValue(repositoryLicense.GetType()),
			Content:     types.StringValue(repositoryLicense.GetContent()),
			Encoding:    types.StringValue(repositoryLicense.GetEncoding()),
		},
	}

	repositoryLicenseListValue, _ := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":         types.StringType,
			"path":         types.StringType,
			"license":      types.ListType{ElemType: types.ObjectType{AttrTypes: map[string]attr.Type{"key": types.StringType, "name": types.StringType, "url": types.StringType, "spdx_id": types.StringType, "html_url": types.StringType, "featured": types.BoolType, "description": types.StringType, "implementation": types.StringType, "permissions": types.SetType{ElemType: types.StringType}, "conditions": types.SetType{ElemType: types.StringType}, "limitations": types.SetType{ElemType: types.StringType}, "body": types.StringType}}},
			"sha":          types.StringType,
			"size":         types.Int64Type,
			"url":          types.StringType,
			"html_url":     types.StringType,
			"git_url":      types.StringType,
			"download_url": types.StringType,
			"type":         types.StringType,
			"content":      types.StringType,
			"encoding":     types.StringType,
		},
	}, repositoryLicenseValue)

	return repositoryLicenseListValue
}
