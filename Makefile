# Makefile for GeoIP Server

# Variables
BINARY_NAME=geoip-server
GO_FILES=$(shell find . -name "*.go" -type f)
DOCKER_IMAGE=geoip-server
DOCKER_TAG=latest

.PHONY: help build run clean test docker-build docker-run deps download-data

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

# Development targets
deps: ## Install dependencies
	go mod download
	go mod tidy

build: deps ## Build the binary
	go build -o $(BINARY_NAME) .

run: build download-data ## Build and run the server
	./$(BINARY_NAME)

clean: ## Clean build artifacts
	go clean
	rm -f $(BINARY_NAME)
	rm -rf data/

test: ## Run tests
	go test -v ./...

download-data: ## Download MaxMind GeoIP data
	@if [ -z "$(MAXMIND_LICENSE_KEY)" ]; then \
		echo "Error: MAXMIND_LICENSE_KEY environment variable is required"; \
		echo "Please set it with: export MAXMIND_LICENSE_KEY=your_license_key"; \
		exit 1; \
	fi
	chmod +x scripts/download-geoip-data.sh
	./scripts/download-geoip-data.sh

# Docker targets
docker-build: ## Build Docker images
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	docker build -f Dockerfile.init -t $(DOCKER_IMAGE)-init:$(DOCKER_TAG) .

docker-run: ## Run the server in Docker
	@if [ -z "$(MAXMIND_LICENSE_KEY)" ]; then \
		echo "Error: MAXMIND_LICENSE_KEY environment variable is required"; \
		echo "Please set it with: export MAXMIND_LICENSE_KEY=your_license_key"; \
		exit 1; \
	fi
	docker run -p 8080:8080 -e MAXMIND_LICENSE_KEY=$(MAXMIND_LICENSE_KEY) $(DOCKER_IMAGE):$(DOCKER_TAG)

docker-compose-up: ## Start with Docker Compose
	docker-compose up --build

docker-compose-down: ## Stop Docker Compose services
	docker-compose down

# Development helpers
fmt: ## Format Go code
	go fmt ./...

lint: ## Run golangci-lint
	golangci-lint run

# API testing
test-api: ## Test API endpoints (requires server to be running)
	@echo "Testing health endpoint..."
	curl -f http://localhost:8080/health
	@echo "\nTesting IP lookup..."
	curl -f http://localhost:8080/geoip/8.8.8.8
	@echo "\nTesting client IP..."
	curl -f http://localhost:8080/myip

# Kubernetes targets
k8s-deploy: ## Deploy to Kubernetes
	kubectl apply -f k8s/deployment.yaml

k8s-delete: ## Delete Kubernetes deployment
	kubectl delete -f k8s/deployment.yaml

# Release targets
release-build: ## Build for multiple platforms
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe .
