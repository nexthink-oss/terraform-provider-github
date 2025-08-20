package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubRepositoryDeploymentBranchPoliciesDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryDeploymentBranchPoliciesDataSource{}
)

type githubRepositoryDeploymentBranchPoliciesDataSource struct {
	client *Owner
}

type githubRepositoryDeploymentBranchPolicyModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

type githubRepositoryDeploymentBranchPoliciesDataSourceModel struct {
	ID                       types.String                                  `tfsdk:"id"`
	Repository               types.String                                  `tfsdk:"repository"`
	EnvironmentName          types.String                                  `tfsdk:"environment_name"`
	DeploymentBranchPolicies []githubRepositoryDeploymentBranchPolicyModel `tfsdk:"deployment_branch_policies"`
}

func NewGithubRepositoryDeploymentBranchPoliciesDataSource() datasource.DataSource {
	return &githubRepositoryDeploymentBranchPoliciesDataSource{}
}

func (d *githubRepositoryDeploymentBranchPoliciesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_deployment_branch_policies"
}

func (d *githubRepositoryDeploymentBranchPoliciesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get the list of deployment branch policies for a given repo / env.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The GitHub repository name.",
				Required:    true,
			},
			"environment_name": schema.StringAttribute{
				Description: "The target environment name.",
				Required:    true,
			},
			"deployment_branch_policies": schema.ListNestedAttribute{
				Description: "List of deployment branch policies for the repository environment.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the deployment branch policy.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the deployment branch policy.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubRepositoryDeploymentBranchPoliciesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *githubRepositoryDeploymentBranchPoliciesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryDeploymentBranchPoliciesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.client.V3Client()
	owner := d.client.Name()
	repository := data.Repository.ValueString()
	environmentName := data.EnvironmentName.ValueString()

	tflog.Debug(ctx, "Reading GitHub repository deployment branch policies", map[string]interface{}{
		"owner":            owner,
		"repository":       repository,
		"environment_name": environmentName,
	})

	policies, _, err := client.Repositories.ListDeploymentBranchPolicies(ctx, owner, repository, environmentName)
	if err != nil {
		// Note: The SDKv2 implementation returns nil on error, which might be intentional
		// for handling cases where the environment doesn't exist or has no policies.
		// We'll follow the same pattern but with proper error handling.
		tflog.Warn(ctx, "Error reading deployment branch policies", map[string]interface{}{
			"error": err.Error(),
		})

		// Set empty list and continue, matching SDKv2 behavior
		data.DeploymentBranchPolicies = []githubRepositoryDeploymentBranchPolicyModel{}
	} else {
		var allPolicies []githubRepositoryDeploymentBranchPolicyModel

		for _, policy := range policies.BranchPolicies {
			policyModel := githubRepositoryDeploymentBranchPolicyModel{
				ID:   types.StringValue(strconv.FormatInt(policy.GetID(), 10)),
				Name: types.StringValue(policy.GetName()),
			}
			allPolicies = append(allPolicies, policyModel)
		}

		data.DeploymentBranchPolicies = allPolicies
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s", repository, environmentName))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
