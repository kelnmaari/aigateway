// Package config provides database configuration types
package config

import "time"

// DatabaseConfig конфигурация database abstraction layer (Version 1.3.0+)
type DatabaseConfig struct {
	// Type определяет тип БД: sqlite или postgresql
	Type DatabaseType `mapstructure:"type"`

	// SQLite specific configuration
	SQLite SQLiteConfig `mapstructure:"sqlite"`

	// PostgreSQL specific configuration
	PostgreSQL PostgreSQLConfig `mapstructure:"postgresql"`

	// Common settings
	AutoMigrate bool `mapstructure:"auto_migrate"` // Автоматически применять миграции при старте
	LogQueries  bool `mapstructure:"log_queries"`  // Логировать SQL запросы (debug mode)
}

// DatabaseType представляет тип БД
type DatabaseType string

const (
	DatabaseTypeSQLite     DatabaseType = "sqlite"
	DatabaseTypePostgreSQL DatabaseType = "postgresql"
)

// SQLiteConfig конфигурация для SQLite
type SQLiteConfig struct {
	// Path путь к файлу БД
	Path string `mapstructure:"path"`

	// CacheSize размер кеша в KB (default: 10000)
	CacheSize int `mapstructure:"cache_size"`

	// JournalMode режим журналирования: DELETE, TRUNCATE, PERSIST, MEMORY, WAL (default: WAL)
	JournalMode string `mapstructure:"journal_mode"`

	// BusyTimeout таймаут ожидания при блокировке БД в миллисекундах (default: 5000)
	BusyTimeout int `mapstructure:"busy_timeout"`
}

// PostgreSQLConfig конфигурация для PostgreSQL
type PostgreSQLConfig struct {
	// Connection parameters
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"sslmode"` // disable, require, verify-ca, verify-full

	// Connection pool settings
	MaxOpenConns    int           `mapstructure:"max_open_conns"`     // Максимум открытых соединений (default: 25)
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`     // Максимум idle соединений (default: 5)
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`  // Максимальное время жизни соединения (default: 5m)
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"` // Максимальное время idle (default: 5m)
}

// DefaultDatabaseConfig возвращает конфигурацию по умолчанию
func DefaultDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		Type: DatabaseTypeSQLite,
		SQLite: SQLiteConfig{
			Path:        "data/proxy.db",
			CacheSize:   10000,
			JournalMode: "WAL",
			BusyTimeout: 5000,
		},
		PostgreSQL: PostgreSQLConfig{
			Host:            "localhost",
			Port:            5432,
			Database:        "ollama_proxy",
			User:            "postgres",
			Password:        "",
			SSLMode:         "disable",
			MaxOpenConns:    25,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
		},
		AutoMigrate: true,
		LogQueries:  false,
	}
}
