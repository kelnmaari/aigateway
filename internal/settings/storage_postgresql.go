// Package settings - PostgreSQL storage implementation
// Version: v3.0.9
package settings

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// PostgreSQLStorage implements Storage interface for PostgreSQL
type PostgreSQLStorage struct {
	db     *sqlx.DB
	logger *logrus.Logger
}

// NewPostgreSQLStorage creates a new PostgreSQL storage
func NewPostgreSQLStorage(db *sqlx.DB, logger *logrus.Logger) *PostgreSQLStorage {
	return &PostgreSQLStorage{
		db:     db,
		logger: logger,
	}
}

// GetSetting retrieves a setting by ID
func (s *PostgreSQLStorage) GetSetting(ctx context.Context, id string) (*Setting, error) {
	var setting Setting
	query := `SELECT * FROM settings WHERE id = $1`

	if err := s.db.GetContext(ctx, &setting, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("setting not found: %s", id)
		}
		return nil, err
	}

	return &setting, nil
}

// GetSettingsByCategory retrieves all settings in a category
func (s *PostgreSQLStorage) GetSettingsByCategory(ctx context.Context, category SettingCategory) ([]*Setting, error) {
	var settings []*Setting
	query := `SELECT * FROM settings WHERE category = $1 ORDER BY key`

	if err := s.db.SelectContext(ctx, &settings, query, category); err != nil {
		return nil, err
	}

	return settings, nil
}

// GetAllSettings retrieves all settings
func (s *PostgreSQLStorage) GetAllSettings(ctx context.Context) ([]*Setting, error) {
	var settings []*Setting
	query := `SELECT * FROM settings ORDER BY category, key`

	if err := s.db.SelectContext(ctx, &settings, query); err != nil {
		return nil, err
	}

	return settings, nil
}

// UpsertSetting creates or updates a setting
func (s *PostgreSQLStorage) UpsertSetting(ctx context.Context, setting *Setting) error {
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
			type = EXCLUDED.type,
			default_value = EXCLUDED.default_value,
			description = EXCLUDED.description,
			is_editable = EXCLUDED.is_editable,
			is_required = EXCLUDED.is_required,
			is_migrated = EXCLUDED.is_migrated,
			validation_rule = EXCLUDED.validation_rule,
			updated_at = EXCLUDED.updated_at,
			updated_by = EXCLUDED.updated_by
	`

	_, err := s.db.ExecContext(ctx, query,
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
		setting.ValidationRule,
		setting.UpdatedAt,
		setting.UpdatedBy,
	)

	return err
}

// UpdateSettingValue updates only the value field
func (s *PostgreSQLStorage) UpdateSettingValue(ctx context.Context, id, value, updatedBy string) error {
	query := `
		UPDATE settings 
		SET value = $1, updated_at = NOW(), updated_by = $2
		WHERE id = $3 AND is_editable = TRUE
	`

	result, err := s.db.ExecContext(ctx, query, value, updatedBy, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("setting not found or not editable: %s", id)
	}

	return nil
}

// DeleteSetting deletes a setting
func (s *PostgreSQLStorage) DeleteSetting(ctx context.Context, id string) error {
	query := `DELETE FROM settings WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// BulkUpsertSettings upserts multiple settings in a transaction
func (s *PostgreSQLStorage) BulkUpsertSettings(ctx context.Context, settings []*Setting) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

	for _, setting := range settings {
		if _, err := tx.ExecContext(ctx, query,
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
			setting.ValidationRule,
			setting.UpdatedAt,
			setting.UpdatedBy,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
