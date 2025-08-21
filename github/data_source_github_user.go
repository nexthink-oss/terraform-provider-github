package github

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &githubUserDataSource{}
	_ datasource.DataSourceWithConfigure = &githubUserDataSource{}
)

type githubUserDataSource struct {
	client *Owner
}

type githubUserDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Username    types.String `tfsdk:"username"`
	Login       types.String `tfsdk:"login"`
	AvatarURL   types.String `tfsdk:"avatar_url"`
	GravatarID  types.String `tfsdk:"gravatar_id"`
	SiteAdmin   types.Bool   `tfsdk:"site_admin"`
	Name        types.String `tfsdk:"name"`
	Company     types.String `tfsdk:"company"`
	Blog        types.String `tfsdk:"blog"`
	Location    types.String `tfsdk:"location"`
	Email       types.String `tfsdk:"email"`
	Bio         types.String `tfsdk:"bio"`
	GPGKeys     types.List   `tfsdk:"gpg_keys"`
	SSHKeys     types.List   `tfsdk:"ssh_keys"`
	PublicRepos types.Int64  `tfsdk:"public_repos"`
	PublicGists types.Int64  `tfsdk:"public_gists"`
	Followers   types.Int64  `tfsdk:"followers"`
	Following   types.Int64  `tfsdk:"following"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
	SuspendedAt types.String `tfsdk:"suspended_at"`
	NodeID      types.String `tfsdk:"node_id"`
}

func NewGithubUserDataSource() datasource.DataSource {
	return &githubUserDataSource{}
}

func (d *githubUserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *githubUserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Get information on a GitHub user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the user.",
				Computed:    true,
			},
			"username": schema.StringAttribute{
				Description: "The username to lookup.",
				Required:    true,
			},
			"login": schema.StringAttribute{
				Description: "The user's login.",
				Computed:    true,
			},
			"avatar_url": schema.StringAttribute{
				Description: "The user's avatar URL.",
				Computed:    true,
			},
			"gravatar_id": schema.StringAttribute{
				Description: "The user's Gravatar ID.",
				Computed:    true,
			},
			"site_admin": schema.BoolAttribute{
				Description: "Whether the user is a site administrator.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The user's public name.",
				Computed:    true,
			},
			"company": schema.StringAttribute{
				Description: "The user's company.",
				Computed:    true,
			},
			"blog": schema.StringAttribute{
				Description: "The user's blog URL.",
				Computed:    true,
			},
			"location": schema.StringAttribute{
				Description: "The user's location.",
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "The user's public email address.",
				Computed:    true,
			},
			"bio": schema.StringAttribute{
				Description: "The user's bio.",
				Computed:    true,
			},
			"gpg_keys": schema.ListAttribute{
				Description: "List of the user's GPG keys.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"ssh_keys": schema.ListAttribute{
				Description: "List of the user's SSH keys.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"public_repos": schema.Int64Attribute{
				Description: "The number of public repositories owned by the user.",
				Computed:    true,
			},
			"public_gists": schema.Int64Attribute{
				Description: "The number of public gists owned by the user.",
				Computed:    true,
			},
			"followers": schema.Int64Attribute{
				Description: "The number of followers of the user.",
				Computed:    true,
			},
			"following": schema.Int64Attribute{
				Description: "The number of users followed by the user.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The date the user was created.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The date the user was last updated.",
				Computed:    true,
			},
			"suspended_at": schema.StringAttribute{
				Description: "The date the user was suspended.",
				Computed:    true,
			},
			"node_id": schema.StringAttribute{
				Description: "The node ID of the user.",
				Computed:    true,
			},
		},
	}
}

func (d *githubUserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *githubUserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data githubUserDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	username := data.Username.ValueString()

	tflog.Debug(ctx, "Reading GitHub user", map[string]any{
		"username": username,
	})

	user, _, err := d.client.V3Client().Users.Get(ctx, username)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub User",
			fmt.Sprintf("An unexpected error occurred while reading the GitHub user %s: %s", username, err.Error()),
		)
		return
	}

	// Get GPG keys
	gpg, _, err := d.client.V3Client().Users.ListGPGKeys(ctx, user.GetLogin(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub User GPG Keys",
			fmt.Sprintf("An unexpected error occurred while reading GPG keys for GitHub user %s: %s", username, err.Error()),
		)
		return
	}

	// Get SSH keys
	ssh, _, err := d.client.V3Client().Users.ListKeys(ctx, user.GetLogin(), nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read GitHub User SSH Keys",
			fmt.Sprintf("An unexpected error occurred while reading SSH keys for GitHub user %s: %s", username, err.Error()),
		)
		return
	}

	// Process GPG keys
	gpgKeys := make([]string, len(gpg))
	for i, v := range gpg {
		gpgKeys[i] = v.GetPublicKey()
	}

	// Process SSH keys
	sshKeys := make([]string, len(ssh))
	for i, v := range ssh {
		sshKeys[i] = v.GetKey()
	}

	// Set values
	data.ID = types.StringValue(strconv.FormatInt(user.GetID(), 10))
	data.Login = types.StringValue(user.GetLogin())
	data.AvatarURL = types.StringValue(user.GetAvatarURL())
	data.GravatarID = types.StringValue(user.GetGravatarID())
	data.SiteAdmin = types.BoolValue(user.GetSiteAdmin())
	data.Name = types.StringValue(user.GetName())
	data.Company = types.StringValue(user.GetCompany())
	data.Blog = types.StringValue(user.GetBlog())
	data.Location = types.StringValue(user.GetLocation())
	data.Email = types.StringValue(user.GetEmail())
	data.Bio = types.StringValue(user.GetBio())
	data.PublicRepos = types.Int64Value(int64(user.GetPublicRepos()))
	data.PublicGists = types.Int64Value(int64(user.GetPublicGists()))
	data.Followers = types.Int64Value(int64(user.GetFollowers()))
	data.Following = types.Int64Value(int64(user.GetFollowing()))
	data.CreatedAt = types.StringValue(user.GetCreatedAt().String())
	data.UpdatedAt = types.StringValue(user.GetUpdatedAt().String())
	data.SuspendedAt = types.StringValue(user.GetSuspendedAt().String())
	data.NodeID = types.StringValue(user.GetNodeID())

	// Convert slices to Framework list values
	gpgList, diags := types.ListValueFrom(ctx, types.StringType, gpgKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.GPGKeys = gpgList

	sshList, diags := types.ListValueFrom(ctx, types.StringType, sshKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.SSHKeys = sshList

	tflog.Debug(ctx, "Successfully read GitHub user", map[string]any{
		"username":     username,
		"id":           data.ID.ValueString(),
		"login":        data.Login.ValueString(),
		"name":         data.Name.ValueString(),
		"public_repos": data.PublicRepos.ValueInt64(),
		"followers":    data.Followers.ValueInt64(),
		"following":    data.Following.ValueInt64(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
