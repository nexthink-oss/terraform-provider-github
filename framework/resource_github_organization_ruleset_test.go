package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGithubOrganizationRulesetResource(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("creates and updates organization rulesets without errors", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name         = "tf-acc-test-%s"
				auto_init    = true
				default_branch = "main"
			}

			resource "github_organization_ruleset" "test" {
				name        = "test"
				target      = "branch"
				enforcement = "active"

				conditions {
					ref_name {
						include = ["refs/heads/main"]
						exclude = []
					}
					repository_name {
						include = ["tf-acc-test-%s"]
						exclude = []
						protected = true
					}
				}

				rules {
					creation = true
					update   = true
					deletion = true
					required_linear_history = true

					pull_request {
						dismiss_stale_reviews_on_push = true
						require_code_owner_review = false
						require_last_push_approval = false
						required_approving_review_count = 1
						required_review_thread_resolution = false
					}

					required_status_checks {
						strict_required_status_checks_policy = true

						required_check {
							context = "ci/test"
						}
					}
				}

				bypass_actors {
					actor_id = 1
					actor_type = "OrganizationAdmin"
					bypass_mode = "always"
				}
			}
		`, randomID, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "name", "test"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "target", "branch"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "enforcement", "active"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "conditions.0.ref_name.0.include.0", "refs/heads/main"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "conditions.0.repository_name.0.include.0", fmt.Sprintf("tf-acc-test-%s", randomID)),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "conditions.0.repository_name.0.protected", "true"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.creation", "true"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.update", "true"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.deletion", "true"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.required_linear_history", "true"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.pull_request.0.dismiss_stale_reviews_on_push", "true"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.pull_request.0.required_approving_review_count", "1"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.required_status_checks.0.strict_required_status_checks_policy", "true"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "bypass_actors.0.actor_id", "1"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "bypass_actors.0.actor_type", "OrganizationAdmin"),
			resource.TestCheckResourceAttr("github_organization_ruleset.test", "bypass_actors.0.bypass_mode", "always"),
			resource.TestCheckResourceAttrSet("github_organization_ruleset.test", "ruleset_id"),
			resource.TestCheckResourceAttrSet("github_organization_ruleset.test", "node_id"),
			resource.TestCheckResourceAttrSet("github_organization_ruleset.test", "etag"),
		)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: config,
					Check:  check,
				},
			},
		})
	})

	t.Run("handles updates to organization rulesets", func(t *testing.T) {
		config1 := fmt.Sprintf(`
			resource "github_repository" "test" {
				name         = "tf-acc-test-%s"
				auto_init    = true
				default_branch = "main"
			}

			resource "github_organization_ruleset" "test" {
				name        = "test"
				target      = "branch"
				enforcement = "active"

				conditions {
					ref_name {
						include = ["refs/heads/main"]
						exclude = []
					}
					repository_name {
						include = ["tf-acc-test-%s"]
						exclude = []
					}
				}

				rules {
					creation = true
					update   = true
				}

				bypass_actors {
					actor_id = 1
					actor_type = "OrganizationAdmin"
					bypass_mode = "always"
				}
			}
		`, randomID, randomID)

		config2 := fmt.Sprintf(`
			resource "github_repository" "test" {
				name         = "tf-acc-test-%s"
				auto_init    = true
				default_branch = "main"
			}

			resource "github_organization_ruleset" "test" {
				name        = "test-updated"
				target      = "branch"
				enforcement = "evaluate"

				conditions {
					ref_name {
						include = ["refs/heads/main", "refs/heads/dev"]
						exclude = ["refs/heads/experimental"]
					}
					repository_name {
						include = ["tf-acc-test-%s"]
						exclude = []
						protected = true
					}
				}

				rules {
					creation = true
					update   = true
					deletion = true
					required_linear_history = true

					pull_request {
						require_code_owner_review = true
						required_approving_review_count = 2
					}
				}

				bypass_actors {
					actor_id = 1
					actor_type = "OrganizationAdmin"
					bypass_mode = "pull_request"
				}
			}
		`, randomID, randomID)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: config1,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "name", "test"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "enforcement", "active"),
					),
				},
				{
					Config: config2,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "name", "test-updated"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "enforcement", "evaluate"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "conditions.0.ref_name.0.include.0", "refs/heads/main"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "conditions.0.ref_name.0.include.1", "refs/heads/dev"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "conditions.0.ref_name.0.exclude.0", "refs/heads/experimental"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "conditions.0.repository_name.0.protected", "true"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.deletion", "true"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.required_linear_history", "true"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.pull_request.0.require_code_owner_review", "true"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.pull_request.0.required_approving_review_count", "2"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "bypass_actors.0.bypass_mode", "pull_request"),
					),
				},
			},
		})
	})

	t.Run("supports repository ID targeting", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name         = "tf-acc-test-%s"
				auto_init    = true
				default_branch = "main"
			}

			resource "github_organization_ruleset" "test" {
				name        = "test-repo-id"
				target      = "branch"
				enforcement = "active"

				conditions {
					ref_name {
						include = ["refs/heads/main"]
						exclude = []
					}
					repository_id = [github_repository.test.repo_id]
				}

				rules {
					creation = true
					update   = true
				}

				bypass_actors {
					actor_id = 1
					actor_type = "OrganizationAdmin"
					bypass_mode = "always"
				}
			}
		`, randomID)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "name", "test-repo-id"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "target", "branch"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "enforcement", "active"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "conditions.0.ref_name.0.include.0", "refs/heads/main"),
						resource.TestCheckResourceAttrSet("github_organization_ruleset.test", "conditions.0.repository_id.0"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.creation", "true"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.update", "true"),
					),
				},
			},
		})
	})

	t.Run("supports pattern rules", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name         = "tf-acc-test-%s"
				auto_init    = true
				default_branch = "main"
			}

			resource "github_organization_ruleset" "test" {
				name        = "test-patterns"
				target      = "branch"
				enforcement = "active"

				conditions {
					ref_name {
						include = ["refs/heads/main"]
						exclude = []
					}
					repository_name {
						include = ["tf-acc-test-%s"]
						exclude = []
					}
				}

				rules {
					commit_message_pattern {
						operator = "starts_with"
						pattern = "feat:"
						name = "Require feat prefix"
						negate = false
					}

					commit_author_email_pattern {
						operator = "ends_with"
						pattern = "@example.com"
						name = "Require company email"
					}

					branch_name_pattern {
						operator = "regex"
						pattern = "^(main|dev|feature/.+)$"
						name = "Branch naming convention"
					}
				}

				bypass_actors {
					actor_id = 1
					actor_type = "OrganizationAdmin"
					bypass_mode = "always"
				}
			}
		`, randomID, randomID)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "name", "test-patterns"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.commit_message_pattern.0.operator", "starts_with"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.commit_message_pattern.0.pattern", "feat:"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.commit_message_pattern.0.name", "Require feat prefix"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.commit_author_email_pattern.0.operator", "ends_with"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.commit_author_email_pattern.0.pattern", "@example.com"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.branch_name_pattern.0.operator", "regex"),
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "rules.0.branch_name_pattern.0.pattern", "^(main|dev|feature/.+)$"),
					),
				},
			},
		})
	})

	t.Run("supports import", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name         = "tf-acc-test-%s"
				auto_init    = true
				default_branch = "main"
			}

			resource "github_organization_ruleset" "test" {
				name        = "test-import"
				target      = "branch"
				enforcement = "active"

				conditions {
					ref_name {
						include = ["refs/heads/main"]
						exclude = []
					}
					repository_name {
						include = ["tf-acc-test-%s"]
						exclude = []
					}
				}

				rules {
					creation = true
					update   = true
				}

				bypass_actors {
					actor_id = 1
					actor_type = "OrganizationAdmin"
					bypass_mode = "always"
				}
			}
		`, randomID, randomID)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "name", "test-import"),
						resource.TestCheckResourceAttrSet("github_organization_ruleset.test", "ruleset_id"),
					),
				},
				{
					ResourceName:            "github_organization_ruleset.test",
					ImportState:             true,
					ImportStateVerify:       true,
					ImportStateIdFunc:       testAccGithubOrganizationRulesetImportStateIdFunc("github_organization_ruleset.test"),
					ImportStateVerifyIgnore: []string{"etag"},
				},
			},
		})
	})

	t.Run("migration compatibility test", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name         = "tf-acc-test-%s"
				auto_init    = true
				default_branch = "main"
			}

			resource "github_organization_ruleset" "test" {
				name        = "test-migration"
				target      = "branch"
				enforcement = "active"

				conditions {
					ref_name {
						include = ["refs/heads/main"]
						exclude = []
					}
					repository_name {
						include = ["tf-acc-test-%s"]
						exclude = []
					}
				}

				rules {
					creation = true
					update   = true
					deletion = true
					required_linear_history = true
				}

				bypass_actors {
					actor_id = 1
					actor_type = "OrganizationAdmin"
					bypass_mode = "always"
				}
			}
		`, randomID, randomID)

		resource.Test(t, resource.TestCase{
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
						resource.TestCheckResourceAttr("github_organization_ruleset.test", "name", "test-migration"),
						resource.TestCheckResourceAttrSet("github_organization_ruleset.test", "ruleset_id"),
					),
				},
				{
					ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
					Config:                   config,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						},
					},
				},
			},
		})
	})
}

// Helper function for import tests
func testAccGithubOrganizationRulesetImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("resource id not set")
		}

		return rs.Primary.Attributes["ruleset_id"], nil
	}
}
