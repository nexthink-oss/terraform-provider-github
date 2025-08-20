package github

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubActionsEnvironmentSecretResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("creates and updates secrets without error", func(t *testing.T) {
		secretValue := base64.StdEncoding.EncodeToString([]byte("super_secret_value"))
		updatedSecretValue := base64.StdEncoding.EncodeToString([]byte("updated_super_secret_value"))

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
			  name = "tf-acc-test-%s"
			}

			resource "github_repository_environment" "test" {
			  repository       = github_repository.test.name
			  environment      = "environment / test"
			}

			resource "github_actions_environment_secret" "plaintext_secret" {
			  repository       = github_repository.test.name
			  environment      = github_repository_environment.test.environment
			  secret_name      = "test_plaintext_secret_name"
			  plaintext_value  = "%s"
			}

			resource "github_actions_environment_secret" "encrypted_secret" {
			  repository       = github_repository.test.name
			  environment      = github_repository_environment.test.environment
			  secret_name      = "test_encrypted_secret_name"
			  encrypted_value  = "%s"
			}
		`, randomID, secretValue, secretValue)

		checks := map[string]resource.TestCheckFunc{
			"before": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_actions_environment_secret.plaintext_secret", "plaintext_value",
					secretValue,
				),
				resource.TestCheckResourceAttr(
					"github_actions_environment_secret.encrypted_secret", "encrypted_value",
					secretValue,
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_environment_secret.plaintext_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_environment_secret.plaintext_secret", "updated_at",
				),
				resource.TestCheckResourceAttr(
					"github_actions_environment_secret.plaintext_secret", "repository",
					fmt.Sprintf("tf-acc-test-%s", randomID),
				),
				resource.TestCheckResourceAttr(
					"github_actions_environment_secret.plaintext_secret", "environment",
					"environment / test",
				),
				resource.TestCheckResourceAttr(
					"github_actions_environment_secret.plaintext_secret", "secret_name",
					"test_plaintext_secret_name",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_environment_secret.plaintext_secret", "id",
				),
			),
			"after": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_actions_environment_secret.plaintext_secret", "plaintext_value",
					updatedSecretValue,
				),
				resource.TestCheckResourceAttr(
					"github_actions_environment_secret.encrypted_secret", "encrypted_value",
					updatedSecretValue,
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_environment_secret.plaintext_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_environment_secret.plaintext_secret", "updated_at",
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
							secretValue,
							updatedSecretValue, 2),
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

	t.Run("deletes secrets without error", func(t *testing.T) {
		secretValue := base64.StdEncoding.EncodeToString([]byte("super_secret_value"))

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%s"
			}

			resource "github_repository_environment" "test" {
				repository       = github_repository.test.name
				environment      = "environment / test"
			}

			resource "github_actions_environment_secret" "plaintext_secret" {
				repository       = github_repository.test.name
				environment      = github_repository_environment.test.environment
				secret_name      = "test_plaintext_secret_name"
				plaintext_value  = "%s"
			}

			resource "github_actions_environment_secret" "encrypted_secret" {
				repository       = github_repository.test.name
				environment      = github_repository_environment.test.environment
				secret_name      = "test_encrypted_secret_name"
				encrypted_value  = "%s"
			}
		`, randomID, secretValue, secretValue)

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

func TestAccGithubActionsEnvironmentSecretResource_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

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
						repository  = github_repository.test.name
						environment = "test_environment"
					}

					resource "github_actions_environment_secret" "test" {
						repository    = github_repository.test.name
						environment   = github_repository_environment.test.environment
						secret_name   = "test_secret"
						plaintext_value = "test_value"
					}
				`, randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_actions_environment_secret.test", "secret_name", "test_secret"),
					resource.TestCheckResourceAttr("github_actions_environment_secret.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_actions_environment_secret.test", "environment", "test_environment"),
					resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "created_at"),
					resource.TestCheckResourceAttrSet("github_actions_environment_secret.test", "updated_at"),
				),
			},
			{
				ResourceName:            "github_actions_environment_secret.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateId:           fmt.Sprintf("tf-acc-test-%s/test_environment/test_secret", randomID),
				ImportStateVerifyIgnore: []string{"plaintext_value", "encrypted_value"}, // These cannot be retrieved from API
			},
		},
	})
}

func TestAccGithubActionsEnvironmentSecretResource_validation(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	// Test invalid secret name (starting with number)
	t.Run("invalid secret name starting with number", func(t *testing.T) {
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
							repository  = github_repository.test.name
							environment = "test_environment"
						}

						resource "github_actions_environment_secret" "test" {
							repository    = github_repository.test.name
							environment   = github_repository_environment.test.environment
							secret_name   = "1invalid_name"
							plaintext_value = "test_value"
						}
					`, randomID),
					ExpectError: regexp.MustCompile("Secret names can only contain alphanumeric characters or underscores and must not start with a number"),
				},
			},
		})
	})

	// Test invalid secret name (GITHUB_ prefix)
	t.Run("invalid secret name with GITHUB_ prefix", func(t *testing.T) {
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
							repository  = github_repository.test.name
							environment = "test_environment"
						}

						resource "github_actions_environment_secret" "test" {
							repository    = github_repository.test.name
							environment   = github_repository_environment.test.environment
							secret_name   = "GITHUB_SECRET"
							plaintext_value = "test_value"
						}
					`, randomID),
					ExpectError: regexp.MustCompile("Secret names must not start with the GITHUB_ prefix"),
				},
			},
		})
	})

	// Test conflicting plaintext_value and encrypted_value
	t.Run("conflicting plaintext_value and encrypted_value", func(t *testing.T) {
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
							repository  = github_repository.test.name
							environment = "test_environment"
						}

						resource "github_actions_environment_secret" "test" {
							repository      = github_repository.test.name
							environment     = github_repository_environment.test.environment
							secret_name     = "test_secret"
							plaintext_value = "test_value"
							encrypted_value = "encrypted_test_value"
						}
					`, randomID),
					ExpectError: regexp.MustCompile("Conflicting Attribute Configuration"),
				},
			},
		})
	})

	// Test invalid base64 encrypted_value
	t.Run("invalid base64 encrypted_value", func(t *testing.T) {
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
							repository  = github_repository.test.name
							environment = "test_environment"
						}

						resource "github_actions_environment_secret" "test" {
							repository      = github_repository.test.name
							environment     = github_repository_environment.test.environment
							secret_name     = "test_secret"
							encrypted_value = "invalid-base64!!"
						}
					`, randomID),
					ExpectError: regexp.MustCompile("Invalid Base64 Value"),
				},
			},
		})
	})

	// Test missing both plaintext_value and encrypted_value
	t.Run("missing both plaintext_value and encrypted_value", func(t *testing.T) {
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
							repository  = github_repository.test.name
							environment = "test_environment"
						}

						resource "github_actions_environment_secret" "test" {
							repository  = github_repository.test.name
							environment = github_repository_environment.test.environment
							secret_name = "test_secret"
						}
					`, randomID),
					ExpectError: regexp.MustCompile("Missing Secret Value"),
				},
			},
		})
	})
}
