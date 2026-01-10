VERSION := $(shell cat VERSION)
COMMIT := $(shell git describe --always --dirty)
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X github.com/j-veylop/antigravity-dashboard-tui/internal/version.Version=$(VERSION) \
           -X github.com/j-veylop/antigravity-dashboard-tui/internal/version.Commit=$(COMMIT) \
           -X github.com/j-veylop/antigravity-dashboard-tui/internal/version.Date=$(DATE)

.PHONY: build run clean

build:
	go build -ldflags "$(LDFLAGS)" -o adt ./cmd/adt

run: build
	./adt

clean:
	rm -f adt
