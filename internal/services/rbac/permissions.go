// Package rbac provides Role-Based Access Control (RBAC) functionality
// Version: 1.11.5+ (Enterprise Suite - Custom Roles & Permissions)
package rbac

import (
	"ollama-openai-proxy/internal/models"
)

// SystemPermissions defines all available permissions in the system
// These are seeded into the database on first run
var SystemPermissions = []models.Permission{
	// API Keys permissions
	{
		Name:        models.PermAPIKeysCreate,
		Description: "Create API keys",
		Resource:    "api_keys",
		Action:      "create",
		Scope:       string(models.ScopeGlobal),
	},
	{
		Name:        models.PermAPIKeysRead,
		Description: "Read API keys",
		Resource:    "api_keys",
		Action:      "read",
		Scope:       string(models.ScopeGlobal),
	},
	{
		Name:        models.PermAPIKeysUpdate,
		Description: "Update API keys",
		Resource:    "api_keys",
		Action:      "update",
		Scope:       string(models.ScopeGlobal),
	},
	{
		Name:        models.PermAPIKeysDelete,
		Description: "Delete API keys",
		Resource:    "api_keys",
		Action:      "delete",
		Scope:       string(models.ScopeGlobal),
	},

	// Users permissions
	{
		Name:        models.PermUsersCreate,
		Description: "Create users",
		Resource:    "users",
		Action:      "create",
		Scope:       string(models.ScopeGlobal),
	},
	{
		Name:        models.PermUsersRead,
		Description: "Read users",
		Resource:    "users",
		Action:      "read",
		Scope:       string(models.ScopeGlobal),
	},
	{
		Name:        models.PermUsersUpdate,
		Description: "Update users",
		Resource:    "users",
		Action:      "update",
		Scope:       string(models.ScopeGlobal),
	},
	{
		Name:        models.PermUsersDelete,
		Description: "Delete users",
		Resource:    "users",
		Action:      "delete",
		Scope:       string(models.ScopeGlobal),
	},

	// Tenants permissions
	{
		Name:        models.PermTenantsCreate,
		Description: "Create tenants",
		Resource:    "tenants",
		Action:      "create",
		Scope:       string(models.ScopeGlobal),
	},
	{
		Name:        models.PermTenantsRead,
		Description: "Read tenants",
		Resource:    "tenants",
		Action:      "read",
		Scope:       string(models.ScopeGlobal),
	},
	{
		Name:        models.PermTenantsUpdate,
		Description: "Update tenants",
		Resource:    "tenants",
		Action:      "update",
		Scope:       string(models.ScopeTenant),
	},
	{
		Name:        models.PermTenantsDelete,
		Description: "Delete tenants",
		Resource:    "tenants",
		Action:      "delete",
		Scope:       string(models.ScopeGlobal),
	},

	// Chat / Models permissions
	{
		Name:        models.PermChatUse,
		Description: "Use chat functionality",
		Resource:    "chat",
		Action:      "use",
		Scope:       string(models.ScopePersonal),
	},
	{
		Name:        models.PermModelsRead,
		Description: "Read models list",
		Resource:    "models",
		Action:      "read",
		Scope:       string(models.ScopeGlobal),
	},
	{
		Name:        models.PermModelsManage,
		Description: "Manage models (load, unload, configure)",
		Resource:    "models",
		Action:      "manage",
		Scope:       string(models.ScopeGlobal),
	},

	// System permissions
	{
		Name:        models.PermSystemConfig,
		Description: "View and modify system configuration",
		Resource:    "system",
		Action:      "config",
		Scope:       string(models.ScopeGlobal),
	},
	{
		Name:        models.PermSystemBackup,
		Description: "Create and restore backups",
		Resource:    "system",
		Action:      "backup",
		Scope:       string(models.ScopeGlobal),
	},
	{
		Name:        models.PermSystemLogs,
		Description: "View system logs",
		Resource:    "system",
		Action:      "logs",
		Scope:       string(models.ScopeGlobal),
	},
	{
		Name:        models.PermAuditRead,
		Description: "Read audit logs",
		Resource:    "audit",
		Action:      "read",
		Scope:       string(models.ScopeGlobal),
	},
}

// GetPermissionByName находит permission по имени
func GetPermissionByName(name string) *models.Permission {
	for i := range SystemPermissions {
		if SystemPermissions[i].Name == name {
			return &SystemPermissions[i]
		}
	}
	return nil
}

// GetPermissionsByResource возвращает все permissions для resource
func GetPermissionsByResource(resource string) []models.Permission {
	var result []models.Permission
	for _, perm := range SystemPermissions {
		if perm.Resource == resource {
			result = append(result, perm)
		}
	}
	return result
}

