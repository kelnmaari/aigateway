// Package schedule provides scheduled dependency scanning for GitLab projects.
package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"aigateway/internal/gitlab/dependencies"
	"aigateway/internal/gitlab/dependencies/changelog"
	"aigateway/internal/models"
	"aigateway/internal/rag/vector"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// ProjectStore is the minimal interface for project access
type ProjectStore interface {
	GetProject(ctx context.Context, id string) (*models.GitLabProject, error)
}

// Scheduler manages scheduled dependency scans for GitLab projects.
type Scheduler struct {
	cron              *cron.Cron
	projectStore      ProjectStore
	scheduleStore     ScheduleStore
	vectorStore       *vector.QdrantStore
	scanner           *dependencies.Scanner
	changelogAnalyzer *changelog.Analyzer
	logger            *logrus.Logger
	llmBaseURL        string
	llmAPIKey         string

	mu       sync.RWMutex
	entries  map[string]cron.EntryID // scheduleID -> cronEntryID
	running  bool
}

// NewScheduler creates a new dependency scan scheduler.
func NewScheduler(
	projectStore ProjectStore,
	scheduleStore ScheduleStore,
	vectorStore *vector.QdrantStore,
	llmBaseURL, llmAPIKey string,
	logger *logrus.Logger,
) *Scheduler {
	return &Scheduler{
		cron:              cron.New(),
		projectStore:      projectStore,
		scheduleStore:     scheduleStore,
		vectorStore:       vectorStore,
		scanner:           dependencies.NewScanner(vectorStore, logger),
		changelogAnalyzer: changelog.NewAnalyzer(llmBaseURL, llmAPIKey, logger),
		logger:            logger,
		llmBaseURL:        llmBaseURL,
		llmAPIKey:         llmAPIKey,
		entries:           make(map[string]cron.EntryID),
		running:           false,
	}
}

// Start initializes all scheduled scans and begins the cron scheduler.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	s.logger.Info("Starting dependency scan scheduler")

	// Load all enabled schedules from database
	schedules, err := s.scheduleStore.GetEnabledSchedules(ctx)
	if err != nil {
		return fmt.Errorf("failed to load enabled schedules: %w", err)
	}

	// Register all schedules
	for _, schedule := range schedules {
		if err := s.addScheduleJob(schedule); err != nil {
			s.logger.WithError(err).WithField("schedule_id", schedule.ID).Error("Failed to add schedule job")
			continue
		}
	}

	// Start cron scheduler
	s.cron.Start()
	s.running = true

	s.logger.WithField("schedules", len(schedules)).Info("Dependency scan scheduler started")

	// Wait for context cancellation
	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	return nil
}

// Stop halts the scheduler and waits for running jobs to complete.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.logger.Info("Stopping dependency scan scheduler")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false
	s.logger.Info("Dependency scan scheduler stopped")
}

// AddSchedule adds a new scheduled scan.
func (s *Scheduler) AddSchedule(ctx context.Context, req *CreateScheduledScanRequest) (*ScheduledScan, error) {
	// Get project to validate and get integration ID
	project, err := s.projectStore.GetProject(ctx, req.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("project not found: %w", err)
	}

	now := time.Now()
	cronExpr := FrequencyToCron(req.Frequency, req.CronExpr)
	
	// Calculate next run time
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}
	nextRun := schedule.Next(now)

	scan := &ScheduledScan{
		ID:            uuid.New().String(),
		ProjectID:     req.ProjectID,
		IntegrationID: project.IntegrationID,
		ScanType:      req.ScanType,
		Frequency:     req.Frequency,
		CronExpr:      cronExpr,
		Enabled:       req.Enabled,
		NotifyEmail:   req.NotifyEmail,
		CreateIssue:   req.CreateIssue,
		OnlyBreaking:  req.OnlyBreaking,
		NextRunAt:     &nextRun,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Save to database
	if err := s.scheduleStore.CreateSchedule(ctx, scan); err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	// Register with cron if enabled
	if scan.Enabled {
		s.mu.Lock()
		if err := s.addScheduleJob(*scan); err != nil {
			s.mu.Unlock()
			s.logger.WithError(err).WithField("schedule_id", scan.ID).Error("Failed to register schedule with cron")
		} else {
			s.mu.Unlock()
		}
	}

	s.logger.WithFields(logrus.Fields{
		"schedule_id": scan.ID,
		"project_id":  scan.ProjectID,
		"scan_type":   scan.ScanType,
		"frequency":   scan.Frequency,
		"next_run":    scan.NextRunAt,
	}).Info("Created scheduled scan")

	return scan, nil
}

