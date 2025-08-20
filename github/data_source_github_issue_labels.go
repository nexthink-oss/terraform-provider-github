package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ datasource.DataSource              = &githubIssueLabelsDataSource{}
	_ datasource.DataSourceWithConfigure = &githubIssueLabelsDataSource{}
)

type githubIssueLabelsDataSource struct {
	client *Owner
}

type githubLabelModel struct {
	Name        types.String `tfsdk:"name"`
	Color       types.String `tfsdk:"color"`
	Description types.String `tfsdk:"description"`
	URL         types.String `tfsdk:"url"`
}

type githubIssueLabelsDataSourceModel struct {
	ID         types.String       `tfsdk:"id"`
	Repository types.String       `tfsdk:"repository"`
	Labels     []githubLabelModel `tfsdk:"labels"`
}

func NewGithubIssueLabelsDataSource() datasource.DataSource {
	return &githubIssueLabelsDataSource{}
}

func (d *githubIssueLabelsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_issue_labels"
}

func (d *githubIssueLabelsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get the labels for a given repository.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The repository name.",
				Required:    true,
			},
			"labels": schema.ListNestedAttribute{
				Description: "List of labels in the repository.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the label.",
							Computed:    true,
						},
						"color": schema.StringAttribute{
							Description: "The color of the label (without the leading #).",
							Computed:    true,
						},
						"description": schema.StringAttribute{
							Description: "A short description of the label.",
							Computed:    true,
						},
						"url": schema.StringAttribute{
							Description: "The URL of the label.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubIssueLabelsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubIssueLabelsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubIssueLabelsDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	owner := d.client.Name()
	repository := data.Repository.ValueString()

	opts := &github.ListOptions{
		PerPage: 100,
	}

	var allLabels []githubLabelModel

	for {
		labels, apiResp, err := d.client.V3Client().Issues.ListLabels(ctx, owner, repository, opts)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read GitHub Issue Labels",
				fmt.Sprintf("Unable to read GitHub Labels (Owner: %q/Repository: %q): %s", owner, repository, err.Error()),
			)
			return
		}

		for _, label := range labels {
			labelModel := githubLabelModel{
				Name:        types.StringValue(label.GetName()),
				Color:       types.StringValue(label.GetColor()),
				Description: types.StringValue(label.GetDescription()),
				URL:         types.StringValue(label.GetURL()),
			}
			allLabels = append(allLabels, labelModel)
		}

		if apiResp.NextPage == 0 {
			break
		}
		opts.Page = apiResp.NextPage
	}

	// Set the results
	data.Labels = allLabels
	data.ID = types.StringValue(repository)

	tflog.Debug(ctx, "Read GitHub issue labels", map[string]interface{}{
		"owner":      owner,
		"repository": repository,
		"count":      len(allLabels),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
