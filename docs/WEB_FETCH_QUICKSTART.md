# 🌐 Web Fetch Quick Start

**Version:** 1.10.4 ✅  
**Status:** Ready to Use  
**Feature:** Web content fetching с SSRF защитой, rate limiting и кешированием

---

## 🔥 Ключевая особенность: Полный контент без обрезания

**С версии 1.10.4** Web Fetcher передает **полное содержимое страницы** модели, без ограничения длины:

### Преимущества

✅ **Весь текст передается целиком** - модель получает полную информацию  
✅ **Идеально для больших контекстов** - работает с моделями 128K+ токенов  
✅ **Нет потери данных** - важные детали не теряются из-за truncation  
✅ **Лучший анализ** - модель видит полную картину документа  

### Как это работает

```go
// В chat.go - TruncateLength: 0 означает "без обрезания"
ProcessMessageOptions{
    AutoFetch:      true,
    MaxURLs:        2,
    TruncateLength: 0,  // 0 = передаем весь контент
}
```

### Пример

**До (старое поведение):**
```
Content: First 3000 chars... [truncated]
```

**После (новое поведение):**
```
Content: <вся страница целиком, 50,000+ символов>

*[Full content length: 52341 chars, 8723 words]*
```

### Контроль размера

Если нужно ограничить размер (для моделей с малым контекстом):

```go
ProcessMessageOptions{
    TruncateLength: 5000,  // Обрезать до 5000 символов
}
```

---

## 🎯 Что реализовано

✅ **Phase 1: HTTP Fetcher & Validation**

- HTTP клиент с retry logic
- URL validator с SSRF protection
- Rate limiter для доменов
- Защита от private IPs и localhost

✅ **Phase 2: HTML Parser**

- goquery integration для HTML парсинга
- Text extraction (чистый текст без HTML)
- Metadata extraction (Open Graph, Twitter Card)
- Links extraction

✅ **Phase 3: Integration**

- Database schema (web_fetches, web_fetch_rate_limits)
- API endpoints (`/api/web/fetch`, `/api/web/fetch/batch`)
- Service layer с кешированием
- Configuration в `dev.yaml`

---

## 📦 Компоненты

### Пакет `internal/webfetch/`

```
internal/webfetch/
├── config.go       # Конфигурация
├── interface.go    # Интерфейсы (WebFetcher, WebPage, etc)
├── validator.go    # URL validator с SSRF защитой
├── ratelimiter.go  # Rate limiter для доменов
├── fetcher.go      # HTTP fetcher с retry
├── parser.go       # HTML parser (goquery)
├── service.go      # High-level service
└── validator_test.go  # Тесты
```

### API Handlers

- `internal/api/handlers/webfetch.go` - REST API endpoints

### Database

- **Migration v31**: `web_fetches`, `web_fetch_rate_limits` таблицы
- **Location**: `internal/storage/sqlite/sqlite.go`

---

## 🚀 Использование

### 1. Автоматическая интеграция с Chat ✨

**Самый простой способ!** Web Fetch автоматически активируется в чате:

```
User: Summarize https://example.com/article

→ Proxy автоматически:
  1. Детектирует URL в сообщении
  2. Fetch content (HTML → plain text)
  3. Добавляет контент в контекст
  4. Отправляет обогащенный запрос LLM

LLM: Based on the article content... [uses actual fetched content]
```

**Примеры использования в чате:**

```
User: What's on https://news.ycombinator.com right now?
→ Fetch homepage → LLM summarizes top stories

User: Compare https://example.com/page1 vs https://example.com/page2
→ Fetch both pages → LLM compares

User: Explain https://docs.python.org/3/library/asyncio.html
→ Fetch docs → LLM explains based on actual content
```

**Настройки авто-fetch:**
- ✅ Max 2 URLs per message (context limit protection)
- ✅ 15s timeout per URL (fast fetch)
- ✅ Works in streaming & non-streaming
- ✅ Cache 1h (repeat requests instant)
- ✅ SSRF protected (no private IPs)

### 2. Конфигурация

```yaml
# configs/dev.yaml
web_fetch:
  enabled: true  # ← Enables automatic chat integration
  timeout: "30s"
  
  # Security (SSRF protection)
  block_private_ips: true
  block_localhost: true
  allowed_domains: []
  blocked_domains: []
  
  # Rate limiting
  rate_limit_enabled: true
  default_requests_per_min: 10
  
  # Cache
  cache_enabled: true
  cache_ttl: "1h"
```

### 3. Direct API Usage (Optional)

#### Fetch Single URL

```bash
curl -X POST http://localhost:8080/api/web/fetch \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com/article",
    "extract_links": true,
    "summarize": false,
    "use_cache": true
  }'
```

