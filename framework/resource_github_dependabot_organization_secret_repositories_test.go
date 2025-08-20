package framework

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubDependabotOrganizationSecretRepositories(t *testing.T) {

	const ORG_SECRET_NAME = "ORG_SECRET_NAME"
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	secret_name, exists := os.LookupEnv(ORG_SECRET_NAME)

	t.Run("set repository allowlist for an organization secret", func(t *testing.T) {
		if !exists {
			t.Skipf("%s environment variable is missing", ORG_SECRET_NAME)
		}

		config := fmt.Sprintf(`
			resource "github_repository" "test_repo_1" {
				name = "tf-acc-test-%s-1"
				visibility = "internal"
				vulnerability_alerts = "true"
			}

			resource "github_repository" "test_repo_2" {
				name = "tf-acc-test-%s-2"
				visibility = "internal"
				vulnerability_alerts = "true"
			}

			resource "github_dependabot_organization_secret_repositories" "org_secret_repos" {
				secret_name = "%s"
				selected_repository_ids = [
					github_repository.test_repo_1.repo_id,
					github_repository.test_repo_2.repo_id
				]
			}
		`, randomID, randomID, secret_name)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet(
				"github_dependabot_organization_secret_repositories.org_secret_repos", "secret_name",
			),
			resource.TestCheckResourceAttr(
				"github_dependabot_organization_secret_repositories.org_secret_repos", "selected_repository_ids.#", "2",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})

		t.Run("with an individual account", func(t *testing.T) {
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubDependabotOrganizationSecretRepositories_Migration(t *testing.T) {
	const ORG_SECRET_NAME = "ORG_SECRET_NAME"
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	secret_name, exists := os.LookupEnv(ORG_SECRET_NAME)

	t.Run("migration from SDKv2 to Framework", func(t *testing.T) {
		if !exists {
			t.Skipf("%s environment variable is missing", ORG_SECRET_NAME)
		}

		config := fmt.Sprintf(`
			resource "github_repository" "test_repo_1" {
				name = "tf-acc-test-%s-1"
				visibility = "internal"
				vulnerability_alerts = "true"
			}

			resource "github_repository" "test_repo_2" {
				name = "tf-acc-test-%s-2"
				visibility = "internal"
				vulnerability_alerts = "true"
			}

			resource "github_dependabot_organization_secret_repositories" "org_secret_repos" {
				secret_name = "%s"
				selected_repository_ids = [
					github_repository.test_repo_1.repo_id,
					github_repository.test_repo_2.repo_id
				]
			}
		`, randomID, randomID, secret_name)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet(
				"github_dependabot_organization_secret_repositories.org_secret_repos", "secret_name",
			),
			resource.TestCheckResourceAttr(
				"github_dependabot_organization_secret_repositories.org_secret_repos", "selected_repository_ids.#", "2",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"github": {
								Source:            "integrations/github",
								VersionConstraint: "~> 6.0",
							},
						},
						Config: config,
						Check:  check,
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectEmptyPlan(),
							},
						},
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubDependabotOrganizationSecretRepositories_ImportBasic(t *testing.T) {
	const ORG_SECRET_NAME = "ORG_SECRET_NAME"
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	secret_name, exists := os.LookupEnv(ORG_SECRET_NAME)

	t.Run("import basic", func(t *testing.T) {
		if !exists {
			t.Skipf("%s environment variable is missing", ORG_SECRET_NAME)
		}

		config := fmt.Sprintf(`
			resource "github_repository" "test_repo_1" {
				name = "tf-acc-test-%s-1"
				visibility = "internal"
				vulnerability_alerts = "true"
			}

			resource "github_repository" "test_repo_2" {
				name = "tf-acc-test-%s-2"
				visibility = "internal"
				vulnerability_alerts = "true"
			}

			resource "github_dependabot_organization_secret_repositories" "org_secret_repos" {
				secret_name = "%s"
				selected_repository_ids = [
					github_repository.test_repo_1.repo_id,
					github_repository.test_repo_2.repo_id
				]
			}
		`, randomID, randomID, secret_name)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttrSet(
								"github_dependabot_organization_secret_repositories.org_secret_repos", "secret_name",
							),
							resource.TestCheckResourceAttr(
								"github_dependabot_organization_secret_repositories.org_secret_repos", "selected_repository_ids.#", "2",
							),
						),
					},
					{
						ResourceName:      "github_dependabot_organization_secret_repositories.org_secret_repos",
						ImportState:       true,
						ImportStateVerify: true,
						ImportStateId:     secret_name,
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
