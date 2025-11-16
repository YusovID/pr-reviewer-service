# ====================================================================================
# CONFIGURATION
# ====================================================================================

APP_NAME := pr-reviewer-service
COMPOSE_FILE := compose.yml

-include .env

export

CONFIG_PATH ?= ./config/local.yml
MIGRATIONS_PATH ?= ${MIGRATIONS_PATH}
MIGRATIONS_TABLE ?= ${MIGRATIONS_TABLE}

DATABASE_URL := postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable

# ====================================================================================
# COMMANDS & TOOLS
# ====================================================================================

RM := rm -f
SLEEP := sleep

COMPOSE := docker compose -f $(COMPOSE_FILE)

GO_OAPI_CODEGEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/cmd/golangci-lint
MIGRATE_APP := go run ./cmd/migrator
MIGRATE_CLI := go run github.com/golang-migrate/migrate/v4/cmd/migrate

# ====================================================================================
# SETUP
# ====================================================================================

.DEFAULT_GOAL := help

.PHONY: all help build up start stop restart down nuke logs ps clean generate fmt lint test test-integration test-cover test-load tools migrate-create migrate-up migrate-down

# ====================================================================================
# GENERAL COMMANDS
# ====================================================================================

all: fmt lint test ## Запустить форматирование, линтер и тесты

help: ## Показать этот список команд и их описания
	@echo "Usage: make <target>"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "; printf "  \033[36m%-20s\033[0m %s\n", "Target", "Description"} /^[a-zA-Z_-]+:.*?## / { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST) | sort

# ====================================================================================
# DOCKER COMPOSE MANAGEMENT
# ====================================================================================

build: ## Собрать или пересобрать образы сервисов
	@echo "Building service images..."
	@$(COMPOSE) build

up: down build ## Собрать образы и запустить сервисы в фоне
	@echo "Starting services..."
	@$(COMPOSE) up -d

start: ## Запустить ранее остановленные контейнеры
	@echo "Starting existing containers..."
	@$(COMPOSE) start

stop: ## Остановить запущенные сервисы
	@echo "Stopping services..."
	@$(COMPOSE) stop

restart: stop start ## Перезапустить сервисы

down: ## Остановить и удалить контейнеры/сети (тома сохраняются)
	@echo "Tearing down services..."
	@$(COMPOSE) down --remove-orphans

nuke: ## ВНИМАНИЕ: Полностью удалить всё (контейнеры, сети, ТОМА)
	@echo "Nuking the entire environment (containers, networks, VOLUMES)..."
	@$(COMPOSE) down -v --remove-orphans

logs: ## Показать логи всех сервисов в реальном времени
	@$(COMPOSE) logs -f

ps: ## Показать статус запущенных контейнеров
	@$(COMPOSE) ps

# ====================================================================================
# GO BUILD & TEST
# ====================================================================================

generate: tools ## Сгенерировать Go код из OpenAPI спецификации
	@echo "Generating Go code from OpenAPI spec..."
	@$(GO_OAPI_CODEGEN) --config=oapi-codegen.yml pkg/api/openapi.yml

fmt: ## Отформатировать весь Go код
	@echo "Formatting Go files..."
	@gofmt -w .

lint: tools ## Запустить линтер для проверки качества кода
	@echo "Running linter..."
	@$(GOLANGCI_LINT) run ./...

test: ## Запустить unit-тесты (без интеграционных)
	@echo "Running fast tests..."
	@go test -v -race -short ./...

test-integration: ## Запустить интеграционные тесты (требует Docker)
	@echo "Running integration tests..."
	@go test -v -race -tags=integration ./...

test-load: nuke up ## ВНИМАНИЕ: Полностью удаляет БД перед тестом!
	@echo "Waiting for services to become healthy..."
	@$(SLEEP) 5
	@echo "Running load tests..."
	@k6 run loadtests/main.js

test-cover: ## Запустить ВСЕ тесты и сгенерировать HTML-отчет о покрытии
	@echo "Running all tests with coverage..."
	@go test -race -short -coverprofile=unit.cover ./...
	@go test -race -tags=integration -coverprofile=integration.cover ./...

	@echo "Merging coverage profiles..."
	@echo "mode: set" > coverage.out
	@cat unit.cover integration.cover | grep -v "^mode:" >> coverage.out

	@echo "Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html

	@echo "Cleaning up intermediate files..."
	@$(RM) unit.cover integration.cover coverage.out

	@echo "Coverage report successfully generated: open coverage.html"

clean: ## Очистить все артефакты сборки и тестирования
	@echo "Cleaning up build and test artifacts..."
	@$(RM) coverage.html coverage.out unit.cover integration.cover
	@$(RM) *.test *.exe

tools: ## Установить/обновить зависимости для утилит
	@echo "Syncing tools dependencies..."
	@go mod -C tools tidy

# ====================================================================================
# DATABASE MIGRATIONS
# ====================================================================================

migrate-create: ## Создать новый файл миграции (интерактивно)
	@read -p "Enter migration name (e.g., add_pr_status_index): " name; \
	$(MIGRATE_CLI) create -ext sql -dir migrations -seq $$name

migrate-up: ## Применить все 'up' миграции
	@echo "Applying database migrations..."
	@$(MIGRATE_APP)

migrate-down: ## Откатить последнюю 'down' миграцию
	@echo "Reverting last database migration..."
	@$(MIGRATE_APP) down