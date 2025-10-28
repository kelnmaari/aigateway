package websocket

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Client представляет WebSocket клиента
type Client struct {
	ID       string
	Hub      *Hub
	Conn     Connection // Абстракция для тестирования
	Send     chan []byte
	UserInfo map[string]string // Дополнительная информация (IP, User-Agent, etc.)
}

// Connection интерфейс для WebSocket connection (для тестирования)
type Connection interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (messageType int, p []byte, err error)
	Close() error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

// Hub управляет WebSocket клиентами
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Per-user client tracking (DESKTOP-03)
	userClients map[string][]*Client // user_id -> clients

	// Broadcast канал для отправки сообщений всем клиентам
	broadcast chan []byte

	// Register запросы от клиентов
	register chan *Client

	// Unregister запросы от клиентов
	unregister chan *Client

	// Канал для остановки
	stopChan chan struct{}

	// WaitGroup для graceful shutdown
	wg sync.WaitGroup

	// Mutex для защиты clients map
	mu sync.RWMutex

	// Logger
	logger *logrus.Logger

	// Статистика
	stats HubStats
}

// HubStats статистика Hub
type HubStats struct {
	mu                sync.RWMutex
	TotalConnections  int64
	ActiveConnections int64
	MessagesSent      int64
	MessagesReceived  int64
	Errors            int64
}

// NewHub создает новый WebSocket hub
func NewHub(logger *logrus.Logger) *Hub {
	if logger == nil {
		logger = logrus.New()
	}

	return &Hub{
		clients:     make(map[*Client]bool),
		userClients: make(map[string][]*Client),
		broadcast:   make(chan []byte, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		stopChan:    make(chan struct{}),
		logger:      logger,
	}
}

// Run запускает Hub
func (h *Hub) Run() {
	h.logger.Info("WebSocket Hub started")
	h.wg.Add(1)

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-h.stopChan:
			h.logger.Info("WebSocket Hub stopping")
			h.shutdown()
			h.wg.Done()
			return
		}
	}
}

// Stop останавливает Hub
func (h *Hub) Stop() {
	close(h.stopChan)
	h.wg.Wait()
	h.logger.Info("WebSocket Hub stopped")
}

// registerClient регистрирует нового клиента
func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	h.clients[client] = true

	// Track user's clients for targeted messaging (DESKTOP-03)
	if userID := client.UserInfo["user_id"]; userID != "" {
		h.userClients[userID] = append(h.userClients[userID], client)
	}

	h.mu.Unlock()

	h.stats.mu.Lock()
	h.stats.TotalConnections++
	h.stats.ActiveConnections++
	h.stats.mu.Unlock()

	h.logger.WithFields(logrus.Fields{
		"client_id":         client.ID,
		"user_id":           client.UserInfo["user_id"],
		"device_name":       client.UserInfo["device_name"],
		"active_clients":    h.GetActiveClientsCount(),
		"total_connections": h.stats.TotalConnections,
	}).Info("WebSocket client connected")
}

// unregisterClient отключает клиента
func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.Send)

		// Remove from user's clients (DESKTOP-03)
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

	h.stats.mu.Lock()
	h.stats.ActiveConnections--
	h.stats.mu.Unlock()

	h.logger.WithFields(logrus.Fields{
		"client_id":      client.ID,
		"user_id":        client.UserInfo["user_id"],
		"active_clients": h.GetActiveClientsCount(),
	}).Info("WebSocket client disconnected")
}

// broadcastMessage отправляет сообщение всем клиентам
func (h *Hub) broadcastMessage(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.Send <- message:
			// Сообщение отправлено успешно
		default:
			// Канал клиента заполнен, отключаем его
			go func(c *Client) {
				h.unregister <- c
			}(client)
		}
	}

	h.stats.mu.Lock()
	h.stats.MessagesSent += int64(len(h.clients))
	h.stats.mu.Unlock()
}

// Broadcast отправляет сообщение всем подключенным клиентам
func (h *Hub) Broadcast(message []byte) {
	select {
	case h.broadcast <- message:
		// Message queued for broadcast
	default:
		h.logger.Warn("Broadcast channel full, message dropped")
		h.stats.mu.Lock()
		h.stats.Errors++
		h.stats.mu.Unlock()
	}
}

// GetActiveClientsCount возвращает количество активных клиентов
func (h *Hub) GetActiveClientsCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetStats возвращает статистику Hub
func (h *Hub) GetStats() HubStats {
	h.stats.mu.RLock()
	defer h.stats.mu.RUnlock()

	return HubStats{
		TotalConnections:  h.stats.TotalConnections,
		ActiveConnections: h.stats.ActiveConnections,
		MessagesSent:      h.stats.MessagesSent,
		MessagesReceived:  h.stats.MessagesReceived,
		Errors:            h.stats.Errors,
	}
}

// shutdown закрывает все соединения
func (h *Hub) shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		close(client.Send)
		client.Conn.Close()
	}

	h.clients = make(map[*Client]bool)
	h.logger.Info("All WebSocket clients disconnected")
}

// GetClients возвращает список активных клиентов (для отладки)
func (h *Hub) GetClients() []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}

	return clients
}

// SendToUser отправляет сообщение всем клиентам конкретного user (DESKTOP-03)
func (h *Hub) SendToUser(userID string, message []byte) {
	h.mu.RLock()
	clients := h.userClients[userID]
	h.mu.RUnlock()

	if len(clients) == 0 {
		h.logger.WithField("user_id", userID).Debug("No active clients for user")
		return
	}

	for _, client := range clients {
		select {
		case client.Send <- message:
			// Message sent successfully
		default:
			// Client buffer full, disconnect slow client
			h.logger.WithFields(logrus.Fields{
				"user_id":   userID,
				"client_id": client.ID,
			}).Warn("Client buffer full, disconnecting")
			go func(c *Client) {
				h.unregister <- c
			}(client)
		}
	}
}

// SendToClient отправляет сообщение конкретному клиенту (DESKTOP-03)
func (h *Hub) SendToClient(clientID string, message []byte) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.ID == clientID {
			select {
			case client.Send <- message:
				return true
			default:
				h.logger.WithField("client_id", clientID).Warn("Client buffer full")
				return false
			}
		}
	}

	h.logger.WithField("client_id", clientID).Debug("Client not found")
	return false
}

