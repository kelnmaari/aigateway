// Package jobs provides job executors for various scan types
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"aigateway/internal/gitlab/deadcode"
	"aigateway/internal/gitlab/dependencies"
	"aigateway/internal/gitlab/quality"
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

// ============================================================================
// Quality Scan Executor
// ============================================================================

// QualityScanExecutor executes code quality analysis jobs
type QualityScanExecutor struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	logger      *logrus.Logger
}

// NewQualityScanExecutor creates a new quality scan executor
func NewQualityScanExecutor(store storage.Store, vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *QualityScanExecutor {
	return &QualityScanExecutor{
		store:       store,
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
}

// Execute runs the quality scan job
func (e *QualityScanExecutor) Execute(ctx context.Context, job *models.UserJob, config *models.UserJobConfig, progressCb func(progress int, msg string)) error {
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

	modelID := config.ModelID
	if modelID == "" {
		modelID = project.AnalysisModelID
	}
	if modelID == "" {
		return fmt.Errorf("no analysis model configured")
	}

	logger.WithFields(logrus.Fields{
		"collection": collectionName,
		"model":      modelID,
	}).Info("Starting quality scan job")

	progressCb(5, "Initializing quality analyzer...")

	// Create analyzer
	analyzer := quality.NewAnalyzer(e.vectorStore, e.llmBaseURL, e.llmAPIKey, e.logger)

	startTime := time.Now()
	scanResult := &models.GitLabScanResult{
		ProjectID:     job.ProjectID,
		IntegrationID: job.IntegrationID,
		ScanType:      models.ScanTypeQuality,
		Status:        models.ScanStatusRunning,
		ModelID:       modelID,
		StartedAt:     startTime,
	}

	progressCb(10, "Analyzing code quality...")

	// Run analysis
	result, err := analyzer.Analyze(ctx, quality.AnalysisRequest{
		ProjectID:      job.ProjectID,
		CollectionName: collectionName,
		ModelID:        modelID,
		MaxFiles:       config.MaxChunks, // Reuse max_chunks as max_files
		Language:       config.Language,
	})

	completedAt := time.Now()
	scanResult.CompletedAt = &completedAt
	scanResult.DurationMs = completedAt.Sub(startTime).Milliseconds()

	if err != nil {
		scanResult.Status = models.ScanStatusFailed
		scanResult.Error = err.Error()
		e.store.SaveScanResult(ctx, scanResult)
		return fmt.Errorf("quality scan failed: %w", err)
	}

	// Save successful result
	scanResult.Status = models.ScanStatusCompleted
	scanResult.FindingsCount = result.Summary.IssuesCount
	scanResult.FilesAffected = len(result.FileScores)
	scanResult.TokensUsed = result.TokensUsed

	if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
		scanResult.ResultsJSON = string(resultJSON)
	}

	if err := e.store.SaveScanResult(ctx, scanResult); err != nil {
		logger.WithError(err).Warn("Failed to save scan result")
	}

	e.store.UpdateUserJobResult(ctx, job.ID, scanResult.ID, "scan_result", "")
	progressCb(100, fmt.Sprintf("Completed: score %d, %d issues", result.OverallScore, result.Summary.IssuesCount))

	logger.WithFields(logrus.Fields{
		"score":     result.OverallScore,
		"issues":    result.Summary.IssuesCount,
		"files":     len(result.FileScores),
		"result_id": scanResult.ID,
	}).Info("Quality scan job completed")

	return nil
}

// ============================================================================
// Dependency Scan Executor
// ============================================================================

// DependencyScanExecutor executes dependency check jobs
type DependencyScanExecutor struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	logger      *logrus.Logger
}

// NewDependencyScanExecutor creates a new dependency scan executor
func NewDependencyScanExecutor(store storage.Store, vectorStore *vector.QdrantStore, logger *logrus.Logger) *DependencyScanExecutor {
	return &DependencyScanExecutor{
		store:       store,
		vectorStore: vectorStore,
		logger:      logger,
	}
}

