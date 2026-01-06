// Package parser provides parsers for various dependency file formats.
package parser

import (
	"encoding/xml"
	"strings"
)

// MavenParser parses Maven POM files (pom.xml).
type MavenParser struct{}

// NewMavenParser creates a new Maven parser.
func NewMavenParser() *MavenParser {
	return &MavenParser{}
}

// pomProject represents the root pom.xml structure.
type pomProject struct {
	XMLName              xml.Name             `xml:"project"`
	GroupID              string               `xml:"groupId"`
	ArtifactID           string               `xml:"artifactId"`
	Version              string               `xml:"version"`
	Parent               *pomParent           `xml:"parent"`
	Properties           pomProperties        `xml:"properties"`
	Dependencies         pomDependencies      `xml:"dependencies"`
	DependencyManagement *pomDepManagement    `xml:"dependencyManagement"`
	Build                *pomBuild            `xml:"build"`
}

// pomParent represents the parent POM reference.
type pomParent struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

// pomProperties represents Maven properties (for variable substitution).
type pomProperties struct {
	Properties []pomProperty `xml:",any"`
}

// pomProperty represents a single property.
type pomProperty struct {
	XMLName xml.Name
	Value   string `xml:",chardata"`
}

// pomDependencies represents the dependencies block.
type pomDependencies struct {
	Dependencies []pomDependency `xml:"dependency"`
}

// pomDependency represents a single dependency.
type pomDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Type       string `xml:"type"`
	Optional   string `xml:"optional"`
}

// pomDepManagement represents dependencyManagement block.
type pomDepManagement struct {
	Dependencies pomDependencies `xml:"dependencies"`
}

// pomBuild represents the build section.
type pomBuild struct {
	Plugins pomPlugins `xml:"plugins"`
}

// pomPlugins represents plugins block.
type pomPlugins struct {
	Plugins []pomPlugin `xml:"plugin"`
}

// pomPlugin represents a build plugin.
type pomPlugin struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

// Parse parses pom.xml content and returns dependencies.
func (p *MavenParser) Parse(content string) ([]Dependency, error) {
	var project pomProject
	if err := xml.Unmarshal([]byte(content), &project); err != nil {
		return nil, err
	}

	// Build properties map for variable substitution
	props := make(map[string]string)
	for _, prop := range project.Properties.Properties {
		props[prop.XMLName.Local] = prop.Value
	}
	
	// Add common implicit properties
	if project.Version != "" {
		props["project.version"] = project.Version
	}
	if project.Parent != nil {
		props["parent.version"] = project.Parent.Version
	}

	var deps []Dependency

	// Parse regular dependencies
	for _, d := range project.Dependencies.Dependencies {
		dep := p.convertDependency(d, props)
		if dep.Name != "" && dep.CurrentVersion != "" {
			deps = append(deps, dep)
		}
	}

	// Parse dependencyManagement (these define versions for dependencies)
	if project.DependencyManagement != nil {
		for _, d := range project.DependencyManagement.Dependencies.Dependencies {
			dep := p.convertDependency(d, props)
			dep.Indirect = true // Management deps are indirect
			if dep.Name != "" && dep.CurrentVersion != "" {
				deps = append(deps, dep)
			}
		}
	}

	return deps, nil
}

// convertDependency converts a pomDependency to Dependency.
func (p *MavenParser) convertDependency(d pomDependency, props map[string]string) Dependency {
	groupID := p.resolveProperty(d.GroupID, props)
	artifactID := p.resolveProperty(d.ArtifactID, props)
	version := p.resolveProperty(d.Version, props)
	scope := strings.ToLower(d.Scope)

	// Skip if version is empty or still contains unresolved variable
	if version == "" || strings.Contains(version, "${") {
		return Dependency{}
	}

	// Format: groupId:artifactId (Maven coordinate)
	name := groupID + ":" + artifactID

	// Determine if indirect (test, provided, runtime scopes)
	indirect := scope == "test" || scope == "provided" || scope == "runtime" || d.Optional == "true"

	return Dependency{
		Name:           name,
		CurrentVersion: version,
		Indirect:       indirect,
	}
}

// resolveProperty substitutes ${property} placeholders with values from props map.
func (p *MavenParser) resolveProperty(value string, props map[string]string) string {
	if !strings.Contains(value, "${") {
		return value
	}

	result := value
	for key, val := range props {
		placeholder := "${" + key + "}"
		result = strings.ReplaceAll(result, placeholder, val)
	}
	return result
}

// CanParse checks if this parser can handle the given file.
func (p *MavenParser) CanParse(filename string) bool {
	return filename == "pom.xml" || strings.HasSuffix(filename, "/pom.xml")
}

// Language returns the language identifier.
func (p *MavenParser) Language() string {
	return "java"
}

