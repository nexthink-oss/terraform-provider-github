package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGithubBranchProtectionV3_defaults(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("configures default settings when empty", func(t *testing.T) {
		config := fmt.Sprintf(`
		resource "github_repository" "test" {
		  name      = "tf-acc-test-%s"
		  auto_init = true
		}

		resource "github_branch_protection_v3" "test" {
		  repository  = github_repository.test.name
		  branch      = "main"
		}
		`, randomID)

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "branch", "main",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "require_signed_commits", "false",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "require_conversation_resolution", "false",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_status_checks.#", "0",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.#", "0",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "restrictions.#", "0",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubBranchProtectionV3_conversation_resolution(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("configures conversation resolution", func(t *testing.T) {
		config := fmt.Sprintf(`
		resource "github_repository" "test" {
		  name      = "tf-acc-test-%s"
		  auto_init = true
		}

		resource "github_branch_protection_v3" "test" {
		  repository  = github_repository.test.name
		  branch      = "main"
		  require_conversation_resolution = true
		}
		`, randomID)

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "require_conversation_resolution", "true",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubBranchProtectionV3_required_status_checks(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("configures required status checks", func(t *testing.T) {
		config := fmt.Sprintf(`
		resource "github_repository" "test" {
		  name      = "tf-acc-test-%s"
		  auto_init = true
		}

		resource "github_branch_protection_v3" "test" {
		  repository  = github_repository.test.name
		  branch      = "main"

		  required_status_checks {
		    strict = true
		    checks = ["ci/test:1234", "ci/build"]
		  }
		}
		`, randomID)

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_status_checks.#", "1",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_status_checks.0.strict", "true",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_status_checks.0.checks.#", "2",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubBranchProtectionV3_required_pull_request_reviews(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("configures required pull request reviews", func(t *testing.T) {
		config := fmt.Sprintf(`
		resource "github_repository" "test" {
		  name      = "tf-acc-test-%s"
		  auto_init = true
		}

		resource "github_branch_protection_v3" "test" {
		  repository  = github_repository.test.name
		  branch      = "main"

		  required_pull_request_reviews {
		    required_approving_review_count = 2
		    dismiss_stale_reviews = true
		    require_code_owner_reviews = true
		    require_last_push_approval = true
		  }
		}
		`, randomID)

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.#", "1",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.0.required_approving_review_count", "2",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.0.dismiss_stale_reviews", "true",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.0.require_code_owner_reviews", "true",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.0.require_last_push_approval", "true",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubBranchProtectionV3_restrictions(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("configures push restrictions", func(t *testing.T) {
		config := fmt.Sprintf(`
		resource "github_repository" "test" {
		  name      = "tf-acc-test-%s"
		  auto_init = true
		}

		resource "github_branch_protection_v3" "test" {
		  repository  = github_repository.test.name
		  branch      = "main"

		  restrictions {
		    users = ["octocat"]
		    teams = ["justice-league"]
		    apps  = ["super-ci"]
		  }
		}
		`, randomID)

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "restrictions.#", "1",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "restrictions.0.users.#", "1",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "restrictions.0.teams.#", "1",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "restrictions.0.apps.#", "1",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubBranchProtectionV3_dismiss_stale_reviews(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("configures dismissal restrictions", func(t *testing.T) {
		config := fmt.Sprintf(`
		resource "github_repository" "test" {
		  name      = "tf-acc-test-%s"
		  auto_init = true
		}

		resource "github_branch_protection_v3" "test" {
		  repository  = github_repository.test.name
		  branch      = "main"

		  required_pull_request_reviews {
		    dismiss_stale_reviews = true
		    dismissal_users       = ["octocat"]
		    dismissal_teams       = ["justice-league"]
		    dismissal_apps        = ["super-ci"]
		  }
		}
		`, randomID)

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.0.dismiss_stale_reviews", "true",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.0.dismissal_users.#", "1",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.0.dismissal_teams.#", "1",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.0.dismissal_apps.#", "1",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubBranchProtectionV3_bypass_pull_request_allowances(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("configures bypass pull request allowances", func(t *testing.T) {
		config := fmt.Sprintf(`
		resource "github_repository" "test" {
		  name      = "tf-acc-test-%s"
		  auto_init = true
		}

		resource "github_branch_protection_v3" "test" {
		  repository  = github_repository.test.name
		  branch      = "main"

		  required_pull_request_reviews {
		    bypass_pull_request_allowances {
		      users = ["octocat"]
		      teams = ["justice-league"]
		      apps  = ["super-ci"]
		    }
		  }
		}
		`, randomID)

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.0.bypass_pull_request_allowances.#", "1",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.0.bypass_pull_request_allowances.0.users.#", "1",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.0.bypass_pull_request_allowances.0.teams.#", "1",
			),
			resource.TestCheckResourceAttr(
				"github_branch_protection_v3.test", "required_pull_request_reviews.0.bypass_pull_request_allowances.0.apps.#", "1",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubBranchProtectionV3_update(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("updates branch protection settings", func(t *testing.T) {
		config1 := fmt.Sprintf(`
		resource "github_repository" "test" {
		  name      = "tf-acc-test-%s"
		  auto_init = true
		}

		resource "github_branch_protection_v3" "test" {
		  repository  = github_repository.test.name
		  branch      = "main"
		  enforce_admins = false
		}
		`, randomID)

		config2 := fmt.Sprintf(`
		resource "github_repository" "test" {
		  name      = "tf-acc-test-%s"
		  auto_init = true
		}

		resource "github_branch_protection_v3" "test" {
		  repository  = github_repository.test.name
		  branch      = "main"
		  enforce_admins = true
		  require_signed_commits = true
		  require_conversation_resolution = true

		  required_status_checks {
		    strict = true
		    checks = ["ci/test", "ci/build"]
		  }

		  required_pull_request_reviews {
		    required_approving_review_count = 2
		    dismiss_stale_reviews = true
		    require_code_owner_reviews = true
		  }
		}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config1,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_branch_protection_v3.test", "enforce_admins", "false",
							),
							resource.TestCheckResourceAttr(
								"github_branch_protection_v3.test", "required_status_checks.#", "0",
							),
						),
					},
					{
						Config: config2,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_branch_protection_v3.test", "enforce_admins", "true",
							),
							resource.TestCheckResourceAttr(
								"github_branch_protection_v3.test", "require_signed_commits", "true",
							),
							resource.TestCheckResourceAttr(
								"github_branch_protection_v3.test", "require_conversation_resolution", "true",
							),
							resource.TestCheckResourceAttr(
								"github_branch_protection_v3.test", "required_status_checks.#", "1",
							),
							resource.TestCheckResourceAttr(
								"github_branch_protection_v3.test", "required_pull_request_reviews.#", "1",
							),
						),
					},
				},
			})
		}

		t.Run("with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})

		t.Run("with an individual account", func(t *testing.T) {
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubBranchProtectionV3_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("imports branch protection", func(t *testing.T) {
		config := fmt.Sprintf(`
		resource "github_repository" "test" {
		  name      = "tf-acc-test-%s"
		  auto_init = true
		}

		resource "github_branch_protection_v3" "test" {
		  repository  = github_repository.test.name
		  branch      = "main"
		  enforce_admins = true
		  require_signed_commits = true

		  required_status_checks {
		    strict = true
		    checks = ["ci/test"]
		  }
		}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
					},
					{
						ResourceName:      "github_branch_protection_v3.test",
						ImportState:       true,
						ImportStateVerify: true,
						ImportStateIdFunc: func(s *terraform.State) (string, error) {
							return fmt.Sprintf("tf-acc-test-%s:main", randomID), nil
						},
					},
				},
			})
		}

		t.Run("with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})

		t.Run("with an individual account", func(t *testing.T) {
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}

func TestAccGithubBranchProtectionV3_migration_compatibility(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("validates compatibility with existing state", func(t *testing.T) {
		config := fmt.Sprintf(`
		resource "github_repository" "test" {
		  name      = "tf-acc-test-%s"
		  auto_init = true
		}

		resource "github_branch_protection_v3" "test" {
		  repository  = github_repository.test.name
		  branch      = "main"
		  enforce_admins = true
		  require_signed_commits = false
		  require_conversation_resolution = true

		  required_status_checks {
		    strict = true
		    checks = ["ci/test:1234", "ci/build"]
		  }

		  required_pull_request_reviews {
		    required_approving_review_count = 1
		    dismiss_stale_reviews = false
		    require_code_owner_reviews = true
		    require_last_push_approval = false
		  }

		  restrictions {
		    users = []
		    teams = []
		    apps  = []
		  }
		}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(
								"github_branch_protection_v3.test", "enforce_admins", "true",
							),
							resource.TestCheckResourceAttr(
								"github_branch_protection_v3.test", "require_signed_commits", "false",
							),
							resource.TestCheckResourceAttr(
								"github_branch_protection_v3.test", "require_conversation_resolution", "true",
							),
						),
					},
					{
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

		t.Run("with an anonymous account", func(t *testing.T) {
			t.Skip("anonymous account not supported for this operation")
		})

		t.Run("with an individual account", func(t *testing.T) {
			t.Skip("individual account not supported for this operation")
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
