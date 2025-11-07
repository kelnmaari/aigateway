package models

import "time"

// ToolExecutionRequest represents a request to execute a tool on the client
// This is sent from Server → Client via WebSocket
type ToolExecutionRequest struct {
	// Unique request ID for tracking request-response pairs
	RequestID string `json:"request_id"`
	
	// Tool name (e.g., "file.read", "file.list", "terminal.execute")
	Tool string `json:"tool"`
	
	// Tool parameters (tool-specific structure)
	Parameters map[string]interface{} `json:"parameters"`
	
	// Metadata for UI and logging
	Description string    `json:"description,omitempty"` // Human-readable description
	Timestamp   time.Time `json:"timestamp"`             // Request timestamp
	
	// Security & Approval
	RequiresApproval bool `json:"requires_approval"` // Does this tool require user approval?
	IsDangerous      bool `json:"is_dangerous"`      // Is this a dangerous operation?
}

// ToolExecutionResponse represents a response from tool execution
// This is sent from Client → Server via WebSocket
type ToolExecutionResponse struct {
	// Request ID (matches ToolExecutionRequest.RequestID)
	RequestID string `json:"request_id"`
	
	// Execution result
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"` // Tool-specific result structure
	Error   string      `json:"error,omitempty"`  // Error message if success=false
	
	// Execution metadata
	Duration    int64     `json:"duration,omitempty"`     // Execution duration in milliseconds
	CompletedAt time.Time `json:"completed_at,omitempty"` // Completion timestamp
	
	// Approval status (for dangerous tools)
	WasApproved   bool   `json:"was_approved,omitempty"`   // Was user approval granted?
	ApprovalError string `json:"approval_error,omitempty"` // Error during approval process
}

// ToolApprovalRequest represents a request for user approval for a dangerous tool
// This is displayed in UI before executing the tool
type ToolApprovalRequest struct {
	// Request ID (matches ToolExecutionRequest.RequestID)
	RequestID string `json:"request_id"`
	
	// Tool information
	Tool        string                 `json:"tool"`
	Parameters  map[string]interface{} `json:"parameters"`
	Description string                 `json:"description"`
	
	// Risk information
	RiskLevel   string `json:"risk_level"`   // "low", "medium", "high"
	RiskWarning string `json:"risk_warning"` // Human-readable warning
	
	// Timeout
	Timeout time.Duration `json:"timeout"` // How long to wait for user response
}

// ToolApprovalResponse represents user's approval decision
type ToolApprovalResponse struct {
	// Request ID (matches ToolApprovalRequest.RequestID)
	RequestID string `json:"request_id"`
	
	// User decision
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"` // User-provided reason (optional)
	
	// Timestamp
	RespondedAt time.Time `json:"responded_at"`
}

// WebSocket Message Types for Tool RPC (v2.5.4+)
const (
	// Server → Client: Execute tool on client machine
	WSMessageTypeToolExecutionRequest = "tool_execution_request"
	
	// Client → Server: Tool execution result
	WSMessageTypeToolExecutionResponse = "tool_execution_response"
	
	// Client → User: Request approval for dangerous tool
	WSMessageTypeToolApprovalRequest = "tool_approval_request"
	
	// User → Client: Approval decision
	WSMessageTypeToolApprovalResponse = "tool_approval_response"
	
	// Server → Client: Cancel tool execution (timeout or user cancel)
	WSMessageTypeToolExecutionCancel = "tool_execution_cancel"
)

// ToolExecutionStatus represents the status of a tool execution request
type ToolExecutionStatus string

const (
	ToolExecutionStatusPending        ToolExecutionStatus = "pending"          // Waiting for client to start
	ToolExecutionStatusAwaitingApproval ToolExecutionStatus = "awaiting_approval" // Waiting for user approval
	ToolExecutionStatusExecuting      ToolExecutionStatus = "executing"        // Currently executing
	ToolExecutionStatusCompleted      ToolExecutionStatus = "completed"        // Successfully completed
	ToolExecutionStatusFailed         ToolExecutionStatus = "failed"           // Execution failed
	ToolExecutionStatusCancelled      ToolExecutionStatus = "cancelled"        // Cancelled by user or timeout
	ToolExecutionStatusTimedOut       ToolExecutionStatus = "timed_out"        // Request timed out
)

