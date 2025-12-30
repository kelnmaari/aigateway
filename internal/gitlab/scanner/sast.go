// Package scanner provides SAST (Static Application Security Testing) functionality.
package scanner

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	"aigateway/internal/rag/vector"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// VulnerabilityType represents the type of security vulnerability.
type VulnerabilityType string

const (
	VulnSQLInjection    VulnerabilityType = "sql_injection"
	VulnXSS             VulnerabilityType = "xss"
	VulnPathTraversal   VulnerabilityType = "path_traversal"
	VulnCommandInjection VulnerabilityType = "command_injection"
	VulnHardcodedIP     VulnerabilityType = "hardcoded_ip"
	VulnHardcodedURL    VulnerabilityType = "hardcoded_url"
	VulnInsecureCrypto  VulnerabilityType = "insecure_crypto"
	VulnInsecureRandom  VulnerabilityType = "insecure_random"
	VulnOpenRedirect    VulnerabilityType = "open_redirect"
	VulnSSRF            VulnerabilityType = "ssrf"
	VulnXXE             VulnerabilityType = "xxe"
	VulnInsecureDeserial VulnerabilityType = "insecure_deserialization"
	VulnHardcodedCreds   VulnerabilityType = "hardcoded_credentials"
)

// SASTPattern defines a pattern for SAST vulnerability detection.
type SASTPattern struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        VulnerabilityType `json:"type"`
	Regex       *regexp.Regexp    `json:"-"`
	Pattern     string            `json:"pattern"`
	Severity    Severity          `json:"severity"`
	CWE         string            `json:"cwe,omitempty"`     // Common Weakness Enumeration ID
	OWASP       string            `json:"owasp,omitempty"`   // OWASP Top 10 category
	Languages   []string          `json:"languages"`         // Applicable languages
	Suggestion  string            `json:"suggestion"`        // How to fix
	FalsePositive []string        `json:"-"`                 // Strings that indicate false positive
}

// SASTFinding represents a detected security vulnerability.
type SASTFinding struct {
	ID          string            `json:"id"`
	PatternID   string            `json:"pattern_id"`
	PatternName string            `json:"pattern_name"`
	Type        VulnerabilityType `json:"type"`
	Severity    Severity          `json:"severity"`
	CWE         string            `json:"cwe,omitempty"`
	OWASP       string            `json:"owasp,omitempty"`
	FilePath    string            `json:"file_path"`
	StartLine   int               `json:"start_line"`
	EndLine     int               `json:"end_line"`
	Match       string            `json:"match"`
	Context     string            `json:"context"`
	Suggestion  string            `json:"suggestion"`
	Language    string            `json:"language"`
}

// SASTScanRequest parameters for SAST scanning.
type SASTScanRequest struct {
	ProjectID      string              `json:"project_id"`
	CollectionName string              `json:"collection_name"`
	Types          []VulnerabilityType `json:"types,omitempty"`
	MinSeverity    Severity            `json:"min_severity,omitempty"`
	Language       string              `json:"language,omitempty"`
}

// SASTScanResult contains SAST scan results.
type SASTScanResult struct {
	ProjectID     string         `json:"project_id"`
	ScanID        string         `json:"scan_id"`
	StartedAt     time.Time      `json:"started_at"`
	CompletedAt   time.Time      `json:"completed_at"`
	Duration      string         `json:"duration"`
	ChunksScanned int            `json:"chunks_scanned"`
	Findings      []SASTFinding  `json:"findings"`
	Summary       SASTScanSummary `json:"summary"`
	Status        string         `json:"status"`
	Error         string         `json:"error,omitempty"`
}

// SASTScanSummary provides overview of SAST findings.
type SASTScanSummary struct {
	TotalFindings  int            `json:"total_findings"`
	BySeverity     map[string]int `json:"by_severity"`
	ByType         map[string]int `json:"by_type"`
	ByCWE          map[string]int `json:"by_cwe"`
	FilesAffected  int            `json:"files_affected"`
	CriticalCount  int            `json:"critical_count"`
	HighCount      int            `json:"high_count"`
	MediumCount    int            `json:"medium_count"`
	LowCount       int            `json:"low_count"`
}

