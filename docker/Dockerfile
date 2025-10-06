# Multi-stage Dockerfile для Ollama-OpenAI Proxy v1.3.0
# Поддержка Go 1.25, PostgreSQL, WebUI

# ================================
# Stage 1: Builder
# ================================
FROM golang:1.25-alpine AS builder

# Установка необходимых инструментов
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    sqlite \
    gcc \
    musl-dev \
    postgresql-dev

# Создание пользователя
RUN adduser -D -s /bin/sh -u 1001 appuser

# Установка рабочей директории
WORKDIR /build

# Копирование go mod файлов
COPY go.mod go.sum ./

# Загрузка зависимостей
RUN go mod download
RUN go mod verify

# Копирование исходного кода
COPY . .

# Сборка приложений с оптимизациями для Go 1.25
ARG VERSION=1.3.0
ARG BUILD_TIME
ARG GIT_COMMIT
ARG CGO_ENABLED=1

ENV CGO_ENABLED=${CGO_ENABLED}

# Сборка server (с WebUI)
RUN go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
    -trimpath \
    -tags "sqlite_fts5 sqlite_json1" \
    -o /build/bin/server \
    ./cmd/server

# Сборка TUI
RUN go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -trimpath \
    -o /build/bin/tui \
    ./cmd/tui

# ================================
# Stage 2: Development
# ================================
FROM golang:1.25-alpine AS development

# Установка инструментов для разработки
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    sqlite \
    gcc \
    musl-dev \
    postgresql-dev \
    postgresql-client \
    curl \
    bash \
    make

# Установка Air для hot reload
RUN go install github.com/cosmtrek/air@latest

# Установка Delve для отладки
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# Установка дополнительных инструментов
RUN go install honnef.co/go/tools/cmd/staticcheck@latest
RUN go install golang.org/x/tools/cmd/goimports@latest
RUN go install golang.org/x/vuln/cmd/govulncheck@latest

WORKDIR /app

# Копирование конфигурации Air
COPY .air.toml .

# Экспозиция портов
EXPOSE 8080 2345

CMD ["air", "-c", ".air.toml"]

# ================================
# Stage 3: Production
# ================================
FROM alpine:latest AS production

# Установка runtime зависимостей
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    sqlite \
    postgresql-client \
    curl \
    && rm -rf /var/cache/apk/*

# Создание пользователя
RUN adduser -D -s /bin/sh -u 1001 appuser

# Создание необходимых директорий
RUN mkdir -p /app/configs /app/data /app/logs /app/web && \
    chown -R appuser:appuser /app

# Копирование бинарников из builder stage
COPY --from=builder /build/bin/server /app/
COPY --from=builder /build/bin/tui /app/

# Копирование конфигураций
COPY --chown=appuser:appuser configs/ /app/configs/

# Копирование WebUI (Version 1.3.0 feature)
COPY --chown=appuser:appuser web/ /app/web/

# Создание health check скрипта
RUN echo '#!/bin/sh\ncurl -f http://localhost:8080/health || exit 1' > /app/healthcheck.sh && \
    chmod +x /app/healthcheck.sh && \
    chown appuser:appuser /app/healthcheck.sh

# Переключение на пользователя
USER appuser

WORKDIR /app

# Экспозиция порта
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD /app/healthcheck.sh

# Запуск приложения
CMD ["./server"]

# ================================
# Stage 4: TUI только
# ================================
FROM alpine:latest AS tui

# Установка runtime зависимостей
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*

# Создание пользователя
RUN adduser -D -s /bin/sh -u 1001 appuser

# Создание директорий
RUN mkdir -p /app && \
    chown -R appuser:appuser /app

# Копирование TUI бинарника
COPY --from=builder /build/bin/tui /app/

# Переключение на пользователя
USER appuser

WORKDIR /app

# Запуск TUI
CMD ["./tui"]

# ================================
# Stage 5: Testing
# ================================
FROM golang:1.25-alpine AS testing

RUN apk add --no-cache \
    git \
    ca-certificates \
    sqlite \
    gcc \
    musl-dev \
    postgresql-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Установка инструментов для тестирования
RUN go install honnef.co/go/tools/cmd/staticcheck@latest
RUN go install golang.org/x/vuln/cmd/govulncheck@latest

# Запуск тестов с покрытием
CMD ["go", "test", "-v", "-race", "-coverprofile=coverage.out", "./..."]

# ================================
# Build аргументы и лейблы
# ================================

ARG VERSION=1.3.0
ARG BUILD_TIME
ARG GIT_COMMIT

LABEL maintainer="Ollama-OpenAI Proxy Team"
LABEL version="${VERSION}"
LABEL build-time="${BUILD_TIME}"
LABEL git-commit="${GIT_COMMIT}"
LABEL org.opencontainers.image.title="Ollama-OpenAI Proxy"
LABEL org.opencontainers.image.description="OpenAI-compatible proxy for local Ollama models with WebUI"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.created="${BUILD_TIME}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/yourusername/ollama-openai-proxy"
