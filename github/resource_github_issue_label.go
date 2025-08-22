package github

import (
	"context"
	"fmt"
	"net/http"
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
	_ resource.Resource                = &githubIssueLabelResource{}
	_ resource.ResourceWithConfigure   = &githubIssueLabelResource{}
	_ resource.ResourceWithImportState = &githubIssueLabelResource{}
)

type githubIssueLabelResource struct {
	client *Owner
}

type githubIssueLabelResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Repository  types.String `tfsdk:"repository"`
	Name        types.String `tfsdk:"name"`
	Color       types.String `tfsdk:"color"`
	Description types.String `tfsdk:"description"`
	URL         types.String `tfsdk:"url"`
	Etag        types.String `tfsdk:"etag"`
}

func NewGithubIssueLabelResource() resource.Resource {
	return &githubIssueLabelResource{}
}

func (r *githubIssueLabelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_issue_label"
}

func (r *githubIssueLabelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub issue label resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the issue label (repository:name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The GitHub repository.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the label.",
				Required:    true,
			},
			"color": schema.StringAttribute{
				Description: "A 6 character hex code, without the leading '#', identifying the color of the label.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "A short description of the label.",
				Optional:    true,
			},
			"url": schema.StringAttribute{
				Description: "The URL to the issue label.",
				Computed:    true,
			},
			"etag": schema.StringAttribute{
				Description: "The ETag of the issue label.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubIssueLabelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubIssueLabelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubIssueLabelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.createOrUpdateGithubIssueLabel(ctx, &data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubIssueLabelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubIssueLabelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubIssueLabel(ctx, &data, &resp.Diagnostics, true)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubIssueLabelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubIssueLabelResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.createOrUpdateGithubIssueLabel(ctx, &data, &resp.Diagnostics, true)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubIssueLabelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubIssueLabelResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgName := r.client.Name()
	repoName := data.Repository.ValueString()
	name := data.Name.ValueString()

	tflog.Debug(ctx, "deleting GitHub issue label", map[string]any{
		"repository": repoName,
		"name":       name,
		"owner":      orgName,
	})

	_, err := r.client.V3Client().Issues.DeleteLabel(ctx, orgName, repoName, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete GitHub Issue Label",
			"An unexpected error occurred when deleting the GitHub issue label. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub issue label", map[string]any{
		"repository": repoName,
		"name":       name,
	})
}

func (r *githubIssueLabelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "repository:name"
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: repository:name. Got: %q", req.ID),
		)
		return
	}

	repoName := parts[0]
	name := parts[1]

	data := &githubIssueLabelResourceModel{
		ID:         types.StringValue(req.ID),
		Repository: types.StringValue(repoName),
		Name:       types.StringValue(name),
	}

	r.readGithubIssueLabel(ctx, data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// createOrUpdateGithubIssueLabel idempotently creates or updates an issue label.
// Issue labels are keyed off of their "name", so pre-existing issue labels
// result in a 422 HTTP error if they exist outside of Terraform. Normally this
// would not be an issue, except new repositories are created with a "default"
// set of labels, and those labels easily conflict with custom ones.
//
// This function will first check if the label exists, and then issue an update,
// otherwise it will create. This is also advantageous in that we get to use the
// same function for both create and update operations.
func (r *githubIssueLabelResource) createOrUpdateGithubIssueLabel(ctx context.Context, data *githubIssueLabelResourceModel, diags *diag.Diagnostics, isUpdate bool) {
	orgName := r.client.Name()
	repoName := data.Repository.ValueString()
	name := data.Name.ValueString()
	color := data.Color.ValueString()

	label := &github.Label{
		Name:  github.Ptr(name),
		Color: github.Ptr(color),
	}

	// Set up context
	if isUpdate {
		ctx = context.WithValue(ctx, CtxId, data.ID.ValueString())
	}

	// Pull out the original name. If we already have a resource, this is the
	// parsed ID. If not, it's the value given to the resource.
	var originalName string
	if data.ID.IsNull() || data.ID.ValueString() == "" {
		originalName = name
	} else {
		// Parse the two-part ID (repository:name)
		parts := strings.SplitN(data.ID.ValueString(), ":", 2)
		if len(parts) != 2 {
			diags.AddError(
				"Invalid Resource ID",
				fmt.Sprintf("Expected ID with format: repository:name. Got: %q", data.ID.ValueString()),
			)
			return
		}
		originalName = parts[1]
	}

	tflog.Debug(ctx, "checking if GitHub issue label exists", map[string]any{
		"repository":    repoName,
		"original_name": originalName,
		"owner":         orgName,
	})

	existing, resp, err := r.client.V3Client().Issues.GetLabel(ctx, orgName, repoName, originalName)
	if err != nil && resp.StatusCode != http.StatusNotFound {
		diags.AddError(
			"Unable to Check GitHub Issue Label",
			"An unexpected error occurred when checking for existing GitHub issue label. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	if existing != nil {
		// Label exists, update it
		if !data.Description.IsNull() {
			label.Description = github.Ptr(data.Description.ValueString())
		}

		tflog.Debug(ctx, "updating existing GitHub issue label", map[string]any{
			"repository":    repoName,
			"original_name": originalName,
			"new_name":      name,
			"owner":         orgName,
		})

		_, _, err := r.client.V3Client().Issues.EditLabel(ctx, orgName, repoName, originalName, label)
		if err != nil {
			diags.AddError(
				"Unable to Update GitHub Issue Label",
				"An unexpected error occurred when updating the GitHub issue label. "+
					"GitHub Client Error: "+err.Error(),
			)
			return
		}
	} else {
		// Label doesn't exist, create it
		if !data.Description.IsNull() {
			label.Description = github.Ptr(data.Description.ValueString())
		}

		tflog.Debug(ctx, "creating new GitHub issue label", map[string]any{
			"repository": repoName,
			"name":       name,
			"owner":      orgName,
		})

		_, _, err := r.client.V3Client().Issues.CreateLabel(ctx, orgName, repoName, label)
		if err != nil {
			diags.AddError(
				"Unable to Create GitHub Issue Label",
				"An unexpected error occurred when creating the GitHub issue label. "+
					"GitHub Client Error: "+err.Error(),
			)
			return
		}
	}

	// Set the ID
	data.ID = types.StringValue(fmt.Sprintf("%s:%s", repoName, name))

	tflog.Debug(ctx, "GitHub issue label operation completed", map[string]any{
		"id":   data.ID.ValueString(),
		"name": name,
	})

	// Read the created/updated resource to populate all computed fields
	r.readGithubIssueLabel(ctx, data, diags, false)
}

func (r *githubIssueLabelResource) readGithubIssueLabel(ctx context.Context, data *githubIssueLabelResourceModel, diags *diag.Diagnostics, useEtag bool) {
	orgName := r.client.Name()
	repoName := data.Repository.ValueString()
	name := data.Name.ValueString()

	// Set up context with ETag if this is not a new resource and useEtag is true
	reqCtx := ctx
	if useEtag && !data.Etag.IsNull() && data.Etag.ValueString() != "" {
		reqCtx = context.WithValue(ctx, CtxEtag, data.Etag.ValueString())
	}
	reqCtx = context.WithValue(reqCtx, CtxId, data.ID.ValueString())

	tflog.Debug(ctx, "reading GitHub issue label", map[string]any{
		"repository": repoName,
		"name":       name,
		"owner":      orgName,
	})

	githubLabel, resp, err := r.client.V3Client().Issues.GetLabel(reqCtx, orgName, repoName, name)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				// Resource hasn't changed, keep current state
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "GitHub issue label not found, removing from state", map[string]any{
					"id":         data.ID.ValueString(),
					"repository": repoName,
					"name":       name,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read GitHub Issue Label",
			"An unexpected error occurred when reading the GitHub issue label. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	// Set computed and read values
	data.Etag = types.StringValue(resp.Header.Get("ETag"))
	data.Repository = types.StringValue(repoName)
	data.Name = types.StringValue(name)
	data.Color = types.StringValue(githubLabel.GetColor())
	data.URL = types.StringValue(githubLabel.GetURL())

	if githubLabel.Description != nil {
		data.Description = types.StringValue(githubLabel.GetDescription())
	} else {
		data.Description = types.StringNull()
	}

	tflog.Debug(ctx, "successfully read GitHub issue label", map[string]any{
		"id":         data.ID.ValueString(),
		"name":       data.Name.ValueString(),
		"repository": repoName,
		"color":      data.Color.ValueString(),
	})
}
