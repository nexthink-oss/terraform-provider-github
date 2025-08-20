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
	_ datasource.DataSource              = &githubActionsOrganizationVariablesDataSource{}
	_ datasource.DataSourceWithConfigure = &githubActionsOrganizationVariablesDataSource{}
)

type githubActionsOrganizationVariablesDataSource struct {
	client *Owner
}

type githubActionsOrganizationVariablesDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Variables types.List   `tfsdk:"variables"`
}

type githubActionsOrganizationVariableModel struct {
	Name       types.String `tfsdk:"name"`
	Value      types.String `tfsdk:"value"`
	Visibility types.String `tfsdk:"visibility"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func NewGithubActionsOrganizationVariablesDataSource() datasource.DataSource {
	return &githubActionsOrganizationVariablesDataSource{}
}

func (d *githubActionsOrganizationVariablesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_organization_variables"
}

func (d *githubActionsOrganizationVariablesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get Actions variables of the organization",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the organization.",
				Computed:    true,
			},
			"variables": schema.ListNestedAttribute{
				Description: "An array of organization actions variables.",
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
						"visibility": schema.StringAttribute{
							Description: "Variable visibility (all, private, selected).",
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

func (d *githubActionsOrganizationVariablesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubActionsOrganizationVariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubActionsOrganizationVariablesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := d.client.Name()

	tflog.Debug(ctx, "Reading GitHub Actions organization variables", map[string]interface{}{
		"organization": owner,
	})

	options := github.ListOptions{
		PerPage: 100,
	}

	var allVariables []githubActionsOrganizationVariableModel
	for {
		variables, resp_github, err := d.client.V3Client().Actions.ListOrgVariables(ctx, owner, &options)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read GitHub Actions Organization Variables",
				fmt.Sprintf("Error reading organization variables: %s", err.Error()),
			)
			return
		}

		for _, variable := range variables.Variables {
			variableModel := githubActionsOrganizationVariableModel{
				Name:       types.StringValue(variable.Name),
				Value:      types.StringValue(variable.Value),
				Visibility: types.StringValue(*variable.Visibility),
				CreatedAt:  types.StringValue(variable.CreatedAt.String()),
				UpdatedAt:  types.StringValue(variable.UpdatedAt.String()),
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
			"visibility": types.StringType,
			"created_at": types.StringType,
			"updated_at": types.StringType,
		},
	}, allVariables)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(owner)
	data.Variables = variablesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
