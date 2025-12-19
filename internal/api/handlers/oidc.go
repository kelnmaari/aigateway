// Package handlers provides OIDC authentication HTTP handlers
// Version: 1.11.1+ (Enterprise Suite - Keycloak SSO Integration)
package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"

	"aigateway/internal/auth/jwt"
	oidcauth "aigateway/internal/auth/oidc"
	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// createAuthLogger creates a separate logger for auth events
func createAuthLogger(cfg *config.Config) *logrus.Logger {
	authLogger := logrus.New()
	authLogger.SetLevel(logrus.DebugLevel)
	authLogger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	// Use auth log file if configured
	logPath := cfg.Logging.AuthLogFilePath
	if logPath == "" {
		logPath = "logs/auth.log"
	}

	// Create logs directory if needed
	if dir := filepath.Dir(logPath); dir != "" {
		os.MkdirAll(dir, 0755)
	}

	// Setup rotating file logger
	authLogger.SetOutput(&lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    50, // MB
		MaxBackups: 10,
		MaxAge:     90, // days
		Compress:   true,
	})

	return authLogger
}

// OIDCHandler обрабатывает OIDC authentication flow
type OIDCHandler struct {
	config            *config.Config
	provider          *oidcauth.OIDCProvider
	db                storage.Database
	jwtMgr            *jwt.Manager
	tenantProvisioner *oidcauth.TenantProvisioner // Version 1.11.2+: Auto-tenant provisioning
	logger            *logrus.Logger
	authLogger        *logrus.Logger // Separate auth logger
}

// NewOIDCHandler создает новый OIDC handler
func NewOIDCHandler(cfg *config.Config, provider *oidcauth.OIDCProvider, db storage.Database, jwtMgr *jwt.Manager, logger *logrus.Logger) *OIDCHandler {
	var tenantProvisioner *oidcauth.TenantProvisioner

	// Initialize tenant provisioner if enabled
	if cfg.Auth.OIDC.TenantProvisioning.Enabled {
		tenantProvisioner = oidcauth.NewTenantProvisioner(db, &cfg.Auth.OIDC.TenantProvisioning, logger)
		logger.Info("Tenant provisioner initialized for OIDC")
	}

	// Create auth logger
	authLogger := createAuthLogger(cfg)

	return &OIDCHandler{
		config:            cfg,
		provider:          provider,
		db:                db,
		jwtMgr:            jwtMgr,
		tenantProvisioner: tenantProvisioner,
		logger:            logger,
		authLogger:        authLogger,
	}
}

