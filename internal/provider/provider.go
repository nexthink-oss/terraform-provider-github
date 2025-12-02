package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nexthink-oss/terraform-provider-github/v7/internal/provider/repository"
)

// Ensure the implementation satisfies the expected interfaces.
var _ provider.Provider = &githubFrameworkProvider{}

// githubFrameworkProvider is the Framework implementation of the GitHub provider.
// It delegates provider configuration to the SDKv2 provider via muxing.
// Only migrated resources are registered here.
type githubFrameworkProvider struct{}

// NewFrameworkProvider returns a function that creates a new Framework provider instance.
// This is used by the muxer to combine with the SDKv2 provider.
func NewFrameworkProvider() func() provider.Provider {
	return func() provider.Provider {
		return &githubFrameworkProvider{}
	}
}

// Metadata returns the provider type name.
func (p *githubFrameworkProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "github"
}

// Schema returns the provider schema.
// This schema must match the SDKv2 provider schema exactly for muxing to work.
// Provider configuration is still handled by the SDKv2 provider via muxing,
// but both providers must expose identical schemas.
func (p *githubFrameworkProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Optional: true,
				Description: "The OAuth token used to connect to GitHub. Anonymous mode is enabled if both `token` and " +
					"`app_auth` are not set.",
			},
			"owner": schema.StringAttribute{
				Optional: true,
				Description: "The GitHub owner name to manage. " +
					"Use this field instead of `organization` when managing individual accounts.",
			},
			"organization": schema.StringAttribute{
				Optional: true,
				Description: "The GitHub organization name to manage. " +
					"Use this field instead of `owner` when managing organization accounts.",
				DeprecationMessage: "Use owner (or GITHUB_OWNER) instead of organization (or GITHUB_ORGANIZATION)",
			},
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "The GitHub Base API URL",
			},
			"insecure": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable `insecure` mode for testing purposes",
			},
			"write_delay_ms": schema.Int64Attribute{
				Optional: true,
				Description: "Amount of time in milliseconds to sleep in between writes to GitHub API. " +
					"Defaults to 1000ms or 1s if not set.",
			},
			"read_delay_ms": schema.Int64Attribute{
				Optional: true,
				Description: "Amount of time in milliseconds to sleep in between non-write requests to GitHub API. " +
					"Defaults to 0ms if not set.",
			},
			"retry_delay_ms": schema.Int64Attribute{
				Optional: true,
				Description: "Amount of time in milliseconds to sleep in between requests to GitHub API after an error response. " +
					"Defaults to 1000ms or 1s if not set, the max_retries must be set to greater than zero.",
			},
			"parallel_requests": schema.BoolAttribute{
				Optional: true,
				Description: "Allow the provider to make parallel API calls to GitHub. " +
					"You may want to set it to true when you have a private Github Enterprise without strict rate limits. " +
					"Although, it is not possible to enable this setting on github.com " +
					"because we enforce the respect of github.com's best practices to avoid hitting abuse rate limits" +
					"Defaults to false if not set",
			},
			"rate_limiter": schema.StringAttribute{
				Optional: true,
				Description: "The rate limiting strategy to use. 'modern' uses go-github-ratelimit for automatic GitHub API rate limit handling. " +
					"'legacy' uses the provider's built-in rate limiting with configurable delays. " +
					"When using 'modern', the read_delay_ms, write_delay_ms, and parallel_requests settings are ignored. " +
					"Defaults to 'modern'.",
				Validators: []validator.String{
					stringvalidator.OneOf("legacy", "modern"),
				},
			},
			"max_retries": schema.Int64Attribute{
				Optional: true,
				Description: "Number of times to retry a request after receiving an error status code" +
					"Defaults to 3",
			},
			"retryable_errors": schema.ListAttribute{
				Optional:    true,
				ElementType: types.Int64Type,
				Description: "Allow the provider to retry after receiving an error status code, the max_retries should be set for this to work" +
					"Defaults to [500, 502, 503, 504]",
			},
		},
		Blocks: map[string]schema.Block{
			"app_auth": schema.ListNestedBlock{
				Description: "The GitHub App credentials used to connect to GitHub. Conflicts with " +
					"`token`. Anonymous mode is enabled if both `token` and `app_auth` are not set.",
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:    true,
							Description: "The GitHub App ID.",
						},
						"installation_id": schema.StringAttribute{
							Required:    true,
							Description: "The GitHub App installation instance ID.",
						},
						"pem_file": schema.StringAttribute{
							Required:    true,
							Sensitive:   true,
							Description: "The GitHub App PEM file contents.",
						},
					},
				},
			},
		},
	}
}

// Configure is a no-op for the Framework provider.
// Provider configuration (token, organization, base_url, etc.) is handled
// entirely by the SDKv2 provider. Framework resources will access the
// configured GitHub client through a shared mechanism.
func (p *githubFrameworkProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Configuration is handled by the SDKv2 provider via muxing.
	// Framework resources need to access the GitHub client configured by SDKv2.
	// This is typically done by storing the client in a shared location or
	// passing it through the provider data.
}

// Resources returns the list of Framework resources.
// Only resources that have been migrated from SDKv2 to Framework are listed here.
func (p *githubFrameworkProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		repository.NewResource,
	}
}

// DataSources returns the list of Framework data sources.
// Only data sources that have been migrated from SDKv2 to Framework are listed here.
func (p *githubFrameworkProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// No data sources migrated yet
	}
}
