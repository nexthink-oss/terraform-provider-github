package framework

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestGithubBranchResource tests the basic resource creation
func TestGithubBranchResource(t *testing.T) {
	resource := NewGithubBranchResource()
	if resource == nil {
		t.Error("Resource should not be nil")
	}
}

func TestAccGithubBranch_Framework(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("creates a branch directly", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%[1]s"
				auto_init = true
			}

			resource "github_branch" "test" {
				repository = github_repository.test.id
				branch     = "test"
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestMatchResourceAttr(
				"github_branch.test", "id",
				regexp.MustCompile(fmt.Sprintf("tf-acc-test-%s:test", randomID)),
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "repository",
				fmt.Sprintf("tf-acc-test-%s", randomID),
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "branch",
				"test",
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "source_branch",
				"main",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch.test", "source_sha",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch.test", "etag",
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "ref",
				"refs/heads/test",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch.test", "sha",
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
					{
						ResourceName:            "github_branch.test",
						ImportState:             true,
						ImportStateId:           fmt.Sprintf("tf-acc-test-%s:test", randomID),
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"source_sha"},
					},
					{
						ResourceName:  "github_branch.test",
						ImportState:   true,
						ImportStateId: fmt.Sprintf("tf-acc-test-%s:nonsense", randomID),
						ExpectError: regexp.MustCompile(
							`Repository tf-acc-test-[a-z0-9]+ does not have a branch named nonsense`,
						),
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

	t.Run("creates a branch named main directly and a repository with a gitignore_template", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%[1]s"
				auto_init = true
				gitignore_template = "Python"
			}

			resource "github_branch" "test" {
				repository = github_repository.test.id
				branch     = "main"
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestMatchResourceAttr(
				"github_branch.test", "id",
				regexp.MustCompile(fmt.Sprintf("tf-acc-test-%s:main", randomID)),
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "repository",
				fmt.Sprintf("tf-acc-test-%s", randomID),
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "branch",
				"main",
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "source_branch",
				"main",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch.test", "source_sha",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch.test", "etag",
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "ref",
				"refs/heads/main",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch.test", "sha",
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
					{
						ResourceName:            "github_branch.test",
						ImportState:             true,
						ImportStateId:           fmt.Sprintf("tf-acc-test-%s:main", randomID),
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"source_sha"},
					},
					{
						ResourceName:  "github_branch.test",
						ImportState:   true,
						ImportStateId: fmt.Sprintf("tf-acc-test-%s:nonsense", randomID),
						ExpectError: regexp.MustCompile(
							`Repository tf-acc-test-[a-z0-9]+ does not have a branch named nonsense`,
						),
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

	t.Run("creates a branch from a source branch", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%[1]s"
				auto_init = true
			}

			resource "github_branch" "source" {
				repository = github_repository.test.id
				branch     = "source"
			}

			resource "github_branch" "test" {
				repository    = github_repository.test.id
				source_branch = github_branch.source.branch
				branch        = "test"
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestMatchResourceAttr(
				"github_branch.test", "id",
				regexp.MustCompile(fmt.Sprintf("tf-acc-test-%s:test", randomID)),
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "repository",
				fmt.Sprintf("tf-acc-test-%s", randomID),
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "branch",
				"test",
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "source_branch",
				"source",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch.test", "source_sha",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch.test", "etag",
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "ref",
				"refs/heads/test",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch.test", "sha",
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
					{
						ResourceName:            "github_branch.test",
						ImportState:             true,
						ImportStateId:           fmt.Sprintf("tf-acc-test-%s:test:source", randomID),
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"source_sha"},
					},
					{
						ResourceName:  "github_branch.test",
						ImportState:   true,
						ImportStateId: fmt.Sprintf("tf-acc-test-%s:nonsense:source", randomID),
						ExpectError: regexp.MustCompile(
							`Repository tf-acc-test-[a-z0-9]+ does not have a branch named nonsense`,
						),
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

	t.Run("creates a branch with specified source SHA", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%[1]s"
				auto_init = true
			}

			resource "github_branch" "test" {
				repository = github_repository.test.id
				branch     = "test-sha"
				source_branch = "main"
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestMatchResourceAttr(
				"github_branch.test", "id",
				regexp.MustCompile(fmt.Sprintf("tf-acc-test-%s:test-sha", randomID)),
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "repository",
				fmt.Sprintf("tf-acc-test-%s", randomID),
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "branch",
				"test-sha",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch.test", "source_sha",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch.test", "etag",
			),
			resource.TestCheckResourceAttr(
				"github_branch.test", "ref",
				"refs/heads/test-sha",
			),
			resource.TestCheckResourceAttrSet(
				"github_branch.test", "sha",
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
					{
						ResourceName:      "github_branch.test",
						ImportState:       true,
						ImportStateId:     fmt.Sprintf("tf-acc-test-%s:test-sha", randomID),
						ImportStateVerify: true,
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

// Test helpers for ID parsing
func TestGithubBranchResource_parseTwoPartID(t *testing.T) {
	r := &githubBranchResource{}

	tests := []struct {
		id         string
		wantRepo   string
		wantBranch string
		wantError  bool
	}{
		{
			id:         "myrepo:mybranch",
			wantRepo:   "myrepo",
			wantBranch: "mybranch",
			wantError:  false,
		},
		{
			id:         "myrepo:feature/branch-name",
			wantRepo:   "myrepo",
			wantBranch: "feature/branch-name",
			wantError:  false,
		},
		{
			id:        "invalid",
			wantError: true,
		},
		{
			id:        "",
			wantError: true,
		},
		{
			id:         "myrepo:",
			wantRepo:   "myrepo",
			wantBranch: "",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			gotRepo, gotBranch, err := r.parseTwoPartID(tt.id)
			if (err != nil) != tt.wantError {
				t.Errorf("parseTwoPartID(%q) error = %v, wantError %v", tt.id, err, tt.wantError)
				return
			}
			if !tt.wantError {
				if gotRepo != tt.wantRepo || gotBranch != tt.wantBranch {
					t.Errorf("parseTwoPartID(%q) = %q, %q, want %q, %q",
						tt.id, gotRepo, gotBranch, tt.wantRepo, tt.wantBranch)
				}
			}
		})
	}
}

func TestGithubBranchResource_buildTwoPartID(t *testing.T) {
	r := &githubBranchResource{}

	tests := []struct {
		repo   string
		branch string
		want   string
	}{
		{
			repo:   "myrepo",
			branch: "mybranch",
			want:   "myrepo:mybranch",
		},
		{
			repo:   "myrepo",
			branch: "feature/branch-name",
			want:   "myrepo:feature/branch-name",
		},
		{
			repo:   "",
			branch: "mybranch",
			want:   ":mybranch",
		},
		{
			repo:   "myrepo",
			branch: "",
			want:   "myrepo:",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s:%s", tt.repo, tt.branch), func(t *testing.T) {
			got := r.buildTwoPartID(tt.repo, tt.branch)
			if got != tt.want {
				t.Errorf("buildTwoPartID(%q, %q) = %q, want %q", tt.repo, tt.branch, got, tt.want)
			}
		})
	}
}
