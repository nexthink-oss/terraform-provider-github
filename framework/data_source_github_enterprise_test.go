package framework

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubEnterpriseDataSource(t *testing.T) {

	t.Run("queries an existing enterprise without error", func(t *testing.T) {

		config := fmt.Sprintf(`
			data "github_enterprise" "test" {
				slug = "%s"
			}
		`, testEnterprise())

		check := resource.ComposeTestCheckFunc(
			resource.TestCheckResourceAttr("data.github_enterprise.test", "slug", testEnterprise()),
			resource.TestCheckResourceAttrSet("data.github_enterprise.test", "id"),
			resource.TestCheckResourceAttrSet("data.github_enterprise.test", "database_id"),
			resource.TestCheckResourceAttrSet("data.github_enterprise.test", "name"),
			resource.TestCheckResourceAttrSet("data.github_enterprise.test", "created_at"),
			resource.TestCheckResourceAttrSet("data.github_enterprise.test", "url"),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessEnterpriseMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check:  check,
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})

	})

	t.Run("queries a non-existent enterprise and returns error", func(t *testing.T) {

		config := `
			data "github_enterprise" "test" {
				slug = "non-existent-enterprise-slug-that-should-not-exist"
			}
		`

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { skipUnlessEnterpriseMode(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
				Steps: []resource.TestStep{
					{
						Config:      config,
						ExpectError: regexp.MustCompile(`GitHub Enterprise Not Found`),
					},
				},
			})
		}

		t.Run("with an enterprise account", func(t *testing.T) {
			testCase(t, enterprise)
		})

	})
}
