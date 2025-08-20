package github

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestGithubReleaseDataSourceMetadata(t *testing.T) {
	// Test that the data source is properly registered and has correct metadata
	p := New()

	metadataReq := provider.MetadataRequest{}
	metadataResp := &provider.MetadataResponse{}
	p.Metadata(context.Background(), metadataReq, metadataResp)

	if metadataResp.TypeName != "github" {
		t.Errorf("Expected provider type name 'github', got %s", metadataResp.TypeName)
	}

	// Get data sources
	dataSources := p.DataSources(context.Background())

	// Check that our data source is included
	var found bool
	for _, dsFunc := range dataSources {
		ds := dsFunc()
		metadataReq := datasource.MetadataRequest{
			ProviderTypeName: "github",
		}
		metadataResp := &datasource.MetadataResponse{}
		ds.Metadata(context.Background(), metadataReq, metadataResp)

		if metadataResp.TypeName == "github_release" {
			found = true
			break
		}
	}

	if !found {
		t.Error("github_release data source not found in provider data sources")
	}
}

func TestGithubReleaseDataSourceSchema(t *testing.T) {
	// Test the data source schema
	ds := NewGithubReleaseDataSource()

	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Errorf("Schema should not have errors: %s", schemaResp.Diagnostics)
	}

	// Check that required attributes are present
	schema := schemaResp.Schema

	// Required attributes
	requiredAttrs := []string{"repository", "owner", "retrieve_by"}
	for _, attr := range requiredAttrs {
		if _, ok := schema.Attributes[attr]; !ok {
			t.Errorf("%s attribute should be present", attr)
		}
	}

	// Optional attributes
	optionalAttrs := []string{"release_tag", "release_id"}
	for _, attr := range optionalAttrs {
		if _, ok := schema.Attributes[attr]; !ok {
			t.Errorf("%s attribute should be present", attr)
		}
	}

	// Computed attributes
	computedAttrs := []string{"id", "target_commitish", "name", "body", "draft", "prerelease", "created_at", "published_at", "url", "html_url", "assets_url", "asserts_url", "upload_url", "zipball_url", "tarball_url"}
	for _, attr := range computedAttrs {
		if _, ok := schema.Attributes[attr]; !ok {
			t.Errorf("%s attribute should be present", attr)
		}
	}

	// Check that assets block is present
	if _, ok := schema.Blocks["assets"]; !ok {
		t.Error("assets block should be present")
	}
}
