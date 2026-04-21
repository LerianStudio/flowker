# Flowker Makefile

# Component-specific variables
SERVICE_NAME := flowker
BIN_DIR := ./.bin
ARTIFACTS_DIR := ./artifacts
DEFAULT_SERVER_PORT ?= 4021

# Ensure artifacts directory exists
$(shell mkdir -p $(ARTIFACTS_DIR))

# Define the root directory of the project
ROOT_DIR := $(shell pwd)
DOCKER_CMD := $(shell \
	if [ "$(shell printf '%s\n' "$(DOCKER_MIN_VERSION)" "$(DOCKER_VERSION)" | sort -V | head -n1)" = "$(DOCKER_MIN_VERSION)" ]; then \
		echo "docker compose"; \
	else \
		echo "docker-compose"; \
	fi \
)

# Include shared color definitions and utility functions
include $(ROOT_DIR)/pkg/shell/makefile_colors.mk
include $(ROOT_DIR)/pkg/shell/makefile_utils.mk

#-------------------------------------------------------
# Core Commands
#-------------------------------------------------------

.PHONY: help
help:
	@echo ""
	@echo "$(BOLD)Flowker Service Commands$(NC)"
	@echo ""
	@echo "$(BOLD)Core Commands:$(NC)"
	@echo "  make help                        - Display this help message"
	@echo "  make build                       - Build the binary to .bin/flowker"
	@echo "  make test                        - Run all tests (unit, integration, e2e)"
	@echo "  make test-unit                   - Run unit tests only"
	@echo "  make test-integration            - Run integration tests (requires Docker for testcontainers)"
	@echo "  make test-e2e                    - Run E2E tests (requires Docker for testcontainers)"
	@echo "  make clean                       - Clean build artifacts"
	@echo "  make run                         - Run the application with .env config"
	@echo "  make cover                       - Run tests with coverage summary"
	@echo "  make cover-html                  - Generate HTML test coverage report"
	@echo ""
	@echo "$(BOLD)Code Quality Commands:$(NC)"
	@echo "  make lint                        - Run linting tools"
	@echo "  make format                      - Format code with go fmt"
	@echo "  make generate                    - Generate code (mocks, etc.)"
	@echo "  make tidy                        - Update and clean dependencies"
	@echo ""
	@echo "$(BOLD)Docker Commands:$(NC)"
	@echo "  make up                          - Start services with Docker Compose"
	@echo "  make down                        - Stop services with Docker Compose"
	@echo "  make start                       - Start existing containers"
	@echo "  make stop                        - Stop running containers"
	@echo "  make restart                     - Restart all containers"
	@echo "  make logs                        - Show logs for all services"
	@echo "  make logs-api                    - Show logs for flowker service"
	@echo "  make ps                          - List container status"
	@echo "  make rebuild-up                  - Rebuild and restart services during development"
	@echo ""
	@echo "$(BOLD)Flowker-Specific Commands:$(NC)"
	@echo "  make generate-docs               - Generate Swagger API documentation"
	@echo "  make verify-api-docs             - Verify API documentation coverage"
	@echo "  make validate-api-docs           - Validate API documentation"
	@echo "  make sync-postman                - Sync Postman collection with OpenAPI documentation"
	@echo ""
	@echo "$(BOLD)Developer Helper Commands:$(NC)"
	@echo "  make dev-setup                   - Set up development environment"
	@echo "  make dev                         - Start local stack (single Mongo RS + Flowker app)"
	@echo "  make clear                       - Stop stack and remove dev containers/volumes/networks (keeps images)"
	@echo ""

#-------------------------------------------------------
# Git Hook Commands
#-------------------------------------------------------

.PHONY: setup-git-hooks
setup-git-hooks:
	$(call title1,"Installing and configuring git hooks")
	@sh ./scripts/setup-git-hooks.sh
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Git hooks installed successfully$(GREEN) ✔️$(NC)"


