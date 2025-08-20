package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubRepositoryEnvironmentsDataSource_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	config := fmt.Sprintf(`
		resource "github_repository" "test" {
			name = "tf-acc-test-%[1]s"
			auto_init = true
		}
		resource "github_repository_environment" "env1" {
			repository = github_repository.test.name
			environment = "env_x"
		}
		data "github_repository_environments" "all" {
			repository = github_repository.test.name
		}
	`, randomID)

	testCase := func(t *testing.T, mode string) {
		resource.Test(t, resource.TestCase{
			PreCheck: func() { skipUnlessMode(t, mode) },
			Steps: []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"github": {
							Source:            "integrations/github",
							VersionConstraint: "~> 6.0",
						},
					},
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("data.github_repository_environments.all", "environments.#", "1"),
						resource.TestCheckResourceAttr("data.github_repository_environments.all", "environments.0.name", "env_x"),
						resource.TestCheckResourceAttrSet("data.github_repository_environments.all", "environments.0.node_id"),
					),
				},
				{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Config:                   config,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						},
					},
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("data.github_repository_environments.all", "environments.#", "1"),
						resource.TestCheckResourceAttr("data.github_repository_environments.all", "environments.0.name", "env_x"),
						resource.TestCheckResourceAttrSet("data.github_repository_environments.all", "environments.0.node_id"),
					),
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
}
