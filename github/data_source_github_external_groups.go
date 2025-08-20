package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubExternalGroupsDataSource{}
	_ datasource.DataSourceWithConfigure = &githubExternalGroupsDataSource{}
)

type githubExternalGroupsDataSource struct {
	client *Owner
}

type githubExternalGroupModel struct {
	GroupID   types.Int64  `tfsdk:"group_id"`
	GroupName types.String `tfsdk:"group_name"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

type githubExternalGroupsDataSourceModel struct {
	ID             types.String               `tfsdk:"id"`
	ExternalGroups []githubExternalGroupModel `tfsdk:"external_groups"`
}

func NewGithubExternalGroupsDataSource() datasource.DataSource {
	return &githubExternalGroupsDataSource{}
}

func (d *githubExternalGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_external_groups"
}

func (d *githubExternalGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieve external groups belonging to an organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"external_groups": schema.ListNestedAttribute{
				Description: "List of external groups in the organization.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_id": schema.Int64Attribute{
							Description: "The ID of the external group.",
							Computed:    true,
						},
						"group_name": schema.StringAttribute{
							Description: "The name of the external group.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "The timestamp when the external group was last updated.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubExternalGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubExternalGroupsDataSource) checkOrganization() error {
	if !d.client.IsOrganization {
		return fmt.Errorf("this data source can only be used with organization accounts")
	}
	return nil
}

func (d *githubExternalGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubExternalGroupsDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization configuration
	err := d.checkOrganization()
	if err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			err.Error(),
		)
		return
	}

	client := d.client.V3Client()
	orgName := d.client.Name()

	tflog.Debug(ctx, "Reading GitHub external groups", map[string]interface{}{
		"organization": orgName,
	})

	// Set up pagination options
	opts := &github.ListExternalGroupsOptions{}
	var allGroups []*github.ExternalGroup

	// Paginate through all external groups
	for {
		externalGroups, response, err := client.Teams.ListExternalGroups(ctx, orgName, opts)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Query GitHub External Groups",
				fmt.Sprintf("An unexpected error occurred while querying GitHub external groups: %s", err.Error()),
			)
			return
		}

		allGroups = append(allGroups, externalGroups.Groups...)

		if response.NextPage == 0 {
			break
		}
		opts.Page = response.NextPage
	}

	// Convert API response to data model
	var externalGroupModels []githubExternalGroupModel
	for _, group := range allGroups {
		groupModel := githubExternalGroupModel{}

		if group.GroupID != nil {
			groupModel.GroupID = types.Int64Value(*group.GroupID)
		} else {
			groupModel.GroupID = types.Int64Null()
		}

		if group.GroupName != nil {
			groupModel.GroupName = types.StringValue(*group.GroupName)
		} else {
			groupModel.GroupName = types.StringNull()
		}

		if group.UpdatedAt != nil {
			groupModel.UpdatedAt = types.StringValue(group.UpdatedAt.String())
		} else {
			groupModel.UpdatedAt = types.StringNull()
		}

		externalGroupModels = append(externalGroupModels, groupModel)
	}

	// Set ID using the same pattern as SDKv2
	data.ID = types.StringValue(fmt.Sprintf("/orgs/%s/external-groups", orgName))
	data.ExternalGroups = externalGroupModels

	tflog.Debug(ctx, "Successfully read GitHub external groups", map[string]interface{}{
		"organization": orgName,
		"groups_count": len(externalGroupModels),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
