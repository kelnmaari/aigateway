// Package testgen provides automatic test generation functionality.
package testgen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"aigateway/internal/rag/vector"

	"github.com/sirupsen/logrus"
)

// Generator generates tests using LLM.
type Generator struct {
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	httpClient  *http.Client
	logger      *logrus.Logger
}

// NewGenerator creates a new test generator.
func NewGenerator(vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *Generator {
	return &Generator{
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Minute,
		},
		logger: logger,
	}
}

// Scan scans a project for testable functions.
func (g *Generator) Scan(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("testgen-%s", time.Now().Format("20060102-150405"))

	g.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
		"scan_id":    scanID,
	}).Info("Starting test generation scan")

	result := &ScanResult{
		ProjectID: req.ProjectID,
		ScanID:    scanID,
		ScannedAt: startTime,
		Status:    "running",
		Functions: []TestableFunction{},
		Summary: ScanSummary{
			ByLanguage:   make(map[string]int),
			ByComplexity: make(map[string]int),
		},
	}

	// Set defaults
	maxFiles := req.MaxFiles
	if maxFiles <= 0 {
		maxFiles = 50
	}

	// Collect code files
	files, err := g.collectCodeFiles(ctx, req.CollectionName, req.ProjectID, maxFiles)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Failed to collect files: %v", err)
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	result.FilesScanned = len(files)

	// Collect test files to check existing tests
	testFiles := g.findTestFiles(files)

	// Scan each file for functions
	for _, file := range files {
		// Skip test files themselves
		if g.isTestFile(file.Path) {
			continue
		}

		functions := g.scanFile(file, testFiles)
		result.Functions = append(result.Functions, functions...)
	}

	// Build summary
	g.buildSummary(result)

	result.Status = "completed"
	result.Duration = time.Since(startTime).String()

	g.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"scan_id":    scanID,
		"functions":  len(result.Functions),
		"duration":   result.Duration,
	}).Info("Test scan completed")

	return result, nil
}

// Generate generates tests for functions.
func (g *Generator) Generate(ctx context.Context, req GenerateRequest, functions []TestableFunction) (*GenerationResult, error) {
	startTime := time.Now()

	g.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"functions":  len(functions),
		"model":      req.ModelID,
	}).Info("Starting test generation")

	result := &GenerationResult{
		ProjectID:   req.ProjectID,
		GeneratedAt: startTime,
		Status:      "running",
		Tests:       []GeneratedTest{},
		ModelID:     req.ModelID,
	}

	// Limit functions
	maxFunctions := req.MaxFunctions
	if maxFunctions <= 0 {
		maxFunctions = 10
	}

	// Filter to only untested functions
	var untested []TestableFunction
	for _, fn := range functions {
		if !fn.HasTests {
			untested = append(untested, fn)
		}
	}

	if len(untested) > maxFunctions {
		untested = untested[:maxFunctions]
	}

	totalTokens := 0

	for _, fn := range untested {
		test, tokens, err := g.generateTest(ctx, req.ModelID, fn, req.Framework)
		if err != nil {
			g.logger.WithError(err).WithField("function", fn.Name).Warn("Failed to generate test")
			continue
		}
		totalTokens += tokens
		result.Tests = append(result.Tests, test)
	}

	result.TokensUsed = totalTokens
	result.Status = "completed"
	result.Duration = time.Since(startTime).String()

	g.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"generated":  len(result.Tests),
		"tokens":     totalTokens,
		"duration":   result.Duration,
	}).Info("Test generation completed")

	return result, nil
}

// codeFile represents a code file
type codeFile struct {
	Path     string
	Content  string
	Language string
}

// collectCodeFiles retrieves code files from vector store
func (g *Generator) collectCodeFiles(ctx context.Context, collection, projectID string, maxFiles int) ([]codeFile, error) {
	var files []codeFile
	seen := make(map[string]bool)

	err := g.vectorStore.ScrollAll(ctx, collection, map[string]interface{}{
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

			if seen[filePath] {
				continue
			}

			if !isCodeFile(filePath) {
				continue
			}

			seen[filePath] = true

			content := doc.Text
			if c, ok := doc.Metadata["content"].(string); ok && c != "" {
				content = c
			}

			files = append(files, codeFile{
				Path:     filePath,
				Content:  content,
				Language: detectLanguage(filePath),
			})
		}
		return nil
	})

	return files, err
}

// findTestFiles finds test file names
func (g *Generator) findTestFiles(files []codeFile) map[string]bool {
	testFiles := make(map[string]bool)
	for _, file := range files {
		if g.isTestFile(file.Path) {
			testFiles[file.Path] = true
		}
	}
	return testFiles
}

// isTestFile checks if a file is a test file
func (g *Generator) isTestFile(path string) bool {
	return strings.HasSuffix(path, "_test.go") ||
		strings.HasSuffix(path, ".test.ts") ||
		strings.HasSuffix(path, ".test.js") ||
		strings.HasSuffix(path, ".spec.ts") ||
		strings.HasSuffix(path, ".spec.js") ||
		strings.Contains(path, "test_") ||
		strings.Contains(path, "__tests__")
}

