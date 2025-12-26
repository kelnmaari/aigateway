// Package processor implements per-file code review with tool calling
package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"aigateway/internal/gitlab/analyzer"
	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/rag"
	"aigateway/internal/models"

	"github.com/sirupsen/logrus"
)

const (
	// DefaultMaxToolIterations default limit for tool calling rounds
	// Most reviews need 0-2 iterations; 10 is generous safety net
	DefaultMaxToolIterations = 10
	// MaxConcurrentFileReviews limits parallel file reviews
	MaxConcurrentFileReviews = 4
)

// FileReviewResult represents review result for a single file
type FileReviewResult struct {
	FilePath    string
	Score       int
	Summary     string
	Issues      []analyzer.Issue
	Suggestions []analyzer.SuggestionItem
	TokensUsed  int
	Error       error
}

// PerFileReviewer reviews MR file by file with tool calling
type PerFileReviewer struct {
	tools       *ReviewTools
	llmBaseURL  string
	llmAPIKey   string
	httpClient  *http.Client
	logger      *logrus.Logger
	maxTokens   int
}

// NewPerFileReviewer creates a per-file reviewer
func NewPerFileReviewer(
	ragService *rag.RAGService,
	projectID string,
	llmBaseURL string,
	llmAPIKey string,
	maxTokens int,
	logger *logrus.Logger,
) *PerFileReviewer {
	return &PerFileReviewer{
		tools:      NewReviewTools(ragService, projectID, logger),
		llmBaseURL: llmBaseURL,
		llmAPIKey:  llmAPIKey,
		httpClient: &http.Client{Timeout: 120 * time.Second},
		logger:     logger,
		maxTokens:  maxTokens,
	}
}

// ReviewFiles reviews all files in parallel and aggregates results
func (r *PerFileReviewer) ReviewFiles(
	ctx context.Context,
	project *models.GitLabProject,
	diffs []client.Diff,
	modelID string,
) (*analyzer.AnalysisResultParsed, int, error) {
	if len(diffs) == 0 {
		return &analyzer.AnalysisResultParsed{
			Summary: "No files to review",
			Score:   100,
		}, 0, nil
	}

	r.logger.WithFields(logrus.Fields{
		"files_count": len(diffs),
		"model":       modelID,
		"max_tokens":  r.maxTokens,
	}).Info("Starting per-file review")

	// Review files with limited concurrency
	results := make(chan FileReviewResult, len(diffs))
	sem := make(chan struct{}, MaxConcurrentFileReviews)
	var wg sync.WaitGroup

	for _, diff := range diffs {
		wg.Add(1)
		go func(d client.Diff) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			result := r.reviewSingleFile(ctx, project, d, modelID)
			results <- result
		}(diff)
	}

	// Wait for all reviews to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var fileResults []FileReviewResult
	totalTokens := 0
	for result := range results {
		fileResults = append(fileResults, result)
		totalTokens += result.TokensUsed
		
		if result.Error != nil {
			r.logger.WithError(result.Error).WithField("file", result.FilePath).Warn("File review failed")
		} else {
			r.logger.WithFields(logrus.Fields{
				"file":   result.FilePath,
				"score":  result.Score,
				"issues": len(result.Issues),
			}).Debug("File review completed")
		}
	}

	// Aggregate results
	aggregated := r.aggregateResults(fileResults)
	
	r.logger.WithFields(logrus.Fields{
		"total_files":   len(diffs),
		"total_issues":  len(aggregated.Issues),
		"overall_score": aggregated.Score,
		"total_tokens":  totalTokens,
	}).Info("Per-file review completed")

	return aggregated, totalTokens, nil
}

