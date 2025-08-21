package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/shurcooL/githubv4"
)

var (
	_ datasource.DataSource              = &githubOrganizationDataSource{}
	_ datasource.DataSourceWithConfigure = &githubOrganizationDataSource{}
)

type githubOrganizationDataSource struct {
	client *Owner
}

type githubOrganizationDataSourceModel struct {
	ID                                                    types.String `tfsdk:"id"`
	Name                                                  types.String `tfsdk:"name"`
	IgnoreArchivedRepos                                   types.Bool   `tfsdk:"ignore_archived_repos"`
	Orgname                                               types.String `tfsdk:"orgname"`
	NodeID                                                types.String `tfsdk:"node_id"`
	Login                                                 types.String `tfsdk:"login"`
	Description                                           types.String `tfsdk:"description"`
	Plan                                                  types.String `tfsdk:"plan"`
	Repositories                                          types.List   `tfsdk:"repositories"`
	Members                                               types.List   `tfsdk:"members"`
	Users                                                 types.List   `tfsdk:"users"`
	DefaultRepositoryPermission                           types.String `tfsdk:"default_repository_permission"`
	MembersCanCreateRepositories                          types.Bool   `tfsdk:"members_can_create_repositories"`
	TwoFactorRequirementEnabled                           types.Bool   `tfsdk:"two_factor_requirement_enabled"`
	MembersAllowedRepositoryCreationType                  types.String `tfsdk:"members_allowed_repository_creation_type"`
	MembersCanCreatePublicRepositories                    types.Bool   `tfsdk:"members_can_create_public_repositories"`
	MembersCanCreatePrivateRepositories                   types.Bool   `tfsdk:"members_can_create_private_repositories"`
	MembersCanCreateInternalRepositories                  types.Bool   `tfsdk:"members_can_create_internal_repositories"`
	MembersCanCreatePages                                 types.Bool   `tfsdk:"members_can_create_pages"`
	MembersCanCreatePublicPages                           types.Bool   `tfsdk:"members_can_create_public_pages"`
	MembersCanCreatePrivatePages                          types.Bool   `tfsdk:"members_can_create_private_pages"`
	MembersCanForkPrivateRepositories                     types.Bool   `tfsdk:"members_can_fork_private_repositories"`
	WebCommitSignoffRequired                              types.Bool   `tfsdk:"web_commit_signoff_required"`
	AdvancedSecurityEnabledForNewRepositories             types.Bool   `tfsdk:"advanced_security_enabled_for_new_repositories"`
	DependabotAlertsEnabledForNewRepositories             types.Bool   `tfsdk:"dependabot_alerts_enabled_for_new_repositories"`
	DependabotSecurityUpdatesEnabledForNewRepositories    types.Bool   `tfsdk:"dependabot_security_updates_enabled_for_new_repositories"`
	DependencyGraphEnabledForNewRepositories              types.Bool   `tfsdk:"dependency_graph_enabled_for_new_repositories"`
	SecretScanningEnabledForNewRepositories               types.Bool   `tfsdk:"secret_scanning_enabled_for_new_repositories"`
	SecretScanningPushProtectionEnabledForNewRepositories types.Bool   `tfsdk:"secret_scanning_push_protection_enabled_for_new_repositories"`
	SummaryOnly                                           types.Bool   `tfsdk:"summary_only"`
}

func NewGithubOrganizationDataSource() datasource.DataSource {
	return &githubOrganizationDataSource{}
}

