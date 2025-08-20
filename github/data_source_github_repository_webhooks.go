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
	_ datasource.DataSource              = &githubRepositoryWebhooksDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryWebhooksDataSource{}
)

type githubRepositoryWebhooksDataSource struct {
	client *Owner
}

type githubRepositoryWebhookModel struct {
	ID     types.Int64  `tfsdk:"id"`
	Type   types.String `tfsdk:"type"`
	Name   types.String `tfsdk:"name"`
	URL    types.String `tfsdk:"url"`
	Active types.Bool   `tfsdk:"active"`
}

type githubRepositoryWebhooksDataSourceModel struct {
	ID         types.String                   `tfsdk:"id"`
	Repository types.String                   `tfsdk:"repository"`
	Webhooks   []githubRepositoryWebhookModel `tfsdk:"webhooks"`
}

func NewGithubRepositoryWebhooksDataSource() datasource.DataSource {
	return &githubRepositoryWebhooksDataSource{}
}

func (d *githubRepositoryWebhooksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_webhooks"
}

func (d *githubRepositoryWebhooksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on all GitHub webhooks of the repository.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"webhooks": schema.ListNestedAttribute{
				Description: "List of webhooks for the repository.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The ID of the webhook.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of the webhook.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the webhook.",
							Computed:    true,
						},
						"url": schema.StringAttribute{
							Description: "The URL of the webhook.",
							Computed:    true,
						},
						"active": schema.BoolAttribute{
							Description: "Whether the webhook is active.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubRepositoryWebhooksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubRepositoryWebhooksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryWebhooksDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.client.V3Client()
	owner := d.client.Name()
	repository := data.Repository.ValueString()

	tflog.Debug(ctx, "Reading GitHub repository webhooks", map[string]interface{}{
		"owner":      owner,
		"repository": repository,
	})

	options := github.ListOptions{
		PerPage: 100,
	}

	var allWebhooks []githubRepositoryWebhookModel
	for {
		hooks, respGH, err := client.Repositories.ListHooks(ctx, owner, repository, &options)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Repository Webhooks",
				fmt.Sprintf("Unable to read webhooks for repository %s/%s: %s", owner, repository, err),
			)
			return
		}

		for _, hook := range hooks {
			webhookModel := githubRepositoryWebhookModel{
				ID:     types.Int64Value(hook.GetID()),
				Type:   types.StringValue(hook.GetType()),
				Name:   types.StringValue(hook.GetName()),
				URL:    types.StringValue(hook.GetURL()),
				Active: types.BoolValue(hook.GetActive()),
			}
			allWebhooks = append(allWebhooks, webhookModel)
		}

		if respGH.NextPage == 0 {
			break
		}
		options.Page = respGH.NextPage
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", owner, repository))
	data.Webhooks = allWebhooks

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
