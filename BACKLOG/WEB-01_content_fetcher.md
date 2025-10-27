# WEB-01: Web Content Fetcher & Summarization

**Версия:** 1.8.0  
**Приоритет:** High  
**Сложность:** Medium  
**Оценка:** 6-10 часов

## Описание

Интеграция функционала получения и анализа контента с веб-страниц. Пользователи смогут отправить URL в чат, система автоматически загрузит контент, извлечет текст и метаданные, а LLM выполнит summarization или ответит на вопросы о содержимом.

## Проблема

В текущей версии чат не может:
- Получать информацию с внешних веб-страниц
- Анализировать статьи, документацию, новости по ссылке
- Отвечать на вопросы о содержимом веб-страницы
- Создавать краткое содержание (summary) статей

## Решение

HTTP client + HTML parser для извлечения текста + LLM для summarization.

### Архитектура

```
User → [URL in chat] → WebFetcher Service → HTML Parser → Text Extraction
                                                                ↓
                                              Context → LLM → Summarization/Answer
```

## Технические детали

### Backend Components

**1. Web Fetcher Service**

```go
// internal/services/webfetcher/fetcher.go

type WebFetcher struct {
    client      *http.Client
    userAgent   string
    timeout     time.Duration
    maxBodySize int64 // 10MB default
}

type FetchedContent struct {
    URL         string
    Title       string
    Description string
    Content     string    // Извлеченный текст
    Author      string
    PublishedAt *time.Time
    Keywords    []string
    Language    string
    WordCount   int
    FetchedAt   time.Time
}

func (f *WebFetcher) Fetch(ctx context.Context, url string) (*FetchedContent, error)
```

**2. HTML Parser**

Используем **goquery** (jQuery-like library для Go):
```go
import "github.com/PuerkitoBio/goquery"

func extractContent(html string) (*FetchedContent, error) {
    doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
    if err != nil {
        return nil, err
    }
    
    // Extract metadata
    title := doc.Find("title").First().Text()
    description := doc.Find("meta[name='description']").AttrOr("content", "")
    
    // Extract main content (remove nav, footer, ads)
    doc.Find("script, style, nav, footer, aside, .ads").Remove()
    mainContent := doc.Find("article, main, .content").First()
    if mainContent.Length() == 0 {
        mainContent = doc.Find("body")
    }
    
    text := strings.TrimSpace(mainContent.Text())
    
    return &FetchedContent{
        Title:       cleanTitle(title),
        Description: description,
        Content:     text,
        WordCount:   len(strings.Fields(text)),
    }, nil
}
```

**3. API Endpoint**

```
POST /api/chat/fetch-url
Authorization: Bearer <token>

Request:
{
  "url": "https://example.com/article",
  "action": "summarize" | "analyze" | "extract",
  "conversation_id": "uuid" (optional)
}

Response:
{
  "content": {
    "url": "https://example.com/article",
    "title": "Article Title",
    "description": "...",
    "text": "Full extracted text...",
    "word_count": 1234,
    "fetched_at": "2025-10-11T12:00:00Z"
  },
  "summary": "AI-generated summary..." (if action=summarize)
}
```

**4. Chat Integration**

Автоматическое распознавание URL в сообщениях:
```
User: "Summarize this article: https://example.com/blog/post"
      ↓
System: [auto-fetch URL] → [extract content] → [send to LLM with prompt]
      ↓
LLM: "This article discusses..."
```

### Frontend (WebUI)

**Chat Interface:**

1. **URL Detection**
   - Автоопределение URL паттернов в тексте
   - Кнопка "🔗 Fetch & Summarize" при обнаружении URL

2. **Fetch Status**
   - Loading indicator: "Fetching content from URL..."
   - Preview карточки с title, description, word count

3. **Actions**
   - "Summarize" - краткое содержание
   - "Analyze" - детальный анализ
   - "Answer" - задать вопрос о содержимом

**UI Mockup:**
```
┌─────────────────────────────────────────┐
│ User: Analyze https://example.com/post  │
├─────────────────────────────────────────┤
│ [🔗 Fetching content...]                │
│ ┌─────────────────────────────────────┐ │
│ │ 📄 Article Title                    │ │
│ │ example.com • 1,234 words           │ │
│ │ "Article description..."            │ │
│ └─────────────────────────────────────┘ │
├─────────────────────────────────────────┤
│ Assistant: This article discusses...    │
└─────────────────────────────────────────┘
```

### Security & Safety

**1. URL Validation**
```go
func isValidURL(urlStr string) error {
    parsedURL, err := url.Parse(urlStr)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }
    
    // Whitelist schemes
    if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
        return errors.New("only HTTP/HTTPS allowed")
    }
    
    // Blacklist private IPs, localhost
    ip := net.ParseIP(parsedURL.Hostname())
    if ip != nil && (ip.IsLoopback() || ip.IsPrivate()) {
        return errors.New("private IPs not allowed")
    }
    
    return nil
}
```

