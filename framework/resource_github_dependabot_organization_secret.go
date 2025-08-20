package framework

import (
	"context"
	"encoding/base64"
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
	"golang.org/x/crypto/nacl/box"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ resource.Resource                = &githubDependabotOrganizationSecretResource{}
	_ resource.ResourceWithConfigure   = &githubDependabotOrganizationSecretResource{}
	_ resource.ResourceWithImportState = &githubDependabotOrganizationSecretResource{}
)

type githubDependabotOrganizationSecretResource struct {
	client *githubpkg.Owner
}

type githubDependabotOrganizationSecretResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	SecretName            types.String `tfsdk:"secret_name"`
	EncryptedValue        types.String `tfsdk:"encrypted_value"`
	PlaintextValue        types.String `tfsdk:"plaintext_value"`
	Visibility            types.String `tfsdk:"visibility"`
	SelectedRepositoryIDs types.Set    `tfsdk:"selected_repository_ids"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
}

func NewGithubDependabotOrganizationSecretResource() resource.Resource {
	return &githubDependabotOrganizationSecretResource{}
}

func (r *githubDependabotOrganizationSecretResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dependabot_organization_secret"
}

func (r *githubDependabotOrganizationSecretResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages an Dependabot Secret within a GitHub organization",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the dependabot organization secret (same as secret_name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_name": schema.StringAttribute{
				Description: "Name of the secret.",
				Required:    true,
				Validators: []validator.String{
					&dependabotOrganizationSecretNameValidator{},
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
					&dependabotOrganizationSecretConflictingWithValidator{conflictsWith: []string{"plaintext_value"}},
					&dependabotOrganizationSecretBase64Validator{},
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
					&dependabotOrganizationSecretConflictingWithValidator{conflictsWith: []string{"encrypted_value"}},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"visibility": schema.StringAttribute{
				Description: "Configures the access that repositories have to the organization secret. Must be one of 'all', 'private', or 'selected'. 'selected_repository_ids' is required if set to 'selected'.",
				Required:    true,
				Validators: []validator.String{
					&dependabotOrganizationSecretVisibilityValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"selected_repository_ids": schema.SetAttribute{
				Description: "An array of repository ids that can access the organization secret.",
				Optional:    true,
				ElementType: types.Int64Type,
				Default:     setdefault.StaticValue(types.SetValueMust(types.Int64Type, []attr.Value{})),
				Validators: []validator.Set{
					&dependabotOrganizationSecretSelectedRepositoriesValidator{},
				},
			},
			"created_at": schema.StringAttribute{
				Description: "Date of 'dependabot_secret' creation.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Date of 'dependabot_secret' update.",
				Computed:    true,
			},
		},
	}
}

func (r *githubDependabotOrganizationSecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubDependabotOrganizationSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubDependabotOrganizationSecretResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	secretName := data.SecretName.ValueString()
	visibility := data.Visibility.ValueString()
	var encryptedValue string

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

	// Get the public key details for encryption
	keyId, publicKey, err := r.getDependabotOrganizationPublicKeyDetails(ctx, owner)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Get Organization Public Key",
			fmt.Sprintf("An unexpected error occurred when retrieving the organization public key: %s", err.Error()),
		)
		return
	}

	// Handle encryption based on whether plaintext or encrypted value is provided
	if !data.EncryptedValue.IsNull() && !data.EncryptedValue.IsUnknown() {
		encryptedValue = data.EncryptedValue.ValueString()
	} else if !data.PlaintextValue.IsNull() && !data.PlaintextValue.IsUnknown() {
		plaintextValue := data.PlaintextValue.ValueString()
		encryptedBytes, err := r.encryptDependabotPlaintext(plaintextValue, publicKey)
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
	eSecret := &github.DependabotEncryptedSecret{
		Name:                  secretName,
		KeyID:                 keyId,
		Visibility:            visibility,
		SelectedRepositoryIDs: selectedRepositoryIDs,
		EncryptedValue:        encryptedValue,
	}

	_, err = client.Dependabot.CreateOrUpdateOrgSecret(ctx, owner, eSecret)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Dependabot Organization Secret",
			fmt.Sprintf("An unexpected error occurred when creating the dependabot organization secret: %s", err.Error()),
		)
		return
	}

	// Set the ID
	data.ID = types.StringValue(secretName)

	tflog.Debug(ctx, "created GitHub dependabot organization secret", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"owner":       owner,
		"secret_name": secretName,
		"visibility":  visibility,
	})

	// Read the created resource to populate computed fields
	r.readGithubDependabotOrganizationSecret(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubDependabotOrganizationSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubDependabotOrganizationSecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubDependabotOrganizationSecret(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubDependabotOrganizationSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// The resource doesn't support update operations as all changes require replacement
	resp.Diagnostics.AddError(
		"Resource Update Not Supported",
		"The github_dependabot_organization_secret resource does not support updates. All changes require replacement.",
	)
}

func (r *githubDependabotOrganizationSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubDependabotOrganizationSecretResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()
	secretName := data.ID.ValueString()

	_, err := client.Dependabot.DeleteOrgSecret(ctx, owner, secretName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Dependabot Organization Secret",
			fmt.Sprintf("An unexpected error occurred when deleting the dependabot organization secret: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub dependabot organization secret", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"owner":       owner,
		"secret_name": secretName,
	})
}

func (r *githubDependabotOrganizationSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	client := r.client.V3Client()
	owner := r.client.Name()
	secretName := req.ID

	// Verify the secret exists
	secret, _, err := client.Dependabot.GetOrgSecret(ctx, owner, secretName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import Dependabot Organization Secret",
			fmt.Sprintf("Unable to read dependabot organization secret for import: %s", err.Error()),
		)
		return
	}

	data := &githubDependabotOrganizationSecretResourceModel{
		ID:         types.StringValue(secretName),
		SecretName: types.StringValue(secretName),
		Visibility: types.StringValue(secret.Visibility),
		CreatedAt:  types.StringValue(secret.CreatedAt.String()),
		UpdatedAt:  types.StringValue(secret.UpdatedAt.String()),
	}

	// Get selected repository IDs if visibility is 'selected'
	if secret.Visibility == "selected" {
		selectedRepositoryIDs := []int64{}
		opt := &github.ListOptions{
			PerPage: 30,
		}
		for {
			results, githubResp, err := client.Dependabot.ListSelectedReposForOrgSecret(ctx, owner, secretName, opt)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Read Selected Repositories",
					fmt.Sprintf("Unable to read selected repositories for organization secret: %s", err.Error()),
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

	// Note: encrypted_value or plaintext_value cannot be imported as they are not retrievable

	tflog.Debug(ctx, "imported GitHub dependabot organization secret", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"owner":       owner,
		"secret_name": secretName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubDependabotOrganizationSecretResource) getDependabotOrganizationPublicKeyDetails(ctx context.Context, owner string) (keyId, pkValue string, err error) {
	client := r.client.V3Client()

	publicKey, _, err := client.Dependabot.GetOrgPublicKey(ctx, owner)
	if err != nil {
		return keyId, pkValue, err
	}

	return publicKey.GetKeyID(), publicKey.GetKey(), err
}

func (r *githubDependabotOrganizationSecretResource) encryptDependabotPlaintext(plaintext, publicKeyB64 string) ([]byte, error) {
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

func (r *githubDependabotOrganizationSecretResource) readGithubDependabotOrganizationSecret(ctx context.Context, data *githubDependabotOrganizationSecretResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()
	secretName := data.ID.ValueString()

	secret, _, err := client.Dependabot.GetOrgSecret(ctx, owner, secretName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing dependabot organization secret from state because it no longer exists in GitHub", map[string]interface{}{
					"owner":       owner,
					"secret_name": secretName,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Dependabot Organization Secret",
			fmt.Sprintf("An unexpected error occurred when reading the dependabot organization secret: %s", err.Error()),
		)
		return
	}

	// Preserve sensitive values from state as they cannot be retrieved from the API
	// This is handled by terraform framework automatically through planning

	data.SecretName = types.StringValue(secretName)
	data.Visibility = types.StringValue(secret.Visibility)
	data.CreatedAt = types.StringValue(secret.CreatedAt.String())

	// Get selected repository IDs if visibility is 'selected'
	selectedRepositoryIDs := []int64{}
	if secret.Visibility == "selected" {
		opt := &github.ListOptions{
			PerPage: 30,
		}
		for {
			results, githubResp, err := client.Dependabot.ListSelectedReposForOrgSecret(ctx, owner, secretName, opt)
			if err != nil {
				diags.AddError(
					"Unable to Read Selected Repositories",
					fmt.Sprintf("Unable to read selected repositories for organization secret: %s", err.Error()),
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
		tflog.Info(ctx, "the dependabot organization secret has been externally updated in GitHub", map[string]interface{}{
			"id":                data.ID.ValueString(),
			"state_updated_at":  data.UpdatedAt.ValueString(),
			"github_updated_at": secret.UpdatedAt.String(),
		})
		data.ID = types.StringNull()
	} else if data.UpdatedAt.IsNull() || data.UpdatedAt.IsUnknown() {
		data.UpdatedAt = types.StringValue(secret.UpdatedAt.String())
	}

	tflog.Debug(ctx, "successfully read GitHub dependabot organization secret", map[string]interface{}{
		"id":          data.ID.ValueString(),
		"owner":       owner,
		"secret_name": secretName,
		"visibility":  secret.Visibility,
	})
}

// Custom Validators for Dependabot Organization Secrets

// dependabotOrganizationSecretNameValidator validates secret names according to GitHub requirements
type dependabotOrganizationSecretNameValidator struct{}

func (v *dependabotOrganizationSecretNameValidator) Description(ctx context.Context) string {
	return "Secret names can only contain alphanumeric characters or underscores and must not start with a number or GITHUB_ prefix"
}

func (v *dependabotOrganizationSecretNameValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *dependabotOrganizationSecretNameValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
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

// dependabotOrganizationSecretConflictingWithValidator implements validation for conflicting attributes
type dependabotOrganizationSecretConflictingWithValidator struct {
	conflictsWith []string
}

func (v *dependabotOrganizationSecretConflictingWithValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("Conflicts with: %v", v.conflictsWith)
}

func (v *dependabotOrganizationSecretConflictingWithValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *dependabotOrganizationSecretConflictingWithValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// Check if any conflicting attributes are also set
	for _, conflictingPath := range v.conflictsWith {
		var conflictingValue types.String
		diags := req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName(conflictingPath), &conflictingValue)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		if !conflictingValue.IsNull() && !conflictingValue.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Conflicting Attribute Configuration",
				fmt.Sprintf("Attribute %q cannot be specified when %q is specified.", req.Path.String(), conflictingPath),
			)
		}
	}
}

// dependabotOrganizationSecretBase64Validator validates that a string is valid base64
type dependabotOrganizationSecretBase64Validator struct{}

func (v *dependabotOrganizationSecretBase64Validator) Description(ctx context.Context) string {
	return "Value must be valid base64"
}

func (v *dependabotOrganizationSecretBase64Validator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *dependabotOrganizationSecretBase64Validator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if _, err := base64.StdEncoding.DecodeString(value); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Base64 Value",
			fmt.Sprintf("Value must be valid base64: %s", err.Error()),
		)
	}
}

// dependabotOrganizationSecretVisibilityValidator validates that visibility is one of the allowed values
type dependabotOrganizationSecretVisibilityValidator struct{}

func (v *dependabotOrganizationSecretVisibilityValidator) Description(ctx context.Context) string {
	return "Value must be one of: all, private, selected"
}

func (v *dependabotOrganizationSecretVisibilityValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *dependabotOrganizationSecretVisibilityValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
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

// dependabotOrganizationSecretSelectedRepositoriesValidator validates the relationship between visibility and selected repositories
type dependabotOrganizationSecretSelectedRepositoriesValidator struct{}

func (v *dependabotOrganizationSecretSelectedRepositoriesValidator) Description(ctx context.Context) string {
	return "Selected repository IDs can only be set when visibility is 'selected'"
}

func (v *dependabotOrganizationSecretSelectedRepositoriesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v *dependabotOrganizationSecretSelectedRepositoriesValidator) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
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
