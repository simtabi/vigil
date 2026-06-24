BINARY := mta
PKG := github.com/simtabi/ms-teams-activity
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(PKG)/internal/cli.version=$(VERSION)
MAIN := ./cmd/mta

.PHONY: build test vet fmt lint cross install clean

build: ## Build the binary for the current OS
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) $(MAIN)

test: ## Run tests with the race detector
	go test -race ./...

vet: ## Run go vet
	go vet ./...

fmt: ## Check formatting (fails if any file needs gofmt)
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "needs gofmt:"; echo "$$out"; exit 1; fi

cross: ## Cross-compile the cgo-free targets as a smoke check
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build ./...
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build ./...

install: build ## Build and install to /usr/local/bin (may need sudo)
	install -m 0755 $(BINARY) /usr/local/bin/$(BINARY)

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf dist
