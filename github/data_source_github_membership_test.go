package github

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubMembershipDataSource(t *testing.T) {

	t.Run("queries the membership for a user in a specified organization", func(t *testing.T) {

		config := fmt.Sprintf(`
			data "github_membership" "test" {
				username = "%s"
				organization = "%s"
			}
		`, testOwnerFunc(), testOrganization())

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_membership.test", "username", testOwnerFunc()),
			resource.TestCheckResourceAttrSet("data.github_membership.test", "role"),
			resource.TestCheckResourceAttrSet("data.github_membership.test", "etag"),
			resource.TestCheckResourceAttrSet("data.github_membership.test", "state"),
			resource.TestCheckResourceAttrSet("data.github_membership.test", "id"),
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

	t.Run("errors when querying with non-existent user", func(t *testing.T) {

		config := fmt.Sprintf(`
			data "github_membership" "test" {
				username = "%s"
				organization = "%s"
			}
		`, "!"+testOwnerFunc(), testOrganization())

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(`Not Found`),
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
