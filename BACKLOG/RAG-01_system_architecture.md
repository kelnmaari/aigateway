# RAG-01: RAG System Architecture & Implementation

**Version:** 1.13.0  
**Priority:** HIGH  
**Estimated Time:** 60-80 hours  
**Status:** 📋 Planned  
**Dependencies:** v1.10.0 (FILE-01, VISION-01), PostgreSQL, Ollama

---

## 🎯 Overview

Реализация полноценной RAG (Retrieval-Augmented Generation) системы для обогащения контекста LLM моделей релевантной информацией из различных источников данных.

### Ключевые возможности

- 📄 **Multi-format файлы**: PDF, DOCX, CSV, TXT, изображения
- 🔌 **Внешние API**: REST endpoints с Bearer token аутентификацией
- 🗄️ **Database queries**: PostgreSQL с параметризованными запросами
- 🌐 **Web scraping**: Извлечение контента из веб-страниц
- 🧠 **Smart chunking**: Семантическое разбиение с учетом контекста
- 🔍 **Vector search**: Быстрый поиск релевантных чанков
- ⚡ **Async processing**: Фоновая обработка больших файлов
- 🎨 **UI-first approach**: Управление источниками через WebUI

---

## 🏗️ Архитектура

### Компоненты системы

```
┌─────────────────────────────────────────────────────────────────┐
│                         WebUI Chat                              │
│  ┌──────────────┐  ┌─────────────────┐  ┌──────────────────┐   │
│  │  File Upload │  │ Data Sources    │  │  RAG Panel       │   │
│  │  Drag & Drop │  │  - Files        │  │  - Select sources│   │
│  │              │  │  - APIs         │  │  - Preview       │   │
│  │              │  │  - Databases    │  │  - Settings      │   │
│  └──────────────┘  └─────────────────┘  └──────────────────┘   │
└───────────────────────────┬─────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    RAG Orchestrator                             │
│  - Query understanding & routing                                │
│  - Source selection & ranking                                   │
│  - Context assembly (max 64k of 128k)                          │
│  - Prompt augmentation                                          │
└──────┬──────────────┬───────────────┬──────────────┬───────────┘
       │              │               │              │
       ▼              ▼               ▼              ▼
┌──────────┐  ┌──────────────┐  ┌─────────────┐  ┌──────────┐
│  Vector  │  │ File Storage │  │  External   │  │   Job    │
│  Search  │  │              │  │   APIs      │  │  Queue   │
│          │  │ - Local FS   │  │             │  │          │
│ pgvector │  │ - MinIO/S3   │  │ - REST      │  │ memory   │
│ or       │  │              │  │ - PostgreSQL│  │ or       │
│ Qdrant   │  │              │  │             │  │ postgres │
└──────────┘  └──────────────┘  └─────────────┘  └──────────┘
       ▲              ▲               ▲              │
       │              │               │              │
       └──────────────┴───────────────┴──────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                  Document Processing Pipeline                   │
│                                                                 │
│  ┌─────────────┐   ┌──────────────┐   ┌──────────────────┐    │
│  │  Extractors │ → │   Chunker    │ → │    Embedder      │    │
│  │             │   │              │   │                  │    │
│  │ - PDF       │   │ - Semantic   │   │ - nomic-embed    │    │
│  │ - DOCX      │   │ - Fixed-size │   │ - bge-m3         │    │
│  │ - CSV       │   │ - Overlap    │   │ - Batch: 32      │    │
│  │ - Images    │   │              │   │ - 2x GPU (4090)  │    │
│  │ - HTML      │   │              │   │                  │    │
│  └─────────────┘   └──────────────┘   └──────────────────┘    │
│                                                                 │
│  Worker Pool: 4 workers (async processing)                     │
└─────────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                      PostgreSQL Database                        │
│                                                                 │
│  - rag_data_sources (sources metadata)                         │
│  - rag_documents (files, status)                               │
│  - rag_chunks (text, tokens, embeddings)                       │
│  - rag_jobs (async queue)                                      │
│  - rag_query_logs (analytics)                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 📊 Database Schema

### 1. RAG Data Sources

```sql
CREATE TABLE rag_data_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- Основная информация
    name VARCHAR(255) NOT NULL,
    description TEXT,
    source_type VARCHAR(50) NOT NULL,  -- 'file', 'api', 'database', 'web'
    
    -- Конфигурация (JSON для гибкости)
    config JSONB NOT NULL,
    
    -- Credentials (зашифрованные AES-256)
    credentials_encrypted TEXT,
    
    -- Статус и метрики
    status VARCHAR(20) DEFAULT 'active',
    last_sync_at TIMESTAMP,
    last_sync_status VARCHAR(20),  -- 'success', 'failed', 'partial'
    last_error TEXT,
    sync_frequency INTERVAL,
    
    -- Настройки индексации
    indexing_config JSONB,
    
    -- Статистика
    total_chunks INT DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    last_chunk_count INT,
    
    -- Метаданные
    tags TEXT[],
    is_shared BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT valid_source_type CHECK (source_type IN ('file', 'api', 'database', 'web'))
);

