package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// planToGithubRepository converts a RepositoryResourceModel to a github.Repository
func (r *RepositoryResource) planToGithubRepository(ctx context.Context, plan *RepositoryResourceModel, isCreate bool) (*github.Repository, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Calculate visibility
	visibility := "public"
	if !plan.Visibility.IsNull() && !plan.Visibility.IsUnknown() {
		visibility = plan.Visibility.ValueString()
	} else if !plan.Private.IsNull() && !plan.Private.IsUnknown() {
		if plan.Private.ValueBool() {
			visibility = "private"
		}
	}

	repo := &github.Repository{
		Name:                     github.Ptr(plan.Name.ValueString()),
		Description:              stringValueOrNil(plan.Description),
		Homepage:                 stringValueOrNil(plan.HomepageURL),
		Visibility:               github.Ptr(visibility),
		Private:                  boolValueOrNil(plan.Private),
		HasDownloads:             boolValueOrNil(plan.HasDownloads),
		HasIssues:                boolValueOrNil(plan.HasIssues),
		HasDiscussions:           boolValueOrNil(plan.HasDiscussions),
		HasProjects:              boolValueOrNil(plan.HasProjects),
		HasWiki:                  boolValueOrNil(plan.HasWiki),
		IsTemplate:               boolValueOrNil(plan.IsTemplate),
		AllowMergeCommit:         boolValueOrNil(plan.AllowMergeCommit),
		AllowSquashMerge:         boolValueOrNil(plan.AllowSquashMerge),
		AllowRebaseMerge:         boolValueOrNil(plan.AllowRebaseMerge),
		AllowAutoMerge:           boolValueOrNil(plan.AllowAutoMerge),
		DeleteBranchOnMerge:      boolValueOrNil(plan.DeleteBranchOnMerge),
		WebCommitSignoffRequired: boolValueOrNil(plan.WebCommitSignoffRequired),
		Archived:                 boolValueOrNil(plan.Archived),
		AllowUpdateBranch:        boolValueOrNil(plan.AllowUpdateBranch),
	}

	// Only set these on create
	if isCreate {
		repo.AutoInit = boolValueOrNil(plan.AutoInit)
		repo.LicenseTemplate = stringValueOrNil(plan.LicenseTemplate)
		repo.GitignoreTemplate = stringValueOrNil(plan.GitignoreTemplate)
	}

	// Only configure merge commit settings if merge commits are allowed
	if !plan.AllowMergeCommit.IsNull() && plan.AllowMergeCommit.ValueBool() {
		repo.MergeCommitTitle = stringValueOrNil(plan.MergeCommitTitle)
		repo.MergeCommitMessage = stringValueOrNil(plan.MergeCommitMessage)
	}

	// Only configure squash commit settings if squash merges are allowed
	if !plan.AllowSquashMerge.IsNull() && plan.AllowSquashMerge.ValueBool() {
		repo.SquashMergeCommitTitle = stringValueOrNil(plan.SquashMergeCommitTitle)
		repo.SquashMergeCommitMessage = stringValueOrNil(plan.SquashMergeCommitMessage)
	}

	// Topics
	if !plan.Topics.IsNull() && !plan.Topics.IsUnknown() {
		var topics []string
		diags.Append(plan.Topics.ElementsAs(ctx, &topics, false)...)
		repo.Topics = topics
	}

	// Security and analysis
	if !plan.SecurityAndAnalysis.IsNull() && !plan.SecurityAndAnalysis.IsUnknown() {
		var secModel SecurityAndAnalysisModel
		diags.Append(plan.SecurityAndAnalysis.As(ctx, &secModel, basetypes.ObjectAsOptions{})...)

		securityAndAnalysis := &github.SecurityAndAnalysis{}

		if !secModel.AdvancedSecurity.IsNull() {
			var advSecModel AdvancedSecurityModel
			diags.Append(secModel.AdvancedSecurity.As(ctx, &advSecModel, basetypes.ObjectAsOptions{})...)
			securityAndAnalysis.AdvancedSecurity = &github.AdvancedSecurity{
				Status: github.Ptr(advSecModel.Status.ValueString()),
			}
		}

		if !secModel.SecretScanning.IsNull() {
			var secretScanModel SecretScanningModel
			diags.Append(secModel.SecretScanning.As(ctx, &secretScanModel, basetypes.ObjectAsOptions{})...)
			securityAndAnalysis.SecretScanning = &github.SecretScanning{
				Status: github.Ptr(secretScanModel.Status.ValueString()),
			}
		}

		if !secModel.SecretScanningPushProtection.IsNull() {
			var pushProtModel SecretScanningPushProtectionModel
			diags.Append(secModel.SecretScanningPushProtection.As(ctx, &pushProtModel, basetypes.ObjectAsOptions{})...)
			securityAndAnalysis.SecretScanningPushProtection = &github.SecretScanningPushProtection{
				Status: github.Ptr(pushProtModel.Status.ValueString()),
			}
		}

		repo.SecurityAndAnalysis = securityAndAnalysis
	}

	return repo, diags
}

