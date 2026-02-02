// Package models provides data structures for GitLab MR Review integration
package models

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

// ============================================================================
// GitLab Integration - connection to GitLab instance
// ============================================================================

// GitLabIntegration represents a connection to a GitLab instance
type GitLabIntegration struct {
	ID            string                    `json:"id" db:"id"`
	OwnerID       string                    `json:"owner_id" db:"owner_id"`             // User who created the integration
	TenantID      string                    `json:"tenant_id,omitempty" db:"tenant_id"` // Optional: shared within tenant
	Name          string                    `json:"name" db:"name"`
	BaseURL       string                    `json:"base_url" db:"base_url"`
	AccessToken   string                    `json:"-" db:"access_token"`   // Encrypted, never exposed
	WebhookSecret string                    `json:"-" db:"webhook_secret"` // For webhook verification
	Status        GitLabIntegrationStatus   `json:"status" db:"status"`
	LastSyncAt    *time.Time                `json:"last_sync_at,omitempty" db:"last_sync_at"`
	LastError     string                    `json:"last_error,omitempty" db:"last_error"`
	Settings      GitLabIntegrationSettings `json:"settings" db:"settings"`
	CreatedAt     time.Time                 `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at" db:"updated_at"`

	// Computed fields (not stored)
	ProjectCount int `json:"project_count,omitempty" db:"-"`
}

// GitLabIntegrationStatus represents the status of a GitLab integration
type GitLabIntegrationStatus string

const (
	GitLabIntegrationStatusActive   GitLabIntegrationStatus = "active"
	GitLabIntegrationStatusDisabled GitLabIntegrationStatus = "disabled"
	GitLabIntegrationStatusError    GitLabIntegrationStatus = "error"
)

// GitLabIntegrationSettings contains settings for GitLab integration
type GitLabIntegrationSettings struct {
	// Connection settings
	APIVersion     string `json:"api_version,omitempty"`     // Default: "v4"
	RequestTimeout int    `json:"request_timeout,omitempty"` // In seconds, default: 30
	MaxRetries     int    `json:"max_retries,omitempty"`     // Default: 3

	// Rate limiting
	RateLimitPerMin int `json:"rate_limit_per_min,omitempty"` // Default: 30 (safe for GitLab 2000/min)
}

// Scan implements sql.Scanner for GitLabIntegrationSettings
func (s *GitLabIntegrationSettings) Scan(value interface{}) error {
	if value == nil {
		*s = GitLabIntegrationSettings{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			*s = GitLabIntegrationSettings{}
			return nil
		}
		bytes = []byte(str)
	}
	return json.Unmarshal(bytes, s)
}

// ============================================================================
// GitLab Project - repository configuration for review
// ============================================================================

