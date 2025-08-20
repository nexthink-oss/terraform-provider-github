package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubRepositoryDeploymentBranchPoliciesDataSource(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("queries deployment branch policies", func(t *testing.T) {

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
		`, randomID)

		config2 := config + `
			data "github_repository_deployment_branch_policies" "all" {
				repository = github_repository.test.name
				environment_name = github_repository_environment.env.environment
			}
		`
		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_repository_deployment_branch_policies.all", "deployment_branch_policies.#", "1"),
			resource.TestCheckResourceAttr("data.github_repository_deployment_branch_policies.all", "deployment_branch_policies.0.name", "foo"),
			resource.TestCheckResourceAttrSet("data.github_repository_deployment_branch_policies.all", "deployment_branch_policies.0.id"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
					},
					{
						Config: config2,
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
}

func TestAccGithubRepositoryDeploymentBranchPoliciesDataSource_EmptyList(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("handles empty deployment branch policies list", func(t *testing.T) {

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
		`, randomID)

		config2 := config + `
			data "github_repository_deployment_branch_policies" "all" {
				repository = github_repository.test.name
				environment_name = github_repository_environment.env.environment
			}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_repository_deployment_branch_policies.all", "deployment_branch_policies.#", "0"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
					},
					{
						Config: config2,
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
}

func TestAccGithubRepositoryDeploymentBranchPoliciesDataSource_MultiplePolicies(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("queries multiple deployment branch policies", func(t *testing.T) {

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
		`, randomID)

		config2 := config + `
			data "github_repository_deployment_branch_policies" "all" {
				repository = github_repository.test.name
				environment_name = github_repository_environment.env.environment
			}
		`

		check := resource.ComposeTestCheckFunc(
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
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
					},
					{
						Config: config2,
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
}
