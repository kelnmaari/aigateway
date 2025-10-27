// Package sqlite provides SQLite implementation of API key-related database operations
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"aigateway/internal/models"
)

// ========================================
// API Keys CRUD Operations
// ========================================

// CreateAPIKey создает новый API ключ
func (s *SQLiteDB) CreateAPIKey(ctx context.Context, key *models.APIKey) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("name", key.Name).Debug("Creating API key")

	// Serialize JSON fields
	modelsJSON, err := json.Marshal(key.Models)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	permissionsJSON, err := json.Marshal(key.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	rateLimitsJSON, err := json.Marshal(key.RateLimits)
	if err != nil {
		return fmt.Errorf("failed to marshal rate_limits: %w", err)
	}

	usageJSON, err := json.Marshal(key.Usage)
	if err != nil {
		return fmt.Errorf("failed to marshal usage: %w", err)
	}

	var metadataJSON []byte
	if key.Metadata != nil {
		metadataJSON, err = json.Marshal(key.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO api_keys (
			id, name, description, key_hash,
			user_id, tenant_id, scope,
			models, permissions, rate_limits,
			status, created_at, updated_at,
			expires_at, last_used_at,
			revoked_at, revoked_reason,
			metadata, usage
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		key.ID,
		key.Name,
		key.Description,
		key.KeyHash,
		key.UserID,
		key.TenantID,
		key.Scope,
		modelsJSON,
		permissionsJSON,
		rateLimitsJSON,
		key.Status,
		key.CreatedAt,
		key.UpdatedAt,
		key.ExpiresAt,
		key.LastUsedAt,
		key.RevokedAt,
		key.RevokedReason,
		metadataJSON,
		usageJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert API key: %w", err)
	}

	s.logger.WithField("key_id", key.ID).Info("API key created successfully")
	return nil
}

// GetAPIKey получает API ключ по ID
func (s *SQLiteDB) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, description, key_hash,
			user_id, tenant_id, scope,
			models, permissions, rate_limits,
			status, created_at, updated_at,
			expires_at, last_used_at,
			revoked_at, revoked_reason,
			metadata, usage
		FROM api_keys
		WHERE id = ?
	`

	key, err := s.scanAPIKey(s.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return key, nil
}

// GetAPIKeyByHash получает API ключ по hash
func (s *SQLiteDB) GetAPIKeyByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, description, key_hash,
			user_id, tenant_id, scope,
			models, permissions, rate_limits,
			status, created_at, updated_at,
			expires_at, last_used_at,
			revoked_at, revoked_reason,
			metadata, usage
		FROM api_keys
		WHERE key_hash = ?
	`

	key, err := s.scanAPIKey(s.db.QueryRowContext(ctx, query, hash))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found by hash")
		}
		return nil, fmt.Errorf("failed to get API key by hash: %w", err)
	}

	return key, nil
}

// UpdateAPIKey обновляет API ключ
func (s *SQLiteDB) UpdateAPIKey(ctx context.Context, key *models.APIKey) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("key_id", key.ID).Debug("Updating API key")

	// Serialize JSON fields
	modelsJSON, err := json.Marshal(key.Models)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	permissionsJSON, err := json.Marshal(key.Permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	rateLimitsJSON, err := json.Marshal(key.RateLimits)
	if err != nil {
		return fmt.Errorf("failed to marshal rate_limits: %w", err)
	}

	usageJSON, err := json.Marshal(key.Usage)
	if err != nil {
		return fmt.Errorf("failed to marshal usage: %w", err)
	}

	var metadataJSON []byte
	if key.Metadata != nil {
		metadataJSON, err = json.Marshal(key.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		UPDATE api_keys SET
			name = ?,
			description = ?,
			user_id = ?,
			tenant_id = ?,
			scope = ?,
			models = ?,
			permissions = ?,
			rate_limits = ?,
			status = ?,
			updated_at = ?,
			expires_at = ?,
			last_used_at = ?,
			revoked_at = ?,
			revoked_reason = ?,
			metadata = ?,
			usage = ?
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		key.Name,
		key.Description,
		key.UserID,
		key.TenantID,
		key.Scope,
		modelsJSON,
		permissionsJSON,
		rateLimitsJSON,
		key.Status,
		key.UpdatedAt,
		key.ExpiresAt,
		key.LastUsedAt,
		key.RevokedAt,
		key.RevokedReason,
		metadataJSON,
		usageJSON,
		key.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found: %s", key.ID)
	}

	s.logger.WithField("key_id", key.ID).Info("API key updated successfully")
	return nil
}

// DeleteAPIKey удаляет API ключ
func (s *SQLiteDB) DeleteAPIKey(ctx context.Context, id string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("key_id", id).Debug("Deleting API key")

	query := `DELETE FROM api_keys WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found: %s", id)
	}

	s.logger.WithField("key_id", id).Info("API key deleted successfully")
	return nil
}

// RevokeAPIKey отзывает API ключ
func (s *SQLiteDB) RevokeAPIKey(ctx context.Context, id string, reason string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("key_id", id).Debug("Revoking API key")

	query := `
		UPDATE api_keys SET
			status = 'revoked',
			revoked_at = CURRENT_TIMESTAMP,
			revoked_reason = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query, reason, id)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found: %s", id)
	}

	s.logger.WithField("key_id", id).Info("API key revoked successfully")
	return nil
}

