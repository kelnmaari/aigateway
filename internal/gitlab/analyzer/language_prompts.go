// Package analyzer provides language-specific prompts for code review
package analyzer

import "strings"

// LanguagePrompt contains language-specific review instructions
type LanguagePrompt struct {
	Language     string   `json:"language"`
	Extensions   []string `json:"extensions"`
	Instructions string   `json:"instructions"`
	Checks       []string `json:"checks"`
	Patterns     []string `json:"patterns"` // Anti-patterns to look for
}

// LanguagePromptManager manages language-specific prompts
type LanguagePromptManager struct {
	prompts       map[string]*LanguagePrompt
	customPrompts map[string]string // User-defined overrides
}

// NewLanguagePromptManager creates a new prompt manager with default prompts
func NewLanguagePromptManager() *LanguagePromptManager {
	m := &LanguagePromptManager{
		prompts:       make(map[string]*LanguagePrompt),
		customPrompts: make(map[string]string),
	}
	m.loadDefaultPrompts()
	return m
}

// GetPromptForFile returns the appropriate prompt for a file
func (m *LanguagePromptManager) GetPromptForFile(filename string) *LanguagePrompt {
	ext := getFileExtension(filename)
	
	// Check for exact extension match
	for _, prompt := range m.prompts {
		for _, e := range prompt.Extensions {
			if e == ext {
				return prompt
			}
		}
	}
	
	return m.prompts["generic"]
}

// GetPromptForLanguage returns the prompt for a specific language
func (m *LanguagePromptManager) GetPromptForLanguage(language string) *LanguagePrompt {
	lang := strings.ToLower(language)
	if prompt, ok := m.prompts[lang]; ok {
		return prompt
	}
	return m.prompts["generic"]
}

// SetCustomPrompt sets a custom prompt for a language
func (m *LanguagePromptManager) SetCustomPrompt(language, prompt string) {
	m.customPrompts[strings.ToLower(language)] = prompt
}

// GetInstructions returns combined instructions for a language
func (m *LanguagePromptManager) GetInstructions(language string) string {
	lang := strings.ToLower(language)
	
	// Check for custom override first
	if custom, ok := m.customPrompts[lang]; ok {
		return custom
	}
	
	prompt := m.GetPromptForLanguage(language)
	if prompt == nil {
		return ""
	}
	
	return prompt.Instructions
}

// BuildReviewPrompt builds a complete review prompt for given languages
func (m *LanguagePromptManager) BuildReviewPrompt(languages []string, basePrompt string) string {
	var sb strings.Builder
	
	sb.WriteString(basePrompt)
	sb.WriteString("\n\n## Language-Specific Guidelines\n\n")
	
	seen := make(map[string]bool)
	for _, lang := range languages {
		lang = strings.ToLower(lang)
		if seen[lang] {
			continue
		}
		seen[lang] = true
		
		prompt := m.GetPromptForLanguage(lang)
		if prompt != nil && prompt.Instructions != "" {
			sb.WriteString("### ")
			sb.WriteString(strings.Title(lang))
			sb.WriteString("\n")
			sb.WriteString(prompt.Instructions)
			sb.WriteString("\n\n")
		}
	}
	
	return sb.String()
}

