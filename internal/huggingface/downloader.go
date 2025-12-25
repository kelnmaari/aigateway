// Package huggingface provides model downloading with resume support
// Version: v3.0.8 - Added retry mechanism (avast/retry-go)
package huggingface

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	retry "github.com/avast/retry-go/v4"
	"github.com/sirupsen/logrus"
)

// DownloadStatus represents download state
type DownloadStatus string

const (
	DownloadStatusPending     DownloadStatus = "pending"
	DownloadStatusDownloading DownloadStatus = "downloading"
	DownloadStatusCompleted   DownloadStatus = "completed"
	DownloadStatusFailed      DownloadStatus = "failed"
	DownloadStatusPaused      DownloadStatus = "paused"
	DownloadStatusCancelled   DownloadStatus = "cancelled"
)

// Download represents a model file download
type Download struct {
	ID             string         `json:"id"`
	ModelID        string         `json:"model_id"`
	Filename       string         `json:"filename"`
	URL            string         `json:"url"`
	DestPath       string         `json:"dest_path"`
	TotalSize      int64          `json:"total_size"`
	DownloadedSize int64          `json:"downloaded_size"`
	Status         DownloadStatus `json:"status"`
	Error          string         `json:"error,omitempty"`
	Progress       float64        `json:"progress"` // 0.0 to 100.0
	Speed          int64          `json:"speed"`    // bytes per second
	ETA            time.Duration  `json:"eta"`      // estimated time remaining
	StartedAt      *time.Time     `json:"started_at,omitempty"`
	CompletedAt    *time.Time     `json:"completed_at,omitempty"`
	SHA256         string         `json:"sha256,omitempty"` // Expected SHA256
	VerifiedSHA256 string         `json:"verified_sha256,omitempty"`

	// Internal
	ctx        context.Context
	cancel     context.CancelFunc
	lastUpdate time.Time
	lastSize   int64
	Mu         sync.RWMutex // Exported for external locking
}

