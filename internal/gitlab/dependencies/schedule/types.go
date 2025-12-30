// Package schedule provides types for scheduled dependency scanning.
package schedule

import (
	"context"
	"time"
)

// ScheduleFrequency represents how often to run the scan
type ScheduleFrequency string

const (
	FrequencyDaily   ScheduleFrequency = "daily"
	FrequencyWeekly  ScheduleFrequency = "weekly"
	FrequencyMonthly ScheduleFrequency = "monthly"
	FrequencyCustom  ScheduleFrequency = "custom" // Custom cron expression
)

// ScanType represents what type of scan to perform
type ScanType string

const (
	ScanTypeDependencies ScanType = "dependencies"
	ScanTypeSecrets      ScanType = "secrets"
	ScanTypeQuality      ScanType = "quality"
	ScanTypeDeadCode     ScanType = "dead_code"
)

// ScheduledScan represents a scheduled scan configuration
type ScheduledScan struct {
	ID            string            `json:"id" db:"id"`
	ProjectID     string            `json:"project_id" db:"project_id"`
	IntegrationID string            `json:"integration_id" db:"integration_id"`
	ScanType      ScanType          `json:"scan_type" db:"scan_type"`
	Frequency     ScheduleFrequency `json:"frequency" db:"frequency"`
	CronExpr      string            `json:"cron_expr,omitempty" db:"cron_expr"`   // For custom frequency
	Enabled       bool              `json:"enabled" db:"enabled"`
	
	// Notification settings
	NotifyEmail       string `json:"notify_email,omitempty" db:"notify_email"`
	CreateIssue       bool   `json:"create_issue" db:"create_issue"`           // Create GitLab issue on findings
	OnlyBreaking      bool   `json:"only_breaking" db:"only_breaking"`         // Only notify on breaking changes
	
	// Execution tracking
	LastRunAt         *time.Time `json:"last_run_at,omitempty" db:"last_run_at"`
	NextRunAt         *time.Time `json:"next_run_at,omitempty" db:"next_run_at"`
	LastRunStatus     string     `json:"last_run_status,omitempty" db:"last_run_status"` // success, failed, running
	LastRunError      string     `json:"last_run_error,omitempty" db:"last_run_error"`
	LastRunDurationMs int64      `json:"last_run_duration_ms,omitempty" db:"last_run_duration_ms"`
	
	// Statistics
	TotalRuns       int `json:"total_runs" db:"total_runs"`
	SuccessfulRuns  int `json:"successful_runs" db:"successful_runs"`
	FailedRuns      int `json:"failed_runs" db:"failed_runs"`
	
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
	
	// Computed fields
	ProjectName   string `json:"project_name,omitempty" db:"-"`
	IntegrationName string `json:"integration_name,omitempty" db:"-"`
}

// ScanHistory represents a single scan execution record
type ScanHistory struct {
	ID              string     `json:"id" db:"id"`
	ScheduleID      string     `json:"schedule_id" db:"schedule_id"`
	ProjectID       string     `json:"project_id" db:"project_id"`
	ScanType        ScanType   `json:"scan_type" db:"scan_type"`
	Status          string     `json:"status" db:"status"`           // pending, running, success, failed
	StartedAt       time.Time  `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs      int64      `json:"duration_ms,omitempty" db:"duration_ms"`
	Error           string     `json:"error,omitempty" db:"error"`
	
	// Results summary
	DependenciesChecked   int `json:"dependencies_checked,omitempty" db:"dependencies_checked"`
	OutdatedDependencies  int `json:"outdated_dependencies,omitempty" db:"outdated_dependencies"`
	VulnerabilitiesFound  int `json:"vulnerabilities_found,omitempty" db:"vulnerabilities_found"`
	BreakingChanges       int `json:"breaking_changes,omitempty" db:"breaking_changes"`
	IssueCreated          bool   `json:"issue_created" db:"issue_created"`
	IssueURL              string `json:"issue_url,omitempty" db:"issue_url"`
	
	// Full results (JSON)
	ResultsJSON string `json:"results_json,omitempty" db:"results_json"`
}

// CreateScheduledScanRequest is the request to create a scheduled scan
type CreateScheduledScanRequest struct {
	ProjectID     string            `json:"project_id" binding:"required"`
	ScanType      ScanType          `json:"scan_type" binding:"required"`
	Frequency     ScheduleFrequency `json:"frequency" binding:"required"`
	CronExpr      string            `json:"cron_expr,omitempty"`
	Enabled       bool              `json:"enabled"`
	NotifyEmail   string            `json:"notify_email,omitempty"`
	CreateIssue   bool              `json:"create_issue"`
	OnlyBreaking  bool              `json:"only_breaking"`
}

// UpdateScheduledScanRequest is the request to update a scheduled scan
type UpdateScheduledScanRequest struct {
	Frequency     *ScheduleFrequency `json:"frequency,omitempty"`
	CronExpr      *string            `json:"cron_expr,omitempty"`
	Enabled       *bool              `json:"enabled,omitempty"`
	NotifyEmail   *string            `json:"notify_email,omitempty"`
	CreateIssue   *bool              `json:"create_issue,omitempty"`
	OnlyBreaking  *bool              `json:"only_breaking,omitempty"`
}

// ListScheduledScansRequest is the request to list scheduled scans
type ListScheduledScansRequest struct {
	ProjectID     string   `form:"project_id,omitempty"`
	IntegrationID string   `form:"integration_id,omitempty"`
	ScanType      ScanType `form:"scan_type,omitempty"`
	Enabled       *bool    `form:"enabled,omitempty"`
	Limit         int      `form:"limit,omitempty"`
	Offset        int      `form:"offset,omitempty"`
}

// ScheduleStore manages scheduled scan persistence.
type ScheduleStore interface {
	// CreateSchedule creates a new scheduled scan
	CreateSchedule(ctx context.Context, schedule *ScheduledScan) error
	
	// GetSchedule retrieves a schedule by ID
	GetSchedule(ctx context.Context, id string) (*ScheduledScan, error)
	
	// ListSchedules lists schedules with filtering
	ListSchedules(ctx context.Context, req *ListScheduledScansRequest) ([]ScheduledScan, int, error)
	
	// UpdateSchedule updates a schedule
	UpdateSchedule(ctx context.Context, id string, req *UpdateScheduledScanRequest) error
	
	// DeleteSchedule deletes a schedule
	DeleteSchedule(ctx context.Context, id string) error
	
	// UpdateScheduleExecution updates execution status after a run
	UpdateScheduleExecution(ctx context.Context, id string, status string, err string, durationMs int64, nextRun time.Time) error
	
	// CreateScanHistory creates a scan history record
	CreateScanHistory(ctx context.Context, history *ScanHistory) error
	
	// ListScanHistory lists scan history for a schedule
	ListScanHistory(ctx context.Context, scheduleID string, limit, offset int) ([]ScanHistory, int, error)
	
	// GetEnabledSchedules returns all enabled schedules
	GetEnabledSchedules(ctx context.Context) ([]ScheduledScan, error)
}

// FrequencyToCron converts a frequency to its cron expression
func FrequencyToCron(freq ScheduleFrequency, custom string) string {
	switch freq {
	case FrequencyDaily:
		return "0 6 * * *" // Every day at 6:00 AM
	case FrequencyWeekly:
		return "0 6 * * 1" // Every Monday at 6:00 AM
	case FrequencyMonthly:
		return "0 6 1 * *" // First day of month at 6:00 AM
	case FrequencyCustom:
		return custom
	default:
		return "0 6 * * *" // Default to daily
	}
}

