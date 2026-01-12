# Contributing to Antigravity Dashboard TUI

Thank you for your interest in contributing! This document provides guidelines and instructions for contributing to the project.

## Development Setup

### Prerequisites

- Go 1.23 or later
- Git
- golangci-lint (install via `make tools`)
- goimports (install via `make tools`)

### Getting Started

1. **Fork and clone the repository**

   ```bash
   git clone https://github.com/YOUR_USERNAME/antigravity-dashboard-tui.git
   cd antigravity-dashboard-tui
   ```

2. **Install development tools**

   ```bash
   make tools
   ```

3. **Build the project**

   ```bash
   make build
   ```

4. **Run tests**

   ```bash
   make test
   ```

## Development Workflow

### Before Making Changes

1. Create a new branch for your work:

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make sure everything works:

   ```bash
   make check
   ```

### Making Changes

1. **Write code** following our coding standards (see below)

2. **Run quality checks frequently:**

   ```bash
   make fmt      # Format code
   make lint     # Check for issues
   make test     # Run tests
   ```

3. **Before committing:**

   ```bash
   make check    # Run all checks (fmt + lint + test)
   ```

### Submitting Changes

1. **Commit your changes** with clear, descriptive messages:

   ```bash
   git add .
   git commit -m "feat: add new feature X"
   ```

   Use conventional commit format:
   - `feat:` New features
   - `fix:` Bug fixes
   - `refactor:` Code changes that don't add features or fix bugs
   - `docs:` Documentation changes
   - `test:` Test additions or fixes
   - `chore:` Build, tooling, or dependency updates

2. **Push to your fork:**

   ```bash
   git push origin feature/your-feature-name
   ```

3. **Create a Pull Request** on GitHub

## Coding Standards

### General Rules

- **Functions should be under 60 lines**
- **Cyclomatic complexity should be under 15**
- **All exported functions need godoc comments**
- **Always check error returns** - never ignore errors
- **Use meaningful variable names**

### Go Style Guide

Follow the official [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) and [Effective Go](https://golang.org/doc/effective_go.html).

### Error Handling

Always handle errors properly:

```go
// Good
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Bad
result, _ := doSomething()  // Never ignore errors!
```

### Defer Cleanup

Always use defer for cleanup, and handle errors:

```go
// Good
defer func() { _ = resource.Close() }()

// Bad
defer resource.Close()  // Error not checked
```

### Security

- Validate all file paths before reading
- Never suppress security warnings without proper validation
- Use `#nosec` comments only after implementing proper checks

```go
// Validate before reading
cleanPath := filepath.Clean(path)
if !strings.HasPrefix(cleanPath, expectedPrefix) {
    return nil
}
content, err := os.ReadFile(cleanPath) // #nosec G304 - validated above
```

## Testing Guidelines

### Writing Tests

- **Aim for 70%+ coverage** on new code
- Use **table-driven tests** where appropriate
- Create **test helpers** to reduce duplication
- Mock external dependencies

### Test Structure

```go
func TestFeatureName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid input", "test", "TEST", false},
        {"empty input", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := YourFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("YourFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("YourFunction() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Helpers

Use `t.Helper()` for test helper functions:

```go
func newTestDB(t *testing.T) *DB {
    t.Helper()
    tmpDir := t.TempDir()
    dbPath := filepath.Join(tmpDir, "test.db")
    db, err := New(dbPath)
    if err != nil {
        t.Fatalf("Failed to create test database: %v", err)
    }
    return db
}
```

## Documentation

### Package Comments

Every package should have a package-level comment:

```go
// Package config provides configuration loading and management
// for the Antigravity Dashboard TUI.
package config
```

### Function Comments

All exported functions need godoc comments:

```go
// LoadConfig loads configuration from all available sources in priority order.
// It returns an error if required credentials are missing.
func LoadConfig() (*Config, error) {
    // ...
}
```

## Pull Request Guidelines

### PR Title

Use conventional commit format in the PR title:

- `feat: add user authentication`
- `fix: resolve quota calculation bug`
- `docs: update installation instructions`

### PR Description

Include:

1. **What** this PR does
2. **Why** the change is needed
3. **How** it was implemented (if complex)
4. **Testing** done

Example:

```markdown
## Summary
Adds support for multiple Google accounts.

## Motivation
Users need to monitor quotas across multiple accounts simultaneously.

## Implementation
- Added account switcher in UI
- Updated state management to handle multiple accounts
- Added account persistence to database

## Testing
- Added unit tests for account management
- Tested manually with 5 accounts
- All existing tests pass
```

### PR Checklist

Before submitting, ensure:

- [ ] `make check` passes
- [ ] New code has tests
- [ ] Documentation is updated
- [ ] Commit messages are clear
- [ ] No unrelated changes included

## Refactoring Guidelines

When refactoring complex functions:

### Extract Message Handlers

For complex `Update()` functions:

```go
// Before
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ComplexMsg:
        // 30 lines of logic...
    }
}

// After
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ComplexMsg:
        cmds = append(cmds, m.handleComplexMsg(msg)...)
    }
}

func (m *Model) handleComplexMsg(msg ComplexMsg) []tea.Cmd {
    // Focused logic in dedicated method
}
```

### Extract Rendering Logic

For complex render functions:

```go
// Before
func renderComplex() string {
    // 100 lines...
}

// After
func renderComplex() string {
    header := renderHeader()
    body := renderBody()
    footer := renderFooter()
    return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
```

## Common Commands

```bash
# Build & Run
make build              # Build binary
make run                # Build and execute
make clean              # Remove artifacts

# Testing
make test               # Run all tests
make coverage           # Generate coverage report

# Code Quality
make lint               # Run linter
make lint-fix           # Auto-fix issues
make fmt                # Format code
make check              # Run all checks

# Help
make help               # Show all targets
```

## Getting Help

- **Issues:** Check existing issues or create a new one
- **Discussions:** Use GitHub Discussions for questions
- **Documentation:** See README.md and CODEBASE_QUALITY_IMPROVEMENTS.md

## Code Review Process

1. **Automated checks** must pass (CI runs lint + test)
2. **Reviewer feedback** should be addressed promptly
3. **Approval required** from at least one maintainer
4. **Squash commits** may be requested before merge

## Recognition

Contributors are recognized in:

- GitHub contributors page
- Release notes (for significant contributions)

Thank you for contributing! 🎉
