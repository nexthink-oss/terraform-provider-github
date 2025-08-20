package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubEnterpriseDataSource_Migration(t *testing.T) {
	t.Run("migration compatibility test", func(t *testing.T) {
		config := fmt.Sprintf(`
			data "github_enterprise" "test" {
				slug = "%s"
			}
		`, testEnterprise())

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessEnterpriseMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Use the muxed provider (includes both SDKv2 and Framework)
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
						Config:                   config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_enterprise.test", "slug", testEnterprise()),
							resource.TestCheckResourceAttrSet("data.github_enterprise.test", "id"),
							resource.TestCheckResourceAttrSet("data.github_enterprise.test", "database_id"),
							resource.TestCheckResourceAttrSet("data.github_enterprise.test", "name"),
							resource.TestCheckResourceAttrSet("data.github_enterprise.test", "created_at"),
							resource.TestCheckResourceAttrSet("data.github_enterprise.test", "url"),
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

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})
}
