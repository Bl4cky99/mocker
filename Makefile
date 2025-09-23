BIN 	?= mocker
PKG 	?= ./cmd/mocker
OUT_DIR ?= bin
CONFIG 	?= config.yaml

GO			?= go
GOTEST_OPTS	?= -race -timeout=2m
COVERFILE 	?= coverage.out

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || echo unknown)

LDFLAGS := -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.date=$(DATE)'

.DEFAULT_GOAL := help

GREEN  := \033[32m
CYAN   := \033[36m
RESET  := \033[0m

.PHONY: help
help: ## Show this help
	@echo "mocker – developer commands"; echo
	@awk -F':.*##' '/^[a-zA-Z0-9_\/-]+:.*##/ \
	{printf "  $(CYAN)%-18s$(RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build binary into ./bin/
	@mkdir -p $(OUT_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(OUT_DIR)/$(BIN) $(PKG)
	@echo "$(GREEN)Built $(OUT_DIR)/$(BIN) [$(VERSION) $(COMMIT)]$(RESET)"

.PHONY: run
run: build ## Build & run 'serve' with CONFIG (override via CONFIG=...)
	./$(OUT_DIR)/$(BIN) serve -c $(CONFIG) -p

.PHONY: serve
serve: build ## Alias für 'run'
	./$(OUT_DIR)/$(BIN) serve -c $(CONFIG)

.PHONY: validate
validate: build ## Validate config (CONFIG=... für andere Datei)
	./$(OUT_DIR)/$(BIN) validate -c $(CONFIG)

.PHONY: test
test: ## Run unit tests (race)
	$(GO) test $(GOTEST_OPTS) ./...

.PHONY: test.v
test.v: ## Run unit tests (race + verbose)
	$(GO) test $(GOTEST_OPTS) -v ./...

.PHONY: cover
cover: ## Run tests with coverage and show summary
	$(GO) test -coverprofile=$(COVERFILE) ./...
	@$(GO) tool cover -func=$(COVERFILE) | tail -n 1

.PHONY: cover-html
cover-html: cover ## Open HTML coverage report
	@$(GO) tool cover -html=$(COVERFILE) -o $(OUT_DIR)/coverage.html
	@echo "$(GREEN)Open $(OUT_DIR)/coverage.html in your browser$(RESET)"

.PHONY: fmt
fmt: ## Format code with go fmt (and goimports if installed)
	$(GO) fmt ./...
	@command -v goimports >/dev/null 2>&1 && goimports -w . || true

.PHONY: vet
vet: ## Static analysis with go vet
	$(GO) vet ./...

.PHONY: tidy
tidy: ## Sync go.mod/go.sum
	$(GO) mod tidy

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(OUT_DIR) $(COVERFILE)

.PHONY: version
version: ## Print build-time version values
	@echo "VERSION=$(VERSION) COMMIT=$(COMMIT) DATE=$(DATE)"