// reviewSingleFile reviews a single file with tool calling
func (r *PerFileReviewer) reviewSingleFile(
	ctx context.Context,
	project *models.GitLabProject,
	diff client.Diff,
	modelID string,
) FileReviewResult {
	result := FileReviewResult{
		FilePath: diff.NewPath,
	}

	// Build initial prompt for single file
	prompt := r.buildFilePrompt(diff)
	
	// Messages for conversation
	messages := []map[string]interface{}{
		{"role": "system", "content": r.getFileReviewSystemPrompt()},
		{"role": "user", "content": prompt},
	}

	// Tool calling loop - model will stop calling tools when it has enough context
	// Track previous queries to detect loops
	var finalResponse string
	seenQueries := make(map[string]int)
	
	for i := 0; i < DefaultMaxToolIterations; i++ {
		response, toolCalls, tokens, err := r.callLLM(ctx, modelID, messages)
		result.TokensUsed += tokens
		
		if err != nil {
			result.Error = err
			return result
		}

		// If no tool calls, we have the final response
		if len(toolCalls) == 0 {
			finalResponse = response
			break
		}

		// Check for repeated tool calls (loop detection)
		allRepeated := true
		for _, tc := range toolCalls {
			key := tc.Function.Name + ":" + tc.Function.Arguments
			seenQueries[key]++
			if seenQueries[key] <= 2 {
				allRepeated = false
			}
		}
		
		if allRepeated {
			r.logger.WithField("file", diff.NewPath).Warn("Detected tool call loop, forcing final response")
			// Add message telling model to give final answer
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": "You've already searched for this. Please provide your final review as JSON now.",
			})
			continue
		}

		r.logger.WithFields(logrus.Fields{
			"file":       diff.NewPath,
			"tool_calls": len(toolCalls),
			"iteration":  i + 1,
		}).Debug("Processing tool calls")

		// Add assistant message with tool calls
		messages = append(messages, map[string]interface{}{
			"role":       "assistant",
			"content":    response,
			"tool_calls": toolCalls,
		})

		// Execute each tool and add results
		for _, tc := range toolCalls {
			toolResult, err := r.tools.ExecuteTool(ctx, tc)
			if err != nil {
				r.logger.WithError(err).Warn("Tool execution failed")
				toolResult = &ToolResult{
					ToolCallID: tc.ID,
					Role:       "tool",
					Content:    fmt.Sprintf("Error: %v", err),
				}
			}
			messages = append(messages, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": toolResult.ToolCallID,
				"content":      toolResult.Content,
			})
		}
	}

	// Parse the final response
	if finalResponse == "" {
		result.Error = fmt.Errorf("no final response after tool iterations")
		return result
	}

	parsed, err := analyzer.ParseAnalysisResponse(finalResponse)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"file":         diff.NewPath,
			"error":        err.Error(),
			"response_len": len(finalResponse),
			"response":     truncateString(finalResponse, 500),
		}).Warn("Failed to parse file review response")
		
		result.Error = fmt.Errorf("parse response: %w", err)
		result.Summary = "[Parse error] " + truncateString(finalResponse, 200)
		result.Score = 50
		return result
	}

	result.Score = parsed.Score
	result.Summary = parsed.Summary
	result.Issues = parsed.Issues
	result.Suggestions = parsed.Suggestions

	return result
}

