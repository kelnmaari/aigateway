// Package storage provides PostgreSQL implementation for scan history data access
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aigateway/internal/models"

	"github.com/google/uuid"
)

// ============================================================================
// Scan History Store Implementation (v4.1.2+)
// ============================================================================

// SaveScanResult saves a scan result to history
func (s *PostgresStore) SaveScanResult(ctx context.Context, result *models.GitLabScanResult) error {
	if result.ID == "" {
		result.ID = uuid.New().String()
	}

	query := `
		INSERT INTO gitlab_scan_history (
			id, project_id, integration_id, scan_type, status, model_id,
			started_at, completed_at, duration_ms, error,
			findings_count, files_affected, tokens_used, results_json
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			completed_at = EXCLUDED.completed_at,
			duration_ms = EXCLUDED.duration_ms,
			error = EXCLUDED.error,
			findings_count = EXCLUDED.findings_count,
			files_affected = EXCLUDED.files_affected,
			tokens_used = EXCLUDED.tokens_used,
			results_json = EXCLUDED.results_json
	`

	_, err := s.db.ExecContext(ctx, query,
		result.ID,
		result.ProjectID,
		nullString(result.IntegrationID),
		result.ScanType,
		result.Status,
		nullString(result.ModelID),
		result.StartedAt,
		nullTime(result.CompletedAt),
		result.DurationMs,
		nullString(result.Error),
		result.FindingsCount,
		result.FilesAffected,
		result.TokensUsed,
		result.ResultsJSON,
	)
	if err != nil {
		return fmt.Errorf("save scan result: %w", err)
	}

	return nil
}

// GetScanResult retrieves a scan result by ID
func (s *PostgresStore) GetScanResult(ctx context.Context, id string) (*models.GitLabScanResult, error) {
	query := `
		SELECT 
			sh.id::text, sh.project_id::text, COALESCE(sh.integration_id::text, '') as integration_id,
			sh.scan_type, sh.status, COALESCE(sh.model_id, '') as model_id,
			sh.started_at, sh.completed_at, sh.duration_ms, sh.error,
			COALESCE(sh.findings_count, 0) as findings_count,
			COALESCE(sh.files_affected, 0) as files_affected,
			COALESCE(sh.tokens_used, 0) as tokens_used,
			sh.results_json,
			COALESCE(p.name, '') as project_name
		FROM gitlab_scan_history sh
		LEFT JOIN gitlab_projects p ON sh.project_id = p.id
		WHERE sh.id = $1::uuid
	`

	var result models.GitLabScanResult
	var completedAt sql.NullTime
	var errorStr sql.NullString
	var resultsJSON sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&result.ID,
		&result.ProjectID,
		&result.IntegrationID,
		&result.ScanType,
		&result.Status,
		&result.ModelID,
		&result.StartedAt,
		&completedAt,
		&result.DurationMs,
		&errorStr,
		&result.FindingsCount,
		&result.FilesAffected,
		&result.TokensUsed,
		&resultsJSON,
		&result.ProjectName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get scan result: %w", err)
	}

	if completedAt.Valid {
		result.CompletedAt = &completedAt.Time
	}
	result.Error = errorStr.String
	result.ResultsJSON = resultsJSON.String

	return &result, nil
}

// ListScanResults lists scan results with filtering
func (s *PostgresStore) ListScanResults(ctx context.Context, req *models.GitLabScanResultsRequest) ([]models.GitLabScanResult, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	if req.ProjectID != "" {
		conditions = append(conditions, fmt.Sprintf("sh.project_id = $%d", argNum))
		args = append(args, req.ProjectID)
		argNum++
	}

	if req.ScanType != nil {
		conditions = append(conditions, fmt.Sprintf("sh.scan_type = $%d", argNum))
		args = append(args, *req.ScanType)
		argNum++
	}

	if req.Status != nil {
		conditions = append(conditions, fmt.Sprintf("sh.status = $%d", argNum))
		args = append(args, *req.Status)
		argNum++
	}

	if len(req.IntegrationIDs) > 0 {
		placeholders := make([]string, len(req.IntegrationIDs))
		for i := range req.IntegrationIDs {
			placeholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, req.IntegrationIDs[i])
			argNum++
		}
		conditions = append(conditions, fmt.Sprintf("sh.integration_id IN (%s)", strings.Join(placeholders, ", ")))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM gitlab_scan_history sh %s`, whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count scan results: %w", err)
	}

	// Data query
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := req.Offset

	query := fmt.Sprintf(`
		SELECT 
			sh.id::text, sh.project_id::text, COALESCE(sh.integration_id::text, '') as integration_id,
			sh.scan_type, sh.status, COALESCE(sh.model_id, '') as model_id,
			sh.started_at, sh.completed_at, sh.duration_ms, sh.error,
			COALESCE(sh.findings_count, 0) as findings_count,
			COALESCE(sh.files_affected, 0) as files_affected,
			COALESCE(sh.tokens_used, 0) as tokens_used,
			COALESCE(p.name, '') as project_name
		FROM gitlab_scan_history sh
		LEFT JOIN gitlab_projects p ON sh.project_id = p.id
		%s
		ORDER BY sh.started_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list scan results: %w", err)
	}
	defer rows.Close()

	var results []models.GitLabScanResult
	for rows.Next() {
		var result models.GitLabScanResult
		var completedAt sql.NullTime
		var errorStr sql.NullString

		if err := rows.Scan(
			&result.ID,
			&result.ProjectID,
			&result.IntegrationID,
			&result.ScanType,
			&result.Status,
			&result.ModelID,
			&result.StartedAt,
			&completedAt,
			&result.DurationMs,
			&errorStr,
			&result.FindingsCount,
			&result.FilesAffected,
			&result.TokensUsed,
			&result.ProjectName,
		); err != nil {
			return nil, 0, fmt.Errorf("scan scan result row: %w", err)
		}

		if completedAt.Valid {
			result.CompletedAt = &completedAt.Time
		}
		result.Error = errorStr.String

		results = append(results, result)
	}

	return results, total, nil
}

// DeleteOldScanResults deletes scan results older than specified days
func (s *PostgresStore) DeleteOldScanResults(ctx context.Context, olderThanDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -olderThanDays)

	query := `DELETE FROM gitlab_scan_history WHERE started_at < $1`
	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old scan results: %w", err)
	}

	return result.RowsAffected()
}

// Helper to marshal results to JSON for storage
func MarshalScanResults(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
