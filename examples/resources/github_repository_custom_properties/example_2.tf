resource "github_repository" "repos" {
  for_each = toset(["api-server", "web-app", "worker-service"])

  name = each.key
}

resource "github_repository_custom_properties" "repos" {
  for_each = github_repository.repos

  repository_name = each.value.name

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
