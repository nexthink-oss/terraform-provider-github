package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ resource.Resource                = &githubBranchDefaultResource{}
	_ resource.ResourceWithConfigure   = &githubBranchDefaultResource{}
	_ resource.ResourceWithImportState = &githubBranchDefaultResource{}
)

type githubBranchDefaultResource struct {
	client *Owner
}

type githubBranchDefaultResourceModel struct {
	Branch     types.String `tfsdk:"branch"`
	Repository types.String `tfsdk:"repository"`
	Rename     types.Bool   `tfsdk:"rename"`
	Etag       types.String `tfsdk:"etag"`
}

func NewGithubBranchDefaultResource() resource.Resource {
	return &githubBranchDefaultResource{}
}

func (r *githubBranchDefaultResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_default"
}

func (r *githubBranchDefaultResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub branch default for a given repository.",
		Attributes: map[string]schema.Attribute{
			"branch": schema.StringAttribute{
				Description: "The branch (e.g. 'main').",
				Required:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The GitHub repository.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rename": schema.BoolAttribute{
				Description: "Indicate if it should rename the branch rather than use an existing branch. Defaults to 'false'.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"etag": schema.StringAttribute{
				Description: "The ETag of the branch default.",
				Computed:    true,
			},
		},
	}
}

func (r *githubBranchDefaultResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubBranchDefaultResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubBranchDefaultResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := data.Repository.ValueString()
	defaultBranch := data.Branch.ValueString()
	rename := data.Rename.ValueBool()

	if rename {
		repository, _, err := r.client.V3Client().Repositories.Get(ctx, owner, repoName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Get Repository",
				"An unexpected error occurred when getting the repository to rename the branch. "+
					"GitHub Client Error: "+err.Error(),
			)
			return
		}

		if _, _, err := r.client.V3Client().Repositories.RenameBranch(ctx, owner, repoName, *repository.DefaultBranch, defaultBranch); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Rename Branch",
				"An unexpected error occurred when renaming the branch. "+
					"GitHub Client Error: "+err.Error(),
			)
			return
		}
	} else {
		repository := &github.Repository{
			DefaultBranch: &defaultBranch,
		}

		if _, _, err := r.client.V3Client().Repositories.Edit(ctx, owner, repoName, repository); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Set Default Branch",
				"An unexpected error occurred when setting the default branch. "+
					"GitHub Client Error: "+err.Error(),
			)
			return
		}
	}

	tflog.Debug(ctx, "created GitHub branch default", map[string]interface{}{
		"repository": data.Repository.ValueString(),
		"branch":     data.Branch.ValueString(),
		"rename":     data.Rename.ValueBool(),
	})

	// Read the created resource to populate all computed fields
	r.readGithubBranchDefault(ctx, &data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubBranchDefaultResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubBranchDefaultResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubBranchDefault(ctx, &data, &resp.Diagnostics, true)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubBranchDefaultResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubBranchDefaultResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := data.Repository.ValueString()
	defaultBranch := data.Branch.ValueString()
	rename := data.Rename.ValueBool()

	if rename {
		repository, _, err := r.client.V3Client().Repositories.Get(ctx, owner, repoName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Get Repository",
				"An unexpected error occurred when getting the repository to rename the branch. "+
					"GitHub Client Error: "+err.Error(),
			)
			return
		}

		if _, _, err := r.client.V3Client().Repositories.RenameBranch(ctx, owner, repoName, *repository.DefaultBranch, defaultBranch); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Rename Branch",
				"An unexpected error occurred when renaming the branch. "+
					"GitHub Client Error: "+err.Error(),
			)
			return
		}
	} else {
		repository := &github.Repository{
			DefaultBranch: &defaultBranch,
		}

		if _, _, err := r.client.V3Client().Repositories.Edit(ctx, owner, repoName, repository); err != nil {
			resp.Diagnostics.AddError(
				"Unable to Update Default Branch",
				"An unexpected error occurred when updating the default branch. "+
					"GitHub Client Error: "+err.Error(),
			)
			return
		}
	}

	tflog.Debug(ctx, "updated GitHub branch default", map[string]interface{}{
		"repository": data.Repository.ValueString(),
		"branch":     data.Branch.ValueString(),
		"rename":     data.Rename.ValueBool(),
	})

	// Read the updated resource to populate all computed fields
	r.readGithubBranchDefault(ctx, &data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubBranchDefaultResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubBranchDefaultResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := r.client.Name()
	repoName := data.Repository.ValueString()

	repository := &github.Repository{
		DefaultBranch: nil,
	}

	_, _, err := r.client.V3Client().Repositories.Edit(ctx, owner, repoName, repository)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Default Branch",
			"An unexpected error occurred when deleting the default branch setting. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub branch default", map[string]interface{}{
		"repository": data.Repository.ValueString(),
	})
}

func (r *githubBranchDefaultResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data := &githubBranchDefaultResourceModel{
		Repository: types.StringValue(req.ID),
	}

	r.readGithubBranchDefault(ctx, data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *githubBranchDefaultResource) readGithubBranchDefault(ctx context.Context, data *githubBranchDefaultResourceModel, diags *diag.Diagnostics, useEtag bool) {
	owner := r.client.Name()
	repoName := data.Repository.ValueString()

	// Set up context with ETag if this is not a new resource and useEtag is true
	reqCtx := ctx
	if useEtag && !data.Etag.IsNull() && data.Etag.ValueString() != "" {
		reqCtx = context.WithValue(ctx, CtxEtag, data.Etag.ValueString())
	}
	reqCtx = context.WithValue(reqCtx, CtxId, repoName)

	repository, resp, err := r.client.V3Client().Repositories.Get(reqCtx, owner, repoName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				// Resource hasn't changed, keep current state
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "GitHub repository not found, removing branch default from state", map[string]interface{}{
					"repository": repoName,
				})
				data.Repository = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read GitHub Repository",
			"An unexpected error occurred when reading the GitHub repository for branch default. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	if repository.DefaultBranch == nil {
		tflog.Info(ctx, "GitHub repository has no default branch, removing from state", map[string]interface{}{
			"repository": repoName,
		})
		data.Repository = types.StringNull()
		return
	}

	data.Etag = types.StringValue(resp.Header.Get("ETag"))
	data.Branch = types.StringValue(*repository.DefaultBranch)
	data.Repository = types.StringValue(*repository.Name)

	tflog.Debug(ctx, "successfully read GitHub branch default", map[string]interface{}{
		"repository": data.Repository.ValueString(),
		"branch":     data.Branch.ValueString(),
	})
}
