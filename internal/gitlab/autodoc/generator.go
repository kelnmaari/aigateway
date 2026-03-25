// Package autodoc provides automatic documentation generation for code.
package autodoc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Generator generates documentation using LLM.
type Generator struct {
	llmBaseURL string
	llmAPIKey  string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewGenerator creates a new documentation generator.
func NewGenerator(llmBaseURL, llmAPIKey string, logger *logrus.Logger) *Generator {
	return &Generator{
		llmBaseURL: llmBaseURL,
		llmAPIKey:  llmAPIKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Minute,
		},
		logger: logger,
	}
}

// Generate generates documentation for symbols.
func (g *Generator) Generate(ctx context.Context, req GenerateRequest, symbols []UndocumentedSymbol) (*GenerationResult, error) {
	startTime := time.Now()

	g.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"symbols":    len(symbols),
		"model":      req.ModelID,
	}).Info("Starting documentation generation")

	result := &GenerationResult{
		ProjectID:   req.ProjectID,
		GeneratedAt: startTime,
		Status:      "running",
		Docs:        []GeneratedDoc{},
		ModelID:     req.ModelID,
	}

	// Limit symbols
	maxSymbols := req.MaxSymbols
	if maxSymbols <= 0 {
		maxSymbols = 20
	}
	if len(symbols) > maxSymbols {
		symbols = symbols[:maxSymbols]
	}

	totalTokens := 0

	for _, sym := range symbols {
		doc, tokens, err := g.generateForSymbol(ctx, req.ModelID, sym)
		if err != nil {
			g.logger.WithError(err).WithField("symbol", sym.Name).Warn("Failed to generate doc")
			continue
		}
		totalTokens += tokens
		result.Docs = append(result.Docs, doc)
	}

	result.TokensUsed = totalTokens
	result.Status = "completed"
	result.Duration = time.Since(startTime).String()

	g.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"generated":  len(result.Docs),
		"tokens":     totalTokens,
		"duration":   result.Duration,
	}).Info("Documentation generation completed")

	return result, nil
}

// generateForSymbol generates documentation for a single symbol
func (g *Generator) generateForSymbol(ctx context.Context, modelID string, sym UndocumentedSymbol) (GeneratedDoc, int, error) {
	doc := GeneratedDoc{
		Symbol:   sym,
		Language: sym.Language,
	}

	// Determine format
	switch sym.Language {
	case "go":
		doc.Format = DocFormatGoDoc
	case "javascript", "typescript":
		doc.Format = DocFormatJSDoc
	case "python":
		doc.Format = DocFormatPyDoc
	default:
		doc.Format = DocFormatGoDoc
	}

	prompt := g.buildPrompt(sym)
	docText, tokens, err := g.callLLM(ctx, modelID, prompt, sym.Language)
	if err != nil {
		return doc, tokens, err
	}

	doc.Documentation = docText
	doc.Preview = g.buildPreview(sym, docText)

	return doc, tokens, nil
}

// buildPrompt builds the LLM prompt for documentation generation
func (g *Generator) buildPrompt(sym UndocumentedSymbol) string {
	var formatInstructions string

	switch sym.Language {
	case "go":
		formatInstructions = `Generate a GoDoc comment for this Go code.
Rules:
- Start with "// {Name} ..." where {Name} is the function/type name
- Use complete sentences
- Describe what it does, not how
- Include parameter descriptions for complex params
- Mention return values
- Keep it concise but informative

Example:
// ProcessRequest handles incoming HTTP requests and returns
// a formatted response. It validates the request body and
// forwards valid requests to the appropriate handler.`

	case "javascript", "typescript":
		formatInstructions = `Generate a JSDoc comment for this JavaScript/TypeScript code.
Rules:
- Use /** ... */ format
- Include @param for each parameter with type and description
- Include @returns for return value
- Include @throws if it throws
- Keep descriptions concise

Example:
/**
 * Processes the incoming request and returns a response.
 * @param {Request} req - The incoming HTTP request
 * @param {Response} res - The response object
 * @returns {Promise<void>} Resolves when processing is complete
 */`

	case "python":
		formatInstructions = `Generate a Python docstring for this code.
Rules:
- Use triple quotes """..."""
- First line: brief summary
- Args section for parameters
- Returns section for return value
- Use Google style docstring format

Example:
"""Process the incoming request and return a response.

Args:
    request: The incoming HTTP request object.
    options: Optional configuration dictionary.

Returns:
    A Response object containing the processed data.
"""`
	}

	return fmt.Sprintf(`%s

Code to document:
%s

File: %s
Type: %s

Generate ONLY the documentation comment, nothing else.
Do not include the code itself.
Respond with the doc comment only.`, formatInstructions, sym.Signature, sym.FilePath, sym.Type)
}

// buildPreview builds a preview of the code with documentation inserted
func (g *Generator) buildPreview(sym UndocumentedSymbol, docText string) string {
	return docText + "\n" + sym.Signature
}

// callLLM makes a request to the LLM API
func (g *Generator) callLLM(ctx context.Context, modelID, prompt, language string) (string, int, error) {
	systemPrompt := fmt.Sprintf("You are a technical documentation writer. Generate clear, concise documentation for %s code. Output ONLY the documentation comment, no explanations.", language)

	reqBody := map[string]any{
		"model": modelID,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"max_tokens":  500,
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

	content := strings.TrimSpace(llmResponse.Choices[0].Message.Content)
	return content, llmResponse.Usage.TotalTokens, nil
}
