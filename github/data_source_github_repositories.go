package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubRepositoriesDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoriesDataSource{}
)

type githubRepositoriesDataSource struct {
	client *Owner
}

type githubRepositoriesDataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	Query          types.String `tfsdk:"query"`
	Sort           types.String `tfsdk:"sort"`
	IncludeRepoID  types.Bool   `tfsdk:"include_repo_id"`
	ResultsPerPage types.Int64  `tfsdk:"results_per_page"`
	FullNames      types.List   `tfsdk:"full_names"`
	Names          types.List   `tfsdk:"names"`
	RepoIDs        types.List   `tfsdk:"repo_ids"`
}

func NewGithubRepositoriesDataSource() datasource.DataSource {
	return &githubRepositoriesDataSource{}
}

func (d *githubRepositoriesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repositories"
}

func (d *githubRepositoriesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Search for GitHub repositories",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"query": schema.StringAttribute{
				Description: "Search query for repositories.",
				Required:    true,
			},
			"sort": schema.StringAttribute{
				Description: "Sorts the repositories returned by the search. Can be one of: stars, fork, updated.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("stars", "fork", "updated"),
				},
			},
			"include_repo_id": schema.BoolAttribute{
				Description: "Whether to include repository IDs in the results.",
				Optional:    true,
				Computed:    true,
			},
			"results_per_page": schema.Int64Attribute{
				Description: "Number of results per page (0-1000).",
				Optional:    true,
				Computed:    true,
				Validators: []validator.Int64{
					int64validator.Between(0, 1000),
				},
			},
			"full_names": schema.ListAttribute{
				Description: "List of full repository names (owner/name).",
				Computed:    true,
				ElementType: types.StringType,
			},
			"names": schema.ListAttribute{
				Description: "List of repository names.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"repo_ids": schema.ListAttribute{
				Description: "List of repository IDs. Only populated when include_repo_id is true.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
		},
	}
}

func (d *githubRepositoriesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubRepositoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoriesDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set defaults for optional computed attributes
	if data.Sort.IsNull() {
		data.Sort = types.StringValue("updated")
	}
	if data.IncludeRepoID.IsNull() {
		data.IncludeRepoID = types.BoolValue(false)
	}
	if data.ResultsPerPage.IsNull() {
		data.ResultsPerPage = types.Int64Value(100)
	}

	query := data.Query.ValueString()
	sort := data.Sort.ValueString()
	includeRepoID := data.IncludeRepoID.ValueBool()
	resultsPerPage := int(data.ResultsPerPage.ValueInt64())

	tflog.Debug(ctx, "Searching GitHub repositories", map[string]interface{}{
		"query":            query,
		"sort":             sort,
		"include_repo_id":  includeRepoID,
		"results_per_page": resultsPerPage,
	})

	// Setup search options
	opt := &github.SearchOptions{
		Sort: sort,
		ListOptions: github.ListOptions{
			PerPage: resultsPerPage,
		},
	}

	// Search repositories using GitHub API
	fullNames, names, repoIDs, err := d.searchGithubRepositories(ctx, query, opt)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Search GitHub Repositories",
			fmt.Sprintf("An unexpected error occurred while searching GitHub repositories: %s", err.Error()),
		)
		return
	}

	// Set ID to the query
	data.ID = types.StringValue(query)

	// Convert slices to Framework list values
	fullNamesList, diags := types.ListValueFrom(ctx, types.StringType, fullNames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.FullNames = fullNamesList

	namesList, diags := types.ListValueFrom(ctx, types.StringType, names)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Names = namesList

	// Only set repo_ids if include_repo_id is true
	if includeRepoID {
		repoIDsList, diags := types.ListValueFrom(ctx, types.Int64Type, repoIDs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.RepoIDs = repoIDsList
	} else {
		// Set empty list when not including repo IDs
		data.RepoIDs = types.ListNull(types.Int64Type)
	}

	tflog.Debug(ctx, "Successfully searched GitHub repositories", map[string]interface{}{
		"query":              query,
		"found_repositories": len(fullNames),
		"full_names":         fullNames,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// searchGithubRepositories searches for GitHub repositories using the GitHub API
// This reimplements the same logic as the SDKv2 version
func (d *githubRepositoriesDataSource) searchGithubRepositories(ctx context.Context, query string, opt *github.SearchOptions) ([]string, []string, []int64, error) {
	client := d.client.V3Client()

	fullNames := make([]string, 0)
	names := make([]string, 0)
	repoIDs := make([]int64, 0)

	for {
		results, resp, err := client.Search.Repositories(ctx, query, opt)
		if err != nil {
			return fullNames, names, repoIDs, err
		}

		for _, repo := range results.Repositories {
			fullNames = append(fullNames, repo.GetFullName())
			names = append(names, repo.GetName())
			repoIDs = append(repoIDs, repo.GetID())
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return fullNames, names, repoIDs, nil
}
