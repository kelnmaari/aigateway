// Package scanner provides code security scanning functionality.
package scanner

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"aigateway/internal/rag/vector"

	"github.com/sirupsen/logrus"
)

// SecretPattern defines a pattern for detecting secrets
type SecretPattern struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Regex       *regexp.Regexp `json:"-"`
	Pattern     string         `json:"pattern"`
	Severity    Severity       `json:"severity"`
	Category    string         `json:"category"`
}

// Severity level for findings
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "info"
)

// Finding represents a detected secret
type Finding struct {
	ID          string   `json:"id"`
	PatternID   string   `json:"pattern_id"`
	PatternName string   `json:"pattern_name"`
	Category    string   `json:"category"`
	Severity    Severity `json:"severity"`
	FilePath    string   `json:"file_path"`
	StartLine   int      `json:"start_line"`
	EndLine     int      `json:"end_line"`
	Match       string   `json:"match"`      // Redacted match
	Context     string   `json:"context"`    // Surrounding code (redacted)
	Suggestion  string   `json:"suggestion"` // How to fix
}

// ScanRequest parameters for scanning
type ScanRequest struct {
	ProjectID      string   `json:"project_id"`
	CollectionName string   `json:"collection_name"`
	Categories     []string `json:"categories,omitempty"` // Filter by categories
	MinSeverity    Severity `json:"min_severity,omitempty"`
}

// ScanResult contains scan results
type ScanResult struct {
	ProjectID     string      `json:"project_id"`
	ScanID        string      `json:"scan_id"`
	StartedAt     time.Time   `json:"started_at"`
	CompletedAt   time.Time   `json:"completed_at"`
	Duration      string      `json:"duration"`
	ChunksScanned int         `json:"chunks_scanned"`
	Findings      []Finding   `json:"findings"`
	Summary       ScanSummary `json:"summary"`
	Status        string      `json:"status"`
	Error         string      `json:"error,omitempty"`
}

// ScanSummary provides overview of findings
type ScanSummary struct {
	TotalFindings int            `json:"total_findings"`
	BySeverity    map[string]int `json:"by_severity"`
	ByCategory    map[string]int `json:"by_category"`
	FilesAffected int            `json:"files_affected"`
}

// Scanner scans code for secrets
type Scanner struct {
	patterns    []SecretPattern
	vectorStore *vector.QdrantStore
	logger      *logrus.Logger
}

// NewScanner creates a new secrets scanner
func NewScanner(vectorStore *vector.QdrantStore, logger *logrus.Logger) *Scanner {
	s := &Scanner{
		vectorStore: vectorStore,
		logger:      logger,
		patterns:    defaultPatterns(),
	}
	return s
}

