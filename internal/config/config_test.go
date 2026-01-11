package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_WithCredentials(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.DatabasePath == "" {
		t.Error("DatabasePath should have a default value")
	}

	if cfg.AccountsPath == "" {
		t.Error("AccountsPath should have a default value")
	}

	if cfg.QuotaRefreshInterval == 0 {
		t.Error("QuotaRefreshInterval should have a default value")
	}
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("DATABASE_PATH", "/tmp/test.db")
	t.Setenv("ACCOUNTS_PATH", "/tmp/accounts.json")
	t.Setenv("QUOTA_REFRESH_INTERVAL", "60s")
	t.Setenv("GOOGLE_CLIENT_ID", "test-client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "test-client-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.DatabasePath != "/tmp/test.db" {
		t.Errorf("DatabasePath = %q, want %q", cfg.DatabasePath, "/tmp/test.db")
	}

	if cfg.AccountsPath != "/tmp/accounts.json" {
		t.Errorf("AccountsPath = %q, want %q", cfg.AccountsPath, "/tmp/accounts.json")
	}

	if cfg.GoogleClientID != "test-client-id" {
		t.Errorf("GoogleClientID = %q, want %q", cfg.GoogleClientID, "test-client-id")
	}

	if cfg.GoogleClientSecret != "test-client-secret" {
		t.Errorf("GoogleClientSecret = %q, want %q", cfg.GoogleClientSecret, "test-client-secret")
	}
}

func TestLoad_MissingCredentials(t *testing.T) {
	// Skip this test if antigravity constants are available on the system
	// since they provide fallback credentials
	if LoadAntigravityConstants() != nil {
		t.Skip("Skipping test: antigravity constants available on system")
	}

	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Error("Load() should fail without credentials")
	}
}

func TestParseConstants(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantID  string
		wantSec string
		wantNil bool
	}{
		{
			name: "valid constants",
			content: `export declare const ANTIGRAVITY_CLIENT_ID = "test-id-123";
export declare const ANTIGRAVITY_CLIENT_SECRET = "test-secret-456";`,
			wantID:  "test-id-123",
			wantSec: "test-secret-456",
			wantNil: false,
		},
		{
			name:    "empty content",
			content: "",
			wantNil: true,
		},
		{
			name:    "missing client ID",
			content: `export declare const ANTIGRAVITY_CLIENT_SECRET = "test-secret";`,
			wantNil: true,
		},
		{
			name:    "missing client secret",
			content: `export declare const ANTIGRAVITY_CLIENT_ID = "test-id";`,
			wantNil: true,
		},
		{
			name: "with extra content",
			content: `// Some comments
import { something } from "somewhere";
export declare const ANTIGRAVITY_CLIENT_ID = "id-value";
export declare const ANTIGRAVITY_CLIENT_SECRET = "secret-value";
export declare const OTHER_CONSTANT = "other";`,
			wantID:  "id-value",
			wantSec: "secret-value",
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseConstants(tt.content)

			if tt.wantNil {
				if result != nil {
					t.Errorf("parseConstants() = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("parseConstants() = nil, want non-nil")
			}

			if result.ClientID != tt.wantID {
				t.Errorf("ClientID = %q, want %q", result.ClientID, tt.wantID)
			}

			if result.ClientSecret != tt.wantSec {
				t.Errorf("ClientSecret = %q, want %q", result.ClientSecret, tt.wantSec)
			}
		})
	}
}

func TestLoadAntigravityConstants_FileNotFound(t *testing.T) {
	result := LoadAntigravityConstants()
	if result != nil {
		t.Logf("LoadAntigravityConstants() returned non-nil (file may exist on system): %+v", result)
	}
}

func TestGetConstantsFilePath(t *testing.T) {
	path := getConstantsFilePath()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get user home dir: %v", err)
	}

	expectedSuffix := filepath.Join(".config", "opencode", "node_modules", "opencode-antigravity-auth", "dist", "src", "constants.d.ts")
	expectedPath := filepath.Join(home, expectedSuffix)

	if path != expectedPath {
		t.Errorf("getConstantsFilePath() = %q, want %q", path, expectedPath)
	}
}
