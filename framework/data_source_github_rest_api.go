package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ datasource.DataSource              = &githubRestApiDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRestApiDataSource{}
)

type githubRestApiDataSource struct {
	client *githubpkg.Owner
}

type githubRestApiDataSourceModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Code     types.Int64  `tfsdk:"code"`
	Status   types.String `tfsdk:"status"`
	Headers  types.String `tfsdk:"headers"`
	Body     types.String `tfsdk:"body"`
	ID       types.String `tfsdk:"id"`
}

func NewGithubRestApiDataSource() datasource.DataSource {
	return &githubRestApiDataSource{}
}

func (d *githubRestApiDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_rest_api"
}

func (d *githubRestApiDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub resource with a custom GET request to GitHub REST API.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Description: "The GitHub REST API endpoint to query.",
				Required:    true,
			},
			"code": schema.Int64Attribute{
				Description: "The HTTP response status code.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "The HTTP response status message.",
				Computed:    true,
			},
			"headers": schema.StringAttribute{
				Description: "The HTTP response headers as a JSON string.",
				Computed:    true,
			},
			"body": schema.StringAttribute{
				Description: "The HTTP response body as a string.",
				Computed:    true,
			},
			"id": schema.StringAttribute{
				Description: "The ID of this data source.",
				Computed:    true,
			},
		},
	}
}

func (d *githubRestApiDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*githubpkg.Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *githubRestApiDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRestApiDataSourceModel

	// Read configuration from state
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := data.Endpoint.ValueString()

	tflog.Debug(ctx, "Making GitHub REST API request", map[string]interface{}{
		"endpoint": endpoint,
	})

	client := d.client.V3Client()

	// Create new request using the GitHub client
	apiReq, err := client.NewRequest("GET", endpoint, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create GitHub API Request",
			fmt.Sprintf("An error occurred while creating the GitHub API request: %s", err.Error()),
		)
		return
	}

	// Make the request - allow 404 errors like the SDKv2 version
	apiResp, err := client.Do(ctx, apiReq, nil)
	if err != nil && (apiResp == nil || apiResp.StatusCode != 404) {
		resp.Diagnostics.AddError(
			"GitHub API Request Failed",
			fmt.Sprintf("An error occurred while making the GitHub API request: %s", err.Error()),
		)
		return
	}

	// Marshal response headers to JSON
	headersJSON, err := json.Marshal(apiResp.Header)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Marshal Response Headers",
			fmt.Sprintf("An error occurred while marshaling response headers: %s", err.Error()),
		)
		return
	}

	// Read response body
	body, err := io.ReadAll(apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Response Body",
			fmt.Sprintf("An error occurred while reading the response body: %s", err.Error()),
		)
		return
	}

	// Set the data model values
	data.Code = types.Int64Value(int64(apiResp.StatusCode))
	data.Status = types.StringValue(apiResp.Status)
	data.Headers = types.StringValue(string(headersJSON))
	data.Body = types.StringValue(string(body))
	data.ID = types.StringValue(apiResp.Header.Get("x-github-request-id"))

	tflog.Debug(ctx, "Successfully read GitHub REST API response", map[string]interface{}{
		"endpoint":    endpoint,
		"status_code": apiResp.StatusCode,
		"request_id":  apiResp.Header.Get("x-github-request-id"),
	})

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