// Downloader manages model downloads
type Downloader struct {
	client        *Client
	downloadsDir  string
	maxConcurrent int
	autoResume    bool
	logger        *logrus.Logger

	downloads       map[string]*Download
	downloadQueue   chan *Download
	activeDownloads int
	mu              sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewDownloader creates a new model downloader
func NewDownloader(client *Client, downloadsDir string, maxConcurrent int, autoResume bool, logger *logrus.Logger) (*Downloader, error) {
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create downloads directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	d := &Downloader{
		client:        client,
		downloadsDir:  downloadsDir,
		maxConcurrent: maxConcurrent,
		autoResume:    autoResume,
		logger:        logger,
		downloads:     make(map[string]*Download),
		downloadQueue: make(chan *Download, 100),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start download workers
	for i := 0; i < maxConcurrent; i++ {
		d.wg.Add(1)
		go d.downloadWorker(i)
	}

	// Log absolute path for debugging
	absPath, _ := filepath.Abs(downloadsDir)
	logger.WithFields(logrus.Fields{
		"downloads_dir":     downloadsDir,
		"downloads_dir_abs": absPath,
		"max_concurrent":    maxConcurrent,
		"auto_resume":       autoResume,
	}).Info("Download manager initialized")

	return d, nil
}

// StartDownload initiates a new download
func (d *Downloader) StartDownload(modelID, filename string, totalSize int64, sha256 string) (*Download, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Generate download ID
	downloadID := generateDownloadID(modelID, filename)

	// Check if already downloading
	if existing, exists := d.downloads[downloadID]; exists {
		if existing.Status == DownloadStatusDownloading || existing.Status == DownloadStatusPending {
			return existing, nil
		}
	}

	// Prepare destination path
	destPath := filepath.Join(d.downloadsDir, modelID, filename)
	absDestPath, _ := filepath.Abs(destPath)
	d.logger.WithFields(logrus.Fields{
		"download_id":   downloadID,
		"dest_path":     destPath,
		"dest_path_abs": absDestPath,
	}).Info("📁 Preparing download destination")
	
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}
	d.logger.WithField("dir", filepath.Dir(absDestPath)).Info("📂 Directory created/verified")

	// Check if file already exists and is complete
	if info, err := os.Stat(destPath); err == nil {
		existingSize := info.Size()
		if existingSize >= totalSize {
			d.logger.WithFields(logrus.Fields{
				"download_id":   downloadID,
				"file_size":     existingSize,
				"expected_size": totalSize,
			}).Info("✅ File already exists and is complete, skipping download")
			
			now := time.Now()
			return &Download{
				ID:             downloadID,
				ModelID:        modelID,
				Filename:       filename,
				DestPath:       destPath,
				TotalSize:      totalSize,
				DownloadedSize: totalSize,
				Status:         DownloadStatusCompleted,
				Progress:       100.0,
				StartedAt:      &now,
				CompletedAt:    &now,
				Mu:             sync.RWMutex{},
			}, nil
		}
		d.logger.WithFields(logrus.Fields{
			"download_id":   downloadID,
			"file_size":     existingSize,
			"expected_size": totalSize,
		}).Info("⚠️ File exists but incomplete, will re-download")
	}

	// Check for partial download
	var downloadedSize int64
	partPath := destPath + ".part"
	if info, err := os.Stat(partPath); err == nil {
		downloadedSize = info.Size()

		// Check if already fully downloaded
		if downloadedSize >= totalSize {
			d.logger.WithFields(logrus.Fields{
				"download_id":     downloadID,
				"downloaded_size": downloadedSize,
				"total_size":      totalSize,
			}).Info("File already fully downloaded, finalizing...")

			// Rename .part to final file
			if err := os.Rename(partPath, destPath); err != nil {
				return nil, fmt.Errorf("failed to finalize download: %w", err)
			}

			// Return completed download
			now := time.Now()
			return &Download{
				ID:             downloadID,
				ModelID:        modelID,
				Filename:       filename,
				DestPath:       destPath,
				TotalSize:      totalSize,
				DownloadedSize: totalSize,
				Status:         DownloadStatusCompleted,
				Progress:       100.0,
				StartedAt:      &now,
				CompletedAt:    &now,
				Mu:             sync.RWMutex{},
			}, nil
		}

		d.logger.WithFields(logrus.Fields{
			"download_id":     downloadID,
			"downloaded_size": downloadedSize,
			"total_size":      totalSize,
		}).Info("Resuming partial download")
	}

	// Create download context
	ctx, cancel := context.WithCancel(d.ctx)

	download := &Download{
		ID:             downloadID,
		ModelID:        modelID,
		Filename:       filename,
		URL:            d.client.GetFileURL(modelID, filename),
		DestPath:       destPath,
		TotalSize:      totalSize,
		DownloadedSize: downloadedSize,
		Status:         DownloadStatusPending,
		SHA256:         sha256,
		ctx:            ctx,
		cancel:         cancel,
		lastUpdate:     time.Now(),
		lastSize:       downloadedSize,
		Mu:             sync.RWMutex{},
	}

	d.downloads[downloadID] = download

	// Add to queue
	select {
	case d.downloadQueue <- download:
		d.logger.WithField("download_id", downloadID).Info("Download queued")
	default:
		return nil, fmt.Errorf("download queue is full")
	}

	return download, nil
}

// downloadWorker processes downloads from queue
func (d *Downloader) downloadWorker(workerID int) {
	defer d.wg.Done()

	logger := d.logger.WithField("worker_id", workerID)
	logger.Debug("Download worker started")

	for {
		select {
		case <-d.ctx.Done():
			logger.Debug("Download worker stopping")
			return

		case download := <-d.downloadQueue:
			d.mu.Lock()
			d.activeDownloads++
			d.mu.Unlock()

			logger.WithFields(logrus.Fields{
				"download_id": download.ID,
				"model_id":    download.ModelID,
				"filename":    download.Filename,
			}).Info("Starting download")

			if err := d.performDownload(download); err != nil {
				download.Mu.Lock()
				download.Status = DownloadStatusFailed
				download.Error = err.Error()
				download.Mu.Unlock()

				logger.WithError(err).Error("Download failed")
			}

			d.mu.Lock()
			d.activeDownloads--
			d.mu.Unlock()
		}
	}
}

// performDownload executes the actual download
func (d *Downloader) performDownload(download *Download) error {
	download.Mu.Lock()
	download.Status = DownloadStatusDownloading
	now := time.Now()
	download.StartedAt = &now
	download.Mu.Unlock()

	// Open partial file
	partPath := download.DestPath + ".part"
	file, err := os.OpenFile(partPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	// Note: file will be closed manually before rename (not using defer)

	// Create HTTP request with Range header for resume
	req, err := http.NewRequestWithContext(download.ctx, "GET", download.URL, nil)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to create request: %w", err)
	}

	if download.DownloadedSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", download.DownloadedSize))
		d.logger.WithFields(logrus.Fields{
			"download_id":     download.ID,
			"downloaded_size": download.DownloadedSize,
			"total_size":      download.TotalSize,
		}).Debug("Resuming download from byte position")
	}

	// Execute request using download client (no timeout - context controls it)
	d.logger.WithFields(logrus.Fields{
		"download_id": download.ID,
		"url":         download.URL,
		"timeout":     "none (context-controlled)",
	}).Debug("Starting HTTP request for file download")

	resp, err := d.client.downloadClient.Do(req)
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to execute request: %w", err)
	}
	// Note: resp.Body will be closed manually before rename (not using defer)

	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		file.Close()
		resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Download with progress tracking
	buf := make([]byte, 32*1024) // 32KB buffer
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	hash := sha256.New()

	for {
		select {
		case <-download.ctx.Done():
			// Cleanup on cancellation
			file.Close()
			resp.Body.Close()
			return download.ctx.Err()

		case <-ticker.C:
			d.updateProgress(download)

		default:
			n, err := resp.Body.Read(buf)
			if n > 0 {
				// Write to file
				if _, writeErr := file.Write(buf[:n]); writeErr != nil {
					file.Close()
					resp.Body.Close()
					return fmt.Errorf("failed to write to file: %w", writeErr)
				}

				// Update hash
				hash.Write(buf[:n])

				// Update progress
				download.Mu.Lock()
				download.DownloadedSize += int64(n)
				download.Mu.Unlock()
			}

			if err != nil {
				if err == io.EOF {
					// Download completed
					d.updateProgress(download)

					// Verify SHA256 if provided
					if download.SHA256 != "" {
						computedHash := hex.EncodeToString(hash.Sum(nil))
						download.Mu.Lock()
						download.VerifiedSHA256 = computedHash
						download.Mu.Unlock()

						if computedHash != download.SHA256 {
							file.Close()
							resp.Body.Close()
							return fmt.Errorf("SHA256 mismatch: expected %s, got %s", download.SHA256, computedHash)
						}
					}

					// CRITICAL: Close file handles before rename (Windows requirement)
					d.logger.WithFields(logrus.Fields{
						"download_id": download.ID,
						"part_path":   partPath,
						"dest_path":   download.DestPath,
					}).Info("🔒 Closing file handles before rename...")

					if err := file.Close(); err != nil {
						d.logger.WithError(err).Warn("⚠️ Error closing file handle, continuing anyway")
					} else {
						d.logger.Debug("✅ File handle closed successfully")
					}

					if err := resp.Body.Close(); err != nil {
						d.logger.WithError(err).Warn("⚠️ Error closing response body, continuing anyway")
					} else {
						d.logger.Debug("✅ Response body closed successfully")
					}

					// Windows-specific: Force garbage collection and wait for file handles to be released
					d.logger.Info("🔄 Forcing GC and waiting for file handles to release...")
					runtime.GC()
					time.Sleep(1 * time.Second) // Increased from 100ms to 1 second
					runtime.GC()                // Second GC pass

					d.logger.WithFields(logrus.Fields{
						"download_id": download.ID,
						"max_retries": 10,
					}).Info("🔄 Attempting file rename with retry mechanism...")

					// Retry rename with exponential backoff using retry-go (v3.0.8)
					renameErr := retry.Do(
						func() error {
							return os.Rename(partPath, download.DestPath)
						},
						retry.Attempts(10),
						retry.Delay(200*time.Millisecond),
						retry.MaxDelay(5*time.Second),
						retry.DelayType(retry.BackOffDelay),
						retry.OnRetry(func(n uint, err error) {
							// Force GC between retries (Windows file handle issue)
							runtime.GC()
							d.logger.WithFields(logrus.Fields{
								"attempt": n + 1,
								"error":   err.Error(),
							}).Warn("⚠️ File rename failed, retrying after backoff")
						}),
						retry.LastErrorOnly(true),
					)

					if renameErr != nil {
						return fmt.Errorf("failed to rename file after 10 retries: %w", renameErr)
					}

					// Verify file exists after rename
					absPath, _ := filepath.Abs(download.DestPath)
					if info, err := os.Stat(download.DestPath); err != nil {
						d.logger.WithFields(logrus.Fields{
							"dest_path":     download.DestPath,
							"dest_path_abs": absPath,
							"error":         err.Error(),
						}).Error("❌ File NOT found after rename!")
					} else {
						d.logger.WithFields(logrus.Fields{
							"dest_path":     download.DestPath,
							"dest_path_abs": absPath,
							"size":          info.Size(),
						}).Info("✅ File verified after rename")
					}

					download.Mu.Lock()
					download.Status = DownloadStatusCompleted
					now := time.Now()
					download.CompletedAt = &now
					download.Progress = 100.0
					totalTime := now.Sub(*download.StartedAt)
					avgSpeed := float64(download.TotalSize) / totalTime.Seconds()
					download.Mu.Unlock()

					d.logger.WithFields(logrus.Fields{
						"download_id":    download.ID,
						"model_id":       download.ModelID,
						"filename":       download.Filename,
						"total_size":     download.TotalSize,
						"total_time":     totalTime.String(),
						"avg_speed_mb_s": fmt.Sprintf("%.2f", avgSpeed/1024/1024),
						"final_path":     download.DestPath,
					}).Info("✅ Download completed successfully")

					d.logger.WithFields(logrus.Fields{
						"download_id": download.ID,
						"model_id":    download.ModelID,
						"filename":    download.Filename,
						"size":        download.TotalSize,
						"duration":    time.Since(*download.StartedAt),
					}).Info("Download completed successfully")

					return nil
				}

				// Cleanup on read error
				file.Close()
				resp.Body.Close()
				return fmt.Errorf("read error: %w", err)
			}
		}
	}
}

