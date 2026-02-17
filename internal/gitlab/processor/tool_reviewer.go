// Package processor implements tool-based code review with LLM function calling
package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aigateway/internal/gitlab/analyzer"
	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/rag"

	"github.com/sirupsen/logrus"
)

const (
	// DefaultMaxToolBasedIterations is the max number of LLM round-trips for tool-based review.
	// Higher than per-file mode because the model both reads context AND writes results.
	DefaultMaxToolBasedIterations = 15
)

// ============================================================================
// ReviewCollector — accumulates results from output tool calls
// ============================================================================

// ReviewCollector accumulates review results from individual tool calls.
type ReviewCollector struct {
	issues      []analyzer.Issue
	suggestions []analyzer.SuggestionItem
	summary     string
	score       int
	scoreSet    bool
	finished    bool
}

// NewReviewCollector creates a new empty collector.
func NewReviewCollector() *ReviewCollector {
	return &ReviewCollector{
		issues:      make([]analyzer.Issue, 0),
		suggestions: make([]analyzer.SuggestionItem, 0),
	}
}

// AddIssue adds a code issue with normalization.
func (c *ReviewCollector) AddIssue(issue analyzer.Issue) {
	issue.Severity = analyzer.NormalizeSeverity(issue.Severity)
	issue.Category = analyzer.NormalizeCategory(issue.Category)
	if issue.Line < 1 {
		issue.Line = 1
	}
	c.issues = append(c.issues, issue)
}

// AddSuggestion adds an improvement suggestion with normalization.
func (c *ReviewCollector) AddSuggestion(sug analyzer.SuggestionItem) {
	sug.Priority = analyzer.NormalizePriority(sug.Priority)
	if sug.Category != "" {
		sug.Category = analyzer.NormalizeCategory(sug.Category)
	}
	c.suggestions = append(c.suggestions, sug)
}

// SetSummary sets the overall review summary and score (clamped 0-100).
func (c *ReviewCollector) SetSummary(summary string, score int) {
	c.summary = summary
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	c.score = score
	c.scoreSet = true
}

// Finish marks the review as complete.
func (c *ReviewCollector) Finish() {
	c.finished = true
}

// IsFinished returns true if the model called finish_review.
func (c *ReviewCollector) IsFinished() bool {
	return c.finished
}

// ToResult converts collected data to standard AnalysisResultParsed.
func (c *ReviewCollector) ToResult() *analyzer.AnalysisResultParsed {
	score := c.score
	if !c.scoreSet {
		if len(c.issues) == 0 {
			score = 100
		} else {
			score = 70
		}
	}

	summary := c.summary
	if summary == "" {
		if len(c.issues) == 0 {
			summary = "No issues found in the code changes."
		} else {
			summary = fmt.Sprintf("Found %d potential issues in the code.", len(c.issues))
		}
	}

	return &analyzer.AnalysisResultParsed{
		Summary:     summary,
		Score:       score,
		Issues:      c.issues,
		Suggestions: c.suggestions,
	}
}

// ============================================================================
// ToolBasedReviewer — reviews using read + output tools
// ============================================================================

// ToolBasedReviewer reviews MR diffs using tool calling for both context gathering and result submission.
type ToolBasedReviewer struct {
	readTools      *ReviewTools
	llmBaseURL     string
	llmAPIKey      string
	httpClient     *http.Client
	logger         *logrus.Logger
	maxTokens      int
	reviewLanguage string
}

// NewToolBasedReviewer creates a new tool-based reviewer.
func NewToolBasedReviewer(
	ragService *rag.RAGService,
	projectID string,
	llmBaseURL string,
	llmAPIKey string,
	maxTokens int,
	reviewLanguage string,
	logger *logrus.Logger,
) *ToolBasedReviewer {
	if reviewLanguage == "" {
		reviewLanguage = "en"
	}
	return &ToolBasedReviewer{
		readTools:      NewReviewTools(ragService, projectID, logger),
		llmBaseURL:     llmBaseURL,
		llmAPIKey:      llmAPIKey,
		httpClient:     &http.Client{Timeout: 180 * time.Second},
		logger:         logger,
		maxTokens:      maxTokens,
		reviewLanguage: reviewLanguage,
	}
}

