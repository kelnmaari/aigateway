// Package handlers provides HTTP handlers for dashboard batch endpoints
// v3.1.0: Optimized AJAX loading with single request
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// DashboardHandler handles dashboard batch endpoints
type DashboardHandler struct {
	db     storage.Database
	logger *logrus.Logger
}

// NewDashboardHandler creates a new DashboardHandler
func NewDashboardHandler(db storage.Database, logger *logrus.Logger) *DashboardHandler {
	return &DashboardHandler{
		db:     db,
		logger: logger,
	}
}

// DashboardStatsResponse represents aggregated dashboard stats
type DashboardStatsResponse struct {
	Conversations int                   `json:"conversations"`
	Tenants       int                   `json:"tenants"`
	APIKeys       int                   `json:"api_keys"`
	Requests30d   int64                 `json:"requests_30d"`
	RecentConvs   []ConversationSummary `json:"recent_conversations"`
	RecentTenants []TenantSummary       `json:"recent_tenants"`
	LoadedAt      time.Time             `json:"loaded_at"`
}

// ConversationSummary represents a summary of a conversation
type ConversationSummary struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Model        string    `json:"model"`
	MessageCount int       `json:"message_count"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TenantSummary represents a summary of a tenant
type TenantSummary struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	MemberCount int       `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetDashboardStats returns aggregated dashboard statistics (AJAX-01)
// GET /api/dashboard/stats
func (h *DashboardHandler) GetDashboardStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Extract user from JWT (set by AuthMiddleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	userIDStr := userID.(string)
	h.logger.WithField("user_id", userIDStr).Debug("Loading dashboard stats")

	// Fetch data in parallel using goroutines
	type result struct {
		conversations []ConversationSummary
		tenants       []TenantSummary
		tenantCount   int
		apiKeyCount   int
		requests30d   int64
		err           error
	}

	resultChan := make(chan result, 1)

	go func() {
		var res result

		// 1. Get conversations count + recent conversations
		convs, err := h.db.ListUserConversations(ctx, userIDStr, models.ConversationFilters{
			Limit: 5,
		})
		if err != nil {
			res.err = err
			resultChan <- res
			return
		}

		// Convert to summaries
		res.conversations = make([]ConversationSummary, 0, len(convs))
		for _, conv := range convs {
			// Get message count for each conversation
			messages, _ := h.db.ListConversationMessages(ctx, conv.ID)
			res.conversations = append(res.conversations, ConversationSummary{
				ID:           conv.ID,
				Title:        conv.Title,
				Model:        conv.Model,
				MessageCount: len(messages),
				UpdatedAt:    conv.UpdatedAt,
			})
		}

		// 2. Get tenants for this user
		tenants, err := h.db.ListUserTenants(ctx, userIDStr)
		if err != nil {
			res.err = err
			resultChan <- res
			return
		}

		res.tenantCount = len(tenants)

		// Convert to summaries (top 3 recent)
		limitTenants := min(len(tenants), 3)
		res.tenants = make([]TenantSummary, 0, limitTenants)
		for i := 0; i < limitTenants; i++ {
			tenant := tenants[i]
			res.tenants = append(res.tenants, TenantSummary{
				ID:          tenant.ID,
				Name:        tenant.Name,
				MemberCount: tenant.MemberCount,
				CreatedAt:   tenant.CreatedAt,
			})
		}

		// 3. Get API keys count (personal)
		apiKeys, err := h.db.ListPersonalAPIKeys(ctx, userIDStr)
		if err != nil {
			res.err = err
			resultChan <- res
			return
		}
		res.apiKeyCount = len(apiKeys)

		// 4. Get requests count (30d) - placeholder for now
		// TODO: Implement GetRequestCount30d in storage layer
		res.requests30d = 0

		resultChan <- res
	}()

	// Wait for result
	res := <-resultChan
	if res.err != nil {
		h.logger.WithError(res.err).Error("Failed to load dashboard stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load dashboard stats"})
		return
	}

	// Build response
	response := DashboardStatsResponse{
		Conversations: len(res.conversations),
		Tenants:       res.tenantCount,
		APIKeys:       res.apiKeyCount,
		Requests30d:   res.requests30d,
		RecentConvs:   res.conversations,
		RecentTenants: res.tenants,
		LoadedAt:      time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// AdminSummaryResponse represents aggregated admin panel stats
type AdminSummaryResponse struct {
	TotalUsers    int       `json:"total_users"`
	TotalTenants  int       `json:"total_tenants"`
	TotalAPIKeys  int       `json:"total_api_keys"`
	TotalRequests int64     `json:"total_requests"`
	LoadedAt      time.Time `json:"loaded_at"`
}

// GetAdminSummary returns aggregated admin panel statistics (AJAX-01)
// GET /api/admin/summary
func (h *DashboardHandler) GetAdminSummary(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Check if user is admin (set by AdminMiddleware)
	isAdmin, exists := c.Get("is_admin")
	if !exists || !isAdmin.(bool) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	h.logger.Debug("Loading admin summary stats")

	// Fetch data in parallel
	type result struct {
		userCount    int
		tenantCount  int
		apiKeyCount  int
		requestCount int64
		err          error
	}

	resultChan := make(chan result, 1)

	go func() {
		var res result

		// 1. Get total users count
		users, err := h.db.ListUsers(ctx, models.UserFilters{})
		if err != nil {
			res.err = err
			resultChan <- res
			return
		}
		res.userCount = len(users)

		// 2. Get total tenants count
		// Count through members (no direct ListTenants method)
		members, err := h.db.ListTenantMembers(ctx, "")
		if err != nil {
			res.err = err
			resultChan <- res
			return
		}

		// Get unique tenants
		tenantIDs := make(map[string]bool)
		for _, m := range members {
			tenantIDs[m.TenantID] = true
		}
		res.tenantCount = len(tenantIDs)

		// 3. Get total API keys count (all users)
		apiKeys, err := h.db.ListAPIKeys(ctx)
		if err != nil {
			res.err = err
			resultChan <- res
			return
		}
		res.apiKeyCount = len(apiKeys)

		// 4. Get total requests count
		// TODO: Implement GetTotalRequestCount in storage layer
		res.requestCount = 0

		resultChan <- res
	}()

	// Wait for result
	res := <-resultChan
	if res.err != nil {
		h.logger.WithError(res.err).Error("Failed to load admin summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load admin summary"})
		return
	}

	// Build response
	response := AdminSummaryResponse{
		TotalUsers:    res.userCount,
		TotalTenants:  res.tenantCount,
		TotalAPIKeys:  res.apiKeyCount,
		TotalRequests: res.requestCount,
		LoadedAt:      time.Now(),
	}

	c.JSON(http.StatusOK, response)
}
