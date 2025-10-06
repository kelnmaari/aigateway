// Package jwt provides JWT token generation and validation
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// Manager управляет JWT токенами
type Manager struct {
	secretKey            []byte
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
	logger               *logrus.Logger
	tokenBlacklist       map[string]time.Time // In-memory blacklist (TODO: migrate to Redis)
}

// Config конфигурация JWT Manager
type Config struct {
	SecretKey            string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	Logger               *logrus.Logger
}

// NewManager создает новый JWT Manager
func NewManager(cfg Config) (*Manager, error) {
	if cfg.SecretKey == "" {
		return nil, fmt.Errorf("JWT secret key is required")
	}

	if cfg.AccessTokenDuration == 0 {
		cfg.AccessTokenDuration = 15 * time.Minute // default
	}

	if cfg.RefreshTokenDuration == 0 {
		cfg.RefreshTokenDuration = 7 * 24 * time.Hour // default 7 days
	}

	if cfg.Logger == nil {
		cfg.Logger = logrus.New()
	}

	return &Manager{
		secretKey:            []byte(cfg.SecretKey),
		accessTokenDuration:  cfg.AccessTokenDuration,
		refreshTokenDuration: cfg.RefreshTokenDuration,
		logger:               cfg.Logger,
		tokenBlacklist:       make(map[string]time.Time),
	}, nil
}

// GenerateTokenPair генерирует пару access и refresh токенов
func (m *Manager) GenerateTokenPair(userID, username, email string, tenantIDs []string) (*TokenPair, error) {
	// Generate access token
	now := time.Now()
	expiresAt := now.Add(m.accessTokenDuration)

	claims := &Claims{
		UserID:    userID,
		Username:  username,
		Email:     email,
		TenantIDs: tenantIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "ollama-openai-proxy",
			Subject:   userID,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessTokenString, err := accessToken.SignedString(m.secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshClaims := &RefreshClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "ollama-openai-proxy",
			Subject:   userID,
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(m.secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"username":   username,
		"expires_at": expiresAt,
	}).Debug("Generated token pair")

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}, nil
}

// ValidateAccessToken валидирует access token и возвращает claims
func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	// Check blacklist
	if expiresAt, blacklisted := m.tokenBlacklist[tokenString]; blacklisted {
		if time.Now().Before(expiresAt) {
			return nil, fmt.Errorf("token is blacklisted")
		}
		// Token expired, remove from blacklist
		delete(m.tokenBlacklist, tokenString)
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// ValidateRefreshToken валидирует refresh token и возвращает claims
func (m *Manager) ValidateRefreshToken(tokenString string) (*RefreshClaims, error) {
	// Check blacklist
	if expiresAt, blacklisted := m.tokenBlacklist[tokenString]; blacklisted {
		if time.Now().Before(expiresAt) {
			return nil, fmt.Errorf("token is blacklisted")
		}
		delete(m.tokenBlacklist, tokenString)
	}

	token, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid refresh token claims")
	}

	return claims, nil
}

// RevokeToken добавляет токен в blacklist
func (m *Manager) RevokeToken(tokenString string, expiresAt time.Time) {
	m.tokenBlacklist[tokenString] = expiresAt

	m.logger.WithFields(logrus.Fields{
		"expires_at": expiresAt,
	}).Debug("Token revoked")
}

// CleanupBlacklist удаляет expired токены из blacklist
// Должен вызываться периодически (например, каждый час)
func (m *Manager) CleanupBlacklist() int {
	now := time.Now()
	count := 0

	for token, expiresAt := range m.tokenBlacklist {
		if now.After(expiresAt) {
			delete(m.tokenBlacklist, token)
			count++
		}
	}

	if count > 0 {
		m.logger.WithField("count", count).Debug("Cleaned up expired tokens from blacklist")
	}

	return count
}

// GetBlacklistSize возвращает размер blacklist (для мониторинга)
func (m *Manager) GetBlacklistSize() int {
	return len(m.tokenBlacklist)
}
