package framework

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubRepositoryDataSource(t *testing.T) {

	t.Run("anonymously queries a repository without error", func(t *testing.T) {

		config := fmt.Sprintf(`
			data "github_repositories" "test" {
				query = "org:%s"
			}

			data "github_repository" "test" {
				full_name = data.github_repositories.test.full_names.0
			}
		`, testOrganizationFunc())

		check := resource.ComposeTestCheckFunc(
			resource.TestMatchResourceAttr(
				"data.github_repositories.test", "full_names.0",
				regexp.MustCompile(`^`+testOrganizationFunc())),
			resource.TestMatchResourceAttr(
				"data.github_repository.test", "full_name",
				regexp.MustCompile(`^`+testOrganizationFunc())),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an anonymous account", func(t *testing.T) {
			testCase(t, anonymous)
		})

	})

	t.Run("queries a repository with pages configured", func(t *testing.T) {

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name         = "tf-acc-%s"
				auto_init    = true
				pages {
					source {
						branch = "main"
					}
				}
			}

			data "github_repository" "test" {
				name = github_repository.test.name
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "pages.0.source.0.branch",
				"main",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
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

	t.Run("checks defaults on a new repository", func(t *testing.T) {

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`

			resource "github_repository" "test" {
				name         = "tf-acc-%s"
				auto_init    = true
			}

			data "github_repository" "test" {
				name = github_repository.test.name
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "name", "tf-acc-"+randomID,
			),
			resource.TestCheckResourceAttrSet(
				"data.github_repository.test", "has_projects",
			),
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "description", "",
			),
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "homepage_url", "",
			),
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "pages.#", "0",
			),
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "fork", "false",
			),
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "allow_update_branch", "true",
			),
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "delete_branch_on_merge", "true",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
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

	t.Run("queries a repository that is a template", func(t *testing.T) {

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name        = "tf-acc-%s"
				is_template = true
			}

			data "github_repository" "test" {
				name = github_repository.test.name
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "is_template",
				"true",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
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

	t.Run("queries a repository that was generated from a template", func(t *testing.T) {

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-%s"
				template {
					owner      = "template-repository"
					repository = "template-repository"
				}
			}

			data "github_repository" "test" {
				name = github_repository.test.name
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "template.0.owner",
				"template-repository",
			),
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "template.0.repository",
				"template-repository",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
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

	t.Run("queries a repository that has no primary_language", func(t *testing.T) {

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-%s"
			}

			data "github_repository" "test" {
				name = github_repository.test.name
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "primary_language",
				"",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
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

	t.Run("queries a repository that has go as primary_language", func(t *testing.T) {

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-%s"
				auto_init = true
			}
			resource "github_repository_file" "test" {
				repository     = github_repository.test.name
				file           = "test.go"
				content        = "package main"
			}

			data "github_repository" "test" {
				name = github_repository_file.test.repository
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "primary_language",
				"Go",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						// Not doing any checks since the language doesnt have time to be updated on the first apply
						Config: config,
					},
					{
						// Re-running the terraform will refresh the language since the go-file has been created
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

	t.Run("queries a repository that has a license", func(t *testing.T) {

		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-%s"
				auto_init = true
			}
			resource "github_repository_file" "test" {
				repository     = github_repository.test.name
				file           = "LICENSE"
				content             = <<EOT

Copyright (c) 2011-2023 GitHub Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE."
EOT
}

			data "github_repository" "test" {
				name = github_repository_file.test.repository
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"data.github_repository.test", "repository_license.0.license.0.spdx_id",
				"MIT",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
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

	t.Run("migration validation - comparing SDKv2 and Framework behavior", func(t *testing.T) {
		randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-%s"
				auto_init = true
				description = "Test repository for migration validation"
				homepage_url = "https://example.com"
				topics = ["terraform", "migration", "test"]
				has_issues = true
				has_projects = true
				has_wiki = true
				allow_merge_commit = true
				allow_squash_merge = true
				allow_rebase_merge = true
				delete_branch_on_merge = true
			}

			data "github_repository" "test" {
				name = github_repository.test.name
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			// Basic attributes
			resource.TestCheckResourceAttr("data.github_repository.test", "name", "tf-acc-"+randomID),
			resource.TestCheckResourceAttr("data.github_repository.test", "description", "Test repository for migration validation"),
			resource.TestCheckResourceAttr("data.github_repository.test", "homepage_url", "https://example.com"),
			resource.TestCheckResourceAttr("data.github_repository.test", "private", "false"),
			resource.TestCheckResourceAttr("data.github_repository.test", "fork", "false"),
			resource.TestCheckResourceAttr("data.github_repository.test", "archived", "false"),
			resource.TestCheckResourceAttr("data.github_repository.test", "is_template", "false"),

			// Boolean settings
			resource.TestCheckResourceAttr("data.github_repository.test", "has_issues", "true"),
			resource.TestCheckResourceAttr("data.github_repository.test", "has_projects", "true"),
			resource.TestCheckResourceAttr("data.github_repository.test", "has_wiki", "true"),
			resource.TestCheckResourceAttr("data.github_repository.test", "allow_merge_commit", "true"),
			resource.TestCheckResourceAttr("data.github_repository.test", "allow_squash_merge", "true"),
			resource.TestCheckResourceAttr("data.github_repository.test", "allow_rebase_merge", "true"),
			resource.TestCheckResourceAttr("data.github_repository.test", "delete_branch_on_merge", "true"),

			// Computed values
			resource.TestCheckResourceAttrSet("data.github_repository.test", "full_name"),
			resource.TestCheckResourceAttrSet("data.github_repository.test", "html_url"),
			resource.TestCheckResourceAttrSet("data.github_repository.test", "ssh_clone_url"),
			resource.TestCheckResourceAttrSet("data.github_repository.test", "git_clone_url"),
			resource.TestCheckResourceAttrSet("data.github_repository.test", "http_clone_url"),
			resource.TestCheckResourceAttrSet("data.github_repository.test", "svn_url"),
			resource.TestCheckResourceAttrSet("data.github_repository.test", "default_branch"),
			resource.TestCheckResourceAttrSet("data.github_repository.test", "node_id"),
			resource.TestCheckResourceAttrSet("data.github_repository.test", "repo_id"),
			resource.TestCheckResourceAttrSet("data.github_repository.test", "visibility"),

			// Topics validation
			resource.TestCheckResourceAttr("data.github_repository.test", "topics.#", "3"),
			resource.TestCheckResourceAttr("data.github_repository.test", "topics.0", "terraform"),
			resource.TestCheckResourceAttr("data.github_repository.test", "topics.1", "migration"),
			resource.TestCheckResourceAttr("data.github_repository.test", "topics.2", "test"),

			// Empty nested structures
			resource.TestCheckResourceAttr("data.github_repository.test", "pages.#", "0"),
			resource.TestCheckResourceAttr("data.github_repository.test", "repository_license.#", "0"),
			resource.TestCheckResourceAttr("data.github_repository.test", "template.#", "0"),
		)

		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheckIndividual(t) },
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
			Steps: []resource.TestStep{
				{
					Config: config,
					Check:  check,
				},
			},
		})
	})
}

// Helper functions for migration validation
func TestAccGithubRepositoryDataSource_ConflictValidation(t *testing.T) {
	config := `
		data "github_repository" "test" {
			full_name = "owner/repo"
			name = "repo"
		}
	`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckIndividual(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`.*`), // Should error due to conflicting attributes
			},
		},
	})
}

func TestAccGithubRepositoryDataSource_FullNameOnly(t *testing.T) {
	config := fmt.Sprintf(`
		data "github_repository" "test" {
			full_name = "%s/terraform-provider-github"
		}
	`, testOrganizationFunc())

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { skipUnlessMode(t, anonymous) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.github_repository.test", "name", "terraform-provider-github"),
					resource.TestCheckResourceAttr("data.github_repository.test", "full_name", testOrganizationFunc()+"/terraform-provider-github"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryDataSource_NameOnly(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	config := fmt.Sprintf(`
		resource "github_repository" "test" {
			name = "tf-acc-%s"
		}

		data "github_repository" "test" {
			name = github_repository.test.name
		}
	`, randomID)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckIndividual(t) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.github_repository.test", "name", "tf-acc-"+randomID),
					resource.TestCheckResourceAttrSet("data.github_repository.test", "full_name"),
				),
			},
		},
	})
}
