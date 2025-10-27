# WEB-FETCH-01: Web Content Fetcher & Summarization

**Version:** 1.10.3  
**Priority:** HIGH  
**Estimated Time:** 6-8 hours  
**Status:** 📋 Planned  
**Dependencies:** None (standalone component)

**Strategic Importance:** 🔄 HTML parser будет основой для web data sources в RAG (v1.13.0)

---

## 🎯 Overview

Система извлечения и обработки контента с веб-страниц с автоматическим парсингом HTML, извлечением текста, metadata и опциональной summarization через LLM.

**Ключевая особенность:** Web fetcher будет расширен в RAG v1.13.0 для автоматической синхронизации внешних веб-источников данных.

### Основные возможности

- 🌐 **HTTP Client**: Robust fetching с retries и timeout
- 🔍 **HTML Parsing**: goquery для извлечения контента
- 📝 **Text Extraction**: Чистый текст без HTML тегов
- 📊 **Metadata Extraction**: Title, description, keywords, Open Graph
- 🔒 **Security**: URL validation, size limits, rate limiting
- 💬 **Chat Integration**: Fetch URL → extract → summarize → context
- 🎯 **RAG-ready**: Интерфейсы готовы для web data sources

---

## 🏗️ Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     WebUI / Chat / API                       │
│  User: "Summarize https://example.com/article"              │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   Web Fetcher Service                        │
│  - URL validation (scheme, domain whitelist)                 │
│  - Rate limiting (per domain)                                │
│  - Cache management (optional)                               │
│  - Retry logic with exponential backoff                      │
└──────┬──────────────────────────────────┬───────────────────┘
       │                                  │
       ▼                                  ▼
┌─────────────────┐              ┌─────────────────────────────┐
│  HTTP Client    │              │    HTML Parser              │
│                 │              │    (goquery)                │
│ - User-Agent    │              │                             │
│ - Timeout: 30s  │              │  ┌──────────────────────┐   │
│ - Redirects     │              │  │  Text Extractor      │   │
│ - Headers       │              │  │  - Remove scripts    │   │
│                 │              │  │  - Remove styles     │   │
│ GET https://... │              │  │  - Clean whitespace  │   │
│                 │              │  └──────────────────────┘   │
│ Response 200 OK │              │                             │
│ Content: HTML   │              │  ┌──────────────────────┐   │
└─────────────────┘              │  │  Metadata Extractor  │   │
                                 │  │  - <title>           │   │
                                 │  │  - <meta> tags       │   │
                                 │  │  - Open Graph        │   │
                                 │  │  - JSON-LD           │   │
                                 │  └──────────────────────┘   │
                                 └─────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Content Processor                           │
│                                                              │
│  1. Clean HTML → Plain text                                 │
│  2. Extract links (for future crawling)                     │
│  3. Detect language                                         │
│  4. Count words/paragraphs                                  │
│                                                              │
│  Optional: Summarization via LLM                            │
│  - Long article → 3-5 sentence summary                      │
└─────────────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                  PostgreSQL Database                         │
│                                                              │
│  Table: web_fetches                                         │
│  - id, url, title, content, metadata                        │
│  - fetch_status, fetch_time_ms                              │
│  - summary (LLM-generated)                                  │
│  - created_at, expires_at (cache)                           │
└─────────────────────────────────────────────────────────────┘
```

### Fetcher Interface

```go
// internal/webfetch/interface.go
type WebFetcher interface {
    // Fetch retrieves and parses web page
    Fetch(ctx context.Context, url string, opts FetchOptions) (*WebPage, error)
    
    // FetchBatch fetches multiple URLs concurrently
    FetchBatch(ctx context.Context, urls []string, opts FetchOptions) ([]*WebPage, error)
    
    // ValidateURL checks if URL is safe to fetch
    ValidateURL(url string) error
}

type FetchOptions struct {
    Timeout         time.Duration
    MaxSize         int64   // Maximum response size
    FollowRedirects bool
    ExtractLinks    bool    // Extract all links from page
    Summarize       bool    // Generate LLM summary
    CacheEnabled    bool    // Use cached result if available
    CacheTTL        time.Duration
}

type WebPage struct {
    URL             string
    Title           string
    Content         string                 // Plain text
    HTMLContent     string                 // Raw HTML (optional)
    Metadata        *PageMetadata
    Links           []string               // Extracted links
    Summary         string                 // LLM summary (optional)
    FetchTimeMS     int64
    FetchedAt       time.Time
    ExpiresAt       time.Time
}

