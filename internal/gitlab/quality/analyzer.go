// Package quality provides code quality analysis functionality.
package quality

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"aigateway/internal/rag/vector"

	"github.com/sirupsen/logrus"
)

// Analyzer performs code quality analysis using LLM.
type Analyzer struct {
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	httpClient  *http.Client
	logger      *logrus.Logger
}

// NewAnalyzer creates a new quality analyzer.
func NewAnalyzer(vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *Analyzer {
	return &Analyzer{
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Minute,
		},
		logger: logger,
	}
}

// Analyze performs quality analysis on a project.
func (a *Analyzer) Analyze(ctx context.Context, req AnalysisRequest) (*QualityScore, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("quality-%s", time.Now().Format("20060102-150405"))

	a.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
		"model":      req.ModelID,
		"scan_id":    scanID,
	}).Info("Starting code quality analysis")

	result := &QualityScore{
		ProjectID:       req.ProjectID,
		ScanID:          scanID,
		ScannedAt:       startTime,
		Status:          "running",
		Breakdown:       make(map[ScoreCategory]int),
		FileScores:      []FileScore{},
		Recommendations: []Recommendation{},
		ModelID:         req.ModelID,
	}

	// Set default max files
	maxFiles := req.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 50
	}

	// Get code files from index
	files, err := a.getCodeFiles(ctx, req.CollectionName, req.ProjectID, maxFiles)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Failed to get code files: %v", err)
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	if len(files) == 0 {
		result.Status = "completed"
		result.Error = "No code files found in index"
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	a.logger.WithField("files", len(files)).Debug("Found code files to analyze")

	// Analyze files concurrently (limit concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, 5) // 5 concurrent analyses
	totalTokens := 0

	for _, file := range files {
		wg.Add(1)
		go func(f codeFile) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			fileScore, tokens, err := a.analyzeFile(ctx, req.ModelID, f)
			if err != nil {
				a.logger.WithError(err).WithField("file", f.Path).Warn("Failed to analyze file")
				return
			}

			mu.Lock()
			result.FileScores = append(result.FileScores, fileScore)
			totalTokens += tokens
			mu.Unlock()
		}(file)
	}

	wg.Wait()

	result.TokensUsed = totalTokens

	// Calculate overall scores
	a.calculateOverallScores(result)

	// Generate recommendations
	a.generateRecommendations(result)

	// Build summary
	a.buildSummary(result)

	result.Status = "completed"
	result.Duration = time.Since(startTime).String()

	a.logger.WithFields(logrus.Fields{
		"project_id":    req.ProjectID,
		"scan_id":       scanID,
		"overall_score": result.OverallScore,
		"files":         len(result.FileScores),
		"issues":        result.Summary.IssuesCount,
		"tokens":        totalTokens,
		"duration":      result.Duration,
	}).Info("Code quality analysis completed")

	return result, nil
}

// codeFile represents a file from the index
type codeFile struct {
	Path     string
	Language string
	Content  string
	Lines    int
}

// getCodeFiles retrieves code files from the vector store
func (a *Analyzer) getCodeFiles(ctx context.Context, collection, projectID string, maxFiles int) ([]codeFile, error) {
	var files []codeFile
	seen := make(map[string]bool)

	err := a.vectorStore.ScrollAll(ctx, collection, map[string]interface{}{
		"project_id": projectID,
	}, func(docs []vector.VectorDocument) error {
		for _, doc := range docs {
			if len(files) >= maxFiles {
				return nil
			}

			filePath, ok := doc.Metadata["file_path"].(string)
			if !ok || filePath == "" {
				continue
			}

			// Skip if already seen
			if seen[filePath] {
				continue
			}

			// Skip non-code files
			if !isCodeFile(filePath) {
				continue
			}

			seen[filePath] = true

			lang := detectLanguage(filePath)
			content := doc.Text
			if c, ok := doc.Metadata["content"].(string); ok && c != "" {
				content = c
			}

			lines := strings.Count(content, "\n") + 1

			files = append(files, codeFile{
				Path:     filePath,
				Language: lang,
				Content:  content,
				Lines:    lines,
			})
		}
		return nil
	})

	return files, err
}

