package github

import (
	"context"

	"github.com/google/go-github/v74/github"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Convert Framework model to GitHub API object
func (r *githubRepositoryRulesetResource) frameworkToAPIRuleset(ctx context.Context, model *githubRepositoryRulesetResourceModel) (*github.RepositoryRuleset, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Basic fields
	ruleset := &github.RepositoryRuleset{
		Name:        model.Name.ValueString(),
		Target:      (*github.RulesetTarget)(github.Ptr(model.Target.ValueString())),
		Source:      model.Repository.ValueString(),
		SourceType:  github.Ptr(github.RulesetSourceTypeRepository),
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

	// Convert conditions
	if !model.Conditions.IsNull() && !model.Conditions.IsUnknown() && len(model.Conditions.Elements()) > 0 {
		var conditionsModels []conditionsModel
		diags.Append(model.Conditions.ElementsAs(ctx, &conditionsModels, false)...)
		if diags.HasError() {
			return nil, diags
		}

		if len(conditionsModels) > 0 {
			conditionsModel := conditionsModels[0]
			conditions := &github.RepositoryRulesetConditions{}

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

			ruleset.Conditions = conditions
		}
	}

	// Convert rules
	if !model.Rules.IsNull() && !model.Rules.IsUnknown() && len(model.Rules.Elements()) > 0 {
		var rulesModels []rulesModel
		diags.Append(model.Rules.ElementsAs(ctx, &rulesModels, false)...)
		if diags.HasError() {
			return nil, diags
		}

		if len(rulesModels) > 0 {
			rulesModel := rulesModels[0]
			rules := &github.RepositoryRulesetRules{}

			// Simple boolean rules
			if !rulesModel.Creation.IsNull() && rulesModel.Creation.ValueBool() {
				rules.Creation = &github.EmptyRuleParameters{}
			}

			if !rulesModel.Update.IsNull() && rulesModel.Update.ValueBool() {
				updateAllowsFetchAndMerge := false
				if !rulesModel.UpdateAllowsFetchAndMerge.IsNull() {
					updateAllowsFetchAndMerge = rulesModel.UpdateAllowsFetchAndMerge.ValueBool()
				}
				rules.Update = &github.UpdateRuleParameters{
					UpdateAllowsFetchAndMerge: updateAllowsFetchAndMerge,
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

			// Required deployments
			if !rulesModel.RequiredDeployments.IsNull() && !rulesModel.RequiredDeployments.IsUnknown() && len(rulesModel.RequiredDeployments.Elements()) > 0 {
				var reqDeploymentModels []requiredDeploymentsModel
				diags.Append(rulesModel.RequiredDeployments.ElementsAs(ctx, &reqDeploymentModels, false)...)
				if diags.HasError() {
					return nil, diags
				}

				if len(reqDeploymentModels) > 0 {
					reqDepModel := reqDeploymentModels[0]
					var envs []string
					diags.Append(reqDepModel.RequiredDeploymentEnvironments.ElementsAs(ctx, &envs, false)...)
					if diags.HasError() {
						return nil, diags
					}

					rules.RequiredDeployments = &github.RequiredDeploymentsRuleParameters{
						RequiredDeploymentEnvironments: envs,
					}
				}
			}

			// Pull request rules
			if !rulesModel.PullRequest.IsNull() && !rulesModel.PullRequest.IsUnknown() && len(rulesModel.PullRequest.Elements()) > 0 {
				var prModels []pullRequestModel
				diags.Append(rulesModel.PullRequest.ElementsAs(ctx, &prModels, false)...)
				if diags.HasError() {
					return nil, diags
				}

				if len(prModels) > 0 {
					prModel := prModels[0]

					// Build AllowedMergeMethods array from boolean fields
					allowedMergeMethods := make([]github.PullRequestMergeMethod, 0)

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

			// Required status checks
			if !rulesModel.RequiredStatusChecks.IsNull() && !rulesModel.RequiredStatusChecks.IsUnknown() && len(rulesModel.RequiredStatusChecks.Elements()) > 0 {
				var rscModels []rulesetRequiredStatusChecksModel
				diags.Append(rulesModel.RequiredStatusChecks.ElementsAs(ctx, &rscModels, false)...)
				if diags.HasError() {
					return nil, diags
				}

				if len(rscModels) > 0 {
					rscModel := rscModels[0]

					var requiredStatusChecks []*github.RuleStatusCheck
					if !rscModel.RequiredCheck.IsNull() && !rscModel.RequiredCheck.IsUnknown() {
						var requiredCheckModels []requiredCheckModel
						diags.Append(rscModel.RequiredCheck.ElementsAs(ctx, &requiredCheckModels, false)...)
						if diags.HasError() {
							return nil, diags
						}

						for _, rcModel := range requiredCheckModels {
							statusCheck := &github.RuleStatusCheck{
								Context: rcModel.Context.ValueString(),
							}

							if !rcModel.IntegrationID.IsNull() && rcModel.IntegrationID.ValueInt64() != 0 {
								statusCheck.IntegrationID = github.Ptr(rcModel.IntegrationID.ValueInt64())
							}

							requiredStatusChecks = append(requiredStatusChecks, statusCheck)
						}
					}

					doNotEnforceOnCreate := rscModel.DoNotEnforceOnCreate.ValueBool()
					rules.RequiredStatusChecks = &github.RequiredStatusChecksRuleParameters{
						RequiredStatusChecks:             requiredStatusChecks,
						StrictRequiredStatusChecksPolicy: rscModel.StrictRequiredStatusChecksPolicy.ValueBool(),
						DoNotEnforceOnCreate:             &doNotEnforceOnCreate,
					}
				}
			}

			// Merge queue
			if !rulesModel.MergeQueue.IsNull() && !rulesModel.MergeQueue.IsUnknown() && len(rulesModel.MergeQueue.Elements()) > 0 {
				var mqModels []mergeQueueModel
				diags.Append(rulesModel.MergeQueue.ElementsAs(ctx, &mqModels, false)...)
				if diags.HasError() {
					return nil, diags
				}

				if len(mqModels) > 0 {
					mqModel := mqModels[0]
					rules.MergeQueue = &github.MergeQueueRuleParameters{
						CheckResponseTimeoutMinutes:  int(mqModel.CheckResponseTimeoutMinutes.ValueInt64()),
						GroupingStrategy:             github.MergeGroupingStrategy(mqModel.GroupingStrategy.ValueString()),
						MaxEntriesToBuild:            int(mqModel.MaxEntriesToBuild.ValueInt64()),
						MaxEntriesToMerge:            int(mqModel.MaxEntriesToMerge.ValueInt64()),
						MergeMethod:                  github.MergeQueueMergeMethod(mqModel.MergeMethod.ValueString()),
						MinEntriesToMerge:            int(mqModel.MinEntriesToMerge.ValueInt64()),
						MinEntriesToMergeWaitMinutes: int(mqModel.MinEntriesToMergeWaitMinutes.ValueInt64()),
					}
				}
			}

			// Pattern rules
			patternRules := []struct {
				model *types.List
				field **github.PatternRuleParameters
			}{
				{&rulesModel.CommitMessagePattern, &rules.CommitMessagePattern},
				{&rulesModel.CommitAuthorEmailPattern, &rules.CommitAuthorEmailPattern},
				{&rulesModel.CommitterEmailPattern, &rules.CommitterEmailPattern},
				{&rulesModel.BranchNamePattern, &rules.BranchNamePattern},
				{&rulesModel.TagNamePattern, &rules.TagNamePattern},
			}

			for _, rule := range patternRules {
				if !rule.model.IsNull() && !rule.model.IsUnknown() && len(rule.model.Elements()) > 0 {
					var patternModels []patternRuleModel
					diags.Append(rule.model.ElementsAs(ctx, &patternModels, false)...)
					if diags.HasError() {
						return nil, diags
					}

					if len(patternModels) > 0 {
						patternModel := patternModels[0]
						name := patternModel.Name.ValueString()
						negate := patternModel.Negate.ValueBool()

						*rule.field = &github.PatternRuleParameters{
							Name:     &name,
							Negate:   &negate,
							Operator: github.PatternRuleOperator(patternModel.Operator.ValueString()),
							Pattern:  patternModel.Pattern.ValueString(),
						}
					}
				}
			}

			// Required code scanning
			if !rulesModel.RequiredCodeScanning.IsNull() && !rulesModel.RequiredCodeScanning.IsUnknown() && len(rulesModel.RequiredCodeScanning.Elements()) > 0 {
				var rcsModels []requiredCodeScanningModel
				diags.Append(rulesModel.RequiredCodeScanning.ElementsAs(ctx, &rcsModels, false)...)
				if diags.HasError() {
					return nil, diags
				}

				if len(rcsModels) > 0 {
					rcsModel := rcsModels[0]

					var codeScanningTools []*github.RuleCodeScanningTool
					if !rcsModel.RequiredCodeScanningTool.IsNull() && !rcsModel.RequiredCodeScanningTool.IsUnknown() {
						var toolModels []requiredCodeScanningToolModel
						diags.Append(rcsModel.RequiredCodeScanningTool.ElementsAs(ctx, &toolModels, false)...)
						if diags.HasError() {
							return nil, diags
						}

						for _, toolModel := range toolModels {
							tool := &github.RuleCodeScanningTool{
								AlertsThreshold:         github.CodeScanningAlertsThreshold(toolModel.AlertsThreshold.ValueString()),
								SecurityAlertsThreshold: github.CodeScanningSecurityAlertsThreshold(toolModel.SecurityAlertsThreshold.ValueString()),
								Tool:                    toolModel.Tool.ValueString(),
							}
							codeScanningTools = append(codeScanningTools, tool)
						}
					}

					rules.CodeScanning = &github.CodeScanningRuleParameters{
						CodeScanningTools: codeScanningTools,
					}
				}
			}

			ruleset.Rules = rules
		}
	}

	return ruleset, diags
}

// Convert GitHub API object to Framework model
func (r *githubRepositoryRulesetResource) apiToFrameworkRuleset(ctx context.Context, ruleset *github.RepositoryRuleset, repoName string) (*githubRepositoryRulesetResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := &githubRepositoryRulesetResourceModel{
		Name:        types.StringValue(ruleset.Name),
		Target:      types.StringValue(string(*ruleset.Target)),
		Repository:  types.StringValue(repoName),
		Enforcement: types.StringValue(string(ruleset.Enforcement)),
		NodeID:      types.StringValue(ruleset.GetNodeID()),
		RulesetID:   types.Int64Value(ruleset.GetID()),
	}

	// Convert bypass actors
	if len(ruleset.BypassActors) > 0 {
		bypassActorElements := make([]attr.Value, 0, len(ruleset.BypassActors))
		bypassActorType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"actor_id":    types.Int64Type,
				"actor_type":  types.StringType,
				"bypass_mode": types.StringType,
			},
		}

		for _, ba := range ruleset.BypassActors {
			bypassActorValue, diagsInner := types.ObjectValue(bypassActorType.AttrTypes, map[string]attr.Value{
				"actor_id":    types.Int64Value(ba.GetActorID()),
				"actor_type":  types.StringValue(string(*ba.ActorType)),
				"bypass_mode": types.StringValue(string(*ba.BypassMode)),
			})
			diags.Append(diagsInner...)
			if diags.HasError() {
				return nil, diags
			}
			bypassActorElements = append(bypassActorElements, bypassActorValue)
		}

		bypassActorsList, diagsInner := types.ListValue(bypassActorType, bypassActorElements)
		diags.Append(diagsInner...)
		if diags.HasError() {
			return nil, diags
		}
		model.BypassActors = bypassActorsList
	} else {
		model.BypassActors = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"actor_id":    types.Int64Type,
				"actor_type":  types.StringType,
				"bypass_mode": types.StringType,
			},
		})
	}

	// Convert conditions
	if ruleset.GetConditions() != nil && ruleset.GetConditions().RefName != nil {
		refNameType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"include": types.ListType{ElemType: types.StringType},
				"exclude": types.ListType{ElemType: types.StringType},
			},
		}

		includeList, diagsInner := types.ListValueFrom(ctx, types.StringType, ruleset.GetConditions().RefName.Include)
		diags.Append(diagsInner...)
		if diags.HasError() {
			return nil, diags
		}

		excludeList, diagsInner := types.ListValueFrom(ctx, types.StringType, ruleset.GetConditions().RefName.Exclude)
		diags.Append(diagsInner...)
		if diags.HasError() {
			return nil, diags
		}

		refNameValue, diagsInner := types.ObjectValue(refNameType.AttrTypes, map[string]attr.Value{
			"include": includeList,
			"exclude": excludeList,
		})
		diags.Append(diagsInner...)
		if diags.HasError() {
			return nil, diags
		}

		refNameList, diagsInner := types.ListValue(refNameType, []attr.Value{refNameValue})
		diags.Append(diagsInner...)
		if diags.HasError() {
			return nil, diags
		}

		conditionsType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"ref_name": types.ListType{ElemType: refNameType},
			},
		}

		conditionsValue, diagsInner := types.ObjectValue(conditionsType.AttrTypes, map[string]attr.Value{
			"ref_name": refNameList,
		})
		diags.Append(diagsInner...)
		if diags.HasError() {
			return nil, diags
		}

		conditionsList, diagsInner := types.ListValue(conditionsType, []attr.Value{conditionsValue})
		diags.Append(diagsInner...)
		if diags.HasError() {
			return nil, diags
		}
		model.Conditions = conditionsList
	} else {
		model.Conditions = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"ref_name": types.ListType{ElemType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"include": types.ListType{ElemType: types.StringType},
						"exclude": types.ListType{ElemType: types.StringType},
					},
				}},
			},
		})
	}

	// Convert rules
	if ruleset.Rules != nil {
		rulesValue, rulesConvertDiags := r.apiRulesToFrameworkRules(ctx, ruleset.Rules)
		diags.Append(rulesConvertDiags...)
		if diags.HasError() {
			return nil, diags
		}
		model.Rules = rulesValue
	} else {
		model.Rules = types.ListNull(types.ObjectType{AttrTypes: r.getRulesAttrTypes()})
	}

	return model, diags
}

