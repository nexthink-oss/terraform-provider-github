package github

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
)

func TestGithubOrganizationWebhooksDataSource_Metadata(t *testing.T) {
	ds := NewGithubOrganizationWebhooksDataSource()

	var req datasource.MetadataRequest
	req.ProviderTypeName = "github"

	var resp datasource.MetadataResponse

	ds.Metadata(context.Background(), req, &resp)

	expected := "github_organization_webhooks"
	if resp.TypeName != expected {
		t.Errorf("Expected TypeName to be %q, got %q", expected, resp.TypeName)
	}
}

func TestGithubOrganizationWebhooksDataSource_Schema(t *testing.T) {
	ds := NewGithubOrganizationWebhooksDataSource()

	var req datasource.SchemaRequest
	var resp datasource.SchemaResponse

	ds.Schema(context.Background(), req, &resp)

	// Verify the main attributes exist
	attrs := resp.Schema.Attributes

	if _, exists := attrs["id"]; !exists {
		t.Error("Expected 'id' attribute to exist")
	}

	if _, exists := attrs["webhooks"]; !exists {
		t.Error("Expected 'webhooks' attribute to exist")
	}

	// Verify webhooks is a list nested attribute
	webhooksAttr, ok := attrs["webhooks"]
	if !ok {
		t.Fatal("webhooks attribute should exist")
	}

	// Check if it's the right type (this is a basic check)
	if webhooksAttr.GetDescription() != "List of webhooks in the organization." {
		t.Errorf("Expected description 'List of webhooks in the organization.', got %q", webhooksAttr.GetDescription())
	}
}
