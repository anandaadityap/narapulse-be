# NaraPulse Backend Makefile

# Variables
APP_NAME=narapulse-be
GO_VERSION=1.21
PORT=8080
DB_URL=postgres://postgres:postgres@localhost:5432/narapulsedb?sslmode=disable

# Colors for output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[1;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help build run test clean deps migrate-up migrate-down migrate-create swagger docker-build docker-run

# Default target
help: ## Show this help message
	@echo "$(BLUE)NaraPulse Backend - Available Commands:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

# Development
run: ## Run the application in development mode
	@echo "$(YELLOW)Starting $(APP_NAME) on port $(PORT)...$(NC)"
	go run main.go

dev: ## Run with hot reload (requires air)
	@echo "$(YELLOW)Starting $(APP_NAME) with hot reload...$(NC)"
	air

build: ## Build the application
	@echo "$(YELLOW)Building $(APP_NAME)...$(NC)"
	go build -o bin/$(APP_NAME) main.go
	@echo "$(GREEN)Build completed: bin/$(APP_NAME)$(NC)"

test: ## Run tests
	@echo "$(YELLOW)Running tests...$(NC)"
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "$(YELLOW)Running tests with coverage...$(NC)"
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

# Dependencies
deps: ## Download and tidy dependencies
	@echo "$(YELLOW)Downloading dependencies...$(NC)"
	go mod download
	go mod tidy
	@echo "$(GREEN)Dependencies updated$(NC)"

deps-update: ## Update all dependencies
	@echo "$(YELLOW)Updating dependencies...$(NC)"
	go get -u ./...
	go mod tidy
	@echo "$(GREEN)Dependencies updated$(NC)"

# Database
migrate-up: ## Run database migrations up
	@echo "$(YELLOW)Running migrations up...$(NC)"
	goose -dir migrations postgres "$(DB_URL)" up
	@echo "$(GREEN)Migrations completed$(NC)"

migrate-down: ## Run database migrations down
	@echo "$(YELLOW)Running migrations down...$(NC)"
	goose -dir migrations postgres "$(DB_URL)" down
	@echo "$(GREEN)Migrations rolled back$(NC)"

migrate-status: ## Check migration status
	@echo "$(YELLOW)Checking migration status...$(NC)"
	goose -dir migrations postgres "$(DB_URL)" status

migrate-create: ## Create a new migration (usage: make migrate-create name=migration_name)
	@if [ -z "$(name)" ]; then \
		echo "$(RED)Error: Please provide a migration name. Usage: make migrate-create name=migration_name$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)Creating migration: $(name)...$(NC)"
	goose -dir migrations create $(name) sql
	@echo "$(GREEN)Migration created$(NC)"

db-reset: ## Reset database (drop and recreate)
	@echo "$(YELLOW)Resetting database...$(NC)"
	goose -dir migrations postgres "$(DB_URL)" reset
	make migrate-up
	@echo "$(GREEN)Database reset completed$(NC)"

# Documentation
swagger: ## Generate Swagger documentation
	@echo "$(YELLOW)Generating Swagger documentation...$(NC)"
	swag init -g main.go -o docs/
	@echo "$(GREEN)Swagger documentation generated$(NC)"

swagger-serve: ## Serve Swagger documentation
	@echo "$(YELLOW)Serving Swagger documentation at http://localhost:$(PORT)/swagger/$(NC)"
	make run

# Code Quality
fmt: ## Format code
	@echo "$(YELLOW)Formatting code...$(NC)"
	go fmt ./...
	@echo "$(GREEN)Code formatted$(NC)"

vet: ## Run go vet
	@echo "$(YELLOW)Running go vet...$(NC)"
	go vet ./...
	@echo "$(GREEN)Vet completed$(NC)"

lint: ## Run golangci-lint (requires golangci-lint)
	@echo "$(YELLOW)Running linter...$(NC)"
	golangci-lint run
	@echo "$(GREEN)Linting completed$(NC)"

# Docker
docker-build: ## Build Docker image
	@echo "$(YELLOW)Building Docker image...$(NC)"
	docker build -t $(APP_NAME):latest .
	@echo "$(GREEN)Docker image built: $(APP_NAME):latest$(NC)"

docker-run: ## Run Docker container
	@echo "$(YELLOW)Running Docker container...$(NC)"
	docker run -p $(PORT):$(PORT) --env-file .env $(APP_NAME):latest

docker-compose-up: ## Start services with docker-compose
	@echo "$(YELLOW)Starting services with docker-compose...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)Services started$(NC)"

docker-compose-down: ## Stop services with docker-compose
	@echo "$(YELLOW)Stopping services with docker-compose...$(NC)"
	docker-compose down
	@echo "$(GREEN)Services stopped$(NC)"

# Utilities
clean: ## Clean build artifacts and cache
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -cache
	@echo "$(GREEN)Clean completed$(NC)"

install-tools: ## Install development tools
	@echo "$(YELLOW)Installing development tools...$(NC)"
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/cosmtrek/air@latest
	@echo "$(GREEN)Development tools installed$(NC)"

env-example: ## Create .env.example file
	@echo "$(YELLOW)Creating .env.example...$(NC)"
	@echo "PORT=8080" > .env.example
	@echo "DATABASE_URL=postgres://postgres:postgres@localhost:5432/narapulsedb?sslmode=disable" >> .env.example
	@echo "JWT_SECRET=your-secret-key-change-this-in-production" >> .env.example
	@echo "ENVIRONMENT=development" >> .env.example
	@echo "$(GREEN).env.example created$(NC)"

# Setup
setup: deps install-tools env-example ## Setup development environment
	@echo "$(GREEN)Development environment setup completed!$(NC)"
	@echo "$(BLUE)Next steps:$(NC)"
	@echo "1. Copy .env.example to .env and update values"
	@echo "2. Create PostgreSQL database"
	@echo "3. Run 'make migrate-up' to setup database"
	@echo "4. Run 'make run' to start the server"

# Production
build-prod: ## Build for production
	@echo "$(YELLOW)Building for production...$(NC)"
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/$(APP_NAME) main.go
	@echo "$(GREEN)Production build completed$(NC)"

# Health check
health: ## Check if the server is running
	@echo "$(YELLOW)Checking server health...$(NC)"
	@curl -f http://localhost:$(PORT)/health || echo "$(RED)Server is not running$(NC)"