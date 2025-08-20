package framework

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubEnterpriseActionsPermissions(t *testing.T) {

	t.Run("test setting of basic actions enterprise permissions", func(t *testing.T) {
		allowedActions := "local_only"
		enabledOrganizations := "all"

		config := fmt.Sprintf(`
			resource "github_enterprise_actions_permissions" "test" {
				enterprise_slug = "%s"
				allowed_actions = "%s"
				enabled_organizations = "%s"
			}
		`, testEnterprise(), allowedActions, enabledOrganizations)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_permissions.test", "allowed_actions", allowedActions,
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_permissions.test", "enabled_organizations", enabledOrganizations,
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessEnterpriseMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})

	t.Run("imports entire set of github action enterprise permissions without error", func(t *testing.T) {
		allowedActions := "selected"
		enabledOrganizations := "selected"
		githubOwnedAllowed := true
		verifiedAllowed := true
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		orgName := fmt.Sprintf("tf-acc-test-displayname%s", randomID)

		displayName := fmt.Sprintf("Tf Acc Test displayname %s", randomID)
		desc := "Initial org description"

		config := fmt.Sprintf(`
			data "github_user" "current" {
				username = ""
			}
	
			resource "github_enterprise_organization" "org" {
				enterprise_slug = "%s"
				name            = "%s"
				display_name    = "%s"
				description     = "%s"
				billing_email   = data.github_user.current.email
				admin_logins    = [
					data.github_user.current.login
				]
			}

			resource "github_enterprise_actions_permissions" "test" {
				enterprise_slug = "%s"
				allowed_actions = "%s"
				enabled_organizations = "%s"
				allowed_actions_config = [{
					github_owned_allowed = %t
					patterns_allowed     = ["actions/cache@*", "actions/checkout@*"]
					verified_allowed     = %t
				}]
				enabled_organizations_config = [{
					organization_ids       = [github_enterprise_organization.org.id]
				}]
			}
		`, testEnterprise(), orgName, displayName, desc, testEnterprise(), allowedActions, enabledOrganizations, githubOwnedAllowed, verifiedAllowed)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_permissions.test", "allowed_actions", allowedActions,
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_permissions.test", "enabled_organizations", enabledOrganizations,
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_permissions.test", "allowed_actions_config.#", "1",
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_permissions.test", "enabled_organizations_config.#", "1",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessEnterpriseMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
					{
						ResourceName:      "github_enterprise_actions_permissions.test",
						ImportState:       true,
						ImportStateVerify: true,
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})

	t.Run("test setting of enterprise allowed actions", func(t *testing.T) {
		allowedActions := "selected"
		enabledOrganizations := "all"
		githubOwnedAllowed := true
		verifiedAllowed := true

		config := fmt.Sprintf(`
			resource "github_enterprise_actions_permissions" "test" {
				enterprise_slug = "%s"
				allowed_actions = "%s"
				enabled_organizations = "%s"
				allowed_actions_config = [{
					github_owned_allowed = %t
					patterns_allowed     = ["actions/cache@*", "actions/checkout@*"]
					verified_allowed     = %t
				}]
			}
		`, testEnterprise(), allowedActions, enabledOrganizations, githubOwnedAllowed, verifiedAllowed)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_permissions.test", "allowed_actions", allowedActions,
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_permissions.test", "enabled_organizations", enabledOrganizations,
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_permissions.test", "allowed_actions_config.#", "1",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessEnterpriseMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})

	t.Run("test setting of enterprise enabled organizations", func(t *testing.T) {
		allowedActions := "all"
		enabledOrganizations := "selected"
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		randomID2 := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		orgName := fmt.Sprintf("tf-acc-test-displayname%s", randomID)
		orgName2 := fmt.Sprintf("tf-acc-test-displayname%s", randomID2)

		displayName := fmt.Sprintf("Tf Acc Test displayname %s", randomID)
		displayName2 := fmt.Sprintf("Tf Acc Test displayname %s", randomID2)

		desc := fmt.Sprintf("Initial org description %s", randomID)
		desc2 := fmt.Sprintf("Initial org description %s", randomID2)

		config := fmt.Sprintf(`
			data "github_user" "current" {
				username = ""
			}
			resource "github_enterprise_organization" "org" {
				enterprise_slug = "%s"
				name            = "%s"
				display_name    = "%s"
				description     = "%s"
				billing_email   = data.github_user.current.email
				admin_logins    = [
					data.github_user.current.login
				]
			}
			resource "github_enterprise_organization" "org2" {
				enterprise_slug = "%s"
				name            = "%s"
				display_name    = "%s"
				description     = "%s"
				billing_email   = data.github_user.current.email
				admin_logins    = [
					data.github_user.current.login
				]
			}
			resource "github_enterprise_actions_permissions" "test" {
				enterprise_slug = "%s"
				allowed_actions = "%s"
				enabled_organizations = "%s"
				enabled_organizations_config = [{
					organization_ids       = [github_enterprise_organization.org.id, github_enterprise_organization.org2.id]
				}]
			}
		`, testEnterprise(), orgName, displayName, desc, testEnterprise(), orgName2, displayName2, desc2, testEnterprise(), allowedActions, enabledOrganizations)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_permissions.test", "allowed_actions", allowedActions,
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_permissions.test", "enabled_organizations", enabledOrganizations,
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_permissions.test", "enabled_organizations_config.#", "1",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessEnterpriseMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})

	t.Run("test behavioral equivalence with SDKv2", func(t *testing.T) {
		allowedActions := "local_only"
		enabledOrganizations := "all"

		config := fmt.Sprintf(`
			resource "github_enterprise_actions_permissions" "test" {
				enterprise_slug = "%s"
				allowed_actions = "%s"
				enabled_organizations = "%s"
			}
		`, testEnterprise(), allowedActions, enabledOrganizations)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessEnterpriseMode(t, mode) },
				Steps: []resource.TestStep{
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"github": {
								Source:            "integrations/github",
								VersionConstraint: "~> 6.0", // SDKv2 version
							},
						},
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("github_enterprise_actions_permissions.test", "allowed_actions", allowedActions),
							resource.TestCheckResourceAttr("github_enterprise_actions_permissions.test", "enabled_organizations", enabledOrganizations),
							resource.TestCheckResourceAttrSet("github_enterprise_actions_permissions.test", "id"),
						),
					},
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories, // Framework version
						Config:                   config,                          // Same config
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectEmptyPlan(), // Should be no-op
							},
						},
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})
}

// Helper functions specific to enterprise testing

const enterprise = "enterprise"

func skipUnlessEnterpriseMode(t *testing.T, mode string) {
	switch mode {
	case enterprise:
		testAccPreCheckEnterprise(t)
	default:
		t.Fatalf("Unknown test mode: %s", mode)
	}
}

func testAccPreCheckEnterprise(t *testing.T) {
	if err := os.Getenv("GITHUB_TOKEN"); err == "" {
		t.Skip("GITHUB_TOKEN must be set for acceptance tests")
	}
	if err := os.Getenv("ENTERPRISE_SLUG"); err == "" {
		t.Skip("ENTERPRISE_SLUG must be set for enterprise acceptance tests")
	}
	if err := os.Getenv("ENTERPRISE_ACCOUNT"); err != "true" {
		t.Skip("ENTERPRISE_ACCOUNT must be set to 'true' for enterprise acceptance tests")
	}
}

func testEnterprise() string {
	return os.Getenv("ENTERPRISE_SLUG")
}
