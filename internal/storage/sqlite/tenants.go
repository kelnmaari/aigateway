// Package sqlite provides SQLite implementation of tenant-related database operations
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"ollama-openai-proxy/internal/models"
)

// ========================================
// Tenants CRUD Operations
// ========================================

// CreateTenant создает новый tenant
func (s *SQLiteDB) CreateTenant(ctx context.Context, tenant *models.Tenant) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("slug", tenant.Slug).Debug("Creating tenant")

	// Serialize settings to JSON
	settingsJSON, err := json.Marshal(tenant.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Serialize metadata to JSON if present
	var metadataJSON []byte
	if tenant.Metadata != nil {
		metadataJSON, err = json.Marshal(tenant.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO tenants (
			id, name, slug, type, description, owner_id,
			status, is_active, created_at, updated_at,
			settings, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.Slug,
		tenant.Type,
		tenant.Description,
		tenant.OwnerID,
		tenant.Status,
		tenant.IsActive,
		tenant.CreatedAt,
		tenant.UpdatedAt,
		settingsJSON,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert tenant: %w", err)
	}

	s.logger.WithField("tenant_id", tenant.ID).Info("Tenant created successfully")
	return nil
}

// GetTenant получает tenant по ID
func (s *SQLiteDB) GetTenant(ctx context.Context, id string) (*models.Tenant, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, slug, type, description, owner_id,
			status, is_active, created_at, updated_at,
			settings, metadata
		FROM tenants
		WHERE id = ?
	`

	tenant, err := s.scanTenant(s.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return tenant, nil
}

// GetTenantBySlug получает tenant по slug
func (s *SQLiteDB) GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, slug, type, description, owner_id,
			status, is_active, created_at, updated_at,
			settings, metadata
		FROM tenants
		WHERE slug = ?
	`

	tenant, err := s.scanTenant(s.db.QueryRowContext(ctx, query, slug))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found: %s", slug)
		}
		return nil, fmt.Errorf("failed to get tenant by slug: %w", err)
	}

	return tenant, nil
}

