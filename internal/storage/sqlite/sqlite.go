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
