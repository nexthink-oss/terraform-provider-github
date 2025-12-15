package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGithubRepositoryCustomProperties(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("manages custom properties for a repository", func(t *testing.T) {
		config := fmt.Sprintf(`
			# Create organization-level custom properties
			resource "github_organization_custom_property" "environment" {
				name           = "environment_%[1]s"
				value_type     = "single_select"
				required       = false
				description    = "Deployment environment"
				allowed_values = ["production", "staging", "development"]
			}

			resource "github_organization_custom_property" "team" {
				name        = "team_%[1]s"
				value_type  = "string"
				required    = false
				description = "Team responsible"
			}

			# Create test repository
			resource "github_repository" "test" {
				name                                 = "tf-acc-test-%[1]s"
				auto_init                            = true
				ignore_vulnerability_alerts_during_read = true
			}

			# Apply custom properties to repository
			resource "github_repository_custom_properties" "test" {
				depends_on = [
					github_organization_custom_property.environment,
					github_organization_custom_property.team,
				]

				repository_name = github_repository.test.name

				property {
					name  = github_organization_custom_property.environment.name
					value = ["production"]
				}

				property {
					name  = github_organization_custom_property.team.name
					value = ["platform-team"]
				}
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			// Check repository name
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository_name", fmt.Sprintf("tf-acc-test-%s", randomID)),

			// Check we have 2 properties
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "property.#", "2"),

			// Check properties exist
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "property.*", map[string]string{
				"name": fmt.Sprintf("environment_%s", randomID),
			}),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "property.*", map[string]string{
				"name": fmt.Sprintf("team_%s", randomID),
			}),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:  func() { skipUnlessMode(t, mode) },
				Providers: testAccProviders,
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
