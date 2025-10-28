# DESKTOP-03: WebSocket Streaming для Desktop Client

**Задача:** DESKTOP-03  
**Версия:** v2.4.3  
**Статус:** ✅ Завершено  
**Приоритет:** HIGH  
**Сложность:** Medium  
**Оценка:** 6-8 часов  
**Фактически:** ~4 часа  
**Дата завершения:** 2025-10-28

---

## 📋 Описание

Добавить **API key authentication** для WebSocket connections, чтобы desktop client мог использовать WebSocket для real-time chat streaming вместо SSE (Server-Sent Events).

### Текущая ситуация

✅ **Уже реализовано:**
- WebSocket infrastructure (`internal/websocket/hub.go`, `handler.go`)
- Ping/Pong heartbeat (60s interval)
- Broadcast система для всех клиентов
- SSE streaming для chat (`/v1/chat/completions` with `stream=true`)
- `EventBroadcaster` для WebSocket events

❌ **НЕ реализовано:**
- **API key authentication** для WebSocket connections
- WebSocket endpoint для chat streaming с API key auth
- Device-specific connection tracking
- Per-user/per-device message routing

### Зачем это нужно для Desktop?

1. **Persistent Connection**: Desktop app держит одно WebSocket соединение вместо множества HTTP requests
2. **Bi-directional**: Server может отправлять notifications в desktop app (model updates, system events)
3. **Lower Latency**: WebSocket имеет меньшую latency чем SSE
4. **Binary Support**: WebSocket поддерживает binary messages (для attachments)

---

## 🎯 Цели

1. **API Key Authentication** для WebSocket (`/ws/chat?token={api_key}`)
2. **Device Tracking** - связать WebSocket connection с device API key
3. **Per-User Routing** - отправлять messages только specific user
4. **Chat Streaming** через WebSocket (альтернатива SSE)
5. **Connection Management** - автоматический disconnect при revoke API key
6. **Backward Compatibility** - SSE продолжает работать для web clients

---

## 📐 Архитектура

### WebSocket Flow для Desktop

```
Desktop App                    AIGateway Server
    │                                 │
    │ 1. WS Connect                   │
    ├─────────────────────────────────>
    │   ws://server/ws/chat           │
    │   ?token={api_key}              │
    │                                 │
    │ 2. API Key Validation           │
    │   + Device Verification         │
    │   + Update last_seen_at         │
    │                                 │
    │ 3. WS Established (200)         │
    <─────────────────────────────────┤
    │                                 │
    │ 4. Send Chat Request            │
    ├─────────────────────────────────>
    │   {                             │
    │     "type": "chat",             │
    │     "payload": {                │
    │       "model": "qwen3-coder",   │
    │       "messages": [...]         │
    │     }                           │
    │   }                             │
    │                                 │
    │ 5. Streaming Chunks             │
    <─────────────────────────────────┤
    │   {"type": "chat_chunk", ...}   │
    <─────────────────────────────────┤
    │   {"type": "chat_chunk", ...}   │
    <─────────────────────────────────┤
    │   {"type": "chat_done", ...}    │
    │                                 │
    │ Ping/Pong (every 54s)           │
    <────────────────────────────────>
    │                                 │
```

### Database Schema (уже существует)

**api_keys table** (из DESKTOP-01):
```sql
CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL,
    
    -- Device fields (DESKTOP-01 v2.4.1)
    device_name TEXT,
    device_os TEXT,
    device_hostname TEXT,
    device_version TEXT,
    device_fingerprint TEXT,
    last_seen_at DATETIME,  -- <-- Обновляется при WebSocket activity
    auto_expire_at DATETIME,
    
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_api_keys_device_fingerprint ON api_keys(device_fingerprint);
CREATE INDEX idx_api_keys_last_seen ON api_keys(last_seen_at);
```

---

## 🔨 Реализация

### 1. WebSocket Handler с API Key Auth

**Файл:** `internal/websocket/handler.go` (UPDATE)

