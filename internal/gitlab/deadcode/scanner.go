// Package deadcode provides dead code detection functionality.
package deadcode

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

// Scanner detects dead code in the codebase using RAG and LLM.
type Scanner struct {
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	httpClient  *http.Client
	logger      *logrus.Logger
}

// NewScanner creates a new dead code scanner.
func NewScanner(vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *Scanner {
	return &Scanner{
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Minute,
		},
		logger: logger,
	}
}

// Scan performs dead code detection on a project.
func (s *Scanner) Scan(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("deadcode-%s", time.Now().Format("20060102-150405"))

	s.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
		"model":      req.ModelID,
		"scan_id":    scanID,
	}).Info("Starting dead code detection")

	result := &ScanResult{
		ProjectID:   req.ProjectID,
		ScanID:      scanID,
		ScannedAt:   startTime,
		Status:      "running",
		DeadSymbols: []DeadSymbol{},
		Summary: ScanSummary{
			ByType:       make(map[SymbolType]int),
			ByConfidence: make(map[Confidence]int),
		},
		ModelID: req.ModelID,
	}

	// Set default max chunks
	maxChunks := req.MaxChunks
	if maxChunks <= 0 {
		maxChunks = 100
	}

	// Step 1: Collect all code chunks
	chunks, err := s.collectChunks(ctx, req.CollectionName, req.ProjectID, maxChunks)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Failed to collect chunks: %v", err)
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	result.ChunksScanned = len(chunks)

	// Count unique files
	fileSet := make(map[string]bool)
	for _, chunk := range chunks {
		fileSet[chunk.FilePath] = true
	}
	result.FilesScanned = len(fileSet)

	s.logger.WithFields(logrus.Fields{
		"chunks": len(chunks),
		"files":  len(fileSet),
	}).Debug("Collected chunks for analysis")

	if len(chunks) == 0 {
		result.Status = "completed"
		result.Error = "No code chunks found in index"
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	// Step 2: Extract symbols from chunks using patterns
	symbols := s.extractSymbols(chunks)

	s.logger.WithField("symbols", len(symbols)).Debug("Extracted symbols")

	// Step 3: For each symbol, search for usages in the codebase
	var deadSymbols []DeadSymbol
	totalTokens := 0

	for _, sym := range symbols {
		// Search for usages using RAG
		usages, err := s.searchUsages(ctx, req.CollectionName, req.ProjectID, sym)
		if err != nil {
			s.logger.WithError(err).WithField("symbol", sym.Name).Debug("Failed to search usages")
			continue
		}

		// If no usages found (except definition), mark as potentially dead
		if len(usages) <= 1 {
			// Use LLM to confirm
			isDead, confidence, reason, tokens, err := s.confirmWithLLM(ctx, req.ModelID, sym, chunks)
			if err != nil {
				s.logger.WithError(err).WithField("symbol", sym.Name).Debug("LLM confirmation failed")
				continue
			}
			totalTokens += tokens

			if isDead {
				sym.Confidence = confidence
				sym.Reason = reason
				deadSymbols = append(deadSymbols, sym)
			}
		}
	}

	result.DeadSymbols = deadSymbols
	result.TokensUsed = totalTokens

	// Build summary
	s.buildSummary(result)

	result.Status = "completed"
	result.Duration = time.Since(startTime).String()

	s.logger.WithFields(logrus.Fields{
		"project_id":   req.ProjectID,
		"scan_id":      scanID,
		"dead_symbols": len(deadSymbols),
		"tokens":       totalTokens,
		"duration":     result.Duration,
	}).Info("Dead code detection completed")

	return result, nil
}

// codeChunk represents a chunk of code
type codeChunk struct {
	FilePath  string
	Content   string
	Language  string
	StartLine int
	EndLine   int
}

