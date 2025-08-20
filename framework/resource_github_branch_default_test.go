package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestGithubBranchDefaultResource tests the basic resource creation
func TestGithubBranchDefaultResource(t *testing.T) {
	resource := NewGithubBranchDefaultResource()
	if resource == nil {
		t.Error("Resource should not be nil")
	}
}

func TestAccGithubBranchDefault_Framework(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("creates and manages branch defaults", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_branch_default" "test" {
				repository     = github_repository.test.name
				branch         = "main"
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_branch_default.test", "branch",
				"main",
			),
			resource.TestCheckResourceAttr(
				"github_branch_default.test", "repository",
				fmt.Sprintf("tf-acc-test-%s", randomID),
			),
			resource.TestCheckResourceAttr(
				"github_branch_default.test", "rename",
				"false",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch_default.test", "etag",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("replaces the default_branch of a repository", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}
			
			resource "github_branch" "test" {
				repository = github_repository.test.name
				branch     = "test"
			}
			  
			resource "github_branch_default" "test"{
				repository = github_repository.test.name
				branch     = github_branch.test.branch
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_branch_default.test", "branch",
				"test",
			),
			resource.TestCheckResourceAttr(
				"github_branch_default.test", "rename",
				"false",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch_default.test", "etag",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("replaces the default_branch of a repository without creating a branch resource prior to", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}
			
			resource "github_branch_default" "test"{
				repository = github_repository.test.name
				branch     = "development"
				rename     = true
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_branch_default.test", "branch",
				"development",
			),
			resource.TestCheckResourceAttr(
				"github_branch_default.test", "rename",
				"true",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch_default.test", "etag",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("imports branch default", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_branch_default" "test" {
				repository     = github_repository.test.name
				branch         = "main"
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
					},
					{
						ResourceName:            "github_branch_default.test",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"rename"}, // rename is optional and may not be preserved during import
					},
				},
			})
		}

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("updates branch default", func(t *testing.T) {
		config1 := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_branch" "test" {
				repository = github_repository.test.name
				branch     = "feature-branch"
			}

			resource "github_branch_default" "test" {
				repository     = github_repository.test.name
				branch         = "main"
			}
		`, randomID)

		config2 := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_branch" "test" {
				repository = github_repository.test.name
				branch     = "feature-branch"
			}

			resource "github_branch_default" "test" {
				repository     = github_repository.test.name
				branch         = github_branch.test.branch
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config1,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_branch_default.test", "branch",
								"main",
							),
						),
					},
					{
						Config: config2,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_branch_default.test", "branch",
								"feature-branch",
							),
						),
					},
				},
			})
		}

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
