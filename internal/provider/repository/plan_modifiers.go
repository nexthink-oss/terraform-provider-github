package repository

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// fullNamePlanModifier marks full_name as unknown when the repository name changes
type fullNamePlanModifier struct{}

// Description returns a human-readable description of the plan modifier
func (m fullNamePlanModifier) Description(ctx context.Context) string {
	return "Marks full_name as unknown when repository name changes."
}

// MarkdownDescription returns a markdown description of the plan modifier
func (m fullNamePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

// PlanModifyString implements the plan modification logic
func (m fullNamePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If we're creating the resource, don't interfere
	if req.State.Raw.IsNull() {
		return
	}

	// If we're destroying the resource, don't interfere
	if req.Plan.Raw.IsNull() {
		return
	}

	// Get the name attribute from both state and plan
	var stateName, planName string

	// Get name from state
	stateNamePath := req.Path.ParentPath().AtName("name")
	diagState := req.State.GetAttribute(ctx, stateNamePath, &stateName)
	resp.Diagnostics.Append(diagState...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get name from plan
	diagPlan := req.Plan.GetAttribute(ctx, stateNamePath, &planName)
	resp.Diagnostics.Append(diagPlan...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If name is changing, mark full_name as unknown
	if stateName != planName {
		resp.PlanValue = types.StringUnknown()
	}
}

// FullNamePlanModifier returns a plan modifier that marks full_name as unknown when name changes
func FullNamePlanModifier() planmodifier.String {
	return fullNamePlanModifier{}
}

// nameBasedUnknownModifier marks a computed field as unknown when the repository name changes
// This is needed for fields that depend on the repository name (URLs, ID, etc.)
type nameBasedUnknownModifier struct{}

func (m nameBasedUnknownModifier) Description(ctx context.Context) string {
	return "Marks the field as unknown when repository name changes."
}

func (m nameBasedUnknownModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m nameBasedUnknownModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If we're creating the resource, use UseStateForUnknown behavior
	if req.State.Raw.IsNull() {
		return
	}

	// If we're destroying the resource, don't interfere
	if req.Plan.Raw.IsNull() {
		return
	}

	// Get the name attribute from both state and plan
	var stateName, planName string

	// Get name from state
	stateNamePath := req.Path.ParentPath().AtName("name")
	diagState := req.State.GetAttribute(ctx, stateNamePath, &stateName)
	resp.Diagnostics.Append(diagState...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get name from plan
	diagPlan := req.Plan.GetAttribute(ctx, stateNamePath, &planName)
	resp.Diagnostics.Append(diagPlan...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If name is changing, mark this field as unknown
	if stateName != planName {
		resp.PlanValue = types.StringUnknown()
		return
	}
}

func NameBasedUnknownModifier() planmodifier.String {
	return nameBasedUnknownModifier{}
}

// suppressEtagDiffModifier suppresses etag differences in plan diffs
// The etag is a computed field used for conditional requests, but changes to
// the etag value don't represent actual resource changes and should not trigger updates
type suppressEtagDiffModifier struct{}

func (m suppressEtagDiffModifier) Description(ctx context.Context) string {
	return "Suppresses spurious etag differences that don't represent actual resource changes."
}

func (m suppressEtagDiffModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m suppressEtagDiffModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If we're creating the resource, allow etag to be set
	if req.State.Raw.IsNull() {
		return
	}

	// If we're destroying the resource, don't interfere
	if req.Plan.Raw.IsNull() {
		return
	}

	// If we have a state value, always use it to prevent etag diffs
	// The etag is only relevant for conditional requests, not for tracking actual changes
	if !req.StateValue.IsNull() && !req.StateValue.IsUnknown() {
		resp.PlanValue = req.StateValue
	}
}

// SuppressEtagDiffPlanModifier returns a plan modifier that suppresses etag differences
func SuppressEtagDiffPlanModifier() planmodifier.String {
	return suppressEtagDiffModifier{}
}
