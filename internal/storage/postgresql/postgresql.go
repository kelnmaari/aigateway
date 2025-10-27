// Package postgresql provides PostgreSQL implementation of the Database interface.
//
// This package implements a PostgreSQL backend using lib/pq driver.
// Features:
//   - Full ACID compliance
//   - Connection pooling with automatic reconnection
//   - Prepared statements for security
//   - Automatic migrations
//   - SSL/TLS support
//
// Usage:
//
//	db, err := postgresql.New(config.PostgreSQLConfig{...}, logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// PostgreSQLDB представляет PostgreSQL имплементацию Database interface
type PostgreSQLDB struct {
	db     *sql.DB
	config config.PostgreSQLConfig
	logger *logrus.Logger
	mu     sync.RWMutex // Для thread-safe операций

	// Prepared statements cache
	stmts map[string]*sql.Stmt
}

// New создает новый экземпляр PostgreSQLDB
func New(cfg config.PostgreSQLConfig, logger *logrus.Logger) (*PostgreSQLDB, error) {
	if logger == nil {
		logger = logrus.New()
	}

	db := &PostgreSQLDB{
		config: cfg,
		logger: logger,
		stmts:  make(map[string]*sql.Stmt),
	}

	return db, nil
}

// ========================================
// Connection Management
// ========================================

// Connect устанавливает соединение с PostgreSQL БД
func (p *PostgreSQLDB) Connect(ctx context.Context) error {
	p.logger.WithFields(logrus.Fields{
		"host":     p.config.Host,
		"port":     p.config.Port,
		"database": p.config.Database,
	}).Info("Connecting to PostgreSQL database")

	// Build DSN (Data Source Name)
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.config.Host,
		p.config.Port,
		p.config.User,
		p.config.Password,
		p.config.Database,
		p.config.SSLMode,
	)

	// Open database connection
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(p.config.MaxOpenConns)
	db.SetMaxIdleConns(p.config.MaxIdleConns)
	db.SetConnMaxLifetime(p.config.ConnMaxLifetime)

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	p.db = db

	p.logger.Info("Successfully connected to PostgreSQL database")
	return nil
}

// Close закрывает соединение с БД
func (p *PostgreSQLDB) Close() error {
	p.logger.Info("Closing PostgreSQL database connection")

	p.mu.Lock()
	defer p.mu.Unlock()

	// Close prepared statements
	for name, stmt := range p.stmts {
		if err := stmt.Close(); err != nil {
			p.logger.WithError(err).WithField("statement", name).Warn("Failed to close prepared statement")
		}
	}
	p.stmts = make(map[string]*sql.Stmt)

	// Close database
	if p.db != nil {
		return p.db.Close()
	}

	return nil
}

// Ping проверяет доступность БД
func (p *PostgreSQLDB) Ping(ctx context.Context) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	return p.db.PingContext(ctx)
}

// ========================================
// Transaction Support
// ========================================

// postgresqlTx представляет PostgreSQL транзакцию
type postgresqlTx struct {
	tx     *sql.Tx
	db     *PostgreSQLDB
	logger *logrus.Logger
}

// BeginTx начинает новую транзакцию
func (p *PostgreSQLDB) BeginTx(ctx context.Context) (storage.Tx, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted, // PostgreSQL default
	})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	p.logger.Debug("Transaction started")

	return &postgresqlTx{
		tx:     tx,
		db:     p,
		logger: p.logger,
	}, nil
}

// Commit фиксирует транзакцию
func (tx *postgresqlTx) Commit() error {
	if err := tx.tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	tx.logger.Debug("Transaction committed")
	return nil
}

// Rollback откатывает транзакцию
func (tx *postgresqlTx) Rollback() error {
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
func (p *PostgreSQLDB) Migrate(ctx context.Context) error {
	p.logger.Info("Starting database migration")

	// Create migrations table if not exists
	if err := p.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current version
	currentVersion, err := p.GetMigrationVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	p.logger.WithField("current_version", currentVersion).Info("Current migration version")

	// Apply migrations
	migrations := p.getMigrations()
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue // Already applied
		}

		p.logger.WithField("version", migration.Version).Info("Applying migration")

		if err := p.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}

		p.logger.WithField("version", migration.Version).Info("Migration applied successfully")
	}

	finalVersion, _ := p.GetMigrationVersion(ctx)
	p.logger.WithField("version", finalVersion).Info("Database migration completed")

	return nil
}