func (d *githubOrganizationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *githubOrganizationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get an organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the organization.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the organization.",
				Required:    true,
			},
			"ignore_archived_repos": schema.BoolAttribute{
				Description: "Whether to ignore archived repositories.",
				Optional:    true,
			},
			"orgname": schema.StringAttribute{
				Description: "The organization name.",
				Computed:    true,
			},
			"node_id": schema.StringAttribute{
				Description: "The node ID of the organization.",
				Computed:    true,
			},
			"login": schema.StringAttribute{
				Description: "The organization's login.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The organization's description.",
				Computed:    true,
			},
			"plan": schema.StringAttribute{
				Description: "The organization's plan.",
				Computed:    true,
			},
			"repositories": schema.ListAttribute{
				Description: "List of organization repositories.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"members": schema.ListAttribute{
				Description:        "List of organization members. Use `users` instead by replacing `github_organization.example.members` to `github_organization.example.users[*].login`. Expect this field to be removed in next major version.",
				DeprecationMessage: "Use `users` instead by replacing `github_organization.example.members` to `github_organization.example.users[*].login`. Expect this field to be removed in next major version.",
				Computed:           true,
				ElementType:        types.StringType,
			},
			"users": schema.ListAttribute{
				Description: "List of organization users with their details.",
				Computed:    true,
				ElementType: types.MapType{ElemType: types.StringType},
			},
			"default_repository_permission": schema.StringAttribute{
				Description: "The default repository permission for organization members.",
				Computed:    true,
			},
			"members_can_create_repositories": schema.BoolAttribute{
				Description: "Whether organization members can create repositories.",
				Computed:    true,
			},
			"two_factor_requirement_enabled": schema.BoolAttribute{
				Description: "Whether two-factor authentication is required for organization members.",
				Computed:    true,
			},
			"members_allowed_repository_creation_type": schema.StringAttribute{
				Description: "The type of repositories organization members are allowed to create.",
				Computed:    true,
			},
			"members_can_create_public_repositories": schema.BoolAttribute{
				Description: "Whether organization members can create public repositories.",
				Computed:    true,
			},
			"members_can_create_private_repositories": schema.BoolAttribute{
				Description: "Whether organization members can create private repositories.",
				Computed:    true,
			},
			"members_can_create_internal_repositories": schema.BoolAttribute{
				Description: "Whether organization members can create internal repositories.",
				Computed:    true,
			},
			"members_can_create_pages": schema.BoolAttribute{
				Description: "Whether organization members can create GitHub Pages sites.",
				Computed:    true,
			},
			"members_can_create_public_pages": schema.BoolAttribute{
				Description: "Whether organization members can create public GitHub Pages sites.",
				Computed:    true,
			},
			"members_can_create_private_pages": schema.BoolAttribute{
				Description: "Whether organization members can create private GitHub Pages sites.",
				Computed:    true,
			},
			"members_can_fork_private_repositories": schema.BoolAttribute{
				Description: "Whether organization members can fork private repositories.",
				Computed:    true,
			},
			"web_commit_signoff_required": schema.BoolAttribute{
				Description: "Whether web commit signoff is required.",
				Computed:    true,
			},
			"advanced_security_enabled_for_new_repositories": schema.BoolAttribute{
				Description: "Whether GitHub Advanced Security is enabled for new repositories.",
				Computed:    true,
			},
			"dependabot_alerts_enabled_for_new_repositories": schema.BoolAttribute{
				Description: "Whether Dependabot alerts are enabled for new repositories.",
				Computed:    true,
			},
			"dependabot_security_updates_enabled_for_new_repositories": schema.BoolAttribute{
				Description: "Whether Dependabot security updates are enabled for new repositories.",
				Computed:    true,
			},
			"dependency_graph_enabled_for_new_repositories": schema.BoolAttribute{
				Description: "Whether dependency graph is enabled for new repositories.",
				Computed:    true,
			},
			"secret_scanning_enabled_for_new_repositories": schema.BoolAttribute{
				Description: "Whether secret scanning is enabled for new repositories.",
				Computed:    true,
			},
			"secret_scanning_push_protection_enabled_for_new_repositories": schema.BoolAttribute{
				Description: "Whether secret scanning push protection is enabled for new repositories.",
				Computed:    true,
			},
			"summary_only": schema.BoolAttribute{
				Description: "Whether to return only summary data (without members, repositories, and other detailed information).",
				Optional:    true,
			},
		},
	}
}