// GitLabProject represents a GitLab project configured for AI review
type GitLabProject struct {
	ID                string              `json:"id" db:"id"`
	IntegrationID     string              `json:"integration_id" db:"integration_id"`
	TenantID          string              `json:"tenant_id,omitempty" db:"tenant_id"`
	GitLabProjectID   int64               `json:"gitlab_project_id" db:"gitlab_project_id"`
	Name              string              `json:"name" db:"name"`
	PathWithNamespace string              `json:"path_with_namespace" db:"path_with_namespace"`
	DefaultBranch     string              `json:"default_branch" db:"default_branch"` // Target branch for indexing
	WebhookID         *int64              `json:"webhook_id,omitempty" db:"webhook_id"`
	Status            GitLabProjectStatus `json:"status" db:"status"`
	AutoReview        bool                `json:"auto_review" db:"auto_review"`

	// Model Configuration
	AnalysisModelID  string `json:"analysis_model_id" db:"analysis_model_id"`   // LLM for review
	EmbeddingModelID string `json:"embedding_model_id" db:"embedding_model_id"` // For code chunking

	// Review Configuration
	ReviewPrompt string                `json:"review_prompt,omitempty" db:"review_prompt"`
	Settings     GitLabProjectSettings `json:"settings" db:"settings"`

	// Indexing status
	IndexStatus   string     `json:"index_status,omitempty" db:"index_status"` // pending, in_progress, completed, failed
	LastIndexedAt *time.Time `json:"last_indexed_at,omitempty" db:"last_indexed_at"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Computed fields (not stored)
	IntegrationName string `json:"integration_name,omitempty" db:"-"`
	ReviewCount     int    `json:"review_count,omitempty" db:"-"`

	// Index statistics
	IndexChunks  int64 `json:"index_chunks,omitempty" db:"index_chunks_count"`
	IndexVectors int64 `json:"index_vectors,omitempty" db:"-"`
}

// GetCollectionName returns the Qdrant collection name for this project.
// If Settings.CollectionName is set, it is returned.
// Otherwise, generates collection name from PathWithNamespace or Name.
func (p *GitLabProject) GetCollectionName() string {
	if p.Settings.CollectionName != "" {
		return p.Settings.CollectionName
	}
	// Generate from path_with_namespace (e.g., "group/project" -> "group-project")
	name := p.PathWithNamespace
	if name == "" {
		name = p.Name
	}
	return SanitizeCollectionName(name)
}

// SanitizeCollectionName converts a project name to a valid Qdrant collection name.
// Converts to lowercase, replaces spaces and special characters with "-".
func SanitizeCollectionName(name string) string {
	// Lowercase
	name = strings.ToLower(name)
	// Replace / with -
	name = strings.ReplaceAll(name, "/", "-")
	// Replace spaces with -
	name = strings.ReplaceAll(name, " ", "-")
	// Replace special characters with -
	re := regexp.MustCompile(`[^a-z0-9_-]`)
	name = re.ReplaceAllString(name, "-")
	// Remove consecutive dashes
	re = regexp.MustCompile(`-+`)
	name = re.ReplaceAllString(name, "-")
	// Trim leading/trailing dashes
	name = strings.Trim(name, "-")
	// Prefix with "gitlab-" for clarity
	if name == "" {
		name = "default"
	}
	return "gitlab-" + name
}

// GitLabProjectStatus represents the status of a GitLab project
type GitLabProjectStatus string

const (
	GitLabProjectStatusActive   GitLabProjectStatus = "active"
	GitLabProjectStatusDisabled GitLabProjectStatus = "disabled"
	GitLabProjectStatusError    GitLabProjectStatus = "error"
)

// GitLabProjectSettings contains settings for a GitLab project
type GitLabProjectSettings struct {
	// File Filters
	IncludePatterns []string `json:"include_patterns,omitempty"` // ["*.go", "*.ts", "*.py"]
	ExcludePatterns []string `json:"exclude_patterns,omitempty"` // ["vendor/*", "node_modules/*", "*.min.js"]

	// Analysis settings
	MaxFilesPerMR   int  `json:"max_files_per_mr,omitempty"`   // Default: 50
	MaxLinesPerFile int  `json:"max_lines_per_file,omitempty"` // Default: 2000
	SkipDraftMRs    bool `json:"skip_draft_mrs,omitempty"`     // Skip WIP/Draft MRs
	SkipBots        bool `json:"skip_bots,omitempty"`          // Skip bot-created MRs

	// LLM settings
	MaxReviewTokens int    `json:"max_review_tokens,omitempty"` // Max tokens for LLM response, default: 8192
	PerFileReview   bool   `json:"per_file_review,omitempty"`   // Review each file separately with tool calling
	ReviewLanguage  string `json:"review_language,omitempty"`   // Language for review output: "en", "ru", etc.

	// Chunking settings
	ChunkSize    int `json:"chunk_size,omitempty"`    // Default: 1000 tokens
	ChunkOverlap int `json:"chunk_overlap,omitempty"` // Default: 100 tokens

	// Branch filters
	TargetBranches []string `json:"target_branches,omitempty"` // Only review MRs to these branches
	IgnoreBranches []string `json:"ignore_branches,omitempty"` // Never review MRs from these branches

	// Qdrant Collection (auto-generated from project name if empty)
	CollectionName string `json:"collection_name,omitempty"` // Qdrant collection for code embeddings
}

// Scan implements sql.Scanner for GitLabProjectSettings
func (s *GitLabProjectSettings) Scan(value interface{}) error {
	if value == nil {
		*s = GitLabProjectSettings{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			*s = GitLabProjectSettings{}
			return nil
		}
		bytes = []byte(str)
	}
	return json.Unmarshal(bytes, s)
}

// ============================================================================
// MR Review - analysis result
// ============================================================================

// GitLabMRReview represents the result of an MR analysis
type GitLabMRReview struct {
	ID            string `json:"id" db:"id"`
	ProjectID     string `json:"project_id" db:"project_id"`
	IntegrationID string `json:"integration_id" db:"integration_id"`

	// MR Information
	MRIID        int    `json:"mr_iid" db:"mr_iid"`
	MRTitle      string `json:"mr_title" db:"mr_title"`
	MRAuthor     string `json:"mr_author" db:"mr_author"`
	MRAuthorID   int64  `json:"mr_author_id" db:"mr_author_id"`
	SourceBranch string `json:"source_branch" db:"source_branch"`
	TargetBranch string `json:"target_branch" db:"target_branch"`
	MRURL        string `json:"mr_url" db:"mr_url"`

	// Review Status
	Status   GitLabReviewStatus   `json:"status" db:"status"`
	Priority GitLabReviewPriority `json:"priority" db:"priority"`

	// Analysis Results
	FilesAnalyzed int                 `json:"files_analyzed" db:"files_analyzed"`
	LinesChanged  int                 `json:"lines_changed" db:"lines_changed"`
	IssuesFound   int                 `json:"issues_found" db:"issues_found"`
	ReviewResult  *GitLabReviewResult `json:"review_result,omitempty" db:"review_result"`

	// GitLab Note
	NoteID       *int64  `json:"note_id,omitempty" db:"note_id"`
	DiscussionID *string `json:"discussion_id,omitempty" db:"discussion_id"`

	// Performance Metrics
	ProcessingTimeMs int64  `json:"processing_time_ms" db:"processing_time_ms"`
	TokensUsed       int    `json:"tokens_used" db:"tokens_used"`
	ModelUsed        string `json:"model_used" db:"model_used"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`

	// Error handling
	Error      string `json:"error,omitempty" db:"error"`
	RetryCount int    `json:"retry_count" db:"retry_count"`
	MaxRetries int    `json:"max_retries" db:"max_retries"`

	// Update tracking
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Computed fields
	ProjectName string `json:"project_name,omitempty" db:"-"`
}

// GitLabReviewStatus represents the status of an MR review
type GitLabReviewStatus string

const (
	GitLabReviewStatusPending   GitLabReviewStatus = "pending"
	GitLabReviewStatusQueued    GitLabReviewStatus = "queued"
	GitLabReviewStatusAnalyzing GitLabReviewStatus = "analyzing"
	GitLabReviewStatusCompleted GitLabReviewStatus = "completed"
	GitLabReviewStatusFailed    GitLabReviewStatus = "failed"
	GitLabReviewStatusCancelled GitLabReviewStatus = "cancelled"
	GitLabReviewStatusSkipped   GitLabReviewStatus = "skipped"
)

// GitLabReviewPriority represents the priority of a review job
type GitLabReviewPriority string

const (
	GitLabReviewPriorityLow    GitLabReviewPriority = "low"
	GitLabReviewPriorityNormal GitLabReviewPriority = "normal"
	GitLabReviewPriorityHigh   GitLabReviewPriority = "high"
	GitLabReviewPriorityUrgent GitLabReviewPriority = "urgent"
)

// GitLabReviewResult contains structured review results
type GitLabReviewResult struct {
	Summary      string                 `json:"summary"`
	OverallScore int                    `json:"overall_score"` // 0-100
	Categories   []GitLabReviewCategory `json:"categories"`
	FileReviews  []GitLabFileReview     `json:"file_reviews,omitempty"`
	Suggestions  []GitLabSuggestion     `json:"suggestions,omitempty"`
}

// Scan implements sql.Scanner for GitLabReviewResult
func (r *GitLabReviewResult) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			return nil
		}
		bytes = []byte(str)
	}
	return json.Unmarshal(bytes, r)
}

// GitLabReviewCategory represents a review category with score
type GitLabReviewCategory struct {
	Name        string `json:"name"`  // "Security", "Performance", "Style", "Bugs"
	Score       int    `json:"score"` // 0-100
	IssueCount  int    `json:"issue_count"`
	Description string `json:"description,omitempty"`
}

// GitLabFileReview contains review for a specific file
type GitLabFileReview struct {
	FilePath     string            `json:"file_path"`
	Language     string            `json:"language,omitempty"`
	LinesAdded   int               `json:"lines_added"`
	LinesRemoved int               `json:"lines_removed"`
	Issues       []GitLabCodeIssue `json:"issues,omitempty"`
	Approved     bool              `json:"approved"`
}

// GitLabCodeIssue represents an issue found in code
type GitLabCodeIssue struct {
	Line        int                 `json:"line"`
	EndLine     *int                `json:"end_line,omitempty"`
	Severity    GitLabIssueSeverity `json:"severity"`
	Category    string              `json:"category"` // "security", "bug", "style", "performance"
	Message     string              `json:"message"`
	Suggestion  string              `json:"suggestion,omitempty"`
	CodeSnippet string              `json:"code_snippet,omitempty"`
}

// GitLabIssueSeverity represents the severity of a code issue
type GitLabIssueSeverity string

const (
	GitLabIssueSeverityCritical GitLabIssueSeverity = "critical"
	GitLabIssueSeverityHigh     GitLabIssueSeverity = "high"
	GitLabIssueSeverityMedium   GitLabIssueSeverity = "medium"
	GitLabIssueSeverityLow      GitLabIssueSeverity = "low"
	GitLabIssueSeverityInfo     GitLabIssueSeverity = "info"
)

// GitLabSuggestion represents a general suggestion for the MR
type GitLabSuggestion struct {
	FilePath    string `json:"file_path,omitempty"` // File this suggestion relates to
	Line        int    `json:"line,omitempty"`      // Line number if applicable
	Type        string `json:"type"`                // "improvement", "best_practice", "documentation"
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"` // "high", "medium", "low"
}

