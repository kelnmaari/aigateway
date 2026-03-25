// Package handlers provides LDAP authentication HTTP handlers
// Version: 1.11.3+ (Enterprise Suite - LDAP Integration)
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"aigateway/internal/auth/jwt"
	ldapauth "aigateway/internal/auth/ldap"
	oidcauth "aigateway/internal/auth/oidc"
	"aigateway/internal/config"
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// LDAPHandler обрабатывает LDAP authentication requests
type LDAPHandler struct {
	config            *config.Config
	ldapClient        *ldapauth.Client
	db                storage.Database
	jwtMgr            *jwt.Manager
	tenantProvisioner *oidcauth.TenantProvisioner // Reuse from OIDC-02
	logger            *logrus.Logger
}

// NewLDAPHandler создает новый LDAP handler
func NewLDAPHandler(cfg *config.Config, ldapClient *ldapauth.Client, db storage.Database, jwtMgr *jwt.Manager, logger *logrus.Logger) *LDAPHandler {
	var tenantProvisioner *oidcauth.TenantProvisioner

	// Initialize tenant provisioner if enabled (reuse OIDC-02 logic)
	if cfg.Auth.LDAP.TenantProvisioning.Enabled {
		tenantProvisioner = oidcauth.NewTenantProvisioner(db, &cfg.Auth.LDAP.TenantProvisioning, logger)
		logger.Info("Tenant provisioner initialized for LDAP")
	}

	return &LDAPHandler{
		config:            cfg,
		ldapClient:        ldapClient,
		db:                db,
		jwtMgr:            jwtMgr,
		tenantProvisioner: tenantProvisioner,
		logger:            logger,
	}
}

// HandleLogin handles LDAP login request
// POST /auth/ldap/login
func (h *LDAPHandler) HandleLogin(c *gin.Context) {
	// Check if LDAP is enabled
	if !h.config.Auth.LDAP.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "LDAP authentication is not enabled",
		})
		return
	}

	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: username and password are required",
		})
		return
	}

	ctx := c.Request.Context()

	// Authenticate via LDAP
	h.logger.WithField("username", req.Username).Info("Attempting LDAP authentication")

	ldapUser, err := h.ldapClient.Authenticate(req.Username, req.Password)
	if err != nil {
		h.logger.WithError(err).WithField("username", req.Username).Warn("LDAP authentication failed")
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid credentials",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"username": ldapUser.Username,
		"ldap_dn":  ldapUser.DN,
		"email":    ldapUser.Email,
	}).Info("LDAP authentication successful")

	// Provision or update user
	user, err := h.provisionUser(ctx, ldapUser)
	if err != nil {
		h.logger.WithError(err).Error("Failed to provision user")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create or update user account",
		})
		return
	}

	// Provision tenants from LDAP groups (reuse OIDC-02 logic)
	if h.tenantProvisioner != nil && len(ldapUser.Groups) > 0 {
		// Parse groups and create tenant mappings
		mappings := oidcauth.ParseGroups(ldapUser.Groups, &h.config.Auth.LDAP.TenantProvisioning.GroupMapping)

		if len(mappings) > 0 {
			h.logger.WithFields(logrus.Fields{
				"user_id":        user.ID,
				"groups_count":   len(ldapUser.Groups),
				"mappings_count": len(mappings),
			}).Info("Provisioning tenants from LDAP groups")

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

	tokenPair, err := h.jwtMgr.GenerateTokenPair(user.ID, user.Username, user.Email, tenantIDs, user.IsAdmin)
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
	}).Info("User successfully authenticated via LDAP")

	// Return JWT token
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"token_type":    tokenPair.TokenType,
		"expires_at":    tokenPair.ExpiresAt,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
	})
}

// provisionUser создает или обновляет пользователя на основе LDAP data
func (h *LDAPHandler) provisionUser(ctx context.Context, ldapUser *ldapauth.User) (*models.User, error) {
	// Try to find existing user by LDAP DN
	existingUser, err := h.db.GetUserByLDAPDN(ctx, ldapUser.DN)
	if err == nil {
		// User exists - update if auto-update is enabled
		if h.config.Auth.LDAP.AutoUpdateUser {
			existingUser.Email = ldapUser.Email
			existingUser.FullName = ldapUser.FullName
			existingUser.UpdatedAt = time.Now()
			now := time.Now()
			existingUser.LastLogin = &now

			if err := h.db.UpdateUser(ctx, existingUser); err != nil {
				h.logger.WithError(err).Error("Failed to update existing LDAP user")
				return nil, fmt.Errorf("failed to update user: %w", err)
			}

			h.logger.WithField("user_id", existingUser.ID).Info("Existing LDAP user updated")
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
	if !h.config.Auth.LDAP.AutoCreateUser {
		return nil, fmt.Errorf("user does not exist and auto-creation is disabled")
	}

	// Create new user
	now := time.Now()
	newUser := &models.User{
		ID:           uuid.New().String(),
		Username:     ldapUser.Username,
		Email:        ldapUser.Email,
		FullName:     ldapUser.FullName,
		AuthProvider: models.AuthProviderLDAP,
		LDAPDN:       &ldapUser.DN,
		Status:       models.UserStatusActive,
		IsAdmin:      h.isAdminGroup(ldapUser.Groups),
		IsActive:     true,
		Verified:     true, // LDAP users are pre-verified
		VerifiedAt:   &now,
		CreatedAt:    now,
		UpdatedAt:    now,
		LastLogin:    &now,
		Preferences: models.UserPreferences{
			Theme:    "light",
			Language: "en",
			Timezone: "UTC",
		},
		Metadata: make(map[string]any), // Empty map for JSONB
	}

	if err := h.db.CreateUser(ctx, newUser); err != nil {
		h.logger.WithError(err).Error("Failed to create new LDAP user")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":  newUser.ID,
		"username": newUser.Username,
		"email":    newUser.Email,
		"ldap_dn":  ldapUser.DN,
	}).Info("New user created via LDAP auto-provisioning")

	return newUser, nil
}

// isAdminGroup проверяет, является ли пользователь администратором на основе LDAP groups
func (h *LDAPHandler) isAdminGroup(groups []string) bool {
	adminGroups := h.config.Auth.LDAP.TenantProvisioning.GroupMapping.AdminGroups

	for _, group := range groups {
		for _, adminGroup := range adminGroups {
			// Case-insensitive comparison
			if strings.EqualFold(group, adminGroup) {
				return true
			}
			// Partial match for nested groups
			if strings.Contains(strings.ToLower(group), strings.ToLower(adminGroup)) {
				return true
			}
		}
	}

	return false
}

// HandleTestConnection handles LDAP connection test request (admin only)
// GET /auth/ldap/test
func (h *LDAPHandler) HandleTestConnection(c *gin.Context) {
	// This should be protected by admin middleware in router

	if !h.config.Auth.LDAP.Enabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "LDAP authentication is not enabled",
		})
		return
	}

	h.logger.Info("Testing LDAP connection...")

	if err := h.ldapClient.TestConnection(); err != nil {
		h.logger.WithError(err).Error("LDAP connection test failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Connection test failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "LDAP connection successful",
	})
}
