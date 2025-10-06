# DB-01: Database Abstraction Layer

**Версия:** 1.0  
**Статус:** 📋 Запланирована  
**Приоритет:** HIGH  
**Оценка времени:** 8-10 часов  
**Зависимости:** Нет

---

## 📋 Описание

Абстрактный слой для работы с базами данных, поддерживающий SQLite (по умолчанию) и PostgreSQL (для production). Единый интерфейс для всех операций с данными, автоматические миграции, connection pooling.

---

## 🎯 Цели

1. **Database Abstraction**: Единый интерфейс для SQLite и PostgreSQL
2. **Migration System**: Автоматическое применение миграций при старте
3. **Type Safety**: Строгая типизация всех запросов
4. **Performance**: Connection pooling, prepared statements
5. **Easy Switch**: Переключение БД через конфиг без изменения кода

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────┐
│            Application Layer                        │
│  (Handlers, Services, Business Logic)               │
└─────────────────┬───────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────┐
│         Storage Interface (internal/storage)        │
│  ┌──────────────────────────────────────────────┐   │
│  │  type Database interface {                   │   │
│  │    CreateUser(...)  error                    │   │
│  │    GetUser(...)     (*User, error)           │   │
│  │    CreateAPIKey(...)  error                  │   │
│  │    ... all CRUD operations                   │   │
│  │  }                                            │   │
│  └──────────────────────────────────────────────┘   │
└─────────────────┬───────────────────────────────────┘
                  │
        ┌─────────┴─────────┐
        │                   │
        ▼                   ▼
┌─────────────┐     ┌──────────────┐
│  SQLite     │     │ PostgreSQL   │
│  Impl       │     │ Impl         │
└─────────────┘     └──────────────┘
```

---

## 📊 Interface Definition

```go
// internal/storage/database.go
package storage

import (
    "context"
    "time"
    "ollama-openai-proxy/internal/models"
)

type Database interface {
    // Connection Management
    Connect(ctx context.Context) error
    Close() error
    Ping(ctx context.Context) error
    
    // Migrations
    Migrate(ctx context.Context) error
    GetMigrationVersion(ctx context.Context) (int, error)
    
    // Transaction Support
    BeginTx(ctx context.Context) (Tx, error)
    
    // Users
    CreateUser(ctx context.Context, user *models.User) error
    GetUser(ctx context.Context, id string) (*models.User, error)
    GetUserByUsername(ctx context.Context, username string) (*models.User, error)
    GetUserByEmail(ctx context.Context, email string) (*models.User, error)
    UpdateUser(ctx context.Context, user *models.User) error
    DeleteUser(ctx context.Context, id string) error
    ListUsers(ctx context.Context, filters UserFilters) ([]*models.User, error)
    
    // Tenants
    CreateTenant(ctx context.Context, tenant *models.Tenant) error
    GetTenant(ctx context.Context, id string) (*models.Tenant, error)
    GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error)
    UpdateTenant(ctx context.Context, tenant *models.Tenant) error
    DeleteTenant(ctx context.Context, id string) error
    ListUserTenants(ctx context.Context, userID string) ([]*models.Tenant, error)
    
    // Tenant Members
    AddTenantMember(ctx context.Context, member *models.TenantMember) error
    GetTenantMember(ctx context.Context, tenantID, userID string) (*models.TenantMember, error)
    UpdateTenantMember(ctx context.Context, member *models.TenantMember) error
    RemoveTenantMember(ctx context.Context, tenantID, userID string) error
    ListTenantMembers(ctx context.Context, tenantID string) ([]*models.TenantMember, error)
    
    // API Keys
    CreateAPIKey(ctx context.Context, key *models.APIKey) error
    GetAPIKey(ctx context.Context, id string) (*models.APIKey, error)
    GetAPIKeyByHash(ctx context.Context, hash string) (*models.APIKey, error)
    UpdateAPIKey(ctx context.Context, key *models.APIKey) error
    DeleteAPIKey(ctx context.Context, id string) error
    RevokeAPIKey(ctx context.Context, id string, reason string) error
    EnableAPIKey(ctx context.Context, id string) error
    ListAPIKeys(ctx context.Context, filters APIKeyFilters) ([]*models.APIKey, error)
    ListPersonalAPIKeys(ctx context.Context, userID string) ([]*models.APIKey, error)
    ListTenantAPIKeys(ctx context.Context, tenantID string) ([]*models.APIKey, error)
    
    // Conversations
    CreateConversation(ctx context.Context, conv *models.Conversation) error
    GetConversation(ctx context.Context, id string) (*models.Conversation, error)
    UpdateConversation(ctx context.Context, conv *models.Conversation) error
    DeleteConversation(ctx context.Context, id string) error
    ListUserConversations(ctx context.Context, userID string, filters ConversationFilters) ([]*models.Conversation, error)
    
    // Messages
    CreateMessage(ctx context.Context, msg *models.Message) error
    GetMessage(ctx context.Context, id string) (*models.Message, error)
    ListConversationMessages(ctx context.Context, convID string) ([]*models.Message, error)
    DeleteConversationMessages(ctx context.Context, convID string) error
    
    // Usage Statistics
    RecordAPIUsage(ctx context.Context, usage *models.APIUsage) error
    GetUserUsageStats(ctx context.Context, userID string, period time.Duration) (*models.UsageStats, error)
    GetTenantUsageStats(ctx context.Context, tenantID string, period time.Duration) (*models.UsageStats, error)
}

