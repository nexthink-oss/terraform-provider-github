package github

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

)

var (
	_ resource.Resource                = &githubRepositoryDependabotSecurityUpdatesResource{}
	_ resource.ResourceWithConfigure   = &githubRepositoryDependabotSecurityUpdatesResource{}
	_ resource.ResourceWithImportState = &githubRepositoryDependabotSecurityUpdatesResource{}
)

type githubRepositoryDependabotSecurityUpdatesResource struct {
	client *Owner
}

type githubRepositoryDependabotSecurityUpdatesResourceModel struct {
	ID         types.String `tfsdk:"id"`
	Repository types.String `tfsdk:"repository"`
	Enabled    types.Bool   `tfsdk:"enabled"`
}

func NewGithubRepositoryDependabotSecurityUpdatesResource() resource.Resource {
	return &githubRepositoryDependabotSecurityUpdatesResource{}
}

func (r *githubRepositoryDependabotSecurityUpdatesResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_dependabot_security_updates"
}

func (r *githubRepositoryDependabotSecurityUpdatesResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Provides a GitHub repository Dependabot security updates resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The repository name.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"repository": schema.StringAttribute{
				Description: "The GitHub repository.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Description: "The state of the automated security fixes.",
				Required:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *githubRepositoryDependabotSecurityUpdatesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
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

func (r *githubRepositoryDependabotSecurityUpdatesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data githubRepositoryDependabotSecurityUpdatesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()
	enabled := data.Enabled.ValueBool()

	var err error
	if enabled {
		_, err = client.Repositories.EnableAutomatedSecurityFixes(ctx, owner, repoName)
	} else {
		_, err = client.Repositories.DisableAutomatedSecurityFixes(ctx, owner, repoName)
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Repository Dependabot Security Updates",
			fmt.Sprintf("An unexpected error occurred when creating the repository dependabot security updates: %s", err.Error()),
		)
		return
	}

	data.ID = types.StringValue(repoName)

	// Read the resource to ensure state consistency
	r.readRepositoryDependabotSecurityUpdates(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryDependabotSecurityUpdatesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data githubRepositoryDependabotSecurityUpdatesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readRepositoryDependabotSecurityUpdates(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryDependabotSecurityUpdatesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data githubRepositoryDependabotSecurityUpdatesResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()
	enabled := data.Enabled.ValueBool()

	var err error
	if enabled {
		_, err = client.Repositories.EnableAutomatedSecurityFixes(ctx, owner, repoName)
	} else {
		_, err = client.Repositories.DisableAutomatedSecurityFixes(ctx, owner, repoName)
	}

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Repository Dependabot Security Updates",
			fmt.Sprintf("An unexpected error occurred when updating the repository dependabot security updates: %s", err.Error()),
		)
		return
	}

	// Read the resource to ensure state consistency
	r.readRepositoryDependabotSecurityUpdates(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryDependabotSecurityUpdatesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data githubRepositoryDependabotSecurityUpdatesResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()

	_, err := client.Repositories.DisableAutomatedSecurityFixes(ctx, owner, repoName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Repository Dependabot Security Updates",
			fmt.Sprintf("An unexpected error occurred when deleting the repository dependabot security updates: %s", err.Error()),
		)
		return
	}

	// Verify the resource was successfully deleted by reading it
	r.readRepositoryDependabotSecurityUpdates(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *githubRepositoryDependabotSecurityUpdatesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	repoName := req.ID

	data := githubRepositoryDependabotSecurityUpdatesResourceModel{
		ID:         types.StringValue(repoName),
		Repository: types.StringValue(repoName),
	}

	r.readRepositoryDependabotSecurityUpdates(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *githubRepositoryDependabotSecurityUpdatesResource) readRepositoryDependabotSecurityUpdates(ctx context.Context, data *githubRepositoryDependabotSecurityUpdatesResourceModel, diagnostics *diag.Diagnostics) {
	client := r.client.V3Client()
	owner := r.client.Name()

	repoName := data.Repository.ValueString()

	automatedSecurityFixes, _, err := client.Repositories.GetAutomatedSecurityFixes(ctx, owner, repoName)
	if err != nil {
		diagnostics.AddError(
			"Unable to Read Repository Dependabot Security Updates",
			fmt.Sprintf("An unexpected error occurred when reading the repository dependabot security updates: %s", err.Error()),
		)
		return
	}

	data.Enabled = types.BoolValue(automatedSecurityFixes.GetEnabled())
}
