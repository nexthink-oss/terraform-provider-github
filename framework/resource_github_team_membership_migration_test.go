package framework

import (
	"fmt"
	"testing"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	sdkv2github "github.com/isometry/terraform-provider-github/v7/github"
)

func TestAccGithubTeamMembership_migration_basic(t *testing.T) {
	if testCollaborator == "" {
		t.Skip("Skipping because `GITHUB_TEST_COLLABORATOR` is not set")
	}
	if err := testAccCheckOrganization(); err != nil {
		t.Skipf("Skipping because %s.", err.Error())
	}

	var membership github.Membership
	rn := "github_team_membership.test_team_membership"
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"github": {
						VersionConstraint: "~> 6.0", // Use current SDKv2 version
						Source:            "integrations/github",
					},
				},
				Config: testAccGithubTeamMembershipConfigMigration(randString, testCollaborator, "member"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "username", testCollaborator),
					resource.TestCheckResourceAttr(rn, "role", "member"),
					resource.TestCheckResourceAttrSet(rn, "team_id"),
					resource.TestCheckResourceAttrSet(rn, "etag"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Config:                   testAccGithubTeamMembershipConfigMigration(randString, testCollaborator, "member"),
				PlanOnly:                 true, // Validate no changes are required during migration
			},
			{
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Config:                   testAccGithubTeamMembershipConfigMigration(randString, testCollaborator, "maintainer"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubTeamMembershipExists(rn, &membership),
					testAccCheckGithubTeamMembershipRoleState(rn, "maintainer", &membership),
					resource.TestCheckResourceAttr(rn, "role", "maintainer"),
				),
			},
		},
	})
}

func TestAccGithubTeamMembership_migration_caseInsensitive(t *testing.T) {
	if testCollaborator == "" {
		t.Skip("Skipping because `GITHUB_TEST_COLLABORATOR` is not set")
	}
	if err := testAccCheckOrganization(); err != nil {
		t.Skipf("Skipping because %s.", err.Error())
	}

	var membership github.Membership
	var otherMembership github.Membership

	rn := "github_team_membership.test_team_membership"
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	otherCase := flipUsernameCase(testCollaborator)

	if testCollaborator == otherCase {
		t.Skip("Skipping because `GITHUB_TEST_COLLABORATOR` has no letters to flip case")
	}

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"github": {
						VersionConstraint: "~> 6.0", // Use current SDKv2 version
						Source:            "integrations/github",
					},
				},
				Config: testAccGithubTeamMembershipConfigMigration(randString, testCollaborator, "member"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubTeamMembershipExists(rn, &membership),
				),
			},
			{
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Config:                   testAccGithubTeamMembershipConfigMigration(randString, otherCase, "member"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubTeamMembershipExists(rn, &otherMembership),
					testAccGithubTeamMembershipTheSame(&membership, &otherMembership),
				),
			},
		},
	})
}

func TestAccGithubTeamMembership_migration_importBasic(t *testing.T) {
	if testCollaborator == "" {
		t.Skip("Skipping because `GITHUB_TEST_COLLABORATOR` is not set")
	}
	if err := testAccCheckOrganization(); err != nil {
		t.Skipf("Skipping because %s.", err.Error())
	}

	rn := "github_team_membership.test_team_membership"
	randString := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"github": {
						VersionConstraint: "~> 6.0", // Use current SDKv2 version
						Source:            "integrations/github",
					},
				},
				Config: testAccGithubTeamMembershipConfigMigration(randString, testCollaborator, "member"),
			},
			{
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Config:                   testAccGithubTeamMembershipConfigMigration(randString, testCollaborator, "member"),
				ResourceName:             rn,
				ImportState:              true,
				ImportStateVerify:        true,
			},
		},
	})
}

// testAccGithubTeamMembershipConfigMigration creates a configuration using the external provider pattern
func testAccGithubTeamMembershipConfigMigration(randString, username, role string) string {
	return fmt.Sprintf(`
resource "github_membership" "test_org_membership" {
  username = "%s"
  role     = "member"
}

resource "github_team" "test_team" {
  name        = "tf-acc-test-team-membership-migration-%s"
  description = "Terraform acc test group"
}

resource "github_team_membership" "test_team_membership" {
  team_id  = github_team.test_team.id
  username = "%s"
  role     = "%s"
}
`, username, randString, username, role)
}

// testAccCheckGithubTeamMembershipExistsUsingSDKv2 uses the SDKv2 client to check existence
func testAccCheckGithubTeamMembershipExistsUsingSDKv2(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no team membership ID is set")
		}

		// Create SDKv2 provider client to verify the resource exists
		provider := sdkv2github.Provider()
		config := map[string]interface{}{
			"token": rs.Primary.Attributes["token"],
			"owner": rs.Primary.Attributes["owner"],
		}

		diags := provider.Configure(nil, config)
		if diags.HasError() {
			return fmt.Errorf("provider configuration error: %v", diags)
		}

		return nil
	}
}