// EnableAPIKey активирует API ключ
func (s *SQLiteDB) EnableAPIKey(ctx context.Context, id string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("key_id", id).Debug("Enabling API key")

	query := `
		UPDATE api_keys SET
			status = 'active',
			revoked_at = NULL,
			revoked_reason = NULL,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to enable API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found: %s", id)
	}

	s.logger.WithField("key_id", id).Info("API key enabled successfully")
	return nil
}

// ListAPIKeys возвращает список всех API ключей (для админа)
func (s *SQLiteDB) ListAPIKeys(ctx context.Context) ([]*models.APIKey, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, description, key_hash,
			user_id, tenant_id, scope,
			models, permissions, rate_limits,
			status, created_at, updated_at,
			expires_at, last_used_at,
			revoked_at, revoked_reason,
			metadata, usage
		FROM api_keys
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []*models.APIKey
	for rows.Next() {
		key, err := s.scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	s.logger.WithField("count", len(keys)).Debug("Listed all API keys")
	return keys, nil
}

// ListPersonalAPIKeys возвращает список персональных API ключей пользователя
func (s *SQLiteDB) ListPersonalAPIKeys(ctx context.Context, userID string) ([]*models.APIKey, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, description, key_hash,
			user_id, tenant_id, scope,
			models, permissions, rate_limits,
			status, created_at, updated_at,
			expires_at, last_used_at,
			revoked_at, revoked_reason,
			metadata, usage
		FROM api_keys
		WHERE user_id = ? AND scope = 'personal'
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list personal API keys: %w", err)
	}
	defer rows.Close()

	var keys []*models.APIKey
	for rows.Next() {
		key, err := s.scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating personal API keys: %w", err)
	}

	s.logger.WithField("count", len(keys)).Debug("Listed personal API keys")
	return keys, nil
}

// ListTenantAPIKeys возвращает список API ключей tenant
func (s *SQLiteDB) ListTenantAPIKeys(ctx context.Context, tenantID string) ([]*models.APIKey, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, description, key_hash,
			user_id, tenant_id, scope,
			models, permissions, rate_limits,
			status, created_at, updated_at,
			expires_at, last_used_at,
			revoked_at, revoked_reason,
			metadata, usage
		FROM api_keys
		WHERE tenant_id = ? AND scope = 'tenant'
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant API keys: %w", err)
	}
	defer rows.Close()

	var keys []*models.APIKey
	for rows.Next() {
		key, err := s.scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenant API keys: %w", err)
	}

	s.logger.WithField("count", len(keys)).Debug("Listed tenant API keys")
	return keys, nil
}

// ========================================
// Helper Methods
// ========================================

// scanAPIKey сканирует строку БД в модель APIKey
func (s *SQLiteDB) scanAPIKey(row scanner) (*models.APIKey, error) {
	var key models.APIKey
	var modelsJSON []byte
	var permissionsJSON []byte
	var rateLimitsJSON []byte
	var usageJSON []byte
	var metadataJSON []byte

	var description sql.NullString
	var userID sql.NullString
	var tenantID sql.NullString
	var expiresAt sql.NullTime
	var lastUsedAt sql.NullTime
	var revokedAt sql.NullTime
	var revokedReason sql.NullString

	err := row.Scan(
		&key.ID,
		&key.Name,
		&description,
		&key.KeyHash,
		&userID,
		&tenantID,
		&key.Scope,
		&modelsJSON,
		&permissionsJSON,
		&rateLimitsJSON,
		&key.Status,
		&key.CreatedAt,
		&key.UpdatedAt,
		&expiresAt,
		&lastUsedAt,
		&revokedAt,
		&revokedReason,
		&metadataJSON,
		&usageJSON,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if description.Valid {
		key.Description = description.String
	}
	if userID.Valid {
		key.UserID = &userID.String
	}
	if tenantID.Valid {
		key.TenantID = &tenantID.String
	}
	if expiresAt.Valid {
		key.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		key.LastUsedAt = &lastUsedAt.Time
	}
	if revokedAt.Valid {
		key.RevokedAt = &revokedAt.Time
	}
	if revokedReason.Valid {
		str := revokedReason.String
		key.RevokedReason = str
	}

	// Deserialize JSON fields
	if err := json.Unmarshal(modelsJSON, &key.Models); err != nil {
		return nil, fmt.Errorf("failed to unmarshal models: %w", err)
	}

	if err := json.Unmarshal(permissionsJSON, &key.Permissions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal permissions: %w", err)
	}

	if err := json.Unmarshal(rateLimitsJSON, &key.RateLimits); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rate_limits: %w", err)
	}

	if err := json.Unmarshal(usageJSON, &key.Usage); err != nil {
		return nil, fmt.Errorf("failed to unmarshal usage: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &key.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &key, nil
}

