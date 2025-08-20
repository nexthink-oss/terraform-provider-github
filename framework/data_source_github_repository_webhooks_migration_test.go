package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubRepositoryWebhooksDataSource_Migration(t *testing.T) {
	t.Run("migrates from SDKv2 to Framework without plan changes", func(t *testing.T) {
		repoName := fmt.Sprintf("tf-acc-test-webhooks-migration-%s", acctest.RandString(5))

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "%s"
				auto_init = true
			}

			resource "github_repository_webhook" "test" {
			  repository = github_repository.test.name

			  configuration {
			    url          = "https://example.com/webhook"
			    content_type = "json"
			    insecure_ssl = true
			  }

			  events = ["pull_request"]
			}

			data "github_repository_webhooks" "test" {
				repository = github_repository.test.name
			}
		`, repoName)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Use external provider (SDKv2 version)
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"github": {
								Source: "integrations/github",
							},
						},
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_repository_webhooks.test", "webhooks.#", "1"),
							resource.TestCheckResourceAttr("data.github_repository_webhooks.test", "webhooks.0.name", "web"),
							resource.TestCheckResourceAttr("data.github_repository_webhooks.test", "webhooks.0.url", "https://example.com/webhook"),
							resource.TestCheckResourceAttr("data.github_repository_webhooks.test", "webhooks.0.active", "true"),
							resource.TestCheckResourceAttrSet("data.github_repository_webhooks.test", "webhooks.0.id"),
							resource.TestCheckResourceAttrSet("data.github_repository_webhooks.test", "id"),
						),
					},
					// Step 2: Switch to Framework version (should be no-op)
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   config, // Same configuration
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
