locals {
  production_repos = ["api-server", "web-app", "worker-service"]
}

resource "github_repository_custom_properties" "production" {
  dynamic "repository" {
    for_each = local.production_repos
    content {
      repository_name = repository.value
      property {
        name  = "environment"
        value = ["production"]
      }
      property {
        name  = "critical"
        value = ["true"]
      }
      property {
        name  = "monitoring"
        value = ["enabled"]
      }
    }
  }
}
