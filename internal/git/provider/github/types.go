// Package github provides GitHub API types
package github

import "time"

// GitHubPullRequest represents a GitHub PR
type GitHubPullRequest struct {
	ID        int64      `json:"id"`
	Number    int        `json:"number"`
	State     string     `json:"state"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Draft     bool       `json:"draft"`
	HTMLURL   string     `json:"html_url"`
	User      *GitHubUser `json:"user"`
	Head      GitHubRef  `json:"head"`
	Base      GitHubRef  `json:"base"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	ClosedAt  *time.Time `json:"closed_at"`
	MergedAt  *time.Time `json:"merged_at"`
	MergeCommitSHA string `json:"merge_commit_sha"`
	Mergeable *bool      `json:"mergeable"`
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
	ChangedFiles int     `json:"changed_files"`
}

// GitHubRef represents a Git reference
type GitHubRef struct {
	Ref  string           `json:"ref"`
	SHA  string           `json:"sha"`
	Repo *GitHubRepository `json:"repo,omitempty"`
}

// GitHubPullRequestFile represents a file in a PR
type GitHubPullRequestFile struct {
	SHA              string `json:"sha"`
	Filename         string `json:"filename"`
	Status           string `json:"status"` // added, removed, modified, renamed
	Additions        int    `json:"additions"`
	Deletions        int    `json:"deletions"`
	Changes          int    `json:"changes"`
	Patch            string `json:"patch"`
	PreviousFilename string `json:"previous_filename,omitempty"`
	BlobURL          string `json:"blob_url"`
	RawURL           string `json:"raw_url"`
	ContentsURL      string `json:"contents_url"`
}

// GitHubUser represents a GitHub user
type GitHubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"` // User, Bot, Organization
}

// GitHubRepository represents a GitHub repository
type GitHubRepository struct {
	ID            int64        `json:"id"`
	Name          string       `json:"name"`
	FullName      string       `json:"full_name"`
	Description   string       `json:"description"`
	DefaultBranch string       `json:"default_branch"`
	HTMLURL       string       `json:"html_url"`
	CloneURL      string       `json:"clone_url"`
	SSHURL        string       `json:"ssh_url"`
	Private       bool         `json:"private"`
	Archived      bool         `json:"archived"`
	Disabled      bool         `json:"disabled"`
	Language      string       `json:"language"`
	Owner         *GitHubUser  `json:"owner"`
	CreatedAt     time.Time    `json:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
}

// GitHubComment represents a comment
type GitHubComment struct {
	ID        int64       `json:"id"`
	Body      string      `json:"body"`
	User      *GitHubUser `json:"user"`
	HTMLURL   string      `json:"html_url"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	
	// For PR review comments
	Path        string `json:"path,omitempty"`
	Line        int    `json:"line,omitempty"`
	Side        string `json:"side,omitempty"`
	CommitID    string `json:"commit_id,omitempty"`
	DiffHunk    string `json:"diff_hunk,omitempty"`
}

// GitHubWebhook represents a webhook
type GitHubWebhook struct {
	ID        int64              `json:"id"`
	Name      string             `json:"name"`
	Active    bool               `json:"active"`
	Events    []string           `json:"events"`
	Config    GitHubWebhookConfig `json:"config"`
	CreatedAt time.Time          `json:"created_at"`
	UpdatedAt time.Time          `json:"updated_at"`
}

// GitHubWebhookConfig represents webhook configuration
type GitHubWebhookConfig struct {
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Secret      string `json:"secret,omitempty"`
	InsecureSSL string `json:"insecure_ssl"`
}

// GitHubContent represents file content
type GitHubContent struct {
	Type        string `json:"type"` // file, dir, symlink, submodule
	Encoding    string `json:"encoding"` // base64
	Size        int64  `json:"size"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Content     string `json:"content"`
	SHA         string `json:"sha"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
}

// GitHubPullRequestEvent represents a PR webhook event
type GitHubPullRequestEvent struct {
	Action      string             `json:"action"`
	Number      int                `json:"number"`
	PullRequest *GitHubPullRequest `json:"pull_request"`
	Repository  *GitHubRepository  `json:"repository"`
	Sender      *GitHubUser        `json:"sender"`
	Changes     *GitHubPRChanges   `json:"changes,omitempty"`
}

// GitHubPRChanges represents changes in a PR event
type GitHubPRChanges struct {
	Title struct {
		From string `json:"from"`
	} `json:"title,omitempty"`
	Body struct {
		From string `json:"from"`
	} `json:"body,omitempty"`
	Base struct {
		Ref struct {
			From string `json:"from"`
		} `json:"ref,omitempty"`
		SHA struct {
			From string `json:"from"`
		} `json:"sha,omitempty"`
	} `json:"base,omitempty"`
}

// GitHubPushEvent represents a push webhook event
type GitHubPushEvent struct {
	Ref        string            `json:"ref"`
	Before     string            `json:"before"`
	After      string            `json:"after"`
	Created    bool              `json:"created"`
	Deleted    bool              `json:"deleted"`
	Forced     bool              `json:"forced"`
	Pusher     *GitHubUser       `json:"pusher"`
	Repository *GitHubRepository `json:"repository"`
	Sender     *GitHubUser       `json:"sender"`
	Commits    []GitHubCommit    `json:"commits"`
	HeadCommit *GitHubCommit     `json:"head_commit"`
}

// GitHubCommit represents a commit
type GitHubCommit struct {
	ID        string      `json:"id"`
	Message   string      `json:"message"`
	Timestamp time.Time   `json:"timestamp"`
	URL       string      `json:"url"`
	Author    *GitHubUser `json:"author"`
	Committer *GitHubUser `json:"committer"`
	Added     []string    `json:"added"`
	Removed   []string    `json:"removed"`
	Modified  []string    `json:"modified"`
}