CREATE INDEX idx_rag_sources_user ON rag_data_sources(user_id);
CREATE INDEX idx_rag_sources_tenant ON rag_data_sources(tenant_id);
CREATE INDEX idx_rag_sources_type ON rag_data_sources(source_type);
CREATE INDEX idx_rag_sources_status ON rag_data_sources(status);
CREATE INDEX idx_rag_sources_tags ON rag_data_sources USING GIN(tags);
```

### 2. RAG Documents

```sql
CREATE TABLE rag_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL REFERENCES rag_data_sources(id) ON DELETE CASCADE,
    
    -- Файл информация
    filename VARCHAR(500),
    mime_type VARCHAR(100),
    size_bytes BIGINT,
    
    -- Хранилище
    storage_backend VARCHAR(20),  -- 'local', 's3'
    storage_path TEXT NOT NULL,
    storage_bucket VARCHAR(255),  -- для S3
    
    -- Обработка
    status VARCHAR(20) DEFAULT 'pending',  -- 'pending', 'processing', 'completed', 'failed'
    processing_started_at TIMESTAMP,
    processing_completed_at TIMESTAMP,
    processing_error TEXT,
    
    -- Метаданные документа
    metadata JSONB,
    
    -- Статистика
    total_chunks INT DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_rag_docs_source ON rag_documents(source_id);
CREATE INDEX idx_rag_docs_status ON rag_documents(status);
```

### 3. RAG Chunks (с векторами)

```sql
-- Включаем pgvector расширение
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE rag_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES rag_documents(id) ON DELETE CASCADE,
    source_id UUID NOT NULL REFERENCES rag_data_sources(id) ON DELETE CASCADE,
    
    -- Chunk контент
    chunk_text TEXT NOT NULL,
    chunk_index INT NOT NULL,  -- позиция в документе
    chunk_tokens INT NOT NULL,
    
    -- Embedding (768 dimensions для nomic-embed-text)
    embedding vector(768),
    
    -- Метаданные чанка
    metadata JSONB,  -- page_number, headers, context, etc.
    
    -- Для overlap detection
    start_offset INT,
    end_offset INT,
    
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_rag_chunks_document ON rag_chunks(document_id);
CREATE INDEX idx_rag_chunks_source ON rag_chunks(source_id);

-- HNSW индекс для быстрого vector search (cosine distance)
CREATE INDEX idx_rag_chunks_embedding ON rag_chunks 
USING hnsw (embedding vector_cosine_ops)
WITH (m = 16, ef_construction = 64);

-- Альтернативно IVFFlat для больших объемов:
-- CREATE INDEX idx_rag_chunks_embedding ON rag_chunks 
-- USING ivfflat (embedding vector_cosine_ops)
-- WITH (lists = 100);
```

### 4. RAG Jobs Queue

```sql
CREATE TABLE rag_jobs (
    id BIGSERIAL PRIMARY KEY,
    job_type VARCHAR(50) NOT NULL,  -- 'file_upload', 'api_sync', 'db_query', 'web_scrape'
    status VARCHAR(20) DEFAULT 'pending',  -- 'pending', 'processing', 'completed', 'failed'
    
    -- Данные
    payload JSONB NOT NULL,
    result JSONB,
    
    -- Приоритет и повторы
    priority INT DEFAULT 0,
    attempts INT DEFAULT 0,
    max_attempts INT DEFAULT 3,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    -- Ошибки
    error TEXT,
    
    -- Для visibility timeout
    locked_until TIMESTAMP
);

