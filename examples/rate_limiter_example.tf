terraform {
  required_providers {
    github = {
      source  = "isometry/github"
      version = ">= 7.0"
    }
  }
}

# Example 1: Using the legacy rate limiter (default)
provider "github" {
  alias = "legacy"
  token = var.github_token
  
  # Legacy rate limiter with manual delays
  rate_limiter     = "legacy"
  write_delay_ms   = 1000
  read_delay_ms    = 0
  parallel_requests = false
  
  # Retry configuration for server errors
  max_retries      = 3
  retry_delay_ms   = 1000
  retryable_errors = [500, 502, 503, 504]
}

# Example 2: Using the advanced rate limiter
provider "github" {
  alias = "advanced"
  token = var.github_token
  
  # Advanced rate limiter - automatic GitHub rate limit handling
  rate_limiter = "advanced"
  
  # These settings are ignored when using advanced rate limiter:
  # write_delay_ms, read_delay_ms, parallel_requests
  
  # Retry configuration for server errors (still applies)
  max_retries      = 3
  retry_delay_ms   = 1000
  retryable_errors = [500, 502, 503, 504]
}

# Example 3: Advanced rate limiter with minimal configuration
provider "github" {
  alias = "minimal"
  token = var.github_token
  
  # Just enable advanced rate limiting, use defaults for everything else
  rate_limiter = "advanced"
}

variable "github_token" {
  description = "GitHub personal access token"
  type        = string
  sensitive   = true
}

# Example resource using the advanced provider
resource "github_repository" "example" {
  provider = github.advanced
  
  name        = "example-repo"
  description = "Example repository using advanced rate limiter"
  private     = true
}