// Package redis provides Redis-based JWT management
// Version: v3.0.6+ - JWT Blacklist & Token Management
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// JWTService provides Redis-based JWT token management
type JWTService struct {
	client         *Client
	logger         *logrus.Logger
	sessionTTL     time.Duration // Default: 7 days
}

// NewJWTService creates a new JWT service
func NewJWTService(client *Client, logger *logrus.Logger) *JWTService {
	return &JWTService{
		client:     client,
		logger:     logger,
		sessionTTL: 7 * 24 * time.Hour, // 7 days default
	}
}

// BlacklistToken adds a token to blacklist
// ttl should match token expiration time
func (s *JWTService) BlacklistToken(ctx context.Context, tokenID string, ttl time.Duration) error {
	key := fmt.Sprintf("jwt_blacklist:%s", tokenID)
	
	if err := s.client.SetString(ctx, key, "revoked", ttl); err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}
	
	s.logger.WithFields(logrus.Fields{
		"token_id": tokenID,
		"ttl":      ttl,
	}).Info("Token blacklisted")
	
	return nil
}

// IsTokenBlacklisted checks if token is blacklisted
func (s *JWTService) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := fmt.Sprintf("jwt_blacklist:%s", tokenID)
	return s.client.Exists(ctx, key)
}

// BlacklistUserTokens blacklists all tokens for a user
// Use when user logs out from all devices or account is compromised
func (s *JWTService) BlacklistUserTokens(ctx context.Context, userID string, ttl time.Duration) error {
	key := fmt.Sprintf("jwt_user_blacklist:%s", userID)
	
	// Store timestamp when all tokens became invalid
	timestamp := time.Now().Unix()
	
	if err := s.client.SetString(ctx, key, fmt.Sprintf("%d", timestamp), ttl); err != nil {
		return fmt.Errorf("failed to blacklist user tokens: %w", err)
	}
	
	s.logger.WithFields(logrus.Fields{
		"user_id":   userID,
		"timestamp": timestamp,
	}).Info("All user tokens blacklisted")
	
	return nil
}

// AreUserTokensBlacklisted checks if all user tokens are blacklisted
// Returns blacklist timestamp if blacklisted
func (s *JWTService) AreUserTokensBlacklisted(ctx context.Context, userID string) (int64, error) {
	key := fmt.Sprintf("jwt_user_blacklist:%s", userID)
	
	timestamp, err := s.client.GetString(ctx, key)
	if err != nil {
		return 0, nil // Not blacklisted
	}
	
	var ts int64
	fmt.Sscanf(timestamp, "%d", &ts)
	return ts, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Refresh Token Management
// ─────────────────────────────────────────────────────────────────────────────

// RefreshToken represents a refresh token
type RefreshToken struct {
	TokenID   string    `json:"token_id"`
	UserID    string    `json:"user_id"`
	TenantID  string    `json:"tenant_id,omitempty"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address,omitempty"`
	UserAgent string    `json:"user_agent,omitempty"`
}

// SaveRefreshToken stores a refresh token
func (s *JWTService) SaveRefreshToken(ctx context.Context, token *RefreshToken, ttl time.Duration) error {
	if token.TokenID == "" {
		return fmt.Errorf("token_id is required")
	}
	
	key := fmt.Sprintf("refresh_token:%s", token.TokenID)
	
	if err := s.client.Set(ctx, key, token, ttl); err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}
	
	// Add to user's refresh tokens set
	if token.UserID != "" {
		userTokensKey := fmt.Sprintf("user_refresh_tokens:%s", token.UserID)
		if err := s.client.SAdd(ctx, userTokensKey, token.TokenID); err != nil {
			s.logger.WithError(err).Warn("Failed to add token to user set")
		}
		// Set expiration on the set
		s.client.Expire(ctx, userTokensKey, ttl)
	}
	
	s.logger.WithFields(logrus.Fields{
		"token_id": token.TokenID,
		"user_id":  token.UserID,
		"ttl":      ttl,
	}).Debug("Refresh token saved")
	
	return nil
}

