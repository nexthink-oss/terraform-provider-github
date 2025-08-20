package github

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/shurcooL/githubv4"
)

var (
	_ datasource.DataSource              = &githubOrganizationIpAllowListDataSource{}
	_ datasource.DataSourceWithConfigure = &githubOrganizationIpAllowListDataSource{}
)

type githubOrganizationIpAllowListDataSource struct {
	client *Owner
}

type githubOrganizationIpAllowListDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	IpAllowList types.List   `tfsdk:"ip_allow_list"`
}

func NewGithubOrganizationIpAllowListDataSource() datasource.DataSource {
	return &githubOrganizationIpAllowListDataSource{}
}

func (d *githubOrganizationIpAllowListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_ip_allow_list"
}

func (d *githubOrganizationIpAllowListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get the IP allow list of an organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the organization.",
				Computed:    true,
			},
			"ip_allow_list": schema.ListNestedAttribute{
				Description: "The list of IP allow list entries for the organization.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The ID of the IP allow list entry.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "The name of the IP allow list entry.",
							Computed:    true,
						},
						"allow_list_value": schema.StringAttribute{
							Description: "The IP address or CIDR block that is allowed.",
							Computed:    true,
						},
						"is_active": schema.BoolAttribute{
							Description: "Whether the IP allow list entry is active.",
							Computed:    true,
						},
						"created_at": schema.StringAttribute{
							Description: "The timestamp when the IP allow list entry was created.",
							Computed:    true,
						},
						"updated_at": schema.StringAttribute{
							Description: "The timestamp when the IP allow list entry was last updated.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *githubOrganizationIpAllowListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubOrganizationIpAllowListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubOrganizationIpAllowListDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if organization is configured
	if !d.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This data source can only be used with an organization.",
		)
		return
	}

	v4client := d.client.V4Client()
	orgName := d.client.Name()

	tflog.Debug(ctx, "Reading GitHub organization IP allow list", map[string]interface{}{
		"organization": orgName,
	})

	type PageInfo struct {
		StartCursor     githubv4.String
		EndCursor       githubv4.String
		HasNextPage     githubv4.Boolean
		HasPreviousPage githubv4.Boolean
	}

	type IpAllowListEntry struct {
		ID             githubv4.String
		Name           githubv4.String
		AllowListValue githubv4.String
		IsActive       githubv4.Boolean
		CreatedAt      githubv4.String
		UpdatedAt      githubv4.String
	}

	type IpAllowListEntries struct {
		Nodes      []IpAllowListEntry
		PageInfo   PageInfo
		TotalCount githubv4.Int
	}

	var query struct {
		Organization struct {
			ID                 githubv4.String
			IpAllowListEntries IpAllowListEntries `graphql:"ipAllowListEntries(first: 100, after: $entriesCursor)"`
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]any{
		"login":         githubv4.String(orgName),
		"entriesCursor": (*githubv4.String)(nil),
	}

	var ipAllowListEntries []IpAllowListEntry

	// Paginate through all IP allow list entries
	for {
		err := v4client.Query(ctx, &query, variables)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read GitHub Organization IP Allow List",
				fmt.Sprintf("An unexpected error occurred while reading the IP allow list for organization %s: %s", orgName, err.Error()),
			)
			return
		}

		ipAllowListEntries = append(ipAllowListEntries, query.Organization.IpAllowListEntries.Nodes...)
		if !query.Organization.IpAllowListEntries.PageInfo.HasNextPage {
			break
		}
		variables["entriesCursor"] = githubv4.NewString(query.Organization.IpAllowListEntries.PageInfo.EndCursor)
	}

	// Convert to Framework types
	ipAllowListElements := make([]attr.Value, 0, len(ipAllowListEntries))
	ipAllowListAttrTypes := map[string]attr.Type{
		"id":               types.StringType,
		"name":             types.StringType,
		"allow_list_value": types.StringType,
		"is_active":        types.BoolType,
		"created_at":       types.StringType,
		"updated_at":       types.StringType,
	}

	for _, entry := range ipAllowListEntries {
		entryAttrs := map[string]attr.Value{
			"id":               types.StringValue(string(entry.ID)),
			"name":             types.StringValue(string(entry.Name)),
			"allow_list_value": types.StringValue(string(entry.AllowListValue)),
			"is_active":        types.BoolValue(bool(entry.IsActive)),
			"created_at":       types.StringValue(string(entry.CreatedAt)),
			"updated_at":       types.StringValue(string(entry.UpdatedAt)),
		}
		entryObj, diags := types.ObjectValue(ipAllowListAttrTypes, entryAttrs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		ipAllowListElements = append(ipAllowListElements, entryObj)
	}

	ipAllowListValue, diags := types.ListValue(types.ObjectType{AttrTypes: ipAllowListAttrTypes}, ipAllowListElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set the data
	data.ID = types.StringValue(string(query.Organization.ID))
	data.IpAllowList = ipAllowListValue

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Debug(ctx, "Successfully read GitHub organization IP allow list", map[string]interface{}{
		"organization": orgName,
		"entries":      len(ipAllowListEntries),
	})
}
