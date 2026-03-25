// Package testing provides test utilities for GitLab integration
package testing

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"

	"aigateway/internal/gitlab/client"
)

// MockGitLabServer simulates GitLab API for testing
type MockGitLabServer struct {
	*httptest.Server
	mu sync.RWMutex

	// Data stores
	projects      map[int64]*client.Project
	mergeRequests map[string]*client.MergeRequest // key: "projectID:mrIID"
	mrChanges     map[string]*client.MergeRequestChanges
	notes         map[string][]*client.Note // key: "projectID:mrIID"
	discussions   map[string][]*client.Discussion
	webhooks      map[string][]*client.Webhook // key: projectID
	files         map[string]*client.File      // key: "projectID:path:ref"

	// Current user
	currentUser *client.User

	// Tracking
	RequestLog      []RequestLogEntry
	WebhooksSent    []WebhookPayload
	NotesCreated    []*client.Note
	DiscussionsMade []*client.Discussion

	// Configuration
	LatencyMs      int
	FailureRate    float64 // 0.0 to 1.0
	RateLimitAfter int     // Requests before rate limiting
	requestCount   int
}

// RequestLogEntry records API requests for verification
type RequestLogEntry struct {
	Method    string
	Path      string
	Body      string
	Timestamp time.Time
}

// WebhookPayload represents a sent webhook
type WebhookPayload struct {
	URL       string
	EventType string
	Body      []byte
	Timestamp time.Time
}

// NewMockGitLabServer creates a new mock GitLab server
func NewMockGitLabServer() *MockGitLabServer {
	m := &MockGitLabServer{
		projects:      make(map[int64]*client.Project),
		mergeRequests: make(map[string]*client.MergeRequest),
		mrChanges:     make(map[string]*client.MergeRequestChanges),
		notes:         make(map[string][]*client.Note),
		discussions:   make(map[string][]*client.Discussion),
		webhooks:      make(map[string][]*client.Webhook),
		files:         make(map[string]*client.File),
		currentUser: &client.User{
			ID:       1,
			Username: "test-user",
			Name:     "Test User",
			Email:    "test@example.com",
			IsAdmin:  true,
		},
		RequestLog:      make([]RequestLogEntry, 0),
		WebhooksSent:    make([]WebhookPayload, 0),
		NotesCreated:    make([]*client.Note, 0),
		DiscussionsMade: make([]*client.Discussion, 0),
	}

	m.Server = httptest.NewServer(m)
	return m
}

// ServeHTTP handles all mock API requests
func (m *MockGitLabServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Log request
	m.mu.Lock()
	m.RequestLog = append(m.RequestLog, RequestLogEntry{
		Method:    r.Method,
		Path:      r.URL.Path,
		Timestamp: time.Now(),
	})
	m.requestCount++
	count := m.requestCount
	m.mu.Unlock()

	// Simulate latency
	if m.LatencyMs > 0 {
		time.Sleep(time.Duration(m.LatencyMs) * time.Millisecond)
	}

	// Check rate limit
	if m.RateLimitAfter > 0 && count > m.RateLimitAfter {
		w.Header().Set("RateLimit-Remaining", "0")
		w.Header().Set("RateLimit-Reset", strconv.FormatInt(time.Now().Add(60*time.Second).Unix(), 10))
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"message": "Rate limit exceeded"})
		return
	}

	// Route request
	path := r.URL.Path

	switch {
	case path == "/api/v4/user" && r.Method == "GET":
		m.handleGetCurrentUser(w, r)
	case strings.HasPrefix(path, "/api/v4/projects/") && strings.HasSuffix(path, "/merge_requests") && !strings.Contains(path, "/notes"):
		m.handleMergeRequests(w, r)
	case strings.Contains(path, "/merge_requests/") && strings.HasSuffix(path, "/changes"):
		m.handleMRChanges(w, r)
	case strings.Contains(path, "/merge_requests/") && strings.HasSuffix(path, "/notes"):
		m.handleMRNotes(w, r)
	case strings.Contains(path, "/merge_requests/") && strings.HasSuffix(path, "/discussions"):
		m.handleMRDiscussions(w, r)
	case strings.Contains(path, "/merge_requests/") && !strings.Contains(path, "/notes") && !strings.Contains(path, "/discussions") && !strings.HasSuffix(path, "/changes"):
		m.handleMR(w, r)
	case strings.Contains(path, "/hooks"):
		m.handleWebhooks(w, r)
	case strings.Contains(path, "/repository/files/"):
		m.handleFiles(w, r)
	case strings.HasPrefix(path, "/api/v4/projects/"):
		m.handleProject(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "404 Not Found"})
	}
}

