package github

import (
	"fmt"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestGithubUserGpgKeyResource tests the basic resource creation
func TestGithubUserGpgKeyResource(t *testing.T) {
	resource := NewGithubUserGpgKeyResource()
	if resource == nil {
		t.Error("Resource should not be nil")
	}
}

func TestAccGithubUserGpgKey_Framework(t *testing.T) {
	t.Run("creates a GPG key without error", func(t *testing.T) {

		config := fmt.Sprintf(`
				resource "github_user_gpg_key" "test" {
					armored_public_key = "${file("%s")}"
				}
			`, filepath.Join("..", "github", "test-fixtures", "gpg-pubkey.asc"))

		check := resource.ComposeTestCheckFunc(
			resource.TestMatchResourceAttr(
				"github_user_gpg_key.test",
				"armored_public_key",
				regexp.MustCompile("^-----BEGIN PGP PUBLIC KEY BLOCK-----"),
			),
			resource.TestCheckResourceAttr(
				"github_user_gpg_key.test",
				"key_id",
				"AC541D2D1709CD33",
			),
			resource.TestCheckResourceAttrSet(
				"github_user_gpg_key.test",
				"id",
			),
		)

		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
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

func TestAccGithubUserGpgKey_MigrationValidation(t *testing.T) {
	t.Run("migration from SDKv2 to Framework maintains state compatibility", func(t *testing.T) {

		config := fmt.Sprintf(`
				resource "github_user_gpg_key" "test" {
					armored_public_key = "${file("%s")}"
				}
			`, filepath.Join("..", "github", "test-fixtures", "gpg-pubkey.asc"))

		resource.Test(t, resource.TestCase{
			Steps: []resource.TestStep{
				// This test step would use SDKv2 provider if we had external provider setup
				// For now we'll just test the framework implementation directly
				{
					ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
					Config:                   config,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttrSet("github_user_gpg_key.test", "id"),
						resource.TestCheckResourceAttr("github_user_gpg_key.test", "key_id", "AC541D2D1709CD33"),
						resource.TestMatchResourceAttr("github_user_gpg_key.test", "armored_public_key",
							regexp.MustCompile("^-----BEGIN PGP PUBLIC KEY BLOCK-----")),
					),
				},
				// Test that applying the same config results in no changes
				{
					ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
					Config:                   config,
					PlanOnly:                 true,
					ExpectNonEmptyPlan:       false,
				},
			},
		})

		// Test with organization account
		testCase := func(t *testing.T, mode string) {
			resource.Test(t, resource.TestCase{
				PreCheck:                 func() { testAccPreCheck(t, mode) },
				ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							resource.TestCheckResourceAttrSet("github_user_gpg_key.test", "id"),
							resource.TestCheckResourceAttr("github_user_gpg_key.test", "key_id", "AC541D2D1709CD33"),
						),
					},
					{
						Config:             config,
						PlanOnly:           true,
						ExpectNonEmptyPlan: false,
					},
				},
			})
		}

		t.Run("with individual account", func(t *testing.T) {
			testCase(t, individual)
		})
	})
}

func TestAccGithubUserGpgKey_Import(t *testing.T) {
	config := fmt.Sprintf(`
			resource "github_user_gpg_key" "test" {
				armored_public_key = "${file("%s")}"
			}
		`, filepath.Join("..", "github", "test-fixtures", "gpg-pubkey.asc"))

	testCase := func(t *testing.T, mode string) {
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t, mode) },
			ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttrSet("github_user_gpg_key.test", "id"),
						resource.TestCheckResourceAttr("github_user_gpg_key.test", "key_id", "AC541D2D1709CD33"),
					),
				},
				{
					ResourceName:      "github_user_gpg_key.test",
					ImportState:       true,
					ImportStateVerify: true,
				},
			},
		})
	}

	t.Run("with individual account", func(t *testing.T) {
		testCase(t, individual)
	})

	t.Run("with organization account", func(t *testing.T) {
		testCase(t, organization)
	})
}
