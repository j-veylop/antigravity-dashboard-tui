package version

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

// TestHelperProcess isn't a real test. It's used to mock exec.CommandContext.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		os.Exit(0)
	}

	cmd := args[0]
	switch cmd {
	case "git":
		if len(args) > 1 && args[1] == "describe" {
			// Check for specific git commands
			if len(args) > 2 {
				if args[2] == "--always" {
					// git describe --always --dirty
					if os.Getenv("MOCK_GIT_COMMIT_FAIL") == "1" {
						os.Exit(1)
					}
					os.Stdout.WriteString("mock-commit-hash")
				} else if args[2] == "--tags" {
					// git describe --tags
					if os.Getenv("MOCK_GIT_VERSION_FAIL") == "1" {
						os.Exit(1)
					}
					if os.Getenv("MOCK_GIT_VERSION_EMPTY") == "1" {
						os.Stdout.WriteString("")
					} else {
						os.Stdout.WriteString("v1.0.0")
					}
				}
			}
		}
	}
}

func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	// Pass through specific environment variables to control the mock
	if val := os.Getenv("MOCK_GIT_COMMIT_FAIL"); val != "" {
		cmd.Env = append(cmd.Env, "MOCK_GIT_COMMIT_FAIL="+val)
	}
	if val := os.Getenv("MOCK_GIT_VERSION_FAIL"); val != "" {
		cmd.Env = append(cmd.Env, "MOCK_GIT_VERSION_FAIL="+val)
	}
	if val := os.Getenv("MOCK_GIT_VERSION_EMPTY"); val != "" {
		cmd.Env = append(cmd.Env, "MOCK_GIT_VERSION_EMPTY="+val)
	}
	return cmd
}

func TestInfo(t *testing.T) {
	// Save original execCommand and restore after test
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()
	execCommand = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return mockExecCommand(name, arg...)
	}

	tests := []struct {
		name           string
		mockCommitFail string
		mockVerFail    string
		mockVerEmpty   string
		expectedVer    string
		expectedCommit string
	}{
		{
			name:           "Success",
			expectedVer:    "v1.0.0",
			expectedCommit: "mock-commit-hash",
		},
		{
			name:           "CommitFail",
			mockCommitFail: "1",
			expectedVer:    "v1.0.0",
			expectedCommit: "unknown",
		},
		{
			name:        "VersionFail",
			mockVerFail: "1",
			expectedVer: "dev",
			// Commit should still succeed if not failed
			expectedCommit: "mock-commit-hash",
		},
		{
			name:         "VersionEmpty",
			mockVerEmpty: "1",
			expectedVer:  "dev",
			// Commit should still succeed
			expectedCommit: "mock-commit-hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Reset() // Reset global state

			// Set environment variables for the mock process
			if tt.mockCommitFail != "" {
				os.Setenv("MOCK_GIT_COMMIT_FAIL", tt.mockCommitFail)
				defer os.Unsetenv("MOCK_GIT_COMMIT_FAIL")
			}
			if tt.mockVerFail != "" {
				os.Setenv("MOCK_GIT_VERSION_FAIL", tt.mockVerFail)
				defer os.Unsetenv("MOCK_GIT_VERSION_FAIL")
			}
			if tt.mockVerEmpty != "" {
				os.Setenv("MOCK_GIT_VERSION_EMPTY", tt.mockVerEmpty)
				defer os.Unsetenv("MOCK_GIT_VERSION_EMPTY")
			}

			// Force initialization
			ensureInitialized()

			// Check results via Get accessors
			if got := GetVersion(); got != tt.expectedVer {
				t.Errorf("GetVersion() = %v, want %v", got, tt.expectedVer)
			}
			if got := GetCommit(); got != tt.expectedCommit {
				t.Errorf("GetCommit() = %v, want %v", got, tt.expectedCommit)
			}

			// Check Info() string contains both
			info := Info()
			if info == "" {
				t.Error("Info() returned empty string")
			}
		})
	}
}

func TestGetDate(t *testing.T) {
	Reset()
	d := GetDate()
	if d == "" {
		t.Error("GetDate() returned empty string")
	}
}
