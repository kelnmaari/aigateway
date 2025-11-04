po# RAG Infrastructure Setup

Полный стек инфраструктуры для RAG (Retrieval-Augmented Generation) системы без самого сервера и Ollama.

## 🏗️ Компоненты

### Базовые (всегда запускаются)

- **PostgreSQL 16 + pgvector** - векторная база данных для embeddings
- **pgAdmin 4** - Web UI для управления PostgreSQL
- **Redis 7** - кеш и очередь задач

### Опциональные (через profiles)

- **Qdrant** - альтернативный vector store
- **MinIO** - S3-совместимое объектное хранилище
- **Prometheus + Grafana** - мониторинг

## 🚀 Quick Start

### 1. Подготовка

Скопируйте пример переменных окружения:

```bash
cp docker/.env.rag.example .env
```

Отредактируйте `.env` и измените пароли:

```env
POSTGRES_PASSWORD=your_secure_password_here
PGADMIN_DEFAULT_PASSWORD=your_pgadmin_password_here
MINIO_ROOT_PASSWORD=your_minio_password_here
GRAFANA_ADMIN_PASSWORD=your_grafana_password_here
```

### 2. Запуск базовой конфигурации

PostgreSQL + Redis:

```bash
docker-compose -f docker-compose.rag-infrastructure.yml up -d
```

### 3. Запуск полного стека

Все компоненты (PostgreSQL, Redis, Qdrant, MinIO, Prometheus, Grafana):

```bash
docker-compose -f docker-compose.rag-infrastructure.yml --profile full up -d
```

### 4. Проверка статуса

```bash
docker-compose -f docker-compose.rag-infrastructure.yml ps
```

## 📋 Профили запуска

| Команда | Компоненты |
|---------|------------|
| `up -d` | PostgreSQL + Redis |
| `--profile qdrant up -d` | + Qdrant |
| `--profile minio up -d` | + MinIO + mc setup |
| `--profile monitoring up -d` | + Prometheus + Grafana |
| `--profile full up -d` | Все компоненты |

## 🔌 Connection Strings

### PostgreSQL (pgvector)

```
postgres://proxy_user:secure_password@localhost:5432/ollama_proxy?sslmode=disable
```

**CLI:** `docker exec -it rag-postgres psql -U proxy_user -d ollama_proxy`

### pgAdmin (PostgreSQL Web UI)

**WebUI:** <http://localhost:5050>

- **Email:** `admin@admin.com` (измените в `.env`)
- **Password:** `admin123` (измените в `.env`)

**Pre-configured servers:**

- RAG PostgreSQL (pgvector) - автоматически подключен

При первом входе используйте пароль от PostgreSQL (`POSTGRES_PASSWORD` из `.env`) для подключения к серверу

### Redis

```
redis://localhost:6379/0
```

**WebUI:** N/A (используйте redis-cli или RedisInsight)

### Qdrant (опционально)

```
http://localhost:6333
```

**WebUI:** <http://localhost:6333/dashboard>

### MinIO (опционально)

```
Endpoint: http://localhost:9000
Access Key: minioadmin
Secret Key: minioadmin123
```

**Console:** <http://localhost:9001>

### Prometheus (опционально)

**WebUI:** <http://localhost:9090>

### Grafana (опционально)

**WebUI:** <http://localhost:3000>

- Username: `admin`
- Password: `admin123` (измените в `.env`)

## 🔧 Конфигурация для вашего приложения

Добавьте в `configs/dev.yaml` или `configs/production.yaml`:

```yaml
# Database
database:
  type: "postgresql"
  postgresql:
    host: "localhost"
    port: 5432
    user: "proxy_user"
    password: "secure_password"
    database: "ollama_proxy"
    sslmode: "disable"

# Redis
auth:
  rate_limiting:
    redis:
      enabled: true
      url: "redis://localhost:6379"

# RAG System
rag:
  enabled: true
  
  # Вариант 1: pgvector
  vector_store:
    type: "pgvector"
    connection_string: "postgres://proxy_user:secure_password@localhost:5432/ollama_proxy?sslmode=disable"
    dimensions: 1024  # для mxbai-embed-large
    distance_metric: "cosine"
    hnsw_m: 16
    hnsw_ef_construction: 64
  
  # Вариант 2: Qdrant (раскомментируйте если используете)
  # vector_store:
  #   type: "qdrant"
  #   url: "http://localhost:6333"
  #   collection: "ollama-proxy-vectors"
  
  # File storage
  file_storage:
    type: "s3"  # или "local"
    s3:
      endpoint: "http://localhost:9000"
      bucket: "rag-documents"
      access_key: "minioadmin"
      secret_key: "minioadmin123"
      use_ssl: false
  
  # Embeddings
  embeddings:
    provider: "ollama"
    model: "mxbai-embed-large"
    dimensions: 1024
```

