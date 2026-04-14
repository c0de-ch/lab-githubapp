package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppID          int64
	PrivateKeyPath string
	PrivateKey     []byte
	WebhookSecret  string
	Port           int
	DatabasePath   string
	BaseURL        string
	LogLevel       string
	LogFormat      string
	DevMode        bool
}

func Load() (*Config, error) {
	loadEnvFile(".env")

	cfg := &Config{
		Port:         getEnvInt("PORT", 8080),
		DatabasePath: getEnv("DATABASE_PATH", "./data/ghapp.db"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		LogFormat:    getEnv("LOG_FORMAT", "text"),
		DevMode:      getEnvBool("DEV_MODE", false),
	}

	appIDStr := os.Getenv("GITHUB_APP_ID")
	if appIDStr == "" {
		return nil, fmt.Errorf("GITHUB_APP_ID is required")
	}
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("GITHUB_APP_ID must be a number: %w", err)
	}
	cfg.AppID = appID

	cfg.WebhookSecret = os.Getenv("GITHUB_WEBHOOK_SECRET")
	if cfg.WebhookSecret == "" {
		return nil, fmt.Errorf("GITHUB_WEBHOOK_SECRET is required")
	}

	cfg.PrivateKeyPath = os.Getenv("GITHUB_PRIVATE_KEY_PATH")
	if cfg.PrivateKeyPath == "" {
		return nil, fmt.Errorf("GITHUB_PRIVATE_KEY_PATH is required")
	}

	key, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("reading private key from %s: %w", cfg.PrivateKeyPath, err)
	}
	cfg.PrivateKey = key

	cfg.BaseURL = getEnv("BASE_URL", fmt.Sprintf("http://localhost:%d", cfg.Port))

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1" || v == "yes"
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
