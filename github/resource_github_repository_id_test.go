package github

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestGithubRepositoryResourceIDAttribute tests that the ID attribute is properly defined
func TestGithubRepositoryResourceIDAttribute(t *testing.T) {
	res := NewGithubRepositoryResource()

	// Test schema contains ID attribute
	var schemaResp resource.SchemaResponse
	res.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema has errors: %v", schemaResp.Diagnostics.Errors())
	}

	// Check that ID attribute exists and is properly configured
	idAttr, exists := schemaResp.Schema.Attributes["id"]
	if !exists {
		t.Fatal("ID attribute is missing from schema")
	}

	stringAttr, ok := idAttr.(schema.StringAttribute)
	if !ok {
		t.Fatal("ID attribute is not a StringAttribute")
	}

	if !stringAttr.Computed {
		t.Error("ID attribute should be Computed")
	}

	if stringAttr.Required {
		t.Error("ID attribute should not be Required")
	}

	if stringAttr.Optional {
		t.Error("ID attribute should not be Optional")
	}

	if stringAttr.Description != "The repository name." {
		t.Errorf("Expected description 'The repository name.', got: %s", stringAttr.Description)
	}
}

// TestGithubRepositoryResourceModelIDField tests that the model has ID field
func TestGithubRepositoryResourceModelIDField(t *testing.T) {
	model := githubRepositoryResourceModel{}

	// Verify ID field exists and can be set
	model.ID = types.StringValue("test-repo")

	if model.ID.IsNull() {
		t.Error("ID field should not be null after setting value")
	}

	if model.ID.ValueString() != "test-repo" {
		t.Errorf("Expected ID value 'test-repo', got: %s", model.ID.ValueString())
	}
}
