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
	_ datasource.DataSource              = &githubActionsRegistrationTokenDataSource{}
	_ datasource.DataSourceWithConfigure = &githubActionsRegistrationTokenDataSource{}
)

type githubActionsRegistrationTokenDataSource struct {
	client *Owner
}

type githubActionsRegistrationTokenDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Repository types.String `tfsdk:"repository"`
	Token      types.String `tfsdk:"token"`
	ExpiresAt  types.Int64  `tfsdk:"expires_at"`
}

func NewGithubActionsRegistrationTokenDataSource() datasource.DataSource {
	return &githubActionsRegistrationTokenDataSource{}
}

func (d *githubActionsRegistrationTokenDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_registration_token"
}

func (d *githubActionsRegistrationTokenDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a GitHub Actions repository registration token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the registration token.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
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

func (d *githubActionsRegistrationTokenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubActionsRegistrationTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubActionsRegistrationTokenDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repository := data.Repository.ValueString()
	owner := d.client.Name()

	tflog.Debug(ctx, "Creating GitHub Actions registration token", map[string]interface{}{
		"owner":      owner,
		"repository": repository,
	})

	token, _, err := d.client.V3Client().Actions.CreateRegistrationToken(ctx, owner, repository)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create GitHub Actions Registration Token",
			fmt.Sprintf("Error creating a GitHub Actions repository registration token for %s/%s: %s", owner, repository, err.Error()),
		)
		return
	}

	// Set the computed values
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", owner, repository))
	data.Token = types.StringValue(token.GetToken())
	data.ExpiresAt = types.Int64Value(token.GetExpiresAt().Unix())

	tflog.Debug(ctx, "Successfully created GitHub Actions registration token", map[string]interface{}{
		"owner":      owner,
		"repository": repository,
		"expires_at": token.GetExpiresAt().Unix(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
