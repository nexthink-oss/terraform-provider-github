package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubOrganizationSecurityManagerResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationSecurityManagerConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("github_team.test", "id", "github_organization_security_manager.test", "id"),
					resource.TestCheckResourceAttrPair("github_team.test", "slug", "github_organization_security_manager.test", "team_slug"),
					resource.TestCheckResourceAttr("github_organization_security_manager.test", "team_slug", fmt.Sprintf("tf-acc-%s", randomID)),
				),
			},
		},
	})
}

func TestAccGithubOrganizationSecurityManagerResource_update(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationSecurityManagerConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("github_team.test", "id", "github_organization_security_manager.test", "id"),
					resource.TestCheckResourceAttrPair("github_team.test", "slug", "github_organization_security_manager.test", "team_slug"),
					resource.TestCheckResourceAttr("github_organization_security_manager.test", "team_slug", fmt.Sprintf("tf-acc-%s", randomID)),
				),
			},
			{
				Config: testAccGithubOrganizationSecurityManagerConfig_updated(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("github_team.test", "id", "github_organization_security_manager.test", "id"),
					resource.TestCheckResourceAttrPair("github_team.test", "slug", "github_organization_security_manager.test", "team_slug"),
					resource.TestCheckResourceAttr("github_organization_security_manager.test", "team_slug", fmt.Sprintf("tf-acc-updated-%s", randomID)),
				),
			},
		},
	})
}

func TestAccGithubOrganizationSecurityManagerResource_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationSecurityManagerConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrPair("github_team.test", "id", "github_organization_security_manager.test", "id"),
					resource.TestCheckResourceAttrPair("github_team.test", "slug", "github_organization_security_manager.test", "team_slug"),
					resource.TestCheckResourceAttr("github_organization_security_manager.test", "team_slug", fmt.Sprintf("tf-acc-%s", randomID)),
				),
			},
			{
				ResourceName:      "github_organization_security_manager.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccGithubOrganizationSecurityManagerConfig_basic(randomID string) string {
	return fmt.Sprintf(`
resource "github_team" "test" {
	name = "tf-acc-%s"
}

resource "github_organization_security_manager" "test" {
	team_slug = github_team.test.slug
}
`, randomID)
}

func testAccGithubOrganizationSecurityManagerConfig_updated(randomID string) string {
	return fmt.Sprintf(`
resource "github_team" "test" {
	name = "tf-acc-updated-%s"
}

resource "github_organization_security_manager" "test" {
	team_slug = github_team.test.slug
}
`, randomID)
}
