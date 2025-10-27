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

	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
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
		{
			Version: 34,
			Name:    "add_oidc_fields_to_users",
			SQL:     s.getAddOIDCFieldsMigration(),
		},
		{
			Version: 35,
			Name:    "add_changelog_v1_11_1",
			SQL:     s.getAddChangelogV1111Migration(),
		},
		{
			Version: 36,
			Name:    "add_unique_index_tenants_name",
			SQL:     s.getAddUniqueIndexTenantsNameMigration(),
		},
		{
			Version: 37,
			Name:    "add_changelog_v1_11_2",
			SQL:     s.getAddChangelogV1112Migration(),
		},
		{
			Version: 38,
			Name:    "add_ldap_dn_to_users",
			SQL:     s.getAddLDAPDNToUsersMigration(),
		},
		{
			Version: 39,
			Name:    "add_changelog_v1_11_3",
			SQL:     s.getAddChangelogV1113Migration(),
		},
		{
			Version: 40,
			Name:    "create_audit_events_table",
			SQL:     s.getCreateAuditEventsTableMigration(),
		},
		{
			Version: 41,
			Name:    "add_changelog_v1_11_4",
			SQL:     s.getAddChangelogV1114Migration(),
		},
		{
			Version: 42,
			Name:    "create_rbac_tables",
			SQL:     s.getCreateRBACTablesMigration(),
		},
		{
			Version: 43,
			Name:    "create_quotas_tables",
			SQL:     s.getCreateQuotasTablesMigration(),
		},
		{
			Version: 44,
			Name:    "add_changelog_v1_11_7",
			SQL:     s.getAddChangelogV1117Migration(),
		},
		{
			Version: 45,
			Name:    "add_changelog_v1_11_8",
			SQL:     s.getAddChangelogV1118Migration(),
		},
		{
			Version: 46,
			Name:    "add_changelog_v1_11_9",
			SQL:     s.getAddChangelogV1119Migration(),
		},
		{
			Version: 47,
			Name:    "add_changelog_v1_12_1",
			SQL:     s.getAddChangelogV1121Migration(),
		},
		{
			Version: 48,
			Name:    "add_advanced_rate_limiting",
			SQL:     s.getAddAdvancedRateLimitingMigration(),
		},
		{
			Version: 49,
			Name:    "add_changelog_v1_12_2",
			SQL:     s.getAddChangelogV1122Migration(),
		},
		{
			Version: 50,
			Name:    "add_changelog_v1_12_3",
			SQL:     s.getAddChangelogV1123Migration(),
		},
		{
			Version: 51,
			Name:    "create_rag_tables",
			SQL:     s.getCreateRAGTablesMigration(),
		},
		{
			Version: 52,
			Name:    "add_changelog_v2_0_0",
			SQL:     s.getAddChangelogV200Migration(),
		},
		{
			Version: 53,
			Name:    "update_changelog_v2_0_0_with_export_import",
			SQL:     s.getUpdateChangelogV200WithExportImportMigration(),
		},
		{
			Version: 54,
			Name:    "add_changelog_v2_1_0",
			SQL:     s.getAddChangelogV210Migration(),
		},
		{
			Version: 55,
			Name:    "update_changelog_v2_1_0_formatting",
			SQL:     s.getUpdateChangelogV210FormattingMigration(),
		},
		{
			Version: 56,
			Name:    "update_changelog_v2_1_0_final_format",
			SQL:     s.getUpdateChangelogV210FinalFormatMigration(),
		},
		{
			Version: 57,
			Name:    "add_invitations_table",
			SQL:     s.getAddInvitationsTableMigration(),
		},
		{
			Version: 58,
			Name:    "add_changelog_v2_2_0",
			SQL:     s.getAddChangelogV220Migration(),
		},
		{
			Version: 59,
			Name:    "seed_rbac_permissions",
			SQL:     s.getSeedRBACPermissionsMigration(),
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

// getAddOIDCFieldsMigration returns SQL for adding OIDC fields to users table (v34 migration)
func (s *SQLiteDB) getAddOIDCFieldsMigration() string {
	return `
-- Add OIDC authentication fields to users table (Version 1.11.1+: Keycloak SSO Integration)

-- auth_provider: Authentication provider type ('local', 'oidc', 'ldap')
ALTER TABLE users ADD COLUMN auth_provider TEXT DEFAULT 'local' NOT NULL;

-- oidc_subject: OIDC 'sub' claim (unique identifier from OIDC provider)
ALTER TABLE users ADD COLUMN oidc_subject TEXT;

-- oidc_issuer: OIDC issuer URL (e.g., https://keycloak.example.com/realms/myrealm)
ALTER TABLE users ADD COLUMN oidc_issuer TEXT;

-- Create index for fast lookup by OIDC subject
CREATE INDEX IF NOT EXISTS idx_users_oidc_subject ON users(oidc_subject) WHERE oidc_subject IS NOT NULL;

-- Create index for filtering by auth provider
CREATE INDEX IF NOT EXISTS idx_users_auth_provider ON users(auth_provider);

-- Create composite index for OIDC issuer + subject (for multi-provider scenarios)
CREATE INDEX IF NOT EXISTS idx_users_oidc_issuer_subject ON users(oidc_issuer, oidc_subject) 
    WHERE oidc_issuer IS NOT NULL AND oidc_subject IS NOT NULL;

-- Add unique constraint for OIDC subject (within same issuer)
-- Note: SQLite doesn't support adding unique constraints to existing columns directly,
-- so we create a unique index instead
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oidc_unique ON users(oidc_issuer, oidc_subject) 
    WHERE oidc_issuer IS NOT NULL AND oidc_subject IS NOT NULL;
	`
}

// getAddChangelogV1111Migration returns SQL for adding changelog v1.11.1 (v35 migration)
func (s *SQLiteDB) getAddChangelogV1111Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.11.1', '2025-10-25', '## [1.11.1] - 2025-10-25

### Added
- **OIDC-01: Keycloak SSO Integration** 🔐
  - OpenID Connect (OIDC) аутентификация для корпоративного Single Sign-On (SSO)
  - Интеграция с Keycloak и другими OIDC providers (Google, Azure AD, Okta)
  - Authorization Code Flow с PKCE для безопасной аутентификации
  - Автоматическое user provisioning при первом входе через SSO
  - Гибкий claims mapping для разных OIDC providers
  - Role-based access control из OIDC groups/roles
  - Session management для OIDC state с защитой от CSRF
  - HTTP endpoints: /api/auth/oidc/login, /api/auth/oidc/callback, /api/auth/oidc/logout

### Technical
- Новые модули: internal/auth/oidc/provider.go - OIDC provider wrapper, internal/auth/oidc/claims.go - OIDC claims structures, internal/api/handlers/oidc.go - HTTP handlers
- Конфигурация: auth.oidc.enabled, auth.oidc.issuer, auth.oidc.client_id, auth.oidc.client_secret, auth.oidc.scopes, auth.oidc.claims mapping, auth.oidc.auto_create_user, auth.oidc.auto_update_user, auth.oidc.default_role
- База данных (Migration v34): users.auth_provider, users.oidc_subject, users.oidc_issuer, индексы для OIDC, unique constraint (issuer, subject)
- Зависимости: coreos/go-oidc v3, golang.org/x/oauth2, gin-contrib/sessions
- Тестирование: 10 unit tests PASS, 5 SKIP (mock OIDC required)

### Security
- CSRF Protection - random state parameter в OAuth2 flow
- ID Token Verification - проверка подписи и claims через coreos/go-oidc
- Session Security - HttpOnly cookies, SameSite=Lax, secure encryption
- Claims Validation - проверка issuer, audience, expiration');
	`
}

// getAddUniqueIndexTenantsNameMigration returns SQL for adding unique index on tenants.name (v36 migration)
// Version 1.11.2+: Auto-tenant Provisioning from OIDC Groups
func (s *SQLiteDB) getAddUniqueIndexTenantsNameMigration() string {
	return `
-- Add unique index on tenants.name for OIDC auto-provisioning (Version 1.11.2+)
-- This ensures tenant names are unique and speeds up GetTenantByName lookups

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_name_unique ON tenants(name);

-- Also create a regular index on tenants.slug if not already exists (for completeness)
CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
	`
}

// getAddChangelogV1112Migration returns SQL for adding changelog v1.11.2 (v37 migration)
func (s *SQLiteDB) getAddChangelogV1112Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.11.2', '2025-10-25', '## [1.11.2] - 2025-10-25

### Added
- **OIDC-02: Auto-tenant Provisioning from OIDC Groups** 🏢
  - Автоматическое создание tenants из OIDC groups claims (Keycloak, Google, Azure AD)
  - Group → Tenant mapping с двумя режимами: Direct (1:1) и Prefix (path extraction)
  - Auto-provisioning: создание tenants и добавление пользователей при первом логине
  - Role assignment: admin/member роли из OIDC groups
  - Orphaned memberships cleanup (опционально)
  - Tenant name normalization

### Technical
- Модули: internal/auth/oidc/tenants.go (parsing), internal/auth/oidc/provisioner.go (provisioner)
- Configuration: auth.oidc.tenant_provisioning (enabled, auto_create_tenants, sync_on_login, group_mapping)
- Database (Migration v36): UNIQUE INDEX на tenants.name, GetTenantByName method
- Integration: OIDC callback с tenant provisioning, JWT с tenant IDs
- Tests: 16 unit tests (ParseGroups, mapping, normalization)');
	`
}

