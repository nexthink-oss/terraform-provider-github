package framework

import (
	"context"
	"encoding/base64"
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
	"golang.org/x/crypto/nacl/box"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ resource.Resource                = &githubCodespacesSecretResource{}
	_ resource.ResourceWithConfigure   = &githubCodespacesSecretResource{}
	_ resource.ResourceWithImportState = &githubCodespacesSecretResource{}
)

type githubCodespacesSecretResource struct {
	client *githubpkg.Owner
}

type githubCodespacesSecretResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Repository     types.String `tfsdk:"repository"`
	SecretName     types.String `tfsdk:"secret_name"`
	EncryptedValue types.String `tfsdk:"encrypted_value"`
	PlaintextValue types.String `tfsdk:"plaintext_value"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func NewGithubCodespacesSecretResource() resource.Resource {
	return &githubCodespacesSecretResource{}
}

func (r *githubCodespacesSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_codespaces_secret"
}

func (r *githubCodespacesSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages an Codespaces Secret within a GitHub repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the codespaces secret (repository:secret_name).",
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
			"secret_name": schema.StringAttribute{
				Description: "Name of the secret.",
				Required:    true,
				Validators: []validator.String{
					&codespacesSecretNameValidator{},
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
					&conflictingWithValidator{conflictsWith: []string{"plaintext_value"}},
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
					&conflictingWithValidator{conflictsWith: []string{"encrypted_value"}},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date of 'codespaces_secret' creation.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Date of 'codespaces_secret' update.",
				Computed:    true,
			},
		},
	}
}

func (r *githubCodespacesSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*githubpkg.Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *githubCodespacesSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubCodespacesSecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repo := data.Repository.ValueString()
	secretName := data.SecretName.ValueString()
	var encryptedValue string

	// Get the public key details for encryption
	keyId, publicKey, err := r.getCodespacesPublicKeyDetails(ctx, owner, repo)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get Repository Public Key",
			fmt.Sprintf("An unexpected error occurred when retrieving the repository public key: %s", err.Error()),
		)
		return
	}

	// Handle encryption based on whether plaintext or encrypted value is provided
	if !data.EncryptedValue.IsNull() && !data.EncryptedValue.IsUnknown() {
		encryptedValue = data.EncryptedValue.ValueString()
	} else if !data.PlaintextValue.IsNull() && !data.PlaintextValue.IsUnknown() {
		plaintextValue := data.PlaintextValue.ValueString()
		encryptedBytes, err := r.encryptPlaintext(plaintextValue, publicKey)
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

	_, err = client.Codespaces.CreateOrUpdateRepoSecret(ctx, owner, repo, eSecret)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Codespaces Secret",
			fmt.Sprintf("An unexpected error occurred when creating the codespaces secret: %s", err.Error()),
		)
		return
	}

	// Set the ID and read the created resource
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", repo, secretName))

	tflog.Debug(ctx, "created GitHub codespaces secret", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"repository":  repo,
		"secret_name": secretName,
	})

	// Read the created resource to populate computed fields
	r.readGithubCodespacesSecret(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubCodespacesSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubCodespacesSecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubCodespacesSecret(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubCodespacesSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// The resource doesn't support update operations as all changes require replacement
	resp.Diagnostics.AddError(
		"Resource Update Not Supported",
		"The github_codespaces_secret resource does not support updates. All changes require replacement.",
	)
}

func (r *githubCodespacesSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubCodespacesSecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName, secretName, err := r.parseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse ID",
			fmt.Sprintf("Unable to parse ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, err = client.Codespaces.DeleteRepoSecret(ctx, owner, repoName, secretName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Codespaces Secret",
			fmt.Sprintf("An unexpected error occurred when deleting the codespaces secret: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub codespaces secret", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"repository":  repoName,
		"secret_name": secretName,
	})
}

func (r *githubCodespacesSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "<repository>/<secret_name>"
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Invalid ID specified. Supplied ID must be written as <repository>/<secret_name>. Got: %q", req.ID),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	repoName := parts[0]
	secretName := parts[1]

	// Verify the secret exists
	secret, _, err := client.Codespaces.GetRepoSecret(ctx, owner, repoName, secretName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import Codespaces Secret",
			fmt.Sprintf("Unable to read codespaces secret for import: %s", err.Error()),
		)
		return
	}

	data := &githubCodespacesSecretResourceModel{
		ID:         types.StringValue(fmt.Sprintf("%s:%s", repoName, secretName)),
		Repository: types.StringValue(repoName),
		SecretName: types.StringValue(secretName),
		CreatedAt:  types.StringValue(secret.CreatedAt.String()),
		UpdatedAt:  types.StringValue(secret.UpdatedAt.String()),
	}

	// Note: encrypted_value or plaintext_value cannot be imported as they are not retrievable

	tflog.Debug(ctx, "imported GitHub codespaces secret", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"repository":  repoName,
		"secret_name": secretName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubCodespacesSecretResource) getCodespacesPublicKeyDetails(ctx context.Context, owner, repository string) (keyId, pkValue string, err error) {
	client := r.client.V3Client()

	publicKey, _, err := client.Codespaces.GetRepoPublicKey(ctx, owner, repository)
	if err != nil {
		return keyId, pkValue, err
	}

	return publicKey.GetKeyID(), publicKey.GetKey(), err
}

func (r *githubCodespacesSecretResource) encryptPlaintext(plaintext, publicKeyB64 string) ([]byte, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return nil, err
	}

	var publicKeyBytes32 [32]byte
	copiedLen := copy(publicKeyBytes32[:], publicKeyBytes)
	if copiedLen == 0 {
		return nil, fmt.Errorf("could not convert publicKey to bytes")
	}

	plaintextBytes := []byte(plaintext)
	var encryptedBytes []byte

	cipherText, err := box.SealAnonymous(encryptedBytes, plaintextBytes, &publicKeyBytes32, nil)
	if err != nil {
		return nil, err
	}

	return cipherText, nil
}

func (r *githubCodespacesSecretResource) parseTwoPartID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected ID format (%q); expected repository:secret_name", id)
	}

	return parts[0], parts[1], nil
}

func (r *githubCodespacesSecretResource) readGithubCodespacesSecret(ctx context.Context, data *githubCodespacesSecretResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	repoName, secretName, err := r.parseTwoPartID(data.ID.ValueString())
	if err != nil {
		diags.AddError(
			"Unable to Parse ID",
			fmt.Sprintf("Unable to parse ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	secret, _, err := client.Codespaces.GetRepoSecret(ctx, owner, repoName, secretName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing codespaces secret from state because it no longer exists in GitHub", map[string]interface{}{
					"owner":       owner,
					"repository":  repoName,
					"secret_name": secretName,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Codespaces Secret",
			fmt.Sprintf("An unexpected error occurred when reading the codespaces secret: %s", err.Error()),
		)
		return
	}

	// Preserve sensitive values from state as they cannot be retrieved from the API
	// This is handled by terraform framework automatically through planning

	data.Repository = types.StringValue(repoName)
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
		tflog.Info(ctx, "the secret has been externally updated in GitHub", map[string]interface{}{
			"id":                data.ID.ValueString(),
			"state_updated_at":  data.UpdatedAt.ValueString(),
			"github_updated_at": secret.UpdatedAt.String(),
		})
		data.ID = types.StringNull()
	} else if data.UpdatedAt.IsNull() || data.UpdatedAt.IsUnknown() {
		data.UpdatedAt = types.StringValue(secret.UpdatedAt.String())
	}

	tflog.Debug(ctx, "successfully read GitHub codespaces secret", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"repository":  repoName,
		"secret_name": secretName,
	})
}

// Custom Validators

// codespacesSecretNameValidator validates secret names according to GitHub requirements
type codespacesSecretNameValidator struct{}

func (v *codespacesSecretNameValidator) Description(ctx context.Context) string {
	return "Secret names can only contain alphanumeric characters or underscores and must not start with a number or GITHUB_ prefix"
}

func (v *codespacesSecretNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *codespacesSecretNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	// https://docs.github.com/en/actions/reference/encrypted-secrets#naming-your-secrets
	secretNameRegexp := regexp.MustCompile("^[a-zA-Z_][a-zA-Z0-9_]*$")

	if !secretNameRegexp.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Secret Name",
			"Secret names can only contain alphanumeric characters or underscores and must not start with a number",
		)
	}

	if strings.HasPrefix(strings.ToUpper(value), "GITHUB_") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Secret Name",
			"Secret names must not start with the GITHUB_ prefix",
		)
	}
}
