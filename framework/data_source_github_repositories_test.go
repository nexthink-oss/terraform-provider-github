package framework

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubRepositoriesDataSource(t *testing.T) {

	// FIXME: Find a way to reduce amount of `GET /search/repositories`
	// t.Skip("Skipping due to API rate limits exceeding")

	t.Run("queries a list of repositories without error", func(t *testing.T) {

		config := fmt.Sprintf(`
			data "github_repositories" "test" {
				query = "org:%s"
			}
		`, testOrganization())

		check := resource.ComposeTestCheckFunc(
			resource.TestMatchResourceAttr(
				"data.github_repositories.test", "full_names.0",
				regexp.MustCompile(`^`+testOrganization()),
			),
			resource.TestCheckResourceAttrSet(
				"data.github_repositories.test", "names.0",
			),
			resource.TestCheckNoResourceAttr(
				"data.github_repositories.test", "repo_ids.0",
			),
			resource.TestCheckResourceAttr(
				"data.github_repositories.test", "sort",
				"updated",
			),
			resource.TestCheckResourceAttr(
				"data.github_repositories.test", "include_repo_id",
				"false",
			),
			resource.TestCheckResourceAttr(
				"data.github_repositories.test", "results_per_page",
				"100",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
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
			testCase(t, anonymous)
		})

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})

	})

	t.Run("queries a list of repositories with repo_ids and results_per_page without error", func(t *testing.T) {

		config := fmt.Sprintf(`
			data "github_repositories" "test" {
				query = "org:%s"
				include_repo_id = true
				results_per_page = 20
			}
		`, testOrganization())

		check := resource.ComposeTestCheckFunc(
			resource.TestMatchResourceAttr(
				"data.github_repositories.test", "full_names.0",
				regexp.MustCompile(`^`+testOrganization()),
			),
			resource.TestCheckResourceAttrSet(
				"data.github_repositories.test", "names.0",
			),
			resource.TestCheckResourceAttrSet(
				"data.github_repositories.test", "repo_ids.0",
			),
			resource.TestCheckResourceAttr(
				"data.github_repositories.test", "sort",
				"updated",
			),
			resource.TestCheckResourceAttr(
				"data.github_repositories.test", "include_repo_id",
				"true",
			),
			resource.TestCheckResourceAttr(
				"data.github_repositories.test", "results_per_page",
				"20",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
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
			testCase(t, anonymous)
		})

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})

	})

	t.Run("queries a list of repositories with different sort option", func(t *testing.T) {

		config := fmt.Sprintf(`
			data "github_repositories" "test" {
				query = "org:%s"
				sort = "stars"
			}
		`, testOrganization())

		check := resource.ComposeTestCheckFunc(
			resource.TestMatchResourceAttr(
				"data.github_repositories.test", "full_names.0",
				regexp.MustCompile(`^`+testOrganization()),
			),
			resource.TestCheckResourceAttrSet(
				"data.github_repositories.test", "names.0",
			),
			resource.TestCheckResourceAttr(
				"data.github_repositories.test", "sort",
				"stars",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
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
			testCase(t, anonymous)
		})

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})

	})

	t.Run("returns an empty list given an invalid query", func(t *testing.T) {

		// FIXME: Find a way to reduce amount of `GET /search/repositories`
		// t.Skip("Skipping due to API rate limits exceeding")

		config := `
			data "github_repositories" "test" {
				query = "klsafj_23434_doesnt_exist"
			}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"data.github_repositories.test", "full_names.#",
				"0",
			),
			resource.TestCheckResourceAttr(
				"data.github_repositories.test", "names.#",
				"0",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
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
			testCase(t, anonymous)
		})

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})

	})
}
