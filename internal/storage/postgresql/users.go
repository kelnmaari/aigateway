// Package postgresql provides PostgreSQL implementation of user-related database operations
package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"aigateway/internal/models"
)

// ========================================
// Users CRUD Operations
// ========================================

// CreateUser создает нового пользователя
func (p *PostgreSQLDB) CreateUser(ctx context.Context, user *models.User) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	p.logger.WithField("username", user.Username).Debug("Creating user")

	// Serialize preferences to JSON
	preferencesJSON, err := json.Marshal(user.Preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// Serialize metadata to JSON if present
	var metadataValue any
	if user.Metadata != nil {
		metadataJSON, err := json.Marshal(user.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataValue = metadataJSON
	} else {
		metadataValue = nil // PostgreSQL NULL
	}

	query := `
		INSERT INTO users (
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err = p.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.FullName,
		user.PasswordHash,
		user.Status,
		user.IsAdmin,
		user.IsActive,
		user.Verified,
		user.VerifiedAt,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastLogin,
		preferencesJSON,
		metadataValue,
	)

	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	p.logger.WithField("user_id", user.ID).Info("User created successfully")
	return nil
}

// GetUser получает пользователя по ID
func (p *PostgreSQLDB) GetUser(ctx context.Context, id string) (*models.User, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata
		FROM users
		WHERE id = $1
	`

	user, err := p.scanUser(p.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByUsername получает пользователя по username
func (p *PostgreSQLDB) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata
		FROM users
		WHERE username = $1
	`

	user, err := p.scanUser(p.db.QueryRowContext(ctx, query, username))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

// GetUserByEmail получает пользователя по email
func (p *PostgreSQLDB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata
		FROM users
		WHERE email = $1
	`

	user, err := p.scanUser(p.db.QueryRowContext(ctx, query, email))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", email)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetUserByLDAPDN получает пользователя по LDAP DN
func (p *PostgreSQLDB) GetUserByLDAPDN(ctx context.Context, ldapDN string) (*models.User, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	p.logger.WithField("ldap_dn", ldapDN).Debug("Getting user by LDAP DN")

	query := `
		SELECT 
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata,
			auth_provider, oidc_subject, oidc_issuer, ldap_dn
		FROM users
		WHERE ldap_dn = $1
	`

	user, err := p.scanUser(p.db.QueryRowContext(ctx, query, ldapDN))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found for LDAP DN=%s", ldapDN)
		}
		return nil, fmt.Errorf("failed to get user by LDAP DN: %w", err)
	}

	return user, nil
}

// UpdateUser обновляет данные пользователя
func (p *PostgreSQLDB) UpdateUser(ctx context.Context, user *models.User) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	p.logger.WithField("user_id", user.ID).Debug("Updating user")

	// Serialize preferences to JSON
	preferencesJSON, err := json.Marshal(user.Preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// Serialize metadata to JSON (empty object if nil)
	var metadataJSON []byte
	if user.Metadata != nil {
		metadataJSON, err = json.Marshal(user.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	} else {
		metadataJSON = []byte("{}")
	}

	query := `
		UPDATE users SET
			username = $1,
			email = $2,
			full_name = $3,
			password_hash = $4,
			status = $5,
			is_admin = $6,
			is_active = $7,
			verified = $8,
			verified_at = $9,
			updated_at = $10,
			last_login = $11,
			preferences = $12,
			metadata = $13
		WHERE id = $14
	`

	result, err := p.db.ExecContext(ctx, query,
		user.Username,
		user.Email,
		user.FullName,
		user.PasswordHash,
		user.Status,
		user.IsAdmin,
		user.IsActive,
		user.Verified,
		user.VerifiedAt,
		user.UpdatedAt,
		user.LastLogin,
		preferencesJSON,
		metadataJSON,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", user.ID)
	}

	p.logger.WithField("user_id", user.ID).Info("User updated successfully")
	return nil
}

// UpdateUserPassword обновляет пароль пользователя
func (p *PostgreSQLDB) UpdateUserPassword(ctx context.Context, userID string, passwordHash string) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	p.logger.WithField("user_id", userID).Debug("Updating user password")

	query := `
		UPDATE users 
		SET password_hash = $1, updated_at = NOW() 
		WHERE id = $2
	`

	result, err := p.db.ExecContext(ctx, query, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", userID)
	}

	p.logger.WithField("user_id", userID).Info("Password updated successfully")
	return nil
}

// DeleteUser удаляет пользователя (soft delete)
func (p *PostgreSQLDB) DeleteUser(ctx context.Context, id string) error {
	if p.db == nil {
		return fmt.Errorf("database not connected")
	}

	p.logger.WithField("user_id", id).Debug("Deleting user (soft delete)")

	// Soft delete: update status to 'deleted'
	query := `
		UPDATE users SET
			status = 'deleted',
			updated_at = CURRENT_TIMESTAMP,
			is_active = 0
		WHERE id = $1
	`

	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", id)
	}

	p.logger.WithField("user_id", id).Info("User deleted successfully (soft delete)")
	return nil
}