// getAddLDAPDNToUsersMigration returns SQL for adding ldap_dn column to users (v38 migration)
// Version 1.11.3+: LDAP/Active Directory Integration
func (s *SQLiteDB) getAddLDAPDNToUsersMigration() string {
	return `
-- Add LDAP DN column for LDAP/Active Directory authentication (Version 1.11.3+)
-- ldap_dn stores the LDAP Distinguished Name of the user

ALTER TABLE users ADD COLUMN ldap_dn TEXT;

-- Create unique index for LDAP DN (cannot use UNIQUE constraint in ALTER TABLE ADD COLUMN)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_ldap_dn_unique ON users(ldap_dn) WHERE ldap_dn IS NOT NULL;
	`
}

// getAddChangelogV1113Migration returns SQL for adding changelog v1.11.3 (v39 migration)
func (s *SQLiteDB) getAddChangelogV1113Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.11.3', '2025-10-25', '## [1.11.3] - 2025-10-25

### Added
- **LDAP-01: LDAP/Active Directory Integration** 🔐
  - LDAP Bind Authentication для корпоративных LDAP/AD серверов
  - User/Group Search с настраиваемыми фильтрами (OpenLDAP, Active Directory)
  - Auto-provisioning users при первом логине
  - Auto-update users синхронизация email/full name
  - Tenant provisioning из LDAP groups (reuse OIDC-02 logic)
  - TLS/LDAPS support с StartTLS и certificate validation
  - Admin detection на основе LDAP groups
  - Test connection endpoint для admin (/api/auth/ldap/test)

### Technical
- Модули: internal/auth/ldap/client.go (LDAP client), internal/api/handlers/ldap.go (handler)
- Configuration: auth.ldap (URL, bind credentials, user/group search, TLS, provisioning)
- Database (Migration v38): ALTER TABLE users ADD COLUMN ldap_dn TEXT UNIQUE
- API Routes: POST /api/auth/ldap/login (public), GET /api/auth/ldap/test (admin)
- Integration: JWT с tenant IDs, tenant provisioner из OIDC-02
- Tests: 17 unit tests (config, authentication, isAdminGroup)
- Support: OpenLDAP, Active Directory, FreeIPA');
	`
}

// getCreateAuditEventsTableMigration returns SQL for creating audit_events table (v40 migration)
// Version 1.11.4+: Enhanced Audit Logging
func (s *SQLiteDB) getCreateAuditEventsTableMigration() string {
	return `
