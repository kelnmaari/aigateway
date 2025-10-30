
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.12.2', '2025-10-26', '## [1.12.2] - 2025-10-26

### Added
- **Advanced Rate Limiting**: Multi-scope rate limiting с sliding window algorithm
  - Scope support: Global, Tenant, User, API Key, Model-specific
  - Priority-based checking (API Key → User → Tenant → Model → Global)
  - Sliding window algorithm для точного подсчета requests (предотвращает burst attacks)
  - RFC 6585 compliance headers: X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset, X-RateLimit-Window, X-RateLimit-Scope, Retry-After
  - 429 Too Many Requests при превышении лимита

### Changed
- **Rate Limiting**: Базовая система расширена multi-scope support
- **Database Schema**: Новые таблицы rate_limits и rate_limit_usage для гибкой настройки

### Technical
- Новый сервис internal/services/ratelimit/sliding_window.go: SlidingWindowLimiter с in-memory cache
- Новый сервис internal/services/ratelimit/advanced_service.go: AdvancedRateLimiter с multi-scope checking
- Новый middleware internal/api/middleware/advanced_rate_limit.go: RFC 6585 headers support
- Data Models в internal/models/rate_limit.go
- Database Migration v48: rate_limits + rate_limit_usage tables

### Performance
- **Sliding Window Algorithm**: Более точный чем fixed window
- **In-Memory Cache**: < 1ms overhead на rate limit check');
    