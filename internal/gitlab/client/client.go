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
	BaseURL         string
	AccessToken     string
	Timeout         time.Duration
	RateLimitPerSec float64 // Default: 30 (safe for GitLab 2000/min limit)
	APIVersion      string  // Default: "v4"
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

// DoRequest performs an HTTP request (exported for indexer usage)
func (c *Client) DoRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	return c.doRequest(ctx, method, path, body)
}

// ParseResponse parses the JSON response into the target struct (exported)
func (c *Client) ParseResponse(resp *http.Response, target interface{}) error {
	return c.parseResponse(resp, target)
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

// ListBranches lists all branches of a project
func (c *Client) ListBranches(ctx context.Context, projectID int64) ([]Branch, error) {
	path := fmt.Sprintf("/projects/%d/repository/branches", projectID)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var branches []Branch
	if err := c.parseResponse(resp, &branches); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return branches, nil
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

// Issue represents a GitLab issue
type Issue struct {
	ID          int64    `json:"id"`
	IID         int      `json:"iid"`
	ProjectID   int64    `json:"project_id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	State       string   `json:"state"`
	Labels      []string `json:"labels"`
	WebURL      string   `json:"web_url"`
	CreatedAt   string   `json:"created_at"`
}

// CreateIssueRequest contains the data for creating an issue
type CreateIssueRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	Assignees   []int64  `json:"assignee_ids,omitempty"`
}

// CreateIssue creates a new issue in a GitLab project
func (c *Client) CreateIssue(ctx context.Context, projectID int64, req *CreateIssueRequest) (*Issue, error) {
	path := fmt.Sprintf("/projects/%d/issues", projectID)

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := c.parseResponse(resp, &issue); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &issue, nil
}

// ============================================================================
// Branch & Commit & Merge Request Creation
// ============================================================================

// Branch represents a GitLab branch
type Branch struct {
	Name               string `json:"name"`
	Merged             bool   `json:"merged"`
	Protected          bool   `json:"protected"`
	Default            bool   `json:"default"`
	DevelopersCanPush  bool   `json:"developers_can_push"`
	DevelopersCanMerge bool   `json:"developers_can_merge"`
	CanPush            bool   `json:"can_push"`
	WebURL             string `json:"web_url"`
	Commit             *struct {
		ID        string `json:"id"`
		ShortID   string `json:"short_id"`
		Title     string `json:"title"`
		CreatedAt string `json:"created_at"`
	} `json:"commit,omitempty"`
}

// CreateBranchRequest contains the data for creating a branch
type CreateBranchRequest struct {
	Branch string `json:"branch"` // New branch name
	Ref    string `json:"ref"`    // Source branch or commit SHA
}

// CreateBranch creates a new branch in a GitLab project
func (c *Client) CreateBranch(ctx context.Context, projectID int64, req *CreateBranchRequest) (*Branch, error) {
	path := fmt.Sprintf("/projects/%d/repository/branches", projectID)

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var branch Branch
	if err := c.parseResponse(resp, &branch); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &branch, nil
}

// DeleteBranch deletes a branch from a GitLab project
func (c *Client) DeleteBranch(ctx context.Context, projectID int64, branchName string) error {
	path := fmt.Sprintf("/projects/%d/repository/branches/%s", projectID, url.PathEscape(branchName))

	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete branch failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CommitAction represents an action in a commit
type CommitAction struct {
	Action          string `json:"action"`                     // create, delete, move, update, chmod
	FilePath        string `json:"file_path"`                  // Full path to file
	PreviousPath    string `json:"previous_path,omitempty"`    // For move action
	Content         string `json:"content,omitempty"`          // File content (base64 for binary)
	Encoding        string `json:"encoding,omitempty"`         // text or base64
	LastCommitID    string `json:"last_commit_id,omitempty"`   // For update to detect conflicts
	ExecuteFilemode bool   `json:"execute_filemode,omitempty"` // For chmod
}

// CreateCommitRequest contains the data for creating a commit
type CreateCommitRequest struct {
	Branch        string         `json:"branch"`
	CommitMessage string         `json:"commit_message"`
	StartBranch   string         `json:"start_branch,omitempty"`  // Create branch from this if not exists
	StartSHA      string         `json:"start_sha,omitempty"`     // Create branch from this SHA
	StartProject  int64          `json:"start_project,omitempty"` // Project ID for start_branch
	Actions       []CommitAction `json:"actions"`
	AuthorEmail   string         `json:"author_email,omitempty"`
	AuthorName    string         `json:"author_name,omitempty"`
	Stats         bool           `json:"stats,omitempty"`
	Force         bool           `json:"force,omitempty"`
}

// CommitResponse represents a GitLab commit response (extended from Commit type)
type CommitResponse struct {
	ID             string   `json:"id"`
	ShortID        string   `json:"short_id"`
	Title          string   `json:"title"`
	Message        string   `json:"message"`
	AuthorName     string   `json:"author_name"`
	AuthorEmail    string   `json:"author_email"`
	AuthoredDate   string   `json:"authored_date"`
	CommitterName  string   `json:"committer_name"`
	CommitterEmail string   `json:"committer_email"`
	CommittedDate  string   `json:"committed_date"`
	CreatedAt      string   `json:"created_at"`
	WebURL         string   `json:"web_url"`
	ParentIDs      []string `json:"parent_ids"`
	Stats          *struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
		Total     int `json:"total"`
	} `json:"stats,omitempty"`
}

// CreateCommit creates a commit with file changes
func (c *Client) CreateCommit(ctx context.Context, projectID int64, req *CreateCommitRequest) (*CommitResponse, error) {
	path := fmt.Sprintf("/projects/%d/repository/commits", projectID)

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var commit CommitResponse
	if err := c.parseResponse(resp, &commit); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &commit, nil
}

// CreateMergeRequestRequest contains the data for creating a merge request
type CreateMergeRequestRequest struct {
	SourceBranch       string  `json:"source_branch"`
	TargetBranch       string  `json:"target_branch"`
	Title              string  `json:"title"`
	Description        string  `json:"description,omitempty"`
	AssigneeID         int64   `json:"assignee_id,omitempty"`
	AssigneeIDs        []int64 `json:"assignee_ids,omitempty"`
	ReviewerIDs        []int64 `json:"reviewer_ids,omitempty"`
	Labels             string  `json:"labels,omitempty"` // Comma-separated
	MilestoneID        int64   `json:"milestone_id,omitempty"`
	RemoveSourceBranch bool    `json:"remove_source_branch,omitempty"`
	AllowCollaboration bool    `json:"allow_collaboration,omitempty"`
	Squash             bool    `json:"squash,omitempty"`
	SquashOnMerge      bool    `json:"squash_on_merge,omitempty"`
	TargetProjectID    int64   `json:"target_project_id,omitempty"`
}

// CreateMergeRequest creates a new merge request
func (c *Client) CreateMergeRequest(ctx context.Context, projectID int64, req *CreateMergeRequestRequest) (*MergeRequest, error) {
	path := fmt.Sprintf("/projects/%d/merge_requests", projectID)

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}

	var mr MergeRequest
	if err := c.parseResponse(resp, &mr); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &mr, nil
}

// CreateMRWithChanges creates a branch, commits changes, and opens a merge request.
// This is a convenience method that combines CreateBranch, CreateCommit, and CreateMergeRequest.
//
// The method performs three steps atomically (with cleanup on failure):
//  1. Creates a new branch from TargetBranch
//  2. Creates a commit with the specified file changes
//  3. Opens a merge request from the new branch to TargetBranch
//
// If any step fails, the method attempts to clean up by deleting the created branch.
// Returns the created MergeRequest on success, or an error with details about which step failed.
func (c *Client) CreateMRWithChanges(ctx context.Context, projectID int64, opts CreateMRWithChangesOptions) (*MergeRequest, error) {
	// Validate required options
	if opts.TargetBranch == "" {
		return nil, fmt.Errorf("target_branch is required")
	}
	if len(opts.Actions) == 0 {
		return nil, fmt.Errorf("at least one file action is required")
	}
	if opts.CommitMessage == "" {
		return nil, fmt.Errorf("commit_message is required")
	}
	if opts.MRTitle == "" {
		return nil, fmt.Errorf("mr_title is required")
	}

	// Step 1: Create branch from target
	branchName := opts.BranchName
	if branchName == "" {
		branchName = fmt.Sprintf("aigateway/%s-%d", opts.BranchPrefix, time.Now().Unix())
	}

	_, err := c.CreateBranch(ctx, projectID, &CreateBranchRequest{
		Branch: branchName,
		Ref:    opts.TargetBranch,
	})
	if err != nil {
		return nil, fmt.Errorf("step 1/3 create branch '%s' from '%s': %w", branchName, opts.TargetBranch, err)
	}

	// Step 2: Create commit with file changes
	_, err = c.CreateCommit(ctx, projectID, &CreateCommitRequest{
		Branch:        branchName,
		CommitMessage: opts.CommitMessage,
		Actions:       opts.Actions,
		AuthorEmail:   opts.AuthorEmail,
		AuthorName:    opts.AuthorName,
	})
	if err != nil {
		// Clean up: delete branch on failure
		if delErr := c.DeleteBranch(ctx, projectID, branchName); delErr != nil {
			return nil, fmt.Errorf("step 2/3 create commit failed: %w (cleanup also failed: %v)", err, delErr)
		}
		return nil, fmt.Errorf("step 2/3 create commit with %d file(s): %w", len(opts.Actions), err)
	}

	// Step 3: Create merge request
	mr, err := c.CreateMergeRequest(ctx, projectID, &CreateMergeRequestRequest{
		SourceBranch:       branchName,
		TargetBranch:       opts.TargetBranch,
		Title:              opts.MRTitle,
		Description:        opts.MRDescription,
		Labels:             opts.Labels,
		RemoveSourceBranch: true, // Auto-delete branch after merge
	})
	if err != nil {
		// Clean up: delete branch on failure
		if delErr := c.DeleteBranch(ctx, projectID, branchName); delErr != nil {
			return nil, fmt.Errorf("step 3/3 create MR failed: %w (cleanup also failed: %v)", err, delErr)
		}
		return nil, fmt.Errorf("step 3/3 create merge request '%s' -> '%s': %w", branchName, opts.TargetBranch, err)
	}

	return mr, nil
}

// CreateMRWithChangesOptions holds options for CreateMRWithChanges
type CreateMRWithChangesOptions struct {
	BranchName    string         // Optional: specific branch name (auto-generated if empty)
	BranchPrefix  string         // Prefix for auto-generated branch name (e.g., "autodocs", "tests")
	TargetBranch  string         // Target branch to merge into (e.g., "main")
	CommitMessage string         // Commit message
	Actions       []CommitAction // File changes
	AuthorName    string         // Optional: commit author name
	AuthorEmail   string         // Optional: commit author email
	MRTitle       string         // Merge request title
	MRDescription string         // Merge request description
	Labels        string         // Comma-separated labels
}
