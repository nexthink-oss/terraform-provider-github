package github

import (
	"context"
	"fmt"
	"log"

	"github.com/google/go-github/v81/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	SINGLE_SELECT = "single_select"
	MULTI_SELECT  = "multi_select"
	STRING        = "string"
	TRUE_FALSE    = "true_false"
)

func resourceGithubOrganizationCustomProperty() *schema.Resource {
	return &schema.Resource{
		Description: "Creates and manages a custom property for a GitHub Organization.",
		Create:      resourceGithubOrganizationCustomPropertyCreate,
		Read:        resourceGithubOrganizationCustomPropertyRead,
		Update:      resourceGithubOrganizationCustomPropertyUpdate,
		Delete:      resourceGithubOrganizationCustomPropertyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the custom property.",
				ForceNew:    true,
			},
			"value_type": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "The type of the value for the property. Can be one of: single_select, multi_select, string, true_false.",
				ForceNew:         true,
				ValidateDiagFunc: toDiagFunc(validation.StringInSlice([]string{SINGLE_SELECT, MULTI_SELECT, STRING, TRUE_FALSE}, false), "value_type"),
			},
			"required": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether the property is required.",
			},
			"default_value": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Default value of the property.",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "A short description of the property.",
			},
			"allowed_values": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of allowed values for the property. Only applies when value_type is single_select or multi_select.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"values_editable_by": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "org_actors",
				Description:      "Who can edit the values of the property. Can be one of: org_actors, org_and_repo_actors.",
				ValidateDiagFunc: toDiagFunc(validation.StringInSlice([]string{"org_actors", "org_and_repo_actors"}, false), "values_editable_by"),
			},
		},
	}
}

func resourceGithubOrganizationCustomPropertyCreate(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	ctx := context.Background()
	orgName := meta.(*Owner).name

	err := checkOrganization(meta)
	if err != nil {
		return err
	}

	propertyName := d.Get("name").(string)
	valueType := d.Get("value_type").(string)

	property := &github.CustomProperty{
		PropertyName:     github.Ptr(propertyName),
		ValueType:        valueType,
		Required:         github.Ptr(d.Get("required").(bool)),
		Description:      github.Ptr(d.Get("description").(string)),
		ValuesEditableBy: github.Ptr(d.Get("values_editable_by").(string)),
	}

	// Set default value if provided
	if v, ok := d.GetOk("default_value"); ok {
		property.DefaultValue = github.Ptr(v.(string))
	}

	// Set allowed values if provided (only for select types)
	if v, ok := d.GetOk("allowed_values"); ok {
		allowedValues := expandStringList(v.([]any))
		if valueType == SINGLE_SELECT || valueType == MULTI_SELECT {
			property.AllowedValues = allowedValues
		} else {
			return fmt.Errorf("allowed_values can only be set for single_select or multi_select value types")
		}
	}

	// Validate that allowed_values is provided for select types
	if (valueType == SINGLE_SELECT || valueType == MULTI_SELECT) && property.AllowedValues == nil {
		return fmt.Errorf("allowed_values is required for %s value type", valueType)
	}

	_, _, err = client.Organizations.CreateOrUpdateCustomProperty(ctx, orgName, propertyName, property)
	if err != nil {
		return fmt.Errorf("error creating organization custom property %s: %s", propertyName, err)
	}

	d.SetId(buildTwoPartID(orgName, propertyName))
	return resourceGithubOrganizationCustomPropertyRead(d, meta)
}

func resourceGithubOrganizationCustomPropertyRead(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	ctx := context.Background()

	orgName, propertyName, err := parseTwoPartID(d.Id(), "organization", "propertyName")
	if err != nil {
		return err
	}

	err = checkOrganization(meta)
	if err != nil {
		return err
	}

	property, err := getOrganizationCustomProperty(ctx, client, orgName, propertyName)
	if err != nil {
		log.Printf("[WARN] Removing organization custom property %s from state because it no longer exists in GitHub", propertyName)
		d.SetId("")
		return nil
	}

	d.SetId(buildTwoPartID(orgName, propertyName))
	_ = d.Set("name", property.PropertyName)
	_ = d.Set("value_type", property.ValueType)
	_ = d.Set("required", property.Required)
	_ = d.Set("description", property.Description)
	_ = d.Set("values_editable_by", property.ValuesEditableBy)

	if property.DefaultValue != nil {
		_ = d.Set("default_value", property.DefaultValue)
	}

	if len(property.AllowedValues) > 0 {
		_ = d.Set("allowed_values", property.AllowedValues)
	}

	return nil
}

func resourceGithubOrganizationCustomPropertyUpdate(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	ctx := context.Background()
	orgName := meta.(*Owner).name

	err := checkOrganization(meta)
	if err != nil {
		return err
	}

	propertyName := d.Get("name").(string)
	valueType := d.Get("value_type").(string)

	property := &github.CustomProperty{
		PropertyName:     github.Ptr(propertyName),
		ValueType:        valueType,
		Required:         github.Ptr(d.Get("required").(bool)),
		Description:      github.Ptr(d.Get("description").(string)),
		ValuesEditableBy: github.Ptr(d.Get("values_editable_by").(string)),
	}

	// Set default value if provided
	if v, ok := d.GetOk("default_value"); ok {
		property.DefaultValue = github.Ptr(v.(string))
	}

	// Set allowed values if provided
	if v, ok := d.GetOk("allowed_values"); ok {
		allowedValues := expandStringList(v.([]any))
		if valueType == SINGLE_SELECT || valueType == MULTI_SELECT {
			property.AllowedValues = allowedValues
		}
	}

	_, _, err = client.Organizations.CreateOrUpdateCustomProperty(ctx, orgName, propertyName, property)
	if err != nil {
		return fmt.Errorf("error updating organization custom property %s: %s", propertyName, err)
	}

	return resourceGithubOrganizationCustomPropertyRead(d, meta)
}

func resourceGithubOrganizationCustomPropertyDelete(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	ctx := context.Background()

	orgName, propertyName, err := parseTwoPartID(d.Id(), "organization", "propertyName")
	if err != nil {
		return err
	}

	err = checkOrganization(meta)
	if err != nil {
		return err
	}

	_, err = client.Organizations.RemoveCustomProperty(ctx, orgName, propertyName)
	if err != nil {
		return fmt.Errorf("error deleting organization custom property %s: %s", propertyName, err)
	}

	return nil
}

// getOrganizationCustomProperty retrieves a specific custom property by name
func getOrganizationCustomProperty(ctx context.Context, client *github.Client, orgName, propertyName string) (*github.CustomProperty, error) {
	properties, _, err := client.Organizations.GetAllCustomProperties(ctx, orgName)
	if err != nil {
		return nil, err
	}

	for _, property := range properties {
		if property.PropertyName != nil && *property.PropertyName == propertyName {
			return property, nil
		}
	}

	return nil, fmt.Errorf("custom property %s not found in organization %s", propertyName, orgName)
}
