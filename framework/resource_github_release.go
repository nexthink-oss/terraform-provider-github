package framework

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ resource.Resource                = &githubReleaseResource{}
	_ resource.ResourceWithConfigure   = &githubReleaseResource{}
	_ resource.ResourceWithImportState = &githubReleaseResource{}
)

type githubReleaseResource struct {
	client *githubpkg.Owner
}

type githubReleaseResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Repository             types.String `tfsdk:"repository"`
	TagName                types.String `tfsdk:"tag_name"`
	TargetCommitish        types.String `tfsdk:"target_commitish"`
	Name                   types.String `tfsdk:"name"`
	Body                   types.String `tfsdk:"body"`
	Draft                  types.Bool   `tfsdk:"draft"`
	Prerelease             types.Bool   `tfsdk:"prerelease"`
	GenerateReleaseNotes   types.Bool   `tfsdk:"generate_release_notes"`
	DiscussionCategoryName types.String `tfsdk:"discussion_category_name"`
	Etag                   types.String `tfsdk:"etag"`
	ReleaseID              types.Int64  `tfsdk:"release_id"`
	NodeID                 types.String `tfsdk:"node_id"`
	CreatedAt              types.String `tfsdk:"created_at"`
	PublishedAt            types.String `tfsdk:"published_at"`
	URL                    types.String `tfsdk:"url"`
	HTMLURL                types.String `tfsdk:"html_url"`
	AssetsURL              types.String `tfsdk:"assets_url"`
	UploadURL              types.String `tfsdk:"upload_url"`
	ZipballURL             types.String `tfsdk:"zipball_url"`
	TarballURL             types.String `tfsdk:"tarball_url"`
}

func NewGithubReleaseResource() resource.Resource {
	return &githubReleaseResource{}
}

func (r *githubReleaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release"
}

func (r *githubReleaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages releases within a single GitHub repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the release.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tag_name": schema.StringAttribute{
				Description: "The name of the tag.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_commitish": schema.StringAttribute{
				Description: "The branch name or commit SHA the tag is created from.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("main"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the release.",
				Optional:    true,
			},
			"body": schema.StringAttribute{
				Description: "Text describing the contents of the tag.",
				Optional:    true,
			},
			"draft": schema.BoolAttribute{
				Description: "Set to 'false' to create a published release.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"prerelease": schema.BoolAttribute{
				Description: "Set to 'false' to identify the release as a full release.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"generate_release_notes": schema.BoolAttribute{
				Description: "Set to 'true' to automatically generate the name and body for this release. If 'name' is specified, the specified name will be used; otherwise, a name will be automatically generated. If 'body' is specified, the body will be pre-pended to the automatically generated notes.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"discussion_category_name": schema.StringAttribute{
				Description: "If specified, a discussion of the specified category is created and linked to the release. The value must be a category that already exists in the repository.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"etag": schema.StringAttribute{
				Description: "ETag of the release.",
				Computed:    true,
			},
			"release_id": schema.Int64Attribute{
				Description: "The ID of the release.",
				Computed:    true,
			},
			"node_id": schema.StringAttribute{
				Description: "The node ID of the release.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The date and time the release was created.",
				Computed:    true,
			},
			"published_at": schema.StringAttribute{
				Description: "The date and time the release was published.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "The URL for the release.",
				Computed:    true,
			},
			"html_url": schema.StringAttribute{
				Description: "The HTML URL for the release.",
				Computed:    true,
			},
			"assets_url": schema.StringAttribute{
				Description: "The URL for the release assets.",
				Computed:    true,
			},
			"upload_url": schema.StringAttribute{
				Description: "The URL for the uploaded assets of release.",
				Computed:    true,
			},
			"zipball_url": schema.StringAttribute{
				Description: "The URL for the zipball of the release.",
				Computed:    true,
			},
			"tarball_url": schema.StringAttribute{
				Description: "The URL for the tarball of the release.",
				Computed:    true,
			},
		},
	}
}

