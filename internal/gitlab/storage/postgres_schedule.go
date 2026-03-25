// Package storage provides PostgreSQL implementation for scheduled scan data access
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"aigateway/internal/gitlab/dependencies/schedule"

	"github.com/google/uuid"
)

// ============================================================================
// Schedule Store Implementation
// ============================================================================

// Ensure PostgresStore implements schedule.ScheduleStore
var _ schedule.ScheduleStore = (*PostgresStore)(nil)

// CreateSchedule creates a new scheduled scan
func (s *PostgresStore) CreateSchedule(ctx context.Context, schedule *schedule.ScheduledScan) error {
	if schedule.ID == "" {
		schedule.ID = uuid.New().String()
	}

	query := `
		INSERT INTO gitlab_scheduled_scans (
			id, project_id, integration_id, scan_type, frequency, cron_expr, enabled,
			notify_email, create_issue, only_breaking,
			last_run_at, next_run_at, last_run_status, last_run_error, last_run_duration_ms,
			total_runs, successful_runs, failed_runs,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	_, err := s.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.ProjectID,
		schedule.IntegrationID,
		schedule.ScanType,
		schedule.Frequency,
		schedule.CronExpr,
		schedule.Enabled,
		nullString(schedule.NotifyEmail),
		schedule.CreateIssue,
		schedule.OnlyBreaking,
		nullTime(schedule.LastRunAt),
		nullTime(schedule.NextRunAt),
		nullString(schedule.LastRunStatus),
		nullString(schedule.LastRunError),
		schedule.LastRunDurationMs,
		schedule.TotalRuns,
		schedule.SuccessfulRuns,
		schedule.FailedRuns,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert scheduled scan: %w", err)
	}

	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *PostgresStore) GetSchedule(ctx context.Context, id string) (*schedule.ScheduledScan, error) {
	query := `
		SELECT 
			ss.id, ss.project_id, ss.integration_id, ss.scan_type, ss.frequency, ss.cron_expr, ss.enabled,
			ss.notify_email, ss.create_issue, ss.only_breaking,
			ss.last_run_at, ss.next_run_at, ss.last_run_status, ss.last_run_error, ss.last_run_duration_ms,
			ss.total_runs, ss.successful_runs, ss.failed_runs,
			ss.created_at, ss.updated_at,
			COALESCE(p.name, '') as project_name,
			COALESCE(i.name, '') as integration_name
		FROM gitlab_scheduled_scans ss
		LEFT JOIN gitlab_projects p ON ss.project_id = p.id
		LEFT JOIN gitlab_integrations i ON ss.integration_id = i.id
		WHERE ss.id = $1
	`

	var schedule schedule.ScheduledScan
	var notifyEmail, lastRunStatus, lastRunError sql.NullString
	var lastRunAt, nextRunAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&schedule.ID,
		&schedule.ProjectID,
		&schedule.IntegrationID,
		&schedule.ScanType,
		&schedule.Frequency,
		&schedule.CronExpr,
		&schedule.Enabled,
		&notifyEmail,
		&schedule.CreateIssue,
		&schedule.OnlyBreaking,
		&lastRunAt,
		&nextRunAt,
		&lastRunStatus,
		&lastRunError,
		&schedule.LastRunDurationMs,
		&schedule.TotalRuns,
		&schedule.SuccessfulRuns,
		&schedule.FailedRuns,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
		&schedule.ProjectName,
		&schedule.IntegrationName,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("query schedule: %w", err)
	}

	schedule.NotifyEmail = notifyEmail.String
	schedule.LastRunStatus = lastRunStatus.String
	schedule.LastRunError = lastRunError.String
	if lastRunAt.Valid {
		schedule.LastRunAt = &lastRunAt.Time
	}
	if nextRunAt.Valid {
		schedule.NextRunAt = &nextRunAt.Time
	}

	return &schedule, nil
}

// ListSchedules lists schedules with filtering
func (s *PostgresStore) ListSchedules(ctx context.Context, req *schedule.ListScheduledScansRequest) ([]schedule.ScheduledScan, int, error) {
	var conditions []string
	var args []any
	argIndex := 1

	if req.ProjectID != "" {
		conditions = append(conditions, fmt.Sprintf("ss.project_id = $%d", argIndex))
		args = append(args, req.ProjectID)
		argIndex++
	}
	if req.IntegrationID != "" {
		conditions = append(conditions, fmt.Sprintf("ss.integration_id = $%d", argIndex))
		args = append(args, req.IntegrationID)
		argIndex++
	}
	if req.ScanType != "" {
		conditions = append(conditions, fmt.Sprintf("ss.scan_type = $%d", argIndex))
		args = append(args, req.ScanType)
		argIndex++
	}
	if req.Enabled != nil {
		conditions = append(conditions, fmt.Sprintf("ss.enabled = $%d", argIndex))
		args = append(args, *req.Enabled)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + joinConditions(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM gitlab_scheduled_scans ss %s`, whereClause)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count schedules: %w", err)
	}

	// Get items
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := req.Offset

	listQuery := fmt.Sprintf(`
		SELECT 
			ss.id, ss.project_id, ss.integration_id, ss.scan_type, ss.frequency, ss.cron_expr, ss.enabled,
			ss.notify_email, ss.create_issue, ss.only_breaking,
			ss.last_run_at, ss.next_run_at, ss.last_run_status, ss.last_run_error, ss.last_run_duration_ms,
			ss.total_runs, ss.successful_runs, ss.failed_runs,
			ss.created_at, ss.updated_at,
			COALESCE(p.name, '') as project_name,
			COALESCE(i.name, '') as integration_name
		FROM gitlab_scheduled_scans ss
		LEFT JOIN gitlab_projects p ON ss.project_id = p.id
		LEFT JOIN gitlab_integrations i ON ss.integration_id = i.id
		%s
		ORDER BY ss.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query schedules: %w", err)
	}
	defer rows.Close()

	var schedules []schedule.ScheduledScan
	for rows.Next() {
		var schedule schedule.ScheduledScan
		var notifyEmail, lastRunStatus, lastRunError sql.NullString
		var lastRunAt, nextRunAt sql.NullTime

		err := rows.Scan(
			&schedule.ID,
			&schedule.ProjectID,
			&schedule.IntegrationID,
			&schedule.ScanType,
			&schedule.Frequency,
			&schedule.CronExpr,
			&schedule.Enabled,
			&notifyEmail,
			&schedule.CreateIssue,
			&schedule.OnlyBreaking,
			&lastRunAt,
			&nextRunAt,
			&lastRunStatus,
			&lastRunError,
			&schedule.LastRunDurationMs,
			&schedule.TotalRuns,
			&schedule.SuccessfulRuns,
			&schedule.FailedRuns,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
			&schedule.ProjectName,
			&schedule.IntegrationName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan schedule row: %w", err)
		}

		schedule.NotifyEmail = notifyEmail.String
		schedule.LastRunStatus = lastRunStatus.String
		schedule.LastRunError = lastRunError.String
		if lastRunAt.Valid {
			schedule.LastRunAt = &lastRunAt.Time
		}
		if nextRunAt.Valid {
			schedule.NextRunAt = &nextRunAt.Time
		}

		schedules = append(schedules, schedule)
	}

	return schedules, total, nil
}

// UpdateSchedule updates a schedule
func (s *PostgresStore) UpdateSchedule(ctx context.Context, id string, req *schedule.UpdateScheduledScanRequest) error {
	var updates []string
	var args []any
	argIndex := 1

	if req.Frequency != nil {
		updates = append(updates, fmt.Sprintf("frequency = $%d", argIndex))
		args = append(args, *req.Frequency)
		argIndex++

		// Also update cron expression
		cronExpr := schedule.FrequencyToCron(*req.Frequency, "")
		if req.CronExpr != nil {
			cronExpr = *req.CronExpr
		}
		updates = append(updates, fmt.Sprintf("cron_expr = $%d", argIndex))
		args = append(args, cronExpr)
		argIndex++
	} else if req.CronExpr != nil {
		updates = append(updates, fmt.Sprintf("cron_expr = $%d", argIndex))
		args = append(args, *req.CronExpr)
		argIndex++
	}

	if req.Enabled != nil {
		updates = append(updates, fmt.Sprintf("enabled = $%d", argIndex))
		args = append(args, *req.Enabled)
		argIndex++
	}

	if req.NotifyEmail != nil {
		updates = append(updates, fmt.Sprintf("notify_email = $%d", argIndex))
		args = append(args, nullString(*req.NotifyEmail))
		argIndex++
	}

	if req.CreateIssue != nil {
		updates = append(updates, fmt.Sprintf("create_issue = $%d", argIndex))
		args = append(args, *req.CreateIssue)
		argIndex++
	}

	if req.OnlyBreaking != nil {
		updates = append(updates, fmt.Sprintf("only_breaking = $%d", argIndex))
		args = append(args, *req.OnlyBreaking)
		argIndex++
	}

	if len(updates) == 0 {
		return nil
	}

	updates = append(updates, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE gitlab_scheduled_scans 
		SET %s 
		WHERE id = $%d
	`, joinConditions(updates, ", "), argIndex)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update schedule: %w", err)
	}

	return nil
}

