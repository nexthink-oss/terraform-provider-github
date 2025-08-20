package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubBranchProtectionRulesDataSource_Migration(t *testing.T) {
	t.Run("migration compatibility test", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%[1]s"
				auto_init = true
			}

			data "github_branch_protection_rules" "all" {
				repository = github_repository.test.name
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"github": {
								Source:            "integrations/github", // Using SDKv2 version
								VersionConstraint: "~> 6.2.0",            // Latest SDKv2 version
							},
						},
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_branch_protection_rules.all", "rules.#", "0"),
							resource.TestCheckResourceAttrSet("data.github_branch_protection_rules.all", "repository"),
						),
					},
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
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

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("migration with branch protection rule", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%[1]s"
				auto_init = true
			}

			resource "github_branch_protection" "protection" {
				repository_id = github_repository.test.name
				pattern = "main*"
			}

			data "github_branch_protection_rules" "all" {
				repository = github_repository.test.name
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"github": {
								Source:            "integrations/github", // Using SDKv2 version
								VersionConstraint: "~> 6.2.0",            // Latest SDKv2 version
							},
						},
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_branch_protection_rules.all", "rules.#", "1"),
							resource.TestCheckResourceAttr("data.github_branch_protection_rules.all", "rules.0.pattern", "main*"),
						),
					},
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
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

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubBranchProtectionRulesDataSource_Mux(t *testing.T) {
	t.Run("mux server validation", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%[1]s"
				auto_init = true
			}

			data "github_branch_protection_rules" "all" {
				repository = github_repository.test.name
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_branch_protection_rules.all", "rules.#", "0"),
							resource.TestCheckResourceAttrSet("data.github_branch_protection_rules.all", "repository"),
						),
					},
				},
			})
		}

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
