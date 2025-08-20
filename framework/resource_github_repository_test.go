package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubRepositoryResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository.test", "name", fmt.Sprintf("tf-acc-test-basic-%s", randomID)),
					resource.TestCheckResourceAttr("github_repository.test", "description", fmt.Sprintf("Terraform acceptance test %s", randomID)),
					resource.TestCheckResourceAttr("github_repository.test", "visibility", "private"),
					resource.TestCheckResourceAttr("github_repository.test", "has_issues", "true"),
					resource.TestCheckResourceAttr("github_repository.test", "has_wiki", "true"),
					resource.TestCheckResourceAttr("github_repository.test", "allow_merge_commit", "true"),
					resource.TestCheckResourceAttr("github_repository.test", "allow_squash_merge", "true"),
					resource.TestCheckResourceAttr("github_repository.test", "allow_rebase_merge", "true"),
					resource.TestCheckResourceAttr("github_repository.test", "delete_branch_on_merge", "false"),
					resource.TestCheckResourceAttrSet("github_repository.test", "full_name"),
					resource.TestCheckResourceAttrSet("github_repository.test", "html_url"),
					resource.TestCheckResourceAttrSet("github_repository.test", "ssh_clone_url"),
					resource.TestCheckResourceAttrSet("github_repository.test", "http_clone_url"),
					resource.TestCheckResourceAttrSet("github_repository.test", "git_clone_url"),
					resource.TestCheckResourceAttrSet("github_repository.test", "svn_url"),
					resource.TestCheckResourceAttrSet("github_repository.test", "node_id"),
					resource.TestCheckResourceAttrSet("github_repository.test", "repo_id"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryResource_update(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	initialDescription := fmt.Sprintf("Terraform acceptance test %s", randomID)
	updatedDescription := fmt.Sprintf("Terraform acceptance test %s updated", randomID)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryConfig_withDescription(randomID, initialDescription),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository.test", "description", initialDescription),
					resource.TestCheckResourceAttr("github_repository.test", "has_issues", "false"),
				),
			},
			{
				Config: testAccGithubRepositoryConfig_withDescription(randomID, updatedDescription),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository.test", "description", updatedDescription),
					resource.TestCheckResourceAttr("github_repository.test", "has_issues", "false"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryResource_topics(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryConfig_withTopics(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository.test", "topics.#", "3"),
					resource.TestCheckTypeSetElemAttr("github_repository.test", "topics.*", "terraform"),
					resource.TestCheckTypeSetElemAttr("github_repository.test", "topics.*", "testing"),
					resource.TestCheckTypeSetElemAttr("github_repository.test", "topics.*", "acceptance"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryResource_mergeOptions(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryConfig_withMergeOptions(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository.test", "allow_merge_commit", "false"),
					resource.TestCheckResourceAttr("github_repository.test", "allow_squash_merge", "true"),
					resource.TestCheckResourceAttr("github_repository.test", "allow_rebase_merge", "false"),
					resource.TestCheckResourceAttr("github_repository.test", "allow_auto_merge", "true"),
					resource.TestCheckResourceAttr("github_repository.test", "delete_branch_on_merge", "true"),
					resource.TestCheckResourceAttr("github_repository.test", "squash_merge_commit_title", "PR_TITLE"),
					resource.TestCheckResourceAttr("github_repository.test", "squash_merge_commit_message", "PR_BODY"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryResource_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository.test", "name", fmt.Sprintf("tf-acc-test-basic-%s", randomID)),
				),
			},
			{
				ResourceName:      "github_repository.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"auto_init", // This is only used during creation
				},
			},
		},
	})
}

func TestAccGithubRepositoryResource_pages(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryConfig_withPages(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository.test", "pages.#", "1"),
					resource.TestCheckResourceAttr("github_repository.test", "pages.0.source.#", "1"),
					resource.TestCheckResourceAttr("github_repository.test", "pages.0.source.0.branch", "main"),
					resource.TestCheckResourceAttr("github_repository.test", "pages.0.source.0.path", "/"),
					resource.TestCheckResourceAttr("github_repository.test", "pages.0.build_type", "legacy"),
					resource.TestCheckResourceAttrSet("github_repository.test", "pages.0.url"),
					resource.TestCheckResourceAttrSet("github_repository.test", "pages.0.html_url"),
				),
			},
		},
	})
}

// Test configuration functions

func testAccGithubRepositoryConfig_basic(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-basic-%s"
  description = "Terraform acceptance test %s"
  visibility  = "private"
  has_issues  = true
  has_wiki    = true
}
`, randomID, randomID)
}

func testAccGithubRepositoryConfig_withDescription(randomID, description string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-description-%s"
  description = "%s"
  visibility  = "private"
  has_issues  = false
}
`, randomID, description)
}

func testAccGithubRepositoryConfig_withTopics(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-topics-%s"
  description = "Terraform acceptance test %s"
  visibility  = "private"
  topics      = ["terraform", "testing", "acceptance"]
}
`, randomID, randomID)
}

func testAccGithubRepositoryConfig_withMergeOptions(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name                         = "tf-acc-test-merge-%s"
  description                  = "Terraform acceptance test %s"
  visibility                   = "private"
  allow_merge_commit           = false
  allow_squash_merge           = true
  allow_rebase_merge           = false
  allow_auto_merge             = true
  delete_branch_on_merge       = true
  squash_merge_commit_title    = "PR_TITLE"
  squash_merge_commit_message  = "PR_BODY"
}
`, randomID, randomID)
}

func testAccGithubRepositoryConfig_withPages(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-pages-%s"
  description = "Terraform acceptance test %s"
  visibility  = "public"
  auto_init   = true

  pages {
    source {
      branch = "main"
      path   = "/"
    }
    build_type = "legacy"
  }
}
`, randomID, randomID)
}
