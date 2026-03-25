// Package sqlite provides SQLite implementation of API key-related database operations
package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// ========================================
// API Keys CRUD Operations
// ========================================

// CreateAPIKey создает новый API ключ
func (db *PostgreSQLDB) CreateAPIKey(ctx context.Context, key *models.APIKey) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	db.logger.WithField("name", key.Name).Debug("Creating API key")

	// Serialize JSON fields - ensure empty arrays/objects instead of nil
	models := key.Models
	if models == nil {
		models = []string{}
	}
	modelsJSON, err := json.Marshal(models)
	if err != nil {
		return fmt.Errorf("failed to marshal models: %w", err)
	}

	permissions := key.Permissions
	if permissions == nil {
		permissions = []string{}
	}
	permissionsJSON, err := json.Marshal(permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	// RateLimits is a struct, never nil
	rateLimitsJSON, err := json.Marshal(key.RateLimits)
	if err != nil {
		return fmt.Errorf("failed to marshal rate_limits: %w", err)
	}

	// Usage is a struct, never nil
	usageJSON, err := json.Marshal(key.Usage)
	if err != nil {
		return fmt.Errorf("failed to marshal usage: %w", err)
	}

	var metadataJSON []byte
	if key.Metadata != nil && len(key.Metadata) > 0 {
		metadataJSON, err = json.Marshal(key.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	query := `
		INSERT INTO api_keys (
			id, name, description, key_hash, key_prefix,
			user_id, tenant_id, scope,
			models, permissions, rate_limits,
			status, created_at, updated_at,
			expires_at, last_used_at,
			revoked_at, revoked_reason,
			metadata, usage,
			device_name, device_os, device_hostname, device_version, device_fingerprint,
			last_seen_at, auto_expire_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27)
	`

	_, err = db.db.ExecContext(ctx, query,
		key.ID,
		key.Name,
		key.Description,
		key.KeyHash,
		key.KeyPrefix,
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
		key.DeviceName,
		key.DeviceOS,
		key.DeviceHostname,
		key.DeviceVersion,
		key.DeviceFingerprint,
		key.LastSeenAt,
		key.AutoExpireAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert API key: %w", err)
	}

	db.logger.WithField("key_id", key.ID).Info("API key created successfully")
	return nil
}

// GetAPIKey получает API ключ по ID
func (db *PostgreSQLDB) GetAPIKey(ctx context.Context, id string) (*models.APIKey, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, description, key_hash, key_prefix,
			user_id, tenant_id, scope,
			models, permissions, rate_limits,
			status, created_at, updated_at,
			expires_at, last_used_at,
			revoked_at, revoked_reason,
			metadata, usage
		FROM api_keys
		WHERE id = $1
	`

	key, err := db.scanAPIKey(db.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return key, nil
}

// GetAPIKeyByHash получает API ключ по hash
func (db *PostgreSQLDB) GetAPIKeyByHash(ctx context.Context, hash string) (*models.APIKey, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, description, key_hash, key_prefix,
			user_id, tenant_id, scope,
			models, permissions, rate_limits,
			status, created_at, updated_at,
			expires_at, last_used_at,
			revoked_at, revoked_reason,
			metadata, usage
		FROM api_keys
		WHERE key_hash = $1
	`

	key, err := db.scanAPIKey(db.db.QueryRowContext(ctx, query, hash))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found by hash")
		}
		return nil, fmt.Errorf("failed to get API key by hash: %w", err)
	}

	return key, nil
}

