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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubIssueResource{}
	_ resource.ResourceWithConfigure   = &githubIssueResource{}
	_ resource.ResourceWithImportState = &githubIssueResource{}
)

type githubIssueResource struct {
	client *Owner
}

type githubIssueResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Repository      types.String `tfsdk:"repository"`
	Number          types.Int64  `tfsdk:"number"`
	Title           types.String `tfsdk:"title"`
	Body            types.String `tfsdk:"body"`
	Labels          types.Set    `tfsdk:"labels"`
	Assignees       types.Set    `tfsdk:"assignees"`
	MilestoneNumber types.Int64  `tfsdk:"milestone_number"`
	IssueID         types.Int64  `tfsdk:"issue_id"`
	Etag            types.String `tfsdk:"etag"`
}

func NewGithubIssueResource() resource.Resource {
	return &githubIssueResource{}
}

func (r *githubIssueResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_issue"
}

func (r *githubIssueResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub issue resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the issue (repository:number).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The GitHub repository name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"number": schema.Int64Attribute{
				Description: "The issue number.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"title": schema.StringAttribute{
				Description: "Title of the issue.",
				Required:    true,
			},
			"body": schema.StringAttribute{
				Description: "Body of the issue.",
				Optional:    true,
				Computed:    true,
			},
			"labels": schema.SetAttribute{
				Description: "List of labels to attach to the issue.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"assignees": schema.SetAttribute{
				Description: "List of Logins to assign to the issue.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"milestone_number": schema.Int64Attribute{
				Description: "Milestone number to assign to the issue.",
				Optional:    true,
			},
			"issue_id": schema.Int64Attribute{
				Description: "The issue id.",
				Computed:    true,
			},
			"etag": schema.StringAttribute{
				Description: "The ETag of the issue.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubIssueResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubIssueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubIssueResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgName := r.client.Name()
	repoName := data.Repository.ValueString()
	title := data.Title.ValueString()

	issueReq := &github.IssueRequest{
		Title: github.Ptr(title),
	}

	if !data.Body.IsNull() {
		issueReq.Body = github.Ptr(data.Body.ValueString())
	}

	// Handle labels
	if !data.Labels.IsNull() && !data.Labels.IsUnknown() {
		var labels []types.String
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		labelsSlice := make([]string, len(labels))
		for i, label := range labels {
			labelsSlice[i] = label.ValueString()
		}
		issueReq.Labels = &labelsSlice
	}

	// Handle assignees
	if !data.Assignees.IsNull() && !data.Assignees.IsUnknown() {
		var assignees []types.String
		resp.Diagnostics.Append(data.Assignees.ElementsAs(ctx, &assignees, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		assigneesSlice := make([]string, len(assignees))
		for i, assignee := range assignees {
			assigneesSlice[i] = assignee.ValueString()
		}
		issueReq.Assignees = &assigneesSlice
	}

	// Handle milestone
	if !data.MilestoneNumber.IsNull() && !data.MilestoneNumber.IsUnknown() {
		milestone := int(data.MilestoneNumber.ValueInt64())
		if milestone > 0 {
			issueReq.Milestone = &milestone
		}
	}

	tflog.Debug(ctx, "creating GitHub issue", map[string]any{
		"repository": repoName,
		"title":      title,
		"owner":      orgName,
	})

	issue, httpResp, err := r.client.V3Client().Issues.Create(ctx, orgName, repoName, issueReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create GitHub Issue",
			"An unexpected error occurred when creating the GitHub issue. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}
	if httpResp != nil {
		tflog.Debug(ctx, "GitHub API response", map[string]any{
			"status": httpResp.Status,
		})
	}

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("%s:%d", repoName, issue.GetNumber()))
	data.Number = types.Int64Value(int64(issue.GetNumber()))
	data.IssueID = types.Int64Value(issue.GetID())

	tflog.Debug(ctx, "created GitHub issue", map[string]any{
		"id":     data.ID.ValueString(),
		"number": data.Number.ValueInt64(),
		"title":  title,
	})

	// Read the created resource to populate all computed fields
	r.readGithubIssue(ctx, &data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubIssueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubIssueResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubIssue(ctx, &data, &resp.Diagnostics, true)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubIssueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubIssueResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgName := r.client.Name()
	repoName := data.Repository.ValueString()
	title := data.Title.ValueString()
	number := int(data.Number.ValueInt64())

	issueReq := &github.IssueRequest{
		Title: github.Ptr(title),
	}

	if !data.Body.IsNull() {
		issueReq.Body = github.Ptr(data.Body.ValueString())
	}

	// Handle labels
	if !data.Labels.IsNull() && !data.Labels.IsUnknown() {
		var labels []types.String
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		labelsSlice := make([]string, len(labels))
		for i, label := range labels {
			labelsSlice[i] = label.ValueString()
		}
		issueReq.Labels = &labelsSlice
	}

	// Handle assignees
	if !data.Assignees.IsNull() && !data.Assignees.IsUnknown() {
		var assignees []types.String
		resp.Diagnostics.Append(data.Assignees.ElementsAs(ctx, &assignees, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		assigneesSlice := make([]string, len(assignees))
		for i, assignee := range assignees {
			assigneesSlice[i] = assignee.ValueString()
		}
		issueReq.Assignees = &assigneesSlice
	}

	// Handle milestone
	if !data.MilestoneNumber.IsNull() && !data.MilestoneNumber.IsUnknown() {
		milestone := int(data.MilestoneNumber.ValueInt64())
		if milestone > 0 {
			issueReq.Milestone = &milestone
		}
	}

	tflog.Debug(ctx, "updating GitHub issue", map[string]any{
		"repository": repoName,
		"number":     number,
		"title":      title,
		"owner":      orgName,
	})

	issue, httpResp, err := r.client.V3Client().Issues.Edit(ctx, orgName, repoName, number, issueReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update GitHub Issue",
			"An unexpected error occurred when updating the GitHub issue. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}
	if httpResp != nil {
		tflog.Debug(ctx, "GitHub API response", map[string]any{
			"status": httpResp.Status,
		})
	}

	data.IssueID = types.Int64Value(issue.GetID())

	tflog.Debug(ctx, "updated GitHub issue", map[string]any{
		"id":     data.ID.ValueString(),
		"number": number,
		"title":  title,
	})

	// Read the updated resource to populate all computed fields
	r.readGithubIssue(ctx, &data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubIssueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubIssueResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgName := r.client.Name()
	repoName := data.Repository.ValueString()
	number := int(data.Number.ValueInt64())

	// GitHub doesn't allow deleting issues, so we close them instead
	closeReq := &github.IssueRequest{
		State: github.Ptr("closed"),
	}

	tflog.Debug(ctx, "closing GitHub issue", map[string]any{
		"repository": repoName,
		"number":     number,
		"owner":      orgName,
	})

	_, _, err := r.client.V3Client().Issues.Edit(ctx, orgName, repoName, number, closeReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Close GitHub Issue",
			"An unexpected error occurred when closing the GitHub issue. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "closed GitHub issue", map[string]any{
		"id":     data.ID.ValueString(),
		"number": number,
	})
}

func (r *githubIssueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "repository:number"
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: repository:number. Got: %q", req.ID),
		)
		return
	}

	repoName := parts[0]
	numberStr := parts[1]

	number, err := strconv.Atoi(numberStr)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Issue Number",
			fmt.Sprintf("Unable to parse issue number %q as integer: %s", numberStr, err.Error()),
		)
		return
	}

	data := &githubIssueResourceModel{
		ID:         types.StringValue(req.ID),
		Repository: types.StringValue(repoName),
		Number:     types.Int64Value(int64(number)),
	}

	r.readGithubIssue(ctx, data, &resp.Diagnostics, false)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *githubIssueResource) readGithubIssue(ctx context.Context, data *githubIssueResourceModel, diags *diag.Diagnostics, useEtag bool) {
	orgName := r.client.Name()
	repoName := data.Repository.ValueString()
	number := int(data.Number.ValueInt64())

	// Set up context with ETag if this is not a new resource and useEtag is true
	reqCtx := ctx
	if useEtag && !data.Etag.IsNull() && data.Etag.ValueString() != "" {
		reqCtx = context.WithValue(ctx, CtxEtag, data.Etag.ValueString())
	}
	reqCtx = context.WithValue(reqCtx, CtxId, data.ID.ValueString())

	tflog.Debug(ctx, "reading GitHub issue", map[string]any{
		"repository": repoName,
		"number":     number,
		"owner":      orgName,
	})

	issue, resp, err := r.client.V3Client().Issues.Get(reqCtx, orgName, repoName, number)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				// Resource hasn't changed, keep current state
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "GitHub issue not found, removing from state", map[string]any{
					"id":         data.ID.ValueString(),
					"repository": repoName,
					"number":     number,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read GitHub Issue",
			"An unexpected error occurred when reading the GitHub issue. "+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	// Set computed and read values
	data.Etag = types.StringValue(resp.Header.Get("ETag"))
	data.Title = types.StringValue(issue.GetTitle())
	data.IssueID = types.Int64Value(issue.GetID())

	if issue.Body != nil {
		data.Body = types.StringValue(issue.GetBody())
	} else {
		data.Body = types.StringNull()
	}

	if issue.Milestone != nil && issue.Milestone.Number != nil {
		data.MilestoneNumber = types.Int64Value(int64(issue.GetMilestone().GetNumber()))
	} else {
		data.MilestoneNumber = types.Int64Null()
	}

	// Handle labels
	if len(issue.Labels) > 0 {
		labels := make([]string, len(issue.Labels))
		for i, label := range issue.Labels {
			labels[i] = label.GetName()
		}
		labelSet, setDiags := types.SetValueFrom(ctx, types.StringType, labels)
		diags.Append(setDiags...)
		if diags.HasError() {
			return
		}
		data.Labels = labelSet
	} else {
		data.Labels = types.SetNull(types.StringType)
	}

	// Handle assignees
	if len(issue.Assignees) > 0 {
		assignees := make([]string, len(issue.Assignees))
		for i, assignee := range issue.Assignees {
			assignees[i] = assignee.GetLogin()
		}
		assigneeSet, setDiags := types.SetValueFrom(ctx, types.StringType, assignees)
		diags.Append(setDiags...)
		if diags.HasError() {
			return
		}
		data.Assignees = assigneeSet
	} else {
		data.Assignees = types.SetNull(types.StringType)
	}

	tflog.Debug(ctx, "successfully read GitHub issue", map[string]any{
		"id":         data.ID.ValueString(),
		"title":      data.Title.ValueString(),
		"repository": repoName,
		"number":     number,
	})
}
