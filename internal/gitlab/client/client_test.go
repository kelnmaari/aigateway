package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient tests client creation
func TestNewClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		client := NewClient(ClientConfig{
			BaseURL:     "https://gitlab.example.com",
			AccessToken: "test-token",
		})
		
		require.NotNil(t, client)
		assert.Equal(t, "https://gitlab.example.com", client.baseURL)
		assert.Equal(t, "test-token", client.accessToken)
	})
	
	t.Run("removes trailing slash", func(t *testing.T) {
		client := NewClient(ClientConfig{
			BaseURL:     "https://gitlab.example.com/",
			AccessToken: "test-token",
		})
		
		assert.Equal(t, "https://gitlab.example.com", client.baseURL)
	})
	
	t.Run("with custom timeout", func(t *testing.T) {
		client := NewClient(ClientConfig{
			BaseURL:     "https://gitlab.example.com",
			AccessToken: "test-token",
			Timeout:     60 * time.Second,
		})
		
		assert.Equal(t, 60*time.Second, client.httpClient.Timeout)
	})
	
	t.Run("default values", func(t *testing.T) {
		client := NewClient(ClientConfig{
			BaseURL:     "https://gitlab.example.com",
			AccessToken: "test-token",
		})
		
		// Default timeout is 30s
		assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
		// Default API version is v4
		assert.Equal(t, "v4", client.apiVersion)
	})
}

// TestGetMergeRequest tests fetching MR details
func TestGetMergeRequest(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/api/v4/projects/123/merge_requests/456", r.URL.Path)
			assert.Equal(t, "test-token", r.Header.Get("PRIVATE-TOKEN"))
			
			author := User{
				ID:       1,
				Username: "testuser",
				Name:     "Test User",
			}
			mr := MergeRequest{
				ID:           789,
				IID:          456,
				Title:        "Test MR",
				Description:  "Test description",
				State:        "opened",
				SourceBranch: "feature/test",
				TargetBranch: "main",
				Author:       &author,
				WebURL:       "https://gitlab.example.com/project/-/merge_requests/456",
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mr)
		}))
		defer server.Close()
		
		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			AccessToken: "test-token",
		})
		
		mr, err := client.GetMergeRequest(context.Background(), 123, 456)
		
		require.NoError(t, err)
		assert.Equal(t, 456, mr.IID)
		assert.Equal(t, "Test MR", mr.Title)
		assert.Equal(t, "feature/test", mr.SourceBranch)
		assert.Equal(t, "main", mr.TargetBranch)
		require.NotNil(t, mr.Author)
		assert.Equal(t, "testuser", mr.Author.Username)
	})
	
	t.Run("not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "404 Merge Request Not Found"}`))
		}))
		defer server.Close()
		
		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			AccessToken: "test-token",
		})
		
		_, err := client.GetMergeRequest(context.Background(), 123, 999)
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})
	
	t.Run("context cancelled", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()
		
		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			AccessToken: "test-token",
		})
		
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		
		_, err := client.GetMergeRequest(ctx, 123, 456)
		
		require.Error(t, err)
	})
}

// TestGetMergeRequestChanges tests fetching MR changes/diff
func TestGetMergeRequestChanges(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/api/v4/projects/123/merge_requests/456/changes", r.URL.Path)
			
			changes := MergeRequestChanges{
				Changes: []Change{
					{
						OldPath:     "file1.go",
						NewPath:     "file1.go",
						Diff:        "@@ -1,3 +1,5 @@\n package main\n+\n+// Added comment",
						NewFile:     false,
						RenamedFile: false,
						DeletedFile: false,
					},
					{
						OldPath:     "",
						NewPath:     "file2.go",
						Diff:        "@@ -0,0 +1,10 @@\n+package main\n+func New() {}",
						NewFile:     true,
						RenamedFile: false,
						DeletedFile: false,
					},
				},
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(changes)
		}))
		defer server.Close()
		
		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			AccessToken: "test-token",
		})
		
		changes, err := client.GetMergeRequestChanges(context.Background(), 123, 456)
		
		require.NoError(t, err)
		require.Len(t, changes.Changes, 2)
		assert.Equal(t, "file1.go", changes.Changes[0].NewPath)
		assert.False(t, changes.Changes[0].NewFile)
		assert.Equal(t, "file2.go", changes.Changes[1].NewPath)
		assert.True(t, changes.Changes[1].NewFile)
	})
}

// TestCreateMRNote tests creating comments on MR
func TestCreateMRNote(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/api/v4/projects/123/merge_requests/456/notes", r.URL.Path)
			
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "Test comment", body["body"])
			
			author := User{
				ID:       2,
				Username: "bot",
				Name:     "Bot User",
			}
			note := Note{
				ID:     999,
				Body:   "Test comment",
				Author: &author,
			}
			
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(note)
		}))
		defer server.Close()
		
		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			AccessToken: "test-token",
		})
		
		note, err := client.CreateMRNote(context.Background(), 123, 456, "Test comment")
		
		require.NoError(t, err)
		assert.Equal(t, int64(999), note.ID)
		assert.Equal(t, "Test comment", note.Body)
	})
}

