package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubRepositoryAutolinkReferencesDataSource_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("migration from SDKv2 to Framework", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%[1]s"
				auto_init = true
			}

			resource "github_repository_autolink_reference" "autolink_default" {
				repository = github_repository.test.name

				key_prefix          = "TEST1-"
				target_url_template = "https://example.com/TEST-<num>"
			}

			data "github_repository_autolink_references" "all" {
				repository = github_repository.test.name
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Use muxed provider (should route to SDKv2)
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
						Config:                   config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_repository_autolink_references.all", "autolink_references.#", "1"),
							resource.TestCheckResourceAttr("data.github_repository_autolink_references.all", "autolink_references.0.key_prefix", "TEST1-"),
							resource.TestCheckResourceAttr("data.github_repository_autolink_references.all", "autolink_references.0.target_url_template", "https://example.com/TEST-<num>"),
							resource.TestCheckResourceAttr("data.github_repository_autolink_references.all", "autolink_references.0.is_alphanumeric", "true"),
						),
					},
					// Step 2: Switch to Framework-only provider - should be no-op
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

		t.Run("with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
