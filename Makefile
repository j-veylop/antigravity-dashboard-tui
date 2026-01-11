VERSION := $(shell cat VERSION)
COMMIT := $(shell git describe --always --dirty)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X github.com/j-veylop/antigravity-dashboard-tui/internal/version.Version=$(VERSION) \
           -X github.com/j-veylop/antigravity-dashboard-tui/internal/version.Commit=$(COMMIT) \
           -X github.com/j-veylop/antigravity-dashboard-tui/internal/version.Date=$(DATE)

.PHONY: all build run clean test lint fmt coverage check tools help release

all: lint test build

build:
	go build -ldflags "$(LDFLAGS)" -o adt ./cmd/adt

release:
	@echo "Building release binary..."
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS) -s -w" -trimpath -o adt ./cmd/adt
	@echo "✓ Release binary created: adt"
	@ls -lh adt

run: build
	./adt

clean:
	rm -f adt
	rm -f coverage.out coverage.html

test:
	go test -race -v ./...

coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo ""
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'
	@echo "Coverage report: coverage.html"

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

fmt:
	go fmt ./...
	@command -v goimports >/dev/null 2>&1 && goimports -w . || echo "goimports not installed, skipping"

check: fmt lint test
	@echo ""
	@echo "✓ All checks passed!"

tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "✓ Tools installed"

help:
	@echo "Available targets:"
	@echo "  make build      - Build the binary"
	@echo "  make release    - Build optimized release binary"
	@echo "  make run        - Build and run the application"
	@echo "  make test       - Run tests"
	@echo "  make coverage   - Generate test coverage report"
	@echo "  make lint       - Run linter"
	@echo "  make lint-fix   - Run linter with auto-fix"
	@echo "  make fmt        - Format code"
	@echo "  make check      - Run fmt + lint + test"
	@echo "  make tools      - Install development tools"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make all        - Run lint + test + build"
