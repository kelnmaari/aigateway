package websocket

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
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
	hub    *Hub
	logger *logrus.Logger
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

// HandleWebSocket обрабатывает WebSocket upgrade и connection
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

// readPump читает сообщения от клиента
func (h *Handler) readPump(client *Client) {
	defer func() {
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
	h.logger.WithFields(logrus.Fields{
		"client_id": client.ID,
		"message":   string(message),
	}).Debug("Received message from client")

	// В базовой реализации просто логируем
	// В будущем можно добавить обработку команд от клиента
}

// generateClientID генерирует уникальный ID для клиента
func generateClientID() string {
	return fmt.Sprintf("client_%d", time.Now().UnixNano())
}

