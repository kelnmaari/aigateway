// Package service provides authentication business logic
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/auth/password"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// BootstrapService handles system initialization and first-time setup
type BootstrapService struct {
	db       storage.Database
	adminKey string // Bootstrap token from config
	logger   *logrus.Logger
}

// NewBootstrapService creates a new BootstrapService
func NewBootstrapService(db storage.Database, adminKey string, logger *logrus.Logger) *BootstrapService {
	return &BootstrapService{
		db:       db,
		adminKey: adminKey,
		logger:   logger,
	}
}

// IsInitialized checks if the system has been initialized (has at least one admin user)
func (s *BootstrapService) IsInitialized(ctx context.Context) (bool, error) {
	// Check if there are any admin users
	status := models.UserStatusActive
	users, err := s.db.ListUsers(ctx, models.UserFilters{
		Status: &status,
		Limit:  1,
	})

	if err != nil {
		return false, fmt.Errorf("failed to check initialization: %w", err)
	}

	// System is initialized if there's at least one active user
	initialized := len(users) > 0

	s.logger.WithField("initialized", initialized).Debug("System initialization check")

	return initialized, nil
}

// BootstrapRequest defines the request structure for system bootstrap
type BootstrapRequest struct {
	AdminToken  string `json:"admin_token" binding:"required"`
	Username    string `json:"username" binding:"required"`
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"display_name,omitempty"`
}

// BootstrapResponse defines the response structure for system bootstrap
type BootstrapResponse struct {
	User    *models.User `json:"user"`
	Message string       `json:"message"`
}

// Bootstrap performs first-time system setup by creating the first superadmin
func (s *BootstrapService) Bootstrap(ctx context.Context, req *BootstrapRequest) (*BootstrapResponse, error) {
	// Check if already initialized
	initialized, err := s.IsInitialized(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check initialization: %w", err)
	}

	if initialized {
		return nil, fmt.Errorf("system is already initialized")
	}

	// Verify admin token
	if req.AdminToken != s.adminKey {
		s.logger.Warn("Invalid bootstrap token attempt")
		return nil, fmt.Errorf("invalid bootstrap token")
	}

	// Validate input
	if err := password.ValidateUsername(req.Username); err != nil {
		return nil, fmt.Errorf("username validation failed: %w", err)
	}
	if err := password.ValidateEmail(req.Email); err != nil {
		return nil, fmt.Errorf("email validation failed: %w", err)
	}
	if err := password.Validate(req.Password); err != nil {
		return nil, fmt.Errorf("password validation failed: %w", err)
	}

	// Hash password
	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create superadmin user
	user := &models.User{
		ID:           generateID("user"),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FullName:     req.DisplayName,
		Status:       models.UserStatusActive,
		IsActive:     true,
		Verified:     true, // Auto-verify first admin
		IsAdmin:      true, // Superadmin flag
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if user.FullName == "" {
		user.FullName = req.Username
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if r := recover(); r != nil {
			s.logger.Errorf("Recovered from panic in Bootstrap transaction: %v", r)
			tx.Rollback()
		}
	}()

	// Create user in database
	if err := tx.CreateUser(ctx, user); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create superadmin user: %w", err)
	}

	// Create personal tenant
	tenant := &models.Tenant{
		ID:          generateID("tenant"),
		Name:        fmt.Sprintf("%s's Workspace", user.FullName),
		Slug:        generateSlug(user.Username),
		Description: "Personal workspace",
		OwnerID:     user.ID,
		Type:        models.TenantTypePersonal,
		Status:      models.TenantStatusActive,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := tx.CreateTenant(ctx, tenant); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create personal tenant: %w", err)
	}

	// Add user as owner of personal tenant
	member := &models.TenantMember{
		TenantID:  tenant.ID,
		UserID:    user.ID,
		Role:      models.TenantRoleOwner,
		JoinedAt:  time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := tx.AddTenantMember(ctx, member); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to add user to tenant: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":   user.ID,
		"username":  user.Username,
		"tenant_id": tenant.ID,
		"is_admin":  user.IsAdmin,
	}).Info("🎉 System bootstrapped successfully - first superadmin created")

	return &BootstrapResponse{
		User:    user,
		Message: "System initialized successfully. First superadmin created.",
	}, nil
}