// Execute runs the dependency scan job
func (e *DependencyScanExecutor) Execute(ctx context.Context, job *models.UserJob, config *models.UserJobConfig, progressCb func(progress int, msg string)) error {
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

	logger.WithField("collection", collectionName).Info("Starting dependency scan job")

	progressCb(5, "Initializing dependency scanner...")

	// Create scanner
	depScanner := dependencies.NewScanner(e.vectorStore, e.logger)

	startTime := time.Now()
	scanResult := &models.GitLabScanResult{
		ProjectID:     job.ProjectID,
		IntegrationID: job.IntegrationID,
		ScanType:      models.ScanTypeDependencies,
		Status:        models.ScanStatusRunning,
		StartedAt:     startTime,
	}

	progressCb(10, "Scanning dependencies...")

	// Run multi-ecosystem scan
	result, err := depScanner.ScanProjectAllEcosystems(ctx, dependencies.ScanRequest{
		ProjectID:      job.ProjectID,
		CollectionName: collectionName,
	})

	completedAt := time.Now()
	scanResult.CompletedAt = &completedAt
	scanResult.DurationMs = completedAt.Sub(startTime).Milliseconds()

	if err != nil {
		scanResult.Status = models.ScanStatusFailed
		scanResult.Error = err.Error()
		e.store.SaveScanResult(ctx, scanResult)
		return fmt.Errorf("dependency scan failed: %w", err)
	}

	// Use aggregated summary
	vulnerableCount := result.TotalSummary.VulnerableCount
	totalDeps := result.TotalSummary.TotalDependencies

	// Save successful result
	scanResult.Status = models.ScanStatusCompleted
	scanResult.FindingsCount = vulnerableCount

	if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
		scanResult.ResultsJSON = string(resultJSON)
	}

	if err := e.store.SaveScanResult(ctx, scanResult); err != nil {
		logger.WithError(err).Warn("Failed to save scan result")
	}

	e.store.UpdateUserJobResult(ctx, job.ID, scanResult.ID, "scan_result", "")
	progressCb(100, fmt.Sprintf("Completed: %d vulnerable of %d total", vulnerableCount, totalDeps))

	logger.WithFields(logrus.Fields{
		"total":      totalDeps,
		"vulnerable": vulnerableCount,
		"ecosystems": len(result.Ecosystems),
		"result_id":  scanResult.ID,
	}).Info("Dependency scan job completed")

	return nil
}

// ============================================================================
// Dead Code Scan Executor
// ============================================================================

// DeadCodeScanExecutor executes dead code detection jobs
type DeadCodeScanExecutor struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	logger      *logrus.Logger
}

// NewDeadCodeScanExecutor creates a new dead code scan executor
func NewDeadCodeScanExecutor(store storage.Store, vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *DeadCodeScanExecutor {
	return &DeadCodeScanExecutor{
		store:       store,
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
}

// Execute runs the dead code scan job
func (e *DeadCodeScanExecutor) Execute(ctx context.Context, job *models.UserJob, config *models.UserJobConfig, progressCb func(progress int, msg string)) error {
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

	modelID := config.ModelID
	if modelID == "" {
		modelID = project.AnalysisModelID
	}
	if modelID == "" {
		return fmt.Errorf("no analysis model configured")
	}

	logger.WithFields(logrus.Fields{
		"collection": collectionName,
		"model":      modelID,
	}).Info("Starting dead code scan job")

	progressCb(5, "Initializing dead code scanner...")

	// Create scanner
	dcScanner := deadcode.NewScanner(e.vectorStore, e.llmBaseURL, e.llmAPIKey, e.logger)

	startTime := time.Now()
	scanResult := &models.GitLabScanResult{
		ProjectID:     job.ProjectID,
		IntegrationID: job.IntegrationID,
		ScanType:      models.ScanTypeDeadCode,
		Status:        models.ScanStatusRunning,
		ModelID:       modelID,
		StartedAt:     startTime,
	}

	progressCb(10, "Detecting dead code...")

	maxChunks := config.MaxChunks
	if maxChunks <= 0 {
		maxChunks = 200
	}

	// Run scan
	result, err := dcScanner.Scan(ctx, deadcode.ScanRequest{
		ProjectID:      job.ProjectID,
		CollectionName: collectionName,
		ModelID:        modelID,
		MaxChunks:      maxChunks,
		Language:       config.Language,
	})

	completedAt := time.Now()
	scanResult.CompletedAt = &completedAt
	scanResult.DurationMs = completedAt.Sub(startTime).Milliseconds()

	if err != nil {
		scanResult.Status = models.ScanStatusFailed
		scanResult.Error = err.Error()
		e.store.SaveScanResult(ctx, scanResult)
		return fmt.Errorf("dead code scan failed: %w", err)
	}

	// Save successful result
	scanResult.Status = models.ScanStatusCompleted
	scanResult.FindingsCount = result.Summary.TotalDeadSymbols
	scanResult.FilesAffected = result.FilesScanned
	scanResult.TokensUsed = result.TokensUsed

	if resultJSON, jsonErr := json.Marshal(result); jsonErr == nil {
		scanResult.ResultsJSON = string(resultJSON)
	}

	if err := e.store.SaveScanResult(ctx, scanResult); err != nil {
		logger.WithError(err).Warn("Failed to save scan result")
	}

	e.store.UpdateUserJobResult(ctx, job.ID, scanResult.ID, "scan_result", "")
	progressCb(100, fmt.Sprintf("Completed: %d dead symbols in %d files", result.Summary.TotalDeadSymbols, result.FilesScanned))

	logger.WithFields(logrus.Fields{
		"dead_symbols": result.Summary.TotalDeadSymbols,
		"files":        result.FilesScanned,
		"tokens":       result.TokensUsed,
		"result_id":    scanResult.ID,
	}).Info("Dead code scan job completed")

	return nil
}
