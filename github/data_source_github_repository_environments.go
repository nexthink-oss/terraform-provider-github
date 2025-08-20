package github

import (
	"context"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ datasource.DataSource              = &githubRepositoryEnvironmentsDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryEnvironmentsDataSource{}
)

type githubRepositoryEnvironmentsDataSource struct {
	client *Owner
}

type githubRepositoryEnvironmentsDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Repository   types.String `tfsdk:"repository"`
	Environments types.List   `tfsdk:"environments"`
}

type githubRepositoryEnvironmentModel struct {
	Name   types.String `tfsdk:"name"`
	NodeID types.String `tfsdk:"node_id"`
}

func NewGithubRepositoryEnvironmentsDataSource() datasource.DataSource {
	return &githubRepositoryEnvironmentsDataSource{}
}

func (d *githubRepositoryEnvironmentsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_environments"
}

func (d *githubRepositoryEnvironmentsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub repository's environments.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this resource.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"environments": schema.ListNestedAttribute{
				Description: "The list of environments in this repository.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the environment.",
							Computed:    true,
						},
						"node_id": schema.StringAttribute{
							Description: "The node ID of the environment.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubRepositoryEnvironmentsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			"Expected *github.Owner, got: %T. Please report this issue to the provider developers.",
		)
		return
	}

	d.client = client
}

func (d *githubRepositoryEnvironmentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryEnvironmentsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.client.V3Client()
	orgName := d.client.Name()
	repoName := data.Repository.ValueString()

	tflog.Debug(ctx, "Reading repository environments", map[string]interface{}{
		"org":  orgName,
		"repo": repoName,
	})

	var allEnvironments []githubRepositoryEnvironmentModel
	var listOptions *github.EnvironmentListOptions

	for {
		environments, response, err := client.Repositories.ListEnvironments(ctx, orgName, repoName, listOptions)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read Repository Environments",
				"An error occurred while reading the repository environments. "+
					"GitHub API returned: "+err.Error(),
			)
			return
		}

		if environments != nil {
			for _, env := range environments.Environments {
				envModel := githubRepositoryEnvironmentModel{
					Name:   types.StringValue(env.GetName()),
					NodeID: types.StringValue(env.GetNodeID()),
				}
				allEnvironments = append(allEnvironments, envModel)
			}
		}

		if response.NextPage == 0 {
			break
		}

		if listOptions == nil {
			listOptions = &github.EnvironmentListOptions{}
		}
		listOptions.Page = response.NextPage
	}

	// Convert to types.List
	environmentsList, diags := types.ListValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":    types.StringType,
			"node_id": types.StringType,
		},
	}, allEnvironments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = types.StringValue(repoName)
	data.Environments = environmentsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
