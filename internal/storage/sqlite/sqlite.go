// Package sqlite provides SQLite implementation of the Database interface.
//
// This package implements a pure Go SQLite backend using modernc.org/sqlite driver.
// Features:
//   - WAL mode for better concurrency
//   - Connection pooling
//   - Prepared statements for security
//   - Automatic migrations
//
// Usage:
//
//	db, err := sqlite.New(config.SQLiteConfig{Path: "data/proxy.db"}, logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite" // Pure Go SQLite driver

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/storage"
)

// SQLiteDB представляет SQLite имплементацию Database interface
type SQLiteDB struct {
	db     *sql.DB
	config config.SQLiteConfig
	logger *logrus.Logger
	mu     sync.RWMutex // Для thread-safe операций

	// Prepared statements cache
	stmts map[string]*sql.Stmt
}

// New создает новый экземпляр SQLiteDB
func New(cfg config.SQLiteConfig, logger *logrus.Logger) (*SQLiteDB, error) {
	if logger == nil {
		logger = logrus.New()
	}

	db := &SQLiteDB{
		config: cfg,
		logger: logger,
		stmts:  make(map[string]*sql.Stmt),
	}

	return db, nil
}

// ========================================
// Connection Management
// ========================================

// Connect устанавливает соединение с SQLite БД
func (s *SQLiteDB) Connect(ctx context.Context) error {
	s.logger.WithField("path", s.config.Path).Info("Connecting to SQLite database")

	// Build DSN with pragmas
	dsn := fmt.Sprintf("file:%s?cache=shared&mode=rwc", s.config.Path)

	// Add pragmas as query parameters
	dsn += fmt.Sprintf("&_journal_mode=%s", s.config.JournalMode)
	dsn += fmt.Sprintf("&_busy_timeout=%d", s.config.BusyTimeout)
	dsn += fmt.Sprintf("&_cache_size=-%d", s.config.CacheSize) // Negative for KB

	// Open database
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite works best with single writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0) // Connections never expire

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping SQLite database: %w", err)
	}

	s.db = db

	// Apply additional pragmas
	if err := s.applyPragmas(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to apply pragmas: %w", err)
	}

	s.logger.Info("Successfully connected to SQLite database")
	return nil
}

// applyPragmas применяет дополнительные SQLite pragmas
func (s *SQLiteDB) applyPragmas(ctx context.Context) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",         // Enable foreign key constraints
		"PRAGMA synchronous = NORMAL",      // Balance between safety and speed
		"PRAGMA temp_store = MEMORY",       // Store temp tables in memory
		"PRAGMA mmap_size = 268435456",     // Memory-mapped I/O (256MB)
		"PRAGMA page_size = 4096",          // Optimal page size
		"PRAGMA auto_vacuum = INCREMENTAL", // Enable incremental vacuum
	}

	for _, pragma := range pragmas {
		if _, err := s.db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("failed to execute pragma '%s': %w", pragma, err)
		}
	}

	return nil
}

// Close закрывает соединение с БД
func (s *SQLiteDB) Close() error {
	s.logger.Info("Closing SQLite database connection")

	s.mu.Lock()
	defer s.mu.Unlock()

	// Close prepared statements
	for name, stmt := range s.stmts {
		if err := stmt.Close(); err != nil {
			s.logger.WithError(err).WithField("statement", name).Warn("Failed to close prepared statement")
		}
	}
	s.stmts = make(map[string]*sql.Stmt)

	// Close database
	if s.db != nil {
		return s.db.Close()
	}

	return nil
}

// Ping проверяет доступность БД
func (s *SQLiteDB) Ping(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	return s.db.PingContext(ctx)
}

// ========================================
// Transaction Support
// ========================================

// sqliteTx представляет SQLite транзакцию
type sqliteTx struct {
	tx     *sql.Tx
	db     *SQLiteDB
	logger *logrus.Logger
}

// BeginTx начинает новую транзакцию
func (s *SQLiteDB) BeginTx(ctx context.Context) (storage.Tx, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable, // SQLite default
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	s.logger.Debug("Transaction started")

	return &sqliteTx{
		tx:     tx,
		db:     s,
		logger: s.logger,
	}, nil
}

