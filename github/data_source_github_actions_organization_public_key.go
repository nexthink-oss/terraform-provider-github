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
	_ datasource.DataSource              = &githubActionsOrganizationPublicKeyDataSource{}
	_ datasource.DataSourceWithConfigure = &githubActionsOrganizationPublicKeyDataSource{}
)

type githubActionsOrganizationPublicKeyDataSource struct {
	client *Owner
}

type githubActionsOrganizationPublicKeyDataSourceModel struct {
	ID    types.String `tfsdk:"id"`
	KeyID types.String `tfsdk:"key_id"`
	Key   types.String `tfsdk:"key"`
}

func NewGithubActionsOrganizationPublicKeyDataSource() datasource.DataSource {
	return &githubActionsOrganizationPublicKeyDataSource{}
}

func (d *githubActionsOrganizationPublicKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_organization_public_key"
}

func (d *githubActionsOrganizationPublicKeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub Actions Organization Public Key.",
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

func (d *githubActionsOrganizationPublicKeyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubActionsOrganizationPublicKeyDataSource) checkOrganization() error {
	if !d.client.IsOrganization {
		return fmt.Errorf("this data source can only be used with organization accounts")
	}
	return nil
}

func (d *githubActionsOrganizationPublicKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubActionsOrganizationPublicKeyDataSourceModel

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

	tflog.Debug(ctx, "Reading GitHub Actions organization public key", map[string]any{
		"owner": owner,
	})

	publicKey, _, err := d.client.V3Client().Actions.GetOrgPublicKey(ctx, owner)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Actions Organization Public Key",
			err.Error(),
		)
		return
	}

	data.ID = types.StringValue(publicKey.GetKeyID())
	data.KeyID = types.StringValue(publicKey.GetKeyID())
	data.Key = types.StringValue(publicKey.GetKey())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
