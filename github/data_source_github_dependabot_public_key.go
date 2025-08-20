package github

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ datasource.DataSource              = &githubDependabotPublicKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &githubDependabotPublicKeyDataSource{}
)

type githubDependabotPublicKeyDataSource struct {
	client *Owner
}

type githubDependabotPublicKeyDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Repository types.String `tfsdk:"repository"`
	KeyID      types.String `tfsdk:"key_id"`
	Key        types.String `tfsdk:"key"`
}

func NewGithubDependabotPublicKeyDataSource() datasource.DataSource {
	return &githubDependabotPublicKeyDataSource{}
}

func (d *githubDependabotPublicKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dependabot_public_key"
}

func (d *githubDependabotPublicKeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub Dependabot Public Key.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the public key.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
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

func (d *githubDependabotPublicKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubDependabotPublicKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubDependabotPublicKeyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repository := data.Repository.ValueString()
	owner := d.client.Name()

	tflog.Debug(ctx, "Reading GitHub Dependabot public key", map[string]interface{}{
		"owner":      owner,
		"repository": repository,
	})

	publicKey, _, err := d.client.V3Client().Dependabot.GetRepoPublicKey(ctx, owner, repository)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Dependabot Public Key",
			err.Error(),
		)
		return
	}

	data.ID = types.StringValue(publicKey.GetKeyID())
	data.KeyID = types.StringValue(publicKey.GetKeyID())
	data.Key = types.StringValue(publicKey.GetKey())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
