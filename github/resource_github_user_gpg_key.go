package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubUserGpgKeyResource{}
	_ resource.ResourceWithConfigure   = &githubUserGpgKeyResource{}
	_ resource.ResourceWithImportState = &githubUserGpgKeyResource{}
)

type githubUserGpgKeyResource struct {
	client *Owner
}

type githubUserGpgKeyResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ArmoredPublicKey types.String `tfsdk:"armored_public_key"`
	KeyID            types.String `tfsdk:"key_id"`
	Etag             types.String `tfsdk:"etag"`
}

func NewGithubUserGpgKeyResource() resource.Resource {
	return &githubUserGpgKeyResource{}
}

func (r *githubUserGpgKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_gpg_key"
}

func (r *githubUserGpgKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub user's GPG key resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the GPG key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"armored_public_key": schema.StringAttribute{
				Description: "Your public GPG key, generated in ASCII-armored format.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_id": schema.StringAttribute{
				Description: "The key ID of the GPG key.",
				Computed:    true,
			},
			"etag": schema.StringAttribute{
				Description: "The ETag of the GPG key.",
				Computed:    true,
			},
		},
	}
}

func (r *githubUserGpgKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubUserGpgKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubUserGpgKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	armoredPublicKey := data.ArmoredPublicKey.ValueString()

	key, _, err := r.client.V3Client().Users.CreateGPGKey(ctx, armoredPublicKey)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create GitHub GPG Key",
			"An unexpected error occurred when creating the GitHub GPG key. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(key.GetID(), 10))

	tflog.Debug(ctx, "created GitHub user GPG key", map[string]interface{}{
		"id":     data.ID.ValueString(),
		"key_id": key.GetKeyID(),
	})

	// Read the created resource to populate all computed fields
	r.readGithubUserGpgKey(ctx, &data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubUserGpgKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubUserGpgKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubUserGpgKey(ctx, &data, &resp.Diagnostics, true)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubUserGpgKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This resource only supports create and delete operations
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"GitHub user GPG keys do not support updates. "+
			"All attributes are marked as RequiresReplace, so updates should result in replacement.",
	)
}

func (r *githubUserGpgKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubUserGpgKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid GPG Key ID",
			fmt.Sprintf("Unable to parse GPG key ID '%s' as integer: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, err = r.client.V3Client().Users.DeleteGPGKey(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete GitHub GPG Key",
			"An unexpected error occurred when deleting the GitHub GPG key. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub user GPG key", map[string]interface{}{
		"id": data.ID.ValueString(),
	})
}

func (r *githubUserGpgKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data := &githubUserGpgKeyResourceModel{
		ID: types.StringValue(req.ID),
	}

	r.readGithubUserGpgKey(ctx, data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *githubUserGpgKeyResource) readGithubUserGpgKey(ctx context.Context, data *githubUserGpgKeyResourceModel, diags *diag.Diagnostics, useEtag bool) {
	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		diags.AddError(
			"Invalid GPG Key ID",
			fmt.Sprintf("Unable to parse GPG key ID '%s' as integer: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	// Set up context with ETag if this is not a new resource and useEtag is true
	reqCtx := ctx
	if useEtag && !data.Etag.IsNull() && data.Etag.ValueString() != "" {
		reqCtx = context.WithValue(ctx, CtxEtag, data.Etag.ValueString())
	}
	reqCtx = context.WithValue(reqCtx, CtxId, data.ID.ValueString())

	key, resp, err := r.client.V3Client().Users.GetGPGKey(reqCtx, id)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				// Resource hasn't changed, keep current state
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "GitHub user GPG key not found, removing from state", map[string]interface{}{
					"id": data.ID.ValueString(),
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read GitHub GPG Key",
			"An unexpected error occurred when reading the GitHub GPG key. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	data.Etag = types.StringValue(resp.Header.Get("ETag"))
	data.KeyID = types.StringValue(key.GetKeyID())

	tflog.Debug(ctx, "successfully read GitHub user GPG key", map[string]interface{}{
		"id":     data.ID.ValueString(),
		"key_id": data.KeyID.ValueString(),
	})
}
