package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGithubReleaseResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	randomVersion := fmt.Sprintf("v1.0.%d", acctest.RandIntRange(0, 9999))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubReleaseConfig_basic(randomID, randomVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_release.test", "tag_name", randomVersion),
					resource.TestCheckResourceAttr("github_release.test", "target_commitish", "main"),
					resource.TestCheckResourceAttr("github_release.test", "name", ""),
					resource.TestCheckResourceAttr("github_release.test", "body", ""),
					resource.TestCheckResourceAttr("github_release.test", "draft", "true"),
					resource.TestCheckResourceAttr("github_release.test", "prerelease", "true"),
					resource.TestCheckResourceAttr("github_release.test", "generate_release_notes", "false"),
					resource.TestCheckResourceAttr("github_release.test", "discussion_category_name", ""),
					resource.TestCheckResourceAttrSet("github_release.test", "id"),
					resource.TestCheckResourceAttrSet("github_release.test", "release_id"),
					resource.TestCheckResourceAttrSet("github_release.test", "node_id"),
					resource.TestCheckResourceAttrSet("github_release.test", "created_at"),
					resource.TestCheckResourceAttrSet("github_release.test", "url"),
					resource.TestCheckResourceAttrSet("github_release.test", "html_url"),
					resource.TestCheckResourceAttrSet("github_release.test", "assets_url"),
					resource.TestCheckResourceAttrSet("github_release.test", "upload_url"),
					resource.TestCheckResourceAttrSet("github_release.test", "zipball_url"),
					resource.TestCheckResourceAttrSet("github_release.test", "tarball_url"),
				),
			},
		},
	})
}

func TestAccGithubReleaseResource_onBranch(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	randomVersion := fmt.Sprintf("v1.0.%d", acctest.RandIntRange(0, 9999))
	testBranchName := "test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubReleaseConfig_onBranch(randomID, randomVersion, testBranchName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_release.test", "tag_name", randomVersion),
					resource.TestCheckResourceAttr("github_release.test", "target_commitish", testBranchName),
					resource.TestCheckResourceAttr("github_release.test", "name", ""),
					resource.TestCheckResourceAttr("github_release.test", "body", ""),
					resource.TestCheckResourceAttr("github_release.test", "draft", "false"),
					resource.TestCheckResourceAttr("github_release.test", "prerelease", "false"),
					resource.TestCheckResourceAttr("github_release.test", "generate_release_notes", "false"),
					resource.TestCheckResourceAttr("github_release.test", "discussion_category_name", ""),
					resource.TestCheckResourceAttrSet("github_release.test", "id"),
					resource.TestCheckResourceAttrSet("github_release.test", "release_id"),
					resource.TestCheckResourceAttrSet("github_release.test", "node_id"),
					resource.TestCheckResourceAttrSet("github_release.test", "created_at"),
					resource.TestCheckResourceAttrSet("github_release.test", "published_at"),
					resource.TestCheckResourceAttrSet("github_release.test", "url"),
					resource.TestCheckResourceAttrSet("github_release.test", "html_url"),
				),
			},
		},
	})
}

func TestAccGithubReleaseResource_withNameAndBody(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	randomVersion := fmt.Sprintf("v1.0.%d", acctest.RandIntRange(0, 9999))
	releaseName := "Test Release"
	releaseBody := "This is a test release body"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubReleaseConfig_withNameAndBody(randomID, randomVersion, releaseName, releaseBody),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_release.test", "tag_name", randomVersion),
					resource.TestCheckResourceAttr("github_release.test", "name", releaseName),
					resource.TestCheckResourceAttr("github_release.test", "body", releaseBody),
					resource.TestCheckResourceAttr("github_release.test", "draft", "true"),
					resource.TestCheckResourceAttr("github_release.test", "prerelease", "true"),
					resource.TestCheckResourceAttrSet("github_release.test", "id"),
				),
			},
		},
	})
}

func TestAccGithubReleaseResource_update(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	randomVersion := fmt.Sprintf("v1.0.%d", acctest.RandIntRange(0, 9999))

	initialName := "Initial Release"
	initialBody := "Initial release body"
	updatedName := "Updated Release"
	updatedBody := "Updated release body"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubReleaseConfig_withNameAndBody(randomID, randomVersion, initialName, initialBody),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_release.test", "name", initialName),
					resource.TestCheckResourceAttr("github_release.test", "body", initialBody),
					resource.TestCheckResourceAttr("github_release.test", "prerelease", "true"),
				),
			},
			{
				Config: testAccGithubReleaseConfig_updated(randomID, randomVersion, updatedName, updatedBody),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_release.test", "name", updatedName),
					resource.TestCheckResourceAttr("github_release.test", "body", updatedBody),
					resource.TestCheckResourceAttr("github_release.test", "prerelease", "false"),
				),
			},
		},
	})
}

