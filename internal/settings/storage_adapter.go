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
	// Type assert to get underlying *sql.DB
	type dbGetter interface {
		GetDB() interface{}
	}
	
	dbg, ok := s.db.(dbGetter)
	if !ok {
		return nil, fmt.Errorf("database does not support GetDB() method")
	}
	
	db, ok := dbg.GetDB().(*sql.DB)
	if !ok || db == nil {
		return nil, fmt.Errorf("database connection is nil or wrong type")
	}
	
	query := `
		SELECT 
			id, category, key, value, type, default_value, 
			description, is_editable, is_required, is_migrated, 
			requires_restart, validation_rule, updated_at, updated_by
		FROM settings
		WHERE id = $1
	`
	
	setting := &Setting{}
	err := db.QueryRowContext(ctx, query, id).Scan(
		&setting.ID,
		&setting.Category,
		&setting.Key,
		&setting.Value,
		&setting.Type,
		&setting.DefaultValue,
		&setting.Description,
		&setting.IsEditable,
		&setting.IsRequired,
		&setting.IsMigrated,
		&setting.RequiresRestart,
		&setting.ValidationRule,
		&setting.UpdatedAt,
		&setting.UpdatedBy,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("setting not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query setting: %w", err)
	}
	
	return setting, nil
}

// GetSettingsByCategory retrieves all settings in a category
func (s *SQLStorageAdapter) GetSettingsByCategory(ctx context.Context, category SettingCategory) ([]*Setting, error) {
	// Type assert to get underlying *sql.DB
	type dbGetter interface {
		GetDB() interface{}
	}
	
	dbg, ok := s.db.(dbGetter)
	if !ok {
		return nil, fmt.Errorf("database does not support GetDB() method")
	}
	
	db, ok := dbg.GetDB().(*sql.DB)
	if !ok || db == nil {
		return nil, fmt.Errorf("database connection is nil or wrong type")
	}
	
	query := `
		SELECT 
			id, category, key, value, type, default_value, 
			description, is_editable, is_required, is_migrated, 
			requires_restart, validation_rule, updated_at, updated_by
		FROM settings
		WHERE category = $1
		ORDER BY key
	`
	
	rows, err := db.QueryContext(ctx, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings by category: %w", err)
	}
	defer rows.Close()
	
	var settings []*Setting
	for rows.Next() {
		setting := &Setting{}
		err := rows.Scan(
			&setting.ID,
			&setting.Category,
			&setting.Key,
			&setting.Value,
			&setting.Type,
			&setting.DefaultValue,
			&setting.Description,
			&setting.IsEditable,
			&setting.IsRequired,
			&setting.IsMigrated,
			&setting.RequiresRestart,
			&setting.ValidationRule,
			&setting.UpdatedAt,
			&setting.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, setting)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settings: %w", err)
	}
	
	return settings, nil
}

// GetAllSettings retrieves all settings
func (s *SQLStorageAdapter) GetAllSettings(ctx context.Context) ([]*Setting, error) {
	// Type assert to get underlying *sql.DB
	type dbGetter interface {
		GetDB() interface{}
	}
	
	dbg, ok := s.db.(dbGetter)
	if !ok {
		return nil, fmt.Errorf("database does not support GetDB() method")
	}
	
	db, ok := dbg.GetDB().(*sql.DB)
	if !ok || db == nil {
		return nil, fmt.Errorf("database connection is nil or wrong type")
	}
	
	query := `
		SELECT 
			id, category, key, value, type, default_value, 
			description, is_editable, is_required, is_migrated, 
			requires_restart, validation_rule, updated_at, updated_by
		FROM settings
		ORDER BY category, key
	`
	
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query settings: %w", err)
	}
	defer rows.Close()
	
	var settings []*Setting
	for rows.Next() {
		setting := &Setting{}
		err := rows.Scan(
			&setting.ID,
			&setting.Category,
			&setting.Key,
			&setting.Value,
			&setting.Type,
			&setting.DefaultValue,
			&setting.Description,
			&setting.IsEditable,
			&setting.IsRequired,
			&setting.IsMigrated,
			&setting.RequiresRestart,
			&setting.ValidationRule,
			&setting.UpdatedAt,
			&setting.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings = append(settings, setting)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating settings: %w", err)
	}
	
	return settings, nil
}

