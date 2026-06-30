# velox — Makefile
# Constitution gates: fmt, vet, lint, race tests, security scan.

BINARY := velox
PKG    := ./...
BIN    := bin/$(BINARY)

VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "0.1.0-dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/koller-nexus/velox/internal/version.Version=$(VERSION) \
	-X github.com/koller-nexus/velox/internal/version.Commit=$(COMMIT) \
	-X github.com/koller-nexus/velox/internal/version.Date=$(DATE)

.PHONY: all build test race vet fmt fmt-check lint security vuln tidy clean cross help

all: fmt-check vet lint test ## Run the full local gate

build: ## Build the static binary into bin/
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/velox

test: ## Run unit tests with the race detector
	go test -race $(PKG)

race: test ## Alias for test (race-enabled)

vet: ## Run go vet
	go vet $(PKG)

fmt: ## Format the codebase
	gofmt -w .

fmt-check: ## Fail if any file is unformatted
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "unformatted:"; echo "$$out"; exit 1; fi

lint: ## Run golangci-lint (if installed)
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; skipping"

security vuln: ## Run gosec + govulncheck security gate (HIGH+ fails)
	./scripts/security.sh

tidy: ## Tidy go.mod/go.sum
	go mod tidy

cross: ## Cross-compile static binaries for all targets
	./scripts/build.sh

clean: ## Remove build artifacts
	rm -rf bin dist coverage.out coverage.html

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
