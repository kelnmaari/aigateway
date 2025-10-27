// Package handlers provides HTTP handlers for changelog endpoints
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/storage"
	"ollama-openai-proxy/internal/version"
)

// ChangelogHandler обрабатывает changelog endpoints
type ChangelogHandler struct {
	config *config.Config
	db     storage.Database
	logger *logrus.Logger
}

// NewChangelogHandler создает новый Changelog Handler
func NewChangelogHandler(cfg *config.Config, db storage.Database, logger *logrus.Logger) *ChangelogHandler {
	return &ChangelogHandler{
		config: cfg,
		db:     db,
		logger: logger,
	}
}

// GetSystemInfo возвращает информацию о системе
// GET /api/system/info
func (h *ChangelogHandler) GetSystemInfo(c *gin.Context) {
	h.logger.Debug("System info requested")

	c.JSON(http.StatusOK, gin.H{
		"version":     version.Version,
		"git_commit":  version.GitCommit,
		"build_date":  version.BuildDate,
		"go_version":  version.GoVersion,
		"rag_enabled": h.config.RAG.Enabled, // v1.13.0: RAG system status
	})
}

// GetChangelogs возвращает список всех версий с описанием изменений
// GET /api/system/changelogs
func (h *ChangelogHandler) GetChangelogs(c *gin.Context) {
	h.logger.Debug("Changelogs list requested")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	changelogs, err := h.db.ListChangelogs(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get changelogs")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve changelogs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"changelogs": changelogs,
		"total":      len(changelogs),
	})
}

// GetChangelog возвращает изменения для конкретной версии
// GET /api/system/changelogs/:version
func (h *ChangelogHandler) GetChangelog(c *gin.Context) {
	version := c.Param("version")

	h.logger.WithField("version", version).Debug("Changelog requested")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	changelog, err := h.db.GetChangelog(ctx, version)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get changelog")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Changelog not found",
		})
		return
	}

	c.JSON(http.StatusOK, changelog)
}