// Commit фиксирует транзакцию
func (tx *sqliteTx) Commit() error {
	if err := tx.tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tx.logger.Debug("Transaction committed")
	return nil
}

// Rollback откатывает транзакцию
func (tx *sqliteTx) Rollback() error {
	if err := tx.tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}

	tx.logger.Debug("Transaction rolled back")
	return nil
}

// ========================================
// Migrations
// ========================================

// Migrate применяет все pending миграции
func (s *SQLiteDB) Migrate(ctx context.Context) error {
	s.logger.Info("Starting database migration")

	// Create migrations table if not exists
	if err := s.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := s.GetMigrationVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	s.logger.WithField("current_version", currentVersion).Info("Current migration version")

	// Apply migrations
	migrations := s.getMigrations()
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue // Already applied
		}

		s.logger.WithField("version", migration.Version).Info("Applying migration")

		if err := s.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		s.logger.WithField("version", migration.Version).Info("Migration applied successfully")
	}

	finalVersion, _ := s.GetMigrationVersion(ctx)
	s.logger.WithField("version", finalVersion).Info("Database migration completed")

	return nil
}

// GetMigrationVersion возвращает текущую версию схемы
func (s *SQLiteDB) GetMigrationVersion(ctx context.Context) (int, error) {
	var version int
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM migrations
	`).Scan(&version)

	if err != nil {
		return 0, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, nil
}

// createMigrationsTable создает таблицу для отслеживания миграций
func (s *SQLiteDB) createMigrationsTable(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)

	return err
}

// migration представляет одну миграцию
type migration struct {
	Version int
	Name    string
	SQL     string
}

// getMigrations возвращает список всех миграций
func (s *SQLiteDB) getMigrations() []migration {
	return []migration{
		{
			Version: 1,
			Name:    "initial_schema",
			SQL:     s.getInitialSchemaMigration(),
		},
		{
			Version: 2,
			Name:    "add_mcp_servers_table",
			SQL:     s.getMCPServersMigration(),
		},
		{
			Version: 3,
			Name:    "add_changelogs_with_data",
			SQL:     s.getChangelogsMigrationWithData(),
		},
		{
			Version: 4,
			Name:    "add_changelog_v1_4_11",
			SQL:     s.getChangelogV1411Migration(),
		},
		{
			Version: 5,
			Name:    "populate_all_changelogs",
			SQL:     s.getPopulateAllChangelogsMigration(),
		},
		// Добавляем новые миграции здесь по мере необходимости
	}
}

// applyMigration применяет одну миграцию
func (s *SQLiteDB) applyMigration(ctx context.Context, m migration) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO migrations (version, applied_at)
		VALUES (?, CURRENT_TIMESTAMP)
	`, m.Version); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// getInitialSchemaMigration возвращает SQL для начальной схемы БД
