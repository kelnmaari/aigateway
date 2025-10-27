// Package handlers provides HTTP handlers for conversation export/import
// Version 1.12.3+: Conversation Export/Import (EXPORT-01)
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/services/export"
	"aigateway/internal/storage"
)

// ConversationExportHandler обрабатывает export/import endpoints
type ConversationExportHandler struct {
	logger   *logrus.Logger
	exporter *export.ConversationExporter
	importer *export.ConversationImporter
}

// NewConversationExportHandler создает новый handler
func NewConversationExportHandler(db storage.Database, logger *logrus.Logger) *ConversationExportHandler {
	return &ConversationExportHandler{
		logger:   logger,
		exporter: export.NewConversationExporter(db, logger),
		importer: export.NewConversationImporter(db, logger),
	}
}

// ExportConversation godoc
// @Summary      Export conversation
// @Description  Export a conversation to JSON, Markdown, or Text format
// @Tags         conversations
// @Security     Bearer
// @Param        id     path     string  true   "Conversation ID"
// @Param        format query    string  false  "Export format: json, markdown, text" default(json)
// @Produce      json
// @Produce      text/markdown
// @Produce      text/plain
// @Success      200 {object} models.ConversationExport
// @Success      200 {string} string "Markdown or Text"
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /api/conversations/{id}/export [get]
func (h *ConversationExportHandler) ExportConversation(c *gin.Context) {
	convID := c.Param("id")
	format := c.DefaultQuery("format", "json")

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "User ID not found in context",
				"type":    "authentication_error",
			},
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Invalid user ID type",
				"type":    "internal_error",
			},
		})
		return
	}

	ctx := c.Request.Context()

	switch models.ExportFormat(format) {
	case models.ExportFormatJSON:
		jsonData, err := h.exporter.ExportToJSON(ctx, convID, userIDStr)
		if err != nil {
			h.logger.WithError(err).Error("Failed to export conversation to JSON")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "Failed to export conversation",
					"type":    "export_error",
					"details": err.Error(),
				},
			})
			return
		}

		c.Header("Content-Disposition", "attachment; filename=conversation_"+convID+".json")
		c.Data(http.StatusOK, "application/json", []byte(jsonData))

	case models.ExportFormatMarkdown:
		mdData, err := h.exporter.ExportToMarkdown(ctx, convID, userIDStr)
		if err != nil {
			h.logger.WithError(err).Error("Failed to export conversation to Markdown")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "Failed to export conversation",
					"type":    "export_error",
					"details": err.Error(),
				},
			})
			return
		}

		c.Header("Content-Disposition", "attachment; filename=conversation_"+convID+".md")
		c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(mdData))

	case models.ExportFormatText:
		textData, err := h.exporter.ExportToText(ctx, convID, userIDStr)
		if err != nil {
			h.logger.WithError(err).Error("Failed to export conversation to Text")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"message": "Failed to export conversation",
					"type":    "export_error",
					"details": err.Error(),
				},
			})
			return
		}

		c.Header("Content-Disposition", "attachment; filename=conversation_"+convID+".txt")
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(textData))

	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid format. Supported: json, markdown, text",
				"type":    "invalid_request_error",
			},
		})
	}
}

// BulkExportConversations godoc
// @Summary      Bulk export conversations
// @Description  Export multiple conversations at once
// @Tags         conversations
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        request body models.BulkExportRequest true "Bulk export request"
// @Success      200 {object} models.BulkExportResult
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /api/conversations/bulk-export [post]
func (h *ConversationExportHandler) BulkExportConversations(c *gin.Context) {
	var req models.BulkExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request body",
				"type":    "invalid_request_error",
				"details": err.Error(),
			},
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "User ID not found in context",
				"type":    "authentication_error",
			},
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Invalid user ID type",
				"type":    "internal_error",
			},
		})
		return
	}

	ctx := c.Request.Context()

	result, err := h.exporter.BulkExport(ctx, req.ConversationIDs, userIDStr, req.Format)
	if err != nil {
		h.logger.WithError(err).Error("Failed to bulk export conversations")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to bulk export conversations",
				"type":    "export_error",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ImportConversation godoc
// @Summary      Import conversation
// @Description  Import a conversation from JSON format
// @Tags         conversations
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        request body models.ImportConversationRequest true "Import request"
// @Success      200 {object} models.ImportResult
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /api/conversations/import [post]
func (h *ConversationExportHandler) ImportConversation(c *gin.Context) {
	var req models.ImportConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request body",
				"type":    "invalid_request_error",
				"details": err.Error(),
			},
		})
		return
	}

	// Get user ID and tenant ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "User ID not found in context",
				"type":    "authentication_error",
			},
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Invalid user ID type",
				"type":    "internal_error",
			},
		})
		return
	}

	tenantIDValue, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantIDValue.(string)

	ctx := c.Request.Context()

	var result *models.ImportResult
	var err error

	switch req.Format {
	case models.ExportFormatJSON:
		result, err = h.importer.ImportFromJSON(ctx, req.Data, userIDStr, tenantIDStr, req.Options)
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Unsupported import format. Only JSON is supported.",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	if err != nil {
		h.logger.WithError(err).Error("Failed to import conversation")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to import conversation",
				"type":    "import_error",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, result)
}


