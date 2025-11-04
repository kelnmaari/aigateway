# ✅ RAG Infrastructure Setup - ГОТОВО

Docker Compose конфигурация для RAG системы успешно создана.

## 📦 Созданные файлы

### Основная конфигурация
```
docker-compose.rag-infrastructure.yml    # Основной compose файл с всеми сервисами
RAG_INFRASTRUCTURE_README.md             # Полная документация
```

### Docker конфигурации (директория `docker/`)
```
docker/
├── postgres-pgvector-init.sql           # Инициализация pgvector + RAG schema
├── postgres-pgvector-tuning.sql         # Performance tuning для PostgreSQL
├── prometheus-rag.yml                   # Конфигурация Prometheus
├── .env.rag.example                     # Пример переменных окружения
├── README-RAG.md                        # Краткая справка
├── rag-quick-start.sh                   # Интерактивный скрипт (Linux/macOS)
├── rag-quick-start.ps1                  # Интерактивный скрипт (Windows)
└── grafana-rag/
    ├── datasources/
    │   ├── prometheus.yml               # Datasource: Prometheus
    │   └── postgres.yml                 # Datasource: PostgreSQL
    └── dashboards/
        ├── dashboard.yml                # Provisioning config
        └── rag-overview.json            # Dashboard: RAG Overview
```

## 🏗️ Включенные компоненты

### Базовые (запускаются всегда)
- ✅ **PostgreSQL 16 + pgvector** - векторная БД для embeddings (dimensions: 1024)
- ✅ **pgAdmin 4** - Web UI для управления PostgreSQL
- ✅ **Redis 7** - кеш и очередь задач

### Опциональные (через profiles)
- ✅ **Qdrant** - альтернативный vector store
- ✅ **MinIO** - S3-совместимое файловое хранилище
- ✅ **Prometheus** - сбор метрик
- ✅ **Grafana** - визуализация и мониторинг

## 🚀 Быстрый старт

### Шаг 1: Создать .env файл
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

### Шаг 2: Запустить сервисы

**Вариант A: Интерактивный скрипт (рекомендуется)**

Windows PowerShell:
```powershell
.\docker\rag-quick-start.ps1
```

Linux/macOS:
```bash
chmod +x ./docker/rag-quick-start.sh
./docker/rag-quick-start.sh
```

**Вариант B: Docker Compose напрямую**

Базовая конфигурация (PostgreSQL + Redis):
```bash
docker-compose -f docker-compose.rag-infrastructure.yml up -d
```

Полный стек (все компоненты):
```bash
docker-compose -f docker-compose.rag-infrastructure.yml --profile full up -d
```

### Шаг 3: Проверить статус
```bash
docker-compose -f docker-compose.rag-infrastructure.yml ps
```

Все сервисы должны быть в состоянии "Up (healthy)".

## 🔌 Connection Strings для вашего сервера

Добавьте в `configs/dev.yaml` или `configs/production.yaml`:

```yaml
# Database Configuration
database:
  type: "postgresql"
  postgresql:
    host: "localhost"
    port: 5432
    user: "proxy_user"
    password: "secure_password_change_me"
    database: "ollama_proxy"
    sslmode: "disable"
    max_open_conns: 100
    max_idle_conns: 10
    conn_max_lifetime: "30m"

# Redis для rate limiting и кеша
auth:
  rate_limiting:
    redis:
      enabled: true
      url: "redis://localhost:6379"
      key_prefix: "aigateway_rl:"

# RAG System Configuration
rag:
  enabled: true
  
  # Vector Store: pgvector
  vector_store:
    type: "pgvector"
    connection_string: "postgres://proxy_user:secure_password@localhost:5432/ollama_proxy?sslmode=disable"
    dimensions: 1024  # для mxbai-embed-large
    distance_metric: "cosine"
    hnsw_m: 16
    hnsw_ef_construction: 64
  
  # Embeddings Configuration
  embeddings:
    provider: "ollama"
    ollama_url: "http://localhost:11434"
    model: "mxbai-embed-large"
    batch_size: 32
    timeout: "30s"
  
  # File Storage: MinIO (опционально)
  file_storage:
    type: "s3"
    s3:
      endpoint: "http://localhost:9000"
      bucket: "rag-documents"
      access_key: "minioadmin"
      secret_key: "minioadmin123"
      use_ssl: false
      region: "us-east-1"
  
  # Processing Configuration
  processing:
    workers: 4
    queue_size: 100
    chunk_size: 512
    chunk_overlap: 50
    splitter_type: "semantic"
  
  # Retrieval Configuration
  retrieval:
    default_top_k: 5
    max_top_k: 20
    min_similarity_score: 0.7
    rerank_enabled: true
  
  # Job Queue
  queue:
    type: "postgres"
    cleanup_interval: "1h"
    max_retries: 3
    visibility_timeout: "5m"
```