```go
package websocket

import (
	"net/http"
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	
	"aigateway/internal/models"
	"aigateway/internal/storage"
)

// Handler обрабатывает WebSocket connections
type Handler struct {
	hub    *Hub
	logger *logrus.Logger
	db     storage.Database // ДОБАВЛЕНО: для API key validation
}

// HandleChatWebSocket обрабатывает WebSocket upgrade с API key auth
// Endpoint: /ws/chat?token={api_key}
func (h *Handler) HandleChatWebSocket(c *gin.Context) {
	// 1. Authenticate via API key from query param
	apiKeyStr := c.Query("token")
	if apiKeyStr == "" {
		h.logger.Warn("WebSocket connection rejected: missing token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing API key token"})
		return
	}
	
	// 2. Validate API key
	apiKey, err := h.validateAPIKey(c.Request.Context(), apiKeyStr)
	if err != nil {
		h.logger.WithError(err).Warn("WebSocket connection rejected: invalid API key")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired API key"})
		return
	}
	
	// 3. Update last_seen_at for device tracking
	if apiKey.DeviceFingerprint != nil && *apiKey.DeviceFingerprint != "" {
		go h.updateDeviceLastSeen(apiKey.ID)
	}
	
	// 4. Upgrade HTTP to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade to WebSocket")
		return
	}
	
	// 5. Create authenticated client
	client := &Client{
		ID:   generateClientID(),
		Hub:  h.hub,
		Conn: &GorillaCon{Conn: conn},
		Send: make(chan []byte, 256),
		UserInfo: map[string]string{
			"ip":          c.ClientIP(),
			"user_agent":  c.Request.UserAgent(),
			"user_id":     apiKey.UserID,
			"api_key_id":  apiKey.ID,
			"device_name": stringValue(apiKey.DeviceName),
			"device_os":   stringValue(apiKey.DeviceOS),
		},
	}
	
	h.logger.WithFields(logrus.Fields{
		"client_id":   client.ID,
		"user_id":     apiKey.UserID,
		"device_name": stringValue(apiKey.DeviceName),
	}).Info("WebSocket client authenticated via API key")
	
	// 6. Register client
	h.hub.register <- client
	
	// 7. Start pumps
	go h.writePump(client)
	go h.readPump(client)
}

// validateAPIKey проверяет API key и возвращает его данные
func (h *Handler) validateAPIKey(ctx context.Context, apiKeyStr string) (*models.APIKey, error) {
	// TODO: Использовать cache для performance (APIKeyCache)
	apiKey, err := h.db.GetAPIKeyByKey(ctx, apiKeyStr)
	if err != nil {
		return nil, err
	}
	
	// Проверяем статус
	if apiKey.Status != models.APIKeyStatusActive {
		return nil, fmt.Errorf("API key is not active: %s", apiKey.Status)
	}
	
	// Проверяем expiration
	if apiKey.AutoExpireAt != nil && apiKey.AutoExpireAt.Before(time.Now()) {
		return nil, fmt.Errorf("API key has expired")
	}
	
	return apiKey, nil
}

// updateDeviceLastSeen обновляет last_seen_at для device API key
func (h *Handler) updateDeviceLastSeen(apiKeyID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := h.db.UpdateAPIKeyLastSeen(ctx, apiKeyID); err != nil {
		h.logger.WithError(err).WithField("api_key_id", apiKeyID).
			Debug("Failed to update device last_seen_at")
	}
}

// stringValue returns string value from pointer (for logging)
func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
```

### 2. Router Integration

**Файл:** `internal/api/router/router.go` (UPDATE)

```go
// setupWebSocketRoutes настраивает WebSocket routes
func (r *Router) setupWebSocketRoutes() {
	// Existing metrics WebSocket (no auth)
	r.engine.GET("/ws", r.wsHandler.HandleWebSocket)
	
	// NEW: Chat WebSocket with API key auth for Desktop
	r.engine.GET("/ws/chat", r.wsHandler.HandleChatWebSocket)
}
```

