package framework

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubTeamRepositoryResource(t *testing.T) {

	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("manages team permissions to a repository", func(t *testing.T) {

		config := fmt.Sprintf(`
			resource "github_team" "test" {
				name        = "tf-acc-test-team-repo-%s"
				description = "test"
			}

			resource "github_repository" "test" {
				name = "tf-acc-test-%[1]s"
			}

			resource "github_team_repository" "test" {
				team_id    = github_team.test.id
				repository = github_repository.test.name
				permission = "pull"
			}
		`, randomID)

		checks := map[string]resource.TestCheckFunc{
			"pull": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_team_repository.test", "permission",
					"pull",
				),
			),
			"triage": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_team_repository.test", "permission",
					"triage",
				),
			),
			"push": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_team_repository.test", "permission",
					"push",
				),
			),
			"maintain": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_team_repository.test", "permission",
					"maintain",
				),
			),
			"admin": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_team_repository.test", "permission",
					"admin",
				),
			),
		}

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  checks["pull"],
					},
					{
						Config: strings.Replace(config,
							`permission = "pull"`,
							`permission = "triage"`, 1),
						Check: checks["triage"],
					},
					{
						Config: strings.Replace(config,
							`permission = "pull"`,
							`permission = "push"`, 1),
						Check: checks["push"],
					},
					{
						Config: strings.Replace(config,
							`permission = "pull"`,
							`permission = "maintain"`, 1),
						Check: checks["maintain"],
					},
					{
						Config: strings.Replace(config,
							`permission = "pull"`,
							`permission = "admin"`, 1),
						Check: checks["admin"],
					},
				},
			})
		}

		t.Run("with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})

		t.Run("with an individual account", func(t *testing.T) {
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, "organization")
		})

	})

	t.Run("accepts both team slug and team ID for `team_id`", func(t *testing.T) {

		config := fmt.Sprintf(`
			resource "github_team" "test" {
				name        = "tf-acc-test-team-repo-%s"
				description = "test"
			}

			resource "github_repository" "test" {
				name = "tf-acc-test-%[1]s"
			}

			resource "github_team_repository" "test" {
				team_id    = github_team.test.slug
				repository = github_repository.test.name
				permission = "pull"
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet("github_team_repository.test", "team_id"),
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
					{
						Config: strings.Replace(config,
							`github_team.test.slug`,
							`github_team.test.id`, 1),
						Check: check,
					},
				},
			})
		}

		t.Run("with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})

		t.Run("with an individual account", func(t *testing.T) {
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, "organization")
		})
	})

	t.Run("can be imported", func(t *testing.T) {

		config := fmt.Sprintf(`
			resource "github_team" "test" {
				name        = "tf-acc-test-team-repo-%s"
				description = "test"
			}

			resource "github_repository" "test" {
				name = "tf-acc-test-%[1]s"
			}

			resource "github_team_repository" "test" {
				team_id    = github_team.test.id
				repository = github_repository.test.name
				permission = "push"
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("github_team_repository.test", "permission", "push"),
			resource.TestCheckResourceAttrSet("github_team_repository.test", "team_id"),
			resource.TestCheckResourceAttrSet("github_team_repository.test", "repository"),
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
					{
						ResourceName:      "github_team_repository.test",
						ImportState:       true,
						ImportStateVerify: true,
						ImportStateVerifyIgnore: []string{"etag"},
					},
				},
			})
		}

		t.Run("with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})

		t.Run("with an individual account", func(t *testing.T) {
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, "organization")
		})
	})
}