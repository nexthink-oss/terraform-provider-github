package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubRepositoryFileResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryFileResource{}
	_ resource.ResourceWithImportState = &githubRepositoryFileResource{}
)

type githubRepositoryFileResource struct {
	client *Owner
}

type githubRepositoryFileResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	Repository                   types.String `tfsdk:"repository"`
	File                         types.String `tfsdk:"file"`
	Content                      types.String `tfsdk:"content"`
	Branch                       types.String `tfsdk:"branch"`
	Ref                          types.String `tfsdk:"ref"`
	CommitSha                    types.String `tfsdk:"commit_sha"`
	CommitMessage                types.String `tfsdk:"commit_message"`
	CommitAuthor                 types.String `tfsdk:"commit_author"`
	CommitEmail                  types.String `tfsdk:"commit_email"`
	Sha                          types.String `tfsdk:"sha"`
	OverwriteOnCreate            types.Bool   `tfsdk:"overwrite_on_create"`
	AutocreateBranch             types.Bool   `tfsdk:"autocreate_branch"`
	AutocreateBranchSourceBranch types.String `tfsdk:"autocreate_branch_source_branch"`
	AutocreateBranchSourceSha    types.String `tfsdk:"autocreate_branch_source_sha"`
}

func NewGithubRepositoryFileResource() resource.Resource {
	return &githubRepositoryFileResource{}
}

func (r *githubRepositoryFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_file"
}

func (r *githubRepositoryFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages files within a GitHub repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the repository file (repository/file).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The repository name",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"file": schema.StringAttribute{
				Description: "The file path to manage",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				Description: "The file's content",
				Required:    true,
			},
			"branch": schema.StringAttribute{
				Description: "The branch name, defaults to the repository's default branch",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ref": schema.StringAttribute{
				Description: "The name of the commit/branch/tag",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"commit_sha": schema.StringAttribute{
				Description: "The SHA of the commit that modified the file",
				Computed:    true,
			},
			"commit_message": schema.StringAttribute{
				Description: "The commit message when creating, updating or deleting the file",
				Optional:    true,
				Computed:    true,
			},
			"commit_author": schema.StringAttribute{
				Description: "The commit author name, defaults to the authenticated user's name. GitHub app users may omit author and email information so GitHub can verify commits as the GitHub App.",
				Optional:    true,
			},
			"commit_email": schema.StringAttribute{
				Description: "The commit author email address, defaults to the authenticated user's email address. GitHub app users may omit author and email information so GitHub can verify commits as the GitHub App.",
				Optional:    true,
			},
			"sha": schema.StringAttribute{
				Description: "The blob SHA of the file",
				Computed:    true,
			},
			"overwrite_on_create": schema.BoolAttribute{
				Description: "Enable overwriting existing files, defaults to \"false\"",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"autocreate_branch": schema.BoolAttribute{
				Description: "Automatically create the branch if it could not be found. Subsequent reads if the branch is deleted will occur from 'autocreate_branch_source_branch'",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"autocreate_branch_source_branch": schema.StringAttribute{
				Description: "The branch name to start from, if 'autocreate_branch' is set. Defaults to 'main'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("main"),
				PlanModifiers: []planmodifier.String{
					autoBranchSourceStringPlanModifier{},
				},
			},
			"autocreate_branch_source_sha": schema.StringAttribute{
				Description: "The commit hash to start from, if 'autocreate_branch' is set. Defaults to the tip of 'autocreate_branch_source_branch'. If provided, 'autocreate_branch_source_branch' is ignored.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					autoBranchSourceStringPlanModifier{},
				},
			},
		},
	}
}

