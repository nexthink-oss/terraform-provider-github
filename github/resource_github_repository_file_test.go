package github

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubRepositoryFileResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryFileConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_file.test", "content", "bar"),
					resource.TestCheckResourceAttr("github_repository_file.test", "sha", "ba0e162e1c47469e3fe4b393a8bf8c569f302116"),
					resource.TestCheckResourceAttr("github_repository_file.test", "ref", "main"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_author"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_email"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_message"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_sha"),
					resource.TestCheckResourceAttr("github_repository_file.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_repository_file.test", "file", "test"),
					resource.TestCheckResourceAttr("github_repository_file.test", "branch", "main"),
					resource.TestCheckResourceAttr("github_repository_file.test", "overwrite_on_create", "false"),
					resource.TestCheckResourceAttr("github_repository_file.test", "autocreate_branch", "false"),
				),
			},
			{
				Config: testAccGithubRepositoryFileConfig_updated(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_file.test", "content", "updated content"),
					resource.TestCheckResourceAttr("github_repository_file.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_repository_file.test", "file", "test"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_sha"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryFileResource_overwriteOnCreate(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccGithubRepositoryFileConfig_overwriteDisabled(randomID),
				ExpectError: regexp.MustCompile(`Refusing to overwrite existing file`),
			},
			{
				Config: testAccGithubRepositoryFileConfig_overwriteEnabled(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_file.test", "content", "overwritten"),
					resource.TestCheckResourceAttr("github_repository_file.test", "sha", "67c1a95c2d9bb138aefeaebb319cca82e531736b"),
					resource.TestCheckResourceAttr("github_repository_file.test", "overwrite_on_create", "true"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_author"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_email"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_message"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_sha"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryFileResource_defaultBranch(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryFileConfig_defaultBranch(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_file.test", "content", "bar"),
					resource.TestCheckResourceAttr("github_repository_file.test", "sha", "ba0e162e1c47469e3fe4b393a8bf8c569f302116"),
					resource.TestCheckResourceAttr("github_repository_file.test", "ref", "test"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_author"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_email"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_message"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_sha"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryFileResource_autocreateBranch(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      testAccGithubRepositoryFileConfig_autocreateBranchDisabled(randomID),
				ExpectError: regexp.MustCompile(`branch .* not found in repository`),
			},
			{
				Config: testAccGithubRepositoryFileConfig_autocreateBranchEnabled(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_file.test", "content", "bar"),
					resource.TestCheckResourceAttr("github_repository_file.test", "sha", "ba0e162e1c47469e3fe4b393a8bf8c569f302116"),
					resource.TestCheckResourceAttr("github_repository_file.test", "ref", "does/not/exist"),
					resource.TestCheckResourceAttr("github_repository_file.test", "autocreate_branch", "true"),
					resource.TestCheckResourceAttr("github_repository_file.test", "autocreate_branch_source_branch", "main"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "autocreate_branch_source_sha"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_author"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_email"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_message"),
					resource.TestCheckResourceAttrSet("github_repository_file.test", "commit_sha"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryFileResource_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryFileConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_file.test", "content", "bar"),
					resource.TestCheckResourceAttr("github_repository_file.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_repository_file.test", "file", "test"),
				),
			},
			{
				ResourceName:            "github_repository_file.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           fmt.Sprintf("tf-acc-test-%s/test", randomID),
				ImportStateVerifyIgnore: []string{"overwrite_on_create"},
			},
		},
	})
}

func TestAccGithubRepositoryFileResource_importWithBranch(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryFileConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_file.test", "content", "bar"),
					resource.TestCheckResourceAttr("github_repository_file.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_repository_file.test", "file", "test"),
					resource.TestCheckResourceAttr("github_repository_file.test", "branch", "main"),
				),
			},
			{
				ResourceName:            "github_repository_file.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           fmt.Sprintf("tf-acc-test-%s/test:main", randomID),
				ImportStateVerifyIgnore: []string{"overwrite_on_create"},
			},
		},
	})
}

func TestAccGithubRepositoryFileResource_migrationCompatibility(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t, individual) },
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"github": {Source: "integrations/github", VersionConstraint: "~> 6.0"},
				},
				Config: testAccGithubRepositoryFileConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_file.test", "content", "bar"),
					resource.TestCheckResourceAttr("github_repository_file.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_repository_file.test", "file", "test"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Config:                   testAccGithubRepositoryFileConfig_basic(randomID),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccGithubRepositoryFileResource_noPlan(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryFileConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_file.test", "content", "bar"),
				),
			},
			{
				Config: testAccGithubRepositoryFileConfig_basic(randomID),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// Configuration functions

func testAccGithubRepositoryFileConfig_basic(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name                 = "tf-acc-test-%s"
  auto_init            = true
  vulnerability_alerts = true
}

resource "github_repository_file" "test" {
  repository     = github_repository.test.name
  branch         = "main"
  file           = "test"
  content        = "bar"
  commit_message = "Managed by Terraform"
  commit_author  = "Terraform User"
  commit_email   = "terraform@example.com"
}
`, randomID)
}

func testAccGithubRepositoryFileConfig_updated(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name                 = "tf-acc-test-%s"
  auto_init            = true
  vulnerability_alerts = true
}

resource "github_repository_file" "test" {
  repository     = github_repository.test.name
  branch         = "main"
  file           = "test"
  content        = "updated content"
  commit_message = "Updated by Terraform"
  commit_author  = "Terraform User"
  commit_email   = "terraform@example.com"
}
`, randomID)
}

