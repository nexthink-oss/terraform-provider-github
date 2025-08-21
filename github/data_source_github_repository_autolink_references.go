package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubRepositoryAutolinkReferencesDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryAutolinkReferencesDataSource{}
)

type githubRepositoryAutolinkReferencesDataSource struct {
	client *Owner
}

type githubRepositoryAutolinkReferenceModel struct {
	KeyPrefix         types.String `tfsdk:"key_prefix"`
	TargetURLTemplate types.String `tfsdk:"target_url_template"`
	IsAlphanumeric    types.Bool   `tfsdk:"is_alphanumeric"`
}

type githubRepositoryAutolinkReferencesDataSourceModel struct {
	ID                 types.String                             `tfsdk:"id"`
	Repository         types.String                             `tfsdk:"repository"`
	AutolinkReferences []githubRepositoryAutolinkReferenceModel `tfsdk:"autolink_references"`
}

func NewGithubRepositoryAutolinkReferencesDataSource() datasource.DataSource {
	return &githubRepositoryAutolinkReferencesDataSource{}
}

func (d *githubRepositoryAutolinkReferencesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_autolink_references"
}

func (d *githubRepositoryAutolinkReferencesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get autolink references for a Github repository.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"autolink_references": schema.ListNestedAttribute{
				Description: "List of autolink references for the repository.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key_prefix": schema.StringAttribute{
							Description: "The key prefix for the autolink reference.",
							Computed:    true,
						},
						"target_url_template": schema.StringAttribute{
							Description: "The target URL template for the autolink reference.",
							Computed:    true,
						},
						"is_alphanumeric": schema.BoolAttribute{
							Description: "Whether the autolink reference is alphanumeric.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubRepositoryAutolinkReferencesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *githubRepositoryAutolinkReferencesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryAutolinkReferencesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := d.client.V3Client()
	owner := d.client.Name()
	repository := data.Repository.ValueString()

	tflog.Debug(ctx, "Reading GitHub repository autolink references", map[string]any{
		"owner":      owner,
		"repository": repository,
	})

	var listOptions *github.ListOptions
	var allAutolinks []githubRepositoryAutolinkReferenceModel

	for {
		autolinks, respGH, err := client.Repositories.ListAutolinks(ctx, owner, repository, listOptions)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Repository Autolink References",
				fmt.Sprintf("Unable to read autolink references for repository %s/%s: %s", owner, repository, err),
			)
			return
		}

		for _, autolink := range autolinks {
			autolinkModel := githubRepositoryAutolinkReferenceModel{
				KeyPrefix:         types.StringValue(autolink.GetKeyPrefix()),
				TargetURLTemplate: types.StringValue(autolink.GetURLTemplate()),
				IsAlphanumeric:    types.BoolValue(autolink.GetIsAlphanumeric()),
			}
			allAutolinks = append(allAutolinks, autolinkModel)
		}

		if respGH.NextPage == 0 {
			break
		}

		if listOptions == nil {
			listOptions = &github.ListOptions{}
		}
		listOptions.Page = respGH.NextPage
	}

	data.ID = types.StringValue(repository)
	data.AutolinkReferences = allAutolinks

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
