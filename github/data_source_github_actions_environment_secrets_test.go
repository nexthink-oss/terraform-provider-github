package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubActionsEnvironmentSecretsDataSource(t *testing.T) {

	t.Run("queries actions secrets from an environment", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_environment" "test" {
				repository       = github_repository.test.name
				environment      = "environment / test"
			  }

			resource "github_actions_environment_secret" "test" {
				secret_name 		= "secret_1"
				environment      	= github_repository_environment.test.environment
				repository  		= github_repository.test.name
				plaintext_value = "foo"
			}
		`, randomID)

		config2 := config + `
			data "github_actions_environment_secrets" "test" {
				name = github_repository.test.name
				environment      	= github_repository_environment.test.environment
			}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_actions_environment_secrets.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
			resource.TestCheckResourceAttr("data.github_actions_environment_secrets.test", "secrets.#", "1"),
			resource.TestCheckResourceAttr("data.github_actions_environment_secrets.test", "secrets.0.name", "SECRET_1"),
			resource.TestCheckResourceAttrSet("data.github_actions_environment_secrets.test", "secrets.0.created_at"),
			resource.TestCheckResourceAttrSet("data.github_actions_environment_secrets.test", "secrets.0.updated_at"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  resource.ComposeTestCheckFunc(),
					},
					{
						Config: config2,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("queries actions secrets using full_name", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_environment" "test" {
				repository       = github_repository.test.name
				environment      = "environment-test"
			  }

			resource "github_actions_environment_secret" "test" {
				secret_name 		= "secret_1"
				environment      	= github_repository_environment.test.environment
				repository  		= github_repository.test.name
				plaintext_value = "foo"
			}
		`, randomID)

		config2 := config + `
			data "github_actions_environment_secrets" "test" {
				full_name = github_repository.test.full_name
				environment = github_repository_environment.test.environment
			}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet("data.github_actions_environment_secrets.test", "full_name"),
			resource.TestCheckResourceAttr("data.github_actions_environment_secrets.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
			resource.TestCheckResourceAttr("data.github_actions_environment_secrets.test", "secrets.#", "1"),
			resource.TestCheckResourceAttr("data.github_actions_environment_secrets.test", "secrets.0.name", "SECRET_1"),
			resource.TestCheckResourceAttrSet("data.github_actions_environment_secrets.test", "secrets.0.created_at"),
			resource.TestCheckResourceAttrSet("data.github_actions_environment_secrets.test", "secrets.0.updated_at"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  resource.ComposeTestCheckFunc(),
					},
					{
						Config: config2,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("handles empty environment secrets list", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_environment" "test" {
				repository       = github_repository.test.name
				environment      = "empty-env"
			  }

			data "github_actions_environment_secrets" "test" {
				name = github_repository.test.name
				environment = github_repository_environment.test.environment
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_actions_environment_secrets.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
			resource.TestCheckResourceAttr("data.github_actions_environment_secrets.test", "secrets.#", "0"),
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

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubActionsEnvironmentSecretsDataSource_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	config := fmt.Sprintf(`
		resource "github_repository" "test" {
			name      = "tf-acc-test-%s"
			auto_init = true
		}

		resource "github_repository_environment" "test" {
			repository       = github_repository.test.name
			environment      = "migration-test"
		  }

		resource "github_actions_environment_secret" "test" {
			secret_name 		= "secret_1"
			environment      	= github_repository_environment.test.environment
			repository  		= github_repository.test.name
			plaintext_value = "foo"
		}

		data "github_actions_environment_secrets" "test" {
			name = github_repository.test.name
			environment = github_repository_environment.test.environment
		}
	`, randomID)

	check := resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr("data.github_actions_environment_secrets.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
		resource.TestCheckResourceAttr("data.github_actions_environment_secrets.test", "secrets.#", "1"),
		resource.TestCheckResourceAttr("data.github_actions_environment_secrets.test", "secrets.0.name", "SECRET_1"),
		resource.TestCheckResourceAttrSet("data.github_actions_environment_secrets.test", "secrets.0.created_at"),
		resource.TestCheckResourceAttrSet("data.github_actions_environment_secrets.test", "secrets.0.updated_at"),
	)

	testCase := func(t *testing.T, mode string) {
		resource.Test(t, resource.TestCase{
			PreCheck: func() { skipUnlessMode(t, mode) },
			Steps: []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"github": {
							Source:            "integrations/github",
							VersionConstraint: "~> 6.0",
						},
					},
					Config: config,
					Check:  check,
				},
				{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Config:                   config,
					Check:                    check,
				},
			},
		})
	}

	t.Run("with an organization account", func(t *testing.T) {
		testCase(t, organization)
	})
}
