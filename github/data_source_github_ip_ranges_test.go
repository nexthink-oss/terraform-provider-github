package github

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubIpRangesDataSource(t *testing.T) {

	t.Run("queries GitHub IP ranges without error", func(t *testing.T) {

		config := `
			data "github_ip_ranges" "test" {}
		`

		check := resource.ComposeTestCheckFunc(
			// Check that the data source has an ID
			resource.TestCheckResourceAttr("data.github_ip_ranges.test", "id", "github-ip-ranges"),

			// Check that IP range lists are populated and contain expected data
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "hooks.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "web.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "api.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "git.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "github_enterprise_importer.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "packages.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "pages.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "importer.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "actions.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "dependabot.#"),

			// Check that IP ranges are in CIDR format by checking a few known ranges
			resource.TestMatchResourceAttr("data.github_ip_ranges.test", "hooks.0", regexp.MustCompile(`^(\d+\.){3}\d+/\d+$|^([0-9a-fA-F:]+)/\d+$`)),
			resource.TestMatchResourceAttr("data.github_ip_ranges.test", "git.0", regexp.MustCompile(`^(\d+\.){3}\d+/\d+$|^([0-9a-fA-F:]+)/\d+$`)),
			resource.TestMatchResourceAttr("data.github_ip_ranges.test", "pages.0", regexp.MustCompile(`^(\d+\.){3}\d+/\d+$|^([0-9a-fA-F:]+)/\d+$`)),
			resource.TestMatchResourceAttr("data.github_ip_ranges.test", "actions.0", regexp.MustCompile(`^(\d+\.){3}\d+/\d+$|^([0-9a-fA-F:]+)/\d+$`)),
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

	t.Run("validates IP range format", func(t *testing.T) {

		config := `
			data "github_ip_ranges" "test" {}
		`

		check := resource.ComposeTestCheckFunc(
			// Verify all expected attributes exist and have values
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "hooks.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "web.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "api.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "git.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "actions.#"),
			resource.TestCheckResourceAttrSet("data.github_ip_ranges.test", "pages.#"),

			// Verify that hooks has the expected number of ranges (6 as seen in the API response)
			resource.TestCheckResourceAttr("data.github_ip_ranges.test", "hooks.#", "6"),

			// Verify that pages has the expected number of ranges (10 as seen in the API response)
			resource.TestCheckResourceAttr("data.github_ip_ranges.test", "pages.#", "10"),
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

}