.PHONY: check-hooks
check-hooks:
	$(call title1,"Verifying git hooks installation status")
	@err=0; \
	for hook_dir in .githooks/*; do \
		hook_name=$$(basename $$hook_dir); \
		if [ ! -f ".git/hooks/$$hook_name" ]; then \
			echo "$(RED)Git hook $$hook_name is not installed$(NC)"; \
			err=1; \
		else \
			echo "$(GREEN)Git hook $$hook_name is installed$(NC)"; \
		fi; \
	done; \
	if [ $$err -eq 0 ]; then \
		echo "$(GREEN)$(BOLD)[ok]$(NC) All git hooks are properly installed$(GREEN) ✔️$(NC)"; \
	else \
		echo "$(RED)$(BOLD)[error]$(NC) Some git hooks are missing. Run 'make setup-git-hooks' to fix.$(RED) ❌$(NC)"; \
		exit 1; \
	fi

.PHONY: check-envs
check-envs:
	$(call title1,"Checking if github hooks are installed and secret env files are not exposed")
	@sh ./scripts/check-envs.sh
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Environment check completed$(GREEN) ✔️$(NC)"

#-------------------------------------------------------
# Setup Commands
#-------------------------------------------------------

.PHONY: set-env
set-env:
	$(call title1,"Setting up environment files")

	@if [ -f ".env.example" ] && [ ! -f ".env" ]; then \
		echo "$(CYAN)Creating .env in plugin from .env.example$(NC)"; \
		cp ".env.example" ".env"; \
	elif [ ! -f ".env.example" ]; then \
		echo "$(YELLOW)Warning: No .env.example found in plugin$(NC)"; \
	else \
		echo "$(GREEN).env already exists in plugin$(NC)"; \
	fi

	@echo "$(GREEN)$(BOLD)[ok]$(NC) Environment files set up successfully$(GREEN) ✔️$(NC)"

#-------------------------------------------------------
# Build Commands
#-------------------------------------------------------

.PHONY: build
build:
	$(call title1,"Building component")
	@mkdir -p $(BIN_DIR)
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BIN_DIR)/$(SERVICE_NAME) ./cmd/app
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Build completed successfully - binary at $(BIN_DIR)/$(SERVICE_NAME)$(GREEN) ✔️$(NC)"

#-------------------------------------------------------
# Test Commands
#-------------------------------------------------------

.PHONY: test
test:
	$(call title1,"Running all tests (unit, integration, e2e)")
	@go test -tags=unit,integration,e2e -v -timeout 5m ./...
	@echo "$(GREEN)$(BOLD)[ok]$(NC) All tests completed successfully$(GREEN) ✔️$(NC)"

.PHONY: test-unit
test-unit:
	$(call title1,"Running unit tests")
	@go test -v ./... -count=1
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Unit tests completed successfully$(GREEN) ✔️$(NC)"

.PHONY: test-integration
test-integration:
	$(call title1,"Running integration tests")
	@echo "$(CYAN)Note: Tests use testcontainers (auto-starts MongoDB)$(NC)"
	@echo "$(CYAN)      Set DISABLE_TESTCONTAINERS=true to use external server instead$(NC)"
	@go test -tags=integration -v ./tests/integration/...
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Integration tests completed successfully$(GREEN) ✔️$(NC)"

.PHONY: test-e2e
test-e2e:
	$(call title1,"Running E2E tests")
	@echo "$(CYAN)Note: Tests use testcontainers (auto-starts MongoDB + Flowker)$(NC)"
	@echo "$(CYAN)      Set E2E_BASE_URL to use an external server instead$(NC)"
	@go test -tags=e2e -v -timeout 5m ./tests/e2e/...
	@echo "$(GREEN)$(BOLD)[ok]$(NC) E2E tests completed successfully$(GREEN) ✔️$(NC)"

.PHONY: cover
cover:
	$(call title1,"Running tests with coverage")
	@PACKAGES=$$(go list -tags=integration,unit ./... | grep -v -f ./scripts/coverage_ignore.txt); \
	go test -tags=integration,unit -coverprofile=$(ARTIFACTS_DIR)/coverage.out $$PACKAGES
	@echo ""
	@echo "$(CYAN)Coverage Summary:$(NC)"
	@echo "$(CYAN)----------------------------------------$(NC)"
	@go tool cover -func=$(ARTIFACTS_DIR)/coverage.out | grep total | awk '{print "Total coverage: " $$3}'
	@echo "$(CYAN)----------------------------------------$(NC)"
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Coverage report generated at $(ARTIFACTS_DIR)/coverage.out$(GREEN) ✔️$(NC)"

.PHONY: cover-html
cover-html:
	$(call title1,"Generating HTML test coverage report")
	@PACKAGES=$$(go list -tags=integration,unit ./... | grep -v -f ./scripts/coverage_ignore.txt); \
	go test -tags=integration,unit -coverprofile=$(ARTIFACTS_DIR)/coverage.out $$PACKAGES
	@go tool cover -html=$(ARTIFACTS_DIR)/coverage.out -o $(ARTIFACTS_DIR)/coverage.html
	@echo "$(GREEN)Coverage report generated at $(ARTIFACTS_DIR)/coverage.html$(NC)"
	@echo ""
	@echo "$(CYAN)Coverage Summary:$(NC)"
	@echo "$(CYAN)----------------------------------------$(NC)"
	@go tool cover -func=$(ARTIFACTS_DIR)/coverage.out | grep total | awk '{print "Total coverage: " $$3}'
	@echo "$(CYAN)----------------------------------------$(NC)"
	@echo "$(YELLOW)Open $(ARTIFACTS_DIR)/coverage.html in your browser to view detailed coverage report$(NC)"

#-------------------------------------------------------
# Test Coverage Commands
#-------------------------------------------------------

.PHONY: check-tests
check-tests:
	$(call title1,"Verifying test coverage")
	@if find . -name "*.go" -type f | grep -q .; then \
		echo "$(CYAN)Running test coverage check...$(NC)"; \
		go test -coverprofile=coverage.tmp ./... > /dev/null 2>&1; \
		if [ -f coverage.tmp ]; then \
			coverage=$$(go tool cover -func=coverage.tmp | grep total | awk '{print $$3}'); \
			echo "$(CYAN)Test coverage: $(GREEN)$$coverage$(NC)"; \
			rm coverage.tmp; \
		else \
			echo "$(YELLOW)No coverage data generated$(NC)"; \
		fi; \
	else \
		echo "$(YELLOW)No Go files found, skipping test coverage check$(NC)"; \
	fi

#-------------------------------------------------------
# Code Quality Commands
#-------------------------------------------------------

.PHONY: lint
lint:
	$(call title1,"Running linters")
	@if find . -name "*.go" -type f | grep -q .; then \
		if ! command -v golangci-lint >/dev/null 2>&1; then \
			echo "$(YELLOW)Installing golangci-lint v2...$(NC)"; \
			go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2; \
		fi; \
		golangci-lint run --fix ./... --verbose; \
		echo "$(GREEN)$(BOLD)[ok]$(NC) Linting completed successfully$(GREEN) ✔️$(NC)"; \
	else \
		echo "$(YELLOW)No Go files found, skipping linting$(NC)"; \
	fi

.PHONY: format
format:
	$(call title1,"Formatting code")
	@go fmt ./...
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Formatting completed successfully$(GREEN) ✔️$(NC)"

.PHONY: generate
generate:
	$(call title1,"Generating code (mocks, etc.)")
	@if ! command -v mockgen >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing mockgen...$(NC)"; \
		go install go.uber.org/mock/mockgen@v0.5.2; \
	fi
	@go generate ./...
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Code generation completed successfully$(GREEN) ✔️$(NC)"

.PHONY: tidy
tidy:
	$(call title1,"Update and Cleaning dependencies")
	@go get -u ./...
	@go mod tidy
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Dependencies updated and cleaned successfully$(GREEN) ✔️$(NC)"

#-------------------------------------------------------
# Security Commands
#-------------------------------------------------------

.PHONY: sec
sec:
	$(call title1,"Running security checks using gosec")
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing gosec...$(NC)"; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@if find . -name "*.go" -type f | grep -q .; then \
		echo "$(CYAN)Running security checks...$(NC)"; \
		gosec -quiet ./...; \
		echo "$(GREEN)$(BOLD)[ok]$(NC) Security checks completed$(GREEN) ✔️$(NC)"; \
	else \
		echo "$(YELLOW)No Go files found, skipping security checks$(NC)"; \
	fi

#-------------------------------------------------------
# Clean Commands
#-------------------------------------------------------

.PHONY: clean
clean:
	$(call title1,"Cleaning build artifacts")
	@rm -rf $(BIN_DIR)/* $(ARTIFACTS_DIR)/*
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Artifacts cleaned successfully$(GREEN) ✔️$(NC)"

#-------------------------------------------------------
# Docker Commands
#-------------------------------------------------------

.PHONY: run
run:
	$(call title1,"Running the application with .env config")
	@go run cmd/app/main.go .env
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Application started successfully$(GREEN) ✔️$(NC)"

.PHONY: build-docker
build-docker:
	$(call title1,"Building Docker images")
	@$(DOCKER_CMD) -f docker-compose.yml build $(c)
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Docker images built successfully$(GREEN) ✔️$(NC)"

.PHONY: up
up:
	$(call title1,"Starting all services in detached mode")
	@$(DOCKER_CMD) -f docker-compose.yml up $(c) -d
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Services started successfully$(GREEN) ✔️$(NC)"

.PHONY: start
start:
	$(call title1,"Starting existing containers")
	@$(DOCKER_CMD) -f docker-compose.yml start $(c)
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Containers started successfully$(GREEN) ✔️$(NC)"

.PHONY: down
down:
	$(call title1,"Stopping and removing containers|networks|volumes")
	@if [ -f "docker-compose.yml" ]; then \
		$(DOCKER_CMD) -f docker-compose.yml down $(c); \
	else \
		echo "$(YELLOW)No docker-compose.yml file found. Skipping down command.$(NC)"; \
	fi
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Services stopped successfully$(GREEN) ✔️$(NC)"

.PHONY: stop
stop:
	$(call title1,"Stopping running containers")
	@$(DOCKER_CMD) -f docker-compose.yml stop $(c)
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Containers stopped successfully$(GREEN) ✔️$(NC)"

.PHONY: restart
restart:
	$(call title1,"Restarting all services")
	@make stop && make up
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Services restarted successfully$(GREEN) ✔️$(NC)"

.PHONY: rebuild-up
rebuild-up:
	$(call title1,"Rebuilding and restarting services")
	@$(DOCKER_CMD) -f docker-compose.yml down
	@$(DOCKER_CMD) -f docker-compose.yml build
	@$(DOCKER_CMD) -f docker-compose.yml up -d
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Services rebuilt and restarted successfully$(GREEN) ✔️$(NC)"

.PHONY: logs
logs:
	$(call title1,"Showing logs for all services")
	@if [ -f "docker-compose.yml" ]; then \
		echo "$(CYAN)Logs for component: $(BOLD)flowker$(NC)"; \
		docker compose -f docker-compose.yml logs --tail=100 -f $(c) 2>/dev/null || docker-compose -f docker-compose.yml logs --tail=100 -f $(c); \
	else \
		echo "$(YELLOW)No docker-compose.yml file found. Skipping logs command.$(NC)"; \
	fi

.PHONY: logs-api
logs-api:
	$(call title1,"Showing logs for flowker service")
	@$(DOCKER_CMD) -f docker-compose.yml logs --tail=100 -f flowker

.PHONY: ps
ps:
	$(call title1,"Listing container status")
	@$(DOCKER_CMD) -f docker-compose.yml ps

#-------------------------------------------------------
# Docs Commands
#-------------------------------------------------------

.PHONY: generate-docs-all
generate-docs-all:
	$(call title1,"Generating Swagger documentation for all services")
	$(call check_command,swag,"go install github.com/swaggo/swag/cmd/swag@latest")
	@echo "$(CYAN)Verifying API documentation coverage...$(NC)"
	@sh ./scripts/verify-api-docs.sh 2>/dev/null || echo "$(YELLOW)Warning: Some API endpoints may not be properly documented. Continuing with documentation generation...$(NC)"
	@echo "$(CYAN)Generating documentation for plugin component...$(NC)"
	$(MAKE) generate-docs 2>&1 | grep -v "warning: "
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Swagger documentation generated successfully$(GREEN) ✔️$(NC)"
	@echo "$(CYAN)Syncing Postman collection with the generated OpenAPI documentation...$(NC)"
	@sh ./scripts/sync-postman.sh
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Postman collection synced successfully with OpenAPI documentation$(GREEN) ✔️$(NC)"

.PHONY: sync-postman
sync-postman:
	$(call title1,"Syncing Postman collection with OpenAPI documentation")
	$(call check_command,jq,"brew install jq")
	@sh ./scripts/sync-postman.sh
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Postman collection synced successfully with OpenAPI documentation$(GREEN) ✔️$(NC)"

.PHONY: verify-api-docs
verify-api-docs:
	$(call title1,"Verifying API documentation coverage")
	@if [ -f "./scripts/package.json" ]; then \
		echo "$(CYAN)Installing npm dependencies...$(NC)"; \
		cd ./scripts && npm install; \
	fi
	@sh ./scripts/verify-api-docs.sh
	@echo "$(GREEN)$(BOLD)[ok]$(NC) API documentation verification completed$(GREEN) ✔️$(NC)"

.PHONY: generate-docs
generate-docs:
	$(call title1,"Generating Swagger API documentation")
	@if ! command -v swag >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing swag...$(NC)"; \
		GOBIN=$(ROOT_DIR)/.bin go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	@SWAG=$$(command -v swag 2>/dev/null || echo "$(ROOT_DIR)/.bin/swag"); \
	cd $(ROOT_DIR) && $$SWAG init -g cmd/app/main.go -o api --parseDependency --parseInternal
	@docker run --rm -v ./:/plugin --user $(shell id -u):$(shell id -g) openapitools/openapi-generator-cli:v5.1.1 generate -i ./plugin/api/swagger.json -g openapi-yaml -o ./plugin/api
	@mv ./api/openapi/openapi.yaml ./api/openapi.yaml
	@rm -rf ./api/README.md ./api/.openapi-generator* ./api/openapi
	@if [ -f "$(ROOT_DIR)/scripts/package.json" ]; then \
		echo "$(YELLOW)Installing npm dependencies for validation...$(NC)"; \
		cd $(ROOT_DIR)/scripts && npm install > /dev/null; \
	fi
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Swagger API documentation generated successfully$(GREEN) ✔️$(NC)"

.PHONY: validate-api-docs
validate-api-docs: generate-docs
	$(call title1,"Validating API documentation")
	@if [ -f "scripts/validate-api-docs.js" ] && [ -f "$(ROOT_DIR)/scripts/package.json" ]; then \
		echo "$(YELLOW)Validating API documentation structure...$(NC)"; \
		cd $(ROOT_DIR)/scripts && node $(ROOT_DIR)/scripts/validate-api-docs.js; \
		echo "$(YELLOW)Validating API implementations...$(NC)"; \
		cd $(ROOT_DIR)/scripts && node $(ROOT_DIR)/scripts/validate-api-implementations.js; \
		echo "$(GREEN)$(BOLD)[ok]$(NC) API documentation validation completed$(GREEN) ✔️$(NC)"; \
	else \
		echo "$(YELLOW)Validation scripts not found. Skipping validation.$(NC)"; \
	fi

#-------------------------------------------------------
# Developer Helper Commands
#-------------------------------------------------------

.PHONY: dev-setup
dev-setup:
	$(call title1,"Setting up development environment")
	@echo "$(CYAN)Installing development tools...$(NC)"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing golangci-lint v2...$(NC)"; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2; \
	fi
	@if ! command -v swag >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing swag...$(NC)"; \
		go install github.com/swaggo/swag/cmd/swag@latest; \
	fi
	@if ! command -v mockgen >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing mockgen...$(NC)"; \
		go install go.uber.org/mock/mockgen@v0.5.2; \
	fi
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing gosec...$(NC)"; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	@echo "$(CYAN)Setting up environment...$(NC)"
	@if [ -f .env.example ] && [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "$(GREEN)Created .env file from template$(NC)"; \
	fi
	@make tidy
	@make check-tests
	@make sec
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Development environment set up successfully$(GREEN) ✔️$(NC)"
	@echo "$(CYAN)You're ready to start developing! Here are some useful commands:$(NC)"
	@echo "  make build         - Build the component"
	@echo "  make test          - Run tests"
	@echo "  make up            - Start services"
	@echo "  make rebuild-up    - Rebuild and restart services during development"

.PHONY: dev
dev:
	$(call title1,"Starting local dev stack (Mongo RS + Audit PostgreSQL + Flowker)")
	$(call check_cmd,$(DOCKER_CMD),docker/docker-compose)
	@echo "$(CYAN)Ensuring .env exists (run 'make set-env' if needed)...$(NC)"
	@if [ ! -f ".env" ] && [ -f ".env.example" ]; then cp .env.example .env; fi
	@echo "$(CYAN)Generating Swagger docs...$(NC)"
	@$(MAKE) generate-docs
	@$(DOCKER_CMD) -f docker-compose.dev.yml up -d flowker-mongodb flowker-mongodb-init flowker-audit-postgres
	@echo "$(CYAN)Waiting for Audit PostgreSQL to be ready...$(NC)"
	@until $(DOCKER_CMD) -f docker-compose.dev.yml exec -T flowker-audit-postgres pg_isready -U flowker_audit > /dev/null 2>&1; do sleep 1; done
	@echo "$(CYAN)Starting Go app locally...$(NC)"
	@env ENV_NAME=development SERVER_ADDRESS=":$(DEFAULT_SERVER_PORT)" MONGO_URI="mongodb://localhost:27017" MONGO_DB_NAME="flowker" \
		AUDIT_DB_HOST=localhost AUDIT_DB_PORT=5432 AUDIT_DB_USER=flowker_audit AUDIT_DB_PASSWORD=flowker_audit AUDIT_DB_NAME=flowker_audit AUDIT_DB_SSL_MODE=disable AUDIT_MIGRATIONS_PATH=./migrations \
		LOG_LEVEL=debug ENABLE_TELEMETRY=false OTEL_LIBRARY_NAME="github.com/LerianStudio/flowker" OTEL_RESOURCE_SERVICE_NAME="flowker" \
		SWAGGER_TITLE="Flowker API" SWAGGER_DESCRIPTION="Flowker local dev" SWAGGER_VERSION="v1" \
		SWAGGER_HOST="localhost:$(DEFAULT_SERVER_PORT)" SWAGGER_BASE_PATH="/" SWAGGER_LEFT_DELIM="{{" SWAGGER_RIGHT_DELIM="}}" SWAGGER_SCHEMES="http" \
		API_KEY_ENABLED=false PLUGIN_AUTH_ENABLED=false CORS_ALLOWED_ORIGINS="*" SSRF_ALLOW_PRIVATE=true \
		go run ./cmd/app

.PHONY: clear
clear:
	$(call title1,"Stopping and clearing local dev stack (containers, volumes, networks)")
	$(call check_cmd,$(DOCKER_CMD),docker/docker-compose)
	@$(DOCKER_CMD) -f docker-compose.dev.yml down -v --remove-orphans
	@$(DOCKER_CMD) -f docker-compose.yml down -v --remove-orphans || true
	@docker network ls --filter "name=flowker" -q | xargs -r docker network rm 2>/dev/null || true
	@echo "$(GREEN)$(BOLD)[ok]$(NC) Local stack cleared (images kept)$(GREEN) ✔️$(NC)"
