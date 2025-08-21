package github

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccProvider_basic tests basic provider configuration acceptance
func TestAccProvider_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderConfigured(),
				),
			},
		},
	})
}

// TestAccProvider_allAttributes tests provider with all configuration attributes
func TestAccProvider_allAttributes(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_allAttributes(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderConfigured(),
				),
			},
		},
	})
}

// TestAccProvider_environmentVariables tests provider configuration via environment variables
func TestAccProvider_environmentVariables(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env_token")
	t.Setenv("GITHUB_OWNER", "env_owner")
	t.Setenv("GITHUB_BASE_URL", "https://enterprise.example.com/")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_minimal(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderConfigured(),
				),
			},
		},
	})
}

// TestAccProvider_configOverridesEnv tests that explicit config overrides environment variables
func TestAccProvider_configOverridesEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env_token")
	t.Setenv("GITHUB_OWNER", "env_owner")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig_overrideEnv(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProviderConfigured(),
				),
			},
		},
	})
}

// TestAccProvider_parallelRequestsGitHubDotComError tests that parallel_requests=true fails for github.com
func TestAccProvider_parallelRequestsGitHubDotComError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccProviderConfig_parallelRequestsGitHubCom(),
				ExpectError: regexp.MustCompile("parallel_requests cannot be true when connecting to public github.com"),
			},
		},
	})
}

// TestAccProvider_negativeDelaysError tests that negative delay values produce errors
func TestAccProvider_negativeDelaysError(t *testing.T) {
	testCases := []struct {
		name        string
		config      string
		expectedErr string
	}{
		{
			name:        "negative read_delay_ms",
			config:      testAccProviderConfig_negativeReadDelay(),
			expectedErr: "read_delay_ms must be greater than or equal to 0",
		},
		{
			name:        "negative write_delay_ms",
			config:      testAccProviderConfig_negativeWriteDelay(),
			expectedErr: "write_delay_ms must be greater than 0ms",
		},
		{
			name:        "zero write_delay_ms",
			config:      testAccProviderConfig_zeroWriteDelay(),
			expectedErr: "write_delay_ms must be greater than 0ms",
		},
		{
			name:        "negative retry_delay_ms",
			config:      testAccProviderConfig_negativeRetryDelay(),
			expectedErr: "retry_delay_ms must be greater than or equal to 0ms",
		},
		{
			name:        "negative max_retries",
			config:      testAccProviderConfig_negativeMaxRetries(),
			expectedErr: "max_retries must be greater than or equal to 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						ExpectError: regexp.MustCompile(regexp.QuoteMeta(tc.expectedErr)),
					},
				},
			})
		})
	}
}

// TestAccProvider_retryableErrors tests various retryable_errors configurations
func TestAccProvider_retryableErrors(t *testing.T) {
	testCases := []struct {
		name   string
		config string
	}{
		{
			name:   "default retryable errors",
			config: testAccProviderConfig_defaultRetryableErrors(),
		},
		{
			name:   "custom retryable errors",
			config: testAccProviderConfig_customRetryableErrors(),
		},
		{
			name:   "single retryable error",
			config: testAccProviderConfig_singleRetryableError(),
		},
		{
			name:   "empty retryable errors with zero retries",
			config: testAccProviderConfig_zeroRetries(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: tc.config,
						Check: resource.ComposeTestCheckFunc(
							testAccCheckProviderConfigured(),
						),
					},
				},
			})
		})
	}
}

// Helper functions for test configurations

func testAccProviderConfig_basic() string {
	return `
provider "github" {
  token = "test_token"
  owner = "test_owner"
}
`
}

func testAccProviderConfig_allAttributes() string {
	return `
provider "github" {
  token               = "test_token"
  owner               = "test_owner"  
  base_url            = "https://enterprise.example.com/"
  parallel_requests   = true
  read_delay_ms       = 100
  write_delay_ms      = 1500
  retry_delay_ms      = 2000
  max_retries         = 5
  retryable_errors    = [500, 502, 503, 504, 429]
  insecure            = true
}
`
}