// UpdateTenant обновляет данные tenant
func (s *SQLiteDB) UpdateTenant(ctx context.Context, tenant *models.Tenant) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("tenant_id", tenant.ID).Debug("Updating tenant")

	// Serialize settings to JSON
	settingsJSON, err := json.Marshal(tenant.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Serialize metadata to JSON if present
	var metadataJSON []byte
	if tenant.Metadata != nil {
		metadataJSON, err = json.Marshal(tenant.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		UPDATE tenants SET
			name = ?,
			slug = ?,
			type = ?,
			description = ?,
			owner_id = ?,
			status = ?,
			is_active = ?,
			updated_at = ?,
			settings = ?,
			metadata = ?
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		tenant.Name,
		tenant.Slug,
		tenant.Type,
		tenant.Description,
		tenant.OwnerID,
		tenant.Status,
		tenant.IsActive,
		tenant.UpdatedAt,
		settingsJSON,
		metadataJSON,
		tenant.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found: %s", tenant.ID)
	}

	s.logger.WithField("tenant_id", tenant.ID).Info("Tenant updated successfully")
	return nil
}

// DeleteTenant удаляет tenant (soft delete)
func (s *SQLiteDB) DeleteTenant(ctx context.Context, id string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("tenant_id", id).Debug("Deleting tenant (soft delete)")

	// Soft delete: update status to 'deleted'
	query := `
		UPDATE tenants SET
			status = 'deleted',
			updated_at = CURRENT_TIMESTAMP,
			is_active = 0
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found: %s", id)
	}

	s.logger.WithField("tenant_id", id).Info("Tenant deleted successfully (soft delete)")
	return nil
}

// ListUserTenants возвращает список tenants пользователя
func (s *SQLiteDB) ListUserTenants(ctx context.Context, userID string) ([]*models.Tenant, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get tenants where user is owner OR member
	query := `
		SELECT DISTINCT
			t.id, t.name, t.slug, t.type, t.description, t.owner_id,
			t.status, t.is_active, t.created_at, t.updated_at,
			t.settings, t.metadata
		FROM tenants t
		LEFT JOIN tenant_members tm ON t.id = tm.tenant_id
		WHERE t.owner_id = ? OR tm.user_id = ?
		ORDER BY t.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		tenant, err := s.scanTenant(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenants = append(tenants, tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenants: %w", err)
	}

	s.logger.WithField("count", len(tenants)).Debug("Listed user tenants")
	return tenants, nil
}

// ========================================
// Tenant Members CRUD Operations
// ========================================

// AddTenantMember добавляет участника в tenant
func (s *SQLiteDB) AddTenantMember(ctx context.Context, member *models.TenantMember) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithFields(map[string]interface{}{
		"tenant_id": member.TenantID,
		"user_id":   member.UserID,
		"role":      member.Role,
	}).Debug("Adding tenant member")

	// Serialize metadata to JSON if present
	var metadataJSON []byte
	var err error
	if member.Metadata != nil {
		metadataJSON, err = json.Marshal(member.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO tenant_members (
			tenant_id, user_id, role,
			joined_at, updated_at, left_at, invited_by,
			metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		member.TenantID,
		member.UserID,
		member.Role,
		member.JoinedAt,
		member.UpdatedAt,
		member.LeftAt,
		member.InvitedBy,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert tenant member: %w", err)
	}

	s.logger.WithFields(map[string]interface{}{
		"tenant_id": member.TenantID,
		"user_id":   member.UserID,
	}).Info("Tenant member added successfully")

	return nil
}

// GetTenantMember получает участника tenant
func (s *SQLiteDB) GetTenantMember(ctx context.Context, tenantID, userID string) (*models.TenantMember, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			tenant_id, user_id, role,
			joined_at, updated_at, left_at, invited_by,
			metadata
		FROM tenant_members
		WHERE tenant_id = ? AND user_id = ?
	`

	member, err := s.scanTenantMember(s.db.QueryRowContext(ctx, query, tenantID, userID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant member not found: tenant=%s, user=%s", tenantID, userID)
		}
		return nil, fmt.Errorf("failed to get tenant member: %w", err)
	}

	return member, nil
}

// UpdateTenantMember обновляет данные участника tenant
func (s *SQLiteDB) UpdateTenantMember(ctx context.Context, member *models.TenantMember) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithFields(map[string]interface{}{
		"tenant_id": member.TenantID,
		"user_id":   member.UserID,
	}).Debug("Updating tenant member")

	// Serialize metadata to JSON if present
	var metadataJSON []byte
	var err error
	if member.Metadata != nil {
		metadataJSON, err = json.Marshal(member.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		UPDATE tenant_members SET
			role = ?,
			updated_at = ?,
			left_at = ?,
			metadata = ?
		WHERE tenant_id = ? AND user_id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		member.Role,
		member.UpdatedAt,
		member.LeftAt,
		metadataJSON,
		member.TenantID,
		member.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update tenant member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant member not found: tenant=%s, user=%s", member.TenantID, member.UserID)
	}

	s.logger.WithFields(map[string]interface{}{
		"tenant_id": member.TenantID,
		"user_id":   member.UserID,
	}).Info("Tenant member updated successfully")

	return nil
}

// RemoveTenantMember удаляет участника из tenant
func (s *SQLiteDB) RemoveTenantMember(ctx context.Context, tenantID, userID string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithFields(map[string]interface{}{
		"tenant_id": tenantID,
		"user_id":   userID,
	}).Debug("Removing tenant member")

	query := `DELETE FROM tenant_members WHERE tenant_id = ? AND user_id = ?`

	result, err := s.db.ExecContext(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete tenant member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant member not found: tenant=%s, user=%s", tenantID, userID)
	}

	s.logger.WithFields(map[string]interface{}{
		"tenant_id": tenantID,
		"user_id":   userID,
	}).Info("Tenant member removed successfully")

	return nil
}

// ListTenantMembers возвращает список участников tenant
func (s *SQLiteDB) ListTenantMembers(ctx context.Context, tenantID string) ([]*models.TenantMember, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			tenant_id, user_id, role,
			joined_at, updated_at, left_at, invited_by,
			metadata
		FROM tenant_members
		WHERE tenant_id = ?
		ORDER BY joined_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant members: %w", err)
	}
	defer rows.Close()

	var members []*models.TenantMember
	for rows.Next() {
		member, err := s.scanTenantMember(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant member: %w", err)
		}
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenant members: %w", err)
	}

	s.logger.WithField("count", len(members)).Debug("Listed tenant members")
	return members, nil
}

// ========================================
// Helper Methods
// ========================================

// scanTenant сканирует строку БД в модель Tenant
func (s *SQLiteDB) scanTenant(row scanner) (*models.Tenant, error) {
	var tenant models.Tenant
	var settingsJSON []byte
	var metadataJSON []byte
	var description sql.NullString

	err := row.Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.Slug,
		&tenant.Type,
		&description,
		&tenant.OwnerID,
		&tenant.Status,
		&tenant.IsActive,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&settingsJSON,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable description
	if description.Valid {
		tenant.Description = description.String
	}

	// Deserialize settings
	if err := json.Unmarshal(settingsJSON, &tenant.Settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}

	// Deserialize metadata if present
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &tenant.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &tenant, nil
}

// scanTenantMember сканирует строку БД в модель TenantMember
func (s *SQLiteDB) scanTenantMember(row scanner) (*models.TenantMember, error) {
	var member models.TenantMember
	var metadataJSON []byte
	var leftAt sql.NullTime
	var invitedBy sql.NullString

	err := row.Scan(
		&member.TenantID,
		&member.UserID,
		&member.Role,
		&member.JoinedAt,
		&member.UpdatedAt,
		&leftAt,
		&invitedBy,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if leftAt.Valid {
		member.LeftAt = &leftAt.Time
	}
	if invitedBy.Valid {
		str := invitedBy.String
		member.InvitedBy = str
	}

	// Deserialize metadata if present
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &member.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &member, nil
}
