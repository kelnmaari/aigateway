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
		httpClient:  &http.Client{Timeout: 3 * time.Minute}, // Increased from 2m
		logger:      logger,
	}
}

// DeepScan performs semantic analysis using LLM
func (s *DeepScanner) DeepScan(ctx context.Context, req DeepScanRequest) (*DeepScanResult, error) {
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
	var allChunks []chunkData
	err := s.vectorStore.ScrollAll(ctx, req.CollectionName, map[string]interface{}{
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

	s.logger.WithField("chunks_collected", len(allChunks)).Info("Collected chunks for deep analysis")

	// Process chunks in batches (5 chunks per LLM call to optimize)
	batchSize := 5
	for i := 0; i < len(allChunks); i += batchSize {
		select {
		case <-ctx.Done():
			result.Status = "cancelled"
			result.Error = "scan cancelled"
			return result, ctx.Err()
		default:
		}

		end := i + batchSize
		if end > len(allChunks) {
			end = len(allChunks)
		}
		batch := allChunks[i:end]
		chunksScanned += len(batch)

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

	requestBody := map[string]interface{}{
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
		return `Ты эксперт по безопасности кода. Анализируй код на наличие:
1. Хардкод секретов (API ключи, пароли, токены)
2. Утечки чувствительных данных
3. Небезопасных практик

⚠️ КРИТИЧЕСКИ ВАЖНО: Отвечай СТРОГО в JSON формате. НЕ добавляй текст до или после JSON!

Формат ответа:
{
  "findings": [
    {
      "chunk_index": 1,
      "type": "hardcoded_secret",
      "severity": "critical",
      "description": "Найден хардкод API ключ AWS",
      "code_snippet": "aws_key = 'AKIA...'",
      "suggestion": "Использовать переменные окружения",
      "confidence": "high"
    }
  ]
}

 If secrets are NOT found, return STICTLY: {"findings": []}

⚠️ ВАЖНО: Никаких вступлений, пояснений или текста до/после JSON. Только "чистый" JSON объект.
⚠️ Начни ответ с символа '{' и закончи символом '}'.`
	}

	return `You are a code security expert. Analyze code for:
1. Hardcoded secrets (API keys, passwords, tokens)
2. Sensitive data leaks
3. Insecure practices

⚠️ CRITICAL: Respond STRICTLY in JSON format. DO NOT add any text before or after JSON!

Response format:
{
  "findings": [
    {
      "chunk_index": 1,
      "type": "hardcoded_secret",
      "severity": "critical",
      "description": "Found hardcoded AWS API key",
      "code_snippet": "aws_key = 'AKIA...'",
      "suggestion": "Use environment variables",
      "confidence": "high"
    }
  ]
}

 If secrets are NOT found, return STICTLY: {"findings": []}

⚠️ IMPORTANT: No introductions, no explanations, no text before or after JSON. Only raw JSON object.
⚠️ Start response with '{' and end with '}'.`
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

	// Remove possible repetitive trash (hallucinations like "I5 I54...")
	if len(content) > 100 && strings.Count(content, "I5") > 10 {
		s.logger.Warn("LLM returned repetitive garbage, skipping batch")
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

		findings = append(findings, finding)
	}

	return findings
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
