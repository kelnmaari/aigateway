// Package rbac provides Role-Based Access Control (RBAC) functionality
// Version: 1.11.5+ (Enterprise Suite - Custom Roles & Permissions)
package rbac

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// Service provides RBAC functionality for permission checking
type Service struct {
	db     storage.Database
	logger *logrus.Logger
}

// NewService creates a new RBAC service
func NewService(db storage.Database, logger *logrus.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
}

// CheckPermission проверяет, имеет ли пользователь указанное разрешение
//
// Parameters:
//   - ctx: context
//   - userID: ID пользователя
//   - permission: требуемое разрешение (e.g., "api_keys:create")
//   - tenantID: ID tenant (nil для глобальных разрешений)
//
// Returns:
//   - true если разрешение есть, false иначе
//   - error если произошла ошибка при проверке
func (s *Service) CheckPermission(
	ctx context.Context,
	userID string,
	permission string,
	tenantID *string,
) (bool, error) {
	s.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"permission": permission,
		"tenant_id":  tenantID,
	}).Debug("Checking permission")

	// Get user roles
	roles, err := s.db.GetUserRoles(ctx, userID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get user roles")
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	if len(roles) == 0 {
		s.logger.Debug("User has no roles assigned")
		return false, nil
	}

	// Check each role
	for _, role := range roles {
		// Scope filtering
		// Skip tenant roles if checking global permission
		if role.TenantID != nil && tenantID == nil {
			continue
		}

		// Skip global roles if checking tenant permission (unless it's a wildcard)
		if role.TenantID == nil && tenantID != nil {
			// Global roles can still access tenant resources if they have the permission
			// This is intentional - admins should be able to manage any tenant
		}

		// Get role permissions
		permissions, err := s.db.GetRolePermissions(ctx, role.RoleID)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to get role permissions")
			continue
		}

		// Check if permission exists
		for _, p := range permissions {
			if matchPermission(p.Name, permission) {
				s.logger.WithFields(logrus.Fields{
					"user_id":    userID,
					"permission": permission,
					"role_id":    role.RoleID,
				}).Debug("Permission granted")
				return true, nil
			}
		}
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"permission": permission,
	}).Debug("Permission denied")
	return false, nil
}

// HasRole проверяет, имеет ли пользователь указанную роль
func (s *Service) HasRole(ctx context.Context, userID string, roleName string) (bool, error) {
	roles, err := s.db.GetUserRoles(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Get role by name to find its ID
	role, err := s.db.GetRoleByName(ctx, roleName, nil) // nil for global roles
	if err != nil {
		return false, fmt.Errorf("failed to get role: %w", err)
	}

	for _, userRole := range roles {
		if userRole.RoleID == role.ID {
			return true, nil
		}
	}

	return false, nil
}

// GetUserPermissions возвращает все разрешения пользователя (для UI)
func (s *Service) GetUserPermissions(ctx context.Context, userID string) ([]models.RBACPermission, error) {
	// Get user roles
	roles, err := s.db.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	// Collect all permissions (deduplicated)
	permissionMap := make(map[string]models.RBACPermission)
	for _, role := range roles {
		permissions, err := s.db.GetRolePermissions(ctx, role.RoleID)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to get role permissions")
			continue
		}

		for _, perm := range permissions {
			permissionMap[perm.ID] = *perm
		}
	}

	// Convert map to slice
	result := make([]models.RBACPermission, 0, len(permissionMap))
	for _, perm := range permissionMap {
		result = append(result, perm)
	}

	return result, nil
}

// matchPermission проверяет соответствие разрешения паттерну с поддержкой wildcards
//
// Поддерживаемые форматы:
//   - "*:*" - все разрешения
//   - "api_keys:*" - все действия с api_keys
//   - "*:create" - create для всех ресурсов
//   - "api_keys:create" - точное совпадение
//
// Parameters:
//   - pattern: паттерн разрешения (может содержать *)
//   - permission: проверяемое разрешение
//
// Returns:
//   - true если разрешение соответствует паттерну
func matchPermission(pattern, permission string) bool {
	// Wildcard для всех разрешений
	if pattern == models.PermAll { // "*:*"
		return true
	}

	// Точное совпадение
	if pattern == permission {
		return true
	}

	// Проверка паттерна с wildcard
	parts := strings.Split(pattern, ":")
	permParts := strings.Split(permission, ":")

	// Оба должны иметь формат "resource:action"
	if len(parts) != 2 || len(permParts) != 2 {
		return pattern == permission
	}

	resourcePattern := parts[0]
	actionPattern := parts[1]

	resourcePerm := permParts[0]
	actionPerm := permParts[1]

	// Проверка resource
	if resourcePattern != "*" && resourcePattern != resourcePerm {
		return false
	}

	// Проверка action
	if actionPattern != "*" && actionPattern != actionPerm {
		return false
	}

	return true
}