CREATE INDEX idx_rag_jobs_status ON rag_jobs(status, priority DESC, created_at);
CREATE INDEX idx_rag_jobs_type ON rag_jobs(job_type);
```

### 5. RAG Query Logs (аналитика)

```sql
CREATE TABLE rag_query_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    conversation_id UUID REFERENCES conversations(id),
    
    -- Запрос
    query_text TEXT NOT NULL,
    
    -- Использованные источники
    source_ids UUID[],
    
    -- Результаты поиска
    chunks_retrieved INT,
    chunks_used INT,
    
    -- Метрики
    search_time_ms INT,
    total_tokens_used INT,
    
    -- Результат
    response_quality_score FLOAT,  -- опционально, от пользователя
    
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_rag_query_logs_user ON rag_query_logs(user_id);
CREATE INDEX idx_rag_query_logs_created ON rag_query_logs(created_at DESC);
```

---

## 🔧 Configuration

### config.yaml

```yaml
rag:
  enabled: true
  
  # Vector Store
  vector_store:
    backend: "pgvector"  # или "qdrant"
    
    pgvector:
      dimensions: 768  # nomic-embed-text
      index_type: "hnsw"  # или "ivfflat"
      distance_metric: "cosine"
      hnsw_m: 16
      hnsw_ef_construction: 64
    
    qdrant:
      url: "http://localhost:6333"
      collection: "ollama_rag"
      prefer_grpc: true
      timeout: 30s
  
  # File Storage
  file_storage:
    backend: "local"  # или "s3"
    
    local:
      base_path: "./data/rag/uploads"
      max_file_size: "100MB"
      allowed_extensions: [".pdf", ".docx", ".txt", ".csv", ".xlsx", ".png", ".jpg"]
    
    s3:
      endpoint: "http://minio:9000"
      bucket: "rag-documents"
      access_key: "${MINIO_ACCESS_KEY}"
      secret_key: "${MINIO_SECRET_KEY}"
      use_ssl: false
      region: "us-east-1"
  
  # Embeddings
  embeddings:
    provider: "ollama"
    model: "nomic-embed-text"  # или "bge-m3"
    dimensions: 768
    batch_size: 32
    timeout: 60s
    
    # Если несколько GPU
    parallel_workers: 2
  
  # Document Processing
  processing:
    # Chunking
    chunk_size: 512  # tokens
    chunk_overlap: 50  # tokens (10%)
    chunking_strategy: "semantic"  # или "fixed", "hierarchical"
    
    # Extractors
    pdf:
      ocr_enabled: true
      ocr_lang: "rus+eng"
    
    docx:
      extract_images: true
      extract_tables: true
    
    csv:
      max_rows: 100000
      encoding: "utf-8"
      delimiter: ","
    
    images:
      vision_model: "llava"  # для OCR и описания
      max_size: "10MB"
  
  # Search & Retrieval
  retrieval:
    # Vector search
    top_k: 20  # сколько чанков извлекать
    similarity_threshold: 0.7  # минимальная схожесть
    
    # Reranking
    rerank_enabled: true
    rerank_top_n: 5
    rerank_model: "llama3.1:8b"  # использовать LLM для rerank
    
    # Context assembly
    max_context_tokens: 65536  # 50% от 128k
    context_strategy: "dynamic"  # или "fixed"
  
  # Job Queue
  queue:
    backend: "postgres"  # или "memory"
    
    memory:
      buffer_size: 1000
      num_workers: 4
    
    postgres:
      poll_interval: "1s"
      visibility_timeout: "5m"
      max_attempts: 3
      num_workers: 4
      cleanup_completed_after: "24h"
  
  # External APIs
  api_sources:
    default_timeout: 30s
    max_retries: 3
    rate_limit_per_source: 100  # requests per minute
  
  # Database Sources
  db_sources:
    max_connections_per_source: 5
    query_timeout: 30s
    max_rows: 10000
  
  # Security
  security:
    encrypt_credentials: true
    encryption_key: "${RAG_ENCRYPTION_KEY}"  # AES-256
    allowed_db_drivers: ["postgresql"]  # безопасность
    allowed_api_domains: []  # пустой = все разрешены
