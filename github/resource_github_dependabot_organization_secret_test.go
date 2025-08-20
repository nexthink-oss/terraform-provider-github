package github

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubDependabotOrganizationSecret(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("creates and updates secrets without error", func(t *testing.T) {
		secretValue := base64.StdEncoding.EncodeToString([]byte("super_secret_value"))
		updatedSecretValue := base64.StdEncoding.EncodeToString([]byte("updated_super_secret_value"))

		config := fmt.Sprintf(`
			resource "github_dependabot_organization_secret" "plaintext_secret" {
			  secret_name      = "test_plaintext_secret_%s"
			  plaintext_value  = "%s"
			  visibility       = "private"
			}

			resource "github_dependabot_organization_secret" "encrypted_secret" {
			  secret_name      = "test_encrypted_secret_%s"
			  encrypted_value  = "%s"
			  visibility       = "private"
			}
		`, randomID, secretValue, randomID, secretValue)

		checks := map[string]resource.TestCheckFunc{
			"before": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_dependabot_organization_secret.plaintext_secret", "plaintext_value",
					secretValue,
				),
				resource.TestCheckResourceAttr(
					"github_dependabot_organization_secret.encrypted_secret", "encrypted_value",
					secretValue,
				),
				resource.TestCheckResourceAttr(
					"github_dependabot_organization_secret.plaintext_secret", "visibility",
					"private",
				),
				resource.TestCheckResourceAttr(
					"github_dependabot_organization_secret.encrypted_secret", "visibility",
					"private",
				),
				resource.TestCheckResourceAttrSet(
					"github_dependabot_organization_secret.plaintext_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_dependabot_organization_secret.plaintext_secret", "updated_at",
				),
			),
			"after": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_dependabot_organization_secret.plaintext_secret", "plaintext_value",
					updatedSecretValue,
				),
				resource.TestCheckResourceAttr(
					"github_dependabot_organization_secret.encrypted_secret", "encrypted_value",
					updatedSecretValue,
				),
				resource.TestCheckResourceAttrSet(
					"github_dependabot_organization_secret.plaintext_secret", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_dependabot_organization_secret.plaintext_secret", "updated_at",
				),
			),
		}

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
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
			testCase(t, organization)
		})
	})

	t.Run("manages secrets with different visibility settings", func(t *testing.T) {
		secretValue := "super_secret_value"

		config := fmt.Sprintf(`
			resource "github_dependabot_organization_secret" "all_secret" {
			  secret_name      = "test_all_secret_%s"
			  plaintext_value  = "%s"
			  visibility       = "all"
			}

			resource "github_dependabot_organization_secret" "private_secret" {
			  secret_name      = "test_private_secret_%s"
			  plaintext_value  = "%s"
			  visibility       = "private"
			}
		`, randomID, secretValue, randomID, secretValue)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_dependabot_organization_secret.all_secret", "visibility",
				"all",
			),
			resource.TestCheckResourceAttr(
				"github_dependabot_organization_secret.private_secret", "visibility",
				"private",
			),
			resource.TestCheckResourceAttr(
				"github_dependabot_organization_secret.all_secret", "plaintext_value",
				secretValue,
			),
			resource.TestCheckResourceAttr(
				"github_dependabot_organization_secret.private_secret", "plaintext_value",
				secretValue,
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
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

	t.Run("deletes secrets without error", func(t *testing.T) {
		config := fmt.Sprintf(`
				resource "github_dependabot_organization_secret" "plaintext_secret" {
					secret_name      = "test_plaintext_secret_%s"
					plaintext_value  = "super_secret_value"
					visibility       = "private"
				}

				resource "github_dependabot_organization_secret" "encrypted_secret" {
					secret_name      = "test_encrypted_secret_%s"
					encrypted_value  = "%s"
					visibility       = "private"
				}
			`, randomID, randomID, base64.StdEncoding.EncodeToString([]byte("super_secret_value")))

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
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

	t.Run("imports secrets without error", func(t *testing.T) {
		secretValue := "super_secret_value"
		secretName := fmt.Sprintf("test_plaintext_secret_%s", randomID)

		config := fmt.Sprintf(`
			resource "github_dependabot_organization_secret" "test_secret" {
				secret_name      = "%s"
				plaintext_value  = "%s"
				visibility       = "private"
			}
		`, secretName, secretValue)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_dependabot_organization_secret.test_secret", "secret_name",
				secretName,
			),
			resource.TestCheckResourceAttr(
				"github_dependabot_organization_secret.test_secret", "visibility",
				"private",
			),
			resource.TestCheckResourceAttrSet(
				"github_dependabot_organization_secret.test_secret", "created_at",
			),
			resource.TestCheckResourceAttrSet(
				"github_dependabot_organization_secret.test_secret", "updated_at",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
					{
						ResourceName:            "github_dependabot_organization_secret.test_secret",
						ImportState:             true,
						ImportStateId:           secretName,
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
			testCase(t, organization)
		})
	})

	t.Run("handles invalid secret names", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_dependabot_organization_secret" "invalid_name" {
			  secret_name      = "1invalid_name_%s"
			  plaintext_value  = "super_secret_value"
			  visibility       = "private"
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("Invalid Secret Name"),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("handles conflicting value fields", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_dependabot_organization_secret" "conflicting" {
			  secret_name      = "test_conflicting_%s"
			  plaintext_value  = "super_secret_value"
			  encrypted_value  = "c3VwZXJfc2VjcmV0X3ZhbHVl"
			  visibility       = "private"
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("Conflicting Attribute Configuration"),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("validates visibility and selected repositories relationship", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_dependabot_organization_secret" "invalid_selected_repos" {
			  secret_name            = "test_invalid_selected_%s"
			  plaintext_value        = "super_secret_value"
			  visibility             = "private"
			  selected_repository_ids = [12345]
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("selected_repository_ids can only be set when visibility is 'selected'"),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubDependabotOrganizationSecret_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	secretValue := "super_secret_value"
	secretName := fmt.Sprintf("test_migration_secret_%s", randomID)

	config := fmt.Sprintf(`
		resource "github_dependabot_organization_secret" "test" {
			secret_name      = "%s"
			plaintext_value  = "%s"
			visibility       = "private"
		}
	`, secretName, secretValue)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"github": {
						Source:            "integrations/github",
						VersionConstraint: "~> 6.0",
					},
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_dependabot_organization_secret.test", "secret_name", secretName),
					resource.TestCheckResourceAttr("github_dependabot_organization_secret.test", "visibility", "private"),
					resource.TestCheckResourceAttrSet("github_dependabot_organization_secret.test", "created_at"),
					resource.TestCheckResourceAttrSet("github_dependabot_organization_secret.test", "updated_at"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Config:                   config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}
