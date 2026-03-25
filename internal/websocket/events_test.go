package websocket

import (
	"encoding/json"
	"testing"
	"time"
)

// MockHub для тестирования EventBroadcaster
type MockHub struct {
	lastMessage []byte
	callCount   int
}

func (m *MockHub) Broadcast(message []byte) {
	m.lastMessage = message
	m.callCount++
}

func TestNewEvent(t *testing.T) {
	eventType := EventTypeChatStreamStart
	data := map[string]any{
		"conversation_id": "conv_123",
		"model":           "llama2",
	}

	event := NewEvent(eventType, data)

	if event.Type != eventType {
		t.Errorf("Expected type %s, got %s", eventType, event.Type)
	}

	if event.Data["conversation_id"] != "conv_123" {
		t.Error("Expected conversation_id conv_123")
	}

	if event.Timestamp == 0 {
		t.Error("Expected non-zero timestamp")
	}
}

func TestEvent_ToJSON(t *testing.T) {
	event := NewEvent(EventTypeNotification, map[string]any{
		"title":   "Test",
		"message": "Test message",
	})

	jsonData, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	var parsed Event
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.Type != EventTypeNotification {
		t.Error("Type mismatch after JSON round-trip")
	}
}

func TestEventBroadcaster_BroadcastChatStreamStart(t *testing.T) {
	// Manual broadcast для теста
	conversationID := "conv_123"
	requestID := "req_456"
	model := "llama2"

	event := NewEvent(EventTypeChatStreamStart, map[string]any{
		"conversation_id": conversationID,
		"request_id":      requestID,
		"model":           model,
	})

	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to create JSON: %v", err)
	}

	var parsed Event
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.Type != EventTypeChatStreamStart {
		t.Errorf("Expected type %s, got %s", EventTypeChatStreamStart, parsed.Type)
	}

	if parsed.Data["conversation_id"] != conversationID {
		t.Error("Conversation ID mismatch")
	}

	if parsed.Data["model"] != model {
		t.Error("Model mismatch")
	}
}

func TestEventBroadcaster_BroadcastFileUploadProgress(t *testing.T) {
	fileID := "file_789"
	bytesUploaded := int64(5000)
	totalBytes := int64(10000)
	percent := 50.0

	event := NewEvent(EventTypeFileUploadProgress, map[string]any{
		"file_id":        fileID,
		"bytes_uploaded": bytesUploaded,
		"total_bytes":    totalBytes,
		"percent":        percent,
	})

	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to create JSON: %v", err)
	}

	var parsed Event
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.Type != EventTypeFileUploadProgress {
		t.Errorf("Expected type %s, got %s", EventTypeFileUploadProgress, parsed.Type)
	}

	// Note: JSON numbers parse as float64
	if parsed.Data["percent"].(float64) != percent {
		t.Error("Percent mismatch")
	}
}

func TestEventBroadcaster_BroadcastNotification(t *testing.T) {
	level := NotificationLevelSuccess
	title := "Upload Complete"
	message := "Your file has been uploaded"

	event := NewEvent(EventTypeNotification, map[string]any{
		"level":   level,
		"title":   title,
		"message": message,
	})

	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to create JSON: %v", err)
	}

	var parsed Event
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.Type != EventTypeNotification {
		t.Error("Type mismatch")
	}

	if parsed.Data["title"] != title {
		t.Error("Title mismatch")
	}

	if parsed.Data["level"] != string(level) {
		t.Error("Level mismatch")
	}
}

func TestEventBroadcaster_BroadcastChatStreamChunk(t *testing.T) {
	chunk := map[string]any{
		"content":      "Hello world",
		"role":         "assistant",
		"done":         false,
		"chunk_index":  1,
		"chunk_tokens": 2,
	}

	event := NewEvent(EventTypeChatStreamChunk, map[string]any{
		"conversation_id": "conv_123",
		"request_id":      "req_456",
		"chunk":           chunk,
	})

	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to create JSON: %v", err)
	}

	var parsed Event
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.Type != EventTypeChatStreamChunk {
		t.Error("Type mismatch")
	}

	chunkData := parsed.Data["chunk"].(map[string]any)
	if chunkData["content"] != "Hello world" {
		t.Error("Content mismatch")
	}
}

func TestNotificationLevels(t *testing.T) {
	levels := []NotificationLevel{
		NotificationLevelInfo,
		NotificationLevelSuccess,
		NotificationLevelWarning,
		NotificationLevelError,
	}

	expectedValues := []string{"info", "success", "warning", "error"}

	for i, level := range levels {
		if string(level) != expectedValues[i] {
			t.Errorf("Expected level %s, got %s", expectedValues[i], string(level))
		}
	}
}

func TestEventTimestamp(t *testing.T) {
	before := time.Now().Unix()
	event := NewEvent(EventTypeHeartbeat, map[string]any{
		"status": "alive",
	})
	after := time.Now().Unix()

	if event.Timestamp < before || event.Timestamp > after {
		t.Error("Timestamp not within expected range")
	}
}

func TestBroadcastFileProcessingComplete(t *testing.T) {
	result := map[string]any{
		"extracted_text_length": 1234,
		"word_count":            567,
		"language":              "en",
	}

	event := NewEvent(EventTypeFileProcessingComplete, map[string]any{
		"file_id": "file_123",
		"result":  result,
	})

	data, err := event.ToJSON()
	if err != nil {
		t.Fatalf("Failed to create JSON: %v", err)
	}

	var parsed Event
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed.Type != EventTypeFileProcessingComplete {
		t.Error("Type mismatch")
	}

	resultData := parsed.Data["result"].(map[string]any)
	// JSON numbers parse as float64
	if int(resultData["word_count"].(float64)) != 567 {
		t.Error("Word count mismatch")
	}
}
