# RAG System Configuration Guide

Comprehensive guide для настройки RAG системы v1.13.0+.

## 🎛️ Enable/Disable RAG System

RAG подсистема может быть полностью включена или выключена через конфигурацию:

### YAML Config

```yaml
# configs/production.yaml
rag:
  enabled: true  # or false to disable
```

### Environment Variable

```bash
export RAG_ENABLED=true
```

### Default Behavior

```go
// internal/rag/config/config.go
func DefaultConfig() RAGConfig {
    return RAGConfig{
        Enabled: true,  // По умолчанию включена
        // ... остальные настройки
    }
}
```

---

## 📊 Full Configuration Example

See [`configs/rag-example.yaml`](../configs/rag-example.yaml) for complete configuration:

```yaml
rag:
  enabled: true
  
  vector_store:
    backend: "pgvector"
    pgvector:
      dimensions: 768
      index_type: "hnsw"
      distance_metric: "cosine"
  
  embeddings:
    provider: "ollama"
    model: "nomic-embed-text"
    dimensions: 768
    batch_size: 32
  
  processing:
    chunk_size: 512
    chunk_overlap: 50
    chunking_strategy: "semantic"
  
  retrieval:
    top_k: 20
    similarity_threshold: 0.7
    rerank_enabled: true
  
  queue:
    backend: "postgres"
  
  security:
    encrypt_credentials: true
    encryption_key: "your-32-byte-key-change-this"
```

---

## 🔧 Configuration Scenarios

### 1. Minimal Config (Disabled)

```yaml
rag:
  enabled: false
```

Все RAG endpoints остаются доступными, но возвращают соответствующие ошибки.

### 2. Development Config

```yaml
rag:
  enabled: true
  
  vector_store:
    backend: "pgvector"
    pgvector:
      dimensions: 768
  
  embeddings:
    model: "nomic-embed-text"
  
  queue:
    backend: "memory"  # In-memory для быстрого dev
  
  security:
    encrypt_credentials: false  # Для упрощения dev
```

### 3. Production Config

```yaml
rag:
  enabled: true
  
  vector_store:
    backend: "pgvector"
    pgvector:
      dimensions: 1024  # mxbai-embed-large
      index_type: "hnsw"
      hnsw_m: 16
      hnsw_ef_construction: 64
  
  embeddings:
    model: "mxbai-embed-large"
    dimensions: 1024
    batch_size: 64
    parallel_workers: 4
  
  processing:
    chunk_size: 512
    chunk_overlap: 100
    chunking_strategy: "semantic"
  
  retrieval:
    top_k: 50
    similarity_threshold: 0.8
    rerank_enabled: true
    rerank_top_n: 10
    max_context_tokens: 131072  # Для длинных контекстов
  
  queue:
    backend: "postgres"
    postgres:
      num_workers: 8
      visibility_timeout: 10m
      max_attempts: 5
  
  security:
    encrypt_credentials: true
    encryption_key: "${RAG_ENCRYPTION_KEY}"  # From env
    allowed_db_drivers: ["postgresql"]
    allowed_api_domains: ["trusted-domain.com"]
```

---

## 🚀 Runtime Behavior

### When RAG Enabled

```go
if cfg.RAG.Enabled {
    // Initialize RAG components
    ragOrchestrator := initRAGOrchestrator()
    ragDataSourceService := initRAGDataSourceService()
    
    // Register RAG routes
    router.setupRAGRoutes()
    
    // Chat handler can use RAG
    chatHandler.SetRAGOrchestrator(ragOrchestrator)
}
```

**Available:**
- ✅ `/api/rag/sources` - Data sources CRUD
- ✅ `/v1/chat/completions` with `rag_enabled: true`
- ✅ Document processing pipeline
- ✅ Vector search & retrieval

### When RAG Disabled

```go
if !cfg.RAG.Enabled {
    logger.Info("RAG system disabled")
    // RAG routes not registered
    // RAG orchestrator = nil
}
```

**Behavior:**
- ❌ RAG endpoints return 404 or appropriate error
- ❌ Chat requests with `rag_enabled: true` ignored
- ✅ Rest of the system works normally

---

## 🔐 Security Configuration

### Encryption Key

**КРИТИЧНО для production!**

```yaml
rag:
  security:
    encrypt_credentials: true
    encryption_key: "12345678901234567890123456789012"  # MUST be 32 bytes
```

