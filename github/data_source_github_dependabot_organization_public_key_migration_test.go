package github

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubDependabotOrganizationPublicKeyDataSource_Migration(t *testing.T) {
	t.Run("migrates from SDKv2 to Framework without plan changes", func(t *testing.T) {

		config := `
			data "github_dependabot_organization_public_key" "test" {}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Use external provider (SDKv2 version)
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"github": {
								Source: "integrations/github",
							},
						},
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttrSet("data.github_dependabot_organization_public_key.test", "id"),
							resource.TestCheckResourceAttrSet("data.github_dependabot_organization_public_key.test", "key_id"),
							resource.TestCheckResourceAttrSet("data.github_dependabot_organization_public_key.test", "key"),
						),
					},
					// Step 2: Switch to Framework version (should be no-op)
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   config, // Same configuration
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
