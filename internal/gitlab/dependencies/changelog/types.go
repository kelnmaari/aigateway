// Package changelog provides changelog fetching and LLM analysis for dependencies.
package changelog

import "time"

// ChangelogInfo contains fetched changelog data.
type ChangelogInfo struct {
	PackageName    string    `json:"package_name"`
	CurrentVersion string    `json:"current_version"`
	LatestVersion  string    `json:"latest_version"`
	Language       string    `json:"language"`
	ChangelogURL   string    `json:"changelog_url,omitempty"`
	ChangelogText  string    `json:"changelog_text,omitempty"` // Raw changelog content
	ReleaseNotes   string    `json:"release_notes,omitempty"`  // Release notes if available
	FetchedAt      time.Time `json:"fetched_at"`
}

// ChangelogAnalysis contains LLM analysis results.
type ChangelogAnalysis struct {
	PackageName       string           `json:"package_name"`
	CurrentVersion    string           `json:"current_version"`
	LatestVersion     string           `json:"latest_version"`
	Language          string           `json:"language"`
	Summary           string           `json:"summary"`            // Brief summary of changes
	BreakingChanges   []BreakingChange `json:"breaking_changes"`   // List of breaking changes
	NewFeatures       []string         `json:"new_features"`       // List of new features
	BugFixes          []string         `json:"bug_fixes"`          // List of bug fixes
	SecurityFixes     []string         `json:"security_fixes"`     // Security-related fixes
	DeprecatedFeatures []string        `json:"deprecated_features"` // Deprecated features
	MigrationGuide    string           `json:"migration_guide"`    // Migration recommendations
	RiskLevel         string           `json:"risk_level"`         // "low", "medium", "high", "critical"
	Confidence        string           `json:"confidence"`         // LLM confidence: "high", "medium", "low"
	TokensUsed        int              `json:"tokens_used"`
	AnalyzedAt        time.Time        `json:"analyzed_at"`
}

// BreakingChange represents a specific breaking change.
type BreakingChange struct {
	Description  string `json:"description"`
	AffectedArea string `json:"affected_area"` // e.g., "API", "Config", "Behavior"
	Severity     string `json:"severity"`      // "high", "medium", "low"
	Workaround   string `json:"workaround,omitempty"`
}

// AnalyzeRequest contains parameters for changelog analysis.
type AnalyzeRequest struct {
	PackageName    string `json:"package_name"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	Language       string `json:"language"`
	ChangelogText  string `json:"changelog_text"`
	ModelID        string `json:"model_id"`
}

// DependencyWithAnalysis extends dependency info with changelog analysis.
type DependencyWithAnalysis struct {
	Name              string             `json:"name"`
	CurrentVersion    string             `json:"current_version"`
	LatestVersion     string             `json:"latest_version"`
	UpdateType        string             `json:"update_type"`
	HasUpdate         bool               `json:"has_update"`
	ChangelogAnalysis *ChangelogAnalysis `json:"changelog_analysis,omitempty"`
}

