package github

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubRepositoryDeployKeyResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryDeployKeyResource{}
	_ resource.ResourceWithImportState = &githubRepositoryDeployKeyResource{}
)

type githubRepositoryDeployKeyResource struct {
	client *Owner
}

type githubRepositoryDeployKeyResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Key        types.String `tfsdk:"key"`
	ReadOnly   types.Bool   `tfsdk:"read_only"`
	Repository types.String `tfsdk:"repository"`
	Title      types.String `tfsdk:"title"`
	Etag       types.String `tfsdk:"etag"`
}

func NewGithubRepositoryDeployKeyResource() resource.Resource {
	return &githubRepositoryDeployKeyResource{}
}

func (r *githubRepositoryDeployKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_deploy_key"
}

func (r *githubRepositoryDeployKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub repository deploy key resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the deploy key (repository:key_id).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Description: "A SSH key.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					&deployKeyDiffSuppressor{},
				},
			},
			"read_only": schema.BoolAttribute{
				Description: "A boolean qualifying the key to be either read only or read/write.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "Name of the GitHub repository.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"title": schema.StringAttribute{
				Description: "A title.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"etag": schema.StringAttribute{
				Description: "An etag representing the state of the deploy key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubRepositoryDeployKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubRepositoryDeployKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubRepositoryDeployKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()
	key := data.Key.ValueString()
	title := data.Title.ValueString()
	readOnly := data.ReadOnly.ValueBool()

	resultKey, _, err := client.Repositories.CreateKey(ctx, owner, repoName, &github.Key{
		Key:      github.Ptr(key),
		Title:    github.Ptr(title),
		ReadOnly: github.Ptr(readOnly),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Repository Deploy Key",
			fmt.Sprintf("An unexpected error occurred when creating the repository deploy key: %s", err.Error()),
		)
		return
	}

	keyID := strconv.FormatInt(resultKey.GetID(), 10)
	data.ID = types.StringValue(r.buildTwoPartID(repoName, keyID))

	tflog.Debug(ctx, "created GitHub repository deploy key", map[string]any{
		"id":         data.ID.ValueString(),
		"repository": repoName,
		"title":      title,
		"key_id":     keyID,
	})

	// Read the created resource to populate computed fields
	r.readGithubRepositoryDeployKey(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryDeployKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubRepositoryDeployKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubRepositoryDeployKey(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryDeployKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Deploy keys are immutable in the GitHub API - all changes require replacement
	resp.Diagnostics.AddError(
		"Resource Update Not Supported",
		"The github_repository_deploy_key resource does not support updates. All changes require replacement.",
	)
}

func (r *githubRepositoryDeployKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubRepositoryDeployKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName, keyIDString, err := r.parseTwoPartID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse ID",
			fmt.Sprintf("Unable to parse ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	keyID, err := strconv.ParseInt(keyIDString, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Key ID",
			fmt.Sprintf("Unable to parse key ID '%s': %s", keyIDString, err.Error()),
		)
		return
	}

	_, err = client.Repositories.DeleteKey(ctx, owner, repoName, keyID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Repository Deploy Key",
			fmt.Sprintf("An unexpected error occurred when deleting the repository deploy key: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub repository deploy key", map[string]any{
		"id":         data.ID.ValueString(),
		"repository": repoName,
		"key_id":     keyIDString,
	})
}

func (r *githubRepositoryDeployKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "<repository>:<key_id>"
	repoName, keyIDString, err := r.parseTwoPartID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Invalid ID specified. Supplied ID must be written as <repository>:<key_id>. Got: %q. Error: %s", req.ID, err.Error()),
		)
		return
	}

	keyID, err := strconv.ParseInt(keyIDString, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Key ID",
			fmt.Sprintf("Unable to parse key ID '%s': %s", keyIDString, err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	// Verify the deploy key exists
	key, _, err := client.Repositories.GetKey(ctx, owner, repoName, keyID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import Repository Deploy Key",
			fmt.Sprintf("Unable to read repository deploy key for import: %s", err.Error()),
		)
		return
	}

	data := &githubRepositoryDeployKeyResourceModel{
		ID:         types.StringValue(req.ID),
		Repository: types.StringValue(repoName),
		Key:        types.StringValue(key.GetKey()),
		Title:      types.StringValue(key.GetTitle()),
		ReadOnly:   types.BoolValue(key.GetReadOnly()),
	}

	tflog.Debug(ctx, "imported GitHub repository deploy key", map[string]any{
		"id":         data.ID.ValueString(),
		"repository": repoName,
		"key_id":     keyIDString,
		"title":      key.GetTitle(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubRepositoryDeployKeyResource) buildTwoPartID(a, b string) string {
	return fmt.Sprintf("%s:%s", a, b)
}

func (r *githubRepositoryDeployKeyResource) parseTwoPartID(id string) (string, string, error) {
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected ID format (%q); expected repository:key_id", id)
	}

	return parts[0], parts[1], nil
}

func (r *githubRepositoryDeployKeyResource) readGithubRepositoryDeployKey(ctx context.Context, data *githubRepositoryDeployKeyResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	repoName, keyIDString, err := r.parseTwoPartID(data.ID.ValueString())
	if err != nil {
		diags.AddError(
			"Unable to Parse ID",
			fmt.Sprintf("Unable to parse ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	keyID, err := strconv.ParseInt(keyIDString, 10, 64)
	if err != nil {
		diags.AddError(
			"Unable to Parse Key ID",
			fmt.Sprintf("Unable to parse key ID '%s': %s", keyIDString, err.Error()),
		)
		return
	}

	// Set up context with etag for conditional requests if we have it
	apiCtx := ctx
	if !data.Etag.IsNull() && !data.Etag.IsUnknown() {
		// The GitHub client expects the etag in a specific context key
		// This follows the pattern from the SDKv2 implementation
		apiCtx = context.WithValue(ctx, CtxEtag, data.Etag.ValueString())
	}

	key, resp, err := client.Repositories.GetKey(apiCtx, owner, repoName, keyID)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				// Resource hasn't changed, no need to update state
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing repository deploy key from state because it no longer exists in GitHub", map[string]any{
					"owner":      owner,
					"repository": repoName,
					"key_id":     keyIDString,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Repository Deploy Key",
			fmt.Sprintf("An unexpected error occurred when reading the repository deploy key: %s", err.Error()),
		)
		return
	}

	// Update the state with the latest values
	data.Repository = types.StringValue(repoName)
	data.Key = types.StringValue(key.GetKey())
	data.Title = types.StringValue(key.GetTitle())
	data.ReadOnly = types.BoolValue(key.GetReadOnly())

	// Update etag if present in response
	if resp != nil && resp.Header.Get("ETag") != "" {
		data.Etag = types.StringValue(resp.Header.Get("ETag"))
	}

	tflog.Debug(ctx, "successfully read GitHub repository deploy key", map[string]any{
		"id":         data.ID.ValueString(),
		"repository": repoName,
		"key_id":     keyIDString,
		"title":      key.GetTitle(),
	})
}

// Custom Plan Modifiers

// deployKeyDiffSuppressor implements a plan modifier to suppress differences
// in deploy keys where only whitespace or email addresses differ
type deployKeyDiffSuppressor struct{}

func (m *deployKeyDiffSuppressor) Description(ctx context.Context) string {
	return "Suppresses deploy key differences for trailing whitespace and email addresses"
}

func (m *deployKeyDiffSuppressor) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m *deployKeyDiffSuppressor) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// Only apply suppression during plan phase
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	configValue := req.ConfigValue.ValueString()
	if req.StateValue.IsNull() {
		return
	}

	stateValue := req.StateValue.ValueString()

	// Apply the same suppression logic as the SDKv2 version
	if m.suppressDeployKeyDiff(stateValue, configValue) {
		resp.PlanValue = req.StateValue
	}
}

func (m *deployKeyDiffSuppressor) suppressDeployKeyDiff(oldV, newV string) bool {
	newV = strings.TrimSpace(newV)
	keyRe := regexp.MustCompile(`^([a-z0-9-]+ [^\s]+)( [^\s]+)?$`)
	newTrimmed := keyRe.ReplaceAllString(newV, "$1")

	return oldV == newTrimmed
}
