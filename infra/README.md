# AIGateway Infrastructure

Docker Compose конфигурация для локальной разработки AIGateway.

## Сервисы

| Сервис | Порт | Описание |
|--------|------|----------|
| PostgreSQL | 5432 | Основная БД |
| Redis | 6379 | Rate limiting и кеширование |
| MinIO | 9000 (API), 9001 (Console) | S3-совместимое хранилище |
| Qdrant | 6333 (HTTP), 6334 (gRPC) | Vector database для RAG |
| Jaeger | 16686 (UI), 4318 (OTLP) | Distributed tracing (опционально) |
| Keycloak | 8180 | OIDC/SSO (опционально) |

## Быстрый старт

```bash
# Из корня проекта
cd infra

# Скопировать конфигурацию
cp env.template .env

# Запустить базовые сервисы (PostgreSQL, Redis, MinIO, Qdrant)
docker-compose up -d

# Проверить статус
docker-compose ps

# Логи
docker-compose logs -f
```

## Запуск с дополнительными профилями

```bash
# Базовые + Jaeger (tracing)
docker-compose --profile tracing up -d

# Базовые + Keycloak (OIDC)
docker-compose --profile auth up -d

# Все сервисы
docker-compose --profile tracing --profile auth up -d
```

## Доступ к сервисам

### Qdrant Dashboard
- URL: http://localhost:6333/dashboard
- REST API: http://localhost:6333
- gRPC: localhost:6334

### MinIO Console
- URL: http://localhost:9001
- Login: minioadmin
- Password: minioadmin123

Buckets создаются автоматически:
- `user-files` - файлы пользователей
- `public-files` - публичные файлы (анонимный доступ)
- `rag-documents` - документы для RAG

### PostgreSQL
```bash
# Подключение через psql
docker exec -it aigateway-postgres psql -U proxy_user -d ollama_proxy
```

### Redis
```bash
# Redis CLI
docker exec -it aigateway-redis redis-cli

# Проверка
127.0.0.1:6379> PING
PONG
```

### Jaeger UI (если включен)
- URL: http://localhost:16686

### Keycloak (если включен)
- URL: http://localhost:8180
- Admin: admin / admin

## Остановка

```bash
# Остановить сервисы
docker-compose down

# Остановить и удалить volumes (ОСТОРОЖНО - удалит данные!)
docker-compose down -v
```

## Конфигурация dev-local.yaml

После запуска infra, используй `configs/dev-local.yaml`:

```yaml
database:
  type: "postgresql"
  postgresql:
    host: "localhost"
    port: 5432
    user: "proxy_user"
    password: "secure_password_change_me"
    database: "ollama_proxy"

auth:
  rate_limiting:
    redis:
      enabled: true
      url: "redis://localhost:6379"

file_storage:
  backend: "s3"
  s3:
    endpoint: "http://localhost:9000"
    bucket: "user-files"
    access_key: "minioadmin"
    secret_key: "minioadmin123"

rag:
  vector_store:
    type: "qdrant"
    dimensions: 1024
    qdrant:
      url: "http://localhost:6333"
      collection: "aigateway_rag"
  file_storage:
    s3_endpoint: "http://localhost:9000"

observability:
  tracing:
    enabled: true
    jaeger:
      endpoint: "http://localhost:4318"
```

## Требования

- Docker 20.10+
- Docker Compose v2+
- ~3GB RAM для базовых сервисов (с Qdrant)
- ~5GB RAM со всеми профилями
