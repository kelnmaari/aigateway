// Package registry provides clients for package registries.
package registry

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// MavenClient interacts with Maven Central repository.
type MavenClient struct {
	searchURL  string // search.maven.org (API)
	repoURL    string // repo1.maven.org (metadata)
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewMavenClient creates a new Maven Central client.
func NewMavenClient(logger *logrus.Logger) *MavenClient {
	return &MavenClient{
		searchURL: "https://search.maven.org/solrsearch/select",
		repoURL:   "https://repo1.maven.org/maven2",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// mavenSearchResponse represents Maven Central search API response.
type mavenSearchResponse struct {
	Response struct {
		NumFound int `json:"numFound"`
		Docs     []struct {
			ID            string `json:"id"`
			GroupID       string `json:"g"`
			ArtifactID    string `json:"a"`
			LatestVersion string `json:"latestVersion"`
			Version       string `json:"v"`
			Timestamp     int64  `json:"timestamp"`
		} `json:"docs"`
	} `json:"response"`
}

// mavenMetadata represents maven-metadata.xml structure.
type mavenMetadata struct {
	XMLName    xml.Name `xml:"metadata"`
	GroupID    string   `xml:"groupId"`
	ArtifactID string   `xml:"artifactId"`
	Versioning struct {
		Latest   string `xml:"latest"`
		Release  string `xml:"release"`
		Versions struct {
			Version []string `xml:"version"`
		} `xml:"versions"`
		LastUpdated string `xml:"lastUpdated"`
	} `xml:"versioning"`
}

// GetLatestVersion returns the latest version for a Maven artifact.
// name format: "groupId:artifactId"
func (c *MavenClient) GetLatestVersion(ctx context.Context, name string) (string, error) {
	parts := strings.SplitN(name, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid Maven coordinate format: %s (expected groupId:artifactId)", name)
	}
	groupID, artifactID := parts[0], parts[1]

	// Try maven-metadata.xml first (more reliable)
	latest, err := c.getLatestFromMetadata(ctx, groupID, artifactID)
	if err == nil && latest != "" {
		return latest, nil
	}

	// Fallback to search API
	return c.getLatestFromSearch(ctx, groupID, artifactID)
}

// getLatestFromMetadata fetches latest version from maven-metadata.xml.
func (c *MavenClient) getLatestFromMetadata(ctx context.Context, groupID, artifactID string) (string, error) {
	// Convert groupId to path: com.google.code.gson -> com/google/code/gson
	groupPath := strings.ReplaceAll(groupID, ".", "/")
	url := fmt.Sprintf("%s/%s/%s/maven-metadata.xml", c.repoURL, groupPath, artifactID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch maven-metadata.xml: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("maven-metadata.xml not found (status %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var metadata mavenMetadata
	if err := xml.Unmarshal(body, &metadata); err != nil {
		return "", fmt.Errorf("failed to parse maven-metadata.xml: %w", err)
	}

	// Prefer release, then latest, then find latest stable from versions
	if metadata.Versioning.Release != "" {
		return metadata.Versioning.Release, nil
	}
	if metadata.Versioning.Latest != "" {
		return metadata.Versioning.Latest, nil
	}

	// Find latest stable version
	versions := metadata.Versioning.Versions.Version
	if len(versions) > 0 {
		return c.findLatestStable(versions), nil
	}

	return "", fmt.Errorf("no versions found in maven-metadata.xml")
}

// getLatestFromSearch fetches latest version from search API.
func (c *MavenClient) getLatestFromSearch(ctx context.Context, groupID, artifactID string) (string, error) {
	url := fmt.Sprintf("%s?q=g:%s+AND+a:%s&rows=1&wt=json", c.searchURL, groupID, artifactID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to search Maven Central: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Maven Central search failed: %s (status %d)", string(body), resp.StatusCode)
	}

	var searchResp mavenSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return "", fmt.Errorf("failed to decode search response: %w", err)
	}

	if searchResp.Response.NumFound == 0 || len(searchResp.Response.Docs) == 0 {
		return "", fmt.Errorf("artifact not found: %s:%s", groupID, artifactID)
	}

	doc := searchResp.Response.Docs[0]
	if doc.LatestVersion != "" {
		return doc.LatestVersion, nil
	}
	return doc.Version, nil
}

// ListVersions returns all available versions for an artifact.
func (c *MavenClient) ListVersions(ctx context.Context, name string) ([]string, error) {
	parts := strings.SplitN(name, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid Maven coordinate format: %s", name)
	}
	groupID, artifactID := parts[0], parts[1]

	groupPath := strings.ReplaceAll(groupID, ".", "/")
	url := fmt.Sprintf("%s/%s/%s/maven-metadata.xml", c.repoURL, groupPath, artifactID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("maven-metadata.xml not found")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var metadata mavenMetadata
	if err := xml.Unmarshal(body, &metadata); err != nil {
		return nil, err
	}

	return metadata.Versioning.Versions.Version, nil
}

// CompareVersions determines the update type between two Maven versions.
func (c *MavenClient) CompareVersions(current, latest string) string {
	currentParts := c.parseVersion(current)
	latestParts := c.parseVersion(latest)

	// Compare major
	if latestParts[0] > currentParts[0] {
		return "major"
	}
	if latestParts[0] < currentParts[0] {
		return "none"
	}

	// Compare minor
	if latestParts[1] > currentParts[1] {
		return "minor"
	}
	if latestParts[1] < currentParts[1] {
		return "none"
	}

	// Compare patch
	if latestParts[2] > currentParts[2] {
		return "patch"
	}

	return "none"
}

// parseVersion parses version string into [major, minor, patch].
func (c *MavenClient) parseVersion(v string) [3]int {
	var result [3]int

	// Remove common prefixes
	v = strings.TrimPrefix(v, "v")

	// Split by common delimiters
	parts := strings.FieldsFunc(v, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	})

	for i := 0; i < len(parts) && i < 3; i++ {
		// Extract numeric part
		var numStr strings.Builder
		for _, r := range parts[i] {
			if r >= '0' && r <= '9' {
				numStr.WriteString(string(r))
			} else {
				break
			}
		}
		if numStr.String() != "" {
			if num, err := strconv.Atoi(numStr.String()); err == nil {
				result[i] = num
			}
		}
	}

	return result
}

// findLatestStable finds the latest stable (non-prerelease) version.
func (c *MavenClient) findLatestStable(versions []string) string {
	// Sort versions in descending order
	sorted := make([]string, len(versions))
	copy(sorted, versions)

	sort.Slice(sorted, func(i, j int) bool {
		iParts := c.parseVersion(sorted[i])
		jParts := c.parseVersion(sorted[j])

		for k := range 3 {
			if iParts[k] != jParts[k] {
				return iParts[k] > jParts[k]
			}
		}
		return false
	})

	// Find first non-prerelease
	for _, v := range sorted {
		lower := strings.ToLower(v)
		if !strings.Contains(lower, "-alpha") &&
			!strings.Contains(lower, "-beta") &&
			!strings.Contains(lower, "-rc") &&
			!strings.Contains(lower, "-snapshot") &&
			!strings.Contains(lower, "-m") { // Milestone
			return v
		}
	}

	// If all are prereleases, return the latest
	if len(sorted) > 0 {
		return sorted[0]
	}

	return ""
}
