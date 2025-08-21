package github

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/shurcooL/githubv4"
)

var (
	_ datasource.DataSource              = &githubOrganizationTeamsDataSource{}
	_ datasource.DataSourceWithConfigure = &githubOrganizationTeamsDataSource{}
)

type githubOrganizationTeamsDataSource struct {
	client *Owner
}

type githubOrganizationTeamParentModel struct {
	ID   types.String `tfsdk:"id"`
	Slug types.String `tfsdk:"slug"`
	Name types.String `tfsdk:"name"`
}

type githubOrganizationTeamModel struct {
	ID           types.Int64                        `tfsdk:"id"`
	NodeID       types.String                       `tfsdk:"node_id"`
	Slug         types.String                       `tfsdk:"slug"`
	Name         types.String                       `tfsdk:"name"`
	Description  types.String                       `tfsdk:"description"`
	Privacy      types.String                       `tfsdk:"privacy"`
	Members      types.List                         `tfsdk:"members"`
	Repositories types.List                         `tfsdk:"repositories"`
	Parent       *githubOrganizationTeamParentModel `tfsdk:"parent"`
}

type githubOrganizationTeamsDataSourceModel struct {
	ID             types.String                  `tfsdk:"id"`
	RootTeamsOnly  types.Bool                    `tfsdk:"root_teams_only"`
	SummaryOnly    types.Bool                    `tfsdk:"summary_only"`
	ResultsPerPage types.Int64                   `tfsdk:"results_per_page"`
	Teams          []githubOrganizationTeamModel `tfsdk:"teams"`
}

func NewGithubOrganizationTeamsDataSource() datasource.DataSource {
	return &githubOrganizationTeamsDataSource{}
}

func (d *githubOrganizationTeamsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_teams"
}

