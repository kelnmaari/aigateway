// Package handlers provides OIDC authentication HTTP handlers
// Version: 1.11.1+ (Enterprise Suite - Keycloak SSO Integration)
package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/auth/jwt"
	oidcauth "ollama-openai-proxy/internal/auth/oidc"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// OIDCHandler обрабатывает OIDC authentication flow
type OIDCHandler struct {
	config            *config.Config
	provider          *oidcauth.OIDCProvider
	db                storage.Database
	jwtMgr            *jwt.Manager
	tenantProvisioner *oidcauth.TenantProvisioner // Version 1.11.2+: Auto-tenant provisioning
	logger            *logrus.Logger
}

// NewOIDCHandler создает новый OIDC handler
func NewOIDCHandler(cfg *config.Config, provider *oidcauth.OIDCProvider, db storage.Database, jwtMgr *jwt.Manager, logger *logrus.Logger) *OIDCHandler {
	var tenantProvisioner *oidcauth.TenantProvisioner
	
	// Initialize tenant provisioner if enabled
	if cfg.Auth.OIDC.TenantProvisioning.Enabled {
		tenantProvisioner = oidcauth.NewTenantProvisioner(db, &cfg.Auth.OIDC.TenantProvisioning, logger)
		logger.Info("Tenant provisioner initialized for OIDC")
	}
	
	return &OIDCHandler{
		config:            cfg,
		provider:          provider,
		db:                db,
		jwtMgr:            jwtMgr,
		tenantProvisioner: tenantProvisioner,
		logger:            logger,
	}
}

// HandleLogin initiate OIDC login flow
// GET /auth/oidc/login
func (h *OIDCHandler) HandleLogin(c *gin.Context) {
	// Check if OIDC is enabled
	if !h.config.Auth.OIDC.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "OIDC authentication is not enabled",
		})
		return
	}

	// Generate random state for CSRF protection
	state, err := generateRandomState()
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate OIDC state")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to initiate OIDC login",
		})
		return
	}

	// Store state in session
	session := sessions.Default(c)
	session.Set("oidc_state", state)
	session.Set("oidc_created_at", time.Now().Unix())
	
	// Save redirect URL from query parameter (optional)
	if redirectURL := c.Query("redirect_url"); redirectURL != "" {
		session.Set("oidc_redirect_url", redirectURL)
	}
	
	if err := session.Save(); err != nil {
		h.logger.WithError(err).Error("Failed to save OIDC state in session")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to save session",
		})
		return
	}

	// Generate authorization URL
	authURL := h.provider.GetAuthURL(state)

	h.logger.WithFields(logrus.Fields{
		"state":    state[:8] + "...", // Log only first 8 chars for privacy
		"auth_url": authURL,
	}).Info("OIDC login initiated, redirecting to provider")

	// Redirect user to OIDC provider
	c.Redirect(http.StatusFound, authURL)
}

