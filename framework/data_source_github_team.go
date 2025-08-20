package framework

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/shurcooL/githubv4"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ datasource.DataSource              = &githubTeamDataSource{}
	_ datasource.DataSourceWithConfigure = &githubTeamDataSource{}
)

type githubTeamDataSource struct {
	client *githubpkg.Owner
}

type githubTeamRepositoryDetailedModel struct {
	RepoID   types.Int64  `tfsdk:"repo_id"`
	RoleName types.String `tfsdk:"role_name"`
}

type githubTeamDataSourceModel struct {
	ID                   types.String                        `tfsdk:"id"`
	Slug                 types.String                        `tfsdk:"slug"`
	Name                 types.String                        `tfsdk:"name"`
	Description          types.String                        `tfsdk:"description"`
	Privacy              types.String                        `tfsdk:"privacy"`
	Permission           types.String                        `tfsdk:"permission"`
	Members              types.List                          `tfsdk:"members"`
	Repositories         types.List                          `tfsdk:"repositories"`
	RepositoriesDetailed []githubTeamRepositoryDetailedModel `tfsdk:"repositories_detailed"`
	NodeID               types.String                        `tfsdk:"node_id"`
	MembershipType       types.String                        `tfsdk:"membership_type"`
	SummaryOnly          types.Bool                          `tfsdk:"summary_only"`
	ResultsPerPage       types.Int64                         `tfsdk:"results_per_page"`
}

func NewGithubTeamDataSource() datasource.DataSource {
	return &githubTeamDataSource{}
}

func (d *githubTeamDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}

func (d *githubTeamDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub team.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the team.",
				Computed:    true,
			},
			"slug": schema.StringAttribute{
				Description: "The slug of the team.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the team.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the team.",
				Computed:    true,
			},
			"privacy": schema.StringAttribute{
				Description: "The privacy setting of the team.",
				Computed:    true,
			},
			"permission": schema.StringAttribute{
				Description: "The permission level of the team.",
				Computed:    true,
			},
			"members": schema.ListAttribute{
				Description: "List of team members.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"repositories": schema.ListAttribute{
				Description: "List of repositories the team has access to.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"repositories_detailed": schema.ListNestedAttribute{
				Description: "Detailed information about repositories the team has access to.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"repo_id": schema.Int64Attribute{
							Description: "The ID of the repository.",
							Computed:    true,
						},
						"role_name": schema.StringAttribute{
							Description: "The role name for the repository.",
							Computed:    true,
						},
					},
				},
			},
			"node_id": schema.StringAttribute{
				Description: "The Node ID of the team.",
				Computed:    true,
			},
			"membership_type": schema.StringAttribute{
				Description: "The type of membership to filter for. Can be either 'all' or 'immediate'.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("all", "immediate"),
				},
			},
			"summary_only": schema.BoolAttribute{
				Description: "Whether to return only summary information (no members or repositories).",
				Optional:    true,
				Computed:    true,
			},
			"results_per_page": schema.Int64Attribute{
				Description: "The number of results per page (max 100).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 100),
				},
			},
		},
	}
}

