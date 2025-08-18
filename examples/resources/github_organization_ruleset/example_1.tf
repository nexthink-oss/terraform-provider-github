resource "github_organization_ruleset" "example" {
  name        = "example"
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

    branch_name_pattern {
      name     = "example"
      negate   = false
      operator = "starts_with"
      pattern  = "ex"
    }
  }
}