// GetMigrationVersion возвращает текущую версию схемы
func (p *PostgreSQLDB) GetMigrationVersion(ctx context.Context) (int, error) {
	var version int
	err := p.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM migrations
	`).Scan(&version)

	if err != nil {
		return 0, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, nil
}

// createMigrationsTable создает таблицу для отслеживания миграций
func (p *PostgreSQLDB) createMigrationsTable(ctx context.Context) error {
	_, err := p.db.ExecContext(ctx, `
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
func (p *PostgreSQLDB) getMigrations() []migration {
	return []migration{
		{
			Version: 1,
			Name:    "initial_schema",
			SQL:     p.getInitialSchemaMigration(),
		},
		{
			Version: 2,
			Name:    "add_mcp_servers_table",
			SQL:     p.getMCPServersMigration(),
		},
		{
			Version: 3,
			Name:    "create_rag_tables",
			SQL:     p.getCreateRAGTablesMigration(),
		},
		// Добавляем новые миграции здесь по мере необходимости
	}
}

// applyMigration применяет одну миграцию
func (p *PostgreSQLDB) applyMigration(ctx context.Context, m migration) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		// Check if it's a duplicate object error (migration already applied)
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "42P07" {
			p.logger.WithField("version", m.Version).Warn("Migration objects already exist, skipping")
			return nil
		}
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO migrations (version, applied_at)
		VALUES ($1, CURRENT_TIMESTAMP)
		ON CONFLICT (version) DO NOTHING
	`, m.Version); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// getInitialSchemaMigration возвращает SQL для начальной схемы БД
func (p *PostgreSQLDB) getInitialSchemaMigration() string {
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
	is_admin BOOLEAN NOT NULL DEFAULT FALSE,
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	verified BOOLEAN NOT NULL DEFAULT FALSE,
	verified_at TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_login TIMESTAMP,
	preferences JSONB NOT NULL DEFAULT '{}',
	metadata JSONB
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- ========================================
-- Tenants Table (AUTH-05: Multi-Tenancy)
-- ========================================
CREATE TABLE IF NOT EXISTS tenants (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	slug TEXT NOT NULL UNIQUE,
	type TEXT NOT NULL DEFAULT 'personal',
	description TEXT,
	owner_id TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'active',
	is_active BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	settings JSONB NOT NULL DEFAULT '{}',
	metadata JSONB,
	FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
CREATE INDEX IF NOT EXISTS idx_tenants_owner_id ON tenants(owner_id);
CREATE INDEX IF NOT EXISTS idx_tenants_type ON tenants(type);

-- ========================================
-- Tenant Members Table (AUTH-05)
-- ========================================
CREATE TABLE IF NOT EXISTS tenant_members (
	tenant_id TEXT NOT NULL,
	user_id TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT 'member',
	joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	left_at TIMESTAMP,
	invited_by TEXT,
	metadata JSONB,
	PRIMARY KEY (tenant_id, user_id),
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tenant_members_user_id ON tenant_members(user_id);
CREATE INDEX IF NOT EXISTS idx_tenant_members_role ON tenant_members(role);

-- ========================================
-- API Keys Table (Обновленная для multi-tenancy)
-- ========================================
CREATE TABLE IF NOT EXISTS api_keys (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT,
	key_hash TEXT NOT NULL UNIQUE,
	user_id TEXT,
	tenant_id TEXT,
	scope TEXT NOT NULL DEFAULT 'global',
	models JSONB NOT NULL DEFAULT '["*"]',
	permissions JSONB NOT NULL DEFAULT '[]',
	rate_limits JSONB NOT NULL DEFAULT '{}',
	status TEXT NOT NULL DEFAULT 'active',
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	expires_at TIMESTAMP,
	last_used_at TIMESTAMP,
	revoked_at TIMESTAMP,
	revoked_reason TEXT,
	metadata JSONB,
	usage JSONB NOT NULL DEFAULT '{}',
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_scope ON api_keys(scope);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);

-- ========================================
-- Conversations Table (WEBUI-03: Chat)
-- ========================================
CREATE TABLE IF NOT EXISTS conversations (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	user_id TEXT NOT NULL,
	tenant_id TEXT,
	model TEXT NOT NULL,
	temperature REAL,
	system_prompt TEXT,
	status TEXT NOT NULL DEFAULT 'active',
	is_archived BOOLEAN NOT NULL DEFAULT FALSE,
	is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_message_at TIMESTAMP,
	message_count INTEGER NOT NULL DEFAULT 0,
	total_tokens INTEGER NOT NULL DEFAULT 0,
	metadata JSONB,
	FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
	FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_tenant_id ON conversations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_conversations_status ON conversations(status);
CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at DESC);

-- ========================================
-- Messages Table (WEBUI-03: Chat)
-- ========================================
CREATE TABLE IF NOT EXISTS messages (
	id TEXT PRIMARY KEY,
	conversation_id TEXT NOT NULL,
	role TEXT NOT NULL,
	content TEXT NOT NULL,
	model TEXT,
	temperature REAL,
	tool_calls JSONB,
	tool_call_id TEXT,
	prompt_tokens INTEGER DEFAULT 0,
	completion_tokens INTEGER DEFAULT 0,
	total_tokens INTEGER DEFAULT 0,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	metadata JSONB,
	FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_role ON messages(role);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

-- ========================================
-- API Usage Table (WEBUI-04: Dashboard)
-- ========================================
CREATE TABLE IF NOT EXISTS api_usage (
	id TEXT PRIMARY KEY,
	user_id TEXT NOT NULL,
	tenant_id TEXT,
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
	metadata JSONB,
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
	`
}

