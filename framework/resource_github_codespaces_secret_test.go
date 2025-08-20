package framework

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGithubCodespacesSecret(t *testing.T) {

	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("reads a repository public key without error", func(t *testing.T) {

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
			  name = "tf-acc-test-%s"
			}

			data "github_codespaces_public_key" "test_pk" {
			  repository = github_repository.test.name
			}
		`, randomID)

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrSet(
				"data.github_codespaces_public_key.test_pk", "key_id",
			),
			resource.TestCheckResourceAttrSet(
				"data.github_codespaces_public_key.test_pk", "key",
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
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})

	})

	t.Run("creates and updates secrets without error", func(t *testing.T) {
		secretValue := base64.StdEncoding.EncodeToString([]byte("super_secret_value"))
		updatedSecretValue := base64.StdEncoding.EncodeToString([]byte("updated_super_secret_value"))

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
			  name = "tf-acc-test-%s"
			}

			resource "github_codespaces_secret" "plaintext_secret" {
			  repository       = github_repository.test.name
			  secret_name      = "test_plaintext_secret"
			  plaintext_value  = "%s"
			}

			resource "github_codespaces_secret" "encrypted_secret" {
			  repository       = github_repository.test.name
			  secret_name      = "test_encrypted_secret"
			  encrypted_value  = "%s"
			}
		`, randomID, secretValue, secretValue)

		checks := map[string]resource.TestCheckFunc{
			"before": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_codespaces_secret.plaintext_secret", "plaintext_value",
					secretValue,
				),
				resource.TestCheckResourceAttr(
					"github_codespaces_secret.encrypted_secret", "encrypted_value",
					secretValue,
				),
				resource.TestCheckResourceAttrSet(
					"github_codespaces_secret.plaintext_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_codespaces_secret.plaintext_secret", "updated_at",
				),
			),
			"after": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_codespaces_secret.plaintext_secret", "plaintext_value",
					updatedSecretValue,
				),
				resource.TestCheckResourceAttr(
					"github_codespaces_secret.encrypted_secret", "encrypted_value",
					updatedSecretValue,
				),
				resource.TestCheckResourceAttrSet(
					"github_codespaces_secret.plaintext_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_codespaces_secret.plaintext_secret", "updated_at",
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

	t.Run("creates and updates repository name without error", func(t *testing.T) {
		repoName := fmt.Sprintf("tf-acc-test-%s", randomID)
		updatedRepoName := fmt.Sprintf("tf-acc-test-%s-updated", randomID)
		secretValue := base64.StdEncoding.EncodeToString([]byte("super_secret_value"))

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
			  name = "%s"
			}
			resource "github_codespaces_secret" "plaintext_secret" {
			  repository       = github_repository.test.name
			  secret_name      = "test_plaintext_secret"
			  plaintext_value  = "%s"
			}
			resource "github_codespaces_secret" "encrypted_secret" {
			  repository       = github_repository.test.name
			  secret_name      = "test_encrypted_secret"
			  encrypted_value  = "%s"
			}
		`, repoName, secretValue, secretValue)

		checks := map[string]resource.TestCheckFunc{
			"before": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_codespaces_secret.plaintext_secret", "repository",
					repoName,
				),
				resource.TestCheckResourceAttr(
					"github_codespaces_secret.plaintext_secret", "plaintext_value",
					secretValue,
				),
				resource.TestCheckResourceAttr(
					"github_codespaces_secret.encrypted_secret", "encrypted_value",
					secretValue,
				),
				resource.TestCheckResourceAttrSet(
					"github_codespaces_secret.plaintext_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_codespaces_secret.plaintext_secret", "updated_at",
				),
			),
			"after": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_codespaces_secret.plaintext_secret", "repository",
					updatedRepoName,
				),
				resource.TestCheckResourceAttr(
					"github_codespaces_secret.plaintext_secret", "plaintext_value",
					secretValue,
				),
				resource.TestCheckResourceAttr(
					"github_codespaces_secret.encrypted_secret", "encrypted_value",
					secretValue,
				),
				resource.TestCheckResourceAttrSet(
					"github_codespaces_secret.plaintext_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_codespaces_secret.plaintext_secret", "updated_at",
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
							repoName,
							updatedRepoName, 2),
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
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%s"
			}

			resource "github_codespaces_secret" "plaintext_secret" {
				repository 	= github_repository.test.name
				secret_name	= "test_plaintext_secret"
			}

			resource "github_codespaces_secret" "encrypted_secret" {
				repository 	= github_repository.test.name
				secret_name	= "test_encrypted_secret"
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

	t.Run("imports secrets without error", func(t *testing.T) {
		secretValue := base64.StdEncoding.EncodeToString([]byte("super_secret_value"))

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
			  name = "tf-acc-test-%s"
			}

			resource "github_codespaces_secret" "plaintext_secret" {
			  repository       = github_repository.test.name
			  secret_name      = "test_plaintext_secret"
			  plaintext_value  = "%s"
			}
		`, randomID, secretValue)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
					},
					{
						ResourceName:            "github_codespaces_secret.plaintext_secret",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateIdFunc:       testAccGithubCodespacesSecretImportStateIdFunc("github_codespaces_secret.plaintext_secret"),
						ImportStateVerifyIgnore: []string{"plaintext_value", "encrypted_value"},
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

func testAccGithubCodespacesSecretImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("not found: %s", resourceName)
		}

		repo := rs.Primary.Attributes["repository"]
		secretName := rs.Primary.Attributes["secret_name"]
		return fmt.Sprintf("%s/%s", repo, secretName), nil
	}
}
