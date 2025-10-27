# RAG System Testing Guide

Comprehensive testing suite для RAG System v1.13.0.

## 📋 Test Coverage

### Core Components

#### 1. **Semantic Chunker** (`internal/rag/chunker/semantic_test.go`)
- ✅ Text chunking strategies (semantic, paragraph-based)
- ✅ Token estimation accuracy
- ✅ Overlap handling между chunks
- ✅ Sentence splitting
- ✅ Edge cases (empty text, single paragraph, long paragraphs)
- ✅ Benchmarks для performance

#### 2. **RAG Orchestrator** (`internal/rag/orchestrator/orchestrator_test.go`)
- ✅ Query processing pipeline
- ✅ Source filtering (by source IDs)
- ✅ Vector search integration
- ✅ Reranking algorithm
- ✅ Context assembly (до 4000 tokens)
- ✅ Error handling
- ✅ Mock embedder & vector store
- ✅ Benchmarks

#### 3. **Data Source Service** (`internal/services/rag/datasource_service_test.go`)
- ✅ CRUD operations (Create, Read, Update, Delete)
- ✅ AES-256 encryption/decryption для credentials
- ✅ List filtering (by user, tenant, type, status)
- ✅ Connection testing
- ✅ Sync operations
- ✅ Input validation
- ✅ Benchmarks для encryption

#### 4. **Chat Handler RAG Integration** (`internal/api/handlers/chat_rag_test.go`)
- ✅ Message enrichment with RAG context
- ✅ System message injection
- ✅ Parameter passing (top_k, min_score, source_ids, rerank)
- ✅ Error handling (no orchestrator, no user message, no results)
- ✅ Full E2E chat completion with RAG
- ✅ Benchmarks

#### 5. **Ollama Embedder** (`internal/rag/embeddings/ollama_test.go`)
- ✅ Single embedding generation
- ✅ Batch embedding processing
- ✅ Model dimensions mapping
- ✅ HTTP client integration (with mock server)
- ✅ Error handling (server errors, timeouts)
- ✅ Default model fallback
- ✅ Benchmarks

#### 6. **Worker Pool** (`internal/rag/processor/worker_test.go`)
- ✅ Worker lifecycle (start/stop)
- ✅ Job submission & processing
- ✅ Concurrent job processing
- ✅ Queue management
- ✅ Failure handling
- ✅ Stats tracking
- ✅ Wait operations
- ✅ Benchmarks

---

## 🚀 Running Tests

### All RAG Tests

```bash
# Run all RAG-related tests
go test ./internal/rag/... -v

# With coverage
go test ./internal/rag/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Specific Component Tests

```bash
# Semantic Chunker
go test ./internal/rag/chunker -v

# RAG Orchestrator
go test ./internal/rag/orchestrator -v

# Data Source Service
go test ./internal/services/rag -v

# Chat Handler RAG
go test ./internal/api/handlers -v -run TestChatHandler.*RAG

# Embeddings
go test ./internal/rag/embeddings -v

# Worker Pool
go test ./internal/rag/processor -v
```

### Race Detection

```bash
# Critical для concurrent code (worker pool, orchestrator)
go test ./internal/rag/... -race
```

### Benchmarks

```bash
# Run all RAG benchmarks
go test ./internal/rag/... -bench=. -benchmem

# Specific benchmarks
go test ./internal/rag/chunker -bench=BenchmarkSemanticChunker
go test ./internal/rag/orchestrator -bench=BenchmarkRAGOrchestrator
go test ./internal/services/rag -bench=BenchmarkDataSourceService
go test ./internal/api/handlers -bench=BenchmarkChatHandler.*RAG
```

### Example Output

```bash
$ go test ./internal/rag/chunker -v

=== RUN   TestSemanticChunker_Name
--- PASS: TestSemanticChunker_Name (0.00s)
=== RUN   TestSemanticChunker_EstimateTokens
--- PASS: TestSemanticChunker_EstimateTokens (0.00s)
=== RUN   TestSemanticChunker_Chunk_EmptyText
--- PASS: TestSemanticChunker_Chunk_EmptyText (0.00s)
=== RUN   TestSemanticChunker_Chunk_SingleParagraph
--- PASS: TestSemanticChunker_Chunk_SingleParagraph (0.01s)
...
PASS
ok      github.com/your-org/ollama-proxy/internal/rag/chunker  1.234s
```

---

## 📊 Coverage Goals

### Target Coverage

| Component                | Target | Notes                                    |
|--------------------------|--------|------------------------------------------|
| **Semantic Chunker**     | 95%+   | Core text processing logic               |
| **RAG Orchestrator**     | 90%+   | Integration component                    |
| **Data Source Service**  | 95%+   | Critical for credentials security        |
| **Chat Handler RAG**     | 90%+   | API integration layer                    |
| **Embeddings**           | 85%+   | External API dependency (mocked)         |
| **Worker Pool**          | 95%+   | Concurrent processing (critical)         |

### Check Coverage

```bash
# Generate coverage report
go test ./internal/rag/... -coverprofile=coverage.out

# View by function
go tool cover -func=coverage.out

# HTML report
go tool cover -html=coverage.out -o coverage.html
```

---

## 🧪 Test Patterns Used

### 1. **Table-Driven Tests**

```go
tests := []struct {
    name     string
    input    string
    expected int
}{
    {"Empty", "", 0},
    {"Short", "test", 1},
    {"Long", strings.Repeat("word ", 100), 133},
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result := chunker.EstimateTokens(tt.input)
        if result != tt.expected {
            t.Errorf("Expected %d, got %d", tt.expected, result)
        }
    })
}
```

### 2. **Mock Objects**

```go
type mockEmbedder struct {
    embeddings []float64
    err        error
}

