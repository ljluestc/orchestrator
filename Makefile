# Makefile for orchestrator project
# Provides commands for development, testing, and deployment

.PHONY: help build test test-coverage test-unit test-integration test-e2e clean fmt vet lint security install-tools pre-commit setup

# Default target
.DEFAULT_GOAL := help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Project parameters
BINARY_NAME=orchestrator
BINARY_UNIX=$(BINARY_NAME)_unix
PROBE_BINARY_NAME=probe-agent
PROBE_BINARY_UNIX=$(PROBE_BINARY_NAME)_unix

# Coverage parameters
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html
COVERAGE_THRESHOLD=80

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

help: ## Show this help message
	@echo "Available commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""

setup: install-tools ## Setup development environment
	@echo "$(GREEN)Setting up development environment...$(NC)"
	@$(GOMOD) download
	@$(GOMOD) tidy
	@echo "$(GREEN)Development environment ready!$(NC)"

install-tools: ## Install required development tools
	@echo "$(BLUE)Installing development tools...$(NC)"
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@go install github.com/tommy-muehle/go-mnd/v2/cmd/go-mnd@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(GREEN)Tools installed successfully!$(NC)"

build: ## Build all binaries
	@echo "$(BLUE)Building orchestrator...$(NC)"
	@$(GOBUILD) -o $(BINARY_NAME) .
	@echo "$(BLUE)Building probe-agent...$(NC)"
	@$(GOBUILD) -o $(PROBE_BINARY_NAME) ./cmd/probe-agent
	@echo "$(GREEN)Build completed!$(NC)"

build-linux: ## Build binaries for Linux
	@echo "$(BLUE)Building for Linux...$(NC)"
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) .
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(PROBE_BINARY_UNIX) ./cmd/probe-agent
	@echo "$(GREEN)Linux build completed!$(NC)"

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning...$(NC)"
	@$(GOCLEAN)
	@rm -f $(BINARY_NAME)
	@rm -f $(PROBE_BINARY_NAME)
	@rm -f $(BINARY_UNIX)
	@rm -f $(PROBE_BINARY_UNIX)
	@rm -f $(COVERAGE_FILE)
	@rm -f $(COVERAGE_HTML)
	@echo "$(GREEN)Clean completed!$(NC)"

fmt: ## Format Go code
	@echo "$(BLUE)Formatting code...$(NC)"
	@$(GOFMT) ./...
	@echo "$(GREEN)Code formatted!$(NC)"

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	@$(GOVET) ./...
	@echo "$(GREEN)go vet passed!$(NC)"

lint: ## Run linter
	@echo "$(BLUE)Running linter...$(NC)"
	@golangci-lint run
	@echo "$(GREEN)Linting passed!$(NC)"

security: ## Run security checks
	@echo "$(BLUE)Running security checks...$(NC)"
	@gosec ./...
	@echo "$(GREEN)Security checks completed!$(NC)"

test: ## Run all tests
	@echo "$(BLUE)Running all tests...$(NC)"
	@$(GOTEST) -v ./...
	@echo "$(GREEN)All tests passed!$(NC)"

test-race: ## Run tests with race detection
	@echo "$(BLUE)Running tests with race detection...$(NC)"
	@$(GOTEST) -race ./...
	@echo "$(GREEN)Race detection tests passed!$(NC)"

test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@$(GOTEST) -v -coverprofile=$(COVERAGE_FILE) ./...
	@echo "$(BLUE)Coverage report:$(NC)"
	@go tool cover -func=$(COVERAGE_FILE) | tail -1
	@echo "$(GREEN)Coverage tests completed!$(NC)"

test-coverage-html: test-coverage ## Generate HTML coverage report
	@echo "$(BLUE)Generating HTML coverage report...$(NC)"
	@go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "$(GREEN)Coverage report generated: $(COVERAGE_HTML)$(NC)"

test-unit: ## Run unit tests only
	@echo "$(BLUE)Running unit tests...$(NC)"
	@$(GOTEST) -v -short ./...
	@echo "$(GREEN)Unit tests passed!$(NC)"

test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	@$(GOTEST) -v -tags=integration ./...
	@echo "$(GREEN)Integration tests passed!$(NC)"

