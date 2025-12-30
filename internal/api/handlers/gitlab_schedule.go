// Package handlers provides HTTP handlers for scheduled scan management.
package handlers

import (
	"net/http"

	"aigateway/internal/gitlab/dependencies/schedule"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GitLabScheduleHandler handles scheduled scan API requests.
type GitLabScheduleHandler struct {
	scheduler     *schedule.Scheduler
	scheduleStore schedule.ScheduleStore
	logger        *logrus.Logger
}

// NewGitLabScheduleHandler creates a new schedule handler.
func NewGitLabScheduleHandler(
	sched *schedule.Scheduler,
	scheduleStore schedule.ScheduleStore,
	logger *logrus.Logger,
) *GitLabScheduleHandler {
	return &GitLabScheduleHandler{
		scheduler:     sched,
		scheduleStore: scheduleStore,
		logger:        logger,
	}
}

// CreateSchedule POST /api/admin/gitlab/schedules
// Creates a new scheduled scan.
func (h *GitLabScheduleHandler) CreateSchedule(c *gin.Context) {
	var req schedule.CreateScheduledScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	createdSchedule, err := h.scheduler.AddSchedule(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create scheduled scan")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, createdSchedule)
}

// GetSchedule GET /api/admin/gitlab/schedules/:id
// Gets a scheduled scan by ID.
func (h *GitLabScheduleHandler) GetSchedule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Schedule ID is required"})
		return
	}

	schedule, err := h.scheduleStore.GetSchedule(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// ListSchedules GET /api/admin/gitlab/schedules
// Lists scheduled scans with optional filtering.
func (h *GitLabScheduleHandler) ListSchedules(c *gin.Context) {
	var req schedule.ListScheduledScansRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters: " + err.Error()})
		return
	}

	if req.Limit <= 0 {
		req.Limit = 50
	}

	schedules, total, err := h.scheduleStore.ListSchedules(c.Request.Context(), &req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list scheduled scans")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list schedules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"schedules": schedules,
		"total":     total,
		"limit":     req.Limit,
		"offset":    req.Offset,
	})
}

// UpdateSchedule PUT /api/admin/gitlab/schedules/:id
// Updates a scheduled scan.
func (h *GitLabScheduleHandler) UpdateSchedule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Schedule ID is required"})
		return
	}

	var req schedule.UpdateScheduledScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if err := h.scheduler.UpdateSchedule(c.Request.Context(), id, &req); err != nil {
		h.logger.WithError(err).Error("Failed to update scheduled scan")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schedule: " + err.Error()})
		return
	}

	// Return updated schedule
	schedule, _ := h.scheduleStore.GetSchedule(c.Request.Context(), id)
	c.JSON(http.StatusOK, schedule)
}

// DeleteSchedule DELETE /api/admin/gitlab/schedules/:id
// Deletes a scheduled scan.
func (h *GitLabScheduleHandler) DeleteSchedule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Schedule ID is required"})
		return
	}

	if err := h.scheduler.DeleteSchedule(c.Request.Context(), id); err != nil {
		h.logger.WithError(err).Error("Failed to delete scheduled scan")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete schedule: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted successfully"})
}

// TriggerSchedule POST /api/admin/gitlab/schedules/:id/trigger
// Manually triggers a scheduled scan.
func (h *GitLabScheduleHandler) TriggerSchedule(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Schedule ID is required"})
		return
	}

	history, err := h.scheduler.TriggerManual(c.Request.Context(), id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to trigger scheduled scan")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to trigger schedule: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Scan triggered successfully",
		"history": history,
	})
}

// GetScheduleHistory GET /api/admin/gitlab/schedules/:id/history
// Gets the execution history for a scheduled scan.
func (h *GitLabScheduleHandler) GetScheduleHistory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Schedule ID is required"})
		return
	}

	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		// Parse limit
	}
	if o := c.Query("offset"); o != "" {
		// Parse offset
	}

	history, total, err := h.scheduleStore.ListScanHistory(c.Request.Context(), id, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get schedule history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetSchedulerStatus GET /api/admin/gitlab/schedules/status
// Gets the scheduler status.
func (h *GitLabScheduleHandler) GetSchedulerStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"running":        h.scheduler.IsRunning(),
		"schedule_count": h.scheduler.GetScheduleCount(),
	})
}

