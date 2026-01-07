.PHONY: help build run clean deps lint test

BINARY_NAME=network-topology
MAIN_PATH=./cmd/network-topology
OUT_DIR=./out

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

deps: ## Download dependencies
	go mod download
	go mod tidy

build: deps ## Build the binary
	go build -o $(BINARY_NAME) $(MAIN_PATH)/

run: build ## Run the application
	./$(BINARY_NAME)

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	rm -f $(OUT_DIR)/topology.dot $(OUT_DIR)/topology.svg $(OUT_DIR)/topology.png $(OUT_DIR)/index.html

lint: ## Run linter
	golangci-lint run ./...

test: ## Run tests
	go test -v ./...

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

check: fmt vet lint ## Run all checks (fmt, vet, lint)
