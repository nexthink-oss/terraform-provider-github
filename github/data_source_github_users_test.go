package github

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// testUsersOwner returns the owner for testing, derived from environment variables
func testUsersOwner() string {
	owner := os.Getenv("GITHUB_OWNER")
	if owner == "" {
		owner = os.Getenv("GITHUB_TEST_OWNER")
	}
	return owner
}

func TestAccGithubUsersDataSource(t *testing.T) {

	t.Run("queries multiple accounts", func(t *testing.T) {

		config := fmt.Sprintf(`
			data "github_users" "test" {
				usernames = ["%[1]s", "!%[1]s"]
			}
		`, testUsersOwner())

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_users.test", "logins.#", "1"),
			resource.TestCheckResourceAttr("data.github_users.test", "logins.0", testUsersOwner()),
			resource.TestCheckResourceAttr("data.github_users.test", "node_ids.#", "1"),
			resource.TestCheckResourceAttr("data.github_users.test", "unknown_logins.#", "1"),
			resource.TestCheckResourceAttr("data.github_users.test", "unknown_logins.0", fmt.Sprintf("!%s", testUsersOwner())),
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

	t.Run("does not fail if called with empty list of usernames", func(t *testing.T) {

		config := `
			data "github_users" "test" {
				usernames = []
			}
		`

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_users.test", "logins.#", "0"),
			resource.TestCheckResourceAttr("data.github_users.test", "node_ids.#", "0"),
			resource.TestCheckResourceAttr("data.github_users.test", "unknown_logins.#", "0"),
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
