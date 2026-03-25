package processor

import (
	"context"
	"encoding/json"
	"testing"

	"aigateway/internal/gitlab/analyzer"
	"aigateway/internal/models"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLogger returns a logger for tests (discards output).
func testLogger() *logrus.Logger {
	l := logrus.New()
	l.SetLevel(logrus.DebugLevel)
	return l
}

// ============================================================================
// ReviewCollector tests
// ============================================================================

func TestReviewCollector_AddIssue_Normalization(t *testing.T) {
	tests := []struct {
		name             string
		severity         string
		category         string
		expectedSeverity string
		expectedCategory string
	}{
		{"critical severity", "critical", "security", "critical", "security"},
		{"high maps to critical", "high", "bugs", "critical", "bugs"},
		{"error maps to critical", "error", "bug", "critical", "bugs"},
		{"warning severity", "warning", "style", "warning", "style"},
		{"medium maps to warning", "medium", "performance", "warning", "performance"},
		{"info severity", "info", "best_practice", "info", "best_practice"},
		{"suggestion severity", "suggestion", "sec", "suggestion", "security"},
		{"unknown severity", "unknown", "unknown", "info", "best_practice"},
		{"case insensitive", "CRITICAL", "SECURITY", "critical", "security"},
		{"trimmed whitespace", "  warning  ", "  bugs  ", "warning", "bugs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewReviewCollector()
			c.AddIssue(analyzer.Issue{
				FilePath: "test.go",
				Line:     10,
				Severity: tt.severity,
				Category: tt.category,
				Message:  "test issue",
			})

			require.Len(t, c.issues, 1)
			assert.Equal(t, tt.expectedSeverity, c.issues[0].Severity)
			assert.Equal(t, tt.expectedCategory, c.issues[0].Category)
		})
	}
}

func TestReviewCollector_AddIssue_LineClamp(t *testing.T) {
	c := NewReviewCollector()
	c.AddIssue(analyzer.Issue{
		FilePath: "test.go",
		Line:     0, // should be clamped to 1
		Severity: "info",
		Category: "bugs",
		Message:  "test",
	})

	require.Len(t, c.issues, 1)
	assert.Equal(t, 1, c.issues[0].Line)
}

func TestReviewCollector_AddSuggestion_Normalization(t *testing.T) {
	tests := []struct {
		name             string
		priority         string
		category         string
		expectedPriority string
		expectedCategory string
	}{
		{"high priority", "high", "security", "high", "security"},
		{"critical maps to high", "critical", "bugs", "high", "bugs"},
		{"medium priority", "medium", "", "medium", ""},
		{"normal maps to medium", "normal", "style", "medium", "style"},
		{"low priority", "low", "performance", "low", "performance"},
		{"unknown maps to medium", "unknown", "", "medium", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewReviewCollector()
			c.AddSuggestion(analyzer.SuggestionItem{
				Title:       "test",
				Description: "desc",
				Priority:    tt.priority,
				Category:    tt.category,
			})

			require.Len(t, c.suggestions, 1)
			assert.Equal(t, tt.expectedPriority, c.suggestions[0].Priority)
			assert.Equal(t, tt.expectedCategory, c.suggestions[0].Category)
		})
	}
}

func TestReviewCollector_SetSummary_ScoreClamping(t *testing.T) {
	tests := []struct {
		name     string
		score    int
		expected int
	}{
		{"normal score", 85, 85},
		{"zero score", 0, 0},
		{"max score", 100, 100},
		{"negative clamped", -10, 0},
		{"over 100 clamped", 150, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewReviewCollector()
			c.SetSummary("test", tt.score)
			assert.Equal(t, tt.expected, c.score)
			assert.True(t, c.scoreSet)
		})
	}
}

func TestReviewCollector_ToResult_NoIssues(t *testing.T) {
	c := NewReviewCollector()
	result := c.ToResult()

	assert.Equal(t, 100, result.Score) // No issues = perfect
	assert.Contains(t, result.Summary, "No issues found")
	assert.Empty(t, result.Issues)
	assert.Empty(t, result.Suggestions)
}

func TestReviewCollector_ToResult_WithIssues(t *testing.T) {
	c := NewReviewCollector()
	c.AddIssue(analyzer.Issue{
		FilePath: "main.go",
		Line:     42,
		Severity: "warning",
		Category: "security",
		Message:  "SQL injection risk",
	})
	c.AddSuggestion(analyzer.SuggestionItem{
		Title:       "Use parameterized queries",
		Description: "Replace string interpolation",
		Priority:    "high",
	})
	c.SetSummary("Found security issues", 60)

	result := c.ToResult()

	assert.Equal(t, 60, result.Score)
	assert.Equal(t, "Found security issues", result.Summary)
	require.Len(t, result.Issues, 1)
	assert.Equal(t, "main.go", result.Issues[0].FilePath)
	assert.Equal(t, 42, result.Issues[0].Line)
	require.Len(t, result.Suggestions, 1)
	assert.Equal(t, "Use parameterized queries", result.Suggestions[0].Title)
}