func (d *githubTeamDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*githubpkg.Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *githubTeamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubTeamDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	slug := data.Slug.ValueString()

	// Set defaults for optional fields
	summaryOnly := false
	if !data.SummaryOnly.IsNull() && !data.SummaryOnly.IsUnknown() {
		summaryOnly = data.SummaryOnly.ValueBool()
	}

	resultsPerPage := 100
	if !data.ResultsPerPage.IsNull() && !data.ResultsPerPage.IsUnknown() {
		resultsPerPage = int(data.ResultsPerPage.ValueInt64())
	}

	membershipType := "all"
	if !data.MembershipType.IsNull() && !data.MembershipType.IsUnknown() {
		membershipType = data.MembershipType.ValueString()
	}

	tflog.Debug(ctx, "Reading GitHub team", map[string]interface{}{
		"slug":             slug,
		"summary_only":     summaryOnly,
		"results_per_page": resultsPerPage,
		"membership_type":  membershipType,
	})

	client := d.client.V3Client()
	orgID := d.client.ID()
	orgName := d.client.Name()

	team, _, err := client.Teams.GetTeamBySlug(ctx, orgName, slug)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Team",
			fmt.Sprintf("An unexpected error occurred while reading team %s: %s", slug, err.Error()),
		)
		return
	}

	var members []types.String
	var repositories []types.String
	var repositoriesDetailed []githubTeamRepositoryDetailedModel

	if !summaryOnly {
		// Get team members
		options := github.TeamListTeamMembersOptions{
			ListOptions: github.ListOptions{
				PerPage: resultsPerPage,
			},
		}

		if membershipType == "all" {
			for {
				membersList, resp_api, err := client.Teams.ListTeamMembersByID(ctx, orgID, team.GetID(), &options)
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to Read GitHub Team Members",
						fmt.Sprintf("An unexpected error occurred while reading team members for %s: %s", slug, err.Error()),
					)
					return
				}

				for _, v := range membersList {
					members = append(members, types.StringValue(v.GetLogin()))
				}

				if resp_api.NextPage == 0 {
					break
				}
				options.Page = resp_api.NextPage
			}
		} else {
			// Use GraphQL for immediate membership
			type member struct {
				Login string
			}
			var query struct {
				Organization struct {
					Team struct {
						Members struct {
							Nodes    []member
							PageInfo struct {
								EndCursor   githubv4.String
								HasNextPage bool
							}
						} `graphql:"members(first:100,after:$memberCursor,membership:IMMEDIATE)"`
					} `graphql:"team(slug:$slug)"`
				} `graphql:"organization(login:$owner)"`
			}
			variables := map[string]any{
				"owner":        githubv4.String(orgName),
				"slug":         githubv4.String(slug),
				"memberCursor": (*githubv4.String)(nil),
			}
			v4client := d.client.V4Client()
			for {
				nameErr := v4client.Query(ctx, &query, variables)
				if nameErr != nil {
					resp.Diagnostics.AddError(
						"Unable to Read GitHub Team Members",
						fmt.Sprintf("An unexpected error occurred while reading immediate team members for %s: %s", slug, nameErr.Error()),
					)
					return
				}
				for _, v := range query.Organization.Team.Members.Nodes {
					members = append(members, types.StringValue(v.Login))
				}
				if query.Organization.Team.Members.PageInfo.HasNextPage {
					variables["memberCursor"] = query.Organization.Team.Members.PageInfo.EndCursor
				} else {
					break
				}
			}
		}

		// Get team repositories
		options.Page = 0 // Reset page counter
		for {
			repository, resp_api, err := client.Teams.ListTeamReposByID(ctx, orgID, team.GetID(), &options.ListOptions)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Read GitHub Team Repositories",
					fmt.Sprintf("An unexpected error occurred while reading team repositories for %s: %s", slug, err.Error()),
				)
				return
			}

			for _, v := range repository {
				repositories = append(repositories, types.StringValue(v.GetName()))
				repositoriesDetailed = append(repositoriesDetailed, githubTeamRepositoryDetailedModel{
					RepoID:   types.Int64Value(v.GetID()),
					RoleName: types.StringValue(v.GetRoleName()),
				})
			}

			if resp_api.NextPage == 0 {
				break
			}
			options.Page = resp_api.NextPage
		}
	}

	// Convert to list types
	membersList, diags := types.ListValueFrom(ctx, types.StringType, members)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	repositoriesList, diags := types.ListValueFrom(ctx, types.StringType, repositories)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set values
	data.ID = types.StringValue(strconv.FormatInt(team.GetID(), 10))
	data.Name = types.StringValue(team.GetName())
	data.Description = types.StringValue(team.GetDescription())
	data.Privacy = types.StringValue(team.GetPrivacy())
	data.Permission = types.StringValue(team.GetPermission())
	data.NodeID = types.StringValue(team.GetNodeID())
	data.Members = membersList
	data.Repositories = repositoriesList
	data.RepositoriesDetailed = repositoriesDetailed

	// Set defaults for optional computed fields if they weren't provided
	if data.MembershipType.IsNull() || data.MembershipType.IsUnknown() {
		data.MembershipType = types.StringValue("all")
	}
	if data.SummaryOnly.IsNull() || data.SummaryOnly.IsUnknown() {
		data.SummaryOnly = types.BoolValue(false)
	}
	if data.ResultsPerPage.IsNull() || data.ResultsPerPage.IsUnknown() {
		data.ResultsPerPage = types.Int64Value(100)
	}

	tflog.Debug(ctx, "Successfully read GitHub team", map[string]interface{}{
		"slug":          slug,
		"id":            data.ID.ValueString(),
		"name":          data.Name.ValueString(),
		"members_count": len(members),
		"repos_count":   len(repositories),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
