// Package bitbucket implements Bitbucket provider for code review integration
package bitbucket

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

// Client implements the Provider interface for Bitbucket
type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
	rateLimiter *rate.Limiter
}

// NewClient creates a new Bitbucket client
func NewClient(config provider.ProviderConfig) *Client {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.bitbucket.org/2.0"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	rateLimit := config.RateLimitPerSec
	if rateLimit <= 0 {
		rateLimit = 30
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
	return provider.ProviderBitbucket
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
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
func (c *Client) GetPullRequest(ctx context.Context, projectID any, prNumber int) (*provider.PullRequest, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repositories/%s/pullrequests/%d", repo, prNumber)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bitbucket API error: %d - %s", resp.StatusCode, string(body))
	}

	var bbPR BitbucketPullRequest
	if err := json.NewDecoder(resp.Body).Decode(&bbPR); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertPullRequest(&bbPR), nil
}

// GetPullRequestChanges fetches PR changes
func (c *Client) GetPullRequestChanges(ctx context.Context, projectID any, prNumber int) (*provider.PullRequestChanges, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repositories/%s/pullrequests/%d/diffstat", repo, prNumber)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bitbucket API error: %d - %s", resp.StatusCode, string(body))
	}

	var diffstat BitbucketDiffstat
	if err := json.NewDecoder(resp.Body).Decode(&diffstat); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Get the PR itself
	pr, err := c.GetPullRequest(ctx, projectID, prNumber)
	if err != nil {
		return nil, fmt.Errorf("get pull request: %w", err)
	}

	// Get the actual diff
	diffPath := fmt.Sprintf("/repositories/%s/pullrequests/%d/diff", repo, prNumber)
	diffResp, err := c.doRequest(ctx, "GET", diffPath, nil)
	if err != nil {
		return nil, fmt.Errorf("get diff: %w", err)
	}
	defer diffResp.Body.Close()

	diffContent, _ := io.ReadAll(diffResp.Body)
	fileDiffs := parseDiff(string(diffContent))

	changes := &provider.PullRequestChanges{
		PullRequest: pr,
		Changes:     make([]provider.FileChange, 0, len(diffstat.Values)),
	}

	var totalAdditions, totalDeletions int
	for _, ds := range diffstat.Values {
		diff := ""
		if d, ok := fileDiffs[ds.New.Path]; ok {
			diff = d
		}

		change := provider.FileChange{
			OldPath:     ds.Old.Path,
			NewPath:     ds.New.Path,
			Diff:        diff,
			NewFile:     ds.Status == "added",
			DeletedFile: ds.Status == "removed",
			RenamedFile: ds.Status == "renamed",
			Additions:   ds.LinesAdded,
			Deletions:   ds.LinesRemoved,
		}
		if change.OldPath == "" {
			change.OldPath = ds.New.Path
		}
		changes.Changes = append(changes.Changes, change)
		totalAdditions += ds.LinesAdded
		totalDeletions += ds.LinesRemoved
	}

	changes.Stats = provider.ChangeStats{
		Additions:  totalAdditions,
		Deletions:  totalDeletions,
		FilesCount: len(diffstat.Values),
	}

	return changes, nil
}

