// Package github implements GitHub provider for code review integration
package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"aigateway/internal/git/provider"

	"golang.org/x/time/rate"
)

// Client implements the Provider interface for GitHub
type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
	rateLimiter *rate.Limiter
}

// NewClient creates a new GitHub client
func NewClient(config provider.ProviderConfig) *Client {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	rateLimit := config.RateLimitPerSec
	if rateLimit <= 0 {
		rateLimit = 30 // GitHub's secondary rate limit is ~100/min
	}

	return &Client{
		baseURL:     baseURL,
		accessToken: config.AccessToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: rate.NewLimiter(rate.Limit(rateLimit), rateLimit),
	}
}

// Type returns the provider type
func (c *Client) Type() provider.ProviderType {
	return provider.ProviderGitHub
}

// doRequest performs an HTTP request with authentication and rate limiting
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}

	return resp, nil
}

// GetPullRequest fetches a pull request
func (c *Client) GetPullRequest(ctx context.Context, projectID interface{}, prNumber int) (*provider.PullRequest, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repos/%s/pulls/%d", repo, prNumber)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var ghPR GitHubPullRequest
	if err := json.NewDecoder(resp.Body).Decode(&ghPR); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertPullRequest(&ghPR), nil
}

// GetPullRequestChanges fetches PR changes/diff
func (c *Client) GetPullRequestChanges(ctx context.Context, projectID interface{}, prNumber int) (*provider.PullRequestChanges, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repos/%s/pulls/%d/files", repo, prNumber)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var files []GitHubPullRequestFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Get the PR itself
	pr, err := c.GetPullRequest(ctx, projectID, prNumber)
	if err != nil {
		return nil, fmt.Errorf("get pull request: %w", err)
	}

	changes := &provider.PullRequestChanges{
		PullRequest: pr,
		Changes:     make([]provider.FileChange, 0, len(files)),
	}

	var totalAdditions, totalDeletions int
	for _, f := range files {
		change := provider.FileChange{
			OldPath:     f.PreviousFilename,
			NewPath:     f.Filename,
			Diff:        f.Patch,
			NewFile:     f.Status == "added",
			DeletedFile: f.Status == "removed",
			RenamedFile: f.Status == "renamed",
			Additions:   f.Additions,
			Deletions:   f.Deletions,
		}
		if change.OldPath == "" {
			change.OldPath = f.Filename
		}
		changes.Changes = append(changes.Changes, change)
		totalAdditions += f.Additions
		totalDeletions += f.Deletions
	}

	changes.Stats = provider.ChangeStats{
		Additions:  totalAdditions,
		Deletions:  totalDeletions,
		FilesCount: len(files),
	}

	return changes, nil
}

