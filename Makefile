# ====================================================================================
# VARIABLES & OS DETECTION
# ====================================================================================

# Имя приложения и compose-файл
APP_NAME := pr-reviewer-service
COMPOSE_FILE := compose.yml

# Загружаем переменные из .env файла
-include .env
export

# Команды для инструментов Go. Они кроссплатформенны по своей природе.
GO_OAPI_CODEGEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/cmd/golangci-lint
MIGRATE := go run github.com/golang-migrate/migrate/v4/cmd/migrate

# Сокращение для docker-compose
COMPOSE := docker compose -f $(COMPOSE_FILE)

# --- OS-specific setup ---
# Определяем операционную систему. GNU Make предоставляет переменную OS.
ifeq ($(OS),Windows_NT)
    # Настройки для Windows
    IS_WINDOWS := 1
    RM := del /q /f
    SLEEP := timeout /t
    # В cmd.exe нет простого способа сделать интерактивный ввод, поэтому делаем его неинтерактивным.
    # Пользователю Windows придется передавать имя миграции как переменную.
    # Пример: make migrate-create name=my_new_migration
    MIGRATE_CREATE_CMD = $(MIGRATE) create -ext sql -dir migrations -seq $(name)
    HELP_CMD = @echo "To get help on Windows, please use a Linux-like shell (Git Bash, WSL) or view the Makefile directly."
else
    # Настройки для Unix-подобных систем (Linux, MacOS, WSL, Git Bash)
    IS_WINDOWS := 0
    RM := rm -f
    SLEEP := sleep
    MIGRATE_CREATE_CMD = @read -p "Enter migration name (e.g., add_pr_status_index): " name; \
                       $(MIGRATE) create -ext sql -dir migrations -seq $$name
    # Команда help для Unix-систем (с цветом и форматированием)
    HELP_CMD = @awk 'BEGIN {FS = ":.*?## "; printf "  \033[36m%-20s\033[0m %s\n", "Target", "Description"} /^[a-zA-Z_-]+:.*?## / { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST) | sort -k 2
endif

# Собираем DSN для миграций. `localhost` будет корректно разрешен Docker Desktop на всех ОС.
DATABASE_URL := postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable

# ====================================================================================
# SETUP
# ====================================================================================

.DEFAULT_GOAL := help

.PHONY: all help build up start stop restart down nuke logs ps clean generate fmt lint test test-integration test-cover test-load tools migrate-create migrate-up migrate-down

# ====================================================================================
# GENERAL COMMANDS
# ====================================================================================

all: fmt lint test ## Запустить форматирование, линтер и тесты

help: ## Показать этот список команд
	@echo "Usage: make <target>"
	@echo ""
	@echo "Available targets:"
	@$(HELP_CMD)

# ====================================================================================
# DOCKER COMPOSE MANAGEMENT
# ====================================================================================
# Эти команды полностью полагаются на docker-compose, который является кроссплатформенным.
# Никаких изменений не требуется.

build: ## Собрать или пересобрать образы сервисов
	@echo "Building service images..."
	@$(COMPOSE) build

up: build ## Собрать образы и запустить сервисы в фоне
	@echo "Starting services..."
	@$(COMPOSE) up -d

start: ## Запустить ранее остановленные контейнеры
	@echo "Starting existing containers..."
	@$(COMPOSE) start

stop: ## Остановить запущенные сервисы
	@echo "Stopping services..."
	@$(COMPOSE) stop

restart: stop start ## Перезапустить сервисы

down: ## Остановить и удалить контейнеры/сети
	@echo "Tearing down services (volumes are preserved)..."
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
# Эти команды полностью полагаются на Go, который является кроссплатформенным.
# Изменения требуются только для команд, взаимодействующих с файловой системой.

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

# Цель test-cover переписана для кроссплатформенности
test-cover: ## Запустить ВСЕ тесты с покрытием и сгенерировать HTML-отчет
	@echo "Running all tests with coverage..."
	@go test -race -short -coverprofile=unit.cover ./...
	@go test -race -tags=integration -coverprofile=integration.cover ./...
	@go run ./tools/cover-merger.go # Используем улучшенный Go-скрипт
	@$(RM) unit.cover integration.cover
	@go tool cover -html=coverage.out

clean: ## Очистить артефакты сборки и тестирования
	@echo "Cleaning up..."
	@$(RM) coverage.out *.test *.exe

tools: ## Установить/обновить зависимости для утилит
	@echo "Syncing tools dependencies..."
	@go mod -C tools tidy

# ====================================================================================
# DATABASE MIGRATIONS
# ====================================================================================
# Используем переменную MIGRATE_CREATE_CMD, определенную в начале файла.

migrate-create: ## Создать новый файл миграции (на Windows: make migrate-create name=...)
	@$(MIGRATE_CREATE_CMD)

migrate-up: ## Применить все 'up' миграции
	@echo "Applying database migrations..."
	@$(MIGRATE) -path ./migrations -database "$(DATABASE_URL)" up

migrate-down: ## Откатить последнюю 'down' миграцию
	@echo "Reverting last database migration..."
	@$(MIGRATE) -path ./migrations -database "$(DATABASE_URL)" down