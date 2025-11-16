// Package settings - SQL storage adapter for storage.Database
// Version: v3.0.9
package settings

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"

	"aigateway/internal/storage"
)

// SQLStorageAdapter adapts storage.Database to settings.Storage interface
type SQLStorageAdapter struct {
	db     storage.Database
	logger *logrus.Logger
}

// NewSQLStorageAdapter creates a new SQL storage adapter
func NewSQLStorageAdapter(db storage.Database, logger *logrus.Logger) *SQLStorageAdapter {
	return &SQLStorageAdapter{
		db:     db,
		logger: logger,
	}
}

// GetSetting retrieves a setting by ID
func (s *SQLStorageAdapter) GetSetting(ctx context.Context, id string) (*Setting, error) {
	// TODO: Phase 2 - implement using storage.Database raw query
	return nil, fmt.Errorf("settings not yet migrated to database (Phase 1: Read-Only)")
}

// GetSettingsByCategory retrieves all settings in a category
func (s *SQLStorageAdapter) GetSettingsByCategory(ctx context.Context, category SettingCategory) ([]*Setting, error) {
	// TODO: Phase 2 - implement
	return []*Setting{}, nil
}

// GetAllSettings retrieves all settings
func (s *SQLStorageAdapter) GetAllSettings(ctx context.Context) ([]*Setting, error) {
	// Phase 1: Return empty (no settings in DB yet)
	// Phase 2: Will load from database
	return []*Setting{}, nil
}

// UpsertSetting creates or updates a setting
func (s *SQLStorageAdapter) UpsertSetting(ctx context.Context, setting *Setting) error {
	// TODO: Phase 2 - implement
	return fmt.Errorf("settings editing not available in Phase 1")
}

// UpdateSettingValue updates only the value field
func (s *SQLStorageAdapter) UpdateSettingValue(ctx context.Context, id, value, updatedBy string) error {
	// TODO: Phase 3 - implement
	return fmt.Errorf("settings editing not available in Phase 1")
}

// DeleteSetting deletes a setting
func (s *SQLStorageAdapter) DeleteSetting(ctx context.Context, id string) error {
	// TODO: Phase 3 - implement
	return fmt.Errorf("settings deletion not available in Phase 1")
}

// BulkUpsertSettings upserts multiple settings in a transaction
func (s *SQLStorageAdapter) BulkUpsertSettings(ctx context.Context, settings []*Setting) error {
	// Phase 2: Будет использоваться для seed из YAML
	
	// Get underlying SQL DB connection
	// This is a workaround until we refactor storage interface
	query := `
		INSERT INTO settings (
			id, category, key, value, type, default_value, 
			description, is_editable, is_required, is_migrated, 
			validation_rule, updated_at, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		ON CONFLICT (id) DO UPDATE SET
			value = EXCLUDED.value,
			default_value = EXCLUDED.default_value,
			description = EXCLUDED.description,
			updated_at = EXCLUDED.updated_at
		WHERE settings.is_migrated = FALSE
	`
	
	_ = query // Placeholder for Phase 2
	
	s.logger.Warn("BulkUpsertSettings called but not yet implemented (Phase 2)")
	return nil
}

// execQuery is a helper to execute raw SQL (Phase 2)
func (s *SQLStorageAdapter) execQuery(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// TODO: Phase 2 - use storage.Database connection
	// This requires adding RawExec method to storage.Database interface
	// or using type assertion to access underlying connection
	return nil, fmt.Errorf("raw SQL execution not available in Phase 1")
}