### 3. Hub Updates для Per-User Routing

**Файл:** `internal/websocket/hub.go` (UPDATE)

```go
// Hub управляет WebSocket клиентами
type Hub struct {
	// Existing fields...
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	
	// NEW: Per-user client tracking
	userClients map[string][]*Client // user_id -> clients
	mu          sync.RWMutex
	
	logger *logrus.Logger
	stats  HubStats
}

// registerClient регистрирует нового клиента
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	h.clients[client] = true
	
	// Track user's clients for targeted messaging
	if userID := client.UserInfo["user_id"]; userID != "" {
		if h.userClients == nil {
			h.userClients = make(map[string][]*Client)
		}
		h.userClients[userID] = append(h.userClients[userID], client)
	}
	
	h.mu.Unlock()
	
	h.stats.mu.Lock()
	h.stats.TotalConnections++
	h.stats.ActiveConnections++
	h.stats.mu.Unlock()
	
	h.logger.WithFields(logrus.Fields{
		"client_id":      client.ID,
		"user_id":        client.UserInfo["user_id"],
		"device_name":    client.UserInfo["device_name"],
		"active_clients": h.GetActiveClientsCount(),
	}).Info("WebSocket client connected")
}

// unregisterClient отключает клиента
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.Send)
		
		// Remove from user's clients
		if userID := client.UserInfo["user_id"]; userID != "" {
			userClients := h.userClients[userID]
			for i, c := range userClients {
				if c == client {
					h.userClients[userID] = append(userClients[:i], userClients[i+1:]...)
					break
				}
			}
			// Clean up empty slice
			if len(h.userClients[userID]) == 0 {
				delete(h.userClients, userID)
			}
		}
	}
	h.mu.Unlock()
	
	// ... rest of unregister logic
}

// SendToUser отправляет сообщение всем клиентам конкретного user
func (h *Hub) SendToUser(userID string, message []byte) {
	h.mu.RLock()
	clients := h.userClients[userID]
	h.mu.RUnlock()
	
	for _, client := range clients {
		select {
		case client.Send <- message:
			// Message sent successfully
		default:
			// Client buffer full, disconnect slow client
			go func(c *Client) {
				h.unregister <- c
			}(client)
		}
	}
}
```

### 4. Message Types для Chat

**Файл:** `internal/websocket/events.go` (UPDATE)

```go
// ChatMessage типы для WebSocket chat streaming
const (
	MessageTypeChatRequest  = "chat_request"   // Desktop -> Server
	MessageTypeChatChunk    = "chat_chunk"     // Server -> Desktop
	MessageTypeChatDone     = "chat_done"      // Server -> Desktop
	MessageTypeChatError    = "chat_error"     // Server -> Desktop
	MessageTypePing         = "ping"           // Heartbeat
	MessageTypePong         = "pong"           // Heartbeat response
)

// ChatRequestMessage запрос от desktop client
type ChatRequestMessage struct {
	Type      string                       `json:"type"` // "chat_request"
	RequestID string                       `json:"request_id"`
	Payload   models.ChatCompletionRequest `json:"payload"`
}

// ChatChunkMessage streaming chunk от сервера
type ChatChunkMessage struct {
	Type      string `json:"type"` // "chat_chunk"
	RequestID string `json:"request_id"`
	Payload   struct {
		Content string `json:"content"`
		Role    string `json:"role"`
		Done    bool   `json:"done"`
	} `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatDoneMessage финальное сообщение
type ChatDoneMessage struct {
	Type      string `json:"type"` // "chat_done"
	RequestID string `json:"request_id"`
	Payload   struct {
		MessageID    string `json:"message_id"`
		TotalTokens  int    `json:"total_tokens"`
		FinishReason string `json:"finish_reason"`
	} `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatErrorMessage сообщение об ошибке
type ChatErrorMessage struct {
	Type      string `json:"type"` // "chat_error"
	RequestID string `json:"request_id"`
	Payload   struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	} `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}