func (s *SQLiteDB) getInitialSchemaMigration() string {
	return `
-- ========================================
-- Users Table (AUTH-05)
-- ========================================
CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	email TEXT NOT NULL UNIQUE,
	full_name TEXT NOT NULL,
	password_hash TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'active',
	is_admin BOOLEAN NOT NULL DEFAULT 0,
	is_active BOOLEAN NOT NULL DEFAULT 1,
	verified BOOLEAN NOT NULL DEFAULT 0,
	verified_at TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_login TIMESTAMP,
	preferences TEXT NOT NULL DEFAULT '{}', -- JSON
	metadata TEXT -- JSON
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);

-- ========================================
-- Tenants Table (AUTH-05: Multi-Tenancy)
-- ========================================
CREATE TABLE IF NOT EXISTS tenants (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	slug TEXT NOT NULL UNIQUE,
	type TEXT NOT NULL DEFAULT 'personal', -- personal, organization
	description TEXT,
	owner_id TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'active',
	is_active BOOLEAN NOT NULL DEFAULT 1,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	settings TEXT NOT NULL DEFAULT '{}', -- JSON
	metadata TEXT, -- JSON
	FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_tenants_slug ON tenants(slug);
CREATE INDEX idx_tenants_owner_id ON tenants(owner_id);
CREATE INDEX idx_tenants_type ON tenants(type);

-- ========================================
-- Tenant Members Table (AUTH-05)
-- ========================================
CREATE TABLE IF NOT EXISTS tenant_members (
	tenant_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT 'member', -- owner, admin, member, viewer
	joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	left_at TIMESTAMP,
	invited_by TEXT,
	metadata TEXT, -- JSON
	PRIMARY KEY (tenant_id, user_id),
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_tenant_members_user_id ON tenant_members(user_id);
CREATE INDEX idx_tenant_members_role ON tenant_members(role);

-- ========================================
-- API Keys Table (Обновленная для multi-tenancy)
-- ========================================
CREATE TABLE IF NOT EXISTS api_keys (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT,
	key_hash TEXT NOT NULL UNIQUE,
	user_id TEXT, -- NULL для global keys (backward compatibility)
	tenant_id TEXT, -- NULL для personal keys
	scope TEXT NOT NULL DEFAULT 'global', -- global, personal, tenant
	models TEXT NOT NULL DEFAULT '["*"]', -- JSON array
	permissions TEXT NOT NULL DEFAULT '[]', -- JSON array
	rate_limits TEXT NOT NULL DEFAULT '{}', -- JSON
	status TEXT NOT NULL DEFAULT 'active',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	expires_at TIMESTAMP,
	last_used_at TIMESTAMP,
	revoked_at TIMESTAMP,
	revoked_reason TEXT,
	metadata TEXT, -- JSON
	usage TEXT NOT NULL DEFAULT '{}', -- JSON (APIKeyUsage struct)
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_scope ON api_keys(scope);
CREATE INDEX idx_api_keys_status ON api_keys(status);

-- ========================================
-- Conversations Table (WEBUI-03: Chat)
-- ========================================
CREATE TABLE IF NOT EXISTS conversations (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	user_id TEXT NOT NULL,
	tenant_id TEXT, -- NULL для personal conversations
	model TEXT NOT NULL,
	temperature REAL,
	system_prompt TEXT,
	status TEXT NOT NULL DEFAULT 'active',
	is_archived BOOLEAN NOT NULL DEFAULT 0,
	is_pinned BOOLEAN NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_message_at TIMESTAMP,
	message_count INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	metadata TEXT, -- JSON
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);
CREATE INDEX idx_conversations_tenant_id ON conversations(tenant_id);
CREATE INDEX idx_conversations_status ON conversations(status);
CREATE INDEX idx_conversations_updated_at ON conversations(updated_at DESC);

-- ========================================
-- Messages Table (WEBUI-03: Chat)
-- ========================================
CREATE TABLE IF NOT EXISTS messages (
	id TEXT PRIMARY KEY,
	conversation_id TEXT NOT NULL,
	role TEXT NOT NULL, -- user, assistant, system, tool
	content TEXT NOT NULL,
	model TEXT,
	temperature REAL,
	tool_calls TEXT, -- JSON array
	tool_call_id TEXT,
	prompt_tokens INTEGER DEFAULT 0,
	completion_tokens INTEGER DEFAULT 0,
	total_tokens INTEGER DEFAULT 0,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	metadata TEXT, -- JSON
	FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_role ON messages(role);
CREATE INDEX idx_messages_created_at ON messages(created_at);

-- ========================================
-- API Usage Table (WEBUI-04: Dashboard)
-- ========================================
CREATE TABLE IF NOT EXISTS api_usage (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	tenant_id TEXT, -- NULL для personal usage
	api_key_id TEXT NOT NULL,
	endpoint TEXT NOT NULL,
	method TEXT NOT NULL,
	model TEXT NOT NULL,
	status_code INTEGER NOT NULL,
	success BOOLEAN NOT NULL,
	error_message TEXT,
	prompt_tokens INTEGER DEFAULT 0,
	completion_tokens INTEGER DEFAULT 0,
	total_tokens INTEGER DEFAULT 0,
	duration_ms INTEGER NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	user_agent TEXT,
	ip_address TEXT,
	conversation_id TEXT,
	metadata TEXT, -- JSON
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
	FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE,
	FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_api_usage_user_id ON api_usage(user_id);
CREATE INDEX IF NOT EXISTS idx_api_usage_tenant_id ON api_usage(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_usage_api_key_id ON api_usage(api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_usage_created_at ON api_usage(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_usage_endpoint ON api_usage(endpoint);
CREATE INDEX IF NOT EXISTS idx_api_usage_model ON api_usage(model);

-- Composite indexes for better query performance (v1.4.6+)
CREATE INDEX IF NOT EXISTS idx_api_usage_user_created ON api_usage(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_usage_tenant_created ON api_usage(tenant_id, created_at DESC) WHERE tenant_id IS NOT NULL;
	`
}

