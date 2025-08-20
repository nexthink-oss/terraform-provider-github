package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ resource.Resource                = &githubIssueLabelsResource{}
	_ resource.ResourceWithConfigure   = &githubIssueLabelsResource{}
	_ resource.ResourceWithImportState = &githubIssueLabelsResource{}
)

type githubIssueLabelsResource struct {
	client *Owner
}

type githubIssueLabelsResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Repository types.String `tfsdk:"repository"`
	Label      types.Set    `tfsdk:"label"`
}

type githubIssueLabelModel struct {
	Name        types.String `tfsdk:"name"`
	Color       types.String `tfsdk:"color"`
	Description types.String `tfsdk:"description"`
	URL         types.String `tfsdk:"url"`
}

// labelObjectType defines the object type for individual labels
var labelObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"name":        types.StringType,
		"color":       types.StringType,
		"description": types.StringType,
		"url":         types.StringType,
	},
}

func NewGithubIssueLabelsResource() resource.Resource {
	return &githubIssueLabelsResource{}
}

func (r *githubIssueLabelsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_issue_labels"
}

func (r *githubIssueLabelsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides GitHub issue labels resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the issue labels resource (repository name).",
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
			"label": schema.SetAttribute{
				Description: "List of labels",
				Optional:    true,
				ElementType: labelObjectType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{},
	}
}

