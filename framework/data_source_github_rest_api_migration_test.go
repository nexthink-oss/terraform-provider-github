package framework

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

// TestAccGithubRestApiDataSource_Migration tests behavioral compatibility between
// SDKv2 and Framework implementations during gradual migration
func TestAccGithubRestApiDataSource_Migration(t *testing.T) {

	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("migrates from SDKv2 to Framework without changes", func(t *testing.T) {

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
			  name = "tf-acc-test-%[1]s"
				auto_init = true
			}

			data "github_rest_api" "test" {
				endpoint = "repos/${github_repository.test.full_name}/git/refs/heads/${github_repository.test.default_branch}"
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestMatchResourceAttr(
				"data.github_rest_api.test", "code", regexp.MustCompile("200"),
			),
			resource.TestMatchResourceAttr(
				"data.github_rest_api.test", "status", regexp.MustCompile("200 OK"),
			),
			resource.TestMatchResourceAttr("data.github_rest_api.test", "body", regexp.MustCompile(".*refs/heads/.*")),
			resource.TestCheckResourceAttrSet("data.github_rest_api.test", "headers"),
			resource.TestCheckResourceAttrSet("data.github_rest_api.test", "id"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Create with SDKv2 provider
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"github": {
								Source:            "integrations/github",
								VersionConstraint: "~> 6.0",
							},
						},
						Config: config,
						Check:  check,
					},
					// Step 2: Migrate to Framework provider - should be no-op
					{
						ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
							"github": func() (tfprotov6.ProviderServer, error) {
								return providerserver.NewProtocol6(New())(), nil
							},
						},
						Config: config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectEmptyPlan(),
							},
						},
					},
				},
			})
		}

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("validates mux server handles both providers correctly", func(t *testing.T) {

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
			  name = "tf-acc-test-%[1]s"
				auto_init = true
			}

			data "github_rest_api" "test" {
				endpoint = "repos/${github_repository.test.full_name}/git/refs/heads/${github_repository.test.default_branch}"
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestMatchResourceAttr(
				"data.github_rest_api.test", "code", regexp.MustCompile("200"),
			),
			resource.TestMatchResourceAttr(
				"data.github_rest_api.test", "status", regexp.MustCompile("200 OK"),
			),
			resource.TestMatchResourceAttr("data.github_rest_api.test", "body", regexp.MustCompile(".*refs/heads/.*")),
			resource.TestCheckResourceAttrSet("data.github_rest_api.test", "headers"),
			resource.TestCheckResourceAttrSet("data.github_rest_api.test", "id"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("validates 404 handling compatibility", func(t *testing.T) {

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
			  name = "tf-acc-test-%[1]s"
				auto_init = true
			}

			data "github_rest_api" "test" {
				endpoint = "repos/${github_repository.test.full_name}/git/refs/heads/nonexistent-branch"
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestMatchResourceAttr(
				"data.github_rest_api.test", "code", regexp.MustCompile("404"),
			),
			resource.TestMatchResourceAttr(
				"data.github_rest_api.test", "status", regexp.MustCompile("404 Not Found"),
			),
			resource.TestCheckResourceAttrSet("data.github_rest_api.test", "body"),
			resource.TestCheckResourceAttrSet("data.github_rest_api.test", "headers"),
			resource.TestCheckResourceAttrSet("data.github_rest_api.test", "id"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					// Step 1: Test with SDKv2 provider
					{
						ExternalProviders: map[string]resource.ExternalProvider{
							"github": {
								Source:            "integrations/github",
								VersionConstraint: "~> 6.0",
							},
						},
						Config: config,
						Check:  check,
					},
					// Step 2: Test with Framework provider - should have same behavior
					{
						ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
							"github": func() (tfprotov6.ProviderServer, error) {
								return providerserver.NewProtocol6(New())(), nil
							},
						},
						Config: config,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