test-e2e: ## Run end-to-end tests
	@echo "$(BLUE)Running end-to-end tests...$(NC)"
	@$(GOTEST) -v -tags=e2e ./...
	@echo "$(GREEN)End-to-end tests passed!$(NC)"

benchmark: ## Run benchmarks
	@echo "$(BLUE)Running benchmarks...$(NC)"
	@$(GOTEST) -bench=. ./...
	@echo "$(GREEN)Benchmarks completed!$(NC)"

pre-commit: ## Run pre-commit checks
	@echo "$(BLUE)Running pre-commit checks...$(NC)"
	@.git/hooks/pre-commit
	@echo "$(GREEN)Pre-commit checks passed!$(NC)"

check-coverage: test-coverage ## Check if coverage meets threshold
	@echo "$(BLUE)Checking coverage threshold...$(NC)"
	@COVERAGE=$$(go tool cover -func=$(COVERAGE_FILE) | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "$(RED)Coverage $$COVERAGE% is below threshold $(COVERAGE_THRESHOLD)%$(NC)"; \
		exit 1; \
	else \
		echo "$(GREEN)Coverage $$COVERAGE% meets threshold $(COVERAGE_THRESHOLD)%$(NC)"; \
	fi

ci: clean fmt vet lint security test-coverage check-coverage ## Run CI pipeline
	@echo "$(GREEN)CI pipeline completed successfully!$(NC)"

dev: ## Start development mode (build and test)
	@echo "$(BLUE)Starting development mode...$(NC)"
	@make clean
	@make build
	@make test-coverage
	@echo "$(GREEN)Development mode ready!$(NC)"

docker-build: ## Build Docker images
	@echo "$(BLUE)Building Docker images...$(NC)"
	@docker build -t orchestrator:latest .
	@docker build -t probe-agent:latest -f cmd/probe-agent/Dockerfile .
	@echo "$(GREEN)Docker images built!$(NC)"

docker-run: ## Run with Docker Compose
	@echo "$(BLUE)Starting with Docker Compose...$(NC)"
	@docker-compose up -d
	@echo "$(GREEN)Docker Compose started!$(NC)"

docker-stop: ## Stop Docker Compose
	@echo "$(BLUE)Stopping Docker Compose...$(NC)"
	@docker-compose down
	@echo "$(GREEN)Docker Compose stopped!$(NC)"

install: build ## Install binaries to GOPATH/bin
	@echo "$(BLUE)Installing binaries...$(NC)"
	@go install .
	@go install ./cmd/probe-agent
	@echo "$(GREEN)Binaries installed!$(NC)"

uninstall: ## Remove installed binaries
	@echo "$(BLUE)Uninstalling binaries...$(NC)"
	@go clean -i .
	@go clean -i ./cmd/probe-agent
	@echo "$(GREEN)Binaries uninstalled!$(NC)"

# Git hooks
install-hooks: ## Install git hooks
	@echo "$(BLUE)Installing git hooks...$(NC)"
	@chmod +x .git/hooks/pre-commit
	@echo "$(GREEN)Git hooks installed!$(NC)"

# Documentation
docs: ## Generate documentation
	@echo "$(BLUE)Generating documentation...$(NC)"
	@mkdir -p docs
	@go doc -all ./... > docs/api.md
	@echo "$(GREEN)Documentation generated!$(NC)"

# Release
release: clean test-coverage check-coverage build-linux ## Create release
	@echo "$(BLUE)Creating release...$(NC)"
	@mkdir -p release
	@cp $(BINARY_UNIX) release/
	@cp $(PROBE_BINARY_UNIX) release/
	@echo "$(GREEN)Release created in release/ directory!$(NC)"

# Show project status
status: ## Show project status
	@echo "$(BLUE)Project Status:$(NC)"
	@echo "  Go version: $(shell go version)"
	@echo "  Go modules: $(shell go list -m all | wc -l) modules"
	@echo "  Test files: $(shell find . -name '*_test.go' | wc -l)"
	@echo "  Source files: $(shell find . -name '*.go' -not -name '*_test.go' | wc -l)"
	@echo "  Coverage: $(shell if [ -f $(COVERAGE_FILE) ]; then go tool cover -func=$(COVERAGE_FILE) | tail -1 | awk '{print $$3}'; else echo 'Not available'; fi)"