func testAccGithubRepositoryFileConfig_overwriteDisabled(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name                 = "tf-acc-test-%s"
  auto_init            = true
  vulnerability_alerts = true
}

resource "github_repository_file" "test" {
  repository          = github_repository.test.name
  branch              = "main"
  file                = "README.md"
  content             = "overwritten"
  overwrite_on_create = false
  commit_message      = "Managed by Terraform"
  commit_author       = "Terraform User"
  commit_email        = "terraform@example.com"
}
`, randomID)
}

func testAccGithubRepositoryFileConfig_overwriteEnabled(randomID string) string {
	return strings.Replace(testAccGithubRepositoryFileConfig_overwriteDisabled(randomID),
		"overwrite_on_create = false",
		"overwrite_on_create = true", 1)
}

func testAccGithubRepositoryFileConfig_defaultBranch(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name                 = "tf-acc-test-%s"
  auto_init            = true
  vulnerability_alerts = true
}

resource "github_branch" "test" {
  repository = github_repository.test.name
  branch     = "test"
}

resource "github_branch_default" "default" {
  repository = github_repository.test.name
  branch     = github_branch.test.branch
}

resource "github_repository_file" "test" {
  depends_on = [github_branch_default.default]

  repository     = github_repository.test.name
  file           = "test"
  content        = "bar"
  commit_message = "Managed by Terraform"
  commit_author  = "Terraform User"
  commit_email   = "terraform@example.com"
}
`, randomID)
}

func testAccGithubRepositoryFileConfig_autocreateBranchDisabled(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name                 = "tf-acc-test-%s"
  auto_init            = true
  vulnerability_alerts = true
}

resource "github_repository_file" "test" {
  repository        = github_repository.test.name
  branch            = "does/not/exist"
  file              = "test"
  content           = "bar"
  commit_message    = "Managed by Terraform"
  commit_author     = "Terraform User"
  commit_email      = "terraform@example.com"
  autocreate_branch = false
}
`, randomID)
}

func testAccGithubRepositoryFileConfig_autocreateBranchEnabled(randomID string) string {
	return strings.Replace(testAccGithubRepositoryFileConfig_autocreateBranchDisabled(randomID),
		"autocreate_branch = false",
		"autocreate_branch = true", 1)
}
