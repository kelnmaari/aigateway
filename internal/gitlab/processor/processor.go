// Package processor implements GitLab MR analysis job processing
package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aigateway/internal/gitlab/analyzer"
	"aigateway/internal/gitlab/chunker"
	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/comment"
	"aigateway/internal/gitlab/rag"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"

	"github.com/sirupsen/logrus"
)

// Processor processes GitLab MR analysis jobs
type Processor struct {
	store           storage.Store
	ragService      *rag.RAGService
	chunkerService  *chunker.CodeChunker
	commentBuilder  *comment.Builder
	logger          *logrus.Logger
	
	// LLM configuration
	llmBaseURL      string
	llmAPIKey       string
	httpClient      *http.Client
	
	// Cache of GitLab clients per integration
	clients         map[string]*client.Client
}

// ProcessorConfig configuration for processor
type ProcessorConfig struct {
	LLMBaseURL string
	LLMAPIKey  string // API key for LLM requests (use internal key)
	Timeout    time.Duration
}

// NewProcessor creates a new job processor
func NewProcessor(
	store storage.Store,
	ragService *rag.RAGService,
	cfg ProcessorConfig,
	logger *logrus.Logger,
) *Processor {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	
	return &Processor{
		store:          store,
		ragService:     ragService,
		chunkerService: chunker.NewCodeChunker(chunker.DefaultConfig()),
		commentBuilder: comment.NewBuilder(),
		logger:         logger,
		llmAPIKey:      cfg.LLMAPIKey,
		llmBaseURL:     cfg.LLMBaseURL,
		httpClient:     &http.Client{Timeout: timeout},
		clients:        make(map[string]*client.Client),
	}
}

