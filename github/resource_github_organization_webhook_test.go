package github

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubOrganizationWebhookResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationWebhookConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_organization_webhook.test", "active", "true"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "events.#", "1"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "configuration.#", "1"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "configuration.0.url", "https://google.de/webhook"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "configuration.0.content_type", "json"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "configuration.0.insecure_ssl", "true"),
					resource.TestCheckResourceAttrSet("github_organization_webhook.test", "id"),
					resource.TestCheckResourceAttrSet("github_organization_webhook.test", "url"),
				),
			},
		},
	})
}

func TestAccGithubOrganizationWebhookResource_update(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationWebhookConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_organization_webhook.test", "events.#", "1"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "configuration.0.url", "https://google.de/webhook"),
				),
			},
			{
				Config: testAccGithubOrganizationWebhookConfig_updated(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_organization_webhook.test", "configuration.0.url", "https://google.de/updated"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "configuration.0.secret", "secret"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "events.#", "1"),
				),
			},
		},
	})
}

func TestAccGithubOrganizationWebhookResource_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationWebhookConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_organization_webhook.test", "active", "true"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "events.#", "1"),
				),
			},
			{
				ResourceName:      "github_organization_webhook.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGithubOrganizationWebhookResource_multipleEvents(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationWebhookConfig_multipleEvents(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_organization_webhook.test", "active", "true"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "events.#", "3"),
					resource.TestCheckTypeSetElemAttr("github_organization_webhook.test", "events.*", "push"),
					resource.TestCheckTypeSetElemAttr("github_organization_webhook.test", "events.*", "pull_request"),
					resource.TestCheckTypeSetElemAttr("github_organization_webhook.test", "events.*", "issues"),
				),
			},
		},
	})
}

func TestAccGithubOrganizationWebhookResource_inactive(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationWebhookConfig_inactive(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_organization_webhook.test", "active", "false"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "events.#", "1"),
				),
			},
		},
	})
}

func TestAccGithubOrganizationWebhookResource_secret(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationWebhookConfig_secret(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_organization_webhook.test", "active", "true"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "events.#", "1"),
					resource.TestCheckResourceAttr("github_organization_webhook.test", "configuration.0.secret", "VGVycmFmb3JtUm9ja3MhCg=="),
				),
			},
		},
	})
}

func TestAccGithubOrganizationWebhookResource_contentType(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubOrganizationWebhookConfig_contentType(randomID, "form"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_organization_webhook.test", "configuration.0.content_type", "form"),
				),
			},
			{
				Config: testAccGithubOrganizationWebhookConfig_contentType(randomID, "json"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_organization_webhook.test", "configuration.0.content_type", "json"),
				),
			},
		},
	})
}

func testAccGithubOrganizationWebhookConfig_basic(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Terraform acceptance tests"
  auto_init   = true
}

resource "github_organization_webhook" "test" {
  depends_on = [github_repository.test]

  configuration {
    url          = "https://google.de/webhook"
    content_type = "json"
    insecure_ssl = true
  }

  events = ["pull_request"]
}
`, randomID)
}

func testAccGithubOrganizationWebhookConfig_updated(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Terraform acceptance tests"
  auto_init   = true
}

resource "github_organization_webhook" "test" {
  depends_on = [github_repository.test]

  configuration {
    secret       = "secret"
    url          = "https://google.de/updated"
    content_type = "json"
    insecure_ssl = true
  }

  events = ["pull_request"]
}
`, randomID)
}

func testAccGithubOrganizationWebhookConfig_multipleEvents(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Terraform acceptance tests"
  auto_init   = true
}

resource "github_organization_webhook" "test" {
  depends_on = [github_repository.test]

  configuration {
    url          = "https://google.de/webhook"
    content_type = "json"
    insecure_ssl = false
  }

  events = ["push", "pull_request", "issues"]
}
`, randomID)
}

func testAccGithubOrganizationWebhookConfig_inactive(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Terraform acceptance tests"
  auto_init   = true
}

resource "github_organization_webhook" "test" {
  depends_on = [github_repository.test]
  active     = false

  configuration {
    url          = "https://google.de/webhook"
    content_type = "json"
    insecure_ssl = false
  }

  events = ["pull_request"]
}
`, randomID)
}

func testAccGithubOrganizationWebhookConfig_secret(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Terraform acceptance tests"
  auto_init   = true
}

resource "github_organization_webhook" "test" {
  depends_on = [github_repository.test]

  configuration {
    url          = "https://google.de/webhook"
    content_type = "json"
    insecure_ssl = false
    secret       = "VGVycmFmb3JtUm9ja3MhCg=="
  }

  events = ["issues"]
}
`, randomID)
}

func testAccGithubOrganizationWebhookConfig_contentType(randomID string, contentType string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Terraform acceptance tests"
  auto_init   = true
}

resource "github_organization_webhook" "test" {
  depends_on = [github_repository.test]

  configuration {
    url          = "https://google.de/webhook"
    content_type = "%s"
    insecure_ssl = false
  }

  events = ["issues"]
}
`, randomID, contentType)
}
