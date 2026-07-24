# Keyway — developer Makefile
# Run `make help` for the annotated target list.

SHELL       := /usr/bin/env bash
GO          ?= go
BIN_DIR     := bin
PKG         := github.com/nometria/keyway
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w \
	-X '$(PKG)/internal/version.Version=$(VERSION)' \
	-X '$(PKG)/internal/version.Commit=$(COMMIT)' \
	-X '$(PKG)/internal/version.Date=$(DATE)'

DB_URL      ?= postgres://keyway:keyway@localhost:5432/keyway?sslmode=disable

.DEFAULT_GOAL := help

## ----------------------------------------------------------------------------
## Build
## ----------------------------------------------------------------------------

.PHONY: build
build: build-cli build-runner ## Build both binaries into ./bin

.PHONY: build-cli
build-cli: ## Build the keyway CLI
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/keyway ./cmd/keyway

.PHONY: build-runner
build-runner: ## Build the keyway-runner daemon
	@mkdir -p $(BIN_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/keyway-runner ./cmd/keyway-runner

.PHONY: install
install: ## go install both binaries
	$(GO) install -ldflags "$(LDFLAGS)" ./cmd/...

## ----------------------------------------------------------------------------
## Quality
## ----------------------------------------------------------------------------

.PHONY: test
test: ## Run unit tests with race detector
	$(GO) test -race -count=1 ./...

.PHONY: test-short
test-short: ## Run fast tests only
	$(GO) test -short ./...

.PHONY: cover
cover: ## Run tests and open an HTML coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format all Go code
	$(GO) fmt ./...
	gofmt -s -w .

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	$(GO) mod tidy

.PHONY: check
check: fmt vet lint test ## Run the full local pre-commit gate

## ----------------------------------------------------------------------------
## Database / dev environment
## ----------------------------------------------------------------------------

.PHONY: dev-up
dev-up: ## Start Postgres (+ reference Keycloak) via docker compose
	docker compose up -d

.PHONY: dev-down
dev-down: ## Stop the dev environment
	docker compose down

.PHONY: migrate-up
migrate-up: ## Apply all database migrations
	$(GO) run ./cmd/keyway migrate up --db "$(DB_URL)"

.PHONY: migrate-down
migrate-down: ## Roll back the last migration
	$(GO) run ./cmd/keyway migrate down --db "$(DB_URL)"

## ----------------------------------------------------------------------------
## Run
## ----------------------------------------------------------------------------

.PHONY: serve
serve: build ## Run the API + scheduler on :8080
	$(BIN_DIR)/keyway serve

## ----------------------------------------------------------------------------
## Web UI
## ----------------------------------------------------------------------------

.PHONY: web-install
web-install: ## Install web dependencies
	cd web && npm install

.PHONY: web-dev
web-dev: ## Start the Vite dev server (proxies /v1 to :8080)
	cd web && npm run dev

.PHONY: web-build
web-build: ## Build the production web bundle into web/dist
	cd web && npm run build

## ----------------------------------------------------------------------------
## Benchmark harness (PRD §13)
## ----------------------------------------------------------------------------

.PHONY: bench
bench: ## Run the accuracy benchmark corpus and emit a scorecard
	$(GO) run ./bench/harness --corpus ./bench/corpus --out ./bench/out

.PHONY: bench-report
bench-report: ## Run the benchmark and emit a human-readable report.html + roc.svg
	$(GO) run ./bench/harness --corpus ./bench/corpus --out ./bench/out --report
	@echo "open bench/out/report.html"

.PHONY: validate
validate: ## Validate against documented real-world CVEs/incidents (bench/realworld)
	$(GO) run ./bench/harness --realworld --ci-gate

.PHONY: bench-l2
bench-l2: ## Score the live-probe layer (L2) against real containerized services
	docker compose -f bench/l2/docker-compose.yml up -d --build
	@echo "waiting for issuer..."; for i in $$(seq 1 40); do curl -sf localhost:9000/healthz >/dev/null 2>&1 && break; sleep 1; done
	@sleep 4  # let validators fetch the JWKS
	@$(GO) run ./bench/l2 score --ci-gate; status=$$?; \
		docker compose -f bench/l2/docker-compose.yml down; \
		exit $$status

## ----------------------------------------------------------------------------
## Meta
## ----------------------------------------------------------------------------

.PHONY: docker
docker: ## Build the distroless container image
	docker build -t keyway:$(VERSION) .

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) coverage.out bench/out

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "; printf "\nKeyway make targets\n\n"} \
		/^[a-zA-Z0-9_-]+:.*## / { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 } \
		/^## -/ { } /^## [A-Z]/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 4) }' $(MAKEFILE_LIST)
	@echo