// Transaction interface
type Tx interface {
    Commit() error
    Rollback() error
    Database  // Все методы Database доступны в транзакции
}
```

---

## 🔧 Configuration

```yaml
# configs/dev.yaml
database:
  type: sqlite  # or postgresql
  
  # SQLite specific
  sqlite:
    path: data/proxy.db
    cache_size: 10000
    journal_mode: WAL
    busy_timeout: 5000
    
  # PostgreSQL specific
  postgresql:
    host: localhost
    port: 5432
    database: ollama_proxy
    user: postgres
    password: secret
    sslmode: disable
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 5m
    
  # Common settings
  auto_migrate: true
  log_queries: false  # Debug mode
```

```go
// internal/config/database.go
type DatabaseConfig struct {
    Type       DatabaseType       `mapstructure:"type"`
    SQLite     SQLiteConfig       `mapstructure:"sqlite"`
    PostgreSQL PostgreSQLConfig   `mapstructure:"postgresql"`
    AutoMigrate bool              `mapstructure:"auto_migrate"`
    LogQueries  bool              `mapstructure:"log_queries"`
}

type DatabaseType string
const (
    DatabaseTypeSQLite     DatabaseType = "sqlite"
    DatabaseTypePostgreSQL DatabaseType = "postgresql"
)

type SQLiteConfig struct {
    Path         string `mapstructure:"path"`
    CacheSize    int    `mapstructure:"cache_size"`
    JournalMode  string `mapstructure:"journal_mode"`
    BusyTimeout  int    `mapstructure:"busy_timeout"`
}

type PostgreSQLConfig struct {
    Host            string        `mapstructure:"host"`
    Port            int           `mapstructure:"port"`
    Database        string        `mapstructure:"database"`
    User            string        `mapstructure:"user"`
    Password        string        `mapstructure:"password"`
    SSLMode         string        `mapstructure:"sslmode"`
    MaxOpenConns    int           `mapstructure:"max_open_conns"`
    MaxIdleConns    int           `mapstructure:"max_idle_conns"`
    ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}
```

---

## 💾 SQLite Implementation

```go
// internal/storage/sqlite/sqlite.go
package sqlite

import (
    "context"
    "database/sql"
    "fmt"
    
    _ "modernc.org/sqlite"  // Pure Go SQLite
    "ollama-openai-proxy/internal/config"
    "ollama-openai-proxy/internal/storage"
)

type SQLiteDB struct {
    db     *sql.DB
    config config.SQLiteConfig
}

func NewSQLiteDB(cfg config.SQLiteConfig) (*SQLiteDB, error) {
    return &SQLiteDB{
        config: cfg,
    }, nil
}