// HandleCallback handles OIDC callback after user authentication
// GET /auth/oidc/callback
func (h *OIDCHandler) HandleCallback(c *gin.Context) {
	// Get session
	session := sessions.Default(c)
	
	// Validate state parameter (CSRF protection)
	savedState := session.Get("oidc_state")
	if savedState == nil {
		h.logger.Error("No OIDC state found in session")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid session, please try logging in again",
		})
		return
	}

	receivedState := c.Query("state")
	if receivedState == "" || receivedState != savedState.(string) {
		h.logger.WithFields(logrus.Fields{
			"saved_state":    savedState.(string)[:8] + "...",
			"received_state": receivedState,
		}).Error("OIDC state mismatch (CSRF attempt?)")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid state parameter",
		})
		return
	}

	// Check session expiry (10 minutes)
	createdAt := session.Get("oidc_created_at")
	if createdAt != nil {
		if time.Since(time.Unix(createdAt.(int64), 0)) > h.config.Auth.OIDC.SessionTTL {
			h.logger.Error("OIDC session expired")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Session expired, please try logging in again",
			})
			return
		}
	}

	// Check for error from OIDC provider
	if errorParam := c.Query("error"); errorParam != "" {
		errorDesc := c.Query("error_description")
		h.logger.WithFields(logrus.Fields{
			"error":       errorParam,
			"description": errorDesc,
		}).Error("OIDC provider returned error")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Authentication failed",
			"error_description": errorDesc,
		})
		return
	}

	// Get authorization code
	code := c.Query("code")
	if code == "" {
		h.logger.Error("No authorization code in OIDC callback")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing authorization code",
		})
		return
	}

	// Exchange authorization code for tokens
	ctx := c.Request.Context()
	oauth2Token, err := h.provider.ExchangeCode(ctx, code)
	if err != nil {
		h.logger.WithError(err).Error("Failed to exchange authorization code")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to exchange authorization code",
		})
		return
	}

	// Extract ID token
	rawIDToken, err := h.provider.ExtractIDToken(oauth2Token)
	if err != nil {
		h.logger.WithError(err).Error("Failed to extract ID token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "No ID token in response",
		})
		return
	}

	// Verify ID token
	idToken, err := h.provider.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		h.logger.WithError(err).Error("Failed to verify ID token")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid ID token",
		})
		return
	}

	// Extract claims from ID token
	var claims oidcauth.KeycloakClaims
	if err := idToken.Claims(&claims); err != nil {
		h.logger.WithError(err).Error("Failed to parse ID token claims")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse user claims",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"subject":  claims.Subject,
		"email":    claims.Email,
		"username": claims.PreferredUsername,
	}).Info("Successfully verified OIDC ID token")

	// Provision or update user
	user, err := h.provisionUser(ctx, idToken.Issuer, &claims)
	if err != nil {
		h.logger.WithError(err).Error("Failed to provision user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create or update user account",
		})
		return
	}

	// Provision tenants from OIDC groups (Version 1.11.2+: Auto-tenant Provisioning)
	if h.tenantProvisioner != nil && len(claims.Groups) > 0 {
		// Parse groups and create tenant mappings
		mappings := oidcauth.ParseGroups(claims.Groups, &h.config.Auth.OIDC.TenantProvisioning.GroupMapping)
		
		if len(mappings) > 0 {
			h.logger.WithFields(logrus.Fields{
				"user_id":        user.ID,
				"groups_count":   len(claims.Groups),
				"mappings_count": len(mappings),
			}).Info("Provisioning tenants from OIDC groups")
			
			// Provision tenants (auto-create, add memberships)
			if err := h.tenantProvisioner.ProvisionTenantsForUser(ctx, user.ID, mappings); err != nil {
				// Don't fail login if tenant provisioning fails
				h.logger.WithError(err).Warn("Failed to provision tenants, continuing with login")
			}
		} else {
			h.logger.WithField("user_id", user.ID).Debug("No valid tenant mappings found for user groups")
		}
	}

	// Generate JWT token for our system
	// Get user's tenants for JWT claims
	tenants, err := h.db.ListUserTenants(ctx, user.ID)
	tenantIDs := make([]string, 0, len(tenants))
	if err == nil {
		for _, tenant := range tenants {
			tenantIDs = append(tenantIDs, tenant.ID)
		}
	} else {
		h.logger.WithError(err).Warn("Failed to list user tenants for JWT, proceeding without tenant IDs")
		tenantIDs = []string{}
	}
	
	tokenPair, err := h.jwtMgr.GenerateTokenPair(user.ID, user.Username, user.Email, tenantIDs, false)
	if err != nil {
		h.logger.WithError(err).Error("Failed to generate JWT token")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate authentication token",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
	}).Info("User successfully authenticated via OIDC")

	// Clear OIDC session data
	session.Delete("oidc_state")
	session.Delete("oidc_created_at")
	redirectURL := session.Get("oidc_redirect_url")
	session.Delete("oidc_redirect_url")
	session.Save()

	// Determine redirect URL
	finalRedirectURL := "/dashboard"
	if redirectURL != nil && redirectURL.(string) != "" {
		finalRedirectURL = redirectURL.(string)
	}

	// Return JWT token and redirect URL
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"token_type":    tokenPair.TokenType,
		"expires_at":    tokenPair.ExpiresAt,
		"redirect_url":  finalRedirectURL,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
	})
}

