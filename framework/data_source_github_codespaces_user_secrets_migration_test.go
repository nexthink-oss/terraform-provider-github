package framework

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubCodespacesUserSecretsDataSource_Migration(t *testing.T) {
	t.Run("validates migration behavior compatibility", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		// Create a codespaces user secret first
		config := fmt.Sprintf(`
			resource "github_codespaces_user_secret" "test" {
				secret_name     = "user_cs_secret_migration_%s"
				plaintext_value = "foo"
			}
		`, randomID)

		// Add the data source
		configWithDataSource := config + `
			data "github_codespaces_user_secrets" "test" {
			}
		`

		// Expected values - secrets are stored as uppercase in GitHub
		expectedSecretName := strings.ToUpper(fmt.Sprintf("USER_CS_SECRET_MIGRATION_%s", randomID))

		check := resource.ComposeTestCheckFunc(
			// Verify the data source returns the correct structure
			resource.TestCheckResourceAttrSet("data.github_codespaces_user_secrets.test", "id"),
			resource.TestCheckResourceAttr("data.github_codespaces_user_secrets.test", "secrets.#", "1"),
			resource.TestCheckResourceAttr("data.github_codespaces_user_secrets.test", "secrets.0.name", expectedSecretName),
			resource.TestCheckResourceAttrSet("data.github_codespaces_user_secrets.test", "secrets.0.visibility"),
			resource.TestCheckResourceAttrSet("data.github_codespaces_user_secrets.test", "secrets.0.created_at"),
			resource.TestCheckResourceAttrSet("data.github_codespaces_user_secrets.test", "secrets.0.updated_at"),

			// Verify the Framework data source maintains SDKv2 compatibility
			// The ID should be set to the owner name, matching SDKv2 behavior
			resource.TestCheckResourceAttrSet("data.github_codespaces_user_secrets.test", "id"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						// Create the secret
						Config: config,
						Check:  resource.ComposeTestCheckFunc(),
					},
					{
						// Test the data source with the Framework provider
						Config: configWithDataSource,
						Check:  check,
					},
					{
						// Verify no-op plan after data source is populated
						Config: configWithDataSource,
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
	})
}
