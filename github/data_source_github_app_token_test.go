package github

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubAppTokenDataSource(t *testing.T) {

	// Skip for now as this requires valid App credentials
	t.Skip("Skipping github_app_token data source tests - requires valid GitHub App credentials")

	t.Run("creates an application token without error", func(t *testing.T) {

		config := `
			data "github_app_token" "test" {
				app_id          = "123456789"
				installation_id = "987654321"
				pem_file        = "dummy-pem-content"
			}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_app_token.test", "app_id", "123456789"),
			resource.TestCheckResourceAttr("data.github_app_token.test", "installation_id", "987654321"),
			resource.TestCheckResourceAttrSet("data.github_app_token.test", "token"),
			resource.TestCheckResourceAttrSet("data.github_app_token.test", "pem_file"),
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
			testCase(t, anonymous)
		})

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})

	t.Run("handles invalid app credentials", func(t *testing.T) {

		config := `
			data "github_app_token" "test" {
				app_id          = "invalid-app-id"
				installation_id = "invalid-installation-id"
				pem_file        = "invalid-pem-file"
			}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(`Unable to Generate GitHub App Token`),
					},
				},
			})
		}

		t.Run("with an anonymous account", func(t *testing.T) {
			testCase(t, anonymous)
		})

		t.Run("with an individual account", func(t *testing.T) {
			testCase(t, individual)
		})

		t.Run("with an organization account", func(t *testing.T) {
			testCase(t, organization)
		})
	})
}