## 🌐 WebUI и Endpoints

После запуска доступны:

| Сервис | URL | Credentials |
|--------|-----|-------------|
| PostgreSQL | `localhost:5432` | user: `proxy_user`, pass: см. `.env` |
| pgAdmin | http://localhost:5050 | email: `admin@admin.com`, pass: см. `.env` |
| Redis | `localhost:6379` | N/A |
| Qdrant API | http://localhost:6333 | N/A |
| Qdrant Dashboard | http://localhost:6333/dashboard | N/A |
| MinIO API | http://localhost:9000 | см. `.env` |
| MinIO Console | http://localhost:9001 | см. `.env` |
| Prometheus | http://localhost:9090 | N/A |
| Grafana | http://localhost:3000 | user: `admin`, pass: см. `.env` |

## 🛠️ Проверка работоспособности

### PostgreSQL + pgvector

**CLI подключение:**
```bash
docker exec -it rag-postgres psql -U proxy_user -d ollama_proxy

# Проверить pgvector
SELECT * FROM pg_extension WHERE extname = 'vector';

# Проверить RAG schema
SELECT * FROM rag.statistics;

# Проверить размер таблиц
SELECT * FROM rag.table_sizes;

# Выход
\q
```

**Web UI (pgAdmin):**
1. Откройте http://localhost:5050
2. Войдите (email: `admin@admin.com`, password: см. `.env`)
3. Сервер "RAG PostgreSQL (pgvector)" уже настроен
4. При первом подключении введите пароль PostgreSQL из `.env`
5. Выполняйте SQL запросы через Query Tool

### Redis
```bash
# Проверка
docker exec -it rag-redis redis-cli ping
# Ожидаемый ответ: PONG

# Статистика
docker exec -it rag-redis redis-cli INFO stats
```

### Qdrant (если запущен)
```bash
curl http://localhost:6333/healthz
# Ожидаемый ответ: {"status":"ok"}

curl http://localhost:6333/collections
# Список коллекций
```

### MinIO (если запущен)
```bash
curl http://localhost:9000/minio/health/live
# Ожидаемый ответ: 200 OK
```

## 📊 Grafana Dashboard

После запуска с профилем `monitoring` откройте http://localhost:3000

**Pre-configured dashboards:**
- ✅ RAG System Overview - статистика документов, chunks, jobs
- ✅ PostgreSQL datasource - прямые SQL запросы
- ✅ Prometheus datasource - метрики сервисов

## 🔄 Примеры использования из Go кода

### Vector Search
```go
// Пример поиска похожих chunks
import (
    "github.com/pgvector/pgvector-go"
)

type SearchResult struct {
    ChunkID    string
    DocumentID string
    ChunkText  string
    Similarity float64
}

func SearchSimilarChunks(db *sql.DB, queryEmbedding []float32, topK int) ([]SearchResult, error) {
    query := `
        SELECT 
            id,
            document_id,
            chunk_text,
            1 - (embedding <=> $1) AS similarity
        FROM rag.document_chunks
        ORDER BY embedding <=> $1
        LIMIT $2
    `
    
    vec := pgvector.NewVector(queryEmbedding)
    rows, err := db.Query(query, vec, topK)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var results []SearchResult
    for rows.Next() {
        var r SearchResult
        if err := rows.Scan(&r.ChunkID, &r.DocumentID, &r.ChunkText, &r.Similarity); err != nil {
            return nil, err
        }
        results = append(results, r)
    }
    
    return results, nil
}
```

