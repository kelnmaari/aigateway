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
	"time"

	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite" // Pure Go SQLite driver

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/models"
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
// Reports Statistics Methods (v1.6.3+) - Delegation to DB
// ========================================

func (tx *sqliteTx) GetUsageStats(ctx context.Context, start, end time.Time) (*models.UsageReportStats, error) {
	return tx.db.GetUsageStats(ctx, start, end)
}

func (tx *sqliteTx) GetPerformanceStats(ctx context.Context, start, end time.Time) (*models.PerformanceReportStats, error) {
	return tx.db.GetPerformanceStats(ctx, start, end)
}

func (tx *sqliteTx) CountActiveUsers(ctx context.Context, period time.Duration) (int, error) {
	return tx.db.CountActiveUsers(ctx, period)
}

func (tx *sqliteTx) CountTotalUsers(ctx context.Context) (int, error) {
	return tx.db.CountTotalUsers(ctx)
}

func (tx *sqliteTx) CountActiveAPIKeys(ctx context.Context) (int, error) {
	return tx.db.CountActiveAPIKeys(ctx)
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
		{
			Version: 6,
			Name:    "add_changelog_v1_5_1",
			SQL:     s.getChangelogV151Migration(),
		},
		{
			Version: 7,
			Name:    "add_changelog_v1_5_2",
			SQL:     s.getChangelogV152Migration(),
		},
		{
			Version: 8,
			Name:    "add_changelog_v1_5_3",
			SQL:     s.getChangelogV153Migration(),
		},
		{
			Version: 9,
			Name:    "add_changelog_v1_5_4",
			SQL:     s.getChangelogV154Migration(),
		},
		{
			Version: 10,
			Name:    "add_changelog_v1_5_5",
			SQL:     s.getChangelogV155Migration(),
		},
		{
			Version: 11,
			Name:    "add_changelog_v1_5_6",
			SQL:     s.getChangelogV156Migration(),
		},
		{
			Version: 12,
			Name:    "add_changelog_v1_5_7",
			SQL:     s.getChangelogV157Migration(),
		},
		{
			Version: 13,
			Name:    "add_changelog_v1_5_8",
			SQL:     s.getChangelogV158Migration(),
		},
		{
			Version: 14,
			Name:    "add_changelog_v1_5_9",
			SQL:     s.getChangelogV159Migration(),
		},
		{
			Version: 15,
			Name:    "add_changelog_v1_5_10",
			SQL:     s.getChangelogV1510Migration(),
		},
		{
			Version: 16,
			Name:    "fix_api_usage_nullable_api_key_id",
			SQL:     s.getFixAPIUsageNullableAPIKeyIDMigration(),
		},
		{
			Version: 17,
			Name:    "add_changelogs_v1_5_11_to_v1_5_16",
			SQL:     s.getAddChangelogsV1511ToV1516Migration(),
		},
		{
			Version: 18,
			Name:    "add_changelog_v1_6_1",
			SQL:     s.getAddChangelogV161Migration(),
		},
		{
			Version: 19,
			Name:    "add_changelog_v1_6_2",
			SQL:     s.getAddChangelogV162Migration(),
		},
		{
			Version: 20,
			Name:    "add_changelog_v1_6_3",
			SQL:     s.getAddChangelogV163Migration(),
		},
		{
			Version: 21,
			Name:    "add_model_configs_table",
			SQL:     s.getModelConfigsTableMigration(),
		},
		{
			Version: 22,
			Name:    "add_changelog_v1_9_1",
			SQL:     s.getAddChangelogV191Migration(),
		},
		{
			Version: 23,
			Name:    "add_changelog_v1_9_2",
			SQL:     s.getAddChangelogV192Migration(),
		},
		{
			Version: 24,
			Name:    "add_changelog_v1_9_3",
			SQL:     s.getAddChangelogV193Migration(),
		},
		{
			Version: 25,
			Name:    "update_changelog_v1_9_3_final",
			SQL:     s.getUpdateChangelogV193FinalMigration(),
		},
		{
			Version: 26,
			Name:    "add_files_table",
			SQL:     s.getFilesTableMigration(),
		},
		{
			Version: 27,
			Name:    "add_message_files_junction",
			SQL:     s.getMessageFilesJunctionMigration(),
		},
		{
			Version: 28,
			Name:    "add_changelog_v1_10_0",
			SQL:     s.getAddChangelogV1100Migration(),
		},
		{
			Version: 29,
			Name:    "add_changelog_v1_9_4",
			SQL:     s.getAddChangelogV194Migration(),
		},
		{
			Version: 30,
			Name:    "add_changelog_v1_10_3",
			SQL:     s.getAddChangelogV1103Migration(),
		},
		{
			Version: 31,
			Name:    "add_web_fetches_table",
			SQL:     s.getWebFetchesMigration(),
		},
		{
			Version: 32,
			Name:    "add_changelog_v1_10_4",
			SQL:     s.getAddChangelogV1104Migration(),
		},
		{
			Version: 33,
			Name:    "add_changelog_v1_10_5",
			SQL:     s.getAddChangelogV1105Migration(),
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
('1.5.16', '2025-10-11', '## [1.5.16] - 2025-10-11

### Changed
- **All WebUI Pages**: Система уведомлений внедрена во все 8 страниц
  - dashboard, chat, profile, tenants, usage, api-keys, mcp, about
  - ~10 alert() → toast notifications
  - ~8 confirm() → modal confirmations
  - Единообразный UX на всех страницах

### Technical
- notifications.css и notifications.js подключены во все HTML
- Консистентный порядок загрузки скриптов
- toast.* и modal.* API используется везде'),

('1.5.15', '2025-10-11', '## [1.5.15] - 2025-10-11

### Added
- **Toast Notification System**: Профессиональная система всплывающих уведомлений
  - Уведомления справа снизу: зеленые (успех, 10с), красные (ошибка, 60с)
  - Желтые (предупреждения) и синие (информация)
  - Возможность закрыть вручную, плавные анимации
  - Поддержка темной темы и mobile

- **Modal Confirmation System**: Модальные окна для подтверждения действий
  - Заменяют стандартные confirm dialogs
  - confirm(), danger(), warning() с Promise-based API
  - Красивый дизайн с иконками

- **Global API**: window.toast и window.modal для всех страниц
  - toast.success/error/warning/info(message)
  - await modal.confirm/danger/warning(message, title)

### Changed
- **Admin Panel**: Полностью переведен на новую систему уведомлений
  - Все alert() → toast notifications
  - Все confirm() → modal confirmations
  - Улучшен UX для операций backup, users, API keys, MCP

### Technical
- Class-based architecture, HTML escape для XSS
- Auto-stacking для multiple toasts
- CSS transitions 300ms, backdrop blur
- Mobile-responsive, dark theme support'),

('1.5.14', '2025-10-11', '## [1.5.14] - 2025-10-11

### Added
- **Backup & Restore System**: Полнофункциональная система резервного копирования и восстановления БД
  - POST /api/admin/backup - создание бэкапа с автоматическим timestamp
  - GET /api/admin/backups - список доступных бэкапов
  - GET /api/admin/backup/:filename - скачивание backup файла
  - POST /api/admin/restore/:filename - восстановление из бэкапа
  - DELETE /api/admin/backup/:filename - удаление старых бэкапов
  - Автоматическая ротация: хранение последних 10 бэкапов
  - Safety backup перед restore операцией
  - Бэкапы сохраняются в ./data/backups/ директорию

### Changed
- internal/api/handlers/backup.go - новый handler для backup операций
- internal/api/router/router.go - добавлены роуты для backup/restore в admin API
- Все backup операции требуют JWT аутентификацию + admin role

### Technical
- Использование io.Copy для эффективного копирования больших файлов
- Path security checks для защиты от path traversal
- Atomic restore with rollback на случай ошибки
- File sync после записи для data integrity
- Structured logging для всех backup операций'),

('1.5.13', '2025-10-11', '## [1.5.13] - 2025-10-11

### Changed
- **Error Type Checking**: Улучшена обработка типов ошибок в admin handlers
  - Реализована функция isNotFoundError() с использованием errors.As
  - Добавлены функции isAlreadyExistsError, isInvalidDataError, isPermissionError
  - Type-safe error checking вместо string comparison

### Technical
- Использование Go 1.20+ errors.As() для type assertions
- Proper unwrapping of wrapped errors
- Удален TODO комментарий из admin.go'),

('1.5.12', '2025-10-11', '## [1.5.12] - 2025-10-11

### Fixed
- **BUG-03: WebUI Chat Usage Tracking**: FOREIGN KEY constraint failed при использовании WebUI чата
  - Проблема: api_key_id ссылался на несуществующие ключи (jwt_auth, unknown)
  - Решение: Сделать api_key_id nullable с ON DELETE SET NULL
  - Теперь статистика WebUI чата корректно учитывается

### Changed
- Database Schema: api_key_id TEXT nullable (было NOT NULL)
- FOREIGN KEY constraint: ON DELETE SET NULL (было CASCADE)
- Миграция v16 автоматически конвертирует jwt_auth/unknown в NULL

### Technical
- Migration v16: создает временную таблицу с правильной структурой
- Автоматическая конвертация существующих записей с fake keys
- Пересоздание индексов с учетом nullable значений'),

('1.5.11', '2025-10-11', '## [1.5.11] - 2025-10-11

### Fixed
- **BUG-02: Tenant API Keys Creation**: 404 при создании API ключей для организации
  - Добавлены endpoints для управления tenant API keys
  - GET /api/tenants/:id/api-keys - получение списка ключей организации
  - POST /api/tenants/:id/api-keys - создание ключа организации
  - DELETE /api/tenants/:id/api-keys/:key_id - удаление ключа организации

### Added
- **Tenant API Keys Management**: Полноценное управление API ключами организаций
  - Handler ListTenantAPIKeys для получения списка ключей
  - Handler CreateTenantAPIKey для создания ключей с правами owner/admin
  - Handler DeleteTenantAPIKey для удаления ключей с проверкой владения
  - Автоматическая проверка прав доступа (owner/admin only)
  - Валидация принадлежности ключа к tenant при удалении

### Security
- **Role-based Access Control**: Только owner и admin могут создавать/удалять tenant API keys
- **Tenant Ownership Verification**: Проверка принадлежности ключа к tenant перед удалением
- **Membership Check**: Проверка членства пользователя в tenant для всех операций

### Technical
- Default rate limits для tenant keys: 100 req/min, 5000 req/hour, 50000 req/day
- Key scope автоматически устанавливается в APIKeyScopeTenant
- TenantID связывается с API key через foreign key
- Plaintext key возвращается только один раз при создании'),

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

// getFixAPIUsageNullableAPIKeyIDMigration исправляет FOREIGN KEY constraint для api_key_id (v1.5.12)
// getAddChangelogsV1511ToV1516Migration adds changelog entries for versions 1.5.11-1.5.16 (Migration v17)
func (s *SQLiteDB) getAddChangelogsV1511ToV1516Migration() string {
	return `
-- ========================================
-- Add Changelogs for v1.5.11 - v1.5.16 (Migration v17)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.16', '2025-10-11', '## [1.5.16] - 2025-10-11

### Changed
- **All WebUI Pages**: Система уведомлений внедрена во все 8 страниц
  - dashboard, chat, profile, tenants, usage, api-keys, mcp, about
  - ~10 alert() → toast notifications
  - ~8 confirm() → modal confirmations
  - Единообразный UX на всех страницах

### Technical
- notifications.css и notifications.js подключены во все HTML
- Консистентный порядок загрузки скриптов
- toast.* и modal.* API используется везде'),

('1.5.15', '2025-10-11', '## [1.5.15] - 2025-10-11

### Added
- **Toast Notification System**: Профессиональная система всплывающих уведомлений
  - Уведомления справа снизу: зеленые (успех, 10с), красные (ошибка, 60с)
  - Желтые (предупреждения) и синие (информация)
  - Возможность закрыть вручную, плавные анимации
  - Поддержка темной темы и mobile

- **Modal Confirmation System**: Модальные окна для подтверждения действий
  - Заменяют стандартные confirm dialogs
  - confirm(), danger(), warning() с Promise-based API
  - Красивый дизайн с иконками

- **Global API**: window.toast и window.modal для всех страниц
  - toast.success/error/warning/info(message)
  - await modal.confirm/danger/warning(message, title)

### Changed
- **Admin Panel**: Полностью переведен на новую систему уведомлений
  - Все alert() → toast notifications
  - Все confirm() → modal confirmations
  - Улучшен UX для операций backup, users, API keys, MCP

### Technical
- Class-based architecture, HTML escape для XSS
- Auto-stacking для multiple toasts
- CSS transitions 300ms, backdrop blur
- Mobile-responsive, dark theme support'),

('1.5.14', '2025-10-11', '## [1.5.14] - 2025-10-11

### Added
- **Backup & Restore System**: Полнофункциональная система резервного копирования и восстановления БД
  - POST /api/admin/backup - создание бэкапа с автоматическим timestamp
  - GET /api/admin/backups - список доступных бэкапов
  - GET /api/admin/backup/:filename - скачивание backup файла
  - POST /api/admin/restore/:filename - восстановление из бэкапа
  - DELETE /api/admin/backup/:filename - удаление старых бэкапов
  - Автоматическая ротация: хранение последних 10 бэкапов
  - Safety backup перед restore операцией
  - Бэкапы сохраняются в ./data/backups/ директорию

### Changed
- internal/api/handlers/backup.go - новый handler для backup операций
- internal/api/router/router.go - добавлены роуты для backup/restore в admin API
- Все backup операции требуют JWT аутентификацию + admin role

### Technical
- Использование io.Copy для эффективного копирования больших файлов
- Path security checks для защиты от path traversal
- Atomic restore with rollback на случай ошибки
- File sync после записи для data integrity
- Structured logging для всех backup операций'),

('1.5.13', '2025-10-11', '## [1.5.13] - 2025-10-11

### Changed
- **Error Type Checking**: Улучшена обработка типов ошибок в admin handlers
  - Реализована функция isNotFoundError() с использованием errors.As
  - Добавлены функции isAlreadyExistsError, isInvalidDataError, isPermissionError
  - Type-safe error checking вместо string comparison

### Technical
- Использование Go 1.20+ errors.As() для type assertions
- Proper unwrapping of wrapped errors
- Удален TODO комментарий из admin.go'),

('1.5.12', '2025-10-11', '## [1.5.12] - 2025-10-11

### Fixed
- **BUG-03: WebUI Chat Usage Tracking**: FOREIGN KEY constraint failed при использовании WebUI чата
  - Схема api_usage.api_key_id теперь nullable
  - Foreign key с ON DELETE SET NULL
  - JWT authenticated requests теперь используют NULL вместо jwt_auth/unknown
  - Миграция v16: конвертация существующих данных

### Changed
- models.APIUsage.APIKeyID изменен с string на *string
- middleware.UsageTracking обновлен для NULL значений
- internal/storage/sqlite/sqlite.go: новая миграция v16

### Technical
- Правильная обработка nullable fields в Go (*string)
- Database migration с data conversion
- Foreign key constraints с ON DELETE SET NULL'),

('1.5.11', '2025-10-11', '## [1.5.11] - 2025-10-11

### Fixed
- **BUG-02: Tenant API Keys Creation**: 404 ошибка при создании API ключей для организаций
  - Реализованы недостающие endpoints для tenant API keys
  - POST /api/tenants/:id/api-keys - создание ключа организации
  - GET /api/tenants/:id/api-keys - список ключей организации
  - DELETE /api/tenants/:id/api-keys/:key_id - удаление ключа

### Added
- internal/api/handlers/tenant.go: три новых метода
  - ListTenantAPIKeys с проверкой membership
  - CreateTenantAPIKey с owner/admin role check
  - DeleteTenantAPIKey с key ownership validation

### Changed
- internal/api/router/router.go: добавлены новые роуты
- Улучшена валидация прав доступа (owner/admin)
- Маскирование sensitive данных в ListTenantAPIKeys

### Technical
- Robust access control с role-based checks
- Proper key ownership validation
- Secure key generation с GenerateAPIKeyWithID
- Default rate limits для tenant keys');
`
}

func (s *SQLiteDB) getFixAPIUsageNullableAPIKeyIDMigration() string {
	return `
-- ========================================
-- Fix API Usage Table: Nullable api_key_id (Migration v6: BUG-03 v1.5.12)
-- ========================================
-- Problem: FOREIGN KEY constraint failed при использовании WebUI чата
-- Solution: Сделать api_key_id nullable с ON DELETE SET NULL

-- Шаг 1: Создаем временную таблицу с правильной структурой
CREATE TABLE api_usage_new (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	tenant_id TEXT, -- NULL для personal usage
	api_key_id TEXT, -- ✅ NULLABLE для JWT auth (было NOT NULL)
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
	FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE SET NULL, -- ✅ SET NULL вместо CASCADE
	FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE SET NULL
);

-- Шаг 2: Копируем существующие данные
-- Фильтруем записи с несуществующими api_key_id ("jwt_auth", "unknown")
INSERT INTO api_usage_new 
SELECT 
	id,
	user_id,
	tenant_id,
	CASE 
		WHEN api_key_id IN ('jwt_auth', 'unknown') THEN NULL
		ELSE api_key_id
	END as api_key_id,
	endpoint,
	method,
	model,
	status_code,
	success,
	error_message,
	prompt_tokens,
	completion_tokens,
	total_tokens,
	duration_ms,
	created_at,
	user_agent,
	ip_address,
	conversation_id,
	metadata
FROM api_usage;

-- Шаг 3: Удаляем старую таблицу
DROP TABLE api_usage;

-- Шаг 4: Переименовываем новую таблицу
ALTER TABLE api_usage_new RENAME TO api_usage;

-- Шаг 5: Пересоздаем индексы
CREATE INDEX IF NOT EXISTS idx_api_usage_user_id ON api_usage(user_id);
CREATE INDEX IF NOT EXISTS idx_api_usage_tenant_id ON api_usage(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_usage_api_key_id ON api_usage(api_key_id) WHERE api_key_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_api_usage_created_at ON api_usage(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_usage_endpoint ON api_usage(endpoint);
CREATE INDEX IF NOT EXISTS idx_api_usage_model ON api_usage(model);
CREATE INDEX IF NOT EXISTS idx_api_usage_user_created ON api_usage(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_usage_tenant_created ON api_usage(tenant_id, created_at DESC) WHERE tenant_id IS NOT NULL;
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

// getChangelogV151Migration добавляет версию 1.5.1 (инкрементальная миграция v6)
func (s *SQLiteDB) getChangelogV151Migration() string {
	return `
-- ========================================
-- Add Changelog v1.5.1 (Migration v6)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.1', '2025-10-10', '## [1.5.1] - 2025-10-10

### Added
- **Enhanced Logs System**: Полноценная система просмотра логов в админке
  - Отдельная вкладка "Logs" в админ-панели
  - Просмотр текущего и архивных лог-файлов
  - Кликабельные фильтры по уровням (DEBUG, INFO, WARN, ERROR)
  - Real-time обновление логов через SSE (Server-Sent Events)
  - Скачивание лог-файлов
  - Dark theme для logs viewer (VS Code style)

### Changed
- **Backend**: LogsHandler с SSE stream, фильтрацией, пагинацией
- **Frontend**: Logs tab, real-time updates, level filters
- **UI**: Dark terminal theme, monospace font

### Technical
- SSE Stream через EventSource API
- Security: path validation
- Performance: tail 500 строк');
	`
}

// getChangelogV152Migration добавляет версию 1.5.2 (инкрементальная миграция v7)
func (s *SQLiteDB) getChangelogV152Migration() string {
	return `
-- ========================================
-- Add Changelog v1.5.2 (Migration v7)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.2', '2025-10-10', '## [1.5.2] - 2025-10-10

### Fixed
- **SSE Authentication**: Real-time logs работает с JWT
  - Токен через query параметр (EventSource не поддерживает headers)
  - SSEAuthMiddleware для SSE endpoints

### Changed
- Backend: sse_auth.go middleware
- Frontend: токен в URL stream

### Technical
- SSE + JWT через query параметр');
	`
}

// getChangelogV153Migration добавляет версию 1.5.3 (инкрементальная миграция v8)
func (s *SQLiteDB) getChangelogV153Migration() string {
	return `
-- ========================================
-- Add Changelog v1.5.3 (Migration v8)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.3', '2025-10-10', '## [1.5.3] - 2025-10-10

### Improved
- **Enhanced Log Parsing**: Полноценный парсер logrus
  - Извлечение всех полей (key=value)
  - Timestamp: HH:MM:SS.mmm формат
  - Context fields в UI

### Changed
- Backend: parseLogrusFields() метод
- Frontend: syntax highlighting для key=value
- CSS: VS Code-style colors

### Technical
- Парсинг: quoted и unquoted values
- Highlighting: #569cd6 / #ce9178');
	`
}

// getChangelogV154Migration добавляет версию 1.5.4 (инкрементальная миграция v9)
func (s *SQLiteDB) getChangelogV154Migration() string {
	return `
-- ========================================
-- Add Changelog v1.5.4 (Migration v9)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.4', '2025-10-10', '## [1.5.4] - 2025-10-10

### Fixed
- **SSE Logs Stream Auth**: Endpoint вне admin group
  - SSEAuthMiddleware читает token из query
  - RequireAdmin после аутентификации

### Changed
- Router: SSE endpoint отдельная регистрация
- Middleware chain: SSEAuth → RequireAdmin

### Technical
- Problem: JWTAuth блокировал SSE
- Solution: endpoint вне group');
	`
}

// getChangelogV155Migration добавляет версию 1.5.5 (инкрементальная миграция v10)
func (s *SQLiteDB) getChangelogV155Migration() string {
	return `
-- ========================================
-- Add Changelog v1.5.5 (Migration v10)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.5', '2025-10-10', '## [1.5.5] - 2025-10-10

### Fixed
- **Logs Stream File Selection**: Real-time логи
  - SSE stream параметр ?file=filename
  - Auto-select текущего лог-файла

### Changed
- Backend: StreamLogs query param file
- Frontend: передача файла в stream

### Technical
- Problem: hardcoded "proxy.log"
- Solution: query param + auto-select');
	`
}

// getChangelogV156Migration добавляет версию 1.5.6 (инкрементальная миграция v11)
func (s *SQLiteDB) getChangelogV156Migration() string {
	return `
-- ========================================
-- Add Changelog v1.5.6 (Migration v11)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.6', '2025-10-10', '## [1.5.6] - 2025-10-10

### Improved
- **Modelfile Display**: LICENSE скрывается
  - Только конфигурация модели
  - 20 строк с прокруткой

### Changed
- Frontend: stripLicenseFromModelfile()
- CSS: code-block-large класс

### Technical
- Frontend-only изменение
- Regex парсинг LICENSE');
	`
}

// getChangelogV157Migration добавляет версию 1.5.7 (инкрементальная миграция v12)
func (s *SQLiteDB) getChangelogV157Migration() string {
	return `
-- ========================================
-- Add Changelog v1.5.7 (Migration v12)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.7', '2025-10-10', '## [1.5.7] - 2025-10-10

### Added
- **Password Change**: Смена пароля
  - Endpoint /api/users/me/password
  - Валидация текущего пароля
  - Проверка силы нового пароля

### Changed
- Backend: UpdateUserPassword() метод
- SQLite/PostgreSQL реализация

### Security
- Bcrypt хеширование
- Audit logging
- Валидация силы пароля

### Technical
- Транзакционная поддержка
- Structured logging');
	`
}

// getChangelogV158Migration добавляет версию 1.5.8 (инкрементальная миграция v13)
func (s *SQLiteDB) getChangelogV158Migration() string {
	return `
-- ========================================
-- Add Changelog v1.5.8 (Migration v13)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.8', '2025-10-10', '## [1.5.8] - 2025-10-10

### Added
- **Tenant Membership Check**: Проверка прав доступа
  - Middleware для проверки членства
  - RBAC для tenant ресурсов
  - TODO удален из GetTenantUsage

- **Enhanced Member Management**: Управление участниками
  - Search endpoint для поиска пользователей
  - UI modal с live search
  - Change Role и Remove кнопки
  - Debounced search (500ms)

### Security
- Access control для tenant ресурсов
- Audit logging (403)
- Directory traversal protection

### Technical
- Middleware chain
- Context-based role storage
- Lazy loading user info');
	`
}

// getChangelogV159Migration добавляет версию 1.5.9 (инкрементальная миграция v14)
func (s *SQLiteDB) getChangelogV159Migration() string {
	return `
-- ========================================
-- Add Changelog v1.5.9 (Migration v14)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.9', '2025-10-10', '## [1.5.9] - 2025-10-10

### Fixed
- **Members Display Issue**: Username и email не отображались
  - SQL JOIN с таблицей users
  - TenantMember model: +Username, +Email
  - scanTenantMemberWithUserInfo()

- **Search Results UX**: Улучшена контрастность
  - Черный текст на белом фоне
  - Светло-зеленый hover
  - Темно-серый email

- **GetTenant Response**: Исправлен wrapper
  - Frontend: response.tenant extraction
  - currentTenant.id fix

### Technical
- SQL JOIN optimization
- Nullable fields handling
- Frontend state management');
	`
}

// getChangelogV1510Migration добавляет версию 1.5.10 (инкрементальная миграция v15)
func (s *SQLiteDB) getChangelogV1510Migration() string {
	return `
-- ========================================
-- Add Changelog v1.5.10 (Migration v15)
-- ========================================

INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.5.10', '2025-10-10', '## [1.5.10] - 2025-10-10

### Fixed
- **Tenant Member Count**: Количество участников не отображалось
  - SQL подзапрос COUNT(*)
  - Tenant model: +MemberCount
  - scanTenantWithRole updated

### Technical
- SQL Subquery optimization
- Minimal code changes
- Frontend compatibility');
	`
}

// getAddChangelogV161Migration returns SQL for adding changelog v1.6.1 (v18 migration)
func (s *SQLiteDB) getAddChangelogV161Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.6.1', '2025-10-12', '## [1.6.1] - 2025-10-12

### Added
- **OpenTelemetry Distributed Tracing**: Полная интеграция OpenTelemetry для distributed tracing
  - Поддержка Jaeger и Zipkin экспортеров
  - Автоматическая трассировка всех HTTP запросов
  - W3C Trace Context propagation
  - Настраиваемый sampling rate (0.0 - 1.0)
  - Span annotations с HTTP metadata (method, URL, status code)
  - Error tracking для запросов со status code >= 400

### Changed
- **Configuration**: Добавлена секция observability.tracing в конфигурацию
  - enabled: включение/отключение tracing
  - provider: выбор между "jaeger" или "zipkin"
  - service_name: название сервиса в traces
  - sampling_rate: процент трассируемых запросов
  - jaeger.endpoint и zipkin.endpoint: настройка экспортеров

### Technical
- Новый пакет internal/observability с TracerProvider
- TracingMiddleware для автоматической трассировки Gin запросов
- Интеграция в router и main.go с graceful shutdown
- Зависимости: go.opentelemetry.io/otel v1.38.0
- Тесты: 100% покрытие для observability и tracing middleware
- Tracing middleware применяется первым в цепочке для полной трассировки');
	`
}

// getAddChangelogV162Migration returns SQL for adding changelog v1.6.2 (v19 migration)
func (s *SQLiteDB) getAddChangelogV162Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.6.2', '2025-10-12', '## [1.6.2] - 2025-10-12

### Added
- **Performance Monitoring**: Continuous performance monitoring в production
  - Runtime metrics collection (CPU, memory, goroutines, GC stats)
  - Automatic performance anomaly detection
  - Baseline comparison для regression tracking
  - Configurable thresholds

- **Leak Detection**: Автоматическое выявление утечек
  - Memory и goroutine leak detection
  - Trend analysis (10 samples, 1/minute)
  - Warning alerts при sustained growth

- **Slow Request Logging**: Медленные запросы
  - Middleware для отслеживания request time
  - Configurable threshold (default: 5s)
  - Детальная информация о каждом slow request

- **pprof Endpoints**: Runtime profiling
  - /api/admin/pprof/* endpoints (admin only)
  - CPU profile, heap, goroutine dump
  - Memory allocations, block, mutex profiles

- **Performance API**: REST API для metrics
  - GET /api/admin/performance/metrics
  - GET /api/admin/performance/leaks
  - POST /api/admin/performance/reset-baseline
  - POST /api/admin/performance/reset-leaks

### Changed
- **Configuration**: Добавлена секция observability.performance
  - enabled, collection_interval, thresholds
  - gc_percentage для GC tuning
  - leak_detection и pprof_enabled flags

### Technical
- internal/observability/perfmon.go - Performance Monitor
- internal/observability/leak_detector.go - Leak Detector
- internal/api/middleware/slow_request.go - Slow Request Logger
- internal/api/handlers/performance.go - Performance API
- Thread-safe metrics, minimal overhead (<1%)
- Graceful shutdown support');
	`
}

// getAddChangelogV163Migration returns SQL for adding changelog v1.6.3 (v20 migration)
func (s *SQLiteDB) getAddChangelogV163Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.6.3', '2025-10-12', '## [1.6.3] - 2025-10-12

### Added
- **Scheduled Reports**: Автоматическая генерация и отправка отчетов по email
  - Cron-based scheduler с поддержкой гибких расписаний
  - Report Generator для Usage, Performance и System Health отчетов
  - Email Mailer с SMTP отправкой (поддержка TLS/SSL)
  - HTML email templates с современным дизайном
  - Поддержка множественных recipients

- **Report Types**: Три типа отчетов
  - **Usage Report**: Total requests, tokens, top models, top users
  - **Performance Report**: Latency metrics (P50/P95/P99), slow requests, error rates
  - **System Health Report**: Uptime, memory, goroutines, database stats

### Changed
- **Configuration**: Добавлена секция reports
  - enabled: включение/отключение scheduler
  - smtp: SMTP конфигурация (host, port, username, password, TLS)
  - schedules: массив scheduled reports с cron expressions
  - Поддержка переменных окружения для паролей

- **Database Interface**: Добавлены методы для reports statistics
  - GetUsageStats(): агрегированная статистика использования
  - GetPerformanceStats(): performance metrics за период
  - CountActiveUsers(), CountTotalUsers(), CountActiveAPIKeys()

### Technical
- Новый пакет internal/reports с полной реализацией
  - scheduler.go: Cron scheduler с github.com/robfig/cron/v3
  - generator.go: Report generation с embedded HTML templates
  - mailer.go: SMTP email delivery с gopkg.in/gomail.v2
  - types.go: Report models и data structures
  - templates/*.html: Beautiful HTML email templates

- SQLite реализация reports statistics методов
  - Percentile calculations для latency metrics
  - JOIN queries для user/model aggregations
  - Optimized queries для больших datasets

- PostgreSQL stub реализация (для будущей поддержки)
- Зависимости: github.com/robfig/cron/v3, gopkg.in/gomail.v2
- Graceful shutdown для scheduler');
	`
}

// getModelConfigsTableMigration returns SQL for creating model_configs table (v21 migration)
func (s *SQLiteDB) getModelConfigsTableMigration() string {
	return `
-- Model Configurations table for dynamic model parameters (v1.9.1)
CREATE TABLE IF NOT EXISTS model_configs (
    id TEXT PRIMARY KEY,
    model_name TEXT NOT NULL,
    scope TEXT NOT NULL CHECK(scope IN ('global', 'tenant', 'user')),
    tenant_id TEXT,
    user_id TEXT,
    created_by TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    parameters TEXT NOT NULL, -- JSON: ModelParameters
    
    -- Foreign keys
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    
    -- Constraints
    CHECK (
        (scope = 'global' AND tenant_id IS NULL AND user_id IS NULL) OR
        (scope = 'tenant' AND tenant_id IS NOT NULL AND user_id IS NULL) OR
        (scope = 'user' AND user_id IS NOT NULL)
    ),
    
    -- Unique constraint для предотвращения дубликатов
    UNIQUE(model_name, scope, tenant_id, user_id)
);

-- Indexes для быстрого поиска config по scope
CREATE INDEX IF NOT EXISTS idx_model_configs_model ON model_configs(model_name);
CREATE INDEX IF NOT EXISTS idx_model_configs_scope ON model_configs(scope);
CREATE INDEX IF NOT EXISTS idx_model_configs_tenant ON model_configs(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_model_configs_user ON model_configs(user_id) WHERE user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_model_configs_lookup ON model_configs(model_name, scope, tenant_id, user_id);

-- Trigger для автоматического обновления updated_at
CREATE TRIGGER IF NOT EXISTS update_model_configs_timestamp 
AFTER UPDATE ON model_configs
FOR EACH ROW
BEGIN
    UPDATE model_configs SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
	`
}

// getAddChangelogV191Migration returns SQL for adding changelog v1.9.1 (v22 migration)
func (s *SQLiteDB) getAddChangelogV191Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.9.1', '2025-10-13', '## [1.9.1] - 2025-10-13

### Added
- **Dynamic Model Parameters Configuration**: Красивая панель управления параметрами модели
  - Chat UI - интуитивная панель над строкой ввода с collapsible design
  - Model Selector с автоматической загрузкой доступных моделей из API
  - Real-time параметры: Temperature, Top P, Max Tokens, Context Window
  - Sliders с синхронизацией с number inputs для точной настройки
  - Tooltips с объяснениями для каждого параметра

- **Quick Presets**: 4 готовых пресета для разных сценариев
  - 🎨 Creative (temp 1.2) - для творческой генерации
  - ⚖️ Balanced (temp 0.7) - универсальный режим
  - 🎯 Precise (temp 0.3) - для точных ответов
  - 💻 Coding (temp 0.2) - оптимизирован для программирования

- **localStorage Persistence**: Автоматическое сохранение настроек
  - Запоминание выбранной модели между сессиями
  - Сохранение параметров в браузере
  - Автовосстановление при перезагрузке страницы

- **Backend Model Configuration System**:
  - ModelConfig и ModelParameters models на основе официального Ollama API
  - Database schema с поддержкой global/tenant/user scopes
  - Priority-based config resolution (user → tenant → global → defaults)
  - Automatic effective config application в chat handler

### Changed
- **Chat Interface**: Переработан UI чата
  - Model selector перемещен из header над строку ввода
  - Добавлена expandable parameters panel
  - Улучшена визуальная иерархия элементов
  - Responsive design для мобильных устройств

- **API Integration**: Обновлен формат запросов
  - api.streamChatMessage теперь принимает объект с параметрами
  - Backward compatibility с old string format
  - Support для Ollama-specific options (num_ctx)

- **Database Interface**: Новые методы для model configs
  - CreateModelConfig, GetModelConfig, UpdateModelConfig, DeleteModelConfig
  - GetModelConfigByScope для получения config по scope
  - GetEffectiveModelConfig с автоматическим priority resolution

### Technical
- **Backend (Go)**:
  - Новый файл internal/models/model_config.go с полными типами параметров
  - Миграция v21: таблица model_configs с indexes и triggers
  - SQLite реализация CRUD для model configs в internal/storage/sqlite/model_configs.go
  - PostgreSQL stubs в internal/storage/postgresql/stubs.go
  - Интеграция в internal/api/handlers/chat.go с type-safe конвертацией

- **Frontend (JavaScript)**:
  - Новый контроллер web/js/model-panel.js для управления панелью
  - CSS стили в web/css/model-panel.css с dark/light mode support
  - Обновлен web/js/chat.js для использования modelPanel
  - Обновлен web/js/api.js с поддержкой параметров в requests

- **Параметры основаны на официальном Ollama API**:
  - Predict options: Temperature, TopP, TopK, NumPredict, RepeatPenalty и др.
  - Runner options: NumCtx, NumBatch, NumGPU, MainGPU, UseMMap, NumThread
  - Полная совместимость с ollama-lib/api/types.go');
	`
}

// getAddChangelogV192Migration returns SQL for adding changelog v1.9.2 (v23 migration)
func (s *SQLiteDB) getAddChangelogV192Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.9.2', '2025-10-13', '## [1.9.2] - 2025-10-13

### Added
- **Compact UI Design (Cursor-style)**: Переработан дизайн панели моделей
  - Компактная горизонтальная панель с минималистичным дизайном
  - Model selector в виде элегантного dropdown без лишних элементов
  - Кнопка параметров в виде иконки (32x32px) для экономии места
  - Адаптивный дизайн для мобильных устройств

- **Context Window Tracking**: Умное управление контекстным окном
  - Real-time индикатор использования контекста (tokens used / max tokens)
  - Визуальные уровни предупреждений:
    - ✅ Нормальный (0-60%%): серый фон
    - ⚠️ Предупреждение (60-75%%): желтый фон
    - 🔴 Критический (75%%+): красный фон
  - Динамическое обновление при изменении num_ctx в параметрах

- **Auto-Summarization**: Автоматическая суммаризация при заполнении контекста
  - Автоматический триггер при достижении 75%% контекста
  - Умная суммаризация с сохранением последних 3 обменов сообщениями
  - Запрос к модели для создания лаконичного summary (2-3 параграфа)
  - Замена старых сообщений на system message с summary
  - Уведомления об успешной суммаризации с метриками токенов

- **Context Manager Module**: Новый модуль для управления контекстом
  - Оценка токенов в реальном времени (~1 токен = 4 символа)
  - Отслеживание всех сообщений с подсчетом токенов
  - API для получения статистики контекста
  - Автоматическая очистка при начале новой беседы

### Changed
- **Model Panel UI**: Компактный дизайн в стиле Cursor
  - Уменьшен padding с 16px до 8px для компактности
  - Model selector без label, только dropdown
  - Parameters toggle в виде иконки вместо текстовой кнопки
  - Уменьшены размеры шрифтов для экономии места
  - Sliders уменьшены с 18px до 14px thumb size

- **Request Parameters**: Ollama-specific options теперь применяются
  - Добавлено поле Options в ChatCompletionRequest model
  - Converter обрабатывает req.Options (num_ctx, top_k, repeat_penalty)
  - Полная интеграция с Ollama API types

- **Chat Flow**: Интеграция Context Manager
  - Автоматическое добавление сообщений в context tracker
  - Обновление context window size при смене параметров
  - Очистка контекста при начале нового чата

### Technical
- **Frontend (JavaScript)**:
  - Новый модуль web/js/context-manager.js с ContextManager class
  - Интеграция в chat.js для tracking user/assistant messages
  - Автоматическая суммаризация через /v1/chat/completions API
  - Notification system для уведомлений о суммаризации

- **Backend (Go)**:
  - Добавлено поле Options map[string]interface{} в models.ChatCompletionRequest
  - Converter применяет num_ctx, top_k, repeat_penalty из req.Options
  - Helper functions intPtr(), float64Ptr() для конвертации типов

- **CSS Updates**:
  - Полная переработка web/css/model-panel.css для compact design
  - Responsive breakpoints для мобильных устройств
  - Dark/Light mode совместимость с новыми стилями');
	`
}

// getAddChangelogV193Migration returns SQL for adding changelog v1.9.3 (v24 migration - initial version)
func (s *SQLiteDB) getAddChangelogV193Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.9.3', '2025-10-14', '## [1.9.3] - 2025-10-14

### Added
- **MoniGo Performance Dashboard**: Real-time performance monitoring интеграция
- **Admin Panel Reorganization**: Улучшенная структура навигации

### Technical
- MoniGo integration
- Admin panel restructuring');
	`
}

// getUpdateChangelogV193FinalMigration returns SQL for updating changelog v1.9.3 with full details (v25 migration)
func (s *SQLiteDB) getUpdateChangelogV193FinalMigration() string {
	return `
UPDATE changelogs 
SET content = '## [1.9.3] - 2025-10-14

### Added
- **MoniGo Performance Dashboard**: Real-time performance monitoring интеграция
  - MoniGo запускается на отдельном порту 9091 для изоляции
  - Quick Stats Cards с автоматическим обновлением каждые 5 секунд
  - Real-time метрики: CPU Usage, Memory Usage, Goroutines, System Health
  - Visual indicators (success/warning/critical) для метрик
  - API proxy для /admin/performance/monigo/api/v1/metrics
  - Direct link на Advanced Dashboard для полных возможностей MoniGo

- **NVIDIA GPU Monitoring**: Мониторинг GPU метрик через nvidia-smi
  - БЕЗ CGO зависимостей - использует nvidia-smi CLI напрямую
  - БЕЗ NVML headers - работает на любой системе с nvidia-smi
  - Поддержка нескольких GPU (multi-GPU configurations)
  - Единый компактный блок для всех GPU с gradient top border
  - Real-time метрики: Temperature, Power, GPU Load, VRAM, Clock, Fan Speed
  - Автообновление каждые 5 секунд
  - Hover эффект с подсветкой для каждой GPU строки
  - Цветовые индикаторы: Green (<70°C), Orange (70-80°C), Red (>80°C)
  - Graceful degradation если GPU не обнаружены или nvidia-smi недоступен
  - Platform-specific builds: Linux/macOS (full GPU support), Windows (stub)

- **Admin Panel Reorganization**: Улучшенная структура навигации
  - Новая вкладка Models с Available Models списком
  - Refresh Models кнопка для обновления списка моделей
  - System Tab переработан для Performance & GPU Monitoring

### Changed
- **System Tab**: Переработан полностью под мониторинг
  - Убраны Available Models (перенесены в Models Tab)
  - MoniGo Dashboard link вместо embedded iframe
  - Quick Stats Cards для основных метрик
  - NVIDIA GPU Metrics секция (если GPU доступны)

- **Models Tab**: Новая навигационная структура
  - Fix: Модели корректно загружаются при первом заходе

- **GPU Monitoring Architecture**:
  - Отказ от go-nvml (CGO зависимость) в пользу nvidia-smi CLI
  - Build tags для platform-specific реализаций
  - Windows: stub версия (GPU monitoring disabled)
  - Linux/macOS: полная функциональность через nvidia-smi

### Fixed
- **Models Tab Loading**: Исправлен баг с загрузкой моделей при первом заходе
- **MoniGo Integration**: Убран iframe, решены проблемы с CORS и static files
- **GPU Monitoring**: Убраны CGO compilation errors на Windows/Linux

### Technical
- **Backend**: internal/metrics/gpu_monitor_smi.go (Linux/macOS), gpu_monitor_windows.go (stub), internal/api/handlers/gpu.go для REST API
- **Frontend**: web/js/gpu-monitor.js - NVIDIA GPU metrics, unified GPU card дизайн, gradient top border, grid layout
- **Migration**: v25 для полного обновления changelog v1.9.3

### Security
- MoniGo dashboard доступен только через JWT authentication
- API proxy защищен Bearer token authentication
- GPU metrics endpoint требует аутентификацию'
WHERE version = '1.9.3';
	`
}

// getFilesTableMigration returns SQL for creating files table (v1.10.0+)
func (s *SQLiteDB) getFilesTableMigration() string {
	return `
-- ========================================
-- Files Table (FILE-STORAGE-01: v1.10.0+)
-- ========================================
CREATE TABLE IF NOT EXISTS files (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	user_id TEXT NOT NULL,
	tenant_id TEXT, -- NULL для personal files
	
	-- File information
	filename TEXT NOT NULL,
	original_filename TEXT NOT NULL,
	mime_type TEXT NOT NULL,
	size_bytes INTEGER NOT NULL,
	checksum_sha256 TEXT, -- For deduplication
	
	-- Storage
	storage_backend TEXT NOT NULL, -- 'local', 's3'
	storage_path TEXT NOT NULL, -- Path within backend
	storage_bucket TEXT, -- S3 bucket name
	
	-- Extracted content (for simple chat integration)
	extracted_text TEXT, -- Full text for simple use cases
	extraction_status TEXT DEFAULT 'pending', -- 'pending', 'completed', 'failed'
	extraction_error TEXT,
	
	-- Metadata (JSON)
	metadata TEXT, -- page_count, author, keywords, etc.
	
	-- Document info
	page_count INTEGER,
	word_count INTEGER,
	language TEXT,
	
	-- Access control
	is_public INTEGER DEFAULT 0, -- Boolean
	shared_with TEXT, -- JSON array of user IDs
	
	-- Usage tracking
	download_count INTEGER DEFAULT 0,
	last_accessed_at TIMESTAMP,
	
	-- Timestamps
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at TIMESTAMP, -- Soft delete
	
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_files_user_id ON files(user_id);
CREATE INDEX IF NOT EXISTS idx_files_tenant_id ON files(tenant_id);
CREATE INDEX IF NOT EXISTS idx_files_mime_type ON files(mime_type);
CREATE INDEX IF NOT EXISTS idx_files_extraction_status ON files(extraction_status);
CREATE INDEX IF NOT EXISTS idx_files_checksum ON files(checksum_sha256);
CREATE INDEX IF NOT EXISTS idx_files_created_at ON files(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_storage_backend ON files(storage_backend);

-- ========================================
-- File Access Logs Table (optional, для audit)
-- ========================================
CREATE TABLE IF NOT EXISTS file_access_logs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	file_id TEXT NOT NULL,
	user_id TEXT,
	action TEXT NOT NULL, -- 'upload', 'download', 'delete', 'view'
	ip_address TEXT,
	user_agent TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	
	FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_file_logs_file_id ON file_access_logs(file_id);
CREATE INDEX IF NOT EXISTS idx_file_logs_user_id ON file_access_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_file_logs_created_at ON file_access_logs(created_at DESC);
	`
}

// getMessageFilesJunctionMigration returns SQL for message_files junction table (v1.10.0+)
func (s *SQLiteDB) getMessageFilesJunctionMigration() string {
	return `
-- ========================================
-- Message Files Junction Table (FILE-STORAGE-01: Phase 4)
-- ========================================
-- Связь между сообщениями и прикрепленными файлами (many-to-many)
CREATE TABLE IF NOT EXISTS message_files (
	message_id TEXT NOT NULL,
	file_id TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (message_id, file_id),
	FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
	FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_message_files_message_id ON message_files(message_id);
CREATE INDEX IF NOT EXISTS idx_message_files_file_id ON message_files(file_id);
	`
}

// getAddChangelogV1100Migration returns SQL for adding changelog v1.10.0 (v28 migration)
func (s *SQLiteDB) getAddChangelogV1100Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.10.0', '2025-10-16', '## [1.10.0] - 2025-10-16

### Added
- **FILE-STORAGE-01: Universal File Storage & Processing System** ✅
  - Storage Backends: Local filesystem и S3-compatible (MinIO)
  - Document Extractors: PDF, DOCX, TXT, CSV с автоматическим определением кодировки
  - Database Integration: Таблицы files, file_access_logs, message_files
  - API Endpoints: /api/files/* для upload, download, delete, list
  - WebUI: Страница Files для управления файлами
  - Admin Panel: Новая вкладка Files для управления всеми файлами
  - Chat Integration: Прикрепление файлов к сообщениям
  - LLM Context Enrichment: Автоматическое включение содержимого файлов
  - Path Traversal Prevention: Robust защита от path traversal
  - Unicode Filenames: Полная поддержка Unicode (Cyrillic, Chinese, Emoji)

- **Cross-Platform PDF Text Extraction** 🚀
  - Pure Go Library: github.com/ledongthuc/pdf
  - Automatic Fallback: pdftotext → go-pdf
  - Three Methods: auto, pdftotext, go-pdf
  - Docker-Ready: Работает без внешних зависимостей

- **Advanced Text Encoding Detection** 🔍
  - UTF-8 with BOM, UTF-16 LE/BE
  - Windows-1251 Fallback для русского текста
  - Cyrillic Detection
  - Reasonable Text Validation

- **Comprehensive Unit Tests** ✅
  - Validator Tests: 13 тестов (100% pass)
  - Local Storage Tests: 15 тестов (100% pass)
  - Coverage: filestorage 46.7%, storage 29.6%

### Changed
- Chat Messages: Добавлено поле file_ids
- Configuration: Секции file_storage и extractors

### Fixed
- File Upload Integrity: SkipContentValidation flag
- Windows Path Separators: filepath.FromSlash()
- File Deletion: Robust path traversal checks
- Text Encoding: Windows-1251 heuristic detection

### Technical
- Dependencies: github.com/ledongthuc/pdf
- Migrations: v26 files table, v27 message_files junction
- Packages: internal/filestorage, internal/extractors
- Tests: validator_test.go, local_test.go');
	`
}

// getAddChangelogV194Migration returns SQL for adding changelog v1.9.4 (v29 migration)
func (s *SQLiteDB) getAddChangelogV194Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.9.4', '2025-10-20', '## [1.9.4] - 2025-10-20

### Added
- **CPU Cache-Friendly Data Structures** 🚀 (Phase 2: Hot/Cold Split & Caching)
  - APIKeyCache Layer: In-memory cache для API keys с TTL expiration
    - Thread-safe concurrent access (RWMutex)
    - Load-through caching pattern с GetOrLoad()
    - Background cleanup goroutine
    - Performance: 22.76ns Get, 78.43ns Set, 45.52ns Parallel
    - 100x faster vs database queries (22ns vs 2ms)
  - APIKeyDBAuthOptimized Middleware: Cache-aware authentication
    - Cache hit path: ~22ns (memory only)
    - Cache miss path: ~2ms (DB + bcrypt)
    - Expected 95%+ cache hit rate → 20x faster auth
  - APIKeyUsageThreadSafe: Fully thread-safe usage tracking
    - Atomic counters с cache line padding (prevents false sharing)
    - RWMutex для map operations (ModelUsage, EndpointUsage, DailyUsage)
    - Performance: 196ns vs 236ns Original + zero race conditions
    - 20% faster + thread-safe

- **Cursor Rules для Cache Optimization**
  - Автоматические подсказки для cache-friendly structures
  - Lint правила для false sharing detection
  - Templates для padded structs, hot/cold splits, concurrent counters

### Changed
- **StatsOptimized Migration**: Migrated GlobalStats to cache-friendly version
  - Cache line padding между atomic counters
  - StatsInterface для backwards compatibility
  - Performance: 6.4x faster under high concurrency

### Fixed
- **Race Condition в APIKey.IncrementUsage**: Replaced ++ with atomic.AddInt64
  - Atomic operations для TotalRequests, SuccessfulRequests, FailedRequests, TotalTokens
  - Map operations (ModelUsage, DailyUsage) все еще require careful handling
  - Recommended: Use APIKeyUsageThreadSafe для new keys

### Technical
- Benchmarks Added:
  - internal/api/handlers/stats_bench_test.go: Stats vs StatsOptimized
  - internal/models/apikey_bench_test.go: APIKey validation benchmarks
  - internal/models/apikey_usage_threadsafe_test.go: Thread-safe usage benchmarks
  - internal/cache/apikey_cache_test.go: Cache performance benchmarks
- Documentation:
  - docs/CACHE_OPTIMIZATION.md: Technical deep-dive
  - docs/CACHE_OPTIMIZATION_SUMMARY.md: Executive summary
  - docs/CACHE_OPTIMIZATION_QUICKSTART.md: Quick start guide
  - PERFORMANCE_IMPROVEMENTS.md: High-level report
  - MIGRATION_APPLIED.md: Phase 1 & 2 migration status
  - PHASE2_COMPLETED.md: Phase 2 completion report
- Dependencies: No new dependencies (pure Go stdlib)
- New Packages:
  - internal/cache: APIKey caching layer
  - internal/api/middleware/apikey_db_auth_optimized.go: Optimized middleware
  - internal/models/apikey_usage_threadsafe.go: Thread-safe usage tracking
  - internal/api/handlers/stats_optimized.go: Cache-friendly stats

### Performance
- Authentication: 20x faster (с cache hit rate 95%+)
- Usage Tracking: 1.2x faster + thread-safe
- Server Throughput: +50-87% expected improvement
- Memory Overhead: ~6 MB для 10,000 API keys (acceptable)

### Security
- Zero Race Conditions: All optimized structures pass -race tests
- Thread-Safe Maps: RWMutex protection для concurrent access
- Atomic Counters: Cache line padding prevents false sharing');
	`
}

// getAddChangelogV1103Migration returns SQL for adding changelog v1.10.3 (v30 migration)
func (s *SQLiteDB) getAddChangelogV1103Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.10.3', '2025-10-20', '## [1.10.3] - 2025-10-20

### Added
- **IMAGE-01: Image Upload & OCR Processing** 🖼️
  - Vision Interface (internal/vision/interface.go): OCREngine interface: ExtractText, DescribeImage, SupportedModels
  - OllamaOCR Engine (internal/vision/ollama_ocr.go): Multimodal models support: LLaVA, BakLLaVA, Llama3.2-Vision
  - ImageProcessor (internal/imageproc/processor.go): Thumbnail generation, image resize, format conversion
  - ImageHandler API (internal/api/handlers/image_handler.go): POST /api/images/upload, GET /api/images/:id, thumbnail endpoints
  - WebSocket integration для upload/OCR progress events

### Changed
- **Ollama Client Extension**: Added Images [][]byte field to ChatMessage for vision model support

### Technical
- Testing: internal/imageproc/processor_test.go, internal/vision/ollama_ocr_test.go
- Dependencies: github.com/disintegration/imaging v1.6.2
- Features: Multi-format support (JPEG, PNG, GIF, WebP), Automatic OCR, Thumbnail generation, Language detection');
	`
}

// getWebFetchesMigration returns SQL for creating web_fetches table (v31 migration)
func (s *SQLiteDB) getWebFetchesMigration() string {
	return `
-- Web Fetches Table for storing fetched web pages
CREATE TABLE IF NOT EXISTS web_fetches (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    user_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    
    -- URL info
    url TEXT NOT NULL,
    url_hash TEXT NOT NULL,
    domain TEXT NOT NULL,
    
    -- Content
    title TEXT,
    content TEXT NOT NULL,
    html_content TEXT,
    
    -- Metadata (JSON)
    metadata TEXT,
    
    -- Links
    links TEXT,
    
    -- LLM processing
    summary TEXT,
    summary_model TEXT,
    
    -- Fetch info
    fetch_status TEXT DEFAULT 'success',
    fetch_time_ms INTEGER,
    status_code INTEGER,
    content_type TEXT,
    content_length INTEGER,
    fetch_error TEXT,
    
    -- Language detection
    language TEXT,
    word_count INTEGER,
    
    -- Cache
    expires_at TIMESTAMP,
    is_cached INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_fetched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

-- Indexes for web_fetches
CREATE INDEX IF NOT EXISTS idx_web_fetches_url_hash ON web_fetches(url_hash);
CREATE INDEX IF NOT EXISTS idx_web_fetches_user ON web_fetches(user_id);
CREATE INDEX IF NOT EXISTS idx_web_fetches_tenant ON web_fetches(tenant_id);
CREATE INDEX IF NOT EXISTS idx_web_fetches_domain ON web_fetches(domain);
CREATE INDEX IF NOT EXISTS idx_web_fetches_expires ON web_fetches(expires_at);
CREATE INDEX IF NOT EXISTS idx_web_fetches_created ON web_fetches(created_at DESC);

-- Web Fetch Rate Limits Table
CREATE TABLE IF NOT EXISTS web_fetch_rate_limits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT NOT NULL UNIQUE,
    requests_per_minute INTEGER DEFAULT 10,
    last_request_at TIMESTAMP,
    request_count INTEGER DEFAULT 0,
    blocked_until TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rate_limits_domain ON web_fetch_rate_limits(domain);
	`
}

// getAddChangelogV1104Migration returns SQL for adding changelog v1.10.4 (v32 migration)
func (s *SQLiteDB) getAddChangelogV1104Migration() string {
	return `
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
	`
}

// getAddChangelogV1105Migration returns SQL for adding changelog v1.10.5 (v33 migration)
func (s *SQLiteDB) getAddChangelogV1105Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.10.5', '2025-10-25', '## [1.10.5] - 2025-10-25

### Changed
- **WEB-FETCH-01: Full Content Processing** 🚀
  - **BREAKING CHANGE**: Web fetcher теперь передает полное содержимое страницы модели без обрезания
  - Удален hardcoded truncation до 3000 символов
  - Добавлен параметр TruncateLength в ProcessMessageOptions для гибкого контроля
  - Default: TruncateLength: 0 (без ограничений) - оптимально для моделей с большим контекстом (128K+)
  - Добавлены поля WordCount и Language в WebPage для статистики
  - Логирование truncation когда применяется

### Technical
- Структуры данных: WebPage добавлены WordCount int и Language string, ParsedContent добавлено Language string, ProcessMessageOptions добавлено TruncateLength int (0 = без ограничений)
- Поведение по умолчанию: Chat Integration TruncateLength: 0 - полный контент для LLM, Старое поведение можно вернуть: TruncateLength: 3000
- Улучшения: Показ статистики для больших страниц (>10K chars), Детальное логирование при truncation, Language detection из HTML metadata');
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
