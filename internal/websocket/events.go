package websocket

import (
	"encoding/json"
	"time"
)

// EventType тип события
type EventType string

const (
	// Server events
	EventTypeServerStats   EventType = "server_stats"
	EventTypeMetricsUpdate EventType = "metrics_update"

	// Request events (TUI-04)
	EventTypeRequestStart    EventType = "request_start"
	EventTypeRequestComplete EventType = "request_complete"
	EventTypeRequestError    EventType = "request_error"
	EventTypeRequestUpdate   EventType = "request_update"

	// API Key events
	EventTypeAPIKeyCreated EventType = "api_key_created"
	EventTypeAPIKeyDeleted EventType = "api_key_deleted"
	EventTypeAPIKeyUpdated EventType = "api_key_updated"

	// Model events
	EventTypeModelLoaded   EventType = "model_loaded"
	EventTypeModelUnloaded EventType = "model_unloaded"

	// Log events
	EventTypeNewLog EventType = "new_log"

	// Chat streaming events (WS-01 v1.10.2)
	EventTypeChatStreamStart EventType = "chat_stream_start"
	EventTypeChatStreamChunk EventType = "chat_stream_chunk"
	EventTypeChatStreamEnd   EventType = "chat_stream_end"
	EventTypeChatStreamError EventType = "chat_stream_error"

	// File processing events (WS-01 v1.10.2)
	EventTypeFileUploadStart    EventType = "file_upload_start"
	EventTypeFileUploadProgress EventType = "file_upload_progress"
	EventTypeFileUploadComplete EventType = "file_upload_complete"
	EventTypeFileUploadError    EventType = "file_upload_error"

	EventTypeFileProcessingStart    EventType = "file_processing_start"
	EventTypeFileProcessingProgress EventType = "file_processing_progress"
	EventTypeFileProcessingComplete EventType = "file_processing_complete"
	EventTypeFileProcessingError    EventType = "file_processing_error"

	// Notification events (WS-01 v1.10.2)
	EventTypeNotification EventType = "notification" // General system notification

	// System events
	EventTypeHeartbeat EventType = "heartbeat"
	EventTypeError     EventType = "error"
)

