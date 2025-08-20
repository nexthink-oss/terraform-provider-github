package framework

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ datasource.DataSource              = &githubRepositoryFileDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryFileDataSource{}
)

func NewGithubRepositoryFileDataSource() datasource.DataSource {
	return &githubRepositoryFileDataSource{}
}

type githubRepositoryFileDataSource struct {
	client *githubpkg.Owner
}

type githubRepositoryFileDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Repository    types.String `tfsdk:"repository"`
	File          types.String `tfsdk:"file"`
	Branch        types.String `tfsdk:"branch"`
	Ref           types.String `tfsdk:"ref"`
	Content       types.String `tfsdk:"content"`
	CommitSha     types.String `tfsdk:"commit_sha"`
	CommitMessage types.String `tfsdk:"commit_message"`
	CommitAuthor  types.String `tfsdk:"commit_author"`
	CommitEmail   types.String `tfsdk:"commit_email"`
	Sha           types.String `tfsdk:"sha"`
}

func (d *githubRepositoryFileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_file"
}

func (d *githubRepositoryFileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads files within a GitHub repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"repository": schema.StringAttribute{
				Description: "The repository name",
				Required:    true,
			},
			"file": schema.StringAttribute{
				Description: "The file path to manage",
				Required:    true,
			},
			"branch": schema.StringAttribute{
				Description: "The branch name, defaults to the repository's default branch",
				Optional:    true,
			},
			"ref": schema.StringAttribute{
				Description: "The name of the commit/branch/tag",
				Computed:    true,
			},
			"content": schema.StringAttribute{
				Description: "The file's content",
				Computed:    true,
			},
			"commit_sha": schema.StringAttribute{
				Description: "The SHA of the commit that modified the file",
				Computed:    true,
			},
			"commit_message": schema.StringAttribute{
				Description: "The commit message when creating or updating the file",
				Computed:    true,
			},
			"commit_author": schema.StringAttribute{
				Description: "The commit author name, defaults to the authenticated user's name",
				Computed:    true,
			},
			"commit_email": schema.StringAttribute{
				Description: "The commit author email address, defaults to the authenticated user's email address",
				Computed:    true,
			},
			"sha": schema.StringAttribute{
				Description: "The blob SHA of the file",
				Computed:    true,
			},
		},
	}
}

func (d *githubRepositoryFileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*githubpkg.Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *githubpkg.Owner, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *githubRepositoryFileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryFileDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.client.V3Client()
	owner := d.client.Name()
	repo := data.Repository.ValueString()

	// checking if repo has a slash in it, which means that full_name was passed
	// split and replace owner and repo
	parts := strings.Split(repo, "/")
	if len(parts) == 2 {
		tflog.Debug(ctx, "repo has a slash, extracting owner", map[string]interface{}{
			"repo": repo,
		})
		owner = parts[0]
		repo = parts[1]

		tflog.Debug(ctx, "extracted owner and repo", map[string]interface{}{
			"owner": owner,
			"repo":  repo,
		})
	}

	file := data.File.ValueString()

	opts := &github.RepositoryContentGetOptions{}
	if !data.Branch.IsNull() && !data.Branch.IsUnknown() {
		opts.Ref = data.Branch.ValueString()
	}

	fc, dc, _, err := client.Repositories.GetContents(ctx, owner, repo, file, opts)
	if err != nil {
		if err, ok := err.(*github.ErrorResponse); ok {
			if err.Response.StatusCode == http.StatusNotFound {
				tflog.Debug(ctx, "Missing GitHub repository file", map[string]interface{}{
					"owner": owner,
					"repo":  repo,
					"file":  file,
				})
				data.ID = types.StringValue("")
				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Unable to Get Repository File",
			fmt.Sprintf("Error getting repository file %s/%s/%s: %v", owner, repo, file, err),
		)
		return
	}

	data.Repository = types.StringValue(repo)
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", repo, file))
	data.File = types.StringValue(file)

	// If the repo is a directory, then there is nothing else we can include in
	// the schema.
	if dc != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	content, err := fc.GetContent()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get File Content",
			fmt.Sprintf("Error getting file content: %v", err),
		)
		return
	}

	data.Content = types.StringValue(content)
	data.Sha = types.StringValue(fc.GetSHA())

	parsedUrl, err := url.Parse(fc.GetURL())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse URL",
			fmt.Sprintf("Error parsing URL: %v", err),
		)
		return
	}
	parsedQuery, err := url.ParseQuery(parsedUrl.RawQuery)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Query",
			fmt.Sprintf("Error parsing query: %v", err),
		)
		return
	}
	if len(parsedQuery["ref"]) > 0 {
		ref := parsedQuery["ref"][0]
		data.Ref = types.StringValue(ref)

		tflog.Debug(ctx, "Data Source fetching commit info for repository file", map[string]interface{}{
			"owner": owner,
			"repo":  repo,
			"file":  file,
		})
		commit, err := d.getFileCommit(ctx, client, owner, repo, file, ref)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Get File Commit",
				fmt.Sprintf("Error getting file commit: %v", err),
			)
			return
		}

		tflog.Debug(ctx, "Found file", map[string]interface{}{
			"owner":      owner,
			"repo":       repo,
			"file":       file,
			"commit_sha": commit.GetSHA(),
		})

		data.CommitSha = types.StringValue(commit.GetSHA())
		data.CommitAuthor = types.StringValue(commit.Commit.GetCommitter().GetName())
		data.CommitEmail = types.StringValue(commit.Commit.GetCommitter().GetEmail())
		data.CommitMessage = types.StringValue(commit.GetCommit().GetMessage())
	} else {
		resp.Diagnostics.AddWarning(
			"Unable to set ref",
			"Unable to extract ref from URL query parameters",
		)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *githubRepositoryFileDataSource) getFileCommit(ctx context.Context, client *github.Client, owner, repo, file, branch string) (*github.RepositoryCommit, error) {
	opts := &github.CommitsListOptions{
		SHA:  branch,
		Path: file,
	}
	allCommits := []*github.RepositoryCommit{}
	for {
		commits, resp, err := client.Repositories.ListCommits(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}

		allCommits = append(allCommits, commits...)

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	for _, c := range allCommits {
		sha := c.GetSHA()

		// Skip merge commits
		if strings.Contains(c.Commit.GetMessage(), "Merge branch") {
			continue
		}

		opts := &github.ListOptions{}
		allFiles := []*github.CommitFile{}
		var rc *github.RepositoryCommit
		var resp *github.Response
		var err error
		for {
			rc, resp, err = client.Repositories.GetCommit(ctx, owner, repo, sha, opts)
			if err != nil {
				return nil, err
			}

			allFiles = append(allFiles, rc.Files...)

			if resp.NextPage == 0 {
				break
			}

			opts.Page = resp.NextPage
		}

		for _, f := range allFiles {
			if f.GetFilename() == file && f.GetStatus() != "removed" {
				return rc, nil
			}
		}
	}

	return nil, fmt.Errorf("cannot find file %s in repo %s/%s", file, owner, repo)
}
