package framework

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ datasource.DataSource              = &githubCodespacesOrganizationPublicKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &githubCodespacesOrganizationPublicKeyDataSource{}
)

type githubCodespacesOrganizationPublicKeyDataSource struct {
	client *githubpkg.Owner
}

type githubCodespacesOrganizationPublicKeyDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	KeyID types.String `tfsdk:"key_id"`
	Key   types.String `tfsdk:"key"`
}

func NewGithubCodespacesOrganizationPublicKeyDataSource() datasource.DataSource {
	return &githubCodespacesOrganizationPublicKeyDataSource{}
}

func (d *githubCodespacesOrganizationPublicKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_codespaces_organization_public_key"
}

func (d *githubCodespacesOrganizationPublicKeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub Codespaces Organization Public Key.",
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

func (d *githubCodespacesOrganizationPublicKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*githubpkg.Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			"Expected *github.Owner, got something else.",
		)
		return
	}

	d.client = client
}

func (d *githubCodespacesOrganizationPublicKeyDataSource) checkOrganization() error {
	if !d.client.IsOrganization {
		return fmt.Errorf("this data source can only be used with organization accounts")
	}
	return nil
}

func (d *githubCodespacesOrganizationPublicKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubCodespacesOrganizationPublicKeyDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check that this is an organization account
	err := d.checkOrganization()
	if err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			err.Error(),
		)
		return
	}

	owner := d.client.Name()

	tflog.Debug(ctx, "Reading GitHub Codespaces organization public key", map[string]interface{}{
		"owner": owner,
	})

	publicKey, _, err := d.client.V3Client().Codespaces.GetOrgPublicKey(ctx, owner)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Codespaces Organization Public Key",
			err.Error(),
		)
		return
	}

	data.ID = types.StringValue(publicKey.GetKeyID())
	data.KeyID = types.StringValue(publicKey.GetKeyID())
	data.Key = types.StringValue(publicKey.GetKey())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
