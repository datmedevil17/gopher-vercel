.PHONY: help build run test clean docker-up docker-down migrate

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	go build -o bin/api ./cmd/api

build-handler: ## Build the request handler
	go build -o bin/request-handler ./cmd/request-handler

run: ## Run the application
	go run ./cmd/api/main.go

run-handler: ## Run the request handler
	go run ./cmd/request-handler/main.go

run-all: ## Run both API and Request Handler
	@trap 'kill 0' SIGINT; \
	go run ./cmd/api/main.go & \
	go run ./cmd/request-handler/main.go & \
	wait

test: ## Run tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf /tmp/deploy-*

docker-up: ## Start Docker containers
	docker-compose up -d

docker-down: ## Stop Docker containers
	docker-compose down

docker-logs: ## View Docker logs
	docker-compose logs -f

docker-rebuild: ## Rebuild and restart Docker containers
	docker-compose up -d --build

migrate: ## Run database migrations
	go run ./cmd/api/main.go migrate

deps: ## Install dependencies
	go mod download
	go mod tidy

lint: ## Run linter
	golangci-lint run