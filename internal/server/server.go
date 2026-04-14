package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	ghclient "github.com/c0de-ch/lab-githubapp/internal/github"
	"github.com/c0de-ch/lab-githubapp/internal/handler"
	"github.com/c0de-ch/lab-githubapp/internal/sse"
	"github.com/c0de-ch/lab-githubapp/internal/store"
	"github.com/c0de-ch/lab-githubapp/internal/templates"
	"github.com/c0de-ch/lab-githubapp/internal/webhook"
)

type Server struct {
	httpServer *http.Server
	handler    http.Handler
	logger     *slog.Logger
}

type Config struct {
	Port          int
	WebhookSecret string
	Store         store.Store
	GitHubClient  *ghclient.Client
	AuthProvider  *ghclient.AuthProvider
	Broker        *sse.Broker
	TemplateEngine *templates.Engine
	StaticFS      http.FileSystem
	Logger        *slog.Logger
}

func New(cfg Config) *Server {
	s := &Server{
		logger: cfg.Logger,
	}

	// Create handlers
	webhookProcessor := webhook.NewEventProcessor(cfg.Store, cfg.Broker, cfg.Logger)
	webhookHandler := webhook.NewHandler(cfg.WebhookSecret, webhookProcessor, cfg.Logger)

	dashboardHandler := handler.NewDashboardHandler(cfg.Store, cfg.TemplateEngine, cfg.Logger)
	repoHandler := handler.NewRepoHandler(cfg.Store, cfg.GitHubClient, cfg.TemplateEngine, cfg.Logger)
	workflowHandler := handler.NewWorkflowHandler(cfg.Store, cfg.GitHubClient, cfg.TemplateEngine, cfg.Logger)
	runHandler := handler.NewRunHandler(cfg.Store, cfg.GitHubClient, cfg.TemplateEngine, cfg.Logger)
	logHandler := handler.NewLogHandler(cfg.Store, cfg.GitHubClient, cfg.TemplateEngine, cfg.Logger)
	sseHandler := handler.NewSSEHandler(cfg.Broker, cfg.Logger)

	s.registerRoutes(
		webhookHandler,
		dashboardHandler,
		repoHandler,
		workflowHandler,
		runHandler,
		logHandler,
		sseHandler,
		cfg.StaticFS,
	)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // SSE needs unlimited write timeout
		IdleTimeout:  120 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	s.logger.Info("server starting", "addr", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("server shutting down")
	return s.httpServer.Shutdown(ctx)
}
