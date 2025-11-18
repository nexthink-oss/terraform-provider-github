# GitHub Organization Custom Property

This example demonstrates how to create and manage custom properties for a GitHub organization.

## Usage

```hcl
resource "github_organization_custom_property" "environment" {
  name               = "environment"
  value_type         = "single_select"
  required           = true
  default_value      = "production"
  description        = "Prod or dev environment"
  allowed_values     = ["production", "development"]
  values_editable_by = "org_actors"
}
```

## Attributes

- `name` - (Required) Name of the custom property
- `value_type` - (Required) Type of the value for the property. Can be one of: `single_select`, `multi_select`, `string`, `true_false`
- `required` - (Optional) Whether the property is required. Defaults to `false`
- `default_value` - (Optional) Default value of the property
- `description` - (Optional) A short description of the property
- `allowed_values` - (Optional) List of allowed values for the property. Only applies when `value_type` is `single_select` or `multi_select`
- `values_editable_by` - (Optional) Who can edit the values of the property. Can be one of: `org_actors`, `org_and_repo_actors`. Defaults to `org_actors`

## Additional Examples

### Multi-select property

```hcl
resource "github_organization_custom_property" "tags" {
  name               = "tags"
  value_type         = "multi_select"
  required           = false
  description        = "Project tags"
  allowed_values     = ["frontend", "backend", "database", "api"]
  values_editable_by = "org_and_repo_actors"
}
```

### String property

```hcl
resource "github_organization_custom_property" "owner" {
  name               = "owner"
  value_type         = "string"
  required           = false
  description        = "Repository owner"
  values_editable_by = "org_actors"
}
```

### True/False property

```hcl
resource "github_organization_custom_property" "archived" {
  name               = "archived"
  value_type         = "true_false"
  required           = false
  description        = "Is this repository archived"
  values_editable_by = "org_actors"
}
```

## Notes

- Custom properties are defined at the organization level and can be applied to repositories within the organization
- When using `single_select` or `multi_select`, the `allowed_values` field is required
- The `name` and `value_type` fields cannot be changed after creation (they are ForceNew)
