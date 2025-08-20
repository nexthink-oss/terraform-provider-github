package framework

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubUserExternalIdentityDataSource(t *testing.T) {
	if os.Getenv("ENTERPRISE_ACCOUNT") != "true" {
		t.Skip("Skipping because `ENTERPRISE_ACCOUNT` is not set or set to false")
	}

	testUser := os.Getenv("GITHUB_TEST_USER")
	if testUser == "" {
		t.Skip("Skipping because `GITHUB_TEST_USER` is not set")
	}

	t.Run("queries without error", func(t *testing.T) {
		config := fmt.Sprintf(`
		data "github_user_external_identity" "test" {
		  username = "%s"
		}`, testUser)

		check := resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrSet("data.github_user_external_identity.test", "id"),
			resource.TestCheckResourceAttrSet("data.github_user_external_identity.test", "login"),
			resource.TestCheckResourceAttr("data.github_user_external_identity.test", "username", testUser),
			resource.TestCheckResourceAttrSet("data.github_user_external_identity.test", "saml_identity.name_id"),
			resource.TestCheckResourceAttrSet("data.github_user_external_identity.test", "scim_identity.username"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
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

func TestAccGithubUserExternalIdentityDataSource_invalidUser(t *testing.T) {
	if os.Getenv("ENTERPRISE_ACCOUNT") != "true" {
		t.Skip("Skipping because `ENTERPRISE_ACCOUNT` is not set or set to false")
	}

	t.Run("fails with invalid username", func(t *testing.T) {
		config := `
		data "github_user_external_identity" "test" {
		  username = "nonexistent-user-12345"
		}`

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      config,
					ExpectError: regexp.MustCompile("There was no external identity found"),
				},
			},
		})
	})
}