// SetAuthLogger sets a custom auth logger (for testing or custom logging)
func (h *OIDCHandler) SetAuthLogger(logger *logrus.Logger) {
	h.authLogger = logger
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

	// Auth log - detailed authentication event with claims
	h.authLogger.WithFields(logrus.Fields{
		"event":        "oidc_login_success",
		"user_id":      user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"is_admin":     user.IsAdmin,
		"oidc_issuer":  h.config.Auth.OIDC.Issuer,
		"oidc_subject": claims.Subject,
		"groups":       claims.Groups,
		"realm_roles":  claims.RealmRoles,
		"realm_access": claims.RealmAccess.Roles,
		"scopes":       h.config.Auth.OIDC.Scopes,
		"client_ip":    c.ClientIP(),
		"user_agent":   c.Request.UserAgent(),
	}).Info("OIDC authentication successful")

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

	// Auth log - token generation
	h.authLogger.WithFields(logrus.Fields{
		"event":        "token_generated",
		"user_id":      user.ID,
		"redirect_url": finalRedirectURL,
		"expires_at":   tokenPair.ExpiresAt.Format(time.RFC3339),
	}).Debug("JWT token generated for OIDC user")

	// Return HTML page that stores token and redirects
	// This is needed because OIDC callback is a browser redirect, not an API call
	// Keys must match auth.svelte.ts: 'access_token', 'refresh_token', 'user'
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Authenticating...</title>
	<script>
		(function() {
			// Store tokens - keys must match authStore in auth.svelte.ts
			localStorage.setItem('access_token', %q);
			localStorage.setItem('refresh_token', %q);
			localStorage.setItem('user', JSON.stringify({
				id: %q,
				username: %q,
				email: %q,
				is_admin: %t
			}));
			
			console.log('[OIDC] Tokens saved to localStorage');
			console.log('[OIDC] Redirecting to:', %q);
			
			// Redirect to dashboard
			window.location.href = %q;
		})();
	</script>
</head>
<body>
	<p>Authenticating... Please wait.</p>
</body>
</html>`,
		tokenPair.AccessToken,
		tokenPair.RefreshToken,
		user.ID,
		user.Username,
		user.Email,
		user.IsAdmin,
		finalRedirectURL,
		finalRedirectURL,
	)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// provisionUser creates or updates user based on OIDC claims
func (h *OIDCHandler) provisionUser(ctx context.Context, issuer string, claims *oidcauth.KeycloakClaims) (*models.User, error) {
	// Try to find existing user by OIDC subject
	existingUser, err := h.db.GetUserByOIDCSubject(ctx, issuer, claims.Subject)
	if err == nil {
		// User exists with OIDC binding - update if auto-update is enabled
		return h.updateExistingUser(ctx, existingUser, claims)
	}

	// User not found by OIDC subject - try to find by email and link
	if claims.Email != "" {
		existingByEmail, err := h.db.GetUserByEmail(ctx, claims.Email)
		if err == nil {
			// Found existing user by email - link OIDC credentials
			// Keep original auth_provider (local) to allow both login methods
			h.logger.WithFields(logrus.Fields{
				"user_id":       existingByEmail.ID,
				"email":         claims.Email,
				"auth_provider": existingByEmail.AuthProvider,
				"oidc_issuer":   issuer,
				"oidc_sub":      claims.Subject,
			}).Info("Linking existing user to OIDC credentials (hybrid auth)")

			existingByEmail.OIDCSubject = &claims.Subject
			existingByEmail.OIDCIssuer = &issuer
			// DO NOT change auth_provider - user can still login with password

			return h.updateExistingUser(ctx, existingByEmail, claims)
		}
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
		Verified:     claims.EmailVerified,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastLogin:    &now,
		Preferences: models.UserPreferences{
			Theme:    "light",
			Language: "en",
			Timezone: "UTC",
		},
		Metadata: make(map[string]interface{}),
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

// updateExistingUser updates an existing user with OIDC claims
func (h *OIDCHandler) updateExistingUser(ctx context.Context, user *models.User, claims *oidcauth.KeycloakClaims) (*models.User, error) {
	now := time.Now()
	user.LastLogin = &now
	user.UpdatedAt = now

	if h.config.Auth.OIDC.AutoUpdateUser {
		user.Email = claims.Email
		user.FullName = claims.Name
		user.IsAdmin = h.isAdminRole(claims)

		if claims.EmailVerified && !user.Verified {
			user.Verified = true
			user.VerifiedAt = &now
		}
	}

	if err := h.db.UpdateUser(ctx, user); err != nil {
		h.logger.WithError(err).Error("Failed to update user")
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	h.logger.WithField("user_id", user.ID).Debug("User updated via OIDC")
	return user, nil
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

// isAdminRole determines if user should be admin based on OIDC claims and config
func (h *OIDCHandler) isAdminRole(claims *oidcauth.KeycloakClaims) bool {
	roleMapping := h.config.Auth.OIDC.RoleMapping

	// Use config-defined admin roles, fallback to defaults if not configured
	adminRoles := roleMapping.AdminRoles
	if len(adminRoles) == 0 {
		adminRoles = []string{"admin", "administrator"}
	}

	adminGroups := roleMapping.AdminGroups
	if len(adminGroups) == 0 {
		adminGroups = []string{"/admin", "/administrators", "admin"}
	}

	// Check realm roles
	for _, role := range claims.RealmRoles {
		for _, adminRole := range adminRoles {
			if role == adminRole {
				return true
			}
		}
	}

	// Check realm access roles
	for _, role := range claims.RealmAccess.Roles {
		for _, adminRole := range adminRoles {
			if role == adminRole {
				return true
			}
		}
	}

	// Check groups
	for _, group := range claims.Groups {
		for _, adminGroup := range adminGroups {
			if group == adminGroup {
				return true
			}
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
