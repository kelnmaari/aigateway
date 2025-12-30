// Package autodoc provides automatic documentation generation for code.
package autodoc

import "time"

// DocFormat represents the documentation format for different languages.
type DocFormat string

const (
	DocFormatGoDoc    DocFormat = "godoc"     // Go: // comments
	DocFormatJSDoc    DocFormat = "jsdoc"     // JS/TS: /** @param */ 
	DocFormatPyDoc    DocFormat = "pydoc"     // Python: """docstring"""
	DocFormatRustDoc  DocFormat = "rustdoc"   // Rust: /// comments
	DocFormatJavaDoc  DocFormat = "javadoc"   // Java: /** @param */
)

// UndocumentedSymbol represents a code symbol missing documentation.
type UndocumentedSymbol struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"`        // function, method, class, type, interface
	FilePath   string    `json:"file_path"`
	StartLine  int       `json:"start_line"`
	EndLine    int       `json:"end_line"`
	Language   string    `json:"language"`
	Signature  string    `json:"signature"`   // Full function/type signature
	Code       string    `json:"code"`        // Relevant code snippet
	IsExported bool      `json:"is_exported"` // Public/exported symbol
	Importance string    `json:"importance"`  // high, medium, low
}

// GeneratedDoc represents LLM-generated documentation for a symbol.
type GeneratedDoc struct {
	Symbol      UndocumentedSymbol `json:"symbol"`
	Documentation string           `json:"documentation"`  // Generated doc comment
	Format      DocFormat          `json:"format"`
	Language    string             `json:"language"`
	Preview     string             `json:"preview"`        // Code with doc inserted
}

// ScanResult contains the result of scanning for undocumented code.
type ScanResult struct {
	ProjectID     string               `json:"project_id"`
	ScanID        string               `json:"scan_id"`
	ScannedAt     time.Time            `json:"scanned_at"`
	Duration      string               `json:"duration"`
	Status        string               `json:"status"` // "completed", "failed"
	Error         string               `json:"error,omitempty"`
	Symbols       []UndocumentedSymbol `json:"symbols"`
	Summary       ScanSummary          `json:"summary"`
	FilesScanned  int                  `json:"files_scanned"`
}

// ScanSummary provides overview statistics.
type ScanSummary struct {
	TotalSymbols     int            `json:"total_symbols"`
	ExportedCount    int            `json:"exported_count"`
	ByType           map[string]int `json:"by_type"`
	ByLanguage       map[string]int `json:"by_language"`
	ByImportance     map[string]int `json:"by_importance"`
	TopAffectedFiles []FileStats    `json:"top_affected_files"`
}

// FileStats represents undocumented code statistics for a file.
type FileStats struct {
	FilePath  string `json:"file_path"`
	Count     int    `json:"count"`
	Exported  int    `json:"exported"`
}

// GenerationResult contains the result of generating documentation.
type GenerationResult struct {
	ProjectID     string         `json:"project_id"`
	GeneratedAt   time.Time      `json:"generated_at"`
	Duration      string         `json:"duration"`
	Status        string         `json:"status"`
	Error         string         `json:"error,omitempty"`
	Docs          []GeneratedDoc `json:"docs"`
	TokensUsed    int            `json:"tokens_used"`
	ModelID       string         `json:"model_id"`
}

// ScanRequest contains parameters for scanning undocumented code.
type ScanRequest struct {
	ProjectID      string `json:"project_id"`
	CollectionName string `json:"collection_name"`
	MaxFiles       int    `json:"max_files,omitempty"`
	Language       string `json:"language,omitempty"`
	ExportedOnly   bool   `json:"exported_only,omitempty"` // Only scan exported symbols
}

// GenerateRequest contains parameters for generating documentation.
type GenerateRequest struct {
	ProjectID      string   `json:"project_id"`
	CollectionName string   `json:"collection_name"`
	ModelID        string   `json:"model_id"`
	SymbolIDs      []string `json:"symbol_ids,omitempty"` // Specific symbols to document
	MaxSymbols     int      `json:"max_symbols,omitempty"`
	Language       string   `json:"language,omitempty"`
}


// ============================================================================
// Bulk Apply + MR Creation (v4.1.0+)
// ============================================================================

// BulkApplyRequest is the request for bulk applying documentation.
type BulkApplyRequest struct {
	ProjectID       string           `json:"project_id"`
	CollectionName  string           `json:"collection_name"`
	Docs            []GeneratedDoc   `json:"docs"`          // Docs to apply
	CreateMR        bool             `json:"create_mr"`     // Create Merge Request
	MRTitle         string           `json:"mr_title,omitempty"`
	MRDescription   string           `json:"mr_description,omitempty"`
	TargetBranch    string           `json:"target_branch,omitempty"`
	SourceBranch    string           `json:"source_branch,omitempty"`
	Labels          []string         `json:"labels,omitempty"`
}

// BulkApplyResult contains the result of bulk applying documentation.
type BulkApplyResult struct {
	ProjectID        string             `json:"project_id"`
	AppliedAt        time.Time          `json:"applied_at"`
	Duration         string             `json:"duration"`
	Status           string             `json:"status"`          // "completed", "failed"
	Error            string             `json:"error,omitempty"`
	AppliedCount     int                `json:"applied_count"`
	FailedCount      int                `json:"failed_count"`
	AppliedFiles     []AppliedFile      `json:"applied_files"`
	MR               *MergeRequestInfo  `json:"mr,omitempty"`    // Created MR info
}

// AppliedFile represents a file with applied documentation.
type AppliedFile struct {
	FilePath      string   `json:"file_path"`
	SymbolsAdded  int      `json:"symbols_added"`
	LinesAdded    int      `json:"lines_added"`
	Status        string   `json:"status"`  // "success", "failed"
	Error         string   `json:"error,omitempty"`
}

// MergeRequestInfo contains information about created MR.
type MergeRequestInfo struct {
	ID            int64        `json:"id,omitempty"`
	IID           int          `json:"iid,omitempty"`
	URL           string       `json:"url,omitempty"`
	Title         string       `json:"title"`
	Description   string       `json:"description"`
	SourceBranch  string       `json:"source_branch"`
	TargetBranch  string       `json:"target_branch"`
	Labels        []string     `json:"labels"`
	CommitMessage string       `json:"commit_message,omitempty"`
	FileChanges   []FileChange `json:"file_changes,omitempty"`
	Status        string       `json:"status,omitempty"`  // "prepared", "created", "failed"
	Message       string       `json:"message,omitempty"` // Status message
}

// FileChange represents a documentation change to apply to a file.
type FileChange struct {
	FilePath      string `json:"file_path"`
	SymbolName    string `json:"symbol_name"`
	SymbolType    string `json:"symbol_type"`
	LineNumber    int    `json:"line_number"`
	Documentation string `json:"documentation"`
	Language      string `json:"language"`
}
