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
	_ datasource.DataSource              = &githubOrganizationWebhooksDataSource{}
	_ datasource.DataSourceWithConfigure = &githubOrganizationWebhooksDataSource{}
)

type githubOrganizationWebhooksDataSource struct {
	client *Owner
}

type githubOrganizationWebhookModel struct {
	ID     types.Int64  `tfsdk:"id"`
	Type   types.String `tfsdk:"type"`
	Name   types.String `tfsdk:"name"`
	URL    types.String `tfsdk:"url"`
	Active types.Bool   `tfsdk:"active"`
}

type githubOrganizationWebhooksDataSourceModel struct {
	ID       types.String                     `tfsdk:"id"`
	Webhooks []githubOrganizationWebhookModel `tfsdk:"webhooks"`
}

func NewGithubOrganizationWebhooksDataSource() datasource.DataSource {
	return &githubOrganizationWebhooksDataSource{}
}

func (d *githubOrganizationWebhooksDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_webhooks"
}

func (d *githubOrganizationWebhooksDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on all GitHub webhooks of the organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the organization.",
				Computed:    true,
			},
			"webhooks": schema.ListNestedAttribute{
				Description: "List of webhooks in the organization.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The webhook's ID.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The webhook's type.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The webhook's name.",
							Computed:    true,
						},
						"url": schema.StringAttribute{
							Description: "The webhook's URL.",
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

func (d *githubOrganizationWebhooksDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubOrganizationWebhooksDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubOrganizationWebhooksDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	owner := d.client.Name()
	client := d.client.V3Client()

	tflog.Debug(ctx, "Reading GitHub organization webhooks", map[string]interface{}{
		"organization": owner,
	})

	options := &github.ListOptions{
		PerPage: maxPerPage,
	}

	var allWebhooks []githubOrganizationWebhookModel
	for {
		hooks, githubResp, err := client.Organizations.ListHooks(ctx, owner, options)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to read GitHub organization webhooks",
				fmt.Sprintf("Error reading organization webhooks: %s", err.Error()),
			)
			return
		}

		webhooks := d.convertWebhooksToModels(hooks)
		allWebhooks = append(allWebhooks, webhooks...)

		if githubResp.NextPage == 0 {
			break
		}

		options.Page = githubResp.NextPage
	}

	// Set the ID to the organization name
	data.ID = types.StringValue(owner)
	data.Webhooks = allWebhooks

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d *githubOrganizationWebhooksDataSource) convertWebhooksToModels(hooks []*github.Hook) []githubOrganizationWebhookModel {
	if hooks == nil {
		return []githubOrganizationWebhookModel{}
	}

	models := make([]githubOrganizationWebhookModel, len(hooks))

	for i, hook := range hooks {
		model := githubOrganizationWebhookModel{}

		if hook.ID != nil {
			model.ID = types.Int64Value(*hook.ID)
		} else {
			model.ID = types.Int64Null()
		}

		if hook.Type != nil {
			model.Type = types.StringValue(*hook.Type)
		} else {
			model.Type = types.StringNull()
		}

		if hook.Name != nil {
			model.Name = types.StringValue(*hook.Name)
		} else {
			model.Name = types.StringNull()
		}

		if hook.URL != nil {
			model.URL = types.StringValue(*hook.URL)
		} else {
			model.URL = types.StringNull()
		}

		if hook.Active != nil {
			model.Active = types.BoolValue(*hook.Active)
		} else {
			model.Active = types.BoolNull()
		}

		models[i] = model
	}

	return models
}
