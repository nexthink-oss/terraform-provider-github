package framework

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ datasource.DataSource              = &githubAppTokenDataSource{}
	_ datasource.DataSourceWithConfigure = &githubAppTokenDataSource{}
)

type githubAppTokenDataSource struct {
	client *githubpkg.Owner
}

type githubAppTokenDataSourceModel struct {
	AppID          types.String `tfsdk:"app_id"`
	InstallationID types.String `tfsdk:"installation_id"`
	PemFile        types.String `tfsdk:"pem_file"`
	Token          types.String `tfsdk:"token"`
}

func NewGithubAppTokenDataSource() datasource.DataSource {
	return &githubAppTokenDataSource{}
}

func (d *githubAppTokenDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_token"
}

func (d *githubAppTokenDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Generate a GitHub APP JWT.",
		Attributes: map[string]schema.Attribute{
			"app_id": schema.StringAttribute{
				Description: "GitHub App ID.",
				Required:    true,
			},
			"installation_id": schema.StringAttribute{
				Description: "GitHub App Installation ID.",
				Required:    true,
			},
			"pem_file": schema.StringAttribute{
				Description: "GitHub App private key in PEM format.",
				Required:    true,
			},
			"token": schema.StringAttribute{
				Description: "The generated token from the credentials.",
				Computed:    true,
				Sensitive:   true,
			},
		},
	}
}

func (d *githubAppTokenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubAppTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubAppTokenDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := data.AppID.ValueString()
	installationID := data.InstallationID.ValueString()
	pemFile := data.PemFile.ValueString()

	tflog.Debug(ctx, "Generating GitHub App token", map[string]interface{}{
		"app_id":          appID,
		"installation_id": installationID,
	})

	baseURL := d.client.V3Client().BaseURL.String()

	// The Go encoding/pem package only decodes PEM formatted blocks
	// that contain new lines. Some platforms, like Terraform Cloud,
	// do not support new lines within Environment Variables.
	// Any occurrence of \n in the `pem_file` argument's value
	// (explicit value, or default value taken from
	// GITHUB_APP_PEM_FILE Environment Variable) is replaced with an
	// actual new line character before decoding.
	pemFile = strings.ReplaceAll(pemFile, `\n`, "\n")

	token, err := githubpkg.GenerateOAuthTokenFromApp(baseURL, appID, installationID, pemFile)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Generate GitHub App Token",
			fmt.Sprintf("An unexpected error occurred while generating the GitHub App token: %s", err.Error()),
		)
		return
	}

	// Set the computed values
	data.Token = types.StringValue(token)

	tflog.Debug(ctx, "Successfully generated GitHub App token", map[string]interface{}{
		"app_id":          appID,
		"installation_id": installationID,
	})

	// Set the ID to a static value since this is a data source
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
