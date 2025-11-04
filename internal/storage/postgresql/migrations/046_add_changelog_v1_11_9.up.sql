
INSERT INTO changelogs (version, release_date, content) VALUES
('1.11.9', '2025-10-26', '## [1.11.9] - 2025-10-26

### Added
- **Enhanced Audit Logging**: Comprehensive audit trail for critical operations
  - User operations: creation, deletion, enable/disable (LogUserCreated, LogUserDeleted, LogUserUpdated)
  - API key operations: creation and deletion tracking (LogAPIKeyCreated, LogAPIKeyDeleted)
  - Tenant operations: creation, updates, -- deletion (LogTenantCreated, LogTenantUpdated, LogTenantDeleted)
  - Backup operations: creation and restoration tracking (LogBackupCreated, LogBackupRestored)
  - Performance monitoring: reduced update frequency from 5s to 10s for GPU and system metrics
  - WebUI performance: monitors now stop when not actively viewing System tab

### Technical
- Added AuditLogger integration to handlers: AdminUserHandler, UserHandler, TenantHandler, BackupHandler
- New audit methods in internal/services/audit/logger.go: LogUserUpdated(), -- LogTenantUpdated()
- Updated handler constructors to accept *audit.AuditLogger parameter
- Router injection of auditLogger into all relevant handlers
- WebUI optimization: admin.js now stops performance/GPU monitors when switching tabs

### Security
- **Audit trail for CRITICAL operations**: User deletion (data loss risk), Backup restoration (overwrites current data), Tenant deletion (organization data loss), API key operations (security credentials)
')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
    