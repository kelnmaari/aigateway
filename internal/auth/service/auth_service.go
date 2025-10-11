// Package service provides authentication business logic
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/auth/jwt"
	"ollama-openai-proxy/internal/auth/password"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

// AuthService управляет аутентификацией пользователей
type AuthService struct {
	db         storage.Database
	jwtManager *jwt.Manager
	logger     *logrus.Logger
}

// NewAuthService создает новый Auth Service
func NewAuthService(db storage.Database, jwtManager *jwt.Manager, logger *logrus.Logger) *AuthService {
	return &AuthService{
		db:         db,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// RegisterRequest запрос на регистрацию
type RegisterRequest struct {
	Username    string `json:"username" binding:"required"`
	Email       string `json:"email" binding:"required"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"display_name,omitempty"`
}

// RegisterResponse ответ на регистрацию
type RegisterResponse struct {
	User           *models.User   `json:"user"`
	TokenPair      *jwt.TokenPair `json:"token"`
	PersonalTenant *models.Tenant `json:"personal_tenant"`
}

// Register регистрирует нового пользователя
func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	// Validate input
	if err := password.ValidateUsername(req.Username); err != nil {
		return nil, err
	}

	if err := password.ValidateEmail(req.Email); err != nil {
		return nil, err
	}

	if err := password.Validate(req.Password); err != nil {
		return nil, err
	}

	// Check if username already exists
	existingUser, err := s.db.GetUserByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, password.ValidationError{
			Field:   "username",
			Message: "username already taken",
		}
	}

	// Check if email already exists
	existingUser, err = s.db.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, password.ValidationError{
			Field:   "email",
			Message: "email already registered",
		}
	}

	// Hash password
	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		ID:           generateID("user"),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FullName:     req.DisplayName,
		Status:       models.UserStatusActive,
		IsActive:     true,
		Verified:     false,
		IsAdmin:      false,
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
	defer tx.Rollback()

	// Create user in database
	if err := tx.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
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
		return nil, fmt.Errorf("failed to add user to tenant: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Generate JWT tokens
	tokenPair, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Username,
		user.Email,
		[]string{tenant.ID},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":   user.ID,
		"username":  user.Username,
		"tenant_id": tenant.ID,
	}).Info("User registered successfully")

	// Remove sensitive data
	user.PasswordHash = ""

	return &RegisterResponse{
		User:           user,
		TokenPair:      tokenPair,
		PersonalTenant: tenant,
	}, nil
}