// Helper function to convert API rules to framework rules
func (r *githubRepositoryRulesetResource) apiRulesToFrameworkRules(ctx context.Context, rules *github.RepositoryRulesetRules) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	rulesAttrTypes := r.getRulesAttrTypes()

	// Build the rules object
	rulesAttrs := make(map[string]attr.Value)

	// Simple boolean rules
	rulesAttrs["creation"] = types.BoolValue(rules.Creation != nil)
	rulesAttrs["update"] = types.BoolValue(rules.Update != nil)
	if rules.Update != nil {
		rulesAttrs["update_allows_fetch_and_merge"] = types.BoolValue(rules.Update.UpdateAllowsFetchAndMerge)
	} else {
		rulesAttrs["update_allows_fetch_and_merge"] = types.BoolValue(false)
	}
	rulesAttrs["deletion"] = types.BoolValue(rules.Deletion != nil)
	rulesAttrs["required_linear_history"] = types.BoolValue(rules.RequiredLinearHistory != nil)
	rulesAttrs["required_signatures"] = types.BoolValue(rules.RequiredSignatures != nil)
	rulesAttrs["non_fast_forward"] = types.BoolValue(rules.NonFastForward != nil)

	// Required deployments
	if rules.RequiredDeployments != nil {
		envsList, diagsInner := types.ListValueFrom(ctx, types.StringType, rules.RequiredDeployments.RequiredDeploymentEnvironments)
		diags.Append(diagsInner...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
		}

		reqDepType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"required_deployment_environments": types.ListType{ElemType: types.StringType},
			},
		}

		reqDepValue, diagsInner := types.ObjectValue(reqDepType.AttrTypes, map[string]attr.Value{
			"required_deployment_environments": envsList,
		})
		diags.Append(diagsInner...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
		}

		reqDepList, diagsInner := types.ListValue(reqDepType, []attr.Value{reqDepValue})
		diags.Append(diagsInner...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
		}
		rulesAttrs["required_deployments"] = reqDepList
	} else {
		rulesAttrs["required_deployments"] = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"required_deployment_environments": types.ListType{ElemType: types.StringType},
			},
		})
	}

	// Pull request rule
	if rules.PullRequest != nil {
		prAttrs := map[string]attr.Value{
			"dismiss_stale_reviews_on_push":         types.BoolValue(rules.PullRequest.DismissStaleReviewsOnPush),
			"require_code_owner_review":             types.BoolValue(rules.PullRequest.RequireCodeOwnerReview),
			"require_last_push_approval":            types.BoolValue(rules.PullRequest.RequireLastPushApproval),
			"required_approving_review_count":       types.Int64Value(int64(rules.PullRequest.RequiredApprovingReviewCount)),
			"required_review_thread_resolution":     types.BoolValue(rules.PullRequest.RequiredReviewThreadResolution),
			"automatic_copilot_code_review_enabled": types.BoolValue(rules.PullRequest.GetAutomaticCopilotCodeReviewEnabled()),
		}

		// Parse AllowedMergeMethods back to boolean fields
		allowMergeCommit := false
		allowSquashMerge := false
		allowRebaseMerge := false

		for _, method := range rules.PullRequest.AllowedMergeMethods {
			switch method {
			case github.PullRequestMergeMethodMerge:
				allowMergeCommit = true
			case github.PullRequestMergeMethodSquash:
				allowSquashMerge = true
			case github.PullRequestMergeMethodRebase:
				allowRebaseMerge = true
			}
		}

		prAttrs["allow_merge_commit"] = types.BoolValue(allowMergeCommit)
		prAttrs["allow_squash_merge"] = types.BoolValue(allowSquashMerge)
		prAttrs["allow_rebase_merge"] = types.BoolValue(allowRebaseMerge)

		prType := types.ObjectType{AttrTypes: r.getPullRequestAttrTypes()}
		prValue, diagsInner := types.ObjectValue(prType.AttrTypes, prAttrs)
		diags.Append(diagsInner...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
		}

		prList, diagsInner := types.ListValue(prType, []attr.Value{prValue})
		diags.Append(diagsInner...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
		}
		rulesAttrs["pull_request"] = prList
	} else {
		rulesAttrs["pull_request"] = types.ListNull(types.ObjectType{AttrTypes: r.getPullRequestAttrTypes()})
	}

	// Required status checks
	if rules.RequiredStatusChecks != nil {
		rscAttrs := map[string]attr.Value{
			"strict_required_status_checks_policy": types.BoolValue(rules.RequiredStatusChecks.StrictRequiredStatusChecksPolicy),
			"do_not_enforce_on_create":             types.BoolValue(rules.RequiredStatusChecks.GetDoNotEnforceOnCreate()),
		}

		// Convert required checks
		if len(rules.RequiredStatusChecks.RequiredStatusChecks) > 0 {
			checkElements := make([]attr.Value, 0, len(rules.RequiredStatusChecks.RequiredStatusChecks))
			checkType := types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"context":        types.StringType,
					"integration_id": types.Int64Type,
				},
			}

			for _, check := range rules.RequiredStatusChecks.RequiredStatusChecks {
				integrationID := int64(0)
				if check.IntegrationID != nil {
					integrationID = *check.IntegrationID
				}

				checkValue, diagsInner := types.ObjectValue(checkType.AttrTypes, map[string]attr.Value{
					"context":        types.StringValue(check.Context),
					"integration_id": types.Int64Value(integrationID),
				})
				diags.Append(diagsInner...)
				if diags.HasError() {
					return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
				}
				checkElements = append(checkElements, checkValue)
			}

			checksSet, diagsInner := types.SetValue(checkType, checkElements)
			diags.Append(diagsInner...)
			if diags.HasError() {
				return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
			}
			rscAttrs["required_check"] = checksSet
		} else {
			rscAttrs["required_check"] = types.SetNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"context":        types.StringType,
					"integration_id": types.Int64Type,
				},
			})
		}

		rscType := types.ObjectType{AttrTypes: r.getRequiredStatusChecksAttrTypes()}
		rscValue, diagsInner := types.ObjectValue(rscType.AttrTypes, rscAttrs)
		diags.Append(diagsInner...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
		}

		rscList, diagsInner := types.ListValue(rscType, []attr.Value{rscValue})
		diags.Append(diagsInner...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
		}
		rulesAttrs["required_status_checks"] = rscList
	} else {
		rulesAttrs["required_status_checks"] = types.ListNull(types.ObjectType{AttrTypes: r.getRequiredStatusChecksAttrTypes()})
	}

	// Merge queue
	if rules.MergeQueue != nil {
		mqAttrs := map[string]attr.Value{
			"check_response_timeout_minutes":    types.Int64Value(int64(rules.MergeQueue.CheckResponseTimeoutMinutes)),
			"grouping_strategy":                 types.StringValue(string(rules.MergeQueue.GroupingStrategy)),
			"max_entries_to_build":              types.Int64Value(int64(rules.MergeQueue.MaxEntriesToBuild)),
			"max_entries_to_merge":              types.Int64Value(int64(rules.MergeQueue.MaxEntriesToMerge)),
			"merge_method":                      types.StringValue(string(rules.MergeQueue.MergeMethod)),
			"min_entries_to_merge":              types.Int64Value(int64(rules.MergeQueue.MinEntriesToMerge)),
			"min_entries_to_merge_wait_minutes": types.Int64Value(int64(rules.MergeQueue.MinEntriesToMergeWaitMinutes)),
		}

		mqType := types.ObjectType{AttrTypes: r.getMergeQueueAttrTypes()}
		mqValue, diagsInner := types.ObjectValue(mqType.AttrTypes, mqAttrs)
		diags.Append(diagsInner...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
		}

		mqList, diagsInner := types.ListValue(mqType, []attr.Value{mqValue})
		diags.Append(diagsInner...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
		}
		rulesAttrs["merge_queue"] = mqList
	} else {
		rulesAttrs["merge_queue"] = types.ListNull(types.ObjectType{AttrTypes: r.getMergeQueueAttrTypes()})
	}

	// Pattern rules
	patternRules := map[string]*github.PatternRuleParameters{
		"commit_message_pattern":      rules.CommitMessagePattern,
		"commit_author_email_pattern": rules.CommitAuthorEmailPattern,
		"committer_email_pattern":     rules.CommitterEmailPattern,
		"branch_name_pattern":         rules.BranchNamePattern,
		"tag_name_pattern":            rules.TagNamePattern,
	}

	patternAttrType := types.ObjectType{AttrTypes: r.getPatternRuleAttrTypes()}

	for ruleType, params := range patternRules {
		if params != nil {
			patternAttrs := map[string]attr.Value{
				"name":     types.StringValue(params.GetName()),
				"negate":   types.BoolValue(params.GetNegate()),
				"operator": types.StringValue(string(params.Operator)),
				"pattern":  types.StringValue(params.Pattern),
			}

			patternValue, diagsInner := types.ObjectValue(patternAttrType.AttrTypes, patternAttrs)
			diags.Append(diagsInner...)
			if diags.HasError() {
				return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
			}

			patternList, diagsInner := types.ListValue(patternAttrType, []attr.Value{patternValue})
			diags.Append(diagsInner...)
			if diags.HasError() {
				return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
			}
			rulesAttrs[ruleType] = patternList
		} else {
			rulesAttrs[ruleType] = types.ListNull(patternAttrType)
		}
	}

	// Required code scanning
	if rules.CodeScanning != nil {
		rcsAttrs := map[string]attr.Value{}

		if len(rules.CodeScanning.CodeScanningTools) > 0 {
			toolElements := make([]attr.Value, 0, len(rules.CodeScanning.CodeScanningTools))
			toolType := types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"alerts_threshold":          types.StringType,
					"security_alerts_threshold": types.StringType,
					"tool":                      types.StringType,
				},
			}

			for _, tool := range rules.CodeScanning.CodeScanningTools {
				toolValue, diagsInner := types.ObjectValue(toolType.AttrTypes, map[string]attr.Value{
					"alerts_threshold":          types.StringValue(string(tool.AlertsThreshold)),
					"security_alerts_threshold": types.StringValue(string(tool.SecurityAlertsThreshold)),
					"tool":                      types.StringValue(tool.Tool),
				})
				diags.Append(diagsInner...)
				if diags.HasError() {
					return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
				}
				toolElements = append(toolElements, toolValue)
			}

			toolsSet, diagsInner := types.SetValue(toolType, toolElements)
			diags.Append(diagsInner...)
			if diags.HasError() {
				return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
			}
			rcsAttrs["required_code_scanning_tool"] = toolsSet
		} else {
			rcsAttrs["required_code_scanning_tool"] = types.SetNull(types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"alerts_threshold":          types.StringType,
					"security_alerts_threshold": types.StringType,
					"tool":                      types.StringType,
				},
			})
		}

		rcsType := types.ObjectType{AttrTypes: r.getRequiredCodeScanningAttrTypes()}
		rcsValue, diagsInner := types.ObjectValue(rcsType.AttrTypes, rcsAttrs)
		diags.Append(diagsInner...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
		}

		rcsList, diagsInner := types.ListValue(rcsType, []attr.Value{rcsValue})
		diags.Append(diagsInner...)
		if diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: rulesAttrTypes}), diags
		}
		rulesAttrs["required_code_scanning"] = rcsList
	} else {
		rulesAttrs["required_code_scanning"] = types.ListNull(types.ObjectType{AttrTypes: r.getRequiredCodeScanningAttrTypes()})
	}

	// Create the rules object
	rulesType := types.ObjectType{AttrTypes: rulesAttrTypes}
	rulesValue, diagsInner := types.ObjectValue(rulesType.AttrTypes, rulesAttrs)
	diags.Append(diagsInner...)
	if diags.HasError() {
		return types.ListNull(rulesType), diags
	}

	// Create rules list
	rulesList, diagsInner := types.ListValue(rulesType, []attr.Value{rulesValue})
	diags.Append(diagsInner...)
	return rulesList, diags
}

