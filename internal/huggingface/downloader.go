// Package huggingface provides model downloading with resume support
// Version: v3.0.0 - HF-02: Model Downloader
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
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DownloadStatus represents download state
type DownloadStatus string

const (
	DownloadStatusPending    DownloadStatus = "pending"
	DownloadStatusDownloading DownloadStatus = "downloading"
	DownloadStatusCompleted   DownloadStatus = "completed"
	DownloadStatusFailed      DownloadStatus = "failed"
	DownloadStatusPaused      DownloadStatus = "paused"
	DownloadStatusCancelled   DownloadStatus = "cancelled"
)

// Download represents a model file download
type Download struct {
	ID              string         `json:"id"`
	ModelID         string         `json:"model_id"`
	Filename        string         `json:"filename"`
	URL             string         `json:"url"`
	DestPath        string         `json:"dest_path"`
	TotalSize       int64          `json:"total_size"`
	DownloadedSize  int64          `json:"downloaded_size"`
	Status          DownloadStatus `json:"status"`
	Error           string         `json:"error,omitempty"`
	Progress        float64        `json:"progress"`         // 0.0 to 100.0
	Speed           int64          `json:"speed"`            // bytes per second
	ETA             time.Duration  `json:"eta"`              // estimated time remaining
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	SHA256          string         `json:"sha256,omitempty"` // Expected SHA256
	VerifiedSHA256  string         `json:"verified_sha256,omitempty"`
	
	// Internal
	ctx        context.Context
	cancel     context.CancelFunc
	lastUpdate time.Time
	lastSize   int64
	Mu         sync.RWMutex // Exported for external locking
}

// Downloader manages model downloads
type Downloader struct {
	client            *Client
	downloadsDir      string
	maxConcurrent     int
	autoResume        bool
	logger            *logrus.Logger
	
	downloads         map[string]*Download
	downloadQueue     chan *Download
	activeDownloads   int
	mu                sync.RWMutex
	
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
}

// NewDownloader creates a new model downloader
func NewDownloader(client *Client, downloadsDir string, maxConcurrent int, autoResume bool, logger *logrus.Logger) (*Downloader, error) {
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create downloads directory: %w", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	d := &Downloader{
		client:          client,
		downloadsDir:    downloadsDir,
		maxConcurrent:   maxConcurrent,
		autoResume:      autoResume,
		logger:          logger,
		downloads:       make(map[string]*Download),
		downloadQueue:   make(chan *Download, 100),
		ctx:             ctx,
		cancel:          cancel,
	}
	
	// Start download workers
	for i := 0; i < maxConcurrent; i++ {
		d.wg.Add(1)
		go d.downloadWorker(i)
	}
	
	logger.WithFields(logrus.Fields{
		"downloads_dir":   downloadsDir,
		"max_concurrent":  maxConcurrent,
		"auto_resume":     autoResume,
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
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// Check for partial download
	var downloadedSize int64
	if info, err := os.Stat(destPath + ".part"); err == nil {
		downloadedSize = info.Size()
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
	defer file.Close()
	
	// Create HTTP request with Range header for resume
	req, err := http.NewRequestWithContext(download.ctx, "GET", download.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	if download.DownloadedSize > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", download.DownloadedSize))
	}
	
	// Execute request
	resp, err := d.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check response
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
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
			return download.ctx.Err()
			
		case <-ticker.C:
			d.updateProgress(download)
			
		default:
			n, err := resp.Body.Read(buf)
			if n > 0 {
				// Write to file
				if _, writeErr := file.Write(buf[:n]); writeErr != nil {
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
							return fmt.Errorf("SHA256 mismatch: expected %s, got %s", download.SHA256, computedHash)
						}
					}
					
					// Rename .part file to final name
					if err := os.Rename(partPath, download.DestPath); err != nil {
						return fmt.Errorf("failed to rename file: %w", err)
					}
					
					download.Mu.Lock()
					download.Status = DownloadStatusCompleted
					now := time.Now()
					download.CompletedAt = &now
					download.Progress = 100.0
					download.Mu.Unlock()
					
					d.logger.WithFields(logrus.Fields{
						"download_id": download.ID,
						"model_id":    download.ModelID,
						"filename":    download.Filename,
						"size":        download.TotalSize,
						"duration":    time.Since(*download.StartedAt),
					}).Info("Download completed successfully")
					
					return nil
				}
				
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