type PageMetadata struct {
    Description     string
    Keywords        []string
    Author          string
    PublishedDate   time.Time
    ModifiedDate    time.Time
    Language        string
    
    // Open Graph
    OGTitle         string
    OGDescription   string
    OGImage         string
    OGType          string
    
    // Twitter Card
    TwitterCard     string
    TwitterTitle    string
    TwitterDescription string
    
    // JSON-LD structured data
    StructuredData  map[string]interface{}
}
```

---

## 📊 Database Schema

### Web Fetches Table

```sql
CREATE TABLE web_fetches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    
    -- URL info
    url TEXT NOT NULL,
    url_hash VARCHAR(64) NOT NULL,  -- SHA256 для быстрого поиска
    domain VARCHAR(255) NOT NULL,
    
    -- Content
    title TEXT,
    content TEXT NOT NULL,  -- Extracted plain text
    html_content TEXT,  -- Raw HTML (optional, для debug)
    
    -- Metadata
    metadata JSONB,
    
    -- Links
    links TEXT[],  -- Extracted URLs
    
    -- LLM processing
    summary TEXT,  -- Generated summary
    summary_model VARCHAR(50),
    
    -- Fetch info
    fetch_status VARCHAR(20) DEFAULT 'success',  -- 'success', 'failed', 'timeout', 'too_large'
    fetch_time_ms INT,
    status_code INT,
    content_type VARCHAR(100),
    content_length BIGINT,
    fetch_error TEXT,
    
    -- Language detection
    language VARCHAR(10),
    word_count INT,
    
    -- Cache
    expires_at TIMESTAMP,  -- Cache expiry
    is_cached BOOLEAN DEFAULT false,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    last_fetched_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_web_fetches_url_hash ON web_fetches(url_hash);
CREATE INDEX idx_web_fetches_user ON web_fetches(user_id);
CREATE INDEX idx_web_fetches_domain ON web_fetches(domain);
CREATE INDEX idx_web_fetches_expires ON web_fetches(expires_at);
CREATE INDEX idx_web_fetches_created ON web_fetches(created_at DESC);

-- Full-text search
CREATE INDEX idx_web_fetches_content_search ON web_fetches USING GIN(to_tsvector('english', content));
```

### Fetch Rate Limits

```sql
CREATE TABLE web_fetch_rate_limits (
    id BIGSERIAL PRIMARY KEY,
    domain VARCHAR(255) NOT NULL UNIQUE,
    requests_per_minute INT DEFAULT 10,
    last_request_at TIMESTAMP,
    request_count INT DEFAULT 0,
    blocked_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_rate_limits_domain ON web_fetch_rate_limits(domain);
```

---

## 🔧 Configuration

### config.yaml

```yaml
web_fetch:
  enabled: true
  
  # HTTP Client
  http:
    timeout: 30s
    user_agent: "OllamaProxy-Bot/1.0 (https://github.com/yourorg/ollama-proxy)"
    follow_redirects: true
    max_redirects: 5
    max_response_size: "10MB"
    
    # Headers
    headers:
      Accept: "text/html,application/xhtml+xml"
      Accept-Language: "en-US,en;q=0.9,ru;q=0.8"
    
  # Security
  security:
    # URL validation
    allowed_schemes: ["http", "https"]
    blocked_domains: []  # Blacklist
    allowed_domains: []  # Whitelist (empty = all allowed)
    
    # Block private IPs (SSRF prevention)
    block_private_ips: true
    block_localhost: true
    
    # Content validation
    max_content_size: "10MB"
    allowed_content_types: ["text/html", "application/xhtml+xml"]
    
  # Rate Limiting
  rate_limit:
    enabled: true
    default_requests_per_minute: 10
    per_domain_limits:
      "wikipedia.org": 60
      "github.com": 60
    
    # Backoff
    retry_attempts: 3
    retry_delay: "1s"
    retry_max_delay: "10s"
    
  # Parsing
  parsing:
    extract_links: true
    extract_images: false
    preserve_formatting: false
    
    # Content cleaning
    remove_scripts: true
    remove_styles: true
    remove_comments: true
    
    # Selectors to remove (ads, navigation, etc.)
    remove_selectors:
      - "nav"
      - "header"
      - "footer"
      - ".advertisement"
      - "#sidebar"
    
    # Main content selectors (try in order)
    content_selectors:
      - "article"
      - "main"
      - ".post-content"
      - ".article-body"
      - "#content"
    
  # Summarization
  summarization:
    enabled: true
    provider: "ollama"
    model: "llama3.1:8b"
    
    prompt: |
      Summarize the following web page content in 3-5 sentences.
      Focus on the main points and key information.
      
      Content:
      {content}
      
      Summary:
    
    max_input_tokens: 4096  # Truncate long articles
    max_output_tokens: 256
    
  # Caching
  cache:
    enabled: true
    default_ttl: "1h"
    max_entries: 10000
    
    # Custom TTL per domain
    ttl_overrides:
      "news.ycombinator.com": "5m"  # News changes fast
      "wikipedia.org": "24h"         # Wikipedia changes slow
```

---

## 📡 API Endpoints

### Fetch Web Page

```http
POST /api/web/fetch
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
  "url": "https://example.com/article",
  "extract_links": false,
  "summarize": true,
  "use_cache": true
}

