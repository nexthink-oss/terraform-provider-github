package framework

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubOrganizationExternalIdentitiesDataSource_Migration(t *testing.T) {
	if os.Getenv("ENTERPRISE_ACCOUNT") != "true" {
		t.Skip("Skipping because `ENTERPRISE_ACCOUNT` is not set or set to false")
	}

	t.Run("migration maintains compatibility", func(t *testing.T) {
		config := `data "github_organization_external_identities" "test" {}`

		resource.Test(t, resource.TestCase{
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
						resource.TestCheckResourceAttrSet("data.github_organization_external_identities.test", "id"),
						resource.TestCheckResourceAttrSet("data.github_organization_external_identities.test", "identities.#"),
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