// expandPages converts a PagesModel to github.Pages
func (r *RepositoryResource) expandPages(ctx context.Context, pagesModel *PagesModel) (*github.Pages, diag.Diagnostics) {
	var diags diag.Diagnostics

	pages := &github.Pages{}

	if !pagesModel.BuildType.IsNull() && !pagesModel.BuildType.IsUnknown() {
		pages.BuildType = github.Ptr(pagesModel.BuildType.ValueString())
	}

	if !pagesModel.CNAME.IsNull() && !pagesModel.CNAME.IsUnknown() {
		pages.CNAME = github.Ptr(pagesModel.CNAME.ValueString())
	}

	if !pagesModel.Source.IsNull() && !pagesModel.Source.IsUnknown() {
		var sourceModel PagesSourceModel
		diags.Append(pagesModel.Source.As(ctx, &sourceModel, basetypes.ObjectAsOptions{})...)

		source := &github.PagesSource{
			Branch: github.Ptr(sourceModel.Branch.ValueString()),
		}

		if !sourceModel.Path.IsNull() && !sourceModel.Path.IsUnknown() {
			path := sourceModel.Path.ValueString()
			if path != "" && path != "/" {
				source.Path = github.Ptr(path)
			}
		}

		pages.Source = source
	}

	return pages, diags
}

// expandPagesUpdate converts a PagesModel to github.PagesUpdate
func (r *RepositoryResource) expandPagesUpdate(ctx context.Context, pagesModel *PagesModel) (*github.PagesUpdate, diag.Diagnostics) {
	var diags diag.Diagnostics

	pagesUpdate := &github.PagesUpdate{}

	if !pagesModel.BuildType.IsNull() && !pagesModel.BuildType.IsUnknown() {
		pagesUpdate.BuildType = github.Ptr(pagesModel.BuildType.ValueString())
	}

	if !pagesModel.CNAME.IsNull() && !pagesModel.CNAME.IsUnknown() {
		cname := pagesModel.CNAME.ValueString()
		pagesUpdate.CNAME = &cname
	}

	if !pagesModel.Source.IsNull() && !pagesModel.Source.IsUnknown() {
		var sourceModel PagesSourceModel
		diags.Append(pagesModel.Source.As(ctx, &sourceModel, basetypes.ObjectAsOptions{})...)

		source := &github.PagesSource{
			Branch: github.Ptr(sourceModel.Branch.ValueString()),
		}

		if !sourceModel.Path.IsNull() && !sourceModel.Path.IsUnknown() {
			path := sourceModel.Path.ValueString()
			if path != "" && path != "/" {
				source.Path = github.Ptr(path)
			}
		}

		pagesUpdate.Source = source
	}

	return pagesUpdate, diags
}