// analyzeFile analyzes a single file using LLM
func (a *Analyzer) analyzeFile(ctx context.Context, modelID string, file codeFile) (FileScore, int, error) {
	score := FileScore{
		FilePath:    file.Path,
		Language:    file.Language,
		LinesOfCode: file.Lines,
		Breakdown:   make(map[ScoreCategory]int),
		Issues:      []QualityIssue{},
	}

	// Truncate content if too long
	content := file.Content
	if len(content) > 8000 {
		content = content[:8000] + "\n... (truncated)"
	}

	codeBlock := "```" + file.Language + "\n" + content + "\n```"
	prompt := fmt.Sprintf(`Analyze the code quality of this %s file.

File: %s
Lines: %d

Code:
%s

Evaluate the following categories (score 0-100 for each):
1. complexity - Code complexity (lower is better, inverse scale)
2. documentation - Comments and documentation quality
3. naming - Variable/function naming quality
4. error_handling - Error handling completeness
5. maintainability - Overall maintainability
6. testing - Test coverage estimation (for test files: quality of tests; for source files: testability and whether tests likely exist)

For testing estimation:
- If this IS a test file (*_test.go, *.test.ts, test_*.py, etc): evaluate test quality, coverage thoroughness, edge cases
- If this is a SOURCE file: estimate how testable the code is and whether corresponding tests likely exist (0-30 if no tests likely, 30-60 if partially tested, 60-100 if well tested)

For each issue found, provide:
- category: one of [complexity, documentation, naming, error_handling, maintainability, security, testing]
- severity: high/medium/low
- line: approximate line number (or 0 if general)
- message: brief description
- suggestion: how to fix

Respond in JSON format ONLY:
{
  "overall_score": 75,
  "breakdown": {"complexity": 80, "documentation": 60, "naming": 85, "error_handling": 70, "maintainability": 75, "testing": 50},
  "issues": [{"category": "testing", "severity": "medium", "line": 0, "message": "No corresponding test file found", "suggestion": "Add unit tests for this file"}],
  "functions_count": 5,
  "is_test_file": false,
  "testable_functions": 3,
  "tested_functions_estimate": 1
}`, file.Language, file.Path, file.Lines, codeBlock)

	// Make LLM request
	response, tokens, err := a.callLLM(ctx, modelID, prompt)
	if err != nil {
		return score, tokens, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse response
	var parsed struct {
		OverallScore            int            `json:"overall_score"`
		Breakdown               map[string]int `json:"breakdown"`
		Issues                  []QualityIssue `json:"issues"`
		FunctionsCount          int            `json:"functions_count"`
		IsTestFile              bool           `json:"is_test_file"`
		TestableFunctions       int            `json:"testable_functions"`
		TestedFunctionsEstimate int            `json:"tested_functions_estimate"`
	}

	// Extract JSON from response
	jsonStr := extractJSON(response)
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		a.logger.WithError(err).WithField("file", file.Path).Debug("Failed to parse LLM response")
		// Return default scores
		score.Score = 70
		score.Breakdown[CategoryComplexity] = 70
		score.Breakdown[CategoryDocumentation] = 70
		score.Breakdown[CategoryMaintainability] = 70
		score.Breakdown[CategoryTesting] = 50 // Default unknown testing score
		return score, tokens, nil
	}

	score.Score = parsed.OverallScore
	score.FunctionsCount = parsed.FunctionsCount
	score.Issues = parsed.Issues
	score.IsTestFile = parsed.IsTestFile
	score.TestableFunctions = parsed.TestableFunctions
	score.TestedFunctionsEstimate = parsed.TestedFunctionsEstimate

	// Convert breakdown
	for cat, val := range parsed.Breakdown {
		score.Breakdown[ScoreCategory(cat)] = val
	}

	// Ensure testing category is set
	if _, ok := score.Breakdown[CategoryTesting]; !ok {
		// Estimate testing score based on test file presence
		if score.IsTestFile {
			score.Breakdown[CategoryTesting] = 75 // Test files get higher base score
		} else if score.TestableFunctions > 0 && score.TestedFunctionsEstimate > 0 {
			// Estimate based on coverage ratio
			coverageRatio := float64(score.TestedFunctionsEstimate) / float64(score.TestableFunctions)
			score.Breakdown[CategoryTesting] = int(coverageRatio * 100)
		} else {
			score.Breakdown[CategoryTesting] = 30 // Low score for untested code
		}
	}

	return score, tokens, nil
}

// callLLM makes a request to the LLM API
func (a *Analyzer) callLLM(ctx context.Context, modelID, prompt string) (string, int, error) {
	reqBody := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a code quality analyzer. Analyze code and provide scores and issues in JSON format only. No explanations outside JSON."},
			{"role": "user", "content": prompt},
		},
		"max_tokens":  2000,
		"temperature": 0.3,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.llmBaseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	if a.llmAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.llmAPIKey)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("LLM API error: %s (status %d)", string(body), resp.StatusCode)
	}

	var llmResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		return "", 0, err
	}

	if llmResponse.Error != nil {
		return "", 0, fmt.Errorf("LLM error: %s", llmResponse.Error.Message)
	}

	if len(llmResponse.Choices) == 0 {
		return "", 0, fmt.Errorf("no response from LLM")
	}

	return llmResponse.Choices[0].Message.Content, llmResponse.Usage.TotalTokens, nil
}

