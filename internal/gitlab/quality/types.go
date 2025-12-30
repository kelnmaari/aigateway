// Package quality provides code quality analysis functionality.
package quality

import "time"

// ScoreCategory represents a category of code quality.
type ScoreCategory string

const (
	CategoryComplexity      ScoreCategory = "complexity"
	CategoryDocumentation   ScoreCategory = "documentation"
	CategoryTesting         ScoreCategory = "testing"
	CategorySecurity        ScoreCategory = "security"
	CategoryMaintainability ScoreCategory = "maintainability"
	CategoryNaming          ScoreCategory = "naming"
	CategoryErrorHandling   ScoreCategory = "error_handling"
)

// QualityScore represents the overall quality score with breakdown.
type QualityScore struct {
	ProjectID       string                   `json:"project_id"`
	ScanID          string                   `json:"scan_id"`
	ScannedAt       time.Time                `json:"scanned_at"`
	Duration        string                   `json:"duration"`
	OverallScore    int                      `json:"overall_score"`    // 0-100
	Breakdown       map[ScoreCategory]int    `json:"breakdown"`        // Category -> 0-100
	FileScores      []FileScore              `json:"file_scores"`      // Per-file breakdown
	Recommendations []Recommendation         `json:"recommendations"`
	Summary         QualitySummary           `json:"summary"`
	Status          string                   `json:"status"` // "completed", "failed"
	Error           string                   `json:"error,omitempty"`
	ModelID         string                   `json:"model_id"`
	TokensUsed      int                      `json:"tokens_used"`
}

// FileScore represents quality score for a single file.
type FileScore struct {
	FilePath                string                `json:"file_path"`
	Language                string                `json:"language"`
	LinesOfCode             int                   `json:"lines_of_code"`
	Score                   int                   `json:"score"`
	Breakdown               map[ScoreCategory]int `json:"breakdown"`
	Issues                  []QualityIssue        `json:"issues"`
	FunctionsCount          int                   `json:"functions_count"`
	IsTestFile              bool                  `json:"is_test_file"`              // Whether this is a test file
	TestableFunctions       int                   `json:"testable_functions"`        // Number of testable functions
	TestedFunctionsEstimate int                   `json:"tested_functions_estimate"` // Estimated tested functions
}

// QualityIssue represents a specific quality issue found.
type QualityIssue struct {
	Category    ScoreCategory `json:"category"`
	Severity    string        `json:"severity"` // "high", "medium", "low"
	FilePath    string        `json:"file_path"`
	Line        int           `json:"line,omitempty"`
	Message     string        `json:"message"`
	Suggestion  string        `json:"suggestion,omitempty"`
	CodeSnippet string        `json:"code_snippet,omitempty"`
}

// Recommendation represents an actionable improvement suggestion.
type Recommendation struct {
	Category    ScoreCategory `json:"category"`
	Priority    string        `json:"priority"` // "high", "medium", "low"
	Title       string        `json:"title"`
	Description string        `json:"description"`
	FilePaths   []string      `json:"file_paths,omitempty"`
	Impact      string        `json:"impact"` // Expected score improvement
}

// QualitySummary provides overview statistics.
type QualitySummary struct {
	TotalFiles           int            `json:"total_files"`
	TotalLinesOfCode     int            `json:"total_lines_of_code"`
	TotalFunctions       int            `json:"total_functions"`
	IssuesCount          int            `json:"issues_count"`
	HighSeverityCount    int            `json:"high_severity_count"`
	MediumSeverityCount  int            `json:"medium_severity_count"`
	LowSeverityCount     int            `json:"low_severity_count"`
	TopIssueCategories   []CategoryStat `json:"top_issue_categories"`
	BestScoringFiles     []string       `json:"best_scoring_files"`
	WorstScoringFiles    []string       `json:"worst_scoring_files"`
	// Test Coverage Estimation (v4.1.0+)
	TestFilesCount            int     `json:"test_files_count"`
	SourceFilesCount          int     `json:"source_files_count"`
	TotalTestableFunctions    int     `json:"total_testable_functions"`
	EstimatedTestedFunctions  int     `json:"estimated_tested_functions"`
	EstimatedCoveragePercent  float64 `json:"estimated_coverage_percent"`
}

// CategoryStat represents statistics for a category.
type CategoryStat struct {
	Category ScoreCategory `json:"category"`
	Count    int           `json:"count"`
	AvgScore int           `json:"avg_score"`
}

// AnalysisRequest contains parameters for quality analysis.
type AnalysisRequest struct {
	ProjectID         string `json:"project_id"`
	CollectionName    string `json:"collection_name"`
	ModelID           string `json:"model_id"`
	MaxFiles          int    `json:"max_files,omitempty"`         // Limit files to analyze (default: 50)
	Language          string `json:"language,omitempty"`          // Filter by language
	DetectDuplication bool   `json:"detect_duplication,omitempty"` // Enable code duplication detection
}

// ============================================================================
// Code Duplication Detection Types (v4.1.0+)
// ============================================================================

// DuplicationResult contains the result of duplication analysis.
type DuplicationResult struct {
	ProjectID      string                `json:"project_id"`
	ScanID         string                `json:"scan_id"`
	ScannedAt      string                `json:"scanned_at"`
	Duration       string                `json:"duration"`
	Status         string                `json:"status"` // "completed", "failed"
	Error          string                `json:"error,omitempty"`
	Duplicates     []DuplicateGroup      `json:"duplicates"`
	Summary        DuplicationSummary    `json:"summary"`
	ModelID        string                `json:"model_id"`
	TokensUsed     int                   `json:"tokens_used"`
}

// DuplicateGroup represents a group of duplicate code blocks.
type DuplicateGroup struct {
	ID            string           `json:"id"`
	Occurrences   []DuplicateBlock `json:"occurrences"`
	LinesCount    int              `json:"lines_count"`    // Lines in each duplicate
	Similarity    float64          `json:"similarity"`     // 0.0 - 1.0
	Description   string           `json:"description"`    // What the duplicate code does
	Suggestion    string           `json:"suggestion"`     // How to refactor
	RefactorType  string           `json:"refactor_type"`  // "extract_function", "extract_class", "template", etc.
}

// DuplicateBlock represents a single occurrence of duplicate code.
type DuplicateBlock struct {
	FilePath    string `json:"file_path"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	CodeSnippet string `json:"code_snippet"`
}

// DuplicationSummary provides overview of duplication analysis.
type DuplicationSummary struct {
	TotalFilesAnalyzed   int     `json:"total_files_analyzed"`
	FilesWithDuplicates  int     `json:"files_with_duplicates"`
	TotalDuplicateGroups int     `json:"total_duplicate_groups"`
	TotalDuplicateLines  int     `json:"total_duplicate_lines"`
	DuplicationPercent   float64 `json:"duplication_percent"`
	TopDuplicatedFiles   []string `json:"top_duplicated_files"`
}

