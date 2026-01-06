package github

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/go-github/v81/github"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGithubActionsVariable() *schema.Resource {
	return &schema.Resource{
		Description: "Creates and manages an Action variable within a GitHub repository",
		Create:      resourceGithubActionsVariableCreate,
		Read:        resourceGithubActionsVariableRead,
		Update:      resourceGithubActionsVariableUpdate,
		Delete:      resourceGithubActionsVariableDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"repository": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the repository.",
			},
			"variable_name": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				Description:      "Name of the variable.",
				ValidateDiagFunc: validateSecretNameFunc,
			},
			"value": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Value of the variable.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date of 'actions_variable' creation.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date of 'actions_variable' update.",
			},
		},
	}
}

func resourceGithubActionsVariableCreate(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	owner := meta.(*Owner).name
	ctx := context.Background()

	repo := d.Get("repository").(string)
	variableName := d.Get("variable_name").(string)
	variable := &github.ActionsVariable{
		Name:  variableName,
		Value: d.Get("value").(string),
	}

	_, err := client.Actions.CreateRepoVariable(ctx, owner, repo, variable)
	if err != nil {
		return err
	}

	d.SetId(buildTwoPartID(repo, variableName))

	if err := waitForVariableConsistency(ctx, client, owner, repo, variableName); err != nil {
		return err
	}

	return resourceGithubActionsVariableRead(d, meta)
}

func resourceGithubActionsVariableUpdate(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	owner := meta.(*Owner).name
	ctx := context.Background()

	repo := d.Get("repository").(string)
	variable := &github.ActionsVariable{
		Name:  d.Get("variable_name").(string),
		Value: d.Get("value").(string),
	}

	_, err := client.Actions.UpdateRepoVariable(ctx, owner, repo, variable)
	if err != nil {
		return err
	}

	d.SetId(buildTwoPartID(repo, d.Get("variable_name").(string)))
	return resourceGithubActionsVariableRead(d, meta)
}

func resourceGithubActionsVariableRead(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	owner := meta.(*Owner).name
	ctx := context.Background()

	repoName, variableName, err := parseTwoPartID(d.Id(), "repository", "variable_name")
	if err != nil {
		return err
	}

	variable, _, err := client.Actions.GetRepoVariable(ctx, owner, repoName, variableName)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok {
			if ghErr.Response.StatusCode == http.StatusNotFound {
				log.Printf("[INFO] Removing actions variable %s from state because it no longer exists in GitHub",
					d.Id())
				d.SetId("")
				return nil
			}
		}
		return err
	}

	if err = d.Set("repository", repoName); err != nil {
		return err
	}
	if err = d.Set("variable_name", variableName); err != nil {
		return err
	}
	if err = d.Set("value", variable.Value); err != nil {
		return err
	}
	if err = d.Set("created_at", variable.CreatedAt.String()); err != nil {
		return err
	}
	if err = d.Set("updated_at", variable.UpdatedAt.String()); err != nil {
		return err
	}

	return nil
}

func resourceGithubActionsVariableDelete(d *schema.ResourceData, meta any) error {
	client := meta.(*Owner).v3client
	orgName := meta.(*Owner).name
	ctx := context.WithValue(context.Background(), ctxId, d.Id())

	repoName, variableName, err := parseTwoPartID(d.Id(), "repository", "variable_name")
	if err != nil {
		return err
	}

	_, err = client.Actions.DeleteRepoVariable(ctx, orgName, repoName, variableName)

	return err
}

func waitForVariableConsistency(
	ctx context.Context,
	client *github.Client,
	owner, repo, variableName string,
) error {
	// Derived context with overall timeout for the wait.
	ctxWithTimeout, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	// Initial delay to match previous behaviour.
	select {
	case <-ctxWithTimeout.Done():
		return fmt.Errorf("context done before waiting for variable %s: %w", variableName, ctxWithTimeout.Err())
	case <-time.After(time.Second):
	}

	backoff := time.Second
	maxBackoff := 10 * time.Second

	for {
		_, _, err := client.Actions.GetRepoVariable(ctxWithTimeout, owner, repo, variableName)
		if err == nil {
			return nil
		}

		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response.StatusCode == http.StatusNotFound {
			// Variable not visible yet, wait with exponential backoff.
			select {
			case <-ctxWithTimeout.Done():
				return fmt.Errorf("timeout waiting for variable %s to be consistent: %w", variableName, ctxWithTimeout.Err())
			case <-time.After(backoff):
			}

			// Exponential backoff with cap.
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Non-404 error: likely not transient.
		return fmt.Errorf("error waiting for variable %s to be consistent: %w", variableName, err)
	}
}