// provisionUser creates or updates user based on OIDC claims
func (h *OIDCHandler) provisionUser(ctx context.Context, issuer string, claims *oidcauth.KeycloakClaims) (*models.User, error) {
	// Try to find existing user by OIDC subject
	existingUser, err := h.db.GetUserByOIDCSubject(ctx, issuer, claims.Subject)
	if err == nil {
		// User exists - update if auto-update is enabled
		if h.config.Auth.OIDC.AutoUpdateUser {
			existingUser.Email = claims.Email
			existingUser.FullName = claims.Name
			existingUser.UpdatedAt = time.Now()
			existingUser.LastLogin = &[]time.Time{time.Now()}[0]

			if err := h.db.UpdateUser(ctx, existingUser); err != nil {
				h.logger.WithError(err).Error("Failed to update existing OIDC user")
				return nil, fmt.Errorf("failed to update user: %w", err)
			}

			h.logger.WithField("user_id", existingUser.ID).Info("Existing OIDC user updated")
		} else {
			// Just update last login time
			now := time.Now()
			existingUser.LastLogin = &now
			if err := h.db.UpdateUser(ctx, existingUser); err != nil {
				h.logger.WithError(err).Warn("Failed to update last login time")
			}
		}

		return existingUser, nil
	}

	// User doesn't exist - create if auto-create is enabled
	if !h.config.Auth.OIDC.AutoCreateUser {
		return nil, fmt.Errorf("user does not exist and auto-creation is disabled")
	}

	// Create new user
	now := time.Now()
	newUser := &models.User{
		ID:           uuid.New().String(),
		Username:     h.generateUsername(claims),
		Email:        claims.Email,
		FullName:     claims.Name,
		AuthProvider: models.AuthProviderOIDC,
		OIDCSubject:  &claims.Subject,
		OIDCIssuer:   &issuer,
		Status:       models.UserStatusActive,
		IsAdmin:      h.isAdminRole(claims),
		IsActive:     true,
		Verified:     claims.EmailVerified, // Use email verification from OIDC provider
		CreatedAt:    now,
		UpdatedAt:    now,
		LastLogin:    &now,
		Preferences: models.UserPreferences{
			Theme:    "light",
			Language: "en",
			Timezone: "UTC",
		},
	}

	if claims.EmailVerified {
		newUser.VerifiedAt = &now
	}

	if err := h.db.CreateUser(ctx, newUser); err != nil {
		h.logger.WithError(err).Error("Failed to create new OIDC user")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":     newUser.ID,
		"username":    newUser.Username,
		"email":       newUser.Email,
		"oidc_issuer": issuer,
		"oidc_sub":    claims.Subject,
	}).Info("New user created via OIDC auto-provisioning")

	return newUser, nil
}

// generateUsername creates a username from OIDC claims
func (h *OIDCHandler) generateUsername(claims *oidcauth.KeycloakClaims) string {
	// Try preferred_username first
	if claims.PreferredUsername != "" {
		return claims.PreferredUsername
	}

	// Fallback to email prefix
	if claims.Email != "" {
		if idx := len(claims.Email); idx > 0 {
			for i, c := range claims.Email {
				if c == '@' {
					return claims.Email[:i]
				}
			}
		}
		return claims.Email
	}

	// Last resort: use subject (usually a UUID or ID)
	return "user_" + claims.Subject[:8]
}

// isAdminRole determines if user should be admin based on OIDC claims
func (h *OIDCHandler) isAdminRole(claims *oidcauth.KeycloakClaims) bool {
	// Check realm roles
	for _, role := range claims.RealmRoles {
		if role == "admin" || role == "administrator" {
			return true
		}
	}

	// Check realm access roles
	for _, role := range claims.RealmAccess.Roles {
		if role == "admin" || role == "administrator" {
			return true
		}
	}

	// Check groups
	for _, group := range claims.Groups {
		if group == "/admin" || group == "/administrators" || group == "admin" {
			return true
		}
	}

	return false
}

// generateRandomState generates a cryptographically secure random state string
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// HandleLogout handles OIDC logout (optional endpoint)
// POST /auth/oidc/logout
func (h *OIDCHandler) HandleLogout(c *gin.Context) {
	// Clear session
	session := sessions.Default(c)
	session.Clear()
	session.Save()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logged out successfully",
	})
}

