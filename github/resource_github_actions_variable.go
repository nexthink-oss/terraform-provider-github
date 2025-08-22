package github

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubActionsVariableResource{}
	_ resource.ResourceWithConfigure   = &githubActionsVariableResource{}
	_ resource.ResourceWithImportState = &githubActionsVariableResource{}
)

type githubActionsVariableResource struct {
	client *Owner
}

type githubActionsVariableResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Repository   types.String `tfsdk:"repository"`
	VariableName types.String `tfsdk:"variable_name"`
	Value        types.String `tfsdk:"value"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func NewGithubActionsVariableResource() resource.Resource {
	return &githubActionsVariableResource{}
}

func (r *githubActionsVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_variable"
}

func (r *githubActionsVariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages an Action variable within a GitHub repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the actions variable (repository:variable_name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "Name of the repository.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"variable_name": schema.StringAttribute{
				Description: "Name of the variable.",
				Required:    true,
				Validators: []validator.String{
					&variableNameValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Description: "Value of the variable.",
				Required:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Date of 'actions_variable' creation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Date of 'actions_variable' update.",
				Computed:    true,
			},
		},
	}
}

func (r *githubActionsVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubActionsVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubActionsVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repo := data.Repository.ValueString()
	variableName := data.VariableName.ValueString()
	value := data.Value.ValueString()

	// Create the variable
	variable := &github.ActionsVariable{
		Name:  variableName,
		Value: value,
	}

	_, err := client.Actions.CreateRepoVariable(ctx, owner, repo, variable)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Actions Variable",
			fmt.Sprintf("An unexpected error occurred when creating the actions variable: %s", err.Error()),
		)
		return
	}

	// Set the ID and read the created resource
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", repo, variableName))

	tflog.Debug(ctx, "created GitHub actions variable", map[string]any{
		"id":            data.ID.ValueString(),
		"repository":    repo,
		"variable_name": variableName,
	})

	// Read the created resource to populate computed fields
	r.readGithubActionsVariable(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubActionsVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubActionsVariable(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubActionsVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repo := data.Repository.ValueString()
	variableName := data.VariableName.ValueString()
	value := data.Value.ValueString()

	// Update the variable
	variable := &github.ActionsVariable{
		Name:  variableName,
		Value: value,
	}

	_, err := client.Actions.UpdateRepoVariable(ctx, owner, repo, variable)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Actions Variable",
			fmt.Sprintf("An unexpected error occurred when updating the actions variable: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "updated GitHub actions variable", map[string]any{
		"id":            data.ID.ValueString(),
		"repository":    repo,
		"variable_name": variableName,
	})

	// Read the updated resource to populate computed fields
	r.readGithubActionsVariable(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubActionsVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName, variableName, err := r.parseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse ID",
			fmt.Sprintf("Unable to parse ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, err = client.Actions.DeleteRepoVariable(ctx, owner, repoName, variableName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Actions Variable",
			fmt.Sprintf("An unexpected error occurred when deleting the actions variable: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub actions variable", map[string]any{
		"id":            data.ID.ValueString(),
		"repository":    repoName,
		"variable_name": variableName,
	})
}

func (r *githubActionsVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "<repository>/<variable_name>" or "<repository>:<variable_name>" (for compatibility)
	var parts []string
	if strings.Contains(req.ID, "/") {
		parts = strings.Split(req.ID, "/")
	} else if strings.Contains(req.ID, ":") {
		parts = strings.Split(req.ID, ":")
	} else {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Invalid ID specified. Supplied ID must be written as <repository>/<variable_name> or <repository>:<variable_name>. Got: %q", req.ID),
		)
		return
	}

	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Invalid ID specified. Supplied ID must be written as <repository>/<variable_name> or <repository>:<variable_name>. Got: %q", req.ID),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	repoName := parts[0]
	variableName := parts[1]

	// Verify the variable exists
	variable, _, err := client.Actions.GetRepoVariable(ctx, owner, repoName, variableName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import Actions Variable",
			fmt.Sprintf("Unable to read actions variable for import: %s", err.Error()),
		)
		return
	}

	data := &githubActionsVariableResourceModel{
		ID:           types.StringValue(fmt.Sprintf("%s:%s", repoName, variableName)),
		Repository:   types.StringValue(repoName),
		VariableName: types.StringValue(variableName),
		Value:        types.StringValue(variable.Value),
		CreatedAt:    types.StringValue(variable.CreatedAt.String()),
		UpdatedAt:    types.StringValue(variable.UpdatedAt.String()),
	}

	tflog.Debug(ctx, "imported GitHub actions variable", map[string]any{
		"id":            data.ID.ValueString(),
		"repository":    repoName,
		"variable_name": variableName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubActionsVariableResource) parseTwoPartID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected ID format (%q); expected repository:variable_name", id)
	}

	return parts[0], parts[1], nil
}

func (r *githubActionsVariableResource) readGithubActionsVariable(ctx context.Context, data *githubActionsVariableResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	repoName, variableName, err := r.parseTwoPartID(data.ID.ValueString())
	if err != nil {
		diags.AddError(
			"Unable to Parse ID",
			fmt.Sprintf("Unable to parse ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	variable, _, err := client.Actions.GetRepoVariable(ctx, owner, repoName, variableName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing actions variable from state because it no longer exists in GitHub", map[string]any{
					"owner":         owner,
					"repository":    repoName,
					"variable_name": variableName,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Actions Variable",
			fmt.Sprintf("An unexpected error occurred when reading the actions variable: %s", err.Error()),
		)
		return
	}

	data.Repository = types.StringValue(repoName)
	data.VariableName = types.StringValue(variableName)
	data.Value = types.StringValue(variable.Value)
	data.CreatedAt = types.StringValue(variable.CreatedAt.String())
	data.UpdatedAt = types.StringValue(variable.UpdatedAt.String())

	tflog.Debug(ctx, "successfully read GitHub actions variable", map[string]any{
		"id":            data.ID.ValueString(),
		"repository":    repoName,
		"variable_name": variableName,
	})
}

// Custom Validators

// variableNameValidator validates variable names according to GitHub requirements
type variableNameValidator struct{}

func (v *variableNameValidator) Description(ctx context.Context) string {
	return "Variable names can only contain alphanumeric characters or underscores and must not start with a number or GITHUB_ prefix"
}

func (v *variableNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *variableNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
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
