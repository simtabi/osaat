# osaat -- Makefile. Run `make help`.
#
# The Go module lives in src/, so every go command names it with -C src. A bare
# `go test ./...` at the root finds no packages and passes.
#
# build/ holds one folder per OS and nothing else (build/macos, build/linux,
# build/windows, build/freebsd, plus targets.txt, checksums.txt and, after
# `make dist`, archives/). The host build is build/<host-os>/osaat, macOS written
# as macos and .exe on Windows. `make check-build` (TestBuildLayout) fails on
# anything else.
#
# Nothing here publishes. A release builds on a pushed vX.Y.Z tag or on request (docs/release.md).

SHELL  := /usr/bin/env bash
.DEFAULT_GOAL := help

BINARY  := osaat
PKG     := github.com/simtabi/osaat/src
CMD     := ./cmd/osaat

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.0.0-dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
           -X $(PKG)/internal/version.Version=$(VERSION) \
           -X $(PKG)/internal/version.Commit=$(COMMIT) \
           -X $(PKG)/internal/version.Date=$(DATE)

HOST_GOOS := $(shell go env GOOS)
HOST_OS   := $(if $(filter darwin,$(HOST_GOOS)),macos,$(HOST_GOOS))
EXE       := $(if $(filter windows,$(HOST_GOOS)),.exe,)
BIN       := build/$(HOST_OS)/$(BINARY)$(EXE)

.PHONY: help build build-local install test test-race lint vet fmt tidy cross dist check-build snapshot run clean

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: build-local ## Build build/<host-os>/osaat (the same as build-local)

build-local: ## Build build/<host-os>/osaat with the host's Go toolchain
	CGO_ENABLED=0 go -C src build -trimpath -ldflags '$(LDFLAGS)' -o ../$(BIN) $(CMD)
	@$(MAKE) --no-print-directory check-build

BINDIR ?= $(HOME)/.local/bin

install: build-local ## Build, then copy the binary to $(BINDIR) (default ~/.local/bin)
	install -d "$(BINDIR)"
	install -m 0755 $(BIN) "$(BINDIR)/$(BINARY)$(EXE)"
	@echo "installed $(BINDIR)/$(BINARY)$(EXE)"

test: ## Run go test (includes the release, build-layout and no-container guards)
	go -C src test ./...

test-race: ## Run go test with the race detector and coverage
	go -C src test -race -coverprofile=../coverage.out ./...

vet: ## Run go vet
	go -C src vet ./...

fmt: ## Format code with gofmt
	gofmt -s -w src

tidy: ## go mod tidy
	go -C src mod tidy

lint: ## Run golangci-lint (must be installed)
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not found — install from https://golangci-lint.run"; exit 1; }
	cd src && golangci-lint run --timeout=5m

cross: ## Cross-compile every target in build/targets.txt into build/<os>/
	ARCHIVES=0 VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE) ./src/scripts/build-all.sh
	@$(MAKE) --no-print-directory check-build

dist: ## cross, plus .tar.gz/.zip archives in build/archives/ (publishes nothing)
	ARCHIVES=1 VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE) ./src/scripts/build-all.sh
	@$(MAKE) --no-print-directory check-build

check-build: ## Fail when build/ holds anything but the OS folders, inputs, checksums.txt and archives/
	go -C src test ./internal/release -run 'TestBuildLayout$$' -count=1

snapshot: ## Build the release artifacts into .cache/goreleaser/ without publishing (requires goreleaser)
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not found — install from https://goreleaser.com"; exit 1; }
	goreleaser release --snapshot --clean --skip=publish --config src/.goreleaser.yaml

run: build-local ## Build, then run the binary
	./$(BIN)

clean: ## Remove generated output (build/ inputs are kept) and GoReleaser's scratch dist
	rm -rf ./bin ./dist ./build/dist ./.cache coverage.out \
	  build/macos build/linux build/windows build/freebsd build/openbsd build/netbsd \
	  build/archives build/checksums.txt