// CreateComment creates a comment on a PR
func (c *Client) CreateComment(ctx context.Context, projectID interface{}, prNumber int, body string) (*provider.Comment, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repos/%s/issues/%d/comments", repo, prNumber)

	payload := map[string]string{"body": body}
	jsonBody, _ := json.Marshal(payload)

	resp, err := c.doRequest(ctx, "POST", path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var ghComment GitHubComment
	if err := json.NewDecoder(resp.Body).Decode(&ghComment); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertComment(&ghComment), nil
}

// CreateInlineComment creates an inline comment on a PR
func (c *Client) CreateInlineComment(ctx context.Context, projectID interface{}, prNumber int, comment *provider.InlineComment) (*provider.Comment, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repos/%s/pulls/%d/comments", repo, prNumber)

	payload := map[string]interface{}{
		"body":      comment.Body,
		"path":      comment.Path,
		"line":      comment.Line,
		"side":      "RIGHT",
	}
	if comment.CommitID != "" {
		payload["commit_id"] = comment.CommitID
	}

	jsonBody, _ := json.Marshal(payload)

	resp, err := c.doRequest(ctx, "POST", path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var ghComment GitHubComment
	if err := json.NewDecoder(resp.Body).Decode(&ghComment); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertComment(&ghComment), nil
}

// GetProject fetches repository details
func (c *Client) GetProject(ctx context.Context, projectID interface{}) (*provider.Project, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repos/%s", repo)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var ghRepo GitHubRepository
	if err := json.NewDecoder(resp.Body).Decode(&ghRepo); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertProject(&ghRepo), nil
}

// GetCurrentUser returns the authenticated user
func (c *Client) GetCurrentUser(ctx context.Context) (*provider.User, error) {
	resp, err := c.doRequest(ctx, "GET", "/user", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var ghUser GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertUser(&ghUser), nil
}

// CreateWebhook creates a repository webhook
func (c *Client) CreateWebhook(ctx context.Context, projectID interface{}, config *provider.WebhookConfig) (*provider.Webhook, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repos/%s/hooks", repo)

	events := config.Events
	if len(events) == 0 {
		events = []string{"pull_request", "pull_request_review"}
	}

	payload := map[string]interface{}{
		"name":   "web",
		"active": config.Active,
		"events": events,
		"config": map[string]interface{}{
			"url":          config.URL,
			"content_type": "json",
			"secret":       config.Secret,
			"insecure_ssl": "0",
		},
	}

	jsonBody, _ := json.Marshal(payload)

	resp, err := c.doRequest(ctx, "POST", path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var ghHook GitHubWebhook
	if err := json.NewDecoder(resp.Body).Decode(&ghHook); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &provider.Webhook{
		ID:        ghHook.ID,
		URL:       ghHook.Config.URL,
		Events:    ghHook.Events,
		Active:    ghHook.Active,
		CreatedAt: ghHook.CreatedAt,
	}, nil
}

// DeleteWebhook deletes a webhook
func (c *Client) DeleteWebhook(ctx context.Context, projectID interface{}, webhookID int64) error {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repos/%s/hooks/%d", repo, webhookID)

	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetFile fetches file content
func (c *Client) GetFile(ctx context.Context, projectID interface{}, filePath, ref string) (*provider.File, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repos/%s/contents/%s", repo, filePath)
	if ref != "" {
		path += "?ref=" + ref
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var ghFile GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&ghFile); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &provider.File{
		Path:     ghFile.Path,
		Content:  ghFile.Content,
		Encoding: ghFile.Encoding,
		Size:     ghFile.Size,
		SHA:      ghFile.SHA,
	}, nil
}

// ValidateWebhook validates the webhook signature
func (c *Client) ValidateWebhook(payload []byte, signature string, secret string) bool {
	if secret == "" {
		return true
	}

	// GitHub sends signature as "sha256=<hash>"
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	signature = strings.TrimPrefix(signature, "sha256=")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}

// Helper functions

func (c *Client) parseProjectID(projectID interface{}) string {
	switch v := projectID.(type) {
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (c *Client) convertPullRequest(gh *GitHubPullRequest) *provider.PullRequest {
	state := provider.PRStateOpen
	if gh.State == "closed" {
		if gh.MergedAt != nil {
			state = provider.PRStateMerged
		} else {
			state = provider.PRStateClosed
		}
	}

	pr := &provider.PullRequest{
		ID:           gh.ID,
		Number:       gh.Number,
		Title:        gh.Title,
		Description:  gh.Body,
		State:        state,
		Draft:        gh.Draft,
		SourceBranch: gh.Head.Ref,
		TargetBranch: gh.Base.Ref,
		WebURL:       gh.HTMLURL,
		CreatedAt:    gh.CreatedAt,
		UpdatedAt:    gh.UpdatedAt,
		MergedAt:     gh.MergedAt,
		ClosedAt:     gh.ClosedAt,
	}

	if gh.User != nil {
		pr.Author = c.convertUser(gh.User)
	}

	return pr
}

func (c *Client) convertUser(gh *GitHubUser) *provider.User {
	return &provider.User{
		ID:        gh.ID,
		Username:  gh.Login,
		Name:      gh.Name,
		Email:     gh.Email,
		AvatarURL: gh.AvatarURL,
		IsBot:     gh.Type == "Bot",
	}
}

func (c *Client) convertProject(gh *GitHubRepository) *provider.Project {
	return &provider.Project{
		ID:            gh.ID,
		Name:          gh.Name,
		FullName:      gh.FullName,
		Description:   gh.Description,
		DefaultBranch: gh.DefaultBranch,
		WebURL:        gh.HTMLURL,
		CloneURL:      gh.CloneURL,
		Private:       gh.Private,
		Archived:      gh.Archived,
		Language:      gh.Language,
	}
}

func (c *Client) convertComment(gh *GitHubComment) *provider.Comment {
	comment := &provider.Comment{
		ID:        gh.ID,
		Body:      gh.Body,
		CreatedAt: gh.CreatedAt,
		UpdatedAt: gh.UpdatedAt,
	}
	if gh.User != nil {
		comment.Author = c.convertUser(gh.User)
	}
	return comment
}

