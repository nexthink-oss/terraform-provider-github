package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &githubRepositoryPullRequestDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryPullRequestDataSource{}
)

func NewGithubRepositoryPullRequestDataSource() datasource.DataSource {
	return &githubRepositoryPullRequestDataSource{}
}

type githubRepositoryPullRequestDataSource struct {
	client *Owner
}

type githubRepositoryPullRequestDataSourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Owner               types.String `tfsdk:"owner"`
	BaseRepository      types.String `tfsdk:"base_repository"`
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

func (d *githubRepositoryPullRequestDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_pull_request"
}

func (d *githubRepositoryPullRequestDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a single GitHub Pull Request.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"owner": schema.StringAttribute{
				Description: "The GitHub organization or user owning the repository. If not provided, the configured owner will be used.",
				Optional:    true,
			},
			"base_repository": schema.StringAttribute{
				Description: "The name of the repository containing the pull request.",
				Required:    true,
			},
			"number": schema.Int64Attribute{
				Description: "The pull request number.",
				Required:    true,
			},
			"base_ref": schema.StringAttribute{
				Description: "The base branch name the PR is merging into.",
				Computed:    true,
			},
			"base_sha": schema.StringAttribute{
				Description: "The SHA of the base branch the PR is merging into.",
				Computed:    true,
			},
			"body": schema.StringAttribute{
				Description: "The body/content of the pull request.",
				Computed:    true,
			},
			"draft": schema.BoolAttribute{
				Description: "Indicates whether or not the pull request is a draft.",
				Computed:    true,
			},
			"head_owner": schema.StringAttribute{
				Description: "The owner of the repository containing the head branch.",
				Computed:    true,
			},
			"head_ref": schema.StringAttribute{
				Description: "The head branch name the PR is merging from.",
				Computed:    true,
			},
			"head_repository": schema.StringAttribute{
				Description: "The name of the repository containing the head branch.",
				Computed:    true,
			},
			"head_sha": schema.StringAttribute{
				Description: "The SHA of the head branch the PR is merging from.",
				Computed:    true,
			},
			"labels": schema.ListAttribute{
				Description: "List of names of labels on the PR.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"maintainer_can_modify": schema.BoolAttribute{
				Description: "Indicates whether maintainers can modify the pull request.",
				Computed:    true,
			},
			"opened_at": schema.Int64Attribute{
				Description: "Unix timestamp indicating when the pull request was opened.",
				Computed:    true,
			},
			"opened_by": schema.StringAttribute{
				Description: "Username of the PR creator.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "The state of the pull request (open, closed, merged).",
				Computed:    true,
			},
			"title": schema.StringAttribute{
				Description: "The title of the pull request.",
				Computed:    true,
			},
			"updated_at": schema.Int64Attribute{
				Description: "Unix timestamp indicating when the pull request was last updated.",
				Computed:    true,
			},
		},
	}
}

func (d *githubRepositoryPullRequestDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Owner, got: %T", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *githubRepositoryPullRequestDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryPullRequestDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.client.V3Client()

	// Determine owner - use explicit owner if provided, otherwise use configured owner
	owner := d.client.Name()
	if !data.Owner.IsNull() && !data.Owner.IsUnknown() {
		owner = data.Owner.ValueString()
	}

	repository := data.BaseRepository.ValueString()
	number := int(data.Number.ValueInt64())

	pullRequest, _, err := client.PullRequests.Get(ctx, owner, repository, number)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Pull Request",
			fmt.Sprintf("Error reading pull request %d in repository %s/%s: %s", number, owner, repository, err.Error()),
		)
		return
	}

	// Map pull request data to model
	if head := pullRequest.GetHead(); head != nil {
		data.HeadRef = types.StringValue(head.GetRef())
		data.HeadSha = types.StringValue(head.GetSHA())

		if headRepo := head.Repo; headRepo != nil {
			data.HeadRepository = types.StringValue(headRepo.GetName())
		}

		if headUser := head.User; headUser != nil {
			data.HeadOwner = types.StringValue(headUser.GetLogin())
		}
	}

	if base := pullRequest.GetBase(); base != nil {
		data.BaseRef = types.StringValue(base.GetRef())
		data.BaseSha = types.StringValue(base.GetSHA())
	}

	data.Body = types.StringValue(pullRequest.GetBody())
	data.Draft = types.BoolValue(pullRequest.GetDraft())
	data.MaintainerCanModify = types.BoolValue(pullRequest.GetMaintainerCanModify())
	data.OpenedAt = types.Int64Value(pullRequest.GetCreatedAt().Unix())
	data.State = types.StringValue(pullRequest.GetState())
	data.Title = types.StringValue(pullRequest.GetTitle())
	data.UpdatedAt = types.Int64Value(pullRequest.GetUpdatedAt().Unix())

	if user := pullRequest.GetUser(); user != nil {
		data.OpenedBy = types.StringValue(user.GetLogin())
	}

	// Handle labels
	labels := make([]types.String, 0)
	for _, label := range pullRequest.Labels {
		labels = append(labels, types.StringValue(label.GetName()))
	}

	labelsList, diags := types.ListValueFrom(ctx, types.StringType, labels)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Labels = labelsList

	// Set ID using the same pattern as SDKv2 version
	data.ID = types.StringValue(buildThreePartID(owner, repository, strconv.Itoa(number)))

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
