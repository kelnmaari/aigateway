
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.11.4', '2025-10-25', '## [1.11.4] - 2025-10-25

### Added
- **AUDIT-01: Enhanced Audit Logging** 🔐
  - Comprehensive Security Events Logging для всех критичных операций
  - Structured Audit Events с полной трассировкой actor/target/action
  - Event Types (24 типа): LOGIN, API_KEY, TENANT, USER, BACKUP, PERMISSIONS
  - Severity Levels: info, warning, critical
  - Query API с фильтрами (event_type, severity, resource, date range)
  - CSV Export для compliance reporting
  - Statistics Dashboard с real-time метриками (24h)
  - Admin UI в WebUI с preview + full audit page
  - Retention Policy с auto-cleanup (90 days default, daily schedule)

### Technical
- Модули: models/audit.go, services/audit (logger, retention), handlers/audit.go, storage/sqlite/audit.go
- Database (Migration v40): CREATE TABLE audit_events (id, event_type, severity, actor, target, action, resource, status, metadata)
- 7 индексов для эффективных запросов
- API Routes: GET /api/admin/audit (query), /stats (metrics), /export (CSV)
- WebUI: web/admin-audit.html (dedicated page), web/admin.html (Audit tab)
- Convenience Methods: LogLogin, LogOIDCLogin, LogLDAPLogin, LogAPIKeyCreated, LogTenantMemberAdded, LogPermissionDenied
- Integration: AuthHandler с audit logging (LOGIN_SUCCESS, LOGIN_FAILED)');
	