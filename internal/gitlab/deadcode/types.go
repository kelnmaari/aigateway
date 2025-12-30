// Package deadcode provides dead code detection functionality.
package deadcode

import "time"

// SymbolType represents the type of code symbol.
type SymbolType string

const (
	SymbolFunction  SymbolType = "function"
	SymbolType_     SymbolType = "type"
	SymbolVariable  SymbolType = "variable"
	SymbolConstant  SymbolType = "constant"
	SymbolInterface SymbolType = "interface"
	SymbolClass     SymbolType = "class"
	SymbolMethod    SymbolType = "method"
)

// Confidence represents the detection confidence level.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// DeadSymbol represents a potentially unused code symbol.
type DeadSymbol struct {
	Name        string     `json:"name"`
	Type        SymbolType `json:"type"`
	FilePath    string     `json:"file_path"`
	StartLine   int        `json:"start_line"`
	EndLine     int        `json:"end_line"`
	Confidence  Confidence `json:"confidence"`
	Reason      string     `json:"reason"`
	LinesOfCode int        `json:"lines_of_code"`
	Exportable  bool       `json:"exportable"` // Is this symbol exported/public?
}

// ScanResult contains the result of a dead code scan.
type ScanResult struct {
	ProjectID    string         `json:"project_id"`
	ScanID       string         `json:"scan_id"`
	ScannedAt    time.Time      `json:"scanned_at"`
	Duration     string         `json:"duration"`
	Status       string         `json:"status"` // "completed", "failed"
	Error        string         `json:"error,omitempty"`
	DeadSymbols  []DeadSymbol   `json:"dead_symbols"`
	Summary      ScanSummary    `json:"summary"`
	ModelID      string         `json:"model_id"`
	TokensUsed   int            `json:"tokens_used"`
	FilesScanned int            `json:"files_scanned"`
	ChunksScanned int           `json:"chunks_scanned"`
}

// ScanSummary provides overview statistics.
type ScanSummary struct {
	TotalDeadSymbols   int            `json:"total_dead_symbols"`
	ByType             map[SymbolType]int `json:"by_type"`
	ByConfidence       map[Confidence]int `json:"by_confidence"`
	EstimatedDeadLines int            `json:"estimated_dead_lines"`
	TopAffectedFiles   []FileStats    `json:"top_affected_files"`
}

// FileStats represents statistics for a file.
type FileStats struct {
	FilePath     string `json:"file_path"`
	DeadSymbols  int    `json:"dead_symbols"`
	DeadLines    int    `json:"dead_lines"`
}

// ScanRequest contains parameters for dead code detection.
type ScanRequest struct {
	ProjectID            string `json:"project_id"`
	CollectionName       string `json:"collection_name"`
	ModelID              string `json:"model_id"`
	MaxChunks            int    `json:"max_chunks,omitempty"`
	Language             string `json:"language,omitempty"`
	DetectUnreachable    bool   `json:"detect_unreachable,omitempty"`    // Detect unreachable code
	DetectCommentedCode  bool   `json:"detect_commented_code,omitempty"` // Detect commented-out code
}

// ============================================================================
// Unreachable Code Detection (v4.1.0+)
// ============================================================================

// UnreachableCodeBlock represents a block of unreachable code.
type UnreachableCodeBlock struct {
	FilePath    string `json:"file_path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	CodeSnippet string `json:"code_snippet"`
	Reason      string `json:"reason"`      // Why it's unreachable
	Confidence  Confidence `json:"confidence"`
	Context     string `json:"context"`     // Surrounding code context
}

// UnreachableCodeResult contains the result of unreachable code detection.
type UnreachableCodeResult struct {
	ProjectID    string                  `json:"project_id"`
	ScanID       string                  `json:"scan_id"`
	ScannedAt    string                  `json:"scanned_at"`
	Duration     string                  `json:"duration"`
	Status       string                  `json:"status"`
	Error        string                  `json:"error,omitempty"`
	Blocks       []UnreachableCodeBlock  `json:"blocks"`
	Summary      UnreachableSummary      `json:"summary"`
	ModelID      string                  `json:"model_id"`
	TokensUsed   int                     `json:"tokens_used"`
}

// UnreachableSummary provides overview of unreachable code.
type UnreachableSummary struct {
	TotalBlocks      int      `json:"total_blocks"`
	TotalLines       int      `json:"total_lines"`
	AffectedFiles    int      `json:"affected_files"`
	ByConfidence     map[Confidence]int `json:"by_confidence"`
	TopAffectedFiles []string `json:"top_affected_files"`
}

// ============================================================================
// Commented-Out Code Detection (v4.1.0+)
// ============================================================================

// CommentedCodeBlock represents a block of commented-out code.
type CommentedCodeBlock struct {
	FilePath    string     `json:"file_path"`
	StartLine   int        `json:"start_line"`
	EndLine     int        `json:"end_line"`
	Content     string     `json:"content"`     // The commented code
	LinesCount  int        `json:"lines_count"`
	Confidence  Confidence `json:"confidence"`
	Reason      string     `json:"reason"`      // Why it looks like code
}

// CommentedCodeResult contains the result of commented-out code detection.
type CommentedCodeResult struct {
	ProjectID    string                `json:"project_id"`
	ScanID       string                `json:"scan_id"`
	ScannedAt    string                `json:"scanned_at"`
	Duration     string                `json:"duration"`
	Status       string                `json:"status"`
	Error        string                `json:"error,omitempty"`
	Blocks       []CommentedCodeBlock  `json:"blocks"`
	Summary      CommentedCodeSummary  `json:"summary"`
	ModelID      string                `json:"model_id"`
	TokensUsed   int                   `json:"tokens_used"`
}

// CommentedCodeSummary provides overview of commented-out code.
type CommentedCodeSummary struct {
	TotalBlocks      int      `json:"total_blocks"`
	TotalLines       int      `json:"total_lines"`
	AffectedFiles    int      `json:"affected_files"`
	TopAffectedFiles []string `json:"top_affected_files"`
}

// ============================================================================
// GitLab Issue Export (v4.1.0+)
// ============================================================================

// CreateDeadCodeIssueRequest is the request to create a GitLab issue.
type CreateDeadCodeIssueRequest struct {
	IntegrationID string       `json:"integration_id" binding:"required"`
	ProjectID     string       `json:"project_id" binding:"required"`
	SymbolIDs     []string     `json:"symbol_ids"` // Selected symbols to include
	Title         string       `json:"title,omitempty"`
	Priority      string       `json:"priority,omitempty"` // high, medium, low
	Labels        []string     `json:"labels,omitempty"`
	Assignee      string       `json:"assignee,omitempty"`
	IncludeAll    bool         `json:"include_all,omitempty"` // Include all dead code
}

// DeadCodeIssueResult is the result of creating a GitLab issue.
type DeadCodeIssueResult struct {
	IssueID     int64    `json:"issue_id"`
	IssueIID    int      `json:"issue_iid"`
	IssueURL    string   `json:"issue_url"`
	Title       string   `json:"title"`
	SymbolCount int      `json:"symbol_count"`
	Labels      []string `json:"labels"`
}

