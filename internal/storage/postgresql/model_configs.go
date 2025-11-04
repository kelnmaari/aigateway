// Model Configurations CRUD implementation for SQLite
package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"aigateway/internal/models"

	"github.com/google/uuid"
)

// ========================================
// Model Configurations CRUD (v1.9.1+)
// ========================================

// CreateModelConfig создает новую конфигурацию модели
func (db *PostgreSQLDB) CreateModelConfig(ctx context.Context, config *models.ModelConfig) error {
	// Generate ID if not provided
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(config.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		INSERT INTO model_configs (id, model_name, scope, tenant_id, user_id, created_by, created_at, updated_at, parameters)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = db.db.ExecContext(ctx, query,
		config.ID,
		config.ModelName,
		config.Scope,
		config.TenantID,
		config.UserID,
		config.CreatedBy,
		config.CreatedAt,
		config.UpdatedAt,
		string(paramsJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to create model config: %w", err)
	}

	return nil
}

// GetModelConfig получает конфигурацию модели по ID
func (db *PostgreSQLDB) GetModelConfig(ctx context.Context, id string) (*models.ModelConfig, error) {
	query := `
		SELECT id, model_name, scope, tenant_id, user_id, created_by, created_at, updated_at, parameters
		FROM model_configs
		WHERE id = $1
	`

	config := &models.ModelConfig{}
	var paramsJSON string
	var tenantID, userID sql.NullString

	err := db.db.QueryRowContext(ctx, query, id).Scan(
		&config.ID,
		&config.ModelName,
		&config.Scope,
		&tenantID,
		&userID,
		&config.CreatedBy,
		&config.CreatedAt,
		&config.UpdatedAt,
		&paramsJSON,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("model config not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model config: %w", err)
	}

	// Set nullable fields
	if tenantID.Valid {
		config.TenantID = &tenantID.String
	}
	if userID.Valid {
		config.UserID = &userID.String
	}

	// Unmarshal parameters
	if err := json.Unmarshal([]byte(paramsJSON), &config.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	return config, nil
}

// GetModelConfigByScope получает конфигурацию модели по scope и ID
func (db *PostgreSQLDB) GetModelConfigByScope(ctx context.Context, modelName, scope string, scopeID *string) (*models.ModelConfig, error) {
	var query string
	var args []interface{}

	switch scope {
	case "global":
		query = `
			SELECT id, model_name, scope, tenant_id, user_id, created_by, created_at, updated_at, parameters
			FROM model_configs
			WHERE model_name = $1 AND scope = 'global'
		`
		args = []interface{}{modelName}

	case "tenant":
		if scopeID == nil {
			return nil, fmt.Errorf("tenant_id required for tenant scope")
		}
		query = `
			SELECT id, model_name, scope, tenant_id, user_id, created_by, created_at, updated_at, parameters
			FROM model_configs
			WHERE model_name = $1 AND scope = 'tenant' AND tenant_id = $2
		`
		args = []interface{}{modelName, *scopeID}

	case "user":
		if scopeID == nil {
			return nil, fmt.Errorf("user_id required for user scope")
		}
		query = `
			SELECT id, model_name, scope, tenant_id, user_id, created_by, created_at, updated_at, parameters
			FROM model_configs
			WHERE model_name = $1 AND scope = 'user' AND user_id = $2
		`
		args = []interface{}{modelName, *scopeID}

	default:
		return nil, fmt.Errorf("invalid scope: %s", scope)
	}

	config := &models.ModelConfig{}
	var paramsJSON string
	var tenantID, userID sql.NullString

	err := db.db.QueryRowContext(ctx, query, args...).Scan(
		&config.ID,
		&config.ModelName,
		&config.Scope,
		&tenantID,
		&userID,
		&config.CreatedBy,
		&config.CreatedAt,
		&config.UpdatedAt,
		&paramsJSON,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No config found (not an error)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model config: %w", err)
	}

	// Set nullable fields
	if tenantID.Valid {
		config.TenantID = &tenantID.String
	}
	if userID.Valid {
		config.UserID = &userID.String
	}

	// Unmarshal parameters
	if err := json.Unmarshal([]byte(paramsJSON), &config.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	return config, nil
}

// ListModelConfigs возвращает список конфигураций по фильтру
func (db *PostgreSQLDB) ListModelConfigs(ctx context.Context, scope string, scopeID *string) ([]*models.ModelConfig, error) {
	var query string
	var args []interface{}

	if scope == "" {
		// List all configs
		query = `
			SELECT id, model_name, scope, tenant_id, user_id, created_by, created_at, updated_at, parameters
			FROM model_configs
			ORDER BY created_at DESC
		`
	} else {
		switch scope {
		case "global":
			query = `
				SELECT id, model_name, scope, tenant_id, user_id, created_by, created_at, updated_at, parameters
				FROM model_configs
				WHERE scope = 'global'
				ORDER BY created_at DESC
			`

		case "tenant":
			if scopeID == nil {
				return nil, fmt.Errorf("tenant_id required for tenant scope")
			}
			query = `
				SELECT id, model_name, scope, tenant_id, user_id, created_by, created_at, updated_at, parameters
				FROM model_configs
				WHERE scope = 'tenant' AND tenant_id = $1
				ORDER BY created_at DESC
			`
			args = []interface{}{*scopeID}

		case "user":
			if scopeID == nil {
				return nil, fmt.Errorf("user_id required for user scope")
			}
			query = `
				SELECT id, model_name, scope, tenant_id, user_id, created_by, created_at, updated_at, parameters
				FROM model_configs
				WHERE scope = 'user' AND user_id = $1
				ORDER BY created_at DESC
			`
			args = []interface{}{*scopeID}

		default:
			return nil, fmt.Errorf("invalid scope: %s", scope)
		}
	}

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list model configs: %w", err)
	}
	defer rows.Close()

	configs := []*models.ModelConfig{}
	for rows.Next() {
		config := &models.ModelConfig{}
		var paramsJSON string
		var tenantID, userID sql.NullString

		err := rows.Scan(
			&config.ID,
			&config.ModelName,
			&config.Scope,
			&tenantID,
			&userID,
			&config.CreatedBy,
			&config.CreatedAt,
			&config.UpdatedAt,
			&paramsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan model config: %w", err)
		}

		// Set nullable fields
		if tenantID.Valid {
			config.TenantID = &tenantID.String
		}
		if userID.Valid {
			config.UserID = &userID.String
		}

		// Unmarshal parameters
		if err := json.Unmarshal([]byte(paramsJSON), &config.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// UpdateModelConfig обновляет конфигурацию модели
func (db *PostgreSQLDB) UpdateModelConfig(ctx context.Context, config *models.ModelConfig) error {
	config.UpdatedAt = time.Now()

	// Marshal parameters to JSON
	paramsJSON, err := json.Marshal(config.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		UPDATE model_configs
		SET model_name = $1, scope = $2, tenant_id = $3, user_id = $4, parameters = $5, updated_at = $6
		WHERE id = $7
	`

	result, err := db.db.ExecContext(ctx, query,
		config.ModelName,
		config.Scope,
		config.TenantID,
		config.UserID,
		string(paramsJSON),
		config.UpdatedAt,
		config.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update model config: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("model config not found")
	}

	return nil
}

// DeleteModelConfig удаляет конфигурацию модели
func (db *PostgreSQLDB) DeleteModelConfig(ctx context.Context, id string) error {
	query := `DELETE FROM model_configs WHERE id = $1`

	result, err := db.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete model config: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("model config not found")
	}

	return nil
}

// GetEffectiveModelConfig возвращает effective конфигурацию с приоритетом:
// user config > tenant config > global config > defaults
func (db *PostgreSQLDB) GetEffectiveModelConfig(ctx context.Context, modelName, userID, tenantID string) (*models.ModelParameters, error) {
	// Start with defaults
	effective := models.DefaultParameters()

	// Try global config
	globalConfig, err := db.GetModelConfigByScope(ctx, modelName, "global", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get global config: %w", err)
	}
	if globalConfig != nil {
		effective = *effective.MergeWith(&globalConfig.Parameters)
	}

	// Try tenant config
	if tenantID != "" {
		tenantConfig, err := db.GetModelConfigByScope(ctx, modelName, "tenant", &tenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to get tenant config: %w", err)
		}
		if tenantConfig != nil {
			effective = *effective.MergeWith(&tenantConfig.Parameters)
		}
	}

	// Try user config (highest priority)
	if userID != "" {
		userConfig, err := db.GetModelConfigByScope(ctx, modelName, "user", &userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user config: %w", err)
		}
		if userConfig != nil {
			effective = *effective.MergeWith(&userConfig.Parameters)
		}
	}

	return &effective, nil
}

// ========================================
// Transaction Support
// ========================================

// Delegate methods for postgresqlTx
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
