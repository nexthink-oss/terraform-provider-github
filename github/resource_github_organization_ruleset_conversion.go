package github

import (
	"context"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Convert Framework model to GitHub API object for organization ruleset
func (r *githubOrganizationRulesetResource) frameworkToAPIRuleset(ctx context.Context, model *githubOrganizationRulesetResourceModel) (*github.RepositoryRuleset, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Basic fields
	ruleset := &github.RepositoryRuleset{
		Name:        model.Name.ValueString(),
		Target:      (*github.RulesetTarget)(github.Ptr(model.Target.ValueString())),
		SourceType:  github.Ptr(github.RulesetSourceTypeOrganization),
		Enforcement: github.RulesetEnforcement(model.Enforcement.ValueString()),
	}

	// Convert bypass actors
	if !model.BypassActors.IsNull() && !model.BypassActors.IsUnknown() {
		var bypassActorModels []bypassActorModel
		diags.Append(model.BypassActors.ElementsAs(ctx, &bypassActorModels, false)...)
		if diags.HasError() {
			return nil, diags
		}

		var bypassActors []*github.BypassActor
		for _, ba := range bypassActorModels {
			actor := &github.BypassActor{
				ActorID:    github.Ptr(ba.ActorID.ValueInt64()),
				ActorType:  (*github.BypassActorType)(github.Ptr(ba.ActorType.ValueString())),
				BypassMode: (*github.BypassMode)(github.Ptr(ba.BypassMode.ValueString())),
			}
			bypassActors = append(bypassActors, actor)
		}
		ruleset.BypassActors = bypassActors
	}

	// Convert organization conditions (more complex than repository)
	if !model.Conditions.IsNull() && !model.Conditions.IsUnknown() && len(model.Conditions.Elements()) > 0 {
		var conditionsModels []organizationConditionsModel
		diags.Append(model.Conditions.ElementsAs(ctx, &conditionsModels, false)...)
		if diags.HasError() {
			return nil, diags
		}

		if len(conditionsModels) > 0 {
			conditionsModel := conditionsModels[0]
			conditions := &github.RepositoryRulesetConditions{}

			// Ref name condition
			if !conditionsModel.RefName.IsNull() && !conditionsModel.RefName.IsUnknown() && len(conditionsModel.RefName.Elements()) > 0 {
				var refNameModels []refNameModel
				diags.Append(conditionsModel.RefName.ElementsAs(ctx, &refNameModels, false)...)
				if diags.HasError() {
					return nil, diags
				}

				if len(refNameModels) > 0 {
					refNameModel := refNameModels[0]

					var include []string
					diags.Append(refNameModel.Include.ElementsAs(ctx, &include, false)...)
					if diags.HasError() {
						return nil, diags
					}

					var exclude []string
					diags.Append(refNameModel.Exclude.ElementsAs(ctx, &exclude, false)...)
					if diags.HasError() {
						return nil, diags
					}

					conditions.RefName = &github.RepositoryRulesetRefConditionParameters{
						Include: include,
						Exclude: exclude,
					}
				}
			}

			// Repository name condition
			if !conditionsModel.RepositoryName.IsNull() && !conditionsModel.RepositoryName.IsUnknown() && len(conditionsModel.RepositoryName.Elements()) > 0 {
				var repositoryNameModels []repositoryNameModel
				diags.Append(conditionsModel.RepositoryName.ElementsAs(ctx, &repositoryNameModels, false)...)
				if diags.HasError() {
					return nil, diags
				}

				if len(repositoryNameModels) > 0 {
					repositoryNameModel := repositoryNameModels[0]

					var include []string
					diags.Append(repositoryNameModel.Include.ElementsAs(ctx, &include, false)...)
					if diags.HasError() {
						return nil, diags
					}

					var exclude []string
					diags.Append(repositoryNameModel.Exclude.ElementsAs(ctx, &exclude, false)...)
					if diags.HasError() {
						return nil, diags
					}

					conditions.RepositoryName = &github.RepositoryRulesetRepositoryNamesConditionParameters{
						Include:   include,
						Exclude:   exclude,
						Protected: github.Ptr(repositoryNameModel.Protected.ValueBool()),
					}
				}
			}

			// Repository ID condition
			if !conditionsModel.RepositoryID.IsNull() && !conditionsModel.RepositoryID.IsUnknown() && len(conditionsModel.RepositoryID.Elements()) > 0 {
				var repositoryIDs []int64
				diags.Append(conditionsModel.RepositoryID.ElementsAs(ctx, &repositoryIDs, false)...)
				if diags.HasError() {
					return nil, diags
				}

				conditions.RepositoryID = &github.RepositoryRulesetRepositoryIDsConditionParameters{
					RepositoryIDs: repositoryIDs,
				}
			}

			ruleset.Conditions = conditions
		}
	}

	// Convert rules - reuse logic from repository ruleset
	if !model.Rules.IsNull() && !model.Rules.IsUnknown() && len(model.Rules.Elements()) > 0 {
		var rulesModels []rulesModel
		diags.Append(model.Rules.ElementsAs(ctx, &rulesModels, false)...)
		if diags.HasError() {
			return nil, diags
		}

		if len(rulesModels) > 0 {
			rulesModel := rulesModels[0]
			rules := &github.RepositoryRulesetRules{}

			// Convert each rule type
			if !rulesModel.Creation.IsNull() && rulesModel.Creation.ValueBool() {
				rules.Creation = &github.EmptyRuleParameters{}
			}

			if !rulesModel.Update.IsNull() && rulesModel.Update.ValueBool() {
				rules.Update = &github.UpdateRuleParameters{
					UpdateAllowsFetchAndMerge: false, // Organization rulesets don't support this
				}
			}

			if !rulesModel.Deletion.IsNull() && rulesModel.Deletion.ValueBool() {
				rules.Deletion = &github.EmptyRuleParameters{}
			}

			if !rulesModel.RequiredLinearHistory.IsNull() && rulesModel.RequiredLinearHistory.ValueBool() {
				rules.RequiredLinearHistory = &github.EmptyRuleParameters{}
			}

			if !rulesModel.RequiredSignatures.IsNull() && rulesModel.RequiredSignatures.ValueBool() {
				rules.RequiredSignatures = &github.EmptyRuleParameters{}
			}

			if !rulesModel.NonFastForward.IsNull() && rulesModel.NonFastForward.ValueBool() {
				rules.NonFastForward = &github.EmptyRuleParameters{}
			}

			// Convert pull request rules
			if !rulesModel.PullRequest.IsNull() && !rulesModel.PullRequest.IsUnknown() && len(rulesModel.PullRequest.Elements()) > 0 {
				var pullRequestModels []pullRequestModel
				diags.Append(rulesModel.PullRequest.ElementsAs(ctx, &pullRequestModels, false)...)
				if diags.HasError() {
					return nil, diags
				}

				if len(pullRequestModels) > 0 {
					prModel := pullRequestModels[0]

					// Build allowed merge methods based on the boolean fields
					var allowedMergeMethods []github.PullRequestMergeMethod
					if prModel.AllowMergeCommit.IsNull() || prModel.AllowMergeCommit.ValueBool() {
						allowedMergeMethods = append(allowedMergeMethods, github.PullRequestMergeMethodMerge)
					}
					if prModel.AllowSquashMerge.IsNull() || prModel.AllowSquashMerge.ValueBool() {
						allowedMergeMethods = append(allowedMergeMethods, github.PullRequestMergeMethodSquash)
					}
					if prModel.AllowRebaseMerge.IsNull() || prModel.AllowRebaseMerge.ValueBool() {
						allowedMergeMethods = append(allowedMergeMethods, github.PullRequestMergeMethodRebase)
					}

					prRule := &github.PullRequestRuleParameters{
						AllowedMergeMethods:            allowedMergeMethods,
						DismissStaleReviewsOnPush:      prModel.DismissStaleReviewsOnPush.ValueBool(),
						RequireCodeOwnerReview:         prModel.RequireCodeOwnerReview.ValueBool(),
						RequireLastPushApproval:        prModel.RequireLastPushApproval.ValueBool(),
						RequiredApprovingReviewCount:   int(prModel.RequiredApprovingReviewCount.ValueInt64()),
						RequiredReviewThreadResolution: prModel.RequiredReviewThreadResolution.ValueBool(),
					}

					// Handle optional automatic_copilot_code_review_enabled
					if !prModel.AutomaticCopilotCodeReviewEnabled.IsNull() {
						copilotEnabled := prModel.AutomaticCopilotCodeReviewEnabled.ValueBool()
						prRule.AutomaticCopilotCodeReviewEnabled = &copilotEnabled
					}

					rules.PullRequest = prRule
				}
			}

			// Convert required status checks
			if !rulesModel.RequiredStatusChecks.IsNull() && !rulesModel.RequiredStatusChecks.IsUnknown() && len(rulesModel.RequiredStatusChecks.Elements()) > 0 {
				var requiredStatusChecksModels []rulesetRequiredStatusChecksModel
				diags.Append(rulesModel.RequiredStatusChecks.ElementsAs(ctx, &requiredStatusChecksModels, false)...)
				if diags.HasError() {
					return nil, diags
				}

				if len(requiredStatusChecksModels) > 0 {
					rscModel := requiredStatusChecksModels[0]
					rscParams := &github.RequiredStatusChecksRuleParameters{}

					// Convert required checks
					if !rscModel.RequiredCheck.IsNull() && !rscModel.RequiredCheck.IsUnknown() {
						var requiredCheckModels []requiredCheckModel
						diags.Append(rscModel.RequiredCheck.ElementsAs(ctx, &requiredCheckModels, false)...)
						if diags.HasError() {
							return nil, diags
						}

						var requiredChecks []*github.RuleStatusCheck
						for _, rc := range requiredCheckModels {
							check := &github.RuleStatusCheck{
								Context: rc.Context.ValueString(),
							}
							if !rc.IntegrationID.IsNull() && rc.IntegrationID.ValueInt64() != 0 {
								check.IntegrationID = github.Ptr(rc.IntegrationID.ValueInt64())
							}
							requiredChecks = append(requiredChecks, check)
						}
						rscParams.RequiredStatusChecks = requiredChecks
					}

					if !rscModel.StrictRequiredStatusChecksPolicy.IsNull() {
						rscParams.StrictRequiredStatusChecksPolicy = rscModel.StrictRequiredStatusChecksPolicy.ValueBool()
					}
					if !rscModel.DoNotEnforceOnCreate.IsNull() {
						doNotEnforce := rscModel.DoNotEnforceOnCreate.ValueBool()
						rscParams.DoNotEnforceOnCreate = &doNotEnforce
					}

					rules.RequiredStatusChecks = rscParams
				}
			}

			// TODO: Add pattern rules (commit message, email, branch/tag patterns)
			// These need to be implemented with correct GitHub API types

			// TODO: Convert required workflows
			// TODO: Convert required code scanning

			ruleset.Rules = rules
		}
	}

	return ruleset, diags
}

// Convert GitHub API response to Framework model for organization ruleset
func (r *githubOrganizationRulesetResource) apiToFrameworkRuleset(ctx context.Context, ruleset *github.RepositoryRuleset, model *githubOrganizationRulesetResourceModel, diags *diag.Diagnostics) {
	model.Name = types.StringValue(ruleset.Name)
	if ruleset.Target != nil {
		model.Target = types.StringValue(string(*ruleset.Target))
	}
	model.Enforcement = types.StringValue(string(ruleset.Enforcement))
	model.NodeID = types.StringValue(ruleset.GetNodeID())
	model.RulesetID = types.Int64Value(ruleset.GetID())

	// Convert bypass actors
	if ruleset.BypassActors != nil {
		bypassActorElements := make([]attr.Value, len(ruleset.BypassActors))
		for i, ba := range ruleset.BypassActors {
			bypassActorAttributes := map[string]attr.Value{
				"actor_id":    types.Int64Value(ba.GetActorID()),
				"actor_type":  types.StringValue(string(*ba.GetActorType())),
				"bypass_mode": types.StringValue(string(*ba.GetBypassMode())),
			}
			bypassActorElements[i] = types.ObjectValueMust(
				map[string]attr.Type{
					"actor_id":    types.Int64Type,
					"actor_type":  types.StringType,
					"bypass_mode": types.StringType,
				},
				bypassActorAttributes,
			)
		}
		model.BypassActors = types.ListValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"actor_id":    types.Int64Type,
					"actor_type":  types.StringType,
					"bypass_mode": types.StringType,
				},
			},
			bypassActorElements,
		)
	} else {
		model.BypassActors = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"actor_id":    types.Int64Type,
				"actor_type":  types.StringType,
				"bypass_mode": types.StringType,
			},
		})
	}

	// Convert conditions (more complex for organization rulesets)
	r.convertConditionsFromAPI(ctx, ruleset.GetConditions(), model, diags)

	// Convert rules
	r.convertRulesFromAPI(ctx, ruleset.GetRules(), model, diags)
}

