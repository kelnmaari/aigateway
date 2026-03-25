// Package scanner provides deep semantic scanning using LLM.
package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"aigateway/internal/rag/vector"

	"github.com/sirupsen/logrus"
)

// DeepScanner uses LLM for semantic secrets detection
type DeepScanner struct {
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	httpClient  *http.Client
	logger      *logrus.Logger
}

// DeepScanRequest parameters for deep scanning
type DeepScanRequest struct {
	ProjectID      string `json:"project_id"`
	CollectionName string `json:"collection_name"`
	ModelID        string `json:"model_id"` // Analysis model alias
	MaxChunks      int    `json:"max_chunks,omitempty"`
	Language       string `json:"language,omitempty"` // "en" or "ru"
}

// ProgressCallback is called during scan to report progress
type ProgressCallback func(progress ScanProgress)

// ScanProgress represents current scan progress
type ScanProgress struct {
	ChunksScanned int    `json:"chunks_scanned"`
	TotalChunks   int    `json:"total_chunks"`
	FindingsCount int    `json:"findings_count"`
	CurrentBatch  int    `json:"current_batch"`
	TotalBatches  int    `json:"total_batches"`
	Status        string `json:"status"`
}

// DeepScanResult contains deep scan results
type DeepScanResult struct {
	ProjectID     string        `json:"project_id"`
	ScanID        string        `json:"scan_id"`
	ModelID       string        `json:"model_id"`
	StartedAt     time.Time     `json:"started_at"`
	CompletedAt   time.Time     `json:"completed_at"`
	Duration      string        `json:"duration"`
	ChunksScanned int           `json:"chunks_scanned"`
	TokensUsed    int           `json:"tokens_used"`
	Findings      []DeepFinding `json:"findings"`
	Summary       ScanSummary   `json:"summary"`
	Status        string        `json:"status"`
	Error         string        `json:"error,omitempty"`
}

// DeepFinding represents an LLM-detected security issue
type DeepFinding struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"` // "hardcoded_secret", "sensitive_data", "security_issue"
	Severity    Severity `json:"severity"`
	FilePath    string   `json:"file_path"`
	StartLine   int      `json:"start_line"`
	EndLine     int      `json:"end_line"`
	Description string   `json:"description"` // LLM's explanation
	CodeSnippet string   `json:"code_snippet"`
	Suggestion  string   `json:"suggestion"`
	Confidence  string   `json:"confidence"` // "high", "medium", "low"
}