// collectChunks retrieves code chunks from the vector store
func (s *Scanner) collectChunks(ctx context.Context, collection, projectID string, maxChunks int) ([]codeChunk, error) {
	var chunks []codeChunk

	err := s.vectorStore.ScrollAll(ctx, collection, map[string]any{
		"project_id": projectID,
	}, func(docs []vector.VectorDocument) error {
		for _, doc := range docs {
			if len(chunks) >= maxChunks {
				return nil
			}

			filePath, _ := doc.Metadata["file_path"].(string)
			if filePath == "" {
				continue
			}

			// Only process code files
			if !isCodeFile(filePath) {
				continue
			}

			content := doc.Text
			if c, ok := doc.Metadata["content"].(string); ok && c != "" {
				content = c
			}

			// Extract line numbers from metadata
			startLine := 1
			endLine := 1
			if sl, ok := doc.Metadata["start_line"].(float64); ok {
				startLine = int(sl)
			}
			if el, ok := doc.Metadata["end_line"].(float64); ok {
				endLine = int(el)
			}
			// Fallback: estimate from content
			if endLine <= startLine {
				lines := strings.Count(content, "\n") + 1
				endLine = startLine + lines - 1
			}

			chunks = append(chunks, codeChunk{
				FilePath:  filePath,
				Content:   content,
				Language:  detectLanguage(filePath),
				StartLine: startLine,
				EndLine:   endLine,
			})
		}
		return nil
	})

	return chunks, err
}

// extractSymbols extracts function/type/variable definitions from code
func (s *Scanner) extractSymbols(chunks []codeChunk) []DeadSymbol {
	var symbols []DeadSymbol

	// Patterns for different languages
	goFuncPattern := regexp.MustCompile(`(?m)^func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(`)
	goTypePattern := regexp.MustCompile(`(?m)^type\s+(\w+)\s+`)
	goConstPattern := regexp.MustCompile(`(?m)^const\s+(\w+)\s*=`)
	goVarPattern := regexp.MustCompile(`(?m)^var\s+(\w+)\s+`)

	jsFuncPattern := regexp.MustCompile(`(?m)^(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(`)
	jsClassPattern := regexp.MustCompile(`(?m)^(?:export\s+)?class\s+(\w+)`)
	jsConstPattern := regexp.MustCompile(`(?m)^(?:export\s+)?const\s+(\w+)\s*=`)

	pyFuncPattern := regexp.MustCompile(`(?m)^def\s+(\w+)\s*\(`)
	pyClassPattern := regexp.MustCompile(`(?m)^class\s+(\w+)`)

	for _, chunk := range chunks {
		var patterns []*regexp.Regexp
		var types []SymbolType

		switch chunk.Language {
		case "go":
			patterns = []*regexp.Regexp{goFuncPattern, goTypePattern, goConstPattern, goVarPattern}
			types = []SymbolType{SymbolFunction, SymbolType_, SymbolConstant, SymbolVariable}
		case "javascript", "typescript":
			patterns = []*regexp.Regexp{jsFuncPattern, jsClassPattern, jsConstPattern}
			types = []SymbolType{SymbolFunction, SymbolClass, SymbolConstant}
		case "python":
			patterns = []*regexp.Regexp{pyFuncPattern, pyClassPattern}
			types = []SymbolType{SymbolFunction, SymbolClass}
		default:
			continue
		}

		for i, pattern := range patterns {
			matches := pattern.FindAllStringSubmatch(chunk.Content, -1)
			for _, match := range matches {
				if len(match) > 1 {
					name := match[1]
					// Skip unexported Go symbols (lowercase first letter)
					isExported := strings.ToUpper(name[:1]) == name[:1]

					// Skip common test/main functions
					if name == "main" || strings.HasPrefix(name, "Test") || strings.HasPrefix(name, "Benchmark") {
						continue
					}

					// Count lines for the symbol (rough estimate)
					lineCount := 1

					symbols = append(symbols, DeadSymbol{
						Name:        name,
						Type:        types[i],
						FilePath:    chunk.FilePath,
						LinesOfCode: lineCount,
						Exportable:  isExported,
					})
				}
			}
		}
	}

	return symbols
}

// searchUsages searches for usages of a symbol in the codebase
func (s *Scanner) searchUsages(ctx context.Context, collection, projectID string, sym DeadSymbol) ([]string, error) {
	// Search for the symbol name in the vector store
	// This is a simplified approach - we just count occurrences

	var usages []string

	err := s.vectorStore.ScrollAll(ctx, collection, map[string]any{
		"project_id": projectID,
	}, func(docs []vector.VectorDocument) error {
		for _, doc := range docs {
			content := doc.Text
			if c, ok := doc.Metadata["content"].(string); ok && c != "" {
				content = c
			}

			// Check if symbol name appears in content
			// Using word boundary regex to avoid partial matches
			pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(sym.Name) + `\b`)
			if pattern.MatchString(content) {
				filePath, _ := doc.Metadata["file_path"].(string)
				usages = append(usages, filePath)
			}
		}
		return nil
	})

	return usages, err
}