// defaultPatterns returns built-in secret detection patterns
func defaultPatterns() []SecretPattern {
	patterns := []struct {
		id          string
		name        string
		description string
		pattern     string
		severity    Severity
		category    string
	}{
		// API Keys
		{"aws-access-key", "AWS Access Key", "AWS Access Key ID", `AKIA[0-9A-Z]{16}`, SeverityCritical, "cloud"},
		{"aws-secret-key", "AWS Secret Key", "AWS Secret Access Key", `(?i)aws.{0,20}secret.{0,20}['"][0-9a-zA-Z/+]{40}['"]`, SeverityCritical, "cloud"},
		{"gcp-api-key", "GCP API Key", "Google Cloud Platform API Key", `AIza[0-9A-Za-z\-_]{35}`, SeverityCritical, "cloud"},
		{"azure-key", "Azure Storage Key", "Azure Storage Account Key", `(?i)azure.{0,20}key.{0,20}['"][0-9a-zA-Z+/=]{86,88}['"]`, SeverityCritical, "cloud"},

		// OAuth & Tokens
		{"github-token", "GitHub Token", "GitHub Personal Access Token", `ghp_[0-9a-zA-Z]{36}`, SeverityCritical, "oauth"},
		{"github-oauth", "GitHub OAuth", "GitHub OAuth Access Token", `gho_[0-9a-zA-Z]{36}`, SeverityCritical, "oauth"},
		{"gitlab-token", "GitLab Token", "GitLab Personal Access Token", `glpat-[0-9a-zA-Z\-_]{20,}`, SeverityCritical, "oauth"},
		{"slack-token", "Slack Token", "Slack Bot/User Token", `xox[baprs]-[0-9a-zA-Z\-]{10,}`, SeverityHigh, "oauth"},
		{"discord-token", "Discord Token", "Discord Bot Token", `[MN][A-Za-z\d]{23,}\.[\w-]{6}\.[\w-]{27}`, SeverityHigh, "oauth"},

		// Database & Connection Strings
		{"postgres-uri", "PostgreSQL URI", "PostgreSQL Connection String with Password", `postgres(?:ql)?://[^:]+:[^@]+@[^/]+`, SeverityHigh, "database"},
		{"mysql-uri", "MySQL URI", "MySQL Connection String with Password", `mysql://[^:]+:[^@]+@[^/]+`, SeverityHigh, "database"},
		{"mongodb-uri", "MongoDB URI", "MongoDB Connection String with Password", `mongodb(?:\+srv)?://[^:]+:[^@]+@[^/]+`, SeverityHigh, "database"},
		{"redis-uri", "Redis URI", "Redis Connection String with Password", `redis://:[^@]+@[^/]+`, SeverityHigh, "database"},

		// Private Keys
		{"private-key-rsa", "RSA Private Key", "RSA Private Key Header", `-----BEGIN RSA PRIVATE KEY-----`, SeverityCritical, "keys"},
		{"private-key-ec", "EC Private Key", "EC Private Key Header", `-----BEGIN EC PRIVATE KEY-----`, SeverityCritical, "keys"},
		{"private-key-openssh", "OpenSSH Private Key", "OpenSSH Private Key Header", `-----BEGIN OPENSSH PRIVATE KEY-----`, SeverityCritical, "keys"},
		{"private-key-pgp", "PGP Private Key", "PGP Private Key Block", `-----BEGIN PGP PRIVATE KEY BLOCK-----`, SeverityCritical, "keys"},

		// Generic Secrets
		{"generic-api-key", "Generic API Key", "Potential API Key Assignment", `(?i)api[_-]?key\s*[:=]\s*['"][0-9a-zA-Z\-_]{20,}['"]`, SeverityMedium, "generic"},
		{"generic-secret", "Generic Secret", "Potential Secret Assignment", `(?i)secret\s*[:=]\s*['"][0-9a-zA-Z\-_]{16,}['"]`, SeverityMedium, "generic"},
		{"generic-password", "Generic Password", "Potential Password Assignment", `(?i)password\s*[:=]\s*['"][^'"]{8,}['"]`, SeverityMedium, "generic"},
		{"generic-token", "Generic Token", "Potential Token Assignment", `(?i)token\s*[:=]\s*['"][0-9a-zA-Z\-_]{20,}['"]`, SeverityMedium, "generic"},

		// JWT
		{"jwt-token", "JWT Token", "JSON Web Token", `eyJ[A-Za-z0-9\-_]+\.eyJ[A-Za-z0-9\-_]+\.[A-Za-z0-9\-_]+`, SeverityMedium, "tokens"},

		// Hardcoded IPs & Internal URLs
		{"internal-ip", "Internal IP", "Hardcoded Internal IP Address", `\b(?:10\.\d{1,3}|172\.(?:1[6-9]|2\d|3[01])|192\.168)\.\d{1,3}\.\d{1,3}\b`, SeverityLow, "network"},

		// Stripe
		{"stripe-key", "Stripe Key", "Stripe API Key", `sk_live_[0-9a-zA-Z]{24,}`, SeverityCritical, "payment"},
		{"stripe-restricted", "Stripe Restricted Key", "Stripe Restricted API Key", `rk_live_[0-9a-zA-Z]{24,}`, SeverityCritical, "payment"},

		// Twilio
		{"twilio-key", "Twilio API Key", "Twilio API Key or Auth Token", `SK[0-9a-fA-F]{32}`, SeverityHigh, "communication"},

		// SendGrid
		{"sendgrid-key", "SendGrid API Key", "SendGrid API Key", `SG\.[0-9A-Za-z\-_]{22}\.[0-9A-Za-z\-_]{43}`, SeverityHigh, "communication"},

		// Mailgun
		{"mailgun-key", "Mailgun API Key", "Mailgun API Key", `key-[0-9a-zA-Z]{32}`, SeverityHigh, "communication"},

		// Heroku
		{"heroku-key", "Heroku API Key", "Heroku API Key", `(?i)heroku.{0,20}[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`, SeverityHigh, "cloud"},

		// npm
		{"npm-token", "npm Token", "npm Access Token", `npm_[0-9a-zA-Z]{36}`, SeverityHigh, "registry"},

		// PyPI
		{"pypi-token", "PyPI Token", "PyPI API Token", `pypi-AgEIcHlwaS5vcmc[0-9A-Za-z\-_]{50,}`, SeverityHigh, "registry"},

		// Docker
		{"docker-auth", "Docker Auth", "Docker Registry Auth", `(?i)docker.{0,20}auth.{0,20}['"][0-9a-zA-Z+/=]{20,}['"]`, SeverityMedium, "container"},

		// Telegram
		{"telegram-token", "Telegram Bot Token", "Telegram Bot API Token", `\d{9,10}:[0-9A-Za-z_-]{35}`, SeverityHigh, "communication"},
	}

	result := make([]SecretPattern, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p.pattern)
		if err != nil {
			continue // Skip invalid patterns
		}
		result = append(result, SecretPattern{
			ID:          p.id,
			Name:        p.name,
			Description: p.description,
			Regex:       re,
			Pattern:     p.pattern,
			Severity:    p.severity,
			Category:    p.category,
		})
	}

	return result
}

