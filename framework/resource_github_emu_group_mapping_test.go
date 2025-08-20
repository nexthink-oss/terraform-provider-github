package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccGithubEmuGroupMapping(t *testing.T) {
	t.Skip("github_emu_group_mapping: requires a GitHub Enterprise Account with EMU configuration")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, organization)
		},
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubEmuGroupMappingConfig("example-team", 12345),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_emu_group_mapping.test", "team_slug", "example-team"),
					resource.TestCheckResourceAttr("github_emu_group_mapping.test", "group_id", "12345"),
					resource.TestCheckResourceAttr("github_emu_group_mapping.test", "id", "teams/example-team/external-groups"),
					resource.TestCheckResourceAttrSet("github_emu_group_mapping.test", "etag"),
				),
			},
		},
	})
}

func TestAccGithubEmuGroupMapping_importBasic(t *testing.T) {
	t.Skip("github_emu_group_mapping: requires a GitHub Enterprise Account with EMU configuration")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t, organization)
		},
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubEmuGroupMappingConfig("example-team", 12345),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_emu_group_mapping.test", "team_slug", "example-team"),
				),
			},
			{
				ResourceName:      "github_emu_group_mapping.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccGithubEmuGroupMappingImportStateIdFunc("github_emu_group_mapping.test"),
			},
		},
	})
}

func testAccGithubEmuGroupMappingImportStateIdFunc(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}

		if rs.Primary.ID == "" {
			return "", fmt.Errorf("resource id not set")
		}

		// Return the group_id for import
		return rs.Primary.Attributes["group_id"], nil
	}
}

func testAccGithubEmuGroupMappingConfig(teamSlug string, groupID int) string {
	return fmt.Sprintf(`
resource "github_emu_group_mapping" "test" {
  team_slug = "%s"
  group_id  = %d
}
`, teamSlug, groupID)
}
