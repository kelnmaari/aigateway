// Package models provides Agent-related data models for Agentic AI capabilities (v2.5.0+)
package models

import (
	"time"
)

// AgentSession represents a single agent session with task planning and execution
type AgentSession struct {
	ID          string           `json:"id" db:"id"`
	UserID      string           `json:"user_id" db:"user_id"`
	TenantID    *string          `json:"tenant_id,omitempty" db:"tenant_id"`
	Task        string           `json:"task" db:"task"`                   // Original user task
	Context     map[string]interface{} `json:"context,omitempty" db:"context"` // Additional context (files, project info, etc.)
	Status      AgentStatus      `json:"status" db:"status"`               // pending, planning, executing, completed, failed, cancelled
	Plan        *AgentPlan       `json:"plan,omitempty" db:"plan"`         // Task decomposition plan
	CurrentStep int              `json:"current_step" db:"current_step"`   // Current step being executed (0-indexed)
	TotalSteps  int              `json:"total_steps" db:"total_steps"`     // Total number of steps in plan
	StartedAt   time.Time        `json:"started_at" db:"started_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
	Error       *string          `json:"error,omitempty" db:"error"`       // Error message if failed
	CreatedAt   time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" db:"updated_at"`
}

// AgentStatus represents the current state of an agent session
type AgentStatus string

const (
	AgentStatusPending   AgentStatus = "pending"   // Session created, waiting to start
	AgentStatusPlanning  AgentStatus = "planning"  // AI is decomposing task into steps
	AgentStatusExecuting AgentStatus = "executing" // Executing planned steps
	AgentStatusCompleted AgentStatus = "completed" // All steps completed successfully
	AgentStatusFailed    AgentStatus = "failed"    // Execution failed with error
	AgentStatusCancelled AgentStatus = "cancelled" // User cancelled the session
)

// AgentPlan represents the decomposed task plan with steps
type AgentPlan struct {
	ID            string      `json:"id"`
	SessionID     string      `json:"session_id"`
	Steps         []AgentStep `json:"steps"`
	EstimatedTime *int        `json:"estimated_time,omitempty"` // Estimated time in seconds (optional)
	CreatedAt     time.Time   `json:"created_at"`
}

// AgentStep represents a single step in the agent plan
type AgentStep struct {
	ID          string                 `json:"id"`
	StepNumber  int                    `json:"step_number"` // 1-indexed
	Description string                 `json:"description"` // Human-readable description
	Action      AgentAction            `json:"action"`      // file_read, file_write, terminal_execute, mcp_invoke, etc.
	Tool        string                 `json:"tool"`        // Tool identifier (e.g., "file.read", "terminal.execute")
	Parameters  map[string]interface{} `json:"parameters"`  // Tool-specific parameters
	Status      AgentStepStatus        `json:"status"`      // pending, running, completed, failed, skipped
	Result      *AgentStepResult       `json:"result,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       *string                `json:"error,omitempty"`
}

// AgentAction represents the type of action to perform
type AgentAction string

const (
	AgentActionFileRead        AgentAction = "file_read"
	AgentActionFileWrite       AgentAction = "file_write"
	AgentActionFileCreate      AgentAction = "file_create"
	AgentActionFileDelete      AgentAction = "file_delete"
	AgentActionFileList        AgentAction = "file_list"
	AgentActionFileSearch      AgentAction = "file_search"
	AgentActionTerminalExecute AgentAction = "terminal_execute"
	AgentActionMCPInvoke       AgentAction = "mcp_invoke"
	AgentActionThinking        AgentAction = "thinking" // AI is analyzing/planning
)

// AgentStepStatus represents the status of a single step
type AgentStepStatus string

const (
	AgentStepStatusPending   AgentStepStatus = "pending"
	AgentStepStatusRunning   AgentStepStatus = "running"
	AgentStepStatusCompleted AgentStepStatus = "completed"
	AgentStepStatusFailed    AgentStepStatus = "failed"
	AgentStepStatusSkipped   AgentStepStatus = "skipped"
)

// AgentStepResult represents the result of executing a step
type AgentStepResult struct {
	Success      bool                   `json:"success"`
	Output       interface{}            `json:"output,omitempty"`       // Tool-specific output
	Error        *string                `json:"error,omitempty"`        // Error message if failed
	Duration     int                    `json:"duration"`               // Execution time in milliseconds
	RequiresApproval bool               `json:"requires_approval"`      // If this step needs user approval
	Metadata     map[string]interface{} `json:"metadata,omitempty"`     // Additional metadata
}

// AgentTool represents a tool that agent can use
type AgentTool struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`        // e.g., "file.read"
	Category    AgentToolCategory      `json:"category"`    // file, terminal, mcp
	Description string                 `json:"description"` // Human-readable description
	Parameters  map[string]interface{} `json:"parameters"`  // JSON Schema for parameters
	Dangerous   bool                   `json:"dangerous"`   // If true, requires approval
	Available   bool                   `json:"available"`   // If tool is currently available
	Version     string                 `json:"version"`
}

// AgentToolCategory represents the category of a tool
type AgentToolCategory string

