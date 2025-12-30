// Package parser provides parsers for various dependency file formats.
package parser

import (
	"bufio"
	"regexp"
	"strings"
)

// Dependency represents a single dependency parsed from a file.
type Dependency struct {
	Name           string `json:"name"`
	CurrentVersion string `json:"current_version"`
	Indirect       bool   `json:"indirect"`
}

// GoModParser parses Go module files (go.mod).
type GoModParser struct{}

// NewGoModParser creates a new Go mod parser.
func NewGoModParser() *GoModParser {
	return &GoModParser{}
}

// Parse parses go.mod content and returns dependencies.
func (p *GoModParser) Parse(content string) ([]Dependency, error) {
	var deps []Dependency
	
	scanner := bufio.NewScanner(strings.NewReader(content))
	inRequireBlock := false
	
	// Regex for require line: module/path version [// indirect]
	requireLineRegex := regexp.MustCompile(`^\s*([^\s]+)\s+(v[^\s]+)(\s*//\s*indirect)?`)
	// Regex for single require: require module/path version
	singleRequireRegex := regexp.MustCompile(`^require\s+([^\s]+)\s+(v[^\s]+)(\s*//\s*indirect)?`)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		
		// Handle require block start
		if strings.HasPrefix(line, "require (") || line == "require (" {
			inRequireBlock = true
			continue
		}
		
		// Handle require block end
		if line == ")" && inRequireBlock {
			inRequireBlock = false
			continue
		}
		
		// Parse inside require block
		if inRequireBlock {
			if matches := requireLineRegex.FindStringSubmatch(line); len(matches) >= 3 {
				dep := Dependency{
					Name:           matches[1],
					CurrentVersion: matches[2],
					Indirect:       len(matches) > 3 && matches[3] != "",
				}
				deps = append(deps, dep)
			}
			continue
		}
		
		// Parse single-line require
		if matches := singleRequireRegex.FindStringSubmatch(line); len(matches) >= 3 {
			dep := Dependency{
				Name:           matches[1],
				CurrentVersion: matches[2],
				Indirect:       len(matches) > 3 && matches[3] != "",
			}
			deps = append(deps, dep)
		}
	}
	
	return deps, scanner.Err()
}

// CanParse checks if this parser can handle the given file.
func (p *GoModParser) CanParse(filename string) bool {
	return filename == "go.mod" || strings.HasSuffix(filename, "/go.mod")
}

// Language returns the language identifier.
func (p *GoModParser) Language() string {
	return "go"
}