func TestReviewCollector_ToResult_NoScoreSet_WithIssues(t *testing.T) {
	c := NewReviewCollector()
	c.AddIssue(analyzer.Issue{
		FilePath: "main.go",
		Line:     1,
		Severity: "info",
		Category: "style",
		Message:  "minor issue",
	})

	result := c.ToResult()
	assert.Equal(t, 70, result.Score) // Default when issues exist but no score set
}

func TestReviewCollector_Finish(t *testing.T) {
	c := NewReviewCollector()
	assert.False(t, c.IsFinished())

	c.Finish()
	assert.True(t, c.IsFinished())
}

// ============================================================================
// Output tool handler tests
// ============================================================================

func TestToolBasedReviewer_HandleReportIssue(t *testing.T) {
	r := &ToolBasedReviewer{}
	collector := NewReviewCollector()

	args, _ := json.Marshal(map[string]any{
		"file_path":  "auth.go",
		"line":       15,
		"severity":   "critical",
		"category":   "security",
		"message":    "Hardcoded password",
		"suggestion": "Use environment variables",
	})

	tc := ToolCall{
		ID:   "call_1",
		Type: "function",
	}
	tc.Function.Name = "report_issue"
	tc.Function.Arguments = string(args)

	result := r.handleReportIssue(tc, collector)

	assert.Contains(t, result.Content, "Issue recorded")
	assert.Equal(t, "call_1", result.ToolCallID)
	require.Len(t, collector.issues, 1)
	assert.Equal(t, "auth.go", collector.issues[0].FilePath)
	assert.Equal(t, 15, collector.issues[0].Line)
	assert.Equal(t, "critical", collector.issues[0].Severity)
	assert.Equal(t, "Use environment variables", collector.issues[0].Suggestion)
}

func TestToolBasedReviewer_HandleReportIssue_InvalidJSON(t *testing.T) {
	r := &ToolBasedReviewer{}
	collector := NewReviewCollector()

	tc := ToolCall{ID: "call_2"}
	tc.Function.Name = "report_issue"
	tc.Function.Arguments = "not valid json"

	result := r.handleReportIssue(tc, collector)

	assert.Contains(t, result.Content, "Error parsing")
	assert.Empty(t, collector.issues)
}

func TestToolBasedReviewer_HandleReportSuggestion(t *testing.T) {
	r := &ToolBasedReviewer{}
	collector := NewReviewCollector()

	args, _ := json.Marshal(map[string]any{
		"title":       "Add input validation",
		"description": "Validate user input before processing",
		"priority":    "high",
		"file_path":   "handler.go",
		"line":        50,
		"category":    "security",
	})

	tc := ToolCall{ID: "call_3"}
	tc.Function.Name = "report_suggestion"
	tc.Function.Arguments = string(args)

	result := r.handleReportSuggestion(tc, collector)

	assert.Contains(t, result.Content, "Suggestion recorded")
	require.Len(t, collector.suggestions, 1)
	assert.Equal(t, "Add input validation", collector.suggestions[0].Title)
	assert.Equal(t, "high", collector.suggestions[0].Priority)
	assert.Equal(t, "handler.go", collector.suggestions[0].FilePath)
}

func TestToolBasedReviewer_HandleSetReviewSummary(t *testing.T) {
	r := &ToolBasedReviewer{}
	collector := NewReviewCollector()

	args, _ := json.Marshal(map[string]any{
		"summary":       "Code looks good with minor issues",
		"overall_score": 85,
	})

	tc := ToolCall{ID: "call_4"}
	tc.Function.Name = "set_review_summary"
	tc.Function.Arguments = string(args)

	result := r.handleSetReviewSummary(tc, collector)

	assert.Contains(t, result.Content, "Summary set with score 85")
	assert.Equal(t, "Code looks good with minor issues", collector.summary)
	assert.Equal(t, 85, collector.score)
	assert.True(t, collector.scoreSet)
}

func TestToolBasedReviewer_HandleFinishReview(t *testing.T) {
	r := &ToolBasedReviewer{}
	collector := NewReviewCollector()

	tc := ToolCall{ID: "call_5"}
	tc.Function.Name = "finish_review"
	tc.Function.Arguments = "{}"

	result := r.handleFinishReview(tc, collector)

	assert.Contains(t, result.Content, "Review completed")
	assert.True(t, collector.IsFinished())
}

// ============================================================================
// Tool routing tests
// ============================================================================

func TestToolBasedReviewer_ExecuteTool_RoutesToOutputTools(t *testing.T) {
	r := &ToolBasedReviewer{}
	collector := NewReviewCollector()

	args, _ := json.Marshal(map[string]any{
		"file_path": "test.go",
		"line":      1,
		"severity":  "info",
		"category":  "style",
		"message":   "test",
	})

	tc := ToolCall{ID: "call_6"}
	tc.Function.Name = "report_issue"
	tc.Function.Arguments = string(args)

	result := r.executeTool(context.Background(), tc, collector)
	assert.Contains(t, result.Content, "Issue recorded")
}

