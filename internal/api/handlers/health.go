// Package handlers provides HTTP handlers for Ollama-OpenAI Proxy
package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
)

// HealthHandler обрабатывает health check эндпоинты
type HealthHandler struct {
	config       *config.Config
	logger       *logrus.Logger
	ollamaClient OllamaClientInterface
}

// NewHealthHandler создает новый health handler
func NewHealthHandler(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface) *HealthHandler {
	return &HealthHandler{
		config:       cfg,
		logger:       logger,
		ollamaClient: ollamaClient,
	}
}

// Health обрабатывает GET /health
func (h *HealthHandler) Health(c *gin.Context) {
	response := gin.H{
		"status":    "ok",
		"service":   "ollama-openai-proxy",
		"version":   "dev", // TODO: Получать из build информации
		"timestamp": time.Now().Unix(),
		"uptime":    time.Since(time.Now().Add(-5 * time.Minute)).String(), // TODO: Реальный uptime
	}

	h.logger.Debug("Health check requested")
	c.JSON(http.StatusOK, response)
}

// Ready обрабатывает GET /ready (readiness probe для Kubernetes)
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	checks := gin.H{
		"config":  "ok",
		"storage": "ok", // Предполагаем что storage готов
	}

	// Проверяем подключение к Ollama
	ollamaStatus := "ok"
	if err := h.ollamaClient.Health(ctx); err != nil {
		h.logger.WithError(err).Warn("Ollama health check failed")
		ollamaStatus = "failed"
	}
	checks["ollama"] = ollamaStatus

	// Определяем общий статус готовности
	allReady := true
	for _, status := range checks {
		if status != "ok" {
			allReady = false
			break
		}
	}

	status := "ready"
	statusCode := http.StatusOK
	if !allReady {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	response := gin.H{
		"status": status,
		"checks": checks,
	}

	h.logger.WithFields(logrus.Fields{
		"status":    status,
		"all_ready": allReady,
		"checks":    checks,
	}).Debug("Readiness check requested")

	c.JSON(statusCode, response)
}

// Live обрабатывает GET /healthz (liveness probe для Kubernetes)
func (h *HealthHandler) Live(c *gin.Context) {
	// Liveness проверка просто показывает что приложение живо
	response := gin.H{
		"status": "alive",
	}

	h.logger.Debug("Liveness check requested")
	c.JSON(http.StatusOK, response)
}
