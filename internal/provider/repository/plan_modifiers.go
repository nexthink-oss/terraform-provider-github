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
