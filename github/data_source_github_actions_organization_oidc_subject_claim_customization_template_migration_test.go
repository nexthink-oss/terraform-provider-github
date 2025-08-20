package github

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSource_Migration(t *testing.T) {
	t.Run("migrate from SDKv2 to Plugin Framework without state change", func(t *testing.T) {

		config := `
			resource "github_actions_organization_oidc_subject_claim_customization_template" "test" {
				include_claim_keys = ["actor", "actor_id", "head_ref", "repository"]
			}

			data "github_actions_organization_oidc_subject_claim_customization_template" "test" {}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_actions_organization_oidc_subject_claim_customization_template.test", "include_claim_keys.#", "4"),
			resource.TestCheckResourceAttr("data.github_actions_organization_oidc_subject_claim_customization_template.test", "include_claim_keys.0", "actor"),
			resource.TestCheckResourceAttr("data.github_actions_organization_oidc_subject_claim_customization_template.test", "include_claim_keys.1", "actor_id"),
			resource.TestCheckResourceAttr("data.github_actions_organization_oidc_subject_claim_customization_template.test", "include_claim_keys.2", "head_ref"),
			resource.TestCheckResourceAttr("data.github_actions_organization_oidc_subject_claim_customization_template.test", "include_claim_keys.3", "repository"),
			resource.TestCheckResourceAttrSet("data.github_actions_organization_oidc_subject_claim_customization_template.test", "id"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
					{
						Config: config,
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
