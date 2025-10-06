// Package sqlite provides SQLite implementation of user-related database operations
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"ollama-openai-proxy/internal/models"
)

// ========================================
// Users CRUD Operations
// ========================================

// CreateUser создает нового пользователя
func (s *SQLiteDB) CreateUser(ctx context.Context, user *models.User) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("username", user.Username).Debug("Creating user")

	// Serialize preferences to JSON
	preferencesJSON, err := json.Marshal(user.Preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// Serialize metadata to JSON if present
	var metadataJSON []byte
	if user.Metadata != nil {
		metadataJSON, err = json.Marshal(user.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO users (
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.FullName,
		user.PasswordHash,
		user.Status,
		user.IsAdmin,
		user.IsActive,
		user.Verified,
		user.VerifiedAt,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastLogin,
		preferencesJSON,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	s.logger.WithField("user_id", user.ID).Info("User created successfully")
	return nil
}

// GetUser получает пользователя по ID
func (s *SQLiteDB) GetUser(ctx context.Context, id string) (*models.User, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata
		FROM users
		WHERE id = ?
	`

	user, err := s.scanUser(s.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetUserByUsername получает пользователя по username
func (s *SQLiteDB) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata
		FROM users
		WHERE username = ?
	`

	user, err := s.scanUser(s.db.QueryRowContext(ctx, query, username))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return user, nil
}

// GetUserByEmail получает пользователя по email
func (s *SQLiteDB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata
		FROM users
		WHERE email = ?
	`

	user, err := s.scanUser(s.db.QueryRowContext(ctx, query, email))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", email)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// UpdateUser обновляет данные пользователя
func (s *SQLiteDB) UpdateUser(ctx context.Context, user *models.User) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("user_id", user.ID).Debug("Updating user")

	// Serialize preferences to JSON
	preferencesJSON, err := json.Marshal(user.Preferences)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// Serialize metadata to JSON if present
	var metadataJSON []byte
	if user.Metadata != nil {
		metadataJSON, err = json.Marshal(user.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		UPDATE users SET
			username = ?,
			email = ?,
			full_name = ?,
			password_hash = ?,
			status = ?,
			is_admin = ?,
			is_active = ?,
			verified = ?,
			verified_at = ?,
			updated_at = ?,
			last_login = ?,
			preferences = ?,
			metadata = ?
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		user.Username,
		user.Email,
		user.FullName,
		user.PasswordHash,
		user.Status,
		user.IsAdmin,
		user.IsActive,
		user.Verified,
		user.VerifiedAt,
		user.UpdatedAt,
		user.LastLogin,
		preferencesJSON,
		metadataJSON,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", user.ID)
	}

	s.logger.WithField("user_id", user.ID).Info("User updated successfully")
	return nil
}

// DeleteUser удаляет пользователя (soft delete)
func (s *SQLiteDB) DeleteUser(ctx context.Context, id string) error {
	if s.db == nil {
		return fmt.Errorf("database not connected")
	}

	s.logger.WithField("user_id", id).Debug("Deleting user (soft delete)")

	// Soft delete: update status to 'deleted'
	query := `
		UPDATE users SET
			status = 'deleted',
			updated_at = CURRENT_TIMESTAMP,
			is_active = 0
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found: %s", id)
	}

	s.logger.WithField("user_id", id).Info("User deleted successfully (soft delete)")
	return nil
}

// ListUsers возвращает список пользователей с фильтрацией
func (s *SQLiteDB) ListUsers(ctx context.Context, filters models.UserFilters) ([]*models.User, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	query := `
		SELECT 
			id, username, email, full_name, password_hash,
			status, is_admin, is_active, verified, verified_at,
			created_at, updated_at, last_login,
			preferences, metadata
		FROM users
		WHERE 1=1
	`

	var args []interface{}

	// Apply filters
	if filters.Status != nil {
		query += " AND status = ?"
		args = append(args, *filters.Status)
	}

	if filters.IsAdmin != nil {
		query += " AND is_admin = ?"
		args = append(args, *filters.IsAdmin)
	}

	if filters.IsActive != nil {
		query += " AND is_active = ?"
		args = append(args, *filters.IsActive)
	}

	if filters.Search != "" {
		query += " AND (username LIKE ? OR email LIKE ? OR full_name LIKE ?)"
		searchPattern := "%" + filters.Search + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	// Add ordering
	query += " ORDER BY created_at DESC"

	// Add pagination
	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}

	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user, err := s.scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	s.logger.WithField("count", len(users)).Debug("Listed users")
	return users, nil
}

// ========================================
// Helper Methods
// ========================================

// scanner is an interface that matches both sql.Row and sql.Rows
type scanner interface {
	Scan(dest ...interface{}) error
}

// scanUser сканирует строку БД в модель User
func (s *SQLiteDB) scanUser(row scanner) (*models.User, error) {
	var user models.User
	var preferencesJSON []byte
	var metadataJSON []byte

	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.FullName,
		&user.PasswordHash,
		&user.Status,
		&user.IsAdmin,
		&user.IsActive,
		&user.Verified,
		&user.VerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLogin,
		&preferencesJSON,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	// Deserialize preferences
	if err := json.Unmarshal(preferencesJSON, &user.Preferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}

	// Deserialize metadata if present
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &user.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &user, nil
}
