// Package jobs provides job executors for various scan types
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"aigateway/internal/gitlab/scanner"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"
	"aigateway/internal/rag/vector"

	"github.com/sirupsen/logrus"
)

// DeepScanExecutor executes deep secrets scan jobs
type DeepScanExecutor struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	logger      *logrus.Logger
}

// NewDeepScanExecutor creates a new deep scan executor
func NewDeepScanExecutor(
	store storage.Store,
	vectorStore *vector.QdrantStore,
	llmBaseURL, llmAPIKey string,
	logger *logrus.Logger,
) *DeepScanExecutor {
	return &DeepScanExecutor{
		store:       store,
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
}

// Execute runs the deep scan job
func (e *DeepScanExecutor) Execute(ctx context.Context, job *models.UserJob, config *models.UserJobConfig, progressCb func(progress int, msg string)) error {
	logger := e.logger.WithFields(logrus.Fields{
		"job_id":     job.ID,
		"project_id": job.ProjectID,
	})

	// Get project
	project, err := e.store.GetProject(ctx, job.ProjectID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}

	// Get collection name
	collectionName := project.GetCollectionName()
	if collectionName == "" {
		return fmt.Errorf("project has no indexed collection")
	}

	// Determine model
	modelID := config.ModelID
	if modelID == "" {
		modelID = project.AnalysisModelID
	}
	if modelID == "" {
		return fmt.Errorf("no analysis model configured")
	}

	// Determine language
	language := config.Language
	if language == "" {
		language = project.Settings.ReviewLanguage
	}
	if language == "" {
		language = "en"
	}

	logger.WithFields(logrus.Fields{
		"collection": collectionName,
		"model":      modelID,
		"language":   language,
	}).Info("Starting deep scan job")

	progressCb(5, "Initializing scanner...")

	// Create scanner
	deepScanner := scanner.NewDeepScanner(e.vectorStore, e.llmBaseURL, e.llmAPIKey, e.logger)

	// Create scan result record
	startTime := time.Now()
	scanResult := &models.GitLabScanResult{
		ProjectID:     job.ProjectID,
		IntegrationID: job.IntegrationID,
		ScanType:      models.ScanTypeSecretsDeep,
		Status:        models.ScanStatusRunning,
		ModelID:       modelID,
		StartedAt:     startTime,
	}

	// Progress callback wrapper
	scanProgressCb := func(progress scanner.ScanProgress) {
		pct := 5 + int(float64(progress.ChunksScanned)/float64(progress.TotalChunks)*90)
		msg := fmt.Sprintf("Scanning: %d/%d chunks, %d findings", progress.ChunksScanned, progress.TotalChunks, progress.FindingsCount)
		progressCb(pct, msg)
	}

	// Run scan
	maxChunks := config.MaxChunks
	if maxChunks <= 0 {
		maxChunks = 500
	}

	result, err := deepScanner.DeepScanWithProgress(ctx, scanner.DeepScanRequest{
		ProjectID:      job.ProjectID,
		CollectionName: collectionName,
		ModelID:        modelID,
		MaxChunks:      maxChunks,
		Language:       language,
	}, scanProgressCb)

	// Update scan result
	completedAt := time.Now()
	scanResult.CompletedAt = &completedAt
	scanResult.DurationMs = completedAt.Sub(startTime).Milliseconds()

	if err != nil {
		scanResult.Status = models.ScanStatusFailed
		scanResult.Error = err.Error()
		e.store.SaveScanResult(ctx, scanResult)
		return fmt.Errorf("deep scan failed: %w", err)
	}

	// Save successful result
	scanResult.Status = models.ScanStatusCompleted
	scanResult.FindingsCount = len(result.Findings)
	scanResult.FilesAffected = result.Summary.FilesAffected
	scanResult.TokensUsed = result.TokensUsed

	if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
		scanResult.ResultsJSON = string(resultJSON)
	}

	if err := e.store.SaveScanResult(ctx, scanResult); err != nil {
		logger.WithError(err).Warn("Failed to save scan result")
	}

	// Update job with result reference
	e.store.UpdateUserJobResult(ctx, job.ID, scanResult.ID, "scan_result", "")

	progressCb(100, fmt.Sprintf("Completed: %d findings", len(result.Findings)))

	logger.WithFields(logrus.Fields{
		"findings":   len(result.Findings),
		"duration":   result.Duration,
		"tokens":     result.TokensUsed,
		"result_id":  scanResult.ID,
	}).Info("Deep scan job completed")

	return nil
}

// SecretsScanExecutor executes regex-based secrets scan jobs
type SecretsScanExecutor struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	logger      *logrus.Logger
}

// NewSecretsScanExecutor creates a new secrets scan executor
func NewSecretsScanExecutor(store storage.Store, vectorStore *vector.QdrantStore, logger *logrus.Logger) *SecretsScanExecutor {
	return &SecretsScanExecutor{
		store:       store,
		vectorStore: vectorStore,
		logger:      logger,
	}
}