// calculateOverallScores calculates aggregate scores
func (a *Analyzer) calculateOverallScores(result *QualityScore) {
	if len(result.FileScores) == 0 {
		return
	}

	// Calculate weighted average (weighted by lines of code)
	var totalLines int
	var weightedSum int

	categoryTotals := make(map[ScoreCategory]int)
	categoryCounts := make(map[ScoreCategory]int)

	for _, fs := range result.FileScores {
		totalLines += fs.LinesOfCode
		weightedSum += fs.Score * fs.LinesOfCode

		for cat, score := range fs.Breakdown {
			categoryTotals[cat] += score
			categoryCounts[cat]++
		}
	}

	if totalLines > 0 {
		result.OverallScore = weightedSum / totalLines
	}

	// Calculate category averages
	for cat := range categoryTotals {
		if categoryCounts[cat] > 0 {
			result.Breakdown[cat] = categoryTotals[cat] / categoryCounts[cat]
		}
	}
}

// generateRecommendations generates actionable recommendations
func (a *Analyzer) generateRecommendations(result *QualityScore) {
	// Collect all issues
	var allIssues []QualityIssue
	for _, fs := range result.FileScores {
		allIssues = append(allIssues, fs.Issues...)
	}

	// Group by category
	categoryIssues := make(map[ScoreCategory][]QualityIssue)
	for _, issue := range allIssues {
		categoryIssues[issue.Category] = append(categoryIssues[issue.Category], issue)
	}

	// Generate recommendations for categories with most issues
	for cat, issues := range categoryIssues {
		if len(issues) >= 2 {
			priority := "medium"
			if len(issues) >= 5 {
				priority = "high"
			}

			var files []string
			for _, issue := range issues[:min(5, len(issues))] {
				if issue.FilePath != "" && !contains(files, issue.FilePath) {
					files = append(files, issue.FilePath)
				}
			}

			result.Recommendations = append(result.Recommendations, Recommendation{
				Category:    cat,
				Priority:    priority,
				Title:       fmt.Sprintf("Improve %s", cat),
				Description: fmt.Sprintf("Found %d issues related to %s across multiple files", len(issues), cat),
				FilePaths:   files,
				Impact:      fmt.Sprintf("+%d points estimated", len(issues)*2),
			})
		}
	}

	// Sort by priority
	sort.Slice(result.Recommendations, func(i, j int) bool {
		return priorityOrder(result.Recommendations[i].Priority) > priorityOrder(result.Recommendations[j].Priority)
	})
}

// buildSummary builds the quality summary
func (a *Analyzer) buildSummary(result *QualityScore) {
	result.Summary.TotalFiles = len(result.FileScores)

	var allIssues []QualityIssue
	categoryStats := make(map[ScoreCategory][]int)

	for _, fs := range result.FileScores {
		result.Summary.TotalLinesOfCode += fs.LinesOfCode
		result.Summary.TotalFunctions += fs.FunctionsCount
		allIssues = append(allIssues, fs.Issues...)

		for cat, score := range fs.Breakdown {
			categoryStats[cat] = append(categoryStats[cat], score)
		}
	}

	result.Summary.IssuesCount = len(allIssues)

	for _, issue := range allIssues {
		switch issue.Severity {
		case "high":
			result.Summary.HighSeverityCount++
		case "medium":
			result.Summary.MediumSeverityCount++
		case "low":
			result.Summary.LowSeverityCount++
		}
	}

	// Top issue categories
	catCounts := make(map[ScoreCategory]int)
	for _, issue := range allIssues {
		catCounts[issue.Category]++
	}

	for cat, count := range catCounts {
		avg := 0
		if scores, ok := categoryStats[cat]; ok && len(scores) > 0 {
			sum := 0
			for _, s := range scores {
				sum += s
			}
			avg = sum / len(scores)
		}
		result.Summary.TopIssueCategories = append(result.Summary.TopIssueCategories, CategoryStat{
			Category: cat,
			Count:    count,
			AvgScore: avg,
		})
	}

	sort.Slice(result.Summary.TopIssueCategories, func(i, j int) bool {
		return result.Summary.TopIssueCategories[i].Count > result.Summary.TopIssueCategories[j].Count
	})

	// Best/Worst files
	sortedFiles := make([]FileScore, len(result.FileScores))
	copy(sortedFiles, result.FileScores)
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].Score > sortedFiles[j].Score
	})

	for i := 0; i < min(3, len(sortedFiles)); i++ {
		result.Summary.BestScoringFiles = append(result.Summary.BestScoringFiles, sortedFiles[i].FilePath)
	}

	for i := len(sortedFiles) - 1; i >= max(0, len(sortedFiles)-3); i-- {
		result.Summary.WorstScoringFiles = append(result.Summary.WorstScoringFiles, sortedFiles[i].FilePath)
	}

	// Build test coverage estimation statistics
	for _, fs := range result.FileScores {
		if fs.IsTestFile {
			result.Summary.TestFilesCount++
		} else {
			result.Summary.SourceFilesCount++
			result.Summary.TotalTestableFunctions += fs.TestableFunctions
			result.Summary.EstimatedTestedFunctions += fs.TestedFunctionsEstimate
		}
	}

	// Calculate estimated coverage percentage
	if result.Summary.TotalTestableFunctions > 0 {
		result.Summary.EstimatedCoveragePercent = float64(result.Summary.EstimatedTestedFunctions) / float64(result.Summary.TotalTestableFunctions) * 100
	}
}