// updateProgress calculates download progress and speed
func (d *Downloader) updateProgress(download *Download) {
	download.Mu.Lock()
	defer download.Mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(download.lastUpdate).Seconds()

	if elapsed > 0 {
		// Calculate speed (bytes per second)
		bytesDownloaded := download.DownloadedSize - download.lastSize
		download.Speed = int64(float64(bytesDownloaded) / elapsed)

		// Calculate progress percentage
		if download.TotalSize > 0 {
			download.Progress = float64(download.DownloadedSize) / float64(download.TotalSize) * 100.0

			// Calculate ETA
			if download.Speed > 0 {
				remainingBytes := download.TotalSize - download.DownloadedSize
				download.ETA = time.Duration(float64(remainingBytes)/float64(download.Speed)) * time.Second
			}
		}

		download.lastUpdate = now
		download.lastSize = download.DownloadedSize
	}
}

// GetDownload retrieves download info
func (d *Downloader) GetDownload(downloadID string) (*Download, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	download, exists := d.downloads[downloadID]
	return download, exists
}

// ListDownloads returns all downloads
func (d *Downloader) ListDownloads() []*Download {
	d.mu.RLock()
	defer d.mu.RUnlock()

	downloads := make([]*Download, 0, len(d.downloads))
	for _, download := range d.downloads {
		downloads = append(downloads, download)
	}

	return downloads
}