// Review reviews all diffs using tool-based approach and returns aggregated results.
func (r *ToolBasedReviewer) Review(
	ctx context.Context,
	diffs []client.Diff,
	ragContext []rag.CodeChunk,
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
	}).Info("Starting tool-based review")

	// Build initial prompt with all diffs
	prompt := r.buildPrompt(diffs, ragContext)

	// Combine read tools + output tools
	allTools := append(r.readTools.GetToolDefinitions(), GetOutputToolDefinitions()...)

	messages := []map[string]interface{}{
		{"role": "system", "content": r.getSystemPrompt()},
		{"role": "user", "content": prompt},
	}

	collector := NewReviewCollector()
	totalTokens := 0
	seenQueries := make(map[string]int)

	for i := 0; i < DefaultMaxToolBasedIterations; i++ {
		response, toolCalls, tokens, err := r.callLLM(ctx, modelID, messages, allTools)
		totalTokens += tokens

		if err != nil {
			r.logger.WithError(err).Error("LLM call failed in tool-based review")
			return nil, totalTokens, err
		}

		// No tool calls — model returned text (fallback to JSON parsing)
		if len(toolCalls) == 0 {
			if response != "" {
				r.logger.Debug("Model returned text instead of tool calls, attempting JSON fallback")
				return r.handleTextFallback(response, collector, totalTokens)
			}
			break
		}

		// Loop detection
		allRepeated := true
		for _, tc := range toolCalls {
			key := tc.Function.Name + ":" + tc.Function.Arguments
			seenQueries[key]++
			if seenQueries[key] <= 2 {
				allRepeated = false
			}
		}

		if allRepeated {
			r.logger.Warn("Detected tool call loop in tool-based review, forcing completion")
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": "You've repeated the same tool calls. Please call set_review_summary and finish_review now.",
			})
			continue
		}

		r.logger.WithFields(logrus.Fields{
			"tool_calls": len(toolCalls),
			"iteration":  i + 1,
		}).Debug("Processing tool calls")

		// Add assistant message with tool calls
		messages = append(messages, map[string]interface{}{
			"role":       "assistant",
			"content":    response,
			"tool_calls": toolCalls,
		})

		// Execute each tool call
		for _, tc := range toolCalls {
			result := r.executeTool(ctx, tc, collector)
			messages = append(messages, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": result.ToolCallID,
				"content":      result.Content,
			})
		}

		if collector.IsFinished() {
			r.logger.Debug("Review finished via finish_review tool")
			break
		}
	}

	analysisResult := collector.ToResult()

	r.logger.WithFields(logrus.Fields{
		"total_issues":  len(analysisResult.Issues),
		"overall_score": analysisResult.Score,
		"total_tokens":  totalTokens,
	}).Info("Tool-based review completed")

	return analysisResult, totalTokens, nil
}

// executeTool routes a tool call to either read tools or output handlers.
func (r *ToolBasedReviewer) executeTool(ctx context.Context, tc ToolCall, collector *ReviewCollector) *ToolResult {
	switch tc.Function.Name {
	// Read tools — delegate to ReviewTools
	case "search_codebase", "get_function_definition", "get_type_definition":
		result, err := r.readTools.ExecuteTool(ctx, tc)
		if err != nil {
			return &ToolResult{
				ToolCallID: tc.ID,
				Role:       "tool",
				Content:    fmt.Sprintf("Error: %v", err),
			}
		}
		return result

	// Output tools — handle locally
	case "report_issue":
		return r.handleReportIssue(tc, collector)
	case "report_suggestion":
		return r.handleReportSuggestion(tc, collector)
	case "set_review_summary":
		return r.handleSetReviewSummary(tc, collector)
	case "finish_review":
		return r.handleFinishReview(tc, collector)

	default:
		return &ToolResult{
			ToolCallID: tc.ID,
			Role:       "tool",
			Content:    fmt.Sprintf("Error: unknown tool '%s'", tc.Function.Name),
		}
	}
}

// handleReportIssue processes a report_issue tool call.
func (r *ToolBasedReviewer) handleReportIssue(tc ToolCall, collector *ReviewCollector) *ToolResult {
	var args struct {
		FilePath   string `json:"file_path"`
		Line       int    `json:"line"`
		EndLine    int    `json:"end_line,omitempty"`
		Severity   string `json:"severity"`
		Category   string `json:"category"`
		Message    string `json:"message"`
		Suggestion string `json:"suggestion,omitempty"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return &ToolResult{
			ToolCallID: tc.ID,
			Role:       "tool",
			Content:    fmt.Sprintf("Error parsing arguments: %v", err),
		}
	}

	collector.AddIssue(analyzer.Issue{
		FilePath:   args.FilePath,
		Line:       args.Line,
		EndLine:    args.EndLine,
		Severity:   args.Severity,
		Category:   args.Category,
		Message:    args.Message,
		Suggestion: args.Suggestion,
	})

	return &ToolResult{
		ToolCallID: tc.ID,
		Role:       "tool",
		Content:    fmt.Sprintf("Issue recorded for %s:%d", args.FilePath, args.Line),
	}
}

// handleReportSuggestion processes a report_suggestion tool call.
func (r *ToolBasedReviewer) handleReportSuggestion(tc ToolCall, collector *ReviewCollector) *ToolResult {
	var args struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
		FilePath    string `json:"file_path,omitempty"`
		Line        int    `json:"line,omitempty"`
		Category    string `json:"category,omitempty"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return &ToolResult{
			ToolCallID: tc.ID,
			Role:       "tool",
			Content:    fmt.Sprintf("Error parsing arguments: %v", err),
		}
	}

	collector.AddSuggestion(analyzer.SuggestionItem{
		FilePath:    args.FilePath,
		Line:        args.Line,
		Category:    args.Category,
		Title:       args.Title,
		Description: args.Description,
		Priority:    args.Priority,
	})

	return &ToolResult{
		ToolCallID: tc.ID,
		Role:       "tool",
		Content:    "Suggestion recorded: " + args.Title,
	}
}