**Response:**

```json
{
  "url": "https://example.com/article",
  "title": "Article Title",
  "content": "Plain text content...",
  "metadata": {
    "description": "Article description",
    "author": "John Doe",
    "og_title": "Social Media Title",
    "language": "en"
  },
  "word_count": 1234,
  "fetch_time_ms": 450,
  "cached": false,
  "fetched_at": "2025-10-24T10:00:00Z"
}
```

#### Batch Fetch

```bash
curl -X POST http://localhost:8080/api/web/fetch/batch \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "urls": [
      "https://example.com/page1",
      "https://example.com/page2",
      "https://example.com/page3"
    ],
    "summarize": false
  }'
```

**Response:**

```json
{
  "results": [
    {
      "url": "https://example.com/page1",
      "status": "success",
      "title": "Page 1",
      "content": "...",
      "fetch_time_ms": 200
    },
    {
      "url": "https://example.com/page2",
      "status": "failed",
      "error": "timeout"
    }
  ],
  "total": 3,
  "succeeded": 2,
  "failed": 1
}
```

---

## 🔒 Security Features

### SSRF Protection

✅ Блокировка private IP ranges:

- `10.0.0.0/8`
- `172.16.0.0/12`
- `192.168.0.0/16`
- `169.254.0.0/16` (link-local)
- `127.0.0.0/8` (loopback)

✅ Блокировка localhost вариантов:

- `localhost`
- `127.0.0.1`
- `::1`

✅ DNS resolution check перед запросом

### Rate Limiting

- Per-domain rate limits
- Default: 10 requests/minute
- Configurable per domain
- Exponential backoff при ретраях

### Content Validation

- Max response size check
- Content-Type validation
- Redirect limits (max 5)

---

## 🧪 Testing

Запустить базовые тесты:

```bash
go test ./internal/webfetch/... -v
```

Benchmark validator:

```bash
go test ./internal/webfetch -bench=BenchmarkURLValidator -benchtime=10s
```

---

## 📊 Performance

**Typical latency:**

- Simple page: ~200-500ms
- With images/scripts: ~1-2s
- Cached page: <10ms

**Rate limiting:**

- Default: 10 req/min per domain
- Burst: 1 request (10% of per-minute)
- Retry: 3 attempts с exponential backoff

**Cache:**

- Default TTL: 1 hour
- Storage: PostgreSQL/SQLite
- Lookup: SHA256 hash URL

---

## 🔗 Связь с RAG (v1.13.0)

Web Fetcher будет **переиспользован** в RAG System:

```go
// v1.13.0 RAG будет импортировать webfetch
import "internal/webfetch"

type WebDataSource struct {
    fetcher webfetch.WebFetcher
}

func (s *WebDataSource) Sync(ctx context.Context) error {
    // 1. Fetch web pages
    page := s.fetcher.Fetch(ctx, url, opts)
    
    // 2. Chunk content (NEW in v1.13)
    chunks := s.chunker.Chunk(page.Content)
    
    // 3. Generate embeddings (NEW in v1.13)
    embeddings := s.embedder.Embed(chunks)
    
    // 4. Store in vector DB (NEW in v1.13)
    s.vectorDB.Store(embeddings)
}
```

**Преимущества:**

- ✅ HTML parser готов
- ✅ SSRF защита работает
- ✅ Rate limiting настроен
- ✅ Кеширование реализовано

---

## 📚 Дальнейшее развитие

**Optional Enhancements**:
- [ ] WebSocket progress updates во время fetch
- [ ] LLM Summarization через Ollama
- [ ] Configurable max URLs per message
- [ ] WebUI indicator для fetched content

**RAG Integration (v1.13.0)**:
- [ ] Web crawling для сайтов
- [ ] Sitemap parsing
- [ ] Scheduled синхронизация
- [ ] Chunk + embeddings integration

---

## ✅ Success Criteria

- [x] SSRF protection работает (блокирует private IPs)
- [x] Rate limiting функционирует (10 req/min per domain)
- [x] HTML parsing корректный (goquery)
- [x] Metadata extraction (Open Graph, Twitter Card)
- [x] API endpoints готовы
- [x] **Chat integration COMPLETE** ✨
  - [x] Автодетектирование URLs
  - [x] Автоматический fetch
  - [x] Context enrichment
  - [x] Streaming support
- [x] Database migration применена
- [x] Configuration в dev.yaml
- [x] Компиляция успешна
- [x] Tests passing (URL detector, validator)

**Status: COMPLETE WITH CHAT INTEGRATION ✅**

---

**Создано:** 2025-10-24  
**Версия:** v1.10.4  
**Команда реализации:** 8 часов (согласно плану)

