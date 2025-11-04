# RAG Worker Guide

## Обзор

**RAG Worker** - фоновый процесс для синхронизации данных из различных источников и создания chunks для RAG (Retrieval-Augmented Generation).

## Архитектура

```
RAG Data Source → RAG Job Queue → RAG Worker → Documents + Chunks → Database
```

### Основные компоненты

1. **RAG Worker** (`internal/rag/worker/worker.go`)
   - Забирает jobs из очереди (`rag_jobs` таблица)
   - Обрабатывает job в зависимости от типа
   - Логирует в отдельный файл `logs/rag-worker.log`
   - Polling interval: 5 секунд
   - Max retries: 3 попытки

2. **API Sync** (`internal/rag/worker/api_sync.go`)
   - Синхронизация данных из REST API
   - Поддерживает GET/POST запросы
   - JSON и plain text response
   - Автоматический chunking и сохранение

3. **Database Sync** (`internal/rag/worker/db_sync.go`)
   - Синхронизация из PostgreSQL/MySQL баз данных
   - Выполнение SQL запросов
   - Конвертация строк в documents + chunks

4. **Web Scraping** (`internal/rag/worker/web_scrape.go`)
   - Скрейпинг HTML страниц
   - Извлечение текста из HTML
   - HTML entities декодирование
   - Batch processing нескольких URLs

## Типы источников данных

### 1. API Source (`api`)

**Config fields:**
```json
{
  "url": "https://api.example.com/data",
  "method": "GET",
  "headers": {
    "Authorization": "Bearer TOKEN"
  },
  "data_path": "data.items",
  "text_fields": ["title", "content"]
}
```

**Процесс:**
1. HTTP запрос к API
2. Парсинг JSON response
3. Извлечение данных по `data_path`
4. Создание документа для каждого item
5. Chunking по `text_fields` или всему JSON
6. Сохранение в БД

### 2. Database Source (`database`)

**Config fields:**
```json
{
  "database_type": "postgres",
  "host": "localhost",
  "port": 5432,
  "database": "mydb",
  "username": "user",
  "query": "SELECT title, content FROM articles WHERE published = true"
}
```

**Credentials (encrypted):**
```json
{
  "password": "secret123"
}
```

**Процесс:**
1. Подключение к БД
2. Выполнение SQL запроса
3. Конвертация каждой строки в document
4. Chunking по столбцам
5. Сохранение в БД

### 3. Web Source (`web`)

**Config fields:**
```json
{
  "url": "https://example.com/page",
  "urls": [
    "https://example.com/page1",
    "https://example.com/page2"
  ]
}
```

**Процесс:**
1. HTTP GET запрос к каждому URL
2. Извлечение текста из HTML (удаление тегов, скриптов)
3. Очистка whitespace
4. Chunking по предложениям/параграфам
5. Сохранение в БД

## Chunking Strategy

### Параметры (IndexingConfig)

```go
type IndexingConfig struct {
    ChunkStrategy  string   `json:"chunk_strategy,omitempty"`  // "fixed", "sentence", "paragraph"
    MaxChunkSize   int      `json:"max_chunk_size,omitempty"`  // Default: 500 chars
    ChunkOverlap   int      `json:"chunk_overlap,omitempty"`   // Default: 50 chars
    EmbedColumns   []string `json:"embed_columns,omitempty"`   // Для DB sources
    EmbedFields    []string `json:"embed_fields,omitempty"`    // Для API sources
}
```

### Стратегии

1. **Fixed** - фиксированный размер chunks с overlap
2. **Sentence** - по предложениям (умный split)
3. **Paragraph** - по параграфам (двойной newline)

### Токены

Простой подсчет: `tokens = len(text) / 4`

Для production: использовать tokenizer (tiktoken, sentence-transformers)

## Job Queue

### RAGJob Schema

```sql
CREATE TABLE rag_jobs (
    id              BIGSERIAL PRIMARY KEY,
    job_type        TEXT NOT NULL,           -- 'api_sync', 'db_query', 'web_scrape'
    payload         JSONB NOT NULL,          -- {"source_id": "..."}
    status          TEXT NOT NULL,           -- 'pending', 'processing', 'completed', 'failed'
    attempts        INTEGER DEFAULT 0,
    started_at      TIMESTAMP,
    completed_at    TIMESTAMP,
    error           TEXT,
    result          JSONB,
    locked_until    TIMESTAMP,
    created_at      TIMESTAMP DEFAULT NOW()
);
```

### Job Lifecycle

```
[pending] → [processing] → [completed|failed]
              ↓
         (locked_until)
              ↓
         (max_retries)
```

## Запуск

### В составе сервера

Worker запускается автоматически при старте сервера:

```go
ragWorker := ragworker.NewRAGWorker(db, ragLogger)
go func() {
    workerCtx := context.Background()
    ragWorker.Start(workerCtx)
}()
defer ragWorker.Stop()
```

### Логирование

Логи пишутся в отдельный файл:
- **Путь**: `logs/rag-worker.log`
- **Rotation**: 100 MB, 7 backups
- **Level**: наследуется из `config.yaml`

