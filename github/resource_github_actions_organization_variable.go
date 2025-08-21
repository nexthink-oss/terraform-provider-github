package github

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubActionsOrganizationVariableResource{}
	_ resource.ResourceWithConfigure   = &githubActionsOrganizationVariableResource{}
	_ resource.ResourceWithImportState = &githubActionsOrganizationVariableResource{}
)

type githubActionsOrganizationVariableResource struct {
	client *Owner
}

type githubActionsOrganizationVariableResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	VariableName          types.String `tfsdk:"variable_name"`
	Value                 types.String `tfsdk:"value"`
	Visibility            types.String `tfsdk:"visibility"`
	SelectedRepositoryIDs types.Set    `tfsdk:"selected_repository_ids"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
}

func NewGithubActionsOrganizationVariableResource() resource.Resource {
	return &githubActionsOrganizationVariableResource{}
}

func (r *githubActionsOrganizationVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_organization_variable"
}

func (r *githubActionsOrganizationVariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages an Action variable within a GitHub organization",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the actions organization variable (same as variable_name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"variable_name": schema.StringAttribute{
				Description: "Name of the variable.",
				Required:    true,
				Validators: []validator.String{
					&organizationVariableNameValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Description: "Value of the variable.",
				Required:    true,
			},
			"visibility": schema.StringAttribute{
				Description: "Configures the access that repositories have to the organization variable. Must be one of 'all', 'private', or 'selected'. 'selected_repository_ids' is required if set to 'selected'.",
				Required:    true,
				Validators: []validator.String{
					&organizationVariableVisibilityValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"selected_repository_ids": schema.SetAttribute{
				Description: "An array of repository ids that can access the organization variable.",
				Optional:    true,
				Computed:    true,
				ElementType: types.Int64Type,
				Default:     setdefault.StaticValue(types.SetValueMust(types.Int64Type, []attr.Value{})),
				Validators: []validator.Set{
					&organizationVariableSelectedRepositoriesValidator{},
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date of 'actions_variable' creation.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Date of 'actions_variable' update.",
				Computed:    true,
			},
		},
	}
}

func (r *githubActionsOrganizationVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *githubActionsOrganizationVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubActionsOrganizationVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	variableName := data.VariableName.ValueString()
	value := data.Value.ValueString()
	visibility := data.Visibility.ValueString()

	// Validate visibility and repository selection relationship
	hasSelectedRepositories := !data.SelectedRepositoryIDs.IsNull() && !data.SelectedRepositoryIDs.IsUnknown()
	selectedRepositoryCount := len(data.SelectedRepositoryIDs.Elements())

	if visibility != "selected" && hasSelectedRepositories && selectedRepositoryCount > 0 {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Cannot use selected_repository_ids without visibility being set to 'selected'",
		)
		return
	}

	// Convert selected repository IDs
	var selectedRepositoryIDs []int64
	if hasSelectedRepositories {
		for _, elem := range data.SelectedRepositoryIDs.Elements() {
			if intVal, ok := elem.(types.Int64); ok && !intVal.IsNull() && !intVal.IsUnknown() {
				selectedRepositoryIDs = append(selectedRepositoryIDs, intVal.ValueInt64())
			}
		}
	}

	repoIDs := github.SelectedRepoIDs(selectedRepositoryIDs)

	// Create the variable
	variable := &github.ActionsVariable{
		Name:                  variableName,
		Value:                 value,
		Visibility:            &visibility,
		SelectedRepositoryIDs: &repoIDs,
	}

	_, err := client.Actions.CreateOrgVariable(ctx, owner, variable)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Actions Organization Variable",
			fmt.Sprintf("An unexpected error occurred when creating the actions organization variable: %s", err.Error()),
		)
		return
	}

	// Set the ID
	data.ID = types.StringValue(variableName)

	tflog.Debug(ctx, "created GitHub actions organization variable", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"owner":         owner,
		"variable_name": variableName,
		"visibility":    visibility,
	})

	// Read the created resource to populate computed fields
	r.readGithubActionsOrganizationVariable(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsOrganizationVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubActionsOrganizationVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubActionsOrganizationVariable(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsOrganizationVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubActionsOrganizationVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	variableName := data.VariableName.ValueString()
	value := data.Value.ValueString()
	visibility := data.Visibility.ValueString()

	// Validate visibility and repository selection relationship
	hasSelectedRepositories := !data.SelectedRepositoryIDs.IsNull() && !data.SelectedRepositoryIDs.IsUnknown()
	selectedRepositoryCount := len(data.SelectedRepositoryIDs.Elements())

	if visibility != "selected" && hasSelectedRepositories && selectedRepositoryCount > 0 {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"Cannot use selected_repository_ids without visibility being set to 'selected'",
		)
		return
	}

	// Convert selected repository IDs
	var selectedRepositoryIDs []int64
	if hasSelectedRepositories {
		for _, elem := range data.SelectedRepositoryIDs.Elements() {
			if intVal, ok := elem.(types.Int64); ok && !intVal.IsNull() && !intVal.IsUnknown() {
				selectedRepositoryIDs = append(selectedRepositoryIDs, intVal.ValueInt64())
			}
		}
	}

	repoIDs := github.SelectedRepoIDs(selectedRepositoryIDs)

	// Update the variable
	variable := &github.ActionsVariable{
		Name:                  variableName,
		Value:                 value,
		Visibility:            &visibility,
		SelectedRepositoryIDs: &repoIDs,
	}

	_, err := client.Actions.UpdateOrgVariable(ctx, owner, variable)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Actions Organization Variable",
			fmt.Sprintf("An unexpected error occurred when updating the actions organization variable: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "updated GitHub actions organization variable", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"owner":         owner,
		"variable_name": variableName,
		"visibility":    visibility,
	})

	// Read the updated resource to populate computed fields
	r.readGithubActionsOrganizationVariable(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsOrganizationVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubActionsOrganizationVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	variableName := data.ID.ValueString()

	_, err := client.Actions.DeleteOrgVariable(ctx, owner, variableName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Actions Organization Variable",
			fmt.Sprintf("An unexpected error occurred when deleting the actions organization variable: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub actions organization variable", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"owner":         owner,
		"variable_name": variableName,
	})
}

func (r *githubActionsOrganizationVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client := r.client.V3Client()
	owner := r.client.Name()
	variableName := req.ID

	// Verify the variable exists
	variable, _, err := client.Actions.GetOrgVariable(ctx, owner, variableName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import Actions Organization Variable",
			fmt.Sprintf("Unable to read actions organization variable for import: %s", err.Error()),
		)
		return
	}

	data := &githubActionsOrganizationVariableResourceModel{
		ID:           types.StringValue(variableName),
		VariableName: types.StringValue(variableName),
		Value:        types.StringValue(variable.Value),
		Visibility:   types.StringValue(*variable.Visibility),
		CreatedAt:    types.StringValue(variable.CreatedAt.String()),
		UpdatedAt:    types.StringValue(variable.UpdatedAt.String()),
	}

	// Get selected repository IDs if visibility is 'selected'
	if *variable.Visibility == "selected" {
		selectedRepositoryIDs := []int64{}
		opt := &github.ListOptions{
			PerPage: 30,
		}
		for {
			results, githubResp, err := client.Actions.ListSelectedReposForOrgVariable(ctx, owner, variableName, opt)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Read Selected Repositories",
					fmt.Sprintf("Unable to read selected repositories for organization variable: %s", err.Error()),
				)
				return
			}

			for _, repo := range results.Repositories {
				selectedRepositoryIDs = append(selectedRepositoryIDs, repo.GetID())
			}

			if githubResp.NextPage == 0 {
				break
			}
			opt.Page = githubResp.NextPage
		}

		selectedRepositoryIDAttrs := []attr.Value{}
		for _, id := range selectedRepositoryIDs {
			selectedRepositoryIDAttrs = append(selectedRepositoryIDAttrs, types.Int64Value(id))
		}
		data.SelectedRepositoryIDs = types.SetValueMust(types.Int64Type, selectedRepositoryIDAttrs)
	} else {
		data.SelectedRepositoryIDs = types.SetValueMust(types.Int64Type, []attr.Value{})
	}

	tflog.Debug(ctx, "imported GitHub actions organization variable", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"owner":         owner,
		"variable_name": variableName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubActionsOrganizationVariableResource) readGithubActionsOrganizationVariable(ctx context.Context, data *githubActionsOrganizationVariableResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()
	variableName := data.ID.ValueString()

	variable, _, err := client.Actions.GetOrgVariable(ctx, owner, variableName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing actions organization variable from state because it no longer exists in GitHub", map[string]interface{}{
					"owner":         owner,
					"variable_name": variableName,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Actions Organization Variable",
			fmt.Sprintf("An unexpected error occurred when reading the actions organization variable: %s", err.Error()),
		)
		return
	}

	data.VariableName = types.StringValue(variableName)
	data.Value = types.StringValue(variable.Value)
	data.Visibility = types.StringValue(*variable.Visibility)
	data.CreatedAt = types.StringValue(variable.CreatedAt.String())
	data.UpdatedAt = types.StringValue(variable.UpdatedAt.String())

	// Get selected repository IDs if visibility is 'selected'
	selectedRepositoryIDs := []int64{}
	if *variable.Visibility == "selected" {
		opt := &github.ListOptions{
			PerPage: 30,
		}
		for {
			results, githubResp, err := client.Actions.ListSelectedReposForOrgVariable(ctx, owner, variableName, opt)
			if err != nil {
				diags.AddError(
					"Unable to Read Selected Repositories",
					fmt.Sprintf("Unable to read selected repositories for organization variable: %s", err.Error()),
				)
				return
			}

			for _, repo := range results.Repositories {
				selectedRepositoryIDs = append(selectedRepositoryIDs, repo.GetID())
			}

			if githubResp.NextPage == 0 {
				break
			}
			opt.Page = githubResp.NextPage
		}
	}

	selectedRepositoryIDAttrs := []attr.Value{}
	for _, id := range selectedRepositoryIDs {
		selectedRepositoryIDAttrs = append(selectedRepositoryIDAttrs, types.Int64Value(id))
	}
	data.SelectedRepositoryIDs = types.SetValueMust(types.Int64Type, selectedRepositoryIDAttrs)

	tflog.Debug(ctx, "successfully read GitHub actions organization variable", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"owner":         owner,
		"variable_name": variableName,
		"visibility":    *variable.Visibility,
	})
}

// Custom Validators for Organization Variables

// organizationVariableNameValidator validates variable names according to GitHub requirements
type organizationVariableNameValidator struct{}

func (v *organizationVariableNameValidator) Description(ctx context.Context) string {
	return "Variable names can only contain alphanumeric characters or underscores and must not start with a number or GITHUB_ prefix"
}

func (v *organizationVariableNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *organizationVariableNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// https://docs.github.com/en/actions/reference/encrypted-secrets#naming-your-secrets
	// The same naming rules apply to variables as secrets
	variableNameRegexp := regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")

	if !variableNameRegexp.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Variable Name",
			"Variable names can only contain alphanumeric characters or underscores and must not start with a number",
		)
	}

	if strings.HasPrefix(strings.ToUpper(value), "GITHUB_") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Variable Name",
			"Variable names must not start with the GITHUB_ prefix",
		)
	}
}

// organizationVariableVisibilityValidator validates that visibility is one of the allowed values
type organizationVariableVisibilityValidator struct{}

func (v *organizationVariableVisibilityValidator) Description(ctx context.Context) string {
	return "Value must be one of: all, private, selected"
}

func (v *organizationVariableVisibilityValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *organizationVariableVisibilityValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	allowedValues := []string{"all", "private", "selected"}

	for _, allowed := range allowedValues {
		if value == allowed {
			return
		}
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid Visibility Value",
		fmt.Sprintf("Value must be one of: %v, got: %s", allowedValues, value),
	)
}

// organizationVariableSelectedRepositoriesValidator validates the relationship between visibility and selected repositories
type organizationVariableSelectedRepositoriesValidator struct{}

func (v *organizationVariableSelectedRepositoriesValidator) Description(ctx context.Context) string {
	return "Selected repository IDs can only be set when visibility is 'selected'"
}

func (v *organizationVariableSelectedRepositoriesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *organizationVariableSelectedRepositoriesValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// Check if selected_repository_ids is set but not empty
	if len(req.ConfigValue.Elements()) == 0 {
		return
	}

	// Get the visibility value
	var visibility types.String
	diags := req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("visibility"), &visibility)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !visibility.IsNull() && !visibility.IsUnknown() && visibility.ValueString() != "selected" {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Configuration",
			"selected_repository_ids can only be set when visibility is 'selected'",
		)
	}
}
