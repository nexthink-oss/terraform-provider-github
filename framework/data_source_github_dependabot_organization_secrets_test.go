package framework

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubDependabotOrganizationSecretsDataSource(t *testing.T) {

	t.Run("queries organization dependabot secrets from a repository", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_dependabot_organization_secret" "test" {
				secret_name 		= "org_dep_secret_1_%s"
				plaintext_value = "foo"
				visibility      = "private"
			}
		`, randomID)

		config2 := config + `
			data "github_dependabot_organization_secrets" "test" {
			}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_dependabot_organization_secrets.test", "secrets.#", "1"),
			resource.TestCheckResourceAttr("data.github_dependabot_organization_secrets.test", "secrets.0.name", strings.ToUpper(fmt.Sprintf("ORG_DEP_SECRET_1_%s", randomID))),
			resource.TestCheckResourceAttr("data.github_dependabot_organization_secrets.test", "secrets.0.visibility", "private"),
			resource.TestCheckResourceAttrSet("data.github_dependabot_organization_secrets.test", "secrets.0.created_at"),
			resource.TestCheckResourceAttrSet("data.github_dependabot_organization_secrets.test", "secrets.0.updated_at"),
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
func TestAccGithubDependabotOrganizationSecretsDataSource_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	config := fmt.Sprintf(`
		resource "github_dependabot_organization_secret" "test" {
			secret_name 		= "org_dep_secret_migration_%s"
			plaintext_value = "foo"
			visibility      = "private"
		}

		data "github_dependabot_organization_secrets" "test" {
		}
	`, randomID)

	check := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("data.github_dependabot_organization_secrets.test", "secrets.#", "1"),
		resource.TestCheckResourceAttr("data.github_dependabot_organization_secrets.test", "secrets.0.name", strings.ToUpper(fmt.Sprintf("ORG_DEP_SECRET_MIGRATION_%s", randomID))),
		resource.TestCheckResourceAttr("data.github_dependabot_organization_secrets.test", "secrets.0.visibility", "private"),
		resource.TestCheckResourceAttrSet("data.github_dependabot_organization_secrets.test", "secrets.0.created_at"),
		resource.TestCheckResourceAttrSet("data.github_dependabot_organization_secrets.test", "secrets.0.updated_at"),
		resource.TestCheckResourceAttrSet("data.github_dependabot_organization_secrets.test", "id"),
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
