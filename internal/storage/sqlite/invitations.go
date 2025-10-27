package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"aigateway/internal/models"
	"github.com/google/uuid"
)

// ========================================
// Invitations CRUD (AUTH-03: Invitation-Only Registration System, v2.2.0)
// ========================================

// CreateInvitation создает новое приглашение
func (s *SQLiteDB) CreateInvitation(ctx context.Context, invitation *models.Invitation) error {
	// Generate IDs if not provided
	if invitation.ID == "" {
		invitation.ID = uuid.New().String()
	}
	if invitation.Token == "" {
		invitation.Token = uuid.New().String()
	}
	if invitation.MaxUses == 0 {
		invitation.MaxUses = 1 // Default: single-use
	}

	invitation.CreatedAt = time.Now()
	invitation.CurrentUses = 0

	query := `
		INSERT INTO invitations (
			id, token, created_by_user_id, created_at,
			email, expires_at, max_uses, current_uses
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		invitation.ID,
		invitation.Token,
		invitation.CreatedByUserID,
		invitation.CreatedAt,
		invitation.Email,
		invitation.ExpiresAt,
		invitation.MaxUses,
		invitation.CurrentUses,
	)

	if err != nil {
		return fmt.Errorf("failed to create invitation: %w", err)
	}

	s.logger.WithField("invitation_id", invitation.ID).Info("Invitation created")
	return nil
}

// GetInvitation получает приглашение по ID
func (s *SQLiteDB) GetInvitation(ctx context.Context, id string) (*models.Invitation, error) {
	query := `
		SELECT id, token, created_by_user_id, created_at,
			   email, expires_at, max_uses, current_uses,
			   used_at, used_by_user_id,
			   revoked_at, revoked_by_user_id, revoke_reason
		FROM invitations
		WHERE id = ?
	`

	invitation := &models.Invitation{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&invitation.ID,
		&invitation.Token,
		&invitation.CreatedByUserID,
		&invitation.CreatedAt,
		&invitation.Email,
		&invitation.ExpiresAt,
		&invitation.MaxUses,
		&invitation.CurrentUses,
		&invitation.UsedAt,
		&invitation.UsedByUserID,
		&invitation.RevokedAt,
		&invitation.RevokedByUserID,
		&invitation.RevokeReason,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invitation: %w", err)
	}

	return invitation, nil
}

// GetInvitationWithUsers получает приглашение с информацией о пользователях (AUTH-03)
func (s *SQLiteDB) GetInvitationWithUsers(ctx context.Context, id string) (*models.InvitationWithUsers, error) {
	query := `
		SELECT 
			i.id, i.token, i.created_by_user_id, i.created_at,
			i.email, i.expires_at, i.max_uses, i.current_uses,
			i.used_at, i.used_by_user_id,
			i.revoked_at, i.revoked_by_user_id, i.revoke_reason,
			-- Creator info
			creator.id, creator.username, creator.email, creator.full_name,
			-- Used by info
			used_by.id, used_by.username, used_by.email, used_by.full_name,
			-- Revoked by info
			revoked_by.id, revoked_by.username, revoked_by.email, revoked_by.full_name
		FROM invitations i
		LEFT JOIN users creator ON i.created_by_user_id = creator.id
		LEFT JOIN users used_by ON i.used_by_user_id = used_by.id
		LEFT JOIN users revoked_by ON i.revoked_by_user_id = revoked_by.id
		WHERE i.id = ?
	`

	inv := &models.InvitationWithUsers{}
	
	var (
		creatorID, creatorUsername, creatorEmail, creatorDisplayName             sql.NullString
		usedByID, usedByUsername, usedByEmail, usedByDisplayName                 sql.NullString
		revokedByID, revokedByUsername, revokedByEmail, revokedByDisplayName     sql.NullString
	)

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		// Invitation fields
		&inv.ID,
		&inv.Token,
		&inv.CreatedByUserID,
		&inv.CreatedAt,
		&inv.Email,
		&inv.ExpiresAt,
		&inv.MaxUses,
		&inv.CurrentUses,
		&inv.UsedAt,
		&inv.UsedByUserID,
		&inv.RevokedAt,
		&inv.RevokedByUserID,
		&inv.RevokeReason,
		// Creator
		&creatorID,
		&creatorUsername,
		&creatorEmail,
		&creatorDisplayName,
		// Used by
		&usedByID,
		&usedByUsername,
		&usedByEmail,
		&usedByDisplayName,
		// Revoked by
		&revokedByID,
		&revokedByUsername,
		&revokedByEmail,
		&revokedByDisplayName,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invitation with users: %w", err)
	}

	// Populate user info if available
	if creatorID.Valid {
		inv.CreatedBy = &models.UserInfo{
			ID:       creatorID.String,
			Username: creatorUsername.String,
			Email:    creatorEmail.String,
			FullName: creatorDisplayName.String,
		}
	}

	if usedByID.Valid {
		inv.UsedBy = &models.UserInfo{
			ID:       usedByID.String,
			Username: usedByUsername.String,
			Email:    usedByEmail.String,
			FullName: usedByDisplayName.String,
		}
	}

	if revokedByID.Valid {
		inv.RevokedBy = &models.UserInfo{
			ID:       revokedByID.String,
			Username: revokedByUsername.String,
			Email:    revokedByEmail.String,
			FullName: revokedByDisplayName.String,
		}
	}

	return inv, nil
}

// GetInvitationByToken получает приглашение по токену
func (s *SQLiteDB) GetInvitationByToken(ctx context.Context, token string) (*models.Invitation, error) {
	query := `
		SELECT id, token, created_by_user_id, created_at,
			   email, expires_at, max_uses, current_uses,
			   used_at, used_by_user_id,
			   revoked_at, revoked_by_user_id, revoke_reason
		FROM invitations
		WHERE token = ?
	`

	invitation := &models.Invitation{}
	err := s.db.QueryRowContext(ctx, query, token).Scan(
		&invitation.ID,
		&invitation.Token,
		&invitation.CreatedByUserID,
		&invitation.CreatedAt,
		&invitation.Email,
		&invitation.ExpiresAt,
		&invitation.MaxUses,
		&invitation.CurrentUses,
		&invitation.UsedAt,
		&invitation.UsedByUserID,
		&invitation.RevokedAt,
		&invitation.RevokedByUserID,
		&invitation.RevokeReason,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found with token: %s", token)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get invitation by token: %w", err)
	}

	return invitation, nil
}

// ListInvitations возвращает список приглашений с фильтрацией
func (s *SQLiteDB) ListInvitations(ctx context.Context, filter models.InvitationListFilter) ([]*models.Invitation, error) {
	query := `
		SELECT id, token, created_by_user_id, created_at,
			   email, expires_at, max_uses, current_uses,
			   used_at, used_by_user_id,
			   revoked_at, revoked_by_user_id, revoke_reason
		FROM invitations
		WHERE 1=1
	`

	args := []interface{}{}

	// Apply filters
	if filter.CreatedByUserID != nil {
		query += " AND created_by_user_id = ?"
		args = append(args, *filter.CreatedByUserID)
	}

	if filter.Email != nil {
		query += " AND email = ?"
		args = append(args, *filter.Email)
	}

	if filter.Status != nil {
		// Filter by status (active, used, expired, revoked)
		switch *filter.Status {
		case models.InvitationStatusActive:
			query += " AND revoked_at IS NULL AND current_uses < max_uses"
			query += " AND (expires_at IS NULL OR expires_at > ?)"
			args = append(args, time.Now())
		case models.InvitationStatusUsed:
			query += " AND current_uses >= max_uses"
		case models.InvitationStatusExpired:
			query += " AND expires_at IS NOT NULL AND expires_at <= ?"
			args = append(args, time.Now())
		case models.InvitationStatusRevoked:
			query += " AND revoked_at IS NOT NULL"
		}
	}

	// Order by creation date (newest first)
	query += " ORDER BY created_at DESC"

	// Pagination
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list invitations: %w", err)
	}
	defer rows.Close()

	invitations := []*models.Invitation{}
	for rows.Next() {
		inv := &models.Invitation{}
		err := rows.Scan(
			&inv.ID,
			&inv.Token,
			&inv.CreatedByUserID,
			&inv.CreatedAt,
			&inv.Email,
			&inv.ExpiresAt,
			&inv.MaxUses,
			&inv.CurrentUses,
			&inv.UsedAt,
			&inv.UsedByUserID,
			&inv.RevokedAt,
			&inv.RevokedByUserID,
			&inv.RevokeReason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invitation: %w", err)
		}
		invitations = append(invitations, inv)
	}

	return invitations, nil
}

// UseInvitation увеличивает счетчик использований приглашения
func (s *SQLiteDB) UseInvitation(ctx context.Context, token string, userID string) error {
	now := time.Now()

	// First check if invitation is valid
	inv, err := s.GetInvitationByToken(ctx, token)
	if err != nil {
		return err
	}

	if !inv.IsValid() {
		return fmt.Errorf("invitation is not valid (status: %s)", inv.GetStatus())
	}

	// Update invitation
	query := `
		UPDATE invitations
		SET current_uses = current_uses + 1,
			used_at = COALESCE(used_at, ?),
			used_by_user_id = COALESCE(used_by_user_id, ?)
		WHERE token = ?
		  AND current_uses < max_uses
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > ?)
	`

	result, err := s.db.ExecContext(ctx, query, now, userID, token, now)
	if err != nil {
		return fmt.Errorf("failed to use invitation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("invitation cannot be used (may be already used, revoked, or expired)")
	}

	s.logger.WithField("invitation_token", token).WithField("user_id", userID).Info("Invitation used")
	return nil
}

// RevokeInvitation отзывает приглашение
func (s *SQLiteDB) RevokeInvitation(ctx context.Context, id string, revokedByUserID string, reason string) error {
	now := time.Now()

	query := `
		UPDATE invitations
		SET revoked_at = ?,
			revoked_by_user_id = ?,
			revoke_reason = ?
		WHERE id = ?
		  AND revoked_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, now, revokedByUserID, reason, id)
	if err != nil {
		return fmt.Errorf("failed to revoke invitation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("invitation not found or already revoked: %s", id)
	}

	s.logger.WithField("invitation_id", id).WithField("revoked_by", revokedByUserID).Info("Invitation revoked")
	return nil
}

