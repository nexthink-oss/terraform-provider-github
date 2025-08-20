package framework

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

const (
	SINGLE_SELECT = "single_select"
	MULTI_SELECT  = "multi_select"
	STRING        = "string"
	TRUE_FALSE    = "true_false"
)

var (
	_ resource.Resource                = &githubRepositoryCustomPropertyResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryCustomPropertyResource{}
	_ resource.ResourceWithImportState = &githubRepositoryCustomPropertyResource{}
)

type githubRepositoryCustomPropertyResource struct {
	client *githubpkg.Owner
}

type githubRepositoryCustomPropertyResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Repository    types.String `tfsdk:"repository"`
	PropertyType  types.String `tfsdk:"property_type"`
	PropertyName  types.String `tfsdk:"property_name"`
	PropertyValue types.Set    `tfsdk:"property_value"`
}

func NewGithubRepositoryCustomPropertyResource() resource.Resource {
	return &githubRepositoryCustomPropertyResource{}
}

func (r *githubRepositoryCustomPropertyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_custom_property"
}

func (r *githubRepositoryCustomPropertyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a specific custom property for a GitHub repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the repository custom property (owner/repository/property_name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "Name of the repository which the custom properties should be on.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"property_type": schema.StringAttribute{
				Description: "Type of the custom property",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(SINGLE_SELECT, MULTI_SELECT, STRING, TRUE_FALSE),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"property_name": schema.StringAttribute{
				Description: "Name of the custom property.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"property_value": schema.SetAttribute{
				Description: "Value of the custom property.",
				ElementType: types.StringType,
				Required:    true,
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.All(
						&customPropertyValueValidator{},
					),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *githubRepositoryCustomPropertyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*githubpkg.Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *githubRepositoryCustomPropertyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubRepositoryCustomPropertyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()
	propertyName := data.PropertyName.ValueString()
	propertyType := data.PropertyType.ValueString()

	// Convert property_value Set to []string
	propertyValueElements := make([]types.String, 0, len(data.PropertyValue.Elements()))
	resp.Diagnostics.Append(data.PropertyValue.ElementsAs(ctx, &propertyValueElements, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	propertyValue := make([]string, len(propertyValueElements))
	for i, elem := range propertyValueElements {
		propertyValue[i] = elem.ValueString()
	}

	if len(propertyValue) == 0 {
		resp.Diagnostics.AddError(
			"Invalid Property Value",
			"Property value cannot be empty",
		)
		return
	}

	customProperty := github.CustomPropertyValue{
		PropertyName: propertyName,
	}

	// The propertyValue can either be a list of strings or a string
	switch propertyType {
	case SINGLE_SELECT, TRUE_FALSE, STRING:
		if len(propertyValue) > 1 {
			resp.Diagnostics.AddError(
				"Invalid Property Value",
				fmt.Sprintf("Property type %s accepts only a single value, but %d values were provided", propertyType, len(propertyValue)),
			)
			return
		}
		customProperty.Value = propertyValue[0]
	case MULTI_SELECT:
		customProperty.Value = propertyValue
	default:
		resp.Diagnostics.AddError(
			"Invalid Property Type",
			fmt.Sprintf("Custom property type is not valid: %v", propertyType),
		)
		return
	}

	_, err := client.Repositories.CreateOrUpdateCustomProperties(ctx, owner, repoName, []*github.CustomPropertyValue{&customProperty})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Repository Custom Property",
			fmt.Sprintf("An unexpected error occurred when creating the repository custom property: %s", err.Error()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%s", owner, repoName, propertyName))

	tflog.Debug(ctx, "created GitHub repository custom property", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"repository":    repoName,
		"property_name": propertyName,
		"property_type": propertyType,
	})

	// Read the created resource to populate all computed fields
	r.readGithubRepositoryCustomProperty(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryCustomPropertyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubRepositoryCustomPropertyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubRepositoryCustomProperty(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryCustomPropertyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This resource doesn't support updates since all attributes are ForceNew/RequiresReplace
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"This resource does not support updates. All attributes require resource replacement.",
	)
}

func (r *githubRepositoryCustomPropertyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubRepositoryCustomPropertyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	// Parse ID to get components
	owner, repoName, propertyName, err := r.parseThreePartID(data.ID.ValueString(), "owner", "repoName", "propertyName")
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Resource ID",
			fmt.Sprintf("Unable to parse resource ID: %s", err.Error()),
		)
		return
	}

	customProperty := github.CustomPropertyValue{
		PropertyName: propertyName,
		Value:        nil,
	}

	_, err = client.Repositories.CreateOrUpdateCustomProperties(ctx, owner, repoName, []*github.CustomPropertyValue{&customProperty})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Repository Custom Property",
			fmt.Sprintf("An unexpected error occurred when deleting the repository custom property: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub repository custom property", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"repository":    repoName,
		"property_name": propertyName,
	})
}

func (r *githubRepositoryCustomPropertyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "<owner>/<repository>/<property_name>"
	owner, repoName, propertyName, err := r.parseThreePartID(req.ID, "owner", "repository", "property_name")
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Invalid ID specified. Supplied ID must be written as <owner>/<repository>/<property_name>. Got: %q", req.ID),
		)
		return
	}

	client := r.client.V3Client()

	// Check if the custom property exists
	wantedCustomPropertyValue, err := r.readRepositoryCustomPropertyValue(ctx, client, owner, repoName, propertyName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import Repository Custom Property",
			fmt.Sprintf("Unable to read repository custom property for import: %s", err.Error()),
		)
		return
	}

	// Create the resource data
	propertyValueSet, diags := types.SetValueFrom(ctx, types.StringType, wantedCustomPropertyValue)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := &githubRepositoryCustomPropertyResourceModel{
		ID:            types.StringValue(req.ID),
		Repository:    types.StringValue(repoName),
		PropertyName:  types.StringValue(propertyName),
		PropertyValue: propertyValueSet,
	}

	// Read to populate all fields including property_type
	r.readGithubRepositoryCustomProperty(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubRepositoryCustomPropertyResource) parseThreePartID(id, left, center, right string) (string, string, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("Unexpected ID format (%q), expected %s/%s/%s", id, left, center, right)
	}
	return parts[0], parts[1], parts[2], nil
}