// SASTScanner scans code for security vulnerabilities.
type SASTScanner struct {
	patterns    []SASTPattern
	vectorStore *vector.QdrantStore
	logger      *logrus.Logger
}

// NewSASTScanner creates a new SAST scanner.
func NewSASTScanner(vectorStore *vector.QdrantStore, logger *logrus.Logger) *SASTScanner {
	return &SASTScanner{
		vectorStore: vectorStore,
		logger:      logger,
		patterns:    defaultSASTPatterns(),
	}
}

// Scan performs SAST scanning on the project.
func (s *SASTScanner) Scan(ctx context.Context, req SASTScanRequest) (*SASTScanResult, error) {
	startTime := time.Now()
	result := &SASTScanResult{
		ProjectID: req.ProjectID,
		ScanID:    uuid.New().String(),
		StartedAt: startTime,
		Status:    "completed",
		Findings:  []SASTFinding{},
		Summary: SASTScanSummary{
			BySeverity: make(map[string]int),
			ByType:     make(map[string]int),
			ByCWE:      make(map[string]int),
		},
	}

	s.logger.WithFields(logrus.Fields{
		"project_id": req.ProjectID,
		"collection": req.CollectionName,
	}).Info("Starting SAST scan")

	// Collect code chunks from vector store
	var chunks []codeChunk
	var chunksScanned int

	err := s.vectorStore.ScrollAll(ctx, req.CollectionName, nil, func(docs []vector.VectorDocument) error {
		for _, doc := range docs {
			chunksScanned++
			filePath, _ := doc.Metadata["file_path"].(string)
			if filePath == "" {
				continue
			}

			content := doc.Text
			if c, ok := doc.Metadata["content"].(string); ok && c != "" {
				content = c
			}

			startLine := 1
			if sl, ok := doc.Metadata["start_line"].(float64); ok {
				startLine = int(sl)
			}

			chunks = append(chunks, codeChunk{
				FilePath:  filePath,
				Content:   content,
				StartLine: startLine,
				Language:  detectLanguage(filePath),
			})
		}
		return nil
	})
	if err != nil {
		result.Status = "failed"
		result.Error = err.Error()
		return result, err
	}

	result.ChunksScanned = chunksScanned

	// Filter patterns by type if specified
	patterns := s.patterns
	if len(req.Types) > 0 {
		patterns = s.filterPatternsByType(req.Types)
	}

	// Scan chunks in parallel
	var wg sync.WaitGroup
	findingsChan := make(chan []SASTFinding, len(chunks))

	for _, chunk := range chunks {
		wg.Add(1)
		go func(c codeChunk) {
			defer wg.Done()
			findings := s.scanChunk(c, patterns, req.Language)
			findingsChan <- findings
		}(chunk)
	}

	go func() {
		wg.Wait()
		close(findingsChan)
	}()

	// Collect findings
	filesAffected := make(map[string]bool)
	for findings := range findingsChan {
		for _, f := range findings {
			// Apply severity filter
			if req.MinSeverity != "" && !isSeverityAtLeast(f.Severity, req.MinSeverity) {
				continue
			}
			result.Findings = append(result.Findings, f)
			filesAffected[f.FilePath] = true
			result.Summary.BySeverity[string(f.Severity)]++
			result.Summary.ByType[string(f.Type)]++
			if f.CWE != "" {
				result.Summary.ByCWE[f.CWE]++
			}
			switch f.Severity {
			case SeverityCritical:
				result.Summary.CriticalCount++
			case SeverityHigh:
				result.Summary.HighCount++
			case SeverityMedium:
				result.Summary.MediumCount++
			case SeverityLow:
				result.Summary.LowCount++
			}
		}
	}

	result.Summary.TotalFindings = len(result.Findings)
	result.Summary.FilesAffected = len(filesAffected)
	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(startTime).String()

	s.logger.WithFields(logrus.Fields{
		"project_id":   req.ProjectID,
		"findings":     len(result.Findings),
		"duration":     result.Duration,
	}).Info("SAST scan completed")

	return result, nil
}

