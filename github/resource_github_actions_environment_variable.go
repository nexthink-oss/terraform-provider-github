package github

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
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
	_ resource.Resource                = &githubActionsEnvironmentVariableResource{}
	_ resource.ResourceWithConfigure   = &githubActionsEnvironmentVariableResource{}
	_ resource.ResourceWithImportState = &githubActionsEnvironmentVariableResource{}
)

type githubActionsEnvironmentVariableResource struct {
	client *Owner
}

type githubActionsEnvironmentVariableResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Repository   types.String `tfsdk:"repository"`
	Environment  types.String `tfsdk:"environment"`
	VariableName types.String `tfsdk:"variable_name"`
	Value        types.String `tfsdk:"value"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func NewGithubActionsEnvironmentVariableResource() resource.Resource {
	return &githubActionsEnvironmentVariableResource{}
}

func (r *githubActionsEnvironmentVariableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_environment_variable"
}

func (r *githubActionsEnvironmentVariableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages an Action variable within a GitHub repository environment",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the actions environment variable (repository:environment:variable_name).",
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
			"environment": schema.StringAttribute{
				Description: "Name of the environment.",
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
			},
			"updated_at": schema.StringAttribute{
				Description: "Date of 'actions_variable' update.",
				Computed:    true,
			},
		},
	}
}

func (r *githubActionsEnvironmentVariableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubActionsEnvironmentVariableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubActionsEnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repo := data.Repository.ValueString()
	environment := data.Environment.ValueString()
	escapedEnvName := url.PathEscape(environment)
	variableName := data.VariableName.ValueString()
	value := data.Value.ValueString()

	// Create the variable
	variable := &github.ActionsVariable{
		Name:  variableName,
		Value: value,
	}

	_, err := client.Actions.CreateEnvVariable(ctx, owner, repo, escapedEnvName, variable)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Actions Environment Variable",
			fmt.Sprintf("An unexpected error occurred when creating the actions environment variable: %s", err.Error()),
		)
		return
	}

	// Set the ID and read the created resource
	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s", repo, environment, variableName))

	tflog.Debug(ctx, "created GitHub actions environment variable", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"repository":    repo,
		"environment":   environment,
		"variable_name": variableName,
	})

	// Read the created resource to populate computed fields
	r.readGithubActionsEnvironmentVariable(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsEnvironmentVariableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubActionsEnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubActionsEnvironmentVariable(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsEnvironmentVariableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubActionsEnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repo := data.Repository.ValueString()
	environment := data.Environment.ValueString()
	escapedEnvName := url.PathEscape(environment)
	variableName := data.VariableName.ValueString()
	value := data.Value.ValueString()

	// Update the variable
	variable := &github.ActionsVariable{
		Name:  variableName,
		Value: value,
	}

	_, err := client.Actions.UpdateEnvVariable(ctx, owner, repo, escapedEnvName, variable)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Actions Environment Variable",
			fmt.Sprintf("An unexpected error occurred when updating the actions environment variable: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "updated GitHub actions environment variable", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"repository":    repo,
		"environment":   environment,
		"variable_name": variableName,
	})

	// Read the updated resource to populate computed fields
	r.readGithubActionsEnvironmentVariable(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsEnvironmentVariableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubActionsEnvironmentVariableResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName, envName, variableName, err := r.parseThreePartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse ID",
			fmt.Sprintf("Unable to parse ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	escapedEnvName := url.PathEscape(envName)

	_, err = client.Actions.DeleteEnvVariable(ctx, owner, repoName, escapedEnvName, variableName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Actions Environment Variable",
			fmt.Sprintf("An unexpected error occurred when deleting the actions environment variable: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub actions environment variable", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"repository":    repoName,
		"environment":   envName,
		"variable_name": variableName,
	})
}

func (r *githubActionsEnvironmentVariableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "<repository>/<environment>/<variable_name>"
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Invalid ID specified. Supplied ID must be written as <repository>/<environment>/<variable_name>. Got: %q", req.ID),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	repoName := parts[0]
	envName := parts[1]
	variableName := parts[2]
	escapedEnvName := url.PathEscape(envName)

	// Verify the variable exists
	variable, _, err := client.Actions.GetEnvVariable(ctx, owner, repoName, escapedEnvName, variableName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import Actions Environment Variable",
			fmt.Sprintf("Unable to read actions environment variable for import: %s", err.Error()),
		)
		return
	}

	data := &githubActionsEnvironmentVariableResourceModel{
		ID:           types.StringValue(fmt.Sprintf("%s:%s:%s", repoName, envName, variableName)),
		Repository:   types.StringValue(repoName),
		Environment:  types.StringValue(envName),
		VariableName: types.StringValue(variableName),
		Value:        types.StringValue(variable.Value),
		CreatedAt:    types.StringValue(variable.CreatedAt.String()),
		UpdatedAt:    types.StringValue(variable.UpdatedAt.String()),
	}

	tflog.Debug(ctx, "imported GitHub actions environment variable", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"repository":    repoName,
		"environment":   envName,
		"variable_name": variableName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubActionsEnvironmentVariableResource) parseThreePartID(id string) (string, string, string, error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("unexpected ID format (%q); expected repository:environment:variable_name", id)
	}

	return parts[0], parts[1], parts[2], nil
}

func (r *githubActionsEnvironmentVariableResource) readGithubActionsEnvironmentVariable(ctx context.Context, data *githubActionsEnvironmentVariableResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	repoName, envName, variableName, err := r.parseThreePartID(data.ID.ValueString())
	if err != nil {
		diags.AddError(
			"Unable to Parse ID",
			fmt.Sprintf("Unable to parse ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	escapedEnvName := url.PathEscape(envName)

	variable, _, err := client.Actions.GetEnvVariable(ctx, owner, repoName, escapedEnvName, variableName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing actions environment variable from state because it no longer exists in GitHub", map[string]interface{}{
					"owner":         owner,
					"repository":    repoName,
					"environment":   envName,
					"variable_name": variableName,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Actions Environment Variable",
			fmt.Sprintf("An unexpected error occurred when reading the actions environment variable: %s", err.Error()),
		)
		return
	}

	data.Repository = types.StringValue(repoName)
	data.Environment = types.StringValue(envName)
	data.VariableName = types.StringValue(variableName)
	data.Value = types.StringValue(variable.Value)
	data.CreatedAt = types.StringValue(variable.CreatedAt.String())
	data.UpdatedAt = types.StringValue(variable.UpdatedAt.String())

	tflog.Debug(ctx, "successfully read GitHub actions environment variable", map[string]interface{}{
		"id":            data.ID.ValueString(),
		"repository":    repoName,
		"environment":   envName,
		"variable_name": variableName,
	})
}
