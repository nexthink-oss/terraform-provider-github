package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubActionsVariablesDataSource{}
	_ datasource.DataSourceWithConfigure = &githubActionsVariablesDataSource{}
)

type githubActionsVariablesDataSource struct {
	client *Owner
}

type githubActionsVariablesDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	FullName  types.String `tfsdk:"full_name"`
	Name      types.String `tfsdk:"name"`
	Variables types.List   `tfsdk:"variables"`
}

type githubActionsVariableModel struct {
	Name      types.String `tfsdk:"name"`
	Value     types.String `tfsdk:"value"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewGithubActionsVariablesDataSource() datasource.DataSource {
	return &githubActionsVariablesDataSource{}
}

func (d *githubActionsVariablesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_variables"
}

func (d *githubActionsVariablesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get Actions variables for a repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the repository.",
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
			"variables": schema.ListNestedAttribute{
				Description: "An array of repository actions variables.",
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

func (d *githubActionsVariablesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubActionsVariablesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubActionsVariablesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := d.client.Name()
	var repoName string

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

	tflog.Debug(ctx, "Reading GitHub Actions variables", map[string]interface{}{
		"owner":      owner,
		"repository": repoName,
	})

	options := github.ListOptions{
		PerPage: maxPerPage,
	}

	var allVariables []githubActionsVariableModel
	for {
		variables, resp_github, err := d.client.V3Client().Actions.ListRepoVariables(ctx, owner, repoName, &options)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read GitHub Actions Variables",
				fmt.Sprintf("Error reading repository variables: %s", err.Error()),
			)
			return
		}

		for _, variable := range variables.Variables {
			variableModel := githubActionsVariableModel{
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

	data.ID = types.StringValue(repoName)
	data.Variables = variablesList

	// Set computed values based on what was provided
	if hasFullName {
		data.Name = types.StringValue(repoName)
	} else {
		data.FullName = types.StringValue(fmt.Sprintf("%s/%s", owner, repoName))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
