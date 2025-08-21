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
	_ datasource.DataSource              = &githubOrganizationTeamSyncGroupsDataSource{}
	_ datasource.DataSourceWithConfigure = &githubOrganizationTeamSyncGroupsDataSource{}
)

func NewGithubOrganizationTeamSyncGroupsDataSource() datasource.DataSource {
	return &githubOrganizationTeamSyncGroupsDataSource{}
}

type githubOrganizationTeamSyncGroupsDataSource struct {
	client *Owner
}

type githubOrganizationTeamSyncGroupModel struct {
	GroupID          types.String `tfsdk:"group_id"`
	GroupName        types.String `tfsdk:"group_name"`
	GroupDescription types.String `tfsdk:"group_description"`
}

type githubOrganizationTeamSyncGroupsDataSourceModel struct {
	ID     types.String                           `tfsdk:"id"`
	Groups []githubOrganizationTeamSyncGroupModel `tfsdk:"groups"`
}

func (d *githubOrganizationTeamSyncGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_team_sync_groups"
}

func (d *githubOrganizationTeamSyncGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get the external identity provider (IdP) groups for an organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the organization.",
				Computed:    true,
			},
			"groups": schema.ListNestedAttribute{
				Description: "List of external identity provider (IdP) groups available to the organization.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"group_id": schema.StringAttribute{
							Description: "The ID of the IdP group.",
							Computed:    true,
						},
						"group_name": schema.StringAttribute{
							Description: "The name of the IdP group.",
							Computed:    true,
						},
						"group_description": schema.StringAttribute{
							Description: "The description of the IdP group.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubOrganizationTeamSyncGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubOrganizationTeamSyncGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubOrganizationTeamSyncGroupsDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.client.V3Client()
	orgName := d.client.Name()

	tflog.Debug(ctx, "Getting team sync groups for organization", map[string]any{
		"org": orgName,
	})

	options := &github.ListIDPGroupsOptions{
		ListCursorOptions: github.ListCursorOptions{
			PerPage: maxPerPage, // maxPerPage from GitHub SDK package
		},
	}

	var allGroups []githubOrganizationTeamSyncGroupModel
	for {
		idpGroupList, response, err := client.Teams.ListIDPGroupsInOrganization(ctx, orgName, options)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to List IdP Groups",
				fmt.Sprintf("Error listing IdP groups for organization %s: %v", orgName, err),
			)
			return
		}

		// Process the groups from this page
		groups, err := d.flattenGithubIDPGroupList(idpGroupList)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Flatten IdP Groups",
				fmt.Sprintf("Error flattening IdP groups for organization %s: %v", orgName, err),
			)
			return
		}

		allGroups = append(allGroups, groups...)

		// Check if there are more pages
		if response.NextPageToken == "" {
			break
		}
		options.Page = response.NextPageToken
	}

	// Set the data
	data.ID = types.StringValue(fmt.Sprintf("%s/github-org-team-sync-groups", orgName))
	data.Groups = allGroups

	tflog.Debug(ctx, "Found team sync groups", map[string]any{
		"org":   orgName,
		"count": len(allGroups),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// flattenGithubIDPGroupList converts GitHub API response to our model
func (d *githubOrganizationTeamSyncGroupsDataSource) flattenGithubIDPGroupList(idpGroupList *github.IDPGroupList) ([]githubOrganizationTeamSyncGroupModel, error) {
	if idpGroupList == nil {
		return []githubOrganizationTeamSyncGroupModel{}, nil
	}

	groups := make([]githubOrganizationTeamSyncGroupModel, 0, len(idpGroupList.Groups))
	for _, group := range idpGroupList.Groups {
		groupModel := githubOrganizationTeamSyncGroupModel{
			GroupID:          types.StringValue(group.GetGroupID()),
			GroupName:        types.StringValue(group.GetGroupName()),
			GroupDescription: types.StringValue(group.GetGroupDescription()),
		}
		groups = append(groups, groupModel)
	}

	return groups, nil
}
