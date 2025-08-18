resource "github_repository" "example" {
  name        = "example"
  description = "Example repository"
}

resource "github_repository_ruleset" "example" {
  name        = "example"
  repository  = github_repository.example.name
  target      = "branch"
  enforcement = "active"

  conditions {
    ref_name {
      include = ["~ALL"]
      exclude = []
    }
  }

  bypass_actors {
    actor_id    = 13473
    actor_type  = "Integration"
    bypass_mode = "always"
  }

  rules {
    creation                = true
    update                  = true
    deletion                = true
    required_linear_history = true
    required_signatures     = true

    required_deployments {
      required_deployment_environments = ["test"]
    }


  }
}