```

### 5. Chat Handler через WebSocket

**Файл:** `internal/websocket/chat_handler.go` (NEW)

```go
package websocket

import (
	"context"
	"encoding/json"
	"time"
	
	"github.com/sirupsen/logrus"
	
	"aigateway/internal/client/ollama"
	"aigateway/internal/config"
	"aigateway/internal/converter"
	"aigateway/internal/models"
)

// ChatHandler обрабатывает chat requests через WebSocket
type ChatHandler struct {
	config          *config.Config
	logger          *logrus.Logger
	ollamaClient    *ollama.Client
	converter       *converter.SimpleConverter
	streamConverter *converter.StreamConverter
	hub             *Hub
}

// NewChatHandler создает новый chat handler для WebSocket
func NewChatHandler(cfg *config.Config, logger *logrus.Logger, ollamaClient *ollama.Client, hub *Hub) *ChatHandler {
	return &ChatHandler{
		config:          cfg,
		logger:          logger,
		ollamaClient:    ollamaClient,
		converter:       converter.NewSimpleConverter(cfg, logger),
		streamConverter: converter.NewStreamConverter(cfg, logger),
		hub:             hub,
	}
}

// HandleChatRequest обрабатывает chat request от WebSocket клиента
func (h *ChatHandler) HandleChatRequest(client *Client, message []byte) {
	var req ChatRequestMessage
	if err := json.Unmarshal(message, &req); err != nil {
		h.sendError(client, "", "Invalid request format", "parse_error")
		return
	}
	
	h.logger.WithFields(logrus.Fields{
		"client_id":  client.ID,
		"user_id":    client.UserInfo["user_id"],
		"request_id": req.RequestID,
		"model":      req.Payload.Model,
	}).Info("Processing WebSocket chat request")
	
	// Process request asynchronously
	go h.processChatRequest(client, &req)
}

// processChatRequest обрабатывает chat request и отправляет streaming ответ
func (h *ChatHandler) processChatRequest(client *Client, req *ChatRequestMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	
	// Reset stream converter state
	h.streamConverter.Reset()
	
	// Convert to Ollama format
	ollamaReq, err := h.converter.ConvertChatRequest(&req.Payload)
	if err != nil {
		h.sendError(client, req.RequestID, "Request conversion failed: "+err.Error(), "conversion_error")
		return
	}
	
	// Create streaming client
	streamingClient := ollama.NewStreamingClient(h.ollamaClient)
	if streamingClient == nil {
		h.sendError(client, req.RequestID, "Failed to create streaming client", "internal_error")
		return
	}
	
	// Start streaming
	responseChan, errorChan := streamingClient.ChatCompletionStream(ctx, ollamaReq)
	
	// Process streaming response
	h.processStreamingResponse(client, req.RequestID, responseChan, errorChan)
}

// processStreamingResponse обрабатывает streaming ответ от Ollama
func (h *ChatHandler) processStreamingResponse(
	client *Client,
	requestID string,
	responseChan <-chan *ollama.ChatResponse,
	errorChan <-chan error,
) {
	totalTokens := 0
	chunksProcessed := 0
	
	for {
		select {
		case ollamaChunk, ok := <-responseChan:
			if !ok {
				// Streaming completed
				h.sendDone(client, requestID, totalTokens, "stop")
				return
			}
			
			// Send chunk to client
			chunkMsg := ChatChunkMessage{
				Type:      MessageTypeChatChunk,
				RequestID: requestID,
				Payload: struct {
					Content string `json:"content"`
					Role    string `json:"role"`
					Done    bool   `json:"done"`
				}{
					Content: ollamaChunk.Message.Content,
					Role:    ollamaChunk.Message.Role,
					Done:    ollamaChunk.Done,
				},
				Timestamp: time.Now(),
			}
			
			if err := h.sendMessage(client, &chunkMsg); err != nil {
				h.logger.WithError(err).Error("Failed to send chunk to WebSocket client")
				return
			}
			
			chunksProcessed++
			totalTokens += len(ollamaChunk.Message.Content) / 4 // Approximate
			
		case err, ok := <-errorChan:
			if !ok {
				return
			}
			h.logger.WithError(err).Error("Streaming error from Ollama")
			h.sendError(client, requestID, err.Error(), "streaming_error")
			return
		}
	}
}