func testAccProviderConfig_minimal() string {
	return `
provider "github" {}
`
}

func testAccProviderConfig_overrideEnv() string {
	return `
provider "github" {
  token = "config_token"
  owner = "config_owner"
}
`
}

func testAccProviderConfig_parallelRequestsGitHubCom() string {
	return `
provider "github" {
  token             = "test_token"
  base_url          = "https://api.github.com/"
  parallel_requests = true
}
`
}

func testAccProviderConfig_negativeReadDelay() string {
	return `
provider "github" {
  token         = "test_token"
  read_delay_ms = -100
}
`
}

func testAccProviderConfig_negativeWriteDelay() string {
	return `
provider "github" {
  token          = "test_token"
  write_delay_ms = -500
}
`
}

func testAccProviderConfig_zeroWriteDelay() string {
	return `
provider "github" {
  token          = "test_token"
  write_delay_ms = 0
}
`
}

func testAccProviderConfig_negativeRetryDelay() string {
	return `
provider "github" {
  token          = "test_token"
  retry_delay_ms = -1000
}
`
}

func testAccProviderConfig_negativeMaxRetries() string {
	return `
provider "github" {
  token       = "test_token"
  max_retries = -1
}
`
}

func testAccProviderConfig_defaultRetryableErrors() string {
	return `
provider "github" {
  token       = "test_token"
  max_retries = 3
}
`
}

func testAccProviderConfig_customRetryableErrors() string {
	return `
provider "github" {
  token            = "test_token"
  max_retries      = 2
  retryable_errors = [500, 429, 503]
}
`
}

func testAccProviderConfig_singleRetryableError() string {
	return `
provider "github" {
  token            = "test_token"
  max_retries      = 1
  retryable_errors = [503]
}
`
}

func testAccProviderConfig_zeroRetries() string {
	return `
provider "github" {
  token            = "test_token"
  max_retries      = 0
  retryable_errors = [500]
}
`
}

// Test check functions

func testAccCheckProviderConfigured() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// This is a basic check that the provider was configured without errors
		// In a real scenario, you might check specific provider state or make API calls

		// For now, we just verify that we have a provider configured
		if s.RootModule().Resources == nil {
			return fmt.Errorf("no resources found in state")
		}

		// The fact that we got this far means the provider configuration was accepted
		return nil
	}
}

// Additional test configurations for edge cases

// TestAccProvider_enterpriseURL tests provider with various enterprise URL formats
func TestAccProvider_enterpriseURL(t *testing.T) {
	testCases := []struct {
		name string
		url  string
	}{
		{
			name: "enterprise with api path",
			url:  "https://github.enterprise.com/api/v3/",
		},
		{
			name: "enterprise without trailing slash",
			url:  "https://github.enterprise.com",
		},
		{
			name: "GHEC data residency URL",
			url:  "https://customer.ghe.com",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := fmt.Sprintf(`
provider "github" {
  token             = "test_token"
  base_url          = "%s"
  parallel_requests = true
}
`, tc.url)

			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							testAccCheckProviderConfigured(),
						),
					},
				},
			})
		})
	}
}

// TestAccProvider_delayRanges tests various delay value ranges
func TestAccProvider_delayRanges(t *testing.T) {
	testCases := []struct {
		name   string
		config string
	}{
		{
			name: "minimum delays",
			config: `
provider "github" {
  token          = "test_token"
  read_delay_ms  = 0
  write_delay_ms = 1
  retry_delay_ms = 0
}
`,
		},
		{
			name: "moderate delays",
			config: `
provider "github" {
  token          = "test_token"
  read_delay_ms  = 500
  write_delay_ms = 1000
  retry_delay_ms = 750
}
`,
		},
		{
			name: "high delays",
			config: `
provider "github" {
  token          = "test_token"
  read_delay_ms  = 2000
  write_delay_ms = 3000
  retry_delay_ms = 5000
}
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config: tc.config,
						Check: resource.ComposeTestCheckFunc(
							testAccCheckProviderConfigured(),
						),
					},
				},
			})
		})
	}
}
