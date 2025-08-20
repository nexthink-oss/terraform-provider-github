package framework

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ datasource.DataSource              = &githubMembershipDataSource{}
	_ datasource.DataSourceWithConfigure = &githubMembershipDataSource{}
)

type githubMembershipDataSource struct {
	client *githubpkg.Owner
}

type githubMembershipDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Username     types.String `tfsdk:"username"`
	Organization types.String `tfsdk:"organization"`
	Role         types.String `tfsdk:"role"`
	Etag         types.String `tfsdk:"etag"`
	State        types.String `tfsdk:"state"`
}

func NewGithubMembershipDataSource() datasource.DataSource {
	return &githubMembershipDataSource{}
}

func (d *githubMembershipDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_membership"
}

func (d *githubMembershipDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on user membership in an organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the membership.",
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username to lookup membership for.",
				Required:    true,
			},
			"organization": schema.StringAttribute{
				Description: "The organization to check membership in. If not specified, the provider's default organization is used.",
				Optional:    true,
			},
			"role": schema.StringAttribute{
				Description: "The role of the user in the organization.",
				Computed:    true,
			},
			"etag": schema.StringAttribute{
				Description: "The ETag of the membership response.",
				Computed:    true,
			},
			"state": schema.StringAttribute{
				Description: "The state of the user's membership in the organization.",
				Computed:    true,
			},
		},
	}
}

func (d *githubMembershipDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubMembershipDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubMembershipDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := data.Username.ValueString()

	// Determine organization to use
	orgName := d.client.Name()
	if !data.Organization.IsNull() && !data.Organization.IsUnknown() {
		orgName = data.Organization.ValueString()
	}

	tflog.Debug(ctx, "Reading GitHub membership", map[string]interface{}{
		"username":     username,
		"organization": orgName,
	})

	membership, resp_api, err := d.client.V3Client().Organizations.GetOrgMembership(ctx, username, orgName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub Membership",
			fmt.Sprintf("An unexpected error occurred while reading membership for user %s in organization %s: %s", username, orgName, err.Error()),
		)
		return
	}

	// Create ID using organization:username format
	membershipID := fmt.Sprintf("%s:%s", membership.GetOrganization().GetLogin(), membership.GetUser().GetLogin())

	// Set values
	data.ID = types.StringValue(membershipID)
	data.Username = types.StringValue(membership.GetUser().GetLogin())
	data.Role = types.StringValue(membership.GetRole())
	data.Etag = types.StringValue(resp_api.Header.Get("ETag"))
	data.State = types.StringValue(membership.GetState())

	tflog.Debug(ctx, "Successfully read GitHub membership", map[string]interface{}{
		"username":     username,
		"organization": orgName,
		"id":           data.ID.ValueString(),
		"role":         data.Role.ValueString(),
		"state":        data.State.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
