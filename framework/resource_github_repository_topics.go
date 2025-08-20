package framework

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ resource.Resource                = &githubRepositoryTopicsResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryTopicsResource{}
	_ resource.ResourceWithImportState = &githubRepositoryTopicsResource{}
)

type githubRepositoryTopicsResource struct {
	client *githubpkg.Owner
}

type githubRepositoryTopicsResourceModel struct {
	Repository types.String `tfsdk:"repository"`
	Topics     types.Set    `tfsdk:"topics"`
}

func NewGithubRepositoryTopicsResource() resource.Resource {
	return &githubRepositoryTopicsResource{}
}

func (r *githubRepositoryTopicsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_topics"
}

func (r *githubRepositoryTopicsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages the topics on a repository",
		Attributes: map[string]schema.Attribute{
			"repository": schema.StringAttribute{
				Description: "The name of the repository. The name is not case sensitive.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[-a-zA-Z0-9_.]{1,100}$`),
						"must include only alphanumeric characters, underscores or hyphens and consist of 100 characters or less",
					),
				},
			},
			"topics": schema.SetAttribute{
				Description: "An array of topics to add to the repository. Pass one or more topics to replace the set of existing topics. Send an empty array ([]) to clear all topics from the repository. Note: Topic names cannot contain uppercase letters.",
				Required:    true,
				ElementType: types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,49}$`),
							"must include only lowercase alphanumeric characters or hyphens and cannot start with a hyphen and consist of 50 characters or less",
						),
					),
				},
			},
		},
	}
}

func (r *githubRepositoryTopicsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*githubpkg.Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *githubRepositoryTopicsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubRepositoryTopicsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()

	topics, diags := r.topicsSetToStringSlice(ctx, data.Topics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Always call ReplaceAllTopics, even with empty topics array to clear existing topics
	_, _, err := client.Repositories.ReplaceAllTopics(ctx, owner, repoName, topics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Repository Topics",
			fmt.Sprintf("An unexpected error occurred when creating the repository topics: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "created GitHub repository topics", map[string]interface{}{
		"repository": repoName,
		"topics":     topics,
	})

	// Read the created resource to populate all computed fields
	r.readGithubRepositoryTopics(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryTopicsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubRepositoryTopicsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubRepositoryTopics(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryTopicsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubRepositoryTopicsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()

	topics, diags := r.topicsSetToStringSlice(ctx, data.Topics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Always call ReplaceAllTopics, even with empty topics array to clear existing topics
	_, _, err := client.Repositories.ReplaceAllTopics(ctx, owner, repoName, topics)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Repository Topics",
			fmt.Sprintf("An unexpected error occurred when updating the repository topics: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "updated GitHub repository topics", map[string]interface{}{
		"repository": repoName,
		"topics":     topics,
	})

	// Read the updated resource to populate all computed fields
	r.readGithubRepositoryTopics(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryTopicsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubRepositoryTopicsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()

	// Clear all topics by passing an empty array
	_, _, err := client.Repositories.ReplaceAllTopics(ctx, owner, repoName, []string{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Repository Topics",
			fmt.Sprintf("An unexpected error occurred when deleting the repository topics: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub repository topics", map[string]interface{}{
		"repository": repoName,
	})
}

func (r *githubRepositoryTopicsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	data := &githubRepositoryTopicsResourceModel{
		Repository: types.StringValue(req.ID),
	}

	r.readGithubRepositoryTopics(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubRepositoryTopicsResource) readGithubRepositoryTopics(ctx context.Context, data *githubRepositoryTopicsResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()

	// Set up context with ID for logging
	reqCtx := context.WithValue(ctx, githubpkg.CtxId, repoName)

	topics, _, err := client.Repositories.ListAllTopics(reqCtx, owner, repoName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotModified {
				// Resource hasn't changed, keep current state
				return
			}
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "GitHub repository not found, removing topics from state", map[string]interface{}{
					"repository": repoName,
				})
				data.Repository = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Repository Topics",
			fmt.Sprintf("An unexpected error occurred when reading the repository topics: %s", err.Error()),
		)
		return
	}

	// Convert string slice to Set
	topicsSet, setDiags := types.SetValueFrom(ctx, types.StringType, topics)
	diags.Append(setDiags...)
	if diags.HasError() {
		return
	}

	data.Topics = topicsSet

	tflog.Debug(ctx, "successfully read GitHub repository topics", map[string]interface{}{
		"repository": repoName,
		"topics":     topics,
	})
}

func (r *githubRepositoryTopicsResource) topicsSetToStringSlice(ctx context.Context, topicsSet types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	var topics []string

	if topicsSet.IsNull() || topicsSet.IsUnknown() {
		return topics, diags
	}

	elements := topicsSet.Elements()
	topics = make([]string, 0, len(elements))

	for _, element := range elements {
		if str, ok := element.(types.String); ok {
			if !str.IsNull() && !str.IsUnknown() {
				topics = append(topics, str.ValueString())
			}
		}
	}

	return topics, diags
}