func (r *githubRepositoryFileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *githubRepositoryFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubRepositoryFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repo := data.Repository.ValueString()
	file := data.File.ValueString()

	checkOpt := github.RepositoryContentGetOptions{}

	// Handle branch checking and autocreate logic
	if !data.Branch.IsNull() && !data.Branch.IsUnknown() {
		branch := data.Branch.ValueString()
		tflog.Debug(ctx, "using explicitly set branch", map[string]interface{}{
			"branch": branch,
		})

		if err := r.checkRepositoryBranchExists(ctx, client, owner, repo, branch); err != nil {
			if data.AutocreateBranch.ValueBool() {
				if err := r.autoCreateBranch(ctx, client, owner, repo, &data); err != nil {
					resp.Diagnostics.AddError(
						"Failed to Auto-Create Branch",
						fmt.Sprintf("Unable to auto-create branch %s: %s", branch, err.Error()),
					)
					return
				}
			} else {
				resp.Diagnostics.AddError(
					"Branch Not Found",
					fmt.Sprintf("Branch %s not found in repository %s/%s: %s", branch, owner, repo, err.Error()),
				)
				return
			}
		}
		checkOpt.Ref = branch
	}

	opts, err := r.buildRepositoryFileOptions(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			fmt.Sprintf("Unable to build repository file options: %s", err.Error()),
		)
		return
	}

	if opts.Message == nil {
		m := fmt.Sprintf("Add %s", file)
		opts.Message = &m
	}

	tflog.Debug(ctx, "checking if overwriting a repository file", map[string]interface{}{
		"owner":      owner,
		"repository": repo,
		"file":       file,
	})

	fileContent, _, httpResp, err := client.Repositories.GetContents(ctx, owner, repo, file, &checkOpt)
	if err != nil {
		if httpResp != nil {
			if httpResp.StatusCode != 404 {
				// 404 is a valid response if the file does not exist
				resp.Diagnostics.AddError(
					"Unable to Check File Existence",
					fmt.Sprintf("Unexpected error checking if file exists: %s", err.Error()),
				)
				return
			}
		} else {
			// Response should be non-nil
			resp.Diagnostics.AddError(
				"Unable to Check File Existence",
				fmt.Sprintf("No response received when checking if file exists: %s", err.Error()),
			)
			return
		}
	}

	if fileContent != nil {
		if data.OverwriteOnCreate.ValueBool() {
			// Overwrite existing file if requested by configuring the options for
			// `client.Repositories.CreateFile` to match the existing file's SHA
			opts.SHA = fileContent.SHA
		} else {
			// Error if overwriting a file is not requested
			resp.Diagnostics.AddError(
				"File Already Exists",
				"Refusing to overwrite existing file: configure `overwrite_on_create` to `true` to override",
			)
			return
		}
	}

	// Create a new or overwritten file
	create, _, err := client.Repositories.CreateFile(ctx, owner, repo, file, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Repository File",
			fmt.Sprintf("An unexpected error occurred when creating the repository file: %s", err.Error()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", repo, file))
	data.CommitSha = types.StringValue(create.GetSHA())

	tflog.Debug(ctx, "created GitHub repository file", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repo,
		"file":       file,
		"commit_sha": data.CommitSha.ValueString(),
	})

	// Read the created resource to populate all computed fields
	r.readGithubRepositoryFile(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubRepositoryFileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubRepositoryFile(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubRepositoryFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repo := data.Repository.ValueString()
	file := data.File.ValueString()

	// Handle branch checking and autocreate logic
	if !data.Branch.IsNull() && !data.Branch.IsUnknown() {
		branch := data.Branch.ValueString()
		tflog.Debug(ctx, "using explicitly set branch", map[string]interface{}{
			"branch": branch,
		})

		if err := r.checkRepositoryBranchExists(ctx, client, owner, repo, branch); err != nil {
			if data.AutocreateBranch.ValueBool() {
				if err := r.autoCreateBranch(ctx, client, owner, repo, &data); err != nil {
					resp.Diagnostics.AddError(
						"Failed to Auto-Create Branch",
						fmt.Sprintf("Unable to auto-create branch %s: %s", branch, err.Error()),
					)
					return
				}
			} else {
				resp.Diagnostics.AddError(
					"Branch Not Found",
					fmt.Sprintf("Branch %s not found in repository %s/%s: %s", branch, owner, repo, err.Error()),
				)
				return
			}
		}
	}

	opts, err := r.buildRepositoryFileOptions(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			fmt.Sprintf("Unable to build repository file options: %s", err.Error()),
		)
		return
	}

	if opts.Message != nil && *opts.Message == fmt.Sprintf("Add %s", file) {
		m := fmt.Sprintf("Update %s", file)
		opts.Message = &m
	}

	create, _, err := client.Repositories.CreateFile(ctx, owner, repo, file, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Repository File",
			fmt.Sprintf("An unexpected error occurred when updating the repository file: %s", err.Error()),
		)
		return
	}

	data.CommitSha = types.StringValue(create.GetSHA())

	tflog.Debug(ctx, "updated GitHub repository file", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repo,
		"file":       file,
		"commit_sha": data.CommitSha.ValueString(),
	})

	// Read the updated resource to populate all computed fields
	r.readGithubRepositoryFile(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubRepositoryFileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repo := data.Repository.ValueString()
	file := data.File.ValueString()

	var branch string

	message := fmt.Sprintf("Delete %s", file)
	if !data.CommitMessage.IsNull() && !data.CommitMessage.IsUnknown() {
		message = data.CommitMessage.ValueString()
	}

	sha := data.Sha.ValueString()
	opts := &github.RepositoryContentFileOptions{
		Message: &message,
		SHA:     &sha,
	}

	if !data.Branch.IsNull() && !data.Branch.IsUnknown() {
		branch = data.Branch.ValueString()
		tflog.Debug(ctx, "using explicitly set branch", map[string]interface{}{
			"branch": branch,
		})

		if err := r.checkRepositoryBranchExists(ctx, client, owner, repo, branch); err != nil {
			if data.AutocreateBranch.ValueBool() {
				if err := r.autoCreateBranch(ctx, client, owner, repo, &data); err != nil {
					resp.Diagnostics.AddError(
						"Failed to Auto-Create Branch",
						fmt.Sprintf("Unable to auto-create branch %s: %s", branch, err.Error()),
					)
					return
				}
			} else {
				resp.Diagnostics.AddError(
					"Branch Not Found",
					fmt.Sprintf("Branch %s not found in repository %s/%s: %s", branch, owner, repo, err.Error()),
				)
				return
			}
		}
		opts.Branch = &branch
	}

	_, _, err := client.Repositories.DeleteFile(ctx, owner, repo, file, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Repository File",
			fmt.Sprintf("An unexpected error occurred when deleting the repository file: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub repository file", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repo,
		"file":       file,
	})
}