// NewDeepScanner creates a new LLM-based scanner
func NewDeepScanner(vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *DeepScanner {
	return &DeepScanner{
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		httpClient:  &http.Client{Timeout: 15 * time.Minute}, // Increased from 3m
		logger:      logger,
	}
}

// DeepScan performs semantic analysis using LLM
func (s *DeepScanner) DeepScan(ctx context.Context, req DeepScanRequest) (*DeepScanResult, error) {
	return s.DeepScanWithProgress(ctx, req, nil)
}

// DeepScanWithProgress performs semantic analysis with progress callback
func (s *DeepScanner) DeepScanWithProgress(ctx context.Context, req DeepScanRequest, progressCb ProgressCallback) (*DeepScanResult, error) {
	startTime := time.Now()
	scanID := fmt.Sprintf("deep-%s", time.Now().Format("20060102-150405"))

	s.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
		"model":      req.ModelID,
		"scan_id":    scanID,
	}).Info("Starting deep secrets scan with LLM")

	result := &DeepScanResult{
		ProjectID: req.ProjectID,
		ScanID:    scanID,
		ModelID:   req.ModelID,
		StartedAt: startTime,
		Status:    "running",
		Findings:  []DeepFinding{},
		Summary: ScanSummary{
			BySeverity: make(map[string]int),
			ByCategory: make(map[string]int),
		},
	}

	if req.MaxChunks <= 0 {
		req.MaxChunks = 500 // Default limit to control costs
	}

	language := req.Language
	if language == "" {
		language = "en"
	}

	var (
		allFindings   []DeepFinding
		findingsMu    sync.Mutex
		totalTokens   int
		tokensMu      sync.Mutex
		chunksScanned int
		filesAffected = make(map[string]bool)
	)

	// Collect chunks (we'll process them in batches)
	// Skip test files and documentation to reduce false positives
	var allChunks []chunkData
	var skippedTestFiles, skippedDocFiles int
	err := s.vectorStore.ScrollAll(ctx, req.CollectionName, map[string]any{
		"project_id": req.ProjectID,
	}, func(docs []vector.VectorDocument) error {
		for _, doc := range docs {
			content := doc.Text
			if content == "" {
				if c, ok := doc.Metadata["content"].(string); ok {
					content = c
				}
			}
			if content == "" || len(content) < 50 {
				continue
			}

			filePath := ""
			if fp, ok := doc.Metadata["file_path"].(string); ok {
				filePath = fp
			}

			// Skip test files - they often contain fake secrets for testing
			if isTestFile(filePath) {
				skippedTestFiles++
				continue
			}

			// Skip documentation files - they often contain example secrets
			if isDocumentationFile(filePath) {
				skippedDocFiles++
				continue
			}

			startLine := 0
			if sl, ok := doc.Metadata["start_line"].(float64); ok {
				startLine = int(sl)
			}
			endLine := 0
			if el, ok := doc.Metadata["end_line"].(float64); ok {
				endLine = int(el)
			}

			allChunks = append(allChunks, chunkData{
				content:   content,
				filePath:  filePath,
				startLine: startLine,
				endLine:   endLine,
			})

			if len(allChunks) >= req.MaxChunks {
				return nil // Stop collecting
			}
		}
		return nil
	})

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	s.logger.WithFields(logrus.Fields{
		"chunks_collected":   len(allChunks),
		"skipped_test_files": skippedTestFiles,
		"skipped_doc_files":  skippedDocFiles,
	}).Info("Collected chunks for deep analysis (test/doc files filtered)")

	// Process chunks in batches (3 chunks per LLM call for better quality)
	// Smaller batches = more focused analysis = fewer hallucinations
	batchSize := 3
	totalBatches := (len(allChunks) + batchSize - 1) / batchSize
	currentBatch := 0

	for i := 0; i < len(allChunks); i += batchSize {
		select {
		case <-ctx.Done():
			result.Status = "cancelled"
			result.Error = "scan cancelled"
			return result, ctx.Err()
		default:
		}

		end := min(i+batchSize, len(allChunks))
		batch := allChunks[i:end]
		chunksScanned += len(batch)
		currentBatch++

		// Report progress via callback
		if progressCb != nil {
			progressCb(ScanProgress{
				ChunksScanned: chunksScanned,
				TotalChunks:   len(allChunks),
				FindingsCount: len(allFindings),
				CurrentBatch:  currentBatch,
				TotalBatches:  totalBatches,
				Status:        "scanning",
			})
		}

		// Analyze batch with LLM
		findings, tokens, err := s.analyzeBatch(ctx, req.ModelID, batch, language)
		if err != nil {
			s.logger.WithError(err).WithField("batch", i/batchSize).Warn("Batch analysis failed, continuing")
			continue
		}

		tokensMu.Lock()
		totalTokens += tokens
		tokensMu.Unlock()

		if len(findings) > 0 {
			findingsMu.Lock()
			for _, f := range findings {
				filesAffected[f.FilePath] = true
			}
			allFindings = append(allFindings, findings...)
			findingsMu.Unlock()
		}

		// Progress log every 10 batches
		if (i/batchSize)%10 == 0 {
			s.logger.WithFields(logrus.Fields{
				"progress":        fmt.Sprintf("%d/%d", chunksScanned, len(allChunks)),
				"findings_so_far": len(allFindings),
			}).Debug("Deep scan progress")
		}
	}

	// Build result
	result.Findings = allFindings
	result.ChunksScanned = chunksScanned
	result.TokensUsed = totalTokens
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(startTime).String()
	result.Status = "completed"

	result.Summary.TotalFindings = len(allFindings)
	result.Summary.FilesAffected = len(filesAffected)

	for _, f := range allFindings {
		result.Summary.BySeverity[string(f.Severity)]++
		result.Summary.ByCategory[f.Type]++
	}

	s.logger.WithFields(logrus.Fields{
		"project_id":     req.ProjectID,
		"scan_id":        scanID,
		"chunks_scanned": chunksScanned,
		"tokens_used":    totalTokens,
		"findings":       len(allFindings),
		"duration":       result.Duration,
	}).Info("Deep secrets scan completed")

	return result, nil
}

