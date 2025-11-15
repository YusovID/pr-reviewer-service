# ====================================================================================
# VARIABLES
# ====================================================================================

# Имя приложения и compose-файл
APP_NAME := pr-reviewer-service
COMPOSE_FILE := compose.yml

# Загружаем переменные из .env файла
-include .env
export

# Собираем DSN для миграций из переменных .env.
DATABASE_URL := postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable

# Команды для инструментов, управляемых через go modules. Гарантирует одинаковые версии для всех.
GO_OAPI_CODEGEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/cmd/golangci-lint
MIGRATE := go run github.com/golang-migrate/migrate/v4/cmd/migrate

# Сокращение для docker-compose команд
COMPOSE := docker compose -f $(COMPOSE_FILE)

# ====================================================================================
# SETUP
# ====================================================================================

# Команда по умолчанию, если `make` запущен без цели.
.DEFAULT_GOAL := help

# .PHONY указывает, что цели не являются файлами.
.PHONY: all help build up start stop restart down nuke logs ps clean generate fmt lint test test-integration test-cover tools migrate-create migrate-up migrate-down

# ====================================================================================
# GENERAL COMMANDS
# ====================================================================================

all: fmt lint test ## Запустить форматирование, линтер и тесты

help: ## Показать этот список команд
	@echo "Usage: make <target>"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "; printf "  \033[36m%-20s\033[0m %s\n", "Target", "Description"} /^[a-zA-Z_-]+:.*?## / { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST) | sort -k 2

# ====================================================================================
# DOCKER COMPOSE MANAGEMENT
# ====================================================================================

build: ## Собрать или пересобрать образы сервисов
	@echo "🛠️  Building service images..."
	@$(COMPOSE) build

up: down build ## Собрать образы и запустить сервисы. Основная команда для старта/обновления.
	@echo "🚀  Starting services..."
	@$(COMPOSE) up -d

start: ## Запустить ранее остановленные контейнеры (быстро, без сборки)
	@echo "▶️  Starting existing containers..."
	@$(COMPOSE) start

stop: ## Остановить запущенные сервисы (сохраняет их состояние)
	@echo "🛑  Stopping services..."
	@$(COMPOSE) stop

restart: ## Перезапустить сервисы (быстрый способ: stop + start)
	@echo "🔄  Restarting services..."
	@$(MAKE) stop
	@$(MAKE) start

down: ## Остановить и удалить контейнеры/сети (сохраняет тома с данными)
	@echo "🗑️  Tearing down services (volumes are preserved)..."
	@$(COMPOSE) down --remove-orphans

nuke: ## ВНИМАНИЕ: Полностью удалить всё (контейнеры, сети, ТОМА С ДАННЫМИ)
	@echo "💥  Nuking the entire environment (containers, networks, VOLUMES)..."
	@$(COMPOSE) down -v --remove-orphans

logs: ## Показать логи всех сервисов в реальном времени
	@$(COMPOSE) logs -f

ps: ## Показать статус запущенных контейнеров
	@$(COMPOSE) ps

# ====================================================================================
# GO BUILD & TEST
# ====================================================================================

generate: tools ## Сгенерировать Go код из OpenAPI спецификации
	@echo "📦  Generating Go code from OpenAPI spec..."
	@$(GO_OAPI_CODEGEN) --config=oapi-codegen.yml pkg/api/openapi.yml

fmt: ## Отформатировать весь Go код
	@echo "🎨  Formatting Go files..."
	@gofmt -w .

lint: tools ## Запустить линтер для проверки качества кода
	@echo "🔍  Running linter..."
	@$(GOLANGCI_LINT) run ./...

test: ## Запустить unit-тесты (без интеграционных)
	@echo "🧪  Running fast tests..."
	@go test -v -race -short ./...

test-integration: ## Запустить интеграционные тесты (требует Docker)
	@echo "🌐  Running integration tests..."
	@go test -v -race -tags=integration ./...

test-load: nuke up ## ВНИМАНИЕ: Полностью удаляет БД перед тестом!
	@echo "⏳  Waiting for services to become healthy..."
	@sleep 5 # Небольшая задержка для стабилизации сервисов
	@echo "📈  Running load tests..."
	@k6 run loadtests/main.js

test-cover: ## Запустить ВСЕ тесты с покрытием и сгенерировать HTML-отчет
	@echo "📊  Running all tests with coverage..."
	@echo "mode: set" > coverage.out
	@go test -race -short -coverprofile=unit.cover ./...
	@go test -race -tags=integration -coverprofile=integration.cover ./...
	@grep -h -v "^mode:" *.cover >> coverage.out
	@rm -f *.cover
	@go tool cover -html=coverage.out

clean: ## Очистить артефакты сборки и тестирования
	@echo "🧹  Cleaning up..."
	@rm -f coverage.out *.cover

tools: ## Установить/обновить зависимости для утилит
	@echo "🛠️  Syncing tools dependencies..."
	@go mod -C tools tidy

# ====================================================================================
# DATABASE MIGRATIONS
# ====================================================================================

migrate-create: ## Создать новый файл миграции (интерактивно)
	@read -p "Enter migration name (e.g., add_pr_status_index): " name; \
	$(MIGRATE) create -ext sql -dir migrations -seq $$name

migrate-up: ## Применить все 'up' миграции (требует запущенного postgres)
	@echo "📈  Applying database migrations..."
	@$(MIGRATE) -path ./migrations -database "$(DATABASE_URL)" up

migrate-down: ## Откатить последнюю 'down' миграцию (требует запущенного postgres)
	@echo "📉  Reverting last database migration..."
	@$(MIGRATE) -path ./migrations -database "$(DATABASE_URL)" down