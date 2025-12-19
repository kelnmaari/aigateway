// Package analyzer provides JSON parsing utilities for LLM responses
package analyzer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// JSONParser parses LLM responses into structured analysis results
type JSONParser struct{}

// NewJSONParser creates a new JSON parser
func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

// ParseAnalysisResult parses full analysis result from LLM response
func (p *JSONParser) ParseAnalysisResult(response string) (*AnalysisResult, error) {
	cleaned := p.cleanResponse(response)
	
	var result AnalysisResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		// Try to fix common JSON issues
		fixed := p.fixCommonJSONIssues(cleaned)
		if err2 := json.Unmarshal([]byte(fixed), &result); err2 != nil {
			return nil, fmt.Errorf("parse analysis result: %w (original error: %v)", err2, err)
		}
	}
	
	// Validate and sanitize
	p.sanitizeResult(&result)
	
	return &result, nil
}

// ParseFileReview parses single file review from LLM response
func (p *JSONParser) ParseFileReview(response string) (*FileReview, error) {
	cleaned := p.cleanResponse(response)
	
	var review FileReview
	if err := json.Unmarshal([]byte(cleaned), &review); err != nil {
		fixed := p.fixCommonJSONIssues(cleaned)
		if err2 := json.Unmarshal([]byte(fixed), &review); err2 != nil {
			return nil, fmt.Errorf("parse file review: %w", err2)
		}
	}
	
	// Validate
	p.sanitizeFileReview(&review)
	
	return &review, nil
}

// ParseSecurityResult parses security-focused analysis
func (p *JSONParser) ParseSecurityResult(response string) (*SecurityAnalysisResult, error) {
	cleaned := p.cleanResponse(response)
	
	var result SecurityAnalysisResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		fixed := p.fixCommonJSONIssues(cleaned)
		if err2 := json.Unmarshal([]byte(fixed), &result); err2 != nil {
			return nil, fmt.Errorf("parse security result: %w", err2)
		}
	}
	
	return &result, nil
}

// SecurityAnalysisResult security-focused analysis result
type SecurityAnalysisResult struct {
	SecurityIssues  []SecurityIssue `json:"security_issues"`
	SecurityScore   int             `json:"security_score"`
	Recommendations []string        `json:"recommendations"`
}

// SecurityIssue security vulnerability
type SecurityIssue struct {
	File              string `json:"file"`
	Line              int    `json:"line"`
	Severity          string `json:"severity"`
	VulnerabilityType string `json:"vulnerability_type"`
	Description       string `json:"description"`
	Remediation       string `json:"remediation"`
}

// cleanResponse cleans LLM response and extracts JSON
func (p *JSONParser) cleanResponse(response string) string {
	response = strings.TrimSpace(response)
	
	// Remove markdown code blocks
	patterns := []string{
		"```json\n", "```json\r\n",
		"```\n", "```\r\n",
		"```",
	}
	
	for _, pattern := range patterns {
		response = strings.ReplaceAll(response, pattern, "")
	}
	
	// Find JSON object boundaries
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	
	if start != -1 && end != -1 && end > start {
		return response[start : end+1]
	}
	
	// Try to find JSON array
	start = strings.Index(response, "[")
	end = strings.LastIndex(response, "]")
	
	if start != -1 && end != -1 && end > start {
		return response[start : end+1]
	}
	
	return response
}

