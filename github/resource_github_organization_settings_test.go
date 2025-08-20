package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubOrganizationSettings(t *testing.T) {
	t.Run("creates organization settings without error", func(t *testing.T) {

		config := `
		resource "github_organization_settings" "test" {
			billing_email = "test@example.com"
			company = "Test Company"
			blog = "https://example.com"
			email = "test@example.com"
			twitter_username = "Test"
			location = "Test Location"
			name = "Test Name"
			description = "Test Description"
			has_organization_projects = true
			has_repository_projects = true
			default_repository_permission = "read"
			members_can_create_repositories = true
			members_can_create_public_repositories = true
			members_can_create_private_repositories = true
			members_can_create_internal_repositories = false
			members_can_create_pages = true
			members_can_create_public_pages = true
			members_can_create_private_pages = true
			members_can_fork_private_repositories = true
			web_commit_signoff_required = true
			advanced_security_enabled_for_new_repositories = false
			dependabot_alerts_enabled_for_new_repositories = false
			dependabot_security_updates_enabled_for_new_repositories = false
			dependency_graph_enabled_for_new_repositories = false
			secret_scanning_enabled_for_new_repositories = false
			secret_scanning_push_protection_enabled_for_new_repositories = false
		}`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"billing_email", "test@example.com",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"company", "Test Company",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"blog", "https://example.com",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"email", "test@example.com",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"twitter_username", "Test",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"location", "Test Location",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"name", "Test Name",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"description", "Test Description",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"has_organization_projects", "true",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"has_repository_projects", "true",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"default_repository_permission", "read",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"members_can_create_repositories", "true",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"members_can_create_public_repositories", "true",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"members_can_create_private_repositories", "true",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"members_can_create_internal_repositories", "false",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"members_can_create_pages", "true",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"members_can_create_public_pages", "true",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"members_can_create_private_pages", "true",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"members_can_fork_private_repositories", "true",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"web_commit_signoff_required", "true",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"advanced_security_enabled_for_new_repositories", "false",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"dependabot_alerts_enabled_for_new_repositories", "false",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"dependabot_security_updates_enabled_for_new_repositories", "false",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"dependency_graph_enabled_for_new_repositories", "false",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"secret_scanning_enabled_for_new_repositories", "false",
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"secret_scanning_push_protection_enabled_for_new_repositories", "false",
			),
			resource.TestCheckResourceAttrSet(
				"github_organization_settings.test",
				"id",
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

		t.Run("run with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})
		t.Run("run with an individual account", func(t *testing.T) {
			t.Skip("individual account not supported for this operation")
		})
		t.Run("run with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("updates organization settings without error", func(t *testing.T) {
		billingEmail := "test1@example.com"
		company := "Test Company"
		blog := "https://test.com"
		updatedBillingEmail := "test2@example.com"
		updatedCompany := "Test Company 2"
		updatedBlog := "https://test2.com"

		configs := map[string]string{
			"before": fmt.Sprintf(`
			resource "github_organization_settings" "test" {
				billing_email = "%s"
				company = "%s"
				blog = "%s"
			}`, billingEmail, company, blog),

			"after": fmt.Sprintf(`
			resource "github_organization_settings" "test" {
				billing_email = "%s"
				company = "%s"
				blog = "%s"
			}`, updatedBillingEmail, updatedCompany, updatedBlog),
		}

		checks := map[string]resource.TestCheckFunc{
			"before": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_organization_settings.test",
					"billing_email", billingEmail,
				),
				resource.TestCheckResourceAttr(
					"github_organization_settings.test",
					"company", company,
				),
				resource.TestCheckResourceAttr(
					"github_organization_settings.test",
					"blog", blog,
				),
				resource.TestCheckResourceAttrSet(
					"github_organization_settings.test",
					"id",
				),
			),
			"after": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_organization_settings.test",
					"billing_email", updatedBillingEmail,
				),
				resource.TestCheckResourceAttr(
					"github_organization_settings.test",
					"company", updatedCompany,
				),
				resource.TestCheckResourceAttr(
					"github_organization_settings.test",
					"blog", updatedBlog,
				),
				resource.TestCheckResourceAttrSet(
					"github_organization_settings.test",
					"id",
				),
			),
		}

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: configs["before"],
						Check:  checks["before"],
					},
					{
						Config: configs["after"],
						Check:  checks["after"],
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

	t.Run("imports organization settings without error", func(t *testing.T) {
		billingEmail := "test@example.com"
		company := "Test Company"
		blog := "https://example.com"

		config := fmt.Sprintf(`
		resource "github_organization_settings" "test" {
			billing_email = "%s"
			company = "%s"
			blog = "%s"
		}`, billingEmail, company, blog)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"billing_email", billingEmail,
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"company", company,
			),
			resource.TestCheckResourceAttr(
				"github_organization_settings.test",
				"blog", blog,
			),
			resource.TestCheckResourceAttrSet(
				"github_organization_settings.test",
				"id",
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
						ResourceName:      "github_organization_settings.test",
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("validates default_repository_permission values", func(t *testing.T) {
		validValues := []string{"read", "write", "admin", "none"}

		for _, value := range validValues {
			t.Run(fmt.Sprintf("accepts_%s", value), func(t *testing.T) {
				config := fmt.Sprintf(`
				resource "github_organization_settings" "test" {
					billing_email = "test@example.com"
					default_repository_permission = "%s"
				}`, value)

				resource.Test(t, resource.TestCase{
					PreCheck:                 func() { skipUnlessMode(t, organization) },
					ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
					Steps: []resource.TestStep{
						{
							Config: config,
							Check: resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"default_repository_permission", value,
							),
						},
					},
				})
			})
		}
	})

	t.Run("plan only tests for computed defaults", func(t *testing.T) {
		config := `
		resource "github_organization_settings" "test" {
			billing_email = "test@example.com"
		}`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectResourceAction("github_organization_settings.test", plancheck.ResourceActionCreate),
							},
						},
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"billing_email", "test@example.com",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"has_organization_projects", "true",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"has_repository_projects", "true",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"default_repository_permission", "read",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"members_can_create_repositories", "true",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"members_can_create_private_repositories", "true",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"members_can_create_public_repositories", "true",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"members_can_create_pages", "true",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"members_can_create_public_pages", "true",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"members_can_create_private_pages", "true",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"members_can_fork_private_repositories", "false",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"web_commit_signoff_required", "false",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"advanced_security_enabled_for_new_repositories", "false",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"dependabot_alerts_enabled_for_new_repositories", "false",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"dependabot_security_updates_enabled_for_new_repositories", "false",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"dependency_graph_enabled_for_new_repositories", "false",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"secret_scanning_enabled_for_new_repositories", "false",
							),
							resource.TestCheckResourceAttr(
								"github_organization_settings.test",
								"secret_scanning_push_protection_enabled_for_new_repositories", "false",
							),
							resource.TestCheckResourceAttrSet(
								"github_organization_settings.test",
								"id",
							),
						),
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