func (r *githubRepositoryCustomPropertyResource) readRepositoryCustomPropertyValue(ctx context.Context, client *github.Client, owner, repoName, propertyName string) ([]string, error) {
	allCustomProperties, _, err := client.Repositories.GetAllCustomPropertyValues(ctx, owner, repoName)
	if err != nil {
		return nil, err
	}

	var wantedCustomProperty *github.CustomPropertyValue
	for _, customProperty := range allCustomProperties {
		if customProperty.PropertyName == propertyName {
			wantedCustomProperty = customProperty
		}
	}

	if wantedCustomProperty == nil {
		return nil, fmt.Errorf("could not find a custom property with name: %s", propertyName)
	}

	wantedPropertyValue, err := r.parseRepositoryCustomPropertyValueToStringSlice(wantedCustomProperty)
	if err != nil {
		return nil, err
	}

	return wantedPropertyValue, nil
}

func (r *githubRepositoryCustomPropertyResource) parseRepositoryCustomPropertyValueToStringSlice(prop *github.CustomPropertyValue) ([]string, error) {
	switch value := prop.Value.(type) {
	case string:
		return []string{value}, nil
	case []string:
		return value, nil
	case []interface{}:
		result := make([]string, len(value))
		for i, v := range value {
			if str, ok := v.(string); ok {
				result[i] = str
			} else {
				return nil, fmt.Errorf("custom property value contains non-string element: %v", v)
			}
		}
		return result, nil
	default:
		return nil, fmt.Errorf("custom property value couldn't be parsed as a string or a list of strings: %v", value)
	}
}

func (r *githubRepositoryCustomPropertyResource) readGithubRepositoryCustomProperty(ctx context.Context, data *githubRepositoryCustomPropertyResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	// Parse ID to get components
	owner, repoName, propertyName, err := r.parseThreePartID(data.ID.ValueString(), "owner", "repository", "property_name")
	if err != nil {
		diags.AddError(
			"Invalid Resource ID",
			fmt.Sprintf("Unable to parse resource ID: %s", err.Error()),
		)
		return
	}

	wantedCustomPropertyValue, err := r.readRepositoryCustomPropertyValue(ctx, client, owner, repoName, propertyName)
	if err != nil {
		diags.AddError(
			"Unable to Read Repository Custom Property",
			fmt.Sprintf("An unexpected error occurred when reading the repository custom property: %s", err.Error()),
		)
		return
	}

	// We need to determine the property_type from the organization custom properties
	orgCustomProperties, _, err := client.Organizations.GetAllCustomProperties(ctx, owner)
	if err != nil {
		diags.AddError(
			"Unable to Read Organization Custom Properties",
			fmt.Sprintf("Unable to read organization custom properties to determine property type: %s", err.Error()),
		)
		return
	}

	var propertyType string
	for _, orgProperty := range orgCustomProperties {
		if orgProperty.GetPropertyName() == propertyName {
			propertyType = orgProperty.ValueType
			break
		}
	}

	if propertyType == "" {
		diags.AddError(
			"Property Type Not Found",
			fmt.Sprintf("Unable to determine property type for custom property: %s", propertyName),
		)
		return
	}

	propertyValueSet, setDiags := types.SetValueFrom(ctx, types.StringType, wantedCustomPropertyValue)
	diags.Append(setDiags...)
	if diags.HasError() {
		return
	}

	data.Repository = types.StringValue(repoName)
	data.PropertyName = types.StringValue(propertyName)
	data.PropertyType = types.StringValue(propertyType)
	data.PropertyValue = propertyValueSet

	tflog.Debug(ctx, "successfully read GitHub repository custom property", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"repository":    repoName,
		"property_name": propertyName,
		"property_type": propertyType,
	})
}

// Custom validator to ensure property_value has correct number of elements based on property_type
type customPropertyValueValidator struct{}

func (v *customPropertyValueValidator) Description(ctx context.Context) string {
	return "validates that property_value size matches property_type requirements"
}

func (v *customPropertyValueValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *customPropertyValueValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	// Skip validation if value is null or unknown
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// Get the property_type from the same config
	var propertyType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("property_type"), &propertyType)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Skip if property_type is null or unknown
	if propertyType.IsNull() || propertyType.IsUnknown() {
		return
	}

	propertyTypeValue := propertyType.ValueString()
	propertyValueElements := req.ConfigValue.Elements()

	// Validate based on property type
	switch propertyTypeValue {
	case SINGLE_SELECT, TRUE_FALSE, STRING:
		if len(propertyValueElements) > 1 {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Invalid Property Value Count",
				fmt.Sprintf("Property type %s accepts only a single value, but %d values were provided", propertyTypeValue, len(propertyValueElements)),
			)
		}
	case MULTI_SELECT:
		// Multi-select can have any number of values (minimum 1 is already validated by SizeAtLeast)
	}
}
