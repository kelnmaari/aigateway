// Package models defines data models for RBAC (Role-Based Access Control)
// Version: 1.11.5+ (Enterprise Suite - Custom Roles & Permissions)
package models

import "time"

// RBACPermission представляет разрешение на выполнение действия с ресурсом
type RBACPermission struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`               // e.g., "api_keys:create"
	Description string    `json:"description" db:"description"` // e.g., "Create API keys"
	Resource    string    `json:"resource" db:"resource"`       // e.g., "api_keys"
	Action      string    `json:"action" db:"action"`           // e.g., "create", "read", "update", "delete"
	Scope       string    `json:"scope" db:"scope"`             // "global", "tenant", "personal"
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// PermissionScope определяет область действия разрешения
type PermissionScope string

const (
	ScopeGlobal   PermissionScope = "global"   // Глобальное разрешение (для всей системы)
	ScopeTenant   PermissionScope = "tenant"   // Разрешение в рамках tenant
	ScopePersonal PermissionScope = "personal" // Личное разрешение (только свои ресурсы)
)

// Role представляет роль с набором разрешений
type Role struct {
	ID          string       `json:"id" db:"id"`
	Name        string       `json:"name" db:"name"`                   // e.g., "api_manager"
	DisplayName string       `json:"display_name" db:"display_name"`   // e.g., "API Manager"
	Description string       `json:"description" db:"description"`
	Type        string       `json:"type" db:"type"`                   // "system", "custom"
	Scope       string       `json:"scope" db:"scope"`                 // "global", "tenant"
	TenantID    *string          `json:"tenant_id,omitempty" db:"tenant_id"` // null for global roles
	Permissions []RBACPermission `json:"permissions,omitempty" db:"-"`     // Loaded separately
	CreatedAt   time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" db:"updated_at"`
}

// RoleType определяет тип роли
type RoleType string

const (
	RoleTypeSystem RoleType = "system" // Системная роль (нельзя удалить/редактировать)
	RoleTypeCustom RoleType = "custom" // Кастомная роль (создана пользователем)
)

// RoleScope определяет область действия роли
type RoleScope string

const (
	RoleScopeGlobal RoleScope = "global" // Глобальная роль
	RoleScopeTenant RoleScope = "tenant" // Роль в рамках tenant
)

// UserRole представляет назначение роли пользователю
type UserRole struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	RoleID    string    `json:"role_id" db:"role_id"`
	TenantID  *string   `json:"tenant_id,omitempty" db:"tenant_id"` // null for global role assignment
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RolePermission представляет связь роль-разрешение (mapping table)
type RolePermission struct {
	RoleID       string `json:"role_id" db:"role_id"`
	PermissionID string `json:"permission_id" db:"permission_id"`
}

// Predefined System Roles (константы для быстрого доступа)
const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleUser       = "user"
	RoleAPIManager = "api_manager"
	RoleReadOnly   = "read_only"
)

// Permission naming conventions:
// Format: "resource:action"
// Examples:
//   - "api_keys:create"
//   - "users:read"
//   - "tenants:delete"
//   - "*:*" - all permissions (super admin)
//   - "api_keys:*" - all actions on api_keys
const (
	// API Keys permissions
	PermAPIKeysCreate = "api_keys:create"
	PermAPIKeysRead   = "api_keys:read"
	PermAPIKeysUpdate = "api_keys:update"
	PermAPIKeysDelete = "api_keys:delete"

	// Users permissions
	PermUsersCreate = "users:create"
	PermUsersRead   = "users:read"
	PermUsersUpdate = "users:update"
	PermUsersDelete = "users:delete"

	// Tenants permissions
	PermTenantsCreate = "tenants:create"
	PermTenantsRead   = "tenants:read"
	PermTenantsUpdate = "tenants:update"
	PermTenantsDelete = "tenants:delete"

	// Chat / Models permissions
	PermChatUse      = "chat:use"
	PermModelsRead   = "models:read"
	PermModelsManage = "models:manage"

	// System permissions
	PermSystemConfig = "system:config"
	PermSystemBackup = "system:backup"
	PermSystemLogs   = "system:logs"
	PermAuditRead    = "audit:read"

	// Wildcard permissions
	PermAll = "*:*" // Super admin permission
)

