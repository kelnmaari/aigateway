// Package analyzer provides code analysis service for GitLab MR reviews
package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Analyzer performs AI-powered code analysis
type Analyzer struct {
	llmBaseURL    string
	llmAPIKey     string
	httpClient    *http.Client
	logger        *logrus.Logger
	
	// Optional: embeddings and vector store for context retrieval
	embeddingsURL string
	vectorStore   VectorStore
}

// VectorStore interface for storing and searching code embeddings
type VectorStore interface {
	// CreateCollection creates a new collection for MR analysis
	CreateCollection(ctx context.Context, name string, dimensions int) error
	
	// DeleteCollection deletes collection after analysis
	DeleteCollection(ctx context.Context, name string) error
	
	// Insert inserts vectors
	InsertBatch(ctx context.Context, collection string, docs []VectorDocument) error
	
	// Search performs similarity search
	Search(ctx context.Context, collection string, query []float64, topK int, minScore float64) ([]VectorSearchResult, error)
}

// VectorDocument document for vector storage
type VectorDocument struct {
	ID       string
	Vector   []float64
	Text     string
	Metadata map[string]interface{}
}

// VectorSearchResult search result
type VectorSearchResult struct {
	ID       string
	Score    float64
	Text     string
	Metadata map[string]interface{}
}

