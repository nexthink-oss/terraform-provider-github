package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubTeam_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("migrates from SDKv2 to Framework", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_team" "test" {
				name                      = "tf-acc-migration-%s"
				description               = "A team for migration testing"
				privacy                   = "closed"
				create_default_maintainer = false
			}
		`, randomID)

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
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("github_team.test", "name", fmt.Sprintf("tf-acc-migration-%s", randomID)),
							resource.TestCheckResourceAttr("github_team.test", "description", "A team for migration testing"),
							resource.TestCheckResourceAttr("github_team.test", "privacy", "closed"),
							resource.TestCheckResourceAttr("github_team.test", "create_default_maintainer", "false"),
							resource.TestCheckResourceAttrSet("github_team.test", "id"),
							resource.TestCheckResourceAttrSet("github_team.test", "slug"),
							resource.TestCheckResourceAttrSet("github_team.test", "node_id"),
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
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("migrates hierarchical teams from SDKv2 to Framework", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_team" "parent" {
				name        = "tf-acc-migration-parent-%s"
				description = "Parent team for migration testing"
				privacy     = "closed"
			}

			resource "github_team" "child" {
				name           = "tf-acc-migration-child-%[1]s"
				description    = "Child team for migration testing"
				privacy        = "closed"
				parent_team_id = github_team.parent.id
			}
		`, randomID)

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
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("github_team.parent", "name", fmt.Sprintf("tf-acc-migration-parent-%s", randomID)),
							resource.TestCheckResourceAttr("github_team.child", "name", fmt.Sprintf("tf-acc-migration-child-%s", randomID)),
							resource.TestCheckResourceAttrSet("github_team.child", "parent_team_id"),
							resource.TestCheckResourceAttrSet("github_team.child", "parent_team_read_id"),
							resource.TestCheckResourceAttrSet("github_team.child", "parent_team_read_slug"),
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
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