func (s *SQLiteDB) Connect(ctx context.Context) error {
    dsn := fmt.Sprintf("file:%s?cache=shared&mode=rwc", s.config.Path)
    
    // SQLite pragmas
    dsn += fmt.Sprintf("&_journal_mode=%s", s.config.JournalMode)
    dsn += fmt.Sprintf("&_busy_timeout=%d", s.config.BusyTimeout)
    dsn += fmt.Sprintf("&_cache_size=%d", s.config.CacheSize)
    dsn += "&_foreign_keys=1"
    dsn += "&_synchronous=NORMAL"
    
    db, err := sql.Open("sqlite", dsn)
    if err != nil {
        return fmt.Errorf("open sqlite: %w", err)
    }
    
    // Connection pool settings (SQLite рекомендует 1 write connection)
    db.SetMaxOpenConns(1)
    db.SetMaxIdleConns(1)
    
    if err := db.PingContext(ctx); err != nil {
        return fmt.Errorf("ping sqlite: %w", err)
    }
    
    s.db = db
    return nil
}

func (s *SQLiteDB) CreateUser(ctx context.Context, user *models.User) error {
    query := `
        INSERT INTO users (
            id, username, email, password_hash, display_name, avatar,
            preferences, status, created_at, updated_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
    
    _, err := s.db.ExecContext(ctx, query,
        user.ID, user.Username, user.Email, user.PasswordHash,
        user.DisplayName, user.Avatar, user.Preferences, user.Status,
        user.CreatedAt, user.UpdatedAt,
    )
    
    return err
}

// ... остальные методы
```

---

## 🐘 PostgreSQL Implementation

```go
// internal/storage/postgres/postgres.go
package postgres

import (
    "context"
    "database/sql"
    "fmt"
    
    _ "github.com/lib/pq"
    "ollama-openai-proxy/internal/config"
    "ollama-openai-proxy/internal/storage"
)

type PostgresDB struct {
    db     *sql.DB
    config config.PostgreSQLConfig
}

func NewPostgresDB(cfg config.PostgreSQLConfig) (*PostgresDB, error) {
    return &PostgresDB{
        config: cfg,
    }, nil
}

func (p *PostgresDB) Connect(ctx context.Context) error {
    dsn := fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
        p.config.Host, p.config.Port, p.config.User,
        p.config.Password, p.config.Database, p.config.SSLMode,
    )
    
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return fmt.Errorf("open postgres: %w", err)
    }
    
    // Connection pool settings
    db.SetMaxOpenConns(p.config.MaxOpenConns)
    db.SetMaxIdleConns(p.config.MaxIdleConns)
    db.SetConnMaxLifetime(p.config.ConnMaxLifetime)
    
    if err := db.PingContext(ctx); err != nil {
        return fmt.Errorf("ping postgres: %w", err)
    }
    
    p.db = db
    return nil
}

func (p *PostgresDB) CreateUser(ctx context.Context, user *models.User) error {
    query := `
        INSERT INTO users (
            id, username, email, password_hash, display_name, avatar,
            preferences, status, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `
    
    _, err := p.db.ExecContext(ctx, query,
        user.ID, user.Username, user.Email, user.PasswordHash,
        user.DisplayName, user.Avatar, user.Preferences, user.Status,
        user.CreatedAt, user.UpdatedAt,
    )
    
    return err
}

// ... остальные методы
```

---

## 🔄 Migration System

```go
// internal/storage/migrations/migrations.go
package migrations

import (
    "context"
    "database/sql"
    "fmt"
)

type Migration struct {
    Version int
    Name    string
    Up      string
    Down    string
}

var AllMigrations = []Migration{
    {
        Version: 1,
        Name:    "initial_schema",
        Up: `
            CREATE TABLE IF NOT EXISTS users (
                id TEXT PRIMARY KEY,
                username TEXT UNIQUE NOT NULL,
                email TEXT UNIQUE NOT NULL,
                password_hash TEXT NOT NULL,
                display_name TEXT,
                avatar TEXT,
                preferences TEXT DEFAULT '{}',
                status TEXT DEFAULT 'active',
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                last_login_at DATETIME
            );
            
            CREATE INDEX idx_users_username ON users(username);
            CREATE INDEX idx_users_email ON users(email);
            
            CREATE TABLE IF NOT EXISTS tenants (
                id TEXT PRIMARY KEY,
                name TEXT NOT NULL,
                slug TEXT UNIQUE NOT NULL,
                description TEXT,
                owner_id TEXT NOT NULL,
                type TEXT DEFAULT 'organization',
                settings TEXT DEFAULT '{}',
                status TEXT DEFAULT 'active',
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
            );
            
            CREATE INDEX idx_tenants_slug ON tenants(slug);
            CREATE INDEX idx_tenants_owner_id ON tenants(owner_id);
            
            CREATE TABLE IF NOT EXISTS tenant_members (
                id TEXT PRIMARY KEY,
                tenant_id TEXT NOT NULL,
                user_id TEXT NOT NULL,
                role TEXT NOT NULL,
                permissions TEXT DEFAULT '[]',
                invited_by TEXT,
                joined_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
                FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
                FOREIGN KEY (invited_by) REFERENCES users(id),
                UNIQUE(tenant_id, user_id)
            );
            
            CREATE INDEX idx_tenant_members_tenant_id ON tenant_members(tenant_id);
            CREATE INDEX idx_tenant_members_user_id ON tenant_members(user_id);
            
            CREATE TABLE IF NOT EXISTS migrations (
                version INTEGER PRIMARY KEY,
                name TEXT NOT NULL,
                applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
            );
        `,
        Down: `
            DROP TABLE IF EXISTS tenant_members;
            DROP TABLE IF EXISTS tenants;
            DROP TABLE IF EXISTS users;
            DROP TABLE IF EXISTS migrations;
        `,
    },
    {
        Version: 2,
        Name:    "api_keys_scoping",
        Up: `
            ALTER TABLE api_keys ADD COLUMN type TEXT DEFAULT 'tenant';
            ALTER TABLE api_keys ADD COLUMN owner_id TEXT;
            ALTER TABLE api_keys ADD COLUMN tenant_id TEXT;
            
            CREATE INDEX idx_api_keys_owner_id ON api_keys(owner_id);
            CREATE INDEX idx_api_keys_tenant_id ON api_keys(tenant_id);
        `,
        Down: `
            DROP INDEX IF EXISTS idx_api_keys_owner_id;
            DROP INDEX IF EXISTS idx_api_keys_tenant_id;
            -- ALTER TABLE DROP COLUMN не поддерживается в SQLite напрямую
        `,
    },
    {
        Version: 3,
        Name:    "conversations_and_messages",
        Up: `
            CREATE TABLE IF NOT EXISTS conversations (
                id TEXT PRIMARY KEY,
                user_id TEXT NOT NULL,
                tenant_id TEXT,
                title TEXT NOT NULL,
                model TEXT NOT NULL,
                system_msg TEXT,
                settings TEXT DEFAULT '{}',
                is_favorite BOOLEAN DEFAULT FALSE,
                is_archived BOOLEAN DEFAULT FALSE,
                tokens_used INTEGER DEFAULT 0,
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
                FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
            );
            
            CREATE INDEX idx_conversations_user_id ON conversations(user_id);
            CREATE INDEX idx_conversations_tenant_id ON conversations(tenant_id);
            
            CREATE TABLE IF NOT EXISTS messages (
                id TEXT PRIMARY KEY,
                conversation_id TEXT NOT NULL,
                role TEXT NOT NULL,
                content TEXT NOT NULL,
                tokens_used INTEGER DEFAULT 0,
                model TEXT,
                metadata TEXT,
                created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
                FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
            );
            
            CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
        `,
        Down: `
            DROP TABLE IF EXISTS messages;
            DROP TABLE IF EXISTS conversations;
        `,
    },
}

func ApplyMigrations(ctx context.Context, db *sql.DB) error {
    // Create migrations table if not exists
    _, err := db.ExecContext(ctx, `
        CREATE TABLE IF NOT EXISTS migrations (
            version INTEGER PRIMARY KEY,
            name TEXT NOT NULL,
            applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        return fmt.Errorf("create migrations table: %w", err)
    }
    
    // Get current version
    var currentVersion int
    err = db.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&currentVersion)
    if err != nil {
        return fmt.Errorf("get current version: %w", err)
    }
    
    // Apply pending migrations
    for _, migration := range AllMigrations {
        if migration.Version <= currentVersion {
            continue
        }
        
        // Execute migration in transaction
        tx, err := db.BeginTx(ctx, nil)
        if err != nil {
            return fmt.Errorf("begin tx: %w", err)
        }
        
        if _, err := tx.ExecContext(ctx, migration.Up); err != nil {
            tx.Rollback()
            return fmt.Errorf("apply migration %d: %w", migration.Version, err)
        }
        
        // Record migration
        if _, err := tx.ExecContext(ctx, 
            "INSERT INTO migrations (version, name) VALUES (?, ?)",
            migration.Version, migration.Name,
        ); err != nil {
            tx.Rollback()
            return fmt.Errorf("record migration %d: %w", migration.Version, err)
        }
        
        if err := tx.Commit(); err != nil {
            return fmt.Errorf("commit migration %d: %w", migration.Version, err)
        }
        
        fmt.Printf("✅ Applied migration %d: %s\n", migration.Version, migration.Name)
    }
    
    return nil
}
```

---

## 🏭 Factory Pattern

```go
// internal/storage/factory.go
package storage