// AnalyzerConfig configuration for analyzer
type AnalyzerConfig struct {
	LLMBaseURL    string
	LLMAPIKey     string
	EmbeddingsURL string
	Timeout       time.Duration
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer(cfg AnalyzerConfig, logger *logrus.Logger) *Analyzer {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	
	return &Analyzer{
		llmBaseURL:    cfg.LLMBaseURL,
		llmAPIKey:     cfg.LLMAPIKey,
		embeddingsURL: cfg.EmbeddingsURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// SetVectorStore sets optional vector store for context retrieval
func (a *Analyzer) SetVectorStore(vs VectorStore) {
	a.vectorStore = vs
}

// Analyze performs full MR analysis
func (a *Analyzer) Analyze(ctx context.Context, req *AnalysisRequest) (*AnalysisResult, error) {
	startTime := time.Now()
	
	a.logger.WithFields(logrus.Fields{
		"project_id":  req.ProjectID,
		"mr_iid":      req.MRIID,
		"files_count": len(req.Changes),
	}).Info("Starting MR analysis")
	
	// Filter files based on config
	filteredChanges := a.filterFiles(req.Changes, req.Config)
	if len(filteredChanges) == 0 {
		return &AnalysisResult{
			Summary:       "No files to analyze after applying filters",
			OverallScore:  100,
			FilesAnalyzed: 0,
			ProcessingTime: time.Since(startTime),
		}, nil
	}
	
	// Update request with filtered changes
	req.Changes = filteredChanges
	
	// Calculate total lines changed
	totalLinesChanged := 0
	for _, change := range req.Changes {
		totalLinesChanged += change.LinesAdded + change.LinesRemoved
	}
	
	// Decide analysis strategy based on size
	var result *AnalysisResult
	var err error
	
	if len(req.Changes) <= 5 && totalLinesChanged <= 500 {
		// Small MR: analyze all at once
		result, err = a.analyzeSmallMR(ctx, req)
	} else {
		// Large MR: analyze files individually and aggregate
		result, err = a.analyzeLargeMR(ctx, req)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Set metrics
	result.FilesAnalyzed = len(req.Changes)
	result.LinesChanged = totalLinesChanged
	result.ProcessingTime = time.Since(startTime)
	result.IssuesFound = countIssues(result)
	
	a.logger.WithFields(logrus.Fields{
		"project_id":      req.ProjectID,
		"mr_iid":          req.MRIID,
		"files_analyzed":  result.FilesAnalyzed,
		"lines_changed":   result.LinesChanged,
		"issues_found":    result.IssuesFound,
		"overall_score":   result.OverallScore,
		"processing_time": result.ProcessingTime,
		"tokens_used":     result.TokensUsed,
	}).Info("MR analysis completed")
	
	return result, nil
}

// analyzeSmallMR analyzes small MR in a single LLM call
func (a *Analyzer) analyzeSmallMR(ctx context.Context, req *AnalysisRequest) (*AnalysisResult, error) {
	a.logger.Debug("Using single-call analysis for small MR")
	
	// Build prompt
	prompt := BuildReviewPrompt(req)
	if req.Config.CustomPrompt != "" {
		prompt = CustomPromptWrapper(req.Config.CustomPrompt, prompt)
	}
	
	// Call LLM
	response, tokensUsed, err := a.callLLM(ctx, req.Config.AnalysisModel, SystemPrompt, prompt, req.Config)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}
	
	// Parse response
	result, err := a.parseAnalysisResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}
	
	result.TokensUsed = tokensUsed
	result.Model = req.Config.AnalysisModel
	
	return result, nil
}

// analyzeLargeMR analyzes large MR by processing files individually
func (a *Analyzer) analyzeLargeMR(ctx context.Context, req *AnalysisRequest) (*AnalysisResult, error) {
	a.logger.Debug("Using file-by-file analysis for large MR")
	
	var fileReviews []FileReview
	totalTokens := 0
	
	// Analyze each file
	for _, change := range req.Changes {
		if change.DeletedFile {
			continue // Skip deleted files
		}
		
		a.logger.WithField("file", change.FilePath).Debug("Analyzing file")
		
		// Build file-specific prompt
		prompt := BuildFilePrompt(change, "")
		
		// Call LLM
		response, tokens, err := a.callLLM(ctx, req.Config.AnalysisModel, SystemPrompt, prompt, req.Config)
		if err != nil {
			a.logger.WithError(err).WithField("file", change.FilePath).Warn("Failed to analyze file, skipping")
			continue
		}
		
		totalTokens += tokens
		
		// Parse file review
		review, err := a.parseFileReview(response)
		if err != nil {
			a.logger.WithError(err).WithField("file", change.FilePath).Warn("Failed to parse file review")
			continue
		}
		
		fileReviews = append(fileReviews, *review)
	}
	
	// Aggregate results
	result := a.aggregateResults(fileReviews)
	result.TokensUsed = totalTokens
	result.Model = req.Config.AnalysisModel
	
	return result, nil
}

// callLLM calls the LLM API
func (a *Analyzer) callLLM(ctx context.Context, model, systemPrompt, userPrompt string, cfg AnalysisConfig) (string, int, error) {
	request := LLMRequest{
		Model: model,
		Messages: []LLMMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: cfg.Temperature,
		MaxTokens:   cfg.MaxTokens,
		Stream:      false,
	}
	
	body, err := json.Marshal(request)
	if err != nil {
		return "", 0, fmt.Errorf("marshal request: %w", err)
	}
	
	url := fmt.Sprintf("%s/v1/chat/completions", strings.TrimSuffix(a.llmBaseURL, "/"))
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	if a.llmAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.llmAPIKey)
	}
	
	a.logger.WithFields(logrus.Fields{
		"model":       model,
		"url":         url,
		"prompt_len":  len(userPrompt),
	}).Debug("Calling LLM API")
	
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read response: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("LLM API error: status=%d body=%s", resp.StatusCode, string(respBody))
	}
	
	var llmResp LLMResponse
	if err := json.Unmarshal(respBody, &llmResp); err != nil {
		return "", 0, fmt.Errorf("unmarshal response: %w", err)
	}
	
	if len(llmResp.Choices) == 0 {
		return "", 0, fmt.Errorf("no choices in LLM response")
	}
	
	content := llmResp.Choices[0].Message.Content
	tokensUsed := llmResp.Usage.TotalTokens
	
	a.logger.WithFields(logrus.Fields{
		"tokens_used":  tokensUsed,
		"response_len": len(content),
	}).Debug("LLM response received")
	
	return content, tokensUsed, nil
}

// parseAnalysisResponse parses the full analysis JSON response
func (a *Analyzer) parseAnalysisResponse(response string) (*AnalysisResult, error) {
	// Clean response - extract JSON if wrapped in markdown
	cleaned := extractJSON(response)
	
	var result AnalysisResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		a.logger.WithError(err).WithField("response", truncate(response, 500)).Warn("Failed to parse JSON response")
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	
	return &result, nil
}

// parseFileReview parses file review JSON response
func (a *Analyzer) parseFileReview(response string) (*FileReview, error) {
	cleaned := extractJSON(response)
	
	var review FileReview
	if err := json.Unmarshal([]byte(cleaned), &review); err != nil {
		return nil, fmt.Errorf("parse file review JSON: %w", err)
	}
	
	return &review, nil
}

