package github

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ datasource.DataSource              = &githubSshKeysDataSource{}
	_ datasource.DataSourceWithConfigure = &githubSshKeysDataSource{}
)

type githubSshKeysDataSource struct {
	client *Owner
}

type githubSshKeysDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Keys types.List   `tfsdk:"keys"`
}

func NewGithubSshKeysDataSource() datasource.DataSource {
	return &githubSshKeysDataSource{}
}

func (d *githubSshKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_keys"
}

func (d *githubSshKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on GitHub's SSH keys.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The identifier of the data source.",
				Computed:    true,
			},
			"keys": schema.ListAttribute{
				Description: "An array of GitHub's SSH keys.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *githubSshKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubSshKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubSshKeysDataSourceModel

	tflog.Debug(ctx, "Reading GitHub SSH keys")

	api, _, err := d.client.V3Client().Meta.Get(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub SSH Keys",
			fmt.Sprintf("An unexpected error occurred while reading GitHub's SSH keys: %s", err.Error()),
		)
		return
	}

	// Convert string slice to terraform types.List
	keys := make([]types.String, len(api.SSHKeys))
	for i, key := range api.SSHKeys {
		keys[i] = types.StringValue(key)
	}

	keysList, diags := types.ListValueFrom(ctx, types.StringType, keys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the data
	data.ID = types.StringValue("github-ssh-keys")
	data.Keys = keysList

	tflog.Debug(ctx, "Successfully read GitHub SSH keys", map[string]interface{}{
		"keys_count": len(api.SSHKeys),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
