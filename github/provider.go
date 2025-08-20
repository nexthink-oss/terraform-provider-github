package github

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

)

type githubProvider struct{}

type githubProviderModel struct {
	Token   types.String `tfsdk:"token"`
	Owner   types.String `tfsdk:"owner"`
	BaseURL types.String `tfsdk:"base_url"`
}

func New() provider.Provider {
	return &githubProvider{}
}

func (p *githubProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "github"
}

func (p *githubProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "GitHub Provider (Plugin Framework)",
		Attributes: map[string]schema.Attribute{
			"token": schema.StringAttribute{
				Description: "The GitHub personal access token or GitHub App installation token.",
				Optional:    true,
				Sensitive:   true,
			},
			"owner": schema.StringAttribute{
				Description: "The GitHub owner name to manage resources under.",
				Optional:    true,
			},
			"base_url": schema.StringAttribute{
				Description: "The GitHub base API URL",
				Optional:    true,
			},
		},
	}
}

func (p *githubProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config githubProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get configuration values
	token := os.Getenv("GITHUB_TOKEN")
	if !config.Token.IsNull() && !config.Token.IsUnknown() {
		token = config.Token.ValueString()
	}

	owner := os.Getenv("GITHUB_OWNER")
	if !config.Owner.IsNull() && !config.Owner.IsUnknown() {
		owner = config.Owner.ValueString()
	}

	baseURL := os.Getenv("GITHUB_BASE_URL")
	if !config.BaseURL.IsNull() && !config.BaseURL.IsUnknown() {
		baseURL = config.BaseURL.ValueString()
	}

	// Create GitHub configuration with basic defaults
	githubConfig := &Config{
		Token:            token,
		Owner:            owner,
		BaseURL:          baseURL,
		MaxRetries:       3,
		ParallelRequests: true,
	}

	// Set BaseURL default if empty
	if githubConfig.BaseURL == "" {
		githubConfig.BaseURL = "https://api.github.com/"
	}

	// Initialize retryable errors with default values
	githubConfig.RetryableErrors = map[int]bool{
		500: true,
		502: true,
		503: true,
		504: true,
	}

	// Initialize the client using the Meta method
	client, err := githubConfig.Meta()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create GitHub API Client",
			"An unexpected error occurred when creating the GitHub API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"GitHub Client Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *githubProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewGithubUserSshKeyResource,
		NewGithubUserGpgKeyResource,
		NewGithubUserInvitationAccepterResource,
		NewGithubBranchResource,
		NewGithubBranchDefaultResource,
		NewGithubIssueResource,
		NewGithubIssueLabelResource,
		NewGithubIssueLabelsResource,
		NewGithubRepositoryFileResource,
		NewGithubActionsSecretResource,
		NewGithubActionsOrganizationSecretResource,
		NewGithubActionsOrganizationSecretRepositoriesResource,
		NewGithubActionsEnvironmentSecretResource,
		NewGithubActionsVariableResource,
		NewGithubActionsOrganizationVariableResource,
		NewGithubActionsEnvironmentVariableResource,
		NewGithubRepositoryDeployKeyResource,
		NewGithubRepositoryMilestoneResource,
		NewGithubRepositoryTopicsResource,
		NewGithubCodespacesSecretResource,
		NewGithubCodespacesUserSecretResource,
		NewGithubCodespacesOrganizationSecretResource,
		NewGithubCodespacesOrganizationSecretRepositoriesResource,
		NewGithubDependabotSecretResource,
		NewGithubDependabotOrganizationSecretResource,
		NewGithubDependabotOrganizationSecretRepositoriesResource,
		NewGithubReleaseResource,
		NewGithubRepositoryResource,
		NewGithubRepositoryCollaboratorResource,
		NewGithubRepositoryCollaboratorsResource,
		NewGithubRepositoryWebhookResource,
		NewGithubOrganizationWebhookResource,
		NewGithubBranchProtectionResource,
		NewGithubBranchProtectionV3Resource,
		NewGithubRepositoryEnvironmentResource,
		NewGithubRepositoryRulesetResource,
		NewGithubOrganizationRulesetResource,
		NewGithubOrganizationSettingsResource,
		NewGithubActionsOrganizationPermissionsResource,
		NewGithubActionsOrganizationOIDCSubjectClaimCustomizationTemplateResource,
		NewGithubActionsRepositoryOIDCSubjectClaimCustomizationTemplateResource,
		NewGithubActionsRunnerGroupResource,
		NewGithubEnterpriseActionsPermissionsResource,
		NewGithubEnterpriseActionsRunnerGroupResource,
		NewGithubEnterpriseOrganizationResource,
		NewGithubActionsRepositoryPermissionsResource,
		NewGithubActionsRepositoryAccessLevelResource,
		NewGithubRepositoryAutolinkReferenceResource,
		NewGithubRepositoryDependabotSecurityUpdatesResource,
		NewGithubOrganizationCustomRoleResource,
		NewGithubOrganizationSecurityManagerResource,
		NewGithubOrganizationBlockResource,
		NewGithubAppInstallationRepositoriesResource,
		NewGithubAppInstallationRepositoryResource,
		NewGithubRepositoryCustomPropertyResource,
		NewGithubEmuGroupMappingResource,
		NewGithubRepositoryDeploymentBranchPolicyResource,
		NewGithubRepositoryEnvironmentDeploymentPolicyResource,
		NewGithubRepositoryPullRequestResource,
		NewGithubTeamResource,
		NewGithubTeamMembersResource,
		NewGithubTeamMembershipResource,
		NewGithubTeamRepositoryResource,
		NewGithubTeamSettingsResource,
		NewGithubTeamSyncGroupMappingResource,
		NewGithubMembershipResource,
	}
}