func (r *githubIssueLabelsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubIssueLabelsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubIssueLabelsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.createOrUpdateGithubIssueLabels(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubIssueLabelsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubIssueLabelsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubIssueLabels(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubIssueLabelsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubIssueLabelsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.createOrUpdateGithubIssueLabels(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubIssueLabelsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubIssueLabelsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgName := r.client.Name()
	repository := data.Repository.ValueString()

	tflog.Debug(ctx, "deleting all GitHub issue labels", map[string]interface{}{
		"repository": repository,
		"owner":      orgName,
	})

	reqCtx := context.WithValue(ctx, CtxId, repository)

	// Get current labels from the state to delete them
	if !data.Label.IsNull() && !data.Label.IsUnknown() {
		var labels []githubIssueLabelModel
		resp.Diagnostics.Append(data.Label.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, label := range labels {
			name := label.Name.ValueString()

			tflog.Debug(ctx, "deleting GitHub issue label", map[string]interface{}{
				"repository": repository,
				"name":       name,
				"owner":      orgName,
			})

			_, err := r.client.V3Client().Issues.DeleteLabel(reqCtx, orgName, repository, name)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Delete GitHub Issue Label",
					fmt.Sprintf("An unexpected error occurred when deleting GitHub issue label %s. GitHub Client Error: %s", name, err.Error()),
				)
				return
			}
		}
	}

	tflog.Debug(ctx, "deleted all GitHub issue labels", map[string]interface{}{
		"repository": repository,
	})
}

func (r *githubIssueLabelsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	repository := req.ID

	data := &githubIssueLabelsResourceModel{
		ID:         types.StringValue(repository),
		Repository: types.StringValue(repository),
	}

	r.readGithubIssueLabels(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *githubIssueLabelsResource) readGithubIssueLabels(ctx context.Context, data *githubIssueLabelsResourceModel, diags *diag.Diagnostics) {
	orgName := r.client.Name()
	repository := data.Repository.ValueString()

	reqCtx := context.WithValue(ctx, CtxId, repository)

	tflog.Debug(ctx, "reading GitHub issue labels", map[string]interface{}{
		"repository": repository,
		"owner":      orgName,
	})

	options := &github.ListOptions{
		PerPage: 100, // maxPerPage constant from github package
	}

	var labels []githubIssueLabelModel

	for {
		githubLabels, resp, err := r.client.V3Client().Issues.ListLabels(reqCtx, orgName, repository, options)
		if err != nil {
			if ghErr, ok := err.(*github.ErrorResponse); ok {
				if ghErr.Response.StatusCode == http.StatusNotFound {
					tflog.Info(ctx, "GitHub repository not found, removing from state", map[string]interface{}{
						"id":         data.ID.ValueString(),
						"repository": repository,
					})
					data.ID = types.StringNull()
					return
				}
			}
			diags.AddError(
				"Unable to Read GitHub Issue Labels",
				fmt.Sprintf("An unexpected error occurred when reading GitHub issue labels for repository %s. GitHub Client Error: %s", repository, err.Error()),
			)
			return
		}

		for _, l := range githubLabels {
			labelModel := githubIssueLabelModel{
				Name:  types.StringValue(l.GetName()),
				Color: types.StringValue(l.GetColor()),
				URL:   types.StringValue(l.GetURL()),
			}

			if l.Description != nil {
				labelModel.Description = types.StringValue(l.GetDescription())
			} else {
				labelModel.Description = types.StringNull()
			}

			labels = append(labels, labelModel)
		}

		if resp.NextPage == 0 {
			break
		}
		options.Page = resp.NextPage
	}

	tflog.Debug(ctx, "found GitHub issue labels", map[string]interface{}{
		"repository":  repository,
		"label_count": len(labels),
	})

	// Convert labels to Set
	labelValues := make([]attr.Value, len(labels))
	for i, label := range labels {
		labelValues[i] = types.ObjectValueMust(labelObjectType.AttrTypes, map[string]attr.Value{
			"name":        label.Name,
			"color":       label.Color,
			"description": label.Description,
			"url":         label.URL,
		})
	}

	if len(labelValues) > 0 {
		data.Label = types.SetValueMust(labelObjectType, labelValues)
	} else {
		data.Label = types.SetValueMust(labelObjectType, []attr.Value{})
	}

	data.ID = types.StringValue(repository)
}

func (r *githubIssueLabelsResource) createOrUpdateGithubIssueLabels(ctx context.Context, data *githubIssueLabelsResourceModel, diags *diag.Diagnostics) {
	orgName := r.client.Name()
	repository := data.Repository.ValueString()

	reqCtx := context.WithValue(ctx, CtxId, repository)

	tflog.Debug(ctx, "updating GitHub issue labels", map[string]interface{}{
		"repository": repository,
		"owner":      orgName,
	})

	// Read current labels from GitHub
	currentData := &githubIssueLabelsResourceModel{
		Repository: data.Repository,
		ID:         data.ID,
	}
	r.readGithubIssueLabels(ctx, currentData, diags)
	if diags.HasError() {
		return
	}

	// Parse old and new labels
	oldLabels := make(map[string]githubIssueLabelModel)
	newLabels := make(map[string]githubIssueLabelModel)

	// Get current labels from GitHub
	if !currentData.Label.IsNull() && !currentData.Label.IsUnknown() {
		var currentLabels []githubIssueLabelModel
		diags.Append(currentData.Label.ElementsAs(ctx, &currentLabels, false)...)
		if diags.HasError() {
			return
		}

		for _, label := range currentLabels {
			name := strings.ToLower(label.Name.ValueString())
			oldLabels[name] = label
		}
	}

	// Get desired labels from configuration
	if !data.Label.IsNull() && !data.Label.IsUnknown() {
		var desiredLabels []githubIssueLabelModel
		diags.Append(data.Label.ElementsAs(ctx, &desiredLabels, false)...)
		if diags.HasError() {
			return
		}

		for _, label := range desiredLabels {
			name := strings.ToLower(label.Name.ValueString())
			newLabels[name] = label
		}
	}

	var resultLabels []githubIssueLabelModel

	// Create new labels
	for name, newLabel := range newLabels {
		if _, exists := oldLabels[name]; !exists {
			tflog.Debug(ctx, "creating GitHub issue label", map[string]interface{}{
				"repository": repository,
				"name":       newLabel.Name.ValueString(),
				"owner":      orgName,
			})

			label := &github.Label{
				Name:  github.Ptr(newLabel.Name.ValueString()),
				Color: github.Ptr(newLabel.Color.ValueString()),
			}

			if !newLabel.Description.IsNull() {
				label.Description = github.Ptr(newLabel.Description.ValueString())
			}

			createdLabel, _, err := r.client.V3Client().Issues.CreateLabel(reqCtx, orgName, repository, label)
			if err != nil {
				diags.AddError(
					"Unable to Create GitHub Issue Label",
					fmt.Sprintf("An unexpected error occurred when creating GitHub issue label %s. GitHub Client Error: %s", newLabel.Name.ValueString(), err.Error()),
				)
				return
			}

			labelModel := githubIssueLabelModel{
				Name:  types.StringValue(createdLabel.GetName()),
				Color: types.StringValue(createdLabel.GetColor()),
				URL:   types.StringValue(createdLabel.GetURL()),
			}

			if createdLabel.Description != nil {
				labelModel.Description = types.StringValue(createdLabel.GetDescription())
			} else {
				labelModel.Description = types.StringNull()
			}

			resultLabels = append(resultLabels, labelModel)
		}
	}

	// Delete removed labels
	for name, oldLabel := range oldLabels {
		if _, exists := newLabels[name]; !exists {
			tflog.Debug(ctx, "deleting GitHub issue label", map[string]interface{}{
				"repository": repository,
				"name":       oldLabel.Name.ValueString(),
				"owner":      orgName,
			})

			_, err := r.client.V3Client().Issues.DeleteLabel(reqCtx, orgName, repository, oldLabel.Name.ValueString())
			if err != nil {
				diags.AddError(
					"Unable to Delete GitHub Issue Label",
					fmt.Sprintf("An unexpected error occurred when deleting GitHub issue label %s. GitHub Client Error: %s", oldLabel.Name.ValueString(), err.Error()),
				)
				return
			}
		}
	}

	// Update existing labels
	for name, newLabel := range newLabels {
		if oldLabel, exists := oldLabels[name]; exists {
			needsUpdate := oldLabel.Name.ValueString() != newLabel.Name.ValueString() ||
				oldLabel.Color.ValueString() != newLabel.Color.ValueString() ||
				!oldLabel.Description.Equal(newLabel.Description)

			if needsUpdate {
				tflog.Debug(ctx, "updating GitHub issue label", map[string]interface{}{
					"repository": repository,
					"old_name":   oldLabel.Name.ValueString(),
					"new_name":   newLabel.Name.ValueString(),
					"owner":      orgName,
				})

				label := &github.Label{
					Name:  github.Ptr(newLabel.Name.ValueString()),
					Color: github.Ptr(newLabel.Color.ValueString()),
				}

				if !newLabel.Description.IsNull() {
					label.Description = github.Ptr(newLabel.Description.ValueString())
				}

				updatedLabel, _, err := r.client.V3Client().Issues.EditLabel(reqCtx, orgName, repository, oldLabel.Name.ValueString(), label)
				if err != nil {
					diags.AddError(
						"Unable to Update GitHub Issue Label",
						fmt.Sprintf("An unexpected error occurred when updating GitHub issue label %s. GitHub Client Error: %s", oldLabel.Name.ValueString(), err.Error()),
					)
					return
				}

				labelModel := githubIssueLabelModel{
					Name:  types.StringValue(updatedLabel.GetName()),
					Color: types.StringValue(updatedLabel.GetColor()),
					URL:   types.StringValue(updatedLabel.GetURL()),
				}

				if updatedLabel.Description != nil {
					labelModel.Description = types.StringValue(updatedLabel.GetDescription())
				} else {
					labelModel.Description = types.StringNull()
				}

				resultLabels = append(resultLabels, labelModel)
			} else {
				// No change needed, keep old label
				resultLabels = append(resultLabels, oldLabel)
			}
		}
	}

	// Set the ID and update the state
	data.ID = types.StringValue(repository)

	// Convert result labels to Set
	labelValues := make([]attr.Value, len(resultLabels))
	for i, label := range resultLabels {
		labelValues[i] = types.ObjectValueMust(labelObjectType.AttrTypes, map[string]attr.Value{
			"name":        label.Name,
			"color":       label.Color,
			"description": label.Description,
			"url":         label.URL,
		})
	}

	if len(labelValues) > 0 {
		data.Label = types.SetValueMust(labelObjectType, labelValues)
	} else {
		data.Label = types.SetValueMust(labelObjectType, []attr.Value{})
	}

	tflog.Debug(ctx, "GitHub issue labels operation completed", map[string]interface{}{
		"repository":  repository,
		"label_count": len(resultLabels),
	})
}
