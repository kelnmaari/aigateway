package inference

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aigateway/internal/huggingface"
	"github.com/sirupsen/logrus"
)

// ModelDownloader manages artifact downloads (HF + generic HTTP) with caching and anti-duplicate behavior.
type ModelDownloader struct {
	hfClient     *huggingface.Client
	hfDownloader *huggingface.Downloader
	httpClient   *http.Client
	hfCacheDir   string
	ggufCacheDir string
	logger       *logrus.Logger
}

// ModelDownloaderConfig contains initialization parameters.
type ModelDownloaderConfig struct {
	HFToken       string
	HFCacheDir    string
	GGUFCacheDir  string
	MaxConcurrent int
	AutoResume    bool
	HTTPTimeout   time.Duration
	Logger        *logrus.Logger
}

// NewModelDownloader constructs a downloader with HF and generic HTTP support.
func NewModelDownloader(cfg ModelDownloaderConfig) (*ModelDownloader, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 2
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = 60 * time.Second
	}
	if cfg.HFCacheDir == "" {
		return nil, fmt.Errorf("HFCacheDir is required")
	}
	if cfg.GGUFCacheDir == "" {
		return nil, fmt.Errorf("GGUFCacheDir is required")
	}

	hfClient := huggingface.NewClient(cfg.HFToken, cfg.Logger)
	hfDownloader, err := huggingface.NewDownloader(hfClient, cfg.HFCacheDir, cfg.MaxConcurrent, cfg.AutoResume, cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("init hf downloader: %w", err)
	}

	httpClient := &http.Client{
		Timeout: cfg.HTTPTimeout,
	}

	return &ModelDownloader{
		hfClient:     hfClient,
		hfDownloader: hfDownloader,
		httpClient:   httpClient,
		hfCacheDir:   cfg.HFCacheDir,
		ggufCacheDir: cfg.GGUFCacheDir,
		logger:       cfg.Logger,
	}, nil
}

// EnsureHFFile downloads (or reuses cached) HF artifact and returns local path.
func (d *ModelDownloader) EnsureHFFile(ctx context.Context, repo, filename, sha string) (string, error) {
	url := d.hfClient.GetFileURL(repo, filename)
	size, err := d.contentLength(ctx, url)
	if err != nil {
		return "", fmt.Errorf("head hf file: %w", err)
	}

	download, err := d.hfDownloader.StartDownload(repo, filename, size, sha)
	if err != nil {
		return "", fmt.Errorf("start hf download: %w", err)
	}

	if err := d.waitForDownload(ctx, download); err != nil {
		return "", err
	}

	return download.DestPath, nil
}

// EnsureGGUF downloads a GGUF file (generic HTTP or HF mirror) into the GGUF cache and returns path.
// Implements retry with exponential backoff on transient errors.
func (d *ModelDownloader) EnsureGGUF(ctx context.Context, url, expectedSHA string) (string, error) {
	filename := filepath.Base(url)
	destPath := filepath.Join(d.ggufCacheDir, filename)
	if err := os.MkdirAll(d.ggufCacheDir, 0755); err != nil {
		return "", fmt.Errorf("prepare gguf cache: %w", err)
	}

	// Skip if already present and hash matches (if provided).
	if fileOK(destPath, expectedSHA) {
		return destPath, nil
	}

	var lastErr error
	for attempt := range maxDownloadRetries {
		if attempt > 0 {
			backoff := retryBackoff(attempt)
			d.logger.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"backoff": backoff,
				"url":     url,
			}).Info("retrying download after backoff")
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(backoff):
			}
		}

		size, err := d.contentLength(ctx, url)
		if err != nil {
			lastErr = fmt.Errorf("head gguf: %w", err)
			if !isTransientError(err) {
				return "", lastErr
			}
			continue
		}

		if err := d.downloadWithResume(ctx, url, destPath, size, expectedSHA); err != nil {
			lastErr = fmt.Errorf("download gguf: %w", err)
			if !isTransientError(err) {
				return "", lastErr
			}
			continue
		}
		return destPath, nil
	}
	return "", fmt.Errorf("download failed after %d retries: %w", maxDownloadRetries, lastErr)
}

const maxDownloadRetries = 3

// retryBackoff returns exponential backoff duration for retry attempt (0-indexed).
func retryBackoff(attempt int) time.Duration {
	// 1s, 2s, 4s, 8s, ...
	base := time.Second
	for range attempt {
		base *= 2
	}
	if base > 30*time.Second {
		base = 30 * time.Second
	}
	return base
}

// isTransientError checks if error is likely transient (network, timeout).
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return false // context errors are not retryable
	}
	// Network errors, timeouts, 5xx are transient
	errStr := strings.ToLower(err.Error())
	transientPatterns := []string{"timeout", "connection refused", "connection reset", "eof", "temporary", "503", "502", "504"}
	for _, p := range transientPatterns {
		if strings.Contains(errStr, p) {
			return true
		}
	}
	return false
}

