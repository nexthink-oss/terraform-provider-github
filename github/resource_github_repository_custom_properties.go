package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGithubRepositoryCustomProperties() *schema.Resource {
	return &schema.Resource{
		Description: "Manages custom properties for multiple GitHub repositories with a single resource. This allows you to set multiple custom properties across multiple repositories efficiently.",
		Create:      resourceGithubRepositoryCustomPropertiesCreateOrUpdate,
		Read:        resourceGithubRepositoryCustomPropertiesRead,
		Update:      resourceGithubRepositoryCustomPropertiesCreateOrUpdate,
		Delete:      resourceGithubRepositoryCustomPropertiesDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				Description: "List of repositories to apply custom properties to.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"repository_name": {
							Type:        schema.TypeString,
							Required:    true,
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
							Set: func(v any) int {
								raw := v.(map[string]any)
								return schema.HashString(raw["name"].(string))
							},
						},
					},
				},
			},
		},
	}
}

func resourceGithubRepositoryCustomPropertiesCreateOrUpdate(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	ctx := context.Background()
	owner := meta.(*Owner).name

	repositories := d.Get("repository").([]any)

	// Process each repository
	for _, repoBlock := range repositories {
		repoMap := repoBlock.(map[string]any)
		repoName := repoMap["repository_name"].(string)
		properties := repoMap["property"].(*schema.Set).List()

		// Build custom properties for this repository
		customProperties := make([]*github.CustomPropertyValue, 0, len(properties))

		for _, propBlock := range properties {
			propMap := propBlock.(map[string]any)
			propertyName := propMap["name"].(string)
			propertyValues := expandStringList(propMap["value"].(*schema.Set).List())

			customProperty := &github.CustomPropertyValue{
				PropertyName: propertyName,
			}

			// Set the value - use single value if only one, otherwise use array
			if len(propertyValues) == 1 {
				customProperty.Value = propertyValues[0]
			} else {
				customProperty.Value = propertyValues
			}

			customProperties = append(customProperties, customProperty)
		}

		// Update all properties for this repository in a single API call
		_, err := client.Repositories.CreateOrUpdateCustomProperties(ctx, owner, repoName, customProperties)
		if err != nil {
			return fmt.Errorf("failed to update custom properties for repository %s: %w", repoName, err)
		}
	}

	// Generate ID from repository names
	repoNames := make([]string, 0, len(repositories))
	for _, repoBlock := range repositories {
		repoMap := repoBlock.(map[string]any)
		repoNames = append(repoNames, repoMap["repository_name"].(string))
	}
	d.SetId(buildCustomPropertiesID(owner, repoNames))

	return resourceGithubRepositoryCustomPropertiesRead(d, meta)
}

func resourceGithubRepositoryCustomPropertiesRead(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	ctx := context.Background()

	owner, repoNames, err := parseCustomPropertiesID(d.Id())
	if err != nil {
		return err
	}

	// Get current repositories from state
	repositoriesFromState := d.Get("repository").([]any)

	// Build a map of repository -> property names we're managing
	managedPropertiesByRepo := make(map[string]map[string]bool) // repo -> propertyName -> exists

	for _, repoBlock := range repositoriesFromState {
		repoMap := repoBlock.(map[string]any)
		repoName := repoMap["repository_name"].(string)
		properties := repoMap["property"].(*schema.Set).List()

		if managedPropertiesByRepo[repoName] == nil {
			managedPropertiesByRepo[repoName] = make(map[string]bool)
		}

		for _, propBlock := range properties {
			propMap := propBlock.(map[string]any)
			propertyName := propMap["name"].(string)
			managedPropertiesByRepo[repoName][propertyName] = true
		}
	}

	// Read actual properties from GitHub
	repositories := make([]any, 0)

	for _, repoName := range repoNames {
		allCustomProperties, _, err := client.Repositories.GetAllCustomPropertyValues(ctx, owner, repoName)
		if err != nil {
			return fmt.Errorf("failed to read custom properties for repository %s: %w", repoName, err)
		}

		// Filter to only the properties we're managing
		managedProperties := make([]any, 0)
		managedPropMap := managedPropertiesByRepo[repoName]

		for _, prop := range allCustomProperties {
			if managedPropMap[prop.PropertyName] {
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

		// Only include repositories that have managed properties
		if len(managedProperties) > 0 {
			repositories = append(repositories, map[string]any{
				"repository_name": repoName,
				"property":        managedProperties,
			})
		}
	}

	// If no repositories have managed properties anymore, the resource should be removed
	if len(repositories) == 0 {
		d.SetId("")
		return nil
	}

	d.SetId(buildCustomPropertiesID(owner, repoNames))
	_ = d.Set("repository", repositories)

	return nil
}

func resourceGithubRepositoryCustomPropertiesDelete(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	ctx := context.Background()
	owner := meta.(*Owner).name

	repositories := d.Get("repository").([]any)

	// Remove custom properties from each repository
	for _, repoBlock := range repositories {
		repoMap := repoBlock.(map[string]any)
		repoName := repoMap["repository_name"].(string)
		properties := repoMap["property"].(*schema.Set).List()

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
		_, err := client.Repositories.CreateOrUpdateCustomProperties(ctx, owner, repoName, customProperties)
		if err != nil {
			return fmt.Errorf("failed to delete custom properties for repository %s: %w", repoName, err)
		}
	}

	return nil
}

// buildCustomPropertiesID creates an ID from owner and repository names
func buildCustomPropertiesID(owner string, repoNames []string) string {
	// Sort repository names for consistency
	sortedRepos := make([]string, len(repoNames))
	copy(sortedRepos, repoNames)

	// Create a deterministic ID using owner and all repository names
	return fmt.Sprintf("%s:%s", owner, strings.Join(sortedRepos, ","))
}

// parseCustomPropertiesID parses the resource ID
func parseCustomPropertiesID(id string) (string, []string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid ID format: %s (expected owner:repo1,repo2,...)", id)
	}

	owner := parts[0]
	repoNames := strings.Split(parts[1], ",")

	if owner == "" || len(repoNames) == 0 {
		return "", nil, fmt.Errorf("invalid ID format: %s", id)
	}

	return owner, repoNames, nil
}