// ProcessJob implements worker.JobProcessor interface
func (p *Processor) ProcessJob(ctx context.Context, job *models.GitLabAnalysisJob) error {
	startTime := time.Now()
	
	p.logger.WithFields(logrus.Fields{
		"job_id":    job.ID,
		"review_id": job.ReviewID,
		"mr_iid":    job.MRIID,
	}).Info("Starting MR analysis")
	
	// Get review details
	review, err := p.store.GetReview(ctx, job.ReviewID)
	if err != nil {
		return fmt.Errorf("get review: %w", err)
	}
	if review == nil {
		return fmt.Errorf("review not found: %s", job.ReviewID)
	}
	
	// Get project configuration
	project, err := p.store.GetProject(ctx, job.ProjectID)
	if err != nil {
		return fmt.Errorf("get project: %w", err)
	}
	if project == nil {
		return fmt.Errorf("project not found: %s", job.ProjectID)
	}
	
	// Get integration for GitLab API access
	integration, err := p.store.GetIntegration(ctx, job.IntegrationID)
	if err != nil {
		return fmt.Errorf("get integration: %w", err)
	}
	if integration == nil {
		return fmt.Errorf("integration not found: %s", job.IntegrationID)
	}
	
	// Get or create GitLab client
	gitlabClient, err := p.getGitLabClient(integration)
	if err != nil {
		return fmt.Errorf("create gitlab client: %w", err)
	}
	
	// Step 1: Get MR diff from GitLab
	p.logger.Debug("Fetching MR diff from GitLab")
	diffs, err := gitlabClient.GetMergeRequestDiffs(ctx, project.GitLabProjectID, review.MRIID)
	if err != nil {
		return fmt.Errorf("get MR diff: %w", err)
	}
	
	if len(diffs) == 0 {
		p.logger.Info("No changes in MR, skipping analysis")
		_ = p.store.UpdateReviewStatus(ctx, review.ID, models.GitLabReviewStatusSkipped, "No changes to analyze")
		return nil
	}
	
	// Step 2: Chunk the diff files
	p.logger.Debug("Chunking diff files")
	var allChunks []chunker.Chunk
	var totalLines int
	
	for _, diff := range diffs {
		if p.shouldSkipFile(diff.NewPath, project.Settings) {
			continue
		}
		
		changeType := "modified"
		if diff.NewFile {
			changeType = "added"
		} else if diff.DeletedFile {
			changeType = "deleted"
		}
		
		chunks, _ := p.chunkerService.ChunkDiff(diff.NewPath, diff.Diff, changeType)
		allChunks = append(allChunks, chunks...)
		totalLines += countLines(diff.Diff)
	}
	
	p.logger.WithFields(logrus.Fields{
		"files":  len(diffs),
		"chunks": len(allChunks),
		"lines":  totalLines,
	}).Debug("Diff chunked")
	
	// Step 3: Generate embeddings and search for context (if RAG enabled)
	var contextChunks []rag.CodeChunk
	if p.ragService != nil && p.ragService.IsEnabled() {
		// Use project-specific embedding model if configured
		embeddingModel := job.EmbeddingModelID
		if embeddingModel != "" {
			p.logger.WithField("embedding_model", embeddingModel).Debug("Searching for relevant context using project-specific embedding model")
		} else {
			p.logger.Debug("Searching for relevant context using default embedding model")
		}
		
		// Build list of changed files
		var changedFiles []rag.ChangedFile
		for _, diff := range diffs {
			changedFiles = append(changedFiles, rag.ChangedFile{
				Path: diff.NewPath,
				Diff: diff.Diff,
			})
		}
		
		// Pass embedding model alias from project configuration
		contextChunks, err = p.ragService.GetContextForReview(ctx, project.ID, changedFiles, 5, embeddingModel)
		if err != nil {
			p.logger.WithError(err).Warn("Failed to get context, continuing without RAG")
		}
	}
	
	// Step 4: Analyze with LLM
	p.logger.Debug("Analyzing with LLM")
	analysisResult, tokensUsed, err := p.analyzeWithLLM(ctx, project, diffs, allChunks, contextChunks)
	if err != nil {
		return fmt.Errorf("LLM analysis: %w", err)
	}
	
	// Step 5: Build review result
	reviewResult := p.buildReviewResult(analysisResult, diffs)
	
	// Step 6: Update review with results
	processingTime := time.Since(startTime).Milliseconds()
	
	if err := p.store.UpdateReviewMetrics(ctx, review.ID, 
		len(diffs), totalLines, reviewResult.IssuesFound,
		processingTime, tokensUsed, project.AnalysisModelID); err != nil {
		p.logger.WithError(err).Warn("Failed to update review metrics")
	}
	
	// Step 7: Post comment to GitLab MR
	p.logger.Debug("Posting review comment to GitLab")
	stats := comment.ReviewStats{
		FilesAnalyzed:    len(diffs),
		LinesChanged:     totalLines,
		IssuesFound:      reviewResult.IssuesFound,
		ProcessingTimeMs: processingTime,
		TokensUsed:       tokensUsed,
		Model:            project.AnalysisModelID,
	}
	commentText := p.commentBuilder.BuildReviewComment(&reviewResult.Result, stats)
	
	note, err := gitlabClient.CreateMRNote(ctx, project.GitLabProjectID, review.MRIID, commentText)
	if err != nil {
		p.logger.WithError(err).Error("Failed to post comment to GitLab")
		// Don't fail the job, just log the error
	}
	
	// Step 8: Save final result
	var noteID int64
	if note != nil {
		noteID = note.ID
	}
	if err := p.store.UpdateReviewResult(ctx, review.ID, &reviewResult.Result, noteID, ""); err != nil {
		return fmt.Errorf("save review result: %w", err)
	}
	
	p.logger.WithFields(logrus.Fields{
		"job_id":          job.ID,
		"review_id":       review.ID,
		"issues_found":    reviewResult.IssuesFound,
		"processing_ms":   processingTime,
		"tokens_used":     tokensUsed,
	}).Info("MR analysis completed")
	
	return nil
}

