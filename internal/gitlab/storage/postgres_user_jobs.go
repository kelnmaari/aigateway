// Package storage provides PostgreSQL storage for user jobs
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aigateway/internal/models"
)

// CreateUserJob creates a new user job
func (s *PostgresStore) CreateUserJob(ctx context.Context, job *models.UserJob) error {
	query := `
		INSERT INTO user_jobs (
			user_id, tenant_id, project_id, integration_id,
			job_type, status, config, progress, progress_msg, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	configJSON := "{}"
	if job.Config != "" {
		configJSON = job.Config
	}

	now := time.Now()
	job.CreatedAt = now
	if job.Status == "" {
		job.Status = models.UserJobStatusPending
	}

	var tenantID *string
	if job.TenantID != "" {
		tenantID = &job.TenantID
	}

	err := s.db.QueryRowContext(ctx, query,
		job.UserID,
		tenantID,
		job.ProjectID,
		job.IntegrationID,
		job.JobType,
		job.Status,
		configJSON,
		job.Progress,
		job.ProgressMsg,
		now,
	).Scan(&job.ID)

	if err != nil {
		return fmt.Errorf("create user job: %w", err)
	}

	return nil
}

// GetUserJob retrieves a user job by ID
func (s *PostgresStore) GetUserJob(ctx context.Context, id string) (*models.UserJob, error) {
	query := `
		SELECT
			uj.id, uj.user_id, uj.tenant_id, uj.project_id, uj.integration_id,
			uj.job_type, uj.status, uj.config, uj.progress, uj.progress_msg,
			uj.created_at, uj.started_at, uj.completed_at,
			uj.result_id, uj.result_type, uj.result_url, uj.error,
			p.name as project_name
		FROM user_jobs uj
		LEFT JOIN gitlab_projects p ON uj.project_id = p.id
		WHERE uj.id = $1
	`

	var job models.UserJob
	var tenantID, resultID, resultType, resultURL, errMsg, progressMsg, config sql.NullString
	var startedAt, completedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID, &job.UserID, &tenantID, &job.ProjectID, &job.IntegrationID,
		&job.JobType, &job.Status, &config, &job.Progress, &progressMsg,
		&job.CreatedAt, &startedAt, &completedAt,
		&resultID, &resultType, &resultURL, &errMsg,
		&job.ProjectName,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user job not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get user job: %w", err)
	}

	if tenantID.Valid {
		job.TenantID = tenantID.String
	}
	if startedAt.Valid {
		job.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if resultID.Valid {
		job.ResultID = resultID.String
	}
	if resultType.Valid {
		job.ResultType = resultType.String
	}
	if resultURL.Valid {
		job.ResultURL = resultURL.String
	}
	if errMsg.Valid {
		job.Error = errMsg.String
	}
	if progressMsg.Valid {
		job.ProgressMsg = progressMsg.String
	}
	if config.Valid {
		job.Config = config.String
	}

	return &job, nil
}

// ListUserJobs lists user jobs with filtering
func (s *PostgresStore) ListUserJobs(ctx context.Context, req *models.UserJobsRequest) ([]models.UserJob, int, error) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if req.UserID != "" {
		conditions = append(conditions, fmt.Sprintf("uj.user_id = $%d", argIndex))
		args = append(args, req.UserID)
		argIndex++
	}

	if req.ProjectID != "" {
		conditions = append(conditions, fmt.Sprintf("uj.project_id = $%d", argIndex))
		args = append(args, req.ProjectID)
		argIndex++
	}

	if req.IntegrationID != "" {
		conditions = append(conditions, fmt.Sprintf("uj.integration_id = $%d", argIndex))
		args = append(args, req.IntegrationID)
		argIndex++
	}

	if req.JobType != nil {
		conditions = append(conditions, fmt.Sprintf("uj.job_type = $%d", argIndex))
		args = append(args, *req.JobType)
		argIndex++
	}

	if req.Status != nil {
		conditions = append(conditions, fmt.Sprintf("uj.status = $%d", argIndex))
		args = append(args, *req.Status)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM user_jobs uj %s", whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user jobs: %w", err)
	}

	// Fetch jobs
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := req.Offset

	query := fmt.Sprintf(`
		SELECT
			uj.id, uj.user_id, uj.tenant_id, uj.project_id, uj.integration_id,
			uj.job_type, uj.status, uj.config, uj.progress, uj.progress_msg,
			uj.created_at, uj.started_at, uj.completed_at,
			uj.result_id, uj.result_type, uj.result_url, uj.error,
			p.name as project_name
		FROM user_jobs uj
		LEFT JOIN gitlab_projects p ON uj.project_id = p.id
		%s
		ORDER BY uj.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list user jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.UserJob
	for rows.Next() {
		var job models.UserJob
		var tenantID, resultID, resultType, resultURL, errMsg, progressMsg, config sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(
			&job.ID, &job.UserID, &tenantID, &job.ProjectID, &job.IntegrationID,
			&job.JobType, &job.Status, &config, &job.Progress, &progressMsg,
			&job.CreatedAt, &startedAt, &completedAt,
			&resultID, &resultType, &resultURL, &errMsg,
			&job.ProjectName,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user job: %w", err)
		}

		if tenantID.Valid {
			job.TenantID = tenantID.String
		}
		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		if resultID.Valid {
			job.ResultID = resultID.String
		}
		if resultType.Valid {
			job.ResultType = resultType.String
		}
		if resultURL.Valid {
			job.ResultURL = resultURL.String
		}
		if errMsg.Valid {
			job.Error = errMsg.String
		}
		if progressMsg.Valid {
			job.ProgressMsg = progressMsg.String
		}
		if config.Valid {
			job.Config = config.String
		}

		jobs = append(jobs, job)
	}

	return jobs, total, nil
}

