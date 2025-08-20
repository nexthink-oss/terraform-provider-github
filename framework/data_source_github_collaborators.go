package framework

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ datasource.DataSource              = &githubCollaboratorsDataSource{}
	_ datasource.DataSourceWithConfigure = &githubCollaboratorsDataSource{}
)

type githubCollaboratorsDataSource struct {
	client *githubpkg.Owner
}

type githubCollaboratorModel struct {
	Login             types.String `tfsdk:"login"`
	ID                types.Int64  `tfsdk:"id"`
	URL               types.String `tfsdk:"url"`
	HtmlURL           types.String `tfsdk:"html_url"`
	FollowersURL      types.String `tfsdk:"followers_url"`
	FollowingURL      types.String `tfsdk:"following_url"`
	GistsURL          types.String `tfsdk:"gists_url"`
	StarredURL        types.String `tfsdk:"starred_url"`
	SubscriptionsURL  types.String `tfsdk:"subscriptions_url"`
	OrganizationsURL  types.String `tfsdk:"organizations_url"`
	ReposURL          types.String `tfsdk:"repos_url"`
	EventsURL         types.String `tfsdk:"events_url"`
	ReceivedEventsURL types.String `tfsdk:"received_events_url"`
	Type              types.String `tfsdk:"type"`
	SiteAdmin         types.Bool   `tfsdk:"site_admin"`
	Permission        types.String `tfsdk:"permission"`
}

type githubCollaboratorsDataSourceModel struct {
	ID           types.String              `tfsdk:"id"`
	Owner        types.String              `tfsdk:"owner"`
	Repository   types.String              `tfsdk:"repository"`
	Affiliation  types.String              `tfsdk:"affiliation"`
	Permission   types.String              `tfsdk:"permission"`
	Collaborator []githubCollaboratorModel `tfsdk:"collaborator"`
}

// getPermission normalizes permission values from GitHub API
func getPermissionNormalized(permission string) string {
	// Permissions for some GitHub API routes are expressed as "read",
	// "write", and "admin"; in other places, they are expressed as "pull",
	// "push", and "admin".
	switch permission {
	case "read":
		return "pull"
	case "write":
		return "push"
	default:
		return permission
	}
}

func NewGithubCollaboratorsDataSource() datasource.DataSource {
	return &githubCollaboratorsDataSource{}
}

func (d *githubCollaboratorsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collaborators"
}

