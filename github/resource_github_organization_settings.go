package github

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &githubOrganizationSettingsResource{}
	_ resource.ResourceWithConfigure   = &githubOrganizationSettingsResource{}
	_ resource.ResourceWithImportState = &githubOrganizationSettingsResource{}
)

func NewGithubOrganizationSettingsResource() resource.Resource {
	return &githubOrganizationSettingsResource{}
}

type githubOrganizationSettingsResource struct {
	client *Owner
}

type githubOrganizationSettingsResourceModel struct {
	// Required attributes
	BillingEmail types.String `tfsdk:"billing_email"`

	// Optional basic organization information
	Company         types.String `tfsdk:"company"`
	Email           types.String `tfsdk:"email"`
	TwitterUsername types.String `tfsdk:"twitter_username"`
	Location        types.String `tfsdk:"location"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Blog            types.String `tfsdk:"blog"`

	// Project settings
	HasOrganizationProjects types.Bool `tfsdk:"has_organization_projects"`
	HasRepositoryProjects   types.Bool `tfsdk:"has_repository_projects"`

	// Repository permissions
	DefaultRepositoryPermission types.String `tfsdk:"default_repository_permission"`

	// Member repository creation permissions
	MembersCanCreateRepositories         types.Bool `tfsdk:"members_can_create_repositories"`
	MembersCanCreateInternalRepositories types.Bool `tfsdk:"members_can_create_internal_repositories"`
	MembersCanCreatePrivateRepositories  types.Bool `tfsdk:"members_can_create_private_repositories"`
	MembersCanCreatePublicRepositories   types.Bool `tfsdk:"members_can_create_public_repositories"`

	// Member pages creation permissions
	MembersCanCreatePages        types.Bool `tfsdk:"members_can_create_pages"`
	MembersCanCreatePublicPages  types.Bool `tfsdk:"members_can_create_public_pages"`
	MembersCanCreatePrivatePages types.Bool `tfsdk:"members_can_create_private_pages"`

	// Additional permissions
	MembersCanForkPrivateRepositories types.Bool `tfsdk:"members_can_fork_private_repositories"`
	WebCommitSignoffRequired          types.Bool `tfsdk:"web_commit_signoff_required"`

	// Security settings for new repositories
	AdvancedSecurityEnabledForNewRepositories             types.Bool `tfsdk:"advanced_security_enabled_for_new_repositories"`
	DependabotAlertsEnabledForNewRepositories             types.Bool `tfsdk:"dependabot_alerts_enabled_for_new_repositories"`
	DependabotSecurityUpdatesEnabledForNewRepositories    types.Bool `tfsdk:"dependabot_security_updates_enabled_for_new_repositories"`
	DependencyGraphEnabledForNewRepositories              types.Bool `tfsdk:"dependency_graph_enabled_for_new_repositories"`
	SecretScanningEnabledForNewRepositories               types.Bool `tfsdk:"secret_scanning_enabled_for_new_repositories"`
	SecretScanningPushProtectionEnabledForNewRepositories types.Bool `tfsdk:"secret_scanning_push_protection_enabled_for_new_repositories"`

	// Computed
	ID types.String `tfsdk:"id"`
}

func (r *githubOrganizationSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization_settings"
}

func (r *githubOrganizationSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages settings for a GitHub Organization.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "GitHub ID of the organization.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"billing_email": schema.StringAttribute{
				Description: "The billing email address for the organization.",
				Required:    true,
			},
			"company": schema.StringAttribute{
				Description: "The company name for the organization.",
				Optional:    true,
			},
			"email": schema.StringAttribute{
				Description: "The email address for the organization.",
				Optional:    true,
			},
			"twitter_username": schema.StringAttribute{
				Description: "The Twitter username for the organization.",
				Optional:    true,
			},
			"location": schema.StringAttribute{
				Description: "The location for the organization.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name for the organization.",
				Optional:    true,
			},
			"description": schema.StringAttribute{
				Description: "The description for the organization.",
				Optional:    true,
			},
			"blog": schema.StringAttribute{
				Description: "The blog URL for the organization.",
				Optional:    true,
			},
			"has_organization_projects": schema.BoolAttribute{
				Description: "Whether or not organization projects are enabled for the organization.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"has_repository_projects": schema.BoolAttribute{
				Description: "Whether or not repository projects are enabled for the organization.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"default_repository_permission": schema.StringAttribute{
				Description: "The default permission for organization members to create new repositories. Can be one of 'read', 'write', 'admin' or 'none'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("read"),
				Validators: []validator.String{
					stringvalidator.OneOf("read", "write", "admin", "none"),
				},
			},
			"members_can_create_repositories": schema.BoolAttribute{
				Description: "Whether or not organization members can create new repositories.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"members_can_create_internal_repositories": schema.BoolAttribute{
				Description: "Whether or not organization members can create new internal repositories. For Enterprise Organizations only.",
				Optional:    true,
			},
			"members_can_create_private_repositories": schema.BoolAttribute{
				Description: "Whether or not organization members can create new private repositories.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"members_can_create_public_repositories": schema.BoolAttribute{
				Description: "Whether or not organization members can create new public repositories.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"members_can_create_pages": schema.BoolAttribute{
				Description: "Whether or not organization members can create new pages.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"members_can_create_public_pages": schema.BoolAttribute{
				Description: "Whether or not organization members can create new public pages.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"members_can_create_private_pages": schema.BoolAttribute{
				Description: "Whether or not organization members can create new private pages.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"members_can_fork_private_repositories": schema.BoolAttribute{
				Description: "Whether or not organization members can fork private repositories.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"web_commit_signoff_required": schema.BoolAttribute{
				Description: "Whether or not commit signatures are required for commits to the organization.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"advanced_security_enabled_for_new_repositories": schema.BoolAttribute{
				Description: "Whether or not advanced security is enabled for new repositories.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"dependabot_alerts_enabled_for_new_repositories": schema.BoolAttribute{
				Description: "Whether or not dependabot alerts are enabled for new repositories.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"dependabot_security_updates_enabled_for_new_repositories": schema.BoolAttribute{
				Description: "Whether or not dependabot security updates are enabled for new repositories.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"dependency_graph_enabled_for_new_repositories": schema.BoolAttribute{
				Description: "Whether or not dependency graph is enabled for new repositories.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"secret_scanning_enabled_for_new_repositories": schema.BoolAttribute{
				Description: "Whether or not secret scanning is enabled for new repositories.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"secret_scanning_push_protection_enabled_for_new_repositories": schema.BoolAttribute{
				Description: "Whether or not secret scanning push protection is enabled for new repositories.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
		},
	}
}

func (r *githubOrganizationSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*Owner)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *github.Owner, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *githubOrganizationSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan githubOrganizationSettingsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	// Create/Update uses the same logic
	state, err := r.updateOrganizationSettings(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Organization Settings",
			"Could not create organization settings: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubOrganizationSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state githubOrganizationSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	client := r.client.V3Client()
	org := r.client.Name()

	orgSettings, _, err := client.Organizations.Get(ctx, org)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading Organization Settings",
			"Could not read organization settings: "+err.Error(),
		)
		return
	}

	// Update state with current values
	state.ID = types.StringValue(strconv.FormatInt(orgSettings.GetID(), 10))
	state.BillingEmail = types.StringValue(orgSettings.GetBillingEmail())
	state.Company = types.StringValue(orgSettings.GetCompany())
	state.Email = types.StringValue(orgSettings.GetEmail())
	state.TwitterUsername = types.StringValue(orgSettings.GetTwitterUsername())
	state.Location = types.StringValue(orgSettings.GetLocation())
	state.Name = types.StringValue(orgSettings.GetName())
	state.Description = types.StringValue(orgSettings.GetDescription())
	state.Blog = types.StringValue(orgSettings.GetBlog())
	state.HasOrganizationProjects = types.BoolValue(orgSettings.GetHasOrganizationProjects())
	state.HasRepositoryProjects = types.BoolValue(orgSettings.GetHasRepositoryProjects())
	state.DefaultRepositoryPermission = types.StringValue(orgSettings.GetDefaultRepoPermission())
	state.MembersCanCreateRepositories = types.BoolValue(orgSettings.GetMembersCanCreateRepos())
	state.MembersCanCreateInternalRepositories = types.BoolValue(orgSettings.GetMembersCanCreateInternalRepos())
	state.MembersCanCreatePrivateRepositories = types.BoolValue(orgSettings.GetMembersCanCreatePrivateRepos())
	state.MembersCanCreatePublicRepositories = types.BoolValue(orgSettings.GetMembersCanCreatePublicRepos())
	state.MembersCanCreatePages = types.BoolValue(orgSettings.GetMembersCanCreatePages())
	state.MembersCanCreatePublicPages = types.BoolValue(orgSettings.GetMembersCanCreatePublicPages())
	state.MembersCanCreatePrivatePages = types.BoolValue(orgSettings.GetMembersCanCreatePrivatePages())
	state.MembersCanForkPrivateRepositories = types.BoolValue(orgSettings.GetMembersCanForkPrivateRepos())
	state.WebCommitSignoffRequired = types.BoolValue(orgSettings.GetWebCommitSignoffRequired())
	state.AdvancedSecurityEnabledForNewRepositories = types.BoolValue(orgSettings.GetAdvancedSecurityEnabledForNewRepos())
	state.DependabotAlertsEnabledForNewRepositories = types.BoolValue(orgSettings.GetDependabotAlertsEnabledForNewRepos())
	state.DependabotSecurityUpdatesEnabledForNewRepositories = types.BoolValue(orgSettings.GetDependabotSecurityUpdatesEnabledForNewRepos())
	state.DependencyGraphEnabledForNewRepositories = types.BoolValue(orgSettings.GetDependencyGraphEnabledForNewRepos())
	state.SecretScanningEnabledForNewRepositories = types.BoolValue(orgSettings.GetSecretScanningEnabledForNewRepos())
	state.SecretScanningPushProtectionEnabledForNewRepositories = types.BoolValue(orgSettings.GetSecretScanningPushProtectionEnabledForNewRepos())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *githubOrganizationSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan githubOrganizationSettingsResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	// Update uses the same logic as create
	state, err := r.updateOrganizationSettings(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Organization Settings",
			"Could not update organization settings: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *githubOrganizationSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state githubOrganizationSettingsResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check organization
	if !r.client.IsOrganization {
		resp.Diagnostics.AddError(
			"Organization Required",
			"This resource can only be used with an organization account.",
		)
		return
	}

	client := r.client.V3Client()
	org := r.client.Name()

	// Reset organization settings to default values
	err := r.resetOrganizationSettingsToDefaults(ctx, client, org, state)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting Organization Settings",
			"Could not reset organization settings to defaults: "+err.Error(),
		)
		return
	}

	log.Printf("[DEBUG] Reverted Organization Settings to default values: %s", org)
}

func (r *githubOrganizationSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Use the organization ID as the import ID
	// The organization can be imported by its GitHub organization name which should match the provider configuration
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
}

// Helper function to update organization settings
func (r *githubOrganizationSettingsResource) updateOrganizationSettings(ctx context.Context, plan githubOrganizationSettingsResourceModel) (*githubOrganizationSettingsResourceModel, error) {
	client := r.client.V3Client()
	org := r.client.Name()

	// Build organization settings object
	settings := &github.Organization{
		BillingEmail:                       github.Ptr(plan.BillingEmail.ValueString()),
		Company:                            github.Ptr(plan.Company.ValueString()),
		Email:                              github.Ptr(plan.Email.ValueString()),
		TwitterUsername:                    github.Ptr(plan.TwitterUsername.ValueString()),
		Location:                           github.Ptr(plan.Location.ValueString()),
		Name:                               github.Ptr(plan.Name.ValueString()),
		Description:                        github.Ptr(plan.Description.ValueString()),
		Blog:                               github.Ptr(plan.Blog.ValueString()),
		HasOrganizationProjects:            github.Ptr(plan.HasOrganizationProjects.ValueBool()),
		HasRepositoryProjects:              github.Ptr(plan.HasRepositoryProjects.ValueBool()),
		DefaultRepoPermission:              github.Ptr(plan.DefaultRepositoryPermission.ValueString()),
		MembersCanCreateRepos:              github.Ptr(plan.MembersCanCreateRepositories.ValueBool()),
		MembersCanCreatePrivateRepos:       github.Ptr(plan.MembersCanCreatePrivateRepositories.ValueBool()),
		MembersCanCreatePublicRepos:        github.Ptr(plan.MembersCanCreatePublicRepositories.ValueBool()),
		MembersCanCreatePages:              github.Ptr(plan.MembersCanCreatePages.ValueBool()),
		MembersCanCreatePublicPages:        github.Ptr(plan.MembersCanCreatePublicPages.ValueBool()),
		MembersCanCreatePrivatePages:       github.Ptr(plan.MembersCanCreatePrivatePages.ValueBool()),
		MembersCanForkPrivateRepos:         github.Ptr(plan.MembersCanForkPrivateRepositories.ValueBool()),
		WebCommitSignoffRequired:           github.Ptr(plan.WebCommitSignoffRequired.ValueBool()),
		AdvancedSecurityEnabledForNewRepos: github.Ptr(plan.AdvancedSecurityEnabledForNewRepositories.ValueBool()),
		DependabotAlertsEnabledForNewRepos: github.Ptr(plan.DependabotAlertsEnabledForNewRepositories.ValueBool()),
		DependabotSecurityUpdatesEnabledForNewRepos:    github.Ptr(plan.DependabotSecurityUpdatesEnabledForNewRepositories.ValueBool()),
		DependencyGraphEnabledForNewRepos:              github.Ptr(plan.DependencyGraphEnabledForNewRepositories.ValueBool()),
		SecretScanningEnabledForNewRepos:               github.Ptr(plan.SecretScanningEnabledForNewRepositories.ValueBool()),
		SecretScanningPushProtectionEnabledForNewRepos: github.Ptr(plan.SecretScanningPushProtectionEnabledForNewRepositories.ValueBool()),
	}

	// Check organization plan to determine if enterprise features are available
	orgPlan, _, err := client.Organizations.Edit(ctx, org, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization plan: %w", err)
	}

	// Set internal repositories flag only for enterprise organizations
	if orgPlan.GetPlan().GetName() == "enterprise" {
		// Only set internal repositories if the value is specified in plan
		if !plan.MembersCanCreateInternalRepositories.IsNull() {
			settings.MembersCanCreateInternalRepos = github.Ptr(plan.MembersCanCreateInternalRepositories.ValueBool())
		}
	}

	// Update the organization
	orgSettings, _, err := client.Organizations.Edit(ctx, org, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to update organization settings: %w", err)
	}

	// Create state model from response
	state := &githubOrganizationSettingsResourceModel{
		ID:                                  types.StringValue(strconv.FormatInt(orgSettings.GetID(), 10)),
		BillingEmail:                        types.StringValue(orgSettings.GetBillingEmail()),
		Company:                             types.StringValue(orgSettings.GetCompany()),
		Email:                               types.StringValue(orgSettings.GetEmail()),
		TwitterUsername:                     types.StringValue(orgSettings.GetTwitterUsername()),
		Location:                            types.StringValue(orgSettings.GetLocation()),
		Name:                                types.StringValue(orgSettings.GetName()),
		Description:                         types.StringValue(orgSettings.GetDescription()),
		Blog:                                types.StringValue(orgSettings.GetBlog()),
		HasOrganizationProjects:             types.BoolValue(orgSettings.GetHasOrganizationProjects()),
		HasRepositoryProjects:               types.BoolValue(orgSettings.GetHasRepositoryProjects()),
		DefaultRepositoryPermission:         types.StringValue(orgSettings.GetDefaultRepoPermission()),
		MembersCanCreateRepositories:        types.BoolValue(orgSettings.GetMembersCanCreateRepos()),
		MembersCanCreatePrivateRepositories: types.BoolValue(orgSettings.GetMembersCanCreatePrivateRepos()),
		MembersCanCreatePublicRepositories:  types.BoolValue(orgSettings.GetMembersCanCreatePublicRepos()),
		MembersCanCreatePages:               types.BoolValue(orgSettings.GetMembersCanCreatePages()),
		MembersCanCreatePublicPages:         types.BoolValue(orgSettings.GetMembersCanCreatePublicPages()),
		MembersCanCreatePrivatePages:        types.BoolValue(orgSettings.GetMembersCanCreatePrivatePages()),
		MembersCanForkPrivateRepositories:   types.BoolValue(orgSettings.GetMembersCanForkPrivateRepos()),
		WebCommitSignoffRequired:            types.BoolValue(orgSettings.GetWebCommitSignoffRequired()),
		AdvancedSecurityEnabledForNewRepositories:             types.BoolValue(orgSettings.GetAdvancedSecurityEnabledForNewRepos()),
		DependabotAlertsEnabledForNewRepositories:             types.BoolValue(orgSettings.GetDependabotAlertsEnabledForNewRepos()),
		DependabotSecurityUpdatesEnabledForNewRepositories:    types.BoolValue(orgSettings.GetDependabotSecurityUpdatesEnabledForNewRepos()),
		DependencyGraphEnabledForNewRepositories:              types.BoolValue(orgSettings.GetDependencyGraphEnabledForNewRepos()),
		SecretScanningEnabledForNewRepositories:               types.BoolValue(orgSettings.GetSecretScanningEnabledForNewRepos()),
		SecretScanningPushProtectionEnabledForNewRepositories: types.BoolValue(orgSettings.GetSecretScanningPushProtectionEnabledForNewRepos()),
	}

	// Set internal repositories for enterprise organizations
	if orgPlan.GetPlan().GetName() == "enterprise" {
		state.MembersCanCreateInternalRepositories = types.BoolValue(orgSettings.GetMembersCanCreateInternalRepos())
	} else {
		// For non-enterprise organizations, internal repositories should be null/unset
		state.MembersCanCreateInternalRepositories = types.BoolNull()
	}

	return state, nil
}

// Helper function to reset organization settings to defaults
func (r *githubOrganizationSettingsResource) resetOrganizationSettingsToDefaults(ctx context.Context, client *github.Client, org string, state githubOrganizationSettingsResourceModel) error {
	// Default settings
	settings := &github.Organization{
		BillingEmail:                       github.Ptr("email@example.com"),
		Company:                            github.Ptr(""),
		Email:                              github.Ptr(""),
		TwitterUsername:                    github.Ptr(""),
		Location:                           github.Ptr(""),
		Name:                               github.Ptr(""),
		Description:                        github.Ptr(""),
		Blog:                               github.Ptr(""),
		HasOrganizationProjects:            github.Ptr(true),
		HasRepositoryProjects:              github.Ptr(true),
		DefaultRepoPermission:              github.Ptr("read"),
		MembersCanCreateRepos:              github.Ptr(true),
		MembersCanCreatePrivateRepos:       github.Ptr(true),
		MembersCanCreatePublicRepos:        github.Ptr(true),
		MembersCanCreatePages:              github.Ptr(false),
		MembersCanCreatePublicPages:        github.Ptr(true),
		MembersCanCreatePrivatePages:       github.Ptr(true),
		MembersCanForkPrivateRepos:         github.Ptr(false),
		WebCommitSignoffRequired:           github.Ptr(false),
		AdvancedSecurityEnabledForNewRepos: github.Ptr(false),
		DependabotAlertsEnabledForNewRepos: github.Ptr(false),
		DependabotSecurityUpdatesEnabledForNewRepos:    github.Ptr(false),
		DependencyGraphEnabledForNewRepos:              github.Ptr(false),
		SecretScanningEnabledForNewRepos:               github.Ptr(false),
		SecretScanningPushProtectionEnabledForNewRepos: github.Ptr(false),
	}

	// Check organization plan
	orgPlan, _, err := client.Organizations.Edit(ctx, org, nil)
	if err != nil {
		return err
	}

	// For enterprise organizations, include internal repositories
	if orgPlan.GetPlan().GetName() == "enterprise" {
		if !state.MembersCanForkPrivateRepositories.IsNull() {
			settings.MembersCanCreateInternalRepos = github.Ptr(true)
		}
	}

	_, _, err = client.Organizations.Edit(ctx, org, settings)
	return err
}