// flattenPages converts github.Pages to types.Object
func (r *RepositoryResource) flattenPages(ctx context.Context, pages *github.Pages) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	pagesAttrTypes := map[string]attr.Type{
		"build_type": types.StringType,
		"cname":      types.StringType,
		"custom_404": types.BoolType,
		"html_url":   types.StringType,
		"status":     types.StringType,
		"url":        types.StringType,
		"source": types.ObjectType{AttrTypes: map[string]attr.Type{
			"branch": types.StringType,
			"path":   types.StringType,
		}},
	}

	if pages == nil {
		return types.ObjectNull(pagesAttrTypes), diags
	}

	sourceObj := types.ObjectNull(map[string]attr.Type{
		"branch": types.StringType,
		"path":   types.StringType,
	})

	if pages.Source != nil {
		var sourceDiags diag.Diagnostics
		sourceObj, sourceDiags = types.ObjectValue(
			map[string]attr.Type{
				"branch": types.StringType,
				"path":   types.StringType,
			},
			map[string]attr.Value{
				"branch": types.StringPointerValue(pages.Source.Branch),
				"path":   types.StringPointerValue(pages.Source.Path),
			},
		)
		diags.Append(sourceDiags...)
	}

	pagesObj, diagsPages := types.ObjectValue(
		pagesAttrTypes,
		map[string]attr.Value{
			"build_type": types.StringPointerValue(pages.BuildType),
			"cname":      types.StringPointerValue(pages.CNAME),
			"custom_404": types.BoolPointerValue(pages.Custom404),
			"html_url":   types.StringPointerValue(pages.HTMLURL),
			"status":     types.StringPointerValue(pages.Status),
			"url":        types.StringPointerValue(pages.URL),
			"source":     sourceObj,
		},
	)
	diags.Append(diagsPages...)

	return pagesObj, diags
}

// flattenSecurityAndAnalysis converts github.SecurityAndAnalysis to types.Object
func (r *RepositoryResource) flattenSecurityAndAnalysis(ctx context.Context, sec *github.SecurityAndAnalysis) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics

	secAttrTypes := map[string]attr.Type{
		"advanced_security":               types.ObjectType{AttrTypes: map[string]attr.Type{"status": types.StringType}},
		"secret_scanning":                 types.ObjectType{AttrTypes: map[string]attr.Type{"status": types.StringType}},
		"secret_scanning_push_protection": types.ObjectType{AttrTypes: map[string]attr.Type{"status": types.StringType}},
	}

	if sec == nil {
		return types.ObjectNull(secAttrTypes), diags
	}

	statusAttrType := map[string]attr.Type{"status": types.StringType}

	advSecObj := types.ObjectNull(statusAttrType)
	if sec.AdvancedSecurity != nil && sec.AdvancedSecurity.Status != nil {
		var advDiags diag.Diagnostics
		advSecObj, advDiags = types.ObjectValue(
			statusAttrType,
			map[string]attr.Value{"status": types.StringPointerValue(sec.AdvancedSecurity.Status)},
		)
		diags.Append(advDiags...)
	}

	secretScanObj := types.ObjectNull(statusAttrType)
	if sec.SecretScanning != nil && sec.SecretScanning.Status != nil {
		var scanDiags diag.Diagnostics
		secretScanObj, scanDiags = types.ObjectValue(
			statusAttrType,
			map[string]attr.Value{"status": types.StringPointerValue(sec.SecretScanning.Status)},
		)
		diags.Append(scanDiags...)
	}

	pushProtObj := types.ObjectNull(statusAttrType)
	if sec.SecretScanningPushProtection != nil && sec.SecretScanningPushProtection.Status != nil {
		var pushDiags diag.Diagnostics
		pushProtObj, pushDiags = types.ObjectValue(
			statusAttrType,
			map[string]attr.Value{"status": types.StringPointerValue(sec.SecretScanningPushProtection.Status)},
		)
		diags.Append(pushDiags...)
	}

	secObj, diagsSec := types.ObjectValue(
		secAttrTypes,
		map[string]attr.Value{
			"advanced_security":               advSecObj,
			"secret_scanning":                 secretScanObj,
			"secret_scanning_push_protection": pushProtObj,
		},
	)
	diags.Append(diagsSec...)

	return secObj, diags
}