func (m *MockGitLabServer) handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m.currentUser)
}

func (m *MockGitLabServer) handleProject(w http.ResponseWriter, r *http.Request) {
	// Extract project ID
	parts := strings.Split(r.URL.Path, "/")
	var projectID int64
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			id, err := strconv.ParseInt(parts[i+1], 10, 64)
			if err == nil {
				projectID = id
			}
			break
		}
	}

	m.mu.RLock()
	project, exists := m.projects[projectID]
	m.mu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "404 Project Not Found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

func (m *MockGitLabServer) handleMergeRequests(w http.ResponseWriter, r *http.Request) {
	// Extract project ID from path
	parts := strings.Split(r.URL.Path, "/")
	var projectID int64
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			id, _ := strconv.ParseInt(parts[i+1], 10, 64)
			projectID = id
			break
		}
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var mrs []*client.MergeRequest
	for key, mr := range m.mergeRequests {
		if strings.HasPrefix(key, fmt.Sprintf("%d:", projectID)) {
			mrs = append(mrs, mr)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mrs)
}

func (m *MockGitLabServer) handleMR(w http.ResponseWriter, r *http.Request) {
	// Extract project ID and MR IID
	parts := strings.Split(r.URL.Path, "/")
	var projectID int64
	var mrIID int
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			projectID, _ = strconv.ParseInt(parts[i+1], 10, 64)
		}
		if p == "merge_requests" && i+1 < len(parts) {
			mrIID, _ = strconv.Atoi(parts[i+1])
		}
	}

	key := fmt.Sprintf("%d:%d", projectID, mrIID)

	m.mu.RLock()
	mr, exists := m.mergeRequests[key]
	m.mu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "404 Merge Request Not Found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mr)
}

func (m *MockGitLabServer) handleMRChanges(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	var projectID int64
	var mrIID int
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			projectID, _ = strconv.ParseInt(parts[i+1], 10, 64)
		}
		if p == "merge_requests" && i+1 < len(parts) {
			mrIID, _ = strconv.Atoi(parts[i+1])
		}
	}

	key := fmt.Sprintf("%d:%d", projectID, mrIID)

	m.mu.RLock()
	changes, exists := m.mrChanges[key]
	m.mu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "404 Not Found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(changes)
}

func (m *MockGitLabServer) handleMRNotes(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	var projectID int64
	var mrIID int
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			projectID, _ = strconv.ParseInt(parts[i+1], 10, 64)
		}
		if p == "merge_requests" && i+1 < len(parts) {
			mrIID, _ = strconv.Atoi(parts[i+1])
		}
	}

	key := fmt.Sprintf("%d:%d", projectID, mrIID)

	if r.Method == "POST" {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)

		note := &client.Note{
			ID:        time.Now().UnixNano(),
			Body:      body["body"],
			Author:    m.currentUser,
			CreatedAt: time.Now(),
		}

		m.mu.Lock()
		m.notes[key] = append(m.notes[key], note)
		m.NotesCreated = append(m.NotesCreated, note)
		m.mu.Unlock()

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(note)
		return
	}

	m.mu.RLock()
	notes := m.notes[key]
	m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notes)
}

