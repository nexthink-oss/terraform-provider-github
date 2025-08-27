package github

import (
	"context"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type githubProvider struct{}

type githubProviderModel struct {
	Token            types.String `tfsdk:"token"`
	Owner            types.String `tfsdk:"owner"`
	Organization     types.String `tfsdk:"organization"`
	BaseURL          types.String `tfsdk:"base_url"`
	ParallelRequests types.Bool   `tfsdk:"parallel_requests"`
	ReadDelayMS      types.Int64  `tfsdk:"read_delay_ms"`
	WriteDelayMS     types.Int64  `tfsdk:"write_delay_ms"`
	RetryDelayMS     types.Int64  `tfsdk:"retry_delay_ms"`
	MaxRetries       types.Int64  `tfsdk:"max_retries"`
	RetryableErrors  types.List   `tfsdk:"retryable_errors"`
	Insecure         types.Bool   `tfsdk:"insecure"`
	RateLimiter      types.String `tfsdk:"rate_limiter"`
	AppAuth          types.List   `tfsdk:"app_auth"`
	AutoImport       types.Bool   `tfsdk:"auto_import"`
}

type appAuthModel struct {
	ID             types.String `tfsdk:"id"`
	InstallationID types.String `tfsdk:"installation_id"`
	PemFile        types.String `tfsdk:"pem_file"`
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
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("app_auth")),
				},
			},
			"owner": schema.StringAttribute{
				Description: "The GitHub owner name to manage resources under.",
				Optional:    true,
			},
			"organization": schema.StringAttribute{
				Description:        "The GitHub organization name to manage resources under.",
				DeprecationMessage: "Use 'owner' instead. This attribute is deprecated and will be removed in a future version.",
				Optional:           true,
			},
			"base_url": schema.StringAttribute{
				Description: "The GitHub base API URL",
				Optional:    true,
			},
			"parallel_requests": schema.BoolAttribute{
				Description: "Allow the provider to make parallel API calls to GitHub. You may want to set it to true when you have a private Github Enterprise without strict rate limits. Although, it is not possible to enable this setting on github.com because we enforce the respect of github.com's best practices to avoid hitting abuse rate limits. Defaults to false if not set.",
				Optional:    true,
			},
			"read_delay_ms": schema.Int64Attribute{
				Description: "Amount of time in milliseconds to sleep in between non-write requests to GitHub API. Defaults to 0ms if not set.",
				Optional:    true,
			},
			"write_delay_ms": schema.Int64Attribute{
				Description: "Amount of time in milliseconds to sleep in between writes to GitHub API. Defaults to 1000ms or 1s if not set.",
				Optional:    true,
			},
			"retry_delay_ms": schema.Int64Attribute{
				Description: "Amount of time in milliseconds to sleep in between requests to GitHub API after an error response. Defaults to 1000ms or 1s if not set, the max_retries must be set to greater than zero.",
				Optional:    true,
			},
			"max_retries": schema.Int64Attribute{
				Description: "Number of times to retry a request after receiving an error status code. Defaults to 3.",
				Optional:    true,
			},
			"retryable_errors": schema.ListAttribute{
				Description: "Allow the provider to retry after receiving an error status code, the max_retries should be set for this to work. Defaults to [500, 502, 503, 504].",
				ElementType: types.Int64Type,
				Optional:    true,
			},
			"insecure": schema.BoolAttribute{
				Description: "Enable insecure mode for testing purposes.",
				Optional:    true,
			},
			"rate_limiter": schema.StringAttribute{
				Description: "The rate limiting strategy to use. 'legacy' uses the provider's built-in rate limiting with configurable delays. 'advanced' uses go-github-ratelimit for automatic GitHub API rate limit handling. When using 'advanced', the read_delay_ms, write_delay_ms, and parallel_requests settings are ignored. Defaults to 'legacy'.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("legacy", "advanced"),
				},
			},
			"auto_import": schema.BoolAttribute{
				Description: "Enable automatic import of existing resources when creation fails with 'already exists' errors. " +
					"Can be set via GITHUB_AUTO_IMPORT environment variable. " +
					"Individual resources can override this setting. Defaults to false.",
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"app_auth": schema.ListNestedBlock{
				Description: "GitHub App authentication configuration. Cannot be used with `token`. Anonymous mode is enabled if both `token` and `app_auth` are not set.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "The GitHub App ID.",
							Required:    true,
						},
						"installation_id": schema.StringAttribute{
							Description: "The GitHub App installation ID.",
							Required:    true,
						},
						"pem_file": schema.StringAttribute{
							Description: "The GitHub App private key PEM file contents.",
							Required:    true,
							Sensitive:   true,
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
					listvalidator.ConflictsWith(path.MatchRoot("token")),
				},
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

	// Handle GitHub App authentication
	if !config.AppAuth.IsNull() && !config.AppAuth.IsUnknown() {
		var appAuthList []appAuthModel
		resp.Diagnostics.Append(config.AppAuth.ElementsAs(ctx, &appAuthList, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if len(appAuthList) > 0 {
			appAuthConfig := appAuthList[0]

			// Get app_auth configuration with environment variable fallback
			appID := os.Getenv("GITHUB_APP_ID")
			if !appAuthConfig.ID.IsNull() && !appAuthConfig.ID.IsUnknown() {
				appID = appAuthConfig.ID.ValueString()
			}

			appInstallationID := os.Getenv("GITHUB_APP_INSTALLATION_ID")
			if !appAuthConfig.InstallationID.IsNull() && !appAuthConfig.InstallationID.IsUnknown() {
				appInstallationID = appAuthConfig.InstallationID.ValueString()
			}

			appPemFile := os.Getenv("GITHUB_APP_PEM_FILE")
			if !appAuthConfig.PemFile.IsNull() && !appAuthConfig.PemFile.IsUnknown() {
				appPemFile = appAuthConfig.PemFile.ValueString()
			}

			// Validate required app_auth fields
			if appID == "" {
				resp.Diagnostics.AddError(
					"Missing GitHub App ID",
					"app_auth.id must be set and contain a non-empty value. "+
						"This can be set in the configuration or via the GITHUB_APP_ID environment variable.",
				)
				return
			}

			if appInstallationID == "" {
				resp.Diagnostics.AddError(
					"Missing GitHub App Installation ID",
					"app_auth.installation_id must be set and contain a non-empty value. "+
						"This can be set in the configuration or via the GITHUB_APP_INSTALLATION_ID environment variable.",
				)
				return
			}

			if appPemFile == "" {
				resp.Diagnostics.AddError(
					"Missing GitHub App PEM File",
					"app_auth.pem_file must be set and contain a non-empty value. "+
						"This can be set in the configuration or via the GITHUB_APP_PEM_FILE environment variable.",
				)
				return
			}

			// Handle \n replacement for PEM file (Terraform Cloud compatibility)
			appPemFile = strings.ReplaceAll(appPemFile, `\n`, "\n")

			// Generate OAuth token from GitHub App credentials
			baseURL := os.Getenv("GITHUB_BASE_URL")
			if !config.BaseURL.IsNull() && !config.BaseURL.IsUnknown() {
				baseURL = config.BaseURL.ValueString()
			}
			if baseURL == "" {
				baseURL = "https://api.github.com/"
			}

			appToken, err := GenerateOAuthTokenFromApp(baseURL, appID, appInstallationID, appPemFile)
			if err != nil {
				resp.Diagnostics.AddError(
					"GitHub App Authentication Error",
					"Unable to generate OAuth token from GitHub App credentials. "+
						"Please verify your app_auth configuration is correct.\n\n"+
						"Error: "+err.Error(),
				)
				return
			}

			token = appToken
		}
	}

	owner := os.Getenv("GITHUB_OWNER")
	if !config.Owner.IsNull() && !config.Owner.IsUnknown() {
		owner = config.Owner.ValueString()
	}

	// Handle deprecated 'organization' attribute with backward compatibility
	// organization takes precedence over owner when both are set
	organization := os.Getenv("GITHUB_ORGANIZATION")
	if !config.Organization.IsNull() && !config.Organization.IsUnknown() {
		organization = config.Organization.ValueString()
	}
	if organization != "" {
		owner = organization
	}

	baseURL := os.Getenv("GITHUB_BASE_URL")
	if !config.BaseURL.IsNull() && !config.BaseURL.IsUnknown() {
		baseURL = config.BaseURL.ValueString()
	}

	// Get parallel_requests setting
	parallelRequests := false // Default to false for github.com safety
	if !config.ParallelRequests.IsNull() && !config.ParallelRequests.IsUnknown() {
		parallelRequests = config.ParallelRequests.ValueBool()
	}

	// Get read_delay_ms setting
	readDelayMS := int64(0) // Default to 0ms
	if !config.ReadDelayMS.IsNull() && !config.ReadDelayMS.IsUnknown() {
		readDelayMS = config.ReadDelayMS.ValueInt64()
	}

	// Get write_delay_ms setting
	writeDelayMS := int64(1000) // Default to 1000ms
	if !config.WriteDelayMS.IsNull() && !config.WriteDelayMS.IsUnknown() {
		writeDelayMS = config.WriteDelayMS.ValueInt64()
	}

	// Get retry_delay_ms setting
	retryDelayMS := int64(1000) // Default to 1000ms
	if !config.RetryDelayMS.IsNull() && !config.RetryDelayMS.IsUnknown() {
		retryDelayMS = config.RetryDelayMS.ValueInt64()
	}

	// Get max_retries setting
	maxRetries := int64(3) // Default to 3
	if !config.MaxRetries.IsNull() && !config.MaxRetries.IsUnknown() {
		maxRetries = config.MaxRetries.ValueInt64()
	}

	// Get insecure setting
	insecure := false // Default to false
	if !config.Insecure.IsNull() && !config.Insecure.IsUnknown() {
		insecure = config.Insecure.ValueBool()
	}

	// Get retryable_errors setting
	retryableErrorsSlice := []int64{500, 502, 503, 504} // Default values
	if !config.RetryableErrors.IsNull() && !config.RetryableErrors.IsUnknown() {
		diags := config.RetryableErrors.ElementsAs(ctx, &retryableErrorsSlice, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Check if this is github.com to enforce parallel_requests safety
	isGithubDotCom, err := regexp.MatchString("^"+regexp.QuoteMeta("https://api.github.com"), baseURL)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Base URL",
			"An unexpected error occurred when validating the base URL: "+err.Error(),
		)
		return
	}

	if parallelRequests && isGithubDotCom {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"parallel_requests cannot be true when connecting to public github.com due to rate limiting restrictions",
		)
		return
	}

	// Validate delay settings are not negative
	if readDelayMS < 0 {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"read_delay_ms must be greater than or equal to 0",
		)
		return
	}

	if writeDelayMS <= 0 {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"write_delay_ms must be greater than 0ms",
		)
		return
	}

	if retryDelayMS < 0 {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"retry_delay_ms must be greater than or equal to 0ms",
		)
		return
	}

	if maxRetries < 0 {
		resp.Diagnostics.AddError(
			"Invalid Configuration",
			"max_retries must be greater than or equal to 0",
		)
		return
	}

	// Get rate_limiter setting
	rateLimiter := "legacy" // Default to legacy
	if !config.RateLimiter.IsNull() && !config.RateLimiter.IsUnknown() {
		rateLimiter = config.RateLimiter.ValueString()
	}

	// Get auto_import setting with environment variable fallback
	autoImport := false
	if envValue := os.Getenv("GITHUB_AUTO_IMPORT"); envValue == "true" {
		autoImport = true
	}
	if !config.AutoImport.IsNull() && !config.AutoImport.IsUnknown() {
		autoImport = config.AutoImport.ValueBool()
	}

	// Convert retryable errors to map
	retryableErrors := make(map[int]bool)
	if maxRetries > 0 {
		for _, statusCode := range retryableErrorsSlice {
			retryableErrors[int(statusCode)] = true
		}
	}

	// Create GitHub configuration with all settings
	githubConfig := &Config{
		Token:            token,
		Owner:            owner,
		BaseURL:          baseURL,
		Insecure:         insecure,
		MaxRetries:       int(maxRetries),
		ParallelRequests: parallelRequests,
		ReadDelay:        time.Duration(readDelayMS) * time.Millisecond,
		WriteDelay:       time.Duration(writeDelayMS) * time.Millisecond,
		RetryDelay:       time.Duration(retryDelayMS) * time.Millisecond,
		RetryableErrors:  retryableErrors,
		RateLimiter:      rateLimiter,
		AutoImport:       autoImport,
	}

	// Set BaseURL default if empty
	if githubConfig.BaseURL == "" {
		githubConfig.BaseURL = "https://api.github.com/"
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
