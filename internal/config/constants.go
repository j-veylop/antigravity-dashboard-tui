// Package config contains everything related to configuration
package config

import (
	"os"
	"path/filepath"
	"regexp"
)

type AntigravityConstants struct {
	ClientID     string
	ClientSecret string
}

func getConstantsFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "opencode", "node_modules",
		"opencode-antigravity-auth", "dist", "src", "constants.d.ts")
}

func LoadAntigravityConstants() *AntigravityConstants {
	path := getConstantsFilePath()
	if path == "" {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	return parseConstants(string(content))
}

func parseConstants(content string) *AntigravityConstants {
	constants := &AntigravityConstants{}

	// Match: export declare const ANTIGRAVITY_CLIENT_ID = "...";
	clientIDRe := regexp.MustCompile(`ANTIGRAVITY_CLIENT_ID\s*=\s*"([^"]+)"`)
	if match := clientIDRe.FindStringSubmatch(content); len(match) > 1 {
		constants.ClientID = match[1]
	}

	// Match: export declare const ANTIGRAVITY_CLIENT_SECRET = "...";
	clientSecretRe := regexp.MustCompile(`ANTIGRAVITY_CLIENT_SECRET\s*=\s*"([^"]+)"`)
	if match := clientSecretRe.FindStringSubmatch(content); len(match) > 1 {
		constants.ClientSecret = match[1]
	}

	if constants.ClientID == "" || constants.ClientSecret == "" {
		return nil
	}

	return constants
}