// getGitLabClient gets or creates a GitLab client for the integration
func (p *Processor) getGitLabClient(integration *models.GitLabIntegration) (*client.Client, error) {
	if c, ok := p.clients[integration.ID]; ok {
		return c, nil
	}
	
	c := client.NewClient(client.ClientConfig{
		BaseURL:     integration.BaseURL,
		AccessToken: integration.AccessToken,
		Timeout:     30 * time.Second,
	})
	
	p.clients[integration.ID] = c
	return c, nil
}

// shouldSkipFile checks if file should be skipped based on project settings
func (p *Processor) shouldSkipFile(path string, settings models.GitLabProjectSettings) bool {
	// Check exclude patterns
	for _, pattern := range settings.ExcludePatterns {
		if matchPattern(pattern, path) {
			return true
		}
	}
	
	// Check include patterns (if specified, only include matching files)
	if len(settings.IncludePatterns) > 0 {
		for _, pattern := range settings.IncludePatterns {
			if matchPattern(pattern, path) {
				return false
			}
		}
		return true // Not in include list
	}
	
	return false
}

// analyzeWithLLM sends the diff to LLM for analysis
func (p *Processor) analyzeWithLLM(
	ctx context.Context,
	project *models.GitLabProject,
	diffs []client.Diff,
	chunks []chunker.Chunk,
	ragContext []rag.CodeChunk,
) (*analyzer.AnalysisResultParsed, int, error) {
	// Check if per-file review mode is enabled
	if project.Settings.PerFileReview {
		return p.analyzePerFile(ctx, project, diffs)
	}
	
	// Build the prompt
	prompt := p.buildAnalysisPrompt(project, diffs, chunks, ragContext)
	
	// Call LLM API (OpenAI-compatible)
	llmURL := p.llmBaseURL
	if llmURL == "" {
		llmURL = "http://localhost:8080" // Default to self
	}
	
	// Use project's analysis model
	modelID := project.AnalysisModelID
	if modelID == "" {
		modelID = "default"
	}
	
	// Determine max_tokens from project settings or use default
	maxTokens := project.Settings.MaxReviewTokens
	if maxTokens <= 0 {
		maxTokens = 8192 // Default
	}
	
	p.logger.WithFields(logrus.Fields{
		"model":      modelID,
		"max_tokens": maxTokens,
		"configured": project.Settings.MaxReviewTokens,
	}).Debug("LLM analysis config")
	
	requestBody := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]string{
			{"role": "system", "content": analyzer.GetSystemPrompt(project.ReviewPrompt)},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"max_tokens":  maxTokens,
	}
	
	bodyBytes, _ := json.Marshal(requestBody)
	
	req, err := http.NewRequestWithContext(ctx, "POST", llmURL+"/v1/chat/completions", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.llmAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.llmAPIKey)
		p.logger.WithField("key_prefix", p.llmAPIKey[:15]+"...").Debug("LLM request with API key")
	} else {
		p.logger.Warn("LLM request WITHOUT API key - will get 401!")
	}
	
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("LLM returned status %d", resp.StatusCode)
	}
	
	var llmResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		return nil, 0, fmt.Errorf("decode LLM response: %w", err)
	}
	
	if len(llmResponse.Choices) == 0 {
		return nil, 0, fmt.Errorf("no response from LLM")
	}
	
	rawContent := llmResponse.Choices[0].Message.Content
	
	// Log raw response for debugging (truncated)
	logContent := rawContent
	if len(logContent) > 500 {
		logContent = logContent[:500] + "...[truncated]"
	}
	p.logger.WithFields(logrus.Fields{
		"response_len":  len(rawContent),
		"tokens_used":   llmResponse.Usage.TotalTokens,
		"response_start": logContent,
	}).Debug("LLM raw response received")
	
	// Parse LLM response
	result, err := analyzer.ParseAnalysisResponse(rawContent)
	if err != nil {
		p.logger.WithError(err).WithField("raw_response", rawContent).Warn("Failed to parse structured response")
		// Return error instead of silent fallback
		return nil, 0, fmt.Errorf("failed to parse LLM response: %w (content length: %d)", err, len(rawContent))
	}
	
	// Validate parsed result
	if result.Score == 0 && len(result.Issues) == 0 && len(result.Suggestions) == 0 {
		p.logger.WithField("raw_response", rawContent).Warn("LLM returned empty analysis - check model response")
	}
	
	return result, llmResponse.Usage.TotalTokens, nil
}

