// Package models provides data models for Conversational Agent Context
package models

import (
	"time"
)

// AgentContext represents the ReAct loop state for conversational agent
// This is stored as JSON in conversations.agent_context column
type AgentContext struct {
	// Current task being worked on
	CurrentTask string `json:"current_task,omitempty"`

	// ReAct loop state
	ThoughtHistory  []AgentThought      `json:"thought_history,omitempty"`  // All thoughts in sequence
	ActionHistory   []AgentActionRecord `json:"action_history,omitempty"`   // All actions executed
	ToolsUsed       []string            `json:"tools_used,omitempty"`       // Tool names used in this conversation
	PendingApprovals []string           `json:"pending_approvals,omitempty"` // IDs of pending approval requests

	// Conversation flow control
	StepNumber      int       `json:"step_number"`                // Current step in ReAct loop
	IsComplete      bool      `json:"is_complete"`                // Task completed?
	LastUpdated     time.Time `json:"last_updated"`               // Last context update
	
	// Optional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// AgentThought represents a single thought/reasoning step
type AgentThought struct {
	Timestamp   time.Time `json:"timestamp"`
	Thought     string    `json:"thought"`                // Reasoning text
	Reasoning   string    `json:"reasoning,omitempty"`    // Additional reasoning details
	Confidence  float64   `json:"confidence,omitempty"`   // 0.0-1.0, agent's confidence
	StepNumber  int       `json:"step_number"`            // Which step this thought belongs to
}

// AgentActionRecord represents a single action/tool execution in the history
// Note: AgentAction (enum) is defined in agent.go for step types
type AgentActionRecord struct {
	Timestamp   time.Time              `json:"timestamp"`
	Action      string                 `json:"action"`               // Action description
	Tool        string                 `json:"tool"`                 // Tool name (e.g., "file.read")
	Parameters  map[string]interface{} `json:"parameters,omitempty"` // Tool parameters
	Result      string                 `json:"result,omitempty"`     // Execution result
	Success     bool                   `json:"success"`              // Was action successful?
	Error       string                 `json:"error,omitempty"`      // Error message if failed
	Duration    int64                  `json:"duration_ms,omitempty"` // Execution time in milliseconds
	StepNumber  int                    `json:"step_number"`          // Which step this action belongs to
}

// NewAgentContext creates a new empty agent context
func NewAgentContext(task string) *AgentContext {
	return &AgentContext{
		CurrentTask:      task,
		ThoughtHistory:   []AgentThought{},
		ActionHistory:    []AgentActionRecord{},
		ToolsUsed:        []string{},
		PendingApprovals: []string{},
		StepNumber:       0,
		IsComplete:       false,
		LastUpdated:      time.Now(),
		Metadata:         make(map[string]interface{}),
	}
}

// AddThought adds a new thought to the context
func (ac *AgentContext) AddThought(thought AgentThought) {
	ac.ThoughtHistory = append(ac.ThoughtHistory, thought)
	ac.StepNumber++
	ac.LastUpdated = time.Now()
}

// AddAction adds a new action to the context
func (ac *AgentContext) AddAction(action AgentActionRecord) {
	ac.ActionHistory = append(ac.ActionHistory, action)
	
	// Track tool usage
	toolUsed := false
	for _, tool := range ac.ToolsUsed {
		if tool == action.Tool {
			toolUsed = true
			break
		}
	}
	if !toolUsed {
		ac.ToolsUsed = append(ac.ToolsUsed, action.Tool)
	}
	
	ac.LastUpdated = time.Now()
}

// MarkComplete marks the task as complete
func (ac *AgentContext) MarkComplete() {
	ac.IsComplete = true
	ac.LastUpdated = time.Now()
}

// GetLastThought returns the most recent thought, or nil if none
func (ac *AgentContext) GetLastThought() *AgentThought {
	if len(ac.ThoughtHistory) == 0 {
		return nil
	}
	return &ac.ThoughtHistory[len(ac.ThoughtHistory)-1]
}

// GetLastAction returns the most recent action, or nil if none
func (ac *AgentContext) GetLastAction() *AgentActionRecord {
	if len(ac.ActionHistory) == 0 {
		return nil
	}
	return &ac.ActionHistory[len(ac.ActionHistory)-1]
}

