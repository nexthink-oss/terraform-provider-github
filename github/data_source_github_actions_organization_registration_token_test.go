package github

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubActionsOrganizationRegistrationTokenDataSource(t *testing.T) {

	t.Run("get an organization registration token without error", func(t *testing.T) {

		config := `
			data "github_actions_organization_registration_token" "test" {
			}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet("data.github_actions_organization_registration_token.test", "token"),
			resource.TestCheckResourceAttrSet("data.github_actions_organization_registration_token.test", "expires_at"),
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
}
