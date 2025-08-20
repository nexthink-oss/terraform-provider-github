package github

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubCodespacesUserPublicKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &githubCodespacesUserPublicKeyDataSource{}
)

type githubCodespacesUserPublicKeyDataSource struct {
	client *Owner
}

type githubCodespacesUserPublicKeyDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	KeyID types.String `tfsdk:"key_id"`
	Key   types.String `tfsdk:"key"`
}

func NewGithubCodespacesUserPublicKeyDataSource() datasource.DataSource {
	return &githubCodespacesUserPublicKeyDataSource{}
}

func (d *githubCodespacesUserPublicKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_codespaces_user_public_key"
}

func (d *githubCodespacesUserPublicKeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub Codespaces User Public Key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the public key.",
				Computed:    true,
			},
			"key_id": schema.StringAttribute{
				Description: "The ID of the public key.",
				Computed:    true,
			},
			"key": schema.StringAttribute{
				Description: "The public key value.",
				Computed:    true,
			},
		},
	}
}

func (d *githubCodespacesUserPublicKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			"Expected *github.Owner, got something else.",
		)
		return
	}

	d.client = client
}

func (d *githubCodespacesUserPublicKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubCodespacesUserPublicKeyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading GitHub Codespaces user public key")

	publicKey, _, err := d.client.V3Client().Codespaces.GetUserPublicKey(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Codespaces User Public Key",
			err.Error(),
		)
		return
	}

	data.ID = types.StringValue(publicKey.GetKeyID())
	data.KeyID = types.StringValue(publicKey.GetKeyID())
	data.Key = types.StringValue(publicKey.GetKey())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
