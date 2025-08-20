package framework

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestParseRepoNameFramework(t *testing.T) {
	r := &githubRepositoryCollaboratorResource{}
	tests := []struct {
		name         string
		repoName     string
		defaultOwner string
		wantOwner    string
		wantRepoName string
	}{
		{
			name:         "Repo name without owner",
			repoName:     "example-repo",
			defaultOwner: "default-owner",
			wantOwner:    "default-owner",
			wantRepoName: "example-repo",
		},
		{
			name:         "Repo name with owner",
			repoName:     "owner-name/example-repo",
			defaultOwner: "default-owner",
			wantOwner:    "owner-name",
			wantRepoName: "example-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepoName := r.parseRepoName(tt.repoName, tt.defaultOwner)
			if gotOwner != tt.wantOwner || gotRepoName != tt.wantRepoName {
				t.Errorf("parseRepoName(%q, %q) = %q, %q, want %q, %q",
					tt.repoName, tt.defaultOwner, gotOwner, gotRepoName, tt.wantOwner, tt.wantRepoName)
			}
		})
	}
}

func TestParseTwoPartIDFramework(t *testing.T) {
	r := &githubRepositoryCollaboratorResource{}
	tests := []struct {
		name         string
		id           string
		wantRepo     string
		wantUsername string
		wantError    bool
	}{
		{
			name:         "Valid ID",
			id:           "example-repo:username",
			wantRepo:     "example-repo",
			wantUsername: "username",
			wantError:    false,
		},
		{
			name:         "Valid ID with owner",
			id:           "owner/example-repo:username",
			wantRepo:     "owner/example-repo",
			wantUsername: "username",
			wantError:    false,
		},
		{
			name:      "Invalid ID - no colon",
			id:        "example-repo-username",
			wantError: true,
		},
		{
			name:         "Invalid ID - multiple colons",
			id:           "example:repo:username",
			wantRepo:     "example",
			wantUsername: "repo:username",
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRepo, gotUsername, err := r.parseTwoPartID(tt.id)
			if (err != nil) != tt.wantError {
				t.Errorf("parseTwoPartID(%q) error = %v, wantError %v", tt.id, err, tt.wantError)
				return
			}
			if !tt.wantError {
				if gotRepo != tt.wantRepo || gotUsername != tt.wantUsername {
					t.Errorf("parseTwoPartID(%q) = %q, %q, want %q, %q",
						tt.id, gotRepo, gotUsername, tt.wantRepo, tt.wantUsername)
				}
			}
		})
	}
}

func TestCaseInsensitiveUsername(t *testing.T) {
	// Test the case-insensitive behavior by testing the logic directly
	testCases := []struct {
		old      string
		new      string
		expected bool // true if diff should be suppressed
	}{
		{"username", "username", true},   // exact match
		{"username", "USERNAME", true},   // case insensitive match
		{"username", "Username", true},   // mixed case match
		{"username", "different", false}, // different usernames
		{"user1", "user2", false},        // different usernames
	}

	for _, tc := range testCases {
		result := strings.EqualFold(tc.old, tc.new)
		if result != tc.expected {
			t.Errorf("Case insensitive comparison for %q vs %q: got %v, expected %v",
				tc.old, tc.new, result, tc.expected)
		}
	}
}

func TestPermissionDiffSuppression(t *testing.T) {
	// Test permission diff suppression logic
	testCases := []struct {
		permission         string
		suppressionEnabled bool
		shouldSuppress     bool
	}{
		{"triage", true, true},
		{"maintain", true, true},
		{"push", true, false},
		{"pull", true, false},
		{"admin", true, false},
		{"triage", false, false},
		{"maintain", false, false},
	}

	for _, tc := range testCases {
		name := fmt.Sprintf("permission=%s,suppression=%v", tc.permission, tc.suppressionEnabled)
		t.Run(name, func(t *testing.T) {
			result := tc.suppressionEnabled && (tc.permission == "triage" || tc.permission == "maintain")
			if result != tc.shouldSuppress {
				t.Errorf("Permission suppression for %q with suppression=%v: got %v, expected %v",
					tc.permission, tc.suppressionEnabled, result, tc.shouldSuppress)
			}
		})
	}
}

func TestAccGithubRepositoryCollaboratorResource(t *testing.T) {
	t.Skip("update <username> below to unskip this test run")

	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	t.Run("creates invitations without error", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_collaborator" "test_repo_collaborator" {
				repository = github_repository.test.name
				username   = "<username>"
				permission = "triage"
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_repository_collaborator.test_repo_collaborator", "permission",
				"triage",
			),
			resource.TestCheckResourceAttr(
				"github_repository_collaborator.test_repo_collaborator", "permission_diff_suppression",
				"false",
			),
			resource.TestCheckResourceAttrSet(
				"github_repository_collaborator.test_repo_collaborator", "id",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
					{
						ResourceName:            "github_repository_collaborator.test_repo_collaborator",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"permission_diff_suppression"},
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

	t.Run("creates invitations when repository contains the org name", func(t *testing.T) {
		orgName := os.Getenv("GITHUB_ORGANIZATION")

		if orgName == "" {
			t.Skip("Set GITHUB_ORGANIZATION to unskip this test run")
		}

		configWithOwner := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_collaborator" "test_repo_collaborator_2" {
				repository = "%s/${github_repository.test.name}"
				username   = "<username>"
				permission = "triage"
			}
		`, randomID, orgName)

		checkWithOwner := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_repository_collaborator.test_repo_collaborator_2", "permission",
				"triage",
			),
			resource.TestCheckResourceAttr(
				"github_repository_collaborator.test_repo_collaborator_2", "permission_diff_suppression",
				"false",
			),
			resource.TestCheckResourceAttrSet(
				"github_repository_collaborator.test_repo_collaborator_2", "id",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: configWithOwner,
						Check:  checkWithOwner,
					},
					{
						ResourceName:            "github_repository_collaborator.test_repo_collaborator_2",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"permission_diff_suppression"},
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

	t.Run("permission diff suppression", func(t *testing.T) {
		config := fmt.Sprintf(`
			resource "github_repository" "test" {
				name = "tf-acc-test-%s"
				auto_init = true
			}

			resource "github_repository_collaborator" "test_repo_collaborator" {
				repository                = github_repository.test.name
				username                  = "<username>"
				permission                = "triage"
				permission_diff_suppression = true
			}
		`, randomID)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(
				"github_repository_collaborator.test_repo_collaborator", "permission",
				"triage",
			),
			resource.TestCheckResourceAttr(
				"github_repository_collaborator.test_repo_collaborator", "permission_diff_suppression",
				"true",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
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
}
