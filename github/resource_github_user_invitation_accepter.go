package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	_ resource.Resource              = &githubUserInvitationAccepterResource{}
	_ resource.ResourceWithConfigure = &githubUserInvitationAccepterResource{}
)

type githubUserInvitationAccepterResource struct {
	client *Owner
}

type githubUserInvitationAccepterResourceModel struct {
	ID           types.String `tfsdk:"id"`
	InvitationID types.String `tfsdk:"invitation_id"`
	AllowEmptyID types.Bool   `tfsdk:"allow_empty_id"`
}

func NewGithubUserInvitationAccepterResource() resource.Resource {
	return &githubUserInvitationAccepterResource{}
}

func (r *githubUserInvitationAccepterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_invitation_accepter"
}

func (r *githubUserInvitationAccepterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a resource to manage GitHub repository collaborator invitations.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the invitation accepter resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"invitation_id": schema.StringAttribute{
				Description: "ID of the invitation to accept. Must be set when 'allow_empty_id' is 'false'.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"allow_empty_id": schema.BoolAttribute{
				Description: "Allow the ID to be unset. This will result in the resource being skipped when the ID is not set instead of returning an error.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *githubUserInvitationAccepterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubUserInvitationAccepterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubUserInvitationAccepterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	invitationIDString := data.InvitationID.ValueString()
	allowEmptyID := data.AllowEmptyID.ValueBool()

	if invitationIDString == "" {
		if allowEmptyID {
			// We're setting a random UUID as resource ID since every resource needs an ID
			// and we can't destroy the resource while we create it.
			data.ID = types.StringValue(uuid.NewString())
			tflog.Debug(ctx, "skipping invitation acceptance due to empty invitation_id with allow_empty_id=true", map[string]any{
				"id": data.ID.ValueString(),
			})
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		} else {
			resp.Diagnostics.AddError(
				"Missing Required Attribute",
				"invitation_id is not set and allow_empty_id is false",
			)
			return
		}
	}

	invitationID, err := strconv.Atoi(invitationIDString)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Invitation ID",
			fmt.Sprintf("Failed to parse invitation ID '%s': %s", invitationIDString, err.Error()),
		)
		return
	}

	_, err = r.client.V3Client().Users.AcceptInvitation(ctx, int64(invitationID))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Accept GitHub Invitation",
			"An unexpected error occurred when accepting the GitHub invitation. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	data.ID = types.StringValue(invitationIDString)

	tflog.Debug(ctx, "accepted GitHub user invitation", map[string]any{
		"id":            data.ID.ValueString(),
		"invitation_id": invitationIDString,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubUserInvitationAccepterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubUserInvitationAccepterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Nothing to read as invitation is removed after it's accepted
	// Just maintain the current state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubUserInvitationAccepterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This resource only supports create operations
	// All attributes are marked as RequiresReplace, so updates should result in replacement
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"GitHub user invitation accepter does not support updates. "+
			"All attributes are marked as RequiresReplace, so updates should result in replacement.",
	)
}

func (r *githubUserInvitationAccepterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubUserInvitationAccepterResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Nothing to remove as invitation is removed after it's accepted
	tflog.Debug(ctx, "removing GitHub user invitation accepter from state", map[string]any{
		"id": data.ID.ValueString(),
	})
}

func (r *githubUserInvitationAccepterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The import ID can be either:
	// 1. An invitation ID (numeric)
	// 2. A UUID (for resources created with allow_empty_id=true)

	importID := req.ID

	// Try to parse as invitation ID (numeric)
	if invitationID, err := strconv.Atoi(importID); err == nil {
		// It's a numeric invitation ID
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(importID))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("invitation_id"), types.StringValue(importID))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_empty_id"), types.BoolValue(false))...)

		tflog.Debug(ctx, "imported GitHub user invitation accepter with invitation ID", map[string]any{
			"id":            importID,
			"invitation_id": invitationID,
		})
	} else {
		// It's likely a UUID from allow_empty_id=true case
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(importID))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("invitation_id"), types.StringValue(""))...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("allow_empty_id"), types.BoolValue(true))...)

		tflog.Debug(ctx, "imported GitHub user invitation accepter with UUID", map[string]any{
			"id":             importID,
			"allow_empty_id": true,
		})
	}
}
