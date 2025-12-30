// Package security provides clients for security vulnerability databases.
package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Vulnerability represents a security vulnerability in a dependency.
type Vulnerability struct {
	ID          string   `json:"id"`          // CVE or GHSA ID
	Summary     string   `json:"summary"`     // Short description
	Details     string   `json:"details"`     // Full description
	Severity    string   `json:"severity"`    // "critical", "high", "medium", "low"
	FixedIn     string   `json:"fixed_in"`    // Version that fixes this
	References  []string `json:"references"`  // Links to advisories
	PublishedAt string   `json:"published_at"`
}

// OSVClient interacts with the OSV.dev vulnerability database.
type OSVClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewOSVClient creates a new OSV client.
func NewOSVClient(logger *logrus.Logger) *OSVClient {
	return &OSVClient{
		baseURL: "https://api.osv.dev/v1",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// OSVQueryRequest is the request format for OSV API.
type OSVQueryRequest struct {
	Package   OSVPackage `json:"package"`
	Version   string     `json:"version"`
}

// OSVPackage represents a package in OSV format.
type OSVPackage struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

// OSVQueryResponse is the response format from OSV API.
type OSVQueryResponse struct {
	Vulns []OSVVulnerability `json:"vulns"`
}

// OSVVulnerability represents a vulnerability from OSV.
type OSVVulnerability struct {
	ID        string           `json:"id"`
	Summary   string           `json:"summary"`
	Details   string           `json:"details"`
	Aliases   []string         `json:"aliases"` // CVE IDs
	Severity  []OSVSeverity    `json:"severity"`
	Affected  []OSVAffected    `json:"affected"`
	References []OSVReference  `json:"references"`
	Published string           `json:"published"`
	Modified  string           `json:"modified"`
}

// OSVSeverity represents severity info.
type OSVSeverity struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}

// OSVAffected represents affected versions.
type OSVAffected struct {
	Package  OSVPackage   `json:"package"`
	Ranges   []OSVRange   `json:"ranges"`
	Versions []string     `json:"versions"`
}

// OSVRange represents a version range.
type OSVRange struct {
	Type   string      `json:"type"`
	Events []OSVEvent  `json:"events"`
}

// OSVEvent represents a version event (introduced/fixed).
type OSVEvent struct {
	Introduced string `json:"introduced,omitempty"`
	Fixed      string `json:"fixed,omitempty"`
}

// OSVReference represents a reference link.
type OSVReference struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// QueryVulnerabilities checks if a package version has known vulnerabilities.
func (c *OSVClient) QueryVulnerabilities(ctx context.Context, ecosystem, packageName, version string) ([]Vulnerability, error) {
	reqBody := OSVQueryRequest{
		Package: OSVPackage{
			Name:      packageName,
			Ecosystem: ecosystem, // "Go", "npm", "PyPI", etc.
		},
		Version: version,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/query"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OSV query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSV returned status %d", resp.StatusCode)
	}

	var osvResp OSVQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&osvResp); err != nil {
		return nil, fmt.Errorf("failed to decode OSV response: %w", err)
	}

	return c.convertVulnerabilities(osvResp.Vulns), nil
}

// QueryBatch queries vulnerabilities for multiple packages at once.
func (c *OSVClient) QueryBatch(ctx context.Context, queries []OSVQueryRequest) (map[string][]Vulnerability, error) {
	type batchRequest struct {
		Queries []OSVQueryRequest `json:"queries"`
	}

	bodyBytes, err := json.Marshal(batchRequest{Queries: queries})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch request: %w", err)
	}

	url := c.baseURL + "/querybatch"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OSV batch query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSV returned status %d", resp.StatusCode)
	}

	type batchResponse struct {
		Results []OSVQueryResponse `json:"results"`
	}

	var batchResp batchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, fmt.Errorf("failed to decode OSV batch response: %w", err)
	}

	results := make(map[string][]Vulnerability)
	for i, result := range batchResp.Results {
		if i < len(queries) {
			key := queries[i].Package.Name + "@" + queries[i].Version
			results[key] = c.convertVulnerabilities(result.Vulns)
		}
	}

	return results, nil
}

// convertVulnerabilities converts OSV vulnerabilities to our format.
func (c *OSVClient) convertVulnerabilities(osvVulns []OSVVulnerability) []Vulnerability {
	var vulns []Vulnerability

	for _, osv := range osvVulns {
		vuln := Vulnerability{
			ID:          c.getPrimaryID(osv),
			Summary:     osv.Summary,
			Details:     osv.Details,
			Severity:    c.determineSeverity(osv),
			FixedIn:     c.getFixedVersion(osv),
			References:  c.getReferences(osv),
			PublishedAt: osv.Published,
		}
		vulns = append(vulns, vuln)
	}

	return vulns
}

// getPrimaryID returns CVE if available, otherwise the OSV ID.
func (c *OSVClient) getPrimaryID(osv OSVVulnerability) string {
	for _, alias := range osv.Aliases {
		if len(alias) > 3 && alias[:3] == "CVE" {
			return alias
		}
	}
	return osv.ID
}

// determineSeverity determines severity from CVSS or other indicators.
func (c *OSVClient) determineSeverity(osv OSVVulnerability) string {
	for _, sev := range osv.Severity {
		if sev.Type == "CVSS_V3" {
			score := parseCVSSScore(sev.Score)
			if score >= 9.0 {
				return "critical"
			} else if score >= 7.0 {
				return "high"
			} else if score >= 4.0 {
				return "medium"
			}
			return "low"
		}
	}

	// Default based on keywords in summary
	summary := osv.Summary + " " + osv.Details
	if containsAny(summary, []string{"remote code execution", "rce", "arbitrary code"}) {
		return "critical"
	}
	if containsAny(summary, []string{"denial of service", "dos", "crash"}) {
		return "high"
	}

	return "medium"
}

// getFixedVersion extracts the version that fixes the vulnerability.
func (c *OSVClient) getFixedVersion(osv OSVVulnerability) string {
	for _, affected := range osv.Affected {
		for _, r := range affected.Ranges {
			for _, event := range r.Events {
				if event.Fixed != "" {
					return event.Fixed
				}
			}
		}
	}
	return ""
}

// getReferences extracts reference URLs.
func (c *OSVClient) getReferences(osv OSVVulnerability) []string {
	var refs []string
	for _, ref := range osv.References {
		refs = append(refs, ref.URL)
	}
	return refs
}

// parseCVSSScore parses CVSS score from string.
func parseCVSSScore(cvss string) float64 {
	// CVSS format: "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H" or just "9.8"
	var score float64
	fmt.Sscanf(cvss, "%f", &score)
	return score
}

// containsAny checks if text contains any of the keywords (case-insensitive).
func containsAny(text string, keywords []string) bool {
	text = strings.ToLower(text)
	for _, kw := range keywords {
		if strings.Contains(text, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

