package github

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ resource.Resource                = &githubRepositoryMilestoneResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryMilestoneResource{}
	_ resource.ResourceWithImportState = &githubRepositoryMilestoneResource{}
)

type githubRepositoryMilestoneResource struct {
	client *Owner
}

type githubRepositoryMilestoneResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Title       types.String `tfsdk:"title"`
	Owner       types.String `tfsdk:"owner"`
	Repository  types.String `tfsdk:"repository"`
	Description types.String `tfsdk:"description"`
	DueDate     types.String `tfsdk:"due_date"`
	State       types.String `tfsdk:"state"`
	Number      types.Int64  `tfsdk:"number"`
}

const (
	layoutISO = "2006-01-02"
)

func NewGithubRepositoryMilestoneResource() resource.Resource {
	return &githubRepositoryMilestoneResource{}
}

func (r *githubRepositoryMilestoneResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_milestone"
}

func (r *githubRepositoryMilestoneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub repository milestone resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the milestone.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Description: "The title of the milestone.",
				Required:    true,
			},
			"owner": schema.StringAttribute{
				Description: "The owner of the GitHub Repository.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The name of the GitHub Repository.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of the milestone.",
				Optional:    true,
			},
			"due_date": schema.StringAttribute{
				Description: "The milestone due date. In 'yyyy-mm-dd' format.",
				Optional:    true,
			},
			"state": schema.StringAttribute{
				Description: "The state of the milestone. Either 'open' or 'closed'. Default: 'open'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("open"),
				Validators: []validator.String{
					stringvalidator.OneOf("open", "closed"),
				},
			},
			"number": schema.Int64Attribute{
				Description: "The number of the milestone.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubRepositoryMilestoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubRepositoryMilestoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubRepositoryMilestoneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := data.Owner.ValueString()
	repoName := data.Repository.ValueString()

	milestone := &github.Milestone{
		Title: github.Ptr(data.Title.ValueString()),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		description := data.Description.ValueString()
		if len(description) > 0 {
			milestone.Description = github.Ptr(description)
		}
	}

	if !data.DueDate.IsNull() && !data.DueDate.IsUnknown() {
		dueDateString := data.DueDate.ValueString()
		if len(dueDateString) > 0 {
			dueDate, err := time.Parse(layoutISO, dueDateString)
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Parse Due Date",
					fmt.Sprintf("Unable to parse due date '%s': %s", dueDateString, err.Error()),
				)
				return
			}
			date := time.Date(dueDate.Year(), dueDate.Month(), dueDate.Day(), 23, 39, 0, 0, time.UTC)
			milestone.DueOn = &github.Timestamp{
				Time: date,
			}
		}
	}

	if !data.State.IsNull() && !data.State.IsUnknown() {
		state := data.State.ValueString()
		if len(state) > 0 {
			milestone.State = github.Ptr(state)
		}
	}

	resultMilestone, _, err := client.Issues.CreateMilestone(ctx, owner, repoName, milestone)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Repository Milestone",
			fmt.Sprintf("An unexpected error occurred when creating the repository milestone: %s", err.Error()),
		)
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s/%d", owner, repoName, resultMilestone.GetNumber()))
	data.Number = types.Int64Value(int64(resultMilestone.GetNumber()))

	tflog.Debug(ctx, "created GitHub repository milestone", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"owner":      owner,
		"repository": repoName,
		"title":      data.Title.ValueString(),
		"number":     resultMilestone.GetNumber(),
	})

	// Read the created resource to populate computed fields
	r.readGithubRepositoryMilestone(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryMilestoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubRepositoryMilestoneResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubRepositoryMilestone(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryMilestoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubRepositoryMilestoneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := data.Owner.ValueString()
	repoName := data.Repository.ValueString()
	number, err := r.parseMilestoneNumber(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Milestone Number",
			fmt.Sprintf("Unable to parse milestone number from ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	var current githubRepositoryMilestoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	milestone := &github.Milestone{}

	if !data.Title.Equal(current.Title) {
		milestone.Title = github.Ptr(data.Title.ValueString())
	}

	if !data.Description.Equal(current.Description) {
		milestone.Description = github.Ptr(data.Description.ValueString())
	}

	if !data.DueDate.Equal(current.DueDate) {
		if !data.DueDate.IsNull() && !data.DueDate.IsUnknown() {
			dueDateString := data.DueDate.ValueString()
			if len(dueDateString) > 0 {
				dueDate, err := time.Parse(layoutISO, dueDateString)
				if err != nil {
					resp.Diagnostics.AddError(
						"Unable to Parse Due Date",
						fmt.Sprintf("Unable to parse due date '%s': %s", dueDateString, err.Error()),
					)
					return
				}
				date := time.Date(dueDate.Year(), dueDate.Month(), dueDate.Day(), 7, 0, 0, 0, time.UTC)
				milestone.DueOn = &github.Timestamp{
					Time: date,
				}
			}
		}
	}

	if !data.State.Equal(current.State) {
		milestone.State = github.Ptr(data.State.ValueString())
	}

	_, _, err = client.Issues.EditMilestone(ctx, owner, repoName, number, milestone)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Repository Milestone",
			fmt.Sprintf("An unexpected error occurred when updating the repository milestone: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "updated GitHub repository milestone", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"owner":      owner,
		"repository": repoName,
		"number":     number,
	})

	// Read the updated resource to populate computed fields
	r.readGithubRepositoryMilestone(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryMilestoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubRepositoryMilestoneResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := data.Owner.ValueString()
	repoName := data.Repository.ValueString()
	number, err := r.parseMilestoneNumber(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Milestone Number",
			fmt.Sprintf("Unable to parse milestone number from ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, err = client.Issues.DeleteMilestone(ctx, owner, repoName, number)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Repository Milestone",
			fmt.Sprintf("An unexpected error occurred when deleting the repository milestone: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub repository milestone", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"owner":      owner,
		"repository": repoName,
		"number":     number,
	})
}

func (r *githubRepositoryMilestoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "OWNER/REPOSITORY/NUMBER"
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Invalid ID format, must be provided as OWNER/REPOSITORY/NUMBER. Got: %q", req.ID),
		)
		return
	}

	owner := parts[0]
	repository := parts[1]
	numberStr := parts[2]

	number, err := strconv.Atoi(numberStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Milestone Number",
			fmt.Sprintf("Unable to parse milestone number '%s': %s", numberStr, err.Error()),
		)
		return
	}

	client := r.client.V3Client()

	// Verify the milestone exists
	milestone, _, err := client.Issues.GetMilestone(ctx, owner, repository, number)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import Repository Milestone",
			fmt.Sprintf("Unable to read repository milestone for import: %s", err.Error()),
		)
		return
	}

	data := &githubRepositoryMilestoneResourceModel{
		ID:         types.StringValue(req.ID),
		Owner:      types.StringValue(owner),
		Repository: types.StringValue(repository),
		Title:      types.StringValue(milestone.GetTitle()),
		Number:     types.Int64Value(int64(milestone.GetNumber())),
		State:      types.StringValue(milestone.GetState()),
	}

	// Set optional fields if they exist
	if milestone.GetDescription() != "" {
		data.Description = types.StringValue(milestone.GetDescription())
	} else {
		data.Description = types.StringNull()
	}

	if dueOn := milestone.GetDueOn(); !dueOn.IsZero() {
		data.DueDate = types.StringValue(dueOn.Format(layoutISO))
	} else {
		data.DueDate = types.StringNull()
	}

	tflog.Debug(ctx, "imported GitHub repository milestone", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"owner":      owner,
		"repository": repository,
		"number":     number,
		"title":      milestone.GetTitle(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubRepositoryMilestoneResource) parseMilestoneNumber(id string) (int, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return -1, fmt.Errorf("ID not properly formatted: %s", id)
	}
	number, err := strconv.Atoi(parts[2])
	if err != nil {
		return -1, err
	}
	return number, nil
}

