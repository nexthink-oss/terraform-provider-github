package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubTeamRepositoryResource_migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("migrates from SDKv2 to Framework with no plan changes", func(t *testing.T) {

		config := fmt.Sprintf(`
			resource "github_team" "test" {
				name        = "tf-acc-test-team-repo-mig-%s"
				description = "test migration"
			}

			resource "github_repository" "test" {
				name = "tf-acc-test-mig-%[1]s"
			}

			resource "github_team_repository" "test" {
				team_id    = github_team.test.id
				repository = github_repository.test.name
				permission = "push"
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { testAccPreCheck(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Create using SDKv2 provider
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"github": {
								Source:            "isometry/terraform-provider-github",
								VersionConstraint: "~> 6.0",
							},
						},
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("github_team_repository.test", "permission", "push"),
							resource.TestCheckResourceAttrSet("github_team_repository.test", "team_id"),
							resource.TestCheckResourceAttrSet("github_team_repository.test", "repository"),
						),
					},
					// Step 2: Migrate to Framework provider (should be no-op)
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, "organization")
		})
	})
}