func TestAccGithubReleaseResource_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	randomVersion := fmt.Sprintf("v1.0.%d", acctest.RandIntRange(0, 9999))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubReleaseConfig_basic(randomID, randomVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_release.test", "tag_name", randomVersion),
				),
			},
			{
				ResourceName:      "github_release.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccGithubReleaseImportStateIdFunc("github_release.test"),
			},
		},
	})
}

func TestAccGithubReleaseResource_generateReleaseNotes(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	randomVersion := fmt.Sprintf("v1.0.%d", acctest.RandIntRange(0, 9999))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubReleaseConfig_generateReleaseNotes(randomID, randomVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_release.test", "tag_name", randomVersion),
					resource.TestCheckResourceAttr("github_release.test", "generate_release_notes", "true"),
					resource.TestCheckResourceAttr("github_release.test", "draft", "false"),
					resource.TestCheckResourceAttr("github_release.test", "prerelease", "false"),
					resource.TestCheckResourceAttrSet("github_release.test", "id"),
					resource.TestCheckResourceAttrSet("github_release.test", "body"), // Should be generated
				),
			},
		},
	})
}

func TestAccGithubReleaseResource_migrationCompatibility(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	randomVersion := fmt.Sprintf("v1.0.%d", acctest.RandIntRange(0, 9999))

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t, individual) },
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"github": {Source: "integrations/github", VersionConstraint: "~> 6.0"},
				},
				Config: testAccGithubReleaseConfig_basic(randomID, randomVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_release.test", "tag_name", randomVersion),
					resource.TestCheckResourceAttr("github_release.test", "target_commitish", "main"),
					resource.TestCheckResourceAttr("github_release.test", "draft", "true"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Config:                   testAccGithubReleaseConfig_basic(randomID, randomVersion),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func TestAccGithubReleaseResource_noPlan(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	randomVersion := fmt.Sprintf("v1.0.%d", acctest.RandIntRange(0, 9999))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubReleaseConfig_basic(randomID, randomVersion),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_release.test", "tag_name", randomVersion),
				),
			},
			{
				Config: testAccGithubReleaseConfig_basic(randomID, randomVersion),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// Helper functions

func testAccGithubReleaseImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("not found: %s", resourceName)
		}

		repository := rs.Primary.Attributes["repository"]
		releaseID := rs.Primary.ID
		return fmt.Sprintf("%s:%s", repository, releaseID), nil
	}
}

// Test configuration functions

func testAccGithubReleaseConfig_basic(randomID, version string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name      = "tf-acc-test-%s"
  auto_init = true
}

resource "github_release" "test" {
  repository = github_repository.test.name
  tag_name   = "%s"
}
`, randomID, version)
}

func testAccGithubReleaseConfig_onBranch(randomID, version, branchName string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name      = "tf-acc-test-%s"
  auto_init = true
}

resource "github_branch" "test" {
  repository    = github_repository.test.name
  branch        = "%s"
  source_branch = github_repository.test.default_branch
}

resource "github_release" "test" {
  repository       = github_repository.test.name
  tag_name         = "%s"
  target_commitish = github_branch.test.branch
  draft            = false
  prerelease       = false
}
`, randomID, branchName, version)
}

func testAccGithubReleaseConfig_withNameAndBody(randomID, version, name, body string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name      = "tf-acc-test-%s"
  auto_init = true
}

resource "github_release" "test" {
  repository = github_repository.test.name
  tag_name   = "%s"
  name       = "%s"
  body       = "%s"
}
`, randomID, version, name, body)
}

func testAccGithubReleaseConfig_updated(randomID, version, name, body string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name      = "tf-acc-test-%s"
  auto_init = true
}

resource "github_release" "test" {
  repository = github_repository.test.name
  tag_name   = "%s"
  name       = "%s"
  body       = "%s"
  prerelease = false
}
`, randomID, version, name, body)
}

func testAccGithubReleaseConfig_generateReleaseNotes(randomID, version string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name      = "tf-acc-test-%s"
  auto_init = true
}

resource "github_release" "test" {
  repository           = github_repository.test.name
  tag_name             = "%s"
  generate_release_notes = true
  draft                = false
  prerelease           = false
}
`, randomID, version)
}
