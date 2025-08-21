package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubIpRangesDataSource{}
	_ datasource.DataSourceWithConfigure = &githubIpRangesDataSource{}
)

type githubIpRangesDataSource struct {
	client *Owner
}

type githubIpRangesDataSourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Hooks                    types.List   `tfsdk:"hooks"`
	Web                      types.List   `tfsdk:"web"`
	API                      types.List   `tfsdk:"api"`
	Git                      types.List   `tfsdk:"git"`
	GithubEnterpriseImporter types.List   `tfsdk:"github_enterprise_importer"`
	Packages                 types.List   `tfsdk:"packages"`
	Pages                    types.List   `tfsdk:"pages"`
	Importer                 types.List   `tfsdk:"importer"`
	Actions                  types.List   `tfsdk:"actions"`
	Dependabot               types.List   `tfsdk:"dependabot"`
}

type githubMetaResponse struct {
	Hooks                    []string `json:"hooks"`
	Web                      []string `json:"web"`
	API                      []string `json:"api"`
	Git                      []string `json:"git"`
	GithubEnterpriseImporter []string `json:"github_enterprise_importer"`
	Packages                 []string `json:"packages"`
	Pages                    []string `json:"pages"`
	Importer                 []string `json:"importer"`
	Actions                  []string `json:"actions"`
	Dependabot               []string `json:"dependabot"`
}

func NewGithubIpRangesDataSource() datasource.DataSource {
	return &githubIpRangesDataSource{}
}

func (d *githubIpRangesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ip_ranges"
}

func (d *githubIpRangesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get GitHub's IP address ranges for various services.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this data source.",
				Computed:    true,
			},
			"hooks": schema.ListAttribute{
				Description: "An array of IP addresses in CIDR format specifying the addresses that incoming service hooks will originate from.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"web": schema.ListAttribute{
				Description: "An array of IP addresses in CIDR format for GitHub's web servers.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"api": schema.ListAttribute{
				Description: "An array of IP addresses in CIDR format for GitHub's API servers.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"git": schema.ListAttribute{
				Description: "An array of IP addresses in CIDR format specifying the Git servers.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"github_enterprise_importer": schema.ListAttribute{
				Description: "An array of IP addresses in CIDR format for the GitHub Enterprise Importer.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"packages": schema.ListAttribute{
				Description: "An array of IP addresses in CIDR format for GitHub Packages.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"pages": schema.ListAttribute{
				Description: "An array of IP addresses in CIDR format specifying the A records for GitHub Pages.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"importer": schema.ListAttribute{
				Description: "An array of IP addresses in CIDR format for the GitHub Importer.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"actions": schema.ListAttribute{
				Description: "An array of IP addresses in CIDR format for GitHub Actions.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"dependabot": schema.ListAttribute{
				Description: "An array of IP addresses in CIDR format for GitHub Dependabot.",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (d *githubIpRangesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubIpRangesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubIpRangesDataSourceModel

	tflog.Debug(ctx, "Reading GitHub IP ranges from Meta API")

	// Make HTTP request to GitHub Meta API
	metaURL := "https://api.github.com/meta"

	// Use the base URL from the client if it's configured for GitHub Enterprise
	if d.client != nil && d.client.V3Client() != nil && d.client.V3Client().BaseURL != nil {
		baseURL := d.client.V3Client().BaseURL.String()
		if baseURL != "https://api.github.com/" && baseURL != "" {
			// Remove trailing slash if present
			if baseURL[len(baseURL)-1] == '/' {
				baseURL = baseURL[:len(baseURL)-1]
			}
			metaURL = baseURL + "/meta"
		}
	}

	tflog.Debug(ctx, "Fetching IP ranges from URL", map[string]any{
		"url": metaURL,
	})

	httpResp, err := http.Get(metaURL)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub IP Ranges",
			fmt.Sprintf("An error occurred while fetching GitHub IP ranges from %s: %s", metaURL, err.Error()),
		)
		return
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		resp.Diagnostics.AddError(
			"GitHub API Request Failed",
			fmt.Sprintf("GitHub API returned status %d when fetching IP ranges from %s", httpResp.StatusCode, metaURL),
		)
		return
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub API Response",
			fmt.Sprintf("An error occurred while reading the GitHub API response: %s", err.Error()),
		)
		return
	}

	var metaResponse githubMetaResponse
	if err := json.Unmarshal(body, &metaResponse); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse GitHub API Response",
			fmt.Sprintf("An error occurred while parsing the GitHub API response: %s", err.Error()),
		)
		return
	}

	// Convert string slices to List types
	data.ID = types.StringValue("github-ip-ranges")

	var diags diag.Diagnostics

	data.Hooks, diags = types.ListValueFrom(ctx, types.StringType, metaResponse.Hooks)
	resp.Diagnostics.Append(diags...)

	data.Web, diags = types.ListValueFrom(ctx, types.StringType, metaResponse.Web)
	resp.Diagnostics.Append(diags...)

	data.API, diags = types.ListValueFrom(ctx, types.StringType, metaResponse.API)
	resp.Diagnostics.Append(diags...)

	data.Git, diags = types.ListValueFrom(ctx, types.StringType, metaResponse.Git)
	resp.Diagnostics.Append(diags...)

	data.GithubEnterpriseImporter, diags = types.ListValueFrom(ctx, types.StringType, metaResponse.GithubEnterpriseImporter)
	resp.Diagnostics.Append(diags...)

	data.Packages, diags = types.ListValueFrom(ctx, types.StringType, metaResponse.Packages)
	resp.Diagnostics.Append(diags...)

	data.Pages, diags = types.ListValueFrom(ctx, types.StringType, metaResponse.Pages)
	resp.Diagnostics.Append(diags...)

	data.Importer, diags = types.ListValueFrom(ctx, types.StringType, metaResponse.Importer)
	resp.Diagnostics.Append(diags...)

	data.Actions, diags = types.ListValueFrom(ctx, types.StringType, metaResponse.Actions)
	resp.Diagnostics.Append(diags...)

	data.Dependabot, diags = types.ListValueFrom(ctx, types.StringType, metaResponse.Dependabot)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Successfully read GitHub IP ranges", map[string]any{
		"hooks_count":                      len(metaResponse.Hooks),
		"web_count":                        len(metaResponse.Web),
		"api_count":                        len(metaResponse.API),
		"git_count":                        len(metaResponse.Git),
		"github_enterprise_importer_count": len(metaResponse.GithubEnterpriseImporter),
		"packages_count":                   len(metaResponse.Packages),
		"pages_count":                      len(metaResponse.Pages),
		"importer_count":                   len(metaResponse.Importer),
		"actions_count":                    len(metaResponse.Actions),
		"dependabot_count":                 len(metaResponse.Dependabot),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