**2. Rate Limiting**
- Max 10 fetches per user per hour
- Max 5 concurrent fetches per user
- Timeout: 30 seconds per fetch

**3. Content Limits**
- Max page size: 10MB
- Max content length for LLM: 50,000 tokens (~200KB text)
- Truncate если больше

**4. Domain Blacklist (optional)**
```yaml
# config.yaml
web_fetcher:
  enabled: true
  timeout: 30s
  max_size: 10485760 # 10MB
  user_agent: "aigateway/1.8.0"
  blacklist_domains:
    - "example-spam.com"
    - "malicious-site.net"
```

## Storage

```sql
-- Кэш загруженных веб-страниц (optional)
CREATE TABLE web_cache (
  id TEXT PRIMARY KEY,
  url TEXT NOT NULL UNIQUE,
  url_hash TEXT NOT NULL,
  title TEXT,
  description TEXT,
  content TEXT NOT NULL,
  word_count INTEGER,
  metadata TEXT, -- JSON
  fetched_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP, -- TTL для cache
  fetch_count INTEGER DEFAULT 1
);

CREATE INDEX idx_web_cache_url_hash ON web_cache(url_hash);
CREATE INDEX idx_web_cache_fetched_at ON web_cache(fetched_at DESC);
```

**Cache Strategy:**
- TTL: 24 hours (configurable)
- Automatic cleanup старых записей
- Инвалидация cache при повторном fetch с параметром `force=true`

## Требования

### Функциональные

1. ✅ Fetch HTML контента по URL (HTTP/HTTPS)
2. ✅ Parse HTML и извлечение текста
3. ✅ Извлечение metadata (title, description, keywords)
4. ✅ Автоматическое распознавание URL в сообщениях
5. ✅ Три режима: summarize, analyze, extract
6. ✅ Cache загруженных страниц (24h TTL)
7. ✅ Rate limiting (10 fetches/hour per user)
8. ✅ Timeout и размер контента
9. ✅ URL validation и безопасность
10. ✅ Preview карточка с metadata

### Нефункциональные

1. **Performance**
   - Fetch + Parse < 5s для обычных страниц
   - LLM summarization зависит от модели
   - Cache hit: instant response

2. **Security**
   - Валидация URL (только HTTP/HTTPS)
   - Blacklist для private IPs, localhost
   - User-Agent для идентификации
   - SSRF protection

3. **Reliability**
   - Retry logic для failed fetches (3 попытки)
   - Graceful handling 404, 500 errors
   - Timeout protection

## Acceptance Criteria

- [ ] Пользователь отправляет URL в чат
- [ ] Система автоматически распознает URL
- [ ] Fetch контента происходит в фоне
- [ ] Preview карточка отображается с metadata
- [ ] LLM получает extracted text как контекст
- [ ] Summarization работает корректно
- [ ] Rate limiting применяется per user
- [ ] Cache работает (24h TTL)
- [ ] Валидация URL блокирует private IPs
- [ ] Error handling для 404, timeout, invalid HTML
- [ ] Unit tests для fetcher, parser
- [ ] Integration tests с mock HTTP server

## Риски и зависимости

### Риски

1. **SSRF атаки** - fetch внутренних ресурсов
   - Mitigation: Валидация URL, blacklist private IPs

2. **Slow/hanging requests** - медленные сайты
   - Mitigation: Timeout 30s, user notification

3. **Malformed HTML** - невалидный HTML
   - Mitigation: Robust parser (goquery), fallback

4. **Large pages** - huge HTML files
   - Mitigation: Max size 10MB, streaming

### Зависимости

1. **goquery** - HTML parsing (github.com/PuerkitoBio/goquery)
2. **net/http** - HTTP client (stdlib)
3. **Rate limiter** - используем существующий из AUTH-03

## Связанные задачи

- **VISION-01**: Vision OCR - схожая логика контента
- **FILE-01**: File Upload - схожая валидация размеров
- **RATE-02**: Advanced Rate Limiting - улучшенный rate limit

## Примечания

- Только статический HTML (не JavaScript-rendered контент)
- No headless browser (Playwright/Puppeteer)
- Не поддерживаются: SPA, React apps, dynamic content
- Для dynamic сайтов можно рекомендовать copy-paste текста

## Пример использования

```javascript
// Frontend: Auto-detect URL and fetch
const message = "Summarize this: https://example.com/article";
const urlMatch = message.match(/https?:\/\/[^\s]+/);

if (urlMatch) {
  const url = urlMatch[0];
  
  // Show loading
  showFetchingIndicator(url);
  
  // Fetch content
  const response = await fetch('/api/chat/fetch-url', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({
      url: url,
      action: 'summarize',
      conversation_id: conversationId
    })
  });
  
  const { content, summary } = await response.json();
  
  // Display preview + summary
  displayContentPreview(content);
  displaySummary(summary);
}
```

---

**Статус:** 📋 Planned for v1.8.0  
**Последнее обновление:** 2025-10-11


