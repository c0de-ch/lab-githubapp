package github

import (
	"context"

	ghlib "github.com/google/go-github/v68/github"
)

type Workflow struct {
	ID        int64
	Name      string
	Path      string
	State     string
	HTMLURL   string
	BadgeURL  string
}

func (c *Client) ListWorkflows(ctx context.Context, owner, repo string) ([]Workflow, error) {
	_, instID, err := c.clientForRepo(owner, repo)
	if err != nil {
		return nil, err
	}
	client := c.auth.ClientForInstallation(instID)

	opts := &ghlib.ListOptions{PerPage: 100}
	result, _, err := client.Actions.ListWorkflows(ctx, owner, repo, opts)
	if err != nil {
		return nil, err
	}

	workflows := make([]Workflow, 0, len(result.Workflows))
	for _, w := range result.Workflows {
		workflows = append(workflows, Workflow{
			ID:       w.GetID(),
			Name:     w.GetName(),
			Path:     w.GetPath(),
			State:    w.GetState(),
			HTMLURL:  w.GetHTMLURL(),
			BadgeURL: w.GetBadgeURL(),
		})
	}
	return workflows, nil
}

func (c *Client) ListWorkflowRuns(ctx context.Context, owner, repo string, workflowID int64) ([]*ghlib.WorkflowRun, error) {
	_, instID, err := c.clientForRepo(owner, repo)
	if err != nil {
		return nil, err
	}
	client := c.auth.ClientForInstallation(instID)

	opts := &ghlib.ListWorkflowRunsOptions{
		ListOptions: ghlib.ListOptions{PerPage: 20},
	}

	var result *ghlib.WorkflowRuns
	if workflowID > 0 {
		result, _, err = client.Actions.ListWorkflowRunsByID(ctx, owner, repo, workflowID, opts)
	} else {
		result, _, err = client.Actions.ListRepositoryWorkflowRuns(ctx, owner, repo, opts)
	}
	if err != nil {
		return nil, err
	}

	return result.WorkflowRuns, nil
}

func (c *Client) DispatchWorkflow(ctx context.Context, owner, repo, workflowFile, ref string, inputs map[string]interface{}) error {
	_, instID, err := c.clientForRepo(owner, repo)
	if err != nil {
		return err
	}
	client := c.auth.ClientForInstallation(instID)

	event := ghlib.CreateWorkflowDispatchEventRequest{
		Ref:    ref,
		Inputs: inputs,
	}

	_, err = client.Actions.CreateWorkflowDispatchEventByFileName(ctx, owner, repo, workflowFile, event)
	return err
}

func (c *Client) RerunWorkflow(ctx context.Context, owner, repo string, runID int64) error {
	_, instID, err := c.clientForRepo(owner, repo)
	if err != nil {
		return err
	}
	client := c.auth.ClientForInstallation(instID)

	_, err = client.Actions.RerunWorkflowByID(ctx, owner, repo, runID)
	return err
}

func (c *Client) RerunFailedJobs(ctx context.Context, owner, repo string, runID int64) error {
	_, instID, err := c.clientForRepo(owner, repo)
	if err != nil {
		return err
	}
	client := c.auth.ClientForInstallation(instID)

	_, err = client.Actions.RerunFailedJobsByID(ctx, owner, repo, runID)
	return err
}

func (c *Client) CancelWorkflowRun(ctx context.Context, owner, repo string, runID int64) error {
	_, instID, err := c.clientForRepo(owner, repo)
	if err != nil {
		return err
	}
	client := c.auth.ClientForInstallation(instID)

	_, err = client.Actions.CancelWorkflowRunByID(ctx, owner, repo, runID)
	return err
}

func (c *Client) GetWorkflowRun(ctx context.Context, owner, repo string, runID int64) (*ghlib.WorkflowRun, error) {
	_, instID, err := c.clientForRepo(owner, repo)
	if err != nil {
		return nil, err
	}
	client := c.auth.ClientForInstallation(instID)

	run, _, err := client.Actions.GetWorkflowRunByID(ctx, owner, repo, runID)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (c *Client) ListWorkflowJobs(ctx context.Context, owner, repo string, runID int64) ([]*ghlib.WorkflowJob, error) {
	_, instID, err := c.clientForRepo(owner, repo)
	if err != nil {
		return nil, err
	}
	client := c.auth.ClientForInstallation(instID)

	opts := &ghlib.ListWorkflowJobsOptions{
		ListOptions: ghlib.ListOptions{PerPage: 100},
	}
	result, _, err := client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, opts)
	if err != nil {
		return nil, err
	}

	return result.Jobs, nil
}
