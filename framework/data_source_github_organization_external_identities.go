package framework

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/shurcooL/githubv4"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ datasource.DataSource              = &githubOrganizationExternalIdentitiesDataSource{}
	_ datasource.DataSourceWithConfigure = &githubOrganizationExternalIdentitiesDataSource{}
)

type githubOrganizationExternalIdentitiesDataSource struct {
	client *githubpkg.Owner
}

type githubOrganizationExternalIdentityModel struct {
	Login        types.String `tfsdk:"login"`
	SamlIdentity types.Map    `tfsdk:"saml_identity"`
	ScimIdentity types.Map    `tfsdk:"scim_identity"`
}

type githubOrganizationExternalIdentitiesDataSourceModel struct {
	ID         types.String                              `tfsdk:"id"`
	Identities []githubOrganizationExternalIdentityModel `tfsdk:"identities"`
}

// GraphQL types for API response - using the same structure as SDKv2
type ExternalIdentities struct {
	Edges []struct {
		Node struct {
			User struct {
				Login githubv4.String
			}
			SamlIdentity struct {
				NameId     githubv4.String
				Username   githubv4.String
				GivenName  githubv4.String
				FamilyName githubv4.String
			}
			ScimIdentity struct {
				Username   githubv4.String
				GivenName  githubv4.String
				FamilyName githubv4.String
			}
		}
	}
	PageInfo struct {
		EndCursor   githubv4.String
		HasNextPage bool
	}
}

func NewGithubOrganizationExternalIdentitiesDataSource() datasource.DataSource {
	return &githubOrganizationExternalIdentitiesDataSource{}
}

func (d *githubOrganizationExternalIdentitiesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_external_identities"
}

func (d *githubOrganizationExternalIdentitiesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a list of organization members and their SAML linked external identity NameID",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this data source.",
				Computed:    true,
			},
			"identities": schema.ListNestedAttribute{
				Description: "List of organization members and their external identities.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"login": schema.StringAttribute{
							Description: "The GitHub username of the organization member.",
							Computed:    true,
						},
						"saml_identity": schema.MapAttribute{
							Description: "SAML identity information for the user.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"scim_identity": schema.MapAttribute{
							Description: "SCIM identity information for the user.",
							ElementType: types.StringType,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubOrganizationExternalIdentitiesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*githubpkg.Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *githubOrganizationExternalIdentitiesDataSource) checkOrganization() error {
	if !d.client.IsOrganization {
		return fmt.Errorf("this data source can only be used with organization accounts")
	}
	return nil
}

func (d *githubOrganizationExternalIdentitiesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubOrganizationExternalIdentitiesDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization configuration
	err := d.checkOrganization()
	if err != nil {
		resp.Diagnostics.AddError(
			"Organization Required",
			err.Error(),
		)
		return
	}

	orgName := d.client.Name()
	client4 := d.client.V4Client()

	tflog.Debug(ctx, "Reading GitHub organization external identities", map[string]interface{}{
		"organization": orgName,
	})

	// GraphQL query structure - same as SDKv2
	var query struct {
		Organization struct {
			SamlIdentityProvider struct {
				ExternalIdentities `graphql:"externalIdentities(first: 100, after: $after)"`
			}
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login": githubv4.String(orgName),
		"after": (*githubv4.String)(nil),
	}

	var identityModels []githubOrganizationExternalIdentityModel

	// Paginate through all external identities
	for {
		err := client4.Query(ctx, &query, variables)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Query GitHub Organization External Identities",
				fmt.Sprintf("An unexpected error occurred while querying GitHub organization external identities: %s", err.Error()),
			)
			return
		}

		for _, edge := range query.Organization.SamlIdentityProvider.Edges {
			identityModel := githubOrganizationExternalIdentityModel{}

			// Set login
			identityModel.Login = types.StringValue(string(edge.Node.User.Login))

			// Handle SAML identity
			if edge.Node.SamlIdentity.NameId != "" {
				samlIdentityMap := map[string]attr.Value{
					"name_id":     types.StringValue(string(edge.Node.SamlIdentity.NameId)),
					"username":    types.StringValue(string(edge.Node.SamlIdentity.Username)),
					"given_name":  types.StringValue(string(edge.Node.SamlIdentity.GivenName)),
					"family_name": types.StringValue(string(edge.Node.SamlIdentity.FamilyName)),
				}
				var diags = resp.Diagnostics
				identityModel.SamlIdentity, diags = types.MapValue(types.StringType, samlIdentityMap)
				resp.Diagnostics.Append(diags...)
			} else {
				identityModel.SamlIdentity = types.MapNull(types.StringType)
			}

			// Handle SCIM identity
			if edge.Node.ScimIdentity.Username != "" {
				scimIdentityMap := map[string]attr.Value{
					"username":    types.StringValue(string(edge.Node.ScimIdentity.Username)),
					"given_name":  types.StringValue(string(edge.Node.ScimIdentity.GivenName)),
					"family_name": types.StringValue(string(edge.Node.ScimIdentity.FamilyName)),
				}
				var diags = resp.Diagnostics
				identityModel.ScimIdentity, diags = types.MapValue(types.StringType, scimIdentityMap)
				resp.Diagnostics.Append(diags...)
			} else {
				identityModel.ScimIdentity = types.MapNull(types.StringType)
			}

			identityModels = append(identityModels, identityModel)
		}

		if !query.Organization.SamlIdentityProvider.PageInfo.HasNextPage {
			break
		}
		variables["after"] = githubv4.NewString(query.Organization.SamlIdentityProvider.PageInfo.EndCursor)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Set ID using organization name (same pattern as SDKv2)
	data.ID = types.StringValue(orgName)
	data.Identities = identityModels

	tflog.Debug(ctx, "Successfully read GitHub organization external identities", map[string]interface{}{
		"organization":     orgName,
		"identities_count": len(identityModels),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
