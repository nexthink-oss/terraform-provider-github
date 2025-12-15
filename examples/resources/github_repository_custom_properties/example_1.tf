resource "github_repository" "repo1" {
  name        = "example-repo"
  description = "Example repository"
}

resource "github_repository_custom_properties" "example" {
  repository_name = github_repository.repo1.name

  property {
    name  = "environment"
    value = ["production"]
  }

  property {
    name  = "team"
    value = ["platform-team"]
  }
}
