# RAG System Deployment Guide

> **Version:** 1.13.0+  
> **Статус:** v1.13.1 Foundation Complete | v1.13.2-v1.13.5 In Development  
> **Последнее обновление:** 2025-10-26

## 📋 Оглавление

- [Обзор](#обзор)
- [Компоненты системы](#компоненты-системы)
- [Минимальные требования](#минимальные-требования)
- [Установка компонентов](#установка-компонентов)
- [Конфигурация](#конфигурация)
- [Миграции БД](#миграции-бд)
- [Тестирование](#тестирование)
- [Production Deployment](#production-deployment)
- [Мониторинг и Troubleshooting](#мониторинг-и-troubleshooting)

---

## Обзор

RAG (Retrieval-Augmented Generation) система состоит из 5 основных фаз развертывания:

| Фаза | Версия | Компоненты | Статус |
|------|--------|------------|--------|
| **v1.13.1** | Foundation | Database, Data Sources API, Queue | ✅ Complete |
| **v1.13.2** | Processing | Document extractors, Chunking, Workers | 📋 Planned |
| **v1.13.3** | Embeddings | Ollama embeddings, pgvector, Search API | 📋 Planned |
| **v1.13.4** | External Sources | REST API, PostgreSQL, Credentials encryption | 📋 Planned |
| **v1.13.5** | Integration | RAG Orchestrator, Chat UI, Reranking | 📋 Planned |

---

## Компоненты системы

### v1.13.1 Foundation (Current)

#### 1. **Database** (PostgreSQL или SQLite)
- **Назначение:** Хранение data sources, documents, chunks, jobs queue
- **Требования:**
  - PostgreSQL 12+ ИЛИ SQLite 3.35+
  - Минимум 1GB свободного места
  - Поддержка JSON (SQLite) или JSONB (PostgreSQL)

#### 2. **Ollama Server**
- **Назначение:** Базовый LLM сервер (будет использован для embeddings в v1.13.3)
- **Требования:**
  - Ollama v0.1.0+
  - Минимум 8GB RAM
  - Рекомендуется GPU (NVIDIA с CUDA)

#### 3. **Proxy Server**
- **Назначение:** Основной сервер с RAG системой
- **Требования:**
  - Go 1.25+
  - 4GB RAM минимум
  - CPU: 2+ cores

---

## Минимальные требования

### Development Environment

```
Hardware:
- CPU: 4 cores
- RAM: 8GB
- Disk: 10GB free space

Software:
- Go 1.25+
- PostgreSQL 12+ ИЛИ SQLite 3.35+
- Ollama v0.1.0+
- (Опционально) Docker + Docker Compose
```

### Production Environment

```
Hardware:
- CPU: 8+ cores
- RAM: 16GB+ (32GB для large documents)
- Disk: 50GB+ free space
- GPU: NVIDIA с 8GB+ VRAM (для embeddings)

Software:
- Go 1.25+
- PostgreSQL 14+ (рекомендуется)
- Ollama v0.1.0+
- (Опционально) MinIO для S3-compatible storage
- (Рекомендуется) Redis для distributed caching
```

---

## Установка компонентов

### 1. PostgreSQL Setup

#### Option A: Native Installation

**Ubuntu/Debian:**
```bash
# Install PostgreSQL 14
sudo apt update
sudo apt install postgresql-14 postgresql-contrib-14

# Start service
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Create database and user
sudo -u postgres psql << EOF
CREATE DATABASE ollama_proxy;
CREATE USER ollama_user WITH ENCRYPTED PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE ollama_proxy TO ollama_user;
EOF
```

**Docker:**
```bash
docker run -d \
  --name postgres \
  -e POSTGRES_PASSWORD=secure_password \
  -e POSTGRES_USER=ollama_user \
  -e POSTGRES_DB=ollama_proxy \
  -p 5432:5432 \
  -v postgres_data:/var/lib/postgresql/data \
  postgres:14-alpine
```

#### Option B: SQLite (Development Only)

```bash
# SQLite included в Go - настройка не требуется
# Database файл создастся автоматически: ./data/proxy.db
```

### 2. Ollama Setup

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Start Ollama service
ollama serve

# Download embedding model (для v1.13.3+)
ollama pull nomic-embed-text

# (Опционально) Download LLM models
ollama pull llama3.1:8b
ollama pull llama3.2-vision:11b  # для image OCR
```

### 3. Proxy Server Build

```bash
# Clone repository
git clone https://github.com/your-org/ollama-proxy.git
cd ollama-proxy

# Build server
go build -o bin/server cmd/server/main.go

# (Опционально) Install systemd service
sudo cp scripts/ollama-proxy.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable ollama-proxy
```

---

## Конфигурация

### 1. Base Configuration (configs/production.yaml)

```yaml
# Server settings
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "30s"

# Ollama connection
ollama:
  url: "http://localhost:11434"
  timeout: "60s"
  retry_attempts: 3

# Database (PostgreSQL)
database:
  type: "postgresql"
  postgresql:
    host: "localhost"
    port: 5432
    user: "ollama_user"
    password: "${DB_PASSWORD}"  # from environment
    database: "ollama_proxy"
    ssl_mode: "require"
    max_connections: 25
    max_idle_connections: 5
    connection_lifetime: "1h"

# Authentication
auth:
  enabled: true
  jwt:
    secret: "${JWT_SECRET}"  # 32+ characters
    access_token_expiry: "15m"
    refresh_token_expiry: "7d"
```

### 2. RAG Configuration (v1.13.0+)

```yaml
# RAG System Configuration
rag:
  enabled: true
  
  # Vector Store (v1.13.3+)
  vector_store:
    backend: "pgvector"  # или "qdrant"
    pgvector:
      dimensions: 768  # nomic-embed-text
      index_type: "hnsw"
      distance_metric: "cosine"
      hnsw_m: 16
      hnsw_ef_construction: 64
  
  # File Storage
  file_storage:
    backend: "local"  # или "s3"
    local:
      base_path: "./data/rag/uploads"
      max_file_size: "100MB"
      allowed_extensions: [".pdf", ".docx", ".txt", ".csv", ".png", ".jpg"]
    s3:  # для production
      endpoint: "http://minio:9000"
      bucket: "rag-documents"
      access_key: "${MINIO_ACCESS_KEY}"
      secret_key: "${MINIO_SECRET_KEY}"
  
  # Embeddings (v1.13.3+)
  embeddings:
    provider: "ollama"
    model: "nomic-embed-text"
    dimensions: 768
    batch_size: 32
    timeout: "60s"
    parallel_workers: 2  # если 2 GPU
  
  # Document Processing (v1.13.2+)
  processing:
    chunk_size: 512  # tokens
    chunk_overlap: 50  # tokens (10%)
    chunking_strategy: "semantic"
    
    pdf:
      ocr_enabled: true
      ocr_lang: "rus+eng"
    
    docx:
      extract_images: true
      extract_tables: true
    
    csv:
      max_rows: 100000
      encoding: "utf-8"
  
  # Retrieval (v1.13.5+)
  retrieval:
    top_k: 20
    similarity_threshold: 0.7
    rerank_enabled: true
    rerank_top_n: 5
    rerank_model: "llama3.1:8b"
    max_context_tokens: 65536  # 50% от 128k context
  
  # Job Queue
  queue:
    backend: "postgres"  # или "memory" для development
    postgres:
      poll_interval: "1s"
      visibility_timeout: "5m"
      max_attempts: 3
      num_workers: 4
  
  # Security
  security:
    encrypt_credentials: true
    encryption_key: "${RAG_ENCRYPTION_KEY}"  # 32 bytes для AES-256
    allowed_db_drivers: ["postgresql"]
```

### 3. Environment Variables

```bash
# Create .env file
cat > .env << 'EOF'
# Database
DB_PASSWORD=secure_db_password

# JWT
JWT_SECRET=your_32_char_secret_key_here_12345

# RAG
RAG_ENCRYPTION_KEY=32_byte_aes_key_1234567890123456

# Optional: MinIO/S3
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# Optional: External services
QDRANT_URL=http://localhost:6333
EOF

# Load environment
export $(cat .env | xargs)
```

---

## Миграции БД

### Automatic Migration (Recommended)

```bash
# Миграции применяются автоматически при старте сервера
./bin/server -config configs/production.yaml

# Server logs:
# INFO: Starting database migration
# INFO: Applying migration version=51 name=create_rag_tables
# INFO: Migration applied successfully
# INFO: Database migration completed version=51
```

### Manual Migration Check

```bash
# PostgreSQL - проверить версию миграции
psql -U ollama_user -d ollama_proxy -c "SELECT MAX(version) FROM migrations;"

# SQLite - проверить версию миграции
sqlite3 ./data/proxy.db "SELECT MAX(version) FROM migrations;"

# Ожидаемая версия для v1.13.1: 51 (SQLite) или 3 (PostgreSQL)
```

### Migration Rollback (Emergency)

```bash
# ВНИМАНИЕ: Rollback удалит все RAG данные!

# PostgreSQL
psql -U ollama_user -d ollama_proxy << EOF
DROP TABLE IF EXISTS rag_query_logs;
DROP TABLE IF EXISTS rag_jobs;
DROP TABLE IF EXISTS rag_chunks;
DROP TABLE IF EXISTS rag_documents;
DROP TABLE IF EXISTS rag_data_sources;
DELETE FROM migrations WHERE version >= 3;  -- RAG migrations start at v3
EOF

# SQLite
sqlite3 ./data/proxy.db << EOF
DROP TABLE IF EXISTS rag_query_logs;
DROP TABLE IF EXISTS rag_jobs;
DROP TABLE IF EXISTS rag_chunks;
DROP TABLE IF EXISTS rag_documents;
DROP TABLE IF EXISTS rag_data_sources;
DELETE FROM migrations WHERE version >= 51;
EOF
```

---

## Тестирование

### 1. Health Check

```bash
# Server health
curl http://localhost:8080/health

# Expected response:
# {"status":"ok","ollama":"connected","database":"connected"}
```

### 2. RAG API Test (v1.13.1+)

```bash
# Authenticate
TOKEN=$(curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' \
  | jq -r '.access_token')

# Create test data source
curl -X POST http://localhost:8080/api/rag/sources \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Documents",
    "description": "Test data source",
    "source_type": "file",
    "config": {
      "storage_backend": "local"
    },
    "tags": ["test"]
  }'

# Expected response:
# {"id":"uuid","name":"Test Documents","source_type":"file","status":"active",...}

# List data sources
curl -X GET http://localhost:8080/api/rag/sources \
  -H "Authorization: Bearer $TOKEN"
```

### 3. End-to-End Test (v1.13.5+ when complete)

```bash
# Upload document
FILE_ID=$(curl -X POST http://localhost:8080/api/files/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.pdf" \
  | jq -r '.id')

# Wait for processing...
sleep 5

# Query with RAG
curl -X POST http://localhost:8080/api/v1/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3.1:8b",
    "messages": [{"role":"user","content":"What is in the uploaded document?"}],
    "rag": {
      "enabled": true,
      "source_ids": ["'$SOURCE_ID'"],
      "top_k": 5
    }
  }'
```

---

## Production Deployment

### Docker Compose Setup

```yaml
# docker-compose.yml
version: '3.8'

services:
  postgres:
    image: postgres:14-alpine
    environment:
      POSTGRES_DB: ollama_proxy
      POSTGRES_USER: ollama_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ollama_user"]
      interval: 10s
      timeout: 5s
      retries: 5
  
  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama_data:/root/.ollama
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
  
  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ACCESS_KEY}
      MINIO_ROOT_PASSWORD: ${MINIO_SECRET_KEY}
    volumes:
      - minio_data:/data
    ports:
      - "9000:9000"
      - "9001:9001"
  
  proxy:
    build: .
    depends_on:
      - postgres
      - ollama
      - minio
    environment:
      DB_PASSWORD: ${DB_PASSWORD}
      JWT_SECRET: ${JWT_SECRET}
      RAG_ENCRYPTION_KEY: ${RAG_ENCRYPTION_KEY}
    ports:
      - "8080:8080"
    volumes:
      - ./configs:/app/configs
      - ./data:/app/data

volumes:
  postgres_data:
  ollama_data:
  minio_data:
```

### Kubernetes Deployment (Advanced)

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ollama-proxy-rag
spec:
  replicas: 3
  selector:
    matchLabels:
      app: ollama-proxy
  template:
    metadata:
      labels:
        app: ollama-proxy
    spec:
      containers:
      - name: proxy
        image: your-registry/ollama-proxy:v1.13.0
        env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: proxy-secrets
              key: db-password
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: proxy-secrets
              key: jwt-secret
        ports:
        - containerPort: 8080
        resources:
          requests:
            memory: "4Gi"
            cpu: "2"
          limits:
            memory: "8Gi"
            cpu: "4"
```

---

## Мониторинг и Troubleshooting

### Logs

```bash
# Server logs
./bin/server -config configs/production.yaml 2>&1 | tee logs/server.log

# Tail logs
tail -f logs/server.log | grep RAG

# Important log patterns:
# "RAG System включен" - RAG initialized successfully
# "RAG Data Source Service initialized" - Service ready
# "RAG System routes configured" - Routes registered
```

### Metrics (Prometheus)

```yaml
# Metrics endpoint: http://localhost:8080/metrics

# Key metrics для RAG:
- rag_sources_total          # Total data sources
- rag_documents_total         # Total documents
- rag_chunks_total            # Total chunks indexed
- rag_jobs_pending            # Pending jobs in queue
- rag_jobs_processing         # Jobs being processed
- rag_query_duration_seconds  # Query latency
- rag_embedding_duration_seconds  # Embedding latency
```

### Common Issues

#### 1. Migration Failed

```bash
# Symptom: Server fails to start with migration error
# Solution: Check database connection and permissions

# Verify connection
psql -U ollama_user -d ollama_proxy -c "SELECT 1;"

# Check migration table
psql -U ollama_user -d ollama_proxy -c "SELECT * FROM migrations ORDER BY version DESC LIMIT 5;"
```

#### 2. RAG Service Not Initialized

```bash
# Symptom: "RAG System disabled - handler not initialized"
# Solution: Check config

# Verify RAG enabled in config
grep "rag:" configs/production.yaml -A 2

# Check encryption key
echo $RAG_ENCRYPTION_KEY | wc -c  # Should be 32+ characters
```

#### 3. Out of Memory (OOM)

```bash
# Symptom: Server crashes during document processing
# Solution: Adjust chunk size and worker count

# config.yaml:
processing:
  chunk_size: 256  # smaller chunks
  
queue:
  postgres:
    num_workers: 2  # fewer workers
```

---

## Next Steps

После завершения v1.13.1 Foundation:

1. **v1.13.2**: Document Processing Pipeline
   - Semantic chunking
   - Worker pool для async processing
   - Progress tracking через WebSocket

2. **v1.13.3**: Embeddings & Vector Search
   - pgvector extension setup
   - Ollama embeddings integration
   - HNSW indexing

3. **v1.13.4**: External Data Sources
   - REST API sources
   - PostgreSQL query sources
   - Scheduled sync

4. **v1.13.5**: RAG Integration
   - Chat UI integration
   - Context assembly
   - Reranking logic

---

## Support

- **Documentation:** [docs/](../README.md)
- **Issues:** [GitHub Issues](https://github.com/your-org/ollama-proxy/issues)
- **Roadmap:** [Roadmap.MD](../Roadmap.MD)

---

**Last Updated:** 2025-10-26  
**Version:** v1.13.1 Foundation Complete

