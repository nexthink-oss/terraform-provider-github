package github

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
)

func TestGithubEnterpriseDataSourceMetadata(t *testing.T) {
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

		if metadataResp.TypeName == "github_enterprise" {
			found = true
			break
		}
	}

	if !found {
		t.Error("github_enterprise data source not found in provider data sources")
	}
}

func TestGithubEnterpriseDataSourceSchema(t *testing.T) {
	// Test the data source schema
	ds := NewGithubEnterpriseDataSource()

	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Errorf("Schema should not have errors: %s", schemaResp.Diagnostics)
	}

	// Check that required attributes are present
	schema := schemaResp.Schema

	// Required attribute
	if _, ok := schema.Attributes["slug"]; !ok {
		t.Error("slug attribute should be present")
	}

	// Computed attributes
	requiredComputedAttrs := []string{"id", "database_id", "name", "description", "created_at", "url"}
	for _, attr := range requiredComputedAttrs {
		if _, ok := schema.Attributes[attr]; !ok {
			t.Errorf("%s attribute should be present", attr)
		}
	}
}
