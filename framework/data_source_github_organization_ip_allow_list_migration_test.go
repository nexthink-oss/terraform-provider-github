package framework

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubOrganizationIpAllowListDataSource_Migration(t *testing.T) {
	t.Run("migrates from SDKv2 to Framework without diff", func(t *testing.T) {
		config := `
			data "github_organization_ip_allow_list" "all" {}
		`

		resource.Test(t, resource.TestCase{
			Steps: []resource.TestStep{
				// First run with muxed provider (includes SDKv2)
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"github": {
							Source:            "isometry/github",
							VersionConstraint: "~> 7.0",
						},
					},
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttrSet("data.github_organization_ip_allow_list.all", "id"),
						resource.TestCheckResourceAttr("data.github_organization_ip_allow_list.all", "ip_allow_list.#", "0"),
					),
				},
				// Then switch to pure Framework provider - should be no-op plan
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
