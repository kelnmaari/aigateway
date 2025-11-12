// Package tools provides thinking/reflection tool for non-reasoning models (v2.5.0+)
package tools

import (
	"context"
	"fmt"

	"aigateway/internal/models"
)

// ========================================
// Think Tool (Pseudo-tool for reasoning)
// ========================================

// ThinkTool is a pseudo-tool that encourages the model to pause and reflect
// It doesn't execute any action - just returns a reflection prompt
// This technique helps non-reasoning models (non-o1) to think before acting
//
// Usage: When model calls think({"reflection": "..."}), it forces the model
// to generate reasoning in the "reflection" parameter, which improves decision quality.
type ThinkTool struct{}

// NewThinkTool creates a new think tool
func NewThinkTool() *ThinkTool {
	return &ThinkTool{}
}

func (t *ThinkTool) GetInfo() models.AgentTool {
	return models.AgentTool{
		ID:          "think",
		Name:        "think",
		Category:    models.AgentToolCategorySystem, // System category (not file/terminal)
		Description: "Pause and reflect on the current situation before taking action. Use this when you need to think through a complex problem or reconsider your approach.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"reflection": map[string]interface{}{
					"type":        "string",
					"description": "Your detailed thoughts about the current situation, what you've learned, and what to do next",
				},
			},
			"required": []string{"reflection"},
		},
		Dangerous: false, // Completely safe - no actions
		Available: true,
		Version:   "1.0.0",
	}
}

func (t *ThinkTool) Validate(params map[string]interface{}) error {
	reflection, ok := params["reflection"].(string)
	if !ok || reflection == "" {
		return fmt.Errorf("'reflection' parameter is required and must be a non-empty string")
	}
	return nil
}

func (t *ThinkTool) Execute(ctx context.Context, params map[string]interface{}) (*models.AgentStepResult, error) {
	reflection := params["reflection"].(string)
	
	// This tool does nothing except acknowledge the reflection
	// The key is that it forces the model to generate the "reflection" text,
	// which helps non-reasoning models think through the problem
	
	acknowledgment := fmt.Sprintf("✓ Reflection acknowledged. You took a moment to think:\n\n\"%s\"\n\nYou may now proceed with your next action.", reflection)
	
	return &models.AgentStepResult{
		Success: true,
		Output:  acknowledgment,
	}, nil
}