func (p *githubProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewGithubActionsPublicKeyDataSource,
		NewGithubActionsOrganizationPublicKeyDataSource,
		NewGithubActionsOrganizationOIDCSubjectClaimCustomizationTemplateDataSource,
		NewGithubActionsOrganizationRegistrationTokenDataSource,
		NewGithubActionsOrganizationSecretsDataSource,
		NewGithubActionsOrganizationVariablesDataSource,
		NewGithubActionsRegistrationTokenDataSource,
		NewGithubActionsRepositoryOIDCSubjectClaimCustomizationTemplateDataSource,
		NewGithubActionsSecretsDataSource,
		NewGithubActionsEnvironmentSecretsDataSource,
		NewGithubActionsEnvironmentVariablesDataSource,
		NewGithubActionsVariablesDataSource,
		NewGithubCodespacesPublicKeyDataSource,
		NewGithubCodespacesOrganizationPublicKeyDataSource,
		NewGithubCodespacesOrganizationSecretsDataSource,
		NewGithubCodespacesSecretsDataSource,
		NewGithubCodespacesUserPublicKeyDataSource,
		NewGithubCodespacesUserSecretsDataSource,
		NewGithubDependabotPublicKeyDataSource,
		NewGithubDependabotSecretsDataSource,
		NewGithubDependabotOrganizationPublicKeyDataSource,
		NewGithubDependabotOrganizationSecretsDataSource,
		NewGithubAppDataSource,
		NewGithubAppTokenDataSource,
		NewGithubBranchDataSource,
		NewGithubBranchProtectionRulesDataSource,
		NewGithubRefDataSource,
		NewGithubRepositoriesDataSource,
		NewGithubRepositoryDataSource,
		NewGithubRepositoryBranchesDataSource,
		NewGithubRepositoryCustomPropertiesDataSource,
		NewGithubRepositoryDeployKeysDataSource,
		NewGithubRepositoryEnvironmentsDataSource,
		NewGithubRepositoryFileDataSource,
		NewGithubRepositoryPullRequestDataSource,
		NewGithubRepositoryPullRequestsDataSource,
		NewGithubRepositoryTeamsDataSource,
		NewGithubRepositoryWebhooksDataSource,
		NewGithubOrganizationWebhooksDataSource,
		NewGithubTreeDataSource,
		NewGithubIssueLabelsDataSource,
		NewGithubIpRangesDataSource,
		NewGithubSshKeysDataSource,
		NewGithubUserDataSource,
		NewGithubUserExternalIdentityDataSource,
		NewGithubUsersDataSource,
		NewGithubMembershipDataSource,
		NewGithubTeamDataSource,
		NewGithubEnterpriseDataSource,
		NewGithubOrganizationDataSource,
		NewGithubOrganizationCustomRoleDataSource,
		NewGithubOrganizationExternalIdentitiesDataSource,
		NewGithubOrganizationIpAllowListDataSource,
		NewGithubOrganizationTeamsDataSource,
		NewGithubOrganizationTeamSyncGroupsDataSource,
		NewGithubCollaboratorsDataSource,
		NewGithubReleaseDataSource,
		NewGithubRepositoryMilestoneDataSource,
		NewGithubRepositoryDeploymentBranchPoliciesDataSource,
		NewGithubExternalGroupsDataSource,
		NewGithubRepositoryAutolinkReferencesDataSource,
		NewGithubRestApiDataSource,
	}
}
