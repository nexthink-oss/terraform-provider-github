package framework

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGithubRepositoryWebhookResource_basic(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryWebhookConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_webhook.test", "active", "true"),
					resource.TestCheckResourceAttr("github_repository_webhook.test", "events.#", "1"),
					resource.TestCheckResourceAttr("github_repository_webhook.test", "configuration.#", "1"),
					resource.TestCheckResourceAttr("github_repository_webhook.test", "configuration.0.url", "https://google.de/webhook"),
					resource.TestCheckResourceAttr("github_repository_webhook.test", "configuration.0.content_type", "json"),
					resource.TestCheckResourceAttr("github_repository_webhook.test", "configuration.0.insecure_ssl", "true"),
					resource.TestCheckResourceAttrSet("github_repository_webhook.test", "id"),
					resource.TestCheckResourceAttrSet("github_repository_webhook.test", "url"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryWebhookResource_update(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryWebhookConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_webhook.test", "events.#", "1"),
				),
			},
			{
				Config: testAccGithubRepositoryWebhookConfig_updated(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_webhook.test", "configuration.0.secret", "secret"),
					resource.TestCheckResourceAttr("github_repository_webhook.test", "events.#", "1"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryWebhookResource_import(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryWebhookConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_webhook.test", "active", "true"),
					resource.TestCheckResourceAttr("github_repository_webhook.test", "events.#", "1"),
				),
			},
			{
				ResourceName:        "github_repository_webhook.test",
				ImportState:         true,
				ImportStateVerify:   true,
				ImportStateIdPrefix: fmt.Sprintf("tf-acc-test-%s/", randomID),
			},
		},
	})
}

func TestAccGithubRepositoryWebhookResource_organization(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, organization) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryWebhookConfig_basic(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_webhook.test", "active", "true"),
					resource.TestCheckResourceAttr("github_repository_webhook.test", "events.#", "1"),
					resource.TestCheckResourceAttr("github_repository_webhook.test", "configuration.#", "1"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryWebhookResource_multipleEvents(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryWebhookConfig_multipleEvents(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_webhook.test", "active", "true"),
					resource.TestCheckResourceAttr("github_repository_webhook.test", "events.#", "3"),
					resource.TestCheckTypeSetElemAttr("github_repository_webhook.test", "events.*", "push"),
					resource.TestCheckTypeSetElemAttr("github_repository_webhook.test", "events.*", "pull_request"),
					resource.TestCheckTypeSetElemAttr("github_repository_webhook.test", "events.*", "issues"),
				),
			},
		},
	})
}

func TestAccGithubRepositoryWebhookResource_inactive(t *testing.T) {
	randomID := acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t, individual) },
		ProtoV6ProviderFactories: testAccMuxedProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccGithubRepositoryWebhookConfig_inactive(randomID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("github_repository_webhook.test", "active", "false"),
					resource.TestCheckResourceAttr("github_repository_webhook.test", "events.#", "1"),
				),
			},
		},
	})
}

func testAccGithubRepositoryWebhookConfig_basic(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Terraform acceptance tests"
}

resource "github_repository_webhook" "test" {
  depends_on = [github_repository.test]
  repository = github_repository.test.name

  configuration {
    url          = "https://google.de/webhook"
    content_type = "json"
    insecure_ssl = true
  }

  events = ["pull_request"]
}
`, randomID)
}

func testAccGithubRepositoryWebhookConfig_updated(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Terraform acceptance tests"
}

resource "github_repository_webhook" "test" {
  depends_on = [github_repository.test]
  repository = github_repository.test.name

  configuration {
    secret       = "secret"
    url          = "https://google.de/webhook"
    content_type = "json"
    insecure_ssl = true
  }

  events = ["pull_request"]
}
`, randomID)
}

func testAccGithubRepositoryWebhookConfig_multipleEvents(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Terraform acceptance tests"
}

resource "github_repository_webhook" "test" {
  depends_on = [github_repository.test]
  repository = github_repository.test.name

  configuration {
    url          = "https://google.de/webhook"
    content_type = "json"
    insecure_ssl = false
  }

  events = ["push", "pull_request", "issues"]
}
`, randomID)
}

func testAccGithubRepositoryWebhookConfig_inactive(randomID string) string {
	return fmt.Sprintf(`
resource "github_repository" "test" {
  name        = "tf-acc-test-%s"
  description = "Terraform acceptance tests"
}

resource "github_repository_webhook" "test" {
  depends_on = [github_repository.test]
  repository = github_repository.test.name
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
