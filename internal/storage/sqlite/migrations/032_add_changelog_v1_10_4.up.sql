
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.10.4', '2025-10-24', '## [1.10.4] - 2025-10-24

### Added
- **WEB-FETCH-01: Web Content Fetcher & Chat Integration** 🌐
  - Web Fetch Infrastructure (internal/webfetch/): HTTP client с retry logic, URL validator с SSRF protection, Rate limiter для доменов, HTML parser (goquery), Metadata extraction (Open Graph, Twitter Card)
  - **Chat Integration** ✨ MAJOR FEATURE: URL auto-detection в сообщениях, Automatic fetch при детектировании URL, Context enrichment для LLM, Работает в streaming и non-streaming режимах, Max 2 URLs per message, 15s timeout per URL
  - API Endpoints: POST /api/web/fetch, POST /api/web/fetch/batch (до 10 URLs)
  - Database Migration v31: web_fetches table, web_fetch_rate_limits table
  - Configuration: web_fetch.enabled, SSRF protection, Rate limiting, Cache settings

### Changed
- ChatHandler: Добавлен webfetchIntegration field, Новый метод enrichMessagesWithWebContent() для auto-fetch, Работает параллельно с file enrichment

### Technical
- Testing: URL detection tests (12 cases), URL validation tests (10 cases), All tests passing ✅
- Dependencies: github.com/PuerkitoBio/goquery v1.10.3, golang.org/x/time/rate
- Security: SSRF protection (блокирует private IPs, localhost), DNS resolution check, Per-domain rate limiting, Content size limits (10MB)
- Performance: Cache с TTL (1 hour), Connection pooling, Truncation для контекста (3000 chars), Parallel fetch
- Documentation: docs/WEB_FETCH_QUICKSTART.md');
	