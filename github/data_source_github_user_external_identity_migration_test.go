package github

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGithubUserExternalIdentityDataSource_Migration(t *testing.T) {
	if os.Getenv("ENTERPRISE_ACCOUNT") != "true" {
		t.Skip("Skipping because `ENTERPRISE_ACCOUNT` is not set or set to false")
	}

	testUser := os.Getenv("GITHUB_TEST_USER")
	if testUser == "" {
		t.Skip("Skipping because `GITHUB_TEST_USER` is not set")
	}

	t.Run("migration maintains compatibility", func(t *testing.T) {
		config := fmt.Sprintf(`
		data "github_user_external_identity" "test" {
		  username = "%s"
		}`, testUser)

		resource.Test(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					ExternalProviders: map[string]resource.ExternalProvider{
						"github": {
							Source:            "integrations/github",
							VersionConstraint: "~> 6.0",
						},
					},
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("data.github_user_external_identity.test", "id"),
						resource.TestCheckResourceAttrSet("data.github_user_external_identity.test", "login"),
						resource.TestCheckResourceAttr("data.github_user_external_identity.test", "username", testUser),
						resource.TestCheckResourceAttrSet("data.github_user_external_identity.test", "saml_identity.name_id"),
						resource.TestCheckResourceAttrSet("data.github_user_external_identity.test", "scim_identity.username"),
					),
				},
				{
					ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
					Config:                   config,
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						},
					},
				},
			},
		})
	})
}

func TestAccGithubUserExternalIdentityDataSource_MuxedProvider(t *testing.T) {
	if os.Getenv("ENTERPRISE_ACCOUNT") != "true" {
		t.Skip("Skipping because `ENTERPRISE_ACCOUNT` is not set or set to false")
	}

	testUser := os.Getenv("GITHUB_TEST_USER")
	if testUser == "" {
		t.Skip("Skipping because `GITHUB_TEST_USER` is not set")
	}

	t.Run("works with muxed provider", func(t *testing.T) {
		config := fmt.Sprintf(`
		data "github_user_external_identity" "test" {
		  username = "%s"
		}`, testUser)

		resource.Test(t, resource.TestCase{
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttrSet("data.github_user_external_identity.test", "id"),
						resource.TestCheckResourceAttrSet("data.github_user_external_identity.test", "login"),
						resource.TestCheckResourceAttr("data.github_user_external_identity.test", "username", testUser),
						resource.TestCheckResourceAttrSet("data.github_user_external_identity.test", "saml_identity.name_id"),
						resource.TestCheckResourceAttrSet("data.github_user_external_identity.test", "scim_identity.username"),
					),
				},
			},
		})
	})
}