type chunkData struct {
	content   string
	filePath  string
	startLine int
	endLine   int
}

// analyzeBatch sends a batch of chunks to LLM for analysis
func (s *DeepScanner) analyzeBatch(ctx context.Context, modelID string, chunks []chunkData, language string) ([]DeepFinding, int, error) {
	// Build prompt
	var sb strings.Builder
	sb.WriteString("Analyze the following code chunks for security issues:\n\n")

	for i, chunk := range chunks {
		sb.WriteString(fmt.Sprintf("### Chunk %d\n", i+1))
		sb.WriteString(fmt.Sprintf("File: %s (lines %d-%d)\n", chunk.filePath, chunk.startLine, chunk.endLine))
		sb.WriteString("```\n")
		// Truncate very long chunks
		content := chunk.content
		if len(content) > 2000 {
			content = content[:2000] + "\n... (truncated)"
		}
		sb.WriteString(content)
		sb.WriteString("\n```\n\n")
	}

	systemPrompt := s.getSystemPrompt(language)

	requestBody := map[string]any{
		"model": modelID,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": sb.String()},
		},
		"temperature": 0.0,
		"max_tokens":  2000,
	}

	bodyBytes, _ := json.Marshal(requestBody)

	req, err := http.NewRequestWithContext(ctx, "POST", s.llmBaseURL+"/v1/chat/completions", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.llmAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.llmAPIKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

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
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&llmResponse); err != nil {
		return nil, 0, err
	}

	if llmResponse.Error != nil {
		return nil, 0, fmt.Errorf("LLM error: %s", llmResponse.Error.Message)
	}

	if len(llmResponse.Choices) == 0 {
		return nil, 0, fmt.Errorf("no choices in response")
	}

	// Parse findings from LLM response
	content := llmResponse.Choices[0].Message.Content
	findings := s.parseFindings(content, chunks)

	return findings, llmResponse.Usage.TotalTokens, nil
}