// confirmWithLLM uses LLM to confirm if a symbol is truly dead code
func (s *Scanner) confirmWithLLM(ctx context.Context, modelID string, sym DeadSymbol, chunks []codeChunk) (bool, Confidence, string, int, error) {
	// Find the chunk containing this symbol
	var relevantContent string
	for _, chunk := range chunks {
		if chunk.FilePath == sym.FilePath && strings.Contains(chunk.Content, sym.Name) {
			relevantContent = chunk.Content
			break
		}
	}

	if relevantContent == "" {
		return false, "", "", 0, fmt.Errorf("symbol content not found")
	}

	// Truncate if too long
	if len(relevantContent) > 3000 {
		relevantContent = relevantContent[:3000] + "..."
	}

	prompt := fmt.Sprintf(`Analyze if this %s named "%s" appears to be dead code (unused).

File: %s
Code context:
%s

Consider:
1. Is this function/type called or referenced elsewhere?
2. Could it be used via reflection or dynamic dispatch?
3. Is it an exported API that might be used externally?
4. Is it a test helper, interface implementation, or handler?

Respond in JSON only:
{
  "is_dead": true/false,
  "confidence": "high"/"medium"/"low",
  "reason": "Brief explanation"
}`, sym.Type, sym.Name, sym.FilePath, relevantContent)

	response, tokens, err := s.callLLM(ctx, modelID, prompt)
	if err != nil {
		return false, "", "", tokens, err
	}

	// Parse response
	var parsed struct {
		IsDead     bool   `json:"is_dead"`
		Confidence string `json:"confidence"`
		Reason     string `json:"reason"`
	}

	jsonStr := extractJSON(response)
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return false, "", "", tokens, err
	}

	return parsed.IsDead, Confidence(parsed.Confidence), parsed.Reason, tokens, nil
}

// callLLM makes a request to the LLM API
func (s *Scanner) callLLM(ctx context.Context, modelID, prompt string) (string, int, error) {
	reqBody := map[string]any{
		"model": modelID,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a code analyzer. Determine if code is unused. Respond in JSON only."},
			{"role": "user", "content": prompt},
		},
		"max_tokens":  500,
		"temperature": 0.1,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.llmBaseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	if s.llmAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.llmAPIKey)
	}

	resp, err := s.httpClient.Do(req)
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

	return llmResponse.Choices[0].Message.Content, llmResponse.Usage.TotalTokens, nil
}

// buildSummary builds the scan summary
func (s *Scanner) buildSummary(result *ScanResult) {
	fileStats := make(map[string]*FileStats)

	for _, sym := range result.DeadSymbols {
		result.Summary.ByType[sym.Type]++
		result.Summary.ByConfidence[sym.Confidence]++
		result.Summary.EstimatedDeadLines += sym.LinesOfCode

		if _, ok := fileStats[sym.FilePath]; !ok {
			fileStats[sym.FilePath] = &FileStats{FilePath: sym.FilePath}
		}
		fileStats[sym.FilePath].DeadSymbols++
		fileStats[sym.FilePath].DeadLines += sym.LinesOfCode
	}

	result.Summary.TotalDeadSymbols = len(result.DeadSymbols)

	// Top affected files
	var files []*FileStats
	for _, fs := range fileStats {
		files = append(files, fs)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].DeadSymbols > files[j].DeadSymbols
	})

	for i := 0; i < min(5, len(files)); i++ {
		result.Summary.TopAffectedFiles = append(result.Summary.TopAffectedFiles, *files[i])
	}
}

// Helper functions

func isCodeFile(path string) bool {
	extensions := []string{".go", ".js", ".ts", ".tsx", ".py", ".rs", ".java", ".cpp", ".c", ".h"}
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

func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}

// ============================================================================
// Unreachable Code Detection (v4.1.0+)
// ============================================================================

