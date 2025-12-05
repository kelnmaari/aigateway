// Package webhook handles GitLab webhook events for MR review
package webhook

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/models"

	"github.com/sirupsen/logrus"
)

// Handler processes GitLab webhook events
type Handler struct {
	store  storage.Store
	logger *logrus.Logger

	// Debounce mechanism for webhook events
	debounce    *Debouncer
	debounceMs  int

	// Rate limiter for webhook flood protection
	rateLimiter WebhookRateLimiter

	// Job enqueue function (injected)
	enqueueJob func(ctx context.Context, job *models.GitLabAnalysisJob) error
}

// WebhookRateLimiter interface for rate limiting webhooks
type WebhookRateLimiter interface {
	Allow(integrationID string) bool
}

// Config holds handler configuration
type Config struct {
	DebounceMs int // Debounce time in milliseconds (default: 5000)
}

// NewHandler creates a new webhook handler
func NewHandler(store storage.Store, logger *logrus.Logger, cfg *Config) *Handler {
	debounceMs := 5000
	if cfg != nil && cfg.DebounceMs > 0 {
		debounceMs = cfg.DebounceMs
	}

	return &Handler{
		store:      store,
		logger:     logger,
		debounce:   NewDebouncer(time.Duration(debounceMs) * time.Millisecond),
		debounceMs: debounceMs,
	}
}

// SetEnqueueFunc sets the job enqueue function
func (h *Handler) SetEnqueueFunc(fn func(ctx context.Context, job *models.GitLabAnalysisJob) error) {
	h.enqueueJob = fn
}

// SetRateLimiter sets the rate limiter
func (h *Handler) SetRateLimiter(rl WebhookRateLimiter) {
	h.rateLimiter = rl
}

// HandleWebhook handles incoming GitLab webhook requests
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request, integrationID string) {
	ctx := r.Context()

	// Rate limit check (per integration)
	if h.rateLimiter != nil && !h.rateLimiter.Allow(integrationID) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Limit body size to prevent abuse
	r.Body = http.MaxBytesReader(w, r.Body, 1*1024*1024) // 1MB max

	// Get integration
	integration, err := h.store.GetIntegration(ctx, integrationID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get integration")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if integration == nil {
		http.Error(w, "Integration not found", http.StatusNotFound)
		return
	}
	if integration.Status != models.GitLabIntegrationStatusActive {
		http.Error(w, "Integration disabled", http.StatusServiceUnavailable)
		return
	}

	// Verify webhook secret
	token := r.Header.Get("X-Gitlab-Token")
	if !h.verifyToken(token, integration.WebhookSecret) {
		h.logger.Warn("Invalid webhook token")
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read webhook body")
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// Determine event type
	eventType := r.Header.Get("X-Gitlab-Event")
	h.logger.WithFields(logrus.Fields{
		"integration_id": integrationID,
		"event_type":     eventType,
	}).Info("Received webhook event")

	// Handle based on event type
	switch eventType {
	case "Merge Request Hook":
		h.handleMergeRequestEvent(ctx, w, integration, body)
	default:
		h.logger.WithField("event_type", eventType).Debug("Ignoring non-MR webhook event")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK - ignored"))
	}
}

// handleMergeRequestEvent processes merge request webhook events
func (h *Handler) handleMergeRequestEvent(ctx context.Context, w http.ResponseWriter, integration *models.GitLabIntegration, body []byte) {
	var event client.MergeRequestEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.logger.WithError(err).Error("Failed to parse MR event")
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Validate event
	if event.ObjectAttributes == nil || event.Project == nil {
		h.logger.Warn("Invalid MR event: missing required fields")
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Only process open, update, reopen actions
	action := event.ObjectAttributes.Action
	if !isReviewableAction(action) {
		h.logger.WithField("action", action).Debug("Ignoring non-reviewable action")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK - action ignored"))
		return
	}

	// Skip draft/WIP MRs (can be configured per project)
	if event.ObjectAttributes.Draft || event.ObjectAttributes.WorkInProgress {
		h.logger.Debug("Ignoring draft/WIP MR")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK - draft ignored"))
		return
	}

	// Find project configuration
	project, err := h.store.GetProjectByGitLabID(ctx, integration.ID, event.Project.ID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get project")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	if project == nil {
		h.logger.WithField("gitlab_project_id", event.Project.ID).Debug("Project not configured for review")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK - project not configured"))
		return
	}
	if project.Status != models.GitLabProjectStatusActive || !project.AutoReview {
		h.logger.Debug("Auto-review disabled for project")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK - auto-review disabled"))
		return
	}

	// Check project settings for draft skip
	if project.Settings.SkipDraftMRs && (event.ObjectAttributes.Draft || event.ObjectAttributes.WorkInProgress) {
		h.logger.Debug("Skipping draft MR per project settings")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK - draft skipped"))
		return
	}

	// Record webhook event for deduplication
	webhookEvent := &models.GitLabWebhookEvent{
		IntegrationID: integration.ID,
		ProjectID:     event.Project.ID,
		MRIID:         event.ObjectAttributes.IID,
		EventType:     "merge_request",
		Action:        action,
		ObjectID:      event.ObjectAttributes.ID,
	}
	if err := h.store.CreateWebhookEvent(ctx, webhookEvent); err != nil {
		h.logger.WithError(err).Warn("Failed to record webhook event")
		// Continue processing anyway
	}

	// Check for duplicate events
	existingEvent, err := h.store.GetWebhookEvent(ctx, integration.ID, event.Project.ID, event.ObjectAttributes.IID, event.ObjectAttributes.ID)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to check for duplicate event")
	}
	if existingEvent != nil && existingEvent.ProcessedAt != nil {
		h.logger.Debug("Duplicate webhook event - already processed")
		h.store.MarkWebhookEventDeduplicated(ctx, webhookEvent.ID)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK - duplicate"))
		return
	}

	// Use debounce to avoid processing multiple events for the same MR
	debounceKey := fmt.Sprintf("%s:%d:%d", integration.ID, event.Project.ID, event.ObjectAttributes.IID)
	
	h.debounce.Debounce(debounceKey, func() {
		// Process in background after debounce
		h.processReview(context.Background(), integration, project, &event, webhookEvent.ID)
	})

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("OK - queued"))
}