func (m *mockEmbedder) Embed(ctx context.Context, req EmbeddingRequest) (*Embedding, error) {
    if m.err != nil {
        return nil, m.err
    }
    return &Embedding{Vector: m.embeddings}, nil
}
```

### 3. **HTTP Mock Servers**

```go
mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"embedding": [0.1, 0.2, 0.3]}`))
}))
defer mockServer.Close()
```

### 4. **Context Testing**

```go
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
defer cancel()

result, err := service.ProcessWithContext(ctx, input)
```

### 5. **Concurrent Testing**

```go
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        _ = pool.Submit(job)
    }(i)
}
wg.Wait()
```

---

## 🐛 Common Test Failures & Fixes

### 1. **Race Conditions**

```bash
# Problem
go test -race ./internal/rag/processor
WARNING: DATA RACE

# Fix
Use atomic operations или proper locking:
atomic.AddInt64(&counter, 1)
```

### 2. **Timeout Issues**

```bash
# Problem
panic: test timed out after 10m0s

# Fix
Reduce test data size или increase timeout:
go test -timeout 30s ./internal/rag/...
```

### 3. **Mock Server Not Responding**

```bash
# Problem
connection refused

# Fix
Ensure mock server is started before use:
mockServer := httptest.NewServer(handler)
defer mockServer.Close() // Always close!
```

### 4. **Coverage Not Showing**

```bash
# Problem
coverage: 0.0% of statements

# Fix
Run tests with -coverprofile:
go test -coverprofile=coverage.out ./internal/rag/...
```

---

## 📝 Adding New Tests

### Checklist для новых RAG tests:

- [ ] **Unit tests** для core logic
- [ ] **Integration tests** с mock dependencies
- [ ] **Error cases** (nil inputs, timeouts, errors)
- [ ] **Edge cases** (empty inputs, large inputs)
- [ ] **Concurrent tests** если используется concurrency
- [ ] **Benchmarks** для performance-critical paths
- [ ] **Race detection** (`go test -race`)
- [ ] **Comments** explaining what is tested

### Example Template

```go
func TestNewFeature_Success(t *testing.T) {
    // Arrange
    setup := createTestSetup()
    input := "test input"
    expected := "expected output"

    // Act
    result, err := setup.service.ProcessFeature(context.Background(), input)

    // Assert
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }

    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}

func TestNewFeature_Error(t *testing.T) {
    // Test error case
}

func BenchmarkNewFeature(b *testing.B) {
    // Benchmark implementation
}
```

---

## 🎯 CI/CD Integration

### GitHub Actions Example

```yaml
name: RAG Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'
      
      - name: Run RAG Tests
        run: |
          go test ./internal/rag/... -v -race -coverprofile=coverage.out
          
      - name: Check Coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Coverage: $COVERAGE%"
          if (( $(echo "$COVERAGE < 85" | bc -l) )); then
            echo "❌ Coverage below 85%"
            exit 1
          fi
          
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

---

## 📈 Performance Benchmarks

### Expected Performance (reference machine: AMD Ryzen 9 5900X)

```
BenchmarkSemanticChunker_Chunk_SmallText-24      50000     30000 ns/op      5000 B/op     50 allocs/op
BenchmarkSemanticChunker_Chunk_LargeText-24       1000   1500000 ns/op    250000 B/op   1500 allocs/op
BenchmarkRAGOrchestrator_Query-24                 500   2500000 ns/op    150000 B/op   1000 allocs/op
BenchmarkDataSourceService_EncryptDecrypt-24    10000    100000 ns/op     10000 B/op    100 allocs/op
BenchmarkWorkerPool_Submit-24                  100000     10000 ns/op      1000 B/op     10 allocs/op
```

### Running Benchmarks

```bash
# Run benchmarks with memory stats
go test ./internal/rag/... -bench=. -benchmem -benchtime=10s

# Compare with baseline
go test -bench=. -benchmem > new.txt
benchstat old.txt new.txt
```

---

## ✅ Test Verification

### Pre-Commit Checklist

```bash
# 1. Run all tests
go test ./internal/rag/... -v

# 2. Check race conditions
go test ./internal/rag/... -race

# 3. Verify coverage
go test ./internal/rag/... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total

# 4. Run linters
golangci-lint run ./internal/rag/...

# 5. Format code
go fmt ./internal/rag/...
```

---

## 🔗 Related Documentation

- [RAG_DEPLOYMENT_GUIDE.md](./RAG_DEPLOYMENT_GUIDE.md) - Deployment instructions
- [Architecture.MD](../Architecture.MD) - Overall architecture
- [Roadmap.MD](../Roadmap.MD) - RAG system roadmap (v1.13.0)

---

## 🤝 Contributing Tests

При добавлении новых RAG features:

1. **Write tests first** (TDD approach)
2. **Follow existing patterns** (table-driven, mocks)
3. **Test edge cases** (not just happy path)
4. **Add benchmarks** для performance-critical code
5. **Run race detector** если есть concurrency
6. **Document complex tests** с комментариями

**Quality > Quantity**: 10 well-written tests лучше 100 shallow tests.

---

Generated: 2025-10-26  
Version: 1.13.0  
Coverage Target: 90%+


