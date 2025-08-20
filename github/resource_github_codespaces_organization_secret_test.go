package github

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubCodespacesOrganizationSecret_Basic(t *testing.T) {
	testCheckOrganizationAccount(t)

	secretValue := "super_secret_value"
	updatedSecretValue := "updated_super_secret_value"
	secretName := "test_plaintext_secret"

	config := fmt.Sprintf(`
		resource "github_codespaces_organization_secret" "test_secret" {
			secret_name      = "%s"
			plaintext_value  = "%s"
			visibility       = "private"
		}
	`, secretName, secretValue)

	updatedConfig := fmt.Sprintf(`
		resource "github_codespaces_organization_secret" "test_secret" {
			secret_name      = "%s"
			plaintext_value  = "%s"
			visibility       = "private"
		}
	`, secretName, updatedSecretValue)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "secret_name",
						secretName,
					),
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "plaintext_value",
						secretValue,
					),
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "visibility",
						"private",
					),
					resource.TestCheckResourceAttrSet(
						"github_codespaces_organization_secret.test_secret", "created_at",
					),
					resource.TestCheckResourceAttrSet(
						"github_codespaces_organization_secret.test_secret", "updated_at",
					),
				),
			},
			{
				Config: updatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "secret_name",
						secretName,
					),
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "plaintext_value",
						updatedSecretValue,
					),
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "visibility",
						"private",
					),
				),
			},
		},
	})
}

func TestAccGithubCodespacesOrganizationSecret_Encrypted(t *testing.T) {
	testCheckOrganizationAccount(t)

	secretValue := base64.StdEncoding.EncodeToString([]byte("super_secret_value"))
	secretName := "test_encrypted_secret"

	config := fmt.Sprintf(`
		resource "github_codespaces_organization_secret" "test_secret" {
			secret_name      = "%s"
			encrypted_value  = "%s"
			visibility       = "private"
		}
	`, secretName, secretValue)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "secret_name",
						secretName,
					),
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "encrypted_value",
						secretValue,
					),
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "visibility",
						"private",
					),
					resource.TestCheckResourceAttrSet(
						"github_codespaces_organization_secret.test_secret", "created_at",
					),
					resource.TestCheckResourceAttrSet(
						"github_codespaces_organization_secret.test_secret", "updated_at",
					),
				),
			},
		},
	})
}

func TestAccGithubCodespacesOrganizationSecret_Visibility_All(t *testing.T) {
	testCheckOrganizationAccount(t)

	secretValue := "super_secret_value"
	secretName := "test_visibility_all_secret"

	config := fmt.Sprintf(`
		resource "github_codespaces_organization_secret" "test_secret" {
			secret_name      = "%s"
			plaintext_value  = "%s"
			visibility       = "all"
		}
	`, secretName, secretValue)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "visibility",
						"all",
					),
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "selected_repository_ids.#",
						"0",
					),
				),
			},
		},
	})
}

func TestAccGithubCodespacesOrganizationSecret_Visibility_Selected(t *testing.T) {
	testCheckOrganizationAccount(t)

	secretValue := "super_secret_value"
	secretName := "test_visibility_selected_secret"
	repoName := "terraform-provider-github-test-repo"

	config := fmt.Sprintf(`
		data "github_repository" "test_repo" {
			name = "%s"
		}

		resource "github_codespaces_organization_secret" "test_secret" {
			secret_name             = "%s"
			plaintext_value         = "%s"
			visibility              = "selected"
			selected_repository_ids = [data.github_repository.test_repo.repo_id]
		}
	`, repoName, secretName, secretValue)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "visibility",
						"selected",
					),
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "selected_repository_ids.#",
						"1",
					),
				),
			},
		},
	})
}