// sendMessage отправляет JSON message через WebSocket
func (h *ChatHandler) sendMessage(client *Client, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	
	select {
	case client.Send <- data:
		return nil
	default:
		return fmt.Errorf("client send buffer full")
	}
}

// sendError отправляет error message клиенту
func (h *ChatHandler) sendError(client *Client, requestID, errorMsg, code string) {
	errMsg := ChatErrorMessage{
		Type:      MessageTypeChatError,
		RequestID: requestID,
		Payload: struct {
			Error string `json:"error"`
			Code  string `json:"code"`
		}{
			Error: errorMsg,
			Code:  code,
		},
		Timestamp: time.Now(),
	}
	
	h.sendMessage(client, &errMsg)
}

// sendDone отправляет done message клиенту
func (h *ChatHandler) sendDone(client *Client, requestID string, totalTokens int, finishReason string) {
	doneMsg := ChatDoneMessage{
		Type:      MessageTypeChatDone,
		RequestID: requestID,
		Payload: struct {
			MessageID    string `json:"message_id"`
			TotalTokens  int    `json:"total_tokens"`
			FinishReason string `json:"finish_reason"`
		}{
			MessageID:    fmt.Sprintf("msg_%s", requestID),
			TotalTokens:  totalTokens,
			FinishReason: finishReason,
		},
		Timestamp: time.Now(),
	}
	
	h.sendMessage(client, &doneMsg)
}
```

### 6. Update readPump для Chat Messages

**Файл:** `internal/websocket/handler.go` (UPDATE `handleClientMessage`)

```go
// handleClientMessage обрабатывает сообщения от клиента
func (h *Handler) handleClientMessage(client *Client, message []byte) {
	h.logger.WithFields(logrus.Fields{
		"client_id": client.ID,
		"message_preview": string(message[:min(100, len(message))]),
	}).Debug("Received message from WebSocket client")
	
	// Parse message type
	var baseMsg struct {
		Type string `json:"type"`
	}
	
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		h.logger.WithError(err).Error("Failed to parse WebSocket message")
		return
	}
	
	// Route message based on type
	switch baseMsg.Type {
	case MessageTypeChatRequest:
		h.chatHandler.HandleChatRequest(client, message)
		
	case MessageTypePing:
		// Respond with pong
		h.sendPong(client)
		
	default:
		h.logger.WithField("type", baseMsg.Type).Warn("Unknown WebSocket message type")
	}
}

// sendPong отправляет pong response
func (h *Handler) sendPong(client *Client) {
	pong := map[string]interface{}{
		"type":      MessageTypePong,
		"timestamp": time.Now(),
	}
	
	data, _ := json.Marshal(pong)
	select {
	case client.Send <- data:
	default:
		h.logger.Warn("Failed to send pong (buffer full)")
	}
}
```

---

## 🧪 Тестирование

### Unit Tests

**Файл:** `internal/websocket/handler_test.go` (UPDATE)

```go
package websocket

import (
	"context"
	"testing"
	"time"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	
	"aigateway/internal/models"
)

// MockDatabase для тестирования API key validation
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) GetAPIKeyByKey(ctx context.Context, key string) (*models.APIKey, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.APIKey), args.Error(1)
}

func (m *MockDatabase) UpdateAPIKeyLastSeen(ctx context.Context, apiKeyID string) error {
	args := m.Called(ctx, apiKeyID)
	return args.Error(0)
}

