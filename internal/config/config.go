// Package config contains everything related to configuration
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the application configuration.
type Config struct {
	DatabasePath         string
	AccountsPath         string
	GoogleClientID       string
	GoogleClientSecret   string
	QuotaRefreshInterval time.Duration
}

// Default values
const (
	defaultQuotaRefreshInterval = 30 * time.Second
)

// Load reads configuration from .env files and environment variables.
func Load() (*Config, error) {
	// Try loading .env from multiple locations
	envPaths := getEnvPaths()
	for _, path := range envPaths {
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
			break
		}
	}

	antigravityConstants := LoadAntigravityConstants()
	var defaultClientID, defaultClientSecret string
	if antigravityConstants != nil {
		defaultClientID = antigravityConstants.ClientID
		defaultClientSecret = antigravityConstants.ClientSecret
	}

	cfg := &Config{
		DatabasePath:         getEnvString("DATABASE_PATH", getDefaultDatabasePath()),
		AccountsPath:         getEnvString("ACCOUNTS_PATH", getDefaultAccountsPath()),
		GoogleClientID:       getEnvString("GOOGLE_CLIENT_ID", defaultClientID),
		GoogleClientSecret:   getEnvString("GOOGLE_CLIENT_SECRET", defaultClientSecret),
		QuotaRefreshInterval: getEnvDuration("QUOTA_REFRESH_INTERVAL", defaultQuotaRefreshInterval),
	}

	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		return nil, fmt.Errorf(
			"GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are required (set via env or opencode-antigravity-auth)")
	}

	// Ensure database directory exists
	if err := ensureDir(filepath.Dir(cfg.DatabasePath)); err != nil {
		return nil, err
	}

	// Ensure accounts directory exists
	if err := ensureDir(filepath.Dir(cfg.AccountsPath)); err != nil {
		return nil, err
	}

	return cfg, nil
}

// getEnvPaths returns a list of paths to check for .env files.
func getEnvPaths() []string {
	var paths []string

	// Current directory
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, ".env"))
	}

	// Home directory locations
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".config", "opencode", "antigravity-tui", ".env"),
			filepath.Join(home, ".config", "opencode", ".env"),
			filepath.Join(home, ".antigravity", ".env"),
		)
	}

	// Parent directories (useful for development)
	if cwd, err := os.Getwd(); err == nil {
		parent := filepath.Dir(cwd)
		paths = append(paths, filepath.Join(parent, ".env"))
		grandparent := filepath.Dir(parent)
		paths = append(paths, filepath.Join(grandparent, ".env"))
	}

	return paths
}

// getDefaultDatabasePath returns the default path for the SQLite database.
func getDefaultDatabasePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "usage.db"
	}
	return filepath.Join(home, ".config", "opencode", "antigravity-tui", "usage.db")
}

// getDefaultAccountsPath returns the default path for the accounts JSON file.
func getDefaultAccountsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "antigravity-accounts.json"
	}
	return filepath.Join(home, ".config", "opencode", "antigravity-accounts.json")
}

// getEnvString retrieves a string environment variable or returns the default.
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvDuration retrieves a duration environment variable or returns the default.
// Accepts values like "30s", "1m", "500ms".
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		// Try parsing as seconds if no unit specified
		if secs, err := strconv.Atoi(value); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return defaultValue
}

// ensureDir creates a directory and all parent directories if they don't exist.
func ensureDir(path string) error {
	if path == "" || path == "." {
		return nil
	}
	return os.MkdirAll(path, 0o750)
}