### Insert Document Chunks
```go
func InsertChunk(db *sql.DB, documentID, text string, embedding []float32) error {
    query := `
        INSERT INTO rag.document_chunks 
            (document_id, document_name, chunk_index, chunk_text, chunk_tokens, embedding)
        VALUES 
            ($1, $2, $3, $4, $5, $6)
    `
    
    vec := pgvector.NewVector(embedding)
    tokens := len(strings.Fields(text))
    
    _, err := db.Exec(query, documentID, "document.pdf", 0, text, tokens, vec)
    return err
}
```

## 🐛 Troubleshooting

### PostgreSQL не запускается
```bash
# Проверить логи
docker logs rag-postgres

# Возможные причины:
# 1. Порт 5432 занят - измените POSTGRES_PORT в .env
# 2. Поврежден volume:
docker volume rm rag_postgres_data
docker-compose -f docker-compose.rag-infrastructure.yml up -d
```

### pgvector extension не найден
Используйте образ `pgvector/pgvector:pg16` (уже в docker-compose.yml)

### Redis out of memory
```bash
# Увеличить maxmemory
docker exec rag-redis redis-cli CONFIG SET maxmemory 4gb
```

### Qdrant медленный search
Увеличьте memory limit в docker-compose.yml:
```yaml
deploy:
  resources:
    limits:
      memory: 8G
```

## 🧹 Остановка и очистка

### Остановить все сервисы
```bash
docker-compose -f docker-compose.rag-infrastructure.yml down
```

### Очистить данные (⚠️ удалит все volumes!)
```bash
docker-compose -f docker-compose.rag-infrastructure.yml down -v
```

## 📚 Следующие шаги

1. ✅ Инфраструктура готова
2. 🔄 Запустите ваш AIGateway сервер с конфигурацией выше
3. 🔄 Интегрируйте RAG endpoints в API
4. 🔄 Загрузите документы через file upload API
5. 🔄 Настройте embeddings через Ollama
6. 🔄 Используйте vector search в chat completions

## 📖 Полная документация

- `RAG_INFRASTRUCTURE_README.md` - полная документация
- `docker/README-RAG.md` - краткая справка
- `configs/rag-example.yaml` - пример конфигурации RAG системы

## 🎯 Performance Tips

### PostgreSQL
- Используется HNSW index для быстрого vector search
- `maintenance_work_mem = 256MB` для построения индексов
- `shared_buffers = 512MB` для кеширования
- Autovacuum настроен для частых INSERT/DELETE

### Redis
- Используется LRU eviction policy
- AOF persistence включен
- Maxmemory: 2GB (настраивается)

### Vector Search
- Cosine distance по умолчанию
- HNSW параметры: m=16, ef_construction=64
- Настраивается через конфиг

## ⚡ Scaling

Для production:
- PostgreSQL: используйте read replicas
- Redis: используйте Redis Cluster
- Qdrant: distributed mode для horizontal scaling
- MinIO: distributed deployment для HA

## 🔒 Security Checklist

- [ ] Измените все пароли в `.env`
- [ ] Используйте `sslmode=require` для PostgreSQL
- [ ] Настройте firewall rules
- [ ] Используйте TLS для production
- [ ] Ограничьте доступ к портам извне
- [ ] Регулярные backups

## 💡 Полезные ссылки

- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [MinIO Documentation](https://min.io/docs/)
- [Redis Documentation](https://redis.io/docs/)

---

**Готово к использованию! 🎉**

Запустите интерактивный скрипт или используйте docker-compose напрямую.

