package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/storage"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512 KB
)

const (
	// Message types for WebSocket communication
	MessageTypeChatRequest = "chat_request"
	MessageTypePing        = "ping"
	MessageTypePong        = "pong"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Разрешаем все origins (в production нужно ограничить)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// GorillaCon адаптер для gorilla/websocket
type GorillaCon struct {
	*websocket.Conn
}

// WriteMessage implements Connection interface
func (g *GorillaCon) WriteMessage(messageType int, data []byte) error {
	return g.Conn.WriteMessage(messageType, data)
}

// ReadMessage implements Connection interface
func (g *GorillaCon) ReadMessage() (int, []byte, error) {
	return g.Conn.ReadMessage()
}

// Close implements Connection interface
func (g *GorillaCon) Close() error {
	return g.Conn.Close()
}

// SetReadDeadline implements Connection interface
func (g *GorillaCon) SetReadDeadline(t time.Time) error {
	return g.Conn.SetReadDeadline(t)
}

// SetWriteDeadline implements Connection interface
func (g *GorillaCon) SetWriteDeadline(t time.Time) error {
	return g.Conn.SetWriteDeadline(t)
}

// Handler обрабатывает WebSocket connections
type Handler struct {
	hub         *Hub
	logger      *logrus.Logger
	db          storage.Database // For API key validation
	chatHandler ChatHandlerInterface
}

// ChatHandlerInterface интерфейс для обработки chat requests
type ChatHandlerInterface interface {
	HandleChatRequest(client *Client, message []byte)
	
	// v2.5.4+: Tool RPC methods
	HandleToolExecutionResponse(client *Client, message []byte)
	CleanupToolRPCClient(clientID string)
}

// NewHandler создает новый WebSocket handler
func NewHandler(hub *Hub, logger *logrus.Logger) *Handler {
	if logger == nil {
		logger = logrus.New()
	}

	return &Handler{
		hub:    hub,
		logger: logger,
	}
}

// SetDatabase устанавливает database для API key validation (DESKTOP-03)
func (h *Handler) SetDatabase(db storage.Database) {
	h.db = db
}

// SetChatHandler устанавливает chat handler (DESKTOP-03)
func (h *Handler) SetChatHandler(chatHandler ChatHandlerInterface) {
	h.chatHandler = chatHandler
}

// HandleWebSocket обрабатывает WebSocket upgrade и connection (для metrics, без auth)
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.WithError(err).Error("Failed to upgrade to WebSocket")
		return
	}

	// Создаем клиента
	client := &Client{
		ID:   generateClientID(),
		Hub:  h.hub,
		Conn: &GorillaCon{Conn: conn},
		Send: make(chan []byte, 256),
		UserInfo: map[string]string{
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		},
	}

	// Регистрируем клиента в hub
	h.hub.register <- client

	// Запускаем reader и writer в отдельных горутинах
	go h.writePump(client)
	go h.readPump(client)
}

// HandleChatWebSocket обрабатывает WebSocket upgrade с API key auth (DESKTOP-03)
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
			"user_id":     stringValue(apiKey.UserID),
			"api_key_id":  apiKey.ID,
			"device_name": stringValue(apiKey.DeviceName),
			"device_os":   stringValue(apiKey.DeviceOS),
		},
	}

	h.logger.WithFields(logrus.Fields{
		"client_id":   client.ID,
		"user_id":     stringValue(apiKey.UserID),
		"device_name": stringValue(apiKey.DeviceName),
	}).Info("WebSocket client authenticated via API key")

	// 6. Register client
	h.hub.register <- client

	// 7. Start pumps
	go h.writePump(client)
	go h.readPump(client)
}

// validateAPIKey проверяет API key и возвращает его данные
func (h *Handler) validateAPIKey(ctx context.Context, plainKey string) (*models.APIKey, error) {
	if h.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	// Проверка формата ключа
	if !models.IsValidAPIKeyFormat(plainKey) {
		return nil, fmt.Errorf("invalid API key format")
	}

	// Получаем все ключи и проверяем каждый с помощью bcrypt
	// (мы не можем искать по хешу, т.к. bcrypt генерирует разные хеши для одного значения)
	allKeys, err := h.db.ListAPIKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	// Ищем ключ, проверяя plain key против каждого хеша
	var matchedKey *models.APIKey
	for _, key := range allKeys {
		if key.VerifyKey(plainKey) {
			matchedKey = key
			break
		}
	}

	if matchedKey == nil {
		return nil, fmt.Errorf("API key not found")
	}

	// Check status
	if matchedKey.Status != models.APIKeyStatusActive {
		return nil, fmt.Errorf("API key is not active: %s", matchedKey.Status)
	}

	// Check expiration
	if matchedKey.AutoExpireAt != nil && matchedKey.AutoExpireAt.Before(time.Now()) {
		return nil, fmt.Errorf("API key has expired")
	}

	return matchedKey, nil
}

// updateDeviceLastSeen обновляет last_seen_at для device API key
func (h *Handler) updateDeviceLastSeen(apiKeyID string) {
	if h.db == nil {
		return
	}

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

// readPump читает сообщения от клиента
func (h *Handler) readPump(client *Client) {
	defer func() {
		// v2.5.4+: Cleanup Tool RPC Client before unregister
		if h.chatHandler != nil {
			h.chatHandler.CleanupToolRPCClient(client.ID)
		}
		
		h.hub.unregister <- client
		client.Conn.Close()
	}()

	client.Conn.SetReadDeadline(time.Now().Add(pongWait))

	for {
		messageType, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.WithError(err).WithField("client_id", client.ID).Error("WebSocket read error")
			}
			break
		}

		// Обновляем deadline при получении pong
		client.Conn.SetReadDeadline(time.Now().Add(pongWait))

		// Обрабатываем сообщение от клиента
		if messageType == websocket.TextMessage {
			h.handleClientMessage(client, message)
		}

		// Обновляем статистику
		client.Hub.stats.mu.Lock()
		client.Hub.stats.MessagesReceived++
		client.Hub.stats.mu.Unlock()
	}
}

// writePump отправляет сообщения клиенту
func (h *Handler) writePump(client *Client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub закрыл канал
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			err := client.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				h.logger.WithError(err).WithField("client_id", client.ID).Error("WebSocket write error")
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleClientMessage обрабатывает сообщения от клиента
func (h *Handler) handleClientMessage(client *Client, message []byte) {
	// Limit message preview for logging
	messagePreview := string(message)
	if len(messagePreview) > 100 {
		messagePreview = messagePreview[:100] + "..."
	}

	h.logger.WithFields(logrus.Fields{
		"client_id":       client.ID,
		"message_preview": messagePreview,
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
		// Handle chat request (DESKTOP-03)
		if h.chatHandler != nil {
			h.chatHandler.HandleChatRequest(client, message)
		} else {
			h.logger.Warn("Chat handler not configured, ignoring chat_request")
		}

	case MessageTypePing:
		// Respond with pong
		h.sendPong(client)
	
	case models.WSMessageTypeToolExecutionResponse:
		// Handle tool execution response (v2.5.4+: RPC Tools)
		if h.chatHandler != nil {
			h.chatHandler.HandleToolExecutionResponse(client, message)
		} else {
			h.logger.Warn("Chat handler not configured, ignoring tool_execution_response")
		}

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

	data, err := json.Marshal(pong)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal pong message")
		return
	}

	select {
	case client.Send <- data:
		// Pong sent successfully
	default:
		h.logger.Warn("Failed to send pong (buffer full)")
	}
}

// generateClientID генерирует уникальный ID для клиента
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

