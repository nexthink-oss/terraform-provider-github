package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGithubRepositoryRulesetResource(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("creates and updates repository rulesets without errors", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name         = "tf-acc-test-%s"
				auto_init    = true
				default_branch = "main"
			}

			resource "github_repository_environment" "example" {
				environment = "test"
				repository  = github_repository.test.name
			}

			resource "github_repository_ruleset" "test" {
				name        = "test"
				repository  = github_repository.test.name
				target      = "branch"
				enforcement = "active"

				conditions {
					ref_name {
						include = ["refs/heads/main"]
						exclude = []
					}
				}

				rules {
					creation = true
					update   = true
					deletion = true
					required_linear_history = true

					required_deployments {
						required_deployment_environments = ["test"]
					}

					required_signatures = false

					merge_queue {
						check_response_timeout_minutes    = 10
						grouping_strategy                 = "ALLGREEN"
						max_entries_to_build              = 5
						max_entries_to_merge              = 5
						merge_method                      = "MERGE"
						min_entries_to_merge              = 1
						min_entries_to_merge_wait_minutes = 60
					}

					pull_request {
						required_approving_review_count   = 2
						required_review_thread_resolution = true
						require_code_owner_review         = true
						dismiss_stale_reviews_on_push     = true
						require_last_push_approval        = true
					}

					required_status_checks {
						required_check {
							context = "ci"
						}

						strict_required_status_checks_policy = true
						do_not_enforce_on_create             = true
					}

					non_fast_forward = true
				}
			}
		`, randomID)

		updatedConfig := fmt.Sprintf(`
			resource "github_repository" "test" {
				name         = "tf-acc-test-%s"
				auto_init    = true
				default_branch = "main"
			}

			resource "github_repository_environment" "example" {
				environment = "test"
				repository  = github_repository.test.name
			}

			resource "github_repository_ruleset" "test" {
				name        = "test-updated"
				repository  = github_repository.test.name
				target      = "branch"
				enforcement = "evaluate"

				conditions {
					ref_name {
						include = ["refs/heads/main", "refs/heads/develop"]
						exclude = ["refs/heads/temp/*"]
					}
				}

				rules {
					creation = true
					update   = true
					deletion = false
					required_linear_history = false

					required_deployments {
						required_deployment_environments = ["test"]
					}

					required_signatures = true

					pull_request {
						required_approving_review_count   = 1
						required_review_thread_resolution = false
						require_code_owner_review         = false
						dismiss_stale_reviews_on_push     = false
						require_last_push_approval        = false
					}

					required_status_checks {
						required_check {
							context = "ci"
						}
						required_check {
							context        = "build"
							integration_id = 123456
						}

						strict_required_status_checks_policy = false
						do_not_enforce_on_create             = false
					}

					non_fast_forward = false
				}
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "name", "test"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "target", "branch"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "enforcement", "active"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.creation", "true"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.update", "true"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.deletion", "true"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.required_linear_history", "true"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.required_signatures", "false"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.non_fast_forward", "true"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.pull_request.0.required_approving_review_count", "2"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.required_status_checks.0.strict_required_status_checks_policy", "true"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "conditions.0.ref_name.0.include.0", "refs/heads/main"),
			resource.TestCheckResourceAttrSet("github_repository_ruleset.test", "ruleset_id"),
			resource.TestCheckResourceAttrSet("github_repository_ruleset.test", "node_id"),
		)

		updatedCheck := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "name", "test-updated"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "enforcement", "evaluate"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.deletion", "false"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.required_linear_history", "false"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.required_signatures", "true"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.non_fast_forward", "false"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.pull_request.0.required_approving_review_count", "1"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.required_status_checks.0.strict_required_status_checks_policy", "false"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "conditions.0.ref_name.0.include.1", "refs/heads/develop"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "conditions.0.ref_name.0.exclude.0", "refs/heads/temp/*"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
					{
						Config: updatedConfig,
						Check:  updatedCheck,
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

	t.Run("creates repository rulesets with enterprise features", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-enterprise-%s"
				auto_init = true
			}

			resource "github_repository_ruleset" "test" {
				name        = "enterprise-test"
				repository  = github_repository.test.name
				target      = "branch"
				enforcement = "active"

				conditions {
					ref_name {
						include = ["refs/heads/main"]
						exclude = []
					}
				}

				rules {
					creation = true

					commit_message_pattern {
						name     = "Must include ticket"
						negate   = false
						operator = "regex"
						pattern  = "^\\[TICKET-\\d+\\]"
					}

					commit_author_email_pattern {
						name     = "Corporate email only"
						negate   = false
						operator = "ends_with"
						pattern  = "@company.com"
					}

					branch_name_pattern {
						name     = "Feature branch naming"
						negate   = false
						operator = "starts_with"
						pattern  = "feature/"
					}

					required_code_scanning {
						required_code_scanning_tool {
							alerts_threshold          = "errors"
							security_alerts_threshold = "high_or_higher"
							tool                      = "CodeQL"
						}
					}
				}
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "name", "enterprise-test"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.commit_message_pattern.0.name", "Must include ticket"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.commit_message_pattern.0.operator", "regex"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.commit_message_pattern.0.pattern", "^\\[TICKET-\\d+\\]"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.commit_author_email_pattern.0.pattern", "@company.com"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.branch_name_pattern.0.pattern", "feature/"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
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

	t.Run("creates repository rulesets with bypass actors", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-bypass-%s"
				auto_init = true
			}

			resource "github_repository_ruleset" "test" {
				name        = "bypass-test"
				repository  = github_repository.test.name
				target      = "branch"
				enforcement = "active"

				bypass_actors {
					actor_id    = 1
					actor_type  = "OrganizationAdmin"
					bypass_mode = "always"
				}

				bypass_actors {
					actor_id    = 2
					actor_type  = "RepositoryRole"
					bypass_mode = "pull_request"
				}

				conditions {
					ref_name {
						include = ["refs/heads/main"]
						exclude = []
					}
				}

				rules {
					creation = true
					deletion = true
				}
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "name", "bypass-test"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "bypass_actors.0.actor_id", "1"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "bypass_actors.0.actor_type", "OrganizationAdmin"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "bypass_actors.0.bypass_mode", "always"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "bypass_actors.1.actor_id", "2"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "bypass_actors.1.actor_type", "RepositoryRole"),
			resource.TestCheckResourceAttr("github_repository_ruleset.test", "bypass_actors.1.bypass_mode", "pull_request"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
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

func TestAccGithubRepositoryRulesetResource_Import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	config := fmt.Sprintf(`
		resource "github_repository" "test" {
			name      = "tf-acc-test-import-%s"
			auto_init = true
		}

		resource "github_repository_ruleset" "test" {
			name        = "import-test"
			repository  = github_repository.test.name
			target      = "branch"
			enforcement = "active"

			conditions {
				ref_name {
					include = ["refs/heads/main"]
					exclude = []
				}
			}

			rules {
				creation = true
				deletion = true
			}
		}
	`, randomID)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_ruleset.test", "name", "import-test"),
					resource.TestCheckResourceAttrSet("github_repository_ruleset.test", "ruleset_id"),
				),
			},
			{
				ResourceName:            "github_repository_ruleset.test",
				ImportState:             true,
				ImportStateIdFunc:       testAccGithubRepositoryRulesetImportStateIdFunc("github_repository_ruleset.test"),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

func TestAccGithubRepositoryRulesetResource_Migration(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	config := fmt.Sprintf(`
		resource "github_repository" "test" {
			name      = "tf-acc-test-migration-%s"
			auto_init = true
		}

		resource "github_repository_ruleset" "test" {
			name        = "migration-test"
			repository  = github_repository.test.name
			target      = "branch"
			enforcement = "active"

			conditions {
				ref_name {
					include = ["refs/heads/main"]
					exclude = []
				}
			}

			rules {
				creation = true
				deletion = true
				required_linear_history = true
			}
		}
	`, randomID)

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
					resource.TestCheckResourceAttr("github_repository_ruleset.test", "name", "migration-test"),
					resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.creation", "true"),
					resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.deletion", "true"),
					resource.TestCheckResourceAttr("github_repository_ruleset.test", "rules.0.required_linear_history", "true"),
					resource.TestCheckResourceAttrSet("github_repository_ruleset.test", "ruleset_id"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
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

// Helper function for import testing
func testAccGithubRepositoryRulesetImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("resource id not set")
		}

		repoName := rs.Primary.Attributes["repository"]
		rulesetID := rs.Primary.Attributes["ruleset_id"]

		return fmt.Sprintf("%s/%s", repoName, rulesetID), nil
	}
}