// aggregateResults aggregates individual file reviews into overall result
func (a *Analyzer) aggregateResults(fileReviews []FileReview) *AnalysisResult {
	result := &AnalysisResult{
		FileReviews: fileReviews,
		Categories:  make([]CategoryResult, 4),
	}
	
	// Initialize categories
	categories := []string{CategorySecurity, CategoryBugs, CategoryStyle, CategoryPerformance}
	categoryIssues := make(map[string]int)
	categoryScores := make(map[string][]int)
	
	for _, cat := range categories {
		categoryIssues[cat] = 0
		categoryScores[cat] = []int{}
	}
	
	// Aggregate from file reviews
	totalScore := 0
	for _, review := range fileReviews {
		totalScore += review.Score
		
		for _, issue := range review.LineIssues {
			categoryIssues[issue.Category]++
		}
	}
	
	// Calculate overall score
	if len(fileReviews) > 0 {
		result.OverallScore = totalScore / len(fileReviews)
	} else {
		result.OverallScore = 100
	}
	
	// Build category results
	for i, cat := range categories {
		issues := categoryIssues[cat]
		score := 100 - (issues * 10) // Deduct 10 points per issue
		if score < 0 {
			score = 0
		}
		
		result.Categories[i] = CategoryResult{
			Name:   cat,
			Score:  score,
			Issues: issues,
		}
	}
	
	// Generate summary
	totalIssues := 0
	for _, issues := range categoryIssues {
		totalIssues += issues
	}
	
	if totalIssues == 0 {
		result.Summary = "No significant issues found. The code looks good overall."
	} else if totalIssues <= 3 {
		result.Summary = fmt.Sprintf("Found %d minor issues. Good code quality with room for small improvements.", totalIssues)
	} else if totalIssues <= 10 {
		result.Summary = fmt.Sprintf("Found %d issues that should be addressed before merging.", totalIssues)
	} else {
		result.Summary = fmt.Sprintf("Found %d issues. Significant improvements needed before this code can be merged.", totalIssues)
	}
	
	return result
}

// filterFiles filters files based on config patterns
func (a *Analyzer) filterFiles(changes []FileChange, cfg AnalysisConfig) []FileChange {
	if len(cfg.FileFilters) == 0 && cfg.MaxFiles == 0 {
		return changes
	}
	
	var filtered []FileChange
	
	for _, change := range changes {
		// Check include/exclude patterns
		if len(cfg.FileFilters) > 0 && !matchesFilters(change.FilePath, cfg.FileFilters) {
			continue
		}
		
		// Check max lines per file
		if cfg.MaxLinesPerFile > 0 && (change.LinesAdded+change.LinesRemoved) > cfg.MaxLinesPerFile {
			a.logger.WithField("file", change.FilePath).Debug("File exceeds max lines, skipping")
			continue
		}
		
		filtered = append(filtered, change)
		
		// Check max files
		if cfg.MaxFiles > 0 && len(filtered) >= cfg.MaxFiles {
			break
		}
	}
	
	return filtered
}

// Helper functions

func extractJSON(response string) string {
	// Try to extract JSON from markdown code blocks
	response = strings.TrimSpace(response)
	
	// Check for ```json ... ``` blocks
	if idx := strings.Index(response, "```json"); idx != -1 {
		start := idx + 7
		end := strings.Index(response[start:], "```")
		if end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}
	
	// Check for ``` ... ``` blocks
	if idx := strings.Index(response, "```"); idx != -1 {
		start := idx + 3
		// Skip language identifier if present
		if newline := strings.Index(response[start:], "\n"); newline != -1 {
			start += newline + 1
		}
		end := strings.Index(response[start:], "```")
		if end != -1 {
			return strings.TrimSpace(response[start : start+end])
		}
	}
	
	// Find JSON object boundaries
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start != -1 && end != -1 && end > start {
		return response[start : end+1]
	}
	
	return response
}

func matchesFilters(filepath string, filters []string) bool {
	for _, filter := range filters {
		// Exclude patterns start with !
		if strings.HasPrefix(filter, "!") {
			pattern := strings.TrimPrefix(filter, "!")
			if matchGlob(filepath, pattern) {
				return false
			}
		}
	}
	
	// Check include patterns
	hasInclude := false
	for _, filter := range filters {
		if !strings.HasPrefix(filter, "!") {
			hasInclude = true
			if matchGlob(filepath, filter) {
				return true
			}
		}
	}
	
	// If no include patterns, include all (that weren't excluded)
	return !hasInclude
}

func matchGlob(filepath, pattern string) bool {
	// Simple glob matching for *.ext and path/* patterns
	if strings.HasPrefix(pattern, "*.") {
		ext := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(filepath, ext)
	}
	
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(filepath, prefix+"/")
	}
	
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return strings.HasPrefix(filepath, prefix+"/")
	}
	
	return strings.Contains(filepath, pattern)
}

func countIssues(result *AnalysisResult) int {
	count := 0
	for _, review := range result.FileReviews {
		count += len(review.LineIssues)
	}
	return count
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

