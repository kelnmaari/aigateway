// Package redis provides Redis-based session management
// Version: v3.0.6+ - Session Storage
package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// SessionService provides Redis-based session storage
type SessionService struct {
	client *Client
	logger *logrus.Logger
}

// NewSessionService creates a new session service
func NewSessionService(client *Client, logger *logrus.Logger) *SessionService {
	return &SessionService{
		client: client,
		logger: logger,
	}
}

// Session represents a user session
type Session struct {
	SessionID string                 `json:"session_id"`
	UserID    string                 `json:"user_id"`
	Username  string                 `json:"username"`
	TenantID  string                 `json:"tenant_id,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	ExpiresAt time.Time              `json:"expires_at"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// CreateSession creates a new session
func (s *SessionService) CreateSession(ctx context.Context, session *Session, ttl time.Duration) error {
	if session.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	session.ExpiresAt = session.CreatedAt.Add(ttl)
	
	key := fmt.Sprintf("session:%s", session.SessionID)
	
	if err := s.client.Set(ctx, key, session, ttl); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	
	// Add to user's session set for tracking
	if session.UserID != "" {
		userSessionsKey := fmt.Sprintf("user_sessions:%s", session.UserID)
		if err := s.client.SAdd(ctx, userSessionsKey, session.SessionID); err != nil {
			s.logger.WithError(err).Warn("Failed to add session to user set")
		}
	}
	
	s.logger.WithFields(logrus.Fields{
		"session_id": session.SessionID,
		"user_id":    session.UserID,
		"ttl":        ttl,
	}).Debug("Session created")
	
	return nil
}

// GetSession retrieves a session by ID
func (s *SessionService) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	
	var session Session
	if err := s.client.Get(ctx, key, &session); err != nil {
		return nil, fmt.Errorf("session not found: %w", err)
	}
	
	return &session, nil
}

// UpdateSession updates an existing session
func (s *SessionService) UpdateSession(ctx context.Context, session *Session) error {
	key := fmt.Sprintf("session:%s", session.SessionID)
	
	// Get current TTL
	ttl, err := s.client.TTL(ctx, key)
	if err != nil || ttl <= 0 {
		return fmt.Errorf("session not found or expired")
	}
	
	return s.client.Set(ctx, key, session, ttl)
}

// RefreshSession extends session TTL
func (s *SessionService) RefreshSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	
	// Check if session exists
	exists, err := s.client.Exists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("session not found")
	}
	
	// Extend expiration
	return s.client.Expire(ctx, key, ttl)
}

// DeleteSession deletes a session
func (s *SessionService) DeleteSession(ctx context.Context, sessionID string) error {
	// Get session to clean up user set
	session, err := s.GetSession(ctx, sessionID)
	if err == nil && session.UserID != "" {
		userSessionsKey := fmt.Sprintf("user_sessions:%s", session.UserID)
		s.client.SRem(ctx, userSessionsKey, sessionID)
	}
	
	key := fmt.Sprintf("session:%s", sessionID)
	return s.client.Delete(ctx, key)
}

// DeleteUserSessions deletes all sessions for a user
func (s *SessionService) DeleteUserSessions(ctx context.Context, userID string) error {
	userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)
	
	// Get all session IDs
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey)
	if err != nil {
		return err
	}
	
	// Delete each session
	for _, sessionID := range sessionIDs {
		key := fmt.Sprintf("session:%s", sessionID)
		s.client.Delete(ctx, key)
	}
	
	// Delete the set
	return s.client.Delete(ctx, userSessionsKey)
}

// GetUserSessions gets all active sessions for a user
func (s *SessionService) GetUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)
	
	sessionIDs, err := s.client.SMembers(ctx, userSessionsKey)
	if err != nil {
		return nil, err
	}
	
	sessions := make([]*Session, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		session, err := s.GetSession(ctx, sessionID)
		if err == nil {
			sessions = append(sessions, session)
		}
	}
	
	return sessions, nil
}

// CountActiveSessions counts active sessions for a user
func (s *SessionService) CountActiveSessions(ctx context.Context, userID string) (int64, error) {
	userSessionsKey := fmt.Sprintf("user_sessions:%s", userID)
	return s.client.SCard(ctx, userSessionsKey)
}

// ─────────────────────────────────────────────────────────────────────────────
// OIDC State Management
// ─────────────────────────────────────────────────────────────────────────────

// OIDCState represents OIDC flow state
type OIDCState struct {
	State        string    `json:"state"`
	Nonce        string    `json:"nonce"`
	RedirectURI  string    `json:"redirect_uri"`
	CreatedAt    time.Time `json:"created_at"`
	CodeVerifier string    `json:"code_verifier,omitempty"` // PKCE
}

// SaveOIDCState saves OIDC state for validation
func (s *SessionService) SaveOIDCState(ctx context.Context, state *OIDCState, ttl time.Duration) error {
	key := fmt.Sprintf("oidc_state:%s", state.State)
	
	if state.CreatedAt.IsZero() {
		state.CreatedAt = time.Now()
	}
	
	return s.client.Set(ctx, key, state, ttl)
}

// GetOIDCState retrieves OIDC state
func (s *SessionService) GetOIDCState(ctx context.Context, state string) (*OIDCState, error) {
	key := fmt.Sprintf("oidc_state:%s", state)
	
	var oidcState OIDCState
	if err := s.client.Get(ctx, key, &oidcState); err != nil {
		return nil, fmt.Errorf("OIDC state not found: %w", err)
	}
	
	return &oidcState, nil
}

// DeleteOIDCState deletes OIDC state (after successful login)
func (s *SessionService) DeleteOIDCState(ctx context.Context, state string) error {
	key := fmt.Sprintf("oidc_state:%s", state)
	return s.client.Delete(ctx, key)
}

