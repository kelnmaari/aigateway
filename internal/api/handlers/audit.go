// Package handlers provides audit log HTTP handlers
// Version: 1.11.4+ (Enterprise Suite - Enhanced Audit Logging)
package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/config"
	"aigateway/internal/storage"
)

// AuditHandler handles audit log HTTP requests
type AuditHandler struct {
	config *config.Config
	db     storage.Database
	logger *logrus.Logger
}

// NewAuditHandler creates a new audit handler
func NewAuditHandler(cfg *config.Config, db storage.Database, logger *logrus.Logger) *AuditHandler {
	return &AuditHandler{
		config: cfg,
		db:     db,
		logger: logger,
	}
}

// GetAuditEvents returns audit events with filters
// GET /api/admin/audit
func (h *AuditHandler) GetAuditEvents(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query parameters
	filters := storage.AuditFilters{
		EventType: c.Query("event_type"),
		ActorID:   c.Query("actor_id"),
		Resource:  c.Query("resource"),
		Severity:  c.Query("severity"),
		Status:    c.Query("status"),
		Limit:     100, // Default limit
		Offset:    0,
	}

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			if limit > 1000 {
				limit = 1000 // Maximum limit
			}
			filters.Limit = limit
		}
	}

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filters.Offset = offset
		}
	}

	// Parse date filters
	if fromDateStr := c.Query("from_date"); fromDateStr != "" {
		if fromDate, err := time.Parse("2006-01-02", fromDateStr); err == nil {
			filters.FromDate = fromDate
		}
	}

	if toDateStr := c.Query("to_date"); toDateStr != "" {
		if toDate, err := time.Parse("2006-01-02", toDateStr); err == nil {
			// Set to end of day
			filters.ToDate = toDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
	}

	h.logger.WithField("filters", filters).Debug("Getting audit events")

	// Get audit events
	events, total, err := h.db.GetAuditEvents(ctx, filters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get audit events")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve audit events",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  total,
		"limit":  filters.Limit,
		"offset": filters.Offset,
	})
}

// ExportAuditEvents exports audit events to CSV
// GET /api/admin/audit/export
func (h *AuditHandler) ExportAuditEvents(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query parameters (same as GetAuditEvents but without pagination)
	filters := storage.AuditFilters{
		EventType: c.Query("event_type"),
		ActorID:   c.Query("actor_id"),
		Resource:  c.Query("resource"),
		Severity:  c.Query("severity"),
		Status:    c.Query("status"),
		Limit:     10000, // Large limit for export
		Offset:    0,
	}

	// Parse date filters
	if fromDateStr := c.Query("from_date"); fromDateStr != "" {
		if fromDate, err := time.Parse("2006-01-02", fromDateStr); err == nil {
			filters.FromDate = fromDate
		}
	}

	if toDateStr := c.Query("to_date"); toDateStr != "" {
		if toDate, err := time.Parse("2006-01-02", toDateStr); err == nil {
			filters.ToDate = toDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}
	}

	h.logger.WithField("filters", filters).Info("Exporting audit events to CSV")

	// Get audit events
	events, _, err := h.db.GetAuditEvents(ctx, filters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get audit events for export")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to export audit events",
		})
		return
	}

	// Set CSV headers
	filename := fmt.Sprintf("audit_log_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	// Write CSV
	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header
	header := []string{
		"Timestamp",
		"Event Type",
		"Severity",
		"Actor ID",
		"Actor Type",
		"Target ID",
		"Target Type",
		"Action",
		"Resource",
		"Status",
		"Error Message",
		"IP Address",
		"User Agent",
		"Metadata",
	}
	if err := writer.Write(header); err != nil {
		h.logger.WithError(err).Error("Failed to write CSV header")
		return
	}

	// Write data rows
	for _, event := range events {
		targetID := ""
		if event.TargetID != nil {
			targetID = *event.TargetID
		}

		targetType := ""
		if event.TargetType != nil {
			targetType = *event.TargetType
		}

		errorMsg := ""
		if event.ErrorMsg != nil {
			errorMsg = *event.ErrorMsg
		}

		userAgent := ""
		if event.UserAgent != nil {
			userAgent = *event.UserAgent
		}

		// Serialize metadata to JSON string
		metadataStr := "{}"
		if event.Metadata != nil {
			if len(event.Metadata) > 0 {
				if metadataBytes, err := json.Marshal(event.Metadata); err == nil {
					metadataStr = string(metadataBytes)
				}
			}
		}

		row := []string{
			event.Timestamp.Format(time.RFC3339),
			event.EventType,
			event.Severity,
			event.ActorID,
			event.ActorType,
			targetID,
			targetType,
			event.Action,
			event.Resource,
			event.Status,
			errorMsg,
			event.IPAddress,
			userAgent,
			metadataStr,
		}

		if err := writer.Write(row); err != nil {
			h.logger.WithError(err).Error("Failed to write CSV row")
			return
		}
	}

	h.logger.WithField("count", len(events)).Info("Audit events exported to CSV")
}

// GetAuditStats returns audit statistics (for dashboard)
// GET /api/admin/audit/stats
func (h *AuditHandler) GetAuditStats(c *gin.Context) {
	ctx := c.Request.Context()

	// Get stats for last 24 hours
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// Count by severity
	criticalFilters := storage.AuditFilters{
		Severity: "critical",
		FromDate: yesterday,
		ToDate:   now,
		Limit:    10000,
	}
	_, criticalCount, err := h.db.GetAuditEvents(ctx, criticalFilters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get critical audit count")
		criticalCount = 0
	}

	warningFilters := storage.AuditFilters{
		Severity: "warning",
		FromDate: yesterday,
		ToDate:   now,
		Limit:    10000,
	}
	_, warningCount, err := h.db.GetAuditEvents(ctx, warningFilters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get warning audit count")
		warningCount = 0
	}

	infoFilters := storage.AuditFilters{
		Severity: "info",
		FromDate: yesterday,
		ToDate:   now,
		Limit:    10000,
	}
	_, infoCount, err := h.db.GetAuditEvents(ctx, infoFilters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get info audit count")
		infoCount = 0
	}

	// Count failed logins
	failedLoginFilters := storage.AuditFilters{
		EventType: "LOGIN_FAILED",
		FromDate:  yesterday,
		ToDate:    now,
		Limit:     10000,
	}
	_, failedLoginCount, err := h.db.GetAuditEvents(ctx, failedLoginFilters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get failed login count")
		failedLoginCount = 0
	}

	// Count permission denied events
	permissionDeniedFilters := storage.AuditFilters{
		EventType: "PERMISSION_DENIED",
		FromDate:  yesterday,
		ToDate:    now,
		Limit:     10000,
	}
	_, permissionDeniedCount, err := h.db.GetAuditEvents(ctx, permissionDeniedFilters)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get permission denied count")
		permissionDeniedCount = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"period": "24h",
		"stats": gin.H{
			"by_severity": gin.H{
				"critical": criticalCount,
				"warning":  warningCount,
				"info":     infoCount,
			},
			"security_events": gin.H{
				"failed_logins":      failedLoginCount,
				"permission_denied":  permissionDeniedCount,
			},
			"total": criticalCount + warningCount + infoCount,
		},
	})
}