// ============================================================================
// Analysis Job - queue item
// ============================================================================

// GitLabAnalysisJob represents a job in the analysis queue
type GitLabAnalysisJob struct {
	ID            string `json:"id" db:"id"`
	ReviewID      string `json:"review_id" db:"review_id"`
	ProjectID     string `json:"project_id" db:"project_id"`
	IntegrationID string `json:"integration_id" db:"integration_id"`

	// Job Info
	Status   GitLabJobStatus      `json:"status" db:"status"`
	Priority GitLabReviewPriority `json:"priority" db:"priority"`

	// MR Info (denormalized for quick access)
	MRIID   int    `json:"mr_iid" db:"mr_iid"`
	MRTitle string `json:"mr_title" db:"mr_title"`

	// Worker Info
	WorkerID *string `json:"worker_id,omitempty" db:"worker_id"`

	// Timestamps
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`

	// Retry Info
	RetryCount  int        `json:"retry_count" db:"retry_count"`
	MaxRetries  int        `json:"max_retries" db:"max_retries"`
	LastError   string     `json:"last_error,omitempty" db:"last_error"`
	NextRetryAt *time.Time `json:"next_retry_at,omitempty" db:"next_retry_at"`

	// Update tracking
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Config (JSON)
	Config *GitLabJobConfig `json:"config,omitempty" db:"config"`

	// Computed fields from joins (not stored in jobs table)
	MRAuthor         string `json:"mr_author,omitempty" db:"-"`
	SourceBranch     string `json:"source_branch,omitempty" db:"-"`
	TargetBranch     string `json:"target_branch,omitempty" db:"-"`
	ProjectName      string `json:"project_name,omitempty" db:"-"`
	AnalysisModelID  string `json:"analysis_model_id,omitempty" db:"-"`
	EmbeddingModelID string `json:"embedding_model_id,omitempty" db:"-"`
	ReviewPrompt     string `json:"review_prompt,omitempty" db:"-"`
}

// GitLabJobConfig configuration for analysis job
type GitLabJobConfig struct {
	// Analysis settings
	AnalysisModel  string   `json:"analysis_model,omitempty"`
	EmbeddingModel string   `json:"embedding_model,omitempty"`
	ReviewPrompt   string   `json:"review_prompt,omitempty"`
	FileFilters    []string `json:"file_filters,omitempty"`
	MaxFiles       int      `json:"max_files,omitempty"`
	MaxTokens      int      `json:"max_tokens,omitempty"`
}

// Scan implements sql.Scanner for GitLabJobConfig
func (c *GitLabJobConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		str, ok := value.(string)
		if !ok {
			return nil
		}
		bytes = []byte(str)
	}
	return json.Unmarshal(bytes, c)
}

// GitLabJobStatus represents the status of an analysis job
type GitLabJobStatus string

const (
	GitLabJobStatusPending    GitLabJobStatus = "pending"
	GitLabJobStatusProcessing GitLabJobStatus = "processing"
	GitLabJobStatusCompleted  GitLabJobStatus = "completed"
	GitLabJobStatusFailed     GitLabJobStatus = "failed"
	GitLabJobStatusCancelled  GitLabJobStatus = "cancelled"
	GitLabJobStatusRetrying   GitLabJobStatus = "retrying"
)

// ============================================================================
// Webhook Event - for deduplication
// ============================================================================

// GitLabWebhookEvent represents a received webhook event
type GitLabWebhookEvent struct {
	ID            string              `json:"id" db:"id"`
	IntegrationID string              `json:"integration_id" db:"integration_id"`
	ProjectID     int64               `json:"project_id" db:"project_id"`
	MRIID         int                 `json:"mr_iid" db:"mr_iid"`
	EventType     string              `json:"event_type" db:"event_type"`     // "merge_request"
	Action        string              `json:"action" db:"action"`             // "open", "update", "reopen"
	ObjectID      int64               `json:"object_id" db:"object_id"`       // GitLab object_attributes.id
	Payload       string              `json:"payload,omitempty" db:"payload"` // JSON payload
	Status        GitLabWebhookStatus `json:"status" db:"status"`
	ReceivedAt    time.Time           `json:"received_at" db:"received_at"`
	ProcessedAt   *time.Time          `json:"processed_at,omitempty" db:"processed_at"`
	Deduplicated  bool                `json:"deduplicated" db:"deduplicated"` // Was this a duplicate?
}

// GitLabWebhookStatus represents the status of a webhook event
type GitLabWebhookStatus string

const (
	GitLabWebhookStatusPending      GitLabWebhookStatus = "pending"
	GitLabWebhookStatusProcessed    GitLabWebhookStatus = "processed"
	GitLabWebhookStatusDeduplicated GitLabWebhookStatus = "deduplicated"
	GitLabWebhookStatusFailed       GitLabWebhookStatus = "failed"
)

// ============================================================================
// Request/Response types for API
// ============================================================================

// CreateGitLabIntegrationRequest represents request to create a GitLab integration
type CreateGitLabIntegrationRequest struct {
	Name        string                     `json:"name" binding:"required"`
	BaseURL     string                     `json:"base_url" binding:"required"`
	AccessToken string                     `json:"access_token" binding:"required"`
	Settings    *GitLabIntegrationSettings `json:"settings,omitempty"`
}

// UpdateGitLabIntegrationRequest represents request to update a GitLab integration
type UpdateGitLabIntegrationRequest struct {
	Name        *string                    `json:"name,omitempty"`
	BaseURL     *string                    `json:"base_url,omitempty"`
	AccessToken *string                    `json:"access_token,omitempty"`
	Status      *GitLabIntegrationStatus   `json:"status,omitempty"`
	Settings    *GitLabIntegrationSettings `json:"settings,omitempty"`
}

// CreateGitLabProjectRequest represents request to add a project for review
type CreateGitLabProjectRequest struct {
	GitLabProjectID  int64                  `json:"gitlab_project_id" binding:"required"`
	AutoReview       bool                   `json:"auto_review"`
	AnalysisModelID  string                 `json:"analysis_model_id" binding:"required"`
	EmbeddingModelID string                 `json:"embedding_model_id" binding:"required"`
	ReviewPrompt     string                 `json:"review_prompt,omitempty"`
	Settings         *GitLabProjectSettings `json:"settings,omitempty"`
}

// UpdateGitLabProjectRequest represents request to update project settings
type UpdateGitLabProjectRequest struct {
	Name             *string                `json:"name,omitempty"`
	TenantID         *string                `json:"tenant_id,omitempty"`
	AutoReview       *bool                  `json:"auto_review,omitempty"`
	Status           *GitLabProjectStatus   `json:"status,omitempty"`
	DefaultBranch    *string                `json:"default_branch,omitempty"`
	AnalysisModelID  *string                `json:"analysis_model_id,omitempty"`
	EmbeddingModelID *string                `json:"embedding_model_id,omitempty"`
	ReviewPrompt     *string                `json:"review_prompt,omitempty"`
	Settings         *GitLabProjectSettings `json:"settings,omitempty"`
}

// GitLabIntegrationListRequest represents request for listing integrations
type GitLabIntegrationListRequest struct {
	Limit  int                      `json:"limit" form:"limit"`
	Offset int                      `json:"offset" form:"offset"`
	Status *GitLabIntegrationStatus `json:"status,omitempty" form:"status"`
	Search *string                  `json:"search,omitempty" form:"search"`
}

// GitLabProjectListRequest represents request for listing projects
type GitLabProjectListRequest struct {
	Limit         int                  `json:"limit" form:"limit"`
	Offset        int                  `json:"offset" form:"offset"`
	IntegrationID *string              `json:"integration_id,omitempty" form:"integration_id"`
	Status        *GitLabProjectStatus `json:"status,omitempty" form:"status"`
	Search        *string              `json:"search,omitempty" form:"search"`
}

// GitLabReviewListRequest represents request for listing reviews
type GitLabReviewListRequest struct {
	Limit         int                 `json:"limit" form:"limit"`
	Offset        int                 `json:"offset" form:"offset"`
	ProjectID     *string             `json:"project_id,omitempty" form:"project_id"`
	IntegrationID *string             `json:"integration_id,omitempty" form:"integration_id"`
	Status        *GitLabReviewStatus `json:"status,omitempty" form:"status"`
	MRIID         *int                `json:"mr_iid,omitempty" form:"mr_iid"`
	DateFrom      *time.Time          `json:"date_from,omitempty" form:"date_from"`
	DateTo        *time.Time          `json:"date_to,omitempty" form:"date_to"`
	Search        *string             `json:"search,omitempty" form:"search"`
}

// GitLabListResponse is a generic paginated response
type GitLabListResponse[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// AvailableModelsResponse represents available models for GitLab config
type AvailableModelsResponse struct {
	LLMModels       []ModelOption `json:"llm_models"`
	EmbeddingModels []ModelOption `json:"embedding_models"`
}

// ModelOption represents a model option for selection
type ModelOption struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Type     string `json:"type"`
}

// ModelUsageResponse represents model usage check result
type ModelUsageResponse struct {
	ModelID        string             `json:"model_id"`
	IsUsedInGitLab bool               `json:"is_used_in_gitlab"`
	GitLabProjects []GitLabProjectRef `json:"gitlab_projects,omitempty"`
	CanDeactivate  bool               `json:"can_deactivate"`
	BlockingReason string             `json:"blocking_reason,omitempty"`
}

// GitLabProjectRef represents a reference to a GitLab project
type GitLabProjectRef struct {
	IntegrationID   string `json:"integration_id"`
	IntegrationName string `json:"integration_name"`
	ProjectID       string `json:"project_id"`
	ProjectName     string `json:"project_name"`
	UsageType       string `json:"usage_type"` // "analysis" | "embedding"
}

// GitLabQueueStats represents queue statistics
type GitLabQueueStats struct {
	Pending    int `json:"pending"`
	Processing int `json:"processing"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	Cancelled  int `json:"cancelled"`

	// Worker stats
	ActiveWorkers int `json:"active_workers"`
	IdleWorkers   int `json:"idle_workers"`
	TotalWorkers  int `json:"total_workers"`

	// Performance
	AvgProcessingTimeMs int64 `json:"avg_processing_time_ms"`
	JobsLastHour        int   `json:"jobs_last_hour"`
	JobsLast24Hours     int   `json:"jobs_last_24_hours"`
	CompletedToday      int   `json:"completed_today"`
}