// ListUsers возвращает список пользователей с фильтрацией
func (p *PostgreSQLDB) ListUsers(ctx context.Context, filters models.UserFilters) ([]*models.User, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata
		FROM users
		WHERE 1=1
	`

	var args []any
	paramIndex := 1

	// Apply filters
	if filters.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", paramIndex)
		args = append(args, *filters.Status)
		paramIndex++
	}

	if filters.IsAdmin != nil {
		query += fmt.Sprintf(" AND is_admin = $%d", paramIndex)
		args = append(args, *filters.IsAdmin)
		paramIndex++
	}

	if filters.IsActive != nil {
		query += fmt.Sprintf(" AND is_active = $%d", paramIndex)
		args = append(args, *filters.IsActive)
		paramIndex++
	}

	if filters.Search != "" {
		query += fmt.Sprintf(" AND (username ILIKE $%d OR email ILIKE $%d OR full_name ILIKE $%d)", paramIndex, paramIndex+1, paramIndex+2)
		searchPattern := "%" + filters.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
		paramIndex += 3
	}

	// Add ordering
	query += " ORDER BY created_at DESC"

	// Add pagination
	if filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", paramIndex)
		args = append(args, filters.Limit)
		paramIndex++
	}

	if filters.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", paramIndex)
		args = append(args, filters.Offset)
		paramIndex++
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user, err := p.scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	p.logger.WithField("count", len(users)).Debug("Listed users")
	return users, nil
}

// ========================================
// Helper Methods
// ========================================

// scanner is an interface that matches both sql.Row and sql.Rows
type scanner interface {
	Scan(dest ...any) error
}

// scanUser сканирует строку БД в модель User
func (p *PostgreSQLDB) scanUser(row scanner) (*models.User, error) {
	var user models.User
	var preferencesJSON []byte
	var metadataJSON []byte

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.FullName,
		&user.PasswordHash,
		&user.Status,
		&user.IsAdmin,
		&user.IsActive,
		&user.Verified,
		&user.VerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLogin,
		&preferencesJSON,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	// Deserialize preferences
	if err := json.Unmarshal(preferencesJSON, &user.Preferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}

	// Deserialize metadata if present
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &user.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &user, nil
}

// GetUserByOIDCSubject получает пользователя по OIDC subject и issuer (Version 1.11.1+)
func (p *PostgreSQLDB) GetUserByOIDCSubject(ctx context.Context, issuer, subject string) (*models.User, error) {
	if p.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	p.logger.WithFields(map[string]any{
		"issuer":  issuer,
		"subject": subject,
	}).Debug("Getting user by OIDC subject")

	query := `
		SELECT 
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata,
			auth_provider, oidc_subject, oidc_issuer
		FROM users
		WHERE oidc_issuer = $1 AND oidc_subject = $2
	`

	user, err := p.scanUser(p.db.QueryRowContext(ctx, query, issuer, subject))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found for OIDC issuer=%s subject=%s", issuer, subject)
		}
		return nil, fmt.Errorf("failed to get user by OIDC subject: %w", err)
	}

	return user, nil
}

// GetUsersWithDetails возвращает список пользователей с enriched данными (roles, tenants) для админ-панели (v2.2.2+)
func (db *PostgreSQLDB) GetUsersWithDetails(ctx context.Context, filters models.UserFilters) ([]*models.UserWithDetails, error) {
	if db.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Получаем базовый список пользователей
	users, err := db.ListUsers(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Enriched users с ролями и тенантами
	enrichedUsers := make([]*models.UserWithDetails, 0, len(users))

	for _, user := range users {
		userDetails := &models.UserWithDetails{
			User: *user,
		}

		// Получаем роли пользователя
		userRoles, err := db.GetUserRoles(ctx, user.ID)
		if err != nil {
			db.logger.WithError(err).WithField("user_id", user.ID).Warn("Failed to get user roles")
		} else {
			// Конвертируем в RoleInfo
			roleInfos := make([]models.RoleInfo, 0, len(userRoles))
			for _, ur := range userRoles {
				// Получаем информацию о роли
				role, err := db.GetRole(ctx, ur.RoleID)
				if err != nil {
					db.logger.WithError(err).WithField("role_id", ur.RoleID).Warn("Failed to get role details")
					continue
				}

				roleInfos = append(roleInfos, models.RoleInfo{
					ID:       role.ID,
					Name:     role.Name,
					TenantID: ur.TenantID,
				})
			}
			userDetails.Roles = roleInfos
		}

		// Получаем тенанты пользователя
		tenants, err := db.ListUserTenants(ctx, user.ID)
		if err != nil {
			db.logger.WithError(err).WithField("user_id", user.ID).Warn("Failed to get user tenants")
		} else {
			// Конвертируем в TenantInfo
			tenantInfos := make([]models.TenantInfo, 0, len(tenants))
			for _, tenant := range tenants {
				tenantInfos = append(tenantInfos, models.TenantInfo{
					ID:   tenant.ID,
					Name: tenant.Name,
					Slug: tenant.Slug,
					Role: string(tenant.Type), // или Role из tenant_members
				})
			}
			userDetails.Tenants = tenantInfos
		}

		enrichedUsers = append(enrichedUsers, userDetails)
	}

	db.logger.WithField("count", len(enrichedUsers)).Debug("Listed users with details")

	return enrichedUsers, nil
}