func (r *githubReleaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubReleaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubReleaseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()
	tagName := data.TagName.ValueString()
	targetCommitish := data.TargetCommitish.ValueString()
	draft := data.Draft.ValueBool()
	prerelease := data.Prerelease.ValueBool()
	generateReleaseNotes := data.GenerateReleaseNotes.ValueBool()

	release := &github.RepositoryRelease{
		TagName:              github.Ptr(tagName),
		TargetCommitish:      github.Ptr(targetCommitish),
		Draft:                github.Ptr(draft),
		Prerelease:           github.Ptr(prerelease),
		GenerateReleaseNotes: github.Ptr(generateReleaseNotes),
	}

	if !data.Body.IsNull() && !data.Body.IsUnknown() {
		release.Body = github.Ptr(data.Body.ValueString())
	}

	if !data.Name.IsNull() && !data.Name.IsUnknown() {
		release.Name = github.Ptr(data.Name.ValueString())
	}

	if !data.DiscussionCategoryName.IsNull() && !data.DiscussionCategoryName.IsUnknown() {
		release.DiscussionCategoryName = github.Ptr(data.DiscussionCategoryName.ValueString())
	}

	createdRelease, resp2, err := client.Repositories.CreateRelease(ctx, owner, repoName, release)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create GitHub Release",
			fmt.Sprintf("An unexpected error occurred when creating the GitHub release: %s", err.Error()),
		)
		return
	}

	// Set the ID and populate the data model
	data.ID = types.StringValue(strconv.FormatInt(createdRelease.GetID(), 10))
	if resp2 != nil {
		data.Etag = types.StringValue(resp2.Header.Get("ETag"))
	}

	r.populateResourceData(ctx, &data, createdRelease)

	tflog.Debug(ctx, "created GitHub release", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repoName,
		"tag_name":   tagName,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubReleaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubReleaseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubRelease(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubReleaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state githubReleaseResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := plan.Repository.ValueString()
	releaseID, err := strconv.ParseInt(plan.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Release ID",
			fmt.Sprintf("Unable to parse release ID '%s': %s", plan.ID.ValueString(), err.Error()),
		)
		return
	}

	release := &github.RepositoryRelease{
		TagName:              github.Ptr(plan.TagName.ValueString()),
		TargetCommitish:      github.Ptr(plan.TargetCommitish.ValueString()),
		Draft:                github.Ptr(plan.Draft.ValueBool()),
		Prerelease:           github.Ptr(plan.Prerelease.ValueBool()),
		GenerateReleaseNotes: github.Ptr(plan.GenerateReleaseNotes.ValueBool()),
	}

	if !plan.Body.IsNull() && !plan.Body.IsUnknown() {
		release.Body = github.Ptr(plan.Body.ValueString())
	}

	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		release.Name = github.Ptr(plan.Name.ValueString())
	}

	if !plan.DiscussionCategoryName.IsNull() && !plan.DiscussionCategoryName.IsUnknown() {
		release.DiscussionCategoryName = github.Ptr(plan.DiscussionCategoryName.ValueString())
	}

	updatedRelease, resp2, err := client.Repositories.EditRelease(ctx, owner, repoName, releaseID, release)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update GitHub Release",
			fmt.Sprintf("An unexpected error occurred when updating the GitHub release: %s", err.Error()),
		)
		return
	}

	// Update the data model
	if resp2 != nil {
		plan.Etag = types.StringValue(resp2.Header.Get("ETag"))
	}

	r.populateResourceData(ctx, &plan, updatedRelease)

	tflog.Debug(ctx, "updated GitHub release", map[string]interface{}{
		"id":         plan.ID.ValueString(),
		"repository": repoName,
		"tag_name":   plan.TagName.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *githubReleaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubReleaseResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()
	releaseID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Release ID",
			fmt.Sprintf("Unable to parse release ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, err = client.Repositories.DeleteRelease(ctx, owner, repoName, releaseID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete GitHub Release",
			fmt.Sprintf("An unexpected error occurred when deleting the GitHub release: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub release", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repoName,
	})
}

func (r *githubReleaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "<repository>:<release_id>"
	parts := strings.Split(req.ID, ":")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Invalid ID specified. Supplied ID must be written as <repository>:<release_id>. Got: %q", req.ID),
		)
		return
	}

	repoName := parts[0]
	releaseIDStr := parts[1]

	releaseID, err := strconv.ParseInt(releaseIDStr, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Release ID",
			fmt.Sprintf("Unable to parse release ID '%s': %s", releaseIDStr, err.Error()),
		)
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	// Verify the repository exists
	repository, _, err := client.Repositories.Get(ctx, owner, repoName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import GitHub Release",
			fmt.Sprintf("Unable to get repository for import: %s", err.Error()),
		)
		return
	}

	// Verify the release exists
	release, _, err := client.Repositories.GetRelease(ctx, owner, *repository.Name, releaseID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Import GitHub Release",
			fmt.Sprintf("Unable to read release for import: %s", err.Error()),
		)
		return
	}

	data := &githubReleaseResourceModel{
		ID:         types.StringValue(strconv.FormatInt(release.GetID(), 10)),
		Repository: types.StringValue(*repository.Name),
	}

	r.populateResourceData(ctx, data, release)

	tflog.Debug(ctx, "imported GitHub release", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": *repository.Name,
		"tag_name":   release.GetTagName(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubReleaseResource) readGithubRelease(ctx context.Context, data *githubReleaseResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	repository := data.Repository.ValueString()
	releaseID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		diags.AddError(
			"Unable to Parse Release ID",
			fmt.Sprintf("Unable to parse release ID '%s': %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	release, resp, err := client.Repositories.GetRelease(ctx, owner, repository, releaseID)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing release from state because it no longer exists in GitHub", map[string]interface{}{
					"owner":      owner,
					"repository": repository,
					"release_id": releaseID,
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read GitHub Release",
			fmt.Sprintf("An unexpected error occurred when reading the GitHub release: %s", err.Error()),
		)
		return
	}

	if resp != nil {
		data.Etag = types.StringValue(resp.Header.Get("ETag"))
	}

	r.populateResourceData(ctx, data, release)

	tflog.Debug(ctx, "successfully read GitHub release", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repository,
		"tag_name":   release.GetTagName(),
	})
}

func (r *githubReleaseResource) populateResourceData(ctx context.Context, data *githubReleaseResourceModel, release *github.RepositoryRelease) {
	data.ReleaseID = types.Int64Value(release.GetID())
	data.NodeID = types.StringValue(release.GetNodeID())
	data.TagName = types.StringValue(release.GetTagName())
	data.TargetCommitish = types.StringValue(release.GetTargetCommitish())

	if release.Name != nil {
		data.Name = types.StringValue(release.GetName())
	} else {
		data.Name = types.StringValue("")
	}

	if release.Body != nil {
		data.Body = types.StringValue(release.GetBody())
	} else {
		data.Body = types.StringValue("")
	}

	data.Draft = types.BoolValue(release.GetDraft())
	data.Prerelease = types.BoolValue(release.GetPrerelease())
	data.GenerateReleaseNotes = types.BoolValue(release.GetGenerateReleaseNotes())

	if release.DiscussionCategoryName != nil {
		data.DiscussionCategoryName = types.StringValue(release.GetDiscussionCategoryName())
	} else {
		data.DiscussionCategoryName = types.StringValue("")
	}

	data.CreatedAt = types.StringValue(release.GetCreatedAt().String())
	data.PublishedAt = types.StringValue(release.GetPublishedAt().String())
	data.URL = types.StringValue(release.GetURL())
	data.HTMLURL = types.StringValue(release.GetHTMLURL())
	data.AssetsURL = types.StringValue(release.GetAssetsURL())
	data.UploadURL = types.StringValue(release.GetUploadURL())
	data.ZipballURL = types.StringValue(release.GetZipballURL())
	data.TarballURL = types.StringValue(release.GetTarballURL())
}
