// Package models provides data models for User Management and Multi-Tenancy
package models

import (
	"time"
)

// User представляет пользователя системы
type User struct {
	// Основная информация
	ID       string `json:"id" db:"id"`               // Уникальный ID (UUID)
	Username string `json:"username" db:"username"`   // Уникальное имя пользователя
	Email    string `json:"email" db:"email"`         // Email (уникальный)
	FullName string `json:"full_name" db:"full_name"` // Полное имя

	// Аутентификация
	PasswordHash string `json:"-" db:"password_hash"` // Bcrypt hash пароля (не возвращается в JSON)

	// OIDC аутентификация (Version 1.11.1+: Keycloak SSO Integration)
	AuthProvider string  `json:"auth_provider" db:"auth_provider"` // Authentication provider: 'local', 'oidc', 'ldap'
	OIDCSubject  *string `json:"oidc_subject,omitempty" db:"oidc_subject"`   // OIDC 'sub' claim (unique identifier)
	OIDCIssuer   *string `json:"oidc_issuer,omitempty" db:"oidc_issuer"`     // OIDC issuer URL
	
	// LDAP аутентификация (Version 1.11.3+: LDAP/AD Integration)
	LDAPDN *string `json:"ldap_dn,omitempty" db:"ldap_dn"` // LDAP Distinguished Name

	// Статус
	Status     UserStatus `json:"status" db:"status"`       // Статус пользователя
	IsAdmin    bool       `json:"is_admin" db:"is_admin"`   // Глобальный администратор
	IsActive   bool       `json:"is_active" db:"is_active"` // Активен ли аккаунт
	Verified   bool       `json:"verified" db:"verified"`   // Email верифицирован
	VerifiedAt *time.Time `json:"verified_at,omitempty" db:"verified_at"`

	// Временные метки
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	LastLogin *time.Time `json:"last_login,omitempty" db:"last_login"`

	// Preferences
	Preferences UserPreferences `json:"preferences" db:"preferences"` // Хранится как JSONB

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" db:"metadata"` // Дополнительные данные
}

// UserStatus представляет статус пользователя
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"    // Активен
	UserStatusInactive  UserStatus = "inactive"  // Неактивен
	UserStatusSuspended UserStatus = "suspended" // Приостановлен
	UserStatusDeleted   UserStatus = "deleted"   // Удален (soft delete)
)

// AuthProvider представляет провайдера аутентификации (Version 1.11.1+)
const (
	AuthProviderLocal string = "local" // Локальная аутентификация (username/password)
	AuthProviderOIDC  string = "oidc"  // OpenID Connect (Keycloak, Google, Azure, etc.)
	AuthProviderLDAP  string = "ldap"  // LDAP/Active Directory
)

// UserPreferences содержит пользовательские настройки
type UserPreferences struct {
	Theme    string `json:"theme"`    // light, dark, auto
	Language string `json:"language"` // ru, en
	Timezone string `json:"timezone"` // Europe/Moscow
}

// UserPublic представляет публичную информацию о пользователе (без sensitive данных)
type UserPublic struct {
	ID        string     `json:"id"`
	Username  string     `json:"username"`
	FullName  string     `json:"full_name"`
	IsAdmin   bool       `json:"is_admin"`
	Status    UserStatus `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
}

// ToPublic преобразует User в UserPublic
func (u *User) ToPublic() *UserPublic {
	return &UserPublic{
		ID:        u.ID,
		Username:  u.Username,
		FullName:  u.FullName,
		IsAdmin:   u.IsAdmin,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
	}
}

// UserFilters представляет фильтры для списка пользователей
type UserFilters struct {
	Status   *UserStatus `json:"status,omitempty"`
	IsAdmin  *bool       `json:"is_admin,omitempty"`
	IsActive *bool       `json:"is_active,omitempty"`
	Search   string      `json:"search,omitempty"` // Поиск по username, email, full_name

	// Pagination
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