func (s *SASTScanner) scanChunk(chunk codeChunk, patterns []SASTPattern, filterLang string) []SASTFinding {
	var findings []SASTFinding
	lines := strings.Split(chunk.Content, "\n")

	for _, pattern := range patterns {
		// Skip if pattern doesn't apply to this language
		if len(pattern.Languages) > 0 && !containsLanguage(pattern.Languages, chunk.Language) {
			continue
		}
		if filterLang != "" && chunk.Language != filterLang {
			continue
		}

		for i, line := range lines {
			if pattern.Regex == nil {
				continue
			}

			matches := pattern.Regex.FindAllString(line, -1)
			for _, match := range matches {
				// Check for false positives
				if s.isFalsePositive(line, pattern.FalsePositive) {
					continue
				}

				finding := SASTFinding{
					ID:          uuid.New().String(),
					PatternID:   pattern.ID,
					PatternName: pattern.Name,
					Type:        pattern.Type,
					Severity:    pattern.Severity,
					CWE:         pattern.CWE,
					OWASP:       pattern.OWASP,
					FilePath:    chunk.FilePath,
					StartLine:   chunk.StartLine + i,
					EndLine:     chunk.StartLine + i,
					Match:       s.redact(match),
					Context:     s.getContext(lines, i),
					Suggestion:  pattern.Suggestion,
					Language:    chunk.Language,
				}
				findings = append(findings, finding)
			}
		}
	}

	return findings
}

