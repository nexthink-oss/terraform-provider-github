package github

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/shurcooL/githubv4"
)

var (
	_ datasource.DataSource              = &githubUserExternalIdentityDataSource{}
	_ datasource.DataSourceWithConfigure = &githubUserExternalIdentityDataSource{}
)

type githubUserExternalIdentityDataSource struct {
	client *Owner
}

type githubUserExternalIdentityDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Username     types.String `tfsdk:"username"`
	Login        types.String `tfsdk:"login"`
	SamlIdentity types.Map    `tfsdk:"saml_identity"`
	ScimIdentity types.Map    `tfsdk:"scim_identity"`
}

func NewGithubUserExternalIdentityDataSource() datasource.DataSource {
	return &githubUserExternalIdentityDataSource{}
}

func (d *githubUserExternalIdentityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_external_identity"
}

func (d *githubUserExternalIdentityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a specific organization member's SAML/SCIM linked external identity",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the user external identity (format: orgName/username).",
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username to lookup the external identity for.",
				Required:    true,
			},
			"login": schema.StringAttribute{
				Description: "The user's GitHub login.",
				Computed:    true,
			},
			"saml_identity": schema.MapAttribute{
				Description: "The user's SAML identity information.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"scim_identity": schema.MapAttribute{
				Description: "The user's SCIM identity information.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *githubUserExternalIdentityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubUserExternalIdentityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubUserExternalIdentityDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := data.Username.ValueString()

	tflog.Debug(ctx, "Reading GitHub user external identity", map[string]interface{}{
		"username": username,
	})

	client := d.client.V4Client()
	orgName := d.client.Name()

	var query struct {
		Organization struct {
			SamlIdentityProvider struct {
				ExternalIdentities ExternalIdentities `graphql:"externalIdentities(first: 1, login:$username)"` // There should only ever be one external identity configured
			}
		} `graphql:"organization(login: $orgName)"`
	}

	variables := map[string]any{
		"orgName":  githubv4.String(orgName),
		"username": githubv4.String(username),
	}

	err := client.Query(ctx, &query, variables)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub User External Identity",
			fmt.Sprintf("An unexpected error occurred while reading the GitHub user external identity for %s: %s", username, err.Error()),
		)
		return
	}

	if len(query.Organization.SamlIdentityProvider.ExternalIdentities.Edges) == 0 {
		resp.Diagnostics.AddError(
			"GitHub User External Identity Not Found",
			fmt.Sprintf("There was no external identity found for username %q in Organization %q", username, orgName),
		)
		return
	}

	externalIdentityNode := query.Organization.SamlIdentityProvider.ExternalIdentities.Edges[0].Node // There should only be one user in this list

	// Create SAML identity map
	samlIdentityMap := map[string]string{
		"family_name": string(externalIdentityNode.SamlIdentity.FamilyName),
		"given_name":  string(externalIdentityNode.SamlIdentity.GivenName),
		"name_id":     string(externalIdentityNode.SamlIdentity.NameId),
		"username":    string(externalIdentityNode.SamlIdentity.Username),
	}

	// Create SCIM identity map
	scimIdentityMap := map[string]string{
		"family_name": string(externalIdentityNode.ScimIdentity.FamilyName),
		"given_name":  string(externalIdentityNode.ScimIdentity.GivenName),
		"username":    string(externalIdentityNode.ScimIdentity.Username),
	}

	login := string(externalIdentityNode.User.Login)

	// Convert maps to Framework map values
	samlIdentityValue, diags := types.MapValueFrom(ctx, types.StringType, samlIdentityMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	scimIdentityValue, diags := types.MapValueFrom(ctx, types.StringType, scimIdentityMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set values
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", orgName, username))
	data.Login = types.StringValue(login)
	data.SamlIdentity = samlIdentityValue
	data.ScimIdentity = scimIdentityValue

	tflog.Debug(ctx, "Successfully read GitHub user external identity", map[string]interface{}{
		"username": username,
		"login":    login,
		"id":       data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
