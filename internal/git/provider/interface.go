// Package provider defines interfaces for multi-provider Git support
package provider

import (
	"context"
	"time"
)

// ProviderType represents the type of Git provider
type ProviderType string

const (
	ProviderGitLab    ProviderType = "gitlab"
	ProviderGitHub    ProviderType = "github"
	ProviderBitbucket ProviderType = "bitbucket"
)

// Provider is the interface that all Git providers must implement
type Provider interface {
	// Type returns the provider type
	Type() ProviderType

	// GetPullRequest fetches pull/merge request details
	GetPullRequest(ctx context.Context, projectID any, prNumber int) (*PullRequest, error)

	// GetPullRequestChanges fetches the diff/changes of a PR
	GetPullRequestChanges(ctx context.Context, projectID any, prNumber int) (*PullRequestChanges, error)

	// CreateComment posts a comment on a PR
	CreateComment(ctx context.Context, projectID any, prNumber int, body string) (*Comment, error)

	// CreateInlineComment posts an inline comment on specific lines
	CreateInlineComment(ctx context.Context, projectID any, prNumber int, comment *InlineComment) (*Comment, error)

	// GetProject fetches project/repository details
	GetProject(ctx context.Context, projectID any) (*Project, error)

	// GetCurrentUser returns the authenticated user
	GetCurrentUser(ctx context.Context) (*User, error)

	// CreateWebhook creates a webhook for a project
	CreateWebhook(ctx context.Context, projectID any, config *WebhookConfig) (*Webhook, error)

	// DeleteWebhook deletes a webhook
	DeleteWebhook(ctx context.Context, projectID any, webhookID int64) error

	// GetFile fetches file content from repository
	GetFile(ctx context.Context, projectID any, path, ref string) (*File, error)

	// ValidateWebhook validates webhook signature
	ValidateWebhook(payload []byte, signature string, secret string) bool
}

// PullRequest represents a unified pull/merge request across providers
type PullRequest struct {
	ID           int64      `json:"id"`
	Number       int        `json:"number"` // IID in GitLab, number in GitHub/Bitbucket
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	State        PRState    `json:"state"`
	Draft        bool       `json:"draft"`
	SourceBranch string     `json:"source_branch"`
	TargetBranch string     `json:"target_branch"`
	Author       *User      `json:"author"`
	WebURL       string     `json:"web_url"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	MergedAt     *time.Time `json:"merged_at,omitempty"`
	ClosedAt     *time.Time `json:"closed_at,omitempty"`

	// Provider-specific data
	ProviderData map[string]any `json:"provider_data,omitempty"`
}

// PRState represents pull request state
type PRState string

const (
	PRStateOpen   PRState = "open"
	PRStateClosed PRState = "closed"
	PRStateMerged PRState = "merged"
)

// PullRequestChanges contains the changes in a PR
type PullRequestChanges struct {
	PullRequest *PullRequest `json:"pull_request"`
	Changes     []FileChange `json:"changes"`
	Stats       ChangeStats  `json:"stats"`
}

// FileChange represents changes to a single file
type FileChange struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	Diff        string `json:"diff"`
	NewFile     bool   `json:"new_file"`
	DeletedFile bool   `json:"deleted_file"`
	RenamedFile bool   `json:"renamed_file"`
	Binary      bool   `json:"binary"`

	// Stats
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
}

// ChangeStats contains overall change statistics
type ChangeStats struct {
	Additions  int `json:"additions"`
	Deletions  int `json:"deletions"`
	FilesCount int `json:"files_count"`
}

// Comment represents a comment on a PR
type Comment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	Author    *User     `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// For discussion threads
	ThreadID string `json:"thread_id,omitempty"`
}

// InlineComment represents a comment on specific lines
type InlineComment struct {
	Body     string `json:"body"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	EndLine  *int   `json:"end_line,omitempty"`
	Side     string `json:"side,omitempty"` // "LEFT" or "RIGHT"
	CommitID string `json:"commit_id,omitempty"`
}

// Project represents a repository/project
type Project struct {
	ID            int64    `json:"id"`
	Name          string   `json:"name"`
	FullName      string   `json:"full_name"` // owner/repo format
	Description   string   `json:"description"`
	DefaultBranch string   `json:"default_branch"`
	WebURL        string   `json:"web_url"`
	CloneURL      string   `json:"clone_url"`
	Private       bool     `json:"private"`
	Archived      bool     `json:"archived"`
	Language      string   `json:"language"`
	Languages     []string `json:"languages,omitempty"`
}

// User represents a user on the platform
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
	IsBot     bool   `json:"is_bot"`
}

// WebhookConfig contains webhook creation configuration
type WebhookConfig struct {
	URL       string   `json:"url"`
	Secret    string   `json:"secret"`
	Events    []string `json:"events"`
	Active    bool     `json:"active"`
	SSLVerify bool     `json:"ssl_verify"`
}

// Webhook represents a configured webhook
type Webhook struct {
	ID        int64     `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

// File represents a file in the repository
type File struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Size     int64  `json:"size"`
	SHA      string `json:"sha"`
}

// WebhookEvent represents a parsed webhook event
type WebhookEvent struct {
	Type       WebhookEventType `json:"type"`
	Action     string           `json:"action"`
	Provider   ProviderType     `json:"provider"`
	ProjectID  any              `json:"project_id"`
	PRNumber   int              `json:"pr_number"`
	PR         *PullRequest     `json:"pull_request,omitempty"`
	User       *User            `json:"user,omitempty"`
	RawPayload map[string]any   `json:"raw_payload,omitempty"`
}

// WebhookEventType represents the type of webhook event
type WebhookEventType string

const (
	WebhookEventPullRequest       WebhookEventType = "pull_request"
	WebhookEventPullRequestReview WebhookEventType = "pull_request_review"
	WebhookEventPush              WebhookEventType = "push"
	WebhookEventComment           WebhookEventType = "comment"
)

// ProviderConfig holds configuration for a provider
type ProviderConfig struct {
	Type            ProviderType `json:"type"`
	BaseURL         string       `json:"base_url"`
	AccessToken     string       `json:"access_token"`
	WebhookSecret   string       `json:"webhook_secret"`
	RateLimitPerSec int          `json:"rate_limit_per_sec"`
}
