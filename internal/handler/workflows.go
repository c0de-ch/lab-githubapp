package handler

import (
	"log/slog"
	"net/http"

	ghclient "github.com/c0de-ch/lab-githubapp/internal/github"
	"github.com/c0de-ch/lab-githubapp/internal/store"
	"github.com/c0de-ch/lab-githubapp/internal/templates"
)

type WorkflowHandler struct {
	store  store.Store
	github *ghclient.Client
	engine *templates.Engine
	logger *slog.Logger
}

func NewWorkflowHandler(s store.Store, gh *ghclient.Client, engine *templates.Engine, logger *slog.Logger) *WorkflowHandler {
	return &WorkflowHandler{store: s, github: gh, engine: engine, logger: logger}
}

func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")

	workflows, err := h.github.ListWorkflows(r.Context(), owner, repo)
	if err != nil {
		h.logger.Error("failed to list workflows", "error", err)
		http.Error(w, "Failed to list workflows", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":     "Workflows - " + owner + "/" + repo,
		"Owner":     owner,
		"RepoName":  repo,
		"Workflows": workflows,
	}

	if isHTMX(r) {
		h.engine.RenderPartial(w, "workflow_list", data)
		return
	}

	if err := h.engine.Render(w, "workflows", data); err != nil {
		h.logger.Error("failed to render workflows", "error", err)
	}
}

func (h *WorkflowHandler) Dispatch(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	workflowIDStr := r.PathValue("workflowID")

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	ref := r.FormValue("ref")
	if ref == "" {
		ref = "main"
	}

	_ = workflowIDStr // Used for logging

	workflowFile := r.FormValue("workflow_file")
	if workflowFile == "" {
		http.Error(w, "Workflow file is required", http.StatusBadRequest)
		return
	}

	// Collect workflow inputs
	inputs := make(map[string]interface{})
	for key, values := range r.Form {
		if len(key) > 6 && key[:6] == "input_" {
			inputName := key[6:]
			if len(values) > 0 && values[0] != "" {
				inputs[inputName] = values[0]
			}
		}
	}

	if err := h.github.DispatchWorkflow(r.Context(), owner, repo, workflowFile, ref, inputs); err != nil {
		h.logger.Error("failed to dispatch workflow",
			"owner", owner, "repo", repo, "workflow", workflowFile,
			"error", err,
		)
		w.Header().Set("HX-Retarget", "#toast-container")
		w.Header().Set("HX-Reswap", "innerHTML")
		h.engine.RenderPartial(w, "toast", map[string]any{
			"Type":    "error",
			"Message": "Failed to dispatch workflow: " + err.Error(),
		})
		return
	}

	h.logger.Info("workflow dispatched",
		"owner", owner, "repo", repo, "workflow", workflowFile,
		"ref", ref, "workflow_id", workflowIDStr,
	)

	w.Header().Set("HX-Retarget", "#toast-container")
	w.Header().Set("HX-Reswap", "innerHTML")
	h.engine.RenderPartial(w, "toast", map[string]any{
		"Type":    "success",
		"Message": "Workflow dispatched successfully on " + ref,
	})
}
