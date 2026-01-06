// Package parser provides parsers for various dependency file formats.
package parser

import (
	"regexp"
	"strings"
)

// GradleParser parses Gradle build files (build.gradle, build.gradle.kts).
type GradleParser struct {
	// Regex patterns for different dependency declaration formats
	
	// Groovy DSL: implementation 'group:artifact:version'
	groovyStringPattern *regexp.Regexp
	// Groovy DSL: implementation "group:artifact:version"
	groovyDoublePattern *regexp.Regexp
	// Groovy DSL: implementation group: 'x', name: 'y', version: 'z'
	groovyMapPattern *regexp.Regexp
	
	// Kotlin DSL: implementation("group:artifact:version")
	kotlinPattern *regexp.Regexp
	// Kotlin DSL with named parameters
	kotlinMapPattern *regexp.Regexp
	
	// Version catalog: implementation(libs.some.library)
	catalogPattern *regexp.Regexp
	
	// Variable pattern: $variable or ${variable}
	variablePattern *regexp.Regexp
}

// NewGradleParser creates a new Gradle parser.
func NewGradleParser() *GradleParser {
	return &GradleParser{
		// Groovy: implementation 'com.google.code.gson:gson:2.8.9'
		groovyStringPattern: regexp.MustCompile(`(?m)^\s*(implementation|api|compileOnly|runtimeOnly|testImplementation|testCompileOnly|testRuntimeOnly|annotationProcessor|kapt|ksp)\s*['"]([^:'"]+):([^:'"]+):([^'"]+)['"]`),
		
		// Groovy double quotes (may contain variables)
		groovyDoublePattern: regexp.MustCompile(`(?m)^\s*(implementation|api|compileOnly|runtimeOnly|testImplementation|testCompileOnly|testRuntimeOnly|annotationProcessor|kapt|ksp)\s*"([^:"]+):([^:"]+):([^"]+)"`),
		
		// Groovy map syntax: implementation group: 'x', name: 'y', version: 'z'
		groovyMapPattern: regexp.MustCompile(`(?m)^\s*(implementation|api|compileOnly|runtimeOnly|testImplementation|testCompileOnly|testRuntimeOnly)\s+group:\s*['"]([^'"]+)['"],\s*name:\s*['"]([^'"]+)['"],\s*version:\s*['"]([^'"]+)['"]`),
		
		// Kotlin DSL: implementation("com.google.code.gson:gson:2.8.9")
		kotlinPattern: regexp.MustCompile(`(?m)^\s*(implementation|api|compileOnly|runtimeOnly|testImplementation|testCompileOnly|testRuntimeOnly|annotationProcessor|kapt|ksp)\s*\(\s*["']([^:"']+):([^:"']+):([^"']+)["']\s*\)`),
		
		// Kotlin DSL map: implementation(group = "x", name = "y", version = "z")
		kotlinMapPattern: regexp.MustCompile(`(?m)^\s*(implementation|api|compileOnly|runtimeOnly|testImplementation|testCompileOnly|testRuntimeOnly)\s*\(\s*group\s*=\s*["']([^"']+)["'],\s*name\s*=\s*["']([^"']+)["'],\s*version\s*=\s*["']([^"']+)["']\s*\)`),
		
		// Version catalog: implementation(libs.gson)
		catalogPattern: regexp.MustCompile(`(?m)^\s*(implementation|api|compileOnly|runtimeOnly|testImplementation|testCompileOnly|testRuntimeOnly)\s*\(\s*(libs\.[a-zA-Z0-9._]+)\s*\)`),
		
		// Variable pattern
		variablePattern: regexp.MustCompile(`\$\{?[a-zA-Z_][a-zA-Z0-9_]*\}?`),
	}
}

// Parse parses build.gradle/build.gradle.kts content and returns dependencies.
func (p *GradleParser) Parse(content string) ([]Dependency, error) {
	var deps []Dependency
	seen := make(map[string]bool) // Dedupe

	// Extract variables/properties from ext block
	vars := p.extractVariables(content)

	// Parse all dependency patterns
	deps = append(deps, p.parseWithPattern(content, p.groovyStringPattern, vars, seen)...)
	deps = append(deps, p.parseWithPattern(content, p.groovyDoublePattern, vars, seen)...)
	deps = append(deps, p.parseMapPattern(content, p.groovyMapPattern, vars, seen)...)
	deps = append(deps, p.parseWithPattern(content, p.kotlinPattern, vars, seen)...)
	deps = append(deps, p.parseMapPattern(content, p.kotlinMapPattern, vars, seen)...)

	return deps, nil
}

