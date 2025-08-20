package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ resource.Resource                = &githubBranchResource{}
	_ resource.ResourceWithConfigure   = &githubBranchResource{}
	_ resource.ResourceWithImportState = &githubBranchResource{}
)

type githubBranchResource struct {
	client *Owner
}

type githubBranchResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Repository   types.String `tfsdk:"repository"`
	Branch       types.String `tfsdk:"branch"`
	SourceBranch types.String `tfsdk:"source_branch"`
	SourceSha    types.String `tfsdk:"source_sha"`
	Etag         types.String `tfsdk:"etag"`
	Ref          types.String `tfsdk:"ref"`
	Sha          types.String `tfsdk:"sha"`
}

func NewGithubBranchResource() resource.Resource {
	return &githubBranchResource{}
}

func (r *githubBranchResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch"
}

func (r *githubBranchResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages branches within GitHub repositories.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The GitHub repository name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"branch": schema.StringAttribute{
				Description: "The repository branch to create.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_branch": schema.StringAttribute{
				Description: "The branch name to start from. Defaults to 'main'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("main"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"source_sha": schema.StringAttribute{
				Description: "The commit hash to start from. Defaults to the tip of 'source_branch'. If provided, 'source_branch' is ignored.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"etag": schema.StringAttribute{
				Description: "An etag representing the Branch object.",
				Computed:    true,
			},
			"ref": schema.StringAttribute{
				Description: "A string representing a branch reference, in the form of 'refs/heads/<branch>'.",
				Computed:    true,
			},
			"sha": schema.StringAttribute{
				Description: "A string storing the reference's HEAD commit's SHA1.",
				Computed:    true,
			},
		},
	}
}

func (r *githubBranchResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubBranchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubBranchResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := data.Repository.ValueString()
	branchName := data.Branch.ValueString()
	branchRefName := "refs/heads/" + branchName
	sourceBranchName := data.SourceBranch.ValueString()
	sourceBranchRefName := "refs/heads/" + sourceBranchName

	// Determine source SHA
	var sourceSha string
	if !data.SourceSha.IsNull() && data.SourceSha.ValueString() != "" {
		sourceSha = data.SourceSha.ValueString()
	} else {
		// Get SHA from source branch
		ref, _, err := r.client.V3Client().Git.GetRef(ctx, owner, repoName, sourceBranchRefName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Get Source Branch Reference",
				fmt.Sprintf("An unexpected error occurred when querying GitHub branch reference %s/%s (%s): %s",
					owner, repoName, sourceBranchRefName, err.Error()),
			)
			return
		}
		sourceSha = *ref.Object.SHA
		data.SourceSha = types.StringValue(sourceSha)
	}

	// Create the branch reference
	_, _, err := r.client.V3Client().Git.CreateRef(ctx, owner, repoName, &github.Reference{
		Ref:    &branchRefName,
		Object: &github.GitObject{SHA: &sourceSha},
	})

	// If the branch already exists, rather than erroring out just continue on to importing the branch
	// This avoids the case where a repo with gitignore_template and branch are being created at the same time crashing terraform
	if err != nil && !strings.HasSuffix(err.Error(), "422 Reference already exists []") {
		resp.Diagnostics.AddError(
			"Unable to Create Branch Reference",
			fmt.Sprintf("An unexpected error occurred when creating GitHub branch reference %s/%s (%s): %s",
				owner, repoName, branchRefName, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(r.buildTwoPartID(repoName, branchName))

	tflog.Debug(ctx, "created GitHub branch", map[string]interface{}{
		"repository": data.Repository.ValueString(),
		"branch":     data.Branch.ValueString(),
		"source_sha": data.SourceSha.ValueString(),
	})

	// Read the created resource to populate all computed fields
	r.readGithubBranch(ctx, &data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubBranchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubBranchResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubBranch(ctx, &data, &resp.Diagnostics, true)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubBranchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This resource doesn't support updates - all attributes require replacement
	resp.Diagnostics.AddError(
		"Resource Update Not Supported",
		"The github_branch resource does not support updates. All changes require resource replacement.",
	)
}

func (r *githubBranchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubBranchResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName, branchName, err := r.parseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Resource ID",
			fmt.Sprintf("Unable to parse resource ID: %s", err.Error()),
		)
		return
	}
	branchRefName := "refs/heads/" + branchName

	_, err = r.client.V3Client().Git.DeleteRef(ctx, owner, repoName, branchRefName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Branch Reference",
			fmt.Sprintf("An unexpected error occurred when deleting GitHub branch reference %s/%s (%s): %s",
				owner, repoName, branchRefName, err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub branch", map[string]interface{}{
		"repository": repoName,
		"branch":     branchName,
	})
}

func (r *githubBranchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	repoName, branchName, err := r.parseTwoPartID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse import ID: %s", err.Error()),
		)
		return
	}

	data := &githubBranchResourceModel{
		ID:         types.StringValue(r.buildTwoPartID(repoName, branchName)),
		Repository: types.StringValue(repoName),
		Branch:     types.StringValue(branchName),
	}

	// Check if import ID includes source branch (format: repo:branch:source_branch)
	parts := strings.SplitN(req.ID, ":", 3)
	sourceBranch := "main" // default
	if len(parts) == 3 {
		sourceBranch = parts[2]
	}
	data.SourceBranch = types.StringValue(sourceBranch)

	r.readGithubBranch(ctx, data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if the branch was found
	if data.Repository.IsNull() {
		resp.Diagnostics.AddError(
			"Branch Not Found",
			fmt.Sprintf("Repository %s does not have a branch named %s", repoName, branchName),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubBranchResource) buildTwoPartID(a, b string) string {
	return fmt.Sprintf("%s:%s", a, b)
}

func (r *githubBranchResource) parseTwoPartID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Unexpected ID format (%q), expected repository:branch", id)
	}

	return parts[0], parts[1], nil
}

