package handler

import (
	"log/slog"
	"net/http"

	ghclient "github.com/c0de-ch/lab-githubapp/internal/github"
	"github.com/c0de-ch/lab-githubapp/internal/store"
	"github.com/c0de-ch/lab-githubapp/internal/templates"
)

type RepoHandler struct {
	store  store.Store
	github *ghclient.Client
	engine *templates.Engine
	logger *slog.Logger
}

func NewRepoHandler(s store.Store, gh *ghclient.Client, engine *templates.Engine, logger *slog.Logger) *RepoHandler {
	return &RepoHandler{store: s, github: gh, engine: engine, logger: logger}
}

func (h *RepoHandler) Show(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")

	repoInfo, err := h.store.GetRepository(owner, repo)
	if err != nil {
		h.logger.Error("repo not found", "owner", owner, "repo", repo, "error", err)
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	workflows, err := h.github.ListWorkflows(r.Context(), owner, repo)
	if err != nil {
		h.logger.Error("failed to list workflows", "owner", owner, "repo", repo, "error", err)
		workflows = nil // Continue with empty workflows
	}

	runs, err := h.store.ListRecentRuns(owner, repo, 20)
	if err != nil {
		h.logger.Error("failed to list runs", "owner", owner, "repo", repo, "error", err)
		runs = nil
	}

	data := map[string]any{
		"Title":     owner + "/" + repo,
		"Repo":      repoInfo,
		"Owner":     owner,
		"RepoName":  repo,
		"Workflows": workflows,
		"Runs":      runs,
	}

	if isHTMX(r) {
		h.engine.RenderPartial(w, "repo_content", data)
		return
	}

	if err := h.engine.Render(w, "repo", data); err != nil {
		h.logger.Error("failed to render repo", "error", err)
	}
}