func (d *githubOrganizationTeamsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on all GitHub teams of an organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the organization.",
				Computed:    true,
			},
			"root_teams_only": schema.BoolAttribute{
				Description: "Only return root teams (teams with no parent).",
				Optional:    true,
			},
			"summary_only": schema.BoolAttribute{
				Description: "Only return basic team information, omitting members and repositories.",
				Optional:    true,
			},
			"results_per_page": schema.Int64Attribute{
				Description: "Number of teams to fetch per page. Must be between 0 and 100.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 100),
				},
			},
			"teams": schema.ListNestedAttribute{
				Description: "List of teams in the organization.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The team's ID.",
							Computed:    true,
						},
						"node_id": schema.StringAttribute{
							Description: "The team's node ID.",
							Computed:    true,
						},
						"slug": schema.StringAttribute{
							Description: "The team's slug.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The team's name.",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "The team's description.",
							Computed:    true,
						},
						"privacy": schema.StringAttribute{
							Description: "The team's privacy setting (open, closed, secret).",
							Computed:    true,
						},
						"members": schema.ListAttribute{
							Description: "List of team members (only returned when summary_only is false).",
							Computed:    true,
							ElementType: types.StringType,
						},
						"repositories": schema.ListAttribute{
							Description: "List of team repositories (only returned when summary_only is false).",
							Computed:    true,
							ElementType: types.StringType,
						},
						"parent": schema.SingleNestedAttribute{
							Description: "Parent team information.",
							Computed:    true,
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Description: "The parent team's ID.",
									Computed:    true,
								},
								"slug": schema.StringAttribute{
									Description: "The parent team's slug.",
									Computed:    true,
								},
								"name": schema.StringAttribute{
									Description: "The parent team's name.",
									Computed:    true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *githubOrganizationTeamsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubOrganizationTeamsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubOrganizationTeamsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !d.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Unable to read organization teams",
			fmt.Sprintf("This resource can only be used in the context of an organization, %q is a user", d.client.Name()),
		)
		return
	}

	// Get configuration values
	rootTeamsOnly := false
	if !data.RootTeamsOnly.IsNull() && !data.RootTeamsOnly.IsUnknown() {
		rootTeamsOnly = data.RootTeamsOnly.ValueBool()
	}

	summaryOnly := false
	if !data.SummaryOnly.IsNull() && !data.SummaryOnly.IsUnknown() {
		summaryOnly = data.SummaryOnly.ValueBool()
	}

	resultsPerPage := 100
	if !data.ResultsPerPage.IsNull() && !data.ResultsPerPage.IsUnknown() {
		resultsPerPage = int(data.ResultsPerPage.ValueInt64())
	}

	tflog.Debug(ctx, "Reading GitHub organization teams", map[string]any{
		"root_teams_only":  rootTeamsOnly,
		"summary_only":     summaryOnly,
		"results_per_page": resultsPerPage,
	})

	client := d.client.V4Client()
	orgName := d.client.Name()

	var query TeamsQuery

	variables := map[string]any{
		"first":         githubv4.Int(resultsPerPage),
		"login":         githubv4.String(orgName),
		"cursor":        (*githubv4.String)(nil),
		"rootTeamsOnly": githubv4.Boolean(rootTeamsOnly),
		"summaryOnly":   githubv4.Boolean(summaryOnly),
	}

	var allTeams []githubOrganizationTeamModel
	for {
		err := client.Query(d.client.StopContext, &query, variables)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to read GitHub organization teams",
				fmt.Sprintf("Error querying GitHub API: %s", err.Error()),
			)
			return
		}

		teams := d.convertTeamsQueryToModels(ctx, query, summaryOnly)
		allTeams = append(allTeams, teams...)

		if !query.Organization.Teams.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(query.Organization.Teams.PageInfo.EndCursor)
	}

	// Set the ID to the organization ID
	data.ID = types.StringValue(string(query.Organization.ID))
	data.Teams = allTeams

	// Set defaults for optional attributes if not set
	if data.RootTeamsOnly.IsNull() {
		data.RootTeamsOnly = types.BoolValue(false)
	}
	if data.SummaryOnly.IsNull() {
		data.SummaryOnly = types.BoolValue(false)
	}
	if data.ResultsPerPage.IsNull() {
		data.ResultsPerPage = types.Int64Value(100)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *githubOrganizationTeamsDataSource) convertTeamsQueryToModels(ctx context.Context, query TeamsQuery, summaryOnly bool) []githubOrganizationTeamModel {
	teams := query.Organization.Teams.Nodes

	if len(teams) == 0 {
		return []githubOrganizationTeamModel{}
	}

	models := make([]githubOrganizationTeamModel, len(teams))

	for i, team := range teams {
		model := githubOrganizationTeamModel{
			ID:          types.Int64Value(int64(team.DatabaseID)),
			NodeID:      types.StringValue(string(team.ID)),
			Slug:        types.StringValue(string(team.Slug)),
			Name:        types.StringValue(string(team.Name)),
			Description: types.StringValue(string(team.Description)),
			Privacy:     types.StringValue(string(team.Privacy)),
		}

		// Handle members (only if not summary only)
		if !summaryOnly && len(team.Members.Nodes) > 0 {
			members := make([]string, len(team.Members.Nodes))
			for j, member := range team.Members.Nodes {
				members[j] = string(member.Login)
			}
			membersList, diags := types.ListValueFrom(ctx, types.StringType, members)
			if diags.HasError() {
				tflog.Error(ctx, "Error converting members to list", map[string]any{
					"diagnostics": diags,
				})
			} else {
				model.Members = membersList
			}
		} else {
			model.Members = types.ListNull(types.StringType)
		}

		// Handle repositories (only if not summary only)
		if !summaryOnly && len(team.Repositories.Nodes) > 0 {
			repositories := make([]string, len(team.Repositories.Nodes))
			for j, repo := range team.Repositories.Nodes {
				repositories[j] = string(repo.Name)
			}
			repositoriesList, diags := types.ListValueFrom(ctx, types.StringType, repositories)
			if diags.HasError() {
				tflog.Error(ctx, "Error converting repositories to list", map[string]any{
					"diagnostics": diags,
				})
			} else {
				model.Repositories = repositoriesList
			}
		} else {
			model.Repositories = types.ListNull(types.StringType)
		}

		// Handle parent team
		if team.Parent.ID != "" {
			model.Parent = &githubOrganizationTeamParentModel{
				ID:   types.StringValue(string(team.Parent.ID)),
				Slug: types.StringValue(string(team.Parent.Slug)),
				Name: types.StringValue(string(team.Parent.Name)),
			}
		}

		models[i] = model
	}

	return models
}