## 🛠️ Полезные команды

### PostgreSQL

**Подключение через psql (CLI):**

```bash
docker exec -it rag-postgres psql -U proxy_user -d ollama_proxy
```

**Подключение через pgAdmin (Web UI):**

1. Откройте <http://localhost:5050>
2. Войдите (email: `admin@admin.com`, password: см. `.env`)
3. Сервер "RAG PostgreSQL (pgvector)" уже настроен
4. При первом подключении введите пароль PostgreSQL из `.env`

**Проверка расширений:**

```sql
SELECT * FROM pg_extension WHERE extname = 'vector';
```

**Статистика RAG:**

```sql
SELECT * FROM rag.statistics;
```

**Размер таблиц:**

```sql
SELECT * FROM rag.table_sizes;
```

**Использование индексов:**

```sql
SELECT * FROM rag.index_usage;
```

**Пример vector search:**

```sql
-- Найти похожие chunks
SELECT 
    chunk_text,
    1 - (embedding <=> query_embedding) AS similarity
FROM rag.document_chunks
ORDER BY embedding <=> query_embedding
LIMIT 5;
```

### Redis

**Подключение:**

```bash
docker exec -it rag-redis redis-cli
```

**Статистика:**

```bash
docker exec -it rag-redis redis-cli INFO stats
```

**Очистка кеша:**

```bash
docker exec -it rag-redis redis-cli FLUSHALL
```

### Qdrant

**Проверка здоровья:**

```bash
curl http://localhost:6333/healthz
```

**Список коллекций:**

```bash
curl http://localhost:6333/collections
```

**Создание коллекции:**

```bash
curl -X PUT "http://localhost:6333/collections/my-collection" \
  -H "Content-Type: application/json" \
  -d '{
    "vectors": {
      "size": 1024,
      "distance": "Cosine"
    }
  }'
```

### MinIO

**Подключение mc client:**

```bash
docker exec -it rag-minio mc alias set local http://localhost:9000 minioadmin minioadmin123
```

**Список buckets:**

```bash
docker exec -it rag-minio mc ls local/
```

**Создание bucket:**

```bash
docker exec -it rag-minio mc mb local/my-new-bucket
```

**Upload файла:**

```bash
docker exec -it rag-minio mc cp /path/to/file local/rag-documents/
```

## 📊 Мониторинг

### Prometheus

Открыть: <http://localhost:9090>

**Полезные queries:**

```promql
# PostgreSQL connections
pg_stat_database_numbackends{datname="ollama_proxy"}

# Redis memory usage
redis_memory_used_bytes

# Qdrant collection size
qdrant_collections_size_bytes

# MinIO storage usage
minio_bucket_usage_total_bytes
```

### Grafana

Открыть: <http://localhost:3000>

**Pre-configured dashboards:**

- PostgreSQL Overview
- Redis Monitoring
- MinIO Metrics
- RAG System Performance

## 🔄 Backup & Restore

### PostgreSQL

**Backup:**

```bash
docker exec -t rag-postgres pg_dump -U proxy_user -d ollama_proxy > backup.sql
```

**Restore:**

```bash
docker exec -i rag-postgres psql -U proxy_user -d ollama_proxy < backup.sql
```

**Backup только RAG schema:**

```bash
docker exec -t rag-postgres pg_dump -U proxy_user -d ollama_proxy -n rag > rag_backup.sql
```

### Redis

**RDB Snapshot:**

```bash
docker exec rag-redis redis-cli SAVE
docker cp rag-redis:/data/dump.rdb ./redis-backup.rdb
```

**Restore:**

```bash
docker cp ./redis-backup.rdb rag-redis:/data/dump.rdb
docker-compose -f docker-compose.rag-infrastructure.yml restart redis
```

### Qdrant

**Создать snapshot:**

```bash
curl -X POST "http://localhost:6333/collections/my-collection/snapshots"
```

**Download snapshot:**

```bash
curl "http://localhost:6333/collections/my-collection/snapshots/snapshot-name" \
  --output snapshot.tar
```

### MinIO

**Mirror backup:**

```bash
docker exec -it rag-minio mc mirror local/rag-documents ./minio-backup/
```

**Restore:**

```bash
docker exec -it rag-minio mc mirror ./minio-backup/ local/rag-documents
```

## 🧹 Очистка

**Остановить все:**

```bash
docker-compose -f docker-compose.rag-infrastructure.yml down
```

**Остановить и удалить volumes (⚠️ удалит все данные!):**

```bash
docker-compose -f docker-compose.rag-infrastructure.yml down -v
```

**Удалить конкретный volume:**

