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

// PyPIClient interacts with the PyPI registry (pypi.org).
type PyPIClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewPyPIClient creates a new PyPI registry client.
func NewPyPIClient(logger *logrus.Logger) *PyPIClient {
	return &PyPIClient{
		baseURL: "https://pypi.org/pypi",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// PyPIPackageInfo represents PyPI package metadata
type PyPIPackageInfo struct {
	Info     PyPIInfo         `json:"info"`
	Releases map[string][]any `json:"releases"`
}

// PyPIInfo represents package info
type PyPIInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"` // Latest version
}

// GetLatestVersion returns the latest version for a PyPI package.
func (c *PyPIClient) GetLatestVersion(ctx context.Context, packageName string) (string, error) {
	url := fmt.Sprintf("%s/%s/json", c.baseURL, packageName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch package info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("package not found: %s", packageName)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("pypi returned status %d", resp.StatusCode)
	}

	var pkgInfo PyPIPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&pkgInfo); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Return the latest version from info
	if pkgInfo.Info.Version != "" {
		return pkgInfo.Info.Version, nil
	}

	// Fallback: find highest version from releases
	if len(pkgInfo.Releases) > 0 {
		var versions []string
		for v := range pkgInfo.Releases {
			// Skip pre-release versions
			if !isPrerelease(v) {
				versions = append(versions, v)
			}
		}
		if len(versions) > 0 {
			sort.Slice(versions, func(i, j int) bool {
				return comparePyPIVersion(versions[i], versions[j]) > 0
			})
			return versions[0], nil
		}
	}

	return "", fmt.Errorf("no versions found for %s", packageName)
}

// CompareVersions determines the update type between two PyPI versions.
func (c *PyPIClient) CompareVersions(current, latest string) string {
	cmp := comparePyPIVersion(current, latest)
	if cmp >= 0 {
		return "none"
	}

	currentParts := parsePyPIVersion(current)
	latestParts := parsePyPIVersion(latest)

	if len(currentParts) > 0 && len(latestParts) > 0 {
		if currentParts[0] != latestParts[0] {
			return "major"
		}
		if len(currentParts) > 1 && len(latestParts) > 1 {
			if currentParts[1] != latestParts[1] {
				return "minor"
			}
		}
	}

	return "patch"
}

// isPrerelease checks if a version is a prerelease
func isPrerelease(version string) bool {
	v := strings.ToLower(version)
	return strings.Contains(v, "dev") ||
		strings.Contains(v, "alpha") ||
		strings.Contains(v, "beta") ||
		strings.Contains(v, "rc") ||
		strings.Contains(v, "pre")
}

// comparePyPIVersion compares two version strings
func comparePyPIVersion(a, b string) int {
	aParts := parsePyPIVersion(a)
	bParts := parsePyPIVersion(b)

	maxLen := max(len(bParts), len(aParts))

	for i := range maxLen {
		av := 0
		bv := 0
		if i < len(aParts) {
			av = aParts[i]
		}
		if i < len(bParts) {
			bv = bParts[i]
		}
		if av > bv {
			return 1
		}
		if av < bv {
			return -1
		}
	}
	return 0
}

// parsePyPIVersion parses a version string into parts
func parsePyPIVersion(version string) []int {
	// Remove prerelease suffix
	if idx := strings.IndexAny(version, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"); idx > 0 {
		version = version[:idx]
	}
	version = strings.TrimSuffix(version, ".")

	parts := strings.Split(version, ".")
	var result []int

	for _, p := range parts {
		var v int
		fmt.Sscanf(p, "%d", &v)
		result = append(result, v)
	}

	return result
}