// Helper functions

func isCodeFile(path string) bool {
	extensions := []string{".go", ".js", ".ts", ".tsx", ".py", ".rs", ".java", ".cpp", ".c", ".h", ".rb", ".php", ".swift", ".kt", ".cs"}
	for _, ext := range extensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func detectLanguage(path string) string {
	switch {
	case strings.HasSuffix(path, ".go"):
		return "go"
	case strings.HasSuffix(path, ".ts"), strings.HasSuffix(path, ".tsx"):
		return "typescript"
	case strings.HasSuffix(path, ".js"):
		return "javascript"
	case strings.HasSuffix(path, ".py"):
		return "python"
	case strings.HasSuffix(path, ".rs"):
		return "rust"
	case strings.HasSuffix(path, ".java"):
		return "java"
	case strings.HasSuffix(path, ".cpp"), strings.HasSuffix(path, ".c"), strings.HasSuffix(path, ".h"):
		return "c++"
	default:
		return "unknown"
	}
}

func extractJSON(s string) string {
	// Find first { and last }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

func priorityOrder(p string) int {
	switch p {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ============================================================================
// Code Duplication Detection (v4.1.0+)
// ============================================================================

// DetectDuplication analyzes code for duplicate patterns using LLM.
func (a *Analyzer) DetectDuplication(ctx context.Context, req AnalysisRequest) (*DuplicationResult, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("duplication-%s", time.Now().Format("20060102-150405"))

	a.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
		"model":      req.ModelID,
		"scan_id":    scanID,
	}).Info("Starting code duplication detection")

	result := &DuplicationResult{
		ProjectID:  req.ProjectID,
		ScanID:     scanID,
		ScannedAt:  startTime.Format(time.RFC3339),
		Status:     "running",
		Duplicates: []DuplicateGroup{},
		ModelID:    req.ModelID,
	}

	maxFiles := req.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 30 // Lower default for duplication detection
	}

	// Get code files
	files, err := a.getCodeFiles(ctx, req.CollectionName, req.ProjectID, maxFiles)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Failed to get code files: %v", err)
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	if len(files) < 2 {
		result.Status = "completed"
		result.Error = "Not enough files for duplication detection"
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	a.logger.WithField("files", len(files)).Debug("Analyzing files for duplication")

	// Build code summary for LLM analysis
	var codeSnippets []string
	for i, file := range files {
		content := file.Content
		if len(content) > 3000 {
			content = content[:3000] + "\n// ... (truncated)"
		}
		codeSnippets = append(codeSnippets, fmt.Sprintf("=== FILE %d: %s ===\n%s", i+1, file.Path, content))
	}

	allCode := strings.Join(codeSnippets, "\n\n")
	if len(allCode) > 50000 {
		allCode = allCode[:50000] + "\n... (truncated)"
	}

	prompt := fmt.Sprintf(`Analyze these code files for duplicate or similar code patterns.

%s

Find code duplications across files. For each duplication found:
1. Identify similar/duplicate code blocks (at least 5 lines)
2. Note all occurrences with file numbers and approximate lines
3. Describe what the duplicate code does
4. Suggest how to refactor (extract function, create shared module, etc.)

Respond in JSON format ONLY:
{
  "duplicates": [
    {
      "id": "dup1",
      "occurrences": [
        {"file_index": 1, "file_path": "path/to/file1.go", "start_line": 10, "end_line": 25, "snippet": "func similar()..."},
        {"file_index": 2, "file_path": "path/to/file2.go", "start_line": 45, "end_line": 60, "snippet": "func similar()..."}
      ],
      "lines_count": 15,
      "similarity": 0.85,
      "description": "Both functions implement the same validation logic",
      "suggestion": "Extract to a shared validation package",
      "refactor_type": "extract_function"
    }
  ],
  "summary": {
    "total_duplicate_groups": 1,
    "files_with_duplicates": 2,
    "estimated_duplicate_lines": 30
  }
}

If no duplicates found, return {"duplicates": [], "summary": {"total_duplicate_groups": 0, "files_with_duplicates": 0, "estimated_duplicate_lines": 0}}`, allCode)

	response, tokens, err := a.callLLM(ctx, req.ModelID, prompt)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("LLM call failed: %v", err)
		result.Duration = time.Since(startTime).String()
		return result, err
	}
	result.TokensUsed = tokens

	// Parse response
	var parsed struct {
		Duplicates []struct {
			ID          string `json:"id"`
			Occurrences []struct {
				FileIndex int    `json:"file_index"`
				FilePath  string `json:"file_path"`
				StartLine int    `json:"start_line"`
				EndLine   int    `json:"end_line"`
				Snippet   string `json:"snippet"`
			} `json:"occurrences"`
			LinesCount   int     `json:"lines_count"`
			Similarity   float64 `json:"similarity"`
			Description  string  `json:"description"`
			Suggestion   string  `json:"suggestion"`
			RefactorType string  `json:"refactor_type"`
		} `json:"duplicates"`
		Summary struct {
			TotalDuplicateGroups    int `json:"total_duplicate_groups"`
			FilesWithDuplicates     int `json:"files_with_duplicates"`
			EstimatedDuplicateLines int `json:"estimated_duplicate_lines"`
		} `json:"summary"`
	}

	jsonStr := extractJSON(response)
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		a.logger.WithError(err).Debug("Failed to parse duplication response")
		result.Status = "completed"
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	// Convert parsed to result
	filesWithDuplicates := make(map[string]bool)
	for _, dup := range parsed.Duplicates {
		group := DuplicateGroup{
			ID:           dup.ID,
			LinesCount:   dup.LinesCount,
			Similarity:   dup.Similarity,
			Description:  dup.Description,
			Suggestion:   dup.Suggestion,
			RefactorType: dup.RefactorType,
			Occurrences:  []DuplicateBlock{},
		}

		for _, occ := range dup.Occurrences {
			filePath := occ.FilePath
			if filePath == "" && occ.FileIndex > 0 && occ.FileIndex <= len(files) {
				filePath = files[occ.FileIndex-1].Path
			}
			filesWithDuplicates[filePath] = true

			group.Occurrences = append(group.Occurrences, DuplicateBlock{
				FilePath:    filePath,
				StartLine:   occ.StartLine,
				EndLine:     occ.EndLine,
				CodeSnippet: occ.Snippet,
			})
		}

		result.Duplicates = append(result.Duplicates, group)
	}

	// Build summary
	result.Summary.TotalFilesAnalyzed = len(files)
	result.Summary.FilesWithDuplicates = len(filesWithDuplicates)
	result.Summary.TotalDuplicateGroups = len(result.Duplicates)
	result.Summary.TotalDuplicateLines = parsed.Summary.EstimatedDuplicateLines

	totalLines := 0
	for _, f := range files {
		totalLines += f.Lines
	}
	if totalLines > 0 {
		result.Summary.DuplicationPercent = float64(result.Summary.TotalDuplicateLines) / float64(totalLines) * 100
	}

	// Top duplicated files
	for fp := range filesWithDuplicates {
		result.Summary.TopDuplicatedFiles = append(result.Summary.TopDuplicatedFiles, fp)
		if len(result.Summary.TopDuplicatedFiles) >= 5 {
			break
		}
	}

	result.Status = "completed"
	result.Duration = time.Since(startTime).String()

	a.logger.WithFields(logrus.Fields{
		"project_id":       req.ProjectID,
		"scan_id":          scanID,
		"duplicate_groups": len(result.Duplicates),
		"files_analyzed":   len(files),
		"tokens":           tokens,
	}).Info("Code duplication detection completed")

	return result, nil
}
