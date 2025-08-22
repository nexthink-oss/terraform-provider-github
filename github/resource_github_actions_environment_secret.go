package github

import (
	"context"
	"encoding/base64"
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

	"github.com/isometry/terraform-provider-github/v7/github/internal/common"
)

var (
	_ resource.Resource                = &githubActionsEnvironmentSecretResource{}
	_ resource.ResourceWithConfigure   = &githubActionsEnvironmentSecretResource{}
	_ resource.ResourceWithImportState = &githubActionsEnvironmentSecretResource{}
)

type githubActionsEnvironmentSecretResource struct {
	client *Owner
}

type githubActionsEnvironmentSecretResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Repository     types.String `tfsdk:"repository"`
	Environment    types.String `tfsdk:"environment"`
	SecretName     types.String `tfsdk:"secret_name"`
	EncryptedValue types.String `tfsdk:"encrypted_value"`
	PlaintextValue types.String `tfsdk:"plaintext_value"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewGithubActionsEnvironmentSecretResource() resource.Resource {
	return &githubActionsEnvironmentSecretResource{}
}

func (r *githubActionsEnvironmentSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_environment_secret"
}

func (r *githubActionsEnvironmentSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages an Action Secret within a GitHub repository environment",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the actions environment secret (repository:environment:secret_name).",
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
			"secret_name": schema.StringAttribute{
				Description: "Name of the secret.",
				Required:    true,
				Validators: []validator.String{
					common.NewSecretNameValidator(),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"encrypted_value": schema.StringAttribute{
				Description: "Encrypted value of the secret using the GitHub public key in Base64 format.",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					common.NewConflictingWithValidator([]string{"plaintext_value"}),
					&base64Validator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"plaintext_value": schema.StringAttribute{
				Description: "Plaintext value of the secret to be encrypted.",
				Optional:    true,
				Sensitive:   true,
				Validators: []validator.String{
					common.NewConflictingWithValidator([]string{"encrypted_value"}),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date of 'actions_environment_secret' creation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Date of 'actions_environment_secret' update.",
				Computed:    true,
			},
		},
	}
}

func (r *githubActionsEnvironmentSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubActionsEnvironmentSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubActionsEnvironmentSecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repo := data.Repository.ValueString()
	environment := data.Environment.ValueString()
	escapedEnvName := url.PathEscape(environment)
	secretName := data.SecretName.ValueString()
	var encryptedValue string

	// Get the repository details first
	repoInfo, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get Repository",
			fmt.Sprintf("An unexpected error occurred when retrieving the repository: %s", err.Error()),
		)
		return
	}

	// Get the public key details for encryption
	keyId, publicKey, err := r.getEnvironmentPublicKeyDetails(ctx, int(repoInfo.GetID()), escapedEnvName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get Environment Public Key",
			fmt.Sprintf("An unexpected error occurred when retrieving the environment public key: %s", err.Error()),
		)
		return
	}

	// Handle encryption based on whether plaintext or encrypted value is provided
	if !data.EncryptedValue.IsNull() && !data.EncryptedValue.IsUnknown() {
		encryptedValue = data.EncryptedValue.ValueString()
	} else if !data.PlaintextValue.IsNull() && !data.PlaintextValue.IsUnknown() {
		plaintextValue := data.PlaintextValue.ValueString()
		encryptedBytes, err := common.EncryptPlaintext(plaintextValue, publicKey)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Encrypt Secret",
				fmt.Sprintf("An unexpected error occurred when encrypting the secret: %s", err.Error()),
			)
			return
		}
		encryptedValue = base64.StdEncoding.EncodeToString(encryptedBytes)
	} else {
		resp.Diagnostics.AddError(
			"Missing Secret Value",
			"Either 'plaintext_value' or 'encrypted_value' must be provided.",
		)
		return
	}

	// Create the encrypted secret
	eSecret := &github.EncryptedSecret{
		Name:           secretName,
		KeyID:          keyId,
		EncryptedValue: encryptedValue,
	}

	_, err = client.Actions.CreateOrUpdateEnvSecret(ctx, int(repoInfo.GetID()), escapedEnvName, eSecret)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Actions Environment Secret",
			fmt.Sprintf("An unexpected error occurred when creating the actions environment secret: %s", err.Error()),
		)
		return
	}

	// Set the ID and read the created resource
	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s", repo, environment, secretName))

	tflog.Debug(ctx, "created GitHub actions environment secret", map[string]any{
		"id":          data.ID.ValueString(),
		"repository":  repo,
		"environment": environment,
		"secret_name": secretName,
	})

	// Read the created resource to populate computed fields
	r.readGithubActionsEnvironmentSecret(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsEnvironmentSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubActionsEnvironmentSecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubActionsEnvironmentSecret(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubActionsEnvironmentSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// The resource doesn't support update operations as all changes require replacement
	resp.Diagnostics.AddError(
		"Resource Update Not Supported",
		"The github_actions_environment_secret resource does not support updates. All changes require replacement.",
	)
}

func (r *githubActionsEnvironmentSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubActionsEnvironmentSecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName, envName, secretName, err := r.parseThreePartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse ID",
			fmt.Sprintf("Unable to parse ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	escapedEnvName := url.PathEscape(envName)
	repo, _, err := client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get Repository",
			fmt.Sprintf("An unexpected error occurred when retrieving the repository: %s", err.Error()),
		)
		return
	}

	_, err = client.Actions.DeleteEnvSecret(ctx, int(repo.GetID()), escapedEnvName, secretName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Actions Environment Secret",
			fmt.Sprintf("An unexpected error occurred when deleting the actions environment secret: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub actions environment secret", map[string]any{
		"id":          data.ID.ValueString(),
		"repository":  repoName,
		"environment": envName,
		"secret_name": secretName,
	})
}

func (r *githubActionsEnvironmentSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "<repository>/<environment>/<secret_name>"
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Invalid ID specified. Supplied ID must be written as <repository>/<environment>/<secret_name>. Got: %q", req.ID),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	repoName := parts[0]
	envName := parts[1]
	secretName := parts[2]
	escapedEnvName := url.PathEscape(envName)

	// Get the repository details first
	repo, _, err := client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get Repository",
			fmt.Sprintf("Unable to read repository for import: %s", err.Error()),
		)
		return
	}

	// Verify the secret exists
	secret, _, err := client.Actions.GetEnvSecret(ctx, int(repo.GetID()), escapedEnvName, secretName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import Actions Environment Secret",
			fmt.Sprintf("Unable to read actions environment secret for import: %s", err.Error()),
		)
		return
	}

	data := &githubActionsEnvironmentSecretResourceModel{
		ID:          types.StringValue(fmt.Sprintf("%s:%s:%s", repoName, envName, secretName)),
		Repository:  types.StringValue(repoName),
		Environment: types.StringValue(envName),
		SecretName:  types.StringValue(secretName),
		CreatedAt:   types.StringValue(secret.CreatedAt.String()),
		UpdatedAt:   types.StringValue(secret.UpdatedAt.String()),
	}

	// Note: encrypted_value or plaintext_value cannot be imported as they are not retrievable

	tflog.Debug(ctx, "imported GitHub actions environment secret", map[string]any{
		"id":          data.ID.ValueString(),
		"repository":  repoName,
		"environment": envName,
		"secret_name": secretName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubActionsEnvironmentSecretResource) getEnvironmentPublicKeyDetails(ctx context.Context, repoID int, envName string) (keyId, pkValue string, err error) {
	client := r.client.V3Client()

	publicKey, _, err := client.Actions.GetEnvPublicKey(ctx, repoID, envName)
	if err != nil {
		return keyId, pkValue, err
	}

	return publicKey.GetKeyID(), publicKey.GetKey(), err
}

func (r *githubActionsEnvironmentSecretResource) parseThreePartID(id string) (string, string, string, error) {
	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("unexpected ID format (%q); expected repository:environment:secret_name", id)
	}

	return parts[0], parts[1], parts[2], nil
}

func (r *githubActionsEnvironmentSecretResource) readGithubActionsEnvironmentSecret(ctx context.Context, data *githubActionsEnvironmentSecretResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	repoName, envName, secretName, err := r.parseThreePartID(data.ID.ValueString())
	if err != nil {
		diags.AddError(
			"Unable to Parse ID",
			fmt.Sprintf("Unable to parse ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	escapedEnvName := url.PathEscape(envName)

	repo, _, err := client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing actions environment secret from state because repository no longer exists in GitHub", map[string]any{
					"owner":       owner,
					"repository":  repoName,
					"environment": envName,
					"secret_name": secretName,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Repository",
			fmt.Sprintf("An unexpected error occurred when reading the repository: %s", err.Error()),
		)
		return
	}

	secret, _, err := client.Actions.GetEnvSecret(ctx, int(repo.GetID()), escapedEnvName, secretName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing actions environment secret from state because it no longer exists in GitHub", map[string]any{
					"owner":       owner,
					"repository":  repoName,
					"environment": envName,
					"secret_name": secretName,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Actions Environment Secret",
			fmt.Sprintf("An unexpected error occurred when reading the actions environment secret: %s", err.Error()),
		)
		return
	}

	// Preserve sensitive values from state as they cannot be retrieved from the API
	// This is handled by terraform framework automatically through planning

	data.Repository = types.StringValue(repoName)
	data.Environment = types.StringValue(envName)
	data.SecretName = types.StringValue(secretName)
	data.CreatedAt = types.StringValue(secret.CreatedAt.String())

	// This is a drift detection mechanism based on timestamps.
	//
	// If we do not currently store the "updated_at" field, it means we've only
	// just created the resource and the value is most likely what we want it to
	// be.
	//
	// If the resource is changed externally in the meantime then reading back
	// the last update timestamp will return a result different than the
	// timestamp we've persisted in the state. In that case, we can no longer
	// trust that the value (which we don't see) is equal to what we've declared
	// previously.
	//
	// The only solution to enforce consistency between is to mark the resource
	// as deleted (unset the ID) in order to fix potential drift by recreating
	// the resource.
	if !data.UpdatedAt.IsNull() && !data.UpdatedAt.IsUnknown() && data.UpdatedAt.ValueString() != secret.UpdatedAt.String() {
		tflog.Info(ctx, "the environment secret has been externally updated in GitHub", map[string]any{
			"id":                data.ID.ValueString(),
			"state_updated_at":  data.UpdatedAt.ValueString(),
			"github_updated_at": secret.UpdatedAt.String(),
		})
		data.ID = types.StringNull()
	} else if data.UpdatedAt.IsNull() || data.UpdatedAt.IsUnknown() {
		data.UpdatedAt = types.StringValue(secret.UpdatedAt.String())
	}

	tflog.Debug(ctx, "successfully read GitHub actions environment secret", map[string]any{
		"id":          data.ID.ValueString(),
		"repository":  repoName,
		"environment": envName,
		"secret_name": secretName,
	})
}

// base64Validator validates that a string is valid base64
type base64Validator struct{}

func (v *base64Validator) Description(ctx context.Context) string {
	return "Value must be valid base64"
}

func (v *base64Validator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *base64Validator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	if _, err := base64.StdEncoding.DecodeString(value); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Base64 Value",
			"The value must be valid base64 encoded string",
		)
	}
}
