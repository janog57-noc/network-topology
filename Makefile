.PHONY: help build run clean deps lint test

BINARY_NAME=network-topology
MAIN_PATH=./cmd/network-topology
OUTPUT_DIR=.

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

deps: ## Download dependencies
	go mod download
	go mod tidy

build: deps ## Build the binary
	go build -o $(OUTPUT_DIR)/$(BINARY_NAME) $(MAIN_PATH)/

run: build ## Run the application
	./$(BINARY_NAME)

clean: ## Clean build artifacts
	rm -f $(OUTPUT_DIR)/$(BINARY_NAME)
	rm -f topology.dot topology.svg topology.png index.html

lint: ## Run linter
	golangci-lint run ./...

test: ## Run tests
	go test -v ./...

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

check: fmt vet lint ## Run all checks (fmt, vet, lint)
