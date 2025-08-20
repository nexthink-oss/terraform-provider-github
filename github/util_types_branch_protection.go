package github

import (
	"github.com/shurcooL/githubv4"
)

// GraphQL types for branch protection operations

type Actor struct {
	ID   githubv4.ID
	Name githubv4.String
	Slug githubv4.String
}

type ActorUser struct {
	ID    githubv4.ID
	Name  githubv4.String
	Login githubv4.String
}

type DismissalActorTypes struct {
	Actor struct {
		App  Actor     `graphql:"... on App"`
		Team Actor     `graphql:"... on Team"`
		User ActorUser `graphql:"... on User"`
	}
}

type BypassForcePushActorTypes struct {
	Actor struct {
		App  Actor     `graphql:"... on App"`
		Team Actor     `graphql:"... on Team"`
		User ActorUser `graphql:"... on User"`
	}
}

type BypassPullRequestActorTypes struct {
	Actor struct {
		App  Actor     `graphql:"... on App"`
		Team Actor     `graphql:"... on Team"`
		User ActorUser `graphql:"... on User"`
	}
}

type PushActorTypes struct {
	Actor struct {
		App  Actor     `graphql:"... on App"`
		Team Actor     `graphql:"... on Team"`
		User ActorUser `graphql:"... on User"`
	}
}

type BranchProtectionRule struct {
	Repository struct {
		ID   githubv4.String
		Name githubv4.String
	}
	PushAllowances struct {
		Nodes []PushActorTypes
	} `graphql:"pushAllowances(first: 100)"`
	ReviewDismissalAllowances struct {
		Nodes []DismissalActorTypes
	} `graphql:"reviewDismissalAllowances(first: 100)"`
	BypassForcePushAllowances struct {
		Nodes []BypassForcePushActorTypes
	} `graphql:"bypassForcePushAllowances(first: 100)"`
	BypassPullRequestAllowances struct {
		Nodes []BypassPullRequestActorTypes
	} `graphql:"bypassPullRequestAllowances(first: 100)"`
	AllowsDeletions                githubv4.Boolean
	AllowsForcePushes              githubv4.Boolean
	BlocksCreations                githubv4.Boolean
	DismissesStaleReviews          githubv4.Boolean
	ID                             githubv4.ID
	IsAdminEnforced                githubv4.Boolean
	Pattern                        githubv4.String
	RequiredApprovingReviewCount   githubv4.Int
	RequiredStatusCheckContexts    []githubv4.String
	RequiresApprovingReviews       githubv4.Boolean
	RequiresCodeOwnerReviews       githubv4.Boolean
	RequiresCommitSignatures       githubv4.Boolean
	RequiresLinearHistory          githubv4.Boolean
	RequiresConversationResolution githubv4.Boolean
	RequiresStatusChecks           githubv4.Boolean
	RequiresStrictStatusChecks     githubv4.Boolean
	RestrictsPushes                githubv4.Boolean
	RestrictsReviewDismissals      githubv4.Boolean
	RequireLastPushApproval        githubv4.Boolean
	LockBranch                     githubv4.Boolean
}

type BranchProtectionResourceData struct {
	AllowsDeletions                bool
	AllowsForcePushes              bool
	BlocksCreations                bool
	BranchProtectionRuleID         string
	BypassForcePushActorIDs        []string
	BypassPullRequestActorIDs      []string
	DismissesStaleReviews          bool
	IsAdminEnforced                bool
	Pattern                        string
	PushActorIDs                   []string
	RepositoryID                   string
	RequiredApprovingReviewCount   int
	RequiredStatusCheckContexts    []string
	RequiresApprovingReviews       bool
	RequiresCodeOwnerReviews       bool
	RequiresCommitSignatures       bool
	RequiresLinearHistory          bool
	RequiresConversationResolution bool
	RequiresStatusChecks           bool
	RequiresStrictStatusChecks     bool
	RestrictsPushes                bool
	RestrictsReviewDismissals      bool
	ReviewDismissalActorIDs        []string
	RequireLastPushApproval        bool
	LockBranch                     bool
}
