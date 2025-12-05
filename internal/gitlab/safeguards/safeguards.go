// Package safeguards provides edge case handling and safety mechanisms
package safeguards

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ============================================================================
// Request Body Limits
// ============================================================================

// MaxWebhookBodySize maximum webhook body size (1MB)
const MaxWebhookBodySize = 1 * 1024 * 1024

// LimitedBodyReader returns a reader limited to max bytes
func LimitedBodyReader(r io.ReadCloser, maxBytes int64) io.ReadCloser {
	return &limitedReader{
		reader: io.LimitReader(r, maxBytes),
		closer: r,
	}
}

type limitedReader struct {
	reader io.Reader
	closer io.Closer
}

func (lr *limitedReader) Read(p []byte) (int, error) {
	return lr.reader.Read(p)
}

func (lr *limitedReader) Close() error {
	return lr.closer.Close()
}

// ============================================================================
// Webhook Rate Limiter (per integration)
// ============================================================================

// WebhookRateLimiter prevents webhook flooding
type WebhookRateLimiter struct {
	mu          sync.RWMutex
	requests    map[string]*rateLimitEntry
	maxPerMin   int
	cleanupTick time.Duration
	logger      *logrus.Logger
}

type rateLimitEntry struct {
	count     int
	windowEnd time.Time
}

// NewWebhookRateLimiter creates a new rate limiter
func NewWebhookRateLimiter(maxPerMin int, logger *logrus.Logger) *WebhookRateLimiter {
	if maxPerMin <= 0 {
		maxPerMin = 60 // Default: 60 webhooks per minute per integration
	}
	
	rl := &WebhookRateLimiter{
		requests:    make(map[string]*rateLimitEntry),
		maxPerMin:   maxPerMin,
		cleanupTick: 5 * time.Minute,
		logger:      logger,
	}
	
	// Start cleanup goroutine
	go rl.cleanup()
	
	return rl
}

// Allow checks if request should be allowed
func (rl *WebhookRateLimiter) Allow(integrationID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	entry, exists := rl.requests[integrationID]
	
	if !exists || now.After(entry.windowEnd) {
		// New window
		rl.requests[integrationID] = &rateLimitEntry{
			count:     1,
			windowEnd: now.Add(time.Minute),
		}
		return true
	}
	
	if entry.count >= rl.maxPerMin {
		rl.logger.WithFields(logrus.Fields{
			"integration_id": integrationID,
			"count":          entry.count,
			"limit":          rl.maxPerMin,
		}).Warn("Webhook rate limit exceeded")
		return false
	}
	
	entry.count++
	return true
}

func (rl *WebhookRateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupTick)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for id, entry := range rl.requests {
			if now.After(entry.windowEnd) {
				delete(rl.requests, id)
			}
		}
		rl.mu.Unlock()
	}
}

// ============================================================================
// Binary File Detection
// ============================================================================

// IsBinaryFile detects if content is likely binary
func IsBinaryFile(content []byte) bool {
	// Check for null bytes (common in binary files)
	if bytes.Contains(content[:min(len(content), 8000)], []byte{0}) {
		return true
	}
	
	// Check for common binary signatures
	signatures := [][]byte{
		{0x7f, 0x45, 0x4c, 0x46}, // ELF
		{0x4d, 0x5a},             // DOS/PE
		{0x89, 0x50, 0x4e, 0x47}, // PNG
		{0xff, 0xd8, 0xff},       // JPEG
		{0x47, 0x49, 0x46},       // GIF
		{0x50, 0x4b, 0x03, 0x04}, // ZIP/DOCX/XLSX
		{0x25, 0x50, 0x44, 0x46}, // PDF
	}
	
	for _, sig := range signatures {
		if len(content) >= len(sig) && bytes.Equal(content[:len(sig)], sig) {
			return true
		}
	}
	
	// Check for high ratio of non-printable characters
	nonPrintable := 0
	checkLen := min(len(content), 1000)
	for i := 0; i < checkLen; i++ {
		b := content[i]
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			nonPrintable++
		}
	}
	
	return float64(nonPrintable)/float64(checkLen) > 0.3
}

// IsBinaryExtension checks if file extension suggests binary
func IsBinaryExtension(filename string) bool {
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".ico": true, ".webp": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true,
		".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
		".bin": true, ".dat": true, ".db": true, ".sqlite": true,
		".woff": true, ".woff2": true, ".ttf": true, ".otf": true, ".eot": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mov": true, ".wav": true,
		".o": true, ".a": true, ".pyc": true, ".class": true,
	}
	
	idx := strings.LastIndex(filename, ".")
	if idx == -1 {
		return false
	}
	ext := strings.ToLower(filename[idx:])
	return binaryExts[ext]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ============================================================================
