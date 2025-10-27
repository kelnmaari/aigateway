# Makefile для AIGateway Platform v2.1.0
# Cross-platform build automation

.PHONY: help build build-all build-linux build-windows package clean test lint run install version

# Variables
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
BUILD_DATE := $(shell date -u +"%Y-%m-%d_%H:%M:%S")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_VERSION := $(shell go version | awk '{print $$3}')
OUTPUT_DIR := dist

# Go build flags with version information
LDFLAGS := -w -s \
	-X 'aigateway/internal/version.Version=$(VERSION)' \
	-X 'aigateway/internal/version.GitCommit=$(GIT_COMMIT)' \
	-X 'aigateway/internal/version.BuildDate=$(BUILD_DATE)'

BUILD_TAGS := sqlite_fts5 sqlite_json1

# Default target
.DEFAULT_GOAL := help

## help: Показать список доступных команд
help:
	@echo "AIGateway Platform v$(VERSION) - Build Commands"
	@echo ""
	@echo "Usage: make [command]"
	@echo ""
	@echo "Commands:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
	@echo ""

## version: Показать информацию о версии
version:
	@echo "Version:    $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Go Version: $(GO_VERSION)"

## build: Сборка для текущей платформы
build:
	@echo "Building for current platform..."
	@go build -ldflags="$(LDFLAGS)" -tags "$(BUILD_TAGS)" -trimpath -o $(OUTPUT_DIR)/aigateway ./cmd/server
	@echo "✓ Built: $(OUTPUT_DIR)/aigateway"

## build-all: Сборка для всех платформ (Linux amd64/arm64, Windows amd64)
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(OUTPUT_DIR)
	
	@echo "Building Linux AMD64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -tags "$(BUILD_TAGS)" -trimpath -o $(OUTPUT_DIR)/aigateway-linux-amd64 ./cmd/server
	
	@echo "Building Linux ARM64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -tags "$(BUILD_TAGS)" -trimpath -o $(OUTPUT_DIR)/aigateway-linux-arm64 ./cmd/server
	
	@echo "Building Windows AMD64..."
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -tags "$(BUILD_TAGS)" -trimpath -o $(OUTPUT_DIR)/aigateway-windows-amd64.exe ./cmd/server
	
	@echo "✓ All builds completed!"
	@ls -lh $(OUTPUT_DIR)/

## build-linux: Сборка только для Linux (amd64 + arm64)
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(OUTPUT_DIR)
	
	@echo "Building Linux AMD64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -tags "$(BUILD_TAGS)" -trimpath -o $(OUTPUT_DIR)/aigateway-linux-amd64 ./cmd/server
	
	@echo "Building Linux ARM64..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -tags "$(BUILD_TAGS)" -trimpath -o $(OUTPUT_DIR)/aigateway-linux-arm64 ./cmd/server
	
	@echo "✓ Linux builds completed!"

## build-windows: Сборка только для Windows (amd64)
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(OUTPUT_DIR)
	
	@echo "Building Windows AMD64..."
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -tags "$(BUILD_TAGS)" -trimpath -o $(OUTPUT_DIR)/aigateway-windows-amd64.exe ./cmd/server
	
	@echo "✓ Windows build completed!"

## package: Создать distribution пакеты (tar.gz для Linux, zip для Windows)
package: build-all
	@echo "Creating distribution packages..."
	@./scripts/package.sh $(VERSION) $(OUTPUT_DIR)
	@echo "✓ Packages created!"

## clean: Очистить build артефакты
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(OUTPUT_DIR)
	@rm -rf data/*.db data/*.db-*
	@rm -rf logs/*.log
	@echo "✓ Cleaned!"

## test: Запустить тесты
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | grep total

## test-coverage: Запустить тесты с coverage report
test-coverage: test
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

## lint: Запустить линтеры
lint:
	@echo "Running linters..."
	@go vet ./...
	@go fmt ./...
	@echo "✓ Linting completed!"

## run: Запустить server локально
run:
	@echo "Starting server..."
	@go run ./cmd/server

## run-tui: Запустить TUI локально
run-tui:
	@echo "Starting TUI..."
	@go run ./cmd/tui

## install: Установить зависимости
install:
	@echo "Installing dependencies..."
	@go mod download
	@go mod verify
	@echo "✓ Dependencies installed!"

## deps: Обновить зависимости
deps:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "✓ Dependencies updated!"

## db-migrate: Запустить database migrations
db-migrate:
	@echo "Running database migrations..."
	@go run ./cmd/server --migrate
	@echo "✓ Migrations completed!"

## docker-build: Собрать Docker image
docker-build:
	@echo "Building Docker image..."
	@docker build -t ollama-openai-proxy:$(VERSION) -f docker/Dockerfile --target production .
	@echo "✓ Docker image built!"

## docker-run: Запустить в Docker
docker-run:
	@echo "Starting with Docker Compose..."
	@docker-compose up -d
	@echo "✓ Started! Access at http://localhost:8080"

## docker-stop: Остановить Docker containers
docker-stop:
	@echo "Stopping Docker containers..."
	@docker-compose down
	@echo "✓ Stopped!"

## release: Подготовить release (build + package + checksums)
release: clean build-all package
	@echo "Creating checksums..."
	@cd $(OUTPUT_DIR) && sha256sum aigateway-$(VERSION)-*.{tar.gz,zip} > checksums.txt 2>/dev/null || true
	@echo "✓ Release ready in $(OUTPUT_DIR)/"
	@ls -lh $(OUTPUT_DIR)/

## version: Показать версию
version:
	@echo "AIGateway Platform"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

