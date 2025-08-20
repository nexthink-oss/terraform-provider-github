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
	_ datasource.DataSource              = &githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSource{}
	_ datasource.DataSourceWithConfigure = &githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSource{}
)

type githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSource struct {
	client *Owner
}

type githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	IncludeClaimKeys types.List   `tfsdk:"include_claim_keys"`
}

func NewGithubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSource() datasource.DataSource {
	return &githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSource{}
}

func (d *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_actions_organization_oidc_subject_claim_customization_template"
}

func (d *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a GitHub Actions organization OpenID Connect customization template",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the organization.",
				Computed:    true,
			},
			"include_claim_keys": schema.ListAttribute{
				Description: "A list of OpenID Connect claims.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			"Expected *github.Owner, got something else.",
		)
		return
	}

	d.client = client
}

func (d *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSource) checkOrganization() error {
	if !d.client.IsOrganization {
		return fmt.Errorf("this data source can only be used with organization accounts")
	}
	return nil
}

func (d *githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check that this is an organization account
	err := d.checkOrganization()
	if err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			err.Error(),
		)
		return
	}

	owner := d.client.Name()

	tflog.Debug(ctx, "Reading GitHub Actions organization OIDC subject claim customization template", map[string]interface{}{
		"owner": owner,
	})

	template, _, err := d.client.V3Client().Actions.GetOrgOIDCSubjectClaimCustomTemplate(ctx, owner)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Actions Organization OIDC Subject Claim Customization Template",
			err.Error(),
		)
		return
	}

	data.ID = types.StringValue(owner)

	// Convert slice to types.List
	includeClaimKeysElements := make([]types.String, 0, len(template.IncludeClaimKeys))
	for _, key := range template.IncludeClaimKeys {
		includeClaimKeysElements = append(includeClaimKeysElements, types.StringValue(key))
	}

	includeClaimKeysList, diags := types.ListValueFrom(ctx, types.StringType, includeClaimKeysElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.IncludeClaimKeys = includeClaimKeysList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