// DeleteSchedule deletes a schedule
func (s *PostgresStore) DeleteSchedule(ctx context.Context, id string) error {
	query := `DELETE FROM gitlab_scheduled_scans WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return nil
}

// UpdateScheduleExecution updates execution status after a run
func (s *PostgresStore) UpdateScheduleExecution(ctx context.Context, id string, status string, errMsg string, durationMs int64, nextRun time.Time) error {
	now := time.Now()

	// Determine success/failure increment
	successInc := 0
	failedInc := 0
	if status == "success" {
		successInc = 1
	} else if status == "failed" {
		failedInc = 1
	}

	query := `
		UPDATE gitlab_scheduled_scans 
		SET 
			last_run_at = $1,
			next_run_at = $2,
			last_run_status = $3,
			last_run_error = $4,
			last_run_duration_ms = $5,
			total_runs = total_runs + 1,
			successful_runs = successful_runs + $6,
			failed_runs = failed_runs + $7,
			updated_at = $8
		WHERE id = $9
	`

	_, err := s.db.ExecContext(ctx, query,
		now,
		nextRun,
		status,
		nullString(errMsg),
		durationMs,
		successInc,
		failedInc,
		now,
		id,
	)
	if err != nil {
		return fmt.Errorf("update schedule execution: %w", err)
	}

	return nil
}

// CreateScanHistory creates a scan history record
func (s *PostgresStore) CreateScanHistory(ctx context.Context, history *schedule.ScanHistory) error {
	if history.ID == "" {
		history.ID = uuid.New().String()
	}

	query := `
		INSERT INTO gitlab_scan_history (
			id, schedule_id, project_id, scan_type, status, started_at, completed_at, duration_ms, error,
			dependencies_checked, outdated_dependencies, vulnerabilities_found, breaking_changes,
			issue_created, issue_url, results_json
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err := s.db.ExecContext(ctx, query,
		history.ID,
		history.ScheduleID,
		history.ProjectID,
		history.ScanType,
		history.Status,
		history.StartedAt,
		nullTime(history.CompletedAt),
		history.DurationMs,
		nullString(history.Error),
		history.DependenciesChecked,
		history.OutdatedDependencies,
		history.VulnerabilitiesFound,
		history.BreakingChanges,
		history.IssueCreated,
		nullString(history.IssueURL),
		nullString(history.ResultsJSON),
	)
	if err != nil {
		return fmt.Errorf("insert scan history: %w", err)
	}

	return nil
}

// ListScanHistory lists scan history for a schedule
func (s *PostgresStore) ListScanHistory(ctx context.Context, scheduleID string, limit, offset int) ([]schedule.ScanHistory, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM gitlab_scan_history WHERE schedule_id = $1`
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, scheduleID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count scan history: %w", err)
	}

	if limit <= 0 {
		limit = 20
	}

	listQuery := `
		SELECT 
			id, schedule_id, project_id, scan_type, status, started_at, completed_at, duration_ms, error,
			dependencies_checked, outdated_dependencies, vulnerabilities_found, breaking_changes,
			issue_created, issue_url, results_json
		FROM gitlab_scan_history
		WHERE schedule_id = $1
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, listQuery, scheduleID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query scan history: %w", err)
	}
	defer rows.Close()

	var history []schedule.ScanHistory
	for rows.Next() {
		var h schedule.ScanHistory
		var completedAt sql.NullTime
		var errStr, issueURL, resultsJSON sql.NullString

		err := rows.Scan(
			&h.ID,
			&h.ScheduleID,
			&h.ProjectID,
			&h.ScanType,
			&h.Status,
			&h.StartedAt,
			&completedAt,
			&h.DurationMs,
			&errStr,
			&h.DependenciesChecked,
			&h.OutdatedDependencies,
			&h.VulnerabilitiesFound,
			&h.BreakingChanges,
			&h.IssueCreated,
			&issueURL,
			&resultsJSON,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan history row: %w", err)
		}

		if completedAt.Valid {
			h.CompletedAt = &completedAt.Time
		}
		h.Error = errStr.String
		h.IssueURL = issueURL.String
		h.ResultsJSON = resultsJSON.String

		history = append(history, h)
	}

	return history, total, nil
}

