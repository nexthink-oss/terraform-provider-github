package github

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubEnterpriseOrganization(t *testing.T) {

	t.Run("creates and updates an enterprise organization without error", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		orgName := fmt.Sprintf("tf-acc-test-%s", randomID)

		desc := "Initial org description"
		updatedDesc := "Updated org description"

		config := fmt.Sprintf(`
		  data "github_enterprise" "enterprise" {
			slug = "%s"
		  }

		  data "github_user" "current" {
			username = ""
		  }

		  resource "github_enterprise_organization" "org" {
			enterprise_id = data.github_enterprise.enterprise.id
			name          = "%s"
			description   = "%s"
			billing_email = data.github_user.current.email
			admin_logins  = [
			  data.github_user.current.login
			]
		  }
			`, testEnterprise, orgName, desc)

		checks := map[string]resource.TestCheckFunc{
			"before": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet(
					"github_enterprise_organization.org", "enterprise_id",
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "name",
					orgName,
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "description",
					desc,
				),
				resource.TestCheckResourceAttrSet(
					"github_enterprise_organization.org", "billing_email",
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "admin_logins.#",
					"1",
				),
			),
			"after": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "description",
					updatedDesc,
				),
			),
		}

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessEnterpriseMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  checks["before"],
					},
					{
						Config: strings.Replace(config,
							desc,
							updatedDesc, 1),
						Check: checks["after"],
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})

	t.Run("deletes an enterprise organization without error", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		orgName := fmt.Sprintf("tf-acc-test-%s", randomID)

		config := fmt.Sprintf(`
		  data "github_enterprise" "enterprise" {
			slug = "%s"
		  }

		  data "github_user" "current" {
			username = ""
		  }

		  resource "github_enterprise_organization" "org" {
			enterprise_id = data.github_enterprise.enterprise.id
			name          = "%s"
			billing_email = data.github_user.current.email
			admin_logins  = [
			  data.github_user.current.login
			]
		  }
			`, testEnterprise, orgName)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessEnterpriseMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:  config,
						Destroy: true,
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})

	t.Run("creates and updates org with display name", func(t *testing.T) {

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		orgName := fmt.Sprintf("tf-acc-test-displayname%s", randomID)

		displayName := fmt.Sprintf("Tf Acc Test displayname %s", randomID)
		updatedDisplayName := fmt.Sprintf("Updated Tf Acc Test Display Name %s", randomID)

		desc := "Initial org description"
		updatedDesc := "Updated org description"

		config := fmt.Sprintf(`
		  data "github_enterprise" "enterprise" {
			slug = "%s"
		  }

		  data "github_user" "current" {
			username = ""
		  }

		  resource "github_enterprise_organization" "org" {
			enterprise_id = data.github_enterprise.enterprise.id
			name          = "%s"
			display_name  = "%s"
			description   = "%s"
			billing_email = data.github_user.current.email
			admin_logins  = [
			  data.github_user.current.login
			]
		  }
			`, testEnterprise, orgName, displayName, desc)

		checks := map[string]resource.TestCheckFunc{
			"before": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet(
					"github_enterprise_organization.org", "enterprise_id",
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "name",
					orgName,
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "display_name",
					displayName,
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "description",
					desc,
				),
				resource.TestCheckResourceAttrSet(
					"github_enterprise_organization.org", "billing_email",
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "admin_logins.#",
					"1",
				),
			),
			"after": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "description",
					updatedDesc,
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "display_name",
					updatedDisplayName,
				),
			),
		}

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessEnterpriseMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  checks["before"],
					},
					{
						Config: strings.Replace(
							strings.Replace(config,
								displayName,
								updatedDisplayName, 1),
							desc,
							updatedDesc, 1),
						Check: checks["after"],
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})

	})

	t.Run("creates org without display name, set and update display name", func(t *testing.T) {

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		orgName := fmt.Sprintf("tf-acc-test-adddisplayname%s", randomID)

		displayName := fmt.Sprintf("Tf Acc Test Add displayname %s", randomID)
		updatedDisplayName := fmt.Sprintf("Updated Tf Acc Test Add Display Name %s", randomID)

		desc := "Initial org description"
		updatedDesc := "Updated org description"

		configWithoutDisplayName := fmt.Sprintf(`
		  data "github_enterprise" "enterprise" {
			slug = "%s"
		  }

		  data "github_user" "current" {
			username = ""
		  }

		  resource "github_enterprise_organization" "org" {
			enterprise_id = data.github_enterprise.enterprise.id
			name          = "%s"
			description   = "%s"
			billing_email = data.github_user.current.email
			admin_logins  = [
			  data.github_user.current.login
			]
		  }
			`, testEnterprise, orgName, desc)

		configWithDisplayName := fmt.Sprintf(`
			data "github_enterprise" "enterprise" {
			  slug = "%s"
			}

			data "github_user" "current" {
			  username = ""
			}

			resource "github_enterprise_organization" "org" {
			  enterprise_id = data.github_enterprise.enterprise.id
			  name          = "%s"
			  display_name  = "%s"
			  description   = "%s"
			  billing_email = data.github_user.current.email
			  admin_logins  = [
				data.github_user.current.login
			  ]
			}
			  `, testEnterprise, orgName, displayName, desc)

		checks := map[string]resource.TestCheckFunc{
			"create": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttrSet(
					"github_enterprise_organization.org", "enterprise_id",
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "name",
					orgName,
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "description",
					desc,
				),
				resource.TestCheckResourceAttrSet(
					"github_enterprise_organization.org", "billing_email",
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "admin_logins.#",
					"1",
				),
			),
			"set": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "description",
					desc,
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "display_name",
					displayName,
				),
			),
			"updateDisplayName": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "description",
					desc,
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "display_name",
					updatedDisplayName,
				),
			),
			"updateDesc": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "description",
					updatedDesc,
				),
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "display_name",
					updatedDisplayName,
				),
			),
			"unset": resource.ComposeTestCheckFunc(
				resource.TestCheckResourceAttr(
					"github_enterprise_organization.org", "description",
					desc,
				),
			),
		}

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessEnterpriseMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: configWithoutDisplayName,
						Check:  checks["create"],
					},
					{
						Config: configWithDisplayName,
						Check:  checks["set"],
					},
					{
						Config: strings.Replace(configWithDisplayName,
							displayName,
							updatedDisplayName, 1),
						Check: checks["updateDisplayName"],
					},
					{
						Config: strings.Replace(
							strings.Replace(configWithDisplayName,
								displayName,
								updatedDisplayName, 1),
							desc,
							updatedDesc, 1),
						Check: checks["updateDesc"],
					},
					{
						Config: configWithoutDisplayName,
						Check:  checks["unset"],
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})

	})

	t.Run("imports enterprise organization without error", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		orgName := fmt.Sprintf("tf-acc-test-import%s", randomID)

		config := fmt.Sprintf(`
			data "github_enterprise" "enterprise" {
			  slug = "%s"
			}

			data "github_user" "current" {
			  username = ""
			}

			resource "github_enterprise_organization" "org" {
			  enterprise_id = data.github_enterprise.enterprise.id
			  name          = "%s"
			  billing_email = data.github_user.current.email
			  admin_logins  = [
				data.github_user.current.login
			  ]
			}
			  `, testEnterprise, orgName)

		check := resource.ComposeTestCheckFunc()

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
						ResourceName:      "github_enterprise_organization.org",
						ImportState:       true,
						ImportStateVerify: true,
						ImportStateId:     fmt.Sprintf(`%s/%s`, testEnterprise, orgName),
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})

	t.Run("imports enterprise organization invalid enterprise name", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		orgName := fmt.Sprintf("tf-acc-test-adddisplayname%s", randomID)

		config := fmt.Sprintf(`
			data "github_enterprise" "enterprise" {
			  slug = "%s"
			}

			data "github_user" "current" {
			  username = ""
			}

			resource "github_enterprise_organization" "org" {
			  enterprise_id = data.github_enterprise.enterprise.id
			  name          = "%s"
			  description   = "org description"
			  billing_email = data.github_user.current.email
			  admin_logins  = [
				data.github_user.current.login
			  ]
			}
			  `, testEnterprise, orgName)

		check := resource.ComposeTestCheckFunc()

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
						ResourceName:  "github_enterprise_organization.org",
						ImportState:   true,
						ImportStateId: fmt.Sprintf(`%s/%s`, randomID, orgName),
						ExpectError:   regexp.MustCompile("Could not resolve to a Business with the URL slug of .*"),
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})

	})

	t.Run("imports enterprise organization invalid organization name", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		orgName := fmt.Sprintf("tf-acc-test-adddisplayname%s", randomID)

		config := fmt.Sprintf(`
			data "github_enterprise" "enterprise" {
			  slug = "%s"
			}

			data "github_user" "current" {
			  username = ""
			}

			resource "github_enterprise_organization" "org" {
			  enterprise_id = data.github_enterprise.enterprise.id
			  name          = "%s"
			  description   = "org description"
			  billing_email = data.github_user.current.email
			  admin_logins  = [
				data.github_user.current.login
			  ]
			}
			  `, testEnterprise, orgName)

		check := resource.ComposeTestCheckFunc()

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
						ResourceName:  "github_enterprise_organization.org",
						ImportState:   true,
						ImportStateId: fmt.Sprintf(`%s/%s`, testEnterprise, randomID),
						ExpectError:   regexp.MustCompile("Could not resolve to an Organization with the login of .*"),
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})

	})

	// Test for migration compatibility - ensures no changes when switching from SDKv2 to Framework
	t.Run("migration compatibility test", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
		orgName := fmt.Sprintf("tf-acc-test-migration-%s", randomID)

		desc := "Migration test description"
		displayName := fmt.Sprintf("Migration Test %s", randomID)

		config := fmt.Sprintf(`
		  data "github_enterprise" "enterprise" {
			slug = "%s"
		  }

		  data "github_user" "current" {
			username = ""
		  }

		  resource "github_enterprise_organization" "org" {
			enterprise_id = data.github_enterprise.enterprise.id
			name          = "%s"
			display_name  = "%s"
			description   = "%s"
			billing_email = data.github_user.current.email
			admin_logins  = [
			  data.github_user.current.login
			]
		  }
		`, testEnterprise, orgName, displayName, desc)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessEnterpriseMode(t, mode) },
				Steps: []resource.TestStep{
					// First create the resource with the muxed provider (which uses SDKv2 for this resource initially)
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
						Config:                   config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("github_enterprise_organization.org", "name", orgName),
							resource.TestCheckResourceAttr("github_enterprise_organization.org", "description", desc),
							resource.TestCheckResourceAttr("github_enterprise_organization.org", "display_name", displayName),
							resource.TestCheckResourceAttrSet("github_enterprise_organization.org", "id"),
							resource.TestCheckResourceAttrSet("github_enterprise_organization.org", "database_id"),
						),
					},
					// Then switch to pure Framework provider - should be no-op plan
					{
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})
	})
}