// parseWithPattern extracts dependencies using a regex pattern.
// Pattern groups: 1=configuration, 2=group, 3=artifact, 4=version
func (p *GradleParser) parseWithPattern(content string, pattern *regexp.Regexp, vars map[string]string, seen map[string]bool) []Dependency {
	var deps []Dependency
	
	matches := pattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) < 5 {
			continue
		}
		
		config := match[1]
		group := match[2]
		artifact := match[3]
		version := p.resolveVariable(match[4], vars)
		
		// Skip if version still contains unresolved variable
		if p.variablePattern.MatchString(version) {
			continue
		}
		
		name := group + ":" + artifact
		if seen[name] {
			continue
		}
		seen[name] = true
		
		deps = append(deps, Dependency{
			Name:           name,
			CurrentVersion: version,
			Indirect:       p.isTestScope(config),
		})
	}
	
	return deps
}

// parseMapPattern extracts dependencies using map-style pattern.
// Pattern groups: 1=configuration, 2=group, 3=name, 4=version
func (p *GradleParser) parseMapPattern(content string, pattern *regexp.Regexp, vars map[string]string, seen map[string]bool) []Dependency {
	var deps []Dependency
	
	matches := pattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) < 5 {
			continue
		}
		
		config := match[1]
		group := match[2]
		artifact := match[3]
		version := p.resolveVariable(match[4], vars)
		
		if p.variablePattern.MatchString(version) {
			continue
		}
		
		name := group + ":" + artifact
		if seen[name] {
			continue
		}
		seen[name] = true
		
		deps = append(deps, Dependency{
			Name:           name,
			CurrentVersion: version,
			Indirect:       p.isTestScope(config),
		})
	}
	
	return deps
}

// extractVariables extracts variables from ext block and buildscript.
func (p *GradleParser) extractVariables(content string) map[string]string {
	vars := make(map[string]string)
	
	// Pattern for: def varName = "value" or val varName = "value"
	defPattern := regexp.MustCompile(`(?m)^\s*(?:def|val|var)\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*["']([^"']+)["']`)
	
	// Pattern for ext { varName = "value" }
	extPattern := regexp.MustCompile(`(?m)^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*["']([^"']+)["']`)
	
	// Pattern for: extra["varName"] = "value"
	extraPattern := regexp.MustCompile(`(?m)extra\s*\[\s*["']([^"']+)["']\s*\]\s*=\s*["']([^"']+)["']`)
	
	for _, match := range defPattern.FindAllStringSubmatch(content, -1) {
		if len(match) >= 3 {
			vars[match[1]] = match[2]
		}
	}
	
	for _, match := range extPattern.FindAllStringSubmatch(content, -1) {
		if len(match) >= 3 {
			vars[match[1]] = match[2]
		}
	}
	
	for _, match := range extraPattern.FindAllStringSubmatch(content, -1) {
		if len(match) >= 3 {
			vars[match[1]] = match[2]
		}
	}
	
	return vars
}

// resolveVariable substitutes $variable and ${variable} with values.
func (p *GradleParser) resolveVariable(value string, vars map[string]string) string {
	if !strings.Contains(value, "$") {
		return value
	}
	
	result := value
	for varName, varValue := range vars {
		// Replace ${varName} and $varName
		result = strings.ReplaceAll(result, "${"+varName+"}", varValue)
		result = strings.ReplaceAll(result, "$"+varName, varValue)
	}
	
	return result
}

// isTestScope checks if configuration is test-related.
func (p *GradleParser) isTestScope(config string) bool {
	config = strings.ToLower(config)
	return strings.HasPrefix(config, "test") ||
		config == "compileonly" ||
		config == "runtimeonly" ||
		config == "annotationprocessor" ||
		config == "kapt" ||
		config == "ksp"
}

// CanParse checks if this parser can handle the given file.
func (p *GradleParser) CanParse(filename string) bool {
	base := filename
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		base = filename[idx+1:]
	}
	return base == "build.gradle" || base == "build.gradle.kts"
}

// Language returns the language identifier.
func (p *GradleParser) Language() string {
	return "java"
}

