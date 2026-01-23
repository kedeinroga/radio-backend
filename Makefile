.PHONY: help build run test test-integration lint fmt clean migrate migrate-up migrate-down security docker-build docker-run cloudrun-logs terraform

# Variables
BINARY_NAME=radio-backend
MAIN_PATH=./cmd/server
BUILD_DIR=./bin
COVERAGE_FILE=coverage.out

# Load environment variables from .env if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

# Colors for output
GREEN=\033[0;32m
YELLOW=\033[0;33m
NC=\033[0m # No Color

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  ${GREEN}%-15s${NC} %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the application
	@echo "${GREEN}Building ${BINARY_NAME}...${NC}"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "${GREEN}Build complete: $(BUILD_DIR)/$(BINARY_NAME)${NC}"

run: swagger-generate ## Run the application
	@echo "${GREEN}Running ${BINARY_NAME}...${NC}"
	@go run $(MAIN_PATH)/*.go

dev: ## Run with live reload (requires air)
	@echo "${GREEN}Starting development server with live reload...${NC}"
	@air

test: ## Run unit tests with coverage
	@echo "${GREEN}Running tests...${NC}"
	@go test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "${GREEN}Coverage report generated: coverage.html${NC}"

test-quick: ## Run tests without coverage (faster)
	@echo "${GREEN}Running quick tests...${NC}"
	@go test ./...

test-coverage: test ## Show test coverage in terminal
	@echo "${GREEN}Test coverage summary:${NC}"
	@go tool cover -func=$(COVERAGE_FILE)

test-coverage-html: test ## Open coverage report in browser
	@echo "${GREEN}Opening coverage report...${NC}"
	@open coverage.html || xdg-open coverage.html

test-unit: ## Run only unit tests
	@echo "${GREEN}Running unit tests...${NC}"
	@go test -v -race ./internal/domain/... ./internal/services/...

test-integration: ## Run integration tests
	@echo "${GREEN}Running integration tests...${NC}"
	@go test -v -tags=integration ./tests/integration/...

test-e2e: ## Run E2E tests
	@echo "${GREEN}Running E2E tests...${NC}"
	@go test -v -tags=e2e ./tests/e2e/...

test-all: test-unit test-integration test-e2e ## Run all test suites
	@echo "${GREEN}All tests completed!${NC}"

test-clean: ## Clean test artifacts
	@echo "${GREEN}Cleaning test artifacts...${NC}"
	@rm -f $(COVERAGE_FILE) coverage.html
	@echo "${GREEN}Test artifacts cleaned${NC}"
	@go tool cover -func=$(COVERAGE_FILE)

lint: ## Run linters
	@echo "${GREEN}Running linters...${NC}"
	@golangci-lint run --timeout=5m

fmt: ## Format code
	@echo "${GREEN}Formatting code...${NC}"
	@gofmt -s -w .
	@goimports -w .
	@echo "${GREEN}Code formatted${NC}"

vet: ## Run go vet
	@echo "${GREEN}Running go vet...${NC}"
	@go vet ./...

security: ## Run security scanner
	@echo "${GREEN}Running security scanner...${NC}"
	@gosec -quiet ./...

clean: ## Clean build artifacts
	@echo "${YELLOW}Cleaning build artifacts...${NC}"
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE) coverage.html
	@echo "${GREEN}Clean complete${NC}"

deps: ## Download dependencies
	@echo "${GREEN}Downloading dependencies...${NC}"
	@go mod download
	@go mod tidy

deps-upgrade: ## Upgrade dependencies
	@echo "${GREEN}Upgrading dependencies...${NC}"
	@go get -u ./...
	@go mod tidy

migrate: ## Run database migrations
	@echo "${GREEN}Running migrations...${NC}"
	@go run $(MAIN_PATH)/main.go migrate

migrate-up: ## Run migrations up
	@echo "${GREEN}Running migrations up...${NC}"
	@$(HOME)/go/bin/migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down: ## Run migrations down
	@echo "${YELLOW}Rolling back migrations...${NC}"
	@$(HOME)/go/bin/migrate -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-create: ## Create new migration (usage: make migrate-create NAME=create_users_table)
	@echo "${GREEN}Creating migration: $(NAME)${NC}"
	@$(HOME)/go/bin/migrate create -ext sql -dir ./migrations -seq $(NAME)

populate-translations: ## Populate initial translations for existing stations
	@echo "${GREEN}Populating translations...${NC}"
	@go run ./cmd/populate-translations/main.go
	@echo "${GREEN}Translations populated successfully${NC}"

docker-build: ## Build Docker image
	@echo "${GREEN}Building Docker image...${NC}"
	@docker build -t $(BINARY_NAME):latest .

docker-run: ## Run Docker container
	@echo "${GREEN}Running Docker container...${NC}"
	@docker run -p 8080:8080 --env-file .env $(BINARY_NAME):latest

generate-keys: ## Generate RSA keys for JWT
	@echo "${GREEN}Generating RSA key pair for JWT...${NC}"
	@mkdir -p keys
	@openssl genrsa -out keys/jwt-private.pem 2048
	@openssl rsa -in keys/jwt-private.pem -pubout -out keys/jwt-public.pem
	@echo "${GREEN}Keys generated in ./keys/${NC}"

swagger-install: ## Install Swagger CLI tool
	@echo "${GREEN}Installing swag CLI...${NC}"
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "${GREEN}swag CLI installed${NC}"

swagger-generate: ## Generate Swagger documentation
	@echo "${GREEN}Generating Swagger documentation...${NC}"
	@$(HOME)/go/bin/swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
	@echo "${GREEN}Swagger documentation generated in ./docs/${NC}"

swagger-clean: ## Clean generated Swagger files
	@echo "${YELLOW}Cleaning Swagger documentation...${NC}"
	@rm -rf docs/docs.go docs/swagger.json docs/swagger.yaml
	@echo "${GREEN}Swagger files cleaned${NC}"

install-tools: ## Install development tools
	@echo "${GREEN}Installing development tools...${NC}"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/cosmtrek/air@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "${GREEN}Tools installed${NC}"

all: clean deps fmt lint test build ## Run all checks and build

# =============================================================================
# Cloud Run Utilities (GitHub Actions handles deployment)
# =============================================================================

.PHONY: cloudrun-logs cloudrun-describe cloudrun-url

# Cloud Run variables
PROJECT_ID ?= $(shell gcloud config get-value project)
REGION ?= us-central1
SERVICE_NAME ?= radio-backend

cloudrun-logs: ## View Cloud Run logs
	@echo "${GREEN}Fetching Cloud Run logs...${NC}"
	@gcloud run services logs read $(SERVICE_NAME) --region $(REGION) --project $(PROJECT_ID) --limit 100

cloudrun-logs-tail: ## Tail Cloud Run logs in real-time
	@echo "${GREEN}Tailing Cloud Run logs...${NC}"
	@gcloud run services logs tail $(SERVICE_NAME) --region $(REGION) --project $(PROJECT_ID)

cloudrun-describe: ## Describe Cloud Run service
	@gcloud run services describe $(SERVICE_NAME) --region $(REGION) --project $(PROJECT_ID)

cloudrun-url: ## Get Cloud Run service URL
	@gcloud run services describe $(SERVICE_NAME) --region $(REGION) --project $(PROJECT_ID) --format='value(status.url)'

# Local Docker testing (multi-stage)
docker-build-prod: ## Build production Docker image locally
	@echo "${GREEN}Building production Docker image...${NC}"
	@docker build -t $(BINARY_NAME):latest -t $(BINARY_NAME):$(shell git rev-parse --short HEAD) .
	@echo "${GREEN}Image size:${NC}"
	@docker images $(BINARY_NAME):latest --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"

docker-test: docker-build-prod ## Test Docker image locally
	@echo "${GREEN}Testing Docker image...${NC}"
	@docker run --rm -p 8080:8080 \
		-e PORT=8080 \
		-e ENV=development \
		-e DB_HOST=host.docker.internal \
		$(BINARY_NAME):latest

docker-inspect: ## Inspect Docker image layers
	@docker history $(BINARY_NAME):latest

docker-scan: docker-build-prod ## Scan Docker image for vulnerabilities
	@echo "${GREEN}Scanning image for vulnerabilities...${NC}"
	@docker scan $(BINARY_NAME):latest || echo "${YELLOW}Docker scan not available${NC}"

# =============================================================================
# Docker Compose (Full Stack)
# =============================================================================

.PHONY: docker-up docker-down docker-logs docker-restart docker-clean docker-migrate docker-status

docker-up: ## Start all services with docker-compose (app, postgres, redis)
	@echo "${GREEN}Starting services with docker-compose...${NC}"
	@./docker.sh up

docker-down: ## Stop all docker-compose services
	@./docker.sh down

docker-logs: ## View application logs
	@./docker.sh logs

docker-logs-all: ## View all services logs
	@./docker.sh logs-all

docker-restart: ## Restart the application container
	@./docker.sh restart

docker-clean: ## Remove all containers and volumes
	@./docker.sh clean

docker-migrate: ## Run database migrations
	@./docker.sh migrate

docker-status: ## Show status of all services
	@./docker.sh status

docker-shell: ## Open shell in app container
	@./docker.sh shell

docker-db: ## Open PostgreSQL shell
	@./docker.sh db-shell

docker-redis: ## Open Redis CLI
	@./docker.sh redis-cli

docker-health: ## Run health checks on all services
	@./docker.sh test

# =============================================================================
# Terraform Infrastructure as Code
# =============================================================================

.PHONY: tf-init tf-plan tf-apply tf-destroy tf-output tf-validate tf-fmt tf-state tf-init-prod tf-init-staging

tf-init: ## Initialize Terraform for production
	@./scripts/terraform-helpers.sh init production

tf-init-staging: ## Initialize Terraform for staging
	@./scripts/terraform-helpers.sh init staging

tf-plan: ## Show Terraform execution plan (production)
	@./scripts/terraform-helpers.sh plan production

tf-plan-staging: ## Show Terraform execution plan (staging)
	@./scripts/terraform-helpers.sh plan staging

tf-apply: ## Apply Terraform changes (production)
	@./scripts/terraform-helpers.sh apply production

tf-apply-staging: ## Apply Terraform changes (staging)
	@./scripts/terraform-helpers.sh apply staging

tf-destroy: ## Destroy Terraform infrastructure (production)
	@./scripts/terraform-helpers.sh destroy production

tf-destroy-staging: ## Destroy Terraform infrastructure (staging)
	@./scripts/terraform-helpers.sh destroy staging

tf-output: ## Show Terraform outputs (production)
	@./scripts/terraform-helpers.sh output production

tf-output-staging: ## Show Terraform outputs (staging)
	@./scripts/terraform-helpers.sh output staging

tf-validate: ## Validate Terraform configuration
	@./scripts/terraform-helpers.sh validate production

tf-fmt: ## Format Terraform files
	@./scripts/terraform-helpers.sh fmt production

tf-state: ## Show Terraform state (production)
	@./scripts/terraform-helpers.sh state production

tf-state-staging: ## Show Terraform state (staging)
	@./scripts/terraform-helpers.sh state staging

tf-clean: ## Clean infrastructure and start fresh
	@./scripts/clean-infrastructure.sh

tf-setup-backend: ## Create GCS bucket for Terraform state
	@echo "${GREEN}Creating GCS bucket for Terraform state...${NC}"
	@gsutil mb -p radio-485022 -l us-central1 gs://radio-485022-terraform-state || echo "Bucket already exists"
	@gsutil versioning set on gs://radio-485022-terraform-state
	@echo "${GREEN}✓ Backend configured${NC}"

tf-github-secrets: ## Show GitHub Actions secrets to configure
	@echo "${YELLOW}Configure these secrets in GitHub:${NC}"
	@echo ""
	@echo "Go to: https://github.com/kedeinroga/radio-backend/settings/secrets/actions"
	@echo ""
	@cd terraform && terraform output -raw workload_identity_provider > /tmp/wip.txt 2>/dev/null && echo "GCP_WORKLOAD_IDENTITY_PROVIDER:" && cat /tmp/wip.txt && echo "" || echo "Run 'make tf-apply' first"
	@cd terraform && terraform output -raw github_actions_service_account_email > /tmp/gasa.txt 2>/dev/null && echo "GCP_SERVICE_ACCOUNT:" && cat /tmp/gasa.txt && echo "" || echo "Run 'make tf-apply' first"
	@rm -f /tmp/wip.txt /tmp/gasa.txt
