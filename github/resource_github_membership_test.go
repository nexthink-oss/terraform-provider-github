package github

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"unicode"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

)

// testCollaborator is defined in resource_github_user_invitation_accepter_test.go

func testAccCheckOrganization() error {
	baseURL := os.Getenv("GITHUB_BASE_URL")
	token := os.Getenv("GITHUB_TOKEN")

	owner := os.Getenv("GITHUB_OWNER")
	if owner == "" {
		organization := os.Getenv("GITHUB_ORGANIZATION")
		if organization == "" {
			return fmt.Errorf("neither `GITHUB_OWNER` or `GITHUB_ORGANIZATION` set in environment")
		}
		owner = organization
	}

	config := Config{
		Token:   token,
		Owner:   owner,
		BaseURL: baseURL,
	}

	meta, err := config.Meta()
	if err != nil {
		return err
	}

	if !meta.(*Owner).IsOrganization {
		return fmt.Errorf("configured owner %q is not a GitHub organization", owner)
	}

	return nil
}

func flipUsernameCase(username string) string {
	oc := []rune(username)

	for i, ch := range oc {
		if unicode.IsLetter(ch) {
			if unicode.IsUpper(ch) {
				oc[i] = unicode.ToLower(ch)
			} else {
				oc[i] = unicode.ToUpper(ch)
			}
			break
		}
	}
	return string(oc)
}

func TestAccGithubMembership_basic(t *testing.T) {
	if testCollaborator == "" {
		t.Skip("Skipping because `GITHUB_TEST_COLLABORATOR` is not set")
	}
	if err := testAccCheckOrganization(); err != nil {
		t.Skipf("Skipping because %s.", err.Error())
	}

	var membership github.Membership
	rn := "github_membership.test_org_membership"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckGithubMembershipDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubMembershipConfig(testCollaborator),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubMembershipExists(rn, &membership),
					testAccCheckGithubMembershipRoleState(rn, &membership),
				),
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGithubMembership_downgrade(t *testing.T) {
	if testCollaborator == "" {
		t.Skip("Skipping because `GITHUB_TEST_COLLABORATOR` is not set")
	}
	if err := testAccCheckOrganization(); err != nil {
		t.Skipf("Skipping because %s.", err.Error())
	}

	var membership github.Membership
	rn := "github_membership.test_org_membership"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckGithubMembershipDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubMembershipConfigDowngradable(testCollaborator),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubMembershipExists(rn, &membership),
					testAccCheckGithubMembershipRoleState(rn, &membership),
				),
			},
			{
				ResourceName: rn,
				ImportState:  true,
			},
		},
	})
}

func TestAccGithubMembership_caseInsensitive(t *testing.T) {
	if testCollaborator == "" {
		t.Skip("Skipping because `GITHUB_TEST_COLLABORATOR` is not set")
	}
	if err := testAccCheckOrganization(); err != nil {
		t.Skipf("Skipping because %s.", err.Error())
	}

	var membership github.Membership
	var otherMembership github.Membership

	rn := "github_membership.test_org_membership"
	otherCase := flipUsernameCase(testCollaborator)

	if testCollaborator == otherCase {
		t.Skip("Skipping because `GITHUB_TEST_COLLABORATOR` has no letters to flip case")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		CheckDestroy:             testAccCheckGithubMembershipDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubMembershipConfig(testCollaborator),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubMembershipExists(rn, &membership),
				),
			},
			{
				Config: testAccGithubMembershipConfig(otherCase),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGithubMembershipExists(rn, &otherMembership),
					testAccGithubMembershipTheSame(&membership, &otherMembership),
				),
			},
			{
				ResourceName:      rn,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckGithubMembershipDestroy(s *terraform.State) error {
	// Create a new GitHub client directly for destroy check
	token := os.Getenv("GITHUB_TOKEN")
	owner := os.Getenv("GITHUB_OWNER")
	if owner == "" {
		owner = os.Getenv("GITHUB_ORGANIZATION")
	}
	baseURL := os.Getenv("GITHUB_BASE_URL")

	config := Config{
		Token:   token,
		Owner:   owner,
		BaseURL: baseURL,
	}

	meta, err := config.Meta()
	if err != nil {
		return err
	}

	conn := meta.(*Owner).V3Client()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_membership" {
			continue
		}

		orgName, username, err := parseTwoPartID(rs.Primary.ID, "organization", "username")
		if err != nil {
			return err
		}

		downgradedOnDestroy := rs.Primary.Attributes["downgrade_on_destroy"] == "true"
		membership, resp, err := conn.Organizations.GetOrgMembership(context.TODO(), username, orgName)
		responseIsSuccessful := err == nil && membership != nil && buildTwoPartID(orgName, username) == rs.Primary.ID

		if downgradedOnDestroy {
			if !responseIsSuccessful {
				return fmt.Errorf("could not load organization membership for %q", rs.Primary.ID)
			}

			if *membership.Role != "member" {
				return fmt.Errorf("organization membership %q is not a member of the org or is not the 'member' role", rs.Primary.ID)
			}

			// Now actually remove them from the org to clean up
			_, removeErr := conn.Organizations.RemoveOrgMembership(context.TODO(), username, orgName)
			if removeErr != nil {
				return fmt.Errorf("organization membership %q could not be removed during membership downgrade test case cleanup: %s", rs.Primary.ID, removeErr)
			}
		} else if responseIsSuccessful {
			return fmt.Errorf("organization membership %q still exists", rs.Primary.ID)
		} else if resp.StatusCode != 404 {
			return err
		}

		return nil
	}
	return nil
}