// UpdateAPIKey обновляет API ключ
func (db *PostgreSQLDB) UpdateAPIKey(ctx context.Context, key *models.APIKey) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	db.logger.WithField("key_id", key.ID).Debug("Updating API key")

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
			name = $1,
			description = $2,
			user_id = $3,
			tenant_id = $4,
			scope = $5,
			models = $6,
			permissions = $7,
			rate_limits = $8,
			status = $9,
			updated_at = $10,
			expires_at = $11,
			last_used_at = $12,
			revoked_at = $13,
			revoked_reason = $14,
			metadata = $15,
			usage = $16
		WHERE id = $17
	`

	result, err := db.db.ExecContext(ctx, query,
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

	db.logger.WithField("key_id", key.ID).Info("API key updated successfully")
	return nil
}

// DeleteAPIKey удаляет API ключ
func (db *PostgreSQLDB) DeleteAPIKey(ctx context.Context, id string) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	db.logger.WithField("key_id", id).Debug("Deleting API key")

	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := db.db.ExecContext(ctx, query, id)
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

	db.logger.WithField("key_id", id).Info("API key deleted successfully")
	return nil
}

// RevokeAPIKey отзывает API ключ
func (db *PostgreSQLDB) RevokeAPIKey(ctx context.Context, id string, reason string) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	db.logger.WithField("key_id", id).Debug("Revoking API key")

	query := `
		UPDATE api_keys SET
			status = 'revoked',
			revoked_at = NOW(),
			revoked_reason = $1,
			updated_at = NOW()
		WHERE id = $2
	`

	result, err := db.db.ExecContext(ctx, query, reason, id)
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

	db.logger.WithField("key_id", id).Info("API key revoked successfully")
	return nil
}

// EnableAPIKey активирует API ключ
func (db *PostgreSQLDB) EnableAPIKey(ctx context.Context, id string) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	db.logger.WithField("key_id", id).Debug("Enabling API key")

	query := `
		UPDATE api_keys SET
			status = 'active',
			revoked_at = NULL,
			revoked_reason = NULL,
			updated_at = NOW()
		WHERE id = $1
	`

	result, err := db.db.ExecContext(ctx, query, id)
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

	db.logger.WithField("key_id", id).Info("API key enabled successfully")
	return nil
}

// ListAPIKeys возвращает список всех API ключей (для админа)
func (db *PostgreSQLDB) ListAPIKeys(ctx context.Context) ([]*models.APIKey, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, description, key_hash, key_prefix,
			user_id, tenant_id, scope,
			models, permissions, rate_limits,
			status, created_at, updated_at,
			expires_at, last_used_at,
			revoked_at, revoked_reason,
			metadata, usage
		FROM api_keys
		ORDER BY created_at DESC
	`

	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []*models.APIKey
	for rows.Next() {
		key, err := db.scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	db.logger.WithField("count", len(keys)).Debug("Listed all API keys")
	return keys, nil
}

// ListPersonalAPIKeys возвращает список персональных API ключей пользователя
func (db *PostgreSQLDB) ListPersonalAPIKeys(ctx context.Context, userID string) ([]*models.APIKey, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, description, key_hash, key_prefix,
			user_id, tenant_id, scope,
			models, permissions, rate_limits,
			status, created_at, updated_at,
			expires_at, last_used_at,
			revoked_at, revoked_reason,
			metadata, usage
		FROM api_keys
		WHERE user_id = $1 AND scope = 'personal'
		ORDER BY created_at DESC
	`

	rows, err := db.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list personal API keys: %w", err)
	}
	defer rows.Close()

	var keys []*models.APIKey
	for rows.Next() {
		key, err := db.scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating personal API keys: %w", err)
	}

	db.logger.WithField("count", len(keys)).Debug("Listed personal API keys")
	return keys, nil
}

// ListTenantAPIKeys возвращает список API ключей tenant
func (db *PostgreSQLDB) ListTenantAPIKeys(ctx context.Context, tenantID string) ([]*models.APIKey, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, description, key_hash, key_prefix,
			user_id, tenant_id, scope,
			models, permissions, rate_limits,
			status, created_at, updated_at,
			expires_at, last_used_at,
			revoked_at, revoked_reason,
			metadata, usage
		FROM api_keys
		WHERE tenant_id = $1 AND scope = 'tenant'
		ORDER BY created_at DESC
	`

	rows, err := db.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant API keys: %w", err)
	}
	defer rows.Close()

	var keys []*models.APIKey
	for rows.Next() {
		key, err := db.scanAPIKey(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenant API keys: %w", err)
	}

	db.logger.WithField("count", len(keys)).Debug("Listed tenant API keys")
	return keys, nil
}