// scanFile scans a file for testable functions
func (g *Generator) scanFile(file codeFile, testFiles map[string]bool) []TestableFunction {
	var functions []TestableFunction

	switch file.Language {
	case "go":
		functions = g.scanGoFile(file, testFiles)
	case "javascript", "typescript":
		functions = g.scanJSFile(file, testFiles)
	case "python":
		functions = g.scanPythonFile(file, testFiles)
	}

	return functions
}

// scanGoFile scans a Go file
func (g *Generator) scanGoFile(file codeFile, testFiles map[string]bool) []TestableFunction {
	var functions []TestableFunction

	lines := strings.Split(file.Content, "\n")
	funcPattern := regexp.MustCompile(`^func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(([^)]*)\)(\s*\([^)]*\)|\s+\w+)?`)

	// Check for corresponding test file
	testFilePath := strings.TrimSuffix(file.Path, ".go") + "_test.go"
	hasTestFile := testFiles[testFilePath]

	for i, line := range lines {
		if matches := funcPattern.FindStringSubmatch(strings.TrimSpace(line)); len(matches) > 1 {
			name := matches[1]

			// Skip unexported, main, init
			if strings.ToLower(name[:1]) == name[:1] || name == "main" || name == "init" {
				continue
			}

			// Skip test functions
			if strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark") {
				continue
			}

			// Extract code (estimate 10 lines)
			endLine := min(i+10, len(lines))
			code := strings.Join(lines[i:endLine], "\n")

			// Estimate complexity
			complexity := g.estimateComplexity(code)

			// Check if test exists
			hasTests := hasTestFile // Simplified check

			functions = append(functions, TestableFunction{
				Name:       name,
				FilePath:   file.Path,
				StartLine:  i + 1,
				Language:   "go",
				Signature:  line,
				Code:       code,
				IsExported: true,
				HasTests:   hasTests,
				Complexity: complexity,
			})
		}
	}

	return functions
}

// scanJSFile scans JS/TS files
func (g *Generator) scanJSFile(file codeFile, testFiles map[string]bool) []TestableFunction {
	var functions []TestableFunction

	lines := strings.Split(file.Content, "\n")
	funcPattern := regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(([^)]*)\)`)
	arrowPattern := regexp.MustCompile(`^(?:export\s+)?(?:const|let)\s+(\w+)\s*=\s*(?:async\s+)?\([^)]*\)\s*=>`)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		var name string
		var signature string

		if matches := funcPattern.FindStringSubmatch(trimmed); len(matches) > 1 {
			name = matches[1]
			signature = trimmed
		} else if matches := arrowPattern.FindStringSubmatch(trimmed); len(matches) > 1 {
			name = matches[1]
			signature = trimmed
		}

		if name != "" {
			endLine := min(i+10, len(lines))
			code := strings.Join(lines[i:endLine], "\n")
			complexity := g.estimateComplexity(code)

			functions = append(functions, TestableFunction{
				Name:       name,
				FilePath:   file.Path,
				StartLine:  i + 1,
				Language:   file.Language,
				Signature:  signature,
				Code:       code,
				IsExported: strings.HasPrefix(trimmed, "export"),
				HasTests:   false,
				Complexity: complexity,
			})
		}
	}

	return functions
}

// scanPythonFile scans Python files
func (g *Generator) scanPythonFile(file codeFile, testFiles map[string]bool) []TestableFunction {
	var functions []TestableFunction

	lines := strings.Split(file.Content, "\n")
	funcPattern := regexp.MustCompile(`^def\s+(\w+)\s*\(([^)]*)\)`)

	for i, line := range lines {
		if matches := funcPattern.FindStringSubmatch(strings.TrimSpace(line)); len(matches) > 1 {
			name := matches[1]

			// Skip private and magic methods
			if strings.HasPrefix(name, "_") {
				continue
			}

			endLine := min(i+10, len(lines))
			code := strings.Join(lines[i:endLine], "\n")
			complexity := g.estimateComplexity(code)

			functions = append(functions, TestableFunction{
				Name:       name,
				FilePath:   file.Path,
				StartLine:  i + 1,
				Language:   "python",
				Signature:  line,
				Code:       code,
				IsExported: true,
				HasTests:   false,
				Complexity: complexity,
			})
		}
	}

	return functions
}

// estimateComplexity estimates function complexity
func (g *Generator) estimateComplexity(code string) string {
	// Count complexity indicators
	indicators := 0
	indicators += strings.Count(code, "if ")
	indicators += strings.Count(code, "for ")
	indicators += strings.Count(code, "switch ")
	indicators += strings.Count(code, "case ")
	indicators += strings.Count(code, "while ")
	indicators += strings.Count(code, "else")
	indicators += strings.Count(code, "&&")
	indicators += strings.Count(code, "||")

	if indicators <= 2 {
		return "low"
	} else if indicators <= 5 {
		return "medium"
	}
	return "high"
}

// generateTest generates a test for a function
func (g *Generator) generateTest(ctx context.Context, modelID string, fn TestableFunction, framework TestFramework) (GeneratedTest, int, error) {
	test := GeneratedTest{
		Function: fn,
		Language: fn.Language,
	}

	// Determine framework
	if framework == "" {
		switch fn.Language {
		case "go":
			framework = FrameworkGoTest
		case "javascript", "typescript":
			framework = FrameworkJest
		case "python":
			framework = FrameworkPytest
		}
	}
	test.Framework = framework

	prompt := g.buildPrompt(fn, framework)
	testCode, tokens, err := g.callLLM(ctx, modelID, prompt, fn.Language)
	if err != nil {
		return test, tokens, err
	}

	test.TestCode = testCode
	test.TestName = "Test" + fn.Name
	test.Description = fmt.Sprintf("Unit tests for %s function", fn.Name)

	return test, tokens, nil
}

// buildPrompt builds the LLM prompt
func (g *Generator) buildPrompt(fn TestableFunction, framework TestFramework) string {
	var frameworkInstructions string

	switch framework {
	case FrameworkGoTest:
		frameworkInstructions = `Generate Go unit tests using the testing package.