// GetRefreshToken retrieves a refresh token
func (s *JWTService) GetRefreshToken(ctx context.Context, tokenID string) (*RefreshToken, error) {
	key := fmt.Sprintf("refresh_token:%s", tokenID)
	
	var token RefreshToken
	if err := s.client.Get(ctx, key, &token); err != nil {
		return nil, fmt.Errorf("refresh token not found: %w", err)
	}
	
	return &token, nil
}

// RevokeRefreshToken revokes a specific refresh token
func (s *JWTService) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	// Get token to clean up user set
	token, err := s.GetRefreshToken(ctx, tokenID)
	if err == nil && token.UserID != "" {
		userTokensKey := fmt.Sprintf("user_refresh_tokens:%s", token.UserID)
		s.client.SRem(ctx, userTokensKey, tokenID)
	}
	
	key := fmt.Sprintf("refresh_token:%s", tokenID)
	return s.client.Delete(ctx, key)
}

// RevokeUserRefreshTokens revokes all refresh tokens for a user
func (s *JWTService) RevokeUserRefreshTokens(ctx context.Context, userID string) error {
	userTokensKey := fmt.Sprintf("user_refresh_tokens:%s", userID)
	
	// Get all token IDs
	tokenIDs, err := s.client.SMembers(ctx, userTokensKey)
	if err != nil {
		return err
	}
	
	// Delete each token
	for _, tokenID := range tokenIDs {
		key := fmt.Sprintf("refresh_token:%s", tokenID)
		s.client.Delete(ctx, key)
	}
	
	// Delete the set
	return s.client.Delete(ctx, userTokensKey)
}

// GetUserRefreshTokens gets all active refresh tokens for a user
func (s *JWTService) GetUserRefreshTokens(ctx context.Context, userID string) ([]*RefreshToken, error) {
	userTokensKey := fmt.Sprintf("user_refresh_tokens:%s", userID)
	
	tokenIDs, err := s.client.SMembers(ctx, userTokensKey)
	if err != nil {
		return nil, err
	}
	
	tokens := make([]*RefreshToken, 0, len(tokenIDs))
	for _, tokenID := range tokenIDs {
		token, err := s.GetRefreshToken(ctx, tokenID)
		if err == nil {
			tokens = append(tokens, token)
		}
	}
	
	return tokens, nil
}

// CountUserRefreshTokens counts active refresh tokens for a user
func (s *JWTService) CountUserRefreshTokens(ctx context.Context, userID string) (int64, error) {
	userTokensKey := fmt.Sprintf("user_refresh_tokens:%s", userID)
	return s.client.SCard(ctx, userTokensKey)
}

// ─────────────────────────────────────────────────────────────────────────────
// JWT Session Storage (v3.0.6+: Active Sessions Tracking)
// ─────────────────────────────────────────────────────────────────────────────

