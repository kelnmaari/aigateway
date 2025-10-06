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

CREATE INDEX idx_api_usage_user_id ON api_usage(user_id);
CREATE INDEX idx_api_usage_tenant_id ON api_usage(tenant_id);
CREATE INDEX idx_api_usage_api_key_id ON api_usage(api_key_id);
CREATE INDEX idx_api_usage_created_at ON api_usage(created_at DESC);
CREATE INDEX idx_api_usage_endpoint ON api_usage(endpoint);
CREATE INDEX idx_api_usage_model ON api_usage(model);
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