func (s *DeepScanner) getSystemPrompt(language string) string {
	if language == "ru" {
		return `Ты эксперт по безопасности кода. Твоя задача — найти ТОЛЬКО РЕАЛЬНЫЕ захардкоженные секреты.

## СТРОГИЕ КРИТЕРИИ — что ЯВЛЯЕТСЯ секретом:
- Реальные API ключи с валидным форматом (AKIA..., sk_live_..., ghp_..., glpat-...)
- Реальные пароли в коде (НЕ переменные окружения, НЕ плейсхолдеры)
- Приватные ключи (-----BEGIN RSA PRIVATE KEY-----)
- Connection strings с реальными credentials

## КРИТЕРИИ ИСКЛЮЧЕНИЯ — что НЕ является секретом:
- Переменные окружения: os.Getenv("API_KEY"), process.env.SECRET
- Плейсхолдеры: "your-api-key", "xxx", "changeme", "<YOUR_TOKEN>", "example"
- Тестовые данные: "test_key", "mock_token", "fake_password"
- Шаблоны конфигов: ${API_KEY}, {{.Secret}}, %SECRET%
- Пустые значения: "", '', nil, null
- Документация и комментарии с примерами
- Константы с именами (без значений): const API_KEY = ""

## ПРИМЕРЫ

### ✅ ЭТО СЕКРЕТ (reportuй):
{"chunk_index": 1, "type": "hardcoded_secret", "severity": "critical", "description": "AWS Access Key ID захардкожен", "code_snippet": "aws_key = \"AKIAIOSFODNN7REAL123\"", "confidence": "high"}

### ❌ НЕ СЕКРЕТ (НЕ reportuй):
- password = os.Getenv("DB_PASSWORD")  → переменная окружения
- API_KEY = "your-api-key-here"  → плейсхолдер
- token = "<INSERT_TOKEN>"  → шаблон
- secret = ""  → пустое значение
- // Example: api_key = "sk_test_xxx"  → комментарий/документация

## УРОВНИ CONFIDENCE:
- "high": 100% уверен что это реальный секрет (валидный формат, не плейсхолдер)
- "medium": похоже на секрет, но может быть тестовым
- "low": возможно секрет, требует ручной проверки

⚠️ КРИТИЧЕСКИ ВАЖНО:
1. Лучше пропустить сомнительный случай, чем создать false positive
2. Отвечай СТРОГО в JSON формате
3. Если секретов НЕТ, верни: {"findings": []}

Формат ответа:
{
  "findings": [
    {
      "chunk_index": 1,
      "type": "hardcoded_secret",
      "severity": "critical",
      "description": "Краткое описание",
      "code_snippet": "строка кода",
      "suggestion": "Как исправить",
      "confidence": "high"
    }
  ]
}`
	}

	return `You are a code security expert. Your task is to find ONLY REAL hardcoded secrets.

## STRICT CRITERIA — what IS a secret:
- Real API keys with valid format (AKIA..., sk_live_..., ghp_..., glpat-...)
- Real passwords in code (NOT environment variables, NOT placeholders)
- Private keys (-----BEGIN RSA PRIVATE KEY-----)
- Connection strings with real credentials

## EXCLUSION CRITERIA — what is NOT a secret:
- Environment variables: os.Getenv("API_KEY"), process.env.SECRET, ENV["KEY"]
- Placeholders: "your-api-key", "xxx", "changeme", "<YOUR_TOKEN>", "example"
- Test data: "test_key", "mock_token", "fake_password", "dummy_secret"
- Config templates: ${API_KEY}, {{.Secret}}, %SECRET%, $SECRET
- Empty values: "", '', nil, null, None
- Documentation and comments with examples
- Constants with names only (no values): const API_KEY = ""
- Base64-encoded placeholders or examples

## EXAMPLES

### ✅ THIS IS A SECRET (report it):
{"chunk_index": 1, "type": "hardcoded_secret", "severity": "critical", "description": "AWS Access Key ID hardcoded", "code_snippet": "aws_key = \"AKIAIOSFODNN7REAL123\"", "confidence": "high"}

### ❌ NOT A SECRET (do NOT report):
- password = os.Getenv("DB_PASSWORD")  → environment variable
- API_KEY = "your-api-key-here"  → placeholder
- token = "<INSERT_TOKEN>"  → template
- secret = ""  → empty value
- // Example: api_key = "sk_test_xxx"  → comment/documentation
- const ApiKey = config.Get("api_key")  → config lookup
- key := viper.GetString("secret")  → config library

## CONFIDENCE LEVELS:
- "high": 100% certain this is a real secret (valid format, not a placeholder)
- "medium": looks like a secret but could be test data
- "low": possibly a secret, requires manual review

## SEVERITY LEVELS:
- "critical": Production API keys, private keys, database passwords
- "high": OAuth tokens, service credentials
- "medium": Generic secrets that may or may not be sensitive
- "low": Internal IPs, non-sensitive configuration

⚠️ CRITICAL RULES:
1. When in doubt, DO NOT report — false negatives are better than false positives
2. Respond STRICTLY in JSON format
3. If NO secrets found, return: {"findings": []}
4. DO NOT report environment variable lookups as secrets
5. DO NOT report placeholder values as secrets

Response format:
{
  "findings": [
    {
      "chunk_index": 1,
      "type": "hardcoded_secret",
      "severity": "critical",
      "description": "Brief description",
      "code_snippet": "the code line",
      "suggestion": "How to fix",
      "confidence": "high"
    }
  ]
}`
}

