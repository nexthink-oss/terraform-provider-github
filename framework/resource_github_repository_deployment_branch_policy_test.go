package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubRepositoryDeploymentBranchPolicyResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryDeploymentBranchPolicyConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_deployment_branch_policy.test", "name", "main"),
					resource.TestCheckResourceAttr("github_repository_deployment_branch_policy.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_repository_deployment_branch_policy.test", "environment_name", "my_env"),
					resource.TestCheckResourceAttrSet("github_repository_deployment_branch_policy.test", "etag"),
					resource.TestCheckResourceAttrSet("github_repository_deployment_branch_policy.test", "id"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryDeploymentBranchPolicyResource_update(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryDeploymentBranchPolicyConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_deployment_branch_policy.test", "name", "main"),
				),
			},
			{
				Config: testAccGithubRepositoryDeploymentBranchPolicyConfig_updated(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_deployment_branch_policy.test", "name", "foo"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryDeploymentBranchPolicyResource_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryDeploymentBranchPolicyConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_deployment_branch_policy.test", "name", "main"),
				),
			},
			{
				ResourceName:      "github_repository_deployment_branch_policy.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGithubRepositoryDeploymentBranchPolicyResource_organization(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryDeploymentBranchPolicyConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_deployment_branch_policy.test", "name", "main"),
					resource.TestCheckResourceAttr("github_repository_deployment_branch_policy.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_repository_deployment_branch_policy.test", "environment_name", "my_env"),
					resource.TestCheckResourceAttrSet("github_repository_deployment_branch_policy.test", "etag"),
					resource.TestCheckResourceAttrSet("github_repository_deployment_branch_policy.test", "id"),
				),
			},
		},
	})
}

func testAccGithubRepositoryDeploymentBranchPolicyConfig_basic(randomID string) string {
	return fmt.Sprintf(`
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

resource "github_repository_deployment_branch_policy" "test" {
  repository       = github_repository.test.name
  environment_name = github_repository_environment.env.environment
  name             = github_repository.test.default_branch
}
`, randomID)
}

func testAccGithubRepositoryDeploymentBranchPolicyConfig_updated(randomID string) string {
	return fmt.Sprintf(`
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

resource "github_repository_deployment_branch_policy" "test" {
  repository       = github_repository.test.name
  environment_name = github_repository_environment.env.environment
  name             = "foo"
}
`, randomID)
}
