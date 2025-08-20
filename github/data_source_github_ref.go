package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/google/go-github/v74/github"
)

var (
	_ datasource.DataSource              = &githubRefDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRefDataSource{}
)

type githubRefDataSource struct {
	client *Owner
}

type githubRefDataSourceModel struct {
	Ref        types.String `tfsdk:"ref"`
	Repository types.String `tfsdk:"repository"`
	Owner      types.String `tfsdk:"owner"`
	Etag       types.String `tfsdk:"etag"`
	Sha        types.String `tfsdk:"sha"`
}

func NewGithubRefDataSource() datasource.DataSource {
	return &githubRefDataSource{}
}

func (d *githubRefDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ref"
}

func (d *githubRefDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information about a repository ref.",
		Attributes: map[string]schema.Attribute{
			"ref": schema.StringAttribute{
				Description: "The ref to read (e.g., 'heads/main', 'tags/v1.0.0').",
				Required:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"owner": schema.StringAttribute{
				Description: "The GitHub owner name. If not specified, the provider's owner will be used.",
				Optional:    true,
			},
			"etag": schema.StringAttribute{
				Description: "An ETag representing the Ref object.",
				Computed:    true,
			},
			"sha": schema.StringAttribute{
				Description: "A string representing the SHA of the ref.",
				Computed:    true,
			},
		},
	}
}

func (d *githubRefDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *githubRefDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRefDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoName := data.Repository.ValueString()
	refName := data.Ref.ValueString()

	// Determine owner - use provided owner or fallback to client owner
	owner := d.client.Name()
	if !data.Owner.IsNull() && !data.Owner.IsUnknown() {
		owner = data.Owner.ValueString()
	}

	tflog.Debug(ctx, "Reading GitHub ref", map[string]interface{}{
		"repository": repoName,
		"ref":        refName,
		"owner":      owner,
	})

	ref, response, err := d.client.V3Client().Git.GetRef(ctx, owner, repoName, refName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Debug(ctx, "Ref not found", map[string]interface{}{
					"repository": repoName,
					"ref":        refName,
					"owner":      owner,
				})
				// Ref not found - set empty computed values and empty ID
				data.Etag = types.StringValue("")
				data.Sha = types.StringValue("")

				resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Ref",
			fmt.Sprintf("An unexpected error occurred while reading the GitHub ref %s in repository %s/%s: %s", refName, owner, repoName, err.Error()),
		)
		return
	}

	// Set computed values
	data.Etag = types.StringValue(response.Header.Get("ETag"))
	data.Sha = types.StringValue(*ref.Object.SHA)

	// Update owner in case it was determined from client
	data.Owner = types.StringValue(owner)

	tflog.Debug(ctx, "Successfully read GitHub ref", map[string]interface{}{
		"repository": repoName,
		"ref":        refName,
		"owner":      owner,
		"sha":        data.Sha.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
