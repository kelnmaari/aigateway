// Package oidc provides OpenID Connect (OIDC) authentication support
// for Keycloak and other OIDC-compatible identity providers
//
// Version: 1.11.1+ (Enterprise Suite)
package oidc

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"aigateway/internal/config"
)

// OIDCProvider предоставляет OIDC authentication functionality
type OIDCProvider struct {
	config       *config.OIDCConfig
	provider     *oidc.Provider // OIDC discovery provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config *oauth2.Config
	logger       *logrus.Logger
}

// NewOIDCProvider creates a new OIDC provider with automatic endpoint discovery
func NewOIDCProvider(ctx context.Context, cfg *config.OIDCConfig, logger *logrus.Logger) (*OIDCProvider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("OIDC is not enabled in configuration")
	}

	if cfg.Issuer == "" {
		return nil, fmt.Errorf("OIDC issuer URL is required")
	}

	if cfg.ClientID == "" {
		return nil, fmt.Errorf("OIDC client ID is required")
	}

	if cfg.ClientSecret == "" {
		return nil, fmt.Errorf("OIDC client secret is required")
	}

	if cfg.RedirectURI == "" {
		return nil, fmt.Errorf("OIDC redirect URI is required")
	}

	// Discover OIDC endpoints from issuer URL
	// This will fetch .well-known/openid-configuration automatically
	logger.WithField("issuer", cfg.Issuer).Info("Discovering OIDC endpoints...")

	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC endpoints: %w", err)
	}

	logger.Info("OIDC endpoints discovered successfully")

	// Configure ID token verifier
	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.ClientID,
		// SkipClientIDCheck: false, // Always verify client ID
		// SkipExpiryCheck: false,   // Always verify expiration
	})

	// Prepare scopes (default to openid, profile, email if not specified)
	scopes := cfg.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}

	// Ensure "openid" scope is always present (required for OIDC)
	hasOpenID := false
	for _, scope := range scopes {
		if scope == oidc.ScopeOpenID {
			hasOpenID = true
			break
		}
	}
	if !hasOpenID {
		scopes = append([]string{oidc.ScopeOpenID}, scopes...)
	}

	// Configure OAuth2
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURI,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	logger.WithFields(logrus.Fields{
		"issuer":       cfg.Issuer,
		"client_id":    cfg.ClientID,
		"redirect_uri": cfg.RedirectURI,
		"scopes":       scopes,
	}).Info("OIDC provider initialized successfully")

	return &OIDCProvider{
		config:       cfg,
		provider:     provider,
		verifier:     verifier,
		oauth2Config: oauth2Config,
		logger:       logger,
	}, nil
}

// GetAuthURL generates the OIDC authorization URL with state parameter
// state is used for CSRF protection and should be stored in the user's session
func (p *OIDCProvider) GetAuthURL(state string) string {
	// oauth2.AccessTypeOffline requests a refresh token
	// oauth2.SetAuthURLParam can be used for additional parameters
	return p.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges the authorization code for OAuth2 tokens
// Returns the token set including access token, ID token, and refresh token
func (p *OIDCProvider) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	// Set timeout for token exchange
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	token, err := p.oauth2Config.Exchange(ctx, code)
	if err != nil {
		p.logger.WithError(err).Error("Failed to exchange authorization code for tokens")
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"expires_in": token.Expiry.Sub(time.Now()).String(),
		"token_type": token.TokenType,
	}).Debug("Successfully exchanged authorization code for tokens")

	return token, nil
}

// VerifyIDToken verifies the ID token signature, issuer, audience, and expiration
// Returns the verified ID token that can be used to extract claims
func (p *OIDCProvider) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	// Set timeout for token verification
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	idToken, err := p.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		p.logger.WithError(err).Error("Failed to verify ID token")
		return nil, fmt.Errorf("ID token verification failed: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"subject":   idToken.Subject,
		"issuer":    idToken.Issuer,
		"audience":  idToken.Audience,
		"issued_at": idToken.IssuedAt.Format(time.RFC3339),
		"expires":   idToken.Expiry.Format(time.RFC3339),
	}).Debug("ID token verified successfully")

	return idToken, nil
}

// ExtractIDToken extracts the ID token from OAuth2 token
// ID token is stored in the "id_token" extra field
func (p *OIDCProvider) ExtractIDToken(token *oauth2.Token) (string, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return "", fmt.Errorf("no id_token in OAuth2 token response")
	}
	return rawIDToken, nil
}

// RefreshToken refreshes an expired access token using the refresh token
// Returns a new token set
func (p *OIDCProvider) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	// Create a token with only the refresh token
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	// Use the TokenSource to refresh the token
	tokenSource := p.oauth2Config.TokenSource(ctx, token)

	newToken, err := tokenSource.Token()
	if err != nil {
		p.logger.WithError(err).Error("Failed to refresh token")
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	p.logger.Debug("Token refreshed successfully")
	return newToken, nil
}

// GetUserInfo fetches user information from the OIDC UserInfo endpoint
// This is optional and provides additional user claims not in the ID token
func (p *OIDCProvider) GetUserInfo(ctx context.Context, tokenSource oauth2.TokenSource) (*oidc.UserInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	userInfo, err := p.provider.UserInfo(ctx, tokenSource)
	if err != nil {
		p.logger.WithError(err).Error("Failed to fetch user info")
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}

	p.logger.WithField("subject", userInfo.Subject).Debug("User info fetched successfully")
	return userInfo, nil
}

// GetConfig returns the OIDC configuration
func (p *OIDCProvider) GetConfig() *config.OIDCConfig {
	return p.config
}

// GetEndpoint returns the OAuth2 endpoint configuration
func (p *OIDCProvider) GetEndpoint() oauth2.Endpoint {
	return p.oauth2Config.Endpoint
}