// Utility functions

func stringValueOrNil(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return github.Ptr(v.ValueString())
}

func boolValueOrNil(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return github.Ptr(v.ValueBool())
}

// getAllCustomPropertiesAsMap fetches all custom properties for a repository and returns them as a map
func (r *RepositoryResource) getAllCustomPropertiesAsMap(ctx context.Context, repoName string) (map[string]attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics

	allCustomProperties, _, err := r.client.Repositories.GetAllCustomPropertyValues(ctx, r.owner, repoName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok && ghErr.Response.StatusCode == http.StatusNotFound {
			return map[string]attr.Value{}, diags
		}
		diags.AddError(
			"Error fetching custom properties",
			fmt.Sprintf("Could not read custom properties for repository %s: %v", repoName, err),
		)
		return nil, diags
	}

	result := make(map[string]attr.Value)
	for _, prop := range allCustomProperties {
		values, err := convertCustomPropertyValueToList(prop)
		if err != nil {
			diags.AddError(
				"Error converting custom property",
				fmt.Sprintf("Could not convert property %s: %v", prop.PropertyName, err),
			)
			return nil, diags
		}

		// Convert the list to a string representation
		if len(values) == 1 {
			result[prop.PropertyName] = types.StringValue(values[0])
		} else {
			// For multiple values, use JSON encoding
			jsonBytes, err := json.Marshal(values)
			if err != nil {
				diags.AddError(
					"Error marshaling custom property",
					fmt.Sprintf("Could not marshal property %s: %v", prop.PropertyName, err),
				)
				return nil, diags
			}
			result[prop.PropertyName] = types.StringValue(string(jsonBytes))
		}
	}

	return result, diags
}

// filterCustomPropertiesFromSet filters a Set of custom properties to only include managed ones
func (r *RepositoryResource) filterCustomPropertiesFromSet(ctx context.Context, allCustomPropsMap map[string]attr.Value, managedProperties types.Set) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Build a set of managed property names
	managedNames := make(map[string]bool)
	if !managedProperties.IsNull() && !managedProperties.IsUnknown() {
		var customProps []CustomPropertyModel
		diags.Append(managedProperties.ElementsAs(ctx, &customProps, false)...)
		if diags.HasError() {
			return types.SetNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":  types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
			}), diags
		}

		for _, prop := range customProps {
			if !prop.Name.IsNull() {
				managedNames[prop.Name.ValueString()] = true
			}
		}
	}

	// Filter properties to only include managed ones
	results := []attr.Value{}
	for propName, value := range allCustomPropsMap {
		// Only include properties that we're explicitly managing
		if managedNames[propName] {
			valueStr := ""
			if strVal, ok := value.(types.String); ok {
				valueStr = strVal.ValueString()
			}

			// Parse the value back to a list
			var values []string
			if strings.HasPrefix(valueStr, "[") && strings.HasSuffix(valueStr, "]") {
				// Multi-value property stored as JSON array
				if err := json.Unmarshal([]byte(valueStr), &values); err != nil {
					log.Printf("[WARN] Failed to unmarshal custom property value for %s: %v", propName, err)
					continue
				}
			} else {
				// Single value property
				values = []string{valueStr}
			}

			// Convert to types.List
			listValues := make([]attr.Value, len(values))
			for i, v := range values {
				listValues[i] = types.StringValue(v)
			}

			valueList, listDiags := types.ListValue(types.StringType, listValues)
			diags.Append(listDiags...)

			propObj, objDiags := types.ObjectValue(
				map[string]attr.Type{
					"name":  types.StringType,
					"value": types.ListType{ElemType: types.StringType},
				},
				map[string]attr.Value{
					"name":  types.StringValue(propName),
					"value": valueList,
				},
			)
			diags.Append(objDiags...)

			results = append(results, propObj)
		}
	}

	resultSet, setDiags := types.SetValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":  types.StringType,
				"value": types.ListType{ElemType: types.StringType},
			},
		},
		results,
	)
	diags.Append(setDiags...)

	return resultSet, diags
}