// ============================================================================
// Feedback Models
// ============================================================================

// GitLabReviewFeedback represents user feedback on a code review
type GitLabReviewFeedback struct {
	ID           string    `json:"id" db:"id"`
	ReviewID     string    `json:"review_id" db:"review_id"`
	UserID       *string   `json:"user_id,omitempty" db:"user_id"`
	Rating       int       `json:"rating" db:"rating"`
	FeedbackType string    `json:"feedback_type" db:"feedback_type"`
	Comment      string    `json:"comment,omitempty" db:"comment"`
	IssueIndex   *int      `json:"issue_index,omitempty" db:"issue_index"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// FeedbackType constants
const (
	FeedbackTypeGeneral       = "general"
	FeedbackTypeAccuracy      = "accuracy"
	FeedbackTypeHelpfulness   = "helpfulness"
	FeedbackTypeFalsePositive = "false_positive"
	FeedbackTypeMissedIssue   = "missed_issue"
)

// GitLabFeedbackListRequest request parameters for listing feedback
type GitLabFeedbackListRequest struct {
	ReviewID     *string `json:"review_id,omitempty"`
	FeedbackType *string `json:"feedback_type,omitempty"`
	MinRating    *int    `json:"min_rating,omitempty"`
	Limit        int     `json:"limit"`
	Offset       int     `json:"offset"`
}

// GitLabFeedbackStats represents aggregated feedback statistics (v4.1.1+)
type GitLabFeedbackStats struct {
	TotalFeedback int                          `json:"total_feedback"`
	ApprovedCount int                          `json:"approved_count"`
	RejectedCount int                          `json:"rejected_count"`
	EditedCount   int                          `json:"edited_count"`
	IgnoredCount  int                          `json:"ignored_count"`
	ApprovalRate  float64                      `json:"approval_rate"`
	AccuracyRate  float64                      `json:"accuracy_rate"`
	ByCategory    []GitLabFeedbackCategoryStat `json:"by_category"`
}

// GitLabFeedbackCategoryStat represents feedback stats per category
type GitLabFeedbackCategoryStat struct {
	Category      string  `json:"category"`
	TotalIssues   int     `json:"total_issues"`
	ApprovedCount int     `json:"approved_count"`
	RejectedCount int     `json:"rejected_count"`
	AccuracyRate  float64 `json:"accuracy_rate"`
}

// ============================================================================
// Analytics Models
// ============================================================================

// GitLabAnalytics represents aggregated analytics data
type GitLabAnalytics struct {
	Range            string         `json:"range"`
	Days             int            `json:"days"`
	TotalReviews     int            `json:"total_reviews"`
	CompletedReviews int            `json:"completed_reviews"`
	FailedReviews    int            `json:"failed_reviews"`
	PendingReviews   int            `json:"pending_reviews"`
	AvgProcessingMs  int64          `json:"avg_processing_ms"`
	TotalIssuesFound int            `json:"total_issues_found"`
	TotalTokensUsed  int64          `json:"total_tokens_used"`
	ReviewsByDay     []DayStats     `json:"reviews_by_day"`
	ReviewsByProject []ProjectStats `json:"reviews_by_project"`
	ReviewsByStatus  map[string]int `json:"reviews_by_status"`
	AvgRating        float64        `json:"avg_rating"`
	FeedbackCount    int            `json:"feedback_count"`
}

// DayStats represents daily statistics
type DayStats struct {
	Date             string `json:"date"`
	TotalReviews     int    `json:"total_reviews"`
	CompletedReviews int    `json:"completed_reviews"`
	FailedReviews    int    `json:"failed_reviews"`
	IssuesFound      int    `json:"issues_found"`
}

// ProjectStats represents per-project statistics
type ProjectStats struct {
	ProjectID        string `json:"project_id"`
	ProjectName      string `json:"project_name"`
	TotalReviews     int    `json:"total_reviews"`
	CompletedReviews int    `json:"completed_reviews"`
	IssuesFound      int    `json:"issues_found"`
}

// ============================================================================
// Scan History Models (v4.1.2+)
// ============================================================================

// GitLabScanType represents the type of scan
type GitLabScanType string

const (
	ScanTypeSecrets      GitLabScanType = "secrets"
	ScanTypeSecretsDeep  GitLabScanType = "secrets_deep"
	ScanTypeDependencies GitLabScanType = "dependencies"
	ScanTypeQuality      GitLabScanType = "quality"
	ScanTypeDeadCode     GitLabScanType = "deadcode"
	ScanTypeAutoDocs     GitLabScanType = "autodocs"
	ScanTypeTestGen      GitLabScanType = "testgen"
	ScanTypeArchitecture GitLabScanType = "architecture"
)

// GitLabScanStatus represents the status of a scan
type GitLabScanStatus string

const (
	ScanStatusPending   GitLabScanStatus = "pending"
	ScanStatusRunning   GitLabScanStatus = "running"
	ScanStatusCompleted GitLabScanStatus = "completed"
	ScanStatusFailed    GitLabScanStatus = "failed"
	ScanStatusCancelled GitLabScanStatus = "cancelled"
)

// GitLabScanResult represents a stored scan result
type GitLabScanResult struct {
	ID            string           `json:"id" db:"id"`
	ProjectID     string           `json:"project_id" db:"project_id"`
	IntegrationID string           `json:"integration_id" db:"integration_id"`
	ScanType      GitLabScanType   `json:"scan_type" db:"scan_type"`
	Status        GitLabScanStatus `json:"status" db:"status"`
	ModelID       string           `json:"model_id,omitempty" db:"model_id"`

	// Timing
	StartedAt   time.Time  `json:"started_at" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs  int64      `json:"duration_ms" db:"duration_ms"`

	// Results summary
	FindingsCount int    `json:"findings_count" db:"findings_count"`
	FilesAffected int    `json:"files_affected" db:"files_affected"`
	TokensUsed    int    `json:"tokens_used" db:"tokens_used"`
	Error         string `json:"error,omitempty" db:"error"`

	// Full results as JSON
	ResultsJSON string `json:"results_json,omitempty" db:"results_json"`

	// For display
	ProjectName string `json:"project_name,omitempty" db:"-"`
}

