// Package client provides GitLab API types
package client

import "time"

// ============================================================================
// User Types
// ============================================================================

// User represents a GitLab user
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
	State     string `json:"state"`
	IsAdmin   bool   `json:"is_admin"`
}

// ============================================================================
// Project Types
// ============================================================================

// Project represents a GitLab project
type Project struct {
	ID                int64     `json:"id"`
	Name              string    `json:"name"`
	NameWithNamespace string    `json:"name_with_namespace"`
	Path              string    `json:"path"`
	PathWithNamespace string    `json:"path_with_namespace"`
	Description       string    `json:"description"`
	DefaultBranch     string    `json:"default_branch"`
	WebURL            string    `json:"web_url"`
	HTTPURL           string    `json:"http_url_to_repo"`
	SSHURL            string    `json:"ssh_url_to_repo"`
	CreatedAt         time.Time `json:"created_at"`
	LastActivityAt    time.Time `json:"last_activity_at"`
	Visibility        string    `json:"visibility"`
	
	// Permissions
	Permissions *ProjectPermissions `json:"permissions,omitempty"`
	
	// Namespace
	Namespace *Namespace `json:"namespace,omitempty"`
}

// ProjectPermissions represents project access permissions
type ProjectPermissions struct {
	ProjectAccess *ProjectAccess `json:"project_access,omitempty"`
	GroupAccess   *GroupAccess   `json:"group_access,omitempty"`
}

// ProjectAccess represents project-level access
type ProjectAccess struct {
	AccessLevel       int `json:"access_level"`
	NotificationLevel int `json:"notification_level"`
}

// GroupAccess represents group-level access
type GroupAccess struct {
	AccessLevel       int `json:"access_level"`
	NotificationLevel int `json:"notification_level"`
}

// Namespace represents a GitLab namespace
type Namespace struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	FullPath string `json:"full_path"`
	WebURL   string `json:"web_url"`
}

// ListProjectsOptions contains options for listing projects
type ListProjectsOptions struct {
	Search         string
	PerPage        int
	Page           int
	Membership     bool
	MinAccessLevel int // 10=Guest, 20=Reporter, 30=Developer, 40=Maintainer, 50=Owner
}

// ============================================================================
// Merge Request Types
// ============================================================================

// MergeRequest represents a GitLab merge request
type MergeRequest struct {
	ID             int64     `json:"id"`
	IID            int       `json:"iid"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	State          string    `json:"state"`
	TargetBranch   string    `json:"target_branch"`
	SourceBranch   string    `json:"source_branch"`
	SourceProjectID int64    `json:"source_project_id"`
	TargetProjectID int64    `json:"target_project_id"`
	WebURL         string    `json:"web_url"`
	Draft          bool      `json:"draft"`
	WorkInProgress bool      `json:"work_in_progress"`
	
	// Author
	Author *User `json:"author,omitempty"`
	
	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	MergedAt  *time.Time `json:"merged_at,omitempty"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
	
	// Stats
	ChangesCount string `json:"changes_count"`
	
	// Labels
	Labels []string `json:"labels,omitempty"`
	
	// Diff refs
	DiffRefs *DiffRefs `json:"diff_refs,omitempty"`
}

// DiffRefs contains the commit references for a merge request
type DiffRefs struct {
	BaseSha  string `json:"base_sha"`
	HeadSha  string `json:"head_sha"`
	StartSha string `json:"start_sha"`
}

// MergeRequestChanges contains the changes in a merge request
type MergeRequestChanges struct {
	MergeRequest
	Changes []Change `json:"changes"`
}