// Helper functions to get attribute types for complex nested objects
func (r *githubRepositoryRulesetResource) getRulesAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"creation":                      types.BoolType,
		"update":                        types.BoolType,
		"update_allows_fetch_and_merge": types.BoolType,
		"deletion":                      types.BoolType,
		"required_linear_history":       types.BoolType,
		"required_signatures":           types.BoolType,
		"non_fast_forward":              types.BoolType,
		"required_deployments": types.ListType{ElemType: types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"required_deployment_environments": types.ListType{ElemType: types.StringType},
			},
		}},
		"pull_request":                types.ListType{ElemType: types.ObjectType{AttrTypes: r.getPullRequestAttrTypes()}},
		"required_status_checks":      types.ListType{ElemType: types.ObjectType{AttrTypes: r.getRequiredStatusChecksAttrTypes()}},
		"merge_queue":                 types.ListType{ElemType: types.ObjectType{AttrTypes: r.getMergeQueueAttrTypes()}},
		"commit_message_pattern":      types.ListType{ElemType: types.ObjectType{AttrTypes: r.getPatternRuleAttrTypes()}},
		"commit_author_email_pattern": types.ListType{ElemType: types.ObjectType{AttrTypes: r.getPatternRuleAttrTypes()}},
		"committer_email_pattern":     types.ListType{ElemType: types.ObjectType{AttrTypes: r.getPatternRuleAttrTypes()}},
		"branch_name_pattern":         types.ListType{ElemType: types.ObjectType{AttrTypes: r.getPatternRuleAttrTypes()}},
		"tag_name_pattern":            types.ListType{ElemType: types.ObjectType{AttrTypes: r.getPatternRuleAttrTypes()}},
		"required_code_scanning":      types.ListType{ElemType: types.ObjectType{AttrTypes: r.getRequiredCodeScanningAttrTypes()}},
	}
}

