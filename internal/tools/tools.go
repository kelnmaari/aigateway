package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// ToolDefinition represents an OpenAI-compatible tool definition.
type ToolDefinition struct {
	Type     string       `json:"type"` // "function"
	Function FunctionDef  `json:"function"`
}

// FunctionDef describes a function that can be called by the model.
type FunctionDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// ToolCall represents a tool call from the model.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // "function"
	Function FunctionCall     `json:"function"`
}

// FunctionCall represents a function call.
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolResult represents the result of a tool call.
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
}

// ToolEvent represents an event during tool execution (for UI display).
type ToolEvent struct {
	Type    string `json:"type"`    // "tool_start", "tool_end", "thinking"
	Tool    string `json:"tool"`    // Tool name (e.g., "web_search")
	Query   string `json:"query"`   // Query/action being performed
	Result  string `json:"result"`  // Result summary (for tool_end)
	Elapsed int64  `json:"elapsed"` // Duration in milliseconds
}

// Registry holds available tools.
type Registry struct {
	tavily *TavilyClient
}

// NewRegistry creates a new tool registry.
func NewRegistry(tavilyAPIKey string) *Registry {
	r := &Registry{}
	if tavilyAPIKey != "" {
		r.tavily = NewTavilyClient(tavilyAPIKey)
	}
	return r
}

// HasTavily returns true if Tavily web search is available.
func (r *Registry) HasTavily() bool {
	return r.tavily != nil
}

// GetToolDefinitions returns OpenAI-compatible tool definitions.
func (r *Registry) GetToolDefinitions() []ToolDefinition {
	var tools []ToolDefinition

	if r.tavily != nil {
		tools = append(tools, ToolDefinition{
			Type: "function",
			Function: FunctionDef{
				Name:        "web_search",
				Description: "Search the web for current information. Use this when you need up-to-date information about events, news, or any topic that requires recent data.",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"query": {
							"type": "string",
							"description": "The search query to look up"
						}
					},
					"required": ["query"]
				}`),
			},
		})
	}

	return tools
}

// ExecuteTool executes a tool call and returns the result.
func (r *Registry) ExecuteTool(ctx context.Context, call ToolCall) (*ToolResult, *ToolEvent, error) {
	switch call.Function.Name {
	case "web_search":
		return r.executeWebSearch(ctx, call)
	default:
		return nil, nil, fmt.Errorf("unknown tool: %s", call.Function.Name)
	}
}

func (r *Registry) executeWebSearch(ctx context.Context, call ToolCall) (*ToolResult, *ToolEvent, error) {
	if r.tavily == nil {
		return nil, nil, fmt.Errorf("web search not configured")
	}

	// Parse arguments
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return nil, nil, fmt.Errorf("parse arguments: %w", err)
	}

	// Execute search
	searchResult, err := r.tavily.Search(ctx, args.Query, 5)
	if err != nil {
		return nil, nil, fmt.Errorf("search failed: %w", err)
	}

	// Format result
	content := searchResult.FormatResultsAsText()

	// Create summary for UI
	resultCount := len(searchResult.Results)
	summary := fmt.Sprintf("Found %d results", resultCount)
	if resultCount > 0 {
		summary += fmt.Sprintf(" (top: %s)", searchResult.Results[0].Title)
	}

	return &ToolResult{
			ToolCallID: call.ID,
			Content:    content,
		}, &ToolEvent{
			Type:   "tool_end",
			Tool:   "web_search",
			Query:  args.Query,
			Result: summary,
		}, nil
}

