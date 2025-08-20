package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubRepositoryPullRequestResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryPullRequestResource{}
	_ resource.ResourceWithImportState = &githubRepositoryPullRequestResource{}
)

type githubRepositoryPullRequestResource struct {
	client *Owner
}

type githubRepositoryPullRequestResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Owner               types.String `tfsdk:"owner"`
	BaseRepository      types.String `tfsdk:"base_repository"`
	BaseRef             types.String `tfsdk:"base_ref"`
	HeadRef             types.String `tfsdk:"head_ref"`
	Title               types.String `tfsdk:"title"`
	Body                types.String `tfsdk:"body"`
	MaintainerCanModify types.Bool   `tfsdk:"maintainer_can_modify"`
	BaseSha             types.String `tfsdk:"base_sha"`
	Draft               types.Bool   `tfsdk:"draft"`
	HeadSha             types.String `tfsdk:"head_sha"`
	Labels              types.List   `tfsdk:"labels"`
	Number              types.Int64  `tfsdk:"number"`
	OpenedAt            types.Int64  `tfsdk:"opened_at"`
	OpenedBy            types.String `tfsdk:"opened_by"`
	State               types.String `tfsdk:"state"`
	UpdatedAt           types.Int64  `tfsdk:"updated_at"`
}

func NewGithubRepositoryPullRequestResource() resource.Resource {
	return &githubRepositoryPullRequestResource{}
}

func (r *githubRepositoryPullRequestResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_pull_request"
}

func (r *githubRepositoryPullRequestResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a single GitHub Pull Request.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the pull request (owner:repository:number).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"owner": schema.StringAttribute{
				Description: "Owner of the repository. If not provided, the provider's default owner is used.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"base_repository": schema.StringAttribute{
				Description: "Name of the base repository to retrieve the Pull Requests from.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"base_ref": schema.StringAttribute{
				Description: "Name of the branch serving as the base of the Pull Request.",
				Required:    true,
			},
			"head_ref": schema.StringAttribute{
				Description: "Name of the branch serving as the head of the Pull Request.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"title": schema.StringAttribute{
				Description: "The title of the Pull Request.",
				Required:    true,
			},
			"body": schema.StringAttribute{
				Description: "Body of the Pull Request.",
				Optional:    true,
			},
			"maintainer_can_modify": schema.BoolAttribute{
				Description: "Controls whether the base repository maintainers can modify the Pull Request. Default: 'false'.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"base_sha": schema.StringAttribute{
				Description: "Head commit SHA of the Pull Request base.",
				Computed:    true,
			},
			"draft": schema.BoolAttribute{
				Description: "Indicates Whether this Pull Request is a draft.",
				Computed:    true,
			},
			"head_sha": schema.StringAttribute{
				Description: "Head commit SHA of the Pull Request head.",
				Computed:    true,
			},
			"labels": schema.ListAttribute{
				Description: "List of names of labels on the PR",
				Computed:    true,
				ElementType: types.StringType,
			},
			"number": schema.Int64Attribute{
				Description: "The number of the Pull Request within the repository.",
				Computed:    true,
			},
			"opened_at": schema.Int64Attribute{
				Description: "Unix timestamp indicating the Pull Request creation time.",
				Computed:    true,
			},
			"opened_by": schema.StringAttribute{
				Description: "Username of the PR creator",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "The current Pull Request state - can be 'open', 'closed' or 'merged'.",
				Computed:    true,
			},
			"updated_at": schema.Int64Attribute{
				Description: "The timestamp of the last Pull Request update.",
				Computed:    true,
			},
		},
	}
}

func (r *githubRepositoryPullRequestResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubRepositoryPullRequestResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubRepositoryPullRequestResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()

	// For convenience, by default we expect that the base repository and head
	// repository owners are the same, and both belong to the caller, indicating
	// a "PR within the same repo" scenario. The head will *always* belong to
	// the current caller, the base - not necessarily. The base will belong to
	// another namespace in case of forks, and this resource supports them.
	headOwner := r.client.Name()

	baseOwner := headOwner
	if !data.Owner.IsNull() && !data.Owner.IsUnknown() {
		baseOwner = data.Owner.ValueString()
	}

	baseRepository := data.BaseRepository.ValueString()

	head := data.HeadRef.ValueString()
	if headOwner != baseOwner {
		head = strings.Join([]string{headOwner, head}, ":")
	}

	pullRequest, _, err := client.PullRequests.Create(ctx, baseOwner, baseRepository, &github.NewPullRequest{
		Title:               github.Ptr(data.Title.ValueString()),
		Head:                github.Ptr(head),
		Base:                github.Ptr(data.BaseRef.ValueString()),
		Body:                github.Ptr(data.Body.ValueString()),
		MaintainerCanModify: github.Ptr(data.MaintainerCanModify.ValueBool()),
	})

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Pull Request",
			fmt.Sprintf("An unexpected error occurred when creating the pull request: %s", err.Error()),
		)
		return
	}

	data.ID = types.StringValue(r.buildThreePartID(baseOwner, baseRepository, strconv.Itoa(pullRequest.GetNumber())))

	tflog.Debug(ctx, "created GitHub repository pull request", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": baseRepository,
		"number":     pullRequest.GetNumber(),
	})

	// Read the created resource to populate all computed fields
	r.readGithubRepositoryPullRequest(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryPullRequestResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubRepositoryPullRequestResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubRepositoryPullRequest(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryPullRequestResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubRepositoryPullRequestResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()

	owner, repository, number, err := r.parsePullRequestID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Pull Request ID",
			fmt.Sprintf("Unable to parse pull request ID: %s", err.Error()),
		)
		return
	}

	update := &github.PullRequest{
		Title:               github.Ptr(data.Title.ValueString()),
		Body:                github.Ptr(data.Body.ValueString()),
		MaintainerCanModify: github.Ptr(data.MaintainerCanModify.ValueBool()),
	}

	// Check if base_ref has changed by getting state and plan values
	var statePlan, currentState githubRepositoryPullRequestResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &statePlan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &currentState)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if !statePlan.BaseRef.Equal(currentState.BaseRef) {
		update.Base = &github.PullRequestBranch{
			Ref: github.Ptr(data.BaseRef.ValueString()),
		}
	}

	_, _, err = client.PullRequests.Edit(ctx, owner, repository, number, update)
	if err == nil {
		// Read the updated resource to populate all computed fields
		r.readGithubRepositoryPullRequest(ctx, &data, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	resp.Diagnostics.AddError(
		"Unable to Update Pull Request",
		fmt.Sprintf("Could not update the Pull Request: %s", err.Error()),
	)

	// Try to read the current state after the failed update
	r.readGithubRepositoryPullRequest(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryPullRequestResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubRepositoryPullRequestResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// It's not entirely clear how to treat PR deletion according to Terraform's
	// CRUD semantics. The approach we're taking here is to close the PR unless
	// it's already closed or merged. Merging it feels intuitively wrong in what
	// effectively is a destructor.
	if !data.State.IsNull() && !data.State.IsUnknown() && data.State.ValueString() != "open" {
		tflog.Debug(ctx, "pull request is not open, skipping close operation", map[string]interface{}{
			"id":    data.ID.ValueString(),
			"state": data.State.ValueString(),
		})
		return
	}

	client := r.client.V3Client()

	owner, repository, number, err := r.parsePullRequestID(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Pull Request ID",
			fmt.Sprintf("Unable to parse pull request ID: %s", err.Error()),
		)
		return
	}

	update := &github.PullRequest{State: github.Ptr("closed")}
	if _, _, err = client.PullRequests.Edit(ctx, owner, repository, number, update); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Close Pull Request",
			fmt.Sprintf("An unexpected error occurred when closing the pull request: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "closed GitHub repository pull request", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repository,
		"number":     number,
	})
}

func (r *githubRepositoryPullRequestResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID to set the base_repository attribute
	owner, baseRepository, _, err := r.parsePullRequestID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Unable to parse import ID: %s", err.Error()),
		)
		return
	}

	data := &githubRepositoryPullRequestResourceModel{
		ID:             types.StringValue(req.ID),
		BaseRepository: types.StringValue(baseRepository),
	}

	// Set the owner if it's different from the provider's default
	if owner != r.client.Name() {
		data.Owner = types.StringValue(owner)
	}

	r.readGithubRepositoryPullRequest(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubRepositoryPullRequestResource) parsePullRequestID(id string) (owner, repository string, number int, err error) {
	var strNumber string

	if owner, repository, strNumber, err = parseThreePartID(id, "owner", "base_repository", "number"); err != nil {
		return
	}

	if number, err = strconv.Atoi(strNumber); err != nil {
		err = fmt.Errorf("invalid PR number %s: %w", strNumber, err)
	}

	return
}

func (r *githubRepositoryPullRequestResource) buildThreePartID(a, b, c string) string {
	return fmt.Sprintf("%s:%s:%s", a, b, c)
}

func (r *githubRepositoryPullRequestResource) readGithubRepositoryPullRequest(ctx context.Context, data *githubRepositoryPullRequestResourceModel, diagnostics *diag.Diagnostics) {
	client := r.client.V3Client()

	owner, repository, number, err := r.parsePullRequestID(data.ID.ValueString())
	if err != nil {
		diagnostics.AddError(
			"Invalid Pull Request ID",
			fmt.Sprintf("Unable to parse pull request ID: %s", err.Error()),
		)
		return
	}

	pullRequest, _, err := client.PullRequests.Get(ctx, owner, repository, number)
	if err != nil {
		diagnostics.AddError(
			"Unable to Read Pull Request",
			fmt.Sprintf("An unexpected error occurred when reading the pull request: %s", err.Error()),
		)
		return
	}

	data.Number = types.Int64Value(int64(pullRequest.GetNumber()))

	if head := pullRequest.GetHead(); head != nil {
		data.HeadRef = types.StringValue(head.GetRef())
		data.HeadSha = types.StringValue(head.GetSHA())
	} else {
		// Totally unexpected condition. Better do that than segfault, I guess?
		tflog.Warn(ctx, "Head branch missing", map[string]interface{}{
			"expected_head_ref": data.HeadRef.ValueString(),
		})
		data.ID = types.StringNull()
		return
	}

	if base := pullRequest.GetBase(); base != nil {
		data.BaseRef = types.StringValue(base.GetRef())
		data.BaseSha = types.StringValue(base.GetSHA())
	} else {
		// Same logic as with the missing head branch.
		tflog.Warn(ctx, "Base branch missing", map[string]interface{}{
			"expected_base_ref": data.BaseRef.ValueString(),
		})
		data.ID = types.StringNull()
		return
	}

	data.Body = types.StringValue(pullRequest.GetBody())
	data.Title = types.StringValue(pullRequest.GetTitle())
	data.Draft = types.BoolValue(pullRequest.GetDraft())
	data.MaintainerCanModify = types.BoolValue(pullRequest.GetMaintainerCanModify())
	data.State = types.StringValue(pullRequest.GetState())
	data.OpenedAt = types.Int64Value(pullRequest.GetCreatedAt().Unix())
	data.UpdatedAt = types.Int64Value(pullRequest.GetUpdatedAt().Unix())

	if user := pullRequest.GetUser(); user != nil {
		data.OpenedBy = types.StringValue(user.GetLogin())
	}

	labels := []string{}
	for _, label := range pullRequest.Labels {
		labels = append(labels, label.GetName())
	}
	labelList, diag := types.ListValueFrom(ctx, types.StringType, labels)
	diagnostics.Append(diag...)
	data.Labels = labelList

	tflog.Debug(ctx, "successfully read GitHub repository pull request", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repository,
		"number":     data.Number.ValueInt64(),
		"state":      data.State.ValueString(),
	})
}
