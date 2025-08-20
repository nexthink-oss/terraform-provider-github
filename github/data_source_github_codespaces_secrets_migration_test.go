package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubCodespacesSecretsDataSource_Migration(t *testing.T) {
	t.Run("migration compatibility between SDKv2 and Framework", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_codespaces_secret" "test" {
				secret_name     = "cs_secret_1"
				repository      = github_repository.test.name
				plaintext_value = "foo"
			}

			data "github_codespaces_secrets" "test" {
				name = github_repository.test.name
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Use muxed provider (SDKv2 + Framework)
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
						Config:                   config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_codespaces_secrets.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
							resource.TestCheckResourceAttr("data.github_codespaces_secrets.test", "secrets.#", "1"),
							resource.TestCheckResourceAttr("data.github_codespaces_secrets.test", "secrets.0.name", "CS_SECRET_1"),
							resource.TestCheckResourceAttrSet("data.github_codespaces_secrets.test", "secrets.0.created_at"),
							resource.TestCheckResourceAttrSet("data.github_codespaces_secrets.test", "secrets.0.updated_at"),
						),
					},
					// Step 2: Switch to pure Framework provider - should be no-op plan
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

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("migration compatibility using full_name", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_codespaces_secret" "test" {
				secret_name     = "cs_secret_1"
				repository      = github_repository.test.name
				plaintext_value = "foo"
			}

			data "github_codespaces_secrets" "test" {
				full_name = github_repository.test.full_name
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Use muxed provider (SDKv2 + Framework)
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
						Config:                   config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_codespaces_secrets.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
							resource.TestCheckResourceAttrSet("data.github_codespaces_secrets.test", "full_name"),
							resource.TestCheckResourceAttr("data.github_codespaces_secrets.test", "secrets.#", "1"),
							resource.TestCheckResourceAttr("data.github_codespaces_secrets.test", "secrets.0.name", "CS_SECRET_1"),
							resource.TestCheckResourceAttrSet("data.github_codespaces_secrets.test", "secrets.0.created_at"),
							resource.TestCheckResourceAttrSet("data.github_codespaces_secrets.test", "secrets.0.updated_at"),
						),
					},
					// Step 2: Switch to pure Framework provider - should be no-op plan
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
