package github

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubCodespacesOrganizationSecretsDataSource_Migration(t *testing.T) {
	t.Run("migration compatibility between SDKv2 and Framework", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_codespaces_organization_secret" "test" {
				secret_name     = "org_cs_secret_1_%s"
				plaintext_value = "foo"
				visibility      = "private"
			}

			data "github_codespaces_organization_secrets" "test" {
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
							resource.TestCheckResourceAttr("data.github_codespaces_organization_secrets.test", "secrets.#", "1"),
							resource.TestCheckResourceAttr("data.github_codespaces_organization_secrets.test", "secrets.0.name", strings.ToUpper(fmt.Sprintf("ORG_CS_SECRET_1_%s", randomID))),
							resource.TestCheckResourceAttr("data.github_codespaces_organization_secrets.test", "secrets.0.visibility", "private"),
							resource.TestCheckResourceAttrSet("data.github_codespaces_organization_secrets.test", "secrets.0.created_at"),
							resource.TestCheckResourceAttrSet("data.github_codespaces_organization_secrets.test", "secrets.0.updated_at"),
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