// Execute runs the secrets scan job
func (e *SecretsScanExecutor) Execute(ctx context.Context, job *models.UserJob, config *models.UserJobConfig, progressCb func(progress int, msg string)) error {
	logger := e.logger.WithFields(logrus.Fields{
		"job_id":     job.ID,
		"project_id": job.ProjectID,
	})

	// Get project
	project, err := e.store.GetProject(ctx, job.ProjectID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}

	collectionName := project.GetCollectionName()
	if collectionName == "" {
		return fmt.Errorf("project has no indexed collection")
	}

	logger.WithField("collection", collectionName).Info("Starting secrets scan job")

	progressCb(5, "Initializing scanner...")

	// Create scanner
	secretsScanner := scanner.NewScanner(e.vectorStore, e.logger)

	// Create scan result record
	startTime := time.Now()
	scanResult := &models.GitLabScanResult{
		ProjectID:     job.ProjectID,
		IntegrationID: job.IntegrationID,
		ScanType:      models.ScanTypeSecrets,
		Status:        models.ScanStatusRunning,
		StartedAt:     startTime,
	}

	progressCb(10, "Scanning code chunks...")

	// Run scan
	result, err := secretsScanner.Scan(ctx, scanner.ScanRequest{
		ProjectID:      job.ProjectID,
		CollectionName: collectionName,
		Categories:     config.Categories,
	})

	// Update scan result
	completedAt := time.Now()
	scanResult.CompletedAt = &completedAt
	scanResult.DurationMs = completedAt.Sub(startTime).Milliseconds()

	if err != nil {
		scanResult.Status = models.ScanStatusFailed
		scanResult.Error = err.Error()
		e.store.SaveScanResult(ctx, scanResult)
		return fmt.Errorf("secrets scan failed: %w", err)
	}

	// Save successful result
	scanResult.Status = models.ScanStatusCompleted
	scanResult.FindingsCount = len(result.Findings)
	scanResult.FilesAffected = result.Summary.FilesAffected

	if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
		scanResult.ResultsJSON = string(resultJSON)
	}

	if err := e.store.SaveScanResult(ctx, scanResult); err != nil {
		logger.WithError(err).Warn("Failed to save scan result")
	}

	// Update job with result reference
	e.store.UpdateUserJobResult(ctx, job.ID, scanResult.ID, "scan_result", "")

	progressCb(100, fmt.Sprintf("Completed: %d findings", len(result.Findings)))

	logger.WithFields(logrus.Fields{
		"findings":  len(result.Findings),
		"duration":  result.Duration,
		"result_id": scanResult.ID,
	}).Info("Secrets scan job completed")

	return nil
}

// SASTScanExecutor executes SAST scan jobs
type SASTScanExecutor struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	logger      *logrus.Logger
}

// NewSASTScanExecutor creates a new SAST scan executor
func NewSASTScanExecutor(store storage.Store, vectorStore *vector.QdrantStore, logger *logrus.Logger) *SASTScanExecutor {
	return &SASTScanExecutor{
		store:       store,
		vectorStore: vectorStore,
		logger:      logger,
	}
}

// Execute runs the SAST scan job
func (e *SASTScanExecutor) Execute(ctx context.Context, job *models.UserJob, config *models.UserJobConfig, progressCb func(progress int, msg string)) error {
	logger := e.logger.WithFields(logrus.Fields{
		"job_id":     job.ID,
		"project_id": job.ProjectID,
	})

	// Get project
	project, err := e.store.GetProject(ctx, job.ProjectID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}

	collectionName := project.GetCollectionName()
	if collectionName == "" {
		return fmt.Errorf("project has no indexed collection")
	}

	logger.WithField("collection", collectionName).Info("Starting SAST scan job")

	progressCb(5, "Initializing SAST scanner...")

	// Create scanner
	sastScanner := scanner.NewSASTScanner(e.vectorStore, e.logger)

	// Create scan result record
	startTime := time.Now()

	progressCb(10, "Scanning for vulnerabilities...")

	// Run scan
	result, err := sastScanner.Scan(ctx, scanner.SASTScanRequest{
		ProjectID:      job.ProjectID,
		CollectionName: collectionName,
	})

	// Update scan result
	completedAt := time.Now()

	scanResult := &models.GitLabScanResult{
		ProjectID:     job.ProjectID,
		IntegrationID: job.IntegrationID,
		ScanType:      models.ScanTypeSecrets, // Using secrets type for SAST for now
		Status:        models.ScanStatusCompleted,
		StartedAt:     startTime,
		CompletedAt:   &completedAt,
		DurationMs:    completedAt.Sub(startTime).Milliseconds(),
	}

	if err != nil {
		scanResult.Status = models.ScanStatusFailed
		scanResult.Error = err.Error()
		e.store.SaveScanResult(ctx, scanResult)
		return fmt.Errorf("SAST scan failed: %w", err)
	}

	// Save successful result
	scanResult.FindingsCount = len(result.Findings)
	scanResult.FilesAffected = result.Summary.FilesAffected

	if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
		scanResult.ResultsJSON = string(resultJSON)
	}

	if err := e.store.SaveScanResult(ctx, scanResult); err != nil {
		logger.WithError(err).Warn("Failed to save scan result")
	}

	// Update job with result reference
	e.store.UpdateUserJobResult(ctx, job.ID, scanResult.ID, "scan_result", "")

	progressCb(100, fmt.Sprintf("Completed: %d findings", len(result.Findings)))

	logger.WithFields(logrus.Fields{
		"findings":  len(result.Findings),
		"duration":  result.Duration,
		"result_id": scanResult.ID,
	}).Info("SAST scan job completed")

	return nil
}
