# Deal Sense — команды разработки

# По умолчанию — список команд
default:
    @just --list

# === Backend ===

# Запуск backend dev-сервера
dev-backend:
    cd backend && go run ./cmd/server

# Запуск тестов backend
test:
    cd backend && go test ./... -v

# Запуск тестов с покрытием
test-cover:
    cd backend && go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out

# Покрытие в HTML
test-cover-html:
    cd backend && go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out -o coverage.html && open coverage.html

# Сборка backend
build-backend:
    cd backend && go build -o bin/server ./cmd/server

# Линтер (go vet + golangci-lint если установлен)
lint:
    cd backend && go vet ./...
    @cd backend && if command -v golangci-lint >/dev/null 2>&1; then \
        golangci-lint run ./...; \
    else \
        echo "golangci-lint not installed — skipping. Install: https://golangci-lint.run/welcome/install/"; \
    fi

# === Frontend ===

# Запуск frontend dev-сервера
dev-frontend:
    cd frontend && npm run dev

# Тесты frontend (unit + integration)
test-frontend:
    cd frontend && npm test

# Тесты frontend e2e
test-e2e:
    cd frontend && npx playwright test

# TypeScript проверка
typecheck:
    cd frontend && npx tsc --noEmit

# Сборка frontend
build-frontend:
    cd frontend && npm run build

# === Всё вместе ===

# Запуск обоих серверов (параллельно)
dev:
    just dev-backend & just dev-frontend & wait

# Полная сборка
build: build-backend build-frontend

# Все тесты
test-all: test test-frontend

# Docker
docker-up:
    docker compose up --build -d

docker-down:
    docker compose down
