terraform {
  required_providers {
    github = {
      source  = "isometry/github"
      version = "~> 7.0"
    }
  }
}

# Configure the GitHub Provider
provider "github" {}

# Add a user to the organization
resource "github_membership" "membership_for_user_x" {
  # ...
}