func (r *githubRepositoryMilestoneResource) readGithubRepositoryMilestone(ctx context.Context, data *githubRepositoryMilestoneResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := data.Owner.ValueString()
	repoName := data.Repository.ValueString()
	number, err := r.parseMilestoneNumber(data.ID.ValueString())
	if err != nil {
		diags.AddError(
			"Unable to Parse Milestone Number",
			fmt.Sprintf("Unable to parse milestone number from ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	milestone, _, err := client.Issues.GetMilestone(ctx, owner, repoName, number)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				// Resource hasn't changed, no need to update state
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing milestone from state because it no longer exists in GitHub", map[string]interface{}{
					"owner":      owner,
					"repository": repoName,
					"number":     number,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Repository Milestone",
			fmt.Sprintf("An unexpected error occurred when reading the repository milestone: %s", err.Error()),
		)
		return
	}

	// Update the state with the latest values
	data.Title = types.StringValue(milestone.GetTitle())
	data.Number = types.Int64Value(int64(milestone.GetNumber()))
	data.State = types.StringValue(milestone.GetState())

	// Handle optional fields
	if milestone.GetDescription() != "" {
		data.Description = types.StringValue(milestone.GetDescription())
	} else {
		data.Description = types.StringNull()
	}

	if dueOn := milestone.GetDueOn(); !dueOn.IsZero() {
		data.DueDate = types.StringValue(dueOn.Format(layoutISO))
	} else {
		data.DueDate = types.StringNull()
	}

	tflog.Debug(ctx, "successfully read GitHub repository milestone", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"owner":      owner,
		"repository": repoName,
		"number":     number,
		"title":      milestone.GetTitle(),
	})
}