// processReview creates a review and enqueues analysis job
func (h *Handler) processReview(ctx context.Context, integration *models.GitLabIntegration, project *models.GitLabProject, event *client.MergeRequestEvent, webhookEventID string) {
	// Determine priority
	priority := h.determinePriority(project, event)

	// Create or update review record
	existingReview, err := h.store.GetReviewByMR(ctx, project.ID, event.ObjectAttributes.IID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to check existing review")
		return
	}

	var reviewID string
	if existingReview != nil && existingReview.Status != models.GitLabReviewStatusCompleted {
		// Update existing review
		reviewID = existingReview.ID
		h.logger.WithField("review_id", reviewID).Info("Updating existing review")
	} else {
		// Create new review
		review := &models.GitLabMRReview{
			ProjectID:     project.ID,
			IntegrationID: integration.ID,
			MRIID:         event.ObjectAttributes.IID,
			MRTitle:       event.ObjectAttributes.Title,
			MRAuthor:      event.User.Username,
			MRAuthorID:    event.ObjectAttributes.AuthorID,
			SourceBranch:  event.ObjectAttributes.SourceBranch,
			TargetBranch:  event.ObjectAttributes.TargetBranch,
			MRURL:         event.ObjectAttributes.URL,
			Status:        models.GitLabReviewStatusPending,
			Priority:      priority,
		}

		if err := h.store.CreateReview(ctx, review); err != nil {
			h.logger.WithError(err).Error("Failed to create review")
			return
		}
		reviewID = review.ID
		h.logger.WithField("review_id", reviewID).Info("Created new review")
	}

	// Mark webhook event as processed
	if webhookEventID != "" {
		h.store.MarkWebhookEventProcessed(ctx, webhookEventID)
	}

	// Create analysis job
	job := &models.GitLabAnalysisJob{
		ReviewID:      reviewID,
		ProjectID:     project.ID,
		IntegrationID: integration.ID,
		Status:        models.GitLabJobStatusPending,
		Priority:      priority,
		MRIID:         event.ObjectAttributes.IID,
		MRTitle:       event.ObjectAttributes.Title,
		MaxRetries:    3,
	}

	if h.enqueueJob != nil {
		if err := h.enqueueJob(ctx, job); err != nil {
			h.logger.WithError(err).Error("Failed to enqueue job")
			h.store.UpdateReviewStatus(ctx, reviewID, models.GitLabReviewStatusFailed, "Failed to enqueue job: "+err.Error())
			return
		}
	} else {
		// Fallback: create job in database
		if err := h.store.CreateJob(ctx, job); err != nil {
			h.logger.WithError(err).Error("Failed to create job")
			h.store.UpdateReviewStatus(ctx, reviewID, models.GitLabReviewStatusFailed, "Failed to create job: "+err.Error())
			return
		}
	}

	// Update review status to queued
	h.store.UpdateReviewStatus(ctx, reviewID, models.GitLabReviewStatusQueued, "")
	h.logger.WithFields(logrus.Fields{
		"review_id": reviewID,
		"job_id":    job.ID,
		"priority":  priority,
	}).Info("Review queued for analysis")
}

