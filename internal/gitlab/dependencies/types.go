// Package dependencies provides dependency scanning and analysis functionality.
package dependencies

import (
	"time"

	"aigateway/internal/gitlab/dependencies/security"
)

// Dependency represents a single dependency in a project.
type Dependency struct {
	Name           string `json:"name"`            // e.g., "github.com/gin-gonic/gin"
	CurrentVersion string `json:"current_version"` // e.g., "v1.9.0"
	LatestVersion  string `json:"latest_version"`  // e.g., "v1.10.0"
	Indirect       bool   `json:"indirect"`        // Is this an indirect dependency?
	UpdateType     string `json:"update_type"`     // "major", "minor", "patch", "none"
	HasUpdate      bool   `json:"has_update"`      // Is there an update available?
}

// Vulnerability is an alias for security.Vulnerability.
type Vulnerability = security.Vulnerability

// DependencyWithVulns combines dependency info with vulnerabilities.
type DependencyWithVulns struct {
	Dependency      Dependency      `json:"dependency"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	IsVulnerable    bool            `json:"is_vulnerable"`
}

// ScanResult contains the full result of a dependency scan.
type ScanResult struct {
	ProjectID       string                `json:"project_id"`
	ScanID          string                `json:"scan_id"`
	Language        string                `json:"language"` // "go", "nodejs", "python", etc.
	FilePath        string                `json:"file_path"` // e.g., "go.mod"
	ScannedAt       time.Time             `json:"scanned_at"`
	Duration        string                `json:"duration"`
	Dependencies    []DependencyWithVulns `json:"dependencies"`
	Summary         ScanSummary           `json:"summary"`
	Status          string                `json:"status"` // "completed", "failed"
	Error           string                `json:"error,omitempty"`
}

// ScanSummary provides overview statistics.
type ScanSummary struct {
	TotalDependencies   int            `json:"total_dependencies"`
	DirectDependencies  int            `json:"direct_dependencies"`
	OutdatedCount       int            `json:"outdated_count"`
	VulnerableCount     int            `json:"vulnerable_count"`
	UpToDateCount       int            `json:"up_to_date_count"`
	ByUpdateType        map[string]int `json:"by_update_type"`        // major/minor/patch counts
	BySeverity          map[string]int `json:"by_severity"`           // critical/high/medium/low counts
	CriticalVulns       int            `json:"critical_vulns"`
}

// VersionInfo contains version metadata from a registry.
type VersionInfo struct {
	Version     string    `json:"version"`
	PublishedAt time.Time `json:"published_at"`
	IsLatest    bool      `json:"is_latest"`
}

// PackageInfo contains full package metadata.
type PackageInfo struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Repository  string        `json:"repository"`
	License     string        `json:"license"`
	Versions    []VersionInfo `json:"versions"`
	Latest      string        `json:"latest"`
}

