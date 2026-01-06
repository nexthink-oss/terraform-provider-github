package github

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-github/v81/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGithubRepositoryCustomProperties() *schema.Resource {
	return &schema.Resource{
		Description: "Manages custom properties for a GitHub repository. This resource allows you to set multiple custom properties on a single repository.",
		Create:      resourceGithubRepositoryCustomPropertiesCreateOrUpdate,
		Read:        resourceGithubRepositoryCustomPropertiesRead,
		Update:      resourceGithubRepositoryCustomPropertiesCreateOrUpdate,
		Delete:      resourceGithubRepositoryCustomPropertiesDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"repository_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the repository.",
			},
			"property": {
				Type:        schema.TypeSet,
				Required:    true,
				MinItems:    1,
				Description: "Set of custom properties for this repository.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Name of the custom property.",
						},
						"value": {
							Type:        schema.TypeSet,
							Required:    true,
							MinItems:    1,
							Description: "Value(s) of the custom property.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
					},
				},
				Set: resourceGithubRepositoryCustomPropertiesHash,
			},
		},
	}
}

// resourceGithubRepositoryCustomPropertiesHash creates a hash for a property block
// that includes both the name and all values to ensure changes are detected
func resourceGithubRepositoryCustomPropertiesHash(v any) int {
	raw := v.(map[string]any)
	name := raw["name"].(string)

	// Get the values set and convert to sorted string slice for consistent hashing
	valueSet := raw["value"].(*schema.Set)
	values := make([]string, 0, valueSet.Len())
	for _, val := range valueSet.List() {
		values = append(values, val.(string))
	}
	sort.Strings(values)

	// Create a composite string that includes both name and all values
	composite := fmt.Sprintf("%s:%s", name, strings.Join(values, ","))
	return schema.HashString(composite)
}

func resourceGithubRepositoryCustomPropertiesCreateOrUpdate(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	ctx := context.Background()
	owner := meta.(*Owner).name

	repoName := d.Get("repository_name").(string)
	properties := d.Get("property").(*schema.Set).List()

	// Get all organization custom property definitions to determine types
	orgProperties, _, err := client.Organizations.GetAllCustomProperties(ctx, owner)
	if err != nil {
		return fmt.Errorf("failed to get organization custom properties: %w", err)
	}

	// Create a map of property names to their types
	propertyTypes := make(map[string]string)
	for _, prop := range orgProperties {
		if prop.PropertyName != nil {
			propertyTypes[*prop.PropertyName] = prop.ValueType
		}
	}

	// Build custom properties for this repository
	customProperties := make([]*github.CustomPropertyValue, 0, len(properties))

	for _, propBlock := range properties {
		propMap := propBlock.(map[string]any)
		propertyName := propMap["name"].(string)
		propertyValues := expandStringList(propMap["value"].(*schema.Set).List())

		customProperty := &github.CustomPropertyValue{
			PropertyName: propertyName,
		}

		// Set the value based on property type
		propertyType := propertyTypes[propertyName]
		if propertyType == MULTI_SELECT {
			// multi_select always needs an array, even with one value
			customProperty.Value = propertyValues
		} else {
			// single_select, string, true_false need a single string
			if len(propertyValues) > 0 {
				customProperty.Value = propertyValues[0]
			}
		}

		customProperties = append(customProperties, customProperty)
	}

	// Update all properties for this repository in a single API call
	_, err = client.Repositories.CreateOrUpdateCustomProperties(ctx, owner, repoName, customProperties)
	if err != nil {
		return fmt.Errorf("failed to update custom properties for repository %s: %w", repoName, err)
	}

	d.SetId(buildCustomPropertiesID(owner, repoName))

	return resourceGithubRepositoryCustomPropertiesRead(d, meta)
}

func resourceGithubRepositoryCustomPropertiesRead(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	ctx := context.Background()

	owner, repoName, err := parseCustomPropertiesID(d.Id())
	if err != nil {
		return err
	}

	// Get current properties from state to know which ones we're managing
	propertiesFromState := d.Get("property").(*schema.Set).List()
	managedPropertyNames := make(map[string]bool)

	for _, propBlock := range propertiesFromState {
		propMap := propBlock.(map[string]any)
		propertyName := propMap["name"].(string)
		managedPropertyNames[propertyName] = true
	}

	// Read actual properties from GitHub
	allCustomProperties, _, err := client.Repositories.GetAllCustomPropertyValues(ctx, owner, repoName)
	if err != nil {
		return fmt.Errorf("failed to read custom properties for repository %s: %w", repoName, err)
	}

	// Filter to only the properties we're managing
	managedProperties := make([]any, 0)

	for _, prop := range allCustomProperties {
		if managedPropertyNames[prop.PropertyName] {
			propertyValue, err := parseRepositoryCustomPropertyValueToStringSlice(prop)
			if err != nil {
				return fmt.Errorf("failed to parse property %s for repository %s: %w", prop.PropertyName, repoName, err)
			}

			managedProperties = append(managedProperties, map[string]any{
				"name":  prop.PropertyName,
				"value": propertyValue,
			})
		}
	}

	// If no managed properties exist anymore, the resource should be removed
	if len(managedProperties) == 0 {
		d.SetId("")
		return nil
	}

	_ = d.Set("repository_name", repoName)
	_ = d.Set("property", managedProperties)

	return nil
}

func resourceGithubRepositoryCustomPropertiesDelete(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	ctx := context.Background()

	// Parse the ID to reliably get owner and repoName
	owner, repoName, err := parseCustomPropertiesID(d.Id())
	if err != nil {
		return err
	}

	// Get properties from state - these are the ones we need to remove
	properties := d.Get("property").(*schema.Set).List()

	// If there are no properties in state, nothing to delete
	if len(properties) == 0 {
		return nil
	}

	// Build list of properties to remove (set to nil)
	customProperties := make([]*github.CustomPropertyValue, 0, len(properties))

	for _, propBlock := range properties {
		propMap := propBlock.(map[string]any)
		propertyName := propMap["name"].(string)

		customProperty := &github.CustomPropertyValue{
			PropertyName: propertyName,
			Value:        nil,
		}

		customProperties = append(customProperties, customProperty)
	}

	// Remove all properties for this repository in a single API call
	_, err = client.Repositories.CreateOrUpdateCustomProperties(ctx, owner, repoName, customProperties)
	if err != nil {
		return fmt.Errorf("failed to delete custom properties for repository %s: %w", repoName, err)
	}

	return nil
}

// buildCustomPropertiesID creates an ID from owner and repository name
func buildCustomPropertiesID(owner string, repoName string) string {
	return fmt.Sprintf("%s:%s", owner, repoName)
}

// parseCustomPropertiesID parses the resource ID
func parseCustomPropertiesID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid ID format: %s (expected owner:repo)", id)
	}

	owner := parts[0]
	repoName := parts[1]

	if owner == "" || repoName == "" {
		return "", "", fmt.Errorf("invalid ID format: %s", id)
	}

	return owner, repoName, nil
}
