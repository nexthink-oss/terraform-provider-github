package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubRepositoryDeploymentBranchPoliciesDataSource_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("migration compatibility test", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_environment" "env" {
				repository  = github_repository.test.name
				environment = "my_env"
				deployment_branch_policy {
					protected_branches     = false
					custom_branch_policies = true
				}
			}

			resource "github_repository_deployment_branch_policy" "br" {
				repository       = github_repository.test.name
				environment_name = github_repository_environment.env.environment
				name             = "foo"
			}

			data "github_repository_deployment_branch_policies" "all" {
				repository = github_repository.test.name
				environment_name = github_repository_environment.env.environment
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					{
						// Test with SDKv2 provider (external provider would be used in real scenarios)
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
						Config:                   config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_repository_deployment_branch_policies.all", "deployment_branch_policies.#", "1"),
							resource.TestCheckResourceAttr("data.github_repository_deployment_branch_policies.all", "deployment_branch_policies.0.name", "foo"),
							resource.TestCheckResourceAttrSet("data.github_repository_deployment_branch_policies.all", "deployment_branch_policies.0.id"),
						),
					},
					{
						// Test with Framework provider - should produce no diff
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

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubRepositoryDeploymentBranchPoliciesDataSource_BehavioralEquivalence(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("behavioral equivalence between SDKv2 and Framework", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_environment" "env" {
				repository  = github_repository.test.name
				environment = "my_env"
				deployment_branch_policy {
					protected_branches     = false
					custom_branch_policies = true
				}
			}

			resource "github_repository_deployment_branch_policy" "br1" {
				repository       = github_repository.test.name
				environment_name = github_repository_environment.env.environment
				name             = "main"
			}

			resource "github_repository_deployment_branch_policy" "br2" {
				repository       = github_repository.test.name
				environment_name = github_repository_environment.env.environment
				name             = "develop"
			}

			data "github_repository_deployment_branch_policies" "all" {
				repository = github_repository.test.name
				environment_name = github_repository_environment.env.environment
			}
		`, randomID)

		// Common assertions for both providers
		commonChecks := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_repository_deployment_branch_policies.all", "deployment_branch_policies.#", "2"),
			resource.TestCheckTypeSetElemNestedAttrs("data.github_repository_deployment_branch_policies.all", "deployment_branch_policies.*", map[string]string{
				"name": "main",
			}),
			resource.TestCheckTypeSetElemNestedAttrs("data.github_repository_deployment_branch_policies.all", "deployment_branch_policies.*", map[string]string{
				"name": "develop",
			}),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck: func() { skipUnlessMode(t, mode) },
				Steps: []resource.TestStep{
					{
						// First test with mux provider (both SDKv2 and Framework available)
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
						Config:                   config,
						Check:                    commonChecks,
					},
					{
						// Then test with pure Framework provider
						ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
						Config:                   config,
						Check:                    commonChecks,
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
