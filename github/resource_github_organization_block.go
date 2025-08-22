package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubOrganizationBlockResource{}
	_ resource.ResourceWithConfigure   = &githubOrganizationBlockResource{}
	_ resource.ResourceWithImportState = &githubOrganizationBlockResource{}
)

type githubOrganizationBlockResource struct {
	client *Owner
}

type githubOrganizationBlockResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Username types.String `tfsdk:"username"`
	Etag     types.String `tfsdk:"etag"`
}

func NewGithubOrganizationBlockResource() resource.Resource {
	return &githubOrganizationBlockResource{}
}

func (r *githubOrganizationBlockResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_block"
}

func (r *githubOrganizationBlockResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub organization block resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The username of the blocked user.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Description: "The name of the user to block.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"etag": schema.StringAttribute{
				Description: "The ETag of the blocked user.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubOrganizationBlockResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubOrganizationBlockResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubOrganizationBlockResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if we're in an organization context
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used in the context of an organization, %q is a user", r.client.Name()),
		)
		return
	}

	client := r.client.V3Client()
	orgName := r.client.Name()
	username := data.Username.ValueString()

	_, err := client.Organizations.BlockUser(ctx, orgName, username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Block User",
			"An unexpected error occurred when blocking the user. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "blocked user from organization", map[string]any{
		"organization": orgName,
		"username":     username,
	})

	// Set the ID to the username
	data.ID = types.StringValue(username)

	// Read the created resource to populate all computed fields
	r.readGithubOrganizationBlock(ctx, &data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubOrganizationBlockResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubOrganizationBlockResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubOrganizationBlock(ctx, &data, &resp.Diagnostics, true)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubOrganizationBlockResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This resource doesn't support update operations
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"The github_organization_block resource does not support updates. Please destroy and recreate the resource.",
	)
}

func (r *githubOrganizationBlockResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubOrganizationBlockResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	orgName := r.client.Name()
	username := data.Username.ValueString()

	requestCtx := context.WithValue(ctx, CtxId, username)

	_, err := client.Organizations.UnblockUser(requestCtx, orgName, username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Unblock User",
			"An unexpected error occurred when unblocking the user. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "unblocked user from organization", map[string]any{
		"organization": orgName,
		"username":     username,
	})
}

func (r *githubOrganizationBlockResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Set the ID and username
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), types.StringValue(req.ID))...)
}

func (r *githubOrganizationBlockResource) readGithubOrganizationBlock(ctx context.Context, data *githubOrganizationBlockResourceModel, diags *diag.Diagnostics, useEtag bool) {
	client := r.client.V3Client()
	orgName := r.client.Name()
	username := data.Username.ValueString()

	// Set up context with ETag if this is not a new resource and useEtag is true
	requestCtx := ctx
	if useEtag && !data.Etag.IsNull() && data.Etag.ValueString() != "" {
		requestCtx = context.WithValue(ctx, CtxEtag, data.Etag.ValueString())
	}
	requestCtx = context.WithValue(requestCtx, CtxId, username)

	blocked, resp, err := client.Organizations.IsBlocked(requestCtx, orgName, username)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				// Resource hasn't changed, no need to update
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "organization block not found, removing from state", map[string]any{
					"organization": orgName,
					"username":     username,
				})
				// Resource no longer exists, mark for removal
				data.Username = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Organization Block",
			"An unexpected error occurred when reading the organization block. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	if !blocked {
		tflog.Info(ctx, "user is not blocked, removing from state", map[string]any{
			"organization": orgName,
			"username":     username,
		})
		// User is not blocked, mark for removal
		data.Username = types.StringNull()
		return
	}

	// Update the model with current values
	data.ID = types.StringValue(username)
	data.Username = types.StringValue(username)
	if resp != nil && resp.Header.Get("ETag") != "" {
		data.Etag = types.StringValue(resp.Header.Get("ETag"))
	} else {
		data.Etag = types.StringNull()
	}

	tflog.Debug(ctx, "read organization block", map[string]any{
		"organization": orgName,
		"username":     username,
		"etag":         data.Etag.ValueString(),
	})
}