// Scan performs secrets scanning on indexed code
func (s *Scanner) Scan(ctx context.Context, req ScanRequest) (*ScanResult, error) {
	startTime := time.Now()
	scanID := generateScanID()

	s.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
		"scan_id":    scanID,
	}).Info("Starting secrets scan")

	result := &ScanResult{
		ProjectID: req.ProjectID,
		ScanID:    scanID,
		StartedAt: startTime,
		Status:    "running",
		Findings:  []Finding{},
		Summary: ScanSummary{
			BySeverity: make(map[string]int),
			ByCategory: make(map[string]int),
		},
	}

	// Filter patterns by categories if specified
	patterns := s.filterPatterns(req.Categories, req.MinSeverity)

	var (
		findings      []Finding
		findingsMu    sync.Mutex
		chunksScanned int
		filesAffected = make(map[string]bool)
	)

	// Scroll through all chunks
	err := s.vectorStore.ScrollAll(ctx, req.CollectionName, map[string]any{
		"project_id": req.ProjectID,
	}, func(docs []vector.VectorDocument) error {
		for _, doc := range docs {
			chunksScanned++

			// Get content and metadata
			content := doc.Text
			if content == "" {
				if c, ok := doc.Metadata["content"].(string); ok {
					content = c
				}
			}
			if content == "" {
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

			// Scan content against all patterns
			for _, pattern := range patterns {
				matches := pattern.Regex.FindAllStringIndex(content, -1)
				for _, match := range matches {
					// Skip if it looks like a placeholder/example
					matchStr := content[match[0]:match[1]]
					if isPlaceholder(matchStr) {
						continue
					}

					finding := Finding{
						ID:          generateFindingID(),
						PatternID:   pattern.ID,
						PatternName: pattern.Name,
						Category:    pattern.Category,
						Severity:    pattern.Severity,
						FilePath:    filePath,
						StartLine:   startLine,
						EndLine:     endLine,
						Match:       redactSecret(matchStr),
						Context:     extractContext(content, match[0], match[1]),
						Suggestion:  getSuggestion(pattern.Category),
					}

					findingsMu.Lock()
					findings = append(findings, finding)
					filesAffected[filePath] = true
					findingsMu.Unlock()
				}
			}
		}
		return nil
	})

	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	// Build summary
	result.Findings = findings
	result.ChunksScanned = chunksScanned
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(startTime).String()
	result.Status = "completed"

	result.Summary.TotalFindings = len(findings)
	result.Summary.FilesAffected = len(filesAffected)

	for _, f := range findings {
		result.Summary.BySeverity[string(f.Severity)]++
		result.Summary.ByCategory[f.Category]++
	}

	s.logger.WithFields(logrus.Fields{
		"project_id":     req.ProjectID,
		"scan_id":        scanID,
		"chunks_scanned": chunksScanned,
		"findings":       len(findings),
		"duration":       result.Duration,
	}).Info("Secrets scan completed")

	return result, nil
}

