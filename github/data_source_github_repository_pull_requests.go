package github

import (
	"context"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubRepositoryPullRequestsDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryPullRequestsDataSource{}
)

type githubRepositoryPullRequestsDataSource struct {
	client *Owner
}

type githubPullRequestModel struct {
	Number              types.Int64  `tfsdk:"number"`
	BaseRef             types.String `tfsdk:"base_ref"`
	BaseSha             types.String `tfsdk:"base_sha"`
	Body                types.String `tfsdk:"body"`
	Draft               types.Bool   `tfsdk:"draft"`
	HeadOwner           types.String `tfsdk:"head_owner"`
	HeadRef             types.String `tfsdk:"head_ref"`
	HeadRepository      types.String `tfsdk:"head_repository"`
	HeadSha             types.String `tfsdk:"head_sha"`
	Labels              types.List   `tfsdk:"labels"`
	MaintainerCanModify types.Bool   `tfsdk:"maintainer_can_modify"`
	OpenedAt            types.Int64  `tfsdk:"opened_at"`
	OpenedBy            types.String `tfsdk:"opened_by"`
	State               types.String `tfsdk:"state"`
	Title               types.String `tfsdk:"title"`
	UpdatedAt           types.Int64  `tfsdk:"updated_at"`
}

type githubRepositoryPullRequestsDataSourceModel struct {
	ID             types.String             `tfsdk:"id"`
	Owner          types.String             `tfsdk:"owner"`
	BaseRepository types.String             `tfsdk:"base_repository"`
	BaseRef        types.String             `tfsdk:"base_ref"`
	HeadRef        types.String             `tfsdk:"head_ref"`
	SortBy         types.String             `tfsdk:"sort_by"`
	SortDirection  types.String             `tfsdk:"sort_direction"`
	State          types.String             `tfsdk:"state"`
	Results        []githubPullRequestModel `tfsdk:"results"`
}

func NewGithubRepositoryPullRequestsDataSource() datasource.DataSource {
	return &githubRepositoryPullRequestsDataSource{}
}

func (d *githubRepositoryPullRequestsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_pull_requests"
}

func (d *githubRepositoryPullRequestsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on multiple GitHub Pull Requests.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"owner": schema.StringAttribute{
				Description: "The owner of the repository.",
				Optional:    true,
			},
			"base_repository": schema.StringAttribute{
				Description: "The repository name.",
				Required:    true,
			},
			"base_ref": schema.StringAttribute{
				Description: "The name of the branch you want your changes pulled into.",
				Optional:    true,
			},
			"head_ref": schema.StringAttribute{
				Description: "The name of the branch where your changes are implemented.",
				Optional:    true,
			},
			"sort_by": schema.StringAttribute{
				Description: "What to sort results by. Can be either `created`, `updated`, `popularity` (comment count) or `long-running` (age, filtering by pulls updated in the last month). Default: `created`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("created", "updated", "popularity", "long-running"),
				},
			},
			"sort_direction": schema.StringAttribute{
				Description: "The direction to sort the results by. Default: `asc`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("asc", "desc"),
				},
			},
			"state": schema.StringAttribute{
				Description: "Either `open`, `closed`, or `all` to filter by state. Default: `open`.",
				Optional:    true,
				Computed:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("open", "closed", "all"),
				},
			},
			"results": schema.ListNestedAttribute{
				Description: "List of pull requests matching the criteria.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"number": schema.Int64Attribute{
							Description: "Per-repository, monotonically increasing ID of this PR",
							Computed:    true,
						},
						"base_ref": schema.StringAttribute{
							Description: "The name of the branch you want your changes pulled into.",
							Computed:    true,
						},
						"base_sha": schema.StringAttribute{
							Description: "The SHA of the base branch.",
							Computed:    true,
						},
						"body": schema.StringAttribute{
							Description: "The body of the pull request.",
							Computed:    true,
						},
						"draft": schema.BoolAttribute{
							Description: "Whether the pull request is a draft.",
							Computed:    true,
						},
						"head_owner": schema.StringAttribute{
							Description: "The owner of the head repository.",
							Computed:    true,
						},
						"head_ref": schema.StringAttribute{
							Description: "The name of the branch where your changes are implemented.",
							Computed:    true,
						},
						"head_repository": schema.StringAttribute{
							Description: "The name of the head repository.",
							Computed:    true,
						},
						"head_sha": schema.StringAttribute{
							Description: "The SHA of the head branch.",
							Computed:    true,
						},
						"labels": schema.ListAttribute{
							Description: "List of names of labels on the PR",
							ElementType: types.StringType,
							Computed:    true,
						},
						"maintainer_can_modify": schema.BoolAttribute{
							Description: "Whether maintainers can modify the pull request.",
							Computed:    true,
						},
						"opened_at": schema.Int64Attribute{
							Description: "The timestamp when the pull request was opened.",
							Computed:    true,
						},
						"opened_by": schema.StringAttribute{
							Description: "Username of the PR creator",
							Computed:    true,
						},
						"state": schema.StringAttribute{
							Description: "The state of the pull request.",
							Computed:    true,
						},
						"title": schema.StringAttribute{
							Description: "The title of the pull request.",
							Computed:    true,
						},
						"updated_at": schema.Int64Attribute{
							Description: "The timestamp when the pull request was last updated.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubRepositoryPullRequestsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			"Expected *github.Owner, got: %T. Please report this issue to the provider developers.",
		)
		return
	}

	d.client = client
}

func (d *githubRepositoryPullRequestsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryPullRequestsDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set defaults if not provided
	if data.SortBy.IsNull() {
		data.SortBy = types.StringValue("created")
	}
	if data.SortDirection.IsNull() {
		data.SortDirection = types.StringValue("asc")
	}
	if data.State.IsNull() {
		data.State = types.StringValue("open")
	}

	// Determine owner
	owner := d.client.Name()
	if !data.Owner.IsNull() && !data.Owner.IsUnknown() {
		owner = data.Owner.ValueString()
	}

	baseRepository := data.BaseRepository.ValueString()
	state := data.State.ValueString()
	head := ""
	if !data.HeadRef.IsNull() && !data.HeadRef.IsUnknown() {
		head = data.HeadRef.ValueString()
	}
	base := ""
	if !data.BaseRef.IsNull() && !data.BaseRef.IsUnknown() {
		base = data.BaseRef.ValueString()
	}
	sort := data.SortBy.ValueString()
	direction := data.SortDirection.ValueString()

	options := &github.PullRequestListOptions{
		ListOptions: github.ListOptions{PerPage: maxPerPage},
		State:       state,
		Head:        head,
		Base:        base,
		Sort:        sort,
		Direction:   direction,
	}

	var results []githubPullRequestModel

	for {
		pullRequests, apiResp, err := d.client.V3Client().PullRequests.List(ctx, owner, baseRepository, options)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read GitHub Repository Pull Requests",
				err.Error(),
			)
			return
		}

		for _, pullRequest := range pullRequests {
			// Convert labels to a list of strings
			labels := []string{}
			for _, label := range pullRequest.Labels {
				labels = append(labels, label.GetName())
			}

			labelsList, diags := types.ListValueFrom(ctx, types.StringType, labels)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			result := githubPullRequestModel{
				Number:              types.Int64Value(int64(pullRequest.GetNumber())),
				Body:                types.StringValue(pullRequest.GetBody()),
				Draft:               types.BoolValue(pullRequest.GetDraft()),
				MaintainerCanModify: types.BoolValue(pullRequest.GetMaintainerCanModify()),
				OpenedAt:            types.Int64Value(pullRequest.GetCreatedAt().Unix()),
				State:               types.StringValue(pullRequest.GetState()),
				Title:               types.StringValue(pullRequest.GetTitle()),
				UpdatedAt:           types.Int64Value(pullRequest.GetUpdatedAt().Unix()),
				Labels:              labelsList,
			}

			if head := pullRequest.GetHead(); head != nil {
				result.HeadRef = types.StringValue(head.GetRef())
				result.HeadSha = types.StringValue(head.GetSHA())

				if headRepo := head.GetRepo(); headRepo != nil {
					result.HeadRepository = types.StringValue(headRepo.GetName())

					if headOwner := headRepo.GetOwner(); headOwner != nil {
						result.HeadOwner = types.StringValue(headOwner.GetLogin())
					} else {
						result.HeadOwner = types.StringNull()
					}
				} else {
					result.HeadRepository = types.StringNull()
					result.HeadOwner = types.StringNull()
				}
			} else {
				result.HeadRef = types.StringNull()
				result.HeadSha = types.StringNull()
				result.HeadRepository = types.StringNull()
				result.HeadOwner = types.StringNull()
			}

			if base := pullRequest.GetBase(); base != nil {
				result.BaseRef = types.StringValue(base.GetRef())
				result.BaseSha = types.StringValue(base.GetSHA())
			} else {
				result.BaseRef = types.StringNull()
				result.BaseSha = types.StringNull()
			}

			if user := pullRequest.GetUser(); user != nil {
				result.OpenedBy = types.StringValue(user.GetLogin())
			} else {
				result.OpenedBy = types.StringNull()
			}

			results = append(results, result)
		}

		if apiResp.NextPage == 0 {
			break
		}

		options.Page = apiResp.NextPage
	}

	// Set the results
	data.Results = results

	// Generate ID based on input parameters
	id := strings.Join([]string{
		owner,
		baseRepository,
		data.State.ValueString(),
		data.HeadRef.ValueString(),
		data.BaseRef.ValueString(),
		data.SortBy.ValueString(),
		data.SortDirection.ValueString(),
	}, "/")
	data.ID = types.StringValue(id)

	// Set the owner if it wasn't explicitly provided
	if data.Owner.IsNull() {
		data.Owner = types.StringValue(owner)
	}

	tflog.Debug(ctx, "Read GitHub repository pull requests", map[string]interface{}{
		"owner":           owner,
		"base_repository": baseRepository,
		"results_count":   len(results),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
