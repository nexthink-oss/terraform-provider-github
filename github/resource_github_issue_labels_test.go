package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubIssueLabelsResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	empty := []map[string]any{}

	testCase := func(t *testing.T, mode string) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { skipUnlessMode(t, mode) },
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				// 0. Check if some labels already exist (indicated by non-empty plan)
				{
					Config:             testAccGithubIssueLabelsConfig(randomID, empty),
					ExpectNonEmptyPlan: true,
				},
				// 1. Check if all the labels are destroyed when the resource is added
				{
					Config: testAccGithubIssueLabelsConfig(randomID, empty),
					Check:  resource.TestCheckResourceAttr("github_issue_labels.test", "label.#", "0"),
				},
				// 2. Check if a label can be created
				{
					Config: testAccGithubIssueLabelsConfig(randomID, append(empty, map[string]any{
						"name":        "foo",
						"color":       "000000",
						"description": "foo",
					})),
					Check: resource.TestCheckResourceAttr("github_issue_labels.test", "label.#", "1"),
				},
				// 3. Check if a label can be recreated (case sensitivity test)
				{
					Config: testAccGithubIssueLabelsConfig(randomID, append(empty, map[string]any{
						"name":        "Foo",
						"color":       "000000",
						"description": "foo",
					})),
					Check: resource.TestCheckResourceAttr("github_issue_labels.test", "label.#", "1"),
				},
				// 4. Check if multiple labels can be created
				{
					Config: testAccGithubIssueLabelsConfig(randomID, append(empty,
						map[string]any{
							"name":        "Foo",
							"color":       "000000",
							"description": "foo",
						},
						map[string]any{
							"name":        "bar",
							"color":       "111111",
							"description": "bar",
						}, map[string]any{
							"name":        "baz",
							"color":       "222222",
							"description": "baz",
						})),
					Check: resource.TestCheckResourceAttr("github_issue_labels.test", "label.#", "3"),
				},
				// 5. Check if labels can be destroyed
				{
					Config: testAccGithubIssueLabelsConfig(randomID, nil),
				},
				// 6. Check if labels were actually destroyed
				{
					Config: testAccGithubIssueLabelsConfig(randomID, empty),
					Check:  resource.TestCheckResourceAttr("github_issue_labels.test", "label.#", "0"),
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
}

func TestAccGithubIssueLabelsResource_updateLabels(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubIssueLabelsConfig(randomID, []map[string]any{
					{
						"name":        "bug",
						"color":       "d73a4a",
						"description": "Something isn't working",
					},
					{
						"name":        "enhancement",
						"color":       "a2eeef",
						"description": "New feature or request",
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue_labels.test", "label.#", "2"),
					resource.TestCheckResourceAttr("github_issue_labels.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttrSet("github_issue_labels.test", "id"),
				),
			},
			{
				Config: testAccGithubIssueLabelsConfig(randomID, []map[string]any{
					{
						"name":        "bug",
						"color":       "ff0000",              // Changed color
						"description": "Something is broken", // Changed description
					},
					{
						"name":        "documentation",
						"color":       "0075ca",
						"description": "Improvements or additions to documentation",
					},
					// enhancement removed, documentation added
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue_labels.test", "label.#", "2"),
				),
			},
		},
	})
}

func TestAccGithubIssueLabelsResource_emptyLabels(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubIssueLabelsConfig_emptyLabels(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue_labels.test", "label.#", "0"),
					resource.TestCheckResourceAttr("github_issue_labels.test", "repository", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttrSet("github_issue_labels.test", "id"),
				),
			},
		},
	})
}

func TestAccGithubIssueLabelsResource_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubIssueLabelsConfig(randomID, []map[string]any{
					{
						"name":        "imported-label",
						"color":       "fbca04",
						"description": "This label was imported",
					},
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue_labels.test", "label.#", "1"),
				),
			},
			{
				ResourceName:      "github_issue_labels.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGithubIssueLabelsResource_migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	testLabels := []map[string]any{
		{
			"name":        "test-label",
			"color":       "ff5722",
			"description": "Test label for migration",
		},
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testAccPreCheck(t, individual) },
		Steps: []resource.TestStep{
			// First step with muxed provider (includes SDKv2)
			{
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Config:                   testAccGithubIssueLabelsConfig(randomID, testLabels),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_issue_labels.test", "label.#", "1"),
					resource.TestCheckResourceAttrSet("github_issue_labels.test", "id"),
				),
			},
			// Second step should be no-op with same provider
			{
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Config:                   testAccGithubIssueLabelsConfig(randomID, testLabels),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

// testAccGithubIssueLabelsConfig generates configuration for the github_issue_labels resource
func testAccGithubIssueLabelsConfig(randomId string, labels []map[string]any) string {
	resource := ""
	if labels != nil {
		dynamic := ""
		for _, label := range labels {
			dynamic += fmt.Sprintf(`
				label {
					name = "%s"
					color = "%s"
					description = "%s"
				}
			`, label["name"], label["color"], label["description"])
		}

		resource = fmt.Sprintf(`
			resource "github_issue_labels" "test" {
				repository = github_repository.test.name

				%s
			}
		`, dynamic)
	}

	return fmt.Sprintf(`
		resource "github_repository" "test" {
			name = "tf-acc-test-%s"
			auto_init = true
		}

		%s
	`, randomId, resource)
}

// testAccGithubIssueLabelsConfig_emptyLabels generates configuration with explicitly empty labels
func testAccGithubIssueLabelsConfig_emptyLabels(randomId string) string {
	return fmt.Sprintf(`
		resource "github_repository" "test" {
			name = "tf-acc-test-%s"
			auto_init = true
		}

		resource "github_issue_labels" "test" {
			repository = github_repository.test.name
		}
	`, randomId)
}

// TestGithubIssueLabelsResource tests the basic resource creation
func TestGithubIssueLabelsResource(t *testing.T) {
	resource := NewGithubIssueLabelsResource()
	if resource == nil {
		t.Error("Resource should not be nil")
	}
}
