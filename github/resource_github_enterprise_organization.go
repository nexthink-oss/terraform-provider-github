package github

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/shurcooL/githubv4"
)

var (
	_ resource.Resource                = &githubEnterpriseOrganizationResource{}
	_ resource.ResourceWithConfigure   = &githubEnterpriseOrganizationResource{}
	_ resource.ResourceWithImportState = &githubEnterpriseOrganizationResource{}
)

func NewGithubEnterpriseOrganizationResource() resource.Resource {
	return &githubEnterpriseOrganizationResource{}
}

type githubEnterpriseOrganizationResource struct {
	client *Owner
}

type githubEnterpriseOrganizationResourceModel struct {
	// Required attributes
	EnterpriseID types.String `tfsdk:"enterprise_id"`
	Name         types.String `tfsdk:"name"`
	AdminLogins  types.Set    `tfsdk:"admin_logins"`
	BillingEmail types.String `tfsdk:"billing_email"`

	// Optional attributes
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`

	// Computed attributes
	ID         types.String `tfsdk:"id"`
	DatabaseID types.Int64  `tfsdk:"database_id"`
}

func (r *githubEnterpriseOrganizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_enterprise_organization"
}

func (r *githubEnterpriseOrganizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Create and manages a GitHub enterprise organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The node ID of the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enterprise_id": schema.StringAttribute{
				Description: "The ID of the enterprise.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "The name of the organization.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"admin_logins": schema.SetAttribute{
				Description: "List of organization owner usernames.",
				Required:    true,
				ElementType: types.StringType,
			},
			"billing_email": schema.StringAttribute{
				Description: "The billing email address.",
				Required:    true,
			},
			"display_name": schema.StringAttribute{
				Description: "The display name of the organization.",
				Optional:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description of the organization.",
				Optional:    true,
				Computed:    true,
			},
			"database_id": schema.Int64Attribute{
				Description: "The database ID of the organization.",
				Computed:    true,
			},
		},
	}
}

func (r *githubEnterpriseOrganizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *githubEnterpriseOrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubEnterpriseOrganizationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var mutate struct {
		CreateEnterpriseOrganization struct {
			Organization struct {
				ID githubv4.ID
			}
		} `graphql:"createEnterpriseOrganization(input:$input)"`
	}

	v3client := r.client.V3Client()
	v4client := r.client.V4Client()

	// Convert admin logins from plan
	var adminLogins []githubv4.String
	adminLoginsSet := plan.AdminLogins
	if !adminLoginsSet.IsNull() && !adminLoginsSet.IsUnknown() {
		var planAdminLogins []string
		resp.Diagnostics.Append(adminLoginsSet.ElementsAs(ctx, &planAdminLogins, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, login := range planAdminLogins {
			adminLogins = append(adminLogins, githubv4.String(login))
		}
	}

	input := githubv4.CreateEnterpriseOrganizationInput{
		EnterpriseID: plan.EnterpriseID.ValueString(),
		Login:        githubv4.String(plan.Name.ValueString()),
		ProfileName:  githubv4.String(plan.Name.ValueString()),
		BillingEmail: githubv4.String(plan.BillingEmail.ValueString()),
		AdminLogins:  adminLogins,
	}

	err := v4client.Mutate(ctx, &mutate, input, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating enterprise organization", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s", mutate.CreateEnterpriseOrganization.Organization.ID))

	// Use the V3 API to set the description and display name of the org,
	// because there is no mutator in the V4 API to edit the org's description and display name
	//
	// NOTE: There is some odd behavior here when using an EMU with SSO. If the user token has been
	// granted permission to ANY ORG in the enterprise, then this works, provided that our token has
	// sufficient permission. If the user token has not been added to any orgs, then this will fail.
	//
	// Unfortunately, there is no way in the api to grant a token permission to access an org. This needs
	// to be done via the UI. This means our resource will work fine if the user has sufficient admin
	// permissions and at least one org exists. It also means that we can't use terraform to automate
	// creation of the very first org in an enterprise. That sucks a little, but seems like a restriction
	// we can live with.
	//
	// It would be nice if there was an API available in github to enable a token for SSO.

	// Set description and display name if configured by the user
	orgUpdate := &github.Organization{}
	needsUpdate := false

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() && plan.Description.ValueString() != "" {
		orgUpdate.Description = github.Ptr(plan.Description.ValueString())
		needsUpdate = true
	}

	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() && plan.DisplayName.ValueString() != "" {
		orgUpdate.Name = github.Ptr(plan.DisplayName.ValueString())
		needsUpdate = true
	}

	if needsUpdate {
		_, _, err = v3client.Organizations.Edit(ctx, plan.Name.ValueString(), orgUpdate)
		if err != nil {
			resp.Diagnostics.AddError("Error setting organization description/display name", err.Error())
			return
		}
	}

	// Read the resource to populate all computed fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubEnterpriseOrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubEnterpriseOrganizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	r.readResource(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubEnterpriseOrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubEnterpriseOrganizationResourceModel
	var state githubEnterpriseOrganizationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	v3client := r.client.V3Client()
	v4client := r.client.V4Client()
	orgName := plan.Name.ValueString()

	// Update display name if changed
	err := r.updateDisplayName(ctx, &plan, &state, v3client)
	if err != nil {
		resp.Diagnostics.AddError("Error updating display name", err.Error())
		return
	}

	// Update description if changed
	err = r.updateDescription(ctx, &plan, &state, v3client)
	if err != nil {
		resp.Diagnostics.AddError("Error updating description", err.Error())
		return
	}

	// Update admin list if changed
	err = r.updateAdminList(ctx, &plan, &state, orgName, v3client, v4client)
	if err != nil {
		resp.Diagnostics.AddError("Error updating admin list", err.Error())
		return
	}

	// Update billing email if changed
	err = r.updateBillingEmail(ctx, &plan, &state, orgName, v3client)
	if err != nil {
		resp.Diagnostics.AddError("Error updating billing email", err.Error())
		return
	}

	// Read the resource to populate all fields
	r.readResource(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *githubEnterpriseOrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubEnterpriseOrganizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	v3client := r.client.V3Client()
	ctx = context.WithValue(ctx, CtxId, state.ID.ValueString())

	_, err := v3client.Organizations.Delete(ctx, state.Name.ValueString())

	// We expect the delete to return with a 202 Accepted error so ignore those
	if _, ok := err.(*github.AcceptedError); ok {
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error deleting enterprise organization", err.Error())
		return
	}
}

func (r *githubEnterpriseOrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Invalid ID specified: supplied ID must be written as <enterprise_slug>/<org_name>",
		)
		return
	}

	v4client := r.client.V4Client()

	enterpriseId, err := r.getEnterpriseId(ctx, v4client, parts[0])
	if err != nil {
		resp.Diagnostics.AddError("Error getting enterprise ID", err.Error())
		return
	}

	orgId, err := r.getOrganizationId(ctx, v4client, parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Error getting organization ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("enterprise_id"), enterpriseId)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), orgId)...)
}

// Helper methods

func (r *githubEnterpriseOrganizationResource) readResource(ctx context.Context, model *githubEnterpriseOrganizationResourceModel, diagnostics *diag.Diagnostics) {
	var query struct {
		Node struct {
			Organization struct {
				ID                       githubv4.ID
				DatabaseId               githubv4.Int
				Name                     githubv4.String
				Login                    githubv4.String
				Description              githubv4.String
				OrganizationBillingEmail githubv4.String
				MembersWithRole          struct {
					Edges []struct {
						User struct {
							Login githubv4.String
						} `graphql:"node"`
						Role githubv4.String
					} `graphql:"edges"`
					PageInfo PageInfo
				} `graphql:"membersWithRole(first:100, after:$cursor)"`
			} `graphql:"... on Organization"`
		} `graphql:"node(id: $id)"`
	}

	variables := map[string]any{
		"id":     model.ID.ValueString(),
		"cursor": (*githubv4.String)(nil),
	}

	var adminLogins []string
	v4client := r.client.V4Client()

	for {
		err := v4client.Query(ctx, &query, variables)
		if err != nil {
			if strings.Contains(err.Error(), "Could not resolve to a node with the global id") {
				log.Printf("[INFO] Removing organization (%s) from state because it no longer exists in GitHub", model.ID.ValueString())
				model.ID = types.StringValue("")
				return
			}
			diagnostics.AddError("Error reading enterprise organization", err.Error())
			return
		}

		for _, v := range query.Node.Organization.MembersWithRole.Edges {
			if v.Role == "ADMIN" {
				adminLogins = append(adminLogins, string(v.User.Login))
			}
		}

		if !query.Node.Organization.MembersWithRole.PageInfo.HasNextPage {
			break
		}

		variables["cursor"] = githubv4.NewString(query.Node.Organization.MembersWithRole.PageInfo.EndCursor)
	}

	// Convert admin logins to set
	adminLoginsSet, diags := types.SetValueFrom(ctx, types.StringType, adminLogins)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return
	}

	model.AdminLogins = adminLoginsSet
	model.Name = types.StringValue(string(query.Node.Organization.Login))
	model.BillingEmail = types.StringValue(string(query.Node.Organization.OrganizationBillingEmail))
	model.DatabaseID = types.Int64Value(int64(query.Node.Organization.DatabaseId))
	model.Description = types.StringValue(string(query.Node.Organization.Description))

	// Set display name only if different from name
	if query.Node.Organization.Name != query.Node.Organization.Login {
		model.DisplayName = types.StringValue(string(query.Node.Organization.Name))
	} else {
		model.DisplayName = types.StringNull()
	}
}

func (r *githubEnterpriseOrganizationResource) getEnterpriseId(ctx context.Context, v4client *githubv4.Client, enterpriseSlug string) (string, error) {
	var query struct {
		Enterprise struct {
			ID githubv4.String
		} `graphql:"enterprise(slug: $enterpriseSlug)"`
	}

	err := v4client.Query(ctx, &query, map[string]any{"enterpriseSlug": githubv4.String(enterpriseSlug)})
	if err != nil {
		return "", err
	}
	return string(query.Enterprise.ID), nil
}

func (r *githubEnterpriseOrganizationResource) getOrganizationId(ctx context.Context, v4client *githubv4.Client, orgName string) (string, error) {
	var query struct {
		Organization struct {
			Id githubv4.String
		} `graphql:"organization(login: $orgName)"`
	}

	err := v4client.Query(ctx, &query, map[string]any{"orgName": githubv4.String(orgName)})
	if err != nil {
		return "", err
	}
	return string(query.Organization.Id), nil
}

func (r *githubEnterpriseOrganizationResource) updateDescription(ctx context.Context, plan, state *githubEnterpriseOrganizationResourceModel, v3client *github.Client) error {
	orgName := plan.Name.ValueString()

	// Only update description if it was configured by the user
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		oldDesc := state.Description.ValueString()
		newDesc := plan.Description.ValueString()

		if oldDesc != newDesc {
			_, _, err := v3client.Organizations.Edit(
				ctx,
				orgName,
				&github.Organization{
					Description: github.Ptr(plan.Description.ValueString()),
				},
			)
			return err
		}
	}
	return nil
}

func (r *githubEnterpriseOrganizationResource) updateDisplayName(ctx context.Context, plan, state *githubEnterpriseOrganizationResourceModel, v3client *github.Client) error {
	orgName := plan.Name.ValueString()
	oldDisplayName := state.DisplayName.ValueString()
	newDisplayName := plan.DisplayName.ValueString()

	if oldDisplayName != newDisplayName {
		_, _, err := v3client.Organizations.Edit(
			ctx,
			orgName,
			&github.Organization{
				Name: github.Ptr(plan.DisplayName.ValueString()),
			},
		)
		return err
	}
	return nil
}

func (r *githubEnterpriseOrganizationResource) updateBillingEmail(ctx context.Context, plan, state *githubEnterpriseOrganizationResourceModel, orgName string, v3client *github.Client) error {
	oldBilling := state.BillingEmail.ValueString()
	newBilling := plan.BillingEmail.ValueString()
	if oldBilling != newBilling {
		_, _, err := v3client.Organizations.Edit(
			ctx,
			orgName,
			&github.Organization{
				BillingEmail: &newBilling,
			},
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *githubEnterpriseOrganizationResource) updateAdminList(ctx context.Context, plan, state *githubEnterpriseOrganizationResourceModel, orgName string, v3client *github.Client, v4client *githubv4.Client) error {
	// Get old and new admin logins
	var oldAdminLogins, newAdminLogins []string

	if !state.AdminLogins.IsNull() {
		state.AdminLogins.ElementsAs(ctx, &oldAdminLogins, false)
	}

	if !plan.AdminLogins.IsNull() {
		plan.AdminLogins.ElementsAs(ctx, &newAdminLogins, false)
	}

	// Calculate differences
	toRemove := stringSliceDiff(oldAdminLogins, newAdminLogins)
	toAdd := stringSliceDiff(newAdminLogins, oldAdminLogins)

	err := r.addUsers(ctx, plan, v4client, toAdd)
	if err != nil {
		return err
	}

	return r.removeUsers(ctx, v3client, v4client, toRemove, orgName)
}

func (r *githubEnterpriseOrganizationResource) removeUsers(ctx context.Context, v3client *github.Client, v4client *githubv4.Client, toRemove []string, orgName string) error {
	for _, user := range toRemove {
		err := r.removeUser(ctx, v3client, v4client, user, orgName)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *githubEnterpriseOrganizationResource) removeUser(ctx context.Context, v3client *github.Client, v4client *githubv4.Client, user string, orgName string) error {
	// How we remove an admin user from an enterprise organization depends on if the user is a member of any teams.
	// If they are a member of any teams, we shouldn't delete them, instead we edit their membership role to be
	// 'MEMBER' instead of 'ADMIN'. If the user is not a member of any teams, then we remove from the org.

	// First, use the v4 API to count how many teams the user is in
	var query struct {
		Organization struct {
			Teams struct {
				TotalCount githubv4.Int
			} `graphql:"teams(first:1, userLogins:[$user])"`
		} `graphql:"organization(login: $org)"`
	}

	err := v4client.Query(
		ctx,
		&query,
		map[string]any{
			"org":  githubv4.String(orgName),
			"user": githubv4.String(user),
		},
	)
	if err != nil {
		return err
	}

	if query.Organization.Teams.TotalCount == 0 {
		_, err = v3client.Organizations.RemoveOrgMembership(ctx, user, orgName)
		return err
	}

	membership, _, err := v3client.Organizations.GetOrgMembership(ctx, user, orgName)
	if err != nil {
		return err
	}

	membership.Role = github.Ptr("member")
	_, _, err = v3client.Organizations.EditOrgMembership(ctx, user, orgName, membership)
	return err
}

func (r *githubEnterpriseOrganizationResource) addUsers(ctx context.Context, plan *githubEnterpriseOrganizationResourceModel, v4client *githubv4.Client, toAdd []string) error {
	if len(toAdd) != 0 {
		var mutate struct {
			AddEnterpriseOrganizationMember struct {
				Ignored string `graphql:"clientMutationId"`
			} `graphql:"addEnterpriseOrganizationMember(input: $input)"`
		}

		adminRole := githubv4.OrganizationMemberRoleAdmin
		userIds, err := r.getUserIds(v4client, toAdd)
		if err != nil {
			return err
		}

		input := githubv4.AddEnterpriseOrganizationMemberInput{
			EnterpriseID:   plan.EnterpriseID.ValueString(),
			OrganizationID: plan.ID.ValueString(),
			UserIDs:        userIds,
			Role:           &adminRole,
		}

		err = v4client.Mutate(ctx, &mutate, input, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *githubEnterpriseOrganizationResource) getUserIds(v4client *githubv4.Client, loginNames []string) ([]githubv4.ID, error) {
	var query struct {
		User struct {
			ID githubv4.String
		} `graphql:"user(login: $login)"`
	}

	var ret []githubv4.ID

	for _, l := range loginNames {
		err := v4client.Query(context.Background(), &query, map[string]any{"login": githubv4.String(l)})
		if err != nil {
			return nil, err
		}
		ret = append(ret, query.User.ID)
	}
	return ret, nil
}

// stringSliceDiff returns elements in slice a that are not in slice b
func stringSliceDiff(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}