// getMCPServersMigration возвращает SQL для создания таблицы mcp_servers (v1.4.5)
func (s *SQLiteDB) getMCPServersMigration() string {
	return `
-- ========================================
-- MCP Servers Table (WEBUI-07: v1.4.5)
-- ========================================
CREATE TABLE IF NOT EXISTS mcp_servers (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT NOT NULL,
	category TEXT NOT NULL,
	installation_guide TEXT NOT NULL,
	website_url TEXT,
	github_url TEXT,
	tags TEXT, -- JSON array stored as TEXT
	is_active BOOLEAN NOT NULL DEFAULT 1,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mcp_servers_category ON mcp_servers(category);
CREATE INDEX idx_mcp_servers_is_active ON mcp_servers(is_active);
CREATE INDEX idx_mcp_servers_created_at ON mcp_servers(created_at DESC);
	`
}

// getChangelogsMigrationWithData возвращает SQL для создания таблицы changelogs с данными из CHANGELOG.md (v1.4.10+)
func (s *SQLiteDB) getChangelogsMigrationWithData() string {
	return `
-- ========================================
-- Changelogs Table (System Info: v1.4.11+)
-- ========================================
CREATE TABLE IF NOT EXISTS changelogs (
	version TEXT PRIMARY KEY,
	release_date DATE NOT NULL,
	content TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_changelogs_release_date ON changelogs(release_date DESC);

-- ========================================
-- Initial Changelog Data from CHANGELOG.md
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.4.10', '2025-10-10', '## [1.4.10] - 2025-10-10

### Added
- **"Remember Me" Функция при входе**:
  - Checkbox "Не выходить из системы 24 часа" на форме логина
  - При включении галочки access token живет **24 часа** вместо 15 минут
  - Refresh token продолжает работать 7 дней как и раньше
  - Логирование использования remember_me в JWT и auth service

### Changed
- **Backend**: LoginRequest, GenerateTokenPair, JWT логирование
- **Frontend**: Добавлен checkbox на форму входа

### Technical
- Без галочки: Access token 15 минут, Refresh token 7 дней
- С галочкой: Access token 24 часа, Refresh token 7 дней
- Фактическая длительность сессии: до 7 дней'),

('1.4.9', '2025-10-10', '## [1.4.9] - 2025-10-10

### Fixed
- **API Keys Status Display**:
  - Problem: Свежесозданные ключи отображались как "Inactive"
  - Solution: Frontend использует поле status из API response
  - Ключ активен если status === ''active'' И не истек срок'),

('1.4.8', '2025-10-10', '## [1.4.8] - 2025-10-10

### Added
- **Debug Logging для Usage Tracking**
- Логирование user_id, api_key_id, model, tokens, success

### Fixed
- **API Key ID для JWT аутентификации**: используется "jwt_auth" ID'),

('1.4.7', '2025-10-10', '## [1.4.7] - 2025-10-10

### Fixed
- **CRITICAL: Usage Statistics Performance**
  - Problem: Запросы 5+ секунд, context canceled
  - Solution: Оптимизированы SQL с SUM(CASE WHEN ...)
  - Performance: < 100ms для большинства запросов'),

('1.4.6', '2025-10-10', '## [1.4.6] - 2025-10-10

### Added
- **API Usage Tracking**: Полная интеграция middleware
- Автоматическая запись в api_usage таблицу
- Сбор метрик: user_id, model, tokens, duration'),

('1.4.5', '2025-10-10', '## [1.4.5] - 2025-10-10

### Added
- **MCP Servers Catalog**: Admin-managed catalog
- Database: таблица mcp_servers
- Frontend: web/mcp.html с публичной страницей'),

('1.4.4', '2025-10-10', '## [1.4.4] - 2025-10-10

### Added
- **Enhanced Models Information**: Accordion UI
- Lazy loading через /api/admin/models/:name/details
- Секции: Basic Info, Specifications, Template, Modelfile'),

('1.4.3', '2025-10-10', '## [1.4.3] - 2025-10-10

### Added
- **Build Version Information**: Реальная версия через ldflags
- Флаг -version для показа информации о билде
- VERSION файл как единственный источник версии'),

('1.4.2', '2025-10-10', '## [1.4.2] - 2025-10-10

### Security
- **TUI Admin Key Configuration**: Убраны hardcoded keys
- Admin token загружается из конфигурации
- Добавлен флаг --config для указания пути'),

('1.4.1', '2025-10-10', '## [1.4.1] - 2025-10-10

### Added
- **Model Copy Button**: Кнопка копирования модели
- Clipboard API с fallback
- Toast notifications для feedback'),

('1.4.0', '2025-10-08', '## [1.4.0] - 2025-10-08

WebUI Enhancements & Code Quality - Base Release

Включает: Model Copy, TUI Admin Config, Build Version, Enhanced Models, MCP Catalog'),

('1.3.0', '2025-10-06', '## [1.3.0] - 2025-10-06

User Experience & Multi-Tenancy

### Added
- Database Abstraction Layer (SQLite + PostgreSQL)
- User Authentication & Multi-Tenancy (JWT, RBAC)
- Interactive Chat Interface
- User Dashboard
- Cross-platform Build System'),

('1.2.0', '2025-10-01', '## [1.2.0] - 2025-10-01

Enhanced Monitoring & Management

### Added
- TUI Request Monitor
- WebUI Metrics Visualization
- Advanced Logs Features
- Enhanced API Key Management');
	`
}

