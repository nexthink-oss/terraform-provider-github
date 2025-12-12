resource "github_repository" "frontend" {
  name = "frontend-app"
}

resource "github_repository_custom_properties" "frontend" {
  repository_name = github_repository.frontend.name

  property {
    name  = "tech-stack"
    value = ["react", "typescript", "nextjs"]
  }

  property {
    name  = "languages"
    value = ["javascript", "typescript"]
  }
}