// getMCPServersMigration возвращает SQL для создания таблицы mcp_servers (v1.4.5)
func (p *PostgreSQLDB) getMCPServersMigration() string {
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
	tags JSONB, -- PostgreSQL JSONB для эффективного хранения массивов
	is_active BOOLEAN NOT NULL DEFAULT true,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_mcp_servers_category ON mcp_servers(category);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_is_active ON mcp_servers(is_active);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_created_at ON mcp_servers(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_tags ON mcp_servers USING GIN (tags);
	`
}

// getCreateRAGTablesMigration returns SQL for creating RAG tables (v3 migration - v1.13.1)
func (p *PostgreSQLDB) getCreateRAGTablesMigration() string {
	return `
-- ========================================
-- RAG Data Sources Table (v1.13.1)
-- ========================================
CREATE TABLE IF NOT EXISTS rag_data_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id TEXT NOT NULL,
    tenant_id TEXT,
    
    -- Основная информация
    name VARCHAR(255) NOT NULL,
    description TEXT,
    source_type VARCHAR(50) NOT NULL CHECK (source_type IN ('file', 'api', 'database', 'web')),
    
    -- Конфигурация (JSON для гибкости)
    config JSONB NOT NULL DEFAULT '{}',
    
    -- Credentials (зашифрованные AES-256)
    credentials_encrypted TEXT,
    
    -- Статус и метрики
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'error', 'syncing')),
    last_sync_at TIMESTAMP,
    last_sync_status VARCHAR(20) CHECK (last_sync_status IN ('success', 'failed', 'partial')),
    last_error TEXT,
    sync_frequency INTERVAL,  -- PostgreSQL INTERVAL: '6 hours', '1 day', '1 week'
    
    -- Настройки индексации
    indexing_config JSONB DEFAULT '{}',
    
    -- Статистика
    total_chunks INT DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    last_chunk_count INT,
    
    -- Метаданные
    tags TEXT[],
    is_shared BOOLEAN DEFAULT false,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rag_sources_user ON rag_data_sources(user_id);
CREATE INDEX IF NOT EXISTS idx_rag_sources_tenant ON rag_data_sources(tenant_id);
CREATE INDEX IF NOT EXISTS idx_rag_sources_type ON rag_data_sources(source_type);
CREATE INDEX IF NOT EXISTS idx_rag_sources_status ON rag_data_sources(status);
CREATE INDEX IF NOT EXISTS idx_rag_sources_tags ON rag_data_sources USING GIN(tags);

-- ========================================
-- RAG Documents Table (v1.13.1)
-- ========================================
CREATE TABLE IF NOT EXISTS rag_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id UUID NOT NULL,
    
    -- Файл информация
    filename VARCHAR(500),
    mime_type VARCHAR(100),
    size_bytes BIGINT,
    
    -- Хранилище
    storage_backend VARCHAR(20),  -- 'local', 's3'
    storage_path TEXT NOT NULL,
    storage_bucket VARCHAR(255),  -- для S3
    
    -- Обработка
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    processing_started_at TIMESTAMP,
    processing_completed_at TIMESTAMP,
    processing_error TEXT,
    
    -- Метаданные документа
    metadata JSONB DEFAULT '{}',
    
    -- Статистика
    total_chunks INT DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (source_id) REFERENCES rag_data_sources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rag_docs_source ON rag_documents(source_id);
CREATE INDEX IF NOT EXISTS idx_rag_docs_status ON rag_documents(status);

-- ========================================
-- RAG Chunks Table (v1.13.1)
-- Note: Vector embeddings будут добавлены в v1.13.3 после pgvector setup
-- ========================================
CREATE TABLE IF NOT EXISTS rag_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL,
    source_id UUID NOT NULL,
    
    -- Chunk контент
    chunk_text TEXT NOT NULL,
    chunk_index INT NOT NULL,  -- позиция в документе
    chunk_tokens INT NOT NULL,
    
    -- Метаданные чанка
    metadata JSONB DEFAULT '{}',  -- page_number, headers, context, etc.
    
    -- Для overlap detection
    start_offset INT,
    end_offset INT,
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (document_id) REFERENCES rag_documents(id) ON DELETE CASCADE,
    FOREIGN KEY (source_id) REFERENCES rag_data_sources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_rag_chunks_document ON rag_chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_rag_chunks_source ON rag_chunks(source_id);

