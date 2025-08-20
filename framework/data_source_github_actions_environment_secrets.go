package framework

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ datasource.DataSource              = &githubActionsEnvironmentSecretsDataSource{}
	_ datasource.DataSourceWithConfigure = &githubActionsEnvironmentSecretsDataSource{}
)

type githubActionsEnvironmentSecretsDataSource struct {
	client *githubpkg.Owner
}

type githubActionsEnvironmentSecretsDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	FullName    types.String `tfsdk:"full_name"`
	Name        types.String `tfsdk:"name"`
	Environment types.String `tfsdk:"environment"`
	Secrets     types.List   `tfsdk:"secrets"`
}

type githubActionsEnvironmentSecretModel struct {
	Name      types.String `tfsdk:"name"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewGithubActionsEnvironmentSecretsDataSource() datasource.DataSource {
	return &githubActionsEnvironmentSecretsDataSource{}
}

func (d *githubActionsEnvironmentSecretsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_environment_secrets"
}

func (d *githubActionsEnvironmentSecretsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get Actions secrets of the repository environment",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source (repository:environment).",
				Computed:    true,
			},
			"full_name": schema.StringAttribute{
				Description: "Full name of the repository (in `owner/name` format).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("name"),
					),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the repository.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(
						path.MatchRoot("full_name"),
					),
				},
			},
			"environment": schema.StringAttribute{
				Description: "The repository environment name.",
				Required:    true,
			},
			"secrets": schema.ListNestedAttribute{
				Description: "An array of repository environment actions secrets.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Secret name.",
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

func (d *githubActionsEnvironmentSecretsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubActionsEnvironmentSecretsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubActionsEnvironmentSecretsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := d.client.Name()
	var repoName string

	envName := data.Environment.ValueString()
	escapedEnvName := url.PathEscape(envName)

	// Validate ConflictsWith behavior for full_name and name
	hasFullName := !data.FullName.IsNull() && !data.FullName.IsUnknown()
	hasName := !data.Name.IsNull() && !data.Name.IsUnknown()

	if hasFullName && hasName {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Cannot specify both 'full_name' and 'name' attributes.",
		)
		return
	}

	if !hasFullName && !hasName {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"One of 'full_name' or 'name' must be specified.",
		)
		return
	}

	if hasFullName {
		var err error
		owner, repoName, err = splitRepoFullName(data.FullName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid Repository Full Name",
				fmt.Sprintf("Unable to parse repository full name: %s", err.Error()),
			)
			return
		}
	}

	if hasName {
		repoName = data.Name.ValueString()
	}

	tflog.Debug(ctx, "Reading GitHub Actions environment secrets", map[string]interface{}{
		"owner":       owner,
		"repository":  repoName,
		"environment": envName,
	})

	// Get repository information first
	repo, _, err := d.client.V3Client().Repositories.Get(ctx, owner, repoName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Repository",
			fmt.Sprintf("Error reading repository %s/%s: %s", owner, repoName, err.Error()),
		)
		return
	}

	options := github.ListOptions{
		PerPage: 100,
	}

	var allSecrets []githubActionsEnvironmentSecretModel
	for {
		secrets, resp_github, err := d.client.V3Client().Actions.ListEnvSecrets(ctx, int(repo.GetID()), escapedEnvName, &options)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read GitHub Actions Environment Secrets",
				fmt.Sprintf("Error reading environment secrets for repository %s/%s, environment %s: %s", owner, repoName, envName, err.Error()),
			)
			return
		}

		for _, secret := range secrets.Secrets {
			secretModel := githubActionsEnvironmentSecretModel{
				Name:      types.StringValue(secret.Name),
				CreatedAt: types.StringValue(secret.CreatedAt.String()),
				UpdatedAt: types.StringValue(secret.UpdatedAt.String()),
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
			"created_at": types.StringType,
			"updated_at": types.StringType,
		},
	}, allSecrets)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set ID using buildTwoPartID equivalent: "repository:environment"
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", repoName, envName))
	data.Secrets = secretsList

	// Set computed values based on what was provided
	if hasFullName {
		data.Name = types.StringValue(repoName)
	} else {
		data.FullName = types.StringValue(fmt.Sprintf("%s/%s", owner, repoName))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
