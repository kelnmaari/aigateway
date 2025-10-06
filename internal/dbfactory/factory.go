// Package dbfactory provides database factory for creating database instances
package dbfactory

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/storage"
	"ollama-openai-proxy/internal/storage/postgresql"
	"ollama-openai-proxy/internal/storage/sqlite"
)

// NewDatabase создает новый экземпляр базы данных на основе конфигурации
func NewDatabase(cfg *config.Config, logger *logrus.Logger) (storage.Database, error) {
	if logger == nil {
		logger = logrus.New()
	}

	switch cfg.Database.Type {
	case "sqlite":
		logger.WithFields(logrus.Fields{
			"type": "sqlite",
			"path": cfg.Database.SQLite.Path,
		}).Info("Initializing SQLite database")

		db, err := sqlite.New(cfg.Database.SQLite, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite database: %w", err)
		}

		return db, nil

	case "postgresql":
		logger.WithFields(logrus.Fields{
			"type":     "postgresql",
			"host":     cfg.Database.PostgreSQL.Host,
			"port":     cfg.Database.PostgreSQL.Port,
			"database": cfg.Database.PostgreSQL.Database,
		}).Info("Initializing PostgreSQL database")

		db, err := postgresql.New(cfg.Database.PostgreSQL, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create PostgreSQL database: %w", err)
		}

		return db, nil

	default:
		return nil, fmt.Errorf("unsupported database type: %s (supported: sqlite, postgresql)", cfg.Database.Type)
	}
}

// InitializeDatabase инициализирует подключение и выполняет миграции
func InitializeDatabase(ctx context.Context, db storage.Database, logger *logrus.Logger) error {
	if logger == nil {
		logger = logrus.New()
	}

	// Подключаемся к БД
	logger.Info("Connecting to database...")
	if err := db.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Проверяем подключение
	logger.Info("Testing database connection...")
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	logger.Info("Database connection established successfully")

	// Выполняем миграции
	logger.Info("Running database migrations...")
	if err := db.Migrate(ctx); err != nil {
		return fmt.Errorf("database migration failed: %w", err)
	}

	// Получаем текущую версию схемы
	version, err := db.GetMigrationVersion(ctx)
	if err != nil {
		logger.WithError(err).Warn("Failed to get migration version")
	} else {
		logger.WithField("schema_version", version).Info("Database migrations completed successfully")
	}

	return nil
}

// CloseDatabase закрывает подключение к БД с логированием
func CloseDatabase(db storage.Database, logger *logrus.Logger) error {
	if db == nil {
		return nil
	}

	if logger == nil {
		logger = logrus.New()
	}

	logger.Info("Closing database connection...")
	if err := db.Close(); err != nil {
		logger.WithError(err).Error("Error closing database connection")
		return fmt.Errorf("failed to close database: %w", err)
	}

	logger.Info("Database connection closed successfully")
	return nil
}
