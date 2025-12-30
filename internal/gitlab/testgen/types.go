// Package testgen provides automatic test generation functionality.
package testgen

import "time"

// TestFramework represents the testing framework to use.
type TestFramework string

const (
	FrameworkGoTest   TestFramework = "go_test"    // Go: testing + testify
	FrameworkJest     TestFramework = "jest"       // JS/TS: Jest
	FrameworkVitest   TestFramework = "vitest"     // JS/TS: Vitest
	FrameworkPytest   TestFramework = "pytest"     // Python: pytest
	FrameworkRust     TestFramework = "cargo_test" // Rust: cargo test
)

// TestableFunction represents a function that can have tests generated.
type TestableFunction struct {
	Name        string   `json:"name"`
	FilePath    string   `json:"file_path"`
	StartLine   int      `json:"start_line"`
	EndLine     int      `json:"end_line"`
	Language    string   `json:"language"`
	Signature   string   `json:"signature"`
	Code        string   `json:"code"`
	Parameters  []string `json:"parameters,omitempty"`
	ReturnType  string   `json:"return_type,omitempty"`
	IsExported  bool     `json:"is_exported"`
	HasTests    bool     `json:"has_tests"`    // Already has tests
	Complexity  string   `json:"complexity"`   // low, medium, high
}

// GeneratedTest represents a generated test.
type GeneratedTest struct {
	Function    TestableFunction `json:"function"`
	TestCode    string           `json:"test_code"`
	TestName    string           `json:"test_name"`
	Framework   TestFramework    `json:"framework"`
	Language    string           `json:"language"`
	Description string           `json:"description"` // What the test covers
}

// ScanResult contains the result of scanning for testable functions.
type ScanResult struct {
	ProjectID    string             `json:"project_id"`
	ScanID       string             `json:"scan_id"`
	ScannedAt    time.Time          `json:"scanned_at"`
	Duration     string             `json:"duration"`
	Status       string             `json:"status"`
	Error        string             `json:"error,omitempty"`
	Functions    []TestableFunction `json:"functions"`
	Summary      ScanSummary        `json:"summary"`
	FilesScanned int                `json:"files_scanned"`
}

// ScanSummary provides overview statistics.
type ScanSummary struct {
	TotalFunctions   int            `json:"total_functions"`
	WithoutTests     int            `json:"without_tests"`
	WithTests        int            `json:"with_tests"`
	ByLanguage       map[string]int `json:"by_language"`
	ByComplexity     map[string]int `json:"by_complexity"`
	TopFiles         []FileStats    `json:"top_files"`
}

// FileStats represents test statistics for a file.
type FileStats struct {
	FilePath      string `json:"file_path"`
	FunctionCount int    `json:"function_count"`
	Untested      int    `json:"untested"`
}

// GenerationResult contains the result of test generation.
type GenerationResult struct {
	ProjectID    string          `json:"project_id"`
	GeneratedAt  time.Time       `json:"generated_at"`
	Duration     string          `json:"duration"`
	Status       string          `json:"status"`
	Error        string          `json:"error,omitempty"`
	Tests        []GeneratedTest `json:"tests"`
	TokensUsed   int             `json:"tokens_used"`
	ModelID      string          `json:"model_id"`
}

// ScanRequest contains parameters for scanning testable functions.
type ScanRequest struct {
	ProjectID      string `json:"project_id"`
	CollectionName string `json:"collection_name"`
	MaxFiles       int    `json:"max_files,omitempty"`
	Language       string `json:"language,omitempty"`
}

// GenerateRequest contains parameters for generating tests.
type GenerateRequest struct {
	ProjectID      string        `json:"project_id"`
	CollectionName string        `json:"collection_name"`
	ModelID        string        `json:"model_id"`
	Framework      TestFramework `json:"framework,omitempty"`
	MaxFunctions   int           `json:"max_functions,omitempty"`
	Language       string        `json:"language,omitempty"`
}

