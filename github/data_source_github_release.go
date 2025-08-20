package github

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/google/go-github/v74/github"
)

var (
	_ datasource.DataSource              = &githubReleaseDataSource{}
	_ datasource.DataSourceWithConfigure = &githubReleaseDataSource{}
)

type githubReleaseDataSource struct {
	client *Owner
}

type githubReleaseDataSourceModel struct {
	ID              types.String                        `tfsdk:"id"`
	Repository      types.String                        `tfsdk:"repository"`
	Owner           types.String                        `tfsdk:"owner"`
	RetrieveBy      types.String                        `tfsdk:"retrieve_by"`
	ReleaseTag      types.String                        `tfsdk:"release_tag"`
	ReleaseID       types.Int64                         `tfsdk:"release_id"`
	TargetCommitish types.String                        `tfsdk:"target_commitish"`
	Name            types.String                        `tfsdk:"name"`
	Body            types.String                        `tfsdk:"body"`
	Draft           types.Bool                          `tfsdk:"draft"`
	Prerelease      types.Bool                          `tfsdk:"prerelease"`
	CreatedAt       types.String                        `tfsdk:"created_at"`
	PublishedAt     types.String                        `tfsdk:"published_at"`
	URL             types.String                        `tfsdk:"url"`
	HTMLURL         types.String                        `tfsdk:"html_url"`
	AssetsURL       types.String                        `tfsdk:"assets_url"`
	AssertsURL      types.String                        `tfsdk:"asserts_url"`
	UploadURL       types.String                        `tfsdk:"upload_url"`
	ZipballURL      types.String                        `tfsdk:"zipball_url"`
	TarballURL      types.String                        `tfsdk:"tarball_url"`
	Assets          []githubReleaseDataSourceAssetModel `tfsdk:"assets"`
}

