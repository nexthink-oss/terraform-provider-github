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
	_ datasource.DataSource              = &githubActionsEnvironmentVariablesDataSource{}
	_ datasource.DataSourceWithConfigure = &githubActionsEnvironmentVariablesDataSource{}
)

type githubActionsEnvironmentVariablesDataSource struct {
	client *githubpkg.Owner
}

type githubActionsEnvironmentVariablesDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	FullName    types.String `tfsdk:"full_name"`
	Name        types.String `tfsdk:"name"`
	Environment types.String `tfsdk:"environment"`
	Variables   types.List   `tfsdk:"variables"`
}

type githubActionsEnvironmentVariableModel struct {
	Name      types.String `tfsdk:"name"`
	Value     types.String `tfsdk:"value"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewGithubActionsEnvironmentVariablesDataSource() datasource.DataSource {
	return &githubActionsEnvironmentVariablesDataSource{}
}

func (d *githubActionsEnvironmentVariablesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_environment_variables"
}

func (d *githubActionsEnvironmentVariablesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get Actions variables of the repository environment",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
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
				Description: "The environment name.",
				Required:    true,
			},
			"variables": schema.ListNestedAttribute{
				Description: "An array of repository environment actions variables.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Variable name.",
							Computed:    true,
						},
						"value": schema.StringAttribute{
							Description: "Variable value.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "Date of 'variable' creation.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "Date of 'variable' update.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubActionsEnvironmentVariablesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubActionsEnvironmentVariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubActionsEnvironmentVariablesDataSourceModel

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

	tflog.Debug(ctx, "Reading GitHub Actions environment variables", map[string]interface{}{
		"owner":       owner,
		"repository":  repoName,
		"environment": envName,
	})

	options := github.ListOptions{
		PerPage: 100,
	}

	var allVariables []githubActionsEnvironmentVariableModel
	for {
		variables, resp_github, err := d.client.V3Client().Actions.ListEnvVariables(ctx, owner, repoName, escapedEnvName, &options)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read GitHub Actions Environment Variables",
				fmt.Sprintf("Error reading repository environment variables: %s", err.Error()),
			)
			return
		}

		for _, variable := range variables.Variables {
			variableModel := githubActionsEnvironmentVariableModel{
				Name:      types.StringValue(variable.Name),
				Value:     types.StringValue(variable.Value),
				CreatedAt: types.StringValue(variable.CreatedAt.String()),
				UpdatedAt: types.StringValue(variable.UpdatedAt.String()),
			}
			allVariables = append(allVariables, variableModel)
		}

		if resp_github.NextPage == 0 {
			break
		}
		options.Page = resp_github.NextPage
	}

	// Convert variables to Framework List
	variablesList, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":       types.StringType,
			"value":      types.StringType,
			"created_at": types.StringType,
			"updated_at": types.StringType,
		},
	}, allVariables)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the ID using the same pattern as SDKv2 (repoName:envName)
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", repoName, envName))
	data.Variables = variablesList

	// Set computed values based on what was provided
	if hasFullName {
		data.Name = types.StringValue(repoName)
	} else {
		data.FullName = types.StringValue(fmt.Sprintf("%s/%s", owner, repoName))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
