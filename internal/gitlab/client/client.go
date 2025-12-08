// Package client provides GitLab API client for MR review integration
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

// Client is a GitLab API client
type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
	rateLimiter *rate.Limiter
	apiVersion  string
}

// ClientConfig holds configuration for the GitLab client
type ClientConfig struct {
	BaseURL        string
	AccessToken    string
	Timeout        time.Duration
	RateLimitPerSec float64 // Default: 30 (safe for GitLab 2000/min limit)
	APIVersion     string   // Default: "v4"
}

// NewClient creates a new GitLab API client
func NewClient(cfg ClientConfig) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.RateLimitPerSec == 0 {
		cfg.RateLimitPerSec = 30 // ~1800 req/min, below GitLab 2000 limit
	}
	if cfg.APIVersion == "" {
		cfg.APIVersion = "v4"
	}

	// Normalize base URL
	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")

	return &Client{
		baseURL:     baseURL,
		accessToken: cfg.AccessToken,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		rateLimiter: rate.NewLimiter(rate.Limit(cfg.RateLimitPerSec), 10),
		apiVersion:  cfg.APIVersion,
	}
}

// apiURL constructs the full API URL
func (c *Client) apiURL(path string) string {
	return fmt.Sprintf("%s/api/%s%s", c.baseURL, c.apiVersion, path)
}

// doRequest performs an HTTP request with rate limiting and auth
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	// Wait for rate limiter
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.apiURL(path), body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return resp, nil
}

// doRequestWithQuery performs an HTTP request with query parameters
func (c *Client) doRequestWithQuery(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Response, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	fullURL := c.apiURL(path)
	if len(query) > 0 {
		fullURL = fullURL + "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", c.accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return resp, nil
}

// parseResponse parses the JSON response into the target struct
func (c *Client) parseResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	if target == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// APIError represents a GitLab API error
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("GitLab API error (status %d): %s", e.StatusCode, e.Message)
}

// ============================================================================
// Connection & Authentication
// ============================================================================

// TestConnection tests the connection to GitLab
func (c *Client) TestConnection(ctx context.Context) (*User, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/user", nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := c.parseResponse(resp, &user); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &user, nil
}

// ============================================================================
// Projects API
// ============================================================================