func (d *githubCollaboratorsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get the collaborators for a given repository.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"owner": schema.StringAttribute{
				Description: "The owner of the repository.",
				Required:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"affiliation": schema.StringAttribute{
				Description: "Filter collaborators by their affiliation. Can be one of: all, direct, outside.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("all", "direct", "outside"),
				},
			},
			"permission": schema.StringAttribute{
				Description: "Filter collaborators by their permission level. Can be one of: pull, triage, push, maintain, admin.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("pull", "triage", "push", "maintain", "admin"),
				},
			},
			"collaborator": schema.ListNestedAttribute{
				Description: "List of collaborators for the repository.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"login": schema.StringAttribute{
							Description: "The login name of the collaborator.",
							Computed:    true,
						},
						"id": schema.Int64Attribute{
							Description: "The ID of the collaborator.",
							Computed:    true,
						},
						"url": schema.StringAttribute{
							Description: "The API URL of the collaborator.",
							Computed:    true,
						},
						"html_url": schema.StringAttribute{
							Description: "The GitHub URL of the collaborator.",
							Computed:    true,
						},
						"followers_url": schema.StringAttribute{
							Description: "The followers URL of the collaborator.",
							Computed:    true,
						},
						"following_url": schema.StringAttribute{
							Description: "The following URL of the collaborator.",
							Computed:    true,
						},
						"gists_url": schema.StringAttribute{
							Description: "The gists URL of the collaborator.",
							Computed:    true,
						},
						"starred_url": schema.StringAttribute{
							Description: "The starred URL of the collaborator.",
							Computed:    true,
						},
						"subscriptions_url": schema.StringAttribute{
							Description: "The subscriptions URL of the collaborator.",
							Computed:    true,
						},
						"organizations_url": schema.StringAttribute{
							Description: "The organizations URL of the collaborator.",
							Computed:    true,
						},
						"repos_url": schema.StringAttribute{
							Description: "The repositories URL of the collaborator.",
							Computed:    true,
						},
						"events_url": schema.StringAttribute{
							Description: "The events URL of the collaborator.",
							Computed:    true,
						},
						"received_events_url": schema.StringAttribute{
							Description: "The received events URL of the collaborator.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of the collaborator (User or Organization).",
							Computed:    true,
						},
						"site_admin": schema.BoolAttribute{
							Description: "Whether the collaborator is a site administrator.",
							Computed:    true,
						},
						"permission": schema.StringAttribute{
							Description: "The permission level of the collaborator.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubCollaboratorsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubCollaboratorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubCollaboratorsDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := data.Owner.ValueString()
	repo := data.Repository.ValueString()

	// Set defaults for optional fields
	affiliation := "all"
	if !data.Affiliation.IsNull() && !data.Affiliation.IsUnknown() {
		affiliation = data.Affiliation.ValueString()
	}

	permission := ""
	if !data.Permission.IsNull() && !data.Permission.IsUnknown() {
		permission = data.Permission.ValueString()
	}

	tflog.Debug(ctx, "Reading GitHub collaborators", map[string]interface{}{
		"owner":       owner,
		"repository":  repo,
		"affiliation": affiliation,
		"permission":  permission,
	})

	client := d.client.V3Client()

	options := &github.ListCollaboratorsOptions{
		Affiliation: affiliation,
		Permission:  permission,
		ListOptions: github.ListOptions{
			PerPage: 100, // Use max per page for efficiency
		},
	}

	// Set ID based on parameters
	if len(permission) == 0 {
		data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", owner, repo, affiliation))
	} else {
		data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s/%s", owner, repo, affiliation, permission))
	}

	var totalCollaborators []githubCollaboratorModel

	// Handle pagination to get all collaborators
	for {
		collaborators, resp_api, err := client.Repositories.ListCollaborators(ctx, owner, repo, options)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read GitHub Collaborators",
				fmt.Sprintf("An unexpected error occurred while reading collaborators for repository %s/%s: %s", owner, repo, err.Error()),
			)
			return
		}

		// Process the current page of collaborators
		for _, c := range collaborators {
			collaborator := githubCollaboratorModel{
				Login:             types.StringValue(c.GetLogin()),
				ID:                types.Int64Value(c.GetID()),
				URL:               types.StringValue(c.GetURL()),
				HtmlURL:           types.StringValue(c.GetHTMLURL()),
				FollowersURL:      types.StringValue(c.GetFollowersURL()),
				FollowingURL:      types.StringValue(c.GetFollowingURL()),
				GistsURL:          types.StringValue(c.GetGistsURL()),
				StarredURL:        types.StringValue(c.GetStarredURL()),
				SubscriptionsURL:  types.StringValue(c.GetSubscriptionsURL()),
				OrganizationsURL:  types.StringValue(c.GetOrganizationsURL()),
				ReposURL:          types.StringValue(c.GetReposURL()),
				EventsURL:         types.StringValue(c.GetEventsURL()),
				ReceivedEventsURL: types.StringValue(c.GetReceivedEventsURL()),
				Type:              types.StringValue(c.GetType()),
				SiteAdmin:         types.BoolValue(c.GetSiteAdmin()),
				Permission:        types.StringValue(getPermissionNormalized(c.GetRoleName())),
			}
			totalCollaborators = append(totalCollaborators, collaborator)
		}

		// Check if there are more pages
		if resp_api.NextPage == 0 {
			break
		}
		options.Page = resp_api.NextPage
	}

	// Set the computed values
	data.Owner = types.StringValue(owner)
	data.Repository = types.StringValue(repo)
	data.Affiliation = types.StringValue(affiliation)
	data.Permission = types.StringValue(permission)
	data.Collaborator = totalCollaborators

	tflog.Debug(ctx, "Successfully read GitHub collaborators", map[string]interface{}{
		"owner":               owner,
		"repository":          repo,
		"affiliation":         affiliation,
		"permission":          permission,
		"collaborators_count": len(totalCollaborators),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