// fixCommonJSONIssues attempts to fix common JSON parsing issues
func (p *JSONParser) fixCommonJSONIssues(jsonStr string) string {
	// Remove trailing commas before } or ]
	re := regexp.MustCompile(`,\s*([}\]])`)
	jsonStr = re.ReplaceAllString(jsonStr, "$1")
	
	// Fix single quotes to double quotes (but preserve escaped quotes)
	// This is a simple heuristic that may not work for all cases
	jsonStr = strings.ReplaceAll(jsonStr, `\'`, `ESCAPED_SINGLE_QUOTE`)
	
	// Replace unescaped single quotes with double quotes in key positions
	// Pattern: 'key': -> "key":
	reKeys := regexp.MustCompile(`'([^']+)'(\s*:)`)
	jsonStr = reKeys.ReplaceAllString(jsonStr, `"$1"$2`)
	
	// Replace unescaped single quotes with double quotes in value positions
	// This is risky but sometimes needed
	// Pattern: : 'value' -> : "value"
	reValues := regexp.MustCompile(`:\s*'([^']*)'`)
	jsonStr = reValues.ReplaceAllString(jsonStr, `: "$1"`)
	
	jsonStr = strings.ReplaceAll(jsonStr, `ESCAPED_SINGLE_QUOTE`, `\'`)
	
	// Fix unescaped newlines in string values
	// This is a heuristic - might cause issues with intentional newlines
	lines := strings.Split(jsonStr, "\n")
	var result []string
	for _, line := range lines {
		result = append(result, strings.TrimRight(line, "\r"))
	}
	jsonStr = strings.Join(result, "\\n")
	
	// Try to re-parse to check if fixed
	// If still invalid, return original
	var test interface{}
	if json.Unmarshal([]byte(jsonStr), &test) != nil {
		// Revert newline fix as it might have broken the JSON
		return strings.ReplaceAll(jsonStr, "\\n", "\n")
	}
	
	return jsonStr
}

// sanitizeResult validates and sanitizes analysis result
func (p *JSONParser) sanitizeResult(result *AnalysisResult) {
	// Clamp overall score
	if result.OverallScore < 0 {
		result.OverallScore = 0
	}
	if result.OverallScore > 100 {
		result.OverallScore = 100
	}
	
	// Sanitize categories
	for i := range result.Categories {
		p.sanitizeCategory(&result.Categories[i])
	}
	
	// Sanitize file reviews
	for i := range result.FileReviews {
		p.sanitizeFileReview(&result.FileReviews[i])
	}
	
	// Sanitize suggestions
	for i := range result.Suggestions {
		p.sanitizeSuggestion(&result.Suggestions[i])
	}
}

// sanitizeCategory validates category result
func (p *JSONParser) sanitizeCategory(cat *CategoryResult) {
	// Validate category name
	validCategories := map[string]bool{
		CategorySecurity:     true,
		CategoryBugs:         true,
		CategoryStyle:        true,
		CategoryPerformance:  true,
		CategoryBestPractice: true,
	}
	
	if !validCategories[cat.Name] {
		cat.Name = CategoryBestPractice // Default
	}
	
	// Clamp score
	if cat.Score < 0 {
		cat.Score = 0
	}
	if cat.Score > 100 {
		cat.Score = 100
	}
	
	// Ensure issues >= 0
	if cat.Issues < 0 {
		cat.Issues = 0
	}
}

// sanitizeFileReview validates file review
func (p *JSONParser) sanitizeFileReview(review *FileReview) {
	// Clamp score
	if review.Score < 0 {
		review.Score = 0
	}
	if review.Score > 100 {
		review.Score = 100
	}
	
	// Sanitize line issues
	for i := range review.LineIssues {
		p.sanitizeLineIssue(&review.LineIssues[i])
	}
}

// sanitizeLineIssue validates line issue
func (p *JSONParser) sanitizeLineIssue(issue *LineIssue) {
	// Ensure line > 0
	if issue.Line < 1 {
		issue.Line = 1
	}
	
	// Validate severity
	validSeverities := map[string]bool{
		SeverityCritical:   true,
		SeverityWarning:    true,
		SeverityInfo:       true,
		SeveritySuggestion: true,
	}
	
	if !validSeverities[issue.Severity] {
		issue.Severity = SeverityInfo // Default
	}
	
	// Validate category
	validCategories := map[string]bool{
		CategorySecurity:     true,
		CategoryBugs:         true,
		CategoryStyle:        true,
		CategoryPerformance:  true,
		CategoryBestPractice: true,
	}
	
	if !validCategories[issue.Category] {
		issue.Category = CategoryBestPractice // Default
	}
}

