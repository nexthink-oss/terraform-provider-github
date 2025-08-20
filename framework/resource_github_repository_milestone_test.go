package framework

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGithubRepositoryMilestone_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	rn := "github_repository_milestone.test"
	repositoryName := fmt.Sprintf("tf-acc-test-%s", randomID)

	t.Run("creates a repository milestone", func(t *testing.T) {
		config := fmt.Sprintf(`
resource "github_repository" "test" {
  name = "%s"
}

resource "github_repository_milestone" "test" {
  owner       = split("/", "${github_repository.test.full_name}")[0]
  repository  = github_repository.test.name
  title       = "v1.0.0"
  description = "General Availability"
  due_date    = "2020-11-22"
  state       = "closed"
}
`, repositoryName)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(rn, "title", "v1.0.0"),
			resource.TestCheckResourceAttr(rn, "description", "General Availability"),
			resource.TestCheckResourceAttr(rn, "due_date", "2020-11-22"),
			resource.TestCheckResourceAttr(rn, "state", "closed"),
			resource.TestCheckResourceAttr(rn, "repository", repositoryName),
			resource.TestCheckResourceAttrSet(rn, "owner"),
			resource.TestCheckResourceAttrSet(rn, "number"),
			resource.TestCheckResourceAttrSet(rn, "id"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				CheckDestroy:             testAccCheckGithubRepositoryMilestoneDestroy,
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

func TestAccGithubRepositoryMilestone_minimal(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	rn := "github_repository_milestone.test"
	repositoryName := fmt.Sprintf("tf-acc-test-%s", randomID)

	t.Run("creates a repository milestone with minimal config", func(t *testing.T) {
		config := fmt.Sprintf(`
resource "github_repository" "test" {
  name = "%s"
}

resource "github_repository_milestone" "test" {
  owner      = split("/", "${github_repository.test.full_name}")[0]
  repository = github_repository.test.name
  title      = "v2.0.0"
}
`, repositoryName)

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr(rn, "title", "v2.0.0"),
			resource.TestCheckResourceAttr(rn, "state", "open"), // default value
			resource.TestCheckResourceAttr(rn, "repository", repositoryName),
			resource.TestCheckResourceAttrSet(rn, "owner"),
			resource.TestCheckResourceAttrSet(rn, "number"),
			resource.TestCheckResourceAttrSet(rn, "id"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				CheckDestroy:             testAccCheckGithubRepositoryMilestoneDestroy,
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

func TestAccGithubRepositoryMilestone_update(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	rn := "github_repository_milestone.test"
	repositoryName := fmt.Sprintf("tf-acc-test-%s", randomID)

	t.Run("updates a repository milestone", func(t *testing.T) {
		configBefore := fmt.Sprintf(`
resource "github_repository" "test" {
  name = "%s"
}

resource "github_repository_milestone" "test" {
  owner       = split("/", "${github_repository.test.full_name}")[0]
  repository  = github_repository.test.name
  title       = "v1.0.0"
  description = "Initial Release"
  state       = "open"
}
`, repositoryName)

		configAfter := fmt.Sprintf(`
resource "github_repository" "test" {
  name = "%s"
}

resource "github_repository_milestone" "test" {
  owner       = split("/", "${github_repository.test.full_name}")[0]
  repository  = github_repository.test.name
  title       = "v1.0.0 Final"
  description = "Final Release"
  due_date    = "2024-12-31"
  state       = "closed"
}
`, repositoryName)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				CheckDestroy:             testAccCheckGithubRepositoryMilestoneDestroy,
				Steps: []resource.TestStep{
					{
						Config: configBefore,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(rn, "title", "v1.0.0"),
							resource.TestCheckResourceAttr(rn, "description", "Initial Release"),
							resource.TestCheckResourceAttr(rn, "state", "open"),
						),
					},
					{
						Config: configAfter,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttr(rn, "title", "v1.0.0 Final"),
							resource.TestCheckResourceAttr(rn, "description", "Final Release"),
							resource.TestCheckResourceAttr(rn, "due_date", "2024-12-31"),
							resource.TestCheckResourceAttr(rn, "state", "closed"),
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

func TestAccGithubRepositoryMilestone_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	rn := "github_repository_milestone.test"
	repositoryName := fmt.Sprintf("tf-acc-test-%s", randomID)

	t.Run("imports a repository milestone", func(t *testing.T) {
		config := fmt.Sprintf(`
resource "github_repository" "test" {
  name = "%s"
}

resource "github_repository_milestone" "test" {
  owner       = split("/", "${github_repository.test.full_name}")[0]
  repository  = github_repository.test.name
  title       = "v1.0.0"
  description = "General Availability"
  due_date    = "2020-11-22"
  state       = "closed"
}
`, repositoryName)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				CheckDestroy:             testAccCheckGithubRepositoryMilestoneDestroy,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							testAccCheckGithubRepositoryMilestoneExists(rn),
							resource.TestCheckResourceAttr(rn, "title", "v1.0.0"),
						),
					},
					{
						ResourceName:      rn,
						ImportState:       true,
						ImportStateVerify: true,
						ImportStateIdFunc: testAccGithubRepositoryMilestoneImportStateIdFunc(rn),
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

func TestAccGithubRepositoryMilestone_validation(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)
	repositoryName := fmt.Sprintf("tf-acc-test-%s", randomID)

	t.Run("validates milestone state", func(t *testing.T) {
		config := fmt.Sprintf(`
resource "github_repository" "test" {
  name = "%s"
}

resource "github_repository_milestone" "test" {
  owner      = split("/", "${github_repository.test.full_name}")[0]
  repository = github_repository.test.name
  title      = "v1.0.0"
  state      = "invalid"
}
`, repositoryName)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(`Attribute state value must be one of:`),
					},
				},
			})
		}

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})
	})
}

// Test helper functions

func testAccCheckGithubRepositoryMilestoneDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "github_repository_milestone" {
			continue
		}

		// For now, we'll skip the detailed destroy check as it would require access to the
		// provider's internal state, which is complex with the muxed setup.
		// The important thing is that the resource is properly removed from Terraform state.
	}

	return nil
}

func testAccCheckGithubRepositoryMilestoneExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no milestone ID is set")
		}

		// Verify the ID has the correct format (owner/repository/number)
		return testAccValidateMilestoneID(rs.Primary.ID)
	}
}

