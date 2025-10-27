// Package sqlite provides SQLite implementation for RBAC operations
// Version: 1.11.5+ (Enterprise Suite - Custom Roles & Permissions)
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"aigateway/internal/models"
)

// ========================================
// Permissions CRUD
// ========================================

// CreatePermission создает новое разрешение
func (s *SQLiteDB) CreatePermission(ctx context.Context, permission *models.RBACPermission) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	if permission.ID == "" {
		permission.ID = uuid.New().String()
	}
	if permission.CreatedAt.IsZero() {
		permission.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO permissions (id, name, description, resource, action, scope, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		permission.ID,
		permission.Name,
		permission.Description,
		permission.Resource,
		permission.Action,
		permission.Scope,
		permission.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create permission: %w", err)
	}

	s.logger.WithField("permission_id", permission.ID).Info("Permission created")
	return nil
}

// GetPermission получает разрешение по ID
func (s *SQLiteDB) GetPermission(ctx context.Context, id string) (*models.RBACPermission, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT id, name, description, resource, action, scope, created_at
		FROM permissions
		WHERE id = ?
	`

	var perm models.RBACPermission
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&perm.ID,
		&perm.Name,
		&perm.Description,
		&perm.Resource,
		&perm.Action,
		&perm.Scope,
		&perm.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("permission not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return &perm, nil
}

// GetPermissionByName получает разрешение по имени
func (s *SQLiteDB) GetPermissionByName(ctx context.Context, name string) (*models.RBACPermission, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT id, name, description, resource, action, scope, created_at
		FROM permissions
		WHERE name = ?
	`

	var perm models.RBACPermission
	err := s.db.QueryRowContext(ctx, query, name).Scan(
		&perm.ID,
		&perm.Name,
		&perm.Description,
		&perm.Resource,
		&perm.Action,
		&perm.Scope,
		&perm.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("permission not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get permission: %w", err)
	}

	return &perm, nil
}

// ListPermissions возвращает все разрешения
func (s *SQLiteDB) ListPermissions(ctx context.Context) ([]*models.RBACPermission, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT id, name, description, resource, action, scope, created_at
		FROM permissions
		ORDER BY resource, action
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*models.RBACPermission
	for rows.Next() {
		var perm models.RBACPermission
		err := rows.Scan(
			&perm.ID,
			&perm.Name,
			&perm.Description,
			&perm.Resource,
			&perm.Action,
			&perm.Scope,
			&perm.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, &perm)
	}

	return permissions, nil
}

// ========================================
// Roles CRUD
// ========================================

// CreateRole создает новую роль
func (s *SQLiteDB) CreateRole(ctx context.Context, role *models.Role) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	if role.ID == "" {
		role.ID = uuid.New().String()
	}
	if role.CreatedAt.IsZero() {
		role.CreatedAt = time.Now()
	}
	if role.UpdatedAt.IsZero() {
		role.UpdatedAt = time.Now()
	}

	query := `
		INSERT INTO roles (id, name, display_name, description, type, scope, tenant_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		role.ID,
		role.Name,
		role.DisplayName,
		role.Description,
		role.Type,
		role.Scope,
		role.TenantID,
		role.CreatedAt,
		role.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	s.logger.WithField("role_id", role.ID).Info("Role created")
	return nil
}

// GetRole получает роль по ID
func (s *SQLiteDB) GetRole(ctx context.Context, id string) (*models.Role, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT id, name, display_name, description, type, scope, tenant_id, created_at, updated_at
		FROM roles
		WHERE id = ?
	`

	var role models.Role
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&role.ID,
		&role.Name,
		&role.DisplayName,
		&role.Description,
		&role.Type,
		&role.Scope,
		&role.TenantID,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &role, nil
}

// GetRoleByName получает роль по имени (с учетом tenant)
func (s *SQLiteDB) GetRoleByName(ctx context.Context, name string, tenantID *string) (*models.Role, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT id, name, display_name, description, type, scope, tenant_id, created_at, updated_at
		FROM roles
		WHERE name = ? AND (tenant_id = ? OR (tenant_id IS NULL AND ? IS NULL))
	`

	var role models.Role
	err := s.db.QueryRowContext(ctx, query, name, tenantID, tenantID).Scan(
		&role.ID,
		&role.Name,
		&role.DisplayName,
		&role.Description,
		&role.Type,
		&role.Scope,
		&role.TenantID,
		&role.CreatedAt,
		&role.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("role not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &role, nil
}

// ListRoles возвращает роли (опционально для конкретного tenant)
func (s *SQLiteDB) ListRoles(ctx context.Context, tenantID *string) ([]*models.Role, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	var query string
	var args []interface{}

	if tenantID == nil {
		// Get only global roles
		query = `
			SELECT id, name, display_name, description, type, scope, tenant_id, created_at, updated_at
			FROM roles
			WHERE tenant_id IS NULL
			ORDER BY type, name
		`
	} else {
		// Get global roles + tenant-specific roles
		query = `
			SELECT id, name, display_name, description, type, scope, tenant_id, created_at, updated_at
			FROM roles
			WHERE tenant_id IS NULL OR tenant_id = ?
			ORDER BY type, name
		`
		args = append(args, *tenantID)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []*models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.DisplayName,
			&role.Description,
			&role.Type,
			&role.Scope,
			&role.TenantID,
			&role.CreatedAt,
			&role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, &role)
	}

	return roles, nil
}

// UpdateRole обновляет роль
func (s *SQLiteDB) UpdateRole(ctx context.Context, role *models.Role) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	role.UpdatedAt = time.Now()

	query := `
		UPDATE roles
		SET display_name = ?, description = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		role.DisplayName,
		role.Description,
		role.UpdatedAt,
		role.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role not found: %s", role.ID)
	}

	s.logger.WithField("role_id", role.ID).Info("Role updated")
	return nil
}

