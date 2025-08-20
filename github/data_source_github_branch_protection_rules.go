package github

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/shurcooL/githubv4"
)

var (
	_ datasource.DataSource              = &githubBranchProtectionRulesDataSource{}
	_ datasource.DataSourceWithConfigure = &githubBranchProtectionRulesDataSource{}
)

// PageInfo represents the pagination information used in GraphQL queries

type githubBranchProtectionRulesDataSource struct {
	client *Owner
}

type githubBranchProtectionRuleModel struct {
	Pattern types.String `tfsdk:"pattern"`
}

type githubBranchProtectionRulesDataSourceModel struct {
	Repository types.String                      `tfsdk:"repository"`
	Rules      []githubBranchProtectionRuleModel `tfsdk:"rules"`
}

func NewGithubBranchProtectionRulesDataSource() datasource.DataSource {
	return &githubBranchProtectionRulesDataSource{}
}

func (d *githubBranchProtectionRulesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branch_protection_rules"
}

func (d *githubBranchProtectionRulesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information about a repository branch protection rules.",
		Attributes: map[string]schema.Attribute{
			"repository": schema.StringAttribute{
				Description: "The name of the repository to retrieve branch protection rules for.",
				Required:    true,
			},
			"rules": schema.ListNestedAttribute{
				Description: "List of branch protection rules for the repository.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"pattern": schema.StringAttribute{
							Description: "The pattern of the branch protection rule.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubBranchProtectionRulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubBranchProtectionRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubBranchProtectionRulesDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoName := data.Repository.ValueString()
	orgName := d.client.Name()

	tflog.Debug(ctx, "Reading GitHub branch protection rules", map[string]interface{}{
		"repository": repoName,
		"owner":      orgName,
	})

	client := d.client.V4Client()

	var query struct {
		Repository struct {
			ID                    githubv4.String
			BranchProtectionRules struct {
				Nodes []struct {
					Pattern githubv4.String
				}
				PageInfo PageInfo
			} `graphql:"branchProtectionRules(first:$first, after:$cursor)"`
		} `graphql:"repository(name: $name, owner: $owner)"`
	}

	variables := map[string]any{
		"first":  githubv4.Int(100),
		"name":   githubv4.String(repoName),
		"owner":  githubv4.String(orgName),
		"cursor": (*githubv4.String)(nil),
	}

	var rules []githubBranchProtectionRuleModel
	for {
		err := client.Query(d.client.StopContext, &query, variables)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read GitHub Branch Protection Rules",
				fmt.Sprintf("An unexpected error occurred while reading branch protection rules for repository %s: %s", repoName, err.Error()),
			)
			return
		}

		// Convert rules from GraphQL response to our model
		for _, rule := range query.Repository.BranchProtectionRules.Nodes {
			rules = append(rules, githubBranchProtectionRuleModel{
				Pattern: types.StringValue(string(rule.Pattern)),
			})
		}

		// Check if there are more pages
		if !query.Repository.BranchProtectionRules.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(query.Repository.BranchProtectionRules.PageInfo.EndCursor)
	}

	// Set the repository ID as the data source ID
	data.Rules = rules

	tflog.Debug(ctx, "Successfully read GitHub branch protection rules", map[string]interface{}{
		"repository":  repoName,
		"rules_count": len(rules),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