func (m *MockGitLabServer) handleMRDiscussions(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	var projectID int64
	var mrIID int
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			projectID, _ = strconv.ParseInt(parts[i+1], 10, 64)
		}
		if p == "merge_requests" && i+1 < len(parts) {
			mrIID, _ = strconv.Atoi(parts[i+1])
		}
	}

	key := fmt.Sprintf("%d:%d", projectID, mrIID)

	if r.Method == "POST" {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		discussion := &client.Discussion{
			ID: fmt.Sprintf("disc-%d", time.Now().UnixNano()),
			Notes: []client.Note{
				{
					ID:        time.Now().UnixNano(),
					Body:      body["body"].(string),
					Author:    m.currentUser,
					CreatedAt: time.Now(),
				},
			},
		}

		m.mu.Lock()
		m.discussions[key] = append(m.discussions[key], discussion)
		m.DiscussionsMade = append(m.DiscussionsMade, discussion)
		m.mu.Unlock()

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(discussion)
		return
	}

	m.mu.RLock()
	discussions := m.discussions[key]
	m.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(discussions)
}

func (m *MockGitLabServer) handleWebhooks(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	var projectID string
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			projectID = parts[i+1]
			break
		}
	}

	switch r.Method {
	case "GET":
		m.mu.RLock()
		hooks := m.webhooks[projectID]
		m.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(hooks)

	case "POST":
		var body client.CreateWebhookRequest
		json.NewDecoder(r.Body).Decode(&body)

		webhook := &client.Webhook{
			ID:                  time.Now().Unix(),
			URL:                 body.URL,
			MergeRequestsEvents: body.MergeRequestsEvents,
			CreatedAt:           time.Now(),
		}

		m.mu.Lock()
		m.webhooks[projectID] = append(m.webhooks[projectID], webhook)
		m.mu.Unlock()

		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(webhook)

	case "DELETE":
		// Extract webhook ID from path
		hookID, _ := strconv.ParseInt(parts[len(parts)-1], 10, 64)

		m.mu.Lock()
		hooks := m.webhooks[projectID]
		for i, h := range hooks {
			if h.ID == hookID {
				m.webhooks[projectID] = append(hooks[:i], hooks[i+1:]...)
				break
			}
		}
		m.mu.Unlock()

		w.WriteHeader(http.StatusNoContent)
	}
}

func (m *MockGitLabServer) handleFiles(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	var projectID string
	var filePath string
	for i, p := range parts {
		if p == "projects" && i+1 < len(parts) {
			projectID = parts[i+1]
		}
		if p == "files" && i+1 < len(parts) {
			filePath = strings.Join(parts[i+1:], "/")
			break
		}
	}

	ref := r.URL.Query().Get("ref")
	key := fmt.Sprintf("%s:%s:%s", projectID, filePath, ref)

	m.mu.RLock()
	file, exists := m.files[key]
	m.mu.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "404 File Not Found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(file)
}

// AddProject adds a project to the mock server
func (m *MockGitLabServer) AddProject(p *client.Project) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.projects[p.ID] = p
}

// AddMergeRequest adds a merge request to the mock server
func (m *MockGitLabServer) AddMergeRequest(projectID int64, mr *client.MergeRequest) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%d:%d", projectID, mr.IID)
	m.mergeRequests[key] = mr
}

// AddMRChanges adds changes for a merge request
func (m *MockGitLabServer) AddMRChanges(projectID int64, mrIID int, changes *client.MergeRequestChanges) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%d:%d", projectID, mrIID)
	m.mrChanges[key] = changes
}

// AddFile adds a file to the mock server
func (m *MockGitLabServer) AddFile(projectID int64, path, ref string, file *client.File) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%d:%s:%s", projectID, path, ref)
	m.files[key] = file
}

// SetCurrentUser sets the current user for the mock server
func (m *MockGitLabServer) SetCurrentUser(user *client.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentUser = user
}

// Reset clears all data and logs
func (m *MockGitLabServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RequestLog = make([]RequestLogEntry, 0)
	m.NotesCreated = make([]*client.Note, 0)
	m.DiscussionsMade = make([]*client.Discussion, 0)
	m.requestCount = 0
}

// GetRequestCount returns the number of requests made
func (m *MockGitLabServer) GetRequestCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requestCount
}

// GetNotesCreated returns all notes created
func (m *MockGitLabServer) GetNotesCreated() []*client.Note {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.NotesCreated
}
