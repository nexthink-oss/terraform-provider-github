package github

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubOrganizationBlock_basic(t *testing.T) {
	if err := testAccCheckOrganization(); err != nil {
		t.Skipf("Skipping because %s.", err.Error())
	}

	rn := "github_organization_block.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationBlockConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "username", "cgriggs01"),
					resource.TestCheckResourceAttrSet(rn, "etag"),
				),
			},
			{
				ResourceName:            rn,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"etag"},
			},
		},
	})
}

const testAccGithubOrganizationBlockConfig = `
resource "github_organization_block" "test" {
  username = "cgriggs01"
}
`
