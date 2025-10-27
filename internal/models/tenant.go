// Package models provides data models for Multi-Tenancy
package models

import (
	"time"
)

// Tenant представляет организацию или workspace
type Tenant struct {
	// Основная информация
	ID   string `json:"id" db:"id"`     // Уникальный ID (UUID)
	Name string `json:"name" db:"name"` // Название tenant
	Slug string `json:"slug" db:"slug"` // URL-friendly идентификатор (unique)

	// Тип tenant
	Type        TenantType `json:"type" db:"type"`               // personal или organization
	Description string     `json:"description" db:"description"` // Описание
	OwnerID     string     `json:"owner_id" db:"owner_id"`       // ID владельца (User.ID)

	// Статус
	Status   TenantStatus `json:"status" db:"status"`       // Статус tenant
	IsActive bool         `json:"is_active" db:"is_active"` // Активен ли

	// Временные метки
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Settings
	Settings TenantSettings `json:"settings" db:"settings"` // Настройки (JSONB)

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"`

	// User's Role in this tenant (populated only in ListUserTenants)
	Role string `json:"role,omitempty" db:"-"`

	// Member Count (populated only in ListUserTenants)
	MemberCount int `json:"member_count,omitempty" db:"-"`
}

// TenantType представляет тип tenant
type TenantType string

const (
	TenantTypePersonal     TenantType = "personal"     // Личный workspace (auto-created)
	TenantTypeOrganization TenantType = "organization" // Организация (multi-user)
)

// TenantStatus представляет статус tenant
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"    // Активен
	TenantStatusSuspended TenantStatus = "suspended" // Приостановлен
	TenantStatusDeleted   TenantStatus = "deleted"   // Удален (soft delete)
)

// TenantSettings содержит настройки tenant
type TenantSettings struct {
	// Rate Limits (переопределяют пользовательские)
	MaxAPIKeys       int `json:"max_api_keys"`      // Максимум API ключей
	MaxConversations int `json:"max_conversations"` // Максимум диалогов

	// Features
	ChatEnabled      bool `json:"chat_enabled"`       // Chat UI включен
	APIAccessEnabled bool `json:"api_access_enabled"` // API доступ включен

	// Branding
	LogoURL string `json:"logo_url,omitempty"` // URL логотипа
	Theme   string `json:"theme,omitempty"`    // Кастомная тема
}

// TenantMember представляет участника tenant с ролью
type TenantMember struct {
	// Связи
	TenantID string `json:"tenant_id" db:"tenant_id"` // ID tenant
	UserID   string `json:"user_id" db:"user_id"`     // ID пользователя

	// Роль и права
	Role TenantRole `json:"role" db:"role"` // Роль в tenant

	// User info (from JOIN)
	Username string `json:"username,omitempty" db:"username"` // Username пользователя
	Email    string `json:"email,omitempty" db:"email"`       // Email пользователя

	// Временные метки
	JoinedAt  time.Time  `json:"joined_at" db:"joined_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	LeftAt    *time.Time `json:"left_at,omitempty" db:"left_at"` // Дата выхода (для истории)

	// Metadata
	InvitedBy string                 `json:"invited_by,omitempty" db:"invited_by"` // Кто пригласил
	Metadata  map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
}

// TenantRole представляет роль участника в tenant
type TenantRole string

const (
	TenantRoleOwner  TenantRole = "owner"  // Владелец (все права)
	TenantRoleAdmin  TenantRole = "admin"  // Администратор (управление, кроме удаления)
	TenantRoleMember TenantRole = "member" // Участник (использование)
	TenantRoleViewer TenantRole = "viewer" // Наблюдатель (только просмотр)
)

// Can проверяет, может ли роль выполнить действие
func (r TenantRole) Can(action TenantPermission) bool {
	permissions := map[TenantRole][]TenantPermission{
		TenantRoleOwner: {
			PermissionManageTenant,
			PermissionManageMembers,
			PermissionManageAPIKeys,
			PermissionUseChat,
			PermissionViewStats,
		},
		TenantRoleAdmin: {
			PermissionManageMembers,
			PermissionManageAPIKeys,
			PermissionUseChat,
			PermissionViewStats,
		},
		TenantRoleMember: {
			PermissionUseChat,
			PermissionViewStats,
		},
		TenantRoleViewer: {
			PermissionViewStats,
		},
	}

	for _, perm := range permissions[r] {
		if perm == action {
			return true
		}
	}
	return false
}

// TenantPermission представляет разрешение в tenant
type TenantPermission string

const (
	PermissionManageTenant  TenantPermission = "manage_tenant"   // Управление tenant (настройки, удаление)
	PermissionManageMembers TenantPermission = "manage_members"  // Управление участниками
	PermissionManageAPIKeys TenantPermission = "manage_api_keys" // Управление API ключами
	PermissionUseChat       TenantPermission = "use_chat"        // Использование Chat UI
	PermissionViewStats     TenantPermission = "view_stats"      // Просмотр статистики
)

// TenantWithMember представляет Tenant с информацией о роли пользователя
type TenantWithMember struct {
	*Tenant
	MemberRole TenantRole `json:"member_role"` // Роль текущего пользователя
}