// GetInvitationStats возвращает статистику по приглашениям
func (s *SQLiteDB) GetInvitationStats(ctx context.Context) (*models.InvitationStats, error) {
	stats := &models.InvitationStats{}

	// Total created
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invitations").Scan(&stats.TotalCreated)
	if err != nil {
		return nil, fmt.Errorf("failed to get total created: %w", err)
	}

	// Active (not revoked, not expired, not fully used)
	query := `
		SELECT COUNT(*)
		FROM invitations
		WHERE revoked_at IS NULL
		  AND current_uses < max_uses
		  AND (expires_at IS NULL OR expires_at > ?)
	`
	err = s.db.QueryRowContext(ctx, query, time.Now()).Scan(&stats.Active)
	if err != nil {
		return nil, fmt.Errorf("failed to get active count: %w", err)
	}

	// Used (fully used)
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invitations WHERE current_uses >= max_uses").Scan(&stats.Used)
	if err != nil {
		return nil, fmt.Errorf("failed to get used count: %w", err)
	}

	// Expired
	query = `
		SELECT COUNT(*)
		FROM invitations
		WHERE expires_at IS NOT NULL
		  AND expires_at <= ?
		  AND revoked_at IS NULL
	`
	err = s.db.QueryRowContext(ctx, query, time.Now()).Scan(&stats.Expired)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired count: %w", err)
	}

	// Revoked
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invitations WHERE revoked_at IS NOT NULL").Scan(&stats.Revoked)
	if err != nil {
		return nil, fmt.Errorf("failed to get revoked count: %w", err)
	}

	return stats, nil
}