Response 200:
{
  "id": "uuid",
  "url": "https://example.com/article",
  "title": "Article Title",
  "content": "Full plain text content...",
  "metadata": {
    "description": "Article description",
    "author": "John Doe",
    "published_date": "2025-01-15T10:00:00Z",
    "language": "en"
  },
  "summary": "This article discusses... (3-5 sentences)",
  "word_count": 1234,
  "fetch_time_ms": 450,
  "cached": false,
  "fetched_at": "2025-01-16T10:00:00Z"
}
```

### Batch Fetch

```http
POST /api/web/fetch/batch
Content-Type: application/json

{
  "urls": [
    "https://example.com/page1",
    "https://example.com/page2",
    "https://example.com/page3"
  ],
  "summarize": false
}

Response 200:
{
  "results": [
    {
      "url": "https://example.com/page1",
      "status": "success",
      "title": "Page 1",
      "content": "..."
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

### Get Cached Page

```http
GET /api/web/cached/{url_hash}
Authorization: Bearer <jwt_token>

Response 200:
{
  "url": "https://example.com/article",
  "title": "...",
  "content": "...",
  "cached": true,
  "fetched_at": "2025-01-16T09:00:00Z",
  "expires_at": "2025-01-16T10:00:00Z"
}
```

### Chat with Web Content

```http
POST /api/v1/chat/completions
Content-Type: application/json

{
  "model": "llama3.1:8b",
  "messages": [
    {
      "role": "user",
      "content": "Summarize https://example.com/article"
    }
  ],
  "web_fetch": {
    "enabled": true,
    "summarize": true
  }
}

# System will:
# 1. Detect URL in message
# 2. Fetch web page
# 3. Add content/summary to context
# 4. Send to LLM

Response 200:
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "The article discusses... [summary based on fetched content]"
    }
  }],
  "web_fetch_metadata": {
    "url": "https://example.com/article",
    "title": "Article Title",
    "fetch_time_ms": 450
  }
}
```

---

## 📦 Package Structure

```
internal/webfetch/
├── config.go              # Configuration
├── service.go             # High-level web fetch service
├── interface.go           # Fetcher interface
├── fetcher.go             # HTTP fetcher implementation
├── parser.go              # HTML parser (goquery)
├── metadata.go            # Metadata extractor
├── cleaner.go             # HTML cleaning
├── validator.go           # URL validation
├── ratelimiter.go         # Rate limiting
├── cache.go               # Fetch cache
└── service_test.go        # Tests
```

---

## 🔧 Implementation Details

### HTML Parser

```go
// internal/webfetch/parser.go
package webfetch

import (
    "io"
    "strings"
    
    "github.com/PuerkitoBio/goquery"
)

type HTMLParser struct {
    config ParserConfig
}

type ParserConfig struct {
    RemoveSelectors   []string
    ContentSelectors  []string
    ExtractLinks      bool
}

func (p *HTMLParser) Parse(html io.Reader) (*ParsedContent, error) {
    doc, err := goquery.NewDocumentFromReader(html)
    if err != nil {
        return nil, err
    }
    
    // Extract title
    title := doc.Find("title").First().Text()
    
    // Remove unwanted elements
    for _, selector := range p.config.RemoveSelectors {
        doc.Find(selector).Remove()
    }
    
    // Find main content
    var content string
    for _, selector := range p.config.ContentSelectors {
        if el := doc.Find(selector).First(); el.Length() > 0 {
            content = p.extractText(el)
            break
        }
    }
    
    // Fallback: use body
    if content == "" {
        content = p.extractText(doc.Find("body"))
    }
    
    // Extract metadata
    metadata := p.extractMetadata(doc)
    
    // Extract links
    var links []string
    if p.config.ExtractLinks {
        doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
            if href, exists := s.Attr("href"); exists {
                links = append(links, href)
            }
        })
    }
    
    return &ParsedContent{
        Title:    strings.TrimSpace(title),
        Content:  p.cleanText(content),
        Metadata: metadata,
        Links:    links,
    }, nil
}

func (p *HTMLParser) extractText(sel *goquery.Selection) string {
    // Remove script and style tags
    sel.Find("script, style").Remove()
    
    // Get text
    return sel.Text()
}

func (p *HTMLParser) cleanText(text string) string {
    // Remove extra whitespace
    lines := strings.Split(text, "\n")
    var cleaned []string
    
    for _, line := range lines {
        line = strings.TrimSpace(line)
        if line != "" {
            cleaned = append(cleaned, line)
        }
    }
    
    return strings.Join(cleaned, "\n")
}

func (p *HTMLParser) extractMetadata(doc *goquery.Document) *PageMetadata {
    meta := &PageMetadata{}
    
    // Meta tags
    doc.Find("meta").Each(func(i int, s *goquery.Selection) {
        name, _ := s.Attr("name")
        property, _ := s.Attr("property")
        content, _ := s.Attr("content")
        
        switch {
        case name == "description":
            meta.Description = content
        case name == "keywords":
            meta.Keywords = strings.Split(content, ",")
        case name == "author":
            meta.Author = content
        case property == "og:title":
            meta.OGTitle = content
        case property == "og:description":
            meta.OGDescription = content
        case property == "og:image":
            meta.OGImage = content
        }
    })
    
    return meta
}
```

### URL Validator

```go
// internal/webfetch/validator.go
package webfetch

import (
    "fmt"
    "net"
    "net/url"
)

type URLValidator struct {
    config ValidationConfig
}

type ValidationConfig struct {
    AllowedSchemes   []string
    BlockedDomains   []string
    AllowedDomains   []string
    BlockPrivateIPs  bool
    BlockLocalhost   bool
}

func (v *URLValidator) Validate(rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }
    
    // Check scheme
    if !v.isAllowedScheme(u.Scheme) {
        return fmt.Errorf("scheme %s not allowed", u.Scheme)
    }
    
    // Check domain whitelist/blacklist
    if len(v.config.AllowedDomains) > 0 {
        if !v.isDomainAllowed(u.Host) {
            return fmt.Errorf("domain %s not in whitelist", u.Host)
        }
    }
    
    if v.isDomainBlocked(u.Host) {
        return fmt.Errorf("domain %s is blocked", u.Host)
    }
    
    // SSRF prevention
    if v.config.BlockPrivateIPs {
        if ip := net.ParseIP(u.Hostname()); ip != nil {
            if ip.IsPrivate() || ip.IsLoopback() {
                return fmt.Errorf("private IP addresses not allowed")
            }
        }
    }
    
    if v.config.BlockLocalhost {
        if u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" {
            return fmt.Errorf("localhost not allowed")
        }
    }
    
    return nil
}

func (v *URLValidator) isAllowedScheme(scheme string) bool {
    for _, s := range v.config.AllowedSchemes {
        if s == scheme {
            return true
        }
    }
    return false
}
```

### Fetcher Service

```go
// internal/webfetch/fetcher.go
package webfetch

import (
    "context"
    "io"
    "net/http"
    "time"
)

type Fetcher struct {
    client     *http.Client
    validator  *URLValidator
    parser     *HTMLParser
    rateLimiter *RateLimiter
}

func NewFetcher(config Config) *Fetcher {
    return &Fetcher{
        client: &http.Client{
            Timeout: config.HTTP.Timeout,
            CheckRedirect: func(req *http.Request, via []*http.Request) error {
                if len(via) >= config.HTTP.MaxRedirects {
                    return fmt.Errorf("too many redirects")
                }
                return nil
            },
        },
        validator:   NewURLValidator(config.Security),
        parser:      NewHTMLParser(config.Parsing),
        rateLimiter: NewRateLimiter(config.RateLimit),
    }
}

func (f *Fetcher) Fetch(ctx context.Context, rawURL string, opts FetchOptions) (*WebPage, error) {
    start := time.Now()
    
    // Validate URL
    if err := f.validator.Validate(rawURL); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }
    
    // Check rate limit
    domain := extractDomain(rawURL)
    if err := f.rateLimiter.Wait(ctx, domain); err != nil {
        return nil, fmt.Errorf("rate limit exceeded: %w", err)
    }
    
    // Make HTTP request
    req, _ := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
    req.Header.Set("User-Agent", opts.UserAgent)
    
    resp, err := f.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("fetch failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
    }
    
    // Limit response size
    limitedReader := io.LimitReader(resp.Body, opts.MaxSize)
    
    // Parse HTML
    parsed, err := f.parser.Parse(limitedReader)
    if err != nil {
        return nil, fmt.Errorf("parse failed: %w", err)
    }
    
    return &WebPage{
        URL:         rawURL,
        Title:       parsed.Title,
        Content:     parsed.Content,
        Metadata:    parsed.Metadata,
        Links:       parsed.Links,
        FetchTimeMS: time.Since(start).Milliseconds(),
        FetchedAt:   time.Now(),
    }, nil
}
```

---

## 🧪 Testing Strategy

### Unit Tests

```go
// internal/webfetch/parser_test.go
func TestHTMLParser_Parse(t *testing.T) {
    tests := []struct {
        name     string
        html     string
        expected string
    }{
        {
            "simple",
            `<html><body><p>Hello World</p></body></html>`,
            "Hello World",
        },
        {
            "with nav",
            `<html><body><nav>Menu</nav><article>Content</article></body></html>`,
            "Content",  // nav should be removed
        },
    }
    
    parser := NewHTMLParser(ParserConfig{
        RemoveSelectors:  []string{"nav"},
        ContentSelectors: []string{"article", "body"},
    })
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            parsed, err := parser.Parse(strings.NewReader(tt.html))
            assert.NoError(t, err)
            assert.Contains(t, parsed.Content, tt.expected)
        })
    }
}
```

### Integration Tests

```go
// internal/webfetch/integration_test.go
func TestFetcher_FetchRealWebsite(t *testing.T) {
    if testing.Short() {
        t.Skip()
    }
    
    fetcher := NewFetcher(testConfig())
    
    page, err := fetcher.Fetch(context.Background(), "https://example.com", FetchOptions{
        MaxSize: 1024 * 1024,
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, page.Title)
    assert.NotEmpty(t, page.Content)
}
```

---

## 🎯 Implementation Plan

### Phase 1: HTTP Fetcher & Validation (2-3 hours)

**Tasks:**

- ✅ HTTP client setup
- ✅ URL validator (SSRF prevention)
- ✅ Rate limiter
- ✅ Retry logic

**Deliverables:**

- Можно fetch веб-страницы
- SSRF protection работает
- Rate limiting функционирует

---

### Phase 2: HTML Parser (2-3 hours)

**Tasks:**

- ✅ goquery integration
- ✅ Text extraction
- ✅ Metadata extraction
- ✅ HTML cleaning

- ✅ Unit tests

**Deliverables:**

- HTML парсится корректно
- Извлекается чистый текст
- Metadata доступна

---

### Phase 3: Integration & Caching (2 hours)

**Tasks:**

- ✅ Database schema

- ✅ Cache implementation
- ✅ API endpoints
- ✅ Chat integration

**Deliverables:**

- API работает
- Кеширование функционирует
- Можно использовать в чате

---

## 🔗 RAG Integration Path (v1.13.0)

### Переиспользование в RAG

```go
// v1.13.0 RAG будет импортировать web fetcher

import "internal/webfetch"

// RAG web data source
type WebDataSource struct {
    fetcher webfetch.WebFetcher
}

func (s *WebDataSource) Sync(ctx context.Context, sourceConfig WebSourceConfig) error {
    // Fetch web pages
    pages := []string{
        sourceConfig.BaseURL,
        // ... crawled URLs
    }
    
    for _, url := range pages {
        // Use v1.10 fetcher
        page, _ := s.fetcher.Fetch(ctx, url, webfetch.FetchOptions{})
        
        // NEW in v1.13: Chunk and embed
        chunks := s.chunker.Chunk(page.Content)
        embeddings := s.embedder.Embed(chunks)
        s.vectorDB.Store(embeddings)
    }

    
    return nil
}
```

**Преимущества:**

- ✅ Web fetcher готов и протестирован
- ✅ RAG получает надежный HTML parser
- ✅ Scheduled sync для web sources

---

## 📚 Dependencies

### Go Libraries

```go
// HTML parsing
"github.com/PuerkitoBio/goquery"  // jQuery-like HTML parsing

// HTTP client (stdlib)
"net/http"
"net/url"

// Rate limiting
"golang.org/x/time/rate"
```

---

## 📖 Success Metrics

- ✅ Fetch speed: <2s для большинства сайтов
- ✅ Parse accuracy: >90% для чистого контента
- ✅ SSRF prevention: 100% (блокировка private IPs)
- ✅ Rate limiting: 0 API blocks
- ✅ Test coverage: >85%

**Created:** 2025-01-16  
**Last Updated:** 2025-01-16  
**Status:** Ready for Implementation  
**Dependencies:** None (standalone)

