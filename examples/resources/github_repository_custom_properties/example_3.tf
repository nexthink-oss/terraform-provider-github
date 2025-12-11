resource "github_repository" "frontend" {
  name = "frontend-app"
}

resource "github_repository" "backend" {
  name = "backend-app"
}

resource "github_repository_custom_properties" "tech_stacks" {
  repository {
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

  repository {
    repository_name = github_repository.backend.name
    property {
      name  = "tech-stack"
      value = ["nodejs", "express", "postgresql"]
    }
    property {
      name  = "languages"
      value = ["typescript", "sql"]
    }
  }
}