// PauseDownload pauses an active download
func (d *Downloader) PauseDownload(downloadID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	download, exists := d.downloads[downloadID]
	if !exists {
		return fmt.Errorf("download not found: %s", downloadID)
	}

	if download.Status != DownloadStatusDownloading {
		return fmt.Errorf("download is not active")
	}

	download.cancel()
	download.Mu.Lock()
	download.Status = DownloadStatusPaused
	download.Mu.Unlock()

	return nil
}

// CancelDownload cancels and removes a download
func (d *Downloader) CancelDownload(downloadID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	download, exists := d.downloads[downloadID]
	if !exists {
		return fmt.Errorf("download not found: %s", downloadID)
	}

	download.cancel()
	download.Mu.Lock()
	download.Status = DownloadStatusCancelled
	download.Mu.Unlock()

	// Remove partial file
	os.Remove(download.DestPath + ".part")

	delete(d.downloads, downloadID)

	return nil
}

// ClearCompleted removes all completed, failed, and cancelled downloads from the list
func (d *Downloader) ClearCompleted() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	cleared := 0
	for id, download := range d.downloads {
		download.Mu.RLock()
		status := download.Status
		download.Mu.RUnlock()

		if status == DownloadStatusCompleted || status == DownloadStatusFailed || status == DownloadStatusCancelled {
			delete(d.downloads, id)
			cleared++
		}
	}

	d.logger.WithField("cleared", cleared).Info("Cleared completed downloads")
	return cleared
}

// Shutdown gracefully stops the downloader
func (d *Downloader) Shutdown() {
	d.logger.Info("Shutting down download manager")
	d.cancel()
	d.wg.Wait()
	close(d.downloadQueue)
	d.logger.Info("Download manager stopped")
}