func (m *LanguagePromptManager) loadDefaultPrompts() {
	// Go
	m.prompts["go"] = &LanguagePrompt{
		Language:   "go",
		Extensions: []string{".go"},
		Instructions: `- Check for proper error handling (don't ignore errors with _)
- Verify context.Context is passed and used correctly
- Look for goroutine leaks and proper cleanup
- Check for race conditions in concurrent code
- Verify defer usage for resource cleanup
- Check for nil pointer dereferences
- Ensure interfaces are used appropriately
- Look for proper use of sync primitives (Mutex, RWMutex)
- Check for efficient string concatenation (use strings.Builder)
- Verify proper package organization`,
		Checks: []string{
			"error-handling",
			"context-usage",
			"concurrency",
			"nil-checks",
			"defer-usage",
		},
		Patterns: []string{
			"_ = someFunction()", // ignored error
			"go func()", // potential goroutine leak
		},
	}

	// TypeScript
	m.prompts["typescript"] = &LanguagePrompt{
		Language:   "typescript",
		Extensions: []string{".ts", ".tsx"},
		Instructions: `- Check for proper TypeScript types (avoid 'any')
- Verify null/undefined checks
- Look for proper async/await usage
- Check for potential memory leaks in React components
- Verify proper cleanup in useEffect
- Check for missing error boundaries
- Look for proper prop types and interfaces
- Verify import/export consistency
- Check for circular dependencies`,
		Checks: []string{
			"type-safety",
			"null-checks",
			"async-await",
			"react-hooks",
			"memory-leaks",
		},
		Patterns: []string{
			": any", // avoid any type
			"as any",
			"// @ts-ignore",
		},
	}

	// JavaScript
	m.prompts["javascript"] = &LanguagePrompt{
		Language:   "javascript",
		Extensions: []string{".js", ".jsx", ".mjs"},
		Instructions: `- Check for proper null/undefined handling
- Look for potential XSS vulnerabilities
- Verify async/await and Promise usage
- Check for memory leaks (event listeners, timers)
- Look for proper error handling in async code
- Check for use of const/let instead of var
- Verify proper equality checks (=== vs ==)
- Look for potential race conditions`,
		Checks: []string{
			"null-checks",
			"xss-prevention",
			"async-patterns",
			"memory-management",
		},
	}

	// Python
	m.prompts["python"] = &LanguagePrompt{
		Language:   "python",
		Extensions: []string{".py"},
		Instructions: `- Check for proper exception handling
- Verify type hints are used consistently
- Look for proper resource management (with statements)
- Check for mutable default arguments
- Verify proper use of list comprehensions
- Look for potential security issues (eval, exec)
- Check for proper logging usage
- Verify docstrings are present
- Look for PEP 8 style violations`,
		Checks: []string{
			"exception-handling",
			"type-hints",
			"resource-management",
			"security",
			"style",
		},
		Patterns: []string{
			"def func(arg=[])", // mutable default
			"eval(",
			"exec(",
		},
	}

	// Rust
	m.prompts["rust"] = &LanguagePrompt{
		Language:   "rust",
		Extensions: []string{".rs"},
		Instructions: `- Check for proper error handling with Result/Option
- Verify ownership and borrowing patterns
- Look for potential panics (unwrap, expect)
- Check for proper use of lifetimes
- Verify unsafe code is justified and correct
- Look for proper trait implementations
- Check for proper async/await patterns
- Verify clippy warnings are addressed`,
		Checks: []string{
			"error-handling",
			"ownership",
			"unsafe-usage",
			"lifetimes",
			"async-patterns",
		},
		Patterns: []string{
			".unwrap()", // potential panic
			".expect(",
			"unsafe {",
		},
	}

	// Java
	m.prompts["java"] = &LanguagePrompt{
		Language:   "java",
		Extensions: []string{".java"},
		Instructions: `- Check for proper null handling (use Optional where appropriate)
- Verify exception handling (don't catch generic Exception)
- Look for resource leaks (use try-with-resources)
- Check for thread safety in concurrent code
- Verify proper equals/hashCode implementation
- Look for proper use of streams and lambdas
- Check for SQL injection vulnerabilities
- Verify proper logging practices`,
		Checks: []string{
			"null-safety",
			"exception-handling",
			"resource-management",
			"thread-safety",
			"security",
		},
	}

	// C/C++
	m.prompts["cpp"] = &LanguagePrompt{
		Language:   "cpp",
		Extensions: []string{".cpp", ".cc", ".cxx", ".c", ".h", ".hpp"},
		Instructions: `- Check for memory leaks (use smart pointers)
- Verify buffer overflow prevention
- Look for null pointer dereferences
- Check for proper RAII patterns
- Verify exception safety guarantees
- Look for undefined behavior
- Check for proper const correctness
- Verify proper use of move semantics`,
		Checks: []string{
			"memory-safety",
			"buffer-overflow",
			"null-checks",
			"raii",
			"const-correctness",
		},
		Patterns: []string{
			"new ", // without smart pointer
			"delete ",
			"malloc(",
			"free(",
		},
	}

	// SQL
	m.prompts["sql"] = &LanguagePrompt{
		Language:   "sql",
		Extensions: []string{".sql"},
		Instructions: `- Check for SQL injection vulnerabilities
- Verify proper use of indexes
- Look for N+1 query patterns
- Check for proper transaction handling
- Verify proper NULL handling
- Look for potential deadlocks
- Check for proper JOIN usage
- Verify query performance considerations`,
		Checks: []string{
			"injection",
			"performance",
			"transactions",
			"null-handling",
		},
	}

	// Shell/Bash
	m.prompts["shell"] = &LanguagePrompt{
		Language:   "shell",
		Extensions: []string{".sh", ".bash"},
		Instructions: `- Check for proper quoting of variables
- Verify error handling with set -e
- Look for command injection vulnerabilities
- Check for proper use of shellcheck recommendations
- Verify proper handling of special characters
- Look for POSIX compatibility issues
- Check for proper exit codes`,
		Checks: []string{
			"quoting",
			"error-handling",
			"security",
			"compatibility",
		},
		Patterns: []string{
			"$VAR", // unquoted variable
			"eval ",
		},
	}

	// YAML
	m.prompts["yaml"] = &LanguagePrompt{
		Language:   "yaml",
		Extensions: []string{".yml", ".yaml"},
		Instructions: `- Check for proper indentation
- Verify proper quoting of strings
- Look for security issues in CI/CD configs
- Check for proper use of anchors and aliases
- Verify proper structure and nesting`,
		Checks: []string{
			"syntax",
			"security",
			"structure",
		},
	}

	// Dockerfile
	m.prompts["dockerfile"] = &LanguagePrompt{
		Language:   "dockerfile",
		Extensions: []string{"Dockerfile", ".dockerfile"},
		Instructions: `- Check for proper base image versioning
- Verify multi-stage builds for smaller images
- Look for security vulnerabilities (running as root)
- Check for proper layer caching
- Verify proper COPY vs ADD usage
- Look for secrets in build args
- Check for proper health checks`,
		Checks: []string{
			"security",
			"optimization",
			"best-practices",
		},
		Patterns: []string{
			"FROM latest",
			"USER root",
			"ADD http",
		},
	}

	// Terraform
	m.prompts["terraform"] = &LanguagePrompt{
		Language:   "terraform",
		Extensions: []string{".tf", ".tfvars"},
		Instructions: `- Check for hardcoded secrets
- Verify proper use of variables
- Look for missing resource tags
- Check for proper state management
- Verify proper module versioning
- Look for security group issues
- Check for proper resource naming`,
		Checks: []string{
			"security",
			"variables",
			"naming",
			"state",
		},
	}

	// Generic fallback
	m.prompts["generic"] = &LanguagePrompt{
		Language:   "generic",
		Extensions: []string{},
		Instructions: `- Check for general code quality issues
- Look for potential bugs
- Verify proper error handling
- Check for security vulnerabilities
- Look for performance issues
- Verify code readability and maintainability`,
		Checks: []string{
			"quality",
			"bugs",
			"security",
			"performance",
		},
	}
}