func (s *SASTScanner) filterPatternsByType(types []VulnerabilityType) []SASTPattern {
	var filtered []SASTPattern
	typeMap := make(map[VulnerabilityType]bool)
	for _, t := range types {
		typeMap[t] = true
	}
	for _, p := range s.patterns {
		if typeMap[p.Type] {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (s *SASTScanner) isFalsePositive(line string, indicators []string) bool {
	for _, indicator := range indicators {
		if strings.Contains(strings.ToLower(line), strings.ToLower(indicator)) {
			return true
		}
	}
	return false
}

func (s *SASTScanner) redact(match string) string {
	if len(match) <= 8 {
		return match
	}
	return match[:4] + "..." + match[len(match)-4:]
}

func (s *SASTScanner) getContext(lines []string, lineIndex int) string {
	start := lineIndex - 2
	end := lineIndex + 3
	if start < 0 {
		start = 0
	}
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start:end], "\n")
}

type codeChunk struct {
	FilePath  string
	Content   string
	StartLine int
	Language  string
}

func detectLanguage(filePath string) string {
	switch {
	case strings.HasSuffix(filePath, ".go"):
		return "go"
	case strings.HasSuffix(filePath, ".ts") || strings.HasSuffix(filePath, ".tsx"):
		return "typescript"
	case strings.HasSuffix(filePath, ".js") || strings.HasSuffix(filePath, ".jsx"):
		return "javascript"
	case strings.HasSuffix(filePath, ".py"):
		return "python"
	case strings.HasSuffix(filePath, ".java"):
		return "java"
	case strings.HasSuffix(filePath, ".php"):
		return "php"
	case strings.HasSuffix(filePath, ".rb"):
		return "ruby"
	case strings.HasSuffix(filePath, ".cs"):
		return "csharp"
	default:
		return "unknown"
	}
}

func containsLanguage(languages []string, lang string) bool {
	for _, l := range languages {
		if l == lang || l == "all" {
			return true
		}
	}
	return false
}

func isSeverityAtLeast(sev, minSev Severity) bool {
	severityRanks := map[Severity]int{
		SeverityCritical: 5,
		SeverityHigh:     4,
		SeverityMedium:   3,
		SeverityLow:      2,
		SeverityInfo:     1,
	}
	return severityRanks[sev] >= severityRanks[minSev]
}

// defaultSASTPatterns returns built-in SAST detection patterns.
func defaultSASTPatterns() []SASTPattern {
	patterns := []SASTPattern{
		// SQL Injection patterns
		{
			ID:          "sql-injection-1",
			Name:        "SQL Injection - String Concatenation",
			Description: "Potential SQL injection via string concatenation",
			Type:        VulnSQLInjection,
			Pattern:     `(?i)(execute|query|exec)\s*\(\s*["'].*\+.*["']`,
			Severity:    SeverityCritical,
			CWE:         "CWE-89",
			OWASP:       "A03:2021-Injection",
			Languages:   []string{"all"},
			Suggestion:  "Use parameterized queries or prepared statements instead of string concatenation",
		},
		{
			ID:          "sql-injection-2",
			Name:        "SQL Injection - Format String",
			Description: "Potential SQL injection via format string",
			Type:        VulnSQLInjection,
			Pattern:     `(?i)(sprintf|fmt\.Sprintf|format|f["'])\s*\([^)]*SELECT|INSERT|UPDATE|DELETE|DROP`,
			Severity:    SeverityCritical,
			CWE:         "CWE-89",
			OWASP:       "A03:2021-Injection",
			Languages:   []string{"go", "python", "php"},
			Suggestion:  "Use parameterized queries instead of formatting SQL strings",
		},
		// XSS patterns
		{
			ID:          "xss-1",
			Name:        "XSS - innerHTML Assignment",
			Description: "Potential XSS via innerHTML",
			Type:        VulnXSS,
			Pattern:     `(?i)\.innerHTML\s*=`,
			Severity:    SeverityHigh,
			CWE:         "CWE-79",
			OWASP:       "A03:2021-Injection",
			Languages:   []string{"javascript", "typescript"},
			Suggestion:  "Use textContent or sanitize input before using innerHTML",
			FalsePositive: []string{"sanitize", "escape", "DOMPurify"},
		},
		{
			ID:          "xss-2",
			Name:        "XSS - document.write",
			Description: "Potential XSS via document.write",
			Type:        VulnXSS,
			Pattern:     `(?i)document\.write\s*\(`,
			Severity:    SeverityHigh,
			CWE:         "CWE-79",
			OWASP:       "A03:2021-Injection",
			Languages:   []string{"javascript", "typescript"},
			Suggestion:  "Avoid document.write; use DOM manipulation instead",
		},
		{
			ID:          "xss-3",
			Name:        "XSS - eval Usage",
			Description: "Dangerous use of eval() function",
			Type:        VulnXSS,
			Pattern:     `(?i)\beval\s*\(`,
			Severity:    SeverityCritical,
			CWE:         "CWE-95",
			OWASP:       "A03:2021-Injection",
			Languages:   []string{"javascript", "typescript", "python"},
			Suggestion:  "Avoid eval(); use safer alternatives like JSON.parse()",
		},
		// Path Traversal patterns
		{
			ID:          "path-traversal-1",
			Name:        "Path Traversal - Dot-Dot-Slash",
			Description: "Potential path traversal vulnerability",
			Type:        VulnPathTraversal,
			Pattern:     `\.\.[\\/]`,
			Severity:    SeverityMedium,
			CWE:         "CWE-22",
			OWASP:       "A01:2021-Broken Access Control",
			Languages:   []string{"all"},
			Suggestion:  "Validate and sanitize file paths; use path.Clean() or similar",
			FalsePositive: []string{"test", "example", "fixture", "mock"},
		},
		{
			ID:          "path-traversal-2",
			Name:        "Path Traversal - User Input in File Path",
			Description: "User input used directly in file path",
			Type:        VulnPathTraversal,
			Pattern:     `(?i)(os\.Open|ioutil\.ReadFile|filepath\.Join|open|read_file)\s*\([^)]*\$|req\.|request\.|params\.|query\.`,
			Severity:    SeverityHigh,
			CWE:         "CWE-22",
			OWASP:       "A01:2021-Broken Access Control",
			Languages:   []string{"go", "python", "javascript"},
			Suggestion:  "Validate file paths against an allowlist; use path.Clean() and check for '..' sequences",
		},
		// Command Injection patterns
		{
			ID:          "cmd-injection-1",
			Name:        "Command Injection - exec/system",
			Description: "Potential command injection via exec or system",
			Type:        VulnCommandInjection,
			Pattern:     `(?i)(exec|system|shell_exec|popen|subprocess\.call|os\.system|child_process)\s*\(`,
			Severity:    SeverityCritical,
			CWE:         "CWE-78",
			OWASP:       "A03:2021-Injection",
			Languages:   []string{"all"},
			Suggestion:  "Avoid shell commands; if needed, use parameterized APIs and validate input",
		},
		// Hardcoded IPs and URLs
		{
			ID:          "hardcoded-ip-1",
			Name:        "Hardcoded IPv4 Address",
			Description: "Hardcoded IP address found",
			Type:        VulnHardcodedIP,
			Pattern:     `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`,
			Severity:    SeverityLow,
			CWE:         "CWE-798",
			Languages:   []string{"all"},
			Suggestion:  "Use environment variables or configuration files for IP addresses",
			FalsePositive: []string{"127.0.0.1", "0.0.0.0", "localhost", "test", "example"},
		},
		{
			ID:          "hardcoded-url-1",
			Name:        "Hardcoded URL",
			Description: "Hardcoded URL found",
			Type:        VulnHardcodedURL,
			Pattern:     `https?://[a-zA-Z0-9\-\.]+\.[a-zA-Z]{2,}/`,
			Severity:    SeverityLow,
			CWE:         "CWE-798",
			Languages:   []string{"all"},
			Suggestion:  "Use environment variables or configuration for URLs",
			FalsePositive: []string{"example.com", "localhost", "test.com", "github.com", "golang.org"},
		},
		// Insecure Crypto patterns
		{
			ID:          "insecure-crypto-md5",
			Name:        "Insecure Cryptography - MD5",
			Description: "MD5 is not cryptographically secure",
			Type:        VulnInsecureCrypto,
			Pattern:     `(?i)\bmd5\b`,
			Severity:    SeverityMedium,
			CWE:         "CWE-328",
			Languages:   []string{"all"},
			Suggestion:  "Use SHA-256 or stronger algorithms for cryptographic purposes",
			FalsePositive: []string{"checksum", "hash for cache"},
		},
		{
			ID:          "insecure-crypto-sha1",
			Name:        "Insecure Cryptography - SHA1",
			Description: "SHA1 is considered weak for cryptographic purposes",
			Type:        VulnInsecureCrypto,
			Pattern:     `(?i)\bsha1\b`,
			Severity:    SeverityMedium,
			CWE:         "CWE-328",
			Languages:   []string{"all"},
			Suggestion:  "Use SHA-256 or stronger algorithms",
			FalsePositive: []string{"git", "checksum"},
		},
		{
			ID:          "insecure-crypto-des",
			Name:        "Insecure Cryptography - DES/3DES",
			Description: "DES/3DES are deprecated encryption algorithms",
			Type:        VulnInsecureCrypto,
			Pattern:     `(?i)\b(des|3des|triple.?des)\b`,
			Severity:    SeverityHigh,
			CWE:         "CWE-327",
			Languages:   []string{"all"},
			Suggestion:  "Use AES-256 or other modern encryption algorithms",
		},
		// Insecure Random patterns
		{
			ID:          "insecure-random-1",
			Name:        "Insecure Random - Math.random",
			Description: "Math.random() is not cryptographically secure",
			Type:        VulnInsecureRandom,
			Pattern:     `Math\.random\s*\(`,
			Severity:    SeverityMedium,
			CWE:         "CWE-330",
			Languages:   []string{"javascript", "typescript"},
			Suggestion:  "Use crypto.getRandomValues() or crypto.randomBytes() for security-sensitive operations",
		},
		{
			ID:          "insecure-random-2",
			Name:        "Insecure Random - rand()",
			Description: "rand() is not cryptographically secure",
			Type:        VulnInsecureRandom,
			Pattern:     `(?i)\brand\s*\(`,
			Severity:    SeverityMedium,
			CWE:         "CWE-330",
			Languages:   []string{"go", "python", "php"},
			Suggestion:  "Use crypto/rand package for security-sensitive operations",
			FalsePositive: []string{"math/rand", "test"},
		},
		// Open Redirect patterns
		{
			ID:          "open-redirect-1",
			Name:        "Open Redirect - Unvalidated Redirect",
			Description: "Potential open redirect vulnerability",
			Type:        VulnOpenRedirect,
			Pattern:     `(?i)(redirect|location\.href|window\.location)\s*=\s*[^"']+\$`,
			Severity:    SeverityMedium,
			CWE:         "CWE-601",
			OWASP:       "A01:2021-Broken Access Control",
			Languages:   []string{"javascript", "typescript", "php"},
			Suggestion:  "Validate redirect URLs against an allowlist",
		},
		// SSRF patterns
		{
			ID:          "ssrf-1",
			Name:        "SSRF - Dynamic URL Fetch",
			Description: "Potential SSRF via dynamic URL",
			Type:        VulnSSRF,
			Pattern:     `(?i)(fetch|axios|http\.get|requests\.get|urllib)\s*\([^)]*\$|req\.|request\.`,
			Severity:    SeverityHigh,
			CWE:         "CWE-918",
			OWASP:       "A10:2021-SSRF",
			Languages:   []string{"javascript", "typescript", "python", "go"},
			Suggestion:  "Validate URLs against an allowlist; block internal IP ranges",
		},
		// XXE patterns
		{
			ID:          "xxe-1",
			Name:        "XXE - Unsafe XML Parser",
			Description: "Potential XXE vulnerability",
			Type:        VulnXXE,
			Pattern:     `(?i)(XMLParser|etree\.parse|xml\.parse|DocumentBuilder|SAXParser)`,
			Severity:    SeverityHigh,
			CWE:         "CWE-611",
			OWASP:       "A05:2017-XXE",
			Languages:   []string{"java", "python", "go"},
			Suggestion:  "Disable external entity processing in XML parsers",
		},
		// Insecure Deserialization patterns
		{
			ID:          "deserial-1",
			Name:        "Insecure Deserialization - pickle",
			Description: "Unsafe pickle deserialization",
			Type:        VulnInsecureDeserial,
			Pattern:     `(?i)pickle\.loads?\s*\(`,
			Severity:    SeverityCritical,
			CWE:         "CWE-502",
			OWASP:       "A08:2017-Insecure Deserialization",
			Languages:   []string{"python"},
			Suggestion:  "Avoid pickle for untrusted data; use JSON or other safe formats",
		},
		{
			ID:          "deserial-2",
			Name:        "Insecure Deserialization - yaml.load",
			Description: "Unsafe YAML loading",
			Type:        VulnInsecureDeserial,
			Pattern:     `(?i)yaml\.load\s*\([^)]*Loader\s*=\s*yaml\.Loader`,
			Severity:    SeverityHigh,
			CWE:         "CWE-502",
			OWASP:       "A08:2017-Insecure Deserialization",
			Languages:   []string{"python"},
			Suggestion:  "Use yaml.safe_load() instead of yaml.load()",
		},
		{
			ID:          "deserial-3",
			Name:        "Insecure Deserialization - gob",
			Description: "Unsafe gob deserialization",
			Type:        VulnInsecureDeserial,
			Pattern:     `gob\.NewDecoder\s*\(`,
			Severity:    SeverityMedium,
			CWE:         "CWE-502",
			Languages:   []string{"go"},
			Suggestion:  "Validate input before gob deserialization; prefer JSON for untrusted data",
		},
		// Hardcoded Credentials patterns
		{
			ID:          "hardcoded-creds-1",
			Name:        "Hardcoded Credentials - Password Variable",
			Description: "Potential hardcoded password",
			Type:        VulnHardcodedCreds,
			Pattern:     `(?i)(password|passwd|pwd)\s*[=:]\s*["'][^"']+["']`,
			Severity:    SeverityCritical,
			CWE:         "CWE-798",
			Languages:   []string{"all"},
			Suggestion:  "Use environment variables or secrets management for credentials",
			FalsePositive: []string{"example", "test", "placeholder", "changeme", "xxx"},
		},
		{
			ID:          "hardcoded-creds-2",
			Name:        "Hardcoded Credentials - Database Connection",
			Description: "Potential hardcoded database credentials",
			Type:        VulnHardcodedCreds,
			Pattern:     `(?i)(mysql|postgres|mongodb|redis)://[^:]+:[^@]+@`,
			Severity:    SeverityCritical,
			CWE:         "CWE-798",
			Languages:   []string{"all"},
			Suggestion:  "Use environment variables for database connection strings",
		},
	}

	// Compile patterns
	for i := range patterns {
		if patterns[i].Pattern != "" {
			patterns[i].Regex = regexp.MustCompile(patterns[i].Pattern)
		}
	}

	return patterns
}