// setRepositoryCustomProperties sets custom properties for a repository
func (r *RepositoryResource) setRepositoryCustomProperties(ctx context.Context, repoName string, customPropsSet types.Set, exclusiveMode bool, currentPropsMap map[string]attr.Value) diag.Diagnostics {
	var diags diag.Diagnostics

	if customPropsSet.IsNull() || len(customPropsSet.Elements()) == 0 {
		if exclusiveMode {
			// In exclusive mode, if no properties are defined, send all current properties with null values to remove them
			if len(currentPropsMap) > 0 {
				propertyValues := make([]*github.CustomPropertyValue, 0, len(currentPropsMap))
				for propName := range currentPropsMap {
					propertyValues = append(propertyValues, &github.CustomPropertyValue{
						PropertyName: propName,
						Value:        nil, // null value to remove
					})
				}
				_, err := r.client.Repositories.CreateOrUpdateCustomProperties(ctx, r.owner, repoName, propertyValues)
				if err != nil {
					diags.AddError(
						"Error removing custom properties",
						fmt.Sprintf("Could not remove custom properties for repository %s: %v", repoName, err),
					)
				}
			}
		}
		// In non-exclusive mode, do nothing if no properties are defined
		return diags
	}

	var customProps []CustomPropertyModel
	diags.Append(customPropsSet.ElementsAs(ctx, &customProps, false)...)
	if diags.HasError() {
		return diags
	}

	propertyValues := make([]*github.CustomPropertyValue, 0)
	newPropNames := make(map[string]bool)

	for _, prop := range customProps {
		propName := prop.Name.ValueString()
		newPropNames[propName] = true

		var valuesList []string
		diags.Append(prop.Value.ElementsAs(ctx, &valuesList, false)...)
		if diags.HasError() {
			return diags
		}

		customProp := &github.CustomPropertyValue{
			PropertyName: propName,
		}

		// Convert values list to appropriate format
		if len(valuesList) == 1 {
			// Single value - set as string
			customProp.Value = valuesList[0]
		} else {
			// Multiple values - set as array
			customProp.Value = valuesList
		}

		propertyValues = append(propertyValues, customProp)
	}

	if exclusiveMode {
		// In exclusive mode, we need to explicitly remove properties that are no longer in the config
		// by sending them with null values
		for propName := range currentPropsMap {
			if !newPropNames[propName] {
				propertyValues = append(propertyValues, &github.CustomPropertyValue{
					PropertyName: propName,
					Value:        nil, // null value to remove
				})
			}
		}
	}

	if len(propertyValues) > 0 {
		_, err := r.client.Repositories.CreateOrUpdateCustomProperties(ctx, r.owner, repoName, propertyValues)
		if err != nil {
			diags.AddError(
				"Error setting custom properties",
				fmt.Sprintf("Could not set custom properties for repository %s: %v", repoName, err),
			)
			return diags
		}
	}

	return diags
}


// convertCustomPropertyValueToList converts a GitHub CustomPropertyValue to a string slice.
// It handles the various types that the API may return (string, []any, []string).
func convertCustomPropertyValueToList(prop *github.CustomPropertyValue) ([]string, error) {
	if prop.Value == nil {
		return []string{}, nil
	}

	switch v := prop.Value.(type) {
	case string:
		return []string{v}, nil
	case []any:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result, nil
	case []string:
		return v, nil
	default:
		return []string{fmt.Sprintf("%v", v)}, nil
	}
}