// buildAnalysisPrompt builds the analysis prompt using the structured template
func (p *Processor) buildAnalysisPrompt(
	project *models.GitLabProject,
	diffs []client.Diff,
	chunks []chunker.Chunk,
	ragContext []rag.CodeChunk,
) string {
	var sb strings.Builder
	
	sb.WriteString("## Code Changes to Review\n\n")
	
	// Add diff content
	for _, diff := range diffs {
		changeStatus := "Modified"
		if diff.NewFile {
			changeStatus = "New file"
		} else if diff.DeletedFile {
			changeStatus = "Deleted"
		}
		
		sb.WriteString(fmt.Sprintf("### File: `%s` (%s)\n", diff.NewPath, changeStatus))
		sb.WriteString("```diff\n")
		sb.WriteString(diff.Diff)
		sb.WriteString("\n```\n\n")
	}
	
	// Add RAG context if available
	if len(ragContext) > 0 {
		sb.WriteString("\n## Related Code Context (from existing codebase)\n")
		sb.WriteString("Use this context to understand patterns and conventions in the codebase:\n\n")
		for _, chunk := range ragContext {
			sb.WriteString(fmt.Sprintf("### From `%s`:\n", chunk.FilePath))
			sb.WriteString(fmt.Sprintf("```%s\n", chunk.Language))
			sb.WriteString(chunk.Content)
			sb.WriteString("\n```\n\n")
		}
	}
	
	// Add response format instructions
	sb.WriteString(`
## Required Response Format

Respond with a valid JSON object in this EXACT structure:

{
  "summary": "Brief overall assessment in 1-2 sentences",
  "overall_score": 85,
  "categories": [
    {"name": "security", "score": 90, "issues": 0, "details": "Brief finding"},
    {"name": "bugs", "score": 80, "issues": 1, "details": "Brief finding"},
    {"name": "style", "score": 85, "issues": 2, "details": "Brief finding"},
    {"name": "performance", "score": 90, "issues": 0, "details": "Brief finding"}
  ],
  "file_reviews": [
    {
      "file_path": "path/to/file.go",
      "score": 80,
      "summary": "Brief file summary",
      "line_issues": [
        {
          "line": 42,
          "severity": "warning",
          "category": "security",
          "message": "What the issue is",
          "suggestion": "How to fix it"
        }
      ]
    }
  ],
  "suggestions": [
    {
      "category": "best_practice",
      "title": "Suggestion title",
      "description": "Detailed suggestion",
      "priority": "medium"
    }
  ]
}

IMPORTANT:
- Score is 0-100 (higher is better)
- severity: "critical", "warning", "info", or "suggestion"
- category: "security", "bugs", "style", "performance", or "best_practice"
- priority: "high", "medium", or "low"
- Only include file_reviews for files with actual issues
- If no issues found, return empty arrays with high scores
- Respond with ONLY valid JSON, no markdown wrapping
`)
	
	return sb.String()
}

