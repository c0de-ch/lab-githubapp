package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	ghclient "github.com/c0de-ch/lab-githubapp/internal/github"
	"github.com/c0de-ch/lab-githubapp/internal/store"
	"github.com/c0de-ch/lab-githubapp/internal/templates"
)

type RunHandler struct {
	store  store.Store
	github *ghclient.Client
	engine *templates.Engine
	logger *slog.Logger
}

func NewRunHandler(s store.Store, gh *ghclient.Client, engine *templates.Engine, logger *slog.Logger) *RunHandler {
	return &RunHandler{store: s, github: gh, engine: engine, logger: logger}
}

func (h *RunHandler) List(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	status := r.URL.Query().Get("status")

	runs, err := h.store.ListWorkflowRuns(store.ListRunsOpts{
		Owner:  owner,
		Repo:   repo,
		Status: status,
		Limit:  50,
	})
	if err != nil {
		h.logger.Error("failed to list runs", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":    "Runs - " + owner + "/" + repo,
		"Owner":    owner,
		"RepoName": repo,
		"Runs":     runs,
		"Status":   status,
	}

	if isHTMX(r) {
		h.engine.RenderPartial(w, "runs_table", data)
		return
	}

	if err := h.engine.Render(w, "runs", data); err != nil {
		h.logger.Error("failed to render runs", "error", err)
	}
}

func (h *RunHandler) Show(w http.ResponseWriter, r *http.Request) {
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
		h.logger.Error("run not found", "runID", runID, "error", err)
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	jobs, err := h.store.ListWorkflowJobs(runID)
	if err != nil {
		h.logger.Error("failed to list jobs", "runID", runID, "error", err)
		jobs = nil
	}

	data := map[string]any{
		"Title":    "Run #" + strconv.Itoa(run.RunNumber) + " - " + owner + "/" + repo,
		"Owner":    owner,
		"RepoName": repo,
		"Run":      run,
		"Jobs":     jobs,
	}

	if isHTMX(r) {
		h.engine.RenderPartial(w, "run_detail_content", data)
		return
	}

	if err := h.engine.Render(w, "run_detail", data); err != nil {
		h.logger.Error("failed to render run detail", "error", err)
	}
}

func (h *RunHandler) Rerun(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	runID, _ := strconv.ParseInt(r.PathValue("runID"), 10, 64)

	if err := h.github.RerunWorkflow(r.Context(), owner, repo, runID); err != nil {
		h.logger.Error("failed to rerun workflow", "runID", runID, "error", err)
		renderToast(w, h.engine, "error", "Failed to re-run: "+err.Error())
		return
	}

	h.logger.Info("workflow rerun triggered", "owner", owner, "repo", repo, "runID", runID)
	renderToast(w, h.engine, "success", "Re-run triggered successfully")
}

func (h *RunHandler) RerunFailed(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	runID, _ := strconv.ParseInt(r.PathValue("runID"), 10, 64)

	if err := h.github.RerunFailedJobs(r.Context(), owner, repo, runID); err != nil {
		h.logger.Error("failed to rerun failed jobs", "runID", runID, "error", err)
		renderToast(w, h.engine, "error", "Failed to re-run failed jobs: "+err.Error())
		return
	}

	h.logger.Info("rerun failed jobs triggered", "owner", owner, "repo", repo, "runID", runID)
	renderToast(w, h.engine, "success", "Re-run of failed jobs triggered")
}

func (h *RunHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	runID, _ := strconv.ParseInt(r.PathValue("runID"), 10, 64)

	if err := h.github.CancelWorkflowRun(r.Context(), owner, repo, runID); err != nil {
		h.logger.Error("failed to cancel workflow", "runID", runID, "error", err)
		renderToast(w, h.engine, "error", "Failed to cancel: "+err.Error())
		return
	}

	h.logger.Info("workflow cancelled", "owner", owner, "repo", repo, "runID", runID)
	renderToast(w, h.engine, "success", "Workflow run cancelled")
}

func renderToast(w http.ResponseWriter, engine *templates.Engine, toastType, message string) {
	w.Header().Set("HX-Retarget", "#toast-container")
	w.Header().Set("HX-Reswap", "innerHTML")
	engine.RenderPartial(w, "toast", map[string]any{
		"Type":    toastType,
		"Message": message,
	})
}