// callLLM calls the LLM API with tools
func (r *PerFileReviewer) callLLM(
	ctx context.Context,
	modelID string,
	messages []map[string]interface{},
) (string, []ToolCall, int, error) {
	requestBody := map[string]interface{}{
		"model":       modelID,
		"messages":    messages,
		"tools":       r.tools.GetToolDefinitions(),
		"temperature": 0.3,
		"max_tokens":  r.maxTokens,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequestWithContext(ctx, "POST", r.llmBaseURL+"/v1/chat/completions", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return "", nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if r.llmAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+r.llmAPIKey)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", nil, 0, err
	}
	defer resp.Body.Close()

	var llmResponse struct {
		Choices []struct {
			Message struct {
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		return "", nil, 0, fmt.Errorf("decode response: %w", err)
	}

	if llmResponse.Error != nil {
		return "", nil, 0, fmt.Errorf("LLM error: %s", llmResponse.Error.Message)
	}

	if len(llmResponse.Choices) == 0 {
		return "", nil, 0, fmt.Errorf("no choices in response")
	}

	choice := llmResponse.Choices[0]
	return choice.Message.Content, choice.Message.ToolCalls, llmResponse.Usage.TotalTokens, nil
}

// buildFilePrompt builds a prompt for reviewing a single file
func (r *PerFileReviewer) buildFilePrompt(diff client.Diff) string {
	var sb strings.Builder

	changeStatus := "Modified"
	if diff.NewFile {
		changeStatus = "New file"
	} else if diff.DeletedFile {
		changeStatus = "Deleted"
	}

	sb.WriteString(fmt.Sprintf("## Review this file: `%s` (%s)\n\n", diff.NewPath, changeStatus))
	sb.WriteString("```diff\n")
	sb.WriteString(diff.Diff)
	sb.WriteString("\n```\n\n")
	
	sb.WriteString(`If you need to understand how functions or types are used elsewhere in the codebase, 
use the available tools to search for related code before making your review.

After gathering necessary context, provide your review as JSON:`)
	
	sb.WriteString(r.getResponseFormat())
	
	return sb.String()
}

func (r *PerFileReviewer) getFileReviewSystemPrompt() string {
	return `You are an expert code reviewer. Your task is to review code changes and output ONLY valid JSON.

## Available Tools
You can use these tools to gather context (use sparingly, max 2-3 calls):
- search_codebase: Find related code semantically
- get_function_definition: Look up function implementation
- get_type_definition: Look up struct/interface definition

## When to Use Tools
- Only if you genuinely need context about an unfamiliar function/type
- Do NOT call the same tool twice with similar queries
- If search returns no useful results, proceed with review anyway

## Output Format
You MUST respond with valid JSON only. No markdown, no explanations, no text before or after JSON.

## Review Guidelines
- Focus on security, bugs, and logic errors
- Be specific with line numbers from the diff
- If no issues found, return empty arrays with score 85-100
- Do not nitpick style issues`
}

func (r *PerFileReviewer) getResponseFormat() string {
	return `

{
  "summary": "One sentence summary of this file's changes",
  "score": 85,
  "issues": [
    {
      "line": 42,
      "severity": "warning",
      "category": "security",
      "message": "What the issue is",
      "suggestion": "How to fix it"
    }
  ],
  "suggestions": [
    {
      "category": "best_practice",
      "title": "Improvement idea",
      "description": "Details",
      "priority": "medium"
    }
  ]
}

severity: "critical", "warning", "info"
category: "security", "bugs", "style", "performance", "best_practice"
priority: "high", "medium", "low"
Return empty arrays if no issues found.`
}

// aggregateResults combines per-file results into overall review
func (r *PerFileReviewer) aggregateResults(results []FileReviewResult) *analyzer.AnalysisResultParsed {
	if len(results) == 0 {
		return &analyzer.AnalysisResultParsed{
			Summary: "No files reviewed",
			Score:   100,
		}
	}

	var allIssues []analyzer.Issue
	var allSuggestions []analyzer.SuggestionItem
	var summaries []string
	totalScore := 0
	reviewedCount := 0

	for _, r := range results {
		if r.Error == nil {
			totalScore += r.Score
			reviewedCount++
		}
		
		// Add file path to each issue if not set
		for _, issue := range r.Issues {
			if issue.FilePath == "" {
				issue.FilePath = r.FilePath
			}
			allIssues = append(allIssues, issue)
		}
		
		allSuggestions = append(allSuggestions, r.Suggestions...)
		
		if r.Summary != "" {
			summaries = append(summaries, fmt.Sprintf("**%s**: %s", r.FilePath, r.Summary))
		}
	}

	// Calculate average score
	avgScore := 80 // Default
	if reviewedCount > 0 {
		avgScore = totalScore / reviewedCount
	}

	// Build overall summary
	overallSummary := fmt.Sprintf("Reviewed %d files. Found %d issues.", len(results), len(allIssues))
	if len(summaries) > 0 && len(summaries) <= 5 {
		overallSummary += "\n\n" + strings.Join(summaries, "\n")
	}

	return &analyzer.AnalysisResultParsed{
		Summary:     overallSummary,
		Score:       avgScore,
		Issues:      allIssues,
		Suggestions: allSuggestions,
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

