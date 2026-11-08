# vigil -- Makefile. Run `make help`.
#
# The Go module lives in src/, so every go command names it with -C src. A bare
# `go test ./...` at the root finds no packages and passes.
#
# build/ holds one folder per OS and nothing else (build/macos, build/linux,
# build/windows, ..., plus targets.txt, nfpm.yaml, checksums.txt and, after
# `make dist`, archives/). The host build is build/<host-os>/vigil, macOS written
# as macos and .exe on Windows. `make check-build` (TestBuildLayout) fails on
# anything else.
#
# Nothing here publishes. Releases are dispatched by hand (docs/release.md).

SHELL := /usr/bin/env bash
# `make` alone builds, as it did before the help target existed.
.DEFAULT_GOAL := build

BINARY  := vigil
PKG     := github.com/simtabi/vigil/src/internal/cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X $(PKG).version=$(VERSION) -X $(PKG).commit=$(COMMIT) -X $(PKG).date=$(DATE)
MAIN    := ./cmd/vigil

HOST_GOOS := $(shell go env GOOS)
HOST_OS   := $(if $(filter darwin,$(HOST_GOOS)),macos,$(HOST_GOOS))
EXE       := $(if $(filter windows,$(HOST_GOOS)),.exe,)
BIN       := build/$(HOST_OS)/$(BINARY)$(EXE)

# `make install` copies the host build here (may need sudo).
BINDIR ?= /usr/local/bin

.PHONY: help build build-local install test vet fmt lint cross cross-check dist check-build icons clean

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

# cgo stays at the toolchain default: the macOS input engine links
# CoreGraphics/IOKit through cgo (src/internal/activity/input_darwin.go).
build: build-local ## Build $(BIN) for the current OS (the same as build-local)

build-local: ## Build $(BIN) with the host's Go toolchain
	go -C src build -trimpath -ldflags "$(LDFLAGS)" -o ../$(BIN) $(MAIN)
	@$(MAKE) --no-print-directory check-build

install: build-local ## Build, then copy $(BIN) to $(BINDIR) (default /usr/local/bin; may need sudo)
	install -d "$(BINDIR)"
	install -m 0755 $(BIN) "$(BINDIR)/$(BINARY)$(EXE)"

test: ## Run the tests with the race detector (includes the release, build-layout and no-container guards)
	go -C src test -race ./...

vet: ## Run go vet
	go -C src vet ./...

fmt: ## Check formatting (fails if any file needs gofmt)
	@out=$$(gofmt -l src); if [ -n "$$out" ]; then echo "needs gofmt:"; echo "$$out"; exit 1; fi

lint: ## Run golangci-lint (config in src/.golangci.yml), if installed
	cd src && golangci-lint run ./...

cross: ## Cross-compile every target in build/targets.txt into build/<os>/
	VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE) ./src/scripts/build-all.sh
	@$(MAKE) --no-print-directory check-build

cross-check: ## Compile every package for the cgo-free targets, writing nothing (the former `make cross`)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go -C src build ./...
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go -C src build ./...
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go -C src build ./...

dist: ## cross, plus .tar.gz/.zip archives in build/archives/ (publishes nothing)
	ARCHIVES=1 VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE) ./src/scripts/build-all.sh
	@$(MAKE) --no-print-directory check-build

check-build: ## Fail when build/ holds anything but the OS folders, inputs, checksums.txt and archives/
	go -C src test ./internal/release -run 'TestBuildLayout$$' -count=1

icons: ## Regenerate brand PNGs + Windows resource (.syso) from assets/vigil.svg
	cd src/tools/iconz && go run .

clean: ## Remove generated output (build/ inputs are kept) and GoReleaser's scratch dist
	rm -rf .cache build/macos build/linux build/windows build/freebsd build/openbsd build/netbsd build/archives build/checksums.txt
	@# Output of the layout before src/: ./vigil and ./dist.
	rm -rf $(BINARY) $(BINARY).exe dist
