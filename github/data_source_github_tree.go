package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/google/go-github/v74/github"
)

var (
	_ datasource.DataSource              = &githubTreeDataSource{}
	_ datasource.DataSourceWithConfigure = &githubTreeDataSource{}
)

type githubTreeDataSource struct {
	client *Owner
}

type githubTreeDataSourceModel struct {
	Repository types.String                     `tfsdk:"repository"`
	TreeSha    types.String                     `tfsdk:"tree_sha"`
	Recursive  types.Bool                       `tfsdk:"recursive"`
	Entries    []githubTreeEntryDataSourceModel `tfsdk:"entries"`
}

type githubTreeEntryDataSourceModel struct {
	Path types.String `tfsdk:"path"`
	Mode types.String `tfsdk:"mode"`
	Type types.String `tfsdk:"type"`
	Size types.Int64  `tfsdk:"size"`
	Sha  types.String `tfsdk:"sha"`
}

func NewGithubTreeDataSource() datasource.DataSource {
	return &githubTreeDataSource{}
}

func (d *githubTreeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tree"
}

func (d *githubTreeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns a single tree using the SHA1 value for that tree.",
		Attributes: map[string]schema.Attribute{
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"tree_sha": schema.StringAttribute{
				Description: "The SHA1 value for the tree.",
				Required:    true,
			},
			"recursive": schema.BoolAttribute{
				Description: "Whether to fetch the tree recursively. Defaults to false.",
				Optional:    true,
			},
			"entries": schema.ListNestedAttribute{
				Description: "The entries in the tree.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Description: "The path of the entry.",
							Computed:    true,
						},
						"mode": schema.StringAttribute{
							Description: "The mode of the entry.",
							Computed:    true,
						},
						"type": schema.StringAttribute{
							Description: "The type of the entry.",
							Computed:    true,
						},
						"size": schema.Int64Attribute{
							Description: "The size of the entry.",
							Computed:    true,
						},
						"sha": schema.StringAttribute{
							Description: "The SHA of the entry.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubTreeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubTreeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubTreeDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoName := data.Repository.ValueString()
	treeSha := data.TreeSha.ValueString()

	// Handle recursive option - default to false if not set
	recursive := false
	if !data.Recursive.IsNull() && !data.Recursive.IsUnknown() {
		recursive = data.Recursive.ValueBool()
	}

	tflog.Debug(ctx, "Reading GitHub tree", map[string]any{
		"repository": repoName,
		"tree_sha":   treeSha,
		"recursive":  recursive,
	})

	tree, _, err := d.client.V3Client().Git.GetTree(ctx, d.client.Name(), repoName, treeSha, recursive)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				tflog.Debug(ctx, "Tree not found", map[string]any{
					"repository": repoName,
					"tree_sha":   treeSha,
				})
				resp.Diagnostics.AddError(
					"GitHub Tree Not Found",
					fmt.Sprintf("The tree %s was not found in repository %s/%s", treeSha, d.client.Name(), repoName),
				)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Tree",
			fmt.Sprintf("An unexpected error occurred while reading the GitHub tree %s in repository %s/%s: %s", treeSha, d.client.Name(), repoName, err.Error()),
		)
		return
	}

	// Convert tree entries to model
	entries := make([]githubTreeEntryDataSourceModel, 0, len(tree.Entries))
	for _, entry := range tree.Entries {
		entryModel := githubTreeEntryDataSourceModel{
			Path: types.StringValue(stringValue(entry.Path)),
			Mode: types.StringValue(stringValue(entry.Mode)),
			Type: types.StringValue(stringValue(entry.Type)),
			Sha:  types.StringValue(stringValue(entry.SHA)),
		}

		// Size can be nil for some entry types
		if entry.Size != nil {
			entryModel.Size = types.Int64Value(int64(*entry.Size))
		} else {
			entryModel.Size = types.Int64Null()
		}

		entries = append(entries, entryModel)
	}

	// Set the ID to the tree SHA
	data.Entries = entries

	// Set recursive value if it wasn't provided (maintain computed state)
	if data.Recursive.IsNull() {
		data.Recursive = types.BoolValue(false)
	}

	tflog.Debug(ctx, "Successfully read GitHub tree", map[string]any{
		"repository":  repoName,
		"tree_sha":    treeSha,
		"recursive":   recursive,
		"entry_count": len(entries),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// stringValue safely dereferences a *string pointer, returning empty string if nil
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
