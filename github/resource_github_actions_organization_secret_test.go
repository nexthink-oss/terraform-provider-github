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

func TestAccGithubActionsOrganizationSecretResource_basic(t *testing.T) {
	t.Run("creates and updates secrets without error", func(t *testing.T) {
		secretValue := base64.StdEncoding.EncodeToString([]byte("super_secret_value"))
		updatedSecretValue := base64.StdEncoding.EncodeToString([]byte("updated_super_secret_value"))

		config := fmt.Sprintf(`
			resource "github_actions_organization_secret" "plaintext_secret" {
				secret_name      = "test_plaintext_secret"
				plaintext_value  = "%s"
				visibility       = "private"
			}

			resource "github_actions_organization_secret" "encrypted_secret" {
				secret_name      = "test_encrypted_secret"
				encrypted_value  = "%s"
				visibility       = "private"
			}
		`, secretValue, secretValue)

		checks := map[string]resource.TestCheckFunc{
			"before": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_actions_organization_secret.plaintext_secret", "plaintext_value",
					secretValue,
				),
				resource.TestCheckResourceAttr(
					"github_actions_organization_secret.encrypted_secret", "encrypted_value",
					secretValue,
				),
				resource.TestCheckResourceAttr(
					"github_actions_organization_secret.plaintext_secret", "visibility",
					"private",
				),
				resource.TestCheckResourceAttr(
					"github_actions_organization_secret.encrypted_secret", "visibility",
					"private",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_organization_secret.plaintext_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_organization_secret.plaintext_secret", "updated_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_organization_secret.encrypted_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_organization_secret.encrypted_secret", "updated_at",
				),
			),
			"after": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_actions_organization_secret.plaintext_secret", "plaintext_value",
					updatedSecretValue,
				),
				resource.TestCheckResourceAttr(
					"github_actions_organization_secret.encrypted_secret", "encrypted_value",
					updatedSecretValue,
				),
				resource.TestCheckResourceAttr(
					"github_actions_organization_secret.plaintext_secret", "visibility",
					"private",
				),
				resource.TestCheckResourceAttr(
					"github_actions_organization_secret.encrypted_secret", "visibility",
					"private",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_organization_secret.plaintext_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_organization_secret.plaintext_secret", "updated_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_organization_secret.encrypted_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_organization_secret.encrypted_secret", "updated_at",
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, "organization")
		})
	})

	t.Run("deletes secrets without error", func(t *testing.T) {
		config := `
			resource "github_actions_organization_secret" "plaintext_secret" {
				secret_name      = "test_plaintext_secret"
				plaintext_value  = "test_value"
				visibility       = "private"
			}

			resource "github_actions_organization_secret" "encrypted_secret" {
				secret_name      = "test_encrypted_secret"
				encrypted_value  = "dGVzdF92YWx1ZQ=="
				visibility       = "private"
			}
		`

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
			testCase(t, "organization")
		})
	})

	t.Run("imports secrets without error", func(t *testing.T) {
		secretValue := "super_secret_value"

		config := fmt.Sprintf(`
			resource "github_actions_organization_secret" "test_secret" {
				secret_name      = "test_plaintext_secret"
				plaintext_value  = "%s"
				visibility       = "private"
			}
		`, secretValue)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_actions_organization_secret.test_secret", "plaintext_value",
				secretValue,
			),
			resource.TestCheckResourceAttr(
				"github_actions_organization_secret.test_secret", "visibility",
				"private",
			),
			resource.TestCheckResourceAttr(
				"github_actions_organization_secret.test_secret", "secret_name",
				"test_plaintext_secret",
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
					{
						ResourceName:            "github_actions_organization_secret.test_secret",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"plaintext_value", "encrypted_value"},
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

func TestAccGithubActionsOrganizationSecretResource_visibility(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("manages secrets with different visibility settings", func(t *testing.T) {
		configAll := fmt.Sprintf(`
			resource "github_actions_organization_secret" "test_all" {
				secret_name      = "test_secret_all_%s"
				plaintext_value  = "test_value"
				visibility       = "all"
			}
		`, randomID)

		configPrivate := fmt.Sprintf(`
			resource "github_actions_organization_secret" "test_private" {
				secret_name      = "test_secret_private_%s"
				plaintext_value  = "test_value"
				visibility       = "private"
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: configAll,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_actions_organization_secret.test_all", "visibility",
								"all",
							),
							resource.TestCheckResourceAttr(
								"github_actions_organization_secret.test_all", "selected_repository_ids.#",
								"0",
							),
						),
					},
					{
						Config: configPrivate,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_actions_organization_secret.test_private", "visibility",
								"private",
							),
							resource.TestCheckResourceAttr(
								"github_actions_organization_secret.test_private", "selected_repository_ids.#",
								"0",
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
			testCase(t, "organization")
		})
	})
}

func TestAccGithubActionsOrganizationSecretResource_selectedRepositories(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("manages secrets with selected repositories", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test_repo_1" {
				name = "tf-acc-test-secret-repo-1-%s"
			}

			resource "github_repository" "test_repo_2" {
				name = "tf-acc-test-secret-repo-2-%s"
			}

			resource "github_actions_organization_secret" "test_selected" {
				secret_name      = "test_secret_selected_%s"
				plaintext_value  = "test_value"
				visibility       = "selected"
				selected_repository_ids = [
					github_repository.test_repo_1.repo_id,
					github_repository.test_repo_2.repo_id,
				]
			}
		`, randomID, randomID, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_actions_organization_secret.test_selected", "visibility",
								"selected",
							),
							resource.TestCheckResourceAttr(
								"github_actions_organization_secret.test_selected", "selected_repository_ids.#",
								"2",
							),
							resource.TestCheckTypeSetElemAttrPair(
								"github_actions_organization_secret.test_selected", "selected_repository_ids.*",
								"github_repository.test_repo_1", "repo_id",
							),
							resource.TestCheckTypeSetElemAttrPair(
								"github_actions_organization_secret.test_selected", "selected_repository_ids.*",
								"github_repository.test_repo_2", "repo_id",
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
			testCase(t, "organization")
		})
	})
}

func TestAccGithubActionsOrganizationSecretResource_validation(t *testing.T) {
	t.Run("validates secret name format", func(t *testing.T) {
		config := `
			resource "github_actions_organization_secret" "test" {
				secret_name      = "123_invalid"
				plaintext_value  = "test_value"
				visibility       = "private"
			}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("Secret names can only contain alphanumeric characters or underscores and must not start with a number"),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, "organization")
		})
	})

	t.Run("validates GITHUB_ prefix", func(t *testing.T) {
		config := `
			resource "github_actions_organization_secret" "test" {
				secret_name      = "GITHUB_SECRET"
				plaintext_value  = "test_value"
				visibility       = "private"
			}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("Secret names must not start with the GITHUB_ prefix"),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, "organization")
		})
	})

	t.Run("validates conflicting value attributes", func(t *testing.T) {
		config := `
			resource "github_actions_organization_secret" "test" {
				secret_name      = "test_secret"
				plaintext_value  = "test_value"
				encrypted_value  = "dGVzdF92YWx1ZQ=="
				visibility       = "private"
			}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("Attribute.*cannot be specified when.*is specified"),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, "organization")
		})
	})

	t.Run("validates invalid base64 encrypted value", func(t *testing.T) {
		config := `
			resource "github_actions_organization_secret" "test" {
				secret_name      = "test_secret"
				encrypted_value  = "invalid_base64!"
				visibility       = "private"
			}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("Value must be valid base64"),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, "organization")
		})
	})

	t.Run("validates invalid visibility", func(t *testing.T) {
		config := `
			resource "github_actions_organization_secret" "test" {
				secret_name      = "test_secret"
				plaintext_value  = "test_value"
				visibility       = "invalid"
			}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("Value must be one of:.*all.*private.*selected"),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, "organization")
		})
	})

	t.Run("validates selected repositories without selected visibility", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test_repo" {
				name = "tf-acc-test-secret-repo-%s"
			}

			resource "github_actions_organization_secret" "test" {
				secret_name      = "test_secret"
				plaintext_value  = "test_value"
				visibility       = "private"
				selected_repository_ids = [
					github_repository.test_repo.repo_id,
				]
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("selected_repository_ids can only be set when visibility is 'selected'"),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, "organization")
		})
	})

	t.Run("validates missing secret value", func(t *testing.T) {
		config := `
			resource "github_actions_organization_secret" "test" {
				secret_name      = "test_secret"
				visibility       = "private"
			}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("Either 'plaintext_value' or 'encrypted_value' must be provided"),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, "organization")
		})
	})
}
