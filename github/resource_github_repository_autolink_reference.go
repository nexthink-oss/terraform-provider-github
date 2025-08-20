package github

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &githubRepositoryAutolinkReferenceResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryAutolinkReferenceResource{}
	_ resource.ResourceWithImportState = &githubRepositoryAutolinkReferenceResource{}
)

type githubRepositoryAutolinkReferenceResource struct {
	client *Owner
}

type githubRepositoryAutolinkReferenceResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Repository        types.String `tfsdk:"repository"`
	KeyPrefix         types.String `tfsdk:"key_prefix"`
	TargetURLTemplate types.String `tfsdk:"target_url_template"`
	IsAlphanumeric    types.Bool   `tfsdk:"is_alphanumeric"`
	Etag              types.String `tfsdk:"etag"`
}

func NewGithubRepositoryAutolinkReferenceResource() resource.Resource {
	return &githubRepositoryAutolinkReferenceResource{}
}

func (r *githubRepositoryAutolinkReferenceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_autolink_reference"
}

func (r *githubRepositoryAutolinkReferenceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages autolink references for a single repository",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the autolink reference.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The repository name",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_prefix": schema.StringAttribute{
				Description: "This prefix appended by a number will generate a link any time it is found in an issue, pull request, or commit",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_url_template": schema.StringAttribute{
				Description: "The template of the target URL used for the links; must be a valid URL and contain `<num>` for the reference number",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^http[s]?:\/\/[a-z0-9-.]*(:[0-9]+)?\/.*?<num>.*?$`),
						"must be a valid URL and contain <num> token",
					),
				},
			},
			"is_alphanumeric": schema.BoolAttribute{
				Description: "Whether this autolink reference matches alphanumeric characters. If false, this autolink reference only matches numeric characters.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"etag": schema.StringAttribute{
				Description: "An etag representing the autolink reference object.",
				Computed:    true,
			},
		},
	}
}

func (r *githubRepositoryAutolinkReferenceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *githubRepositoryAutolinkReferenceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubRepositoryAutolinkReferenceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()
	keyPrefix := data.KeyPrefix.ValueString()
	targetURLTemplate := data.TargetURLTemplate.ValueString()
	isAlphanumeric := data.IsAlphanumeric.ValueBool()

	opts := &github.AutolinkOptions{
		KeyPrefix:      &keyPrefix,
		URLTemplate:    &targetURLTemplate,
		IsAlphanumeric: &isAlphanumeric,
	}

	autolinkRef, _, err := client.Repositories.AddAutolink(ctx, owner, repoName, opts)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Repository Autolink Reference",
			fmt.Sprintf("An unexpected error occurred when creating the repository autolink reference: %s", err.Error()),
		)
		return
	}

	data.ID = types.StringValue(strconv.FormatInt(autolinkRef.GetID(), 10))

	tflog.Debug(ctx, "created GitHub repository autolink reference", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repoName,
	})

	// Read the created resource to populate all computed fields
	r.readGithubRepositoryAutolinkReference(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryAutolinkReferenceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubRepositoryAutolinkReferenceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readGithubRepositoryAutolinkReference(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryAutolinkReferenceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// This resource only supports replacement updates since all attributes are ForceNew
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"All attributes of github_repository_autolink_reference require replacement. This should not be called.",
	)
}

func (r *githubRepositoryAutolinkReferenceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubRepositoryAutolinkReferenceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()
	autolinkRefID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Autolink Reference ID",
			fmt.Sprintf("Unexpected ID format (%q), expected numerical ID: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, err = client.Repositories.DeleteAutolink(ctx, owner, repoName, autolinkRefID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Repository Autolink Reference",
			fmt.Sprintf("An unexpected error occurred when deleting the repository autolink reference: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "deleted GitHub repository autolink reference", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repoName,
	})
}

func (r *githubRepositoryAutolinkReferenceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse the import ID in the format "<repository>/<autolink_reference_id>" or "<repository>/<key_prefix>"
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Invalid ID specified: supplied ID must be written as <repository>/<autolink_reference_id> or <repository>/<key_prefix>. Got: %q", req.ID),
		)
		return
	}

	repository := parts[0]
	id := parts[1]

	client := r.client.V3Client()
	owner := r.client.Name()

	// If the second part of the provided ID isn't an integer, assume that the
	// caller provided the key prefix for the autolink reference, and look up
	// the autolink by the key prefix.
	_, err := strconv.Atoi(id)
	if err != nil {
		autolink, err := r.getAutolinkByKeyPrefix(ctx, client, owner, repository, id)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Find Autolink Reference",
				fmt.Sprintf("Unable to find autolink reference by key prefix: %s", err.Error()),
			)
			return
		}
		id = strconv.FormatInt(*autolink.ID, 10)
	}

	data := &githubRepositoryAutolinkReferenceResourceModel{
		ID:         types.StringValue(id),
		Repository: types.StringValue(repository),
	}

	r.readGithubRepositoryAutolinkReference(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Helper functions

func (r *githubRepositoryAutolinkReferenceResource) readGithubRepositoryAutolinkReference(ctx context.Context, data *githubRepositoryAutolinkReferenceResourceModel, diags *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()
	autolinkRefID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		diags.AddError(
			"Invalid Autolink Reference ID",
			fmt.Sprintf("Unexpected ID format (%q), expected numerical ID: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	// Use context with etag if we have it for conditional requests
	requestCtx := ctx
	if !data.Etag.IsNull() && !data.Etag.IsUnknown() {
		requestCtx = context.WithValue(ctx, CtxEtag, data.Etag.ValueString())
	}

	autolinkRef, resp, err := client.Repositories.GetAutolink(requestCtx, owner, repoName, autolinkRefID)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Info(ctx, "removing autolink reference from state because it no longer exists in GitHub", map[string]interface{}{
					"owner":      owner,
					"repository": repoName,
					"id":         data.ID.ValueString(),
				})
				data.ID = types.StringNull()
				return
			}
		}
		diags.AddError(
			"Unable to Read Repository Autolink Reference",
			fmt.Sprintf("An unexpected error occurred when reading the repository autolink reference: %s", err.Error()),
		)
		return
	}

	// Set resource fields
	data.ID = types.StringValue(strconv.FormatInt(autolinkRef.GetID(), 10))
	data.Repository = types.StringValue(repoName)
	data.KeyPrefix = types.StringValue(autolinkRef.GetKeyPrefix())
	data.TargetURLTemplate = types.StringValue(autolinkRef.GetURLTemplate())
	data.IsAlphanumeric = types.BoolValue(autolinkRef.GetIsAlphanumeric())

	// Set etag if available
	if resp != nil {
		if etag := resp.Header.Get("ETag"); etag != "" {
			data.Etag = types.StringValue(etag)
		}
	}

	tflog.Debug(ctx, "successfully read GitHub repository autolink reference", map[string]interface{}{
		"id":         data.ID.ValueString(),
		"repository": repoName,
	})
}

// getAutolinkByKeyPrefix finds an autolink reference by its key prefix
func (r *githubRepositoryAutolinkReferenceResource) getAutolinkByKeyPrefix(ctx context.Context, client *github.Client, owner, repo, keyPrefix string) (*github.Autolink, error) {
	autolinks, err := r.listAutolinks(ctx, client, owner, repo)
	if err != nil {
		return nil, err
	}

	for _, autolink := range autolinks {
		if autolink.GetKeyPrefix() == keyPrefix {
			return autolink, nil
		}
	}

	return nil, fmt.Errorf("cannot find autolink reference %s in repo %s/%s", keyPrefix, owner, repo)
}

// listAutolinks returns all autolink references for the given repository
func (r *githubRepositoryAutolinkReferenceResource) listAutolinks(ctx context.Context, client *github.Client, owner, repo string) ([]*github.Autolink, error) {
	opts := &github.ListOptions{
		PerPage: maxPerPage, // GitHub's maximum per page
	}

	var allAutolinks []*github.Autolink
	for {
		autolinks, resp, err := client.Repositories.ListAutolinks(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("error listing autolinks for repository %s/%s: %w", owner, repo, err)
		}

		allAutolinks = append(allAutolinks, autolinks...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allAutolinks, nil
}
