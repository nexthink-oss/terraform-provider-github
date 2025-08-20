package framework

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TestGithubRepositoryEnvironmentResourceSchema validates that the schema is correctly defined
func TestGithubRepositoryEnvironmentResourceSchema(t *testing.T) {
	ctx := context.Background()

	// Create resource instance
	r := NewGithubRepositoryEnvironmentResource()

	// Get schema
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema has errors: %v", schemaResp.Diagnostics.Errors())
	}

	schema := schemaResp.Schema

	// Validate required attributes exist
	requiredAttrs := []string{"repository", "environment"}
	for _, attr := range requiredAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Required attribute '%s' not found in schema", attr)
		}
	}

	// Validate optional attributes exist
	optionalAttrs := []string{"can_admins_bypass", "prevent_self_review", "wait_timer", "reviewers", "deployment_branch_policy"}
	for _, attr := range optionalAttrs {
		if _, exists := schema.Attributes[attr]; !exists {
			t.Errorf("Optional attribute '%s' not found in schema", attr)
		}
	}

	// Validate computed attributes exist
	if _, exists := schema.Attributes["id"]; !exists {
		t.Error("Computed attribute 'id' not found in schema")
	}

	// Validate attribute types
	if schema.Attributes["repository"].GetType() != types.StringType {
		t.Error("repository attribute should be StringType")
	}

	if schema.Attributes["environment"].GetType() != types.StringType {
		t.Error("environment attribute should be StringType")
	}

	if schema.Attributes["can_admins_bypass"].GetType() != types.BoolType {
		t.Error("can_admins_bypass attribute should be BoolType")
	}

	if schema.Attributes["prevent_self_review"].GetType() != types.BoolType {
		t.Error("prevent_self_review attribute should be BoolType")
	}

	if schema.Attributes["wait_timer"].GetType() != types.Int64Type {
		t.Error("wait_timer attribute should be Int64Type")
	}
}

// TestGithubRepositoryEnvironmentResourceModel validates the model structure
func TestGithubRepositoryEnvironmentResourceModel(t *testing.T) {
	// Test that we can create and populate the model
	model := githubRepositoryEnvironmentResourceModel{
		Repository:             types.StringValue("test-repo"),
		Environment:            types.StringValue("test-env"),
		CanAdminsBypass:        types.BoolValue(true),
		PreventSelfReview:      types.BoolValue(false),
		WaitTimer:              types.Int64Value(10000),
		Reviewers:              types.ListNull(types.ObjectType{AttrTypes: reviewersAttrTypes()}),
		DeploymentBranchPolicy: types.ListNull(types.ObjectType{AttrTypes: deploymentBranchPolicyAttrTypes()}),
		ID:                     types.StringValue("test-repo:test-env"),
	}

	// Validate that the model fields are populated correctly
	if model.Repository.ValueString() != "test-repo" {
		t.Error("Repository field not set correctly")
	}

	if model.Environment.ValueString() != "test-env" {
		t.Error("Environment field not set correctly")
	}

	if !model.CanAdminsBypass.ValueBool() {
		t.Error("CanAdminsBypass field not set correctly")
	}

	if model.PreventSelfReview.ValueBool() {
		t.Error("PreventSelfReview field not set correctly")
	}

	if model.WaitTimer.ValueInt64() != 10000 {
		t.Error("WaitTimer field not set correctly")
	}

	if model.ID.ValueString() != "test-repo:test-env" {
		t.Error("ID field not set correctly")
	}
}