func TestValidateAPIKey_Success(t *testing.T) {
	mockDB := new(MockDatabase)
	handler := &Handler{
		db:     mockDB,
		logger: logrus.New(),
	}
	
	expectedKey := &models.APIKey{
		ID:     "key-123",
		UserID: "user-456",
		Status: models.APIKeyStatusActive,
		DeviceFingerprint: stringPtr("device-fingerprint"),
	}
	
	mockDB.On("GetAPIKeyByKey", mock.Anything, "valid-key").Return(expectedKey, nil)
	
	apiKey, err := handler.validateAPIKey(context.Background(), "valid-key")
	
	assert.NoError(t, err)
	assert.Equal(t, expectedKey.ID, apiKey.ID)
	assert.Equal(t, expectedKey.UserID, apiKey.UserID)
	mockDB.AssertExpectations(t)
}

func TestValidateAPIKey_Expired(t *testing.T) {
	mockDB := new(MockDatabase)
	handler := &Handler{
		db:     mockDB,
		logger: logrus.New(),
	}
	
	expiredTime := time.Now().Add(-24 * time.Hour)
	expiredKey := &models.APIKey{
		ID:           "key-123",
		Status:       models.APIKeyStatusActive,
		AutoExpireAt: &expiredTime,
	}
	
	mockDB.On("GetAPIKeyByKey", mock.Anything, "expired-key").Return(expiredKey, nil)
	
	_, err := handler.validateAPIKey(context.Background(), "expired-key")
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
	mockDB.AssertExpectations(t)
}

func TestValidateAPIKey_Inactive(t *testing.T) {
	mockDB := new(MockDatabase)
	handler := &Handler{
		db:     mockDB,
		logger: logrus.New(),
	}
	
	inactiveKey := &models.APIKey{
		ID:     "key-123",
		Status: models.APIKeyStatusRevoked,
	}
	
	mockDB.On("GetAPIKeyByKey", mock.Anything, "inactive-key").Return(inactiveKey, nil)
	
	_, err := handler.validateAPIKey(context.Background(), "inactive-key")
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
	mockDB.AssertExpectations(t)
}

func stringPtr(s string) *string {
	return &s
}
```

### Integration Test (manual)

**Desktop Client Example (Go):**

```go
package main

import (
	"encoding/json"
	"log"
	"time"
	
	"github.com/gorilla/websocket"
)

func main() {
	apiKey := "your-device-api-key-here"
	wsURL := "ws://localhost:8080/ws/chat?token=" + apiKey
	
	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer conn.Close()
	
	log.Println("✅ WebSocket connected")
	
	// Send chat request
	chatRequest := map[string]interface{}{
		"type":       "chat_request",
		"request_id": "req-001",
		"payload": map[string]interface{}{
			"model": "qwen3-coder",
			"messages": []map[string]string{
				{"role": "user", "content": "Hello, how are you?"},
			},
			"stream": true,
		},
	}
	
	if err := conn.WriteJSON(chatRequest); err != nil {
		log.Fatal("Failed to send request:", err)
	}
	
	log.Println("📤 Chat request sent")
	
	// Read streaming responses
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("Connection closed:", err)
			break
		}
		
		msgType := msg["type"].(string)
		
		switch msgType {
		case "chat_chunk":
			payload := msg["payload"].(map[string]interface{})
			content := payload["content"].(string)
			log.Printf("📥 Chunk: %s", content)
			
		case "chat_done":
			payload := msg["payload"].(map[string]interface{})
			tokens := payload["total_tokens"].(float64)
			log.Printf("✅ Done! Total tokens: %.0f", tokens)
			return
			
		case "chat_error":
			payload := msg["payload"].(map[string]interface{})
			errMsg := payload["error"].(string)
			log.Printf("❌ Error: %s", errMsg)
			return
			
		case "pong":
			log.Println("🏓 Pong received")
		}
	}
}
```

---

## 📝 Acceptance Criteria

- [ ] `GET /ws/chat?token={api_key}` endpoint работает
- [ ] API key validation работает (active/expired/revoked checks)
- [ ] Device `last_seen_at` обновляется при WebSocket activity
- [ ] Chat request/response через WebSocket работает
- [ ] Streaming chunks отправляются корректно
- [ ] Ping/Pong heartbeat работает (54s interval)
- [ ] Reconnection logic работает на клиенте
- [ ] Per-user message routing работает (SendToUser)
- [ ] Revoked API keys отключаются автоматически
- [ ] Unit tests для API key validation (3+ tests)
- [ ] Integration test (manual desktop client)
- [ ] Documentation обновлена

---

## 📚 Документация

### Desktop Client Integration

**Wails App WebSocket Client:**

```javascript
// frontend/src/websocket.js