// TODO: Helper functions for pattern rules, workflows, and code scanning
// These need to be implemented with correct GitHub API types

// Helper function to convert conditions from API
func (r *githubOrganizationRulesetResource) convertConditionsFromAPI(ctx context.Context, conditions *github.RepositoryRulesetConditions, model *githubOrganizationRulesetResourceModel, diags *diag.Diagnostics) {
	if conditions == nil {
		model.Conditions = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"ref_name":        types.ListType{ElemType: types.ObjectType{}},
				"repository_name": types.ListType{ElemType: types.ObjectType{}},
				"repository_id":   types.ListType{ElemType: types.Int64Type},
			},
		})
		return
	}

	// Build the conditions object
	conditionsAttrs := map[string]attr.Value{}

	// Ref name condition
	if conditions.RefName != nil {
		refNameAttrs := map[string]attr.Value{
			"include": types.ListValueMust(types.StringType, convertStringSliceToAttrValues(conditions.RefName.Include)),
			"exclude": types.ListValueMust(types.StringType, convertStringSliceToAttrValues(conditions.RefName.Exclude)),
		}
		refNameObj := types.ObjectValueMust(
			map[string]attr.Type{
				"include": types.ListType{ElemType: types.StringType},
				"exclude": types.ListType{ElemType: types.StringType},
			},
			refNameAttrs,
		)
		conditionsAttrs["ref_name"] = types.ListValueMust(
			types.ObjectType{AttrTypes: map[string]attr.Type{
				"include": types.ListType{ElemType: types.StringType},
				"exclude": types.ListType{ElemType: types.StringType},
			}},
			[]attr.Value{refNameObj},
		)
	} else {
		conditionsAttrs["ref_name"] = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"include": types.ListType{ElemType: types.StringType},
				"exclude": types.ListType{ElemType: types.StringType},
			},
		})
	}

	// Repository name condition
	if conditions.RepositoryName != nil {
		repositoryNameAttrs := map[string]attr.Value{
			"include":   types.ListValueMust(types.StringType, convertStringSliceToAttrValues(conditions.RepositoryName.Include)),
			"exclude":   types.ListValueMust(types.StringType, convertStringSliceToAttrValues(conditions.RepositoryName.Exclude)),
			"protected": types.BoolValue(conditions.RepositoryName.GetProtected()),
		}
		repositoryNameObj := types.ObjectValueMust(
			map[string]attr.Type{
				"include":   types.ListType{ElemType: types.StringType},
				"exclude":   types.ListType{ElemType: types.StringType},
				"protected": types.BoolType,
			},
			repositoryNameAttrs,
		)
		conditionsAttrs["repository_name"] = types.ListValueMust(
			types.ObjectType{AttrTypes: map[string]attr.Type{
				"include":   types.ListType{ElemType: types.StringType},
				"exclude":   types.ListType{ElemType: types.StringType},
				"protected": types.BoolType,
			}},
			[]attr.Value{repositoryNameObj},
		)
	} else {
		conditionsAttrs["repository_name"] = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"include":   types.ListType{ElemType: types.StringType},
				"exclude":   types.ListType{ElemType: types.StringType},
				"protected": types.BoolType,
			},
		})
	}

	// Repository ID condition
	if conditions.RepositoryID != nil && len(conditions.RepositoryID.RepositoryIDs) > 0 {
		repositoryIDValues := make([]attr.Value, len(conditions.RepositoryID.RepositoryIDs))
		for i, id := range conditions.RepositoryID.RepositoryIDs {
			repositoryIDValues[i] = types.Int64Value(id)
		}
		conditionsAttrs["repository_id"] = types.ListValueMust(types.Int64Type, repositoryIDValues)
	} else {
		conditionsAttrs["repository_id"] = types.ListNull(types.Int64Type)
	}

	conditionsObj := types.ObjectValueMust(
		map[string]attr.Type{
			"ref_name":        types.ListType{ElemType: types.ObjectType{}},
			"repository_name": types.ListType{ElemType: types.ObjectType{}},
			"repository_id":   types.ListType{ElemType: types.Int64Type},
		},
		conditionsAttrs,
	)

	model.Conditions = types.ListValueMust(
		types.ObjectType{AttrTypes: map[string]attr.Type{
			"ref_name":        types.ListType{ElemType: types.ObjectType{}},
			"repository_name": types.ListType{ElemType: types.ObjectType{}},
			"repository_id":   types.ListType{ElemType: types.Int64Type},
		}},
		[]attr.Value{conditionsObj},
	)
}