-- Create audit_events table for comprehensive security and compliance logging (Version 1.11.4+)
CREATE TABLE IF NOT EXISTS audit_events (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL DEFAULT 'info',
    
    -- Actor (who performed the action)
    actor_id TEXT NOT NULL,
    actor_type TEXT NOT NULL DEFAULT 'user',
    
    -- Target (what was affected)
    target_id TEXT,
    target_type TEXT,
    
    -- Context
    action TEXT NOT NULL,
    resource TEXT NOT NULL,
    status TEXT NOT NULL,
    error_msg TEXT,
    metadata TEXT, -- JSON
    
    -- Request info
    ip_address TEXT NOT NULL,
    user_agent TEXT,
    
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_actor_id ON audit_events(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit_events(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_events_severity ON audit_events(severity);
CREATE INDEX IF NOT EXISTS idx_audit_events_resource ON audit_events(resource);
CREATE INDEX IF NOT EXISTS idx_audit_events_target_id ON audit_events(target_id) WHERE target_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_audit_events_status ON audit_events(status);
	`
}

// getAddChangelogV1114Migration returns SQL for adding changelog v1.11.4 (v41 migration)
// Version 1.11.4+: Enhanced Audit Logging
func (s *SQLiteDB) getAddChangelogV1114Migration() string {
	return `
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
	`
}

// getCreateRBACTablesMigration returns SQL for creating RBAC tables (v42 migration)
// Version 1.11.5+: Custom Roles & Permissions
func (s *SQLiteDB) getCreateRBACTablesMigration() string {
	return `
-- Create RBAC tables for Role-Based Access Control (Version 1.11.5+)

-- Permissions table
CREATE TABLE IF NOT EXISTS permissions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT 'global', -- 'global', 'tenant', 'personal'
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Roles table
CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL DEFAULT 'custom', -- 'system', 'custom'
    scope TEXT NOT NULL DEFAULT 'global', -- 'global', 'tenant'
    tenant_id TEXT, -- null for global roles
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE(name, tenant_id) -- Unique name per tenant (or global if tenant_id is null)
);

-- Role-Permission mapping (many-to-many)
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id TEXT NOT NULL,
    permission_id TEXT NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- User-Role assignments (many-to-many)
CREATE TABLE IF NOT EXISTS user_roles (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    tenant_id TEXT, -- null for global role assignment
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    UNIQUE(user_id, role_id, tenant_id) -- Can't assign same role twice
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_roles_tenant_id ON roles(tenant_id) WHERE tenant_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_permissions_resource ON permissions(resource);
CREATE INDEX IF NOT EXISTS idx_permissions_name ON permissions(name);
CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id);
	`
}

// getCreateQuotasTablesMigration returns SQL for creating quotas tables (v43 migration)
// Version 1.11.7+: Usage Quotas System
func (s *SQLiteDB) getCreateQuotasTablesMigration() string {
	return `
-- Create Quotas tables for Usage Quotas System (Version 1.11.7+)

-- Quotas table: defines limits for users or tenants
CREATE TABLE IF NOT EXISTS quotas (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    scope TEXT NOT NULL, -- 'user', 'tenant'
    target_id TEXT NOT NULL, -- user_id or tenant_id
    
    -- Token limits
    tokens_per_day INTEGER,
    tokens_per_month INTEGER,
    
    -- Request limits
    requests_per_day INTEGER,
    requests_per_month INTEGER,
    max_concurrent INTEGER,
    
    -- Storage limits
    max_storage_bytes INTEGER,
    max_conversations INTEGER,
    max_file_size INTEGER,
    
    -- Model restrictions (JSON array of model names)
    allowed_models TEXT,
    
    -- Metadata
    enabled INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(scope, target_id)
);

-- Quota usage table: tracks current usage against quotas
CREATE TABLE IF NOT EXISTS quota_usage (
    id TEXT PRIMARY KEY,
    quota_id TEXT NOT NULL,
    target_id TEXT NOT NULL, -- user_id or tenant_id (matches quota)
    
    -- Current usage counters
    tokens_used_today INTEGER NOT NULL DEFAULT 0,
    tokens_used_month INTEGER NOT NULL DEFAULT 0,
    requests_today INTEGER NOT NULL DEFAULT 0,
    requests_month INTEGER NOT NULL DEFAULT 0,
    current_concurrent INTEGER NOT NULL DEFAULT 0,
    storage_used_bytes INTEGER NOT NULL DEFAULT 0,
    conversations_count INTEGER NOT NULL DEFAULT 0,
    
    -- Reset timestamps
    last_daily_reset TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_monthly_reset TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (quota_id) REFERENCES quotas(id) ON DELETE CASCADE,
    UNIQUE(quota_id, target_id)
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_quotas_target ON quotas(scope, target_id) WHERE enabled = 1;
CREATE INDEX IF NOT EXISTS idx_quotas_scope ON quotas(scope);
CREATE INDEX IF NOT EXISTS idx_quota_usage_target ON quota_usage(target_id);
CREATE INDEX IF NOT EXISTS idx_quota_usage_quota ON quota_usage(quota_id);
	`
}

// getAddChangelogV1118Migration returns SQL for adding changelog v1.11.8 (v45 migration)
// Version 1.11.8: WebUI Functionality Restoration
func (s *SQLiteDB) getAddChangelogV1118Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.11.8', '2025-10-26', '## [1.11.8] - 2025-10-26

### Added
- **WebUI Complete Restoration**: Full synchronization of functionality from /web_old/ to /web/
  - Dashboard: Restored gradient background, stats grid, recent conversations, tenants list, quick actions
  - Admin Panel: Complete 11-tab interface (Dashboard, Users, API Keys, Files, Models, MCP Servers, Backups, Audit, RBAC, Logs, System)
  - Navigation: Unified navbar.js component across all pages with prominent Admin link
  - Chat: Full feature parity with model panel, context manager, file attachments
  - API Keys: Personal and organization keys management with full CRUD operations
  - Files: Drag & drop upload, filters, grid view, preview functionality
  - Tenants: Organization management with member roles and permissions
  - Profile: User information editing, password change, account management
  - Usage: Statistics and analytics dashboard
  - About: System information and changelogs display
  - MCP: MCP servers management interface

### Changed
- **UI/UX Improvements**: Applied theme.css consistently across all pages for modern, cohesive design
  - Gradient backgrounds for visual appeal
  - Improved button and tab styling with hover effects
  - Better readability with white headings and text shadows
- **Performance Optimization**: Reduced frequent data request intervals
  - GPU monitor: 5s → 10s update frequency
  - Performance monitor: 5s → 10s update frequency
  - Smart monitor management: auto-stop when tab not active

### Technical
- All 13 HTML pages synchronized and updated
- Consistent navigation component across entire application
- Modern CSS with gradient themes and responsive design
- Optimized JavaScript for better performance
- Fixed console errors and improved error handling
');
    `
}

// getAddChangelogV1119Migration returns SQL for adding changelog v1.11.9 (v46 migration)
// Version 1.11.9: Enhanced Audit Logging
func (s *SQLiteDB) getAddChangelogV1119Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.11.9', '2025-10-26', '## [1.11.9] - 2025-10-26

### Added
- **Enhanced Audit Logging**: Comprehensive audit trail for critical operations
  - User operations: creation, deletion, enable/disable (LogUserCreated, LogUserDeleted, LogUserUpdated)
  - API key operations: creation and deletion tracking (LogAPIKeyCreated, LogAPIKeyDeleted)
  - Tenant operations: creation, updates, deletion (LogTenantCreated, LogTenantUpdated, LogTenantDeleted)
  - Backup operations: creation and restoration tracking (LogBackupCreated, LogBackupRestored)
  - Performance monitoring: reduced update frequency from 5s to 10s for GPU and system metrics
  - WebUI performance: monitors now stop when not actively viewing System tab

### Technical
- Added AuditLogger integration to handlers: AdminUserHandler, UserHandler, TenantHandler, BackupHandler
- New audit methods in internal/services/audit/logger.go: LogUserUpdated(), LogTenantUpdated()
- Updated handler constructors to accept *audit.AuditLogger parameter
- Router injection of auditLogger into all relevant handlers
- WebUI optimization: admin.js now stops performance/GPU monitors when switching tabs

### Security
- **Audit trail for CRITICAL operations**: User deletion (data loss risk), Backup restoration (overwrites current data), Tenant deletion (organization data loss), API key operations (security credentials)
');
    `
}

// getAddChangelogV1117Migration returns SQL for adding changelog v1.11.7 (v44 migration)
// Version 1.11.7: Usage Quotas System
func (s *SQLiteDB) getAddChangelogV1117Migration() string {
	return `
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
    `
}

// getAddChangelogV1121Migration returns SQL for adding changelog v1.12.1 (v47 migration)
func (s *SQLiteDB) getAddChangelogV1121Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.12.1', '2025-10-26', '## [1.12.1] - 2025-10-26

### Added
- **Model Preloading & Warming**: Механизм предзагрузки моделей для устранения cold start задержки
  - Preload моделей при старте сервера (настраиваемый список в конфигурации)
  - Health check loop для поддержания моделей в горячем состоянии
  - Автоматическая выгрузка неиспользуемых моделей через настраиваемый timeout
  - Track model usage для оптимизации preloading
  - Admin API endpoints:
    - GET /api/admin/models/loaded - список загруженных моделей с статусом
    - POST /api/admin/models/:name/preload - ручная загрузка модели

### Changed
- **Chat Handler**: Автоматический tracking использования моделей при каждом запросе
- **Configuration**: Добавлена секция models.preload с полной настройкой preloading

### Technical
- Новый сервис internal/services/model/preloader.go:
  - ModelPreloader с async startup и health check loops
  - Thread-safe tracking загруженных моделей
  - Graceful shutdown при остановке сервера
- Новый handler internal/api/handlers/model_preload.go для Admin API
- Integration в Router через NewOptions.ModelPreloader
- Integration в ChatHandler через ModelPreloader interface
- Конфигурация:
  - models.preload.enabled - включить/выключить preloading
  - models.preload.on_startup - загружать при старте
  - models.preload.keep_warm - поддерживать в горячем состоянии
  - models.preload.health_check_interval - интервал проверки (default: 5m)
  - models.preload.warm_up_prompt - тестовый промпт (default: "Hello")
  - models.preload.max_loaded_models - лимит одновременно загруженных (0 = unlimited)
  - models.preload.unload_after - timeout выгрузки (0 = never)

### Performance
- **First Request Latency**: Сокращение времени первого ответа с 5-30s до <1s для preloaded моделей
- **Memory Management**: LRU eviction через Ollama при достижении лимита памяти
- **Non-blocking**: Async preload не блокирует startup сервера

### Documentation
- Updated configs/dev.yaml с примером конфигурации preloading
- API documentation для Admin endpoints в Roadmap');
    `
}

// getAddAdvancedRateLimitingMigration returns SQL for advanced rate limiting (v48 migration)
func (s *SQLiteDB) getAddAdvancedRateLimitingMigration() string {
	return `
-- Advanced Rate Limits Table (v1.12.2+: RATE-02)
CREATE TABLE IF NOT EXISTS rate_limits (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    
    -- Scope: "global", "tenant", "user", "api_key", "model"
    scope TEXT NOT NULL DEFAULT 'global',
    
    -- Target ID: depends on scope (user_id, tenant_id, api_key_id, model_name, "all" для global)
    target_id TEXT,
    
    -- Model-specific rate limit (optional)
    model_name TEXT,
    
    -- Rate limits (NULL = no limit)
    requests_per_second INTEGER,
    requests_per_minute INTEGER,
    requests_per_hour INTEGER,
    requests_per_day INTEGER,
    
    -- Burst allowance
    burst_size INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_rate_limits_scope_target ON rate_limits(scope, target_id);
CREATE INDEX IF NOT EXISTS idx_rate_limits_model ON rate_limits(model_name) WHERE model_name IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rate_limits_scope ON rate_limits(scope);

-- Rate Limit Usage Tracking (for sliding window)
CREATE TABLE IF NOT EXISTS rate_limit_usage (
    id TEXT PRIMARY KEY,
    rate_limit_id TEXT NOT NULL,
    window_type TEXT NOT NULL, -- "second", "minute", "hour", "day"
    window_start TIMESTAMP NOT NULL,
    request_count INTEGER DEFAULT 0,
    last_request_at TIMESTAMP,
    
    FOREIGN KEY (rate_limit_id) REFERENCES rate_limits(id) ON DELETE CASCADE
);

-- Index for sliding window queries
CREATE INDEX IF NOT EXISTS idx_rate_limit_usage_window ON rate_limit_usage(rate_limit_id, window_type, window_start DESC);
CREATE INDEX IF NOT EXISTS idx_rate_limit_usage_cleanup ON rate_limit_usage(window_start);
    `
}

// getAddChangelogV1122Migration returns SQL for adding changelog v1.12.2 (v49 migration)
func (s *SQLiteDB) getAddChangelogV1122Migration() string {
	return `
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
    `
}

// getAddChangelogV1123Migration returns SQL for adding changelog v1.12.3 (v50 migration)
func (s *SQLiteDB) getAddChangelogV1123Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.12.3', '2025-10-26', '## [1.12.3] - 2025-10-26

### Added
- **Conversation Export/Import**: Полная система экспорта и импорта conversations
  - Export форматы: JSON, Markdown, Text
  - Single conversation export через GET /api/conversations/{id}/export?format=json|markdown|text
  - Bulk export через POST /api/conversations/bulk-export
  - Import из JSON через POST /api/conversations/import
  - Import опции: Merge into existing, Preserve timestamps/IDs
  - Metadata export: total messages, tokens used, model, dates

### Technical
- Новый сервис internal/services/export/conversation_exporter.go
- Новый сервис internal/services/export/conversation_importer.go
- Новый handler internal/api/handlers/conversation_export.go
- Data Models в internal/models/conversation_export.go

### Security
- **Access Control**: Verify conversation ownership при export/import
- **User Isolation**: Импорт только в свой tenant/user scope');
    `
}

// getCreateRAGTablesMigration returns SQL for creating RAG tables (v51 migration - v1.13.1)
func (s *SQLiteDB) getCreateRAGTablesMigration() string {
	return `
-- ========================================
-- RAG Data Sources Table (v1.13.1)
-- ========================================
CREATE TABLE IF NOT EXISTS rag_data_sources (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    tenant_id TEXT,
    
    -- Основная информация
    name TEXT NOT NULL,
    description TEXT,
    source_type TEXT NOT NULL CHECK (source_type IN ('file', 'api', 'database', 'web')),
    
    -- Конфигурация (JSON)
    config TEXT NOT NULL DEFAULT '{}',
    
    -- Credentials (зашифрованные AES-256)
    credentials_encrypted TEXT,
    
    -- Статус и метрики
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'error', 'syncing')),
    last_sync_at DATETIME,
    last_sync_status TEXT CHECK (last_sync_status IN ('success', 'failed', 'partial')),
    last_error TEXT,
    sync_frequency TEXT,  -- "6h", "daily", "weekly"
    
    -- Настройки индексации
    indexing_config TEXT NOT NULL DEFAULT '{}',
    
    -- Статистика
    total_chunks INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    last_chunk_count INTEGER,
    
    -- Метаданные
    tags TEXT,  -- JSON array
    is_shared INTEGER DEFAULT 0,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rag_sources_user ON rag_data_sources(user_id);