// Shutdown gracefully stops background workers.
func (d *ModelDownloader) Shutdown() {
	if d.hfDownloader != nil {
		d.hfDownloader.Shutdown()
	}
}

// DownloadRepository downloads all files for a HuggingFace model repository.
// Returns RepoDownload for tracking progress.
func (d *ModelDownloader) DownloadRepository(ctx context.Context, modelID string) (*huggingface.RepoDownload, error) {
	if d.hfDownloader == nil {
		return nil, fmt.Errorf("HuggingFace downloader not configured")
	}
	return d.hfDownloader.DownloadRepository(ctx, modelID)
}

// DownloadSingleFile downloads a specific file from a HuggingFace repository.
func (d *ModelDownloader) DownloadSingleFile(ctx context.Context, modelID, filename string) (*huggingface.RepoDownload, error) {
	if d.hfDownloader == nil {
		return nil, fmt.Errorf("HuggingFace downloader not configured")
	}
	return d.hfDownloader.DownloadSingleFile(ctx, modelID, filename)
}

// GetRepoDownload returns status of a repository download.
func (d *ModelDownloader) GetRepoDownload(modelID string) (*huggingface.RepoDownload, bool) {
	if d.hfDownloader == nil {
		return nil, false
	}
	return d.hfDownloader.GetRepoDownload(modelID)
}

// ListRepoDownloads returns all repository downloads.
func (d *ModelDownloader) ListRepoDownloads() []*huggingface.RepoDownload {
	if d.hfDownloader == nil {
		return nil
	}
	return d.hfDownloader.ListRepoDownloads()
}

// CancelRepoDownload cancels a repository download.
func (d *ModelDownloader) CancelRepoDownload(modelID string) error {
	if d.hfDownloader == nil {
		return fmt.Errorf("downloader not configured")
	}
	return d.hfDownloader.CancelRepoDownload(modelID)
}

// RemoveRepoDownload removes a repository download from the list.
func (d *ModelDownloader) RemoveRepoDownload(modelID string) {
	if d.hfDownloader == nil {
		return
	}
	d.hfDownloader.RemoveRepoDownload(modelID)
}

// waitForDownload blocks until download completes or fails.
func (d *ModelDownloader) waitForDownload(ctx context.Context, download *huggingface.Download) error {
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			download.Mu.RLock()
			status := download.Status
			errMsg := download.Error
			download.Mu.RUnlock()

			switch status {
			case huggingface.DownloadStatusCompleted:
				return nil
			case huggingface.DownloadStatusFailed, huggingface.DownloadStatusCancelled:
				if errMsg == "" {
					errMsg = "download failed"
				}
				return errors.New(errMsg)
			}
		}
	}
}

// contentLength fetches Content-Length via HEAD.
func (d *ModelDownloader) contentLength(ctx context.Context, url string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	if resp.ContentLength <= 0 {
		return 0, fmt.Errorf("missing content length")
	}
	return resp.ContentLength, nil
}

// downloadWithResume handles generic HTTP download with resume support.
func (d *ModelDownloader) downloadWithResume(ctx context.Context, url, destPath string, totalSize int64, expectedSHA string) error {
	partPath := destPath + ".part"
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	var downloaded int64
	if info, err := os.Stat(partPath); err == nil {
		downloaded = info.Size()
		if downloaded == totalSize && totalSize > 0 {
			if err := os.Rename(partPath, destPath); err == nil {
				if fileOK(destPath, expectedSHA) {
					return nil
				}
			}
			downloaded = 0
		}
	}

	file, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open part: %w", err)
	}
	defer file.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if downloaded > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", downloaded))
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	hasher := sha256.New()
	if downloaded > 0 {
		if err := hashExisting(partPath, hasher); err != nil {
			return fmt.Errorf("hash resume: %w", err)
		}
	}

	buf := make([]byte, 64*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := file.Write(buf[:n]); err != nil {
				return fmt.Errorf("write: %w", err)
			}
			hasher.Write(buf[:n])
			downloaded += int64(n)
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return fmt.Errorf("read: %w", readErr)
		}
	}

	if expectedSHA != "" {
		computed := hex.EncodeToString(hasher.Sum(nil))
		if computed != expectedSHA {
			return fmt.Errorf("sha mismatch: expected %s got %s", expectedSHA, computed)
		}
	}

	// Close file before rename (required on Windows)
	if err := file.Close(); err != nil {
		return fmt.Errorf("close part: %w", err)
	}

	if err := os.Rename(partPath, destPath); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

func hashExisting(path string, hasher io.Writer) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(hasher, f)
	return err
}

func fileOK(path, expectedSHA string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if expectedSHA == "" {
		return true
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return false
	}
	return hex.EncodeToString(hash.Sum(nil)) == expectedSHA
}
