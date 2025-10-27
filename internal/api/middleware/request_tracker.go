// Package middleware provides HTTP middleware for Ollama-OpenAI Proxy
package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"aigateway/internal/request"
	"aigateway/internal/websocket"
)

// RequestTracker middleware отслеживает все запросы для monitoring
func RequestTracker(storage *request.Storage, broadcaster *websocket.EventBroadcaster, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Пропускаем health checks, stats endpoints и статические файлы
		path := c.Request.URL.Path
		if path == "/health" || path == "/healthz" || path == "/ready" ||
			path == "/api/stats" || path == "/api/config" || path == "/metrics" ||
			path == "/ws" {
			c.Next()
			return
		}

		// Создаем RequestInfo
		reqInfo := &request.RequestInfo{
			ID:         generateRequestID(),
			Timestamp:  time.Now().Unix(),
			Method:     c.Request.Method,
			Endpoint:   path,
			Status:     request.StatusPending,
			RemoteAddr: c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
		}

		// Извлекаем информацию из request body (если это chat/completions или completions)
		if path == "/v1/chat/completions" || path == "/v1/completions" {
			extractRequestInfo(c, reqInfo)
		}

		// Извлекаем API Key информацию (если есть)
		if apiKey, exists := c.Get("api_key_id"); exists {
			if keyID, ok := apiKey.(string); ok {
				reqInfo.APIKeyID = keyID
			}
		}
		if apiKeyName, exists := c.Get("api_key_name"); exists {
			if keyName, ok := apiKeyName.(string); ok {
				reqInfo.APIKeyName = keyName
			}
		}

		// Сохраняем request info в context
		c.Set("request_info", reqInfo)

		// Добавляем в storage
		storage.Add(reqInfo)

		// Отправляем WebSocket событие о начале запроса
		if broadcaster != nil {
			reqData := requestInfoToMap(reqInfo)
			if err := broadcaster.BroadcastRequestStart(reqData); err != nil {
				logger.WithError(err).Debug("Failed to broadcast request start")
			}
		}

		// Создаем custom ResponseWriter для захвата status code
		blw := &statusRecorder{
			ResponseWriter: c.Writer,
			statusCode:     200, // default
		}
		c.Writer = blw

		startTime := time.Now()

		// Обрабатываем запрос
		c.Next()

		// Обновляем request info после обработки
		duration := time.Since(startTime)
		reqInfo.Duration = duration.Milliseconds()
		reqInfo.StatusCode = blw.statusCode

		// Определяем статус
		if blw.statusCode >= 200 && blw.statusCode < 300 {
			reqInfo.Status = request.StatusSuccess
		} else if blw.statusCode >= 400 {
			reqInfo.Status = request.StatusError

			// Пытаемся извлечь error message из context
			if err, exists := c.Get("error"); exists {
				if errStr, ok := err.(string); ok {
					reqInfo.ErrorMessage = errStr
				}
			}
		}

		// Обновляем в storage
		storage.Update(reqInfo)

		// Отправляем WebSocket событие о завершении запроса
		if broadcaster != nil {
			reqData := requestInfoToMap(reqInfo)

			if reqInfo.IsError() {
				if err := broadcaster.BroadcastRequestError(reqData); err != nil {
					logger.WithError(err).Debug("Failed to broadcast request error")
				}
			} else {
				if err := broadcaster.BroadcastRequestComplete(reqData); err != nil {
					logger.WithError(err).Debug("Failed to broadcast request complete")
				}
			}
		}

		logger.WithFields(logrus.Fields{
			"request_id": reqInfo.ID,
			"method":     reqInfo.Method,
			"endpoint":   reqInfo.Endpoint,
			"status":     reqInfo.Status,
			"duration":   reqInfo.Duration,
			"model":      reqInfo.Model,
		}).Debug("Request tracked")
	}
}

// extractRequestInfo извлекает информацию из request body
func extractRequestInfo(c *gin.Context, reqInfo *request.RequestInfo) {
	// Читаем body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return
	}

	// Восстанавливаем body для последующих обработчиков
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Парсим JSON
	var body map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		return
	}

	// Извлекаем model
	if model, ok := body["model"].(string); ok {
		reqInfo.Model = model
	}

	// Извлекаем messages (для chat/completions)
	if messages, ok := body["messages"].([]interface{}); ok {
		reqInfo.Messages = len(messages)
	}

	// Извлекаем tools
	if tools, ok := body["tools"].([]interface{}); ok {
		reqInfo.Tools = len(tools)
	}

	// Извлекаем stream
	if stream, ok := body["stream"].(bool); ok {
		reqInfo.Stream = stream
	}

	// Извлекаем temperature
	if temp, ok := body["temperature"].(float64); ok {
		reqInfo.Temperature = &temp
	}
}

// statusRecorder захватывает status code response
type statusRecorder struct {
	gin.ResponseWriter
	statusCode int
}

// WriteHeader захватывает status code
func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

// Write override для захвата status code
func (r *statusRecorder) Write(b []byte) (int, error) {
	return r.ResponseWriter.Write(b)
}

// generateRequestID генерирует уникальный ID для запроса
func generateRequestID() string {
	return "req_" + uuid.New().String()[:8]
}

// requestInfoToMap конвертирует RequestInfo в map для WebSocket
func requestInfoToMap(req *request.RequestInfo) map[string]interface{} {
	data := map[string]interface{}{
		"id":          req.ID,
		"timestamp":   req.Timestamp,
		"method":      req.Method,
		"endpoint":    req.Endpoint,
		"status":      req.Status,
		"duration_ms": req.Duration,
		"remote_addr": req.RemoteAddr,
	}

	// Добавляем опциональные поля если они заполнены
	if req.Model != "" {
		data["model"] = req.Model
	}
	if req.APIKeyID != "" {
		data["api_key_id"] = req.GetMaskedKey()
	}
	if req.APIKeyName != "" {
		data["api_key_name"] = req.APIKeyName
	}
	if req.StatusCode > 0 {
		data["status_code"] = req.StatusCode
	}
	if req.ErrorMessage != "" {
		data["error"] = req.ErrorMessage
	}
	if req.Messages > 0 {
		data["messages"] = req.Messages
	}
	if req.Tools > 0 {
		data["tools"] = req.Tools
	}
	if req.Stream {
		data["stream"] = req.Stream
	}
	if req.Temperature != nil {
		data["temperature"] = *req.Temperature
	}
	if req.TotalTokens > 0 {
		data["total_tokens"] = req.TotalTokens
	}

	return data
}

