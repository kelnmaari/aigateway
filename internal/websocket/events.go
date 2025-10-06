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
