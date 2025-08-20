package github

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestGithubRepositoryTopicsResource tests the basic resource creation
func TestGithubRepositoryTopicsResource(t *testing.T) {
	resource := NewGithubRepositoryTopicsResource()
	if resource == nil {
		t.Error("Resource should not be nil")
	}
}

func TestAccGithubRepositoryTopics_Framework(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("create repository topics and import them", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_topics" "test" {
				repository    = github_repository.test.name
				topics        = ["test", "test-2"]
			}
		`, randomID)

		const resourceName = "github_repository_topics.test"

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "topics.#", "2"),
			resource.TestCheckTypeSetElemAttr(resourceName, "topics.*", "test"),
			resource.TestCheckTypeSetElemAttr(resourceName, "topics.*", "test-2"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
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
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("create repository topics and update them", func(t *testing.T) {
		configBefore := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_topics" "test" {
				repository    = github_repository.test.name
				topics        = ["test", "test-2"]
			}
		`, randomID)

		configAfter := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_topics" "test" {
				repository    = github_repository.test.name
				topics        = ["test", "test-2", "extra-topic"]
			}
		`, randomID)

		const resourceName = "github_repository_topics.test"

		checkBefore := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "topics.#", "2"),
			resource.TestCheckTypeSetElemAttr(resourceName, "topics.*", "test"),
			resource.TestCheckTypeSetElemAttr(resourceName, "topics.*", "test-2"),
		)
		checkAfter := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "topics.#", "3"),
			resource.TestCheckTypeSetElemAttr(resourceName, "topics.*", "test"),
			resource.TestCheckTypeSetElemAttr(resourceName, "topics.*", "test-2"),
			resource.TestCheckTypeSetElemAttr(resourceName, "topics.*", "extra-topic"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: configBefore,
						Check:  checkBefore,
					},
					{
						Config: configAfter,
						Check:  checkAfter,
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

	t.Run("update repository topics by removing topics", func(t *testing.T) {
		configBefore := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_topics" "test" {
				repository    = github_repository.test.name
				topics        = ["test", "test-2", "extra-topic"]
			}
		`, randomID)

		configAfter := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_topics" "test" {
				repository    = github_repository.test.name
				topics        = ["test"]
			}
		`, randomID)

		const resourceName = "github_repository_topics.test"

		checkBefore := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "topics.#", "3"),
			resource.TestCheckTypeSetElemAttr(resourceName, "topics.*", "test"),
			resource.TestCheckTypeSetElemAttr(resourceName, "topics.*", "test-2"),
			resource.TestCheckTypeSetElemAttr(resourceName, "topics.*", "extra-topic"),
		)
		checkAfter := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "topics.#", "1"),
			resource.TestCheckTypeSetElemAttr(resourceName, "topics.*", "test"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: configBefore,
						Check:  checkBefore,
					},
					{
						Config: configAfter,
						Check:  checkAfter,
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

	t.Run("create repository with empty topics set", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_topics" "test" {
				repository    = github_repository.test.name
				topics        = []
			}
		`, randomID)

		const resourceName = "github_repository_topics.test"

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(resourceName, "topics.#", "0"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
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

	t.Run("validate topic name constraints", func(t *testing.T) {
		configInvalidUppercase := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_topics" "test" {
				repository    = github_repository.test.name
				topics        = ["INVALID-UPPERCASE"]
			}
		`, randomID)

		configInvalidHyphenStart := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_topics" "test" {
				repository    = github_repository.test.name
				topics        = ["-invalid-start"]
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      configInvalidUppercase,
						ExpectError: regexp.MustCompile("must include only lowercase alphanumeric characters"),
					},
					{
						Config:      configInvalidHyphenStart,
						ExpectError: regexp.MustCompile("cannot start with a hyphen"),
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
