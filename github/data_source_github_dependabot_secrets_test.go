package github

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubDependabotSecretsDataSource(t *testing.T) {
	t.Run("queries dependabot secrets from a repository", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_dependabot_secret" "test" {
				secret_name     = "dep_secret_1"
				repository      = github_repository.test.name
				plaintext_value = "foo"
			}
		`, randomID)

		config2 := config + `
			data "github_dependabot_secrets" "test" {
				name = github_repository.test.name
			}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_dependabot_secrets.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
			resource.TestCheckResourceAttr("data.github_dependabot_secrets.test", "secrets.#", "1"),
			resource.TestCheckResourceAttr("data.github_dependabot_secrets.test", "secrets.0.name", "DEP_SECRET_1"),
			resource.TestCheckResourceAttrSet("data.github_dependabot_secrets.test", "secrets.0.created_at"),
			resource.TestCheckResourceAttrSet("data.github_dependabot_secrets.test", "secrets.0.updated_at"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  resource.ComposeTestCheckFunc(),
					},
					{
						Config: config2,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("queries dependabot secrets using full_name", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name      = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_dependabot_secret" "test" {
				secret_name     = "dep_secret_1"
				repository      = github_repository.test.name
				plaintext_value = "foo"
			}

			data "github_dependabot_secrets" "test" {
				full_name = github_repository.test.full_name
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_dependabot_secrets.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
			resource.TestCheckResourceAttrSet("data.github_dependabot_secrets.test", "full_name"),
			resource.TestCheckResourceAttr("data.github_dependabot_secrets.test", "secrets.#", "1"),
			resource.TestCheckResourceAttr("data.github_dependabot_secrets.test", "secrets.0.name", "DEP_SECRET_1"),
			resource.TestCheckResourceAttrSet("data.github_dependabot_secrets.test", "secrets.0.created_at"),
			resource.TestCheckResourceAttrSet("data.github_dependabot_secrets.test", "secrets.0.updated_at"),
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

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("fails when both name and full_name are provided", func(t *testing.T) {
		config := `
			data "github_dependabot_secrets" "test" {
				name      = "test-repo"
				full_name = "test-owner/test-repo"
			}
		`

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
				},
			},
		})
	})

	t.Run("fails when neither name nor full_name are provided", func(t *testing.T) {
		config := `
			data "github_dependabot_secrets" "test" {
			}
		`

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile(`Invalid Configuration`),
				},
			},
		})
	})
}
