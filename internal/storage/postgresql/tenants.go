// Package postgresql provides PostgreSQL implementation of tenant-related database operations
package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"aigateway/internal/models"
)

// ========================================
// Tenants CRUD Operations
// ========================================

// CreateTenant создает новый tenant
func (db *PostgreSQLDB) CreateTenant(ctx context.Context, tenant *models.Tenant) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	db.logger.WithField("slug", tenant.Slug).Debug("Creating tenant")

	// Serialize settings to JSON
	settingsJSON, err := json.Marshal(tenant.Settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Serialize metadata to JSON if present
	var metadataValue any
	if tenant.Metadata != nil {
		metadataJSON, err := json.Marshal(tenant.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataValue = metadataJSON
	} else {
		metadataValue = nil // PostgreSQL NULL
	}

	query := `
		INSERT INTO tenants (
			id, name, slug, type, description, owner_id,
			status, is_active, created_at, updated_at,
			settings, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err = db.db.ExecContext(ctx, query,
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
		metadataValue,
	)

	if err != nil {
		return fmt.Errorf("failed to insert tenant: %w", err)
	}

	db.logger.WithField("tenant_id", tenant.ID).Info("Tenant created successfully")
	return nil
}

// GetTenant получает tenant по ID
func (db *PostgreSQLDB) GetTenant(ctx context.Context, id string) (*models.Tenant, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, slug, type, description, owner_id,
			status, is_active, created_at, updated_at,
			settings, metadata
		FROM tenants
		WHERE id = $1
	`

	tenant, err := db.scanTenant(db.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return tenant, nil
}

// GetTenantBySlug получает tenant по slug
func (db *PostgreSQLDB) GetTenantBySlug(ctx context.Context, slug string) (*models.Tenant, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, slug, type, description, owner_id,
			status, is_active, created_at, updated_at,
			settings, metadata
		FROM tenants
		WHERE slug = $1
	`

	tenant, err := db.scanTenant(db.db.QueryRowContext(ctx, query, slug))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found: %s", slug)
		}
		return nil, fmt.Errorf("failed to get tenant by slug: %w", err)
	}

	return tenant, nil
}

// GetTenantByName получает tenant по имени (Version 1.11.2+: для OIDC auto-provisioning)
func (db *PostgreSQLDB) GetTenantByName(ctx context.Context, name string) (*models.Tenant, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, slug, type, description, owner_id,
			status, is_active, created_at, updated_at,
			settings, metadata
		FROM tenants
		WHERE name = $1
	`

	tenant, err := db.scanTenant(db.db.QueryRowContext(ctx, query, name))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get tenant by name: %w", err)
	}

	return tenant, nil
}

// UpdateTenant обновляет данные tenant
func (db *PostgreSQLDB) UpdateTenant(ctx context.Context, tenant *models.Tenant) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	db.logger.WithField("tenant_id", tenant.ID).Debug("Updating tenant")

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
			name = $1,
			slug = $2,
			type = $3,
			description = $4,
			owner_id = $5,
			status = $6,
			is_active = $7,
			updated_at = $8,
			settings = $9,
			metadata = $10
		WHERE id = $11
	`

	result, err := db.db.ExecContext(ctx, query,
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

	db.logger.WithField("tenant_id", tenant.ID).Info("Tenant updated successfully")
	return nil
}

// DeleteTenant удаляет tenant (soft delete)
func (db *PostgreSQLDB) DeleteTenant(ctx context.Context, id string) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	db.logger.WithField("tenant_id", id).Debug("Deleting tenant (soft delete)")

	// Soft delete: update status to 'deleted'
	query := `
		UPDATE tenants SET
			status = 'deleted',
			updated_at = NOW(),
			is_active = 0
		WHERE id = $1
	`

	result, err := db.db.ExecContext(ctx, query, id)
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

	db.logger.WithField("tenant_id", id).Info("Tenant deleted successfully (soft delete)")
	return nil
}

// ListUserTenants возвращает список tenants пользователя с их ролями
func (db *PostgreSQLDB) ListUserTenants(ctx context.Context, userID string) ([]*models.Tenant, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Get tenants where user is owner OR member (with role and member count)
	query := `
		SELECT DISTINCT
			t.id, t.name, t.slug, t.type, t.description, t.owner_id,
			t.status, t.is_active, t.created_at, t.updated_at,
			t.settings, t.metadata,
			CASE 
				WHEN t.owner_id = $1 THEN 'owner'
				ELSE COALESCE(tm.role, 'member')
			END as role,
			(SELECT COUNT(*) FROM tenant_members WHERE tenant_id = t.id) as member_count
		FROM tenants t
		LEFT JOIN tenant_members tm ON t.id = tm.tenant_id AND tm.user_id = $2
		WHERE t.owner_id = $3 OR tm.user_id = $4
		ORDER BY t.created_at DESC
	`

	rows, err := db.db.QueryContext(ctx, query, userID, userID, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		tenant, err := db.scanTenantWithRole(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenants = append(tenants, tenant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenants: %w", err)
	}

	db.logger.WithField("count", len(tenants)).Debug("Listed user tenants")
	return tenants, nil
}

// ========================================
// Tenant Members CRUD Operations
// ========================================

// AddTenantMember добавляет участника в tenant
func (db *PostgreSQLDB) AddTenantMember(ctx context.Context, member *models.TenantMember) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	db.logger.WithFields(map[string]any{
		"tenant_id": member.TenantID,
		"user_id":   member.UserID,
		"role":      member.Role,
	}).Debug("Adding tenant member")

	// Serialize metadata to JSON if present
	var metadataValue any
	if member.Metadata != nil {
		metadataJSON, err := json.Marshal(member.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataValue = metadataJSON
	} else {
		metadataValue = nil // PostgreSQL NULL
	}

	query := `
		INSERT INTO tenant_members (
			tenant_id, user_id, role,
			joined_at, updated_at, left_at, invited_by,
			metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := db.db.ExecContext(ctx, query,
		member.TenantID,
		member.UserID,
		member.Role,
		member.JoinedAt,
		member.UpdatedAt,
		member.LeftAt,
		member.InvitedBy,
		metadataValue,
	)

	if err != nil {
		return fmt.Errorf("failed to insert tenant member: %w", err)
	}

	db.logger.WithFields(map[string]any{
		"tenant_id": member.TenantID,
		"user_id":   member.UserID,
	}).Info("Tenant member added successfully")

	return nil
}

// GetTenantMember получает участника tenant
func (db *PostgreSQLDB) GetTenantMember(ctx context.Context, tenantID, userID string) (*models.TenantMember, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			tenant_id, user_id, role,
			joined_at, updated_at, left_at, invited_by,
			metadata
		FROM tenant_members
		WHERE tenant_id = $1 AND user_id = $2
	`

	member, err := db.scanTenantMember(db.db.QueryRowContext(ctx, query, tenantID, userID))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant member not found: tenant=%s, user=%s", tenantID, userID)
		}
		return nil, fmt.Errorf("failed to get tenant member: %w", err)
	}

	return member, nil
}

// UpdateTenantMember обновляет данные участника tenant
func (db *PostgreSQLDB) UpdateTenantMember(ctx context.Context, member *models.TenantMember) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	db.logger.WithFields(map[string]any{
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
			role = $1,
			updated_at = $2,
			left_at = $3,
			metadata = $4
		WHERE tenant_id = $5 AND user_id = $6
	`

	result, err := db.db.ExecContext(ctx, query,
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

	db.logger.WithFields(map[string]any{
		"tenant_id": member.TenantID,
		"user_id":   member.UserID,
	}).Info("Tenant member updated successfully")

	return nil
}

// RemoveTenantMember удаляет участника из tenant
func (db *PostgreSQLDB) RemoveTenantMember(ctx context.Context, tenantID, userID string) error {
	if db.db == nil {
		return fmt.Errorf("database not connected")
	}

	db.logger.WithFields(map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
	}).Debug("Removing tenant member")

	query := `DELETE FROM tenant_members WHERE tenant_id = $1 AND user_id = $2`

	result, err := db.db.ExecContext(ctx, query, tenantID, userID)
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

	db.logger.WithFields(map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
	}).Info("Tenant member removed successfully")

	return nil
}