func TestAccGithubCodespacesOrganizationSecret_Import(t *testing.T) {
	testCheckOrganizationAccount(t)

	secretValue := "super_secret_value"
	secretName := "test_import_secret"

	config := fmt.Sprintf(`
		resource "github_codespaces_organization_secret" "test_secret" {
			secret_name      = "%s"
			plaintext_value  = "%s"
			visibility       = "private"
		}
	`, secretName, secretValue)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "secret_name",
						secretName,
					),
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "visibility",
						"private",
					),
				),
			},
			{
				ResourceName:            "github_codespaces_organization_secret.test_secret",
				ImportState:             true,
				ImportStateId:           secretName,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"plaintext_value", "encrypted_value"},
			},
		},
	})
}

func TestAccGithubCodespacesOrganizationSecret_ValidationErrors(t *testing.T) {
	testCheckOrganizationAccount(t)

	t.Run("Both plaintext_value and encrypted_value set", func(t *testing.T) {
		config := `
			resource "github_codespaces_organization_secret" "test_secret" {
				secret_name      = "test_validation_secret"
				plaintext_value  = "secret"
				encrypted_value  = "ZW5jcnlwdGVkX3ZhbHVl"
				visibility       = "private"
			}
		`

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile("Conflicting Attribute Configuration"),
				},
			},
		})
	})

	t.Run("Invalid secret name", func(t *testing.T) {
		config := `
			resource "github_codespaces_organization_secret" "test_secret" {
				secret_name      = "GITHUB_INVALID_NAME"
				plaintext_value  = "secret"
				visibility       = "private"
			}
		`

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile("Secret names must not start with the GITHUB_ prefix"),
				},
			},
		})
	})

	t.Run("Invalid visibility value", func(t *testing.T) {
		config := `
			resource "github_codespaces_organization_secret" "test_secret" {
				secret_name      = "test_validation_secret"
				plaintext_value  = "secret"
				visibility       = "invalid"
			}
		`

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile("Invalid Visibility Value"),
				},
			},
		})
	})

	t.Run("Selected repository IDs without selected visibility", func(t *testing.T) {
		config := `
			resource "github_codespaces_organization_secret" "test_secret" {
				secret_name             = "test_validation_secret"
				plaintext_value         = "secret"
				visibility              = "private"
				selected_repository_ids = [123456]
			}
		`

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile("selected_repository_ids can only be set when visibility is 'selected'"),
				},
			},
		})
	})

	t.Run("Invalid base64 encrypted value", func(t *testing.T) {
		config := `
			resource "github_codespaces_organization_secret" "test_secret" {
				secret_name      = "test_validation_secret"
				encrypted_value  = "not-valid-base64!"
				visibility       = "private"
			}
		`

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile("Invalid Base64 Value"),
				},
			},
		})
	})
}

func TestAccGithubCodespacesOrganizationSecret_Update_RequiresReplace(t *testing.T) {
	testCheckOrganizationAccount(t)

	secretName1 := "test_replace_secret_1"
	secretName2 := "test_replace_secret_2"
	secretValue := "super_secret_value"

	config1 := fmt.Sprintf(`
		resource "github_codespaces_organization_secret" "test_secret" {
			secret_name      = "%s"
			plaintext_value  = "%s"
			visibility       = "private"
		}
	`, secretName1, secretValue)

	config2 := fmt.Sprintf(`
		resource "github_codespaces_organization_secret" "test_secret" {
			secret_name      = "%s"
			plaintext_value  = "%s"
			visibility       = "private"
		}
	`, secretName2, secretValue)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "secret_name",
						secretName1,
					),
				),
			},
			{
				Config: config2,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("github_codespaces_organization_secret.test_secret", plancheck.ResourceActionDestroyBeforeCreate),
					},
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(
						"github_codespaces_organization_secret.test_secret", "secret_name",
						secretName2,
					),
				),
			},
		},
	})
}

func TestAccGithubCodespacesOrganizationSecret_MissingSecretValue(t *testing.T) {
	testCheckOrganizationAccount(t)

	config := `
		resource "github_codespaces_organization_secret" "test_secret" {
			secret_name      = "test_missing_value_secret"
			visibility       = "private"
		}
	`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile("Missing Secret Value"),
			},
		},
	})
}

// Helper function for account type checking
func testCheckOrganizationAccount(t *testing.T) {
	testAccPreCheckOrganization(t)
}
