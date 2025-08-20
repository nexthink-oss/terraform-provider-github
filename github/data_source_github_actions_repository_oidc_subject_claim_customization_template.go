package github

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

)

var (
	_ datasource.DataSource              = &githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSource{}
	_ datasource.DataSourceWithConfigure = &githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSource{}
)

type githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSource struct {
	client *Owner
}

type githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	UseDefault       types.Bool   `tfsdk:"use_default"`
	IncludeClaimKeys types.List   `tfsdk:"include_claim_keys"`
}

func NewGithubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSource() datasource.DataSource {
	return &githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSource{}
}

func (d *githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_repository_oidc_subject_claim_customization_template"
}

func (d *githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a GitHub Actions repository's OpenID Connect customization template",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this resource.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the repository.",
				Required:    true,
			},
			"use_default": schema.BoolAttribute{
				Description: "Whether the repository uses the default template.",
				Computed:    true,
			},
			"include_claim_keys": schema.ListAttribute{
				Description: "The list of included claim keys in the customization template.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repositoryName := data.Name.ValueString()
	owner := d.client.Name()

	tflog.Debug(ctx, "Reading GitHub Actions repository OIDC subject claim customization template", map[string]interface{}{
		"owner":      owner,
		"repository": repositoryName,
	})

	template, _, err := d.client.V3Client().Actions.GetRepoOIDCSubjectClaimCustomTemplate(ctx, owner, repositoryName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Actions Repository OIDC Subject Claim Customization Template",
			fmt.Sprintf("Error reading OIDC subject claim customization template for repository %s/%s: %s", owner, repositoryName, err.Error()),
		)
		return
	}

	// Set the computed attributes
	data.ID = types.StringValue(repositoryName)
	data.UseDefault = types.BoolValue(*template.UseDefault)

	// Convert include_claim_keys to framework list
	var includeClaimKeys []types.String
	for _, key := range template.IncludeClaimKeys {
		includeClaimKeys = append(includeClaimKeys, types.StringValue(key))
	}

	includeClaimKeysList, diags := types.ListValueFrom(ctx, types.StringType, includeClaimKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.IncludeClaimKeys = includeClaimKeysList

	tflog.Debug(ctx, "Successfully read GitHub Actions repository OIDC subject claim customization template", map[string]interface{}{
		"owner":              owner,
		"repository":         repositoryName,
		"use_default":        template.UseDefault,
		"include_claim_keys": template.IncludeClaimKeys,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