func (r *githubBranchResource) readGithubBranch(ctx context.Context, data *githubBranchResourceModel, diags *diag.Diagnostics, useEtag bool) {
	owner := r.client.Name()
	repoName, branchName, err := r.parseTwoPartID(data.ID.ValueString())
	if err != nil {
		diags.AddError(
			"Invalid Resource ID",
			fmt.Sprintf("Unable to parse resource ID: %s", err.Error()),
		)
		return
	}
	branchRefName := "refs/heads/" + branchName

	// Set up context with ETag if this is not a new resource and useEtag is true
	reqCtx := ctx
	if useEtag && !data.Etag.IsNull() && data.Etag.ValueString() != "" {
		reqCtx = context.WithValue(ctx, CtxEtag, data.Etag.ValueString())
	}
	reqCtx = context.WithValue(reqCtx, CtxId, data.ID.ValueString())

	ref, resp, err := r.client.V3Client().Git.GetRef(reqCtx, owner, repoName, branchRefName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				// Resource hasn't changed, keep current state
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "GitHub branch not found, removing from state", map[string]interface{}{
					"repository": repoName,
					"branch":     branchName,
				})
				data.Repository = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read GitHub Branch Reference",
			fmt.Sprintf("An unexpected error occurred when reading the GitHub branch reference %s/%s (%s): %s",
				owner, repoName, branchRefName, err.Error()),
		)
		return
	}

	data.ID = types.StringValue(r.buildTwoPartID(repoName, branchName))
	data.Etag = types.StringValue(resp.Header.Get("ETag"))
	data.Repository = types.StringValue(repoName)
	data.Branch = types.StringValue(branchName)
	data.Ref = types.StringValue(*ref.Ref)
	data.Sha = types.StringValue(*ref.Object.SHA)

	tflog.Debug(ctx, "successfully read GitHub branch", map[string]interface{}{
		"repository": data.Repository.ValueString(),
		"branch":     data.Branch.ValueString(),
		"sha":        data.Sha.ValueString(),
	})
}
