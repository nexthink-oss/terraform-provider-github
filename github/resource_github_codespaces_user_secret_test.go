package github

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubCodespacesUserSecret(t *testing.T) {
	t.Run("creates and updates secrets without error", func(t *testing.T) {
		secretValue := base64.StdEncoding.EncodeToString([]byte("super_secret_value"))
		updatedSecretValue := base64.StdEncoding.EncodeToString([]byte("updated_super_secret_value"))

		config := fmt.Sprintf(`
			resource "github_codespaces_user_secret" "plaintext_secret" {
			  secret_name      = "test_plaintext_secret"
			  plaintext_value  = "%s"
			}

			resource "github_codespaces_user_secret" "encrypted_secret" {
			  secret_name      = "test_encrypted_secret"
			  encrypted_value  = "%s"
			}
		`, secretValue, secretValue)

		checks := map[string]resource.TestCheckFunc{
			"before": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_codespaces_user_secret.plaintext_secret", "plaintext_value",
					secretValue,
				),
				resource.TestCheckResourceAttr(
					"github_codespaces_user_secret.encrypted_secret", "encrypted_value",
					secretValue,
				),
				resource.TestCheckResourceAttrSet(
					"github_codespaces_user_secret.plaintext_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_codespaces_user_secret.plaintext_secret", "updated_at",
				),
			),
			"after": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_codespaces_user_secret.plaintext_secret", "plaintext_value",
					updatedSecretValue,
				),
				resource.TestCheckResourceAttr(
					"github_codespaces_user_secret.encrypted_secret", "encrypted_value",
					updatedSecretValue,
				),
				resource.TestCheckResourceAttrSet(
					"github_codespaces_user_secret.plaintext_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_codespaces_user_secret.plaintext_secret", "updated_at",
				),
			),
		}

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
	})

	t.Run("deletes secrets without error", func(t *testing.T) {
		config := `
				resource "github_codespaces_user_secret" "plaintext_secret" {
					secret_name      = "test_plaintext_secret"
				}

				resource "github_codespaces_user_secret" "encrypted_secret" {
					secret_name      = "test_encrypted_secret"
				}
			`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
	})

	t.Run("imports secrets without error", func(t *testing.T) {
		secretValue := "super_secret_value"

		config := fmt.Sprintf(`
			resource "github_codespaces_user_secret" "test_secret" {
				secret_name      = "test_plaintext_secret"
				plaintext_value  = "%s"
			}
		`, secretValue)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_codespaces_user_secret.test_secret", "plaintext_value",
				secretValue,
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
					{
						ResourceName:            "github_codespaces_user_secret.test_secret",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"plaintext_value"},
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
	})

	t.Run("manages selected repository ids", func(t *testing.T) {
		secretValue := "test_secret_value"

		config := fmt.Sprintf(`
			resource "github_codespaces_user_secret" "test_secret" {
				secret_name      = "test_secret_with_repos"
				plaintext_value  = "%s"
				selected_repository_ids = [123456, 789012]
			}
		`, secretValue)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_codespaces_user_secret.test_secret", "secret_name",
								"test_secret_with_repos",
							),
							resource.TestCheckResourceAttr(
								"github_codespaces_user_secret.test_secret", "plaintext_value",
								secretValue,
							),
							resource.TestCheckResourceAttr(
								"github_codespaces_user_secret.test_secret", "selected_repository_ids.#",
								"2",
							),
							resource.TestCheckTypeSetElemAttr(
								"github_codespaces_user_secret.test_secret", "selected_repository_ids.*",
								"123456",
							),
							resource.TestCheckTypeSetElemAttr(
								"github_codespaces_user_secret.test_secret", "selected_repository_ids.*",
								"789012",
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
			testCase(t, individual)
		})
	})

	t.Run("validates secret name format", func(t *testing.T) {
		testCases := []struct {
			name        string
			secretName  string
			expectError bool
		}{
			{
				name:        "valid name",
				secretName:  "VALID_SECRET_NAME",
				expectError: false,
			},
			{
				name:        "invalid name starting with number",
				secretName:  "1INVALID_NAME",
				expectError: true,
			},
			{
				name:        "invalid name with GITHUB prefix",
				secretName:  "GITHUB_SECRET",
				expectError: true,
			},
			{
				name:        "invalid name with special chars",
				secretName:  "INVALID-NAME",
				expectError: true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := fmt.Sprintf(`
					resource "github_codespaces_user_secret" "test_secret" {
						secret_name      = "%s"
						plaintext_value  = "test_value"
					}
				`, tc.secretName)

				testCase := func(t *testing.T, mode string) {
					resource.Test(t, resource.TestCase{
						PreCheck:                 func() { skipUnlessMode(t, mode) },
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Steps: []resource.TestStep{
							{
								Config: config,
								ExpectError: func() *regexp.Regexp {
									if tc.expectError {
										return regexp.MustCompile("Invalid Secret Name")
									}
									return nil
								}(),
							},
						},
					})
				}

				t.Run("with an individual account", func(t *testing.T) {
					testCase(t, individual)
				})
			})
		}
	})

	t.Run("validates conflicting values", func(t *testing.T) {
		config := `
			resource "github_codespaces_user_secret" "test_secret" {
				secret_name      = "test_secret"
				plaintext_value  = "plain_value"
				encrypted_value  = "ZW5jcnlwdGVkX3ZhbHVl"
			}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("Conflicting Attribute Configured"),
					},
				},
			})
		}

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})
	})

	t.Run("validates base64 encrypted value", func(t *testing.T) {
		config := `
			resource "github_codespaces_user_secret" "test_secret" {
				secret_name      = "test_secret"
				encrypted_value  = "invalid_base64!"
			}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("Invalid Base64 Value"),
					},
				},
			})
		}

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})
	})
}
