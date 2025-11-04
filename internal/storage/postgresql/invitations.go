package postgresql

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
func (db *PostgreSQLDB) CreateInvitation(ctx context.Context, invitation *models.Invitation) error {
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
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := db.db.ExecContext(ctx, query,
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

	db.logger.WithField("invitation_id", invitation.ID).Info("Invitation created")
	return nil
}

// GetInvitation получает приглашение по ID
func (db *PostgreSQLDB) GetInvitation(ctx context.Context, id string) (*models.Invitation, error) {
	query := `
		SELECT id, token, created_by_user_id, created_at,
			   email, expires_at, max_uses, current_uses,
			   used_at, used_by_user_id,
			   revoked_at, revoked_by_user_id, revoke_reason
		FROM invitations
		WHERE id = $1
	`

	invitation := &models.Invitation{}
	err := db.db.QueryRowContext(ctx, query, id).Scan(
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
func (db *PostgreSQLDB) GetInvitationWithUsers(ctx context.Context, id string) (*models.InvitationWithUsers, error) {
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
		WHERE i.id = $1
	`

	inv := &models.InvitationWithUsers{}

	var (
		creatorID, creatorUsername, creatorEmail, creatorDisplayName         sql.NullString
		usedByID, usedByUsername, usedByEmail, usedByDisplayName             sql.NullString
		revokedByID, revokedByUsername, revokedByEmail, revokedByDisplayName sql.NullString
	)

	err := db.db.QueryRowContext(ctx, query, id).Scan(
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
func (db *PostgreSQLDB) GetInvitationByToken(ctx context.Context, token string) (*models.Invitation, error) {
	query := `
		SELECT id, token, created_by_user_id, created_at,
			   email, expires_at, max_uses, current_uses,
			   used_at, used_by_user_id,
			   revoked_at, revoked_by_user_id, revoke_reason
		FROM invitations
		WHERE token = $1
	`

	invitation := &models.Invitation{}
	err := db.db.QueryRowContext(ctx, query, token).Scan(
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
func (db *PostgreSQLDB) ListInvitations(ctx context.Context, filter models.InvitationListFilter) ([]*models.Invitation, error) {
	query := `
		SELECT id, token, created_by_user_id, created_at,
			   email, expires_at, max_uses, current_uses,
			   used_at, used_by_user_id,
			   revoked_at, revoked_by_user_id, revoke_reason
		FROM invitations
		WHERE 1=1
	`

	args := []interface{}{}
	paramIndex := 1 // PostgreSQL uses $1, $2, $3...

	// Apply filters
	if filter.CreatedByUserID != nil {
		query += fmt.Sprintf(" AND created_by_user_id = $%d", paramIndex)
		args = append(args, *filter.CreatedByUserID)
		paramIndex++
	}

	if filter.Email != nil {
		query += fmt.Sprintf(" AND email = $%d", paramIndex)
		args = append(args, *filter.Email)
		paramIndex++
	}

	if filter.Status != nil {
		// Filter by status (active, used, expired, revoked)
		switch *filter.Status {
		case models.InvitationStatusActive:
			query += " AND revoked_at IS NULL AND current_uses < max_uses"
			query += fmt.Sprintf(" AND (expires_at IS NULL OR expires_at > $%d)", paramIndex)
			args = append(args, time.Now())
			paramIndex++
		case models.InvitationStatusUsed:
			query += " AND current_uses >= max_uses"
		case models.InvitationStatusExpired:
			query += fmt.Sprintf(" AND expires_at IS NOT NULL AND expires_at <= $%d", paramIndex)
			args = append(args, time.Now())
			paramIndex++
		case models.InvitationStatusRevoked:
			query += " AND revoked_at IS NOT NULL"
		}
	}

	// Order by creation date (newest first)
	query += " ORDER BY created_at DESC"

	// Pagination
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", paramIndex)
		args = append(args, filter.Limit)
		paramIndex++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", paramIndex)
		args = append(args, filter.Offset)
		paramIndex++
	}

	rows, err := db.db.QueryContext(ctx, query, args...)
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
func (db *PostgreSQLDB) UseInvitation(ctx context.Context, token string, userID string) error {
	now := time.Now()

	// First check if invitation is valid
	inv, err := db.GetInvitationByToken(ctx, token)
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
			used_at = COALESCE(used_at, $1),
			used_by_user_id = COALESCE(used_by_user_id, $2)
		WHERE token = $3
		  AND current_uses < max_uses
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > $4)
	`

	result, err := db.db.ExecContext(ctx, query, now, userID, token, now)
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

	db.logger.WithField("invitation_token", token).WithField("user_id", userID).Info("Invitation used")
	return nil
}

// RevokeInvitation отзывает приглашение
func (db *PostgreSQLDB) RevokeInvitation(ctx context.Context, id string, revokedByUserID string, reason string) error {
	now := time.Now()

	query := `
		UPDATE invitations
		SET revoked_at = $1,
			revoked_by_user_id = $2,
			revoke_reason = $3
		WHERE id = $4
		  AND revoked_at IS NULL
	`

	result, err := db.db.ExecContext(ctx, query, now, revokedByUserID, reason, id)
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

	db.logger.WithField("invitation_id", id).WithField("revoked_by", revokedByUserID).Info("Invitation revoked")
	return nil
}

// GetInvitationStats возвращает статистику по приглашениям
func (db *PostgreSQLDB) GetInvitationStats(ctx context.Context) (*models.InvitationStats, error) {
	stats := &models.InvitationStats{}

	// Total created
	err := db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invitations").Scan(&stats.TotalCreated)
	if err != nil {
		return nil, fmt.Errorf("failed to get total created: %w", err)
	}

	// Active (not revoked, not expired, not fully used)
	query := `
		SELECT COUNT(*)
		FROM invitations
		WHERE revoked_at IS NULL
		  AND current_uses < max_uses
		  AND (expires_at IS NULL OR expires_at > $1)
	`
	err = db.db.QueryRowContext(ctx, query, time.Now()).Scan(&stats.Active)
	if err != nil {
		return nil, fmt.Errorf("failed to get active count: %w", err)
	}

	// Used (fully used)
	err = db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invitations WHERE current_uses >= max_uses").Scan(&stats.Used)
	if err != nil {
		return nil, fmt.Errorf("failed to get used count: %w", err)
	}

	// Expired
	query = `
		SELECT COUNT(*)
		FROM invitations
		WHERE expires_at IS NOT NULL
		  AND expires_at <= $1
		  AND revoked_at IS NULL
	`
	err = db.db.QueryRowContext(ctx, query, time.Now()).Scan(&stats.Expired)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired count: %w", err)
	}

	// Revoked
	err = db.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM invitations WHERE revoked_at IS NOT NULL").Scan(&stats.Revoked)
	if err != nil {
		return nil, fmt.Errorf("failed to get revoked count: %w", err)
	}

	return stats, nil
}