// LoginRequest запрос на вход
type LoginRequest struct {
	Username   string `json:"username"` // username или email
	Email      string `json:"email"`    // альтернативный способ входа
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"remember_me"` // "Не выходить из системы 24 часа"
}

// LoginResponse ответ на вход
type LoginResponse struct {
	User      *models.User     `json:"user"`
	TokenPair *jwt.TokenPair   `json:"token"`
	Tenants   []*models.Tenant `json:"tenants"`
}

// Login выполняет вход пользователя
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	if req.Password == "" {
		return nil, password.ValidationError{
			Field:   "password",
			Message: "password is required",
		}
	}

	// Find user by username or email
	var user *models.User
	var err error

	if req.Username != "" {
		user, err = s.db.GetUserByUsername(ctx, req.Username)
		s.logger.WithFields(logrus.Fields{
			"username": req.Username,
			"found":    user != nil,
			"error":    err,
		}).Debug("Looking up user by username")
	} else if req.Email != "" {
		user, err = s.db.GetUserByEmail(ctx, req.Email)
		s.logger.WithFields(logrus.Fields{
			"email": req.Email,
			"found": user != nil,
			"error": err,
		}).Debug("Looking up user by email")
	} else {
		return nil, password.ValidationError{
			Field:   "username",
			Message: "username or email is required",
		}
	}

	if err != nil || user == nil {
		s.logger.WithField("error", err).Warn("User not found during login")
		return nil, password.ValidationError{
			Field:   "username",
			Message: "user not found",
		}
	}

	// Check user status
	s.logger.WithFields(logrus.Fields{
		"user_id":   user.ID,
		"status":    user.Status,
		"is_active": user.IsActive,
		"verified":  user.Verified,
	}).Debug("Checking user status")

	if user.Status != models.UserStatusActive {
		s.logger.WithFields(logrus.Fields{
			"user_id": user.ID,
			"status":  user.Status,
		}).Warn("User login rejected - account not active")
		return nil, password.ValidationError{
			Field:   "username",
			Message: fmt.Sprintf("account is %s", user.Status),
		}
	}

	// Verify password
	s.logger.WithFields(logrus.Fields{
		"user_id":         user.ID,
		"username":        user.Username,
		"hash_length":     len(user.PasswordHash),
		"password_length": len(req.Password),
	}).Debug("Verifying password")

	if err := password.Verify(user.PasswordHash, req.Password); err != nil {
		s.logger.WithFields(logrus.Fields{
			"user_id":  user.ID,
			"username": user.Username,
			"error":    err.Error(),
		}).Warn("Failed login attempt - incorrect password")

		return nil, password.ValidationError{
			Field:   "password",
			Message: "incorrect password",
		}
	}

	s.logger.WithField("user_id", user.ID).Debug("Password verified successfully")

	// Get user's tenants
	tenants, err := s.db.ListUserTenants(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tenants: %w", err)
	}

	// Extract tenant IDs
	tenantIDs := make([]string, len(tenants))
	for i, t := range tenants {
		tenantIDs[i] = t.ID
	}

	// Generate JWT tokens (with rememberMe support)
	tokenPair, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Username,
		user.Email,
		tenantIDs,
		req.RememberMe, // "Не выходить из системы 24 часа"
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update last login time
	now := time.Now()
	user.LastLogin = &now
	user.UpdatedAt = now

	if err := s.db.UpdateUser(ctx, user); err != nil {
		// Log but don't fail login
		s.logger.WithError(err).Warn("Failed to update last login time")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":     user.ID,
		"username":    user.Username,
		"remember_me": req.RememberMe,
	}).Info("User logged in successfully")

	// Remove sensitive data
	user.PasswordHash = ""

	return &LoginResponse{
		User:      user,
		TokenPair: tokenPair,
		Tenants:   tenants,
	}, nil
}

// Logout выполняет выход пользователя (добавляет токен в blacklist)
func (s *AuthService) Logout(ctx context.Context, accessToken string, expiresAt time.Time) error {
	s.jwtManager.RevokeToken(accessToken, expiresAt)

	s.logger.Debug("User logged out successfully")

	return nil
}

// RefreshToken обновляет access token используя refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user
	user, err := s.db.GetUser(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check user status
	if user.Status != models.UserStatusActive {
		return nil, fmt.Errorf("account is %s", user.Status)
	}

	// Get user's tenants
	tenants, err := s.db.ListUserTenants(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user tenants: %w", err)
	}

	tenantIDs := make([]string, len(tenants))
	for i, t := range tenants {
		tenantIDs[i] = t.ID
	}

	// Generate new token pair
	newTokenPair, err := s.jwtManager.GenerateTokenPair(
		user.ID,
		user.Username,
		user.Email,
		tenantIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Revoke old refresh token
	s.jwtManager.RevokeToken(refreshToken, time.Now().Add(7*24*time.Hour))

	s.logger.WithField("user_id", user.ID).Debug("Token refreshed successfully")

	return newTokenPair, nil
}

// ChangePasswordRequest запрос на смену пароля
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required,min=8"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=100"`
}

// ChangePasswordResponse ответ на смену пароля
type ChangePasswordResponse struct {
	Message string `json:"message"`
}

// ChangePassword изменяет пароль пользователя
func (s *AuthService) ChangePassword(ctx context.Context, userID string, req ChangePasswordRequest) (*ChangePasswordResponse, error) {
	// Get user from database
	user, err := s.db.GetUser(ctx, userID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get user")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Verify current password
	if err := password.Verify(user.PasswordHash, req.CurrentPassword); err != nil {
		s.logger.WithField("user_id", userID).Warn("Invalid current password")
		return nil, fmt.Errorf("current password is incorrect")
	}

	// Validate new password strength
	if err := password.Validate(req.NewPassword); err != nil {
		return nil, err // ValidationError from password package
	}

	// Check if new password is same as current
	if req.CurrentPassword == req.NewPassword {
		return nil, password.ValidationError{
			Field:   "new_password",
			Message: "new password must be different from current password",
		}
	}

	// Hash new password
	newHash, err := password.Hash(req.NewPassword)
	if err != nil {
		s.logger.WithError(err).Error("Failed to hash password")
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password in database
	if err := s.db.UpdateUserPassword(ctx, userID, newHash); err != nil {
		s.logger.WithError(err).Error("Failed to update password")
		return nil, fmt.Errorf("failed to update password: %w", err)
	}

	// Log the password change
	s.logger.WithField("user_id", userID).Info("Password changed successfully")

	// TODO: Optionally invalidate all existing sessions except current
	// This would require additional implementation

	return &ChangePasswordResponse{
		Message: "Password changed successfully",
	}, nil
}

// Helper functions

func generateID(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, uuid.New().String())
}

func generateSlug(username string) string {
	// Convert to lowercase and replace spaces with hyphens
	slug := strings.ToLower(username)
	slug = strings.ReplaceAll(slug, " ", "-")

	// Add random suffix to ensure uniqueness
	slug = fmt.Sprintf("%s-%s", slug, uuid.New().String()[:8])

	return slug
}
