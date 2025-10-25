// Package rbac provides Role-Based Access Control (RBAC) functionality
// Version: 1.11.5+ (Enterprise Suite - Custom Roles & Permissions)
package rbac

import (
	"ollama-openai-proxy/internal/models"
)

// RoleTemplate defines a role with its permissions (used for seeding)
type RoleTemplate struct {
	Name        string
	DisplayName string
	Description string
	Type        models.RoleType
	Scope       models.RoleScope
	Permissions []string // Permission names
}

// SystemRoles defines predefined roles that are seeded into the database
var SystemRoles = []RoleTemplate{
	{
		Name:        models.RoleSuperAdmin,
		DisplayName: "Super Administrator",
		Description: "Full system access with all permissions",
		Type:        models.RoleTypeSystem,
		Scope:       models.RoleScopeGlobal,
		Permissions: []string{
			models.PermAll, // Wildcard - все разрешения
		},
	},
	{
		Name:        models.RoleAdmin,
		DisplayName: "Administrator",
		Description: "Administrative access to manage users, tenants, and API keys",
		Type:        models.RoleTypeSystem,
		Scope:       models.RoleScopeGlobal,
		Permissions: []string{
			// Users management
			models.PermUsersCreate,
			models.PermUsersRead,
			models.PermUsersUpdate,
			models.PermUsersDelete,

			// Tenants management
			models.PermTenantsCreate,
			models.PermTenantsRead,
			models.PermTenantsUpdate,
			models.PermTenantsDelete,

			// API Keys management
			models.PermAPIKeysCreate,
			models.PermAPIKeysRead,
			models.PermAPIKeysUpdate,
			models.PermAPIKeysDelete,

			// System access
			models.PermAuditRead,
			models.PermSystemLogs,
			models.PermSystemBackup,

			// Chat & Models (for testing)
			models.PermChatUse,
			models.PermModelsRead,
		},
	},
	{
		Name:        models.RoleAPIManager,
		DisplayName: "API Manager",
		Description: "Manage API keys and view models",
		Type:        models.RoleTypeSystem,
		Scope:       models.RoleScopeGlobal,
		Permissions: []string{
			models.PermAPIKeysCreate,
			models.PermAPIKeysRead,
			models.PermAPIKeysUpdate,
			models.PermAPIKeysDelete,
			models.PermModelsRead,
			models.PermChatUse,
		},
	},
	{
		Name:        models.RoleUser,
		DisplayName: "Regular User",
		Description: "Basic user with chat and model access",
		Type:        models.RoleTypeSystem,
		Scope:       models.RoleScopeGlobal,
		Permissions: []string{
			models.PermChatUse,
			models.PermModelsRead,
			models.PermAPIKeysRead, // Only own keys (enforced by handler)
		},
	},
	{
		Name:        models.RoleReadOnly,
		DisplayName: "Read Only",
		Description: "Read-only access to system resources",
		Type:        models.RoleTypeSystem,
		Scope:       models.RoleScopeGlobal,
		Permissions: []string{
			models.PermUsersRead,
			models.PermTenantsRead,
			models.PermAPIKeysRead,
			models.PermModelsRead,
			models.PermAuditRead,
			models.PermSystemLogs,
		},
	},
}

// GetRoleTemplate возвращает template роли по имени
func GetRoleTemplate(name string) *RoleTemplate {
	for i := range SystemRoles {
		if SystemRoles[i].Name == name {
			return &SystemRoles[i]
		}
	}
	return nil
}

// IsSystemRole проверяет, является ли роль системной
func IsSystemRole(roleName string) bool {
	for _, role := range SystemRoles {
		if role.Name == roleName {
			return true
		}
	}
	return false
}

