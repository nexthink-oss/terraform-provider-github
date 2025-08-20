package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubRepositoryDeployKeysDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryDeployKeysDataSource{}
)

type githubRepositoryDeployKeysDataSource struct {
	client *Owner
}

type githubRepositoryDeployKeyModel struct {
	ID       types.Int64  `tfsdk:"id"`
	Key      types.String `tfsdk:"key"`
	Title    types.String `tfsdk:"title"`
	Verified types.Bool   `tfsdk:"verified"`
}

type githubRepositoryDeployKeysDataSourceModel struct {
	ID         types.String                     `tfsdk:"id"`
	Repository types.String                     `tfsdk:"repository"`
	Keys       []githubRepositoryDeployKeyModel `tfsdk:"keys"`
}

func NewGithubRepositoryDeployKeysDataSource() datasource.DataSource {
	return &githubRepositoryDeployKeysDataSource{}
}

func (d *githubRepositoryDeployKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_deploy_keys"
}

func (d *githubRepositoryDeployKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get all deploy keys of a repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"keys": schema.ListNestedAttribute{
				Description: "List of deploy keys for the repository.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The ID of the deploy key.",
							Computed:    true,
						},
						"key": schema.StringAttribute{
							Description: "The SSH key content.",
							Computed:    true,
						},
						"title": schema.StringAttribute{
							Description: "The title of the deploy key.",
							Computed:    true,
						},
						"verified": schema.BoolAttribute{
							Description: "Whether the key has been verified.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubRepositoryDeployKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubRepositoryDeployKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryDeployKeysDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.client.V3Client()
	owner := d.client.Name()
	repository := data.Repository.ValueString()

	tflog.Debug(ctx, "Reading GitHub repository deploy keys", map[string]interface{}{
		"owner":      owner,
		"repository": repository,
	})

	options := github.ListOptions{
		PerPage: maxPerPage,
	}

	var allKeys []githubRepositoryDeployKeyModel
	for {
		keys, respGH, err := client.Repositories.ListKeys(ctx, owner, repository, &options)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Repository Deploy Keys",
				fmt.Sprintf("Unable to read deploy keys for repository %s/%s: %s", owner, repository, err),
			)
			return
		}

		for _, key := range keys {
			keyModel := githubRepositoryDeployKeyModel{
				ID:       types.Int64Value(key.GetID()),
				Key:      types.StringValue(key.GetKey()),
				Title:    types.StringValue(key.GetTitle()),
				Verified: types.BoolValue(key.GetVerified()),
			}
			allKeys = append(allKeys, keyModel)
		}

		if respGH.NextPage == 0 {
			break
		}
		options.Page = respGH.NextPage
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", owner, repository))
	data.Keys = allKeys

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