// generateDownloadID creates unique download identifier
func generateDownloadID(modelID, filename string) string {
	hash := sha256.Sum256([]byte(modelID + "/" + filename))
	return hex.EncodeToString(hash[:8])
}

// RepoDownload represents a full repository download (multiple files)
type RepoDownload struct {
	ID            string         `json:"id"`
	ModelID       string         `json:"model_id"`
	Status        DownloadStatus `json:"status"`
	TotalFiles    int            `json:"total_files"`
	CompletedFiles int           `json:"completed_files"`
	FailedFiles   int            `json:"failed_files"`
	TotalSize     int64          `json:"total_size"`
	DownloadedSize int64         `json:"downloaded_size"`
	Progress      float64        `json:"progress"` // 0.0 to 100.0
	Error         string         `json:"error,omitempty"`
	Files         []*Download    `json:"files"`
	StartedAt     *time.Time     `json:"started_at,omitempty"`
	CompletedAt   *time.Time     `json:"completed_at,omitempty"`
	LocalPath     string         `json:"local_path"` // Path where model is saved
	
	mu sync.RWMutex
}

// repoDownloads tracks repository-level downloads
var repoDownloads = struct {
	sync.RWMutex
	m map[string]*RepoDownload
}{m: make(map[string]*RepoDownload)}

// DownloadRepository downloads all files for a HuggingFace model
// Downloads safetensors/bin, config.json, tokenizer files, etc.
func (d *Downloader) DownloadRepository(ctx context.Context, modelID string) (*RepoDownload, error) {
	// Check if already downloading
	repoDownloads.RLock()
	if existing, ok := repoDownloads.m[modelID]; ok {
		if existing.Status == DownloadStatusDownloading || existing.Status == DownloadStatusPending {
			repoDownloads.RUnlock()
			return existing, nil
		}
	}
	repoDownloads.RUnlock()

	// Use background context for the actual download work
	// The HTTP request context is only used for initial model info fetch
	modelInfo, err := d.client.GetModelInfo(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model info: %w", err)
	}

	if len(modelInfo.Siblings) == 0 {
		return nil, fmt.Errorf("no files found in repository %s", modelID)
	}

	// Filter important files for model loading
	var filesToDownload []File
	for _, f := range modelInfo.Siblings {
		// Skip large unnecessary files
		if shouldDownloadFile(f.Filename) {
			filesToDownload = append(filesToDownload, f)
		}
	}

	if len(filesToDownload) == 0 {
		return nil, fmt.Errorf("no downloadable model files found in %s", modelID)
	}

	// Calculate total size
	var totalSize int64
	for _, f := range filesToDownload {
		if f.LFS != nil {
			totalSize += f.LFS.Size
		} else {
			totalSize += f.Size
		}
	}

	// Create repo download tracker
	now := time.Now()
	repoDownload := &RepoDownload{
		ID:         generateDownloadID(modelID, "repo"),
		ModelID:    modelID,
		Status:     DownloadStatusDownloading,
		TotalFiles: len(filesToDownload),
		TotalSize:  totalSize,
		StartedAt:  &now,
		LocalPath:  filepath.Join(d.downloadsDir, modelID),
		Files:      make([]*Download, 0, len(filesToDownload)),
	}

	repoDownloads.Lock()
	repoDownloads.m[modelID] = repoDownload
	repoDownloads.Unlock()

	d.logger.WithFields(logrus.Fields{
		"model_id":    modelID,
		"total_files": len(filesToDownload),
		"total_size":  totalSize,
		"local_path":  repoDownload.LocalPath,
	}).Info("Starting repository download")

	// Start downloading files in background
	// Use d.ctx (downloader's context) instead of HTTP request context
	// This prevents cancellation when HTTP response is sent
	go func() {
		for _, file := range filesToDownload {
			var fileSize int64
			var sha string
			if file.LFS != nil {
				fileSize = file.LFS.Size
				sha = file.LFS.OID
			} else {
				fileSize = file.Size
			}

			d.logger.WithFields(logrus.Fields{
				"model_id": modelID,
				"filename": file.Filename,
				"size":     fileSize,
				"sha":      sha,
			}).Debug("Starting file download")

			download, err := d.StartDownload(modelID, file.Filename, fileSize, sha)
			if err != nil {
				d.logger.WithError(err).WithField("file", file.Filename).Error("Failed to start file download")
				repoDownload.mu.Lock()
				repoDownload.FailedFiles++
				repoDownload.mu.Unlock()
				continue
			}

			repoDownload.mu.Lock()
			repoDownload.Files = append(repoDownload.Files, download)
			repoDownload.mu.Unlock()
		}

		// Wait for all downloads to complete (polling)
		// Use downloader's context, not HTTP request context
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-d.ctx.Done():
				repoDownload.mu.Lock()
				repoDownload.Status = DownloadStatusCancelled
				repoDownload.Error = "downloader shutdown"
				repoDownload.mu.Unlock()
				return
			case <-ticker.C:
				completed, failed, downloaded := d.checkRepoProgress(repoDownload)
				
				repoDownload.mu.Lock()
				repoDownload.CompletedFiles = completed
				repoDownload.FailedFiles = failed
				repoDownload.DownloadedSize = downloaded
				if repoDownload.TotalSize > 0 {
					repoDownload.Progress = float64(downloaded) / float64(repoDownload.TotalSize) * 100
				}

				allDone := (completed + failed) >= repoDownload.TotalFiles
				if allDone {
					if failed > 0 {
						repoDownload.Status = DownloadStatusFailed
						repoDownload.Error = fmt.Sprintf("%d files failed to download", failed)
					} else {
						repoDownload.Status = DownloadStatusCompleted
						now := time.Now()
						repoDownload.CompletedAt = &now
						repoDownload.Progress = 100
					}
					repoDownload.mu.Unlock()
					
					d.logger.WithFields(logrus.Fields{
						"model_id":   modelID,
						"completed":  completed,
						"failed":     failed,
						"status":     repoDownload.Status,
						"local_path": repoDownload.LocalPath,
					}).Info("Repository download finished")
					return
				}
				repoDownload.mu.Unlock()
			}
		}
	}()

	return repoDownload, nil
}