// handleSetReviewSummary processes a set_review_summary tool call.
func (r *ToolBasedReviewer) handleSetReviewSummary(tc ToolCall, collector *ReviewCollector) *ToolResult {
	var args struct {
		Summary      string `json:"summary"`
		OverallScore int    `json:"overall_score"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return &ToolResult{
			ToolCallID: tc.ID,
			Role:       "tool",
			Content:    fmt.Sprintf("Error parsing arguments: %v", err),
		}
	}

	collector.SetSummary(args.Summary, args.OverallScore)

	return &ToolResult{
		ToolCallID: tc.ID,
		Role:       "tool",
		Content:    fmt.Sprintf("Summary set with score %d", args.OverallScore),
	}
}

// handleFinishReview processes a finish_review tool call.
func (r *ToolBasedReviewer) handleFinishReview(tc ToolCall, collector *ReviewCollector) *ToolResult {
	collector.Finish()
	return &ToolResult{
		ToolCallID: tc.ID,
		Role:       "tool",
		Content:    "Review completed.",
	}
}

// handleTextFallback falls back to JSON parsing when model returns text instead of tool calls.
func (r *ToolBasedReviewer) handleTextFallback(
	response string,
	collector *ReviewCollector,
	totalTokens int,
) (*analyzer.AnalysisResultParsed, int, error) {
	// If collector already has data, prefer that
	if len(collector.issues) > 0 || len(collector.suggestions) > 0 || collector.scoreSet {
		r.logger.Debug("Using collector data with text fallback")
		result := collector.ToResult()
		return result, totalTokens, nil
	}

	// Try JSON parsing
	parsed, err := analyzer.ParseAnalysisResponse(response)
	if err != nil {
		return nil, totalTokens, fmt.Errorf("text fallback: failed to parse response: %w", err)
	}
	return parsed, totalTokens, nil
}

// callLLM calls the LLM API with tools.
func (r *ToolBasedReviewer) callLLM(
	ctx context.Context,
	modelID string,
	messages []map[string]interface{},
	tools []ToolDefinition,
) (string, []ToolCall, int, error) {
	requestBody := map[string]interface{}{
		"model":       modelID,
		"messages":    messages,
		"tools":       tools,
		"tool_choice": "auto",
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

// buildPrompt builds the initial prompt with all diffs and RAG context.
func (r *ToolBasedReviewer) buildPrompt(diffs []client.Diff, ragContext []rag.CodeChunk) string {
	var sb strings.Builder

	sb.WriteString("## Code Changes to Review\n\n")

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

	if len(ragContext) > 0 {
		sb.WriteString("\n## Related Code Context (from existing codebase)\n\n")
		for _, chunk := range ragContext {
			sb.WriteString(fmt.Sprintf("### From `%s`:\n", chunk.FilePath))
			sb.WriteString(fmt.Sprintf("```%s\n", chunk.Language))
			content := chunk.Content
			if len(content) > 2000 {
				content = content[:2000] + "\n... (truncated)"
			}
			sb.WriteString(content)
			sb.WriteString("\n```\n\n")
		}
	}

	sb.WriteString(`
## Instructions

Review the code changes above. Use the available tools to:

1. **Gather context** (if needed): Use search_codebase, get_function_definition, get_type_definition
   to understand unfamiliar functions or types. Use sparingly (max 2-3 calls).

2. **Report issues**: Call report_issue for each problem you find.
   - Focus on security vulnerabilities, bugs, and logic errors
   - Use line numbers from diff @@ markers
   - Be specific and actionable

3. **Report suggestions**: Call report_suggestion for general improvements.

4. **Set summary**: Call set_review_summary with an overall assessment and score (0-100).

5. **Finish**: Call finish_review when done.

If no issues are found, call set_review_summary with a high score and finish_review.
`)

	return sb.String()
}

// getSystemPrompt returns the system prompt for tool-based review.
func (r *ToolBasedReviewer) getSystemPrompt() string {
	langInstruction := "Write all text (messages, suggestions, summary) in English."
	if r.reviewLanguage == "ru" {
		langInstruction = "Пиши весь текст (messages, suggestions, summary) на русском языке."
	}

	return fmt.Sprintf(`You are an expert code reviewer. Review code changes using the provided tools.

## Language
%s

## Available Tools

### Context tools (read-only, use sparingly):
- search_codebase: Semantic search for related code
- get_function_definition: Look up function implementation
- get_type_definition: Look up struct/interface definition

### Output tools (use to submit results):
- report_issue: Report a code issue (call for each issue found)
- report_suggestion: Report an improvement suggestion
- set_review_summary: Set overall summary and score (call once)
- finish_review: Signal review completion (call last)

## Review Guidelines
- Focus on security, bugs, and logic errors
- Be specific with line numbers from the diff
- Do NOT nitpick style issues unless they affect readability
- If no issues found, set a high score (85-100) and finish
- Always call set_review_summary before finish_review`, langInstruction)
}