// ListTenantMembers возвращает список участников tenant
func (db *PostgreSQLDB) ListTenantMembers(ctx context.Context, tenantID string) ([]*models.TenantMember, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			tm.tenant_id, tm.user_id, tm.role,
			tm.joined_at, tm.updated_at, tm.left_at, tm.invited_by,
			tm.metadata,
			u.username, u.email
		FROM tenant_members tm
		LEFT JOIN users u ON tm.user_id = u.id
		WHERE tm.tenant_id = $1
		ORDER BY tm.joined_at ASC
	`

	rows, err := db.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant members: %w", err)
	}
	defer rows.Close()

	var members []*models.TenantMember
	for rows.Next() {
		member, err := db.scanTenantMemberWithUserInfo(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant member: %w", err)
		}
		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenant members: %w", err)
	}

	db.logger.WithField("count", len(members)).Debug("Listed tenant members")
	return members, nil
}

// ========================================
// Helper Methods
// ========================================

// scanTenant сканирует строку БД в модель Tenant
func (db *PostgreSQLDB) scanTenant(row scanner) (*models.Tenant, error) {
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

// scanTenantWithRole сканирует строку БД в модель Tenant с ролью пользователя
func (db *PostgreSQLDB) scanTenantWithRole(row scanner) (*models.Tenant, error) {
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
		&tenant.Role,        // Добавлено поле role
		&tenant.MemberCount, // Добавлено поле member_count
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
func (db *PostgreSQLDB) scanTenantMember(row scanner) (*models.TenantMember, error) {
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

// scanTenantMemberWithUserInfo сканирует строку БД в модель TenantMember с информацией о пользователе
func (db *PostgreSQLDB) scanTenantMemberWithUserInfo(row scanner) (*models.TenantMember, error) {
	var member models.TenantMember
	var metadataJSON []byte
	var leftAt sql.NullTime
	var invitedBy sql.NullString
	var username sql.NullString
	var email sql.NullString

	err := row.Scan(
		&member.TenantID,
		&member.UserID,
		&member.Role,
		&member.JoinedAt,
		&member.UpdatedAt,
		&leftAt,
		&invitedBy,
		&metadataJSON,
		&username,
		&email,
	)

	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if leftAt.Valid {
		member.LeftAt = &leftAt.Time
	}
	if invitedBy.Valid {
		member.InvitedBy = invitedBy.String
	}
	if username.Valid {
		member.Username = username.String
	}
	if email.Valid {
		member.Email = email.String
	}

	// Deserialize metadata if present
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &member.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &member, nil
}

// ListAllTenants возвращает список всех tenants (для админа)
func (db *PostgreSQLDB) ListAllTenants(ctx context.Context) ([]*models.Tenant, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, name, slug, type, description, owner_id,
			status, is_active, created_at, updated_at,
			settings, metadata
		FROM tenants
		WHERE is_active = true
		ORDER BY created_at DESC
	`

	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		tenant, err := db.scanTenant(rows)
		if err != nil {
			db.logger.WithError(err).Error("Failed to scan tenant row")
			continue
		}
		tenants = append(tenants, tenant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tenants: %w", err)
	}

	db.logger.WithField("count", len(tenants)).Debug("Listed all tenants")

	return tenants, nil
}
