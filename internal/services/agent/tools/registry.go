// Package tools provides tool registry and execution for Agent system (v2.5.0+)
package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
)

// Tool represents an executable tool that agent can use
type Tool interface {
	// GetInfo returns tool metadata
	GetInfo() models.AgentTool

	// Execute runs the tool with given parameters
	Execute(ctx context.Context, params map[string]any) (*models.AgentStepResult, error)

	// Validate checks if parameters are valid for this tool
	Validate(params map[string]any) error
}

// Registry manages available tools for agent
type Registry struct {
	tools  map[string]Tool // tool name -> Tool
	mu     sync.RWMutex
	logger *logrus.Logger
}

// NewRegistry creates a new tool registry
func NewRegistry(logger *logrus.Logger) *Registry {
	return &Registry{
		tools:  make(map[string]Tool),
		logger: logger,
	}
}

// Register registers a new tool
func (r *Registry) Register(tool Tool) error {
	info := tool.GetInfo()

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[info.Name]; exists {
		return fmt.Errorf("tool already registered: %s", info.Name)
	}

	r.tools[info.Name] = tool
	r.logger.WithFields(logrus.Fields{
		"tool":      info.Name,
		"category":  info.Category,
		"dangerous": info.Dangerous,
	}).Debug("Tool registered")

	return nil
}

// Unregister removes a tool from registry
func (r *Registry) Unregister(toolName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[toolName]; !exists {
		return fmt.Errorf("tool not found: %s", toolName)
	}

	delete(r.tools, toolName)
	r.logger.WithField("tool", toolName).Debug("Tool unregistered")

	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(toolName string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[toolName]
	if !exists {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	return tool, nil
}

// List returns all registered tools
func (r *Registry) List() []models.AgentTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]models.AgentTool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool.GetInfo())
	}

	return tools
}

// ListByCategory returns tools filtered by category
func (r *Registry) ListByCategory(category models.AgentToolCategory) []models.AgentTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]models.AgentTool, 0)
	for _, tool := range r.tools {
		info := tool.GetInfo()
		if info.Category == category {
			tools = append(tools, info)
		}
	}

	return tools
}

// Execute executes a tool with given parameters
func (r *Registry) Execute(ctx context.Context, toolName string, params map[string]any, timeout time.Duration) (*models.AgentStepResult, error) {
	tool, err := r.Get(toolName)
	if err != nil {
		return nil, err
	}

	// Validate parameters
	if err := tool.Validate(params); err != nil {
		return nil, fmt.Errorf("parameter validation failed: %w", err)
	}

	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute tool
	startTime := time.Now()

	r.logger.WithFields(logrus.Fields{
		"tool":    toolName,
		"timeout": timeout,
	}).Debug("Executing tool")

	result, err := tool.Execute(execCtx, params)

	duration := time.Since(startTime)

	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"tool":     toolName,
			"duration": duration,
			"error":    err,
		}).Error("Tool execution failed")

		errMsg := err.Error()
		return &models.AgentStepResult{
			Success:  false,
			Error:    &errMsg,
			Duration: int(duration.Milliseconds()),
		}, nil // Return result with error, not execution error
	}

	// Update duration if not set
	if result.Duration == 0 {
		result.Duration = int(duration.Milliseconds())
	}

	r.logger.WithFields(logrus.Fields{
		"tool":     toolName,
		"duration": duration,
		"success":  result.Success,
	}).Debug("Tool execution completed")

	return result, nil
}

// IsAvailable checks if a tool is available
func (r *Registry) IsAvailable(toolName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[toolName]
	if !exists {
		return false
	}

	return tool.GetInfo().Available
}

// Count returns the number of registered tools
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}
