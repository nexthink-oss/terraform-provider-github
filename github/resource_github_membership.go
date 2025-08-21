package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubMembershipResource{}
	_ resource.ResourceWithConfigure   = &githubMembershipResource{}
	_ resource.ResourceWithImportState = &githubMembershipResource{}
)

func NewGithubMembershipResource() resource.Resource {
	return &githubMembershipResource{}
}

type githubMembershipResource struct {
	client *Owner
}

type githubMembershipResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Username           types.String `tfsdk:"username"`
	Role               types.String `tfsdk:"role"`
	Etag               types.String `tfsdk:"etag"`
	DowngradeOnDestroy types.Bool   `tfsdk:"downgrade_on_destroy"`
}

func (r *githubMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_membership"
}

func (r *githubMembershipResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub membership resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the membership.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Description: "The user to add to the organization.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role": schema.StringAttribute{
				Description: "The role of the user within the organization. Must be one of 'member' or 'admin'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("member"),
				Validators: []validator.String{
					stringvalidator.OneOf("member", "admin"),
				},
			},
			"etag": schema.StringAttribute{
				Description: "The etag for the membership.",
				Computed:    true,
			},
			"downgrade_on_destroy": schema.BoolAttribute{
				Description: "Instead of removing the member from the org, you can choose to downgrade their membership to 'member' when this resource is destroyed. This is useful when wanting to downgrade admins while keeping them in the organization",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *githubMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *githubMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubMembershipResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used in the context of an organization, %q is a user", r.client.Name()),
		)
		return
	}

	orgName := r.client.Name()
	username := plan.Username.ValueString()
	roleName := plan.Role.ValueString()

	tflog.Debug(ctx, "Creating GitHub membership", map[string]any{
		"org":      orgName,
		"username": username,
		"role":     roleName,
	})

	_, _, err := r.client.V3Client().Organizations.EditOrgMembership(ctx,
		username,
		orgName,
		&github.Membership{
			Role: github.Ptr(roleName),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating GitHub Membership",
			"Could not create membership, unexpected error: "+err.Error(),
		)
		return
	}

	// Set the ID using buildTwoPartID pattern
	plan.ID = types.StringValue(buildTwoPartID(orgName, username))

	// Read the created membership to populate computed attributes
	r.readMembership(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubMembershipResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readMembership(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubMembershipResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used in the context of an organization, %q is a user", r.client.Name()),
		)
		return
	}

	orgName := r.client.Name()
	username := plan.Username.ValueString()
	roleName := plan.Role.ValueString()

	tflog.Debug(ctx, "Updating GitHub membership", map[string]any{
		"org":      orgName,
		"username": username,
		"role":     roleName,
	})

	_, _, err := r.client.V3Client().Organizations.EditOrgMembership(ctx,
		username,
		orgName,
		&github.Membership{
			Role: github.Ptr(roleName),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating GitHub Membership",
			"Could not update membership, unexpected error: "+err.Error(),
		)
		return
	}

	// Read the updated membership to populate computed attributes
	r.readMembership(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubMembershipResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used in the context of an organization, %q is a user", r.client.Name()),
		)
		return
	}

	orgName := r.client.Name()
	username := state.Username.ValueString()
	downgradeOnDestroy := state.DowngradeOnDestroy.ValueBool()
	downgradeTo := "member"

	if downgradeOnDestroy {
		tflog.Info(ctx, "Downgrading membership instead of removing", map[string]any{
			"org":         orgName,
			"username":    username,
			"downgradeTo": downgradeTo,
		})

		// Check to make sure this member still has access to the organization before downgrading.
		// If we don't do this, the member would just be re-added to the organization.
		membership, _, err := r.client.V3Client().Organizations.GetOrgMembership(ctx, username, orgName)
		if err != nil {
			if ghErr, ok := err.(*github.ErrorResponse); ok {
				if ghErr.Response.StatusCode == http.StatusNotFound {
					tflog.Info(ctx, "Not downgrading membership because user is not a member of the org anymore", map[string]any{
						"org":      orgName,
						"username": username,
					})
					return
				}
			}

			resp.Diagnostics.AddError(
				"Error Reading GitHub Membership",
				"Could not read membership during delete, unexpected error: "+err.Error(),
			)
			return
		}

		if membership.GetRole() == downgradeTo {
			tflog.Info(ctx, "Not downgrading membership because user is already at target role", map[string]any{
				"org":      orgName,
				"username": username,
				"role":     downgradeTo,
			})
			return
		}

		_, _, err = r.client.V3Client().Organizations.EditOrgMembership(ctx, username, orgName, &github.Membership{
			Role: github.Ptr(downgradeTo),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Downgrading GitHub Membership",
				"Could not downgrade membership, unexpected error: "+err.Error(),
			)
			return
		}
	} else {
		tflog.Info(ctx, "Removing membership from organization", map[string]any{
			"org":      orgName,
			"username": username,
		})

		_, err := r.client.V3Client().Organizations.RemoveOrgMembership(ctx, username, orgName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Deleting GitHub Membership",
				"Could not delete membership, unexpected error: "+err.Error(),
			)
			return
		}
	}
}

func (r *githubMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID should be in format "org:username"
	_, username, err := parseTwoPartID(req.ID, "organization", "username")
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Importing GitHub Membership",
			"Could not parse import ID, expected format: organization:username. Error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), username)...)

	// Trigger a read to populate the rest of the attributes
	resp.State.SetAttribute(ctx, path.Root("role"), types.StringNull())
	resp.State.SetAttribute(ctx, path.Root("etag"), types.StringNull())
	resp.State.SetAttribute(ctx, path.Root("downgrade_on_destroy"), types.BoolValue(false))
}

// readMembership is a helper function to read membership data from GitHub API
func (r *githubMembershipResource) readMembership(ctx context.Context, model *githubMembershipResourceModel, diags *diag.Diagnostics) {
	if !r.client.IsOrganization {
		diags.AddError(
			"Organization Required",
			fmt.Sprintf("This resource can only be used in the context of an organization, %q is a user", r.client.Name()),
		)
		return
	}

	orgName := r.client.Name()
	id := model.ID.ValueString()

	_, username, err := parseTwoPartID(id, "organization", "username")
	if err != nil {
		diags.AddError(
			"Error Parsing Membership ID",
			"Could not parse membership ID, unexpected error: "+err.Error(),
		)
		return
	}

	// Use etag if available for conditional requests
	reqCtx := ctx
	if !model.Etag.IsNull() && !model.Etag.IsUnknown() {
		// Add ETag to context for conditional requests
		reqCtx = context.WithValue(ctx, CtxEtag, model.Etag.ValueString())
	}

	membership, resp, err := r.client.V3Client().Organizations.GetOrgMembership(reqCtx, username, orgName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				// No changes, return early
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "Removing membership from state because it no longer exists in GitHub", map[string]any{
					"id": id,
				})
				// Set ID to null to indicate resource should be removed from state
				model.ID = types.StringNull()
				return
			}
		}

		diags.AddError(
			"Error Reading GitHub Membership",
			"Could not read membership, unexpected error: "+err.Error(),
		)
		return
	}

	// Update model with API response data
	model.Username = types.StringValue(username)
	model.Role = types.StringValue(membership.GetRole())

	if resp != nil && resp.Header != nil {
		if etag := resp.Header.Get("ETag"); etag != "" {
			model.Etag = types.StringValue(etag)
		}
	}
}