func (r *githubRepositoryFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "<repository>/<file path>" or "<repository>/<file path>:<branch>"
	parts := strings.Split(req.ID, ":")

	if len(parts) > 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Invalid ID specified. Supplied ID must be written as <repository>/<file path> (when branch is \"main\") or <repository>/<file path>:<branch>. Got: %q", req.ID),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	repo, file := r.splitRepoFilePath(parts[0])

	// Test if a file exists in a repository.
	opts := &github.RepositoryContentGetOptions{}
	if len(parts) == 2 {
		opts.Ref = parts[1]
	}

	fc, _, _, err := client.Repositories.GetContents(ctx, owner, repo, file, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import Repository File",
			fmt.Sprintf("Unable to read repository file for import: %s", err.Error()),
		)
		return
	}
	if fc == nil {
		resp.Diagnostics.AddError(
			"File Not Found",
			fmt.Sprintf("File %s is not a file in repository %s/%s or repository is not readable", file, owner, repo),
		)
		return
	}

	data := &githubRepositoryFileResourceModel{
		ID:                types.StringValue(fmt.Sprintf("%s/%s", repo, file)),
		Repository:        types.StringValue(repo),
		File:              types.StringValue(file),
		OverwriteOnCreate: types.BoolValue(false),
	}

	if len(parts) == 2 {
		data.Branch = types.StringValue(parts[1])
	}

	r.readGithubRepositoryFile(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubRepositoryFileResource) buildRepositoryFileOptions(ctx context.Context, data *githubRepositoryFileResourceModel) (*github.RepositoryContentFileOptions, error) {
	opts := &github.RepositoryContentFileOptions{
		Content: []byte(data.Content.ValueString()),
	}

	if !data.Branch.IsNull() && !data.Branch.IsUnknown() {
		opts.Branch = github.Ptr(data.Branch.ValueString())
	}

	if !data.CommitMessage.IsNull() && !data.CommitMessage.IsUnknown() {
		opts.Message = github.Ptr(data.CommitMessage.ValueString())
	}

	if !data.Sha.IsNull() && !data.Sha.IsUnknown() {
		opts.SHA = github.Ptr(data.Sha.ValueString())
	}

	commitAuthor := data.CommitAuthor.ValueString()
	commitEmail := data.CommitEmail.ValueString()
	hasCommitAuthor := !data.CommitAuthor.IsNull() && !data.CommitAuthor.IsUnknown()
	hasCommitEmail := !data.CommitEmail.IsNull() && !data.CommitEmail.IsUnknown()

	if hasCommitAuthor && !hasCommitEmail {
		return nil, fmt.Errorf("cannot set commit_author without setting commit_email")
	}

	if hasCommitEmail && !hasCommitAuthor {
		return nil, fmt.Errorf("cannot set commit_email without setting commit_author")
	}

	if hasCommitAuthor && hasCommitEmail {
		opts.Author = &github.CommitAuthor{
			Name:  &commitAuthor,
			Email: &commitEmail,
		}
		opts.Committer = &github.CommitAuthor{
			Name:  &commitAuthor,
			Email: &commitEmail,
		}
	}

	return opts, nil
}

