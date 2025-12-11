package github

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccGithubRepositoryCustomProperties(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("manages custom properties for multiple repositories", func(t *testing.T) {
		config := fmt.Sprintf(`
			# Create organization-level custom properties
			resource "github_organization_custom_property" "environment" {
				name           = "environment_%[1]s"
				value_type     = "single_select"
				required       = false
				description    = "Deployment environment"
				allowed_values = ["production", "staging", "development"]
			}

			resource "github_organization_custom_property" "team" {
				name        = "team_%[1]s"
				value_type  = "string"
				required    = false
				description = "Team responsible"
			}

			resource "github_organization_custom_property" "tags" {
				name           = "tags_%[1]s"
				value_type     = "multi_select"
				required       = false
				description    = "Repository tags"
				allowed_values = ["frontend", "backend", "api", "database"]
			}

			# Create test repositories
			resource "github_repository" "test1" {
				name                                 = "tf-acc-test-%[1]s-1"
				auto_init                            = true
				ignore_vulnerability_alerts_during_read = true
			}

			resource "github_repository" "test2" {
				name                                 = "tf-acc-test-%[1]s-2"
				auto_init                            = true
				ignore_vulnerability_alerts_during_read = true
			}

			# Apply custom properties to repositories
			resource "github_repository_custom_properties" "test" {
				depends_on = [
					github_organization_custom_property.environment,
					github_organization_custom_property.team,
					github_organization_custom_property.tags,
				]

				repository {
					repository_name = github_repository.test1.name
					property {
						name  = github_organization_custom_property.environment.name
						value = ["production"]
					}
					property {
						name  = github_organization_custom_property.team.name
						value = ["platform-team"]
					}
				}

				repository {
					repository_name = github_repository.test2.name
					property {
						name  = github_organization_custom_property.environment.name
						value = ["staging"]
					}
					property {
						name  = github_organization_custom_property.tags.name
						value = ["frontend", "api"]
					}
				}
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			// Check we have 2 repositories
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.#", "2"),

			// Check first repository has 2 properties
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.0.property.#", "2"),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.0.property.*", map[string]string{
				"name": fmt.Sprintf("environment_%s", randomID),
			}),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.0.property.*", map[string]string{
				"name": fmt.Sprintf("team_%s", randomID),
			}),

			// Check second repository has 2 properties
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.1.property.#", "2"),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.1.property.*", map[string]string{
				"name": fmt.Sprintf("environment_%s", randomID),
			}),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.1.property.*", map[string]string{
				"name": fmt.Sprintf("tags_%s", randomID),
			}),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:  func() { skipUnlessMode(t, mode) },
				Providers: testAccProviders,
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

	t.Run("updates custom properties across repositories", func(t *testing.T) {
		configInitial := fmt.Sprintf(`
			resource "github_organization_custom_property" "status" {
				name           = "status_%[1]s"
				value_type     = "single_select"
				required       = false
				description    = "Repository status"
				allowed_values = ["active", "inactive"]
			}

			resource "github_repository" "test1" {
				name                                 = "tf-acc-test-%[1]s-update-1"
				auto_init                            = true
				ignore_vulnerability_alerts_during_read = true
			}

			resource "github_repository" "test2" {
				name                                 = "tf-acc-test-%[1]s-update-2"
				auto_init                            = true
				ignore_vulnerability_alerts_during_read = true
			}

			resource "github_repository_custom_properties" "test" {
				depends_on = [github_organization_custom_property.status]

				repository {
					repository_name = github_repository.test1.name
					property {
						name  = github_organization_custom_property.status.name
						value = ["active"]
					}
				}

				repository {
					repository_name = github_repository.test2.name
					property {
						name  = github_organization_custom_property.status.name
						value = ["active"]
					}
				}
			}
		`, randomID)

		configUpdated := fmt.Sprintf(`
			resource "github_organization_custom_property" "status" {
				name           = "status_%[1]s"
				value_type     = "single_select"
				required       = false
				description    = "Repository status"
				allowed_values = ["active", "inactive"]
			}

			resource "github_organization_custom_property" "critical" {
				name        = "critical_%[1]s"
				value_type  = "true_false"
				required    = false
				description = "Is critical"
			}

			resource "github_repository" "test1" {
				name                                 = "tf-acc-test-%[1]s-update-1"
				auto_init                            = true
				ignore_vulnerability_alerts_during_read = true
			}

			resource "github_repository" "test2" {
				name                                 = "tf-acc-test-%[1]s-update-2"
				auto_init                            = true
				ignore_vulnerability_alerts_during_read = true
			}

			resource "github_repository_custom_properties" "test" {
				depends_on = [
					github_organization_custom_property.status,
					github_organization_custom_property.critical,
				]

				repository {
					repository_name = github_repository.test1.name
					property {
						name  = github_organization_custom_property.status.name
						value = ["inactive"]
					}
					property {
						name  = github_organization_custom_property.critical.name
						value = ["true"]
					}
				}

				repository {
					repository_name = github_repository.test2.name
					property {
						name  = github_organization_custom_property.status.name
						value = ["active"]
					}
					property {
						name  = github_organization_custom_property.critical.name
						value = ["false"]
					}
				}
			}
		`, randomID)

		checkInitial := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.#", "2"),
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.0.property.#", "1"),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.0.property.*", map[string]string{
				"name": fmt.Sprintf("status_%s", randomID),
			}),
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.1.property.#", "1"),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.1.property.*", map[string]string{
				"name": fmt.Sprintf("status_%s", randomID),
			}),
		)

		checkUpdated := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.#", "2"),
			// First repository
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.0.property.#", "2"),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.0.property.*", map[string]string{
				"name": fmt.Sprintf("status_%s", randomID),
			}),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.0.property.*", map[string]string{
				"name": fmt.Sprintf("critical_%s", randomID),
			}),
			// Second repository
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.1.property.#", "2"),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.1.property.*", map[string]string{
				"name": fmt.Sprintf("status_%s", randomID),
			}),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.1.property.*", map[string]string{
				"name": fmt.Sprintf("critical_%s", randomID),
			}),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:  func() { skipUnlessMode(t, mode) },
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: configInitial,
						Check:  checkInitial,
					},
					{
						Config: configUpdated,
						Check:  checkUpdated,
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

	t.Run("errors when organization property does not exist", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name                                 = "tf-acc-test-%[1]s-error"
				auto_init                            = true
				ignore_vulnerability_alerts_during_read = true
			}

			resource "github_repository_custom_properties" "test" {
				repository {
					repository_name = github_repository.test.name
					property {
						name  = "nonexistent_property_%[1]s"
						value = ["some-value"]
					}
				}
			}
		`, randomID)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:  func() { skipUnlessMode(t, mode) },
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile("failed to update custom properties|not found|invalid"),
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

	t.Run("handles single value properties correctly", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_organization_custom_property" "owner" {
				name        = "owner_%[1]s"
				value_type  = "string"
				required    = false
				description = "Repository owner"
			}

			resource "github_organization_custom_property" "archived" {
				name        = "archived_%[1]s"
				value_type  = "true_false"
				required    = false
				description = "Is archived"
			}

			resource "github_repository" "test" {
				name                                 = "tf-acc-test-%[1]s-single"
				auto_init                            = true
				ignore_vulnerability_alerts_during_read = true
			}

			resource "github_repository_custom_properties" "test" {
				depends_on = [
					github_organization_custom_property.owner,
					github_organization_custom_property.archived,
				]

				repository {
					repository_name = github_repository.test.name
					property {
						name  = github_organization_custom_property.owner.name
						value = ["platform-team"]
					}
					property {
						name  = github_organization_custom_property.archived.name
						value = ["false"]
					}
				}
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.#", "1"),
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.0.property.#", "2"),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.0.property.*", map[string]string{
				"name": fmt.Sprintf("owner_%s", randomID),
			}),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.0.property.*", map[string]string{
				"name": fmt.Sprintf("archived_%s", randomID),
			}),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:  func() { skipUnlessMode(t, mode) },
				Providers: testAccProviders,
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

	t.Run("removes properties when removed from configuration", func(t *testing.T) {
		configWithProperties := fmt.Sprintf(`
			resource "github_organization_custom_property" "label" {
				name        = "label_%[1]s"
				value_type  = "string"
				required    = false
				description = "Repository label"
			}

			resource "github_organization_custom_property" "visibility" {
				name           = "visibility_%[1]s"
				value_type     = "single_select"
				required       = false
				description    = "Visibility level"
				allowed_values = ["public", "private"]
			}

			resource "github_repository" "test" {
				name                                 = "tf-acc-test-%[1]s-remove"
				auto_init                            = true
				ignore_vulnerability_alerts_during_read = true
			}

			resource "github_repository_custom_properties" "test" {
				depends_on = [
					github_organization_custom_property.label,
					github_organization_custom_property.visibility,
				]

				repository {
					repository_name = github_repository.test.name
					property {
						name  = github_organization_custom_property.label.name
						value = ["test-label"]
					}
					property {
						name  = github_organization_custom_property.visibility.name
						value = ["public"]
					}
				}
			}
		`, randomID)

		configWithoutOneProperty := fmt.Sprintf(`
			resource "github_organization_custom_property" "label" {
				name        = "label_%[1]s"
				value_type  = "string"
				required    = false
				description = "Repository label"
			}

			resource "github_organization_custom_property" "visibility" {
				name           = "visibility_%[1]s"
				value_type     = "single_select"
				required       = false
				description    = "Visibility level"
				allowed_values = ["public", "private"]
			}

			resource "github_repository" "test" {
				name                                 = "tf-acc-test-%[1]s-remove"
				auto_init                            = true
				ignore_vulnerability_alerts_during_read = true
			}

			resource "github_repository_custom_properties" "test" {
				depends_on = [
					github_organization_custom_property.label,
					github_organization_custom_property.visibility,
				]

				repository {
					repository_name = github_repository.test.name
					property {
						name  = github_organization_custom_property.label.name
						value = ["test-label"]
					}
				}
			}
		`, randomID)

		checkWithBoth := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.0.property.#", "2"),
		)

		checkWithOne := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.0.property.#", "1"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:  func() { skipUnlessMode(t, mode) },
				Providers: testAccProviders,
				Steps: []resource.TestStep{
					{
						Config: configWithProperties,
						Check:  checkWithBoth,
					},
					{
						Config: configWithoutOneProperty,
						Check:  checkWithOne,
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

	t.Run("handles multi-select properties with multiple values", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_organization_custom_property" "technologies" {
				name           = "technologies_%[1]s"
				value_type     = "multi_select"
				required       = false
				description    = "Technologies used"
				allowed_values = ["go", "python", "javascript", "typescript", "rust"]
			}

			resource "github_repository" "test" {
				name                                 = "tf-acc-test-%[1]s-multiselect"
				auto_init                            = true
				ignore_vulnerability_alerts_during_read = true
			}

			resource "github_repository_custom_properties" "test" {
				depends_on = [github_organization_custom_property.technologies]

				repository {
					repository_name = github_repository.test.name
					property {
						name  = github_organization_custom_property.technologies.name
						value = ["go", "python", "typescript"]
					}
				}
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.#", "1"),
			resource.TestCheckResourceAttr("github_repository_custom_properties.test", "repository.0.property.#", "1"),
			resource.TestCheckTypeSetElemNestedAttrs("github_repository_custom_properties.test", "repository.0.property.*", map[string]string{
				"name": fmt.Sprintf("technologies_%s", randomID),
			}),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:  func() { skipUnlessMode(t, mode) },
				Providers: testAccProviders,
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
