package server

import (
	"net/http"

	"github.com/c0de-ch/lab-githubapp/internal/handler"
	"github.com/c0de-ch/lab-githubapp/internal/webhook"
)

func (s *Server) registerRoutes(
	webhookHandler *webhook.Handler,
	dashboardHandler *handler.DashboardHandler,
	repoHandler *handler.RepoHandler,
	workflowHandler *handler.WorkflowHandler,
	runHandler *handler.RunHandler,
	logHandler *handler.LogHandler,
	sseHandler *handler.SSEHandler,
	staticFS http.FileSystem,
) {
	mux := http.NewServeMux()

	// Static assets
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(staticFS)))

	// Webhook endpoint
	mux.Handle("POST /api/webhooks/github", webhookHandler)

	// Dashboard
	mux.HandleFunc("GET /{$}", dashboardHandler.Index)

	// Repository detail
	mux.HandleFunc("GET /repos/{owner}/{repo}", repoHandler.Show)

	// Workflows
	mux.HandleFunc("GET /repos/{owner}/{repo}/workflows", workflowHandler.List)
	mux.HandleFunc("POST /repos/{owner}/{repo}/workflows/{workflowID}/dispatch", workflowHandler.Dispatch)

	// Workflow Runs
	mux.HandleFunc("GET /repos/{owner}/{repo}/runs", runHandler.List)
	mux.HandleFunc("GET /repos/{owner}/{repo}/runs/{runID}", runHandler.Show)
	mux.HandleFunc("POST /repos/{owner}/{repo}/runs/{runID}/rerun", runHandler.Rerun)
	mux.HandleFunc("POST /repos/{owner}/{repo}/runs/{runID}/rerun-failed", runHandler.RerunFailed)
	mux.HandleFunc("POST /repos/{owner}/{repo}/runs/{runID}/cancel", runHandler.Cancel)

	// Logs
	mux.HandleFunc("GET /repos/{owner}/{repo}/runs/{runID}/logs", logHandler.Show)
	mux.HandleFunc("GET /repos/{owner}/{repo}/jobs/{jobID}/logs", logHandler.JobLogs)

	// SSE streams
	mux.HandleFunc("GET /api/sse/repos/{owner}/{repo}", sseHandler.RepoStream)
	mux.HandleFunc("GET /api/sse/repos/{owner}/{repo}/runs/{runID}", sseHandler.RunStream)

	// Apply middleware
	s.handler = recoveryMiddleware(s.logger, loggingMiddleware(s.logger, mux))
}