func (d *githubOrganizationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubOrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubOrganizationDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()

	tflog.Debug(ctx, "Reading GitHub organization", map[string]any{
		"name": name,
	})

	client3 := d.client.V3Client()
	client4 := d.client.V4Client()

	organization, _, err := client3.Organizations.Get(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Organization",
			fmt.Sprintf("An unexpected error occurred while reading the GitHub organization %s: %s", name, err.Error()),
		)
		return
	}

	var planName string
	if plan := organization.GetPlan(); plan != nil {
		planName = plan.GetName()
	}

	// Set basic organization details
	data.ID = types.StringValue(strconv.FormatInt(organization.GetID(), 10))
	data.Login = types.StringValue(organization.GetLogin())
	data.Orgname = types.StringValue(name)
	data.NodeID = types.StringValue(organization.GetNodeID())
	data.Description = types.StringValue(organization.GetDescription())
	data.Plan = types.StringValue(planName)

	// Handle summary_only flag
	summaryOnly := false
	if !data.SummaryOnly.IsNull() && !data.SummaryOnly.IsUnknown() {
		summaryOnly = data.SummaryOnly.ValueBool()
	}

	// If not summary only, fetch detailed information
	if !summaryOnly {
		// Fetch repositories
		opts := &github.RepositoryListByOrgOptions{
			ListOptions: github.ListOptions{PerPage: maxPerPage, Page: 1},
		}

		var repoList []string
		var allRepos []*github.Repository

		for {
			repos, httpResp, err := client3.Repositories.ListByOrg(ctx, name, opts)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Read GitHub Organization Repositories",
					fmt.Sprintf("An unexpected error occurred while reading repositories for GitHub organization %s: %s", name, err.Error()),
				)
				return
			}
			allRepos = append(allRepos, repos...)

			opts.Page = httpResp.NextPage

			if httpResp.NextPage == 0 {
				break
			}
		}

		// Handle ignore_archived_repos flag
		ignoreArchiveRepos := false
		if !data.IgnoreArchivedRepos.IsNull() && !data.IgnoreArchivedRepos.IsUnknown() {
			ignoreArchiveRepos = data.IgnoreArchivedRepos.ValueBool()
		}

		for _, repo := range allRepos {
			if ignoreArchiveRepos && repo.GetArchived() {
				continue
			}
			repoList = append(repoList, repo.GetFullName())
		}

		// Fetch organization members using GraphQL
		var query struct {
			Organization struct {
				MembersWithRole struct {
					Edges []struct {
						Role githubv4.String
						Node struct {
							Id    githubv4.String
							Login githubv4.String
							Email githubv4.String
						}
					}
					PageInfo struct {
						EndCursor   githubv4.String
						HasNextPage bool
					}
				} `graphql:"membersWithRole(first: 100, after: $after)"`
			} `graphql:"organization(login: $login)"`
		}

		variables := map[string]any{
			"login": githubv4.String(name),
			"after": (*githubv4.String)(nil),
		}

		var members []string
		var users []map[string]string

		for {
			err := client4.Query(ctx, &query, variables)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Read GitHub Organization Members",
					fmt.Sprintf("An unexpected error occurred while reading members for GitHub organization %s: %s", name, err.Error()),
				)
				return
			}

			for _, edge := range query.Organization.MembersWithRole.Edges {
				members = append(members, string(edge.Node.Login))
				users = append(users, map[string]string{
					"id":    string(edge.Node.Id),
					"login": string(edge.Node.Login),
					"email": string(edge.Node.Email),
					"role":  string(edge.Role),
				})
			}

			if !query.Organization.MembersWithRole.PageInfo.HasNextPage {
				break
			}
			variables["after"] = githubv4.NewString(query.Organization.MembersWithRole.PageInfo.EndCursor)
		}

		// Convert slices to Framework list values
		reposList, diags := types.ListValueFrom(ctx, types.StringType, repoList)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Repositories = reposList

		membersList, diags := types.ListValueFrom(ctx, types.StringType, members)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Members = membersList

		usersList, diags := types.ListValueFrom(ctx, types.MapType{ElemType: types.StringType}, users)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Users = usersList

		// Set organization settings
		data.TwoFactorRequirementEnabled = types.BoolValue(organization.GetTwoFactorRequirementEnabled())
		data.DefaultRepositoryPermission = types.StringValue(organization.GetDefaultRepoPermission())
		data.MembersCanCreateRepositories = types.BoolValue(organization.GetMembersCanCreateRepos())
		data.MembersAllowedRepositoryCreationType = types.StringValue(organization.GetMembersAllowedRepositoryCreationType())
		data.MembersCanCreatePublicRepositories = types.BoolValue(organization.GetMembersCanCreatePublicRepos())
		data.MembersCanCreatePrivateRepositories = types.BoolValue(organization.GetMembersCanCreatePrivateRepos())
		data.MembersCanCreateInternalRepositories = types.BoolValue(organization.GetMembersCanCreateInternalRepos())
		data.MembersCanForkPrivateRepositories = types.BoolValue(organization.GetMembersCanCreatePrivateRepos())
		data.WebCommitSignoffRequired = types.BoolValue(organization.GetWebCommitSignoffRequired())
		data.MembersCanCreatePages = types.BoolValue(organization.GetMembersCanCreatePages())
		data.MembersCanCreatePublicPages = types.BoolValue(organization.GetMembersCanCreatePublicPages())
		data.MembersCanCreatePrivatePages = types.BoolValue(organization.GetMembersCanCreatePrivatePages())
		data.AdvancedSecurityEnabledForNewRepositories = types.BoolValue(organization.GetAdvancedSecurityEnabledForNewRepos())
		data.DependabotAlertsEnabledForNewRepositories = types.BoolValue(organization.GetDependabotAlertsEnabledForNewRepos())
		data.DependabotSecurityUpdatesEnabledForNewRepositories = types.BoolValue(organization.GetDependabotSecurityUpdatesEnabledForNewRepos())
		data.DependencyGraphEnabledForNewRepositories = types.BoolValue(organization.GetDependencyGraphEnabledForNewRepos())
		data.SecretScanningEnabledForNewRepositories = types.BoolValue(organization.GetSecretScanningEnabledForNewRepos())
		data.SecretScanningPushProtectionEnabledForNewRepositories = types.BoolValue(organization.GetSecretScanningPushProtectionEnabledForNewRepos())
	}

	tflog.Debug(ctx, "Successfully read GitHub organization", map[string]any{
		"name":         name,
		"id":           data.ID.ValueString(),
		"login":        data.Login.ValueString(),
		"summary_only": summaryOnly,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
