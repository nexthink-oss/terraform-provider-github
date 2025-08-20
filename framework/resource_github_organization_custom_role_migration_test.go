package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubOrganizationCustomRoleResource_migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t, organization) },
		Steps: []resource.TestStep{
			{
				// Test with SDKv2 provider (via external provider or legacy test setup)
				ExternalProviders: map[string]resource.ExternalProvider{
					"github": {
						Source:            "integrations/github",
						VersionConstraint: "~> 6.0", // Last known SDKv2 version
					},
				},
				Config: testAccGithubOrganizationCustomRoleMigrationConfig(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("github_organization_custom_role.test", "id"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "base_role", "read"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "permissions.#", "2"),
				),
			},
			{
				// Test migration to Plugin Framework provider
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Config:                   testAccGithubOrganizationCustomRoleMigrationConfig(randomID),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func testAccGithubOrganizationCustomRoleMigrationConfig(randomID string) string {
	return fmt.Sprintf(`
resource "github_organization_custom_role" "test" {
  name        = "tf-acc-test-%s"
  description = "Migration test role description"
  base_role   = "read"
  permissions = [
    "reopen_issue",
    "reopen_pull_request",
  ]
}
`, randomID)
}
