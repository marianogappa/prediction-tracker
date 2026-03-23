.DEFAULT_GOAL := help

.PHONY: build test lint run clean help

build: ## Build server and ingest binaries
	go build -o bin/predictions-server ./cmd/server
	go build -o bin/predictions-ingest ./cmd/ingest

test: ## Run all tests with race detector
	go test -race -count=1 ./...

lint: ## Run go vet and staticcheck
	go vet ./...
	@which staticcheck > /dev/null 2>&1 && staticcheck ./... || echo "staticcheck not installed, skipping"

run: build ## Build and run the server
	./bin/predictions-server

clean: ## Remove build artifacts
	rm -rf bin/

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