// Event представляет WebSocket событие
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// NewEvent создает новое событие
func NewEvent(eventType EventType, data map[string]interface{}) *Event {
	return &Event{
		Type:      eventType,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
}

// ToJSON конвертирует событие в JSON
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// EventBroadcaster отвечает за отправку событий через Hub
type EventBroadcaster struct {
	hub *Hub
}

// NewEventBroadcaster создает новый broadcaster
func NewEventBroadcaster(hub *Hub) *EventBroadcaster {
	return &EventBroadcaster{
		hub: hub,
	}
}

// BroadcastEvent отправляет событие всем клиентам
func (eb *EventBroadcaster) BroadcastEvent(event *Event) error {
	data, err := event.ToJSON()
	if err != nil {
		return err
	}

	eb.hub.Broadcast(data)
	return nil
}

// BroadcastServerStats отправляет обновление server stats
func (eb *EventBroadcaster) BroadcastServerStats(stats map[string]interface{}) error {
	event := NewEvent(EventTypeServerStats, stats)
	return eb.BroadcastEvent(event)
}

// BroadcastMetricsUpdate отправляет обновление метрик
func (eb *EventBroadcaster) BroadcastMetricsUpdate(metrics map[string]interface{}) error {
	event := NewEvent(EventTypeMetricsUpdate, metrics)
	return eb.BroadcastEvent(event)
}

// BroadcastRequestStart уведомляет о начале запроса (TUI-04)
func (eb *EventBroadcaster) BroadcastRequestStart(requestData map[string]interface{}) error {
	event := NewEvent(EventTypeRequestStart, requestData)
	return eb.BroadcastEvent(event)
}

// BroadcastRequestComplete уведомляет о завершении запроса (TUI-04)
func (eb *EventBroadcaster) BroadcastRequestComplete(requestData map[string]interface{}) error {
	event := NewEvent(EventTypeRequestComplete, requestData)
	return eb.BroadcastEvent(event)
}

// BroadcastRequestError уведомляет об ошибке запроса (TUI-04)
func (eb *EventBroadcaster) BroadcastRequestError(requestData map[string]interface{}) error {
	event := NewEvent(EventTypeRequestError, requestData)
	return eb.BroadcastEvent(event)
}

// BroadcastRequestUpdate уведомляет об обновлении запроса (TUI-04)
func (eb *EventBroadcaster) BroadcastRequestUpdate(requestData map[string]interface{}) error {
	event := NewEvent(EventTypeRequestUpdate, requestData)
	return eb.BroadcastEvent(event)
}

// BroadcastAPIKeyCreated уведомляет о создании API ключа
func (eb *EventBroadcaster) BroadcastAPIKeyCreated(keyData map[string]interface{}) error {
	event := NewEvent(EventTypeAPIKeyCreated, keyData)
	return eb.BroadcastEvent(event)
}

// BroadcastAPIKeyDeleted уведомляет об удалении API ключа
func (eb *EventBroadcaster) BroadcastAPIKeyDeleted(keyID string) error {
	event := NewEvent(EventTypeAPIKeyDeleted, map[string]interface{}{
		"key_id": keyID,
	})
	return eb.BroadcastEvent(event)
}

// BroadcastNewLog отправляет новую запись лога
func (eb *EventBroadcaster) BroadcastNewLog(logData map[string]interface{}) error {
	event := NewEvent(EventTypeNewLog, logData)
	return eb.BroadcastEvent(event)
}

// BroadcastHeartbeat отправляет heartbeat для проверки соединения
func (eb *EventBroadcaster) BroadcastHeartbeat() error {
	event := NewEvent(EventTypeHeartbeat, map[string]interface{}{
		"status": "alive",
	})
	return eb.BroadcastEvent(event)
}

// BroadcastError отправляет сообщение об ошибке
func (eb *EventBroadcaster) BroadcastError(errorMsg string, details map[string]interface{}) error {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["error"] = errorMsg

	event := NewEvent(EventTypeError, details)
	return eb.BroadcastEvent(event)
}

// ========================================
// Chat Streaming Events (WS-01 v1.10.2)
// ========================================

// BroadcastChatStreamStart уведомляет о начале chat streaming
func (eb *EventBroadcaster) BroadcastChatStreamStart(conversationID, requestID string, model string) error {
	event := NewEvent(EventTypeChatStreamStart, map[string]interface{}{
		"conversation_id": conversationID,
		"request_id":      requestID,
		"model":           model,
	})
	return eb.BroadcastEvent(event)
}

// BroadcastChatStreamChunk отправляет chunk streaming response
func (eb *EventBroadcaster) BroadcastChatStreamChunk(conversationID, requestID string, chunk map[string]interface{}) error {
	data := map[string]interface{}{
		"conversation_id": conversationID,
		"request_id":      requestID,
		"chunk":           chunk,
	}

	event := NewEvent(EventTypeChatStreamChunk, data)
	return eb.BroadcastEvent(event)
}

// BroadcastChatStreamEnd уведомляет о завершении streaming
func (eb *EventBroadcaster) BroadcastChatStreamEnd(conversationID, requestID string, messageID string, totalTokens int) error {
	event := NewEvent(EventTypeChatStreamEnd, map[string]interface{}{
		"conversation_id": conversationID,
		"request_id":      requestID,
		"message_id":      messageID,
		"total_tokens":    totalTokens,
	})
	return eb.BroadcastEvent(event)
}

// BroadcastChatStreamError уведомляет об ошибке streaming
func (eb *EventBroadcaster) BroadcastChatStreamError(conversationID, requestID string, errorMsg string) error {
	event := NewEvent(EventTypeChatStreamError, map[string]interface{}{
		"conversation_id": conversationID,
		"request_id":      requestID,
		"error":           errorMsg,
	})
	return eb.BroadcastEvent(event)
}

// ========================================
// File Processing Events (WS-01 v1.10.2)
// ========================================

// BroadcastFileUploadStart уведомляет о начале загрузки файла
func (eb *EventBroadcaster) BroadcastFileUploadStart(fileID, filename string, size int64) error {
	event := NewEvent(EventTypeFileUploadStart, map[string]interface{}{
		"file_id":  fileID,
		"filename": filename,
		"size":     size,
	})
	return eb.BroadcastEvent(event)
}

// BroadcastFileUploadProgress отправляет progress bar updates
func (eb *EventBroadcaster) BroadcastFileUploadProgress(fileID string, bytesUploaded, totalBytes int64, percent float64) error {
	event := NewEvent(EventTypeFileUploadProgress, map[string]interface{}{
		"file_id":        fileID,
		"bytes_uploaded": bytesUploaded,
		"total_bytes":    totalBytes,
		"percent":        percent,
	})
	return eb.BroadcastEvent(event)
}

// BroadcastFileUploadComplete уведомляет о завершении загрузки
func (eb *EventBroadcaster) BroadcastFileUploadComplete(fileID, filename string, downloadURL string) error {
	event := NewEvent(EventTypeFileUploadComplete, map[string]interface{}{
		"file_id":      fileID,
		"filename":     filename,
		"download_url": downloadURL,
	})
	return eb.BroadcastEvent(event)
}

// BroadcastFileUploadError уведомляет об ошибке загрузки
func (eb *EventBroadcaster) BroadcastFileUploadError(fileID, filename string, errorMsg string) error {
	event := NewEvent(EventTypeFileUploadError, map[string]interface{}{
		"file_id":  fileID,
		"filename": filename,
		"error":    errorMsg,
	})
	return eb.BroadcastEvent(event)
}

// BroadcastFileProcessingStart уведомляет о начале обработки файла
func (eb *EventBroadcaster) BroadcastFileProcessingStart(fileID, filename string, processingType string) error {
	event := NewEvent(EventTypeFileProcessingStart, map[string]interface{}{
		"file_id":         fileID,
		"filename":        filename,
		"processing_type": processingType, // "pdf_extract", "ocr", "csv_parse", etc.
	})
	return eb.BroadcastEvent(event)
}

// BroadcastFileProcessingProgress отправляет progress updates для обработки
func (eb *EventBroadcaster) BroadcastFileProcessingProgress(fileID string, stage string, percent float64) error {
	event := NewEvent(EventTypeFileProcessingProgress, map[string]interface{}{
		"file_id": fileID,
		"stage":   stage, // "extracting", "parsing", "analyzing", etc.
		"percent": percent,
	})
	return eb.BroadcastEvent(event)
}

// BroadcastFileProcessingComplete уведомляет о завершении обработки
func (eb *EventBroadcaster) BroadcastFileProcessingComplete(fileID string, result map[string]interface{}) error {
	data := map[string]interface{}{
		"file_id": fileID,
		"result":  result,
	}

	event := NewEvent(EventTypeFileProcessingComplete, data)
	return eb.BroadcastEvent(event)
}

// BroadcastFileProcessingError уведомляет об ошибке обработки
func (eb *EventBroadcaster) BroadcastFileProcessingError(fileID string, errorMsg string) error {
	event := NewEvent(EventTypeFileProcessingError, map[string]interface{}{
		"file_id": fileID,
		"error":   errorMsg,
	})
	return eb.BroadcastEvent(event)
}

// ========================================
// Notification Events (WS-01 v1.10.2)
// ========================================

// NotificationLevel уровень важности notification
type NotificationLevel string

const (
	NotificationLevelInfo    NotificationLevel = "info"
	NotificationLevelSuccess NotificationLevel = "success"
	NotificationLevelWarning NotificationLevel = "warning"
	NotificationLevelError   NotificationLevel = "error"
)

// BroadcastNotification отправляет system notification
func (eb *EventBroadcaster) BroadcastNotification(level NotificationLevel, title, message string, action map[string]interface{}) error {
	data := map[string]interface{}{
		"level":   level,
		"title":   title,
		"message": message,
	}

	if action != nil {
		data["action"] = action
	}

	event := NewEvent(EventTypeNotification, data)
	return eb.BroadcastEvent(event)
}
