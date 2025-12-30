// Package handlers provides HTTP handlers for GitLab analytics API.
package handlers

import (
	"context"
	"net/http"
	"time"

	"aigateway/internal/gitlab/analytics"
	"aigateway/internal/gitlab/storage"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GitLabAnalyticsHandler handles analytics API requests.
type GitLabAnalyticsHandler struct {
	store   storage.Store
	service *analytics.Service
	logger  *logrus.Logger
}

// NewGitLabAnalyticsHandler creates a new analytics handler.
func NewGitLabAnalyticsHandler(store storage.Store, analyticsStore analytics.Store, logger *logrus.Logger) *GitLabAnalyticsHandler {
	return &GitLabAnalyticsHandler{
		store:   store,
		service: analytics.NewService(analyticsStore),
		logger:  logger,
	}
}

// GetDashboard GET /api/admin/gitlab/analytics/dashboard
// Returns the main analytics dashboard.
func (h *GitLabAnalyticsHandler) GetDashboard(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Parse filter parameters
	filter := h.parseFilter(c)

	h.logger.WithFields(logrus.Fields{
		"start_date": filter.StartDate,
		"end_date":   filter.EndDate,
	}).Info("Fetching analytics dashboard")

	dashboard, err := h.service.GetDashboard(ctx, filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get analytics dashboard")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get dashboard: " + err.Error()})
		return
	}

	// Calculate additional metrics
	dashboard.DependencyHealth = h.calculateDependencyHealth(ctx)
	dashboard.SecurityScore = h.calculateSecurityScore(ctx)
	dashboard.TeamProductivity = h.calculateTeamProductivity(ctx, filter)

	c.JSON(http.StatusOK, dashboard)
}

// GetProjectAnalytics GET /api/admin/gitlab/projects/:id/analytics
// Returns detailed analytics for a specific project.
func (h *GitLabAnalyticsHandler) GetProjectAnalytics(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	filter := h.parseFilter(c)

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"start_date": filter.StartDate,
		"end_date":   filter.EndDate,
	}).Info("Fetching project analytics")

	analytics, err := h.service.GetProjectAnalytics(ctx, projectID, filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get project analytics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get analytics: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

// GetModelComparison GET /api/admin/gitlab/analytics/models
// Returns model comparison analytics.
func (h *GitLabAnalyticsHandler) GetModelComparison(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	filter := h.parseFilter(c)

	h.logger.Info("Fetching model comparison")

	comparison, err := h.service.GetModelComparison(ctx, filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get model comparison")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get comparison: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, comparison)
}

// GetSecurityOverview GET /api/admin/gitlab/analytics/security
// Returns security analytics overview.
func (h *GitLabAnalyticsHandler) GetSecurityOverview(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	h.logger.Info("Fetching security overview")

	securityScore := h.calculateSecurityScore(ctx)

	c.JSON(http.StatusOK, securityScore)
}

// GetDependencyHealth GET /api/admin/gitlab/analytics/dependencies
// Returns dependency health analytics.
func (h *GitLabAnalyticsHandler) GetDependencyHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	h.logger.Info("Fetching dependency health")

	health := h.calculateDependencyHealth(ctx)

	c.JSON(http.StatusOK, health)
}

// ExportReport GET /api/admin/gitlab/analytics/export
// Exports analytics report.
func (h *GitLabAnalyticsHandler) ExportReport(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	format := c.Query("format")
	if format == "" {
		format = "json"
	}

	filter := h.parseFilter(c)

	h.logger.WithField("format", format).Info("Exporting analytics report")

	data, err := h.service.ExportReport(ctx, filter, format)
	if err != nil {
		h.logger.WithError(err).Error("Failed to export report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export: " + err.Error()})
		return
	}

	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=analytics_report.csv")
	default:
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=analytics_report.json")
	}

	c.Data(http.StatusOK, c.Writer.Header().Get("Content-Type"), data)
}

func (h *GitLabAnalyticsHandler) parseFilter(c *gin.Context) *analytics.StatsFilter {
	filter := &analytics.StatsFilter{}

	if integrationID := c.Query("integration_id"); integrationID != "" {
		filter.IntegrationID = integrationID
	}
	if projectID := c.Query("project_id"); projectID != "" {
		filter.ProjectID = projectID
	}
	if modelID := c.Query("model_id"); modelID != "" {
		filter.ModelID = modelID
	}

	// Default to last 30 days
	filter.EndDate = time.Now()
	filter.StartDate = filter.EndDate.AddDate(0, 0, -30)

	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			filter.StartDate = t
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			filter.EndDate = t
		}
	}

	return filter
}

func (h *GitLabAnalyticsHandler) calculateDependencyHealth(ctx context.Context) *analytics.DependencyHealthStats {
	// This would query actual dependency scan data from the database
	// For now, return sample data structure
	return &analytics.DependencyHealthStats{
		TotalDependencies:  0,
		OutdatedCount:      0,
		OutdatedPercentage: 0,
		VulnerableCount:    0,
		CriticalVulns:      0,
		HighVulns:          0,
		MediumVulns:        0,
	}
}

func (h *GitLabAnalyticsHandler) calculateSecurityScore(ctx context.Context) *analytics.SecurityScoreStats {
	// This would query actual security scan data from the database
	// For now, return sample data structure
	return &analytics.SecurityScoreStats{
		OverallScore:     100,
		SecretsScanScore: 100,
		SASTScore:        100,
		DependencyScore:  100,
		TotalFindings:    0,
		CriticalFindings: 0,
		HighFindings:     0,
		MediumFindings:   0,
		LowFindings:      0,
	}
}

func (h *GitLabAnalyticsHandler) calculateTeamProductivity(ctx context.Context, filter *analytics.StatsFilter) *analytics.TeamProductivityStats {
	// This would calculate actual productivity metrics from review data
	// For now, return sample data structure
	return &analytics.TeamProductivityStats{
		TotalMRsReviewed:     0,
		AvgReviewTimeMinutes: 0,
		IssuesFoundPerMR:     0,
		AutoFixApplied:       0,
		TimesSavedHours:      0,
	}
}

