package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubRepositoryCustomPropertiesDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryCustomPropertiesDataSource{}
)

type githubRepositoryCustomPropertiesDataSource struct {
	client *Owner
}

type githubRepositoryCustomPropertiesDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Repository types.String `tfsdk:"repository"`
	Property   types.Set    `tfsdk:"property"`
}

type githubRepositoryCustomPropertyModel struct {
	PropertyName  types.String `tfsdk:"property_name"`
	PropertyValue types.Set    `tfsdk:"property_value"`
}

func NewGithubRepositoryCustomPropertiesDataSource() datasource.DataSource {
	return &githubRepositoryCustomPropertiesDataSource{}
}

func (d *githubRepositoryCustomPropertiesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_custom_properties"
}

func (d *githubRepositoryCustomPropertiesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get all custom properties of a repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "Name of the repository which the custom properties should be on.",
				Required:    true,
			},
			"property": schema.SetNestedAttribute{
				Description: "List of custom properties",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"property_name": schema.StringAttribute{
							Description: "Name of the custom property.",
							Computed:    true,
						},
						"property_value": schema.SetAttribute{
							Description: "Value of the custom property.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (d *githubRepositoryCustomPropertiesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *githubRepositoryCustomPropertiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryCustomPropertiesDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repositoryName := data.Repository.ValueString()
	owner := d.client.Name()

	tflog.Debug(ctx, "Reading GitHub repository custom properties", map[string]interface{}{
		"owner":      owner,
		"repository": repositoryName,
	})

	// Fetch custom properties from GitHub API
	allCustomProperties, _, err := d.client.V3Client().Repositories.GetAllCustomPropertyValues(ctx, owner, repositoryName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Repository Custom Properties",
			fmt.Sprintf("An unexpected error occurred while reading GitHub repository custom properties: %s", err.Error()),
		)
		return
	}

	// Convert GitHub API response to Terraform data model
	properties, diags := d.flattenRepositoryCustomProperties(ctx, allCustomProperties)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the data source fields
	data.ID = types.StringValue(d.buildTwoPartID(owner, repositoryName))
	data.Repository = types.StringValue(repositoryName)
	data.Property = properties

	tflog.Debug(ctx, "Successfully read GitHub repository custom properties", map[string]interface{}{
		"owner":            owner,
		"repository":       repositoryName,
		"properties_count": len(allCustomProperties),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// flattenRepositoryCustomProperties converts GitHub API custom properties to Terraform model
func (d *githubRepositoryCustomPropertiesDataSource) flattenRepositoryCustomProperties(ctx context.Context, customProperties []*github.CustomPropertyValue) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(customProperties) == 0 {
		return types.SetNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"property_name":  types.StringType,
				"property_value": types.SetType{ElemType: types.StringType},
			},
		}), diags
	}

	var properties []githubRepositoryCustomPropertyModel
	for _, prop := range customProperties {
		propertyValues, err := d.parseRepositoryCustomPropertyValueToStringSlice(prop)
		if err != nil {
			diags.AddError(
				"Unable to Parse Custom Property Value",
				fmt.Sprintf("An unexpected error occurred while parsing custom property value: %s", err.Error()),
			)
			return types.SetNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"property_name":  types.StringType,
					"property_value": types.SetType{ElemType: types.StringType},
				},
			}), diags
		}

		propertyValueSet, setDiags := types.SetValueFrom(ctx, types.StringType, propertyValues)
		diags.Append(setDiags...)
		if diags.HasError() {
			return types.SetNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"property_name":  types.StringType,
					"property_value": types.SetType{ElemType: types.StringType},
				},
			}), diags
		}

		property := githubRepositoryCustomPropertyModel{
			PropertyName:  types.StringValue(prop.PropertyName),
			PropertyValue: propertyValueSet,
		}
		properties = append(properties, property)
	}

	propertiesSet, setDiags := types.SetValueFrom(ctx, types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"property_name":  types.StringType,
			"property_value": types.SetType{ElemType: types.StringType},
		},
	}, properties)
	diags.Append(setDiags...)

	return propertiesSet, diags
}

// parseRepositoryCustomPropertyValueToStringSlice handles different property value types
func (d *githubRepositoryCustomPropertiesDataSource) parseRepositoryCustomPropertyValueToStringSlice(prop *github.CustomPropertyValue) ([]string, error) {
	switch value := prop.Value.(type) {
	case string:
		return []string{value}, nil
	case []string:
		return value, nil
	default:
		return nil, fmt.Errorf("custom property value couldn't be parsed as a string or a list of strings: %v", value)
	}
}

// buildTwoPartID creates an ID from two parts (same as SDKv2 util function)
func (d *githubRepositoryCustomPropertiesDataSource) buildTwoPartID(a, b string) string {
	return fmt.Sprintf("%s:%s", a, b)
}
