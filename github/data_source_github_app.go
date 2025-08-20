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
	_ datasource.DataSource              = &githubAppDataSource{}
	_ datasource.DataSourceWithConfigure = &githubAppDataSource{}
)

type githubAppDataSource struct {
	client *Owner
}

type githubAppDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Slug        types.String `tfsdk:"slug"`
	Description types.String `tfsdk:"description"`
	Name        types.String `tfsdk:"name"`
	NodeID      types.String `tfsdk:"node_id"`
}

func NewGithubAppDataSource() datasource.DataSource {
	return &githubAppDataSource{}
}

func (d *githubAppDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app"
}

func (d *githubAppDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information about an app.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the app.",
				Computed:    true,
			},
			"slug": schema.StringAttribute{
				Description: "The slug of the app.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the app.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the app.",
				Computed:    true,
			},
			"node_id": schema.StringAttribute{
				Description: "The node ID of the app.",
				Computed:    true,
			},
		},
	}
}

func (d *githubAppDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubAppDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubAppDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	slug := data.Slug.ValueString()

	tflog.Debug(ctx, "Reading GitHub app", map[string]interface{}{
		"slug": slug,
	})

	app, _, err := d.client.V3Client().Apps.Get(ctx, slug)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub App",
			fmt.Sprintf("An unexpected error occurred while reading the GitHub app %s: %s", slug, err.Error()),
		)
		return
	}

	// Set values
	data.ID = types.StringValue(strconv.FormatInt(app.GetID(), 10))
	data.Description = types.StringValue(app.GetDescription())
	data.Name = types.StringValue(app.GetName())
	data.NodeID = types.StringValue(app.GetNodeID())

	tflog.Debug(ctx, "Successfully read GitHub app", map[string]interface{}{
		"slug":        slug,
		"id":          data.ID.ValueString(),
		"name":        data.Name.ValueString(),
		"description": data.Description.ValueString(),
		"node_id":     data.NodeID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
