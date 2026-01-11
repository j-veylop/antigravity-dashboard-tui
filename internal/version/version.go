// Package version provides build version information and runtime metadata.
package version

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	// These are set via ldflags at build time
	Version = ""
	Commit  = ""
	Date    = ""

	once sync.Once
)

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
	cmd := exec.Command("git", "describe", "--always", "--dirty")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "unknown"
	}
	return strings.TrimSpace(out.String())
}

func getGitVersion() string {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		v := strings.TrimSpace(out.String())
		if v != "" {
			return strings.TrimPrefix(v, "v")
		}
	}
	return "dev"
}

func Info() string {
	ensureInitialized()
	return fmt.Sprintf("antigravity-dashboard-tui %s (commit: %s, built: %s, %s/%s)",
		Version, Commit, Date, runtime.GOOS, runtime.GOARCH)
}
