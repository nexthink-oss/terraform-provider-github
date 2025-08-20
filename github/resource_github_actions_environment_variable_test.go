package github

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubActionsEnvironmentVariableResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("creates and updates environment variables without error", func(t *testing.T) {
		value := "my_variable_value"
		updatedValue := "my_updated_variable_value"

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
			  value  = "%s"
			}
			`, randomID, value)

		checks := map[string]resource.TestCheckFunc{
			"before": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_actions_environment_variable.variable", "value",
					value,
				),
				resource.TestCheckResourceAttr(
					"github_actions_environment_variable.variable", "repository",
					fmt.Sprintf("tf-acc-test-%s", randomID),
				),
				resource.TestCheckResourceAttr(
					"github_actions_environment_variable.variable", "environment",
					"environment / test",
				),
				resource.TestCheckResourceAttr(
					"github_actions_environment_variable.variable", "variable_name",
					"test_variable",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_environment_variable.variable", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_environment_variable.variable", "updated_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_environment_variable.variable", "id",
				),
			),
			"after": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_actions_environment_variable.variable", "value",
					updatedValue,
				),
				resource.TestCheckResourceAttr(
					"github_actions_environment_variable.variable", "repository",
					fmt.Sprintf("tf-acc-test-%s", randomID),
				),
				resource.TestCheckResourceAttr(
					"github_actions_environment_variable.variable", "environment",
					"environment / test",
				),
				resource.TestCheckResourceAttr(
					"github_actions_environment_variable.variable", "variable_name",
					"test_variable",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_environment_variable.variable", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_environment_variable.variable", "updated_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_environment_variable.variable", "id",
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
						Check:  checks["before"],
					},
					{
						Config: strings.Replace(config,
							value,
							updatedValue, 1),
						Check: checks["after"],
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

	t.Run("deletes environment variables without error", func(t *testing.T) {
		config := fmt.Sprintf(`
				resource "github_repository" "test" {
					name = "tf-acc-test-%s"
				}

				resource "github_repository_environment" "test" {
					repository       = github_repository.test.name
					environment      = "environment / test"
				}

				resource "github_actions_environment_variable" "variable" {
					repository 		= github_repository.test.name
					environment     = github_repository_environment.test.environment
					variable_name	= "test_variable"
					value 			= "my_variable_value"
				}
			`, randomID)

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
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})

	})
}

func TestAccGithubActionsEnvironmentVariableResource_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	value := "my_variable_value"
	envName := "environment / test"
	varName := "test_variable"

	t.Run("imports environment variables without error", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
			  name = "tf-acc-test-%s"
			}

			resource "github_repository_environment" "test" {
			  repository       = github_repository.test.name
			  environment      = "%s"
			}

			resource "github_actions_environment_variable" "variable" {
			  repository       = github_repository.test.name
			  environment      = github_repository_environment.test.environment
			  variable_name    = "%s"
			  value  = "%s"
			}
			`, randomID, envName, varName, value)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("github_actions_environment_variable.variable", "variable_name", varName),
							resource.TestCheckResourceAttr("github_actions_environment_variable.variable", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
							resource.TestCheckResourceAttr("github_actions_environment_variable.variable", "environment", envName),
							resource.TestCheckResourceAttr("github_actions_environment_variable.variable", "value", value),
							resource.TestCheckResourceAttrSet("github_actions_environment_variable.variable", "created_at"),
							resource.TestCheckResourceAttrSet("github_actions_environment_variable.variable", "updated_at"),
						),
					},
					{
						ResourceName:      "github_actions_environment_variable.variable",
						ImportStateId:     fmt.Sprintf(`tf-acc-test-%s/%s/%s`, randomID, envName, varName),
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
}

func TestAccGithubActionsEnvironmentVariableResource_validation(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	// Test invalid variable name (starting with number)
	t.Run("invalid variable name starting with number", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t, individual) },
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
						resource "github_repository" "test" {
							name = "tf-acc-test-%s"
						}

						resource "github_repository_environment" "test" {
						  repository = github_repository.test.name
						  environment = "test_environment"
						}

						resource "github_actions_environment_variable" "test" {
							repository    = github_repository.test.name
							environment   = github_repository_environment.test.environment
							variable_name = "1invalid_name"
							value         = "test_value"
						}
					`, randomID),
					ExpectError: regexp.MustCompile("Variable names can only contain alphanumeric characters or underscores and must not start with a number"),
				},
			},
		})
	})

	// Test invalid variable name (GITHUB_ prefix)
	t.Run("invalid variable name with GITHUB_ prefix", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t, individual) },
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
						resource "github_repository" "test" {
							name = "tf-acc-test-%s"
						}

						resource "github_repository_environment" "test" {
						  repository = github_repository.test.name
						  environment = "test_environment"
						}

						resource "github_actions_environment_variable" "test" {
							repository    = github_repository.test.name
							environment   = github_repository_environment.test.environment
							variable_name = "GITHUB_VARIABLE"
							value         = "test_value"
						}
					`, randomID),
					ExpectError: regexp.MustCompile("Variable names must not start with the GITHUB_ prefix"),
				},
			},
		})
	})

	// Test invalid variable name (special characters)
	t.Run("invalid variable name with special characters", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t, individual) },
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
						resource "github_repository" "test" {
							name = "tf-acc-test-%s"
						}

						resource "github_repository_environment" "test" {
						  repository = github_repository.test.name
						  environment = "test_environment"
						}

						resource "github_actions_environment_variable" "test" {
							repository    = github_repository.test.name
							environment   = github_repository_environment.test.environment
							variable_name = "invalid-name"
							value         = "test_value"
						}
					`, randomID),
					ExpectError: regexp.MustCompile("Variable names can only contain alphanumeric characters or underscores and must not start with a number"),
				},
			},
		})
	})
}