// Panic Recovery for Workers
// ============================================================================

// SafeExecute executes fn with panic recovery
func SafeExecute(logger *logrus.Logger, jobID string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.WithFields(logrus.Fields{
				"job_id": jobID,
				"panic":  r,
			}).Error("Panic recovered in job execution")
			err = &PanicError{JobID: jobID, Value: r}
		}
	}()
	return fn()
}

// PanicError represents a recovered panic
type PanicError struct {
	JobID string
	Value interface{}
}

func (e *PanicError) Error() string {
	return "panic in job " + e.JobID
}

// ============================================================================
// Stale Job Detection
// ============================================================================

// StaleJobDetector finds and handles stuck jobs
type StaleJobDetector struct {
	store         StaleJobStore
	logger        *logrus.Logger
	staleTimeout  time.Duration
	checkInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
}

// StaleJobStore interface for stale job operations
type StaleJobStore interface {
	GetStaleJobs(ctx context.Context, staleAfter time.Time) ([]StaleJob, error)
	RequeueStaleJob(ctx context.Context, jobID string) error
	FailStaleJob(ctx context.Context, jobID string, reason string) error
}

// StaleJob represents a potentially stuck job
type StaleJob struct {
	ID          string
	StartedAt   time.Time
	WorkerID    string
	RetryCount  int
	MaxRetries  int
}

// NewStaleJobDetector creates a new detector
func NewStaleJobDetector(store StaleJobStore, logger *logrus.Logger) *StaleJobDetector {
	ctx, cancel := context.WithCancel(context.Background())
	return &StaleJobDetector{
		store:         store,
		logger:        logger,
		staleTimeout:  45 * time.Minute, // Job considered stale after 45 min
		checkInterval: 5 * time.Minute,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins periodic stale job detection
func (d *StaleJobDetector) Start() {
	go d.run()
}

// Stop stops the detector
func (d *StaleJobDetector) Stop() {
	d.cancel()
}

func (d *StaleJobDetector) run() {
	ticker := time.NewTicker(d.checkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.checkStaleJobs()
		}
	}
}

func (d *StaleJobDetector) checkStaleJobs() {
	staleAfter := time.Now().Add(-d.staleTimeout)
	
	jobs, err := d.store.GetStaleJobs(d.ctx, staleAfter)
	if err != nil {
		d.logger.WithError(err).Error("Failed to get stale jobs")
		return
	}
	
	for _, job := range jobs {
		d.logger.WithFields(logrus.Fields{
			"job_id":     job.ID,
			"worker_id":  job.WorkerID,
			"started_at": job.StartedAt,
		}).Warn("Detected stale job")
		
		// Requeue if retries available, otherwise fail
		if job.RetryCount < job.MaxRetries {
			if err := d.store.RequeueStaleJob(d.ctx, job.ID); err != nil {
				d.logger.WithError(err).Error("Failed to requeue stale job")
			}
		} else {
			if err := d.store.FailStaleJob(d.ctx, job.ID, "Job timed out after "+d.staleTimeout.String()); err != nil {
				d.logger.WithError(err).Error("Failed to fail stale job")
			}
		}
	}
}

// ============================================================================
// LLM Response Validation
// ============================================================================

// ValidateLLMResponse checks if LLM response is valid JSON or usable
func ValidateLLMResponse(response string) error {
	response = strings.TrimSpace(response)
	
	if response == "" {
		return &LLMResponseError{Reason: "empty response"}
	}
	
	if len(response) < 10 {
		return &LLMResponseError{Reason: "response too short"}
	}
	
	// Check for common error patterns
	errorPatterns := []string{
		"I cannot",
		"I'm sorry",
		"I apologize",
		"Error:",
		"error:",
		"Internal server error",
	}
	
	lower := strings.ToLower(response)
	for _, pattern := range errorPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) && len(response) < 200 {
			return &LLMResponseError{Reason: "response appears to be an error message"}
		}
	}
	
	return nil
}

// LLMResponseError represents an LLM response error
type LLMResponseError struct {
	Reason string
}

func (e *LLMResponseError) Error() string {
	return "invalid LLM response: " + e.Reason
}

// ============================================================================
// GitLab API Pagination Helper
// ============================================================================

// PaginatedFetcher handles GitLab API pagination
type PaginatedFetcher struct {
	client    *http.Client
	baseURL   string
	token     string
	perPage   int
	maxPages  int
	logger    *logrus.Logger
}

