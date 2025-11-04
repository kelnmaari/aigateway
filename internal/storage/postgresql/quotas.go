// Package sqlite provides SQLite-specific quota implementations
// Version: 1.11.7+ (Enterprise Suite - Usage Quotas System)
package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"aigateway/internal/models"
)

// ========================================
// Quota Management
// ========================================

// CreateQuota создает новую квоту
func (db *PostgreSQLDB) CreateQuota(ctx context.Context, quota *models.Quota) error {
	if quota.ID == "" {
		quota.ID = uuid.New().String()
	}

	now := time.Now()
	quota.CreatedAt = now
	quota.UpdatedAt = now

	query := `
		INSERT INTO quotas (
			id, name, scope, target_id,
			tokens_per_day, tokens_per_month,
			requests_per_day, requests_per_month, max_concurrent,
			max_storage_bytes, max_conversations, max_file_size,
			allowed_models, enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err := db.db.ExecContext(ctx, query,
		quota.ID, quota.Name, quota.Scope, quota.TargetID,
		quota.TokensPerDay, quota.TokensPerMonth,
		quota.RequestsPerDay, quota.RequestsPerMonth, quota.MaxConcurrent,
		quota.MaxStorageBytes, quota.MaxConversations, quota.MaxFileSize,
		quota.AllowedModels, quota.Enabled, quota.CreatedAt, quota.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create quota: %w", err)
	}

	// Create initial quota usage
	usage := &models.QuotaUsage{
		ID:               uuid.New().String(),
		QuotaID:          quota.ID,
		TargetID:         quota.TargetID,
		LastDailyReset:   now,
		LastMonthlyReset: now,
		UpdatedAt:        now,
	}

	usageQuery := `
		INSERT INTO quota_usage (
			id, quota_id, target_id,
			tokens_used_today, tokens_used_month,
			requests_today, requests_month, current_concurrent,
			storage_used_bytes, conversations_count,
			last_daily_reset, last_monthly_reset, updated_at
		) VALUES ($1, $2, $3, 0, 0, 0, 0, 0, 0, 0, $4, $5, $6)
	`

	_, err = db.db.ExecContext(ctx, usageQuery,
		usage.ID, usage.QuotaID, usage.TargetID,
		usage.LastDailyReset, usage.LastMonthlyReset, usage.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create quota usage: %w", err)
	}

	return nil
}

// GetQuota получает квоту по ID
func (db *PostgreSQLDB) GetQuota(ctx context.Context, id string) (*models.Quota, error) {
	query := `
		SELECT id, name, scope, target_id,
			tokens_per_day, tokens_per_month,
			requests_per_day, requests_per_month, max_concurrent,
			max_storage_bytes, max_conversations, max_file_size,
			allowed_models, enabled, created_at, updated_at
		FROM quotas
		WHERE id = $1
	`

	var quota models.Quota
	err := db.db.QueryRowContext(ctx, query, id).Scan(
		&quota.ID, &quota.Name, &quota.Scope, &quota.TargetID,
		&quota.TokensPerDay, &quota.TokensPerMonth,
		&quota.RequestsPerDay, &quota.RequestsPerMonth, &quota.MaxConcurrent,
		&quota.MaxStorageBytes, &quota.MaxConversations, &quota.MaxFileSize,
		&quota.AllowedModels, &quota.Enabled, &quota.CreatedAt, &quota.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quota not found")
		}
		return nil, fmt.Errorf("failed to get quota: %w", err)
	}

	return &quota, nil
}

// GetQuotaByTarget получает квоту по scope и target_id
func (db *PostgreSQLDB) GetQuotaByTarget(ctx context.Context, scope models.QuotaScope, targetID string) (*models.Quota, error) {
	query := `
		SELECT id, name, scope, target_id,
			tokens_per_day, tokens_per_month,
			requests_per_day, requests_per_month, max_concurrent,
			max_storage_bytes, max_conversations, max_file_size,
			allowed_models, enabled, created_at, updated_at
		FROM quotas
		WHERE scope = $1 AND target_id = $2 AND enabled = true
	`

	var quota models.Quota
	err := db.db.QueryRowContext(ctx, query, scope, targetID).Scan(
		&quota.ID, &quota.Name, &quota.Scope, &quota.TargetID,
		&quota.TokensPerDay, &quota.TokensPerMonth,
		&quota.RequestsPerDay, &quota.RequestsPerMonth, &quota.MaxConcurrent,
		&quota.MaxStorageBytes, &quota.MaxConversations, &quota.MaxFileSize,
		&quota.AllowedModels, &quota.Enabled, &quota.CreatedAt, &quota.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quota not found for %s:%s", scope, targetID)
		}
		return nil, fmt.Errorf("failed to get quota: %w", err)
	}

	return &quota, nil
}

// ListQuotas возвращает список квот (опционально по scope)
func (db *PostgreSQLDB) ListQuotas(ctx context.Context, scope *models.QuotaScope) ([]*models.Quota, error) {
	query := `
		SELECT id, name, scope, target_id,
			tokens_per_day, tokens_per_month,
			requests_per_day, requests_per_month, max_concurrent,
			max_storage_bytes, max_conversations, max_file_size,
			allowed_models, enabled, created_at, updated_at
		FROM quotas
	`

	var args []interface{}
	if scope != nil {
		query += " WHERE scope = ?"
		args = append(args, *scope)
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list quotas: %w", err)
	}
	defer rows.Close()

	var quotas []*models.Quota
	for rows.Next() {
		var q models.Quota
		err := rows.Scan(
			&q.ID, &q.Name, &q.Scope, &q.TargetID,
			&q.TokensPerDay, &q.TokensPerMonth,
			&q.RequestsPerDay, &q.RequestsPerMonth, &q.MaxConcurrent,
			&q.MaxStorageBytes, &q.MaxConversations, &q.MaxFileSize,
			&q.AllowedModels, &q.Enabled, &q.CreatedAt, &q.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan quota: %w", err)
		}
		quotas = append(quotas, &q)
	}

	return quotas, rows.Err()
}

// UpdateQuota обновляет квоту
func (db *PostgreSQLDB) UpdateQuota(ctx context.Context, quota *models.Quota) error {
	quota.UpdatedAt = time.Now()

	query := `
		UPDATE quotas SET
			name = $1, tokens_per_day = $2, tokens_per_month = $3,
			requests_per_day = $4, requests_per_month = $5, max_concurrent = $6,
			max_storage_bytes = $7, max_conversations = $8, max_file_size = $9,
			allowed_models = $10, enabled = $11, updated_at = $12
		WHERE id = $13
	`

	result, err := db.db.ExecContext(ctx, query,
		quota.Name, quota.TokensPerDay, quota.TokensPerMonth,
		quota.RequestsPerDay, quota.RequestsPerMonth, quota.MaxConcurrent,
		quota.MaxStorageBytes, quota.MaxConversations, quota.MaxFileSize,
		quota.AllowedModels, quota.Enabled, quota.UpdatedAt,
		quota.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update quota: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("quota not found")
	}

	return nil
}

// DeleteQuota удаляет квоту (and cascade delete usage)
func (db *PostgreSQLDB) DeleteQuota(ctx context.Context, id string) error {
	query := `DELETE FROM quotas WHERE id = $1`

	result, err := db.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete quota: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("quota not found")
	}

	return nil
}

// ========================================
// Quota Usage Tracking
// ========================================

// GetQuotaUsage получает usage по quota_id
func (db *PostgreSQLDB) GetQuotaUsage(ctx context.Context, quotaID string) (*models.QuotaUsage, error) {
	query := `
		SELECT id, quota_id, target_id,
			tokens_used_today, tokens_used_month,
			requests_today, requests_month, current_concurrent,
			storage_used_bytes, conversations_count,
			last_daily_reset, last_monthly_reset, updated_at
		FROM quota_usage
		WHERE quota_id = $1
	`

	var usage models.QuotaUsage
	err := db.db.QueryRowContext(ctx, query, quotaID).Scan(
		&usage.ID, &usage.QuotaID, &usage.TargetID,
		&usage.TokensUsedToday, &usage.TokensUsedMonth,
		&usage.RequestsToday, &usage.RequestsMonth, &usage.CurrentConcurrent,
		&usage.StorageUsedBytes, &usage.ConversationsCount,
		&usage.LastDailyReset, &usage.LastMonthlyReset, &usage.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quota usage not found")
		}
		return nil, fmt.Errorf("failed to get quota usage: %w", err)
	}

	return &usage, nil
}

// GetQuotaUsageByTarget получает usage по target_id
func (db *PostgreSQLDB) GetQuotaUsageByTarget(ctx context.Context, targetID string) (*models.QuotaUsage, error) {
	query := `
		SELECT id, quota_id, target_id,
			tokens_used_today, tokens_used_month,
			requests_today, requests_month, current_concurrent,
			storage_used_bytes, conversations_count,
			last_daily_reset, last_monthly_reset, updated_at
		FROM quota_usage
		WHERE target_id = $1
	`

	var usage models.QuotaUsage
	err := db.db.QueryRowContext(ctx, query, targetID).Scan(
		&usage.ID, &usage.QuotaID, &usage.TargetID,
		&usage.TokensUsedToday, &usage.TokensUsedMonth,
		&usage.RequestsToday, &usage.RequestsMonth, &usage.CurrentConcurrent,
		&usage.StorageUsedBytes, &usage.ConversationsCount,
		&usage.LastDailyReset, &usage.LastMonthlyReset, &usage.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("quota usage not found")
		}
		return nil, fmt.Errorf("failed to get quota usage: %w", err)
	}

	return &usage, nil
}

// UpdateQuotaUsage обновляет usage
func (db *PostgreSQLDB) UpdateQuotaUsage(ctx context.Context, usage *models.QuotaUsage) error {
	usage.UpdatedAt = time.Now()

	query := `
		UPDATE quota_usage SET
			tokens_used_today = $1, tokens_used_month = $2,
			requests_today = $3, requests_month = $4, current_concurrent = $5,
			storage_used_bytes = $6, conversations_count = $7,
			last_daily_reset = $8, last_monthly_reset = $9, updated_at = $10
		WHERE id = $11
	`

	result, err := db.db.ExecContext(ctx, query,
		usage.TokensUsedToday, usage.TokensUsedMonth,
		usage.RequestsToday, usage.RequestsMonth, usage.CurrentConcurrent,
		usage.StorageUsedBytes, usage.ConversationsCount,
		usage.LastDailyReset, usage.LastMonthlyReset, usage.UpdatedAt,
		usage.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update quota usage: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("quota usage not found")
	}

	return nil
}

// ResetQuotaUsage сбрасывает счетчики (daily или monthly)
func (db *PostgreSQLDB) ResetQuotaUsage(ctx context.Context, quotaID string, resetType string) error {
	now := time.Now()

	var query string
	switch resetType {
	case "daily":
		query = `
			UPDATE quota_usage SET
				tokens_used_today = 0,
				requests_today = 0,
				last_daily_reset = $1,
				updated_at = $2
			WHERE quota_id = $3
		`
	case "monthly":
		query = `
			UPDATE quota_usage SET
				tokens_used_month = 0,
				requests_month = 0,
				last_monthly_reset = $1,
				updated_at = $2
			WHERE quota_id = $3
		`
	default:
		return fmt.Errorf("invalid reset type: %s (expected 'daily' or 'monthly')", resetType)
	}

	result, err := db.db.ExecContext(ctx, query, now, now, quotaID)
	if err != nil {
		return fmt.Errorf("failed to reset quota usage: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("quota usage not found")
	}

	return nil
}

// ========================================
// Combined Operations
// ========================================

// GetQuotaWithUsage получает quota и usage вместе
func (db *PostgreSQLDB) GetQuotaWithUsage(ctx context.Context, scope models.QuotaScope, targetID string) (*models.Quota, *models.QuotaUsage, error) {
	// Get quota
	quota, err := db.GetQuotaByTarget(ctx, scope, targetID)
	if err != nil {
		return nil, nil, err
	}

	// Get usage
	usage, err := db.GetQuotaUsage(ctx, quota.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get usage for quota: %w", err)
	}

	return quota, usage, nil
}