// getChangelogV1411Migration добавляет только версию 1.4.11 (инкрементальная миграция v4)
func (s *SQLiteDB) getChangelogV1411Migration() string {
	return `
-- ========================================
-- Fix Changelog Table Structure (Migration v4)
-- ========================================

-- Удаляем старую таблицу если она была с неправильной структурой
DROP TABLE IF EXISTS changelogs;

-- Создаем таблицу с правильной структурой
CREATE TABLE changelogs (
	version TEXT PRIMARY KEY,
	release_date DATE NOT NULL,
	content TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_changelogs_release_date ON changelogs(release_date DESC);

-- Добавляем только версию 1.4.11
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.4.11', '2025-10-10', '## [1.4.11] - 2025-10-10

### Added
- **Система "О Системе"**: Новая страница
  - Отображение версии, git commit, build date
  - Accordion UI для changelog всех версий
  - Пункт "О Системе" в меню профиля

### Changed
- **Backend**: Migration, changelog handler, API endpoints
- **Frontend**: about.html, navbar, accordion UI

### Technical
- Database: таблица changelogs
- API: /api/system/info, /api/system/changelogs');
	`
}

// getPopulateAllChangelogsMigration добавляет все версии changelog (миграция v5)
func (s *SQLiteDB) getPopulateAllChangelogsMigration() string {
	return `
-- ========================================
-- Populate All Changelogs (Migration v5)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
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
- Advanced Logs');
	`
}

// ========================================
// NOTE: CRUD Operations Implementation
// ========================================
//
// All CRUD operations are implemented in separate files:
//   - users.go:         Users CRUD
//   - tenants.go:       Tenants & TenantMembers CRUD
//   - apikeys.go:       API Keys CRUD
//   - conversations.go: Conversations & Messages CRUD
//   - usage.go:         Usage Stats & Analytics
//
// This file contains only:
//   - Connection management
//   - Transaction support
//   - Migration system
// ========================================

// Ensure sqliteTx implements storage.Tx interface
var _ storage.Tx = (*sqliteTx)(nil)

// ========================================
// Transaction Methods (Delegation)
// ========================================
//
// Transaction methods are implemented in transactions.go
// They delegate to the parent DB's methods
// ========================================