## Мониторинг

### Метрики

```sql
-- Статистика jobs
SELECT 
    job_type,
    status,
    COUNT(*) as count,
    AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) as avg_duration_sec
FROM rag_jobs
GROUP BY job_type, status;

-- Последние ошибки
SELECT 
    id, job_type, error, created_at
FROM rag_jobs
WHERE status = 'failed'
ORDER BY created_at DESC
LIMIT 10;
```

### Source Statistics

```sql
-- Статистика по sources
SELECT 
    id, name, source_type,
    status, last_sync_status,
    total_chunks, total_tokens,
    last_sync_at
FROM rag_data_sources
ORDER BY last_sync_at DESC;
```

## API Endpoints

### Создание RAG Source

```http
POST /api/rag/sources
Content-Type: application/json

{
  "name": "Company API",
  "description": "Internal REST API",
  "source_type": "api",
  "config": {
    "url": "https://api.company.com/data",
    "method": "GET",
    "headers": {"Authorization": "Bearer TOKEN"},
    "data_path": "results",
    "text_fields": ["title", "description"]
  },
  "credentials": null,
  "indexing_config": {
    "chunk_strategy": "sentence",
    "max_chunk_size": 500,
    "chunk_overlap": 50
  }
}
```

**Response:**
```json
{
  "id": "src_abc123",
  "status": "inactive",
  "total_chunks": 0,
  "created_at": "2024-11-04T00:00:00Z"
}
```

### Запуск синхронизации

```http
POST /api/rag/sources/{source_id}/sync
```

**Response:**
```json
{
  "job_id": 123,
  "message": "Sync job queued successfully",
  "estimated_time": "5m"
}
```

### Просмотр chunks

```http
GET /api/rag/sources/{source_id}/chunks?limit=10&offset=0
```

**Response:**
```json
{
  "chunks": [
    {
      "id": "chunk_123",
      "document_id": "doc_456",
      "chunk_text": "Sample text...",
      "chunk_index": 0,
      "chunk_tokens": 125,
      "created_at": "2024-11-04T00:00:00Z"
    }
  ],
  "total": 100
}
```

## Использование в Chat

### Автоматическое enrichment

Когда RAG включен в `configs/dev.yaml`:

```yaml
rag:
  enabled: true
  max_chunks: 5
  similarity_threshold: 0.7
```

При отправке сообщения в чат:
1. Запрос пользователя → RAG Orchestrator
2. Поиск релевантных chunks в БД
3. Добавление контекста в system prompt
4. Отправка в LLM

### Пример использования

**User Query:**
```
What is our company's refund policy?
```

**RAG Context (автоматически добавляется):**
```
Based on internal documentation:

Chunk 1 (from "Company Policies" document):
"Our refund policy allows full refunds within 30 days of purchase..."

Chunk 2 (from "Customer FAQ" document):
"Refund requests can be submitted through the customer portal..."
```

**LLM Response:**
```
According to our company policy, we offer full refunds within 30 days...
```

## Troubleshooting

### Worker не обрабатывает jobs

**Проверить:**
```sql
-- Есть ли pending jobs?
SELECT COUNT(*) FROM rag_jobs WHERE status = 'pending';

-- Есть ли locked jobs?
SELECT id, locked_until, status FROM rag_jobs 
WHERE locked_until > NOW() ORDER BY locked_until DESC;
```

**Решение:**
```sql
-- Разблокировать expired jobs
UPDATE rag_jobs 
SET status = 'pending', locked_until = NULL 
WHERE locked_until < NOW() AND status = 'processing';
```

### API sync fails

**Проверить:**
- Корректность `url` в config
- Наличие headers для авторизации
- Формат response (JSON/text)
- Network connectivity

**Логи:**
```bash
tail -f logs/rag-worker.log | grep "API sync"
```

### Database sync fails

**Проверить:**
- `connection_string` в config
- Credentials (зашифрованы корректно)
- SQL query syntax
- Database connectivity

**Тест connection:**
```bash
psql -h host -U username -d database -c "SELECT 1;"
```

### Web scraping fails

**Проверить:**
- URL доступность
- HTTP status code
- User-Agent блокировка
- Rate limiting

**Тест:**
```bash
curl -A "Mozilla/5.0" https://example.com/page
```

## Roadmap

### v1.14.0 (В разработке)
- ✅ API Sync реализован
- ✅ Database Sync реализован
- ✅ Web Scraping реализован
- ✅ Chunking strategies (fixed, sentence, paragraph)
- ⏳ Embeddings generation (OpenAI, HuggingFace)
- ⏳ Vector search (pgvector)
- ⏳ File upload support (PDF, DOCX, TXT)

### v1.15.0 (Планируется)
- Scheduled sync (cron-like)
- Incremental sync (delta detection)
- Multi-language support
- Advanced HTML parsing (Readability.js)
- OAuth2 support для API sources

## Ссылки

- [RAG Overview](../README.md#rag-retrieval-augmented-generation)
- [Configuration Guide](../configs/README.md)
- [API Documentation](./API.md)
- [Database Schema](./DATABASE_SCHEMA.md)