// sanitizeSuggestion validates suggestion
func (p *JSONParser) sanitizeSuggestion(suggestion *Suggestion) {
	// Validate category
	validCategories := map[string]bool{
		CategorySecurity:     true,
		CategoryBugs:         true,
		CategoryStyle:        true,
		CategoryPerformance:  true,
		CategoryBestPractice: true,
	}
	
	if !validCategories[suggestion.Category] {
		suggestion.Category = CategoryBestPractice
	}
	
	// Validate priority
	validPriorities := map[string]bool{
		"high":   true,
		"medium": true,
		"low":    true,
	}
	
	if !validPriorities[suggestion.Priority] {
		suggestion.Priority = "medium"
	}
}

// TryParsePartialJSON attempts to parse potentially incomplete JSON
// Returns best-effort result even if parsing partially fails
func (p *JSONParser) TryParsePartialJSON(response string) (*AnalysisResult, []string) {
	var errors []string
	result := &AnalysisResult{
		OverallScore: 50, // Default moderate score
	}
	
	cleaned := p.cleanResponse(response)
	
	// Try full parse first
	if err := json.Unmarshal([]byte(cleaned), result); err == nil {
		return result, nil
	}
	
	// Try to extract individual fields
	
	// Extract summary
	if match := regexp.MustCompile(`"summary"\s*:\s*"([^"]+)"`).FindStringSubmatch(cleaned); len(match) > 1 {
		result.Summary = match[1]
	} else {
		errors = append(errors, "could not extract summary")
	}
	
	// Extract overall_score
	if match := regexp.MustCompile(`"overall_score"\s*:\s*(\d+)`).FindStringSubmatch(cleaned); len(match) > 1 {
		var score int
		fmt.Sscanf(match[1], "%d", &score)
		result.OverallScore = score
	} else {
		errors = append(errors, "could not extract overall_score")
	}
	
	// Try to extract file_reviews array
	if idx := strings.Index(cleaned, `"file_reviews"`); idx != -1 {
		// Find the array start
		arrayStart := strings.Index(cleaned[idx:], "[")
		if arrayStart != -1 {
			// Find matching bracket
			start := idx + arrayStart
			bracketCount := 1
			end := start + 1
			
			for end < len(cleaned) && bracketCount > 0 {
				switch cleaned[end] {
				case '[':
					bracketCount++
				case ']':
					bracketCount--
				}
				end++
			}
			
			if bracketCount == 0 {
				arrayJSON := cleaned[start:end]
				var reviews []FileReview
				if err := json.Unmarshal([]byte(arrayJSON), &reviews); err == nil {
					result.FileReviews = reviews
				}
			}
		}
	}
	
	return result, errors
}

// ============================================================================
// Convenience functions for processor
// ============================================================================

// GetSystemPrompt returns the system prompt, optionally with custom additions
func GetSystemPrompt(customPrompt string) string {
	if customPrompt != "" {
		return SystemPrompt + "\n\nAdditional instructions:\n" + customPrompt
	}
	return SystemPrompt
}

// ParseAnalysisResponse parses LLM response into AnalysisResultParsed
func ParseAnalysisResponse(response string) (*AnalysisResultParsed, error) {
	parser := NewJSONParser()
	
	// First try to parse as full AnalysisResult
	fullResult, err := parser.ParseAnalysisResult(response)
	if err != nil {
		return nil, err
	}
	
	// Convert to simplified format for processor
	result := &AnalysisResultParsed{
		Summary: fullResult.Summary,
		Score:   fullResult.OverallScore,
		Issues:  make([]Issue, 0),
		Suggestions: make([]SuggestionItem, 0),
	}
	
	// Extract issues from file reviews
	for _, fileReview := range fullResult.FileReviews {
		for _, lineIssue := range fileReview.LineIssues {
			result.Issues = append(result.Issues, Issue{
				FilePath:   fileReview.FilePath,
				Line:       lineIssue.Line,
				EndLine:    lineIssue.EndLine,
				Severity:   lineIssue.Severity,
				Category:   lineIssue.Category,
				Message:    lineIssue.Message,
				Suggestion: lineIssue.Suggestion,
			})
		}
	}
	
	// Convert suggestions
	for _, suggestion := range fullResult.Suggestions {
		result.Suggestions = append(result.Suggestions, SuggestionItem{
			Type:        suggestion.Category,
			Title:       suggestion.Title,
			Description: suggestion.Description,
			Priority:    suggestion.Priority,
		})
	}
	
	return result, nil
}

