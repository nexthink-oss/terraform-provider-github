# This example demonstrates collaborative management of repository custom properties.
# It assumes the organization already defines the referenced custom properties.

resource "github_repository" "example" {
  name        = "example-shared-custom-props"
  description = "Repository with collaboratively managed custom properties"

  exclusive_custom_properties = false

  custom_property {
    name  = "deployment-environment"
    value = ["production"]
  }
}

resource "github_repository_custom_property" "compliance" {
  repository     = github_repository.example.name
  property_name  = "compliance-tier"
  property_type  = "string"
  property_value = ["regulated"]
}
