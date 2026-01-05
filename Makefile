.PHONY: help build run test test-integration lint fmt clean migrate migrate-up migrate-down security docker-build docker-run

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
