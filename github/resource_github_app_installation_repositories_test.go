package github

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubAppInstallationRepositoriesResource(t *testing.T) {
	const APP_INSTALLATION_ID = "APP_INSTALLATION_ID"
	randomID1 := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	randomID2 := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	installationID, exists := os.LookupEnv(APP_INSTALLATION_ID)

	t.Run("installs an app to multiple repositories", func(t *testing.T) {

		if !exists {
			t.Skipf("%s environment variable is missing", APP_INSTALLATION_ID)
		}

		config := fmt.Sprintf(`
		resource "github_repository" "test1" {
			name      = "tf-acc-test-%s"
			auto_init = true
		}

		resource "github_repository" "test2" {
			name      = "tf-acc-test-%s"  
			auto_init = true
		}

		resource "github_app_installation_repositories" "test" {
			# The installation id of the app (in the organization).
			installation_id       = "%s"
			selected_repositories = [github_repository.test1.name, github_repository.test2.name]
		}
		`, randomID1, randomID2, installationID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet(
				"github_app_installation_repositories.test", "installation_id",
			),
			resource.TestCheckResourceAttr(
				"github_app_installation_repositories.test", "selected_repositories.#", "2",
			),
			resource.TestCheckResourceAttr(
				"github_app_installation_repositories.test", "installation_id", installationID,
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("updates repository list", func(t *testing.T) {
		if !exists {
			t.Skipf("%s environment variable is missing", APP_INSTALLATION_ID)
		}

		randomID3 := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		configInitial := fmt.Sprintf(`
		resource "github_repository" "test1" {
			name      = "tf-acc-test-%s"
			auto_init = true
		}

		resource "github_repository" "test2" {
			name      = "tf-acc-test-%s"
			auto_init = true
		}

		resource "github_app_installation_repositories" "test" {
			installation_id       = "%s"
			selected_repositories = [github_repository.test1.name, github_repository.test2.name]
		}
		`, randomID1, randomID2, installationID)

		configUpdated := fmt.Sprintf(`
		resource "github_repository" "test1" {
			name      = "tf-acc-test-%s"
			auto_init = true
		}

		resource "github_repository" "test2" {
			name      = "tf-acc-test-%s"
			auto_init = true
		}

		resource "github_repository" "test3" {
			name      = "tf-acc-test-%s"
			auto_init = true
		}

		resource "github_app_installation_repositories" "test" {
			installation_id       = "%s"
			selected_repositories = [github_repository.test1.name, github_repository.test3.name]
		}
		`, randomID1, randomID2, randomID3, installationID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: configInitial,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_app_installation_repositories.test", "selected_repositories.#", "2",
							),
						),
					},
					{
						Config: configUpdated,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_app_installation_repositories.test", "selected_repositories.#", "2",
							),
						),
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

	t.Run("deletes resource without error", func(t *testing.T) {
		if !exists {
			t.Skipf("%s environment variable is missing", APP_INSTALLATION_ID)
		}

		config := fmt.Sprintf(`
		resource "github_repository" "test1" {
			name      = "tf-acc-test-%s"
			auto_init = true
		}

		resource "github_repository" "test2" {
			name      = "tf-acc-test-%s"
			auto_init = true
		}

		resource "github_app_installation_repositories" "test" {
			installation_id       = "%s"
			selected_repositories = [github_repository.test1.name, github_repository.test2.name]
		}
		`, randomID1, randomID2, installationID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:  config,
						Destroy: true,
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

func TestAccGithubAppInstallationRepositoriesResource_import(t *testing.T) {
	const APP_INSTALLATION_ID = "APP_INSTALLATION_ID"
	randomID1 := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	randomID2 := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	installationID, exists := os.LookupEnv(APP_INSTALLATION_ID)

	if !exists {
		t.Skipf("%s environment variable is missing", APP_INSTALLATION_ID)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
					resource "github_repository" "test1" {
						name      = "tf-acc-test-%s"
						auto_init = true
					}

					resource "github_repository" "test2" {
						name      = "tf-acc-test-%s"
						auto_init = true
					}

					resource "github_app_installation_repositories" "test" {
						installation_id       = "%s"
						selected_repositories = [github_repository.test1.name, github_repository.test2.name]
					}
				`, randomID1, randomID2, installationID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_app_installation_repositories.test", "installation_id", installationID),
					resource.TestCheckResourceAttr("github_app_installation_repositories.test", "selected_repositories.#", "2"),
					resource.TestCheckResourceAttrSet("github_app_installation_repositories.test", "id"),
				),
			},
			{
				ResourceName:      "github_app_installation_repositories.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateId:     installationID,
			},
		},
	})
}

func TestAccGithubAppInstallationRepositoriesResource_validation(t *testing.T) {
	t.Run("invalid installation_id", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t, organization) },
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: `
						resource "github_app_installation_repositories" "test" {
							installation_id       = "not_a_number"
							selected_repositories = ["test-repo"]
						}
					`,
					ExpectError: regexp.MustCompile("unexpected ID format"),
				},
			},
		})
	})
}