// analyzePerFile uses per-file review mode with tool calling
func (p *Processor) analyzePerFile(
	ctx context.Context,
	project *models.GitLabProject,
	diffs []client.Diff,
) (*analyzer.AnalysisResultParsed, int, error) {
	p.logger.WithFields(logrus.Fields{
		"project":     project.Name,
		"files_count": len(diffs),
	}).Info("Using per-file review mode with tool calling")
	
	// Get LLM configuration
	llmURL := p.llmBaseURL
	if llmURL == "" {
		llmURL = "http://localhost:8080"
	}
	
	modelID := project.AnalysisModelID
	if modelID == "" {
		modelID = "default"
	}
	
	maxTokens := project.Settings.MaxReviewTokens
	if maxTokens <= 0 {
		maxTokens = 4096 // Lower default for per-file mode
	}
	
	// Create per-file reviewer
	reviewLang := project.Settings.ReviewLanguage
	if reviewLang == "" {
		reviewLang = "en"
	}
	
	reviewer := NewPerFileReviewer(
		p.ragService,
		project.ID,
		llmURL,
		p.llmAPIKey,
		maxTokens,
		reviewLang,
		p.logger,
	)
	
	return reviewer.ReviewFiles(ctx, project, diffs, modelID)
}

// buildReviewResult converts analysis result to review result
func (p *Processor) buildReviewResult(analysis *analyzer.AnalysisResultParsed, diffs []client.Diff) *ReviewResult {
	// Calculate score: if no issues found and LLM returned 0 or didn't provide score,
	// give perfect score. If issues exist but score is 0, calculate based on issues.
	score := analysis.Score
	if score == 0 {
		if len(analysis.Issues) == 0 {
			score = 100 // No issues = perfect score
		} else {
			// Estimate score based on number of issues per file
			issuesPerFile := float64(len(analysis.Issues)) / float64(max(len(diffs), 1))
			score = max(30, 100-int(issuesPerFile*20)) // Each issue reduces score, min 30
		}
	}
	
	result := &ReviewResult{
		IssuesFound: len(analysis.Issues),
		Result: models.GitLabReviewResult{
			Summary:      analysis.Summary,
			OverallScore: score,
			Categories:   make([]models.GitLabReviewCategory, 0),
			FileReviews:  make([]models.GitLabFileReview, 0),
			Suggestions:  make([]models.GitLabSuggestion, 0),
		},
	}
	
	// Group issues by category
	categoryIssues := make(map[string]int)
	for _, issue := range analysis.Issues {
		categoryIssues[issue.Category]++
	}
	
	for cat, count := range categoryIssues {
		result.Result.Categories = append(result.Result.Categories, models.GitLabReviewCategory{
			Name:       cat,
			IssueCount: count,
		})
	}
	
	// Group issues by file
	fileIssues := make(map[string][]models.GitLabCodeIssue)
	for _, issue := range analysis.Issues {
		fileIssues[issue.FilePath] = append(fileIssues[issue.FilePath], models.GitLabCodeIssue{
			Line:       issue.Line,
			Severity:   models.GitLabIssueSeverity(issue.Severity),
			Category:   issue.Category,
			Message:    issue.Message,
			Suggestion: issue.Suggestion,
		})
	}
	
	for file, issues := range fileIssues {
		result.Result.FileReviews = append(result.Result.FileReviews, models.GitLabFileReview{
			FilePath: file,
			Issues:   issues,
			Approved: len(issues) == 0,
		})
	}
	
	// Add suggestions
	for _, suggestion := range analysis.Suggestions {
		result.Result.Suggestions = append(result.Result.Suggestions, models.GitLabSuggestion{
			FilePath:    suggestion.FilePath,
			Line:        suggestion.Line,
			Type:        suggestion.Type,
			Title:       suggestion.Title,
			Description: suggestion.Description,
			Priority:    suggestion.Priority,
		})
	}
	
	return result
}

// ReviewResult internal result structure
type ReviewResult struct {
	IssuesFound int
	Result      models.GitLabReviewResult
}

// Helper functions

func countLines(diff string) int {
	return strings.Count(diff, "\n")
}

func matchPattern(pattern, path string) bool {
	// Simple glob matching
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(path, suffix)
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		return strings.HasPrefix(path, parts[0]) && strings.HasSuffix(path, parts[1])
	}
	return strings.Contains(path, pattern)
}

