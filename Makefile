.PHONY: help build test test-race lint vet fmt tidy snapshot clean

BINARY  := osaat
PKG     := github.com/simtabi/osaat
CMD     := ./cmd/osaat

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.0.0-dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
           -X $(PKG)/internal/version.Version=$(VERSION) \
           -X $(PKG)/internal/version.Commit=$(COMMIT) \
           -X $(PKG)/internal/version.Date=$(DATE)

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the osaat binary into ./bin/
	go build -ldflags '$(LDFLAGS)' -o ./bin/$(BINARY) $(CMD)

test: ## Run go test
	go test ./...

test-race: ## Run go test with the race detector and coverage
	go test -race -coverprofile=coverage.out ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Format code with gofmt
	gofmt -s -w .

tidy: ## go mod tidy
	go mod tidy

lint: ## Run golangci-lint (must be installed)
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found — install from https://golangci-lint.run"; exit 1; }
	golangci-lint run --timeout=5m

snapshot: ## Build release artifacts locally (requires goreleaser)
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not found — install from https://goreleaser.com"; exit 1; }
	goreleaser release --snapshot --clean

clean: ## Clean build artifacts
	rm -rf ./bin ./dist ./build/dist coverage.out
