package github

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGithubTeamMembers(t *testing.T) {
	if testCollaborator == "" {
		t.Skip("Skipping because `GITHUB_TEST_COLLABORATOR` is not set")
	}

	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	resourceName := "github_team_members.test_team_members"

	var membership github.Membership

	t.Run("creates a team & members configured with defaults", func(t *testing.T) {
		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				CheckDestroy:             testAccCheckGithubTeamMembersDestroy,
				Steps: []resource.TestStep{
					{
						Config: testAccGithubTeamMembersConfig(randomID, testCollaborator, "member"),
						Check: resource.ComposeTestCheckFunc(
							testAccCheckGithubTeamMembersExists(resourceName, &membership),
							testAccCheckGithubTeamMembersRoleState(resourceName, "member", &membership),
						),
					},
					{
						Config: testAccGithubTeamMembersConfig(randomID, testCollaborator, "maintainer"),
						Check: resource.ComposeTestCheckFunc(
							testAccCheckGithubTeamMembersExists(resourceName, &membership),
							testAccCheckGithubTeamMembersRoleState(resourceName, "maintainer", &membership),
						),
					},
					{
						ResourceName:      resourceName,
						ImportState:       true,
						ImportStateVerify: true,
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
}

func testAccCheckGithubTeamMembersDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_team_members" {
			continue
		}

		teamIdString := rs.Primary.Attributes["team_id"]
		if teamIdString == "" {
			return fmt.Errorf("no team ID found in state")
		}

		teamId, err := strconv.ParseInt(teamIdString, 10, 64)
		if err != nil {
			return fmt.Errorf("could not parse team ID %s: %s", teamIdString, err.Error())
		}

		// Since we're using the Framework, we need to create a GitHub client to check destroy
		// In a real implementation, you might want to get this from test configuration
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return fmt.Errorf("GITHUB_TOKEN environment variable is required for destroy check")
		}

		owner := os.Getenv("GITHUB_OWNER")
		if owner == "" {
			return fmt.Errorf("GITHUB_OWNER environment variable is required for destroy check")
		}

		config := &Config{
			Token:   token,
			Owner:   owner,
			BaseURL: "https://api.github.com/",
		}

		meta, err := config.Meta()
		if err != nil {
			return fmt.Errorf("error creating GitHub client: %s", err.Error())
		}

		conn := meta.(*Owner).V3Client()
		orgId := meta.(*Owner).ID()

		members, resp, err := conn.Teams.ListTeamMembersByID(context.TODO(),
			orgId, teamId, nil)
		if err == nil {
			if len(members) > 0 {
				return fmt.Errorf("team has still members: %v", members)
			}
		}
		if resp.StatusCode != 404 {
			return err
		}
		return nil
	}
	return nil
}

func testAccCheckGithubTeamMembersExists(n string, membership *github.Membership) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.Attributes["team_id"] == "" {
			return fmt.Errorf("no team ID is set")
		}

		// Create GitHub client for verification
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return fmt.Errorf("GITHUB_TOKEN environment variable is required")
		}

		owner := os.Getenv("GITHUB_OWNER")
		if owner == "" {
			return fmt.Errorf("GITHUB_OWNER environment variable is required")
		}

		config := &Config{
			Token:   token,
			Owner:   owner,
			BaseURL: "https://api.github.com/",
		}

		meta, err := config.Meta()
		if err != nil {
			return fmt.Errorf("error creating GitHub client: %s", err.Error())
		}

		conn := meta.(*Owner).V3Client()
		orgId := meta.(*Owner).ID()
		teamIdString := rs.Primary.Attributes["team_id"]

		teamId, err := strconv.ParseInt(teamIdString, 10, 64)
		if err != nil {
			return fmt.Errorf("could not parse team ID %s: %s", teamIdString, err.Error())
		}

		members, _, err := conn.Teams.ListTeamMembersByID(context.TODO(), orgId, teamId, nil)
		if err != nil {
			return err
		}

		if len(members) != 1 {
			return fmt.Errorf("team has not one member: %d", len(members))
		}

		teamMembership, _, err := conn.Teams.GetTeamMembershipByID(context.TODO(), orgId, teamId, *members[0].Login)

		if err != nil {
			return err
		}
		*membership = *teamMembership
		return nil
	}
}

func testAccCheckGithubTeamMembersRoleState(n, expected string, membership *github.Membership) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.Attributes["team_id"] == "" {
			return fmt.Errorf("no team ID is set")
		}

		// Create GitHub client for verification
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return fmt.Errorf("GITHUB_TOKEN environment variable is required")
		}

		owner := os.Getenv("GITHUB_OWNER")
		if owner == "" {
			return fmt.Errorf("GITHUB_OWNER environment variable is required")
		}

		config := &Config{
			Token:   token,
			Owner:   owner,
			BaseURL: "https://api.github.com/",
		}

		meta, err := config.Meta()
		if err != nil {
			return fmt.Errorf("error creating GitHub client: %s", err.Error())
		}

		conn := meta.(*Owner).V3Client()
		orgId := meta.(*Owner).ID()
		teamIdString := rs.Primary.Attributes["team_id"]

		teamId, err := strconv.ParseInt(teamIdString, 10, 64)
		if err != nil {
			return fmt.Errorf("could not parse team ID %s: %s", teamIdString, err.Error())
		}

		members, _, err := conn.Teams.ListTeamMembersByID(context.TODO(), orgId, teamId, nil)
		if err != nil {
			return err
		}

		if len(members) != 1 {
			return fmt.Errorf("team has not one member: %d", len(members))
		}

		teamMembers, _, err := conn.Teams.GetTeamMembershipByID(context.TODO(),
			orgId, teamId, *members[0].Login)
		if err != nil {
			return err
		}

		resourceRole := membership.GetRole()
		actualRole := teamMembers.GetRole()

		if resourceRole != expected {
			return fmt.Errorf("team membership role %v in resource does match expected state of %v", resourceRole, expected)
		}

		if resourceRole != actualRole {
			return fmt.Errorf("team membership role %v in resource does match actual state of %v", resourceRole, actualRole)
		}
		return nil
	}
}

func testAccGithubTeamMembersConfig(randString, username, role string) string {
	return fmt.Sprintf(`
resource "github_membership" "test_org_membership" {
  username = "%s"
  role     = "member"
}

resource "github_team" "test_team" {
  name        = "tf-acc-test-team-membership-%s"
  description = "Terraform acc test group"
}

resource "github_team_members" "test_team_members" {
  team_id  = github_team.test_team.id
  members {
    username = "%s"
    role     = "%s"
  }

  depends_on = [github_membership.test_org_membership]
}
`, username, randString, username, role)
}