// Change represents a file change in a merge request
type Change struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	AMode       string `json:"a_mode"`
	BMode       string `json:"b_mode"`
	Diff        string `json:"diff"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
}

// Diff represents a diff in a merge request
type Diff struct {
	OldPath     string `json:"old_path"`
	NewPath     string `json:"new_path"`
	AMode       string `json:"a_mode"`
	BMode       string `json:"b_mode"`
	Diff        string `json:"diff"`
	NewFile     bool   `json:"new_file"`
	RenamedFile bool   `json:"renamed_file"`
	DeletedFile bool   `json:"deleted_file"`
}

// ============================================================================
// File Types
// ============================================================================

// File represents a repository file
type File struct {
	FileName      string `json:"file_name"`
	FilePath      string `json:"file_path"`
	Size          int64  `json:"size"`
	Encoding      string `json:"encoding"`
	ContentSha256 string `json:"content_sha256"`
	Ref           string `json:"ref"`
	BlobID        string `json:"blob_id"`
	CommitID      string `json:"commit_id"`
	LastCommitID  string `json:"last_commit_id"`
	Content       string `json:"content"` // Base64 encoded
}

// ============================================================================
// Note (Comment) Types
// ============================================================================

// Note represents a comment on a merge request
type Note struct {
	ID         int64     `json:"id"`
	Body       string    `json:"body"`
	Author     *User     `json:"author,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	System     bool      `json:"system"`
	NoteableID int64     `json:"noteable_id"`
	NoteableType string  `json:"noteable_type"`
	Resolvable bool      `json:"resolvable"`
	Resolved   bool      `json:"resolved"`
	
	// Position for inline comments
	Position *NotePosition `json:"position,omitempty"`
}

// NotePosition represents the position of an inline comment
type NotePosition struct {
	BaseSha      string  `json:"base_sha"`
	StartSha     string  `json:"start_sha"`
	HeadSha      string  `json:"head_sha"`
	OldPath      string  `json:"old_path"`
	NewPath      string  `json:"new_path"`
	PositionType string  `json:"position_type"` // "text" or "image"
	OldLine      *int    `json:"old_line,omitempty"`
	NewLine      *int    `json:"new_line,omitempty"`
	LineRange    *LineRange `json:"line_range,omitempty"`
}

// LineRange represents a range of lines for multi-line comments
type LineRange struct {
	Start *LinePosition `json:"start,omitempty"`
	End   *LinePosition `json:"end,omitempty"`
}

// LinePosition represents a line position
type LinePosition struct {
	LineCode string `json:"line_code"`
	Type     string `json:"type"` // "new" or "old"
	OldLine  *int   `json:"old_line,omitempty"`
	NewLine  *int   `json:"new_line,omitempty"`
}

// ============================================================================
// Discussion Types
// ============================================================================

// Discussion represents a discussion thread on a merge request
type Discussion struct {
	ID             string  `json:"id"`
	IndividualNote bool    `json:"individual_note"`
	Notes          []Note  `json:"notes"`
}

// CreateDiscussionRequest represents a request to create a discussion
type CreateDiscussionRequest struct {
	Body     string                    `json:"body"`
	Position *CreateDiscussionPosition `json:"position,omitempty"`
}

// CreateDiscussionPosition represents the position for a new discussion
type CreateDiscussionPosition struct {
	BaseSha      string `json:"base_sha"`
	StartSha     string `json:"start_sha"`
	HeadSha      string `json:"head_sha"`
	OldPath      string `json:"old_path,omitempty"`
	NewPath      string `json:"new_path"`
	PositionType string `json:"position_type"` // "text"
	OldLine      *int   `json:"old_line,omitempty"`
	NewLine      *int   `json:"new_line,omitempty"`
}

// ============================================================================
// Webhook Types
// ============================================================================

// Webhook represents a GitLab webhook
type Webhook struct {
	ID                    int64     `json:"id"`
	URL                   string    `json:"url"`
	ProjectID             int64     `json:"project_id"`
	PushEvents            bool      `json:"push_events"`
	MergeRequestsEvents   bool      `json:"merge_requests_events"`
	TagPushEvents         bool      `json:"tag_push_events"`
	NoteEvents            bool      `json:"note_events"`
	PipelineEvents        bool      `json:"pipeline_events"`
	WikiPageEvents        bool      `json:"wiki_page_events"`
	EnableSSLVerification bool      `json:"enable_ssl_verification"`
	CreatedAt             time.Time `json:"created_at"`
}

