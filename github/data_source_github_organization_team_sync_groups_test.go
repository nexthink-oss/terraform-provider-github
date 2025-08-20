package github

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubOrganizationTeamSyncGroupsDataSource(t *testing.T) {
	if isEnterprise != "true" {
		t.Skip("Skipping because `ENTERPRISE_ACCOUNT` is not set or set to true")
	}

	t.Run("queries without error", func(t *testing.T) {
		config := `
			data "github_organization_team_sync_groups" "test" {}
		`

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrSet("data.github_organization_team_sync_groups.test", "groups.#"),
			resource.TestCheckResourceAttrSet("data.github_organization_team_sync_groups.test", "id"),
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

	t.Run("verifies attributes when groups exist", func(t *testing.T) {
		config := `
			data "github_organization_team_sync_groups" "test" {}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttrSet("data.github_organization_team_sync_groups.test", "id"),
							// We can't guarantee groups exist, but we can check that the attribute is present
							resource.TestCheckResourceAttrSet("data.github_organization_team_sync_groups.test", "groups.#"),
							// If groups exist, check their structure
							resource.TestCheckTypeSetElemNestedAttrs("data.github_organization_team_sync_groups.test", "groups.*", map[string]string{}),
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

func TestAccGithubOrganizationTeamSyncGroupsDataSource_Migration(t *testing.T) {
	if isEnterprise != "true" {
		t.Skip("Skipping because `ENTERPRISE_ACCOUNT` is not set or set to true")
	}

	t.Run("migration compatibility test", func(t *testing.T) {
		config := `
			data "github_organization_team_sync_groups" "test" {}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Use the muxed provider (includes both SDKv2 and Framework)
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
						Config:                   config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttrSet("data.github_organization_team_sync_groups.test", "id"),
							resource.TestCheckResourceAttrSet("data.github_organization_team_sync_groups.test", "groups.#"),
						),
					},
					// Step 2: Use the pure Framework provider to ensure no diff
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

	t.Run("behavior equivalence test", func(t *testing.T) {
		config := `
			data "github_organization_team_sync_groups" "test" {}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Framework provider - capture initial state
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttrSet("data.github_organization_team_sync_groups.test", "id"),
							resource.TestCheckResourceAttrSet("data.github_organization_team_sync_groups.test", "groups.#"),
						),
					},
					// Step 2: Muxed provider - ensure identical behavior
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
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
