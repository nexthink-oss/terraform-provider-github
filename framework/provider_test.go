package framework

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/tf5to6server"
	"github.com/hashicorp/terraform-plugin-mux/tf6muxserver"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/isometry/terraform-provider-github/v7/github"
)

const (
	individual   = "individual"
	organization = "organization"
	anonymous    = "anonymous"
)

// testAccProtoV6ProviderFactories returns a pure Framework provider server for testing
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"github": func() (tfprotov6.ProviderServer, error) {
		return providerserver.NewProtocol6(New())(), nil
	},
}

// testAccMuxedProtoV6ProviderFactories returns a muxed provider server that combines
// the SDKv2 provider (upgraded to v6 protocol) with the new Framework provider
func testAccMuxedProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"github": func() (tfprotov6.ProviderServer, error) {
			ctx := context.Background()

			// Upgrade existing SDKv2 provider to protocol v6
			upgradedSdkProvider, err := tf5to6server.UpgradeServer(
				ctx,
				github.Provider().GRPCProvider,
			)
			if err != nil {
				return nil, err
			}

			providers := []func() tfprotov6.ProviderServer{
				// Existing SDKv2 provider (upgraded to v6 protocol)
				func() tfprotov6.ProviderServer {
					return upgradedSdkProvider
				},
				// New Plugin Framework provider
				func() tfprotov6.ProviderServer {
					return providerserver.NewProtocol6(New())()
				},
			}

			// Mux the providers together
			muxServer, err := tf6muxserver.NewMuxServer(ctx, providers...)
			if err != nil {
				return nil, err
			}

			return muxServer.ProviderServer(), nil
		},
	}
}

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

// TestProvider tests the framework provider creation
func TestProvider(t *testing.T) {
	// Just verify that the provider can be created without panicking
	provider := New()
	if provider == nil {
		t.Error("Provider should not be nil")
	}
}

func TestMain(m *testing.M) {
	resource.TestMain(m)
}

func init() {
	resource.AddTestSweepers("github_user_ssh_key", &resource.Sweeper{
		Name: "github_user_ssh_key",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_user_ssh_key sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_user_gpg_key", &resource.Sweeper{
		Name: "github_user_gpg_key",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_user_gpg_key sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_branch", &resource.Sweeper{
		Name: "github_branch",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_branch sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_branch_default", &resource.Sweeper{
		Name: "github_branch_default",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_branch_default sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_issue", &resource.Sweeper{
		Name: "github_issue",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_issue sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_issue_labels", &resource.Sweeper{
		Name: "github_issue_labels",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_issue_labels sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_repository_file", &resource.Sweeper{
		Name: "github_repository_file",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_repository_file sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_actions_secret", &resource.Sweeper{
		Name: "github_actions_secret",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_actions_secret sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_actions_organization_secret", &resource.Sweeper{
		Name: "github_actions_organization_secret",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_actions_organization_secret sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_actions_variable", &resource.Sweeper{
		Name: "github_actions_variable",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_actions_variable sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_repository_deploy_key", &resource.Sweeper{
		Name: "github_repository_deploy_key",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_repository_deploy_key sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_repository_milestone", &resource.Sweeper{
		Name: "github_repository_milestone",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_repository_milestone sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_codespaces_secret", &resource.Sweeper{
		Name: "github_codespaces_secret",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_codespaces_secret sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_codespaces_user_secret", &resource.Sweeper{
		Name: "github_codespaces_user_secret",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_codespaces_user_secret sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_codespaces_organization_secret", &resource.Sweeper{
		Name: "github_codespaces_organization_secret",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_codespaces_organization_secret sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_dependabot_secret", &resource.Sweeper{
		Name: "github_dependabot_secret",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_dependabot_secret sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_dependabot_organization_secret", &resource.Sweeper{
		Name: "github_dependabot_organization_secret",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_dependabot_organization_secret sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_release", &resource.Sweeper{
		Name: "github_release",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_release sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_repository_environment", &resource.Sweeper{
		Name: "github_repository_environment",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_repository_environment sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_actions_environment_variable", &resource.Sweeper{
		Name: "github_actions_environment_variable",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_actions_environment_variable sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_repository_topics", &resource.Sweeper{
		Name: "github_repository_topics",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_repository_topics sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_actions_repository_permissions", &resource.Sweeper{
		Name: "github_actions_repository_permissions",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_actions_repository_permissions sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_repository_webhook", &resource.Sweeper{
		Name: "github_repository_webhook",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_repository_webhook sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_repository_autolink_reference", &resource.Sweeper{
		Name: "github_repository_autolink_reference",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_repository_autolink_reference sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_app_installation_repositories", &resource.Sweeper{
		Name: "github_app_installation_repositories",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_app_installation_repositories sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_emu_group_mapping", &resource.Sweeper{
		Name: "github_emu_group_mapping",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_emu_group_mapping sweeper")
			return nil
		},
	})

	resource.AddTestSweepers("github_team_repository", &resource.Sweeper{
		Name: "github_team_repository",
		F: func(region string) error {
			// Add cleanup code if needed
			log.Printf("[DEBUG] Running github_team_repository sweeper")
			return nil
		},
	})
}
