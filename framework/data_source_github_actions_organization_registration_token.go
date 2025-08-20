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
	_ datasource.DataSource              = &githubActionsOrganizationRegistrationTokenDataSource{}
	_ datasource.DataSourceWithConfigure = &githubActionsOrganizationRegistrationTokenDataSource{}
)

type githubActionsOrganizationRegistrationTokenDataSource struct {
	client *githubpkg.Owner
}

type githubActionsOrganizationRegistrationTokenDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Token     types.String `tfsdk:"token"`
	ExpiresAt types.Int64  `tfsdk:"expires_at"`
}

func NewGithubActionsOrganizationRegistrationTokenDataSource() datasource.DataSource {
	return &githubActionsOrganizationRegistrationTokenDataSource{}
}

func (d *githubActionsOrganizationRegistrationTokenDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_organization_registration_token"
}

func (d *githubActionsOrganizationRegistrationTokenDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a GitHub Actions organization registration token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the registration token.",
				Computed:    true,
			},
			"token": schema.StringAttribute{
				Description: "The generated registration token.",
				Computed:    true,
				Sensitive:   true,
			},
			"expires_at": schema.Int64Attribute{
				Description: "The Unix timestamp when the registration token expires.",
				Computed:    true,
			},
		},
	}
}

func (d *githubActionsOrganizationRegistrationTokenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubActionsOrganizationRegistrationTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubActionsOrganizationRegistrationTokenDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := d.client.Name()

	tflog.Debug(ctx, "Creating GitHub Actions organization registration token", map[string]interface{}{
		"owner": owner,
	})

	token, _, err := d.client.V3Client().Actions.CreateOrganizationRegistrationToken(ctx, owner)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create GitHub Actions Organization Registration Token",
			fmt.Sprintf("Error creating a GitHub Actions organization registration token for %s: %s", owner, err.Error()),
		)
		return
	}

	// Set the computed values
	data.ID = types.StringValue(owner)
	data.Token = types.StringValue(token.GetToken())
	data.ExpiresAt = types.Int64Value(token.GetExpiresAt().Unix())

	tflog.Debug(ctx, "Successfully created GitHub Actions organization registration token", map[string]interface{}{
		"owner":      owner,
		"expires_at": token.GetExpiresAt().Unix(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
