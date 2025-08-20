package framework

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubAppInstallationRepository(t *testing.T) {

	const APP_INSTALLATION_ID = "APP_INSTALLATION_ID"
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	installationID, exists := os.LookupEnv(APP_INSTALLATION_ID)

	t.Run("installs an app to a repository", func(t *testing.T) {

		if !exists {
			t.Skipf("%s environment variable is missing", APP_INSTALLATION_ID)
		}

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_app_installation_repository" "test" {
				# The installation id of the app (in the organization).
				installation_id    = "%s"
				repository         = github_repository.test.name
			}
		`, randomID, installationID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet(
				"github_app_installation_repository.test", "installation_id",
			),
			resource.TestCheckResourceAttrSet(
				"github_app_installation_repository.test", "repo_id",
			),
			resource.TestCheckResourceAttrSet(
				"github_app_installation_repository.test", "id",
			),
			resource.TestCheckResourceAttr(
				"github_app_installation_repository.test", "installation_id", installationID,
			),
			resource.TestCheckResourceAttr(
				"github_app_installation_repository.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID),
			),
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("imports app installation repository", func(t *testing.T) {
		if !exists {
			t.Skipf("%s environment variable is missing", APP_INSTALLATION_ID)
		}

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_app_installation_repository" "test" {
				installation_id    = "%s"
				repository         = github_repository.test.name
			}
		`, randomID, installationID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
					},
					{
						ResourceName:      "github_app_installation_repository.test",
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