func (r *githubRepositoryFileResource) checkRepositoryBranchExists(ctx context.Context, client *github.Client, owner, repo, branch string) error {
	_, _, err := client.Repositories.GetBranch(ctx, owner, repo, branch, 2)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				return fmt.Errorf("branch %s not found in repository %s/%s or repository is not readable", branch, owner, repo)
			}
		}
		return err
	}
	return nil
}

func (r *githubRepositoryFileResource) autoCreateBranch(ctx context.Context, client *github.Client, owner, repo string, data *githubRepositoryFileResourceModel) error {
	branch := data.Branch.ValueString()
	branchRefName := "refs/heads/" + branch
	sourceBranchName := data.AutocreateBranchSourceBranch.ValueString()
	sourceBranchRefName := "refs/heads/" + sourceBranchName

	if data.AutocreateBranchSourceSha.IsNull() || data.AutocreateBranchSourceSha.IsUnknown() {
		ref, _, err := client.Git.GetRef(ctx, owner, repo, sourceBranchRefName)
		if err != nil {
			return fmt.Errorf("error querying GitHub branch reference %s/%s (%s): %s", owner, repo, sourceBranchRefName, err)
		}
		data.AutocreateBranchSourceSha = types.StringValue(*ref.Object.SHA)
	}

	sourceBranchSHA := data.AutocreateBranchSourceSha.ValueString()
	_, _, err := client.Git.CreateRef(ctx, owner, repo, &github.Reference{
		Ref:    &branchRefName,
		Object: &github.GitObject{SHA: &sourceBranchSHA},
	})
	return err
}

func (r *githubRepositoryFileResource) splitRepoFilePath(path string) (string, string) {
	parts := strings.Split(path, "/")
	return parts[0], strings.Join(parts[1:], "/")
}

func (r *githubRepositoryFileResource) getFileCommit(ctx context.Context, client *github.Client, owner, repo, file, branch string) (*github.RepositoryCommit, error) {
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

		listOpts := &github.ListOptions{}
		allFiles := []*github.CommitFile{}
		var rc *github.RepositoryCommit
		var resp *github.Response
		var err error
		for {
			rc, resp, err = client.Repositories.GetCommit(ctx, owner, repo, sha, listOpts)
			if err != nil {
				return nil, err
			}

			allFiles = append(allFiles, rc.Files...)

			if resp.NextPage == 0 {
				break
			}

			listOpts.Page = resp.NextPage
		}

		for _, f := range allFiles {
			if f.GetFilename() == file && f.GetStatus() != "removed" {
				return rc, nil
			}
		}
	}

	return nil, fmt.Errorf("cannot find file %s in repo %s/%s", file, owner, repo)
}

