// Package registry provides clients for package registries.
package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// NPMClient interacts with the npm registry (registry.npmjs.org).
type NPMClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewNPMClient creates a new npm registry client.
func NewNPMClient(logger *logrus.Logger) *NPMClient {
	return &NPMClient{
		baseURL: "https://registry.npmjs.org",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// NPMPackageInfo represents npm package metadata
type NPMPackageInfo struct {
	Name     string                    `json:"name"`
	DistTags map[string]string         `json:"dist-tags"`
	Versions map[string]NPMVersionInfo `json:"versions"`
	Time     map[string]string         `json:"time"`
}

// NPMVersionInfo represents version-specific info
type NPMVersionInfo struct {
	Version string `json:"version"`
}

// GetLatestVersion returns the latest version for an npm package.
func (c *NPMClient) GetLatestVersion(ctx context.Context, packageName string) (string, error) {
	// Use abbreviated metadata endpoint for faster response
	url := fmt.Sprintf("%s/%s", c.baseURL, packageName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	// Request abbreviated metadata
	req.Header.Set("Accept", "application/vnd.npm.install-v1+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch package info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("package not found: %s", packageName)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("npm registry returned status %d", resp.StatusCode)
	}

	var pkgInfo NPMPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&pkgInfo); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Get "latest" from dist-tags
	if latest, ok := pkgInfo.DistTags["latest"]; ok {
		return latest, nil
	}

	// Fallback: find the highest version
	if len(pkgInfo.Versions) > 0 {
		var versions []string
		for v := range pkgInfo.Versions {
			versions = append(versions, v)
		}
		sort.Slice(versions, func(i, j int) bool {
			return compareSemver(versions[i], versions[j]) > 0
		})
		return versions[0], nil
	}

	return "", fmt.Errorf("no versions found for %s", packageName)
}

// CompareVersions determines the update type between two npm versions.
func (c *NPMClient) CompareVersions(current, latest string) string {
	// Normalize versions
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")

	cmp := compareSemver(current, latest)
	if cmp >= 0 {
		return "none" // current is same or newer
	}

	// Parse major.minor.patch
	currentParts := parseSemver(current)
	latestParts := parseSemver(latest)

	if currentParts[0] != latestParts[0] {
		return "major"
	}
	if currentParts[1] != latestParts[1] {
		return "minor"
	}
	return "patch"
}

// compareSemver compares two semver versions.
// Returns: >0 if a > b, <0 if a < b, 0 if equal
func compareSemver(a, b string) int {
	aParts := parseSemver(a)
	bParts := parseSemver(b)

	for i := 0; i < 3; i++ {
		if aParts[i] > bParts[i] {
			return 1
		}
		if aParts[i] < bParts[i] {
			return -1
		}
	}
	return 0
}

// parseSemver parses a version string into [major, minor, patch]
func parseSemver(version string) [3]int {
	// Remove any prerelease suffix
	if idx := strings.IndexAny(version, "-+"); idx >= 0 {
		version = version[:idx]
	}

	parts := strings.Split(version, ".")
	var result [3]int

	for i := 0; i < 3 && i < len(parts); i++ {
		fmt.Sscanf(parts[i], "%d", &result[i])
	}

	return result
}

