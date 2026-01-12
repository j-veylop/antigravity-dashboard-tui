VERSION := $(shell git describe --tags --always --dirty)
COMMIT := $(shell git describe --always --dirty)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X github.com/j-veylop/antigravity-dashboard-tui/internal/version.Version=$(VERSION) \
           -X github.com/j-veylop/antigravity-dashboard-tui/internal/version.Commit=$(COMMIT) \
           -X github.com/j-veylop/antigravity-dashboard-tui/internal/version.Date=$(DATE)

.PHONY: all build run clean test lint lint-go lint-md fmt coverage check tools help release release-major release-minor release-patch

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

lint: lint-go lint-md

lint-go:
	golangci-lint run

lint-md:
	npx -y markdownlint-cli '**/*.md' --ignore node_modules

lint-fix: lint-go-fix lint-md-fix

lint-go-fix:
	golangci-lint run --fix

lint-md-fix:
	npx -y markdownlint-cli '**/*.md' --ignore node_modules --fix

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
	@echo "  make lint       - Run all linters"
	@echo "  make lint-go    - Run Go linter"
	@echo "  make lint-md    - Run Markdown linter"
	@echo "  make lint-fix   - Run all linters with auto-fix"
	@echo "  make lint-go-fix - Run Go linter with auto-fix"
	@echo "  make lint-md-fix - Run Markdown linter with auto-fix"
	@echo "  make fmt        - Format code"
	@echo "  make check      - Run fmt + lint + test"
	@echo "  make tools      - Install development tools"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make all        - Run lint + test + build"
	@echo "  make release-major - Create and push a new major release tag"
	@echo "  make release-minor - Create and push a new minor release tag"
	@echo "  make release-patch - Create and push a new patch release tag"

CURRENT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)

release-patch:
	@echo "Current version: $(CURRENT_TAG)"
	@VERSION=$$(echo $(CURRENT_TAG) | sed 's/^v//'); \
	IFS='.' read -r major minor patch <<< "$$VERSION"; \
	new_patch=$$((patch + 1)); \
	NEW_TAG="v$$major.$$minor.$$new_patch"; \
	echo "Creating release $$NEW_TAG..."; \
	git tag -a $$NEW_TAG -m "Release $$NEW_TAG" && \
	git push origin $$NEW_TAG

release-minor:
	@echo "Current version: $(CURRENT_TAG)"
	@VERSION=$$(echo $(CURRENT_TAG) | sed 's/^v//'); \
	IFS='.' read -r major minor patch <<< "$$VERSION"; \
	new_minor=$$((minor + 1)); \
	NEW_TAG="v$$major.$$new_minor.0"; \
	echo "Creating release $$NEW_TAG..."; \
	git tag -a $$NEW_TAG -m "Release $$NEW_TAG" && \
	git push origin $$NEW_TAG

release-major:
	@echo "Current version: $(CURRENT_TAG)"
	@VERSION=$$(echo $(CURRENT_TAG) | sed 's/^v//'); \
	IFS='.' read -r major minor patch <<< "$$VERSION"; \
	new_major=$$((major + 1)); \
	NEW_TAG="v$$new_major.0.0"; \
	echo "Creating release $$NEW_TAG..."; \
	git tag -a $$NEW_TAG -m "Release $$NEW_TAG" && \
	git push origin $$NEW_TAG