func (r *githubRepositoryRulesetResource) getPullRequestAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"dismiss_stale_reviews_on_push":         types.BoolType,
		"require_code_owner_review":             types.BoolType,
		"require_last_push_approval":            types.BoolType,
		"required_approving_review_count":       types.Int64Type,
		"required_review_thread_resolution":     types.BoolType,
		"allow_merge_commit":                    types.BoolType,
		"allow_squash_merge":                    types.BoolType,
		"allow_rebase_merge":                    types.BoolType,
		"automatic_copilot_code_review_enabled": types.BoolType,
	}
}

func (r *githubRepositoryRulesetResource) getRequiredStatusChecksAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"required_check": types.SetType{ElemType: types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"context":        types.StringType,
				"integration_id": types.Int64Type,
			},
		}},
		"strict_required_status_checks_policy": types.BoolType,
		"do_not_enforce_on_create":             types.BoolType,
	}
}

func (r *githubRepositoryRulesetResource) getMergeQueueAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"check_response_timeout_minutes":    types.Int64Type,
		"grouping_strategy":                 types.StringType,
		"max_entries_to_build":              types.Int64Type,
		"max_entries_to_merge":              types.Int64Type,
		"merge_method":                      types.StringType,
		"min_entries_to_merge":              types.Int64Type,
		"min_entries_to_merge_wait_minutes": types.Int64Type,
	}
}

func (r *githubRepositoryRulesetResource) getPatternRuleAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":     types.StringType,
		"negate":   types.BoolType,
		"operator": types.StringType,
		"pattern":  types.StringType,
	}
}

func (r *githubRepositoryRulesetResource) getRequiredCodeScanningAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"required_code_scanning_tool": types.SetType{ElemType: types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"alerts_threshold":          types.StringType,
				"security_alerts_threshold": types.StringType,
				"tool":                      types.StringType,
			},
		}},
	}
}