// GetEnabledSchedules returns all enabled schedules
func (s *PostgresStore) GetEnabledSchedules(ctx context.Context) ([]schedule.ScheduledScan, error) {
	query := `
		SELECT 
			ss.id, ss.project_id, ss.integration_id, ss.scan_type, ss.frequency, ss.cron_expr, ss.enabled,
			ss.notify_email, ss.create_issue, ss.only_breaking,
			ss.last_run_at, ss.next_run_at, ss.last_run_status, ss.last_run_error, ss.last_run_duration_ms,
			ss.total_runs, ss.successful_runs, ss.failed_runs,
			ss.created_at, ss.updated_at,
			COALESCE(p.name, '') as project_name,
			COALESCE(i.name, '') as integration_name
		FROM gitlab_scheduled_scans ss
		LEFT JOIN gitlab_projects p ON ss.project_id = p.id
		LEFT JOIN gitlab_integrations i ON ss.integration_id = i.id
		WHERE ss.enabled = true
		ORDER BY ss.next_run_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query enabled schedules: %w", err)
	}
	defer rows.Close()

	var schedules []schedule.ScheduledScan
	for rows.Next() {
		var schedule schedule.ScheduledScan
		var notifyEmail, lastRunStatus, lastRunError sql.NullString
		var lastRunAt, nextRunAt sql.NullTime

		err := rows.Scan(
			&schedule.ID,
			&schedule.ProjectID,
			&schedule.IntegrationID,
			&schedule.ScanType,
			&schedule.Frequency,
			&schedule.CronExpr,
			&schedule.Enabled,
			&notifyEmail,
			&schedule.CreateIssue,
			&schedule.OnlyBreaking,
			&lastRunAt,
			&nextRunAt,
			&lastRunStatus,
			&lastRunError,
			&schedule.LastRunDurationMs,
			&schedule.TotalRuns,
			&schedule.SuccessfulRuns,
			&schedule.FailedRuns,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
			&schedule.ProjectName,
			&schedule.IntegrationName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan schedule row: %w", err)
		}

		schedule.NotifyEmail = notifyEmail.String
		schedule.LastRunStatus = lastRunStatus.String
		schedule.LastRunError = lastRunError.String
		if lastRunAt.Valid {
			schedule.LastRunAt = &lastRunAt.Time
		}
		if nextRunAt.Valid {
			schedule.NextRunAt = &nextRunAt.Time
		}

		schedules = append(schedules, schedule)
	}

	return schedules, nil
}

// Helper functions

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func nullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func joinConditions(conditions []string, sep string) string {
	if len(conditions) == 0 {
		return ""
	}
	var result strings.Builder
	result.WriteString(conditions[0])
	for i := 1; i < len(conditions); i++ {
		result.WriteString(sep + conditions[i])
	}
	return result.String()
}
