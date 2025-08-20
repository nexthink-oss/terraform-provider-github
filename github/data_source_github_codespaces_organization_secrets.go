package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ datasource.DataSource              = &githubCodespacesOrganizationSecretsDataSource{}
	_ datasource.DataSourceWithConfigure = &githubCodespacesOrganizationSecretsDataSource{}
)

type githubCodespacesOrganizationSecretsDataSource struct {
	client *Owner
}

type githubCodespacesOrganizationSecretsDataSourceModel struct {
	ID      types.String `tfsdk:"id"`
	Secrets types.List   `tfsdk:"secrets"`
}

type githubCodespacesOrganizationSecretModel struct {
	Name       types.String `tfsdk:"name"`
	Visibility types.String `tfsdk:"visibility"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func NewGithubCodespacesOrganizationSecretsDataSource() datasource.DataSource {
	return &githubCodespacesOrganizationSecretsDataSource{}
}

func (d *githubCodespacesOrganizationSecretsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_codespaces_organization_secrets"
}

func (d *githubCodespacesOrganizationSecretsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get codespaces secrets of the organization",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the organization.",
				Computed:    true,
			},
			"secrets": schema.ListNestedAttribute{
				Description: "An array of organization codespaces secrets.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Secret name.",
							Computed:    true,
						},
						"visibility": schema.StringAttribute{
							Description: "Visibility of the secret.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Date of 'secret' creation.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "Date of 'secret' update.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubCodespacesOrganizationSecretsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubCodespacesOrganizationSecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubCodespacesOrganizationSecretsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := d.client.Name()

	tflog.Debug(ctx, "Reading GitHub Codespaces organization secrets", map[string]interface{}{
		"owner": owner,
	})

	options := github.ListOptions{
		PerPage: 100,
	}

	var allSecrets []githubCodespacesOrganizationSecretModel
	for {
		secrets, resp_github, err := d.client.V3Client().Codespaces.ListOrgSecrets(ctx, owner, &options)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read GitHub Codespaces Organization Secrets",
				fmt.Sprintf("Error reading organization codespaces secrets: %s", err.Error()),
			)
			return
		}

		for _, secret := range secrets.Secrets {
			secretModel := githubCodespacesOrganizationSecretModel{
				Name:       types.StringValue(secret.Name),
				Visibility: types.StringValue(secret.Visibility),
				CreatedAt:  types.StringValue(secret.CreatedAt.String()),
				UpdatedAt:  types.StringValue(secret.UpdatedAt.String()),
			}
			allSecrets = append(allSecrets, secretModel)
		}

		if resp_github.NextPage == 0 {
			break
		}
		options.Page = resp_github.NextPage
	}

	// Convert secrets to Framework List
	secretsList, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":       types.StringType,
			"visibility": types.StringType,
			"created_at": types.StringType,
			"updated_at": types.StringType,
		},
	}, allSecrets)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(owner)
	data.Secrets = secretsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
