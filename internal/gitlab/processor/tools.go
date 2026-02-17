// Package processor implements tool-based code review with LLM function calling
package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"aigateway/internal/gitlab/rag"

	"github.com/sirupsen/logrus"
)

// Tool definitions for OpenAI-compatible function calling
type ToolDefinition struct {
	Type     string       `json:"type"`
	Function FunctionDef  `json:"function"`
}

type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Role       string `json:"role"`
	Content    string `json:"content"`
}

// ReviewTools provides tools for LLM to search codebase
type ReviewTools struct {
	ragService *rag.RAGService
	projectID  string
	logger     *logrus.Logger
}

// NewReviewTools creates tool executor
func NewReviewTools(ragService *rag.RAGService, projectID string, logger *logrus.Logger) *ReviewTools {
	return &ReviewTools{
		ragService: ragService,
		projectID:  projectID,
		logger:     logger,
	}
}

// GetToolDefinitions returns OpenAI-compatible tool definitions
func (t *ReviewTools) GetToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "search_codebase",
				Description: "Search the project codebase for relevant code using semantic search. Use this to find related functions, implementations, or patterns.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Natural language query describing what you're looking for (e.g., 'authentication middleware', 'database connection handling', 'error handling patterns')",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum number of results to return (default: 3, max: 10)",
							"default":     3,
						},
					},
					"required": []string{"query"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "get_function_definition",
				Description: "Get the full definition of a specific function or method by name. Use this when you see a function call and need to understand what it does.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"function_name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the function to find (e.g., 'handleRequest', 'ValidateToken', 'NewService')",
						},
						"file_hint": map[string]interface{}{
							"type":        "string",
							"description": "Optional: partial file path to narrow search (e.g., 'auth/', 'handlers/')",
						},
					},
					"required": []string{"function_name"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "get_type_definition",
				Description: "Get the definition of a struct, interface, or type. Use this when you need to understand a data structure.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type_name": map[string]interface{}{
							"type":        "string",
							"description": "Name of the type/struct/interface to find (e.g., 'User', 'Config', 'Handler')",
						},
					},
					"required": []string{"type_name"},
				},
			},
		},
	}
}

// GetOutputToolDefinitions returns tool definitions for structured review output.
// These tools allow the LLM to submit review results incrementally via tool calls
// instead of returning raw JSON, which is more reliable and validates per-call.
func GetOutputToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "report_issue",
				Description: "Report a code issue found during review. Call this for each issue you find.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"file_path": map[string]interface{}{
							"type":        "string",
							"description": "Path to the file containing the issue",
						},
						"line": map[string]interface{}{
							"type":        "integer",
							"description": "Line number where the issue starts (from diff @@ markers)",
						},
						"end_line": map[string]interface{}{
							"type":        "integer",
							"description": "End line number for multi-line issues (optional)",
						},
						"severity": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"critical", "warning", "info", "suggestion"},
							"description": "Issue severity level",
						},
						"category": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"security", "bugs", "style", "performance", "best_practice"},
							"description": "Issue category",
						},
						"message": map[string]interface{}{
							"type":        "string",
							"description": "Clear description of what the issue is",
						},
						"suggestion": map[string]interface{}{
							"type":        "string",
							"description": "How to fix the issue (optional)",
						},
					},
					"required": []string{"file_path", "line", "severity", "category", "message"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "report_suggestion",
				Description: "Report an improvement suggestion for the code. Call this for general recommendations.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Short title for the suggestion",
						},
						"description": map[string]interface{}{
							"type":        "string",
							"description": "Detailed description of the improvement",
						},
						"priority": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"high", "medium", "low"},
							"description": "Suggestion priority",
						},
						"file_path": map[string]interface{}{
							"type":        "string",
							"description": "File this suggestion relates to (optional)",
						},
						"line": map[string]interface{}{
							"type":        "integer",
							"description": "Line number if applicable (optional)",
						},
						"category": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"security", "bugs", "style", "performance", "best_practice"},
							"description": "Suggestion category (optional)",
						},
					},
					"required": []string{"title", "description", "priority"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "set_review_summary",
				Description: "Set the overall review summary and score. Call this once after reviewing all files.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"summary": map[string]interface{}{
							"type":        "string",
							"description": "Brief overall assessment of the code changes in 1-3 sentences",
						},
						"overall_score": map[string]interface{}{
							"type":        "integer",
							"description": "Overall code quality score from 0 to 100 (higher is better)",
							"minimum":     0,
							"maximum":     100,
						},
					},
					"required": []string{"summary", "overall_score"},
				},
			},
		},
		{
			Type: "function",
			Function: FunctionDef{
				Name:        "finish_review",
				Description: "Signal that the review is complete. Call this as the last tool after reporting all issues and setting the summary.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
	}
}

// ExecuteTool executes a tool call and returns the result
func (t *ReviewTools) ExecuteTool(ctx context.Context, call ToolCall) (*ToolResult, error) {
	t.logger.WithFields(logrus.Fields{
		"tool":      call.Function.Name,
		"arguments": call.Function.Arguments,
	}).Debug("Executing tool call")

	var content string
	var err error

	switch call.Function.Name {
	case "search_codebase":
		content, err = t.searchCodebase(ctx, call.Function.Arguments)
	case "get_function_definition":
		content, err = t.getFunctionDefinition(ctx, call.Function.Arguments)
	case "get_type_definition":
		content, err = t.getTypeDefinition(ctx, call.Function.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", call.Function.Name)
	}

	if err != nil {
		content = fmt.Sprintf("Error: %v", err)
	}

	return &ToolResult{
		ToolCallID: call.ID,
		Role:       "tool",
		Content:    content,
	}, nil
}

func (t *ReviewTools) searchCodebase(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Limit <= 0 {
		args.Limit = 3
	}
	if args.Limit > 10 {
		args.Limit = 10
	}

	if t.ragService == nil {
		return "RAG service not available", nil
	}

	chunks, err := t.ragService.FindSimilarCode(ctx, args.Query, t.projectID, args.Limit, "")
	if err != nil {
		return "", err
	}

	if len(chunks) == 0 {
		return "No relevant code found for this query.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d relevant code sections:\n\n", len(chunks)))
	
	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("### Result %d: %s (score: %.2f)\n", i+1, chunk.FilePath, chunk.Score))
		sb.WriteString(fmt.Sprintf("```%s\n", chunk.Language))
		// Truncate very long chunks
		content := chunk.Content
		if len(content) > 2000 {
			content = content[:2000] + "\n... (truncated)"
		}
		sb.WriteString(content)
		sb.WriteString("\n```\n\n")
	}

	return sb.String(), nil
}

func (t *ReviewTools) getFunctionDefinition(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		FunctionName string `json:"function_name"`
		FileHint     string `json:"file_hint"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Search for function definition using semantic search
	query := fmt.Sprintf("function definition %s implementation", args.FunctionName)
	if args.FileHint != "" {
		query += " in " + args.FileHint
	}

	return t.searchCodebase(ctx, fmt.Sprintf(`{"query": %q, "limit": 3}`, query))
}

func (t *ReviewTools) getTypeDefinition(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		TypeName string `json:"type_name"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Search for type/struct/interface definition
	query := fmt.Sprintf("type struct interface definition %s", args.TypeName)

	return t.searchCodebase(ctx, fmt.Sprintf(`{"query": %q, "limit": 3}`, query))
}

