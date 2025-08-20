package github

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubActionsOrganizationVariablesDataSource(t *testing.T) {

	t.Run("queries organization actions variables from a repository", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_actions_organization_variable" "test" {
				variable_name 		= "org_variable_%s"
				value = "foo"
				visibility       = "all"
			}
	`, randomID)

		config2 := config + `
			data "github_actions_organization_variables" "test" {
			}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_actions_organization_variables.test", "variables.#", "1"),
			resource.TestCheckResourceAttr("data.github_actions_organization_variables.test", "variables.0.name", strings.ToUpper(fmt.Sprintf("ORG_VARIABLE_%s", randomID))),
			resource.TestCheckResourceAttr("data.github_actions_organization_variables.test", "variables.0.value", "foo"),
			resource.TestCheckResourceAttr("data.github_actions_organization_variables.test", "variables.0.visibility", "all"),
			resource.TestCheckResourceAttrSet("data.github_actions_organization_variables.test", "variables.0.created_at"),
			resource.TestCheckResourceAttrSet("data.github_actions_organization_variables.test", "variables.0.updated_at"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  resource.ComposeTestCheckFunc(),
					},
					{
						Config: config2,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

// Migration test to ensure compatibility with SDKv2 version
func TestAccGithubActionsOrganizationVariablesDataSource_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	config := fmt.Sprintf(`
		resource "github_actions_organization_variable" "test" {
			variable_name 		= "org_variable_migration_%s"
			value = "foo"
			visibility      = "all"
		}

		data "github_actions_organization_variables" "test" {
		}
	`, randomID)

	check := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("data.github_actions_organization_variables.test", "variables.#", "1"),
		resource.TestCheckResourceAttr("data.github_actions_organization_variables.test", "variables.0.name", strings.ToUpper(fmt.Sprintf("ORG_VARIABLE_MIGRATION_%s", randomID))),
		resource.TestCheckResourceAttr("data.github_actions_organization_variables.test", "variables.0.value", "foo"),
		resource.TestCheckResourceAttr("data.github_actions_organization_variables.test", "variables.0.visibility", "all"),
		resource.TestCheckResourceAttrSet("data.github_actions_organization_variables.test", "variables.0.created_at"),
		resource.TestCheckResourceAttrSet("data.github_actions_organization_variables.test", "variables.0.updated_at"),
		resource.TestCheckResourceAttrSet("data.github_actions_organization_variables.test", "id"),
	)

	testCase := func(t *testing.T, mode string) {
		resource.Test(t, resource.TestCase{
			PreCheck: func() { skipUnlessMode(t, mode) },
			Steps: []resource.TestStep{
				// Test with SDKv2 version (external provider)
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
				// Test with Framework version (should be identical)
				{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Config:                   config,
					Check:                    check,
				},
			},
		})
	}

	t.Run("with an organization account", func(t *testing.T) {
		testCase(t, organization)
	})
}