CREATE INDEX IF NOT EXISTS idx_rag_sources_tenant ON rag_data_sources(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rag_sources_type ON rag_data_sources(source_type);
CREATE INDEX IF NOT EXISTS idx_rag_sources_status ON rag_data_sources(status);

-- ========================================
-- RAG Documents Table (v1.13.1)
-- ========================================
CREATE TABLE IF NOT EXISTS rag_documents (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    
    -- Файл информация
    filename TEXT,
    mime_type TEXT,
    size_bytes INTEGER,
    
    -- Хранилище
    storage_backend TEXT,  -- 'local', 's3'
    storage_path TEXT NOT NULL,
    storage_bucket TEXT,  -- для S3
    
    -- Обработка
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    processing_started_at DATETIME,
    processing_completed_at DATETIME,
    processing_error TEXT,
    
    -- Метаданные документа
    metadata TEXT DEFAULT '{}',
    
    -- Статистика
    total_chunks INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (source_id) REFERENCES rag_data_sources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rag_docs_source ON rag_documents(source_id);
CREATE INDEX IF NOT EXISTS idx_rag_docs_status ON rag_documents(status);

-- ========================================
-- RAG Chunks Table (v1.13.1)
-- Note: Embeddings будут добавлены в v1.13.3 после pgvector setup
-- ========================================
CREATE TABLE IF NOT EXISTS rag_chunks (
    id TEXT PRIMARY KEY,
    document_id TEXT NOT NULL,
    source_id TEXT NOT NULL,
    
    -- Chunk контент
    chunk_text TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,  -- позиция в документе
    chunk_tokens INTEGER NOT NULL,
    
    -- Метаданные чанка
    metadata TEXT DEFAULT '{}',  -- page_number, headers, context, etc.
    
    -- Для overlap detection
    start_offset INTEGER,
    end_offset INTEGER,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (document_id) REFERENCES rag_documents(id) ON DELETE CASCADE,
    FOREIGN KEY (source_id) REFERENCES rag_data_sources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rag_chunks_document ON rag_chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_source ON rag_chunks(source_id);

-- ========================================
-- RAG Jobs Queue (v1.13.1)
-- ========================================
CREATE TABLE IF NOT EXISTS rag_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_type TEXT NOT NULL,  -- 'file_upload', 'api_sync', 'db_query', 'web_scrape'
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    
    -- Данные
    payload TEXT NOT NULL,  -- JSON
    result TEXT,            -- JSON
    
    -- Приоритет и повторы
    priority INTEGER DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    
    -- Timestamps
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME,
    
    -- Ошибки
    error TEXT,
    
    -- Для visibility timeout
    locked_until DATETIME
);

CREATE INDEX IF NOT EXISTS idx_rag_jobs_status ON rag_jobs(status, priority DESC, created_at);
CREATE INDEX IF NOT EXISTS idx_rag_jobs_type ON rag_jobs(job_type);

-- ========================================
-- RAG Query Logs (v1.13.1 - аналитика)
-- ========================================
CREATE TABLE IF NOT EXISTS rag_query_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT,
    conversation_id TEXT,
    
    -- Запрос
    query_text TEXT NOT NULL,
    
    -- Использованные источники (JSON array of IDs)
    source_ids TEXT,
    
    -- Результаты поиска
    chunks_retrieved INTEGER,
    chunks_used INTEGER,
    
    -- Метрики
    search_time_ms INTEGER,
    total_tokens_used INTEGER,
    
    -- Результат
    response_quality_score REAL,  -- опционально, от пользователя
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX IF NOT EXISTS idx_rag_query_logs_user ON rag_query_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_rag_query_logs_created ON rag_query_logs(created_at DESC);
    `
}

// getAddChangelogV200Migration returns SQL for adding changelog v2.0.0 (v52 migration)
func (s *SQLiteDB) getAddChangelogV200Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('2.0.0', '2025-10-27', '## [2.0.0] - 2025-10-27

### 🚀 Major Features

- **RAG System (Retrieval-Augmented Generation)**: Полная интеграция системы RAG для работы с внешними источниками данных
  - Поддержка REST API, PostgreSQL, File Upload, Web Scraping
  - Semantic chunking с intelligent text splitting
  - Vector embeddings через Ollama (mxbai-embed-large, nomic-embed-text)
  - PgVector для similarity search с HNSW indexing
  - RAG Orchestrator с reranking и context assembly
  - Query logging для analytics

- **RAG Management WebUI**: Полнофункциональный интерфейс управления RAG
  - User Dashboard (/rag-sources.html) - CRUD операции для личных RAG источников
  - Admin Panel (/admin-rag.html) - управление всеми источниками
  - Chat Integration - RAG toggle, source selector, параметры в /chat.html

- **Browser Testing Integration**: MCP browser extension для E2E тестирования
  - Chrome automation, accessibility snapshots, screenshots

### 🎨 UI/UX Improvements

- **Modal Windows Centering**: Модалки теперь по центру экрана (horizontal + vertical)
- **Consistent Dashboard Styling**: Единообразный dark theme на всех страницах
- **Login Page Redesign**: Двухколоночный layout с gradient background и info panel
- **Error Messages Styling**: Улучшенный контраст и visibility
- **RBAC/Audit Page Refactoring**: Удален Bootstrap, full custom CSS

### 🔐 Security & Authentication

- **JWT Token Rotation**: Refresh token rotation для enhanced security
- **Authentication Flow Fixes**: Исправлен logout loop
- **Public API Endpoints**: /api/models доступен без аутентификации
- **RAG Credentials Encryption**: AES-256 шифрование credentials

### 📊 Logging & Monitoring

- **Separate Error Logging**: Dedicated error log file с rotation
- **Audit Events Metadata Fix**: JSON serialization для metadata

### 🛠️ Technical Improvements

- **API Client Enhancements**: Generic HTTP methods + RAG/RBAC/Audit methods
- **Context Parsing Fix**: User/Tenant ID prefix stripping
- **CSS Conflicts Resolution**: Исправлено позиционирование модалок
- **Navigation Component**: Поддержка новых RAG страниц

### 📚 Documentation

- RAG Deployment Guide, Config Guide, Testing Guide
- Error Logging Guide

### 🔧 Configuration

- **RAG Configuration**: Полная секция rag в config
- **Logging Enhancements**: error_log_* настройки

### 🗃️ Database

- **RAG Schema**: rag_data_sources, rag_documents, rag_chunks, rag_jobs, rag_query_logs

### 🧪 Testing

- **Go Unit Tests**: 70+ tests для RAG components
- **Playwright E2E Tests**: Browser-based UI testing

### 🐛 Bug Fixes

- Fixed modal windows appearing off-center
- Fixed logout loop when refresh token is blacklisted
- Fixed user_id UUID parsing with prefix
- Fixed audit events metadata serialization error
- Fixed white-on-white text readability
- Fixed RBAC page non-clickable buttons
- Fixed MCP Catalog styling issues

### ⚡ Performance

- Worker Pool для document processing
- Batch embeddings для Ollama
- Connection pooling для HTTP и PostgreSQL

### 🔄 Breaking Changes

- **Version Jump**: 1.12.3 → 2.0.0 (major release)
- **New Dependencies**: PostgreSQL with pgvector, Ollama с embedding models
- **Configuration Changes**: Новый раздел rag в config
- **Database Schema**: Новые таблицы для RAG

### 📦 Dependencies

- Added github.com/pgvector/pgvector-go
- Added Playwright для E2E testing
- Added lumberjack для log rotation');
    `
}

// getUpdateChangelogV200WithExportImportMigration returns SQL for updating changelog v2.0.0 with Export/Import UI info (v53 migration)
func (s *SQLiteDB) getUpdateChangelogV200WithExportImportMigration() string {
	return `
UPDATE changelogs SET content = '## [2.0.0] - 2025-10-27

### 🚀 Major Features

- **RAG System (Retrieval-Augmented Generation)**: Полная интеграция системы RAG для работы с внешними источниками данных
  - Поддержка REST API, PostgreSQL, File Upload, Web Scraping
  - Semantic chunking с intelligent text splitting
  - Vector embeddings через Ollama (mxbai-embed-large, nomic-embed-text)
  - PgVector для similarity search с HNSW indexing
  - RAG Orchestrator с reranking и context assembly
  - Query logging для analytics

- **RAG Management WebUI**: Полнофункциональный интерфейс управления RAG
  - User Dashboard (/rag-sources.html) - CRUD операции для личных RAG источников
  - Admin Panel (/admin-rag.html) - управление всеми источниками
  - Chat Integration - RAG toggle, source selector, параметры в /chat.html

- **Chat Export/Import UI**: Полнофункциональный интерфейс экспорта и импорта conversations
  - Export Dropdown в Chat Header: JSON, Markdown, Text форматы
  - Автоматическое скачивание файла с sanitized filename
  - Import Modal: Upload JSON с опциями preserve timestamps/IDs
  - Validation JSON структуры перед импортом
  - Success notification с количеством imported messages

- **Browser Testing Integration**: MCP browser extension для E2E тестирования
  - Chrome automation, accessibility snapshots, screenshots

### 🎨 UI/UX Improvements

- **Modal Windows Centering**: Модалки теперь по центру экрана (horizontal + vertical)
- **Consistent Dashboard Styling**: Единообразный dark theme на всех страницах
- **Login Page Redesign**: Двухколоночный layout с gradient background и info panel
- **Error Messages Styling**: Улучшенный контраст и visibility
- **RBAC/Audit Page Refactoring**: Удален Bootstrap, full custom CSS
- **Export/Import Dropdown**: Stylish dropdown menu с animations

### 🔐 Security & Authentication

- **JWT Token Rotation**: Refresh token rotation для enhanced security
- **Authentication Flow Fixes**: Исправлен logout loop
- **Public API Endpoints**: /api/models доступен без аутентификации
- **RAG Credentials Encryption**: AES-256 шифрование credentials

### 📊 Logging & Monitoring

- **Separate Error Logging**: Dedicated error log file с rotation
- **Audit Events Metadata Fix**: JSON serialization для metadata

### 🛠️ Technical Improvements

- **API Client Enhancements**: Generic HTTP methods + RAG/RBAC/Audit methods
- **Context Parsing Fix**: User/Tenant ID prefix stripping
- **CSS Conflicts Resolution**: Исправлено позиционирование модалок
- **Navigation Component**: Поддержка новых RAG страниц
- **Export API Integration**: Frontend integration для conversation export/import

### 📚 Documentation

- RAG Deployment Guide, Config Guide, Testing Guide
- Error Logging Guide

### 🔧 Configuration

- **RAG Configuration**: Полная секция rag в config
- **Logging Enhancements**: error_log_* настройки

### 🗃️ Database

- **RAG Schema**: rag_data_sources, rag_documents, rag_chunks, rag_jobs, rag_query_logs

### 🧪 Testing

- **Go Unit Tests**: 70+ tests для RAG components
- **Playwright E2E Tests**: Browser-based UI testing

### 🐛 Bug Fixes

- Fixed modal windows appearing off-center
- Fixed logout loop when refresh token is blacklisted
- Fixed user_id UUID parsing with prefix
- Fixed audit events metadata serialization error
- Fixed white-on-white text readability
- Fixed RBAC page non-clickable buttons
- Fixed MCP Catalog styling issues

### ⚡ Performance

- Worker Pool для document processing
- Batch embeddings для Ollama
- Connection pooling для HTTP и PostgreSQL

### 🔄 Breaking Changes

- **Version Jump**: 1.12.3 → 2.0.0 (major release)
- **New Dependencies**: PostgreSQL with pgvector, Ollama с embedding models
- **Configuration Changes**: Новый раздел rag в config
- **Database Schema**: Новые таблицы для RAG

### 📦 Dependencies

- Added github.com/pgvector/pgvector-go
- Added Playwright для E2E testing
- Added lumberjack для log rotation' 
WHERE version = '2.0.0';
    `
}

// getAddChangelogV210Migration returns SQL for adding changelog v2.1.0 (v54 migration)
func (s *SQLiteDB) getAddChangelogV210Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('2.1.0', '2025-10-27', '## [2.1.0] - 2025-10-27

### 🏷️ Major Rebranding

**Project renamed from "Ollama-OpenAI Proxy" to "AIGateway Platform"**

### Changed

- **Project Identity**:
  - Repository name: ollama-openai-proxy → aigateway
  - Module path: ollama-openai-proxy → aigateway
  - Container name: ollama-openai-proxy → aigateway
  - Database file: proxy.db → aigateway.db
  - Log files: proxy.log → aigateway.log

- **Environment Variables** (Breaking Change ⚠️):
  - All PROXY_* variables → AIGATEWAY_*
  - Example: PROXY_SERVER_PORT → AIGATEWAY_SERVER_PORT

- **Documentation**:
  - ✅ README.md - complete rewrite для AIGateway Platform
  - ✅ Architecture.MD - updated diagrams with RAG System, Model Registry
  - ✅ All docs/*.md files - branding updates (20+ files)

- **Code**:
  - ✅ go.mod module path updated
  - ✅ All import statements across entire codebase
  - ✅ Docker Compose configuration
  - ✅ Dockerfile с новым VERSION=2.1.0

- **WebUI**:
  - ✅ All HTML page titles: "AIGateway Platform"
  - ✅ Navigation labels и headers
  - ✅ About System page
  - ✅ Footer copyright

### Why Rebranding?

1. **Expanded Scope**: No longer just an Ollama proxy - now supports multiple model providers (vLLM, future: OpenAI, Anthropic)
2. **RAG System**: Built-in RAG capabilities make it more than a proxy
3. **Enterprise Positioning**: "Gateway" better represents the platform''s role as AI infrastructure
4. **Scalability**: Name allows for future expansion to cloud providers and custom models

### 🔄 Migration Required

**This is a BREAKING release.** Existing deployments need migration.

See [MIGRATION_GUIDE_v2.1.0.md](docs/MIGRATION_GUIDE_v2.1.0.md) for detailed migration steps.

**Quick Migration Checklist:**
- Update environment variables: PROXY_* → AIGATEWAY_*
- Update Docker image names
- Rename database file (optional): proxy.db → aigateway.db
- Update any scripts/configs referencing old names
- Pull new Docker images: aigateway:2.1.0

### Technical

- Module path: aigateway (was ollama-openai-proxy)
- All import paths updated throughout codebase
- Docker Compose volumes: aigateway_data, aigateway_logs
- Zero functional changes - pure rebranding release');
    `
}

// getUpdateChangelogV210FormattingMigration returns SQL for updating changelog v2.1.0 formatting (v55 migration)
func (s *SQLiteDB) getUpdateChangelogV210FormattingMigration() string {
	return `
UPDATE changelogs 
SET content = '## [2.1.0] - 2025-10-27

### 🏷️ Major Rebranding

**Project renamed from "Ollama-OpenAI Proxy" to "AIGateway Platform"**

---

### Changed

#### **Project Identity**

- Repository name: ollama-openai-proxy → aigateway
- Module path: ollama-openai-proxy → aigateway  
- Container name: ollama-openai-proxy → aigateway
- Database file: proxy.db → aigateway.db
- Log files: proxy.log → aigateway.log

#### **Environment Variables** (Breaking Change ⚠️)

- All PROXY_* variables → AIGATEWAY_*
- Example: PROXY_SERVER_PORT → AIGATEWAY_SERVER_PORT
- See [Migration Guide](docs/MIGRATION_GUIDE_v2.1.0.md) for complete variable mapping

#### **Documentation**

- ✅ README.md - complete rewrite для AIGateway Platform
- ✅ Architecture.MD - updated diagrams with RAG System, Model Registry  
- ✅ All docs/*.md files - branding updates (20+ files)
- ✅ All BACKLOG/*.md files - task descriptions updated

#### **Code**

- ✅ go.mod module path updated
- ✅ All import statements across entire codebase
- ✅ Docker Compose configuration
- ✅ Dockerfile с новым VERSION=2.1.0

#### **WebUI**

- ✅ All HTML page titles: "AIGateway Platform"
- ✅ Navigation labels и headers
- ✅ About System page
- ✅ Footer copyright

---

### Why Rebranding?

**Reasons for transition to "AIGateway":**

1. **Expanded Scope**: No longer just an Ollama proxy - now supports multiple model providers (vLLM, future: OpenAI, Anthropic)

2. **RAG System**: Built-in RAG capabilities make it more than a proxy

3. **Enterprise Positioning**: "Gateway" better represents the platform''s role as AI infrastructure

4. **Scalability**: Name allows for future expansion to cloud providers and custom models

---

### 🔄 Migration Required

**This is a BREAKING release.** Existing deployments need migration.

See [MIGRATION_GUIDE_v2.1.0.md](docs/MIGRATION_GUIDE_v2.1.0.md) for detailed migration steps.

**Quick Migration Checklist:**

- Update environment variables: PROXY_* → AIGATEWAY_*
- Update Docker image names
- Rename database file (optional): proxy.db → aigateway.db
- Update any scripts/configs referencing old names
- Pull new Docker images: aigateway:2.1.0

---

### Technical

- **Module path**: aigateway (was ollama-openai-proxy)
- **Import paths**: Updated throughout codebase
- **Docker volumes**: aigateway_data, aigateway_logs
- **Functional changes**: Zero - pure rebranding release
- **Compilation**: Verified ✅'
WHERE version = '2.1.0';
    `
}

// getUpdateChangelogV210FinalFormatMigration returns SQL for final v2.1.0 formatting fix (v56 migration)
func (s *SQLiteDB) getUpdateChangelogV210FinalFormatMigration() string {
	return `
UPDATE changelogs SET content = '## [2.1.0] - 2025-10-27

### 🏷️ Major Rebranding

**Project renamed from "Ollama-OpenAI Proxy" to "AIGateway Platform"**

### Changed

- **Repository name**: ollama-openai-proxy → aigateway
- **Module path**: ollama-openai-proxy → aigateway
- **Container name**: ollama-openai-proxy → aigateway
- **Database file**: proxy.db → aigateway.db (optional rename)
- **Log files**: proxy.log → aigateway.log

- **Environment Variables (⚠️ Breaking Change)**: All PROXY_* → AIGATEWAY_*
  - Example: PROXY_SERVER_PORT → AIGATEWAY_SERVER_PORT
  - PROXY_OLLAMA_URL → AIGATEWAY_OLLAMA_URL
  - PROXY_DATABASE_TYPE → AIGATEWAY_DATABASE_TYPE
  - See [Migration Guide](docs/MIGRATION_GUIDE_v2.1.0.md) for complete mapping

- **Documentation Updates**: 40+ files rebranded
  - README.md - complete rewrite для AIGateway Platform
  - Architecture.MD - updated diagrams with RAG System, Model Registry
  - All docs/*.md files (20+ files)
  - All BACKLOG/*.md files

- **Code Changes**: Zero functional changes - pure rebranding
  - go.mod module path updated
  - All import statements across entire codebase
  - Docker Compose configuration
  - Dockerfile VERSION=2.1.0

- **WebUI Branding**: Complete frontend rebranding
  - All HTML page titles: "AIGateway Platform"
  - Navigation labels и headers
  - About System page
  - Footer copyright

### Why Rebranding?

**Reasons for transition to "AIGateway":**

1. **Expanded Scope**: No longer just an Ollama proxy
  - Multi-provider support: vLLM (v2.2.0), future: OpenAI, Anthropic
  - Model registry для unified API access

2. **RAG System**: Built-in RAG capabilities
  - Vector search, embeddings, document processing
  - Enterprise-ready data integration

3. **Enterprise Positioning**: "Gateway" better represents platform role
  - Central AI infrastructure component
  - Unified API для multiple backends

4. **Scalability**: Name allows future expansion
  - Cloud provider integration
  - Custom model support

### 🔄 Migration Required

**This is a BREAKING release.** Existing deployments need migration.

See [MIGRATION_GUIDE_v2.1.0.md](docs/MIGRATION_GUIDE_v2.1.0.md) for detailed steps.

**Quick Migration Checklist:**
  - Update environment variables: PROXY_* → AIGATEWAY_*
  - Update Docker image names
  - Rename database file (optional): proxy.db → aigateway.db
  - Update scripts/configs referencing old names
  - Pull new Docker images: aigateway:2.1.0

### Technical

- **Module path**: aigateway (was ollama-openai-proxy)
- **Import paths**: Updated throughout codebase (~150+ Go files)
- **Docker Compose**: Service name aigateway, volumes aigateway_data/aigateway_logs
- **Functional changes**: Zero - pure rebranding release
- **Compilation**: Verified ✅
- **Database schema**: Unchanged (backward compatible)'
WHERE version = '2.1.0';
    `
}

// getAddInvitationsTableMigration returns SQL for invitations table (v57, AUTH-03, v2.2.0)
func (s *SQLiteDB) getAddInvitationsTableMigration() string {
	return `
-- ========================================
-- Invitations Table (AUTH-03: Invitation-Only Registration System, v2.2.0)
-- ========================================
CREATE TABLE IF NOT EXISTS invitations (
	id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
	token TEXT UNIQUE NOT NULL,
	
	-- Creation metadata
	created_by_user_id TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	
	-- Constraints
	email TEXT,                      -- Optional: bind to specific email
	expires_at TIMESTAMP,            -- Optional: expiration date
	max_uses INTEGER NOT NULL DEFAULT 1,  -- Default: single-use
	
	-- Usage tracking
	current_uses INTEGER NOT NULL DEFAULT 0,
	used_at TIMESTAMP,               -- First successful registration
	used_by_user_id TEXT,
	
	-- Revocation
	revoked_at TIMESTAMP,
	revoked_by_user_id TEXT,
	revoke_reason TEXT,
	
	-- Foreign keys
	FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (used_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
	FOREIGN KEY (revoked_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
	
	-- Constraints
	CHECK (current_uses <= max_uses)
);

-- Indexes for performance
CREATE INDEX idx_invitations_token ON invitations(token);
CREATE INDEX idx_invitations_created_by ON invitations(created_by_user_id);
CREATE INDEX idx_invitations_status ON invitations(expires_at, revoked_at, current_uses, max_uses);
CREATE INDEX idx_invitations_email ON invitations(email) WHERE email IS NOT NULL;
	`
}

// getAddChangelogV220Migration returns SQL for adding changelog v2.2.0 (v58 migration)
func (s *SQLiteDB) getAddChangelogV220Migration() string {
	return `
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('2.2.0', '2025-10-28', '## [2.2.0] - 2025-10-28

### Added

- **Invitation-Only Registration System (AUTH-03)**: Полная система управления приглашениями для контролируемой регистрации пользователей
  - Database schema с таблицей invitations (migration v57 для SQLite, v4 для PostgreSQL)
  - REST API endpoints для создания, просмотра, отзыва и валидации приглашений
  - Admin WebUI: /admin-invitations.html - страница управления приглашениями с фильтрацией и статистикой
  - Поддержка ограничений: по email, сроку действия, количеству использований
  - Детальная информация о пользователях: кто создал, кто использовал, кто отозвал приглашение
  - Регистрация по приглашению: обновлен /register.html с поддержкой invitation tokens

- **Invitation Management Features**:
  - Создание приглашений с настраиваемыми параметрами (email restriction, expiry, max uses)
  - Автоматическая генерация уникальных invitation links
  - Статистика приглашений (Active, Pending, Used, Expired, Revoked)
  - Фильтрация по статусу, email, создателю
  - View modal с полной информацией включая user details через LEFT JOIN
  - One-click копирование invitation links и tokens

- **Configuration Options**: Новые настройки в configs/dev.yaml
  - auth.registration.mode: "open" | "invitation_only" | "disabled"
  - auth.invitations.enabled: включение системы приглашений
  - auth.invitations.default_expiry_days: срок действия по умолчанию
  - auth.invitations.max_uses_default: количество использований
  - Rate limits для создания и валидации приглашений

### Changed

- **WebUI Improvements**:
  - Улучшен контраст текста в статистических карточках (белый текст на цветных градиентах)
  - Markdown форматирование для пользовательских сообщений в ChatUI
  - WYSIWYG-подобная панель форматирования текста при выделении (bold, italic, code, lists)
  - Сохранение переносов строк в сообщениях чата (white-space: pre-wrap)

- **Registration Flow**: Обновлен процесс регистрации с проверкой invitation tokens
  - Валидация токена перед показом формы регистрации
  - Автоматическое использование приглашения после успешной регистрации
  - Email restriction check для приглашений привязанных к конкретному email

### Fixed

- **Database Schema**: Исправлена ошибка с колонкой display_name → full_name в запросах с JOIN к таблице users
- **Invitation Links**: Исправлена генерация ссылок - добавлено .html расширение (/register.html?invite=...)

### Technical

- **Backend (Go)**:
  - Новые модели: Invitation, InvitationWithUsers, UserInfo, InvitationStatus
  - Storage layer: полная реализация CRUD операций для SQLite и PostgreSQL
  - Handler: InvitationHandler с 6 endpoint''ами (create, list, stats, details, revoke, validate)
  - AuthService: интеграция invitation token validation в процесс регистрации
  - Transaction delegation: добавлены методы в sqliteTx и postgresqlTx

- **Database Migrations**:
  - SQLite migration v57: создание таблицы invitations с индексами
  - PostgreSQL migration v4: аналогичная схема для PostgreSQL
  - LEFT JOIN queries для получения информации о пользователях

- **API Endpoints**:
  - POST /api/admin/invitations - создание приглашения
  - GET /api/admin/invitations - список приглашений с фильтрацией
  - GET /api/admin/invitations/stats - статистика
  - GET /api/admin/invitations/:id - детальная информация с user info
  - DELETE /api/admin/invitations/:id - отзыв приглашения
  - GET /api/invitations/:token/validate - публичная валидация токена

- **Frontend**:
  - web/admin-invitations.html (623 строки) - полнофункциональная админ-панель
  - web/js/api.js - 6 новых методов для работы с invitations API
  - web/register.html - поддержка ?invite= query parameter
  - Formatting toolbar для ChatUI с keyboard shortcuts (Ctrl+B, Ctrl+I, Ctrl+K, Ctrl+L)

### Security

- **Access Control**: Все admin endpoints защищены JWT authentication + RequireAdmin middleware
- **Rate Limiting**: Настраиваемые лимиты для создания приглашений и валидации токенов
- **Token Security**: UUID v4 tokens для приглашений, проверка валидности перед использованием
- **Email Verification**: Опциональная привязка приглашения к конкретному email');
	`
}

// getSeedRBACPermissionsMigration returns SQL for seeding RBAC permissions (v59 migration)
func (s *SQLiteDB) getSeedRBACPermissionsMigration() string {
	return `
-- Seed RBAC Permissions (v2.2.0+)
INSERT OR IGNORE INTO permissions (id, name, description, resource, action, scope, created_at) VALUES
-- API Keys permissions
('perm_api_keys_create', 'api_keys:create', 'Create API keys', 'api_keys', 'create', 'global', CURRENT_TIMESTAMP),
('perm_api_keys_read', 'api_keys:read', 'View API keys', 'api_keys', 'read', 'global', CURRENT_TIMESTAMP),
('perm_api_keys_update', 'api_keys:update', 'Update API keys', 'api_keys', 'update', 'global', CURRENT_TIMESTAMP),
('perm_api_keys_delete', 'api_keys:delete', 'Delete API keys', 'api_keys', 'delete', 'global', CURRENT_TIMESTAMP),

-- Users permissions
('perm_users_create', 'users:create', 'Create users', 'users', 'create', 'global', CURRENT_TIMESTAMP),
('perm_users_read', 'users:read', 'View users', 'users', 'read', 'global', CURRENT_TIMESTAMP),
('perm_users_update', 'users:update', 'Update users', 'users', 'update', 'global', CURRENT_TIMESTAMP),
('perm_users_delete', 'users:delete', 'Delete users', 'users', 'delete', 'global', CURRENT_TIMESTAMP),

-- Tenants permissions
('perm_tenants_create', 'tenants:create', 'Create tenants', 'tenants', 'create', 'global', CURRENT_TIMESTAMP),
('perm_tenants_read', 'tenants:read', 'View tenants', 'tenants', 'read', 'global', CURRENT_TIMESTAMP),
('perm_tenants_update', 'tenants:update', 'Update tenants', 'tenants', 'update', 'global', CURRENT_TIMESTAMP),
('perm_tenants_delete', 'tenants:delete', 'Delete tenants', 'tenants', 'delete', 'global', CURRENT_TIMESTAMP),

-- Chat / Models permissions
('perm_chat_use', 'chat:use', 'Use chat interface', 'chat', 'use', 'global', CURRENT_TIMESTAMP),
('perm_models_read', 'models:read', 'View available models', 'models', 'read', 'global', CURRENT_TIMESTAMP),
('perm_models_manage', 'models:manage', 'Manage models', 'models', 'manage', 'global', CURRENT_TIMESTAMP),

-- System permissions
('perm_system_config', 'system:config', 'Configure system settings', 'system', 'config', 'global', CURRENT_TIMESTAMP),
('perm_system_backup', 'system:backup', 'Create system backups', 'system', 'backup', 'global', CURRENT_TIMESTAMP),
('perm_system_logs', 'system:logs', 'View system logs', 'system', 'logs', 'global', CURRENT_TIMESTAMP),
('perm_audit_read', 'audit:read', 'View audit logs', 'audit', 'read', 'global', CURRENT_TIMESTAMP),

-- Files permissions
('perm_files_upload', 'files:upload', 'Upload files', 'files', 'upload', 'global', CURRENT_TIMESTAMP),
('perm_files_read', 'files:read', 'View files', 'files', 'read', 'global', CURRENT_TIMESTAMP),
('perm_files_delete', 'files:delete', 'Delete files', 'files', 'delete', 'global', CURRENT_TIMESTAMP),

-- Conversations permissions
('perm_conversations_create', 'conversations:create', 'Create conversations', 'conversations', 'create', 'global', CURRENT_TIMESTAMP),
('perm_conversations_read', 'conversations:read', 'View conversations', 'conversations', 'read', 'global', CURRENT_TIMESTAMP),
('perm_conversations_update', 'conversations:update', 'Update conversations', 'conversations', 'update', 'global', CURRENT_TIMESTAMP),
('perm_conversations_delete', 'conversations:delete', 'Delete conversations', 'conversations', 'delete', 'global', CURRENT_TIMESTAMP),

-- Invitations permissions (v2.2.0+)
('perm_invitations_create', 'invitations:create', 'Create invitations', 'invitations', 'create', 'global', CURRENT_TIMESTAMP),
('perm_invitations_read', 'invitations:read', 'View invitations', 'invitations', 'read', 'global', CURRENT_TIMESTAMP),
('perm_invitations_revoke', 'invitations:revoke', 'Revoke invitations', 'invitations', 'revoke', 'global', CURRENT_TIMESTAMP),

-- RBAC permissions
('perm_roles_create', 'roles:create', 'Create roles', 'roles', 'create', 'global', CURRENT_TIMESTAMP),
('perm_roles_read', 'roles:read', 'View roles', 'roles', 'read', 'global', CURRENT_TIMESTAMP),
('perm_roles_update', 'roles:update', 'Update roles', 'roles', 'update', 'global', CURRENT_TIMESTAMP),
('perm_roles_delete', 'roles:delete', 'Delete roles', 'roles', 'delete', 'global', CURRENT_TIMESTAMP),
('perm_roles_assign', 'roles:assign', 'Assign roles to users', 'roles', 'assign', 'global', CURRENT_TIMESTAMP),

-- Wildcard permission (Super Admin)
('perm_all', '*:*', 'All permissions (Super Admin)', '*', '*', 'global', CURRENT_TIMESTAMP);
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