// JWTSession represents an active JWT session
type JWTSession struct {
	SessionID    string                 `json:"session_id"`    // Unique session ID (from JWT jti claim)
	TokenID      string                 `json:"token_id"`      // JWT token ID (jti)
	UserID       string                 `json:"user_id"`
	Username     string                 `json:"username"`
	Email        string                 `json:"email,omitempty"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	Role         string                 `json:"role,omitempty"`
	Permissions  []string               `json:"permissions,omitempty"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	DeviceType   string                 `json:"device_type,omitempty"`   // web, mobile, desktop
	DeviceName   string                 `json:"device_name,omitempty"`   // Chrome, Safari, Mobile App
	Location     string                 `json:"location,omitempty"`      // City, Country
	IssuedAt     time.Time              `json:"issued_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	LastActivity time.Time              `json:"last_activity"`
	ActivityCount int64                 `json:"activity_count"`          // Number of requests
	Extra        map[string]interface{} `json:"extra,omitempty"`
}

// SaveJWTSession saves an active JWT session
// TTL defaults to 7 days (session_ttl) or can be custom
func (s *JWTService) SaveJWTSession(ctx context.Context, session *JWTSession, ttl ...time.Duration) error {
	if session.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if session.TokenID == "" {
		session.TokenID = session.SessionID
	}
	
	if session.IssuedAt.IsZero() {
		session.IssuedAt = time.Now()
	}
	session.LastActivity = time.Now()
	
	// Use provided TTL or default
	sessionTTL := s.sessionTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		sessionTTL = ttl[0]
	}
	
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = session.IssuedAt.Add(sessionTTL)
	}
	
	key := fmt.Sprintf("jwt_session:%s", session.SessionID)
	
	if err := s.client.Set(ctx, key, session, sessionTTL); err != nil {
		return fmt.Errorf("failed to save JWT session: %w", err)
	}
	
	// Add to user's sessions set for tracking
	if session.UserID != "" {
		userSessionsKey := fmt.Sprintf("user_jwt_sessions:%s", session.UserID)
		if err := s.client.SAdd(ctx, userSessionsKey, session.SessionID); err != nil {
			s.logger.WithError(err).Warn("Failed to add JWT session to user set")
		}
		// Set expiration on the set
		s.client.Expire(ctx, userSessionsKey, sessionTTL)
	}
	
	s.logger.WithFields(logrus.Fields{
		"session_id": session.SessionID,
		"user_id":    session.UserID,
		"username":   session.Username,
		"ip":         session.IPAddress,
		"device":     session.DeviceName,
		"ttl":        sessionTTL,
	}).Debug("JWT session saved")
	
	return nil
}

// GetJWTSession retrieves a JWT session by ID
func (s *JWTService) GetJWTSession(ctx context.Context, sessionID string) (*JWTSession, error) {
	key := fmt.Sprintf("jwt_session:%s", sessionID)
	
	var session JWTSession
	if err := s.client.Get(ctx, key, &session); err != nil {
		return nil, fmt.Errorf("JWT session not found: %w", err)
	}
	
	return &session, nil
}

// UpdateJWTSessionActivity updates last activity timestamp and counter
func (s *JWTService) UpdateJWTSessionActivity(ctx context.Context, sessionID string) error {
	session, err := s.GetJWTSession(ctx, sessionID)
	if err != nil {
		return err
	}
	
	session.LastActivity = time.Now()
	session.ActivityCount++
	
	key := fmt.Sprintf("jwt_session:%s", sessionID)
	
	// Get current TTL to preserve it
	ttl, err := s.client.TTL(ctx, key)
	if err != nil || ttl <= 0 {
		return fmt.Errorf("session not found or expired")
	}
	
	return s.client.Set(ctx, key, session, ttl)
}

// RefreshJWTSession extends session TTL
func (s *JWTService) RefreshJWTSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	key := fmt.Sprintf("jwt_session:%s", sessionID)
	
	// Check if session exists
	exists, err := s.client.Exists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("JWT session not found")
	}
	
	// Update session expiration
	session, err := s.GetJWTSession(ctx, sessionID)
	if err != nil {
		return err
	}
	
	session.ExpiresAt = time.Now().Add(ttl)
	
	// Extend expiration
	if err := s.client.Expire(ctx, key, ttl); err != nil {
		return err
	}
	
	// Update session data
	return s.client.Set(ctx, key, session, ttl)
}

// RevokeJWTSession revokes a specific JWT session (logout)
func (s *JWTService) RevokeJWTSession(ctx context.Context, sessionID string) error {
	// Get session to clean up user set and blacklist token
	session, err := s.GetJWTSession(ctx, sessionID)
	if err == nil {
		// Remove from user's sessions set
		if session.UserID != "" {
			userSessionsKey := fmt.Sprintf("user_jwt_sessions:%s", session.UserID)
			s.client.SRem(ctx, userSessionsKey, sessionID)
		}
		
		// Blacklist the token
		if session.TokenID != "" {
			ttl := time.Until(session.ExpiresAt)
			if ttl > 0 {
				s.BlacklistToken(ctx, session.TokenID, ttl)
			}
		}
	}
	
	key := fmt.Sprintf("jwt_session:%s", sessionID)
	return s.client.Delete(ctx, key)
}

// RevokeUserJWTSessions revokes all JWT sessions for a user
func (s *JWTService) RevokeUserJWTSessions(ctx context.Context, userID string) error {
	userSessionsKey := fmt.Sprintf("user_jwt_sessions:%s", userID)
	
	// Get all session IDs
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey)
	if err != nil {
		return err
	}
	
	s.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"sessions_count": len(sessionIDs),
	}).Info("Revoking all user JWT sessions")
	
	// Delete each session
	for _, sessionID := range sessionIDs {
		s.RevokeJWTSession(ctx, sessionID)
	}
	
	// Delete the set
	s.client.Delete(ctx, userSessionsKey)
	
	// Also blacklist all user tokens
	return s.BlacklistUserTokens(ctx, userID, s.sessionTTL)
}

// GetUserJWTSessions gets all active JWT sessions for a user
func (s *JWTService) GetUserJWTSessions(ctx context.Context, userID string) ([]*JWTSession, error) {
	userSessionsKey := fmt.Sprintf("user_jwt_sessions:%s", userID)
	
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey)
	if err != nil {
		return nil, err
	}
	
	sessions := make([]*JWTSession, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		session, err := s.GetJWTSession(ctx, sessionID)
		if err == nil {
			sessions = append(sessions, session)
		} else {
			// Clean up stale session ID from set
			s.client.SRem(ctx, userSessionsKey, sessionID)
		}
	}
	
	return sessions, nil
}

// CountUserJWTSessions counts active JWT sessions for a user
func (s *JWTService) CountUserJWTSessions(ctx context.Context, userID string) (int64, error) {
	userSessionsKey := fmt.Sprintf("user_jwt_sessions:%s", userID)
	return s.client.SCard(ctx, userSessionsKey)
}

// IsJWTSessionValid checks if a JWT session is valid (exists and not expired)
func (s *JWTService) IsJWTSessionValid(ctx context.Context, sessionID string) (bool, error) {
	key := fmt.Sprintf("jwt_session:%s", sessionID)
	exists, err := s.client.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	
	if !exists {
		return false, nil
	}
	
	// Check if token is blacklisted
	session, err := s.GetJWTSession(ctx, sessionID)
	if err != nil {
		return false, err
	}
	
	isBlacklisted, err := s.IsTokenBlacklisted(ctx, session.TokenID)
	if err != nil {
		return false, err
	}
	
	return !isBlacklisted, nil
}

// CleanupExpiredSessions manually cleans up expired sessions (usually done by Redis TTL)
// This is a maintenance function, Redis TTL should handle expiration automatically
func (s *JWTService) CleanupExpiredSessions(ctx context.Context) (int, error) {
	pattern := "jwt_session:*"
	keys, err := s.client.Keys(ctx, pattern)
	if err != nil {
		return 0, err
	}
	
	cleaned := 0
	now := time.Now()
	
	for _, key := range keys {
		var session JWTSession
		if err := s.client.Get(ctx, key, &session); err == nil {
			if now.After(session.ExpiresAt) {
				sessionID := session.SessionID
				if err := s.RevokeJWTSession(ctx, sessionID); err == nil {
					cleaned++
				}
			}
		}
	}
	
	s.logger.WithField("cleaned", cleaned).Info("Cleaned up expired JWT sessions")
	return cleaned, nil
}

// GetJWTSessionsStats returns statistics about JWT sessions
func (s *JWTService) GetJWTSessionsStats(ctx context.Context) (map[string]interface{}, error) {
	pattern := "jwt_session:*"
	keys, err := s.client.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}
	
	stats := map[string]interface{}{
		"total_sessions": len(keys),
		"by_device":      make(map[string]int),
		"by_location":    make(map[string]int),
		"active_users":   make(map[string]bool),
	}
	
	for _, key := range keys {
		var session JWTSession
		if err := s.client.Get(ctx, key, &session); err == nil {
			// Count by device
			if session.DeviceType != "" {
				deviceStats := stats["by_device"].(map[string]int)
				deviceStats[session.DeviceType]++
			}
			
			// Count by location
			if session.Location != "" {
				locationStats := stats["by_location"].(map[string]int)
				locationStats[session.Location]++
			}
			
			// Track unique users
			if session.UserID != "" {
				activeUsers := stats["active_users"].(map[string]bool)
				activeUsers[session.UserID] = true
			}
		}
	}
	
	// Convert active users set to count
	stats["unique_users"] = len(stats["active_users"].(map[string]bool))
	delete(stats, "active_users")
	
	return stats, nil
}