-- ========================================
-- RAG Jobs Queue (v1.13.1)
-- ========================================
CREATE TABLE IF NOT EXISTS rag_jobs (
    id BIGSERIAL PRIMARY KEY,
    job_type VARCHAR(50) NOT NULL,  -- 'file_upload', 'api_sync', 'db_query', 'web_scrape'
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    
    -- Данные
    payload JSONB NOT NULL,
    result JSONB,
    
    -- Приоритет и повторы
    priority INT DEFAULT 0,
    attempts INT DEFAULT 0,
    max_attempts INT DEFAULT 3,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    -- Ошибки
    error TEXT,
    
    -- Для visibility timeout
    locked_until TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rag_jobs_status ON rag_jobs(status, priority DESC, created_at);
CREATE INDEX IF NOT EXISTS idx_rag_jobs_type ON rag_jobs(job_type);

-- ========================================
-- RAG Query Logs (v1.13.1 - аналитика)
-- ========================================
CREATE TABLE IF NOT EXISTS rag_query_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT,
    conversation_id TEXT,
    
    -- Запрос
    query_text TEXT NOT NULL,
    
    -- Использованные источники
    source_ids UUID[],
    
    -- Результаты поиска
    chunks_retrieved INT,
    chunks_used INT,
    
    -- Метрики
    search_time_ms INT,
    total_tokens_used INT,
    
    -- Результат
    response_quality_score FLOAT,  -- опционально, от пользователя
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    FOREIGN KEY (user_id) REFERENCES users(id),
    FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE INDEX IF NOT EXISTS idx_rag_query_logs_user ON rag_query_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_rag_query_logs_created ON rag_query_logs(created_at DESC);
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

// ========================================
// Reports Statistics Methods (v1.6.3+) - Delegation to DB
// ========================================

func (tx *postgresqlTx) GetUsageStats(ctx context.Context, start, end time.Time) (*models.UsageReportStats, error) {
	return tx.db.GetUsageStats(ctx, start, end)
}

func (tx *postgresqlTx) GetPerformanceStats(ctx context.Context, start, end time.Time) (*models.PerformanceReportStats, error) {
	return tx.db.GetPerformanceStats(ctx, start, end)
}

func (tx *postgresqlTx) CountActiveUsers(ctx context.Context, period time.Duration) (int, error) {
	return tx.db.CountActiveUsers(ctx, period)
}

func (tx *postgresqlTx) CountTotalUsers(ctx context.Context) (int, error) {
	return tx.db.CountTotalUsers(ctx)
}

func (tx *postgresqlTx) CountActiveAPIKeys(ctx context.Context) (int, error) {
	return tx.db.CountActiveAPIKeys(ctx)
}

// Model Configurations delegation (v1.9.1+)
func (tx *postgresqlTx) CreateModelConfig(ctx context.Context, config *models.ModelConfig) error {
	return tx.db.CreateModelConfig(ctx, config)
}

func (tx *postgresqlTx) GetModelConfig(ctx context.Context, id string) (*models.ModelConfig, error) {
	return tx.db.GetModelConfig(ctx, id)
}

func (tx *postgresqlTx) GetModelConfigByScope(ctx context.Context, modelName, scope string, scopeID *string) (*models.ModelConfig, error) {
	return tx.db.GetModelConfigByScope(ctx, modelName, scope, scopeID)
}

func (tx *postgresqlTx) ListModelConfigs(ctx context.Context, scope string, scopeID *string) ([]*models.ModelConfig, error) {
	return tx.db.ListModelConfigs(ctx, scope, scopeID)
}

func (tx *postgresqlTx) UpdateModelConfig(ctx context.Context, config *models.ModelConfig) error {
	return tx.db.UpdateModelConfig(ctx, config)
}

func (tx *postgresqlTx) DeleteModelConfig(ctx context.Context, id string) error {
	return tx.db.DeleteModelConfig(ctx, id)
}

func (tx *postgresqlTx) GetEffectiveModelConfig(ctx context.Context, modelName, userID, tenantID string) (*models.ModelParameters, error) {
	return tx.db.GetEffectiveModelConfig(ctx, modelName, userID, tenantID)
}

// Ensure postgresqlTx implements storage.Tx interface
var _ storage.Tx = (*postgresqlTx)(nil)

// ========================================
// Transaction Methods (Delegation)
// ========================================
//
// Transaction methods are implemented in transactions.go
// They delegate to the parent DB's methods
// ========================================

