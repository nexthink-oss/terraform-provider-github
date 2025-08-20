package github

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubEnterpriseActionsRunnerGroup(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	if os.Getenv("ENTERPRISE_ACCOUNT") != "true" {
		t.Skip("Skipping because `ENTERPRISE_ACCOUNT` is not set or set to false")
	}

	if os.Getenv("ENTERPRISE_SLUG") == "" {
		t.Skip("Skipping because `ENTERPRISE_SLUG` is not set")
	}

	t.Run("creates enterprise runner groups without error", func(t *testing.T) {
		config := fmt.Sprintf(`
			data "github_enterprise" "enterprise" {
				slug = "%s"
			}

			resource "github_enterprise_actions_runner_group" "test" {
				enterprise_slug				= data.github_enterprise.enterprise.slug
				name						= "tf-acc-test-%s"
				visibility					= "all"
				allows_public_repositories	= true
			}
		`, testEnterprise, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet(
				"github_enterprise_actions_runner_group.test", "name",
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "name",
				fmt.Sprintf(`tf-acc-test-%s`, randomID),
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "visibility",
				"all",
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "allows_public_repositories",
				"true",
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "restricted_to_workflows",
				"false",
			),
			resource.TestCheckResourceAttrSet(
				"github_enterprise_actions_runner_group.test", "id",
			),
			resource.TestCheckResourceAttrSet(
				"github_enterprise_actions_runner_group.test", "etag",
			),
			resource.TestCheckResourceAttrSet(
				"github_enterprise_actions_runner_group.test", "runners_url",
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "default",
				"false",
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

	t.Run("manages runner group visibility to selected orgs", func(t *testing.T) {

		config := fmt.Sprintf(`
			data "github_enterprise" "enterprise" {
				slug = "%s"
			}

			data "github_organization" "org" {
				name 			= "%s"
			}

			resource "github_enterprise_actions_runner_group" "test" {
				enterprise_slug				= data.github_enterprise.enterprise.slug
				name						= "tf-acc-test-%s"
				visibility					= "selected"
				selected_organization_ids	= [data.github_organization.org.id]
			}
		`, testEnterprise, testOrganization, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet(
				"github_enterprise_actions_runner_group.test", "name",
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "name",
				fmt.Sprintf(`tf-acc-test-%s`, randomID),
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "visibility",
				"selected",
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "selected_organization_ids.#",
				"1",
			),
			resource.TestCheckResourceAttrSet(
				"github_enterprise_actions_runner_group.test", "selected_organizations_url",
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "allows_public_repositories",
				"false",
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "restricted_to_workflows",
				"false",
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

	t.Run("imports an all runner group without error", func(t *testing.T) {
		config := fmt.Sprintf(`
			data "github_enterprise" "enterprise" {
				slug = "%s"
			}

			resource "github_enterprise_actions_runner_group" "test" {
				enterprise_slug = data.github_enterprise.enterprise.slug
				name       		= "tf-acc-test-%s"
				visibility 		= "all"
			}
	`, testEnterprise, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet("github_enterprise_actions_runner_group.test", "name"),
			resource.TestCheckResourceAttrSet("github_enterprise_actions_runner_group.test", "visibility"),
			resource.TestCheckResourceAttr("github_enterprise_actions_runner_group.test", "visibility", "all"),
			resource.TestCheckResourceAttr("github_enterprise_actions_runner_group.test", "name", fmt.Sprintf(`tf-acc-test-%s`, randomID)),
			resource.TestCheckResourceAttr("github_enterprise_actions_runner_group.test", "allows_public_repositories", "false"),
			resource.TestCheckResourceAttr("github_enterprise_actions_runner_group.test", "restricted_to_workflows", "false"),
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
						ResourceName:        "github_enterprise_actions_runner_group.test",
						ImportState:         true,
						ImportStateVerify:   true,
						ImportStateIdPrefix: fmt.Sprintf(`%s/`, testEnterprise),
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})

	t.Run("imports a runner group with selected orgs without error", func(t *testing.T) {

		config := fmt.Sprintf(`
			data "github_enterprise" "enterprise" {
				slug = "%s"
			}

			data "github_organization" "org" {
				name 			= "%s"
			}

			resource "github_enterprise_actions_runner_group" "test" {
				enterprise_slug				= data.github_enterprise.enterprise.slug
				name						= "tf-acc-test-%s"
				visibility					= "selected"
				selected_organization_ids	= [data.github_organization.org.id]
			}
		`, testEnterprise, testOrganization, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet("github_enterprise_actions_runner_group.test", "name"),
			resource.TestCheckResourceAttr("github_enterprise_actions_runner_group.test", "name", fmt.Sprintf(`tf-acc-test-%s`, randomID)),
			resource.TestCheckResourceAttrSet("github_enterprise_actions_runner_group.test", "visibility"),
			resource.TestCheckResourceAttr("github_enterprise_actions_runner_group.test", "visibility", "selected"),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "selected_organization_ids.#",
				"1",
			),
			resource.TestCheckResourceAttr("github_enterprise_actions_runner_group.test", "allows_public_repositories", "false"),
			resource.TestCheckResourceAttr("github_enterprise_actions_runner_group.test", "restricted_to_workflows", "false"),
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
						ResourceName:        "github_enterprise_actions_runner_group.test",
						ImportState:         true,
						ImportStateVerify:   true,
						ImportStateIdPrefix: fmt.Sprintf(`%s/`, testEnterprise),
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})

	t.Run("manages workflows restriction", func(t *testing.T) {
		config := fmt.Sprintf(`
			data "github_enterprise" "enterprise" {
				slug = "%s"
			}

			resource "github_enterprise_actions_runner_group" "test" {
				enterprise_slug				= data.github_enterprise.enterprise.slug
				name						= "tf-acc-test-%s"
				visibility					= "all"
				restricted_to_workflows		= true
				selected_workflows			= [".github/workflows/test.yml"]
			}
		`, testEnterprise, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "restricted_to_workflows",
				"true",
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "selected_workflows.#",
				"1",
			),
			resource.TestCheckResourceAttr(
				"github_enterprise_actions_runner_group.test", "selected_workflows.0",
				".github/workflows/test.yml",
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

	t.Run("handles update operations", func(t *testing.T) {
		configCreate := fmt.Sprintf(`
			data "github_enterprise" "enterprise" {
				slug = "%s"
			}

			resource "github_enterprise_actions_runner_group" "test" {
				enterprise_slug				= data.github_enterprise.enterprise.slug
				name						= "tf-acc-test-%s"
				visibility					= "all"
				allows_public_repositories	= false
			}
		`, testEnterprise, randomID)

		configUpdate := fmt.Sprintf(`
			data "github_enterprise" "enterprise" {
				slug = "%s"
			}

			resource "github_enterprise_actions_runner_group" "test" {
				enterprise_slug				= data.github_enterprise.enterprise.slug
				name						= "tf-acc-test-%s-updated"
				visibility					= "all"
				allows_public_repositories	= true
				restricted_to_workflows		= true
				selected_workflows			= [".github/workflows/ci.yml", ".github/workflows/test.yml"]
			}
		`, testEnterprise, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessEnterpriseMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: configCreate,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_enterprise_actions_runner_group.test", "name",
								fmt.Sprintf(`tf-acc-test-%s`, randomID),
							),
							resource.TestCheckResourceAttr(
								"github_enterprise_actions_runner_group.test", "allows_public_repositories",
								"false",
							),
							resource.TestCheckResourceAttr(
								"github_enterprise_actions_runner_group.test", "restricted_to_workflows",
								"false",
							),
						),
					},
					{
						Config: configUpdate,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_enterprise_actions_runner_group.test", "name",
								fmt.Sprintf(`tf-acc-test-%s-updated`, randomID),
							),
							resource.TestCheckResourceAttr(
								"github_enterprise_actions_runner_group.test", "allows_public_repositories",
								"true",
							),
							resource.TestCheckResourceAttr(
								"github_enterprise_actions_runner_group.test", "restricted_to_workflows",
								"true",
							),
							resource.TestCheckResourceAttr(
								"github_enterprise_actions_runner_group.test", "selected_workflows.#",
								"2",
							),
						),
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})
}

// Test behavioral equivalence between SDKv2 and Framework versions
func TestAccGithubEnterpriseActionsRunnerGroup_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	if os.Getenv("ENTERPRISE_ACCOUNT") != "true" {
		t.Skip("Skipping because `ENTERPRISE_ACCOUNT` is not set or set to false")
	}

	if os.Getenv("ENTERPRISE_SLUG") == "" {
		t.Skip("Skipping because `ENTERPRISE_SLUG` is not set")
	}

	config := fmt.Sprintf(`
		data "github_enterprise" "enterprise" {
			slug = "%s"
		}

		resource "github_enterprise_actions_runner_group" "test" {
			enterprise_slug				= data.github_enterprise.enterprise.slug
			name						= "tf-acc-test-%s"
			visibility					= "all"
			allows_public_repositories	= true
			restricted_to_workflows		= false
		}
	`, testEnterprise, randomID)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"github": {Source: "integrations/github", VersionConstraint: "~> 6.0"},
				},
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_enterprise_actions_runner_group.test", "name", fmt.Sprintf(`tf-acc-test-%s`, randomID)),
					resource.TestCheckResourceAttr("github_enterprise_actions_runner_group.test", "visibility", "all"),
					resource.TestCheckResourceAttr("github_enterprise_actions_runner_group.test", "allows_public_repositories", "true"),
					resource.TestCheckResourceAttrSet("github_enterprise_actions_runner_group.test", "id"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config, // Same config
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// Enterprise helper functions are defined in resource_github_enterprise_actions_permissions_test.go
