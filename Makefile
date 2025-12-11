.PHONY: help install check lint format test clean docker-build docker-up docker-down

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

help: ## Show this help message
	@echo "$(BLUE)QuietHire - Available Commands:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

install: ## Install development dependencies
	@echo "$(BLUE)Installing Go tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "$(BLUE)Installing Python tools...$(NC)"
	@pip install --upgrade pip
	@pip install pre-commit ruff mypy
	@echo "$(GREEN)✓ All tools installed$(NC)"

check: ## Run all code quality checks (linting + type checking)
	@echo "$(BLUE)========================================$(NC)"
	@echo "$(BLUE)Running All Code Quality Checks$(NC)"
	@echo "$(BLUE)========================================$(NC)"
	@$(MAKE) lint
	@$(MAKE) type-check
	@echo ""
	@echo "$(GREEN)========================================$(NC)"
	@echo "$(GREEN)✓ All checks passed!$(NC)"
	@echo "$(GREEN)========================================$(NC)"

lint: lint-go lint-python ## Run all linters

lint-go: ## Lint Go code
	@echo "$(YELLOW)Linting Go code...$(NC)"
	@cd apps/api && golangci-lint run --config ../../.golangci.yml ./... || (echo "$(RED)✗ Go linting failed in apps/api$(NC)" && exit 1)
	@cd apps/proxy-manager && golangci-lint run --config ../../.golangci.yml ./... || (echo "$(RED)✗ Go linting failed in apps/proxy-manager$(NC)" && exit 1)
	@echo "$(GREEN)✓ Go linting passed$(NC)"

lint-python: ## Lint Python code
	@echo "$(YELLOW)Linting Python code...$(NC)"
	@ruff check apps/parser apps/crawler-python apps/osint-discovery || (echo "$(RED)✗ Python linting failed$(NC)" && exit 1)
	@echo "$(GREEN)✓ Python linting passed$(NC)"

type-check: type-check-go type-check-python ## Run all type checkers

type-check-go: ## Type check Go code (via golangci-lint)
	@echo "$(YELLOW)Type checking Go code...$(NC)"
	@cd apps/api && go vet ./... || (echo "$(RED)✗ Go type checking failed$(NC)" && exit 1)
	@cd apps/proxy-manager && go vet ./... || (echo "$(RED)✗ Go type checking failed$(NC)" && exit 1)
	@echo "$(GREEN)✓ Go type checking passed$(NC)"

type-check-python: ## Type check Python code with mypy
	@echo "$(YELLOW)Type checking Python code...$(NC)"
	@mypy apps/parser/main.py --config-file pyproject.toml || (echo "$(RED)✗ Python type checking failed$(NC)" && exit 1)
	@mypy apps/crawler-python/main.py --config-file pyproject.toml || (echo "$(RED)✗ Python type checking failed$(NC)" && exit 1)
	@mypy apps/osint-discovery/main.py --config-file pyproject.toml || (echo "$(RED)✗ Python type checking failed$(NC)" && exit 1)
	@echo "$(GREEN)✓ Python type checking passed$(NC)"

format: format-go format-python ## Format all code

format-go: ## Format Go code
	@echo "$(YELLOW)Formatting Go code...$(NC)"
	@gofmt -w -s apps/api apps/proxy-manager
	@goimports -w apps/api apps/proxy-manager
	@echo "$(GREEN)✓ Go code formatted$(NC)"

format-python: ## Format Python code
	@echo "$(YELLOW)Formatting Python code...$(NC)"
	@ruff format apps/parser apps/crawler-python apps/osint-discovery
	@echo "$(GREEN)✓ Python code formatted$(NC)"

fix: ## Auto-fix linting issues where possible
	@echo "$(YELLOW)Auto-fixing issues...$(NC)"
	@ruff check --fix apps/parser apps/crawler-python apps/osint-discovery
	@$(MAKE) format
	@echo "$(GREEN)✓ Auto-fix complete$(NC)"

test: ## Run tests (placeholder - add tests later)
	@echo "$(YELLOW)Running tests...$(NC)"
	@echo "$(BLUE)Note: Tests not yet implemented$(NC)"

clean: ## Clean build artifacts and caches
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@find . -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true
	@find . -type d -name ".mypy_cache" -exec rm -rf {} + 2>/dev/null || true
	@find . -type d -name ".ruff_cache" -exec rm -rf {} + 2>/dev/null || true
	@find . -type d -name ".pytest_cache" -exec rm -rf {} + 2>/dev/null || true
	@find . -type f -name "*.pyc" -delete 2>/dev/null || true
	@echo "$(GREEN)✓ Cleaned$(NC)"

docker-build: ## Build all Docker images
	@echo "$(YELLOW)Building Docker images...$(NC)"
	@docker-compose build
	@echo "$(GREEN)✓ Docker images built$(NC)"

docker-up: ## Start all Docker services
	@echo "$(YELLOW)Starting Docker services...$(NC)"
	@docker-compose up -d
	@echo "$(GREEN)✓ Docker services started$(NC)"

docker-down: ## Stop all Docker services
	@echo "$(YELLOW)Stopping Docker services...$(NC)"
	@docker-compose down
	@echo "$(GREEN)✓ Docker services stopped$(NC)"

docker-logs: ## Show Docker service logs
	@docker-compose logs -f

pre-commit-install: ## Install pre-commit hooks
	@echo "$(YELLOW)Installing pre-commit hooks...$(NC)"
	@pre-commit install
	@echo "$(GREEN)✓ Pre-commit hooks installed$(NC)"

pre-commit-run: ## Run pre-commit on all files
	@echo "$(YELLOW)Running pre-commit checks...$(NC)"
	@pre-commit run --all-files

ci: check test ## Run CI checks (lint + type-check + test)
	@echo "$(GREEN)✓ All CI checks passed$(NC)"

.DEFAULT_GOAL := help
