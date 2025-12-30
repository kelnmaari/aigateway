// Package parser provides parsers for various dependency file formats.
package parser

import (
	"bufio"
	"regexp"
	"strings"
)

// PipParser parses Python requirements.txt files.
type PipParser struct{}

// NewPipParser creates a new pip parser.
func NewPipParser() *PipParser {
	return &PipParser{}
}

// Parse parses requirements.txt content and returns dependencies.
func (p *PipParser) Parse(content string) ([]Dependency, error) {
	var deps []Dependency

	scanner := bufio.NewScanner(strings.NewReader(content))

	// Regex patterns for requirements.txt
	// package==1.0.0, package>=1.0.0, package~=1.0.0, etc.
	versionRegex := regexp.MustCompile(`^([a-zA-Z0-9_-]+)\s*(?:(\[.*\])?\s*)(?:(==|>=|<=|~=|!=|>|<)\s*([0-9a-zA-Z._-]+))?`)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip options like -r, -e, --hash, etc.
		if strings.HasPrefix(line, "-") {
			continue
		}

		// Skip URLs and git dependencies
		if strings.Contains(line, "://") || strings.Contains(line, "git+") {
			continue
		}

		matches := versionRegex.FindStringSubmatch(line)
		if len(matches) >= 2 {
			name := matches[1]
			version := ""
			if len(matches) >= 5 && matches[4] != "" {
				version = matches[4]
			}

			deps = append(deps, Dependency{
				Name:           name,
				CurrentVersion: version,
				Indirect:       false,
			})
		}
	}

	return deps, scanner.Err()
}

// CanParse checks if this parser can handle the given file.
func (p *PipParser) CanParse(filename string) bool {
	return filename == "requirements.txt" || 
		strings.HasSuffix(filename, "/requirements.txt") ||
		strings.HasSuffix(filename, "-requirements.txt") ||
		strings.Contains(filename, "requirements") && strings.HasSuffix(filename, ".txt")
}

// Language returns the language identifier.
func (p *PipParser) Language() string {
	return "python"
}