func (s *DeepScanner) parseFindings(content string, chunks []chunkData) []DeepFinding {
	var findings []DeepFinding

	// Try to extract JSON from response
	content = strings.TrimSpace(content)
	originalContent := content

	// Handle markdown code blocks
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json") + 7
		end := strings.LastIndex(content, "```")
		if end > start {
			content = strings.TrimSpace(content[start:end])
		}
	} else if strings.Contains(content, "```") {
		start := strings.Index(content, "```") + 3
		end := strings.LastIndex(content, "```")
		if end > start {
			content = strings.TrimSpace(content[start:end])
		}
	}

	// Detect hallucinations - repetitive garbage patterns
	if s.isHallucination(content) {
		s.logger.WithField("content_preview", truncate(content, 100)).Warn("LLM returned hallucinated garbage, skipping batch")
		return findings
	}

	// Try to find JSON object boundaries
	if !strings.HasPrefix(content, "{") {
		if idx := strings.Index(content, "{"); idx >= 0 {
			content = content[idx:]
		}
	}
	if !strings.HasSuffix(content, "}") {
		if idx := strings.LastIndex(content, "}"); idx >= 0 {
			content = content[:idx+1]
		}
	}

	// Parse JSON
	var parsed struct {
		Findings []struct {
			ChunkIndex  int    `json:"chunk_index"`
			Type        string `json:"type"`
			Severity    string `json:"severity"`
			Description string `json:"description"`
			CodeSnippet string `json:"code_snippet"`
			Suggestion  string `json:"suggestion"`
			Confidence  string `json:"confidence"`
		} `json:"findings"`
	}

	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"content_preview": truncate(content, 200),
			"original_length": len(originalContent),
			"cleaned_length":  len(content),
		}).Warn("Failed to parse LLM findings JSON - model returned invalid format")
		return findings
	}

	for _, f := range parsed.Findings {
		// Validate chunk index
		chunkIdx := f.ChunkIndex - 1 // LLM uses 1-based
		if chunkIdx < 0 || chunkIdx >= len(chunks) {
			continue
		}

		chunk := chunks[chunkIdx]

		// Additional validation: skip findings that look like false positives
		if s.isLikelyFalsePositive(f.CodeSnippet, f.Description, f.Confidence) {
			s.logger.WithFields(logrus.Fields{
				"code_snippet": truncate(f.CodeSnippet, 100),
				"confidence":   f.Confidence,
				"file_path":    chunk.filePath,
			}).Debug("Skipping likely false positive finding")
			continue
		}

		finding := DeepFinding{
			ID:          fmt.Sprintf("deep-%d", time.Now().UnixNano()),
			Type:        f.Type,
			Severity:    Severity(f.Severity),
			FilePath:    chunk.filePath,
			StartLine:   chunk.startLine,
			EndLine:     chunk.endLine,
			Description: f.Description,
			CodeSnippet: truncate(f.CodeSnippet, 200),
			Suggestion:  f.Suggestion,
			Confidence:  f.Confidence,
		}

		// Validate severity
		switch finding.Severity {
		case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow, SeverityInfo:
			// OK
		default:
			finding.Severity = SeverityMedium
		}

		// Validate confidence - only accept high/medium confidence findings
		switch finding.Confidence {
		case "high", "medium":
			// OK
		case "low":
			// Skip low confidence findings to reduce false positives
			s.logger.WithFields(logrus.Fields{
				"code_snippet": truncate(f.CodeSnippet, 100),
				"file_path":    chunk.filePath,
			}).Debug("Skipping low confidence finding")
			continue
		default:
			finding.Confidence = "medium"
		}

		findings = append(findings, finding)
	}

	return findings
}

// isHallucination detects if LLM output is garbage/hallucinated
func (s *DeepScanner) isHallucination(content string) bool {
	if len(content) < 50 {
		return false
	}

	// Pattern 1: Repetitive number sequences (e.g., "444446644444646444461284...")
	digitCount := 0
	for _, r := range content {
		if r >= '0' && r <= '9' {
			digitCount++
		}
	}
	// If more than 60% of content is digits, it's likely garbage
	if float64(digitCount)/float64(len(content)) > 0.6 {
		return true
	}

	// Pattern 2: Repetitive "I" + number patterns (e.g., "I5 I54 I6 I128...")
	iPatterns := []string{"I4", "I5", "I6", "I12", "I28", "I64", "I128"}
	iCount := 0
	for _, p := range iPatterns {
		iCount += strings.Count(content, p)
	}
	if iCount > 10 {
		return true
	}

	// Pattern 3: Very long strings without spaces (likely binary/encoded garbage)
	words := strings.FieldsSeq(content)
	for word := range words {
		if len(word) > 200 && !strings.HasPrefix(word, "{") && !strings.HasPrefix(word, "\"") {
			return true
		}
	}

	// Pattern 4: Repetitive character sequences
	if len(content) > 100 {
		// Check for any character repeated more than 20 times in a row
		for i := 0; i < len(content)-20; i++ {
			allSame := true
			char := content[i]
			for j := 1; j < 20; j++ {
				if content[i+j] != char {
					allSame = false
					break
				}
			}
			if allSame && char != ' ' && char != '\n' {
				return true
			}
		}
	}

	// Pattern 5: No valid JSON structure at all
	if !strings.Contains(content, "{") && !strings.Contains(content, "findings") {
		return true
	}

	return false
}

