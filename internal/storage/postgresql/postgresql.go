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

// GetDB returns underlying *sql.DB connection (for internal use)
func (p *PostgreSQLDB) GetDB() any {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.db
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

// Migration management methods delegation (REFACTOR-01)
func (tx *postgresqlTx) RollbackMigrations(ctx context.Context, targetVersion int) error {
	return tx.db.RollbackMigrations(ctx, targetVersion)
}

func (tx *postgresqlTx) ListMigrations(ctx context.Context) ([]storage.MigrationInfo, error) {
	return tx.db.ListMigrations(ctx)
}

func (tx *postgresqlTx) DestroyDatabase(ctx context.Context) error {
	return tx.db.DestroyDatabase(ctx)
}

// Ensure postgresqlTx implements storage.Tx interface
var _ storage.Tx = (*postgresqlTx)(nil)

// ========================================
// Migration Functions
// ========================================

// getCreateModelRegistryTablesMigration returns SQL for creating model registry tables (v5 migration)
func (p *PostgreSQLDB) getCreateModelRegistryTablesMigration() string {
	return `
-- Model Providers Table
-- Хранит конфигурацию для каждого model provider (Ollama, vLLM, OpenAI, etc)
CREATE TABLE IF NOT EXISTS model_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,              -- "ollama-local", "vllm-primary", "openai"
    provider_type TEXT NOT NULL,            -- "ollama", "vllm", "openai", "anthropic", "custom"
    base_url TEXT NOT NULL,                 -- "http://localhost:11434"
    api_key TEXT,                           -- Encrypted (для OpenAI, Anthropic)
    
    -- Configuration
    enabled BOOLEAN DEFAULT true,           -- Provider включен/выключен
    priority INTEGER DEFAULT 100,           -- Higher = preferred (для fallback)
    config JSONB DEFAULT '{}'::jsonb,       -- Provider-specific config
    
    -- Health tracking
    health_status TEXT DEFAULT 'unknown',   -- "healthy", "unhealthy", "unknown"
    last_health_check TIMESTAMP,
    error_message TEXT,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT valid_provider_type CHECK (provider_type IN ('ollama', 'vllm', 'openai', 'anthropic', 'custom'))
);

CREATE INDEX IF NOT EXISTS idx_model_providers_type ON model_providers(provider_type);
CREATE INDEX IF NOT EXISTS idx_model_providers_enabled ON model_providers(enabled);
CREATE INDEX IF NOT EXISTS idx_model_providers_priority ON model_providers(priority DESC);

-- Model Registry Table
-- Универсальный реестр всех моделей из всех providers
CREATE TABLE IF NOT EXISTS model_registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Model identification
    model_id TEXT UNIQUE NOT NULL,          -- "llama2:7b", "mistral-7b-instruct", "gpt-4"
    model_name TEXT NOT NULL,               -- Display name
    provider_id UUID NOT NULL,              -- FK to model_providers
    
    -- Capabilities (JSON array)
    capabilities JSONB DEFAULT '[]'::jsonb, -- ["chat", "embeddings", "vision", "function-calling"]
    parameters JSONB DEFAULT '{}'::jsonb,   -- Model-specific parameters (max_tokens, etc)
    
    -- Requirements
    requires_gpu BOOLEAN DEFAULT 0,
    min_vram_gb INTEGER,
    context_length INTEGER,
    
    -- Status
    status TEXT DEFAULT 'active',           -- "active", "inactive", "loading", "error"
    health_status TEXT DEFAULT 'unknown',   -- "healthy", "unhealthy", "unknown"
    last_health_check TIMESTAMP,
    
    -- Metadata
    description TEXT,
    tags TEXT[],                            -- Array of tags: {"opensource", "7b", "instruct"}
    
    -- Performance metrics (updated periodically)
    avg_latency_ms REAL,
    tokens_per_second REAL,
    total_requests BIGINT DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    -- Constraints
    FOREIGN KEY (provider_id) REFERENCES model_providers(id) ON DELETE CASCADE,
    CONSTRAINT valid_status CHECK (status IN ('active', 'inactive', 'loading', 'error')),
    CONSTRAINT valid_health CHECK (health_status IN ('healthy', 'unhealthy', 'unknown'))
);

CREATE INDEX IF NOT EXISTS idx_model_registry_provider ON model_registry(provider_id);
CREATE INDEX IF NOT EXISTS idx_model_registry_status ON model_registry(status);
CREATE INDEX IF NOT EXISTS idx_model_registry_health ON model_registry(health_status);
CREATE INDEX IF NOT EXISTS idx_model_registry_model_id ON model_registry(model_id);
CREATE INDEX IF NOT EXISTS idx_model_registry_capabilities ON model_registry USING GIN (capabilities);

-- Triggers для auto-update updated_at
CREATE OR REPLACE FUNCTION update_model_providers_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_model_providers_timestamp
BEFORE UPDATE ON model_providers
FOR EACH ROW
EXECUTE FUNCTION update_model_providers_timestamp();

CREATE OR REPLACE FUNCTION update_model_registry_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_model_registry_timestamp
BEFORE UPDATE ON model_registry
FOR EACH ROW
EXECUTE FUNCTION update_model_registry_timestamp();
	`
}

// ========================================
// Transaction Methods (Delegation)
// ========================================
//
// Transaction methods are implemented in transactions.go
// They delegate to the parent DB's methods
// ========================================
