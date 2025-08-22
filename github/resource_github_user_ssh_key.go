package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	_ resource.Resource                = &githubUserSshKeyResource{}
	_ resource.ResourceWithConfigure   = &githubUserSshKeyResource{}
	_ resource.ResourceWithImportState = &githubUserSshKeyResource{}
)

type githubUserSshKeyResource struct {
	client *Owner
}

type githubUserSshKeyResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Title types.String `tfsdk:"title"`
	Key   types.String `tfsdk:"key"`
	URL   types.String `tfsdk:"url"`
	Etag  types.String `tfsdk:"etag"`
}

func NewGithubUserSshKeyResource() resource.Resource {
	return &githubUserSshKeyResource{}
}

func (r *githubUserSshKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_ssh_key"
}

func (r *githubUserSshKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub user's SSH key resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the SSH key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Description: "A descriptive name for the new key.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Description: "The public SSH key to add to your GitHub account.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				Description: "The URL of the SSH key.",
				Computed:    true,
			},
			"etag": schema.StringAttribute{
				Description: "The ETag of the SSH key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubUserSshKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubUserSshKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubUserSshKeyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	title := data.Title.ValueString()
	key := data.Key.ValueString()

	userKey, _, err := r.client.V3Client().Users.CreateKey(ctx, &github.Key{
		Title: github.Ptr(title),
		Key:   github.Ptr(key),
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create GitHub SSH Key",
			"An unexpected error occurred when creating the GitHub SSH key. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(*userKey.ID, 10))

	tflog.Debug(ctx, "created GitHub user SSH key", map[string]any{
		"id":    data.ID.ValueString(),
		"title": data.Title.ValueString(),
	})

	// Read the created resource to populate all computed fields
	r.readGithubUserSshKey(ctx, &data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubUserSshKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubUserSshKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubUserSshKey(ctx, &data, &resp.Diagnostics, true)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubUserSshKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This resource only supports create and delete operations
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"GitHub user SSH keys do not support updates. "+
			"All attributes are marked as RequiresReplace, so updates should result in replacement.",
	)
}

func (r *githubUserSshKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubUserSshKeyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid SSH Key ID",
			fmt.Sprintf("Unable to parse SSH key ID '%s' as integer: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, err = r.client.V3Client().Users.DeleteKey(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete GitHub SSH Key",
			"An unexpected error occurred when deleting the GitHub SSH key. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub user SSH key", map[string]any{
		"id": data.ID.ValueString(),
	})
}

func (r *githubUserSshKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data := &githubUserSshKeyResourceModel{
		ID: types.StringValue(req.ID),
	}

	r.readGithubUserSshKey(ctx, data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *githubUserSshKeyResource) readGithubUserSshKey(ctx context.Context, data *githubUserSshKeyResourceModel, diags *diag.Diagnostics, useEtag bool) {
	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		diags.AddError(
			"Invalid SSH Key ID",
			fmt.Sprintf("Unable to parse SSH key ID '%s' as integer: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	// Set up context with ETag if this is not a new resource and useEtag is true
	reqCtx := ctx
	if useEtag && !data.Etag.IsNull() && data.Etag.ValueString() != "" {
		reqCtx = context.WithValue(ctx, CtxEtag, data.Etag.ValueString())
	}
	reqCtx = context.WithValue(reqCtx, CtxId, data.ID.ValueString())

	key, resp, err := r.client.V3Client().Users.GetKey(reqCtx, id)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				// Resource hasn't changed, keep current state
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "GitHub user SSH key not found, removing from state", map[string]any{
					"id": data.ID.ValueString(),
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read GitHub SSH Key",
			"An unexpected error occurred when reading the GitHub SSH key. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	data.Etag = types.StringValue(resp.Header.Get("ETag"))
	data.Title = types.StringValue(key.GetTitle())

	// Trim whitespace from the key similar to the SDKv2 diff suppress function
	keyValue := strings.TrimSpace(key.GetKey())
	data.Key = types.StringValue(keyValue)

	data.URL = types.StringValue(key.GetURL())

	tflog.Debug(ctx, "successfully read GitHub user SSH key", map[string]any{
		"id":    data.ID.ValueString(),
		"title": data.Title.ValueString(),
		"url":   data.URL.ValueString(),
	})
}