**Generate secure key:**

```bash
# Linux/Mac
openssl rand -hex 16 | cut -c1-32

# Python
python3 -c "import secrets; print(secrets.token_hex(16))"

# Go
go run -c 'package main; import ("crypto/rand"; "encoding/hex"; "fmt"); func main() { b := make([]byte, 16); rand.Read(b); fmt.Println(hex.EncodeToString(b)) }'
```

### Allowed Drivers

```yaml
rag:
  security:
    allowed_db_drivers:
      - "postgresql"
    # - "mysql"  # Uncomment to allow MySQL sources
```

### Allowed API Domains

```yaml
rag:
  security:
    allowed_api_domains:
      - "api.internal.company.com"
      - "external-api.trusted.com"
    # [] = all domains allowed (⚠️ risky)
```

---

## 📈 Performance Tuning

### High Throughput

```yaml
rag:
  embeddings:
    batch_size: 128
    parallel_workers: 8
  
  queue:
    postgres:
      num_workers: 16
      poll_interval: 500ms
  
  retrieval:
    top_k: 100
    rerank_top_n: 20
```

### Low Latency

```yaml
rag:
  embeddings:
    batch_size: 16
    parallel_workers: 2
  
  retrieval:
    top_k: 10
    rerank_top_n: 5
    similarity_threshold: 0.85  # More selective
  
  queue:
    backend: "memory"  # Faster but не persistent
```

### Memory Optimization

```yaml
rag:
  processing:
    chunk_size: 256  # Smaller chunks
    chunk_overlap: 25
  
  retrieval:
    top_k: 10  # Fewer results
    max_context_tokens: 32768  # Smaller context
  
  queue:
    memory:
      buffer_size: 500  # Smaller queue
```

---

## 🧪 Testing Configuration

```yaml
rag:
  enabled: true
  
  vector_store:
    backend: "pgvector"
    pgvector:
      dimensions: 768
  
  embeddings:
    model: "nomic-embed-text"
  
  queue:
    backend: "memory"
    memory:
      buffer_size: 100
      num_workers: 2
  
  security:
    encrypt_credentials: false  # Simplify tests
```

---

## 🔍 Configuration Validation

### Check Config at Startup

```go
// cmd/server/main.go
if cfg.RAG.Enabled {
    if err := validateRAGConfig(cfg.RAG); err != nil {
        log.Fatalf("Invalid RAG config: %v", err)
    }
}
```

### Validation Rules

```go
func validateRAGConfig(cfg ragconfig.RAGConfig) error {
    if cfg.Embeddings.Dimensions != cfg.VectorStore.PGVector.Dimensions {
        return errors.New("embeddings dimensions must match vector store dimensions")
    }
    
    if cfg.Security.EncryptCredentials && cfg.Security.EncryptionKey == "" {
        return errors.New("encryption key required when encrypt_credentials=true")
    }
    
    if len(cfg.Security.EncryptionKey) != 32 {
        return errors.New("encryption key must be exactly 32 bytes for AES-256")
    }
    
    return nil
}
```

---

## 🎯 Quick Reference

| Setting | Type | Default | Notes |
|---------|------|---------|-------|
| `enabled` | bool | `true` | Master switch |
| `embeddings.model` | string | `nomic-embed-text` | Ollama model name |
| `embeddings.dimensions` | int | `768` | Must match model |
| `vector_store.backend` | string | `pgvector` | "pgvector" or "qdrant" |
| `processing.chunk_size` | int | `512` | tokens |
| `processing.chunk_overlap` | int | `50` | tokens (10%) |
| `retrieval.top_k` | int | `20` | chunks to retrieve |
| `retrieval.similarity_threshold` | float | `0.7` | 0.0-1.0 |
| `queue.backend` | string | `postgres` | "postgres" or "memory" |
| `security.encryption_key` | string | - | **32 bytes required** |

---

## 📚 Related Documentation

- [RAG_DEPLOYMENT_GUIDE.md](./RAG_DEPLOYMENT_GUIDE.md) - Deployment instructions
- [RAG_TESTING.md](./RAG_TESTING.md) - Testing guide
- [Architecture.MD](../Architecture.MD) - System architecture

---

**Version:** 1.13.0  
**Last Updated:** 2025-10-26  
**Enable/Disable:** Set `rag.enabled: false` to disable entire subsystem