class WSClient {
  constructor(apiKey) {
    this.apiKey = apiKey;
    this.ws = null;
    this.reconnectAttempts = 0;
    this.maxReconnects = 5;
    this.handlers = new Map();
  }
  
  connect() {
    const wsURL = `ws://localhost:8080/ws/chat?token=${this.apiKey}`;
    
    this.ws = new WebSocket(wsURL);
    
    this.ws.onopen = () => {
      console.log('✅ WebSocket connected');
      this.reconnectAttempts = 0;
    };
    
    this.ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);
      this.handleMessage(msg);
    };
    
    this.ws.onerror = (error) => {
      console.error('❌ WebSocket error:', error);
    };
    
    this.ws.onclose = () => {
      console.log('🔌 WebSocket disconnected');
      this.reconnect();
    };
  }
  
  handleMessage(msg) {
    const handlers = this.handlers.get(msg.type) || [];
    handlers.forEach(h => h(msg.payload));
  }
  
  on(type, handler) {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, []);
    }
    this.handlers.get(type).push(handler);
  }
  
  sendChatRequest(model, messages) {
    const msg = {
      type: 'chat_request',
      request_id: `req-${Date.now()}`,
      payload: {
        model,
        messages,
        stream: true,
      },
    };
    
    this.ws.send(JSON.stringify(msg));
  }
  
  reconnect() {
    if (this.reconnectAttempts >= this.maxReconnects) {
      console.error('❌ Max reconnect attempts reached');
      return;
    }
    
    this.reconnectAttempts++;
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000);
    
    setTimeout(() => {
      console.log(`🔄 Reconnecting... (attempt ${this.reconnectAttempts})`);
      this.connect();
    }, delay);
  }
  
  disconnect() {
    if (this.ws) {
      this.ws.close();
    }
  }
}

// Usage
const wsClient = new WSClient(apiKey);

wsClient.on('chat_chunk', (payload) => {
  console.log('Chunk:', payload.content);
  // Append to UI
});

wsClient.on('chat_done', (payload) => {
  console.log('Done! Tokens:', payload.total_tokens);
});

wsClient.on('chat_error', (payload) => {
  console.error('Error:', payload.error);
});

wsClient.connect();
wsClient.sendChatRequest('qwen3-coder', [
  { role: 'user', content: 'Hello!' }
]);
```

---

## 🔗 Связанные задачи

- **DESKTOP-01** (v2.4.1): Auto-Generated API Keys → используется для WebSocket auth
- **DESKTOP-02** (v2.4.2): Device Management → device tracking через WebSocket
- **WS-01** (v1.10.2): WebSocket Real-time Updates → базовая инфраструктура

---

## 🎯 Следующие шаги (Future)

- [ ] **Binary Message Support** для attachments (images, files)
- [ ] **Multiple Model Streaming** (параллельные запросы к разным моделям)
- [ ] **Server-to-Client Notifications** (model updates, system events)
- [ ] **Connection Limits** per user/device (rate limiting)
- [ ] **Redis Pub/Sub** для multi-instance scaling
- [ ] **WebSocket Metrics** (Prometheus)

---

**Дата создания:** 2025-10-28  
**Последнее обновление:** 2025-10-28