// GetProject retrieves a project by ID
func (c *Client) GetProject(ctx context.Context, projectID int64) (*Project, error) {
	path := fmt.Sprintf("/projects/%d", projectID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := c.parseResponse(resp, &project); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &project, nil
}

// ListProjects lists projects accessible to the authenticated user
func (c *Client) ListProjects(ctx context.Context, opts *ListProjectsOptions) ([]Project, error) {
	query := url.Values{}
	if opts != nil {
		if opts.Search != "" {
			query.Set("search", opts.Search)
		}
		if opts.PerPage > 0 {
			query.Set("per_page", strconv.Itoa(opts.PerPage))
		}
		if opts.Page > 0 {
			query.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.Membership {
			query.Set("membership", "true")
		}
		if opts.MinAccessLevel > 0 {
			query.Set("min_access_level", strconv.Itoa(opts.MinAccessLevel))
		}
	}

	resp, err := c.doRequestWithQuery(ctx, http.MethodGet, "/projects", query, nil)
	if err != nil {
		return nil, err
	}

	var projects []Project
	if err := c.parseResponse(resp, &projects); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return projects, nil
}

// ============================================================================
// Merge Requests API
// ============================================================================

// GetMergeRequest retrieves a merge request by project and IID
func (c *Client) GetMergeRequest(ctx context.Context, projectID int64, mrIID int) (*MergeRequest, error) {
	path := fmt.Sprintf("/projects/%d/merge_requests/%d", projectID, mrIID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var mr MergeRequest
	if err := c.parseResponse(resp, &mr); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &mr, nil
}

// GetMergeRequestChanges retrieves the changes (diff) of a merge request
func (c *Client) GetMergeRequestChanges(ctx context.Context, projectID int64, mrIID int) (*MergeRequestChanges, error) {
	path := fmt.Sprintf("/projects/%d/merge_requests/%d/changes", projectID, mrIID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var changes MergeRequestChanges
	if err := c.parseResponse(resp, &changes); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &changes, nil
}

// GetMergeRequestDiffs retrieves the diffs of a merge request
func (c *Client) GetMergeRequestDiffs(ctx context.Context, projectID int64, mrIID int) ([]Diff, error) {
	path := fmt.Sprintf("/projects/%d/merge_requests/%d/diffs", projectID, mrIID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var diffs []Diff
	if err := c.parseResponse(resp, &diffs); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return diffs, nil
}

// ============================================================================
// Repository Files API
// ============================================================================

// GetFile retrieves a file from the repository
func (c *Client) GetFile(ctx context.Context, projectID int64, filePath, ref string) (*File, error) {
	path := fmt.Sprintf("/projects/%d/repository/files/%s", projectID, url.PathEscape(filePath))
	query := url.Values{}
	query.Set("ref", ref)

	resp, err := c.doRequestWithQuery(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}

	var file File
	if err := c.parseResponse(resp, &file); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &file, nil
}

// GetFileRaw retrieves raw file content
func (c *Client) GetFileRaw(ctx context.Context, projectID int64, filePath, ref string) ([]byte, error) {
	path := fmt.Sprintf("/projects/%d/repository/files/%s/raw", projectID, url.PathEscape(filePath))
	query := url.Values{}
	query.Set("ref", ref)

	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait: %w", err)
	}

	fullURL := c.apiURL(path) + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return io.ReadAll(resp.Body)
}

// ============================================================================
// Notes (Comments) API
// ============================================================================

// CreateMRNote creates a note (comment) on a merge request
func (c *Client) CreateMRNote(ctx context.Context, projectID int64, mrIID int, body string) (*Note, error) {
	path := fmt.Sprintf("/projects/%d/merge_requests/%d/notes", projectID, mrIID)

	payload := map[string]string{"body": body}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var note Note
	if err := c.parseResponse(resp, &note); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &note, nil
}

// UpdateMRNote updates a note on a merge request
func (c *Client) UpdateMRNote(ctx context.Context, projectID int64, mrIID int, noteID int64, body string) (*Note, error) {
	path := fmt.Sprintf("/projects/%d/merge_requests/%d/notes/%d", projectID, mrIID, noteID)

	payload := map[string]string{"body": body}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPut, path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var note Note
	if err := c.parseResponse(resp, &note); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &note, nil
}

// DeleteMRNote deletes a note from a merge request
func (c *Client) DeleteMRNote(ctx context.Context, projectID int64, mrIID int, noteID int64) error {
	path := fmt.Sprintf("/projects/%d/merge_requests/%d/notes/%d", projectID, mrIID, noteID)

	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	return c.parseResponse(resp, nil)
}

// ============================================================================
// Discussions API (for inline comments)
// ============================================================================

// CreateMRDiscussion creates a new discussion (thread) on a merge request
func (c *Client) CreateMRDiscussion(ctx context.Context, projectID int64, mrIID int, req *CreateDiscussionRequest) (*Discussion, error) {
	path := fmt.Sprintf("/projects/%d/merge_requests/%d/discussions", projectID, mrIID)

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var discussion Discussion
	if err := c.parseResponse(resp, &discussion); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &discussion, nil
}

// ============================================================================
// Webhooks API
// ============================================================================

// CreateWebhook creates a webhook for a project
func (c *Client) CreateWebhook(ctx context.Context, projectID int64, req *CreateWebhookRequest) (*Webhook, error) {
	path := fmt.Sprintf("/projects/%d/hooks", projectID)

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := c.parseResponse(resp, &webhook); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &webhook, nil
}

// DeleteWebhook deletes a webhook from a project
func (c *Client) DeleteWebhook(ctx context.Context, projectID int64, webhookID int64) error {
	path := fmt.Sprintf("/projects/%d/hooks/%d", projectID, webhookID)

	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	return c.parseResponse(resp, nil)
}

// GetWebhook retrieves a webhook by ID
func (c *Client) GetWebhook(ctx context.Context, projectID int64, webhookID int64) (*Webhook, error) {
	path := fmt.Sprintf("/projects/%d/hooks/%d", projectID, webhookID)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var webhook Webhook
	if err := c.parseResponse(resp, &webhook); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &webhook, nil
}

// ============================================================================
// User API
// ============================================================================

// GetCurrentUser returns the currently authenticated user
func (c *Client) GetCurrentUser(ctx context.Context) (*User, error) {
	path := "/user"

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := c.parseResponse(resp, &user); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &user, nil
}

// CreateProjectWebhook creates a webhook for a GitLab project with MR events enabled
func (c *Client) CreateProjectWebhook(ctx context.Context, projectID int64, webhookURL, secret string) (*Webhook, error) {
	req := &CreateWebhookRequest{
		URL:                   webhookURL,
		MergeRequestsEvents:   true,
		PushEvents:            false,
		Token:                 secret,
		EnableSSLVerification: true,
	}
	return c.CreateWebhook(ctx, projectID, req)
}

