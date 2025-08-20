package github

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubOrganizationTeamsDataSource_Migration(t *testing.T) {
	t.Run("migration compatibility test", func(t *testing.T) {
		config := `
			data "github_organization_teams" "all" {}
		`

		resource.Test(t, resource.TestCase{
			PreCheck: func() { skipUnlessMode(t, organization) },
			Steps: []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"github": {Source: "integrations/github"},
					},
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("data.github_organization_teams.all", "id"),
						resource.TestCheckResourceAttrSet("data.github_organization_teams.all", "teams.#"),
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
	})

	t.Run("migration compatibility test with root_teams_only", func(t *testing.T) {
		config := `
			data "github_organization_teams" "root_teams" {
				root_teams_only = true
			}
		`

		resource.Test(t, resource.TestCase{
			PreCheck: func() { skipUnlessMode(t, organization) },
			Steps: []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"github": {Source: "integrations/github"},
					},
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("data.github_organization_teams.root_teams", "id"),
						resource.TestCheckResourceAttrSet("data.github_organization_teams.root_teams", "teams.#"),
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
	})

	t.Run("migration compatibility test with summary_only", func(t *testing.T) {
		config := `
			data "github_organization_teams" "summary" {
				summary_only = true
			}
		`

		resource.Test(t, resource.TestCase{
			PreCheck: func() { skipUnlessMode(t, organization) },
			Steps: []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"github": {Source: "integrations/github"},
					},
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("data.github_organization_teams.summary", "id"),
						resource.TestCheckResourceAttrSet("data.github_organization_teams.summary", "teams.#"),
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
	})

	t.Run("migration compatibility test with results_per_page", func(t *testing.T) {
		config := `
			data "github_organization_teams" "paginated" {
				results_per_page = 50
			}
		`

		resource.Test(t, resource.TestCase{
			PreCheck: func() { skipUnlessMode(t, organization) },
			Steps: []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"github": {Source: "integrations/github"},
					},
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("data.github_organization_teams.paginated", "id"),
						resource.TestCheckResourceAttrSet("data.github_organization_teams.paginated", "teams.#"),
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
	})
}
