
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.11.7', '2025-10-25', '## [1.11.7] - 2025-10-25

### Added

- **QUOTA-01: Usage Quotas System** 📊
  - **Flexible Quota System** для per-user и per-tenant limits
  - **Token Quotas**: Daily token limits (tokens_per_day), Monthly token limits (tokens_per_month), Automatic usage tracking с prompt/completion tokens
  - **Request Quotas**: Daily request limits (requests_per_day), Monthly request limits (requests_per_month), Concurrent request limiting (max_concurrent)
  - **Storage Quotas** (future-ready): Max file upload size (max_file_size), Max total storage per user/tenant (max_storage_bytes), Max conversations count (max_conversations)
  - **Model Restrictions**: Per-quota model allow-list (allowed_models), Block specific models for certain users/tenants
  - **Auto-Reset Logic**: Daily quota reset (24h sliding window), Monthly quota reset (calendar month boundary), Background reset при первом request after reset time
  - **Quota Service** (internal/services/quota/service.go): CheckQuota() - проверка before request processing, RecordUsage() - tracking actual usage after request, IncrementConcurrent() / DecrementConcurrent() - concurrent tracking, GetQuotaStats() - статистика для UI display
  - **Quota Middleware** (internal/api/middleware/quota.go): Автоматическая проверка квот для chat/completion endpoints, 429 Too Many Requests при quota exceeded, Concurrent request tracking with defer cleanup
  - **Prometheus Integration**: ollama_proxy_quota_usage - Current usage by target_id/type, ollama_proxy_quota_limit - Quota limits, ollama_proxy_quota_exceeded_total - Exceeded events counter, Periodic collection (30s interval) в MetricsCollector
  - **Admin API** (/api/admin/quotas): GET /quotas - List all quotas (filter by scope), POST /quotas - Create quota, GET /quotas/:id - Get quota details, PUT /quotas/:id - Update quota, DELETE /quotas/:id - Delete quota (cascade delete usage), GET /quotas/:id/usage - Get current usage
  - **User API** (/api/quota/me): Get current user''s quota stats with percentages, Tenant-scoped quota support
  - **Database Schema** (migration v43): quotas table - quota definitions, quota_usage table - usage tracking, Indexes for efficient queries по scope/target_id, Foreign key constraints с cascade delete
  - **Data Models**: Quota - quota definition (limits, scope, target), QuotaUsage - current usage counters, QuotaStats - computed stats для UI (percentages, remaining), QuotaCheck - result of quota validation

### Changed

- **Router**: Quota service и middleware инициализируются автоматически при наличии database
- **Metrics Collector**: Добавлен periodic collection для quota usage/limits (каждые 30s)

### Technical

- **Testing**: All internal/* package tests passing ✅ (кроме deprecated extractors tests)
- **Build**: Server binary собирается успешно с QUOTA-01 ✅
- **Race Detector**: Tests pass с -race flag ✅
- **SQLite Implementation**: Full QUOTA CRUD operations в internal/storage/sqlite/quotas.go
- **PostgreSQL**: Stubs added для будущей реализации
- **Transactions**: Transaction wrappers delegating to DB methods для quotas
');
    