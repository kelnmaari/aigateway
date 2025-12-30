// Package registry provides clients for package registries.
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/mod/semver"
)

// GolangClient interacts with the Go module proxy (proxy.golang.org).
type GolangClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewGolangClient creates a new Go proxy client.
func NewGolangClient(logger *logrus.Logger) *GolangClient {
	return &GolangClient{
		baseURL: "https://proxy.golang.org",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// VersionInfo contains version details.
type VersionInfo struct {
	Version string    `json:"Version"`
	Time    time.Time `json:"Time"`
}

// GetLatestVersion returns the latest version for a module.
func (c *GolangClient) GetLatestVersion(ctx context.Context, modulePath string) (string, error) {
	// Try to get @latest first
	url := fmt.Sprintf("%s/%s/@latest", c.baseURL, escapePath(modulePath))
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest version: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		var info VersionInfo
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			return "", fmt.Errorf("failed to decode version info: %w", err)
		}
		return info.Version, nil
	}
	
	// If @latest fails, get all versions and find the latest
	versions, err := c.ListVersions(ctx, modulePath)
	if err != nil {
		return "", err
	}
	
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions found for %s", modulePath)
	}
	
	// Sort versions and return the latest stable
	return c.findLatestStable(versions), nil
}

// ListVersions returns all available versions for a module.
func (c *GolangClient) ListVersions(ctx context.Context, modulePath string) ([]string, error) {
	url := fmt.Sprintf("%s/%s/@v/list", c.baseURL, escapePath(modulePath))
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list versions: %s (status %d)", string(body), resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	var versions []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			versions = append(versions, line)
		}
	}
	
	return versions, nil
}

// GetVersionInfo returns detailed info about a specific version.
func (c *GolangClient) GetVersionInfo(ctx context.Context, modulePath, version string) (*VersionInfo, error) {
	url := fmt.Sprintf("%s/%s/@v/%s.info", c.baseURL, escapePath(modulePath), version)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get version info: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("version not found: %s@%s", modulePath, version)
	}
	
	var info VersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	
	return &info, nil
}

// CompareVersions determines the update type between two versions.
func (c *GolangClient) CompareVersions(current, latest string) string {
	// Ensure versions have 'v' prefix
	if !strings.HasPrefix(current, "v") {
		current = "v" + current
	}
	if !strings.HasPrefix(latest, "v") {
		latest = "v" + latest
	}
	
	cmp := semver.Compare(current, latest)
	if cmp >= 0 {
		return "none" // current is same or newer
	}
	
	// Parse major.minor.patch
	currentMajor := semver.Major(current)
	latestMajor := semver.Major(latest)
	if currentMajor != latestMajor {
		return "major"
	}
	
	currentMinor := extractMinor(current)
	latestMinor := extractMinor(latest)
	if currentMinor != latestMinor {
		return "minor"
	}
	
	return "patch"
}

// findLatestStable finds the latest stable (non-prerelease) version.
func (c *GolangClient) findLatestStable(versions []string) string {
	// Sort versions in descending order
	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare(versions[i], versions[j]) > 0
	})
	
	// Find first non-prerelease
	for _, v := range versions {
		if semver.Prerelease(v) == "" {
			return v
		}
	}
	
	// If all are prereleases, return the latest one
	if len(versions) > 0 {
		return versions[0]
	}
	
	return ""
}

// escapePath escapes module path for URL.
func escapePath(path string) string {
	// Go proxy uses case-encoded paths
	// Uppercase letters are escaped as !lowercase
	var result strings.Builder
	for _, r := range path {
		if r >= 'A' && r <= 'Z' {
			result.WriteRune('!')
			result.WriteRune(r + 32) // to lowercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// extractMinor extracts the minor version number.
func extractMinor(v string) string {
	// v1.2.3 -> 1.2
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return v
}

