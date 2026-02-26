.PHONY: build test lint clean help all

BIN       := cloudflare-tui
BUILD_DIR := ./cmd/cloudflare-tui

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	go build -trimpath -ldflags="-s -w" -o $(BIN) $(BUILD_DIR)

test: ## Run all tests
	go test ./...

lint: ## Run golangci-lint
	golangci-lint run ./...

clean: ## Remove build artifacts
	rm -f $(BIN)

all: lint test build ## Lint, test, and build (CI target)