```

---

## 📡 API Endpoints

### Data Sources Management

```http
# Создать источник данных
POST /api/rag/sources
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
  "name": "CRM Database",
  "description": "Customer database with balance information",
  "source_type": "database",
  "config": {
    "driver": "postgresql",
    "host": "db.company.local",
    "port": 5432,
    "database": "crm",
    "ssl_mode": "require",
    "query_template": "SELECT * FROM customers WHERE balance > {{min_balance}}",
    "parameters": [
      {"name": "min_balance", "type": "integer", "default": 10000}
    ]
  },
  "credentials": {
    "username": "readonly_user",
    "password": "secure_password"
  },
  "sync_frequency": "6h",
  "indexing_config": {
    "chunk_strategy": "row_based",
    "embed_columns": ["name", "email", "notes"]
  },
  "tags": ["crm", "customers", "database"],
  "is_shared": false
}

Response 201:
{
  "id": "uuid",
  "name": "CRM Database",
  "source_type": "database",
  "status": "active",
  "created_at": "2025-01-15T10:00:00Z"
}
```

```http
# Список источников
GET /api/rag/sources?source_type=database&tags=crm&limit=20&offset=0

Response 200:
{
  "sources": [
    {
      "id": "uuid",
      "name": "CRM Database",
      "description": "Customer database...",
      "source_type": "database",
      "status": "active",
      "last_sync_at": "2025-01-15T09:00:00Z",
      "last_sync_status": "success",
      "total_chunks": 1240,
      "total_tokens": 450000,
      "tags": ["crm", "customers"],
      "created_at": "2025-01-10T10:00:00Z"
    }
  ],
  "total": 5,
  "limit": 20,
  "offset": 0
}
```

```http
# Тестировать подключение
POST /api/rag/sources/test-connection

{
  "source_type": "database",
  "config": {...},
  "credentials": {...}
}

Response 200:
{
  "success": true,
  "message": "Connection successful",
  "details": {
    "response_time_ms": 245,
    "row_count": 124,
    "sample_data": "customer_id | name | email\n1001 | Ivan | ivan@..."
  }
}
```

```http
# Запустить синхронизацию вручную
POST /api/rag/sources/{id}/sync

Response 202:
{
  "job_id": 12345,
  "message": "Sync job queued",
  "estimated_time": "5m"
}
```

```http
# Получить детали источника
GET /api/rag/sources/{id}

# Обновить источник
PUT /api/rag/sources/{id}

# Удалить источник
DELETE /api/rag/sources/{id}
```

### RAG Query API

```http
# Поиск релевантных чанков (для тестирования)
POST /api/rag/search
Content-Type: application/json

{
  "query": "клиенты с балансом выше 10000",
  "source_ids": ["uuid1", "uuid2"],  // опционально
  "top_k": 10,
  "filters": {
    "tags": ["crm"],
    "source_type": "database"
  }
}

Response 200:
{
  "chunks": [
    {
      "id": "chunk-uuid",
      "source_id": "source-uuid",
      "source_name": "CRM Database",
      "text": "customer_id: 1001, name: Ivan Petrov, balance: 15000...",
      "similarity_score": 0.89,
      "metadata": {
        "row_index": 1001,
        "columns": ["customer_id", "name", "balance"]
      }
    }
  ],
  "total_found": 124,
  "search_time_ms": 45
}
```

### Chat with RAG

```http
# Обычный chat endpoint, но с RAG параметрами
POST /api/v1/chat/completions
Content-Type: application/json

{
  "model": "llama3.1:8b",
  "messages": [
    {
      "role": "user",
      "content": "У меня в CSV есть клиенты у которых баланс выше 10000 рублей, дай мне список"
    }
  ],
  
  // RAG специфичные параметры
  "rag": {
    "enabled": true,
    "source_ids": ["csv-source-uuid"],  // какие источники использовать
    "auto_select": false,  // или авто-выбор на основе query
    "top_k": 10,
    "rerank": true,
    "include_sources_in_response": true  // показать какие чанки использовались
  }
}

