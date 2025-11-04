
-- ========================================
-- Populate All Changelogs (Migration v5)
-- ========================================

INSERT INTO changelogs (version, release_date, content) VALUES
('1.4.10', '2025-10-10', '## [1.4.10] - 2025-10-10

### Added
- **"Remember Me"**: Checkbox 24 часа на форме входа
- Access token: 15m → 24h при включенной галочке

### Changed
- Backend: LoginRequest, GenerateTokenPair
- Frontend: Checkbox на login.html'),

('1.4.9', '2025-10-10', '## [1.4.9] - 2025-10-10

### Fixed
- **API Keys Status**: Ключи отображались Inactive
- Frontend использует status из API'),

('1.4.8', '2025-10-10', '## [1.4.8] - 2025-10-10

### Added
- **Debug Logging**: Usage Tracking

### Fixed
- API Key ID для JWT: "jwt_auth"'),

('1.4.7', '2025-10-10', '## [1.4.7] - 2025-10-10

### Fixed
- **CRITICAL**: Usage Statistics Performance
- SQL оптимизация: < 100ms'),

('1.4.6', '2025-10-10', '## [1.4.6] - 2025-10-10

### Added
- **API Usage Tracking**: middleware
- Запись в api_usage таблицу'),

('1.4.5', '2025-10-10', '## [1.4.5] - 2025-10-10

### Added
- **MCP Catalog**: Admin-managed
- Frontend: mcp.html'),

('1.4.4', '2025-10-10', '## [1.4.4] - 2025-10-10

### Added
- **Enhanced Models**: Accordion UI
- Lazy loading деталей'),

('1.4.3', '2025-10-10', '## [1.4.3] - 2025-10-10

### Added
- **Build Version**: ldflags
- VERSION файл'),

('1.4.2', '2025-10-10', '## [1.4.2] - 2025-10-10

### Security
- TUI Admin Key от конфигурации'),

('1.4.1', '2025-10-10', '## [1.4.1] - 2025-10-10

### Added
- Model Copy Button
- Clipboard API'),

('1.4.0', '2025-10-08', '## [1.4.0] - 2025-10-08

WebUI Enhancements & Code Quality'),

('1.3.0', '2025-10-06', '## [1.3.0] - 2025-10-06

User Experience & Multi-Tenancy

### Added
- Database Layer (SQLite + PostgreSQL)
- JWT Authentication & RBAC
- Chat Interface'),

('1.2.0', '2025-10-01', '## [1.2.0] - 2025-10-01

Enhanced Monitoring & Management

### Added
- TUI Request Monitor
- WebUI Metrics
- Advanced Logs')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	