// GitLabScanResultsRequest for listing scan results
type GitLabScanResultsRequest struct {
	ProjectID      string            `json:"project_id,omitempty"`
	IntegrationID  string            `json:"integration_id,omitempty"`
	IntegrationIDs []string          `json:"integration_ids,omitempty"`
	ScanType       *GitLabScanType   `json:"scan_type,omitempty"`
	Status         *GitLabScanStatus `json:"status,omitempty"`
	Limit          int               `json:"limit,omitempty"`
	Offset         int               `json:"offset,omitempty"`
}

// ============================================================================
// User Background Jobs (v4.8.9+)
// ============================================================================

// UserJobType represents the type of background job
type UserJobType string

const (
	JobTypeSecretsScn     UserJobType = "secrets_scan"
	JobTypeDeepSecretsScan UserJobType = "deep_secrets_scan"
	JobTypeSASTScan        UserJobType = "sast_scan"
	JobTypeDependencyScan  UserJobType = "dependency_scan"
	JobTypeQualityScan     UserJobType = "quality_scan"
	JobTypeDeadCodeScan    UserJobType = "deadcode_scan"
	JobTypeAutoDocsScan    UserJobType = "autodocs_scan"
	JobTypeTestGenScan     UserJobType = "testgen_scan"
	JobTypeProjectIndex    UserJobType = "project_index"
	JobTypeGenerateDocs    UserJobType = "generate_docs"
	JobTypeGenerateTests   UserJobType = "generate_tests"
	JobTypeCreateMR        UserJobType = "create_mr"
)

