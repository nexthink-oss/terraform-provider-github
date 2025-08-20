package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubOrganizationCustomRoleResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationCustomRoleConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("github_organization_custom_role.test", "id"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "description", "Test role description"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "base_role", "read"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "permissions.#", "2"),
					resource.TestCheckTypeSetElemAttr("github_organization_custom_role.test", "permissions.*", "reopen_issue"),
					resource.TestCheckTypeSetElemAttr("github_organization_custom_role.test", "permissions.*", "reopen_pull_request"),
				),
			},
		},
	})
}

func TestAccGithubOrganizationCustomRoleResource_update(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationCustomRoleConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("github_organization_custom_role.test", "id"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "description", "Test role description"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "base_role", "read"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "permissions.#", "2"),
				),
			},
			{
				Config: testAccGithubOrganizationCustomRoleConfig_updated(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("github_organization_custom_role.test", "id"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "name", fmt.Sprintf("tf-acc-test-rename-%s", randomID)),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "description", "Updated test role description after"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "base_role", "write"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "permissions.#", "3"),
					resource.TestCheckTypeSetElemAttr("github_organization_custom_role.test", "permissions.*", "reopen_issue"),
					resource.TestCheckTypeSetElemAttr("github_organization_custom_role.test", "permissions.*", "read_code_scanning"),
					resource.TestCheckTypeSetElemAttr("github_organization_custom_role.test", "permissions.*", "reopen_pull_request"),
				),
			},
		},
	})
}

func TestAccGithubOrganizationCustomRoleResource_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationCustomRoleConfig_import(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("github_organization_custom_role.test", "id"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "description", "Test role description"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "base_role", "read"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "permissions.#", "3"),
				),
			},
			{
				ResourceName:      "github_organization_custom_role.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGithubOrganizationCustomRoleResource_withoutDescription(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationCustomRoleConfig_withoutDescription(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("github_organization_custom_role.test", "id"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "name", fmt.Sprintf("tf-acc-test-%s", randomID)),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "base_role", "triage"),
					resource.TestCheckResourceAttr("github_organization_custom_role.test", "permissions.#", "1"),
					resource.TestCheckTypeSetElemAttr("github_organization_custom_role.test", "permissions.*", "reopen_issue"),
					// description should be empty/null when not set
					resource.TestCheckNoResourceAttr("github_organization_custom_role.test", "description"),
				),
			},
		},
	})
}

func testAccGithubOrganizationCustomRoleConfig_basic(randomID string) string {
	return fmt.Sprintf(`
resource "github_organization_custom_role" "test" {
  name        = "tf-acc-test-%s"
  description = "Test role description"
  base_role   = "read"
  permissions = [
    "reopen_issue",
    "reopen_pull_request",
  ]
}
`, randomID)
}

func testAccGithubOrganizationCustomRoleConfig_updated(randomID string) string {
	return fmt.Sprintf(`
resource "github_organization_custom_role" "test" {
  name        = "tf-acc-test-rename-%s"
  description = "Updated test role description after"
  base_role   = "write"
  permissions = [
    "reopen_issue",
    "read_code_scanning",
    "reopen_pull_request",
  ]
}
`, randomID)
}

func testAccGithubOrganizationCustomRoleConfig_import(randomID string) string {
	return fmt.Sprintf(`
resource "github_organization_custom_role" "test" {
  name        = "tf-acc-test-%s"
  description = "Test role description"
  base_role   = "read"
  permissions = [
    "reopen_issue",
    "reopen_pull_request",
    "read_code_scanning",
  ]
}
`, randomID)
}

func testAccGithubOrganizationCustomRoleConfig_withoutDescription(randomID string) string {
	return fmt.Sprintf(`
resource "github_organization_custom_role" "test" {
  name      = "tf-acc-test-%s"
  base_role = "triage"
  permissions = [
    "reopen_issue",
  ]
}
`, randomID)
}