// UpdateSchedule updates an existing scheduled scan.
func (s *Scheduler) UpdateSchedule(ctx context.Context, id string, req *UpdateScheduledScanRequest) error {
	// Get current schedule
	schedule, err := s.scheduleStore.GetSchedule(ctx, id)
	if err != nil {
		return fmt.Errorf("schedule not found: %w", err)
	}

	// Update in database
	if err := s.scheduleStore.UpdateSchedule(ctx, id, req); err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	// Handle cron registration changes
	s.mu.Lock()
	defer s.mu.Unlock()

	wasEnabled := schedule.Enabled
	isEnabled := wasEnabled
	if req.Enabled != nil {
		isEnabled = *req.Enabled
	}

	// Remove old job if exists
	if entryID, exists := s.entries[id]; exists {
		s.cron.Remove(entryID)
		delete(s.entries, id)
	}

	// Add new job if enabled
	if isEnabled {
		// Reload schedule with updated values
		updatedSchedule, err := s.scheduleStore.GetSchedule(ctx, id)
		if err != nil {
			return nil // Already updated in DB, just log error
		}
		if err := s.addScheduleJob(*updatedSchedule); err != nil {
			s.logger.WithError(err).WithField("schedule_id", id).Error("Failed to re-register schedule with cron")
		}
	}

	s.logger.WithField("schedule_id", id).Info("Updated scheduled scan")
	return nil
}

// DeleteSchedule removes a scheduled scan.
func (s *Scheduler) DeleteSchedule(ctx context.Context, id string) error {
	// Remove from cron
	s.mu.Lock()
	if entryID, exists := s.entries[id]; exists {
		s.cron.Remove(entryID)
		delete(s.entries, id)
	}
	s.mu.Unlock()

	// Delete from database
	if err := s.scheduleStore.DeleteSchedule(ctx, id); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	s.logger.WithField("schedule_id", id).Info("Deleted scheduled scan")
	return nil
}

// TriggerManual manually triggers a scheduled scan.
func (s *Scheduler) TriggerManual(ctx context.Context, scheduleID string) (*ScanHistory, error) {
	schedule, err := s.scheduleStore.GetSchedule(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("schedule not found: %w", err)
	}

	return s.executeScan(ctx, *schedule)
}

