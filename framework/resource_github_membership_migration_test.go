package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubMembership_migration(t *testing.T) {
	if testCollaborator == "" {
		t.Skip("Skipping because `GITHUB_TEST_COLLABORATOR` is not set")
	}
	if err := testAccCheckOrganization(); err != nil {
		t.Skipf("Skipping because %s.", err.Error())
	}

	t.Run("migrates from SDKv2 to Framework without changes", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"github": {
							Source:            "integrations/github",
							VersionConstraint: "~> 6.0",
						},
					},
					Config: testAccGithubMembershipConfig(testCollaborator),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "username", testCollaborator),
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "role", "member"),
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "downgrade_on_destroy", "false"),
						resource.TestCheckResourceAttrSet("github_membership.test_org_membership", "id"),
						resource.TestCheckResourceAttrSet("github_membership.test_org_membership", "etag"),
					),
				},
				{
					ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
					Config:                   testAccGithubMembershipConfig(testCollaborator),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						},
					},
				},
			},
		})
	})

	t.Run("migrates from SDKv2 to Framework with downgrade_on_destroy", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"github": {
							Source:            "integrations/github",
							VersionConstraint: "~> 6.0",
						},
					},
					Config: testAccGithubMembershipConfigDowngradable(testCollaborator),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "username", testCollaborator),
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "role", "admin"),
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "downgrade_on_destroy", "true"),
						resource.TestCheckResourceAttrSet("github_membership.test_org_membership", "id"),
						resource.TestCheckResourceAttrSet("github_membership.test_org_membership", "etag"),
					),
				},
				{
					ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
					Config:                   testAccGithubMembershipConfigDowngradable(testCollaborator),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						},
					},
				},
			},
		})
	})

	t.Run("case insensitive usernames work consistently", func(t *testing.T) {
		otherCase := flipUsernameCase(testCollaborator)
		if testCollaborator == otherCase {
			t.Skip("Skipping because `GITHUB_TEST_COLLABORATOR` has no letters to flip case")
		}

		resource.Test(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"github": {
							Source:            "integrations/github",
							VersionConstraint: "~> 6.0",
						},
					},
					Config: testAccGithubMembershipConfig(testCollaborator),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "username", testCollaborator),
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "role", "member"),
					),
				},
				{
					ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
					Config:                   testAccGithubMembershipConfig(otherCase),
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

func TestAccGithubMembership_muxValidation(t *testing.T) {
	if testCollaborator == "" {
		t.Skip("Skipping because `GITHUB_TEST_COLLABORATOR` is not set")
	}
	if err := testAccCheckOrganization(); err != nil {
		t.Skipf("Skipping because %s.", err.Error())
	}

	t.Run("validates mux server handles github_membership resource correctly", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t, organization) },
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: testAccGithubMembershipConfig(testCollaborator),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "username", testCollaborator),
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "role", "member"),
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "downgrade_on_destroy", "false"),
						resource.TestCheckResourceAttrSet("github_membership.test_org_membership", "id"),
						resource.TestCheckResourceAttrSet("github_membership.test_org_membership", "etag"),
					),
				},
				{
					Config: testAccGithubMembershipConfigWithRoleUpdate(testCollaborator, "admin"),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "username", testCollaborator),
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "role", "admin"),
						resource.TestCheckResourceAttr("github_membership.test_org_membership", "downgrade_on_destroy", "false"),
						resource.TestCheckResourceAttrSet("github_membership.test_org_membership", "id"),
						resource.TestCheckResourceAttrSet("github_membership.test_org_membership", "etag"),
					),
				},
				{
					ResourceName:      "github_membership.test_org_membership",
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	})
}

func testAccGithubMembershipConfigWithRoleUpdate(username, role string) string {
	return fmt.Sprintf(`
  resource "github_membership" "test_org_membership" {
    username = "%s"
    role = "%s"
  }
`, username, role)
}