import (
    "context"
    "fmt"
    
    "ollama-openai-proxy/internal/config"
    "ollama-openai-proxy/internal/storage/sqlite"
    "ollama-openai-proxy/internal/storage/postgres"
)

func NewDatabase(cfg config.DatabaseConfig) (Database, error) {
    var db Database
    var err error
    
    switch cfg.Type {
    case config.DatabaseTypeSQLite:
        db, err = sqlite.NewSQLiteDB(cfg.SQLite)
    case config.DatabaseTypePostgreSQL:
        db, err = postgres.NewPostgresDB(cfg.PostgreSQL)
    default:
        return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
    }
    
    if err != nil {
        return nil, fmt.Errorf("create database: %w", err)
    }
    
    // Connect
    ctx := context.Background()
    if err := db.Connect(ctx); err != nil {
        return nil, fmt.Errorf("connect database: %w", err)
    }
    
    // Auto-migrate
    if cfg.AutoMigrate {
        if err := db.Migrate(ctx); err != nil {
            return nil, fmt.Errorf("auto-migrate: %w", err)
        }
    }
    
    return db, nil
}
```

---

## 🔄 Migration from JSON Storage

```go
// cmd/migrate/main.go
func main() {
    // 1. Load old JSON data
    oldKeys := loadAPIKeysFromJSON("data/api_keys.json")
    
    // 2. Connect to new database
    db, err := storage.NewDatabase(cfg.Database)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // 3. Create default admin user
    adminUser := &models.User{
        ID:       uuid.New().String(),
        Username: "admin",
        Email:    "admin@localhost",
        // ... load from env or prompt
    }
    db.CreateUser(ctx, adminUser)
    
    // 4. Create default tenant
    defaultTenant := &models.Tenant{
        ID:      uuid.New().String(),
        Name:    "Default Organization",
        Slug:    "default",
        OwnerID: adminUser.ID,
        Type:    models.TenantTypeOrganization,
    }
    db.CreateTenant(ctx, defaultTenant)
    
    // 5. Migrate API keys
    for _, oldKey := range oldKeys {
        newKey := &models.APIKey{
            // ... копируем данные
            Type:     models.APIKeyTypeTenant,
            TenantID: &defaultTenant.ID,
        }
        db.CreateAPIKey(ctx, newKey)
    }
    
    // 6. Backup old JSON
    os.Rename("data/api_keys.json", "data/api_keys.json.backup")
    
    fmt.Println("✅ Migration completed successfully!")
}
```

---

## 🎯 Success Criteria

- ✅ Единый интерфейс для SQLite и PostgreSQL
- ✅ Переключение через конфиг без изменения кода
- ✅ Автоматические миграции работают
- ✅ Connection pooling настроен правильно
- ✅ Все CRUD операции протестированы
- ✅ Migration tool из JSON в DB работает
- ✅ Rollback миграций работает
- ✅ 100% test coverage для database layer

---

**Автор:** AI Assistant  
**Дата создания:** 2025-10-05  
**Последнее обновление:** 2025-10-05
