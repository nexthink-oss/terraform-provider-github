package github

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubOrganizationWebhooksDataSource_Migration(t *testing.T) {
	t.Run("migration maintains behavioral compatibility", func(t *testing.T) {
		config := `
			resource "github_organization_webhook" "test" {
			  configuration {
			    url          = "https://google.de/webhook"
			    content_type = "json"
			    insecure_ssl = true
			  }

			  events = ["pull_request"]
			}

			data "github_organization_webhooks" "test" {}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Use muxed provider (both SDKv2 and Framework available)
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"github": {
								Source:            "integrations/github",
								VersionConstraint: "~> 6.0",
							},
						},
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.#", "1"),
							resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.0.name", "web"),
							resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.0.url", "https://google.de/webhook"),
							resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.0.active", "true"),
							resource.TestCheckResourceAttrSet("data.github_organization_webhooks.test", "webhooks.0.id"),
							resource.TestCheckResourceAttrSet("data.github_organization_webhooks.test", "id"),
						),
					},
					// Step 2: Migrate to Framework provider - should be no-op
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

func TestAccGithubOrganizationWebhooksDataSource_FrameworkMigration(t *testing.T) {
	config := `
		resource "github_organization_webhook" "test" {
		  configuration {
		    url          = "https://google.de/webhook"
		    content_type = "json"
		    insecure_ssl = true
		  }

		  events = ["pull_request"]
		}

		data "github_organization_webhooks" "test" {}
	`

	resource.Test(t, resource.TestCase{
		PreCheck: func() { skipUnlessMode(t, organization) },
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"github": {
						Source:            "integrations/github",
						VersionConstraint: "~> 6.0",
					},
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.#", "1"),
					resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.0.name", "web"),
					resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.0.url", "https://google.de/webhook"),
					resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.0.active", "true"),
					resource.TestCheckResourceAttrSet("data.github_organization_webhooks.test", "webhooks.0.id"),
					resource.TestCheckResourceAttrSet("data.github_organization_webhooks.test", "id"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.#", "1"),
					resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.0.name", "web"),
					resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.0.url", "https://google.de/webhook"),
					resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.0.active", "true"),
					resource.TestCheckResourceAttrSet("data.github_organization_webhooks.test", "webhooks.0.id"),
					resource.TestCheckResourceAttr("data.github_organization_webhooks.test", "webhooks.0.type", "Repository"),
				),
			},
		},
	})
}