// checkRepoProgress checks progress of all files in repo download
func (d *Downloader) checkRepoProgress(repo *RepoDownload) (completed, failed int, downloaded int64) {
	repo.mu.RLock()
	files := repo.Files
	repo.mu.RUnlock()

	for _, f := range files {
		f.Mu.RLock()
		switch f.Status {
		case DownloadStatusCompleted:
			completed++
			downloaded += f.TotalSize
		case DownloadStatusFailed, DownloadStatusCancelled:
			failed++
		case DownloadStatusDownloading, DownloadStatusPending:
			downloaded += f.DownloadedSize
		}
		f.Mu.RUnlock()
	}
	return
}

// GetRepoDownload returns the status of a repository download
func (d *Downloader) GetRepoDownload(modelID string) (*RepoDownload, bool) {
	repoDownloads.RLock()
	defer repoDownloads.RUnlock()
	repo, ok := repoDownloads.m[modelID]
	return repo, ok
}

// ListRepoDownloads returns all repository downloads
func (d *Downloader) ListRepoDownloads() []*RepoDownload {
	repoDownloads.RLock()
	defer repoDownloads.RUnlock()
	
	result := make([]*RepoDownload, 0, len(repoDownloads.m))
	for _, repo := range repoDownloads.m {
		result = append(result, repo)
	}
	return result
}

// shouldDownloadFile determines if a file should be downloaded for model loading
func shouldDownloadFile(filename string) bool {
	// Always download these
	essentialFiles := []string{
		"config.json",
		"tokenizer.json",
		"tokenizer_config.json",
		"special_tokens_map.json",
		"vocab.json",
		"merges.txt",
		"vocab.txt",
		"generation_config.json",
		"preprocessor_config.json",
	}
	
	for _, ef := range essentialFiles {
		if filename == ef {
			return true
		}
	}
	
	// Download model weight files
	if hasAnySuffix(filename, ".safetensors", ".bin", ".pt", ".pth", ".gguf") {
		return true
	}
	
	// Download sentence-transformers specific files
	if hasAnyPrefix(filename, "1_Pooling/", "2_Normalize/") {
		return true
	}
	
	// Skip README, license, git files, etc
	if hasAnySuffix(filename, ".md", ".txt", ".gitattributes") {
		return false
	}
	
	// Skip model card data
	if filename == "README.md" || filename == "LICENSE" {
		return false
	}
	
	return false
}

func hasAnySuffix(s string, suffixes ...string) bool {
	for _, suffix := range suffixes {
		if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
			return true
		}
	}
	return false
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}