// ScanUnreachable detects unreachable code blocks using LLM analysis.
func (s *Scanner) ScanUnreachable(ctx context.Context, req ScanRequest) (*UnreachableCodeResult, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("unreachable-%s", time.Now().Format("20060102-150405"))

	s.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
		"model":      req.ModelID,
		"scan_id":    scanID,
	}).Info("Starting unreachable code detection")

	result := &UnreachableCodeResult{
		ProjectID: req.ProjectID,
		ScanID:    scanID,
		ScannedAt: startTime.Format(time.RFC3339),
		Status:    "running",
		Blocks:    []UnreachableCodeBlock{},
		Summary: UnreachableSummary{
			ByConfidence: make(map[Confidence]int),
		},
		ModelID: req.ModelID,
	}

	maxChunks := req.MaxChunks
	if maxChunks <= 0 {
		maxChunks = 50
	}

	// Collect chunks
	chunks, err := s.collectChunks(ctx, req.CollectionName, req.ProjectID, maxChunks)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Failed to collect chunks: %v", err)
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	if len(chunks) == 0 {
		result.Status = "completed"
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	// Analyze chunks for unreachable code
	totalTokens := 0
	affectedFiles := make(map[string]bool)

	for _, chunk := range chunks {
		content := chunk.Content
		if len(content) > 5000 {
			content = content[:5000]
		}

		prompt := fmt.Sprintf(`Analyze this %s code for UNREACHABLE code blocks.

File: %s
Lines: %d-%d

Code:
%s

Identify code that will NEVER be executed due to:
1. Code after return/break/continue/throw statements
2. Dead branches in conditionals (e.g., if (false) {...})
3. Code after infinite loops
4. Unreachable catch/except blocks
5. Never-called private functions in the same scope

Respond in JSON format ONLY:
{
  "unreachable_blocks": [
    {
      "start_line": 15,
      "end_line": 20,
      "code_snippet": "// code here",
      "reason": "Code after return statement",
      "confidence": "high"
    }
  ]
}

If no unreachable code found, return: {"unreachable_blocks": []}`, chunk.Language, chunk.FilePath, chunk.StartLine, chunk.EndLine, content)

		response, tokens, err := s.callLLM(ctx, req.ModelID, prompt)
		totalTokens += tokens
		if err != nil {
			s.logger.WithError(err).WithField("file", chunk.FilePath).Debug("LLM call failed")
			continue
		}

		var parsed struct {
			UnreachableBlocks []struct {
				StartLine   int    `json:"start_line"`
				EndLine     int    `json:"end_line"`
				CodeSnippet string `json:"code_snippet"`
				Reason      string `json:"reason"`
				Confidence  string `json:"confidence"`
			} `json:"unreachable_blocks"`
		}

		if err := json.Unmarshal([]byte(extractJSON(response)), &parsed); err != nil {
			continue
		}

		for _, block := range parsed.UnreachableBlocks {
			conf := Confidence(block.Confidence)
			if conf != ConfidenceHigh && conf != ConfidenceMedium && conf != ConfidenceLow {
				conf = ConfidenceMedium
			}

			result.Blocks = append(result.Blocks, UnreachableCodeBlock{
				FilePath:    chunk.FilePath,
				StartLine:   chunk.StartLine + block.StartLine - 1,
				EndLine:     chunk.StartLine + block.EndLine - 1,
				CodeSnippet: block.CodeSnippet,
				Reason:      block.Reason,
				Confidence:  conf,
			})
			result.Summary.ByConfidence[conf]++
			affectedFiles[chunk.FilePath] = true
		}
	}

	// Build summary
	result.TokensUsed = totalTokens
	result.Summary.TotalBlocks = len(result.Blocks)
	result.Summary.AffectedFiles = len(affectedFiles)
	for _, b := range result.Blocks {
		result.Summary.TotalLines += (b.EndLine - b.StartLine + 1)
	}
	for fp := range affectedFiles {
		result.Summary.TopAffectedFiles = append(result.Summary.TopAffectedFiles, fp)
		if len(result.Summary.TopAffectedFiles) >= 5 {
			break
		}
	}

	result.Status = "completed"
	result.Duration = time.Since(startTime).String()

	s.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"blocks":     len(result.Blocks),
		"tokens":     totalTokens,
	}).Info("Unreachable code detection completed")

	return result, nil
}

// ============================================================================
// Commented-Out Code Detection (v4.1.0+)
// ============================================================================

