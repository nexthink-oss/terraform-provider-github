package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSource_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("migration test for framework compatibility", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%s"
				visibility = "private"
			}
	
			resource "github_actions_repository_oidc_subject_claim_customization_template" "test" {
				repository = github_repository.test.name
				use_default = false
				include_claim_keys = ["repo", "context", "job_workflow_ref"]
			}

			data "github_actions_repository_oidc_subject_claim_customization_template" "test" {
				name = github_repository.test.name
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_actions_repository_oidc_subject_claim_customization_template.test", "use_default", "false"),
							resource.TestCheckResourceAttr("data.github_actions_repository_oidc_subject_claim_customization_template.test", "include_claim_keys.#", "3"),
							resource.TestCheckResourceAttr("data.github_actions_repository_oidc_subject_claim_customization_template.test", "include_claim_keys.0", "repo"),
							resource.TestCheckResourceAttr("data.github_actions_repository_oidc_subject_claim_customization_template.test", "include_claim_keys.1", "context"),
							resource.TestCheckResourceAttr("data.github_actions_repository_oidc_subject_claim_customization_template.test", "include_claim_keys.2", "job_workflow_ref"),
							resource.TestCheckResourceAttrSet("data.github_actions_repository_oidc_subject_claim_customization_template.test", "id"),
						),
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

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