Response 200:
{
  "id": "chatcmpl-123",
  "model": "llama3.1:8b",
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "Найдено 8 клиентов с балансом выше 10000₽:\n\n1. Иван Петров (15,000₽)..."
      },
      "finish_reason": "stop"
    }
  ],
  
  // RAG метаданные
  "rag_metadata": {
    "sources_used": [
      {
        "source_id": "uuid",
        "source_name": "customers.csv",
        "chunks_retrieved": 10,
        "chunks_used": 5
      }
    ],
    "total_context_tokens": 3450,
    "search_time_ms": 45
  }
}
```

---

## 📦 Package Structure

```
internal/rag/
├── config/
│   └── config.go              # RAG конфигурация
├── storage/
│   ├── interface.go           # FileStorage interface
│   ├── local.go              # Локальное хранилище
│   └── s3.go                 # S3/MinIO
├── vector/
│   ├── interface.go           # VectorStore interface
│   ├── pgvector.go           # PostgreSQL pgvector
│   └── qdrant.go             # Qdrant client
├── embeddings/
│   ├── interface.go           # Embedder interface
│   ├── ollama.go             # Ollama embeddings
│   └── batch.go              # Batch processing
├── extractors/
│   ├── interface.go           # Extractor interface
│   ├── pdf.go                # PDF extraction
│   ├── docx.go               # DOCX extraction
│   ├── csv.go                # CSV parsing
│   ├── image.go              # Image OCR
│   └── html.go               # HTML parsing
├── chunker/
│   ├── interface.go           # Chunker interface
│   ├── semantic.go           # Semantic chunking
│   ├── fixed.go              # Fixed-size chunking
│   └── hierarchical.go       # Hierarchical chunking
├── sources/
│   ├── interface.go           # DataSource interface
│   ├── file.go               # File source
│   ├── api.go                # REST API source
│   ├── database.go           # PostgreSQL source
│   └── web.go                # Web scraping
├── queue/
│   ├── interface.go           # Queue interface
│   ├── memory.go             # In-memory queue
│   └── postgres.go           # PostgreSQL queue
├── processor/
│   ├── pipeline.go           # Document processing pipeline
│   └── worker.go             # Worker pool
├── retrieval/
│   ├── searcher.go           # Vector search
│   ├── reranker.go           # Reranking logic
│   └── assembler.go          # Context assembly
├── orchestrator/
│   └── orchestrator.go       # Main RAG orchestrator
└── service/
    └── service.go            # High-level RAG service
```

---

## 🎯 Implementation Plan

### Phase 1: Foundation (15-20 hours) → v1.13.1

**Tasks:**

- ✅ Database migrations (tables)
- ✅ Basic data sources CRUD API
- ✅ File storage (Local + S3)
- ✅ Simple in-memory queue
- ✅ WebUI: Data Sources management page

**Deliverables:**

- Можно создавать/редактировать источники через UI
- Файлы сохраняются в Local/S3
- Basic API endpoints работают

---

### Phase 2: Document Processing (15-20 hours) → v1.13.2

**Tasks:**

- ✅ PDF extractor (pypdf2 или pdfplumber через exec)
- ✅ DOCX extractor (docx library)
- ✅ CSV parser
- ✅ Semantic chunker
- ✅ Async job queue (PostgreSQL)
- ✅ Worker pool implementation

**Deliverables:**

- Загрузка файлов → автоматическая обработка
- Файлы разбиваются на чанки
- Прогресс видно в UI

---

### Phase 3: Embeddings & Vector Search (12-16 hours) → v1.13.3

**Tasks:**

- ✅ Ollama embeddings integration
- ✅ pgvector setup + migrations
- ✅ Batch embedding processing
- ✅ Vector search API
- ✅ HNSW indexing

**Deliverables:**

- Чанки эмбедятся автоматически
- Работает vector search
- Можно тестировать поиск через API

---

### Phase 4: External Sources (10-14 hours) → v1.13.4

**Tasks:**

- ✅ REST API source implementation
- ✅ PostgreSQL database source
- ✅ Credentials encryption (AES-256)
- ✅ Connection testing
- ✅ Scheduled sync

**Deliverables:**

- Можно подключать внешние API
- Можно подключать PostgreSQL БД
- Автоматическая синхронизация работает

---

### Phase 5: RAG Integration (8-10 hours) → v1.13.5

**Tasks:**

- ✅ RAG Orchestrator
- ✅ Reranking logic
- ✅ Context assembly
- ✅ Chat API integration
- ✅ WebUI: Source selector in chat

**Deliverables:**

- RAG работает end-to-end
- Можно выбрать источники в чате
- Релевантный контекст добавляется в промпт

---

## 🧪 Testing Strategy

### Unit Tests (>90% coverage)

```go
// internal/rag/chunker/semantic_test.go
func TestSemanticChunker_ChunkDocument(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        maxSize  int
        expected int  // число чанков
    }{
        {"simple text", "Hello world", 512, 1},
        {"large text", strings.Repeat("text ", 1000), 512, 5},
    }
    // ...
}