// UpsertSetting creates or updates a setting
func (s *SQLStorageAdapter) UpsertSetting(ctx context.Context, setting *Setting) error {
	// TODO: Phase 2 - implement
	return fmt.Errorf("settings editing not available in Phase 1")
}

// UpdateSettingValue updates only the value field
func (s *SQLStorageAdapter) UpdateSettingValue(ctx context.Context, id, value, updatedBy string) error {
	// Type assert to get underlying *sql.DB
	type dbGetter interface {
		GetDB() interface{}
	}
	
	dbg, ok := s.db.(dbGetter)
	if !ok {
		return fmt.Errorf("database does not support GetDB() method")
	}
	
	db, ok := dbg.GetDB().(*sql.DB)
	if !ok || db == nil {
		return fmt.Errorf("database connection is nil or wrong type")
	}
	
	query := `
		UPDATE settings
		SET 
			value = $1,
			updated_at = NOW(),
			updated_by = $2
		WHERE id = $3 AND is_editable = TRUE
	`
	
	result, err := db.ExecContext(ctx, query, value, updatedBy, id)
	if err != nil {
		return fmt.Errorf("failed to update setting value: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("setting not found or not editable: %s", id)
	}
	
	s.logger.WithFields(map[string]interface{}{
		"id":         id,
		"new_value":  value,
		"updated_by": updatedBy,
	}).Info("Setting value updated")
	
	return nil
}

// DeleteSetting deletes a setting
func (s *SQLStorageAdapter) DeleteSetting(ctx context.Context, id string) error {
	// Type assert to get underlying *sql.DB
	type dbGetter interface {
		GetDB() interface{}
	}
	
	dbg, ok := s.db.(dbGetter)
	if !ok {
		return fmt.Errorf("database does not support GetDB() method")
	}
	
	db, ok := dbg.GetDB().(*sql.DB)
	if !ok || db == nil {
		return fmt.Errorf("database connection is nil or wrong type")
	}
	
	query := `DELETE FROM settings WHERE id = $1`
	
	result, err := db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("setting not found: %s", id)
	}
	
	s.logger.WithField("id", id).Info("Setting deleted")
	return nil
}

// BulkUpsertSettings upserts multiple settings in a transaction
func (s *SQLStorageAdapter) BulkUpsertSettings(ctx context.Context, settings []*Setting) error {
	// Type assert to get underlying *sql.DB
	type dbGetter interface {
		GetDB() interface{}
	}
	
	dbg, ok := s.db.(dbGetter)
	if !ok {
		return fmt.Errorf("database does not support GetDB() method")
	}
	
	db, ok := dbg.GetDB().(*sql.DB)
	if !ok || db == nil {
		return fmt.Errorf("database connection is nil or wrong type")
	}
	
	// Start transaction
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	query := `
		INSERT INTO settings (
			id, category, key, value, type, default_value, 
			description, is_editable, is_required, is_migrated, 
			requires_restart, validation_rule, updated_at, updated_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		ON CONFLICT (id) DO UPDATE SET
			value = EXCLUDED.value,
			default_value = EXCLUDED.default_value,
			description = EXCLUDED.description,
			is_editable = EXCLUDED.is_editable,
			requires_restart = EXCLUDED.requires_restart,
			updated_at = EXCLUDED.updated_at
	`
	
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	// Insert all settings
	for _, setting := range settings {
		_, err := stmt.ExecContext(ctx,
			setting.ID,
			setting.Category,
			setting.Key,
			setting.Value,
			setting.Type,
			setting.DefaultValue,
			setting.Description,
			setting.IsEditable,
			setting.IsRequired,
			setting.IsMigrated,
			setting.RequiresRestart,
			setting.ValidationRule,
			setting.UpdatedAt,
			"system", // System user for initial seed
		)
		if err != nil {
			return fmt.Errorf("failed to insert setting %s: %w", setting.ID, err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	s.logger.WithField("count", len(settings)).Info("✅ Bulk upserted settings")
	return nil
}

// execQuery is a helper to execute raw SQL (Phase 2)
func (s *SQLStorageAdapter) execQuery(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	// TODO: Phase 2 - use storage.Database connection
	// This requires adding RawExec method to storage.Database interface
	// or using type assertion to access underlying connection
	return nil, fmt.Errorf("raw SQL execution not available in Phase 1")
}

