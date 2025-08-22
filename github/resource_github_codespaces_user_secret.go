package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/isometry/terraform-provider-github/v7/github/internal/common"
)

var (
	_ resource.Resource                = &githubCodespacesUserSecretResource{}
	_ resource.ResourceWithConfigure   = &githubCodespacesUserSecretResource{}
	_ resource.ResourceWithImportState = &githubCodespacesUserSecretResource{}
)

type githubCodespacesUserSecretResource struct {
	client *Owner
}

type githubCodespacesUserSecretResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	SecretName            types.String `tfsdk:"secret_name"`
	EncryptedValue        types.String `tfsdk:"encrypted_value"`
	PlaintextValue        types.String `tfsdk:"plaintext_value"`
	SelectedRepositoryIds types.Set    `tfsdk:"selected_repository_ids"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
}

func NewGithubCodespacesUserSecretResource() resource.Resource {
	return &githubCodespacesUserSecretResource{}
}

func (r *githubCodespacesUserSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_codespaces_user_secret"
}

func (r *githubCodespacesUserSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages a Codespaces Secret within a GitHub user account",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the codespaces user secret.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
			"selected_repository_ids": schema.SetAttribute{
				Description: "An array of repository ids that can access the user secret.",
				Optional:    true,
				ElementType: types.Int64Type,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date of 'codespaces_secret' creation.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "Date of 'codespaces_secret' update.",
				Computed:    true,
			},
		},
	}
}

func (r *githubCodespacesUserSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubCodespacesUserSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubCodespacesUserSecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	secretName := data.SecretName.ValueString()
	var encryptedValue string

	// Process selected repository IDs
	var selectedRepositoryIDs github.SelectedRepoIDs
	if !data.SelectedRepositoryIds.IsNull() && !data.SelectedRepositoryIds.IsUnknown() {
		var repoIds []types.Int64
		resp.Diagnostics.Append(data.SelectedRepositoryIds.ElementsAs(ctx, &repoIds, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, id := range repoIds {
			selectedRepositoryIDs = append(selectedRepositoryIDs, id.ValueInt64())
		}
	}

	// Get the public key details for encryption
	keyId, publicKey, err := r.getCodespacesUserPublicKeyDetails(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get User Public Key",
			fmt.Sprintf("An unexpected error occurred when retrieving the user public key: %s", err.Error()),
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
		Name:                  secretName,
		KeyID:                 keyId,
		SelectedRepositoryIDs: selectedRepositoryIDs,
		EncryptedValue:        encryptedValue,
	}

	_, err = client.Codespaces.CreateOrUpdateUserSecret(ctx, eSecret)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Codespaces User Secret",
			fmt.Sprintf("An unexpected error occurred when creating the codespaces user secret: %s", err.Error()),
		)
		return
	}

	// Set the ID
	data.ID = types.StringValue(secretName)

	tflog.Debug(ctx, "created GitHub codespaces user secret", map[string]any{
		"id":          data.ID.ValueString(),
		"secret_name": secretName,
	})

	// Read the created resource to populate computed fields
	r.readGithubCodespacesUserSecret(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubCodespacesUserSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubCodespacesUserSecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubCodespacesUserSecret(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubCodespacesUserSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubCodespacesUserSecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	secretName := data.SecretName.ValueString()
	var encryptedValue string

	// Process selected repository IDs
	var selectedRepositoryIDs github.SelectedRepoIDs
	if !data.SelectedRepositoryIds.IsNull() && !data.SelectedRepositoryIds.IsUnknown() {
		var repoIds []types.Int64
		resp.Diagnostics.Append(data.SelectedRepositoryIds.ElementsAs(ctx, &repoIds, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, id := range repoIds {
			selectedRepositoryIDs = append(selectedRepositoryIDs, id.ValueInt64())
		}
	}

	// Get the public key details for encryption
	keyId, publicKey, err := r.getCodespacesUserPublicKeyDetails(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get User Public Key",
			fmt.Sprintf("An unexpected error occurred when retrieving the user public key: %s", err.Error()),
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

	// Create the encrypted secret (same API call as create)
	eSecret := &github.EncryptedSecret{
		Name:                  secretName,
		KeyID:                 keyId,
		SelectedRepositoryIDs: selectedRepositoryIDs,
		EncryptedValue:        encryptedValue,
	}

	_, err = client.Codespaces.CreateOrUpdateUserSecret(ctx, eSecret)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Codespaces User Secret",
			fmt.Sprintf("An unexpected error occurred when updating the codespaces user secret: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "updated GitHub codespaces user secret", map[string]any{
		"id":          data.ID.ValueString(),
		"secret_name": secretName,
	})

	// Read the updated resource to populate computed fields
	r.readGithubCodespacesUserSecret(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubCodespacesUserSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubCodespacesUserSecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	secretName := data.ID.ValueString()

	_, err := client.Codespaces.DeleteUserSecret(ctx, secretName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Codespaces User Secret",
			fmt.Sprintf("An unexpected error occurred when deleting the codespaces user secret: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub codespaces user secret", map[string]any{
		"id":          data.ID.ValueString(),
		"secret_name": secretName,
	})
}

func (r *githubCodespacesUserSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client := r.client.V3Client()
	secretName := req.ID

	// Verify the secret exists
	secret, _, err := client.Codespaces.GetUserSecret(ctx, secretName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import Codespaces User Secret",
			fmt.Sprintf("Unable to read codespaces user secret for import: %s", err.Error()),
		)
		return
	}

	data := &githubCodespacesUserSecretResourceModel{
		ID:         types.StringValue(secretName),
		SecretName: types.StringValue(secretName),
		CreatedAt:  types.StringValue(secret.CreatedAt.String()),
		UpdatedAt:  types.StringValue(secret.UpdatedAt.String()),
	}

	// Read selected repository IDs
	selectedRepositoryIDs := []int64{}
	opt := &github.ListOptions{
		PerPage: 30,
	}
	for {
		results, ghResp, err := client.Codespaces.ListSelectedReposForUserSecret(ctx, secretName, opt)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Import Codespaces User Secret",
				fmt.Sprintf("Unable to read selected repositories for import: %s", err.Error()),
			)
			return
		}

		for _, repo := range results.Repositories {
			selectedRepositoryIDs = append(selectedRepositoryIDs, repo.GetID())
		}

		if ghResp.NextPage == 0 {
			break
		}
		opt.Page = ghResp.NextPage
	}

	// Convert to types.Set
	if len(selectedRepositoryIDs) > 0 {
		var repoIdTypes []types.Int64
		for _, id := range selectedRepositoryIDs {
			repoIdTypes = append(repoIdTypes, types.Int64Value(id))
		}
		setVal, diags := types.SetValueFrom(ctx, types.Int64Type, repoIdTypes)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.SelectedRepositoryIds = setVal
	}

	// Note: encrypted_value or plaintext_value cannot be imported as they are not retrievable

	tflog.Debug(ctx, "imported GitHub codespaces user secret", map[string]any{
		"id":          data.ID.ValueString(),
		"secret_name": secretName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubCodespacesUserSecretResource) getCodespacesUserPublicKeyDetails(ctx context.Context) (keyId, pkValue string, err error) {
	client := r.client.V3Client()

	publicKey, _, err := client.Codespaces.GetUserPublicKey(ctx)
	if err != nil {
		return keyId, pkValue, err
	}

	return publicKey.GetKeyID(), publicKey.GetKey(), err
}

func (r *githubCodespacesUserSecretResource) readGithubCodespacesUserSecret(ctx context.Context, data *githubCodespacesUserSecretResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	secretName := data.ID.ValueString()

	secret, _, err := client.Codespaces.GetUserSecret(ctx, secretName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing codespaces user secret from state because it no longer exists in GitHub", map[string]any{
					"secret_name": secretName,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Codespaces User Secret",
			fmt.Sprintf("An unexpected error occurred when reading the codespaces user secret: %s", err.Error()),
		)
		return
	}

	// Preserve sensitive values from state as they cannot be retrieved from the API
	// This is handled by terraform framework automatically through planning

	data.SecretName = types.StringValue(secretName)
	data.CreatedAt = types.StringValue(secret.CreatedAt.String())

	// Read selected repository IDs
	selectedRepositoryIDs := []int64{}
	opt := &github.ListOptions{
		PerPage: 30,
	}
	for {
		results, ghResp, err := client.Codespaces.ListSelectedReposForUserSecret(ctx, secretName, opt)
		if err != nil {
			diags.AddError(
				"Unable to Read Selected Repositories",
				fmt.Sprintf("An unexpected error occurred when reading selected repositories: %s", err.Error()),
			)
			return
		}

		for _, repo := range results.Repositories {
			selectedRepositoryIDs = append(selectedRepositoryIDs, repo.GetID())
		}

		if ghResp.NextPage == 0 {
			break
		}
		opt.Page = ghResp.NextPage
	}

	// Convert to types.Set
	if len(selectedRepositoryIDs) > 0 {
		var repoIdTypes []types.Int64
		for _, id := range selectedRepositoryIDs {
			repoIdTypes = append(repoIdTypes, types.Int64Value(id))
		}
		setVal, diagsConvert := types.SetValueFrom(ctx, types.Int64Type, repoIdTypes)
		diags.Append(diagsConvert...)
		if diags.HasError() {
			return
		}
		data.SelectedRepositoryIds = setVal
	} else {
		data.SelectedRepositoryIds = types.SetNull(types.Int64Type)
	}

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
		tflog.Info(ctx, "the secret has been externally updated in GitHub", map[string]any{
			"id":                data.ID.ValueString(),
			"state_updated_at":  data.UpdatedAt.ValueString(),
			"github_updated_at": secret.UpdatedAt.String(),
		})
		data.ID = types.StringNull()
	} else if data.UpdatedAt.IsNull() || data.UpdatedAt.IsUnknown() {
		data.UpdatedAt = types.StringValue(secret.UpdatedAt.String())
	}

	tflog.Debug(ctx, "successfully read GitHub codespaces user secret", map[string]any{
		"id":          data.ID.ValueString(),
		"secret_name": secretName,
	})
}
