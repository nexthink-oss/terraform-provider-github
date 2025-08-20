package github

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const anonymous = "anonymous"
const individual = "individual"
const organization = "organization"
const enterprise = "enterprise"

// testAccProtoV6ProviderFactories returns a pure Framework provider server for testing
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"github": func() (tfprotov6.ProviderServer, error) {
		return providerserver.NewProtocol6(New())(), nil
	},
}

// testAccMuxedProtoV6ProviderFactories is now an alias for testAccProtoV6ProviderFactories
// since we're Framework-only now (kept for backward compatibility with existing tests)
var testAccMuxedProtoV6ProviderFactories = testAccProtoV6ProviderFactories

func testAccPreCheck(t *testing.T, mode string) {
	switch mode {
	case individual:
		testAccPreCheckIndividual(t)
	case organization:
		testAccPreCheckOrganization(t)
	case anonymous:
		testAccPreCheckAnonymous(t)
	default:
		t.Fatalf("Unknown test mode: %s", mode)
	}
}

func skipUnlessMode(t *testing.T, providerMode string) {
	switch providerMode {
	case anonymous:
		if os.Getenv("GITHUB_BASE_URL") != "" &&
			os.Getenv("GITHUB_BASE_URL") != "https://api.github.com/" {
			t.Log("anonymous mode not supported for GHES deployments")
			break
		}

		if os.Getenv("GITHUB_TOKEN") == "" {
			return
		}

		t.Skip("Skipping because GITHUB_TOKEN is present")
	case individual:
		testAccPreCheckIndividual(t)
	case organization:
		testAccPreCheckOrganization(t)
	}
}

func testAccPreCheckIndividual(t *testing.T) {
	if err := os.Getenv("GITHUB_TOKEN"); err == "" {
		t.Skip("GITHUB_TOKEN must be set for acceptance tests")
	}
	if err := os.Getenv("GITHUB_OWNER"); err == "" {
		t.Skip("GITHUB_OWNER must be set for acceptance tests")
	}
}

func testAccPreCheckOrganization(t *testing.T) {
	testAccPreCheckIndividual(t)
	if err := os.Getenv("GITHUB_ORGANIZATION"); err == "" {
		t.Skip("GITHUB_ORGANIZATION must be set for acceptance tests")
	}
}

func testAccPreCheckAnonymous(t *testing.T) {
	if err := os.Getenv("GITHUB_BASE_URL"); err == "" {
		t.Skip("GITHUB_BASE_URL must be set for acceptance tests")
	}
}

func testAccPreCheckEnterprise(t *testing.T) {
	if err := os.Getenv("GITHUB_TOKEN"); err == "" {
		t.Skip("GITHUB_TOKEN must be set for acceptance tests")
	}
	if err := os.Getenv("ENTERPRISE_SLUG"); err == "" {
		t.Skip("ENTERPRISE_SLUG must be set for enterprise acceptance tests")
	}
	if err := os.Getenv("ENTERPRISE_ACCOUNT"); err != "true" {
		t.Skip("ENTERPRISE_ACCOUNT must be set to 'true' for enterprise acceptance tests")
	}
}

func skipUnlessEnterpriseMode(t *testing.T, mode string) {
	switch mode {
	case enterprise:
		testAccPreCheckEnterprise(t)
	default:
		t.Fatalf("Unknown test mode: %s", mode)
	}
}

// TestProvider tests the framework provider creation
func TestProvider(t *testing.T) {
	// Just verify that the provider can be created without panicking
	provider := New()
	if provider == nil {
		t.Error("Provider should not be nil")
	}
}
