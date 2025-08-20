package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGithubRepositoryMilestoneDataSource_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("migrates from SDKv2 to Framework without state changes", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
			}

			resource "github_repository_milestone" "test" {
				owner = split("/", "${github_repository.test.full_name}")[0]
				repository = github_repository.test.name
				title = "v1.0.0"
				description = "General Availability"
				due_date = "2020-11-22"
				state = "closed"
			}

			data "github_repository_milestone" "test" {
				owner = github_repository_milestone.test.owner
				repository = github_repository_milestone.test.repository
				number = github_repository_milestone.test.number
			}
		`, randomID)

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
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_repository_milestone.test", "state", "closed"),
							resource.TestCheckResourceAttr("data.github_repository_milestone.test", "title", "v1.0.0"),
							resource.TestCheckResourceAttr("data.github_repository_milestone.test", "description", "General Availability"),
							resource.TestCheckResourceAttr("data.github_repository_milestone.test", "due_date", "2020-11-22"),
							resource.TestCheckResourceAttrSet("data.github_repository_milestone.test", "id"),
						),
					},
					// Step 2: Refresh with muxed provider (includes Framework data source)
					{
						ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
						Config:                   config,
						ConfigPlanChecks: resource.ConfigPlanChecks{
							PreApply: []plancheck.PlanCheck{
								plancheck.ExpectEmptyPlan(),
							},
						},
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr("data.github_repository_milestone.test", "state", "closed"),
							resource.TestCheckResourceAttr("data.github_repository_milestone.test", "title", "v1.0.0"),
							resource.TestCheckResourceAttr("data.github_repository_milestone.test", "description", "General Availability"),
							resource.TestCheckResourceAttr("data.github_repository_milestone.test", "due_date", "2020-11-22"),
							resource.TestCheckResourceAttrSet("data.github_repository_milestone.test", "id"),
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
	})
}

func TestAccGithubRepositoryMilestoneDataSource_Framework_PureTest(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("framework provider handles all attributes correctly", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
			}

			resource "github_repository_milestone" "test" {
				owner = split("/", "${github_repository.test.full_name}")[0]
				repository = github_repository.test.name
				title = "v1.0.0"
				description = "General Availability"
				due_date = "2020-11-22"
				state = "closed"
			}

			data "github_repository_milestone" "test" {
				owner = github_repository_milestone.test.owner
				repository = github_repository_milestone.test.repository
				number = github_repository_milestone.test.number
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							// Basic attribute checks
							resource.TestCheckResourceAttr("data.github_repository_milestone.test", "state", "closed"),
							resource.TestCheckResourceAttr("data.github_repository_milestone.test", "title", "v1.0.0"),
							resource.TestCheckResourceAttr("data.github_repository_milestone.test", "description", "General Availability"),
							resource.TestCheckResourceAttr("data.github_repository_milestone.test", "due_date", "2020-11-22"),
							resource.TestCheckResourceAttrSet("data.github_repository_milestone.test", "id"),
							resource.TestCheckResourceAttrSet("data.github_repository_milestone.test", "number"),
							// Cross-reference checks with the resource
							resource.TestCheckResourceAttrPair("data.github_repository_milestone.test", "owner", "github_repository_milestone.test", "owner"),
							resource.TestCheckResourceAttrPair("data.github_repository_milestone.test", "repository", "github_repository_milestone.test", "repository"),
							resource.TestCheckResourceAttrPair("data.github_repository_milestone.test", "number", "github_repository_milestone.test", "number"),
							resource.TestCheckResourceAttrPair("data.github_repository_milestone.test", "title", "github_repository_milestone.test", "title"),
							resource.TestCheckResourceAttrPair("data.github_repository_milestone.test", "description", "github_repository_milestone.test", "description"),
							resource.TestCheckResourceAttrPair("data.github_repository_milestone.test", "due_date", "github_repository_milestone.test", "due_date"),
							resource.TestCheckResourceAttrPair("data.github_repository_milestone.test", "state", "github_repository_milestone.test", "state"),
							// Type validation
							func(s *terraform.State) error {
								// Verify the data source exists in state
								rs, ok := s.RootModule().Resources["data.github_repository_milestone.test"]
								if !ok {
									return fmt.Errorf("data source not found: data.github_repository_milestone.test")
								}

								// Verify required attributes are present
								if rs.Primary.Attributes["id"] == "" {
									return fmt.Errorf("data source id is empty")
								}
								if rs.Primary.Attributes["owner"] == "" {
									return fmt.Errorf("data source owner is empty")
								}
								if rs.Primary.Attributes["repository"] == "" {
									return fmt.Errorf("data source repository is empty")
								}
								if rs.Primary.Attributes["number"] == "" {
									return fmt.Errorf("data source number is empty")
								}
								if rs.Primary.Attributes["title"] == "" {
									return fmt.Errorf("data source title is empty")
								}

								return nil
							},
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
	})
}