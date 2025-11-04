# RAG Infrastructure - Quick Reference

## 🚀 Быстрый старт

### Windows (PowerShell)
```powershell
.\docker\rag-quick-start.ps1
```

### Linux/macOS (Bash)
```bash
chmod +x ./docker/rag-quick-start.sh
./docker/rag-quick-start.sh
```

## 📁 Файлы в этой директории

### Основные
- `docker-compose.rag-infrastructure.yml` - основной compose файл (в корне проекта)
- `.env.rag.example` - пример переменных окружения
- `rag-quick-start.ps1` / `rag-quick-start.sh` - интерактивные скрипты запуска

### PostgreSQL + pgvector
- `postgres-pgvector-init.sql` - инициализация pgvector и RAG schema
- `postgres-pgvector-tuning.sql` - оптимизация производительности

### Мониторинг
- `prometheus-rag.yml` - конфигурация Prometheus
- `grafana-rag/` - datasources и dashboards для Grafana

## 🔧 Ручной запуск

### 1. Создать .env файл
```bash
cp docker/.env.rag.example .env
# Отредактируйте .env и измените пароли!
```

### 2. Запустить сервисы

**Базовая конфигурация (PostgreSQL + Redis):**
```bash
docker-compose -f docker-compose.rag-infrastructure.yml up -d
```

**С Qdrant:**
```bash
docker-compose -f docker-compose.rag-infrastructure.yml --profile qdrant up -d
```

**С MinIO:**
```bash
docker-compose -f docker-compose.rag-infrastructure.yml --profile minio up -d
```

**С мониторингом:**
```bash
docker-compose -f docker-compose.rag-infrastructure.yml --profile monitoring up -d
```

**Полный стек:**
```bash
docker-compose -f docker-compose.rag-infrastructure.yml --profile full up -d
```

### 3. Проверить статус
```bash
docker-compose -f docker-compose.rag-infrastructure.yml ps
```

## 🔌 Connection Strings

### PostgreSQL
```
postgres://proxy_user:secure_password@localhost:5432/ollama_proxy?sslmode=disable
```

**CLI:** `docker exec -it rag-postgres psql -U proxy_user -d ollama_proxy`

**WebUI (pgAdmin):** http://localhost:5050 (email: `admin@admin.com`, password: см. `.env`)

### Redis
```
redis://localhost:6379/0
```

**Подключение:**
```bash
docker exec -it rag-redis redis-cli
```

### Qdrant
```
http://localhost:6333
```

**WebUI:** http://localhost:6333/dashboard

### MinIO
```
Endpoint: http://localhost:9000
Access Key: minioadmin
Secret Key: minioadmin123
```

**Console:** http://localhost:9001

### Grafana
**WebUI:** http://localhost:3000
- Username: `admin`
- Password: `admin123` (измените в `.env`)

## 🛠️ Полезные команды

### PostgreSQL
```bash
# Проверить pgvector
docker exec -it rag-postgres psql -U proxy_user -d ollama_proxy -c "SELECT * FROM pg_extension WHERE extname = 'vector';"

# Статистика RAG
docker exec -it rag-postgres psql -U proxy_user -d ollama_proxy -c "SELECT * FROM rag.statistics;"

# Размер таблиц
docker exec -it rag-postgres psql -U proxy_user -d ollama_proxy -c "SELECT * FROM rag.table_sizes;"
```

### Redis
```bash
# Проверка
docker exec -it rag-redis redis-cli ping

# Статистика
docker exec -it rag-redis redis-cli INFO stats

# Очистка
docker exec -it rag-redis redis-cli FLUSHALL
```

### Qdrant
```bash
# Healthcheck
curl http://localhost:6333/healthz

# Список коллекций
curl http://localhost:6333/collections
```

### MinIO
```bash
# Setup mc client
docker exec -it rag-minio mc alias set local http://localhost:9000 minioadmin minioadmin123

# Список buckets
docker exec -it rag-minio mc ls local/
```

## 🔄 Backup

### PostgreSQL
```bash
# Backup
docker exec -t rag-postgres pg_dump -U proxy_user -d ollama_proxy > backup.sql

# Restore
docker exec -i rag-postgres psql -U proxy_user -d ollama_proxy < backup.sql
```

### Redis
```bash
docker exec rag-redis redis-cli SAVE
docker cp rag-redis:/data/dump.rdb ./redis-backup.rdb
```

## 🧹 Остановка и очистка

### Остановить
```bash
docker-compose -f docker-compose.rag-infrastructure.yml down
```

### Очистить данные (⚠️ удалит все!)
```bash
docker-compose -f docker-compose.rag-infrastructure.yml down -v
```

## 📖 Полная документация

См. `RAG_INFRASTRUCTURE_README.md` в корне проекта.

