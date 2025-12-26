// Package handlers provides GitLab webhook HTTP handler
package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"aigateway/internal/gitlab/webhook"
	"aigateway/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// WebhookHandler interface for handling webhooks
type WebhookHandler interface {
	HandleWebhook(c *gin.Context)
}

// GitLabWebhookHandler handles GitLab webhook requests
type GitLabWebhookHandler struct {
	handler       *webhook.Handler
	webhookSecret string
	integrationID string // Default integration ID
	logger        *logrus.Logger
}

// NewGitLabWebhookHandler creates a new GitLab webhook handler
func NewGitLabWebhookHandler(handler *webhook.Handler, webhookSecret, integrationID string, logger *logrus.Logger) *GitLabWebhookHandler {
	return &GitLabWebhookHandler{
		handler:       handler,
		webhookSecret: webhookSecret,
		integrationID: integrationID,
		logger:        logger,
	}
}

// HandleWebhook handles incoming GitLab webhooks
func (h *GitLabWebhookHandler) HandleWebhook(c *gin.Context) {
	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.WithError(err).Error("Failed to read webhook body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Verify webhook secret
	if h.webhookSecret != "" {
		token := c.GetHeader("X-Gitlab-Token")
		if token == "" {
			// Try HMAC signature verification
			signature := c.GetHeader("X-Gitlab-Signature")
			if signature != "" {
				if !h.verifyHMACSignature(body, signature) {
					h.logger.Warn("Invalid webhook signature")
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
					return
				}
			} else {
				h.logger.Warn("Missing webhook token")
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing webhook token"})
				return
			}
		} else if token != h.webhookSecret {
			h.logger.Warn("Invalid webhook token")
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}
	}

	// Get event type
	eventType := c.GetHeader("X-Gitlab-Event")
	if eventType == "" {
		h.logger.Warn("Missing X-Gitlab-Event header")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing event type"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"event_type":     eventType,
		"content_length": len(body),
	}).Info("Received GitLab webhook")

	// Get integration ID from path or use default
	integrationID := c.Param("integration_id")
	if integrationID == "" {
		integrationID = h.integrationID
	}

	// Create response writer wrapper
	rw := &responseRecorder{ResponseWriter: c.Writer}
	
	// Restore body for handler
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	
	// Process webhook via internal handler
	h.handler.HandleWebhook(rw, c.Request, integrationID)

	// Response already sent by handler
}

// responseRecorder wraps http.ResponseWriter to capture status
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseRecorder) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// verifyHMACSignature verifies the HMAC-SHA256 signature
func (h *GitLabWebhookHandler) verifyHMACSignature(body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	expectedSignature := "sha256=" + hex.EncodeToString(expectedMAC)
	return hmac.Equal([]byte(expectedSignature), []byte(signature))
}

// SetOnMergeCallback sets the callback for merge events (to trigger reindexing)
func (h *GitLabWebhookHandler) SetOnMergeCallback(fn func(integration *models.GitLabIntegration, project *models.GitLabProject, targetBranch string)) {
	if h.handler != nil {
		h.handler.SetOnMergeCallback(fn)
	}
}

