package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubRepositoryMilestoneDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryMilestoneDataSource{}
)

type githubRepositoryMilestoneDataSource struct {
	client *Owner
}

type githubRepositoryMilestoneDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Owner       types.String `tfsdk:"owner"`
	Repository  types.String `tfsdk:"repository"`
	Number      types.Int64  `tfsdk:"number"`
	Description types.String `tfsdk:"description"`
	DueDate     types.String `tfsdk:"due_date"`
	State       types.String `tfsdk:"state"`
	Title       types.String `tfsdk:"title"`
}

func NewGithubRepositoryMilestoneDataSource() datasource.DataSource {
	return &githubRepositoryMilestoneDataSource{}
}

func (d *githubRepositoryMilestoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_milestone"
}

func (d *githubRepositoryMilestoneDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub Repository Milestone.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the milestone.",
				Computed:    true,
			},
			"owner": schema.StringAttribute{
				Description: "The repository owner.",
				Required:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"number": schema.Int64Attribute{
				Description: "The number of the milestone.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the milestone.",
				Computed:    true,
			},
			"due_date": schema.StringAttribute{
				Description: "The due date of the milestone in ISO 8601 format (YYYY-MM-DD).",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "The state of the milestone.",
				Computed:    true,
			},
			"title": schema.StringAttribute{
				Description: "The title of the milestone.",
				Computed:    true,
			},
		},
	}
}

func (d *githubRepositoryMilestoneDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubRepositoryMilestoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryMilestoneDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := data.Owner.ValueString()
	repository := data.Repository.ValueString()
	number := int(data.Number.ValueInt64())

	tflog.Debug(ctx, "Reading GitHub repository milestone", map[string]any{
		"owner":      owner,
		"repository": repository,
		"number":     number,
	})

	milestone, _, err := d.client.V3Client().Issues.GetMilestone(ctx, owner, repository, number)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Repository Milestone",
			fmt.Sprintf("An unexpected error occurred while reading the GitHub repository milestone: %s", err.Error()),
		)
		return
	}

	// Set the computed values
	data.ID = types.StringValue(strconv.FormatInt(milestone.GetID(), 10))
	data.Description = types.StringValue(milestone.GetDescription())

	// Handle due date - check if it's not zero before formatting
	if dueOn := milestone.GetDueOn(); !dueOn.IsZero() {
		data.DueDate = types.StringValue(dueOn.Format(layoutISO))
	} else {
		data.DueDate = types.StringValue("")
	}

	data.State = types.StringValue(milestone.GetState())
	data.Title = types.StringValue(milestone.GetTitle())

	tflog.Debug(ctx, "Successfully read GitHub repository milestone", map[string]any{
		"owner":      owner,
		"repository": repository,
		"number":     number,
		"id":         data.ID.ValueString(),
		"title":      data.Title.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