// ScanCommentedCode detects commented-out code blocks.
func (s *Scanner) ScanCommentedCode(ctx context.Context, req ScanRequest) (*CommentedCodeResult, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("commented-%s", time.Now().Format("20060102-150405"))

	s.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
		"model":      req.ModelID,
		"scan_id":    scanID,
	}).Info("Starting commented-out code detection")

	result := &CommentedCodeResult{
		ProjectID: req.ProjectID,
		ScanID:    scanID,
		ScannedAt: startTime.Format(time.RFC3339),
		Status:    "running",
		Blocks:    []CommentedCodeBlock{},
		Summary:   CommentedCodeSummary{},
		ModelID:   req.ModelID,
	}

	maxChunks := req.MaxChunks
	if maxChunks <= 0 {
		maxChunks = 50
	}

	// Collect chunks
	chunks, err := s.collectChunks(ctx, req.CollectionName, req.ProjectID, maxChunks)
	if err != nil {
		result.Status = "failed"
		result.Error = fmt.Sprintf("Failed to collect chunks: %v", err)
		result.Duration = time.Since(startTime).String()
		return result, err
	}

	if len(chunks) == 0 {
		result.Status = "completed"
		result.Duration = time.Since(startTime).String()
		return result, nil
	}

	// Analyze chunks for commented-out code
	totalTokens := 0
	affectedFiles := make(map[string]bool)

	for _, chunk := range chunks {
		content := chunk.Content
		if len(content) > 5000 {
			content = content[:5000]
		}

		prompt := fmt.Sprintf(`Analyze this %s code for COMMENTED-OUT CODE blocks.

File: %s
Lines: %d-%d

Code:
%s

Identify blocks of COMMENTED-OUT CODE (not regular comments/documentation).
Look for:
1. Multiple consecutive lines of commented code
2. Commented function/method definitions
3. Commented conditional blocks or loops
4. TODO/FIXME comments with actual code disabled
5. Debug code that was commented instead of removed

DO NOT flag:
- Regular documentation comments
- License headers
- TODO comments without code
- Single-line explanatory comments

Respond in JSON format ONLY:
{
  "commented_code_blocks": [
    {
      "start_line": 25,
      "end_line": 35,
      "content": "// func oldFunction() {...}",
      "lines_count": 10,
      "confidence": "high",
      "reason": "Commented function definition"
    }
  ]
}

If no commented-out code found, return: {"commented_code_blocks": []}`, chunk.Language, chunk.FilePath, chunk.StartLine, chunk.EndLine, content)

		response, tokens, err := s.callLLM(ctx, req.ModelID, prompt)
		totalTokens += tokens
		if err != nil {
			s.logger.WithError(err).WithField("file", chunk.FilePath).Debug("LLM call failed")
			continue
		}

		var parsed struct {
			CommentedCodeBlocks []struct {
				StartLine  int    `json:"start_line"`
				EndLine    int    `json:"end_line"`
				Content    string `json:"content"`
				LinesCount int    `json:"lines_count"`
				Confidence string `json:"confidence"`
				Reason     string `json:"reason"`
			} `json:"commented_code_blocks"`
		}

		if err := json.Unmarshal([]byte(extractJSON(response)), &parsed); err != nil {
			continue
		}

		for _, block := range parsed.CommentedCodeBlocks {
			conf := Confidence(block.Confidence)
			if conf != ConfidenceHigh && conf != ConfidenceMedium && conf != ConfidenceLow {
				conf = ConfidenceMedium
			}

			result.Blocks = append(result.Blocks, CommentedCodeBlock{
				FilePath:   chunk.FilePath,
				StartLine:  chunk.StartLine + block.StartLine - 1,
				EndLine:    chunk.StartLine + block.EndLine - 1,
				Content:    block.Content,
				LinesCount: block.LinesCount,
				Confidence: conf,
				Reason:     block.Reason,
			})
			affectedFiles[chunk.FilePath] = true
		}
	}

	// Build summary
	result.TokensUsed = totalTokens
	result.Summary.TotalBlocks = len(result.Blocks)
	result.Summary.AffectedFiles = len(affectedFiles)
	for _, b := range result.Blocks {
		result.Summary.TotalLines += b.LinesCount
	}
	for fp := range affectedFiles {
		result.Summary.TopAffectedFiles = append(result.Summary.TopAffectedFiles, fp)
		if len(result.Summary.TopAffectedFiles) >= 5 {
			break
		}
	}

	result.Status = "completed"
	result.Duration = time.Since(startTime).String()

	s.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"blocks":     len(result.Blocks),
		"tokens":     totalTokens,
	}).Info("Commented-out code detection completed")

	return result, nil
}
