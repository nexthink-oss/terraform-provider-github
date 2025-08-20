package github

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubExternalGroupsDataSource(t *testing.T) {

	t.Run("queries external groups without error", func(t *testing.T) {

		config := `
			data "github_external_groups" "test" {}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet("data.github_external_groups.test", "id"),
			resource.TestCheckResourceAttrSet("data.github_external_groups.test", "external_groups.#"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
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

	t.Run("validates external groups structure", func(t *testing.T) {

		config := `
			data "github_external_groups" "test" {}
		`

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttrSet("data.github_external_groups.test", "id"),
			resource.TestCheckResourceAttrSet("data.github_external_groups.test", "external_groups.#"),
			// Check that if groups exist, they have the expected structure
			resource.TestCheckResourceAttr("data.github_external_groups.test", "external_groups.#", "0"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessMode(t, mode) },
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