// FileReadParams represents parameters for file.read tool
type FileReadParams struct {
	Path      string `json:"path"`                 // Relative path from project root
	StartLine *int   `json:"start_line,omitempty"` // Optional: start reading from this line (1-based)
	EndLine   *int   `json:"end_line,omitempty"`   // Optional: end reading at this line (1-based, inclusive)
}

// FileReadResult represents the result of file.read tool
type FileReadResult struct {
	Content   string `json:"content"`             // File content
	Lines     int    `json:"lines"`               // Total number of lines
	Size      int64  `json:"size"`                // File size in bytes
	Path      string `json:"path"`                // Full absolute path
	IsTruncated bool `json:"is_truncated,omitempty"` // Was content truncated?
}

// FileListParams represents parameters for file.list tool
type FileListParams struct {
	Path      string `json:"path"`                // Relative path from project root (e.g., "." for root)
	Recursive bool   `json:"recursive,omitempty"` // List recursively (default: false)
	MaxDepth  int    `json:"max_depth,omitempty"` // Max recursion depth (default: 1)
}

// FileListResult represents the result of file.list tool
type FileListResult struct {
	Files []FileListEntry `json:"files"` // List of files and directories
	Count int             `json:"count"` // Total count
	Path  string          `json:"path"`  // Base path that was listed
}

// FileListEntry represents a single file or directory entry
type FileListEntry struct {
	Name  string `json:"name"`   // File or directory name
	Path  string `json:"path"`   // Relative path from project root
	IsDir bool   `json:"is_dir"` // Is this a directory?
	Size  int64  `json:"size"`   // File size in bytes (0 for directories)
}

// FileWriteParams represents parameters for file.write tool
type FileWriteParams struct {
	Path    string `json:"path"`    // Relative path from project root
	Content string `json:"content"` // Content to write
	Append  bool   `json:"append,omitempty"` // Append to file (default: false = overwrite)
}

// FileWriteResult represents the result of file.write tool
type FileWriteResult struct {
	Path         string `json:"path"`          // Full absolute path
	BytesWritten int    `json:"bytes_written"` // Number of bytes written
	Created      bool   `json:"created"`       // Was the file created (vs updated)?
}

// FileDeleteParams represents parameters for file.delete tool
type FileDeleteParams struct {
	Path      string `json:"path"`                // Relative path from project root
	Recursive bool   `json:"recursive,omitempty"` // Delete directory recursively (default: false)
}

// FileDeleteResult represents the result of file.delete tool
type FileDeleteResult struct {
	Path    string `json:"path"`     // Full absolute path
	Deleted bool   `json:"deleted"`  // Was deletion successful?
	IsDir   bool   `json:"is_dir"`   // Was this a directory?
	Count   int    `json:"count,omitempty"` // Number of items deleted (if recursive)
}

// TerminalExecuteParams represents parameters for terminal.execute tool
type TerminalExecuteParams struct {
	Command     string            `json:"command"`                // Command to execute
	Args        []string          `json:"args,omitempty"`         // Command arguments
	WorkingDir  string            `json:"working_dir,omitempty"`  // Working directory (relative to project root)
	Env         map[string]string `json:"env,omitempty"`          // Environment variables
	Timeout     int               `json:"timeout,omitempty"`      // Timeout in seconds (default: 30)
	CaptureOutput bool            `json:"capture_output,omitempty"` // Capture stdout/stderr (default: true)
}

// TerminalExecuteResult represents the result of terminal.execute tool
type TerminalExecuteResult struct {
	Stdout     string `json:"stdout,omitempty"`      // Standard output
	Stderr     string `json:"stderr,omitempty"`      // Standard error
	ExitCode   int    `json:"exit_code"`             // Process exit code
	Success    bool   `json:"success"`               // Was execution successful (exit_code == 0)?
	Duration   int64  `json:"duration,omitempty"`    // Execution duration in milliseconds
	TimedOut   bool   `json:"timed_out,omitempty"`   // Did execution time out?
	WorkingDir string `json:"working_dir,omitempty"` // Working directory used
}