// DeleteRole удаляет роль
func (s *SQLiteDB) DeleteRole(ctx context.Context, id string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Check if it's a system role
	role, err := s.GetRole(ctx, id)
	if err != nil {
		return err
	}

	if role.Type == string(models.RoleTypeSystem) {
		return fmt.Errorf("cannot delete system role: %s", role.Name)
	}

	query := `DELETE FROM roles WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("role not found: %s", id)
	}

	s.logger.WithField("role_id", id).Info("Role deleted")
	return nil
}

// ========================================
// Role-Permission Mapping
// ========================================

// AssignPermissionToRole назначает разрешение роли
func (s *SQLiteDB) AssignPermissionToRole(ctx context.Context, roleID, permissionID string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := `
		INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
		VALUES (?, ?)
	`

	_, err := s.db.ExecContext(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to assign permission to role: %w", err)
	}

	s.logger.WithField("role_id", roleID).WithField("permission_id", permissionID).Info("Permission assigned to role")
	return nil
}

// RemovePermissionFromRole удаляет разрешение у роли
func (s *SQLiteDB) RemovePermissionFromRole(ctx context.Context, roleID, permissionID string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := `DELETE FROM role_permissions WHERE role_id = ? AND permission_id = ?`

	_, err := s.db.ExecContext(ctx, query, roleID, permissionID)
	if err != nil {
		return fmt.Errorf("failed to remove permission from role: %w", err)
	}

	s.logger.WithField("role_id", roleID).WithField("permission_id", permissionID).Info("Permission removed from role")
	return nil
}

// GetRolePermissions возвращает все разрешения роли
func (s *SQLiteDB) GetRolePermissions(ctx context.Context, roleID string) ([]*models.RBACPermission, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT p.id, p.name, p.description, p.resource, p.action, p.scope, p.created_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = ?
		ORDER BY p.resource, p.action
	`

	rows, err := s.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}
	defer rows.Close()

	var permissions []*models.RBACPermission
	for rows.Next() {
		var perm models.RBACPermission
		err := rows.Scan(
			&perm.ID,
			&perm.Name,
			&perm.Description,
			&perm.Resource,
			&perm.Action,
			&perm.Scope,
			&perm.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, &perm)
	}

	return permissions, nil
}

// ========================================
// User-Role Assignments
// ========================================

// AssignRoleToUser назначает роль пользователю
func (s *SQLiteDB) AssignRoleToUser(ctx context.Context, userRole *models.UserRole) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	if userRole.ID == "" {
		userRole.ID = uuid.New().String()
	}
	if userRole.CreatedAt.IsZero() {
		userRole.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO user_roles (id, user_id, role_id, tenant_id, created_at)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		userRole.ID,
		userRole.UserID,
		userRole.RoleID,
		userRole.TenantID,
		userRole.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to assign role to user: %w", err)
	}

	s.logger.WithField("user_id", userRole.UserID).WithField("role_id", userRole.RoleID).Info("Role assigned to user")
	return nil
}

// RemoveRoleFromUser удаляет роль у пользователя
func (s *SQLiteDB) RemoveRoleFromUser(ctx context.Context, userID, roleID string, tenantID *string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	query := `
		DELETE FROM user_roles
		WHERE user_id = ? AND role_id = ? AND (tenant_id = ? OR (tenant_id IS NULL AND ? IS NULL))
	`

	_, err := s.db.ExecContext(ctx, query, userID, roleID, tenantID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to remove role from user: %w", err)
	}

	s.logger.WithField("user_id", userID).WithField("role_id", roleID).Info("Role removed from user")
	return nil
}

// GetUserRoles возвращает все роли пользователя
func (s *SQLiteDB) GetUserRoles(ctx context.Context, userID string) ([]*models.UserRole, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT id, user_id, role_id, tenant_id, created_at
		FROM user_roles
		WHERE user_id = ?
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}
	defer rows.Close()

	var userRoles []*models.UserRole
	for rows.Next() {
		var ur models.UserRole
		err := rows.Scan(
			&ur.ID,
			&ur.UserID,
			&ur.RoleID,
			&ur.TenantID,
			&ur.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user role: %w", err)
		}
		userRoles = append(userRoles, &ur)
	}

	return userRoles, nil
}

// GetRoleUsers возвращает всех пользователей с данной ролью
func (s *SQLiteDB) GetRoleUsers(ctx context.Context, roleID string) ([]*models.User, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT u.id, u.username, u.email, u.full_name, u.password_hash, u.status,
		       u.is_admin, u.is_active, u.verified, u.verified_at, u.created_at, u.updated_at,
		       u.last_login, u.preferences, u.metadata,
		       u.auth_provider, u.oidc_subject, u.oidc_issuer, u.ldap_dn
		FROM users u
		INNER JOIN user_roles ur ON u.id = ur.user_id
		WHERE ur.role_id = ?
		ORDER BY u.username
	`

	rows, err := s.db.QueryContext(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get role users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user, err := s.scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}