// ========================================
// Helper Methods
// ========================================

// scanAPIKey сканирует строку БД в модель APIKey
func (db *PostgreSQLDB) scanAPIKey(row scanner) (*models.APIKey, error) {
	var key models.APIKey
	var modelsJSON []byte
	var permissionsJSON []byte
	var rateLimitsJSON []byte
	var usageJSON []byte
	var metadataJSON []byte

	var description sql.NullString
	var keyPrefix sql.NullString
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
		&keyPrefix,
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
	if keyPrefix.Valid {
		key.KeyPrefix = keyPrefix.String
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

// FindAPIKeyByDeviceFingerprint finds API key by device fingerprint for specific user (Version 2.4.0+)
func (db *PostgreSQLDB) FindAPIKeyByDeviceFingerprint(ctx context.Context, userID, fingerprint string) (*models.APIKey, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, description, key_hash, user_id, tenant_id, scope,
			models, permissions, rate_limits, status,
			created_at, updated_at, expires_at, last_used_at,
			revoked_at, revoked_reason, metadata, usage,
			device_name, device_os, device_hostname, device_version, device_fingerprint,
			last_seen_at, auto_expire_at
		FROM api_keys 
		WHERE user_id = $1 AND device_fingerprint = $2 AND status = 'active'
		LIMIT 1
	`

	var key models.APIKey
	var description, userIDNullable, tenantID sql.NullString
	var expiresAt, lastUsedAt, revokedAt sql.NullTime
	var revokedReason sql.NullString
	var modelsJSON, permissionsJSON, rateLimitsJSON, metadataJSON, usageJSON []byte

	// Device fields
	var deviceName, deviceOS, deviceHostname, deviceVersion, deviceFingerprint sql.NullString
	var lastSeenAt, autoExpireAt sql.NullTime

	err := db.db.QueryRowContext(ctx, query, userID, fingerprint).Scan(
		&key.ID,
		&key.Name,
		&description,
		&key.KeyHash,
		&userIDNullable,
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
		&deviceName,
		&deviceOS,
		&deviceHostname,
		&deviceVersion,
		&deviceFingerprint,
		&lastSeenAt,
		&autoExpireAt,
	)

	if err == sql.ErrNoRows {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query device: %w", err)
	}

	// Handle nullable fields
	if description.Valid {
		key.Description = description.String
	}
	if userIDNullable.Valid {
		key.UserID = &userIDNullable.String
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
		key.RevokedReason = revokedReason.String
	}

	// Device fields
	if deviceName.Valid {
		key.DeviceName = &deviceName.String
	}
	if deviceOS.Valid {
		key.DeviceOS = &deviceOS.String
	}
	if deviceHostname.Valid {
		key.DeviceHostname = &deviceHostname.String
	}
	if deviceVersion.Valid {
		key.DeviceVersion = &deviceVersion.String
	}
	if deviceFingerprint.Valid {
		key.DeviceFingerprint = &deviceFingerprint.String
	}
	if lastSeenAt.Valid {
		key.LastSeenAt = &lastSeenAt.Time
	}
	if autoExpireAt.Valid {
		key.AutoExpireAt = &autoExpireAt.Time
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

// UpdateAPIKeyLastSeen updates last_seen_at timestamp for device API key (Version 2.4.0+)
func (db *PostgreSQLDB) UpdateAPIKeyLastSeen(ctx context.Context, keyID string) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := `
		UPDATE api_keys 
		SET last_seen_at = NOW()
		WHERE id = $1 AND device_fingerprint IS NOT NULL
	`

	result, err := db.db.ExecContext(ctx, query, keyID)
	if err != nil {
		return fmt.Errorf("failed to update last_seen_at: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		// Not an error, just no device key to update
		return nil
	}

	return nil
}

// ListDeviceAPIKeys lists all device API keys for a user (Version 2.4.2+)
func (db *PostgreSQLDB) ListDeviceAPIKeys(ctx context.Context, userID string, filters models.DeviceFilters) ([]*models.APIKey, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, description, key_hash, user_id, tenant_id, scope,
			models, permissions, rate_limits, status,
			created_at, updated_at, expires_at, last_used_at,
			revoked_at, revoked_reason, metadata, usage,
			device_name, device_os, device_hostname, device_version, device_fingerprint,
			last_seen_at, auto_expire_at
		FROM api_keys 
		WHERE user_id = $1 AND device_fingerprint IS NOT NULL
	`

	// Add status filter
	args := []any{userID}
	if filters.Status == "active" {
		query += " AND status = 'active'"
	} else if filters.Status == "expired" {
		query += " AND (status = 'expired' OR (auto_expire_at IS NOT NULL AND auto_expire_at < NOW()))"
	}
	// "all" - no additional filter

	// Add sorting
	sortColumn := "last_seen_at"
	if filters.Sort == "created_at" {
		sortColumn = "created_at"
	} else if filters.Sort == "name" {
		sortColumn = "device_name"
	}

	sortOrder := "DESC"
	if filters.Order == "asc" {
		sortOrder = "ASC"
	}

	query += fmt.Sprintf(" ORDER BY %s %s", sortColumn, sortOrder)

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query device keys: %w", err)
	}
	defer rows.Close()

	var keys []*models.APIKey

	for rows.Next() {
		var key models.APIKey
		var description, userIDNullable, tenantID sql.NullString
		var expiresAt, lastUsedAt, revokedAt sql.NullTime
		var revokedReason sql.NullString
		var modelsJSON, permissionsJSON, rateLimitsJSON, metadataJSON, usageJSON []byte

		// Device fields
		var deviceName, deviceOS, deviceHostname, deviceVersion, deviceFingerprint sql.NullString
		var lastSeenAt, autoExpireAt sql.NullTime

		err := rows.Scan(
			&key.ID,
			&key.Name,
			&description,
			&key.KeyHash,
			&userIDNullable,
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
			&deviceName,
			&deviceOS,
			&deviceHostname,
			&deviceVersion,
			&deviceFingerprint,
			&lastSeenAt,
			&autoExpireAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan device key: %w", err)
		}

		// Handle nullable fields
		if description.Valid {
			key.Description = description.String
		}
		if userIDNullable.Valid {
			key.UserID = &userIDNullable.String
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
			key.RevokedReason = revokedReason.String
		}

		// Device fields
		if deviceName.Valid {
			key.DeviceName = &deviceName.String
		}
		if deviceOS.Valid {
			key.DeviceOS = &deviceOS.String
		}
		if deviceHostname.Valid {
			key.DeviceHostname = &deviceHostname.String
		}
		if deviceVersion.Valid {
			key.DeviceVersion = &deviceVersion.String
		}
		if deviceFingerprint.Valid {
			key.DeviceFingerprint = &deviceFingerprint.String
		}
		if lastSeenAt.Valid {
			key.LastSeenAt = &lastSeenAt.Time
		}
		if autoExpireAt.Valid {
			key.AutoExpireAt = &autoExpireAt.Time
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

		keys = append(keys, &key)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating device keys: %w", err)
	}

	return keys, nil
}

// UpdateDeviceName updates device name for a device API key (Version 2.4.2+)
func (db *PostgreSQLDB) UpdateDeviceName(ctx context.Context, keyID, userID, newName string) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := `
		UPDATE api_keys 
		SET device_name = $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3 AND device_fingerprint IS NOT NULL
	`

	result, err := db.db.ExecContext(ctx, query, newName, keyID, userID)
	if err != nil {
		return fmt.Errorf("failed to update device name: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return storage.ErrNotFound
	}

	return nil
}