```bash
docker volume rm rag_postgres_data
docker volume rm rag_pgadmin_data
docker volume rm rag_redis_data
docker volume rm rag_qdrant_data
docker volume rm rag_minio_data
```

## 📈 Performance Tuning

### PostgreSQL

Настройки производительности применяются автоматически через `postgres-pgvector-tuning.sql`.

**Ключевые параметры:**

- `shared_buffers = 512MB` - кеш для векторов
- `maintenance_work_mem = 256MB` - для построения HNSW индексов
- `work_mem = 64MB` - для сортировки результатов
- `effective_io_concurrency = 200` - SSD оптимизация

**Проверка настроек:**

```sql
SELECT name, setting, unit FROM pg_settings 
WHERE name IN ('shared_buffers', 'work_mem', 'maintenance_work_mem');
```

### Redis

**Изменить memory policy:**

```bash
docker exec rag-redis redis-cli CONFIG SET maxmemory-policy allkeys-lru
docker exec rag-redis redis-cli CONFIG SET maxmemory 2gb
```

### HNSW Index Tuning

**Для больших dataset (>1M vectors):**

```sql
CREATE INDEX idx_document_chunks_embedding_hnsw 
    ON rag.document_chunks 
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 32, ef_construction = 128);  -- Увеличенные параметры
```

**Trade-offs:**

- `m`: Higher = better recall, more memory (16-64)
- `ef_construction`: Higher = better quality, slower build (64-512)

## 🐛 Troubleshooting

### PostgreSQL не запускается

**Проверка логов:**

```bash
docker logs rag-postgres
```

**Возможные причины:**

- Занят порт 5432: измените `POSTGRES_PORT` в `.env`
- Недостаточно памяти: уменьшите `shared_buffers` в tuning.sql
- Поврежден volume: `docker volume rm rag_postgres_data` и пересоздайте

### pgvector extension не найден

**Проверка:**

```sql
SELECT * FROM pg_available_extensions WHERE name = 'vector';
```

**Решение:**
Используйте образ `pgvector/pgvector:pg16` вместо `postgres:16`.

### Redis out of memory

**Увеличить maxmemory:**

```bash
docker exec rag-redis redis-cli CONFIG SET maxmemory 4gb
```

**Или в docker-compose.yml:**

```yaml
command: >
  redis-server
  --maxmemory 4gb
```

### Qdrant медленный search

**Проверить размер коллекции:**

```bash
curl "http://localhost:6333/collections/my-collection" | jq '.result'
```

**Увеличить memory:**

```yaml
deploy:
  resources:
    limits:
      memory: 8G  # Увеличить с 4G
```

### MinIO не создает buckets

**Проверка логов setup:**

```bash
docker logs rag-minio-setup
```

**Пересоздать setup:**

```bash
docker-compose -f docker-compose.rag-infrastructure.yml up minio-setup --force-recreate
```

## 📚 Дополнительная документация

- [pgvector GitHub](https://github.com/pgvector/pgvector)
- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [MinIO Documentation](https://min.io/docs/)
- [Redis Documentation](https://redis.io/docs/)
- [PostgreSQL Performance Tuning](https://wiki.postgresql.org/wiki/Performance_Optimization)

## 🔗 Связанные файлы

- `docker-compose.rag-infrastructure.yml` - основная конфигурация
- `docker/postgres-pgvector-init.sql` - инициализация pgvector
- `docker/postgres-pgvector-tuning.sql` - настройки производительности
- `docker/pgadmin-servers.json` - pre-configured pgAdmin servers
- `docker/prometheus-rag.yml` - конфигурация Prometheus
- `docker/.env.rag.example` - пример переменных окружения
- `configs/rag-example.yaml` - пример конфигурации RAG системы

## 💡 Best Practices

1. **Security:**
   - Измените все пароли по умолчанию в `.env`
   - Используйте SSL для production (`sslmode=require`)
   - Ограничьте доступ через firewall

2. **Performance:**
   - Используйте HNSW индексы для больших dataset
   - Настройте `maintenance_work_mem` для быстрого построения индексов
   - Monitor query performance через `pg_stat_statements`

3. **Scaling:**
   - PostgreSQL: используйте read replicas для распределения нагрузки
   - Redis: используйте Redis Cluster для больших dataset
   - Qdrant: горизонтальное масштабирование через distributed mode

4. **Backup:**
   - Настройте регулярные backup (daily/weekly)
   - Храните backup в отдельном location (S3, другой сервер)
   - Тестируйте restore процедуру

## 📞 Support

Если возникли проблемы:

1. Проверьте логи: `docker logs <container-name>`
2. Проверьте healthcheck: `docker ps`
3. Проверьте сеть: `docker network inspect rag_network`
4. Создайте issue с полным описанием проблемы