// Helper function to convert rules from API
func (r *githubOrganizationRulesetResource) convertRulesFromAPI(ctx context.Context, rules *github.RepositoryRulesetRules, model *githubOrganizationRulesetResourceModel, diags *diag.Diagnostics) {
	if rules == nil {
		model.Rules = types.ListNull(types.ObjectType{AttrTypes: map[string]attr.Type{}}) // Simplified for now
		return
	}

	// Build rules object with all the boolean fields and nested blocks
	rulesAttrs := map[string]attr.Value{
		"creation":                types.BoolNull(),
		"update":                  types.BoolNull(),
		"deletion":                types.BoolNull(),
		"required_linear_history": types.BoolNull(),
		"required_signatures":     types.BoolNull(),
		"non_fast_forward":        types.BoolNull(),
		"pull_request":            types.ListNull(types.ObjectType{}),
		"required_status_checks":  types.ListNull(types.ObjectType{}),
		// Add other rule attributes as needed
	}

	if rules.Creation != nil {
		rulesAttrs["creation"] = types.BoolValue(true)
	}
	if rules.Update != nil {
		rulesAttrs["update"] = types.BoolValue(true)
	}
	if rules.Deletion != nil {
		rulesAttrs["deletion"] = types.BoolValue(true)
	}
	if rules.RequiredLinearHistory != nil {
		rulesAttrs["required_linear_history"] = types.BoolValue(true)
	}
	if rules.RequiredSignatures != nil {
		rulesAttrs["required_signatures"] = types.BoolValue(true)
	}
	if rules.NonFastForward != nil {
		rulesAttrs["non_fast_forward"] = types.BoolValue(true)
	}

	// For complex nested rules like pull_request, required_status_checks, etc.,
	// we would need to add more detailed conversion logic here
	// For now, keeping it simple to match the basic structure

	rulesObj := types.ObjectValueMust(
		map[string]attr.Type{
			"creation":                types.BoolType,
			"update":                  types.BoolType,
			"deletion":                types.BoolType,
			"required_linear_history": types.BoolType,
			"required_signatures":     types.BoolType,
			"non_fast_forward":        types.BoolType,
			"pull_request":            types.ListType{ElemType: types.ObjectType{}},
			"required_status_checks":  types.ListType{ElemType: types.ObjectType{}},
		},
		rulesAttrs,
	)

	model.Rules = types.ListValueMust(
		types.ObjectType{AttrTypes: map[string]attr.Type{
			"creation":                types.BoolType,
			"update":                  types.BoolType,
			"deletion":                types.BoolType,
			"required_linear_history": types.BoolType,
			"required_signatures":     types.BoolType,
			"non_fast_forward":        types.BoolType,
			"pull_request":            types.ListType{ElemType: types.ObjectType{}},
			"required_status_checks":  types.ListType{ElemType: types.ObjectType{}},
		}},
		[]attr.Value{rulesObj},
	)
}

// Helper function to convert string slices to attr values
func convertStringSliceToAttrValues(strings []string) []attr.Value {
	values := make([]attr.Value, len(strings))
	for i, s := range strings {
		values[i] = types.StringValue(s)
	}
	return values
}