// filterPatterns filters patterns by categories and severity
func (s *Scanner) filterPatterns(categories []string, minSeverity Severity) []SecretPattern {
	if len(categories) == 0 && minSeverity == "" {
		return s.patterns
	}

	severityOrder := map[Severity]int{
		SeverityCritical: 5,
		SeverityHigh:     4,
		SeverityMedium:   3,
		SeverityLow:      2,
		SeverityInfo:     1,
	}

	minSevVal := 0
	if minSeverity != "" {
		minSevVal = severityOrder[minSeverity]
	}

	categorySet := make(map[string]bool)
	for _, c := range categories {
		categorySet[c] = true
	}

	result := make([]SecretPattern, 0)
	for _, p := range s.patterns {
		if minSevVal > 0 && severityOrder[p.Severity] < minSevVal {
			continue
		}
		if len(categorySet) > 0 && !categorySet[p.Category] {
			continue
		}
		result = append(result, p)
	}

	return result
}

// GetPatterns returns all available patterns
func (s *Scanner) GetPatterns() []SecretPattern {
	return s.patterns
}

// isPlaceholder checks if the match looks like a placeholder
func isPlaceholder(match string) bool {
	placeholders := []string{
		"xxx", "XXX", "your_", "YOUR_", "<your", "<YOUR",
		"example", "EXAMPLE", "placeholder", "PLACEHOLDER",
		"changeme", "CHANGEME", "secret123", "password123",
		"xxxxxxxx", "XXXXXXXX", "12345678", "00000000",
		"test_", "TEST_", "demo_", "DEMO_",
	}

	matchLower := strings.ToLower(match)
	for _, p := range placeholders {
		if strings.Contains(matchLower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// redactSecret partially hides the secret
func redactSecret(secret string) string {
	if len(secret) <= 8 {
		return strings.Repeat("*", len(secret))
	}
	// Show first 4 and last 4 characters
	return secret[:4] + strings.Repeat("*", len(secret)-8) + secret[len(secret)-4:]
}

// extractContext extracts surrounding code for context
func extractContext(content string, start, end int) string {
	// Get 50 chars before and after
	contextStart := max(start-50, 0)
	contextEnd := min(end+50, len(content))

	ctx := content[contextStart:contextEnd]
	// Redact the actual secret in context
	if start > contextStart && end <= contextEnd {
		secretInCtx := content[start:end]
		ctx = strings.Replace(ctx, secretInCtx, redactSecret(secretInCtx), 1)
	}

	return strings.TrimSpace(ctx)
}

// getSuggestion returns a fix suggestion based on category
func getSuggestion(category string) string {
	suggestions := map[string]string{
		"cloud":         "Use environment variables or a secrets manager (AWS Secrets Manager, HashiCorp Vault)",
		"oauth":         "Store tokens in environment variables and rotate them regularly",
		"database":      "Use connection string from environment variables, never commit credentials",
		"keys":          "Store private keys outside of repository, use a secrets manager",
		"generic":       "Move sensitive values to environment variables or .env file (add to .gitignore)",
		"tokens":        "Use short-lived tokens and store them securely",
		"network":       "Use DNS names or configuration files for internal addresses",
		"payment":       "Use environment variables for payment provider keys, enable key rotation",
		"communication": "Store API keys in environment variables or secrets manager",
		"registry":      "Use CI/CD secrets or environment variables for registry tokens",
		"container":     "Use Docker secrets or environment variables for auth",
	}

	if s, ok := suggestions[category]; ok {
		return s
	}
	return "Move sensitive values to environment variables or a secrets manager"
}

func generateScanID() string {
	return time.Now().Format("20060102-150405")
}

func generateFindingID() string {
	return time.Now().Format("20060102150405.000000000")
}
