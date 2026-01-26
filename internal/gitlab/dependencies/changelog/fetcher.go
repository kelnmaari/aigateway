// Package changelog provides changelog fetching and LLM analysis for dependencies.
package changelog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Fetcher fetches changelogs from various sources.
type Fetcher struct {
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewFetcher creates a new changelog fetcher.
func NewFetcher(logger *logrus.Logger) *Fetcher {
	return &Fetcher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// FetchChangelog fetches changelog for a package.
func (f *Fetcher) FetchChangelog(ctx context.Context, packageName, currentVersion, latestVersion, language string) (*ChangelogInfo, error) {
	info := &ChangelogInfo{
		PackageName:    packageName,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		Language:       language,
		FetchedAt:      time.Now(),
	}

	var err error
	switch language {
	case "go":
		err = f.fetchGoChangelog(ctx, info)
	case "nodejs":
		err = f.fetchNPMChangelog(ctx, info)
	case "python":
		err = f.fetchPyPIChangelog(ctx, info)
	case "java":
		err = f.fetchJavaChangelog(ctx, info)
	default:
		return info, fmt.Errorf("unsupported language: %s", language)
	}

	if err != nil {
		f.logger.WithError(err).WithFields(logrus.Fields{
			"package":  packageName,
			"language": language,
		}).Debug("Failed to fetch changelog, will use minimal info")
		// Don't fail completely - return what we have
	}

	return info, nil
}

// fetchGoChangelog fetches changelog for a Go module from GitHub.
func (f *Fetcher) fetchGoChangelog(ctx context.Context, info *ChangelogInfo) error {
	// Most Go modules are on GitHub
	// Extract owner/repo from module path
	// e.g., github.com/gin-gonic/gin -> gin-gonic/gin
	owner, repo := f.extractGitHubRepo(info.PackageName)
	if owner == "" || repo == "" {
		return fmt.Errorf("could not extract GitHub repo from module path: %s", info.PackageName)
	}

	// Try to fetch release notes from GitHub API
	releaseNotes, err := f.fetchGitHubReleaseNotes(ctx, owner, repo, info.LatestVersion)
	if err == nil && releaseNotes != "" {
		info.ReleaseNotes = releaseNotes
		info.ChangelogURL = fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", owner, repo, info.LatestVersion)
		return nil
	}

	// Fallback: Try to fetch CHANGELOG.md
	changelogText, err := f.fetchGitHubFile(ctx, owner, repo, "CHANGELOG.md")
	if err == nil && changelogText != "" {
		// Extract relevant section for the version
		info.ChangelogText = f.extractVersionSection(changelogText, info.CurrentVersion, info.LatestVersion)
		info.ChangelogURL = fmt.Sprintf("https://github.com/%s/%s/blob/main/CHANGELOG.md", owner, repo)
		return nil
	}

	return fmt.Errorf("no changelog found for %s", info.PackageName)
}

// fetchNPMChangelog fetches changelog for an npm package.
func (f *Fetcher) fetchNPMChangelog(ctx context.Context, info *ChangelogInfo) error {
	// Get repository info from npm registry
	url := fmt.Sprintf("https://registry.npmjs.org/%s", info.PackageName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("npm registry returned %d", resp.StatusCode)
	}

	var npmInfo struct {
		Repository struct {
			URL string `json:"url"`
		} `json:"repository"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&npmInfo); err != nil {
		return err
	}

	// Extract GitHub repo from repository URL
	owner, repo := f.extractGitHubRepoFromURL(npmInfo.Repository.URL)
	if owner == "" || repo == "" {
		return fmt.Errorf("could not extract GitHub repo for %s", info.PackageName)
	}

	// Try release notes first
	releaseNotes, err := f.fetchGitHubReleaseNotes(ctx, owner, repo, "v"+info.LatestVersion)
	if err == nil && releaseNotes != "" {
		info.ReleaseNotes = releaseNotes
		info.ChangelogURL = fmt.Sprintf("https://github.com/%s/%s/releases/tag/v%s", owner, repo, info.LatestVersion)
		return nil
	}

	// Try without 'v' prefix
	releaseNotes, err = f.fetchGitHubReleaseNotes(ctx, owner, repo, info.LatestVersion)
	if err == nil && releaseNotes != "" {
		info.ReleaseNotes = releaseNotes
		info.ChangelogURL = fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", owner, repo, info.LatestVersion)
		return nil
	}

	// Fallback to CHANGELOG.md
	changelogText, err := f.fetchGitHubFile(ctx, owner, repo, "CHANGELOG.md")
	if err == nil && changelogText != "" {
		info.ChangelogText = f.extractVersionSection(changelogText, info.CurrentVersion, info.LatestVersion)
		info.ChangelogURL = fmt.Sprintf("https://github.com/%s/%s/blob/main/CHANGELOG.md", owner, repo)
		return nil
	}

	return fmt.Errorf("no changelog found for %s", info.PackageName)
}

// fetchPyPIChangelog fetches changelog for a PyPI package.
func (f *Fetcher) fetchPyPIChangelog(ctx context.Context, info *ChangelogInfo) error {
	// Get project URLs from PyPI
	url := fmt.Sprintf("https://pypi.org/pypi/%s/json", info.PackageName)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pypi returned %d", resp.StatusCode)
	}

	var pypiInfo struct {
		Info struct {
			ProjectURLs map[string]string `json:"project_urls"`
			HomePage    string            `json:"home_page"`
		} `json:"info"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pypiInfo); err != nil {
		return err
	}

	// Try to find GitHub URL from project URLs
	var githubURL string
	for key, val := range pypiInfo.Info.ProjectURLs {
		keyLower := strings.ToLower(key)
		if strings.Contains(keyLower, "source") || strings.Contains(keyLower, "repository") || strings.Contains(keyLower, "github") {
			if strings.Contains(val, "github.com") {
				githubURL = val
				break
			}
		}
	}

	if githubURL == "" && strings.Contains(pypiInfo.Info.HomePage, "github.com") {
		githubURL = pypiInfo.Info.HomePage
	}

	if githubURL == "" {
		return fmt.Errorf("no GitHub repo found for %s", info.PackageName)
	}

	owner, repo := f.extractGitHubRepoFromURL(githubURL)
	if owner == "" || repo == "" {
		return fmt.Errorf("could not extract GitHub repo from %s", githubURL)
	}

	// Try release notes
	releaseNotes, err := f.fetchGitHubReleaseNotes(ctx, owner, repo, "v"+info.LatestVersion)
	if err == nil && releaseNotes != "" {
		info.ReleaseNotes = releaseNotes
		info.ChangelogURL = fmt.Sprintf("https://github.com/%s/%s/releases/tag/v%s", owner, repo, info.LatestVersion)
		return nil
	}

	// Try without 'v' prefix
	releaseNotes, err = f.fetchGitHubReleaseNotes(ctx, owner, repo, info.LatestVersion)
	if err == nil && releaseNotes != "" {
		info.ReleaseNotes = releaseNotes
		info.ChangelogURL = fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", owner, repo, info.LatestVersion)
		return nil
	}

	// Fallback to CHANGELOG.md or HISTORY.md
	for _, filename := range []string{"CHANGELOG.md", "HISTORY.md", "CHANGES.md", "CHANGELOG.rst", "HISTORY.rst"} {
		changelogText, err := f.fetchGitHubFile(ctx, owner, repo, filename)
		if err == nil && changelogText != "" {
			info.ChangelogText = f.extractVersionSection(changelogText, info.CurrentVersion, info.LatestVersion)
			info.ChangelogURL = fmt.Sprintf("https://github.com/%s/%s/blob/main/%s", owner, repo, filename)
			return nil
		}
	}

	return fmt.Errorf("no changelog found for %s", info.PackageName)
}

// fetchGitHubReleaseNotes fetches release notes from GitHub API.
func (f *Fetcher) fetchGitHubReleaseNotes(ctx context.Context, owner, repo, tag string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, tag)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release struct {
		Body string `json:"body"`
		Name string `json:"name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	if release.Body != "" {
		return release.Body, nil
	}

	return release.Name, nil
}

// fetchGitHubFile fetches a file from GitHub repository.
func (f *Fetcher) fetchGitHubFile(ctx context.Context, owner, repo, filename string) (string, error) {
	// Try main branch first, then master
	for _, branch := range []string{"main", "master"} {
		url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, branch, filename)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}

		resp, err := f.httpClient.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			return string(body), nil
		}
	}

	return "", fmt.Errorf("file %s not found in %s/%s", filename, owner, repo)
}

// extractGitHubRepo extracts owner/repo from Go module path.
func (f *Fetcher) extractGitHubRepo(modulePath string) (string, string) {
	// github.com/owner/repo/... -> owner, repo
	if !strings.HasPrefix(modulePath, "github.com/") {
		return "", ""
	}

	parts := strings.Split(strings.TrimPrefix(modulePath, "github.com/"), "/")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}

	return "", ""
}

// extractGitHubRepoFromURL extracts owner/repo from a GitHub URL.
func (f *Fetcher) extractGitHubRepoFromURL(url string) (string, string) {
	// Handle various URL formats:
	// https://github.com/owner/repo
	// git+https://github.com/owner/repo.git
	// git://github.com/owner/repo.git
	// github.com/owner/repo

	// Remove common prefixes and suffixes
	url = strings.TrimPrefix(url, "git+")
	url = strings.TrimPrefix(url, "git://")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")

	if !strings.HasPrefix(url, "github.com/") {
		return "", ""
	}

	parts := strings.Split(strings.TrimPrefix(url, "github.com/"), "/")
	if len(parts) >= 2 {
		return parts[0], parts[1]
	}

	return "", ""
}

// extractVersionSection extracts changelog section between two versions.
func (f *Fetcher) extractVersionSection(changelog, currentVersion, latestVersion string) string {
	// Clean version prefixes
	currentVersion = strings.TrimPrefix(currentVersion, "v")
	latestVersion = strings.TrimPrefix(latestVersion, "v")

	lines := strings.Split(changelog, "\n")
	var result strings.Builder
	var capturing bool
	var depth int

	// Regex to match version headers
	versionRe := regexp.MustCompile(`(?i)^#+\s*\[?v?(\d+\.\d+(\.\d+)?)\]?`)

	for _, line := range lines {
		matches := versionRe.FindStringSubmatch(line)
		if len(matches) > 1 {
			version := matches[1]

			// Check if this is the latest version or between current and latest
			if version == latestVersion {
				capturing = true
				depth = strings.Count(line, "#")
			} else if version == currentVersion {
				// Stop when we reach the current version
				break
			} else if capturing {
				// Check if we've moved past the latest version section
				currentDepth := strings.Count(line, "#")
				if currentDepth > 0 && currentDepth <= depth {
					// Same or higher level header, check if version is older
					// For now, just continue capturing until we hit current version
				}
			}
		}

		if capturing {
			result.WriteString(line)
			result.WriteString("\n")
		}

		// Limit size to avoid huge changelogs
		if result.Len() > 10000 {
			result.WriteString("\n... (truncated)")
			break
		}
	}

	return result.String()
}

// fetchJavaChangelog fetches changelog for a Java (Maven) package.
func (f *Fetcher) fetchJavaChangelog(ctx context.Context, info *ChangelogInfo) error {
	// Parse group:artifact
	parts := strings.Split(info.PackageName, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid Maven package name: %s", info.PackageName)
	}
	group, artifact := parts[0], parts[1]

	// Get info from Maven Central
	url := fmt.Sprintf("https://search.maven.org/solrsearch/select?q=g:%s+AND+a:%s&rows=1&wt=json", group, artifact)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("maven central returned %d", resp.StatusCode)
	}

	var mavenResult struct {
		Response struct {
			Docs []struct {
				V string `json:"v"`
			} `json:"docs"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&mavenResult); err != nil {
		return err
	}

	// Maven Central doesn't always provide repository URL in the search API.
	// We might need to fetch the POM file.
	pomURL := fmt.Sprintf("https://search.maven.org/remotecontent?filepath=%s/%s/%s/%s-%s.pom",
		strings.ReplaceAll(group, ".", "/"), artifact, info.LatestVersion, artifact, info.LatestVersion)

	pomReq, err := http.NewRequestWithContext(ctx, "GET", pomURL, nil)
	if err != nil {
		return err
	}

	pomResp, err := f.httpClient.Do(pomReq)
	if err != nil {
		return err
	}
	defer pomResp.Body.Close()

	if pomResp.StatusCode == http.StatusOK {
		pomBody, _ := io.ReadAll(pomResp.Body)
		// Basic regex to find SCM URL in POM
		scmRe := regexp.MustCompile(`(?s)<scm>.*?<url>(.*?)</url>.*? </scm>`)
		matches := scmRe.FindStringSubmatch(string(pomBody))
		var githubURL string
		if len(matches) > 1 {
			githubURL = strings.TrimSpace(matches[1])
		}

		if strings.Contains(githubURL, "github.com") {
			owner, repo := f.extractGitHubRepoFromURL(githubURL)
			if owner != "" && repo != "" {
				// Try release notes
				releaseNotes, err := f.fetchGitHubReleaseNotes(ctx, owner, repo, info.LatestVersion)
				if err == nil && releaseNotes != "" {
					info.ReleaseNotes = releaseNotes
					info.ChangelogURL = fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", owner, repo, info.LatestVersion)
					return nil
				}

				// Try with 'v' prefix
				releaseNotes, err = f.fetchGitHubReleaseNotes(ctx, owner, repo, "v"+info.LatestVersion)
				if err == nil && releaseNotes != "" {
					info.ReleaseNotes = releaseNotes
					info.ChangelogURL = fmt.Sprintf("https://github.com/%s/%s/releases/tag/v%s", owner, repo, info.LatestVersion)
					return nil
				}

				// Try CHANGELOG.md
				changelogText, err := f.fetchGitHubFile(ctx, owner, repo, "CHANGELOG.md")
				if err == nil && changelogText != "" {
					info.ChangelogText = f.extractVersionSection(changelogText, info.CurrentVersion, info.LatestVersion)
					info.ChangelogURL = fmt.Sprintf("https://github.com/%s/%s/blob/main/CHANGELOG.md", owner, repo)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("no changelog found for %s", info.PackageName)
}
