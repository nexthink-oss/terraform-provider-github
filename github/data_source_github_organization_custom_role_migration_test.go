package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubOrganizationCustomRoleDataSource_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("validates migration from SDKv2 to Framework maintains identical behavior", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_organization_custom_role" "test" {
				name        = "tf-acc-test-%s"
				description = "Test role description"
				base_role   = "read"
				permissions = [
					"reopen_issue",
					"reopen_pull_request",
				]
			}
		`, randomID)

		config2 := config + `
			data "github_organization_custom_role" "test" {
				name = github_organization_custom_role.test.name
			}
		`

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
						Check:  resource.ComposeTestCheckFunc(),
					},
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"github": {
								Source:            "integrations/github",
								VersionConstraint: "~> 6.0",
							},
						},
						Config: config2,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttrSet("data.github_organization_custom_role.test", "id"),
							resource.TestCheckResourceAttrSet("data.github_organization_custom_role.test", "name"),
							resource.TestCheckResourceAttr("data.github_organization_custom_role.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
							resource.TestCheckResourceAttr("data.github_organization_custom_role.test", "description", "Test role description"),
							resource.TestCheckResourceAttr("data.github_organization_custom_role.test", "base_role", "read"),
							resource.TestCheckResourceAttr("data.github_organization_custom_role.test", "permissions.#", "2"),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   config2,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectEmptyPlan(),
							},
						},
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttrSet("data.github_organization_custom_role.test", "id"),
							resource.TestCheckResourceAttrSet("data.github_organization_custom_role.test", "name"),
							resource.TestCheckResourceAttr("data.github_organization_custom_role.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
							resource.TestCheckResourceAttr("data.github_organization_custom_role.test", "description", "Test role description"),
							resource.TestCheckResourceAttr("data.github_organization_custom_role.test", "base_role", "read"),
							resource.TestCheckResourceAttr("data.github_organization_custom_role.test", "permissions.#", "2"),
						),
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