// UserJobStatus represents the status of a background job
type UserJobStatus string

const (
	UserJobStatusPending    UserJobStatus = "pending"
	UserJobStatusRunning    UserJobStatus = "running"
	UserJobStatusCompleted  UserJobStatus = "completed"
	UserJobStatusFailed     UserJobStatus = "failed"
	UserJobStatusCancelled  UserJobStatus = "cancelled"
)

// UserJob represents a user-initiated background job
type UserJob struct {
	ID            string        `json:"id" db:"id"`
	UserID        string        `json:"user_id" db:"user_id"`
	TenantID      string        `json:"tenant_id,omitempty" db:"tenant_id"`
	ProjectID     string        `json:"project_id" db:"project_id"`
	IntegrationID string        `json:"integration_id" db:"integration_id"`
	JobType       UserJobType   `json:"job_type" db:"job_type"`
	Status        UserJobStatus `json:"status" db:"status"`

	// Job configuration (JSON)
	Config string `json:"config,omitempty" db:"config"`

	// Progress tracking
	Progress    int    `json:"progress" db:"progress"`         // 0-100 percentage
	ProgressMsg string `json:"progress_msg,omitempty" db:"progress_msg"` // Current step description

	// Timing
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`

	// Results
	ResultID    string `json:"result_id,omitempty" db:"result_id"`       // Reference to scan result or MR
	ResultType  string `json:"result_type,omitempty" db:"result_type"`   // "scan_result", "merge_request", etc.
	ResultURL   string `json:"result_url,omitempty" db:"result_url"`     // External URL (e.g., MR URL)
	Error       string `json:"error,omitempty" db:"error"`

	// For display
	ProjectName string `json:"project_name,omitempty" db:"-"`
}

// UserJobConfig contains configuration for a background job
type UserJobConfig struct {
	// Common
	ModelID  string `json:"model_id,omitempty"`
	Language string `json:"language,omitempty"`

	// Scan specific
	MaxChunks   int      `json:"max_chunks,omitempty"`
	Categories  []string `json:"categories,omitempty"`
	MinSeverity string   `json:"min_severity,omitempty"`

	// Generation specific
	TargetBranch string   `json:"target_branch,omitempty"`
	FilePaths    []string `json:"file_paths,omitempty"`
	CreateMR     bool     `json:"create_mr,omitempty"`
}

// UserJobsRequest for listing user jobs
type UserJobsRequest struct {
	UserID        string         `json:"user_id,omitempty"`
	ProjectID     string         `json:"project_id,omitempty"`
	IntegrationID string         `json:"integration_id,omitempty"`
	JobType       *UserJobType   `json:"job_type,omitempty"`
	Status        *UserJobStatus `json:"status,omitempty"`
	Limit         int            `json:"limit,omitempty"`
	Offset        int            `json:"offset,omitempty"`
}

// ToScanType converts UserJobType to GitLabScanType where applicable
func (jt UserJobType) ToScanType() (GitLabScanType, bool) {
	switch jt {
	case JobTypeSecretsScn:
		return ScanTypeSecrets, true
	case JobTypeDeepSecretsScan:
		return ScanTypeSecretsDeep, true
	case JobTypeDependencyScan:
		return ScanTypeDependencies, true
	case JobTypeQualityScan:
		return ScanTypeQuality, true
	case JobTypeDeadCodeScan:
		return ScanTypeDeadCode, true
	case JobTypeAutoDocsScan:
		return ScanTypeAutoDocs, true
	case JobTypeTestGenScan:
		return ScanTypeTestGen, true
	default:
		return "", false
	}
}
