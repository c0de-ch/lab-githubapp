package webhook

import (
	"encoding/json"
	"log/slog"

	"github.com/c0de-ch/lab-githubapp/internal/sse"
	"github.com/c0de-ch/lab-githubapp/internal/store"
	ghlib "github.com/google/go-github/v68/github"
)

type EventProcessor struct {
	store  store.Store
	broker *sse.Broker
	logger *slog.Logger
}

func NewEventProcessor(s store.Store, broker *sse.Broker, logger *slog.Logger) *EventProcessor {
	return &EventProcessor{
		store:  s,
		broker: broker,
		logger: logger,
	}
}

func (ep *EventProcessor) Process(eventType string, payload []byte) error {
	switch eventType {
	case "installation":
		return ep.handleInstallation(payload)
	case "installation_repositories":
		return ep.handleInstallationRepositories(payload)
	case "workflow_run":
		return ep.handleWorkflowRun(payload)
	case "workflow_job":
		return ep.handleWorkflowJob(payload)
	default:
		ep.logger.Debug("unhandled event type", "type", eventType)
		return nil
	}
}

func (ep *EventProcessor) handleInstallation(payload []byte) error {
	var event ghlib.InstallationEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	action := event.GetAction()
	inst := event.GetInstallation()

	switch action {
	case "created":
		installation := &store.Installation{
			ID:           inst.GetID(),
			AppID:        inst.GetAppID(),
			AccountLogin: inst.GetAccount().GetLogin(),
			AccountType:  inst.GetAccount().GetType(),
			TargetType:   inst.GetTargetType(),
		}
		if err := ep.store.UpsertInstallation(installation); err != nil {
			return err
		}
		for _, r := range event.Repositories {
			repo := &store.Repository{
				ID:             r.GetID(),
				InstallationID: inst.GetID(),
				FullName:       r.GetFullName(),
				Private:        r.GetPrivate(),
			}
			parts := splitFullName(r.GetFullName())
			repo.Owner = parts[0]
			repo.Name = parts[1]
			if err := ep.store.UpsertRepository(repo); err != nil {
				ep.logger.Error("failed to upsert repo", "repo", r.GetFullName(), "error", err)
			}
		}
	case "deleted":
		if err := ep.store.DeleteInstallation(inst.GetID()); err != nil {
			return err
		}
	}

	return nil
}

func (ep *EventProcessor) handleInstallationRepositories(payload []byte) error {
	var event ghlib.InstallationRepositoriesEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	instID := event.GetInstallation().GetID()

	for _, r := range event.RepositoriesAdded {
		repo := &store.Repository{
			ID:             r.GetID(),
			InstallationID: instID,
			FullName:       r.GetFullName(),
			Private:        r.GetPrivate(),
		}
		parts := splitFullName(r.GetFullName())
		repo.Owner = parts[0]
		repo.Name = parts[1]
		if err := ep.store.UpsertRepository(repo); err != nil {
			ep.logger.Error("failed to upsert repo", "repo", r.GetFullName(), "error", err)
		}
	}

	for _, r := range event.RepositoriesRemoved {
		if err := ep.store.DeleteRepository(r.GetID()); err != nil {
			ep.logger.Error("failed to delete repo", "repo", r.GetFullName(), "error", err)
		}
	}

	return nil
}

func (ep *EventProcessor) handleWorkflowRun(payload []byte) error {
	var event ghlib.WorkflowRunEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	run := event.GetWorkflowRun()
	repo := event.GetRepo()

	workflowRun := &store.WorkflowRun{
		ID:           run.GetID(),
		RepositoryID: repo.GetID(),
		WorkflowID:   run.GetWorkflowID(),
		WorkflowName: run.GetName(),
		RunNumber:    run.GetRunNumber(),
		HeadBranch:   run.GetHeadBranch(),
		HeadSHA:      run.GetHeadSHA(),
		Event:        run.GetEvent(),
		Status:       run.GetStatus(),
		Conclusion:   run.GetConclusion(),
		HTMLURL:      run.GetHTMLURL(),
		CreatedAt:    run.GetCreatedAt().Time,
		UpdatedAt:    run.GetUpdatedAt().Time,
	}

	if err := ep.store.UpsertWorkflowRun(workflowRun); err != nil {
		return err
	}

	// Broadcast SSE event
	topic := "repo:" + repo.GetFullName()
	ep.broker.Publish(topic, sse.Event{
		Name: "run-update",
		Data: workflowRun.ToJSON(),
	})

	return nil
}

func (ep *EventProcessor) handleWorkflowJob(payload []byte) error {
	var event ghlib.WorkflowJobEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return err
	}

	job := event.GetWorkflowJob()
	repo := event.GetRepo()

	stepsJSON, _ := json.Marshal(job.Steps)

	workflowJob := &store.WorkflowJob{
		ID:         job.GetID(),
		RunID:      job.GetRunID(),
		Name:       job.GetName(),
		Status:     job.GetStatus(),
		Conclusion: job.GetConclusion(),
		RunnerName: job.GetRunnerName(),
		Steps:      string(stepsJSON),
	}

	if job.StartedAt != nil {
		workflowJob.StartedAt = job.GetStartedAt().Time
	}
	if job.CompletedAt != nil {
		workflowJob.CompletedAt = job.GetCompletedAt().Time
	}

	if err := ep.store.UpsertWorkflowJob(workflowJob); err != nil {
		return err
	}

	// Broadcast SSE event for the run
	topic := "repo:" + repo.GetFullName()
	ep.broker.Publish(topic, sse.Event{
		Name: "job-update",
		Data: workflowJob.ToJSON(),
	})

	return nil
}

func splitFullName(fullName string) [2]string {
	for i, c := range fullName {
		if c == '/' {
			return [2]string{fullName[:i], fullName[i+1:]}
		}
	}
	return [2]string{fullName, ""}
}