// isLikelyFalsePositive checks if a finding looks like a false positive
func (s *DeepScanner) isLikelyFalsePositive(codeSnippet, description, confidence string) bool {
	snippetLower := strings.ToLower(codeSnippet)
	descLower := strings.ToLower(description)

	// Placeholder patterns - these are NOT real secrets
	placeholderPatterns := []string{
		"your-api-key",
		"your_api_key",
		"your-secret",
		"your_secret",
		"your-token",
		"your_token",
		"insert-here",
		"insert_here",
		"changeme",
		"change-me",
		"change_me",
		"xxxxxxxx",
		"xxx-xxx",
		"example",
		"sample",
		"placeholder",
		"<your",
		"<insert",
		"<api",
		"<token",
		"<secret",
		"<password",
		"${",
		"{{",
		"%s",
		"todo:",
		"fixme:",
		"replace_with",
		"replace-with",
		"dummy",
		"fake",
		"mock",
		"test_key",
		"test_token",
		"test_secret",
		"test_password",
		"demo_",
		"dev_key",
		"dev_token",
	}

	for _, pattern := range placeholderPatterns {
		if strings.Contains(snippetLower, pattern) {
			return true
		}
	}

	// Environment variable patterns - these are SAFE
	envPatterns := []string{
		"os.getenv",
		"os.environ",
		"process.env",
		"env[",
		"env.get",
		"getenv(",
		"viper.get",
		"config.get",
		"settings.",
		"${env:",
		"${env.",
	}

	for _, pattern := range envPatterns {
		if strings.Contains(snippetLower, pattern) {
			return true
		}
	}

	// Empty or trivial values
	trivialPatterns := []string{
		`= ""`,
		`= ''`,
		`= nil`,
		`= null`,
		`= none`,
		`= ""`,
		`: ""`,
		`: ''`,
	}

	for _, pattern := range trivialPatterns {
		if strings.Contains(snippetLower, pattern) {
			return true
		}
	}

	// Description suggests it's not a real secret
	falsePositiveDescriptions := []string{
		"placeholder",
		"example",
		"template",
		"sample",
		"mock",
		"test",
		"dummy",
		"default value",
		"empty",
	}

	for _, pattern := range falsePositiveDescriptions {
		if strings.Contains(descLower, pattern) && confidence != "high" {
			return true
		}
	}

	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// isTestFile checks if the file path indicates a test/mock/fixture file
func isTestFile(filePath string) bool {
	if filePath == "" {
		return false
	}

	lowerPath := strings.ToLower(filePath)

	// Test file patterns
	testPatterns := []string{
		"_test.go",
		"_test.py",
		"_test.js",
		"_test.ts",
		"_test.rb",
		"_spec.go",
		"_spec.py",
		"_spec.js",
		"_spec.ts",
		"_spec.rb",
		".test.go",
		".test.js",
		".test.ts",
		".spec.js",
		".spec.ts",
		"test_",
		"mock_",
		"fake_",
		"stub_",
	}

	for _, pattern := range testPatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	// Test directories
	testDirs := []string{
		"/test/",
		"/tests/",
		"/testing/",
		"/__tests__/",
		"/spec/",
		"/specs/",
		"/fixtures/",
		"/testdata/",
		"/test_data/",
		"/mocks/",
		"/mock/",
		"/fakes/",
		"/stubs/",
		"/__mocks__/",
		"/examples/",
		"/example/",
		"/samples/",
		"/sample/",
		"/demo/",
		"/demos/",
	}

	for _, dir := range testDirs {
		if strings.Contains(lowerPath, dir) {
			return true
		}
	}

	return false
}

// isDocumentationFile checks if the file is documentation
func isDocumentationFile(filePath string) bool {
	if filePath == "" {
		return false
	}

	lowerPath := strings.ToLower(filePath)

	docPatterns := []string{
		".md",
		".rst",
		".txt",
		"readme",
		"changelog",
		"contributing",
		"license",
		"/docs/",
		"/doc/",
		"/documentation/",
	}

	for _, pattern := range docPatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}
