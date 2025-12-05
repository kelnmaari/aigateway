// Package analyzer provides code analysis prompts
package analyzer

import (
	"fmt"
	"strings"
)

// SystemPrompt базовый системный промпт для code review
const SystemPrompt = `You are an expert code reviewer with deep knowledge in software security, design patterns, and best practices. Your task is to analyze code changes and provide actionable, specific feedback.

## Your Expertise Includes:
- Security vulnerabilities (SQL injection, XSS, CSRF, authentication flaws, etc.)
- Common bug patterns and logic errors
- Code quality and maintainability
- Performance optimizations
- Language-specific best practices

## Review Guidelines:
1. Focus on issues that matter - don't nitpick minor style issues
2. Provide specific line numbers when reporting issues
3. Explain WHY something is problematic
4. Suggest concrete fixes when possible
5. Consider the context of the change
6. Be constructive and professional

## Output Format:
You MUST respond with valid JSON only. No markdown, no explanations outside JSON.`

// ReviewPromptTemplate шаблон для промпта ревью
const ReviewPromptTemplate = `Analyze the following merge request and provide a structured code review.

## Merge Request Information
- **Title**: %s
- **Author**: %s  
- **Source Branch**: %s → %s
- **Description**: %s

## Files Changed
%s

## Your Task
Review each file and identify:

1. **Security Issues** (critical priority)
   - SQL injection, XSS, CSRF vulnerabilities
   - Hardcoded secrets or credentials
   - Improper input validation
   - Authentication/authorization flaws
   - Insecure data handling

2. **Bugs** (high priority)
   - Logic errors
   - Null pointer/undefined access
   - Resource leaks
   - Race conditions
   - Error handling issues
   - Edge cases not handled

3. **Code Quality** (medium priority)
   - Code duplication
   - Complex/unreadable code
   - Naming conventions
   - Missing documentation for public APIs
   - SOLID principles violations

4. **Performance** (medium priority)
   - N+1 queries
   - Unnecessary allocations
   - Inefficient algorithms
   - Missing caching opportunities

## Response Format
Respond with a JSON object in this EXACT structure:

{
  "summary": "Brief overall assessment in 1-2 sentences",
  "overall_score": 85,
  "categories": [
    {
      "name": "security",
      "score": 90,
      "issues": 1,
      "details": "Brief description of security findings"
    },
    {
      "name": "bugs",
      "score": 80,
      "issues": 2,
      "details": "Brief description of bug findings"
    },
    {
      "name": "style",
      "score": 85,
      "issues": 3,
      "details": "Brief description of style findings"
    },
    {
      "name": "performance",
      "score": 90,
      "issues": 0,
      "details": "Brief description of performance findings"
    }
  ],
  "file_reviews": [
    {
      "file_path": "path/to/file.go",
      "score": 80,
      "summary": "Brief file summary",
      "line_issues": [
        {
          "line": 42,
          "severity": "warning",
          "category": "security",
          "message": "What the issue is",
          "suggestion": "How to fix it"
        }
      ]
    }
  ],
  "suggestions": [
    {
      "category": "best_practice",
      "title": "Suggestion title",
      "description": "Detailed suggestion",
      "priority": "medium"
    }
  ]
}

IMPORTANT:
- Score is 0-100 (higher is better)
- severity: "critical", "warning", "info", or "suggestion"
- category: "security", "bugs", "style", "performance", or "best_practice"
- priority: "high", "medium", or "low"
- Only include file_reviews for files with actual issues
- Be specific with line numbers - they should match the diff
- If no issues found, set empty arrays and high scores`

// BuildReviewPrompt создает промпт для ревью MR
func BuildReviewPrompt(req *AnalysisRequest) string {
	// Build files section
	var filesSection strings.Builder
	for i, change := range req.Changes {
		filesSection.WriteString(fmt.Sprintf("\n### File %d: `%s`\n", i+1, change.FilePath))
		
		if change.NewFile {
			filesSection.WriteString("**Status**: New file\n")
		} else if change.DeletedFile {
			filesSection.WriteString("**Status**: Deleted\n")
		} else if change.RenamedFile {
			filesSection.WriteString(fmt.Sprintf("**Status**: Renamed from `%s`\n", change.OldPath))
		} else {
			filesSection.WriteString("**Status**: Modified\n")
		}
		
		if change.Language != "" {
			filesSection.WriteString(fmt.Sprintf("**Language**: %s\n", change.Language))
		}
		
		filesSection.WriteString(fmt.Sprintf("**Changes**: +%d/-%d lines\n", change.LinesAdded, change.LinesRemoved))
		
		// Include diff
		if change.Diff != "" {
			filesSection.WriteString("\n```diff\n")
			filesSection.WriteString(change.Diff)
			filesSection.WriteString("\n```\n")
		}
	}
	
	description := req.MRDescription
	if description == "" {
		description = "(No description provided)"
	}
	
	return fmt.Sprintf(ReviewPromptTemplate,
		req.MRTitle,
		req.MRAuthor,
		req.SourceBranch,
		req.TargetBranch,
		description,
		filesSection.String(),
	)
}

