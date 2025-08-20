package framework

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubActionsEnvironmentVariablesDataSource(t *testing.T) {
	t.Run("queries actions variables from an environment", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
			  name = "tf-acc-test-%s"
			}

			resource "github_repository_environment" "test" {
			  repository       = github_repository.test.name
			  environment      = "environment / test"
			}

			resource "github_actions_environment_variable" "variable" {
			  repository       = github_repository.test.name
			  environment      = github_repository_environment.test.environment
			  variable_name    = "test_variable"
			  value  		   = "foo"
			}
			`, randomID)

		config2 := config + `
			data "github_actions_environment_variables" "test" {
				environment      = github_repository_environment.test.environment
				name 		     = github_repository.test.name
			}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_actions_environment_variables.test", "variables.#", "1"),
			resource.TestCheckResourceAttr("data.github_actions_environment_variables.test", "variables.0.name", strings.ToUpper("test_variable")),
			resource.TestCheckResourceAttr("data.github_actions_environment_variables.test", "variables.0.value", "foo"),
			resource.TestCheckResourceAttrSet("data.github_actions_environment_variables.test", "variables.0.created_at"),
			resource.TestCheckResourceAttrSet("data.github_actions_environment_variables.test", "variables.0.updated_at"),
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

	t.Run("queries actions variables using full_name", func(t *testing.T) {
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

			resource "github_actions_environment_variable" "test" {
				variable_name = "variable_1"
				repository    = github_repository.test.name
				environment   = github_repository_environment.test.environment
				value         = "foo"
			}

			data "github_actions_environment_variables" "test" {
				full_name   = github_repository.test.full_name
				environment = github_repository_environment.test.environment
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_actions_environment_variables.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
			resource.TestCheckResourceAttrSet("data.github_actions_environment_variables.test", "full_name"),
			resource.TestCheckResourceAttr("data.github_actions_environment_variables.test", "variables.#", "1"),
			resource.TestCheckResourceAttr("data.github_actions_environment_variables.test", "variables.0.name", "VARIABLE_1"),
			resource.TestCheckResourceAttr("data.github_actions_environment_variables.test", "variables.0.value", "foo"),
			resource.TestCheckResourceAttrSet("data.github_actions_environment_variables.test", "variables.0.created_at"),
			resource.TestCheckResourceAttrSet("data.github_actions_environment_variables.test", "variables.0.updated_at"),
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

	t.Run("fails when both name and full_name are provided", func(t *testing.T) {
		config := `
			data "github_actions_environment_variables" "test" {
				name        = "test-repo"
				full_name   = "test-owner/test-repo"
				environment = "test-env"
			}
		`

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
				},
			},
		})
	})

	t.Run("fails when neither name nor full_name are provided", func(t *testing.T) {
		config := `
			data "github_actions_environment_variables" "test" {
				environment = "test-env"
			}
		`

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile(`Invalid Configuration`),
				},
			},
		})
	})

	t.Run("validates migration compatibility between SDKv2 and Framework", func(t *testing.T) {
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

			resource "github_actions_environment_variable" "test" {
				variable_name = "variable_1"
				repository    = github_repository.test.name
				environment   = github_repository_environment.test.environment
				value         = "foo"
			}

			data "github_actions_environment_variables" "test" {
				name        = github_repository.test.name
				environment = github_repository_environment.test.environment
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_actions_environment_variables.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
			resource.TestCheckResourceAttr("data.github_actions_environment_variables.test", "variables.#", "1"),
			resource.TestCheckResourceAttr("data.github_actions_environment_variables.test", "variables.0.name", "VARIABLE_1"),
			resource.TestCheckResourceAttr("data.github_actions_environment_variables.test", "variables.0.value", "foo"),
			resource.TestCheckResourceAttrSet("data.github_actions_environment_variables.test", "variables.0.created_at"),
			resource.TestCheckResourceAttrSet("data.github_actions_environment_variables.test", "variables.0.updated_at"),
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

		t.Run("with muxed provider (SDKv2 + Framework)", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