func testAccCheckGithubMembershipExists(n string, membership *github.Membership) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no membership ID is set")
		}

		// Create a new GitHub client for the check
		token := os.Getenv("GITHUB_TOKEN")
		owner := os.Getenv("GITHUB_OWNER")
		if owner == "" {
			owner = os.Getenv("GITHUB_ORGANIZATION")
		}
		baseURL := os.Getenv("GITHUB_BASE_URL")

		config := Config{
			Token:   token,
			Owner:   owner,
			BaseURL: baseURL,
		}

		meta, err := config.Meta()
		if err != nil {
			return err
		}

		conn := meta.(*Owner).V3Client()
		orgName, username, err := parseTwoPartID(rs.Primary.ID, "organization", "username")
		if err != nil {
			return err
		}

		githubMembership, _, err := conn.Organizations.GetOrgMembership(context.TODO(), username, orgName)
		if err != nil {
			return err
		}
		*membership = *githubMembership
		return nil
	}
}

func testAccCheckGithubMembershipRoleState(n string, membership *github.Membership) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not Found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no membership ID is set")
		}

		// Create a new GitHub client for the check
		token := os.Getenv("GITHUB_TOKEN")
		owner := os.Getenv("GITHUB_OWNER")
		if owner == "" {
			owner = os.Getenv("GITHUB_ORGANIZATION")
		}
		baseURL := os.Getenv("GITHUB_BASE_URL")

		config := Config{
			Token:   token,
			Owner:   owner,
			BaseURL: baseURL,
		}

		meta, err := config.Meta()
		if err != nil {
			return err
		}

		conn := meta.(*Owner).V3Client()
		orgName, username, err := parseTwoPartID(rs.Primary.ID, "organization", "username")
		if err != nil {
			return err
		}

		githubMembership, _, err := conn.Organizations.GetOrgMembership(context.TODO(), username, orgName)
		if err != nil {
			return err
		}

		resourceRole := membership.GetRole()
		actualRole := githubMembership.GetRole()

		if resourceRole != actualRole {
			return fmt.Errorf("membership role %v in resource does match actual state of %v",
				resourceRole, actualRole)
		}
		return nil
	}
}

func testAccGithubMembershipConfig(username string) string {
	return fmt.Sprintf(`
  resource "github_membership" "test_org_membership" {
    username = "%s"
    role = "member"
  }
`, username)
}

func testAccGithubMembershipConfigDowngradable(username string) string {
	return fmt.Sprintf(`
  resource "github_membership" "test_org_membership" {
    username = "%s"
    role = "admin"
    downgrade_on_destroy = %t
  }
`, username, true)
}

func testAccGithubMembershipTheSame(orig, other *github.Membership) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if orig.GetURL() != other.GetURL() {
			return errors.New("users are different")
		}

		return nil
	}
}

// Helper functions are defined in resource_github_membership.go