// UpdateUserJobStatus updates job status
func (s *PostgresStore) UpdateUserJobStatus(ctx context.Context, id string, status models.UserJobStatus, errMsg string) error {
	var query string
	var args []interface{}

	now := time.Now()

	switch status {
	case models.UserJobStatusRunning:
		query = `UPDATE user_jobs SET status = $1, started_at = $2 WHERE id = $3`
		args = []interface{}{status, now, id}
	case models.UserJobStatusCompleted, models.UserJobStatusFailed, models.UserJobStatusCancelled:
		query = `UPDATE user_jobs SET status = $1, completed_at = $2, error = $3 WHERE id = $4`
		args = []interface{}{status, now, errMsg, id}
	default:
		query = `UPDATE user_jobs SET status = $1 WHERE id = $2`
		args = []interface{}{status, id}
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update user job status: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user job not found: %s", id)
	}

	return nil
}

// UpdateUserJobProgress updates job progress
func (s *PostgresStore) UpdateUserJobProgress(ctx context.Context, id string, progress int, progressMsg string) error {
	query := `UPDATE user_jobs SET progress = $1, progress_msg = $2 WHERE id = $3`

	result, err := s.db.ExecContext(ctx, query, progress, progressMsg, id)
	if err != nil {
		return fmt.Errorf("update user job progress: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user job not found: %s", id)
	}

	return nil
}

// UpdateUserJobResult updates job result
func (s *PostgresStore) UpdateUserJobResult(ctx context.Context, id string, resultID, resultType, resultURL string) error {
	query := `UPDATE user_jobs SET result_id = $1, result_type = $2, result_url = $3 WHERE id = $4`

	var resID, resType, resURL *string
	if resultID != "" {
		resID = &resultID
	}
	if resultType != "" {
		resType = &resultType
	}
	if resultURL != "" {
		resURL = &resultURL
	}

	result, err := s.db.ExecContext(ctx, query, resID, resType, resURL, id)
	if err != nil {
		return fmt.Errorf("update user job result: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user job not found: %s", id)
	}

	return nil
}

