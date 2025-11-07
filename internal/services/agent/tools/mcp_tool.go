// Package tools provides MCP (Model Context Protocol) tool integration for Agent (v2.5.0+)
package tools

import (
	"context"
	"fmt"
	"time"

	"aigateway/internal/models"
)

// MCPTool represents a tool from an MCP server
type MCPTool struct {
	serverName  string // MCP server name
	toolName    string // Tool name from MCP
	description string
	dangerous   bool
	schema      map[string]interface{} // JSON Schema for parameters
	
	// MCP connection
	client *MCPClient
}

// NewMCPTool creates a new MCP tool wrapper
func NewMCPTool(client *MCPClient, serverName, toolName, description string, schema map[string]interface{}, dangerous bool) *MCPTool {
	return &MCPTool{
		serverName:  serverName,
		toolName:    toolName,
		description: description,
		dangerous:   dangerous,
		schema:      schema,
		client:      client,
	}
}

func (t *MCPTool) GetInfo() models.AgentTool {
	return models.AgentTool{
		ID:          fmt.Sprintf("mcp.%s.%s", t.serverName, t.toolName),
		Name:        fmt.Sprintf("mcp.%s", t.toolName),
		Category:    models.AgentToolCategoryMCP,
		Description: fmt.Sprintf("[%s] %s", t.serverName, t.description),
		Parameters:  t.schema,
		Dangerous:   t.dangerous,
		Available:   t.client != nil && t.client.IsConnected(),
		Version:     "1.0.0",
	}
}

func (t *MCPTool) Validate(params map[string]interface{}) error {
	// TODO: Validate against JSON Schema
	// For MVP: basic validation
	if t.schema != nil {
		if required, ok := t.schema["required"].([]interface{}); ok {
			for _, req := range required {
				if reqStr, ok := req.(string); ok {
					if _, exists := params[reqStr]; !exists {
						return fmt.Errorf("required parameter '%s' is missing", reqStr)
					}
				}
			}
		}
	}
	return nil
}

func (t *MCPTool) Execute(ctx context.Context, params map[string]interface{}) (*models.AgentStepResult, error) {
	if t.client == nil {
		errMsg := "MCP client not initialized"
		return &models.AgentStepResult{
			Success: false,
			Error:   &errMsg,
		}, nil
	}

	if !t.client.IsConnected() {
		errMsg := fmt.Sprintf("MCP server '%s' is not connected", t.serverName)
		return &models.AgentStepResult{
			Success: false,
			Error:   &errMsg,
		}, nil
	}

	// Execute tool via MCP client
	startTime := time.Now()
	result, err := t.client.CallTool(ctx, t.toolName, params)
	duration := time.Since(startTime)

	if err != nil {
		errMsg := fmt.Sprintf("MCP tool execution failed: %v", err)
		return &models.AgentStepResult{
			Success:  false,
			Error:    &errMsg,
			Duration: int(duration.Milliseconds()),
		}, nil
	}

	return &models.AgentStepResult{
		Success:  true,
		Output:   result,
		Duration: int(duration.Milliseconds()),
	}, nil
}

// ========================================
// MCP Client (simplified for MVP)
// ========================================

// MCPClient represents a connection to an MCP server
type MCPClient struct {
	serverName string
	connected  bool
	
	// TODO: Actual MCP protocol implementation
	// For MVP: stub implementation
}

// NewMCPClient creates a new MCP client
func NewMCPClient(serverName string) *MCPClient {
	return &MCPClient{
		serverName: serverName,
		connected:  false,
	}
}

// Connect establishes connection to MCP server
func (c *MCPClient) Connect(ctx context.Context) error {
	// TODO: Implement actual MCP protocol connection
	// For MVP: stub
	c.connected = true
	return nil
}

// Disconnect closes connection to MCP server
func (c *MCPClient) Disconnect() error {
	c.connected = false
	return nil
}

// IsConnected checks if client is connected
func (c *MCPClient) IsConnected() bool {
	return c.connected
}

// ListTools returns available tools from MCP server
func (c *MCPClient) ListTools(ctx context.Context) ([]MCPToolInfo, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	// TODO: Implement actual MCP protocol tool listing
	// For MVP: return empty list
	return []MCPToolInfo{}, nil
}

// CallTool invokes a tool on MCP server
func (c *MCPClient) CallTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	if !c.connected {
		return nil, fmt.Errorf("not connected to MCP server")
	}

	// TODO: Implement actual MCP protocol tool invocation
	// For MVP: stub
	return map[string]interface{}{
		"status":  "success",
		"message": "MCP tool execution not implemented yet (stub)",
		"tool":    toolName,
		"params":  params,
	}, nil
}

// MCPToolInfo represents tool metadata from MCP server
type MCPToolInfo struct {
	Name        string
	Description string
	Schema      map[string]interface{}
	Dangerous   bool
}

// ========================================
// MCP Tool Registration Helper
// ========================================

// RegisterMCPTools discovers and registers tools from MCP servers
func RegisterMCPTools(registry *Registry, servers []string) error {
	for _, serverName := range servers {
		client := NewMCPClient(serverName)
		
		// Connect to server
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := client.Connect(ctx)
		cancel()
		
		if err != nil {
			// Log but don't fail - continue with other servers
			continue
		}

		// Discover tools
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		tools, err := client.ListTools(ctx)
		cancel()
		
		if err != nil {
			client.Disconnect()
			continue
		}

		// Register each tool
		for _, toolInfo := range tools {
			mcpTool := NewMCPTool(
				client,
				serverName,
				toolInfo.Name,
				toolInfo.Description,
				toolInfo.Schema,
				toolInfo.Dangerous,
			)
			
			if err := registry.Register(mcpTool); err != nil {
				// Log but continue
				continue
			}
		}
	}

	return nil
}

