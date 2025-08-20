package github

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubActionsOrganizationVariableResource_basic(t *testing.T) {
	t.Run("creates and updates a private organization variable without error", func(t *testing.T) {
		value := "my_variable_value"
		updatedValue := "my_updated_variable_value"

		config := fmt.Sprintf(`
			resource "github_actions_organization_variable" "variable" {
			  variable_name    = "test_variable"
			  value  		   = "%s"
			  visibility       = "private"
			}
			`, value)

		checks := map[string]resource.TestCheckFunc{
			"before": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_actions_organization_variable.variable", "value",
					value,
				),
				resource.TestCheckResourceAttr(
					"github_actions_organization_variable.variable", "visibility",
					"private",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_organization_variable.variable", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_organization_variable.variable", "updated_at",
				),
			),
			"after": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_actions_organization_variable.variable", "value",
					updatedValue,
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_organization_variable.variable", "created_at",
				),
				resource.TestCheckResourceAttrSet(
					"github_actions_organization_variable.variable", "updated_at",
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
}

func TestAccGithubActionsOrganizationVariableResource_selected(t *testing.T) {
	t.Run("creates an organization variable scoped to a repo without error", func(t *testing.T) {
		value := "my_variable_value"
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%s"
			}

			resource "github_actions_organization_variable" "variable" {
			  variable_name    = "test_variable"
			  value  		   = "%s"
			  visibility       = "selected"
			  selected_repository_ids = [github_repository.test.repo_id]
			}
			`, randomID, value)

		checks := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_actions_organization_variable.variable", "value",
				value,
			),
			resource.TestCheckResourceAttr(
				"github_actions_organization_variable.variable", "visibility",
				"selected",
			),
			resource.TestCheckResourceAttr(
				"github_actions_organization_variable.variable", "selected_repository_ids.#",
				"1",
			),
			resource.TestCheckResourceAttrSet(
				"github_actions_organization_variable.variable", "created_at",
			),
			resource.TestCheckResourceAttrSet(
				"github_actions_organization_variable.variable", "updated_at",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  checks,
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

func TestAccGithubActionsOrganizationVariableResource_all(t *testing.T) {
	t.Run("creates an organization variable with 'all' visibility without error", func(t *testing.T) {
		value := "my_variable_value"

		config := fmt.Sprintf(`
			resource "github_actions_organization_variable" "variable" {
			  variable_name    = "test_variable_all"
			  value  		   = "%s"
			  visibility       = "all"
			}
			`, value)

		checks := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_actions_organization_variable.variable", "value",
				value,
			),
			resource.TestCheckResourceAttr(
				"github_actions_organization_variable.variable", "visibility",
				"all",
			),
			resource.TestCheckResourceAttr(
				"github_actions_organization_variable.variable", "selected_repository_ids.#",
				"0",
			),
			resource.TestCheckResourceAttrSet(
				"github_actions_organization_variable.variable", "created_at",
			),
			resource.TestCheckResourceAttrSet(
				"github_actions_organization_variable.variable", "updated_at",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  checks,
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

func TestAccGithubActionsOrganizationVariableResource_import(t *testing.T) {
	t.Run("imports organization variable without error", func(t *testing.T) {
		value := "my_variable_value"
		variableName := "test_variable_import"

		config := fmt.Sprintf(`
			resource "github_actions_organization_variable" "variable" {
			  variable_name    = "%s"
			  value  		   = "%s"
			  visibility       = "private"
			}
			`, variableName, value)

		checks := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_actions_organization_variable.variable", "variable_name",
				variableName,
			),
			resource.TestCheckResourceAttr(
				"github_actions_organization_variable.variable", "value",
				value,
			),
			resource.TestCheckResourceAttr(
				"github_actions_organization_variable.variable", "visibility",
				"private",
			),
			resource.TestCheckResourceAttrSet(
				"github_actions_organization_variable.variable", "created_at",
			),
			resource.TestCheckResourceAttrSet(
				"github_actions_organization_variable.variable", "updated_at",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  checks,
					},
					{
						ResourceName:      "github_actions_organization_variable.variable",
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

func TestAccGithubActionsOrganizationVariableResource_validation(t *testing.T) {
	t.Run("validates variable name requirements", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t, individual) },
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: `
						resource "github_actions_organization_variable" "variable" {
							variable_name = "123invalid"
							value         = "test_value"
							visibility    = "private"
						}
					`,
					ExpectError: regexp.MustCompile("Variable names can only contain alphanumeric characters or underscores and must not start with a number"),
				},
				{
					Config: `
						resource "github_actions_organization_variable" "variable" {
							variable_name = "GITHUB_invalid"
							value         = "test_value"
							visibility    = "private"
						}
					`,
					ExpectError: regexp.MustCompile("Variable names must not start with the GITHUB_ prefix"),
				},
			},
		})
	})

	t.Run("validates visibility requirements", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t, individual) },
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: `
						resource "github_actions_organization_variable" "variable" {
							variable_name = "test_variable"
							value         = "test_value"
							visibility    = "invalid"
						}
					`,
					ExpectError: regexp.MustCompile("Value must be one of: \\[all private selected\\]"),
				},
			},
		})
	})

	t.Run("validates selected_repository_ids with visibility", func(t *testing.T) {
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

						resource "github_actions_organization_variable" "variable" {
							variable_name           = "test_variable"
							value                   = "test_value"
							visibility              = "private"
							selected_repository_ids = [github_repository.test.repo_id]
						}
					`, randomID),
					ExpectError: regexp.MustCompile("selected_repository_ids can only be set when visibility is 'selected'"),
				},
			},
		})
	})
}
