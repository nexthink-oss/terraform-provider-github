package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGithubBranchProtectionResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubBranchProtectionConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_branch_protection.test", "pattern", "main"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "allows_deletions", "false"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "allows_force_pushes", "false"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "enforce_admins", "false"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "require_signed_commits", "false"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "required_linear_history", "false"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "require_conversation_resolution", "false"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "lock_branch", "false"),
					resource.TestCheckResourceAttrSet("github_branch_protection.test", "id"),
				),
			},
		},
	})
}

func TestAccGithubBranchProtectionResource_requireStatusChecks(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubBranchProtectionConfig_requireStatusChecks(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_branch_protection.test", "pattern", "main"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "required_status_checks.#", "1"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "required_status_checks.0.strict", "true"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "required_status_checks.0.contexts.#", "2"),
				),
			},
		},
	})
}

func TestAccGithubBranchProtectionResource_requirePullRequestReviews(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubBranchProtectionConfig_requirePullRequestReviews(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_branch_protection.test", "pattern", "main"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "required_pull_request_reviews.#", "1"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "required_pull_request_reviews.0.required_approving_review_count", "2"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "required_pull_request_reviews.0.require_code_owner_reviews", "true"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "required_pull_request_reviews.0.dismiss_stale_reviews", "true"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "required_pull_request_reviews.0.require_last_push_approval", "false"),
				),
			},
		},
	})
}

func TestAccGithubBranchProtectionResource_restrictPushes(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubBranchProtectionConfig_restrictPushes(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_branch_protection.test", "pattern", "main"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "restrict_pushes.#", "1"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "restrict_pushes.0.blocks_creations", "true"),
					resource.TestCheckResourceAttr("github_branch_protection.test", "restrict_pushes.0.push_allowances.#", "0"),
				),
			},
		},
	})
}

func TestAccGithubBranchProtectionResource_importBasic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubBranchProtectionConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_branch_protection.test", "pattern", "main"),
					resource.TestCheckResourceAttrSet("github_branch_protection.test", "id"),
				),
			},
			{
				ResourceName:      "github_branch_protection.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccGithubBranchProtectionImportStateIdFunc(),
			},
		},
	})
}

// Configuration functions

func testAccGithubBranchProtectionConfig_basic(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name      = "tf-acc-test-branch-protection-%s"
  auto_init = true
}

resource "github_branch_protection" "test" {
  repository_id = github_repository.test.node_id
  pattern       = "main"
}
`, randomID)
}

func testAccGithubBranchProtectionConfig_requireStatusChecks(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name      = "tf-acc-test-branch-protection-%s"
  auto_init = true
}

resource "github_branch_protection" "test" {
  repository_id = github_repository.test.node_id
  pattern       = "main"

  required_status_checks {
    strict   = true
    contexts = ["continuous-integration/travis-ci", "continuous-integration/appveyor"]
  }
}
`, randomID)
}

func testAccGithubBranchProtectionConfig_requirePullRequestReviews(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name      = "tf-acc-test-branch-protection-%s"
  auto_init = true
}

resource "github_branch_protection" "test" {
  repository_id = github_repository.test.node_id
  pattern       = "main"

  required_pull_request_reviews {
    required_approving_review_count = 2
    require_code_owner_reviews      = true
    dismiss_stale_reviews           = true
    require_last_push_approval      = false
  }
}
`, randomID)
}

func testAccGithubBranchProtectionConfig_restrictPushes(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name      = "tf-acc-test-branch-protection-%s"
  auto_init = true
}

resource "github_branch_protection" "test" {
  repository_id = github_repository.test.node_id
  pattern       = "main"

  restrict_pushes {
    blocks_creations = true
  }
}
`, randomID)
}

// Helper functions

func testAccGithubBranchProtectionImportStateIdFunc() resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources["github_branch_protection.test"]
		if !ok {
			return "", fmt.Errorf("not found: github_branch_protection.test")
		}

		repoName := s.RootModule().Resources["github_repository.test"].Primary.Attributes["name"]
		pattern := rs.Primary.Attributes["pattern"]

		return fmt.Sprintf("%s:%s", repoName, pattern), nil
	}
}
