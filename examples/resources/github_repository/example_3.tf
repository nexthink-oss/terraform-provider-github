# This example matches the user's requested format with native HCL lists
resource "github_repository" "example" {
  name        = "example"
  description = "My awesome codebase"

  custom_property {
    name  = "foo"
    value = ["bar"]
  }

  custom_property {
    name  = "boolean"
    value = ["false"]
  }

  custom_property {
    name  = "multiselect"
    value = ["goo", "zoo"]
  }

  custom_property {
    name  = "singleselect"
    value = ["acme"]
  }
}
