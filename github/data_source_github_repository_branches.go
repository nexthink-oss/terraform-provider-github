package github

import (
	"context"
	"fmt"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubRepositoryBranchesDataSource{}
	_ datasource.DataSourceWithConfigure = &githubRepositoryBranchesDataSource{}
)

type githubRepositoryBranchesDataSource struct {
	client *Owner
}

type githubBranchModel struct {
	Name      types.String `tfsdk:"name"`
	Protected types.Bool   `tfsdk:"protected"`
}

type githubRepositoryBranchesDataSourceModel struct {
	ID                       types.String        `tfsdk:"id"`
	Repository               types.String        `tfsdk:"repository"`
	OnlyProtectedBranches    types.Bool          `tfsdk:"only_protected_branches"`
	OnlyNonProtectedBranches types.Bool          `tfsdk:"only_non_protected_branches"`
	Branches                 []githubBranchModel `tfsdk:"branches"`
}

func NewGithubRepositoryBranchesDataSource() datasource.DataSource {
	return &githubRepositoryBranchesDataSource{}
}

func (d *githubRepositoryBranchesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_branches"
}

func (d *githubRepositoryBranchesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub repository's branches.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"repository": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"only_protected_branches": schema.BoolAttribute{
				Description: "If true, only return protected branches.",
				Optional:    true,
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(
						path.MatchRoot("only_non_protected_branches"),
					),
				},
			},
			"only_non_protected_branches": schema.BoolAttribute{
				Description: "If true, only return non-protected branches.",
				Optional:    true,
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(
						path.MatchRoot("only_protected_branches"),
					),
				},
			},
			"branches": schema.ListNestedAttribute{
				Description: "The list of repository branches.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "The name of the branch.",
							Computed:    true,
						},
						"protected": schema.BoolAttribute{
							Description: "Whether the branch is protected.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubRepositoryBranchesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubRepositoryBranchesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubRepositoryBranchesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repository := data.Repository.ValueString()
	onlyProtectedBranches := data.OnlyProtectedBranches.ValueBool()
	onlyNonProtectedBranches := data.OnlyNonProtectedBranches.ValueBool()

	tflog.Debug(ctx, "Reading repository branches", map[string]any{
		"repository":                  repository,
		"only_protected_branches":     onlyProtectedBranches,
		"only_non_protected_branches": onlyNonProtectedBranches,
	})

	var listBranchOptions *github.BranchListOptions
	if onlyProtectedBranches {
		listBranchOptions = &github.BranchListOptions{
			Protected: &onlyProtectedBranches,
		}
	} else if onlyNonProtectedBranches {
		falseValue := false
		listBranchOptions = &github.BranchListOptions{
			Protected: &falseValue,
		}
	} else {
		listBranchOptions = &github.BranchListOptions{}
	}

	owner := d.client.Name()

	var allBranches []*github.Branch
	for {
		branches, response, err := d.client.V3Client().Repositories.ListBranches(ctx, owner, repository, listBranchOptions)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Reading Repository Branches",
				fmt.Sprintf("Could not read repository branches for %s/%s: %s", owner, repository, err),
			)
			return
		}

		allBranches = append(allBranches, branches...)

		if response.NextPage == 0 {
			break
		}
		listBranchOptions.Page = response.NextPage
	}

	// Convert to model
	branches := make([]githubBranchModel, len(allBranches))
	for i, branch := range allBranches {
		branches[i] = githubBranchModel{
			Name:      types.StringValue(branch.GetName()),
			Protected: types.BoolValue(branch.GetProtected()),
		}
	}

	data.ID = types.StringValue(fmt.Sprintf("%s/%s", owner, repository))
	data.Branches = branches

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