func getFileExtension(filename string) string {
	// Handle special filenames like "Dockerfile"
	if strings.Contains(filename, "Dockerfile") {
		return "Dockerfile"
	}
	
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return ""
	}
	return filename[idx:]
}

// DetectLanguage detects the primary language from a list of files
func DetectLanguage(files []string) string {
	counts := make(map[string]int)
	
	extToLang := map[string]string{
		".go":         "go",
		".ts":         "typescript",
		".tsx":        "typescript",
		".js":         "javascript",
		".jsx":        "javascript",
		".py":         "python",
		".rs":         "rust",
		".java":       "java",
		".cpp":        "cpp",
		".cc":         "cpp",
		".c":          "c",
		".h":          "cpp",
		".hpp":        "cpp",
		".sql":        "sql",
		".sh":         "shell",
		".bash":       "shell",
		".yml":        "yaml",
		".yaml":       "yaml",
		".tf":         "terraform",
		"Dockerfile":  "dockerfile",
	}
	
	for _, file := range files {
		ext := getFileExtension(file)
		if lang, ok := extToLang[ext]; ok {
			counts[lang]++
		}
	}
	
	// Find most common language
	maxCount := 0
	maxLang := "generic"
	for lang, count := range counts {
		if count > maxCount {
			maxCount = count
			maxLang = lang
		}
	}
	
	return maxLang
}

// GetAllLanguages returns all files' languages
func GetAllLanguages(files []string) []string {
	seen := make(map[string]bool)
	var languages []string
	
	extToLang := map[string]string{
		".go":         "go",
		".ts":         "typescript",
		".tsx":        "typescript",
		".js":         "javascript",
		".jsx":        "javascript",
		".py":         "python",
		".rs":         "rust",
		".java":       "java",
		".cpp":        "cpp",
		".cc":         "cpp",
		".c":          "c",
		".h":          "cpp",
		".hpp":        "cpp",
		".sql":        "sql",
		".sh":         "shell",
		".bash":       "shell",
		".yml":        "yaml",
		".yaml":       "yaml",
		".tf":         "terraform",
		"Dockerfile":  "dockerfile",
	}
	
	for _, file := range files {
		ext := getFileExtension(file)
		if lang, ok := extToLang[ext]; ok {
			if !seen[lang] {
				seen[lang] = true
				languages = append(languages, lang)
			}
		}
	}
	
	return languages
}

