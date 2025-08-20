package framework

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestGithubRepositoryEnvironmentsDataSource_Metadata(t *testing.T) {
	d := NewGithubRepositoryEnvironmentsDataSource()

	req := datasource.MetadataRequest{
		ProviderTypeName: "github",
	}
	resp := &datasource.MetadataResponse{}

	d.Metadata(context.Background(), req, resp)

	if resp.TypeName != "github_repository_environments" {
		t.Errorf("Expected TypeName to be 'github_repository_environments', got %s", resp.TypeName)
	}
}

func TestGithubRepositoryEnvironmentsDataSource_Schema(t *testing.T) {
	d := NewGithubRepositoryEnvironmentsDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	d.Schema(context.Background(), req, resp)

	// Verify basic schema structure
	attrs := resp.Schema.Attributes

	// Check that required attributes exist
	if _, ok := attrs["id"]; !ok {
		t.Error("Expected 'id' attribute to exist")
	}

	if _, ok := attrs["repository"]; !ok {
		t.Error("Expected 'repository' attribute to exist")
	}

	if _, ok := attrs["environments"]; !ok {
		t.Error("Expected 'environments' attribute to exist")
	}

	// Verify repository is required
	repoAttr := attrs["repository"].(schema.StringAttribute)
	if !repoAttr.Required {
		t.Error("Expected 'repository' attribute to be required")
	}

	// Verify environments is computed
	envAttr := attrs["environments"].(schema.ListNestedAttribute)
	if !envAttr.Computed {
		t.Error("Expected 'environments' attribute to be computed")
	}

	// Verify nested environment attributes
	nestedAttrs := envAttr.NestedObject.Attributes
	if _, ok := nestedAttrs["name"]; !ok {
		t.Error("Expected 'name' attribute in environments nested object")
	}

	if _, ok := nestedAttrs["node_id"]; !ok {
		t.Error("Expected 'node_id' attribute in environments nested object")
	}
}

func TestGithubRepositoryEnvironmentModel_Basic(t *testing.T) {
	model := githubRepositoryEnvironmentModel{
		Name:   types.StringValue("test-env"),
		NodeID: types.StringValue("node123"),
	}

	if model.Name.ValueString() != "test-env" {
		t.Errorf("Expected Name to be 'test-env', got %s", model.Name.ValueString())
	}

	if model.NodeID.ValueString() != "node123" {
		t.Errorf("Expected NodeID to be 'node123', got %s", model.NodeID.ValueString())
	}
}
