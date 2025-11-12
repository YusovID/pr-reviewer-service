APP_NAME=pr-reviewer-service
COMPOSE_FILE=compose.yml

.PHONY: up
up: ## Запустить все сервисы в фоновом режиме
	docker compose -f $(COMPOSE_FILE) up -d --build

.PHONY: down
down: ## Остановить и удалить все сервисы, тома и сети
	docker compose -f $(COMPOSE_FILE) down -v --remove-orphans

.PHONY: logs
logs: ## Показать логи всех сервисов в реальном времени
	docker compose -f $(COMPOSE_FILE) logs -f

.PHONY: ps
ps: ## Показать статус запущенных контейнеров
	docker compose -f $(COMPOSE_FILE) ps

.PHONY: restart
restart: down up ## Перезапустить все сервисы

.PHONY: generate
generate: ## Сгенерировать Go код из OpenAPI спецификации
	go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=oapi-codegen.yml pkg/api/openapi.yml

.PHONY: lint
lint: ## Запустить линтер для проверки качества кода
	golangci-lint run ./...

.PHONY: test
test: ## Запустить unit-тесты
	go test ./...

.PHONY: test-cover
test-cover: ## Запустить тесты с покрытием и сгенерировать HTML-отчет
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out

.PHONY: migrate-create
migrate-create: ## Создать новый файл миграции (интерактивно)
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

.PHONY: migrate-up
migrate-up: ## Применить все 'up' миграции (требует запущенного postgres)
	migrate -path ./migrations -database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5432/${POSTGRES_DB}?sslmode=disable" up

.PHONY: migrate-down
migrate-down: ## Откатить последнюю 'down' миграцию (требует запущенного postgres)
	migrate -path ./migrations -database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5432/${POSTGRES_DB}?sslmode=disable" down

.PHONY: help
help: ## Показать список всех доступных команд
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'