const (
	AgentToolCategoryFile     AgentToolCategory = "file"
	AgentToolCategoryTerminal AgentToolCategory = "terminal"
	AgentToolCategoryMCP      AgentToolCategory = "mcp"
	AgentToolCategorySystem   AgentToolCategory = "system"
)

// AgentEvent represents a real-time event for WebSocket streaming
type AgentEvent struct {
	Type      AgentEventType         `json:"type"`
	SessionID string                 `json:"session_id"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// AgentEventType represents the type of agent event
type AgentEventType string

const (
	// Session lifecycle events
	AgentEventSessionCreated   AgentEventType = "agent_session_created"  // Session created
	AgentEventPlanningStarted  AgentEventType = "agent_planning_started" // Planning started
	AgentEventPlanningCompleted AgentEventType = "agent_planning_completed" // Plan ready
	AgentEventExecutionStarted AgentEventType = "agent_execution_started" // Execution started
	AgentEventSessionCompleted AgentEventType = "agent_session_completed" // Session completed
	AgentEventSessionFailed    AgentEventType = "agent_session_failed"   // Session failed
	AgentEventSessionCancelled AgentEventType = "agent_session_cancelled" // Session cancelled
	
	// Step events
	AgentEventStepStarted   AgentEventType = "agent_step_started"   // Step started
	AgentEventStepCompleted AgentEventType = "agent_step_completed" // Step completed
	AgentEventStepFailed    AgentEventType = "agent_step_failed"    // Step failed
	AgentEventStepSkipped   AgentEventType = "agent_step_skipped"   // Step skipped due to rejection
	
	// Approval events
	AgentEventApprovalNeeded    AgentEventType = "agent_approval_needed"    // User approval required
	AgentEventApprovalResponded AgentEventType = "agent_approval_responded" // User responded to approval
	
	// Progress events
	AgentEventProgressUpdate AgentEventType = "agent_progress_update" // Progress update
	
	// Legacy/compatibility events
	AgentEventThinking     AgentEventType = "agent_thinking"      // AI is analyzing/planning (legacy)
	AgentEventPlanCreated  AgentEventType = "agent_plan_created"  // Plan generated (legacy)
	AgentEventStepStart    AgentEventType = "agent_step_start"    // Step execution started (legacy)
	AgentEventStepProgress AgentEventType = "agent_step_progress" // Progress update during step (legacy)
	AgentEventStepComplete AgentEventType = "agent_step_complete" // Step completed (legacy)
	AgentEventCompleted    AgentEventType = "agent_completed"     // Session completed (legacy)
	AgentEventFailed       AgentEventType = "agent_failed"        // Session failed (legacy)
	AgentEventCancelled    AgentEventType = "agent_cancelled"     // Session cancelled (legacy)
)

// ========================================
// Request/Response Models
// ========================================

// CreateAgentSessionRequest represents the request to create a new agent session
type CreateAgentSessionRequest struct {
	Task     string                 `json:"task" binding:"required,min=1,max=5000"`
	Model    string                 `json:"model,omitempty"`     // Model to use for planning (optional, defaults to config)
	Context  map[string]interface{} `json:"context,omitempty"`
	MaxSteps *int                   `json:"max_steps,omitempty"` // Max steps to plan (default: 50)
}

// AgentSessionResponse represents the response for agent session
type AgentSessionResponse struct {
	Session *AgentSession `json:"session"`
	Plan    *AgentPlan    `json:"plan,omitempty"`
}

// AgentToolsResponse represents the response for available tools
type AgentToolsResponse struct {
	Tools []AgentTool `json:"tools"`
	Total int         `json:"total"`
}

// ========================================
// Approval System Models (v2.5.0+)
// ========================================

// ApprovalRequest represents a pending approval for dangerous operation
type ApprovalRequest struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	StepNumber  int                    `json:"step_number"`
	Description string                 `json:"description"`
	Action      AgentAction            `json:"action"`
	Tool        string                 `json:"tool"`
	Parameters  map[string]interface{} `json:"parameters"`
	Reason      string                 `json:"reason"`      // Why approval is needed
	Status      ApprovalStatus         `json:"status"`      // pending, approved, rejected
	CreatedAt   time.Time              `json:"created_at"`
	RespondedAt *time.Time             `json:"responded_at,omitempty"`
	Decision    *ApprovalDecision      `json:"decision,omitempty"`
}

// ApprovalStatus represents approval request status
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
	ApprovalStatusExpired  ApprovalStatus = "expired"
)

// ApprovalDecision represents user's approval decision
type ApprovalDecision struct {
	Approved bool   `json:"approved"`
	Comment  string `json:"comment,omitempty"` // Optional comment from user
}

// ApproveStepRequest represents request to approve a step
type ApproveStepRequest struct {
	Approved bool   `json:"approved"`
	Comment  string `json:"comment,omitempty"`
}

// RollbackRequest represents request to rollback a completed step
type RollbackRequest struct {
	SessionID  string `json:"session_id"`
	StepNumber int    `json:"step_number"`
	Reason     string `json:"reason,omitempty"`
}