func (r *githubRepositoryFileResource) readGithubRepositoryFile(ctx context.Context, data *githubRepositoryFileResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	repo, file := r.splitRepoFilePath(data.ID.ValueString())

	opts := &github.RepositoryContentGetOptions{}

	if !data.Branch.IsNull() && !data.Branch.IsUnknown() {
		branch := data.Branch.ValueString()
		tflog.Debug(ctx, "using explicitly set branch", map[string]interface{}{
			"branch": branch,
		})

		if err := r.checkRepositoryBranchExists(ctx, client, owner, repo, branch); err != nil {
			if data.AutocreateBranch.ValueBool() {
				branch = data.AutocreateBranchSourceBranch.ValueString()
			} else {
				tflog.Info(ctx, "removing repository path from state because the branch no longer exists in GitHub", map[string]interface{}{
					"owner":      owner,
					"repository": repo,
					"file":       file,
				})
				data.ID = types.StringNull()
				return
			}
		}
		opts.Ref = branch
	}

	fc, _, _, err := client.Repositories.GetContents(ctx, owner, repo, file, opts)
	if err != nil {
		var errorResponse *github.ErrorResponse
		if errors.As(err, &errorResponse) && errorResponse.Response.StatusCode == http.StatusTooManyRequests {
			diags.AddError(
				"Rate Limited",
				fmt.Sprintf("GitHub API rate limit reached when reading repository file: %s", err.Error()),
			)
			return
		}

		if errors.As(err, &errorResponse) && errorResponse.Response.StatusCode == http.StatusNotFound {
			tflog.Info(ctx, "removing repository path from state because it no longer exists in GitHub", map[string]interface{}{
				"owner":      owner,
				"repository": repo,
				"file":       file,
			})
			data.ID = types.StringNull()
			return
		}

		diags.AddError(
			"Unable to Read Repository File",
			fmt.Sprintf("An unexpected error occurred when reading the repository file: %s", err.Error()),
		)
		return
	}

	if fc == nil {
		tflog.Info(ctx, "removing repository path from state because it no longer exists in GitHub", map[string]interface{}{
			"owner":      owner,
			"repository": repo,
			"file":       file,
		})
		data.ID = types.StringNull()
		return
	}

	content, err := fc.GetContent()
	if err != nil {
		diags.AddError(
			"Unable to Decode File Content",
			fmt.Sprintf("Unable to decode file content: %s", err.Error()),
		)
		return
	}

	data.Content = types.StringValue(content)
	data.Repository = types.StringValue(repo)
	data.File = types.StringValue(file)
	data.Sha = types.StringValue(fc.GetSHA())

	var commit *github.RepositoryCommit

	parsedUrl, err := url.Parse(fc.GetURL())
	if err != nil {
		diags.AddError(
			"Unable to Parse File URL",
			fmt.Sprintf("Unable to parse file URL: %s", err.Error()),
		)
		return
	}
	parsedQuery, err := url.ParseQuery(parsedUrl.RawQuery)
	if err != nil {
		diags.AddError(
			"Unable to Parse File URL Query",
			fmt.Sprintf("Unable to parse file URL query: %s", err.Error()),
		)
		return
	}
	ref := parsedQuery["ref"][0]
	data.Ref = types.StringValue(ref)

	// Use the SHA to lookup the commit info if we know it, otherwise loop through commits
	if !data.CommitSha.IsNull() && !data.CommitSha.IsUnknown() {
		sha := data.CommitSha.ValueString()
		tflog.Debug(ctx, "using known commit SHA", map[string]interface{}{
			"sha": sha,
		})
		commit, _, err = client.Repositories.GetCommit(ctx, owner, repo, sha, nil)
	} else {
		tflog.Debug(ctx, "commit SHA unknown for file, looking for commit", map[string]interface{}{
			"owner":      owner,
			"repository": repo,
			"file":       file,
		})
		commit, err = r.getFileCommit(ctx, client, owner, repo, file, ref)
		if commit != nil {
			tflog.Debug(ctx, "found file in commit", map[string]interface{}{
				"owner":      owner,
				"repository": repo,
				"file":       file,
				"commit_sha": commit.GetSHA(),
			})
		}
	}
	if err != nil {
		diags.AddError(
			"Unable to Find File Commit",
			fmt.Sprintf("Unable to find commit for file: %s", err.Error()),
		)
		return
	}

	data.CommitSha = types.StringValue(commit.GetSHA())

	commitAuthor := commit.Commit.GetCommitter().GetName()
	commitEmail := commit.Commit.GetCommitter().GetEmail()

	hasCommitAuthor := !data.CommitAuthor.IsNull() && !data.CommitAuthor.IsUnknown()
	hasCommitEmail := !data.CommitEmail.IsNull() && !data.CommitEmail.IsUnknown()

	// Read from state if author+email is set explicitly, and if it was not github signing it for you previously
	if commitAuthor != "GitHub" && commitEmail != "noreply@github.com" && hasCommitAuthor && hasCommitEmail {
		data.CommitAuthor = types.StringValue(commitAuthor)
		data.CommitEmail = types.StringValue(commitEmail)
	}
	data.CommitMessage = types.StringValue(commit.GetCommit().GetMessage())

	tflog.Debug(ctx, "successfully read GitHub repository file", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repo,
		"file":       file,
		"sha":        data.Sha.ValueString(),
		"commit_sha": data.CommitSha.ValueString(),
	})
}

// Custom plan modifier to suppress diffs for autocreate_branch_source_* when autocreate_branch is false
type autoBranchSourceStringPlanModifier struct{}

func (m autoBranchSourceStringPlanModifier) Description(ctx context.Context) string {
	return "Suppresses diff for autocreate branch source attributes when autocreate_branch is false"
}

func (m autoBranchSourceStringPlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m autoBranchSourceStringPlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Don't modify plan if we're in a destroy operation
	if req.Plan.Raw.IsNull() {
		return
	}

	// Get the autocreate_branch value from the plan
	var autocreateBranch types.Bool
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, req.Path.ParentPath().AtName("autocreate_branch"), &autocreateBranch)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If autocreate_branch is false (or unknown/null), suppress the diff by using the prior state value
	if autocreateBranch.IsNull() || autocreateBranch.IsUnknown() || !autocreateBranch.ValueBool() {
		resp.PlanValue = req.StateValue
	}
}
