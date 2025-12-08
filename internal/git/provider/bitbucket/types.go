// Package bitbucket provides Bitbucket API types
package bitbucket

import "time"

// BitbucketPullRequest represents a Bitbucket PR
type BitbucketPullRequest struct {
	ID          int64             `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	State       string            `json:"state"` // OPEN, MERGED, DECLINED, SUPERSEDED
	Author      *BitbucketUser    `json:"author"`
	Source      BitbucketRef      `json:"source"`
	Destination BitbucketRef      `json:"destination"`
	Links       BitbucketLinks    `json:"links"`
	CreatedOn   time.Time         `json:"created_on"`
	UpdatedOn   time.Time         `json:"updated_on"`
	MergeCommit *BitbucketCommit  `json:"merge_commit,omitempty"`
	CloseSource bool              `json:"close_source_branch"`
	Reviewers   []BitbucketUser   `json:"reviewers"`
	Participants []BitbucketParticipant `json:"participants"`
}

// BitbucketRef represents a branch reference
type BitbucketRef struct {
	Branch     BitbucketBranch     `json:"branch"`
	Commit     BitbucketCommit     `json:"commit"`
	Repository *BitbucketRepository `json:"repository,omitempty"`
}

// BitbucketBranch represents a branch
type BitbucketBranch struct {
	Name string `json:"name"`
}

// BitbucketCommit represents a commit
type BitbucketCommit struct {
	Hash    string             `json:"hash"`
	Message string             `json:"message,omitempty"`
	Author  *BitbucketUser     `json:"author,omitempty"`
	Date    time.Time          `json:"date,omitempty"`
	Links   BitbucketLinks     `json:"links,omitempty"`
	Parents []BitbucketCommit  `json:"parents,omitempty"`
}

// BitbucketUser represents a user
type BitbucketUser struct {
	UUID        string         `json:"uuid"`
	Username    string         `json:"username"`
	DisplayName string         `json:"display_name"`
	AccountID   string         `json:"account_id"`
	Type        string         `json:"type"` // user, team
	Links       BitbucketLinks `json:"links"`
}

// BitbucketParticipant represents a PR participant
type BitbucketParticipant struct {
	User     *BitbucketUser `json:"user"`
	Role     string         `json:"role"` // PARTICIPANT, REVIEWER
	Approved bool           `json:"approved"`
}

// BitbucketRepository represents a repository
type BitbucketRepository struct {
	UUID        string             `json:"uuid"`
	Name        string             `json:"name"`
	FullName    string             `json:"full_name"`
	Description string             `json:"description"`
	IsPrivate   bool               `json:"is_private"`
	Language    string             `json:"language"`
	MainBranch  BitbucketBranch    `json:"mainbranch"`
	Owner       *BitbucketUser     `json:"owner"`
	Project     *BitbucketProject  `json:"project,omitempty"`
	Links       BitbucketRepoLinks `json:"links"`
	CreatedOn   time.Time          `json:"created_on"`
	UpdatedOn   time.Time          `json:"updated_on"`
}

// BitbucketProject represents a project
type BitbucketProject struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

// BitbucketLinks represents common links
type BitbucketLinks struct {
	Self   BitbucketLink `json:"self,omitempty"`
	HTML   BitbucketLink `json:"html,omitempty"`
	Avatar BitbucketLink `json:"avatar,omitempty"`
}

// BitbucketRepoLinks represents repository links
type BitbucketRepoLinks struct {
	Self   BitbucketLink   `json:"self"`
	HTML   BitbucketLink   `json:"html"`
	Avatar BitbucketLink   `json:"avatar"`
	Clone  []BitbucketLink `json:"clone"`
}

// BitbucketLink represents a single link
type BitbucketLink struct {
	Href string `json:"href"`
	Name string `json:"name,omitempty"`
}

// BitbucketComment represents a comment
type BitbucketComment struct {
	ID        int64              `json:"id"`
	Content   BitbucketContent   `json:"content"`
	User      *BitbucketUser     `json:"user"`
	Inline    *BitbucketInline   `json:"inline,omitempty"`
	Links     BitbucketLinks     `json:"links"`
	CreatedOn time.Time          `json:"created_on"`
	UpdatedOn time.Time          `json:"updated_on"`
	Parent    *BitbucketComment  `json:"parent,omitempty"`
}

// BitbucketContent represents content
type BitbucketContent struct {
	Raw    string `json:"raw"`
	Markup string `json:"markup"`
	HTML   string `json:"html"`
}

// BitbucketInline represents inline comment position
type BitbucketInline struct {
	Path string `json:"path"`
	From *int   `json:"from,omitempty"`
	To   *int   `json:"to,omitempty"`
}

// BitbucketWebhook represents a webhook
type BitbucketWebhook struct {
	UUID        string         `json:"uuid"`
	URL         string         `json:"url"`
	Description string         `json:"description"`
	Active      bool           `json:"active"`
	Events      []string       `json:"events"`
	CreatedOn   time.Time      `json:"created_at"`
	Links       BitbucketLinks `json:"links"`
}

// BitbucketDiffstat represents diff statistics
type BitbucketDiffstat struct {
	Values []BitbucketDiffstatEntry `json:"values"`
	Page   int                      `json:"page"`
	Size   int                      `json:"size"`
}

// BitbucketDiffstatEntry represents a single file diff stat
type BitbucketDiffstatEntry struct {
	Type         string              `json:"type"`
	Status       string              `json:"status"` // added, removed, modified, renamed
	LinesAdded   int                 `json:"lines_added"`
	LinesRemoved int                 `json:"lines_removed"`
	Old          BitbucketDiffFile   `json:"old"`
	New          BitbucketDiffFile   `json:"new"`
}

// BitbucketDiffFile represents a file in diff
type BitbucketDiffFile struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	Commit struct {
		Hash string `json:"hash"`
	} `json:"commit"`
}

// BitbucketPullRequestEvent represents a PR webhook event
type BitbucketPullRequestEvent struct {
	Actor       *BitbucketUser       `json:"actor"`
	PullRequest *BitbucketPullRequest `json:"pullrequest"`
	Repository  *BitbucketRepository  `json:"repository"`
}

// BitbucketPushEvent represents a push webhook event
type BitbucketPushEvent struct {
	Actor      *BitbucketUser       `json:"actor"`
	Repository *BitbucketRepository `json:"repository"`
	Push       BitbucketPushData    `json:"push"`
}

// BitbucketPushData represents push data
type BitbucketPushData struct {
	Changes []BitbucketPushChange `json:"changes"`
}

// BitbucketPushChange represents a push change
type BitbucketPushChange struct {
	New       *BitbucketRef     `json:"new"`
	Old       *BitbucketRef     `json:"old"`
	Created   bool              `json:"created"`
	Closed    bool              `json:"closed"`
	Forced    bool              `json:"forced"`
	Commits   []BitbucketCommit `json:"commits"`
	Truncated bool              `json:"truncated"`
}

