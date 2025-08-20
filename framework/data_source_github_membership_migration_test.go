package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubMembershipDataSource_Migration(t *testing.T) {
	t.Run("migration compatibility test", func(t *testing.T) {
		config := fmt.Sprintf(`
			data "github_membership" "test" {
				username = "%s"
				organization = "%s"
			}
		`, testOwnerFunc(), testOrganization())

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Use the muxed provider (includes both SDKv2 and Framework)
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
						Config:                   config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_membership.test", "username", testOwnerFunc()),
							resource.TestCheckResourceAttrSet("data.github_membership.test", "role"),
							resource.TestCheckResourceAttrSet("data.github_membership.test", "etag"),
							resource.TestCheckResourceAttrSet("data.github_membership.test", "state"),
							resource.TestCheckResourceAttrSet("data.github_membership.test", "id"),
						),
					},
					// Step 2: Switch to pure Framework provider - should be no-op
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   config, // Same config
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectEmptyPlan(), // Should be no-op
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
