// Package version provides build version information and runtime metadata.
package version

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	// Version is the build version, set via ldflags.
	Version = ""
	// Commit is the git commit hash, set via ldflags.
	Commit = ""
	// Date is the build date, set via ldflags.
	Date = ""

	once sync.Once

	// execCommand allows mocking exec.CommandContext for testing
	execCommand = exec.CommandContext
)

// Reset resets the package state for testing.
func Reset() {
	once = sync.Once{}
	Version = ""
	Commit = ""
	Date = ""
}

func ensureInitialized() {
	once.Do(func() {
		if Date == "" {
			Date = time.Now().Format("2006-01-02")
		}
		if Commit == "" {
			Commit = getGitCommit()
		}
		if Version == "" {
			Version = getGitVersion()
		}
	})
}

func getGitCommit() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := execCommand(ctx, "git", "describe", "--always", "--dirty")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "unknown"
	}
	return strings.TrimSpace(out.String())
}

func getGitVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := execCommand(ctx, "git", "describe", "--tags")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		v := strings.TrimSpace(out.String())
		if v != "" {
			return v
		}
	}
	return "dev"
}

// Info returns the full version string.
func Info() string {
	ensureInitialized()
	return fmt.Sprintf("antigravity-dashboard-tui %s (commit: %s, built: %s, %s/%s)",
		Version, Commit, Date, runtime.GOOS, runtime.GOARCH)
}

func GetVersion() string {
	ensureInitialized()
	return Version
}

func GetCommit() string {
	ensureInitialized()
	return Commit
}

func GetDate() string {
	ensureInitialized()
	return Date
}