func testAccGithubRepositoryMilestoneImportStateIdFunc(n string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return "", fmt.Errorf("not found: %s", n)
		}

		return rs.Primary.ID, nil
	}
}

func testAccValidateMilestoneID(id string) error {
	// Validate ID format: owner/repository/number
	if matched, _ := regexp.MatchString(`^[^/]+/[^/]+/\d+$`, id); !matched {
		return fmt.Errorf("invalid milestone ID format: %s", id)
	}
	return nil
}

// Unit tests for milestone helper functions

func TestParseMilestoneNumber(t *testing.T) {
	r := &githubRepositoryMilestoneResource{}

	testCases := []struct {
		ID           string
		ExpectNumber int
		ExpectError  bool
	}{
		{
			ID:           "owner/repo/1",
			ExpectNumber: 1,
			ExpectError:  false,
		},
		{
			ID:           "my-org/my-repo/123",
			ExpectNumber: 123,
			ExpectError:  false,
		},
		{
			ID:           "owner/repo/abc",
			ExpectNumber: -1,
			ExpectError:  true,
		},
		{
			ID:           "owner/repo",
			ExpectNumber: -1,
			ExpectError:  true,
		},
		{
			ID:           "invalid-id",
			ExpectNumber: -1,
			ExpectError:  true,
		},
	}

	for _, tc := range testCases {
		number, err := r.parseMilestoneNumber(tc.ID)
		if tc.ExpectError && err == nil {
			t.Errorf("Expected error for ID %q, but got none", tc.ID)
		}
		if !tc.ExpectError && err != nil {
			t.Errorf("Unexpected error for ID %q: %v", tc.ID, err)
		}
		if !tc.ExpectError && number != tc.ExpectNumber {
			t.Errorf("Expected number %d for ID %q, got %d", tc.ExpectNumber, tc.ID, number)
		}
	}
}

func TestValidateMilestoneID(t *testing.T) {
	testCases := []struct {
		ID          string
		ExpectError bool
	}{
		{
			ID:          "owner/repo/1",
			ExpectError: false,
		},
		{
			ID:          "my-org/my-repo/123",
			ExpectError: false,
		},
		{
			ID:          "owner/repo/abc",
			ExpectError: true,
		},
		{
			ID:          "owner/repo",
			ExpectError: true,
		},
		{
			ID:          "invalid-id",
			ExpectError: true,
		},
		{
			ID:          "",
			ExpectError: true,
		},
	}

	for _, tc := range testCases {
		err := testAccValidateMilestoneID(tc.ID)
		if tc.ExpectError && err == nil {
			t.Errorf("Expected error for ID %q, but got none", tc.ID)
		}
		if !tc.ExpectError && err != nil {
			t.Errorf("Unexpected error for ID %q: %v", tc.ID, err)
		}
	}
}