// NewPaginatedFetcher creates a paginated fetcher
func NewPaginatedFetcher(client *http.Client, baseURL, token string, logger *logrus.Logger) *PaginatedFetcher {
	return &PaginatedFetcher{
		client:   client,
		baseURL:  baseURL,
		token:    token,
		perPage:  100,
		maxPages: 10, // Safety limit
		logger:   logger,
	}
}

// FetchAllPages fetches all pages for a paginated endpoint
func (f *PaginatedFetcher) FetchAllPages(ctx context.Context, endpoint string, processor func(body []byte) (hasMore bool, err error)) error {
	page := 1
	
	for page <= f.maxPages {
		url := f.buildURL(endpoint, page)
		
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("PRIVATE-TOKEN", f.token)
		
		resp, err := f.client.Do(req)
		if err != nil {
			return err
		}
		
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		if err != nil {
			return err
		}
		
		if resp.StatusCode != http.StatusOK {
			return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
		}
		
		hasMore, err := processor(body)
		if err != nil {
			return err
		}
		
		if !hasMore {
			break
		}
		
		// Check X-Next-Page header
		nextPage := resp.Header.Get("X-Next-Page")
		if nextPage == "" {
			break
		}
		
		page++
	}
	
	return nil
}

func (f *PaginatedFetcher) buildURL(endpoint string, page int) string {
	separator := "?"
	if strings.Contains(endpoint, "?") {
		separator = "&"
	}
	return f.baseURL + endpoint + separator + "page=" + string(rune('0'+page)) + "&per_page=" + string(rune('0'+f.perPage/10)) + string(rune('0'+f.perPage%10))
}

// APIError represents an API error
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return "API error: status=" + string(rune('0'+e.StatusCode/100)) + string(rune('0'+(e.StatusCode/10)%10)) + string(rune('0'+e.StatusCode%10))
}

// ============================================================================
// Empty Diff Handler
// ============================================================================

// IsEmptyDiff checks if diff is effectively empty
func IsEmptyDiff(diff string) bool {
	diff = strings.TrimSpace(diff)
	
	if diff == "" {
		return true
	}
	
	// Check if only whitespace changes
	lines := strings.Split(diff, "\n")
	meaningfulChanges := 0
	
	for _, line := range lines {
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			// Skip header lines
			if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
				continue
			}
			// Check if line has non-whitespace content
			content := strings.TrimPrefix(strings.TrimPrefix(line, "+"), "-")
			if strings.TrimSpace(content) != "" {
				meaningfulChanges++
			}
		}
	}
	
	return meaningfulChanges == 0
}

// ============================================================================
// Circuit Breaker for External Services
// ============================================================================

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	mu           sync.RWMutex
	failures     int
	successCount int
	state        CircuitState
	threshold    int
	resetTimeout time.Duration
	lastFailure  time.Time
	logger       *logrus.Logger
}

// CircuitState represents circuit breaker state
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int, resetTimeout time.Duration, logger *logrus.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:    threshold,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
		logger:       logger,
	}
}

// Execute executes fn with circuit breaker protection
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.AllowRequest() {
		return &CircuitOpenError{}
	}
	
	err := fn()
	cb.RecordResult(err)
	return err
}

// AllowRequest checks if request should be allowed
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.RLock()
	state := cb.state
	lastFailure := cb.lastFailure
	cb.mu.RUnlock()
	
	switch state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(lastFailure) > cb.resetTimeout {
			cb.mu.Lock()
			cb.state = CircuitHalfOpen
			cb.mu.Unlock()
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	
	return true
}

// RecordResult records the result of an operation
func (cb *CircuitBreaker) RecordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		cb.successCount = 0
		
		if cb.failures >= cb.threshold {
			cb.state = CircuitOpen
			cb.logger.WithField("failures", cb.failures).Warn("Circuit breaker opened")
		}
	} else {
		if cb.state == CircuitHalfOpen {
			cb.successCount++
			if cb.successCount >= 3 {
				cb.state = CircuitClosed
				cb.failures = 0
				cb.logger.Info("Circuit breaker closed")
			}
		} else {
			cb.failures = 0
		}
	}
}

// CircuitOpenError indicates circuit is open
type CircuitOpenError struct{}

func (e *CircuitOpenError) Error() string {
	return "circuit breaker is open"
}

// ============================================================================
// Utility: Read int64 from binary
// ============================================================================

func readInt64(data []byte) int64 {
	if len(data) < 8 {
		return 0
	}
	return int64(binary.LittleEndian.Uint64(data))
}