// TestGetCurrentUser tests user info retrieval
func TestGetCurrentUser(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/api/v4/user", r.URL.Path)
			
			user := User{
				ID:       1,
				Username: "admin",
				Name:     "Admin User",
				Email:    "admin@example.com",
				IsAdmin:  true,
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(user)
		}))
		defer server.Close()
		
		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			AccessToken: "test-token",
		})
		
		user, err := client.GetCurrentUser(context.Background())
		
		require.NoError(t, err)
		assert.Equal(t, "admin", user.Username)
		assert.True(t, user.IsAdmin)
	})
	
	t.Run("unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message": "401 Unauthorized"}`))
		}))
		defer server.Close()
		
		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			AccessToken: "invalid-token",
		})
		
		_, err := client.GetCurrentUser(context.Background())
		
		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})
}

// TestGetProject tests project info retrieval
func TestGetProject(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Equal(t, "/api/v4/projects/123", r.URL.Path)
			
			project := Project{
				ID:                123,
				Name:              "Test Project",
				PathWithNamespace: "group/test-project",
				DefaultBranch:     "main",
				WebURL:            "https://gitlab.example.com/group/test-project",
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(project)
		}))
		defer server.Close()
		
		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			AccessToken: "test-token",
		})
		
		project, err := client.GetProject(context.Background(), 123)
		
		require.NoError(t, err)
		assert.Equal(t, int64(123), project.ID)
		assert.Equal(t, "Test Project", project.Name)
		assert.Equal(t, "group/test-project", project.PathWithNamespace)
	})
}

// TestRateLimiter tests rate limiting behavior
func TestRateLimiter(t *testing.T) {
	t.Run("respects rate limit", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(User{ID: 1, Username: "test"})
		}))
		defer server.Close()
		
		client := NewClient(ClientConfig{
			BaseURL:         server.URL,
			AccessToken:     "test-token",
			RateLimitPerSec: 100, // High rate for fast test
		})
		
		start := time.Now()
		
		// Make 5 requests
		for i := 0; i < 5; i++ {
			_, err := client.GetCurrentUser(context.Background())
			require.NoError(t, err)
		}
		
		elapsed := time.Since(start)
		
		// Should complete quickly with high rate limit
		assert.Less(t, elapsed, 2*time.Second)
		assert.Equal(t, 5, requestCount)
	})
}

// TestWebhookOperations tests webhook CRUD
func TestWebhookOperations(t *testing.T) {
	t.Run("create webhook", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "/api/v4/projects/123/hooks", r.URL.Path)
			
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			assert.Equal(t, "https://example.com/webhook", body["url"])
			assert.Equal(t, true, body["merge_requests_events"])
			
			webhook := Webhook{
				ID:                  1,
				URL:                 "https://example.com/webhook",
				MergeRequestsEvents: true,
			}
			
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(webhook)
		}))
		defer server.Close()
		
		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			AccessToken: "test-token",
		})
		
		webhook, err := client.CreateWebhook(context.Background(), 123, &CreateWebhookRequest{
			URL:                 "https://example.com/webhook",
			Token:               "secret123",
			MergeRequestsEvents: true,
		})
		
		require.NoError(t, err)
		assert.Equal(t, int64(1), webhook.ID)
		assert.True(t, webhook.MergeRequestsEvents)
	})
	
	t.Run("delete webhook", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "DELETE", r.Method)
			assert.Equal(t, "/api/v4/projects/123/hooks/456", r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()
		
		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			AccessToken: "test-token",
		})
		
		err := client.DeleteWebhook(context.Background(), 123, 456)
		
		require.NoError(t, err)
	})
}

// TestGetFile tests file content retrieval
func TestGetFile(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "GET", r.Method)
			assert.Contains(t, r.URL.Path, "/api/v4/projects/123/repository/files/")
			
			file := File{
				FileName:      "main.go",
				FilePath:      "cmd/main.go",
				Content:       "cGFja2FnZSBtYWluCg==", // base64 for "package main\n"
				ContentSha256: "abc123",
				Encoding:      "base64",
			}
			
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(file)
		}))
		defer server.Close()
		
		client := NewClient(ClientConfig{
			BaseURL:     server.URL,
			AccessToken: "test-token",
		})
		
		content, err := client.GetFile(context.Background(), 123, "cmd/main.go", "main")
		
		require.NoError(t, err)
		assert.Equal(t, "main.go", content.FileName)
		assert.Equal(t, "cmd/main.go", content.FilePath)
	})
}

// TestAPIURL tests URL construction
func TestAPIURL(t *testing.T) {
	client := NewClient(ClientConfig{
		BaseURL:     "https://gitlab.example.com",
		AccessToken: "test-token",
		APIVersion:  "v4",
	})
	
	url := client.apiURL("/projects/123/merge_requests/456")
	assert.Equal(t, "https://gitlab.example.com/api/v4/projects/123/merge_requests/456", url)
}
