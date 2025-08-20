package framework

import (
	"context"
	"crypto/md5"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/shurcooL/githubv4"

	githubpkg "github.com/isometry/terraform-provider-github/v7/github"
)

var (
	_ datasource.DataSource              = &githubUsersDataSource{}
	_ datasource.DataSourceWithConfigure = &githubUsersDataSource{}
)

type githubUsersDataSource struct {
	client *githubpkg.Owner
}

type githubUsersDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Usernames     types.List   `tfsdk:"usernames"`
	Logins        types.List   `tfsdk:"logins"`
	Emails        types.List   `tfsdk:"emails"`
	NodeIDs       types.List   `tfsdk:"node_ids"`
	UnknownLogins types.List   `tfsdk:"unknown_logins"`
}

func NewGithubUsersDataSource() datasource.DataSource {
	return &githubUsersDataSource{}
}

func (d *githubUsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *githubUsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information about multiple GitHub users.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the data source.",
				Computed:    true,
			},
			"usernames": schema.ListAttribute{
				Description: "List of usernames to lookup.",
				Required:    true,
				ElementType: types.StringType,
			},
			"logins": schema.ListAttribute{
				Description: "List of found user logins.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"emails": schema.ListAttribute{
				Description: "List of found user emails.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"node_ids": schema.ListAttribute{
				Description: "List of found user node IDs.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"unknown_logins": schema.ListAttribute{
				Description: "List of usernames that could not be found.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *githubUsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubUsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubUsersDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract usernames list
	var usernames []string
	resp.Diagnostics.Append(data.Usernames.ElementsAs(ctx, &usernames, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading GitHub users", map[string]interface{}{
		"usernames_count": len(usernames),
		"usernames":       usernames,
	})

	// Create GraphQL variables and query struct - same pattern as SDKv2
	type UserFragment struct {
		Id    string
		Login string
		Email string
	}

	var fields []reflect.StructField
	variables := make(map[string]any)
	for idx, username := range usernames {
		label := fmt.Sprintf("User%d", idx)
		variables[label] = githubv4.String(username)
		fields = append(fields, reflect.StructField{
			Name: label,
			Type: reflect.TypeOf(UserFragment{}),
			Tag:  reflect.StructTag(fmt.Sprintf("graphql:\"%[1]s: user(login: $%[1]s)\"", label)),
		})
	}
	query := reflect.New(reflect.StructOf(fields)).Elem()

	// Execute GraphQL query if we have usernames
	if len(usernames) > 0 {
		err := d.client.V4Client().Query(ctx, query.Addr().Interface(), variables)
		if err != nil && !strings.Contains(err.Error(), "Could not resolve to a User with the login of") {
			resp.Diagnostics.AddError(
				"Unable to Query GitHub Users",
				fmt.Sprintf("An unexpected error occurred while querying GitHub users: %s", err.Error()),
			)
			return
		}
	}

	// Process results
	var logins, emails, nodeIDs, unknownLogins []string
	for idx, username := range usernames {
		label := fmt.Sprintf("User%d", idx)
		user := query.FieldByName(label).Interface().(UserFragment)
		if user.Login != "" {
			logins = append(logins, user.Login)
			emails = append(emails, user.Email)
			nodeIDs = append(nodeIDs, user.Id)
		} else {
			unknownLogins = append(unknownLogins, username)
		}
	}

	// Calculate ID using same pattern as SDKv2 (buildChecksumID)
	id := buildChecksumID(usernames)
	data.ID = types.StringValue(id)

	// Convert slices to Framework list values
	loginsList, diags := types.ListValueFrom(ctx, types.StringType, logins)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Logins = loginsList

	emailsList, diags := types.ListValueFrom(ctx, types.StringType, emails)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Emails = emailsList

	nodeIDsList, diags := types.ListValueFrom(ctx, types.StringType, nodeIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.NodeIDs = nodeIDsList

	unknownLoginsList, diags := types.ListValueFrom(ctx, types.StringType, unknownLogins)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.UnknownLogins = unknownLoginsList

	tflog.Debug(ctx, "Successfully read GitHub users", map[string]interface{}{
		"usernames_count":     len(usernames),
		"found_users_count":   len(logins),
		"unknown_users_count": len(unknownLogins),
		"found_logins":        logins,
		"unknown_logins":      unknownLogins,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// buildChecksumID creates a checksum ID from a slice of strings
// This is a copy of the buildChecksumID function from github/util.go
func buildChecksumID(v []string) string {
	sort.Strings(v)

	h := md5.New()
	// Hash.Write never returns an error. See https://pkg.go.dev/hash#Hash
	_, _ = h.Write([]byte(strings.Join(v, "")))
	bs := h.Sum(nil)

	return fmt.Sprintf("%x", bs)
}
