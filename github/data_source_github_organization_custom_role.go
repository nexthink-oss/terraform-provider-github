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
	_ datasource.DataSource              = &githubOrganizationCustomRoleDataSource{}
	_ datasource.DataSourceWithConfigure = &githubOrganizationCustomRoleDataSource{}
)

type githubOrganizationCustomRoleDataSource struct {
	client *Owner
}

type githubOrganizationCustomRoleDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	BaseRole    types.String `tfsdk:"base_role"`
	Permissions types.Set    `tfsdk:"permissions"`
	Description types.String `tfsdk:"description"`
}

func NewGithubOrganizationCustomRoleDataSource() datasource.DataSource {
	return &githubOrganizationCustomRoleDataSource{}
}

func (d *githubOrganizationCustomRoleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_custom_role"
}

func (d *githubOrganizationCustomRoleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get a custom role from a GitHub Organization for use in repositories.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the custom role.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the custom role.",
				Required:    true,
			},
			"base_role": schema.StringAttribute{
				Description: "The base role of the custom role.",
				Computed:    true,
			},
			"permissions": schema.SetAttribute{
				Description: "List of additional permissions added to the base role.",
				ElementType: types.StringType,
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the custom role.",
				Computed:    true,
			},
		},
	}
}

func (d *githubOrganizationCustomRoleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubOrganizationCustomRoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubOrganizationCustomRoleDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check that this is an organization account
	if !d.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This data source can only be used with organization accounts",
		)
		return
	}

	orgName := d.client.Name()

	tflog.Debug(ctx, "Reading GitHub organization custom role", map[string]any{
		"org":  orgName,
		"name": data.Name.ValueString(),
	})

	// ListCustomRepoRoles returns a list of all custom repository roles for an organization.
	// There is an API endpoint for getting a single custom repository role, but is not
	// implemented in the go-github library.
	roleList, _, err := d.client.V3Client().Organizations.ListCustomRepoRoles(ctx, orgName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading GitHub Custom Repository Roles",
			fmt.Sprintf("Error querying GitHub custom repository roles %s: %s", orgName, err.Error()),
		)
		return
	}

	var role *github.CustomRepoRoles
	roleName := data.Name.ValueString()
	for _, r := range roleList.CustomRepoRoles {
		if r.Name != nil && *r.Name == roleName {
			role = r
			break
		}
	}

	if role == nil {
		resp.Diagnostics.AddError(
			"Custom Role Not Found",
			fmt.Sprintf("GitHub custom repository role (%s) not found in organization %s", roleName, orgName),
		)
		return
	}

	// Set the computed attributes
	data.ID = types.StringValue(fmt.Sprint(*role.ID))

	if role.Name != nil {
		data.Name = types.StringValue(*role.Name)
	}

	if role.Description != nil {
		data.Description = types.StringValue(*role.Description)
	} else {
		data.Description = types.StringNull()
	}

	if role.BaseRole != nil {
		data.BaseRole = types.StringValue(*role.BaseRole)
	} else {
		data.BaseRole = types.StringNull()
	}

	// Convert permissions slice to Set
	permissionsSet, diags := types.SetValueFrom(ctx, types.StringType, role.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Permissions = permissionsSet

	tflog.Debug(ctx, "Successfully read GitHub organization custom role", map[string]any{
		"org":  orgName,
		"name": data.Name.ValueString(),
		"id":   data.ID.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
