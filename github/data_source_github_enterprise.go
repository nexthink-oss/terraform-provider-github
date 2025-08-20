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
	_ datasource.DataSource              = &githubEnterpriseDataSource{}
	_ datasource.DataSourceWithConfigure = &githubEnterpriseDataSource{}
)

type githubEnterpriseDataSource struct {
	client *Owner
}

type githubEnterpriseDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	DatabaseID  types.Int64  `tfsdk:"database_id"`
	Slug        types.String `tfsdk:"slug"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CreatedAt   types.String `tfsdk:"created_at"`
	URL         types.String `tfsdk:"url"`
}

func NewGithubEnterpriseDataSource() datasource.DataSource {
	return &githubEnterpriseDataSource{}
}

func (d *githubEnterpriseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_enterprise"
}

func (d *githubEnterpriseDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get basic information about a GitHub enterprise.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the enterprise.",
				Computed:    true,
			},
			"database_id": schema.Int64Attribute{
				Description: "The database ID of the enterprise.",
				Computed:    true,
			},
			"slug": schema.StringAttribute{
				Description: "The slug of the enterprise.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the enterprise.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the enterprise.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The timestamp of when the enterprise was created.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "The URL of the enterprise.",
				Computed:    true,
			},
		},
	}
}

func (d *githubEnterpriseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubEnterpriseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubEnterpriseDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	slug := data.Slug.ValueString()

	tflog.Debug(ctx, "Reading GitHub enterprise", map[string]interface{}{
		"slug": slug,
	})

	var query struct {
		Enterprise struct {
			ID          githubv4.String
			DatabaseId  githubv4.Int
			Name        githubv4.String
			Description githubv4.String
			CreatedAt   githubv4.String
			Url         githubv4.String
		} `graphql:"enterprise(slug: $slug)"`
	}

	variables := map[string]any{
		"slug": githubv4.String(slug),
	}

	err := d.client.V4Client().Query(ctx, &query, variables)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Enterprise",
			fmt.Sprintf("An unexpected error occurred while reading the GitHub enterprise %s: %s", slug, err.Error()),
		)
		return
	}

	if query.Enterprise.ID == "" {
		resp.Diagnostics.AddError(
			"GitHub Enterprise Not Found",
			fmt.Sprintf("Could not find enterprise %s", slug),
		)
		return
	}

	// Set values
	data.ID = types.StringValue(string(query.Enterprise.ID))
	data.DatabaseID = types.Int64Value(int64(query.Enterprise.DatabaseId))
	data.Name = types.StringValue(string(query.Enterprise.Name))
	data.Description = types.StringValue(string(query.Enterprise.Description))
	data.CreatedAt = types.StringValue(string(query.Enterprise.CreatedAt))
	data.URL = types.StringValue(string(query.Enterprise.Url))

	tflog.Debug(ctx, "Successfully read GitHub enterprise", map[string]interface{}{
		"slug":        slug,
		"id":          data.ID.ValueString(),
		"database_id": data.DatabaseID.ValueInt64(),
		"name":        data.Name.ValueString(),
		"description": data.Description.ValueString(),
		"created_at":  data.CreatedAt.ValueString(),
		"url":         data.URL.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