// CreateComment creates a comment on a PR
func (c *Client) CreateComment(ctx context.Context, projectID any, prNumber int, body string) (*provider.Comment, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repositories/%s/pullrequests/%d/comments", repo, prNumber)

	payload := map[string]any{
		"content": map[string]string{
			"raw": body,
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
		return nil, fmt.Errorf("Bitbucket API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var bbComment BitbucketComment
	if err := json.NewDecoder(resp.Body).Decode(&bbComment); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertComment(&bbComment), nil
}

// CreateInlineComment creates an inline comment
func (c *Client) CreateInlineComment(ctx context.Context, projectID any, prNumber int, comment *provider.InlineComment) (*provider.Comment, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repositories/%s/pullrequests/%d/comments", repo, prNumber)

	payload := map[string]any{
		"content": map[string]string{
			"raw": comment.Body,
		},
		"inline": map[string]any{
			"path": comment.Path,
			"to":   comment.Line,
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
		return nil, fmt.Errorf("Bitbucket API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var bbComment BitbucketComment
	if err := json.NewDecoder(resp.Body).Decode(&bbComment); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertComment(&bbComment), nil
}

// GetProject fetches repository details
func (c *Client) GetProject(ctx context.Context, projectID any) (*provider.Project, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repositories/%s", repo)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bitbucket API error: %d - %s", resp.StatusCode, string(body))
	}

	var bbRepo BitbucketRepository
	if err := json.NewDecoder(resp.Body).Decode(&bbRepo); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertProject(&bbRepo), nil
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
		return nil, fmt.Errorf("Bitbucket API error: %d - %s", resp.StatusCode, string(body))
	}

	var bbUser BitbucketUser
	if err := json.NewDecoder(resp.Body).Decode(&bbUser); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return c.convertUser(&bbUser), nil
}

// CreateWebhook creates a repository webhook
func (c *Client) CreateWebhook(ctx context.Context, projectID any, config *provider.WebhookConfig) (*provider.Webhook, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repositories/%s/hooks", repo)

	events := config.Events
	if len(events) == 0 {
		events = []string{"pullrequest:created", "pullrequest:updated"}
	}

	payload := map[string]any{
		"description": "AIGateway Code Review",
		"url":         config.URL,
		"active":      config.Active,
		"events":      events,
		"secret":      config.Secret,
	}

	jsonBody, _ := json.Marshal(payload)

	resp, err := c.doRequest(ctx, "POST", path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bitbucket API error: %d - %s", resp.StatusCode, string(respBody))
	}

	var bbHook BitbucketWebhook
	if err := json.NewDecoder(resp.Body).Decode(&bbHook); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &provider.Webhook{
		ID:        int64(bbHook.UUID[1:17][0]), // Simplified UUID to int conversion
		URL:       bbHook.URL,
		Events:    bbHook.Events,
		Active:    bbHook.Active,
		CreatedAt: bbHook.CreatedOn,
	}, nil
}

// DeleteWebhook deletes a webhook
func (c *Client) DeleteWebhook(ctx context.Context, projectID any, webhookID int64) error {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repositories/%s/hooks/%d", repo, webhookID)

	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Bitbucket API error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetFile fetches file content
func (c *Client) GetFile(ctx context.Context, projectID any, filePath, ref string) (*provider.File, error) {
	repo := c.parseProjectID(projectID)
	path := fmt.Sprintf("/repositories/%s/src/%s/%s", repo, ref, filePath)

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Bitbucket API error: %d - %s", resp.StatusCode, string(body))
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read file content: %w", err)
	}

	return &provider.File{
		Path:     filePath,
		Content:  string(content),
		Encoding: "utf-8",
		Size:     int64(len(content)),
	}, nil
}

// ValidateWebhook validates the webhook signature
func (c *Client) ValidateWebhook(payload []byte, signature string, secret string) bool {
	if secret == "" {
		return true
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedMAC))
}

// Helper functions

func (c *Client) parseProjectID(projectID any) string {
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

func (c *Client) convertPullRequest(bb *BitbucketPullRequest) *provider.PullRequest {
	state := provider.PRStateOpen
	switch bb.State {
	case "MERGED":
		state = provider.PRStateMerged
	case "DECLINED", "SUPERSEDED":
		state = provider.PRStateClosed
	}

	pr := &provider.PullRequest{
		ID:           bb.ID,
		Number:       int(bb.ID),
		Title:        bb.Title,
		Description:  bb.Description,
		State:        state,
		SourceBranch: bb.Source.Branch.Name,
		TargetBranch: bb.Destination.Branch.Name,
		WebURL:       bb.Links.HTML.Href,
		CreatedAt:    bb.CreatedOn,
		UpdatedAt:    bb.UpdatedOn,
	}

	if bb.Author != nil {
		pr.Author = c.convertUser(bb.Author)
	}

	return pr
}

func (c *Client) convertUser(bb *BitbucketUser) *provider.User {
	return &provider.User{
		ID:        0, // Bitbucket uses UUIDs
		Username:  bb.Username,
		Name:      bb.DisplayName,
		AvatarURL: bb.Links.Avatar.Href,
		IsBot:     bb.Type == "team",
	}
}

func (c *Client) convertProject(bb *BitbucketRepository) *provider.Project {
	return &provider.Project{
		ID:            0, // Bitbucket uses UUIDs
		Name:          bb.Name,
		FullName:      bb.FullName,
		Description:   bb.Description,
		DefaultBranch: bb.MainBranch.Name,
		WebURL:        bb.Links.HTML.Href,
		CloneURL:      bb.Links.Clone[0].Href,
		Private:       bb.IsPrivate,
		Language:      bb.Language,
	}
}

func (c *Client) convertComment(bb *BitbucketComment) *provider.Comment {
	comment := &provider.Comment{
		ID:        bb.ID,
		Body:      bb.Content.Raw,
		CreatedAt: bb.CreatedOn,
		UpdatedAt: bb.UpdatedOn,
	}
	if bb.User != nil {
		comment.Author = c.convertUser(bb.User)
	}
	return comment
}

// parseDiff parses unified diff format into file-based map
func parseDiff(diff string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(diff, "\n")

	var currentFile string
	var currentDiff strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			if currentFile != "" {
				result[currentFile] = currentDiff.String()
			}
			// Extract filename from "diff --git a/path b/path"
			parts := strings.Split(line, " ")
			if len(parts) >= 4 {
				currentFile = strings.TrimPrefix(parts[3], "b/")
			}
			currentDiff.Reset()
		}
		currentDiff.WriteString(line)
		currentDiff.WriteString("\n")
	}

	if currentFile != "" {
		result[currentFile] = currentDiff.String()
	}

	return result
}
