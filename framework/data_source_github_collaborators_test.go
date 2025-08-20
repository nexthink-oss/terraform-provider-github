package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubCollaboratorsDataSource(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("queries repository collaborators without error", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-collab-%s"
			}

			data "github_collaborators" "test" {
				owner      = "%s"
				repository = github_repository.test.name
			}
		`, randomID, testOwnerFunc())

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrSet("data.github_collaborators.test", "collaborator.#"),
			resource.TestCheckResourceAttr("data.github_collaborators.test", "affiliation", "all"),
			resource.TestCheckResourceAttr("data.github_collaborators.test", "owner", testOwnerFunc()),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

	t.Run("queries repository collaborators with permission filter", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-collab-perm-%s"
			}

			data "github_collaborators" "test" {
				owner      = "%s"
				repository = github_repository.test.name
				permission = "admin"
			}
		`, randomID, testOwnerFunc())

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrSet("data.github_collaborators.test", "collaborator.#"),
			resource.TestCheckResourceAttr("data.github_collaborators.test", "affiliation", "all"),
			resource.TestCheckResourceAttr("data.github_collaborators.test", "permission", "admin"),
			resource.TestCheckResourceAttr("data.github_collaborators.test", "owner", testOwnerFunc()),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

	t.Run("queries repository collaborators with affiliation filter", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-collab-aff-%s"
			}

			data "github_collaborators" "test" {
				owner       = "%s"
				repository  = github_repository.test.name
				affiliation = "direct"
			}
		`, randomID, testOwnerFunc())

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrSet("data.github_collaborators.test", "collaborator.#"),
			resource.TestCheckResourceAttr("data.github_collaborators.test", "affiliation", "direct"),
			resource.TestCheckResourceAttr("data.github_collaborators.test", "owner", testOwnerFunc()),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
}
