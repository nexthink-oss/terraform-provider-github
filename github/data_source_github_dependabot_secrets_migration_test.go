package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubDependabotSecretsDataSource_Migration(t *testing.T) {
	t.Run("migrates from SDKv2 to Framework without breaking changes", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_dependabot_secret" "test" {
				secret_name     = "dep_secret_1"
				repository      = github_repository.test.name
				plaintext_value = "foo"
			}

			data "github_dependabot_secrets" "test" {
				name = github_repository.test.name
			}
		`, randomID)

		// Shared test checks for both providers
		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_dependabot_secrets.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
			resource.TestCheckResourceAttr("data.github_dependabot_secrets.test", "secrets.#", "1"),
			resource.TestCheckResourceAttr("data.github_dependabot_secrets.test", "secrets.0.name", "DEP_SECRET_1"),
			resource.TestCheckResourceAttrSet("data.github_dependabot_secrets.test", "secrets.0.created_at"),
			resource.TestCheckResourceAttrSet("data.github_dependabot_secrets.test", "secrets.0.updated_at"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Use muxed provider (should use SDKv2 implementation)
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
						Config:                   config,
						Check:                    check,
					},
					// Step 2: Use pure Framework provider (should produce no-op plan)
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

	t.Run("validates full_name functionality consistency between SDKv2 and Framework", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_dependabot_secret" "test" {
				secret_name     = "dep_secret_1"
				repository      = github_repository.test.name
				plaintext_value = "foo"
			}

			data "github_dependabot_secrets" "test" {
				full_name = github_repository.test.full_name
			}
		`, randomID)

		// Shared test checks for both providers
		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_dependabot_secrets.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
			resource.TestCheckResourceAttrSet("data.github_dependabot_secrets.test", "full_name"),
			resource.TestCheckResourceAttr("data.github_dependabot_secrets.test", "secrets.#", "1"),
			resource.TestCheckResourceAttr("data.github_dependabot_secrets.test", "secrets.0.name", "DEP_SECRET_1"),
			resource.TestCheckResourceAttrSet("data.github_dependabot_secrets.test", "secrets.0.created_at"),
			resource.TestCheckResourceAttrSet("data.github_dependabot_secrets.test", "secrets.0.updated_at"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Use muxed provider (should use SDKv2 implementation)
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
						Config:                   config,
						Check:                    check,
					},
					// Step 2: Use pure Framework provider (should produce no-op plan)
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
