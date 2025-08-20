package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubRepositoryCustomPropertiesDataSource_Migration(t *testing.T) {
	t.Skip("Migration test requires pre-existing custom properties in organization")

	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	_ = randomID // Test is skipped but variable kept for consistency

	config := fmt.Sprintf(`
		resource "github_repository" "test" {
			name = "tf-acc-test-%s"
			auto_init = true
		}
		
		data "github_repository_custom_properties" "test" {
			repository = github_repository.test.name
		}
	`, randomID)

	resource.Test(t, resource.TestCase{
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
					resource.TestCheckResourceAttr("data.github_repository_custom_properties.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttrSet("data.github_repository_custom_properties.test", "id"),
					resource.TestCheckResourceAttrSet("data.github_repository_custom_properties.test", "property.#"),
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
					resource.TestCheckResourceAttr("data.github_repository_custom_properties.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttrSet("data.github_repository_custom_properties.test", "id"),
					resource.TestCheckResourceAttrSet("data.github_repository_custom_properties.test", "property.#"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryCustomPropertiesDataSource_Mux(t *testing.T) {
	t.Skip("Mux test requires pre-existing custom properties in organization")

	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	config := fmt.Sprintf(`
		resource "github_repository" "test" {
			name = "tf-acc-test-%s"
			auto_init = true
		}
		
		data "github_repository_custom_properties" "test" {
			repository = github_repository.test.name
		}
	`, randomID)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.github_repository_custom_properties.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttrSet("data.github_repository_custom_properties.test", "id"),
					resource.TestCheckResourceAttrSet("data.github_repository_custom_properties.test", "property.#"),
				),
			},
		},
	})
}