// BuildFilePrompt создает промпт для анализа конкретного файла
func BuildFilePrompt(change FileChange, context string) string {
	return fmt.Sprintf(`Analyze this single file change and provide detailed feedback.

## File: %s
**Language**: %s
**Changes**: +%d/-%d lines

%s

## Diff:
%s%s

Respond with JSON containing ONLY this file's review:
{
  "file_path": "%s",
  "score": 85,
  "summary": "Brief summary",
  "line_issues": [
    {
      "line": 42,
      "severity": "warning",
      "category": "security",
      "message": "Issue description",
      "suggestion": "How to fix"
    }
  ]
}`,
		change.FilePath,
		getLanguage(change),
		change.LinesAdded,
		change.LinesRemoved,
		formatContext(context),
		change.Diff,
		formatNewContent(change),
		change.FilePath,
	)
}

// BuildSecurityFocusPrompt промпт для security-focused анализа
func BuildSecurityFocusPrompt(req *AnalysisRequest) string {
	var filesSection strings.Builder
	for _, change := range req.Changes {
		if change.DeletedFile {
			continue
		}
		filesSection.WriteString(fmt.Sprintf("\n### %s\n```diff\n%s\n```\n", change.FilePath, change.Diff))
	}
	
	return fmt.Sprintf(`You are a security expert. Analyze these code changes for security vulnerabilities ONLY.

## Files to Analyze
%s

Focus exclusively on:
- SQL Injection
- XSS (Cross-Site Scripting)
- CSRF vulnerabilities
- Authentication/Authorization flaws
- Hardcoded secrets/credentials
- Insecure deserialization
- Path traversal
- Command injection
- Insecure cryptography
- Sensitive data exposure

Respond with JSON:
{
  "security_issues": [
    {
      "file": "path/to/file",
      "line": 42,
      "severity": "critical",
      "vulnerability_type": "SQL Injection",
      "description": "Detailed description",
      "remediation": "How to fix"
    }
  ],
  "security_score": 85,
  "recommendations": ["General recommendation 1", "General recommendation 2"]
}

If no security issues found, return empty security_issues array and score 100.`,
		filesSection.String(),
	)
}

// CustomPromptWrapper оборачивает custom prompt пользователя
func CustomPromptWrapper(customPrompt string, files string) string {
	return fmt.Sprintf(`%s

## Files to Review
%s

Remember to respond with valid JSON matching the expected schema.`,
		customPrompt,
		files,
	)
}

// Helper functions

func getLanguage(change FileChange) string {
	if change.Language != "" {
		return change.Language
	}
	return DetectLanguage(change.FilePath)
}

func formatContext(context string) string {
	if context == "" {
		return ""
	}
	return fmt.Sprintf("## Additional Context (from codebase)\n%s\n", context)
}

func formatNewContent(change FileChange) string {
	if change.NewContent == "" || change.DeletedFile {
		return ""
	}
	// For new files, show full content if diff is empty
	if change.NewFile && change.Diff == "" {
		return fmt.Sprintf("\n## Full Content:\n```%s\n%s\n```", change.Language, change.NewContent)
	}
	return ""
}

// DetectLanguage определяет язык по расширению файла
func DetectLanguage(filepath string) string {
	extensions := map[string]string{
		".go":     "go",
		".py":     "python",
		".js":     "javascript",
		".ts":     "typescript",
		".tsx":    "typescript",
		".jsx":    "javascript",
		".java":   "java",
		".kt":     "kotlin",
		".rs":     "rust",
		".rb":     "ruby",
		".php":    "php",
		".c":      "c",
		".cpp":    "cpp",
		".cc":     "cpp",
		".h":      "c",
		".hpp":    "cpp",
		".cs":     "csharp",
		".swift":  "swift",
		".m":      "objective-c",
		".sql":    "sql",
		".sh":     "bash",
		".bash":   "bash",
		".yaml":   "yaml",
		".yml":    "yaml",
		".json":   "json",
		".xml":    "xml",
		".html":   "html",
		".css":    "css",
		".scss":   "scss",
		".less":   "less",
		".md":     "markdown",
		".proto":  "protobuf",
		".graphql": "graphql",
		".tf":     "terraform",
		".vue":    "vue",
		".svelte": "svelte",
	}
	
	filepath = strings.ToLower(filepath)
	for ext, lang := range extensions {
		if strings.HasSuffix(filepath, ext) {
			return lang
		}
	}
	
	// Check for specific filenames
	filenames := map[string]string{
		"dockerfile":    "dockerfile",
		"makefile":      "makefile",
		"cmakelists.txt": "cmake",
		".gitignore":   "gitignore",
		".env":         "dotenv",
	}
	
	for name, lang := range filenames {
		if strings.HasSuffix(filepath, name) {
			return lang
		}
	}
	
	return "text"
}

