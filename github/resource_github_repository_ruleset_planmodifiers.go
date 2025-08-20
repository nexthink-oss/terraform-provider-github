package github

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// rulesetTargetConflictValidator validates that branch_name_pattern and tag_name_pattern
// are only used with appropriate target values
type rulesetTargetConflictValidator struct{}

func (v rulesetTargetConflictValidator) Description(ctx context.Context) string {
	return "validates pattern rules are used with appropriate targets"
}

func (v rulesetTargetConflictValidator) MarkdownDescription(ctx context.Context) string {
	return "validates pattern rules are used with appropriate targets"
}

func (v rulesetTargetConflictValidator) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// This plan modifier validates that branch_name_pattern and tag_name_pattern
	// are only used with appropriate target values

	// Get the target value from the configuration
	var target types.String
	diags := req.Config.GetAttribute(ctx, path.Root("target"), &target)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if target.IsNull() || target.IsUnknown() {
		return
	}

	targetValue := target.ValueString()

	// Get the current path to determine which pattern rule we're validating
	currentPath := req.Path

	// Check if this is a branch_name_pattern or tag_name_pattern
	if len(currentPath.Steps()) >= 3 {
		// Path structure: rules.0.branch_name_pattern or rules.0.tag_name_pattern
		if pathStep, ok := currentPath.Steps()[2].(path.PathStepAttributeName); ok {
			stepName := pathStep.String()
			if stepName == "branch_name_pattern" {
				if targetValue != "branch" && !req.PlanValue.IsNull() && !req.PlanValue.IsUnknown() {
					resp.Diagnostics.AddAttributeError(
						req.Path,
						"Invalid Configuration",
						"branch_name_pattern can only be used when target is set to 'branch'",
					)
					return
				}
			} else if stepName == "tag_name_pattern" {
				if targetValue != "tag" && !req.PlanValue.IsNull() && !req.PlanValue.IsUnknown() {
					resp.Diagnostics.AddAttributeError(
						req.Path,
						"Invalid Configuration",
						"tag_name_pattern can only be used when target is set to 'tag'",
					)
					return
				}
			}
		}
	}
}

// RequiredWithValidator validates that update_allows_fetch_and_merge requires update to be true
type requiredWithValidator struct {
	requiredPath path.Path
}

func (v requiredWithValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("validates that this attribute is only set when %s is true", v.requiredPath)
}

func (v requiredWithValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("validates that this attribute is only set when %s is true", v.requiredPath)
}

func (v requiredWithValidator) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	// If the current attribute is null or unknown, no validation needed
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	// Get the required attribute value
	var requiredAttr types.Bool
	diags := req.Config.GetAttribute(ctx, v.requiredPath, &requiredAttr)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If the required attribute is not set to true, but this attribute is being set
	if (requiredAttr.IsNull() || !requiredAttr.ValueBool()) && !req.PlanValue.IsNull() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Configuration",
			fmt.Sprintf("This attribute can only be set when %s is set to true", v.requiredPath),
		)
	}
}

// NewRulesetTargetConflictValidator creates a new validator for target conflicts
func NewRulesetTargetConflictValidator() planmodifier.List {
	return rulesetTargetConflictValidator{}
}

// NewRequiredWithValidator creates a new validator that requires another attribute to be true
func NewRequiredWithValidator(requiredPath path.Path) planmodifier.Bool {
	return requiredWithValidator{
		requiredPath: requiredPath,
	}
}
