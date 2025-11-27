package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

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

// Schema returns an empty schema.
// Provider configuration is handled entirely by the SDKv2 provider via muxing.
// The mux server ensures that provider configuration from the SDKv2 provider
// is available to Framework resources through the shared GitHub client.
func (p *githubFrameworkProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "GitHub provider (Framework resources). Configuration is handled by the SDKv2 provider.",
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
