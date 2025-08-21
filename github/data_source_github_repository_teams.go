package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubRepositoryTeamsDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryTeamsDataSource{}
)

type githubRepositoryTeamsDataSource struct {
	client *Owner
}

type githubRepositoryTeamModel struct {
	Name       types.String `tfsdk:"name"`
	Slug       types.String `tfsdk:"slug"`
	Permission types.String `tfsdk:"permission"`
}

type githubRepositoryTeamsDataSourceModel struct {
	ID       types.String                `tfsdk:"id"`
	FullName types.String                `tfsdk:"full_name"`
	Name     types.String                `tfsdk:"name"`
	Teams    []githubRepositoryTeamModel `tfsdk:"teams"`
}

func NewGithubRepositoryTeamsDataSource() datasource.DataSource {
	return &githubRepositoryTeamsDataSource{}
}

func (d *githubRepositoryTeamsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_teams"
}

func (d *githubRepositoryTeamsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get teams which have permission on the given repo.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"full_name": schema.StringAttribute{
				Description: "The full name of the repository (owner/repo_name).",
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
			"teams": schema.ListNestedAttribute{
				Description: "List of teams with permission on the repository.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the team.",
							Computed:    true,
						},
						"slug": schema.StringAttribute{
							Description: "The slug of the team.",
							Computed:    true,
						},
						"permission": schema.StringAttribute{
							Description: "The permission level of the team on the repository.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubRepositoryTeamsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubRepositoryTeamsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryTeamsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.client.V3Client()
	owner := d.client.Name()
	var repoName string

	if !data.FullName.IsNull() && !data.FullName.IsUnknown() {
		fullName := data.FullName.ValueString()
		parts := strings.Split(fullName, "/")
		if len(parts) != 2 {
			resp.Diagnostics.AddError(
				"Invalid Repository Full Name",
				fmt.Sprintf("Unexpected full name format %q, expected owner/repo_name", fullName),
			)
			return
		}
		owner = parts[0]
		repoName = parts[1]
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		repoName = data.Name.ValueString()
	}

	if repoName == "" {
		resp.Diagnostics.AddError(
			"Missing Repository Name",
			"One of 'full_name' or 'name' must be provided",
		)
		return
	}

	tflog.Debug(ctx, "Reading GitHub repository teams", map[string]any{
		"owner": owner,
		"repo":  repoName,
	})

	options := github.ListOptions{
		PerPage: maxPerPage,
	}

	var allTeams []githubRepositoryTeamModel
	for {
		teams, respGH, err := client.Repositories.ListTeams(ctx, owner, repoName, &options)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Repository Teams",
				fmt.Sprintf("Unable to read teams for repository %s/%s: %s", owner, repoName, err),
			)
			return
		}

		for _, team := range teams {
			teamModel := githubRepositoryTeamModel{
				Name:       types.StringValue(team.GetName()),
				Slug:       types.StringValue(team.GetSlug()),
				Permission: types.StringValue(team.GetPermission()),
			}
			allTeams = append(allTeams, teamModel)
		}

		if respGH.NextPage == 0 {
			break
		}
		options.Page = respGH.NextPage
	}

	data.ID = types.StringValue(repoName)
	data.Teams = allTeams

	// Set computed values if they weren't provided
	if data.Name.IsNull() || data.Name.IsUnknown() {
		data.Name = types.StringValue(repoName)
	}
	if data.FullName.IsNull() || data.FullName.IsUnknown() {
		data.FullName = types.StringValue(fmt.Sprintf("%s/%s", owner, repoName))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
