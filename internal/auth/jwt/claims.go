// Package jwt provides JWT token generation and validation for user authentication
package jwt

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims представляет структуру JWT claims для пользователя
type Claims struct {
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	Email     string   `json:"email"`
	TenantIDs []string `json:"tenant_ids"` // Все доступные tenants
	jwt.RegisteredClaims
}

// TokenPair представляет пару access и refresh токенов
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"` // "Bearer"
}

// RefreshClaims представляет claims для refresh токена
type RefreshClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}
