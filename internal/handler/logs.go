package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	ghclient "github.com/c0de-ch/lab-githubapp/internal/github"
	"github.com/c0de-ch/lab-githubapp/internal/store"
	"github.com/c0de-ch/lab-githubapp/internal/templates"
)

type LogHandler struct {
	store  store.Store
	github *ghclient.Client
	engine *templates.Engine
	logger *slog.Logger
}

func NewLogHandler(s store.Store, gh *ghclient.Client, engine *templates.Engine, logger *slog.Logger) *LogHandler {
	return &LogHandler{store: s, github: gh, engine: engine, logger: logger}
}

func (h *LogHandler) Show(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	runIDStr := r.PathValue("runID")

	runID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid run ID", http.StatusBadRequest)
		return
	}

	run, err := h.store.GetWorkflowRun(runID)
	if err != nil {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	var logEntries []ghclient.LogEntry
	if run.Status == "completed" {
		logEntries, err = h.github.GetRunLogs(r.Context(), owner, repo, runID)
		if err != nil {
			h.logger.Error("failed to get logs", "runID", runID, "error", err)
			// Don't fail the page, just show empty logs with error message
		}
	}

	// Group logs by job name
	jobLogs := make(map[string][]ghclient.LogEntry)
	var jobNames []string
	seen := make(map[string]bool)
	for _, entry := range logEntries {
		if !seen[entry.JobName] {
			seen[entry.JobName] = true
			jobNames = append(jobNames, entry.JobName)
		}
		jobLogs[entry.JobName] = append(jobLogs[entry.JobName], entry)
	}

	data := map[string]any{
		"Title":    "Logs - Run #" + strconv.Itoa(run.RunNumber),
		"Owner":    owner,
		"RepoName": repo,
		"Run":      run,
		"JobNames": jobNames,
		"JobLogs":  jobLogs,
	}

	if isHTMX(r) {
		h.engine.RenderPartial(w, "logs_content", data)
		return
	}

	if err := h.engine.Render(w, "logs", data); err != nil {
		h.logger.Error("failed to render logs", "error", err)
	}
}

func (h *LogHandler) JobLogs(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	jobIDStr := r.PathValue("jobID")

	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid job ID", http.StatusBadRequest)
		return
	}

	logs, err := h.github.GetJobLogs(r.Context(), owner, repo, jobID)
	if err != nil {
		h.logger.Error("failed to get job logs", "jobID", jobID, "error", err)
		http.Error(w, "Failed to get logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(logs))
}