// CreateWebhookRequest represents a request to create a webhook
type CreateWebhookRequest struct {
	URL                   string `json:"url"`
	Token                 string `json:"token,omitempty"` // Secret token
	PushEvents            bool   `json:"push_events"`
	MergeRequestsEvents   bool   `json:"merge_requests_events"`
	TagPushEvents         bool   `json:"tag_push_events"`
	NoteEvents            bool   `json:"note_events"`
	PipelineEvents        bool   `json:"pipeline_events"`
	WikiPageEvents        bool   `json:"wiki_page_events"`
	EnableSSLVerification bool   `json:"enable_ssl_verification"`
}

// ============================================================================
// Webhook Event Payloads
// ============================================================================

// MergeRequestEvent represents a merge request webhook event
type MergeRequestEvent struct {
	ObjectKind       string                      `json:"object_kind"` // "merge_request"
	EventType        string                      `json:"event_type"`
	User             *User                       `json:"user"`
	Project          *WebhookProject             `json:"project"`
	ObjectAttributes *MergeRequestEventAttributes `json:"object_attributes"`
	Labels           []Label                     `json:"labels,omitempty"`
	Changes          *MergeRequestEventChanges   `json:"changes,omitempty"`
}

// WebhookProject represents a project in webhook payload
type WebhookProject struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	WebURL            string `json:"web_url"`
	GitHTTPURL        string `json:"git_http_url"`
	GitSSHURL         string `json:"git_ssh_url"`
	Namespace         string `json:"namespace"`
	VisibilityLevel   int    `json:"visibility_level"`
	PathWithNamespace string `json:"path_with_namespace"`
	DefaultBranch     string `json:"default_branch"`
}

// MergeRequestEventAttributes represents MR attributes in webhook
type MergeRequestEventAttributes struct {
	ID              int64      `json:"id"`
	IID             int        `json:"iid"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	State           string     `json:"state"`
	TargetBranch    string     `json:"target_branch"`
	SourceBranch    string     `json:"source_branch"`
	SourceProjectID int64      `json:"source_project_id"`
	TargetProjectID int64      `json:"target_project_id"`
	AuthorID        int64      `json:"author_id"`
	URL             string     `json:"url"`
	Action          string     `json:"action"` // "open", "close", "reopen", "update", "merge"
	Draft           bool       `json:"draft"`
	WorkInProgress  bool       `json:"work_in_progress"`
	CreatedAt       string     `json:"created_at"`
	UpdatedAt       string     `json:"updated_at"`
	LastCommit      *Commit    `json:"last_commit,omitempty"`
}

// MergeRequestEventChanges represents changes in MR webhook
type MergeRequestEventChanges struct {
	Title        *StringChange `json:"title,omitempty"`
	Description  *StringChange `json:"description,omitempty"`
	State        *StringChange `json:"state,omitempty"`
	TargetBranch *StringChange `json:"target_branch,omitempty"`
	SourceBranch *StringChange `json:"source_branch,omitempty"`
	Draft        *BoolChange   `json:"draft,omitempty"`
	Labels       *LabelsChange `json:"labels,omitempty"`
}

// StringChange represents a string value change
type StringChange struct {
	Previous string `json:"previous"`
	Current  string `json:"current"`
}

// BoolChange represents a boolean value change
type BoolChange struct {
	Previous bool `json:"previous"`
	Current  bool `json:"current"`
}

// LabelsChange represents a labels change
type LabelsChange struct {
	Previous []Label `json:"previous"`
	Current  []Label `json:"current"`
}

// Label represents a GitLab label
type Label struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Color       string `json:"color"`
	ProjectID   int64  `json:"project_id"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// Commit represents a Git commit
type Commit struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	Title     string    `json:"title"`
	Timestamp time.Time `json:"timestamp"`
	URL       string    `json:"url"`
	Author    *Author   `json:"author,omitempty"`
}

// Author represents a commit author
type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// ============================================================================
// Access Level Constants
// ============================================================================

const (
	AccessLevelGuest      = 10
	AccessLevelReporter   = 20
	AccessLevelDeveloper  = 30
	AccessLevelMaintainer = 40
	AccessLevelOwner      = 50
)