// determinePriority determines review priority based on target branch
func (h *Handler) determinePriority(project *models.GitLabProject, event *client.MergeRequestEvent) models.GitLabReviewPriority {
	targetBranch := event.ObjectAttributes.TargetBranch

	// Urgent for main/master branches
	if targetBranch == "main" || targetBranch == "master" {
		return models.GitLabReviewPriorityHigh
	}

	// Check project target branches for priority
	for _, branch := range project.Settings.TargetBranches {
		if branch == targetBranch {
			return models.GitLabReviewPriorityHigh
		}
	}

	// Check for release branches
	if len(targetBranch) > 7 && targetBranch[:7] == "release" {
		return models.GitLabReviewPriorityHigh
	}

	return models.GitLabReviewPriorityNormal
}

// verifyToken securely compares the webhook token
func (h *Handler) verifyToken(provided, expected string) bool {
	if expected == "" {
		return true // No secret configured
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

// isReviewableAction checks if the MR action should trigger a review
func isReviewableAction(action string) bool {
	switch action {
	case "open", "reopen", "update":
		return true
	default:
		return false
	}
}

// ============================================================================
// Debouncer - Prevents duplicate processing of rapid webhook events
// ============================================================================

// Debouncer provides debouncing for webhook events
type Debouncer struct {
	mu       sync.Mutex
	pending  map[string]*debounceEntry
	duration time.Duration
}

type debounceEntry struct {
	timer  *time.Timer
	cancel chan struct{}
}

// NewDebouncer creates a new debouncer
func NewDebouncer(duration time.Duration) *Debouncer {
	return &Debouncer{
		pending:  make(map[string]*debounceEntry),
		duration: duration,
	}
}

// Debounce debounces the execution of fn for the given key
func (d *Debouncer) Debounce(key string, fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Cancel existing timer for this key
	if entry, ok := d.pending[key]; ok {
		entry.timer.Stop()
		close(entry.cancel)
		delete(d.pending, key)
	}

	// Create new timer
	cancel := make(chan struct{})
	timer := time.AfterFunc(d.duration, func() {
		d.mu.Lock()
		delete(d.pending, key)
		d.mu.Unlock()
		
		// Execute the function
		fn()
	})

	d.pending[key] = &debounceEntry{
		timer:  timer,
		cancel: cancel,
	}
}

// Cancel cancels a pending debounced call
func (d *Debouncer) Cancel(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if entry, ok := d.pending[key]; ok {
		entry.timer.Stop()
		close(entry.cancel)
		delete(d.pending, key)
	}
}

// Stop stops all pending debounced calls
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for key, entry := range d.pending {
		entry.timer.Stop()
		close(entry.cancel)
		delete(d.pending, key)
	}
}

