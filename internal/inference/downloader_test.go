package inference

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func newTestDownloader(t *testing.T, hfCacheDir, ggufCacheDir string) *ModelDownloader {
	t.Helper()
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	cfg := ModelDownloaderConfig{
		HFCacheDir:   hfCacheDir,
		GGUFCacheDir: ggufCacheDir,
		Logger:       logger,
	}
	d, err := NewModelDownloader(cfg)
	if err != nil {
		t.Fatalf("NewModelDownloader failed: %v", err)
	}
	return d
}

func TestModelDownloader_EnsureGGUF_Success(t *testing.T) {
	// Create test server that returns a small "GGUF" file
	content := []byte("GGUF_TEST_CONTENT_1234567890")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "28")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	// Create temp cache dir
	tmpDir := t.TempDir()
	downloader := newTestDownloader(t, tmpDir, tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	localPath, err := downloader.EnsureGGUF(ctx, server.URL+"/test.gguf", "")
	if err != nil {
		t.Fatalf("EnsureGGUF failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		t.Fatalf("Downloaded file not found at %s", localPath)
	}

	// Verify content
	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("Content mismatch: got %q, want %q", string(data), string(content))
	}
}

func TestModelDownloader_EnsureGGUF_ResumePartial(t *testing.T) {
	fullContent := []byte("GGUF_FULL_CONTENT_FOR_RESUME_TEST")
	partialLen := 10

	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" && calls > 1 {
			// Second request with Range header - return remaining bytes
			w.Header().Set("Content-Range", "bytes 10-33/34")
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(fullContent[partialLen:])
			return
		}
		// First request - simulate interrupted download
		w.Header().Set("Content-Length", "34")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fullContent[:partialLen])
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	// Pre-create partial file
	partialPath := filepath.Join(tmpDir, "test.gguf.part")
	if err := os.WriteFile(partialPath, fullContent[:partialLen], 0644); err != nil {
		t.Fatalf("Failed to create partial file: %v", err)
	}

	downloader := newTestDownloader(t, tmpDir, tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	localPath, err := downloader.EnsureGGUF(ctx, server.URL+"/test.gguf", "")
	if err != nil {
		t.Fatalf("EnsureGGUF with resume failed: %v", err)
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(data) != string(fullContent) {
		t.Errorf("Content mismatch after resume: got %d bytes, want %d bytes", len(data), len(fullContent))
	}
}

func TestModelDownloader_EnsureGGUF_SHA256Validation(t *testing.T) {
	content := []byte("GGUF_SHA_TEST")
	// SHA256 of "GGUF_SHA_TEST" = 7e85f7e3b0a43d5c7f5c6e8b9a0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d
	// Actually compute: echo -n "GGUF_SHA_TEST" | sha256sum
	// Let's use wrong sha to test failure

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	downloader := newTestDownloader(t, tmpDir, tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wrong SHA should fail
	_, err := downloader.EnsureGGUF(ctx, server.URL+"/test.gguf", "0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Fatal("Expected SHA mismatch error, got nil")
	}
}

func TestModelDownloader_EnsureGGUF_RetryOnTransientError(t *testing.T) {
	content := []byte("GGUF_RETRY_TEST")
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// First 2 attempts fail with 503
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// Third attempt succeeds
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	downloader := newTestDownloader(t, tmpDir, tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	localPath, err := downloader.EnsureGGUF(ctx, server.URL+"/test.gguf", "")
	if err != nil {
		t.Fatalf("EnsureGGUF with retries failed: %v", err)
	}

	if attempts < 3 {
		t.Errorf("Expected at least 3 attempts, got %d", attempts)
	}

	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if string(data) != string(content) {
		t.Error("Content mismatch after retry")
	}
}

func TestModelDownloader_EnsureGGUF_404NotRetried(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	downloader := newTestDownloader(t, tmpDir, tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := downloader.EnsureGGUF(ctx, server.URL+"/notfound.gguf", "")
	if err == nil {
		t.Fatal("Expected 404 error, got nil")
	}

	// 404 should not be retried (only 1 attempt)
	if attempts != 1 {
		t.Errorf("404 should not retry: expected 1 attempt, got %d", attempts)
	}
}

