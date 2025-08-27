package github

import (
	"context"
	"errors"

	"github.com/shurcooL/githubv4"
)

func getRepositoryID(name string, meta any) (githubv4.ID, error) {

	// Interpret `name` as a node ID
	exists, nodeIDerr := repositoryNodeIDExists(name, meta)
	if exists {
		return githubv4.ID(name), nil
	}

	// Could not find repo by node ID, interpret `name` as repo name
	var query struct {
		Repository struct {
			ID githubv4.ID
		} `graphql:"repository(owner:$owner, name:$name)"`
	}
	variables := map[string]any{
		"owner": githubv4.String(meta.(*Owner).name),
		"name":  githubv4.String(name),
	}
	ctx := context.Background()
	client := meta.(*Owner).v4client
	nameErr := client.Query(ctx, &query, variables)
	if nameErr != nil {
		if nodeIDerr != nil {
			// Could not find repo by node ID or repo name, return both errors
			return nil, errors.New(nodeIDerr.Error() + nameErr.Error())
		}
		return nil, nameErr
	}

	return query.Repository.ID, nil
}

func repositoryNodeIDExists(name string, meta any) (bool, error) {

	// API check if node ID exists
	var query struct {
		Node struct {
			ID githubv4.ID
			Typename string `graphql:"__typename"`
		} `graphql:"node(id:$id)"`
	}
	variables := map[string]any{
		"id": githubv4.ID(name),
	}
	ctx := context.Background()
	client := meta.(*Owner).v4client
	err := client.Query(ctx, &query, variables)
	if err != nil {
		return false, err
	}

	// Only return true if it's actually a Repository node
	if query.Node.Typename != "Repository" {
		return false, nil
	}

	return query.Node.ID.(string) == name, nil
}
