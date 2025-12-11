resource "github_repository" "repo1" {
  name        = "example-repo-1"
  description = "First example repository"
}

resource "github_repository" "repo2" {
  name        = "example-repo-2"
  description = "Second example repository"
}

resource "github_repository_custom_properties" "example" {
  repository {
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

  repository {
    repository_name = github_repository.repo2.name
    property {
      name  = "environment"
      value = ["staging"]
    }
    property {
      name  = "team"
      value = ["dev-team"]
    }
  }
}