// internal/rag/embeddings/ollama_test.go (with mock)
func TestOllamaEmbedder_Embed(t *testing.T) {
    mock := &MockOllamaClient{}
    mock.On("Embed", mock.Anything).Return([]float32{0.1, 0.2}, nil)
    // ...
}
```

### Integration Tests

```go
// internal/rag/integration_test.go
func TestRAGPipeline_EndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip()
    }
    
    // Upload file → Process → Embed → Search
    // ...
}
```

### Performance Tests

```go
// internal/rag/vector/pgvector_bench_test.go
func BenchmarkVectorSearch(b *testing.B) {
    // Benchmark с 1M векторов
}
```

---

## 📊 Success Metrics

- ✅ Можно загрузить 100MB PDF и обработать за <5 минут
- ✅ Vector search <100ms на 1M векторов
- ✅ Точность поиска >80% (precision@5)
- ✅ Поддержка минимум 4 типов источников
- ✅ WebUI интуитивный (<5 минут на создание источника)
- ✅ 100% test coverage критичных компонентов
- ✅ Документация API complete

---

## 🔒 Security Considerations

1. **Credentials encryption**: AES-256 для паролей и токенов
2. **Input validation**: Все user inputs валидируются
3. **SQL injection prevention**: Только prepared statements
4. **File upload limits**: Размер + тип файлов
5. **Rate limiting**: На external API calls
6. **Access control**: Per-user/per-tenant isolation
7. **Audit logging**: Все RAG queries логируются

---

## 📚 Dependencies

### Go Libraries

```go
// File processing
"github.com/ledongthuc/pdf"              // PDF extraction
"github.com/nguyenthenguyen/docx"        // DOCX parsing
"encoding/csv"                            // CSV (stdlib)

// Vector operations
"github.com/pgvector/pgvector-go"        // pgvector client

// Embeddings
"github.com/ollama/ollama/api"           // Ollama API

// Storage
"github.com/minio/minio-go/v7"           // S3/MinIO client

// Encryption
"crypto/aes"                              // AES-256 (stdlib)
"golang.org/x/crypto/bcrypt"              // Key derivation

// HTML parsing
"github.com/PuerkitoBio/goquery"         // Web scraping
```

### External Services

- **PostgreSQL** with pgvector extension
- **Ollama** for embeddings (nomic-embed-text или bge-m3)
- **MinIO** (опционально для S3 storage)

---

## 📖 Documentation

- [ ] User Guide: "How to use RAG"
- [ ] Admin Guide: "Setting up data sources"
- [ ] API Documentation (Swagger/OpenAPI)
- [ ] Architecture decision records (ADR)
- [ ] Troubleshooting guide

---

## 🚀 Future Enhancements (v1.14+)

- **Qdrant migration** для >10M векторов
- **Web scraping** с follow links
- **GraphQL APIs** support
- **MongoDB** data source
- **Hybrid search** (keyword + vector)
- **Multi-language support** (переключение embedding моделей)
- **Query caching** для популярных запросов
- **Analytics dashboard** для RAG usage

---

**Created:** 2025-01-15  
**Last Updated:** 2025-01-15  
**Status:** Ready for Implementation
