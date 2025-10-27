// Package models provides data models for Invitation System (AUTH-03, v2.2.0)
package models

import (
	"time"
)

// Invitation представляет пригласительную ссылку для регистрации
type Invitation struct {
	// Основная информация
	ID    string `json:"id" db:"id"`       // Уникальный ID
	Token string `json:"token" db:"token"` // Уникальный токен для ссылки (UUID)

	// Creation metadata
	CreatedByUserID string    `json:"created_by_user_id" db:"created_by_user_id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`

	// Constraints
	Email     *string    `json:"email,omitempty" db:"email"`           // Опциональная привязка к email
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"` // Опциональный срок действия
	MaxUses   int        `json:"max_uses" db:"max_uses"`               // Максимальное количество использований (default: 1)

	// Usage tracking
	CurrentUses    int        `json:"current_uses" db:"current_uses"`             // Текущее количество использований
	UsedAt         *time.Time `json:"used_at,omitempty" db:"used_at"`             // Время первого использования
	UsedByUserID   *string    `json:"used_by_user_id,omitempty" db:"used_by_user_id"` // ID пользователя, который использовал приглашение

	// Revocation
	RevokedAt       *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`             // Время отзыва
	RevokedByUserID *string    `json:"revoked_by_user_id,omitempty" db:"revoked_by_user_id"` // ID пользователя, который отозвал приглашение
	RevokeReason    *string    `json:"revoke_reason,omitempty" db:"revoke_reason"`       // Причина отзыва
}

// InvitationWithUsers расширенная информация о приглашении с данными пользователей
type InvitationWithUsers struct {
	Invitation
	
	// Расширенная информация о пользователях
	CreatedBy *UserInfo `json:"created_by,omitempty"` // Кто создал приглашение
	UsedBy    *UserInfo `json:"used_by,omitempty"`    // Кто использовал приглашение
	RevokedBy *UserInfo `json:"revoked_by,omitempty"` // Кто отозвал приглашение
}

// UserInfo базовая информация о пользователе для invitations
type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"full_name,omitempty"`
}

// InvitationStatus представляет статус приглашения
type InvitationStatus string

const (
	InvitationStatusActive   InvitationStatus = "active"   // Активное приглашение
	InvitationStatusUsed     InvitationStatus = "used"     // Использовано
	InvitationStatusExpired  InvitationStatus = "expired"  // Срок действия истек
	InvitationStatusRevoked  InvitationStatus = "revoked"  // Отозвано администратором
)

// GetStatus возвращает текущий статус приглашения
func (i *Invitation) GetStatus() InvitationStatus {
	// Проверка отзыва
	if i.RevokedAt != nil {
		return InvitationStatusRevoked
	}

	// Проверка срока действия
	if i.ExpiresAt != nil && time.Now().After(*i.ExpiresAt) {
		return InvitationStatusExpired
	}

	// Проверка использования
	if i.CurrentUses >= i.MaxUses {
		return InvitationStatusUsed
	}

	return InvitationStatusActive
}

// IsValid проверяет, можно ли использовать приглашение
func (i *Invitation) IsValid() bool {
	return i.GetStatus() == InvitationStatusActive
}

// CanBeUsedByEmail проверяет, может ли приглашение быть использовано с указанным email
func (i *Invitation) CanBeUsedByEmail(email string) bool {
	// Если email не указан в приглашении, то приглашение может быть использовано любым
	if i.Email == nil {
		return true
	}

	// Если email указан, проверяем совпадение
	return *i.Email == email
}

// CreateInvitationRequest представляет запрос на создание приглашения
type CreateInvitationRequest struct {
	Email       *string    `json:"email,omitempty"`        // Опциональная привязка к email
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`   // Опциональный срок действия
	MaxUses     int        `json:"max_uses"`               // Максимальное количество использований
}

// InvitationListFilter представляет фильтр для списка приглашений
type InvitationListFilter struct {
	Status         *InvitationStatus `json:"status,omitempty"`          // Фильтр по статусу
	CreatedByUserID *string           `json:"created_by_user_id,omitempty"` // Фильтр по создателю
	Email          *string           `json:"email,omitempty"`           // Фильтр по email
	Limit          int               `json:"limit"`                     // Лимит результатов
	Offset         int               `json:"offset"`                    // Смещение для пагинации
}

// InvitationStats представляет статистику по приглашениям
type InvitationStats struct {
	TotalCreated int `json:"total_created"` // Всего создано приглашений
	Active       int `json:"active"`        // Активных приглашений
	Used         int `json:"used"`          // Использованных приглашений
	Expired      int `json:"expired"`       // Истекших приглашений
	Revoked      int `json:"revoked"`       // Отозванных приглашений
}

// InvitationLink генерирует полную ссылку для приглашения
func (i *Invitation) InvitationLink(baseURL string) string {
	return baseURL + "/register.html?invite=" + i.Token
}