// addScheduleJob adds a cron job for a schedule.
func (s *Scheduler) addScheduleJob(schedule ScheduledScan) error {
	cronExpr := schedule.CronExpr
	if cronExpr == "" {
		cronExpr = FrequencyToCron(schedule.Frequency, "")
	}

	// Capture schedule in closure
	sched := schedule

	entryID, err := s.cron.AddFunc(cronExpr, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		
		_, execErr := s.executeScan(ctx, sched)
		if execErr != nil {
			s.logger.WithError(execErr).WithField("schedule_id", sched.ID).Error("Scheduled scan failed")
		}
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	s.entries[schedule.ID] = entryID

	s.logger.WithFields(logrus.Fields{
		"schedule_id": schedule.ID,
		"cron_expr":   cronExpr,
		"entry_id":    entryID,
	}).Debug("Added cron job for schedule")

	return nil
}

// executeScan runs a dependency scan for a schedule.
func (s *Scheduler) executeScan(ctx context.Context, schedule ScheduledScan) (*ScanHistory, error) {
	startTime := time.Now()

	s.logger.WithFields(logrus.Fields{
		"schedule_id": schedule.ID,
		"project_id":  schedule.ProjectID,
		"scan_type":   schedule.ScanType,
	}).Info("Executing scheduled dependency scan")

	// Create history record
	history := &ScanHistory{
		ID:         uuid.New().String(),
		ScheduleID: schedule.ID,
		ProjectID:  schedule.ProjectID,
		ScanType:   schedule.ScanType,
		Status:     "running",
		StartedAt:  startTime,
	}

	if err := s.scheduleStore.CreateScanHistory(ctx, history); err != nil {
		s.logger.WithError(err).Error("Failed to create scan history record")
	}

	// Get project details
	project, err := s.projectStore.GetProject(ctx, schedule.ProjectID)
	if err != nil {
		return s.failScan(ctx, history, schedule, fmt.Errorf("project not found: %w", err))
	}

	// Execute the scan based on type
	var scanErr error
	switch schedule.ScanType {
	case ScanTypeDependencies:
		scanErr = s.executeDependencyScan(ctx, project, history, schedule)
	default:
		scanErr = fmt.Errorf("unsupported scan type: %s", schedule.ScanType)
	}

	if scanErr != nil {
		return s.failScan(ctx, history, schedule, scanErr)
	}

	// Mark success
	return s.completeScan(ctx, history, schedule)
}

// executeDependencyScan performs the actual dependency scan.
func (s *Scheduler) executeDependencyScan(ctx context.Context, project *models.GitLabProject, history *ScanHistory, schedule ScheduledScan) error {
	// Use scanner to check dependencies
	result, err := s.scanner.ScanProject(ctx, dependencies.ScanRequest{
		ProjectID:      schedule.ProjectID,
		CollectionName: project.GetCollectionName(),
	})
	if err != nil {
		return fmt.Errorf("dependency scan failed: %w", err)
	}

	// Update history with results
	history.DependenciesChecked = len(result.Dependencies)
	
	outdated := 0
	vulns := 0
	breaking := 0
	
	for _, depWithVulns := range result.Dependencies {
		if depWithVulns.Dependency.HasUpdate {
			outdated++
		}
		vulns += len(depWithVulns.Vulnerabilities)
	}
	
	history.OutdatedDependencies = outdated
	history.VulnerabilitiesFound = vulns

	// Analyze changelogs for breaking changes if there are outdated deps
	if outdated > 0 && project.AnalysisModelID != "" {
		for _, depWithVulns := range result.Dependencies {
			if !depWithVulns.Dependency.HasUpdate {
				continue
			}
			
			analysis, err := s.changelogAnalyzer.AnalyzeChangelog(ctx, changelog.AnalyzeRequest{
				PackageName:    depWithVulns.Dependency.Name,
				CurrentVersion: depWithVulns.Dependency.CurrentVersion,
				LatestVersion:  depWithVulns.Dependency.LatestVersion,
				Language:       result.Language,
				ModelID:        project.AnalysisModelID,
			})
			if err != nil {
				s.logger.WithError(err).WithField("package", depWithVulns.Dependency.Name).Debug("Failed to analyze changelog")
				continue
			}
			
			if len(analysis.BreakingChanges) > 0 {
				breaking++
			}
		}
	}
	
	history.BreakingChanges = breaking

	// Serialize results to JSON
	resultsJSON, _ := json.Marshal(result)
	history.ResultsJSON = string(resultsJSON)

	// Create GitLab issue if configured and there are findings
	if schedule.CreateIssue && (outdated > 0 || vulns > 0) {
		if schedule.OnlyBreaking && breaking == 0 {
			// Skip issue creation if only breaking changes should trigger
			s.logger.Debug("Skipping issue creation: no breaking changes found")
		} else {
			// TODO: Create GitLab issue with findings
			s.logger.Info("Would create GitLab issue with dependency findings")
		}
	}

	return nil
}

// failScan marks a scan as failed and updates records.
func (s *Scheduler) failScan(ctx context.Context, history *ScanHistory, schedule ScheduledScan, err error) (*ScanHistory, error) {
	now := time.Now()
	duration := now.Sub(history.StartedAt).Milliseconds()
	
	history.Status = "failed"
	history.CompletedAt = &now
	history.DurationMs = duration
	history.Error = err.Error()

	// Calculate next run
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSchedule, _ := parser.Parse(schedule.CronExpr)
	nextRun := cronSchedule.Next(now)

	// Update schedule execution status
	if updateErr := s.scheduleStore.UpdateScheduleExecution(ctx, schedule.ID, "failed", err.Error(), duration, nextRun); updateErr != nil {
		s.logger.WithError(updateErr).Error("Failed to update schedule execution status")
	}

	s.logger.WithFields(logrus.Fields{
		"schedule_id": schedule.ID,
		"history_id":  history.ID,
		"error":       err.Error(),
		"duration_ms": duration,
	}).Error("Scheduled scan failed")

	return history, err
}

// completeScan marks a scan as successful and updates records.
func (s *Scheduler) completeScan(ctx context.Context, history *ScanHistory, schedule ScheduledScan) (*ScanHistory, error) {
	now := time.Now()
	duration := now.Sub(history.StartedAt).Milliseconds()
	
	history.Status = "success"
	history.CompletedAt = &now
	history.DurationMs = duration

	// Calculate next run
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	cronSchedule, _ := parser.Parse(schedule.CronExpr)
	nextRun := cronSchedule.Next(now)

	// Update schedule execution status
	if updateErr := s.scheduleStore.UpdateScheduleExecution(ctx, schedule.ID, "success", "", duration, nextRun); updateErr != nil {
		s.logger.WithError(updateErr).Error("Failed to update schedule execution status")
	}

	s.logger.WithFields(logrus.Fields{
		"schedule_id":        schedule.ID,
		"history_id":         history.ID,
		"duration_ms":        duration,
		"dependencies":       history.DependenciesChecked,
		"outdated":           history.OutdatedDependencies,
		"vulnerabilities":    history.VulnerabilitiesFound,
		"breaking_changes":   history.BreakingChanges,
	}).Info("Scheduled scan completed successfully")

	return history, nil
}

// IsRunning returns whether the scheduler is currently running.
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetScheduleCount returns the number of active schedules.
func (s *Scheduler) GetScheduleCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

