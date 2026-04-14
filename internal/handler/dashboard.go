package handler

import (
	"log/slog"
	"net/http"

	"github.com/c0de-ch/lab-githubapp/internal/store"
	"github.com/c0de-ch/lab-githubapp/internal/templates"
)

type DashboardHandler struct {
	store  store.Store
	engine *templates.Engine
	logger *slog.Logger
}

func NewDashboardHandler(s store.Store, engine *templates.Engine, logger *slog.Logger) *DashboardHandler {
	return &DashboardHandler{store: s, engine: engine, logger: logger}
}

func (h *DashboardHandler) Index(w http.ResponseWriter, r *http.Request) {
	repos, err := h.store.ListAllRepositories()
	if err != nil {
		h.logger.Error("failed to list repositories", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	installations, err := h.store.ListInstallations()
	if err != nil {
		h.logger.Error("failed to list installations", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := map[string]any{
		"Title":         "Dashboard",
		"Repos":         repos,
		"Installations": installations,
	}

	if isHTMX(r) {
		h.engine.RenderPartial(w, "dashboard_content", data)
		return
	}

	if err := h.engine.Render(w, "dashboard", data); err != nil {
		h.logger.Error("failed to render dashboard", "error", err)
	}
}

func isHTMX(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}
