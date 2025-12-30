// Package changelog provides changelog fetching and LLM analysis for dependencies.
package changelog

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

// Analyzer analyzes changelogs using LLM.
type Analyzer struct {
	llmBaseURL string
	llmAPIKey  string
	fetcher    *Fetcher
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewAnalyzer creates a new changelog analyzer.
func NewAnalyzer(llmBaseURL, llmAPIKey string, logger *logrus.Logger) *Analyzer {
	return &Analyzer{
		llmBaseURL: llmBaseURL,
		llmAPIKey:  llmAPIKey,
		fetcher:    NewFetcher(logger),
		httpClient: &http.Client{Timeout: 2 * time.Minute},
		logger:     logger,
	}
}

// AnalyzeChangelog analyzes a dependency's changelog for breaking changes and important updates.
func (a *Analyzer) AnalyzeChangelog(ctx context.Context, req AnalyzeRequest) (*ChangelogAnalysis, error) {
	startTime := time.Now()

	a.logger.WithFields(logrus.Fields{
		"package":         req.PackageName,
		"current_version": req.CurrentVersion,
		"latest_version":  req.LatestVersion,
		"language":        req.Language,
	}).Info("Analyzing changelog")

	result := &ChangelogAnalysis{
		PackageName:    req.PackageName,
		CurrentVersion: req.CurrentVersion,
		LatestVersion:  req.LatestVersion,
		Language:       req.Language,
		AnalyzedAt:     startTime,
	}

	// If no changelog text provided, fetch it
	changelogText := req.ChangelogText
	if changelogText == "" {
		info, err := a.fetcher.FetchChangelog(ctx, req.PackageName, req.CurrentVersion, req.LatestVersion, req.Language)
		if err != nil {
			a.logger.WithError(err).Debug("Failed to fetch changelog, using minimal analysis")
			// Continue with minimal analysis
			result.Summary = fmt.Sprintf("Update from %s to %s available. Could not fetch changelog for detailed analysis.", req.CurrentVersion, req.LatestVersion)
			result.RiskLevel = "medium"
			result.Confidence = "low"
			return result, nil
		}

		if info.ReleaseNotes != "" {
			changelogText = info.ReleaseNotes
		} else if info.ChangelogText != "" {
			changelogText = info.ChangelogText
		}
	}

	if changelogText == "" {
		result.Summary = fmt.Sprintf("Update from %s to %s available. No changelog found.", req.CurrentVersion, req.LatestVersion)
		result.RiskLevel = "medium"
		result.Confidence = "low"
		return result, nil
	}

	// Analyze with LLM
	systemPrompt := a.getSystemPrompt(req.Language)
	userPrompt := a.getUserPrompt(req.PackageName, req.CurrentVersion, req.LatestVersion, changelogText)

	responseText, tokensUsed, err := a.callLLM(ctx, req.ModelID, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	result.TokensUsed = tokensUsed

	// Parse LLM response
	if err := a.parseLLMResponse(responseText, result); err != nil {
		a.logger.WithError(err).Warn("Failed to parse LLM response, using raw response")
		result.Summary = responseText
		result.RiskLevel = "medium"
		result.Confidence = "low"
	}

	a.logger.WithFields(logrus.Fields{
		"package":    req.PackageName,
		"risk_level": result.RiskLevel,
		"breaking":   len(result.BreakingChanges),
		"duration":   time.Since(startTime).String(),
	}).Info("Changelog analysis completed")

	return result, nil
}

// callLLM makes a request to the LLM API.
func (a *Analyzer) callLLM(ctx context.Context, modelID, systemPrompt, userPrompt string) (string, int, error) {
	reqBody := map[string]interface{}{
		"model": modelID,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.3,
		"max_tokens":  2048,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimSuffix(a.llmBaseURL, "/") + "/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if a.llmAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+a.llmAPIKey)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("LLM API returned %d: %s", resp.StatusCode, string(body))
	}

	var llmResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &llmResp); err != nil {
		return "", 0, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if len(llmResp.Choices) == 0 {
		return "", 0, fmt.Errorf("LLM returned no choices")
	}

	return llmResp.Choices[0].Message.Content, llmResp.Usage.TotalTokens, nil
}

// FetchAndAnalyze fetches changelog and analyzes it.
func (a *Analyzer) FetchAndAnalyze(ctx context.Context, packageName, currentVersion, latestVersion, language, modelID string) (*ChangelogAnalysis, error) {
	return a.AnalyzeChangelog(ctx, AnalyzeRequest{
		PackageName:    packageName,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		Language:       language,
		ModelID:        modelID,
	})
}

func (a *Analyzer) getSystemPrompt(language string) string {
	return fmt.Sprintf(`You are an expert software engineer specialized in analyzing changelogs and release notes for %s packages.
Your task is to analyze the changelog between two versions and identify:
1. Breaking changes that could cause issues during upgrade
2. New features
3. Bug fixes
4. Security fixes
5. Deprecated features
6. Migration steps if needed

Assess the overall risk level of the upgrade:
- "low": Only bug fixes and minor improvements, safe to upgrade
- "medium": New features or minor breaking changes that are easy to handle
- "high": Significant breaking changes requiring code modifications
- "critical": Major breaking changes, security issues, or complete API overhauls

Your response MUST be a valid JSON object with this structure:
{
  "summary": "Brief 1-2 sentence summary of the upgrade",
  "breaking_changes": [
    {
      "description": "What changed",
      "affected_area": "API|Config|Behavior|Types|Dependencies",
      "severity": "high|medium|low",
      "workaround": "How to fix or work around this (optional)"
    }
  ],
  "new_features": ["Feature 1", "Feature 2"],
  "bug_fixes": ["Fix 1", "Fix 2"],
  "security_fixes": ["Security fix 1"],
  "deprecated_features": ["Deprecated 1"],
  "migration_guide": "Step by step migration instructions if needed, or empty string",
  "risk_level": "low|medium|high|critical",
  "confidence": "high|medium|low"
}

If the changelog is unclear or incomplete, set confidence to "low".
Always respond with valid JSON only, no additional text.
`, language)
}

func (a *Analyzer) getUserPrompt(packageName, currentVersion, latestVersion, changelogText string) string {
	// Truncate very long changelogs
	if len(changelogText) > 8000 {
		changelogText = changelogText[:8000] + "\n... (truncated)"
	}

	return fmt.Sprintf(`Analyze the changelog for package "%s" for upgrade from version %s to %s:

---
%s
---

Provide your analysis as JSON.`, packageName, currentVersion, latestVersion, changelogText)
}

func (a *Analyzer) parseLLMResponse(response string, result *ChangelogAnalysis) error {
	// Clean response - remove markdown code blocks if present
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var parsed struct {
		Summary            string `json:"summary"`
		BreakingChanges    []struct {
			Description  string `json:"description"`
			AffectedArea string `json:"affected_area"`
			Severity     string `json:"severity"`
			Workaround   string `json:"workaround"`
		} `json:"breaking_changes"`
		NewFeatures        []string `json:"new_features"`
		BugFixes           []string `json:"bug_fixes"`
		SecurityFixes      []string `json:"security_fixes"`
		DeprecatedFeatures []string `json:"deprecated_features"`
		MigrationGuide     string   `json:"migration_guide"`
		RiskLevel          string   `json:"risk_level"`
		Confidence         string   `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	result.Summary = parsed.Summary
	result.NewFeatures = parsed.NewFeatures
	result.BugFixes = parsed.BugFixes
	result.SecurityFixes = parsed.SecurityFixes
	result.DeprecatedFeatures = parsed.DeprecatedFeatures
	result.MigrationGuide = parsed.MigrationGuide
	result.RiskLevel = parsed.RiskLevel
	result.Confidence = parsed.Confidence

	for _, bc := range parsed.BreakingChanges {
		result.BreakingChanges = append(result.BreakingChanges, BreakingChange{
			Description:  bc.Description,
			AffectedArea: bc.AffectedArea,
			Severity:     bc.Severity,
			Workaround:   bc.Workaround,
		})
	}

	// Validate and set defaults
	if result.RiskLevel == "" {
		result.RiskLevel = "medium"
	}
	if result.Confidence == "" {
		result.Confidence = "medium"
	}

	return nil
}