// CancelUserJob cancels a pending or running job
func (s *PostgresStore) CancelUserJob(ctx context.Context, id string) error {
	query := `
		UPDATE user_jobs
		SET status = $1, completed_at = $2
		WHERE id = $3 AND status IN ($4, $5)
	`

	result, err := s.db.ExecContext(ctx, query,
		models.UserJobStatusCancelled,
		time.Now(),
		id,
		models.UserJobStatusPending,
		models.UserJobStatusRunning,
	)
	if err != nil {
		return fmt.Errorf("cancel user job: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("job not found or already completed: %s", id)
	}

	return nil
}

// GetActiveUserJobs retrieves all running jobs for a user
func (s *PostgresStore) GetActiveUserJobs(ctx context.Context, userID string) ([]models.UserJob, error) {
	query := `
		SELECT
			uj.id, uj.user_id, uj.tenant_id, uj.project_id, uj.integration_id,
			uj.job_type, uj.status, uj.config, uj.progress, uj.progress_msg,
			uj.created_at, uj.started_at, uj.completed_at,
			uj.result_id, uj.result_type, uj.result_url, uj.error,
			p.name as project_name
		FROM user_jobs uj
		LEFT JOIN gitlab_projects p ON uj.project_id = p.id
		WHERE uj.user_id = $1 AND uj.status IN ($2, $3)
		ORDER BY uj.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, models.UserJobStatusPending, models.UserJobStatusRunning)
	if err != nil {
		return nil, fmt.Errorf("get active user jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.UserJob
	for rows.Next() {
		var job models.UserJob
		var tenantID, resultID, resultType, resultURL, errMsg, progressMsg, config sql.NullString
		var startedAt, completedAt sql.NullTime

		if err := rows.Scan(
			&job.ID, &job.UserID, &tenantID, &job.ProjectID, &job.IntegrationID,
			&job.JobType, &job.Status, &config, &job.Progress, &progressMsg,
			&job.CreatedAt, &startedAt, &completedAt,
			&resultID, &resultType, &resultURL, &errMsg,
			&job.ProjectName,
		); err != nil {
			return nil, fmt.Errorf("scan active user job: %w", err)
		}

		if tenantID.Valid {
			job.TenantID = tenantID.String
		}
		if startedAt.Valid {
			job.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			job.CompletedAt = &completedAt.Time
		}
		if resultID.Valid {
			job.ResultID = resultID.String
		}
		if resultType.Valid {
			job.ResultType = resultType.String
		}
		if resultURL.Valid {
			job.ResultURL = resultURL.String
		}
		if errMsg.Valid {
			job.Error = errMsg.String
		}
		if progressMsg.Valid {
			job.ProgressMsg = progressMsg.String
		}
		if config.Valid {
			job.Config = config.String
		}

		jobs = append(jobs, job)
	}

	return jobs, nil
}

// CleanupOldUserJobs deletes jobs older than specified days
func (s *PostgresStore) CleanupOldUserJobs(ctx context.Context, olderThanDays int) (int64, error) {
	query := `
		DELETE FROM user_jobs
		WHERE created_at < NOW() - INTERVAL '1 day' * $1
		AND status IN ($2, $3, $4)
	`

	result, err := s.db.ExecContext(ctx, query,
		olderThanDays,
		models.UserJobStatusCompleted,
		models.UserJobStatusFailed,
		models.UserJobStatusCancelled,
	)
	if err != nil {
		return 0, fmt.Errorf("cleanup old user jobs: %w", err)
	}

	return result.RowsAffected()
}

// Helper to convert UserJobConfig to JSON string
func UserJobConfigToJSON(config *models.UserJobConfig) string {
	if config == nil {
		return "{}"
	}
	bytes, err := json.Marshal(config)
	if err != nil {
		return "{}"
	}
	return string(bytes)
}

// Helper to parse JSON config to UserJobConfig
func ParseUserJobConfig(configJSON string) (*models.UserJobConfig, error) {
	if configJSON == "" || configJSON == "{}" {
		return &models.UserJobConfig{}, nil
	}
	var config models.UserJobConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("parse job config: %w", err)
	}
	return &config, nil
}
