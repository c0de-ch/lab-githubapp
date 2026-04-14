package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/c0de-ch/lab-githubapp/internal/config"
	ghclient "github.com/c0de-ch/lab-githubapp/internal/github"
	"github.com/c0de-ch/lab-githubapp/internal/server"
	"github.com/c0de-ch/lab-githubapp/internal/sse"
	"github.com/c0de-ch/lab-githubapp/internal/store"
	"github.com/c0de-ch/lab-githubapp/internal/templates"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Setup logger
	var logHandler slog.Handler
	opts := &slog.HandlerOptions{}
	switch cfg.LogLevel {
	case "debug":
		opts.Level = slog.LevelDebug
	case "warn":
		opts.Level = slog.LevelWarn
	case "error":
		opts.Level = slog.LevelError
	default:
		opts.Level = slog.LevelInfo
	}

	if cfg.LogFormat == "json" {
		logHandler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, opts)
	}
	logger := slog.New(logHandler)

	// Initialize database
	db, err := store.NewSQLiteStore(cfg.DatabasePath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}
	logger.Info("database initialized", "path", cfg.DatabasePath)

	// Initialize GitHub auth
	auth, err := ghclient.NewAuthProvider(cfg.AppID, cfg.PrivateKey)
	if err != nil {
		return fmt.Errorf("creating github auth provider: %w", err)
	}

	ghClient := ghclient.NewClient(auth, db)

	// Initialize SSE broker
	broker := sse.NewBroker()

	// Setup context with cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Start SSE broker
	go broker.Run(ctx)

	// Sync installations on startup
	go func() {
		logger.Info("syncing installations from GitHub...")
		if err := ghClient.SyncInstallations(ctx); err != nil {
			logger.Error("failed to sync installations", "error", err)
		} else {
			logger.Info("installations synced successfully")
		}
	}()

	// Load templates
	tmplFS, err := getTemplateFS(cfg.DevMode)
	if err != nil {
		return fmt.Errorf("loading templates: %w", err)
	}

	engine, err := templates.NewEngine(tmplFS, cfg.DevMode)
	if err != nil {
		return fmt.Errorf("creating template engine: %w", err)
	}

	// Load static files
	staticFileSystem, err := getStaticFS(cfg.DevMode)
	if err != nil {
		return fmt.Errorf("loading static files: %w", err)
	}

	// Create and start server
	srv := server.New(server.Config{
		Port:           cfg.Port,
		WebhookSecret:  cfg.WebhookSecret,
		Store:          db,
		GitHubClient:   ghClient,
		AuthProvider:   auth,
		Broker:         broker,
		TemplateEngine: engine,
		StaticFS:       staticFileSystem,
		Logger:         logger,
	})

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	logger.Info("server started",
		"port", cfg.Port,
		"base_url", cfg.BaseURL,
		"dev_mode", cfg.DevMode,
	)

	// Wait for shutdown signal or error
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	return srv.Shutdown(shutdownCtx)
}

func getTemplateFS(devMode bool) (fs.FS, error) {
	if devMode {
		return os.DirFS("web"), nil
	}
	return fs.Sub(embeddedTemplateFS, "web")
}

func getStaticFS(devMode bool) (http.FileSystem, error) {
	if devMode {
		return http.Dir("web/static"), nil
	}
	sub, err := fs.Sub(embeddedStaticFS, "web/static")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}