Rules:
- Use t.Run for subtests
- Include table-driven tests when appropriate  
- Test happy path and edge cases
- Use t.Errorf for assertions
- Optionally use testify/assert if complex assertions needed
- IMPORTANT: Do NOT use placeholder import paths like "yourapp/..." - use only standard library imports or leave a TODO comment for imports that need the actual module path
- If you need to import the package being tested, add a comment: // TODO: import "MODULE_PATH/package" - replace MODULE_PATH with actual go.mod module

Example format:
func TestFunctionName(t *testing.T) {
    tests := []struct{
        name string
        input  Type
        want   Type
    }{
        {"happy path", input, expected},
        {"edge case", input2, expected2},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := FunctionName(tt.input)
            if got != tt.want {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}`

	case FrameworkJest, FrameworkVitest:
		frameworkInstructions = `Generate Jest/Vitest unit tests.
Rules:
- Use describe/it blocks
- Use expect assertions
- Test happy path and edge cases
- Include beforeEach if needed

Example:
describe('functionName', () => {
    it('should return expected value', () => {
        expect(functionName(input)).toBe(expected);
    });
});`

	case FrameworkPytest:
		frameworkInstructions = `Generate pytest unit tests.
Rules:
- Use test_ prefix for functions
- Use assert statements
- Use @pytest.mark.parametrize for multiple cases
- Include edge cases

Example:
def test_function_name_happy_path():
    result = function_name(input)
    assert result == expected

@pytest.mark.parametrize("input,expected", [
    (case1_input, case1_expected),
    (case2_input, case2_expected),
])
def test_function_name_cases(input, expected):
    assert function_name(input) == expected`
	}

	return fmt.Sprintf(`%s

Generate tests for this function:

Function: %s
File: %s
Signature: %s

Code:
%s

Generate ONLY the test code. No explanations.
Include 2-4 test cases covering:
1. Happy path
2. Edge cases
3. Error cases (if applicable)`, frameworkInstructions, fn.Name, fn.FilePath, fn.Signature, fn.Code)
}

// callLLM makes a request to the LLM API
func (g *Generator) callLLM(ctx context.Context, modelID, prompt, language string) (string, int, error) {
	systemPrompt := fmt.Sprintf("You are a test engineer. Generate clean, comprehensive unit tests for %s code. Output ONLY the test code, no explanations.", language)

	reqBody := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"max_tokens":  1500,
		"temperature": 0.3,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.llmBaseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	if g.llmAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+g.llmAPIKey)
	}

	resp, err := g.httpClient.Do(req)
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
	}

	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		return "", 0, err
	}

	if len(llmResponse.Choices) == 0 {
		return "", 0, fmt.Errorf("no response from LLM")
	}

	return strings.TrimSpace(llmResponse.Choices[0].Message.Content), llmResponse.Usage.TotalTokens, nil
}

// buildSummary builds the scan summary
func (g *Generator) buildSummary(result *ScanResult) {
	fileStats := make(map[string]*FileStats)

	for _, fn := range result.Functions {
		result.Summary.ByLanguage[fn.Language]++
		result.Summary.ByComplexity[fn.Complexity]++

		if fn.HasTests {
			result.Summary.WithTests++
		} else {
			result.Summary.WithoutTests++
		}

		if _, ok := fileStats[fn.FilePath]; !ok {
			fileStats[fn.FilePath] = &FileStats{FilePath: fn.FilePath}
		}
		fileStats[fn.FilePath].FunctionCount++
		if !fn.HasTests {
			fileStats[fn.FilePath].Untested++
		}
	}

	result.Summary.TotalFunctions = len(result.Functions)

	// Top files by untested count
	var files []*FileStats
	for _, fs := range fileStats {
		files = append(files, fs)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Untested > files[j].Untested
	})

	for i := 0; i < min(10, len(files)); i++ {
		result.Summary.TopFiles = append(result.Summary.TopFiles, *files[i])
	}
}

// Helper functions

func isCodeFile(path string) bool {
	extensions := []string{".go", ".js", ".ts", ".tsx", ".py"}
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
	default:
		return "unknown"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
