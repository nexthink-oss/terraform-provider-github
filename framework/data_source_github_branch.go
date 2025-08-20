package framework

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/google/go-github/v74/github"
	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ datasource.DataSource              = &githubBranchDataSource{}
	_ datasource.DataSourceWithConfigure = &githubBranchDataSource{}
)

type githubBranchDataSource struct {
	client *githubpkg.Owner
}

type githubBranchDataSourceModel struct {
	Repository types.String `tfsdk:"repository"`
	Branch     types.String `tfsdk:"branch"`
	Etag       types.String `tfsdk:"etag"`
	Ref        types.String `tfsdk:"ref"`
	Sha        types.String `tfsdk:"sha"`
}

func NewGithubBranchDataSource() datasource.DataSource {
	return &githubBranchDataSource{}
}

func (d *githubBranchDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch"
}

func (d *githubBranchDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information about a repository branch.",
		Attributes: map[string]schema.Attribute{
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"branch": schema.StringAttribute{
				Description: "The name of the branch.",
				Required:    true,
			},
			"etag": schema.StringAttribute{
				Description: "An ETag representing the Branch object.",
				Computed:    true,
			},
			"ref": schema.StringAttribute{
				Description: "A string representing a branch reference, in the form of 'refs/heads/<branch>'.",
				Computed:    true,
			},
			"sha": schema.StringAttribute{
				Description: "A string representing the SHA of the branch reference.",
				Computed:    true,
			},
		},
	}
}

func (d *githubBranchDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*githubpkg.Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *githubBranchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubBranchDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoName := data.Repository.ValueString()
	branchName := data.Branch.ValueString()
	branchRefName := "refs/heads/" + branchName

	tflog.Debug(ctx, "Reading GitHub branch", map[string]interface{}{
		"repository": repoName,
		"branch":     branchName,
		"ref":        branchRefName,
	})

	ref, response, err := d.client.V3Client().Git.GetRef(ctx, d.client.Name(), repoName, branchRefName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Debug(ctx, "Branch not found", map[string]interface{}{
					"repository": repoName,
					"branch":     branchName,
				})
				// Branch not found - set empty computed values
				data.Etag = types.StringValue("")
				data.Ref = types.StringValue("")
				data.Sha = types.StringValue("")

				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Branch",
			fmt.Sprintf("An unexpected error occurred while reading the GitHub branch %s in repository %s: %s", branchName, repoName, err.Error()),
		)
		return
	}

	// Set computed values
	data.Etag = types.StringValue(response.Header.Get("ETag"))
	data.Ref = types.StringValue(*ref.Ref)
	data.Sha = types.StringValue(*ref.Object.SHA)

	tflog.Debug(ctx, "Successfully read GitHub branch", map[string]interface{}{
		"repository": repoName,
		"branch":     branchName,
		"ref":        data.Ref.ValueString(),
		"sha":        data.Sha.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
