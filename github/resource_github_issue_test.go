package github

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGithubIssueResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	title := "Test issue title"
	body := "Test issue body"
	updatedTitle := "Updated test issue title"
	updatedBody := "Updated test issue body"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubIssueConfig_basic(randomID, title, body),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue.test", "title", title),
					resource.TestCheckResourceAttr("github_issue.test", "body", body),
					resource.TestCheckResourceAttr("github_issue.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttrSet("github_issue.test", "number"),
					resource.TestCheckResourceAttrSet("github_issue.test", "issue_id"),
					resource.TestCheckResourceAttrSet("github_issue.test", "id"),
					resource.TestCheckResourceAttrSet("github_issue.test", "etag"),
				),
			},
			{
				Config: testAccGithubIssueConfig_basic(randomID, updatedTitle, updatedBody),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue.test", "title", updatedTitle),
					resource.TestCheckResourceAttr("github_issue.test", "body", updatedBody),
					resource.TestCheckResourceAttr("github_issue.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttrSet("github_issue.test", "number"),
					resource.TestCheckResourceAttrSet("github_issue.test", "issue_id"),
					resource.TestCheckResourceAttrSet("github_issue.test", "id"),
					resource.TestCheckResourceAttrSet("github_issue.test", "etag"),
				),
			},
		},
	})
}

func TestAccGithubIssueResource_withLabelsAndAssignees(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	title := "Test issue with labels and assignees"
	body := "Test issue body with labels and assignees"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubIssueConfig_withLabelsAndAssignees(randomID, title, body),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue.test", "title", title),
					resource.TestCheckResourceAttr("github_issue.test", "body", body),
					resource.TestCheckResourceAttr("github_issue.test", "labels.#", "2"),
					resource.TestCheckTypeSetElemAttr("github_issue.test", "labels.*", "bug"),
					resource.TestCheckTypeSetElemAttr("github_issue.test", "labels.*", "enhancement"),
					resource.TestCheckResourceAttr("github_issue.test", "assignees.#", "1"),
					resource.TestCheckResourceAttrSet("github_issue.test", "number"),
					resource.TestCheckResourceAttrSet("github_issue.test", "issue_id"),
					resource.TestCheckResourceAttrSet("github_issue.test", "id"),
				),
			},
			{
				Config: testAccGithubIssueConfig_withUpdatedLabels(randomID, title, body),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue.test", "title", title),
					resource.TestCheckResourceAttr("github_issue.test", "body", body),
					resource.TestCheckResourceAttr("github_issue.test", "labels.#", "1"),
					resource.TestCheckTypeSetElemAttr("github_issue.test", "labels.*", "documentation"),
					resource.TestCheckResourceAttr("github_issue.test", "assignees.#", "1"),
				),
			},
		},
	})
}

func TestAccGithubIssueResource_withMilestone(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	title := "Test issue with milestone"
	body := "Test issue body with milestone"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubIssueConfig_withMilestone(randomID, title, body),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue.test", "title", title),
					resource.TestCheckResourceAttr("github_issue.test", "body", body),
					resource.TestCheckResourceAttrSet("github_issue.test", "milestone_number"),
					resource.TestCheckResourceAttrSet("github_issue.test", "number"),
					resource.TestCheckResourceAttrSet("github_issue.test", "issue_id"),
					resource.TestCheckResourceAttrSet("github_issue.test", "id"),
					testAccCheckMilestoneMatch,
				),
			},
		},
	})
}

func TestAccGithubIssueResource_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	title := "Test issue for import"
	body := "Test issue body for import"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubIssueConfig_basic(randomID, title, body),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue.test", "title", title),
					resource.TestCheckResourceAttr("github_issue.test", "body", body),
				),
			},
			{
				ResourceName:      "github_issue.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGithubIssueResource_disappears(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	title := "Test issue disappears"
	body := "Test issue body disappears"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubIssueConfig_basic(randomID, title, body),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue.test", "title", title),
					resource.TestCheckResourceAttr("github_issue.test", "body", body),
				),
			},
			{
				Config: testAccGithubIssueConfig_basic(randomID, title, body),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("github_issue.test", plancheck.ResourceActionNoop),
					},
				},
			},
		},
	})
}

func TestAccGithubIssueResource_minimalConfig(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	title := "Minimal test issue"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubIssueConfig_minimal(randomID, title),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue.test", "title", title),
					resource.TestCheckResourceAttr("github_issue.test", "body", ""),
					resource.TestCheckResourceAttr("github_issue.test", "labels.#", "0"),
					resource.TestCheckResourceAttr("github_issue.test", "assignees.#", "0"),
					resource.TestCheckResourceAttrSet("github_issue.test", "number"),
					resource.TestCheckResourceAttrSet("github_issue.test", "issue_id"),
					resource.TestCheckResourceAttrSet("github_issue.test", "id"),
				),
			},
		},
	})
}

// Test helper functions

func testAccCheckMilestoneMatch(state *terraform.State) error {
	issue := state.RootModule().Resources["github_issue.test"].Primary
	issueMilestone := issue.Attributes["milestone_number"]

	milestone := state.RootModule().Resources["github_repository_milestone.test"].Primary
	milestoneNumber := milestone.Attributes["number"]

	if issueMilestone != milestoneNumber {
		return fmt.Errorf("issue milestone number %s does not match repository milestone number %s",
			issueMilestone, milestoneNumber)
	}
	return nil
}

// Test configuration functions

func testAccGithubIssueConfig_basic(randomID, title, body string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Test repository for github_issue resource"
  auto_init   = true
  has_issues  = true
}

resource "github_issue" "test" {
  repository = github_repository.test.name
  title      = "%s"
  body       = "%s"
}
`, randomID, title, body)
}

func testAccGithubIssueConfig_withLabelsAndAssignees(randomID, title, body string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Test repository for github_issue resource"
  auto_init   = true
  has_issues  = true
}

resource "github_issue" "test" {
  repository = github_repository.test.name
  title      = "%s"
  body       = "%s"
  labels     = ["bug", "enhancement"]
  assignees  = ["%s"]
}
`, randomID, title, body, testOwner())
}

func testAccGithubIssueConfig_withUpdatedLabels(randomID, title, body string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Test repository for github_issue resource"
  auto_init   = true
  has_issues  = true
}

resource "github_issue" "test" {
  repository = github_repository.test.name
  title      = "%s"
  body       = "%s"
  labels     = ["documentation"]
  assignees  = ["%s"]
}
`, randomID, title, body, testOwner())
}

func testAccGithubIssueConfig_withMilestone(randomID, title, body string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Test repository for github_issue resource"
  auto_init   = true
  has_issues  = true
}

resource "github_repository_milestone" "test" {
  owner       = split("/", github_repository.test.full_name)[0]
  repository  = github_repository.test.name
  title       = "v1.0.0"
  description = "General Availability"
  due_date    = "2024-12-31"
  state       = "open"
}

resource "github_issue" "test" {
  repository       = github_repository.test.name
  title            = "%s"
  body             = "%s"
  milestone_number = github_repository_milestone.test.number
}
`, randomID, title, body)
}

func testAccGithubIssueConfig_minimal(randomID, title string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Test repository for github_issue resource"
  auto_init   = true
  has_issues  = true
}

resource "github_issue" "test" {
  repository = github_repository.test.name
  title      = "%s"
}
`, randomID, title)
}

// Helper function to get the test owner (similar to testOwnerFunc() in SDKv2 tests)
func testOwner() string {
	owner := os.Getenv("GITHUB_OWNER")
	if owner == "" {
		owner = os.Getenv("GITHUB_TEST_OWNER")
	}
	return owner
}