type githubReleaseDataSourceAssetModel struct {
	ID                 types.Int64  `tfsdk:"id"`
	URL                types.String `tfsdk:"url"`
	NodeID             types.String `tfsdk:"node_id"`
	Name               types.String `tfsdk:"name"`
	Label              types.String `tfsdk:"label"`
	ContentType        types.String `tfsdk:"content_type"`
	Size               types.Int64  `tfsdk:"size"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
	BrowserDownloadURL types.String `tfsdk:"browser_download_url"`
}

func NewGithubReleaseDataSource() datasource.DataSource {
	return &githubReleaseDataSource{}
}

func (d *githubReleaseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release"
}

func (d *githubReleaseDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub release.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the release.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"owner": schema.StringAttribute{
				Description: "The repository owner.",
				Required:    true,
			},
			"retrieve_by": schema.StringAttribute{
				Description: "How to retrieve the release. Must be one of `latest`, `id`, or `tag`.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("latest", "id", "tag"),
				},
			},
			"release_tag": schema.StringAttribute{
				Description: "The tag of the release to retrieve. Required if `retrieve_by = \"tag\"`.",
				Optional:    true,
			},
			"release_id": schema.Int64Attribute{
				Description: "The ID of the release to retrieve. Required if `retrieve_by = \"id\"`.",
				Optional:    true,
			},
			"target_commitish": schema.StringAttribute{
				Description: "The target commitish of the release.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the release.",
				Computed:    true,
			},
			"body": schema.StringAttribute{
				Description: "The body of the release.",
				Computed:    true,
			},
			"draft": schema.BoolAttribute{
				Description: "Whether the release is a draft.",
				Computed:    true,
			},
			"prerelease": schema.BoolAttribute{
				Description: "Whether the release is a prerelease.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The creation date of the release.",
				Computed:    true,
			},
			"published_at": schema.StringAttribute{
				Description: "The publication date of the release.",
				Computed:    true,
			},
			"url": schema.StringAttribute{
				Description: "The URL of the release.",
				Computed:    true,
			},
			"html_url": schema.StringAttribute{
				Description: "The HTML URL of the release.",
				Computed:    true,
			},
			"assets_url": schema.StringAttribute{
				Description: "The URL of the release assets.",
				Computed:    true,
			},
			"asserts_url": schema.StringAttribute{
				Description:        "The URL of the release assets. Deprecated, use `assets_url` instead.",
				DeprecationMessage: "use assets_url instead",
				Computed:           true,
			},
			"upload_url": schema.StringAttribute{
				Description: "The upload URL of the release.",
				Computed:    true,
			},
			"zipball_url": schema.StringAttribute{
				Description: "The zipball URL of the release.",
				Computed:    true,
			},
			"tarball_url": schema.StringAttribute{
				Description: "The tarball URL of the release.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"assets": schema.ListNestedBlock{
				Description: "The release assets.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Description: "The ID of the asset.",
							Computed:    true,
						},
						"url": schema.StringAttribute{
							Description: "The URL of the asset.",
							Computed:    true,
						},
						"node_id": schema.StringAttribute{
							Description: "The Node ID of the asset.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the asset.",
							Computed:    true,
						},
						"label": schema.StringAttribute{
							Description: "The label of the asset.",
							Computed:    true,
						},
						"content_type": schema.StringAttribute{
							Description: "The content type of the asset.",
							Computed:    true,
						},
						"size": schema.Int64Attribute{
							Description: "The size of the asset in bytes.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The creation date of the asset.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "The last update date of the asset.",
							Computed:    true,
						},
						"browser_download_url": schema.StringAttribute{
							Description: "The browser download URL of the asset.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubReleaseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *githubReleaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubReleaseDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repository := data.Repository.ValueString()
	owner := data.Owner.ValueString()
	retrieveBy := strings.ToLower(data.RetrieveBy.ValueString())

	tflog.Debug(ctx, "Reading GitHub release", map[string]interface{}{
		"repository":  repository,
		"owner":       owner,
		"retrieve_by": retrieveBy,
	})

	var err error
	var release *github.RepositoryRelease

	switch retrieveBy {
	case "latest":
		release, _, err = d.client.V3Client().Repositories.GetLatestRelease(ctx, owner, repository)
	case "id":
		if data.ReleaseID.IsNull() || data.ReleaseID.IsUnknown() {
			resp.Diagnostics.AddError(
				"Missing Required Field",
				"`release_id` must be set when `retrieve_by` = `id`",
			)
			return
		}
		releaseID := data.ReleaseID.ValueInt64()
		release, _, err = d.client.V3Client().Repositories.GetRelease(ctx, owner, repository, releaseID)
	case "tag":
		if data.ReleaseTag.IsNull() || data.ReleaseTag.IsUnknown() || data.ReleaseTag.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing Required Field",
				"`release_tag` must be set when `retrieve_by` = `tag`",
			)
			return
		}
		tag := data.ReleaseTag.ValueString()
		release, _, err = d.client.V3Client().Repositories.GetReleaseByTag(ctx, owner, repository, tag)
	default:
		resp.Diagnostics.AddError(
			"Invalid Value",
			"one of: `latest`, `id`, `tag` must be set for `retrieve_by`",
		)
		return
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Release",
			fmt.Sprintf("An unexpected error occurred while reading the GitHub release: %s", err.Error()),
		)
		return
	}

	// Set the computed values
	data.ID = types.StringValue(strconv.FormatInt(release.GetID(), 10))
	data.ReleaseTag = types.StringValue(release.GetTagName())
	data.TargetCommitish = types.StringValue(release.GetTargetCommitish())
	data.Name = types.StringValue(release.GetName())
	data.Body = types.StringValue(release.GetBody())
	data.Draft = types.BoolValue(release.GetDraft())
	data.Prerelease = types.BoolValue(release.GetPrerelease())
	data.CreatedAt = types.StringValue(release.GetCreatedAt().String())
	data.PublishedAt = types.StringValue(release.GetPublishedAt().String())
	data.URL = types.StringValue(release.GetURL())
	data.HTMLURL = types.StringValue(release.GetHTMLURL())
	data.AssetsURL = types.StringValue(release.GetAssetsURL())
	data.AssertsURL = types.StringValue(release.GetAssetsURL()) // Deprecated, original version of assets_url
	data.UploadURL = types.StringValue(release.GetUploadURL())
	data.ZipballURL = types.StringValue(release.GetZipballURL())
	data.TarballURL = types.StringValue(release.GetTarballURL())

	// Set release assets
	assets := make([]githubReleaseDataSourceAssetModel, 0, len(release.Assets))
	for _, releaseAsset := range release.Assets {
		if releaseAsset == nil {
			continue
		}

		asset := githubReleaseDataSourceAssetModel{
			ID:                 types.Int64Value(releaseAsset.GetID()),
			URL:                types.StringValue(releaseAsset.GetURL()),
			NodeID:             types.StringValue(releaseAsset.GetNodeID()),
			Name:               types.StringValue(releaseAsset.GetName()),
			Label:              types.StringValue(releaseAsset.GetLabel()),
			ContentType:        types.StringValue(releaseAsset.GetContentType()),
			Size:               types.Int64Value(int64(releaseAsset.GetSize())),
			CreatedAt:          types.StringValue(releaseAsset.GetCreatedAt().String()),
			UpdatedAt:          types.StringValue(releaseAsset.GetUpdatedAt().String()),
			BrowserDownloadURL: types.StringValue(releaseAsset.GetBrowserDownloadURL()),
		}
		assets = append(assets, asset)
	}
	data.Assets = assets

	tflog.Debug(ctx, "Successfully read GitHub release", map[string]interface{}{
		"repository": repository,
		"owner":      owner,
		"id":         strconv.FormatInt(release.GetID(), 10),
		"tag":        data.ReleaseTag.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