func TestToolBasedReviewer_ExecuteTool_UnknownTool(t *testing.T) {
	r := &ToolBasedReviewer{}
	collector := NewReviewCollector()

	tc := ToolCall{ID: "call_7"}
	tc.Function.Name = "nonexistent_tool"
	tc.Function.Arguments = "{}"

	result := r.executeTool(context.Background(), tc, collector)
	assert.Contains(t, result.Content, "unknown tool")
}

// ============================================================================
// Fallback tests
// ============================================================================

func TestToolBasedReviewer_HandleTextFallback_ValidJSON(t *testing.T) {
	r := &ToolBasedReviewer{}
	collector := NewReviewCollector()

	// Use overall_score (the primary field name in the alt format parser)
	jsonResponse := `{
		"summary": "Code looks good",
		"overall_score": 90,
		"issues": [],
		"suggestions": []
	}`

	result, tokens, err := r.handleTextFallback(jsonResponse, collector, 100)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 100, tokens)
	assert.Equal(t, "Code looks good", result.Summary)
}

func TestToolBasedReviewer_HandleTextFallback_InvalidJSON(t *testing.T) {
	r := &ToolBasedReviewer{}
	collector := NewReviewCollector()

	_, _, err := r.handleTextFallback("not json at all", collector, 0)
	assert.Error(t, err)
}

func TestToolBasedReviewer_HandleTextFallback_PreferCollectorData(t *testing.T) {
	r := &ToolBasedReviewer{logger: testLogger()}
	collector := NewReviewCollector()

	// Add some data to collector first
	collector.AddIssue(analyzer.Issue{
		FilePath: "main.go",
		Line:     10,
		Severity: "warning",
		Category: "bugs",
		Message:  "potential bug",
	})
	collector.SetSummary("Review from tools", 75)

	// Even with valid JSON text, should prefer collector data
	result, _, err := r.handleTextFallback(`{"summary": "different"}`, collector, 0)

	require.NoError(t, err)
	assert.Equal(t, "Review from tools", result.Summary)
	assert.Equal(t, 75, result.Score)
	require.Len(t, result.Issues, 1)
}

// ============================================================================
// Schema tests
// ============================================================================

func TestGetOutputToolDefinitions_AllPresent(t *testing.T) {
	tools := GetOutputToolDefinitions()

	require.Len(t, tools, 4)

	names := make(map[string]bool)
	for _, tool := range tools {
		assert.Equal(t, "function", tool.Type)
		assert.NotEmpty(t, tool.Function.Name)
		assert.NotEmpty(t, tool.Function.Description)
		assert.NotNil(t, tool.Function.Parameters)
		names[tool.Function.Name] = true
	}

	assert.True(t, names["report_issue"])
	assert.True(t, names["report_suggestion"])
	assert.True(t, names["set_review_summary"])
	assert.True(t, names["finish_review"])
}

func TestGetOutputToolDefinitions_ReportIssueSchema(t *testing.T) {
	tools := GetOutputToolDefinitions()

	var reportIssue *ToolDefinition
	for i := range tools {
		if tools[i].Function.Name == "report_issue" {
			reportIssue = &tools[i]
			break
		}
	}

	require.NotNil(t, reportIssue)

	params := reportIssue.Function.Parameters
	properties, ok := params["properties"].(map[string]any)
	require.True(t, ok)

	// Check required properties exist
	assert.Contains(t, properties, "file_path")
	assert.Contains(t, properties, "line")
	assert.Contains(t, properties, "severity")
	assert.Contains(t, properties, "category")
	assert.Contains(t, properties, "message")
	assert.Contains(t, properties, "suggestion")
	assert.Contains(t, properties, "end_line")

	// Check required list
	required, ok := params["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "file_path")
	assert.Contains(t, required, "line")
	assert.Contains(t, required, "severity")
	assert.Contains(t, required, "category")
	assert.Contains(t, required, "message")
}

// ============================================================================
// Model tests (GetReviewMode)
// ============================================================================

func TestGetReviewMode_BackwardCompat(t *testing.T) {
	s := models.GitLabProjectSettings{
		PerFileReview: true,
	}
	assert.Equal(t, models.ReviewModePerFile, s.GetReviewMode())
}

func TestGetReviewMode_ExplicitMode(t *testing.T) {
	s := models.GitLabProjectSettings{
		ReviewMode:    models.ReviewModeToolBased,
		PerFileReview: true, // Should be overridden by ReviewMode
	}
	assert.Equal(t, models.ReviewModeToolBased, s.GetReviewMode())
}

func TestGetReviewMode_Default(t *testing.T) {
	s := models.GitLabProjectSettings{}
	assert.Equal(t, models.ReviewModeStandard, s.GetReviewMode())
}

func TestGetReviewMode_AllModes(t *testing.T) {
	tests := []struct {
		mode     string
		expected string
	}{
		{models.ReviewModeStandard, models.ReviewModeStandard},
		{models.ReviewModePerFile, models.ReviewModePerFile},
		{models.ReviewModeToolBased, models.ReviewModeToolBased},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			s := models.GitLabProjectSettings{ReviewMode: tt.mode}
			assert.Equal(t, tt.expected, s.GetReviewMode())
		})
	}
}
