package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubTeamDataSource_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("validates migration from SDKv2 to Framework maintains identical behavior", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_team" "test" {
				name = "tf-acc-test-%s"
			}

			data "github_team" "test" {
				slug = github_team.test.slug
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
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttrSet("data.github_team.test", "id"),
							resource.TestCheckResourceAttrSet("data.github_team.test", "name"),
							resource.TestCheckResourceAttrSet("data.github_team.test", "slug"),
							resource.TestCheckResourceAttrSet("data.github_team.test", "node_id"),
							resource.TestCheckResourceAttr("data.github_team.test", "membership_type", "all"),
							resource.TestCheckResourceAttr("data.github_team.test", "summary_only", "false"),
							resource.TestCheckResourceAttr("data.github_team.test", "results_per_page", "100"),
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
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttrSet("data.github_team.test", "id"),
							resource.TestCheckResourceAttrSet("data.github_team.test", "name"),
							resource.TestCheckResourceAttrSet("data.github_team.test", "slug"),
							resource.TestCheckResourceAttrSet("data.github_team.test", "node_id"),
							resource.TestCheckResourceAttr("data.github_team.test", "membership_type", "all"),
							resource.TestCheckResourceAttr("data.github_team.test", "summary_only", "false"),
							resource.TestCheckResourceAttr("data.github_team.test", "results_per_page", "100"),
						),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("validates migration with custom settings maintains behavior", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_team" "test" {
				name = "tf-acc-test-%s"
			}

			data "github_team" "test" {
				slug             = github_team.test.slug
				membership_type  = "immediate"
				summary_only     = true
				results_per_page = 50
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
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttrSet("data.github_team.test", "id"),
							resource.TestCheckResourceAttrSet("data.github_team.test", "name"),
							resource.TestCheckResourceAttr("data.github_team.test", "membership_type", "immediate"),
							resource.TestCheckResourceAttr("data.github_team.test", "summary_only", "true"),
							resource.TestCheckResourceAttr("data.github_team.test", "results_per_page", "50"),
							resource.TestCheckResourceAttr("data.github_team.test", "members.#", "0"),
							resource.TestCheckResourceAttr("data.github_team.test", "repositories.#", "0"),
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
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttrSet("data.github_team.test", "id"),
							resource.TestCheckResourceAttrSet("data.github_team.test", "name"),
							resource.TestCheckResourceAttr("data.github_team.test", "membership_type", "immediate"),
							resource.TestCheckResourceAttr("data.github_team.test", "summary_only", "true"),
							resource.TestCheckResourceAttr("data.github_team.test", "results_per_page", "50"),
							resource.TestCheckResourceAttr("data.github_team.test", "members.#", "0"),
							resource.TestCheckResourceAttr("data.github_team.test", "repositories.#", "0"),
						),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubTeamDataSource_Mux(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("validates mux server with both SDKv2 and Framework providers", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_team" "test" {
				name = "tf-acc-test-%s"
			}

			data "github_team" "test" {
				slug = github_team.test.slug
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeAggregateTestCheckFunc(
							resource.TestCheckResourceAttrSet("data.github_team.test", "id"),
							resource.TestCheckResourceAttrSet("data.github_team.test", "name"),
							resource.TestCheckResourceAttrSet("data.github_team.test", "slug"),
							resource.TestCheckResourceAttrSet("data.github_team.test", "node_id"),
							resource.TestCheckResourceAttr("data.github_team.test", "membership_type", "all"),
							resource.TestCheckResourceAttr("data.github_team.test", "summary_only", "false"),
							resource.TestCheckResourceAttr("data.github_team.test", "results_per_page", "100"),
						),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
