// Package parser provides parsers for various dependency file formats.
package parser

import (
	"encoding/json"
	"strings"
)

// PackageJSON represents a package.json file structure
type PackageJSON struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// NPMParser parses npm package.json files.
type NPMParser struct{}

// NewNPMParser creates a new npm parser.
func NewNPMParser() *NPMParser {
	return &NPMParser{}
}

// Parse parses package.json content and returns dependencies.
func (p *NPMParser) Parse(content string) ([]Dependency, error) {
	var pkg PackageJSON
	if err := json.Unmarshal([]byte(content), &pkg); err != nil {
		return nil, err
	}

	var deps []Dependency

	// Production dependencies
	for name, version := range pkg.Dependencies {
		deps = append(deps, Dependency{
			Name:           name,
			CurrentVersion: cleanNPMVersion(version),
			Indirect:       false,
		})
	}

	// Dev dependencies (marked as indirect)
	for name, version := range pkg.DevDependencies {
		deps = append(deps, Dependency{
			Name:           name,
			CurrentVersion: cleanNPMVersion(version),
			Indirect:       true, // devDependencies as indirect
		})
	}

	return deps, nil
}

// CanParse checks if this parser can handle the given file.
func (p *NPMParser) CanParse(filename string) bool {
	return filename == "package.json" || strings.HasSuffix(filename, "/package.json")
}

// Language returns the language identifier.
func (p *NPMParser) Language() string {
	return "nodejs"
}

// cleanNPMVersion removes version range prefixes like ^, ~, >=, etc.
func cleanNPMVersion(version string) string {
	// Remove common prefixes
	version = strings.TrimPrefix(version, "^")
	version = strings.TrimPrefix(version, "~")
	version = strings.TrimPrefix(version, ">=")
	version = strings.TrimPrefix(version, ">")
	version = strings.TrimPrefix(version, "<=")
	version = strings.TrimPrefix(version, "<")
	version = strings.TrimPrefix(version, "=")
	version = strings.TrimSpace(version)
	
	// Handle version ranges like "1.0.0 - 2.0.0" - take the first part
	if idx := strings.Index(version, " "); idx > 0 {
		version = version[:idx]
	}
	
	return version
}

