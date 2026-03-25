// Package realtime provides WebSocket support for real-time queue updates
package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Hub manages WebSocket connections and broadcasts
type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	upgrader   websocket.Upgrader
	logger     *logrus.Logger
	ctx        context.Context
	cancel     context.CancelFunc
}

// Client represents a WebSocket client connection
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	userID    string
	projectID string // Subscribe to specific project
	filters   SubscriptionFilters
}

// SubscriptionFilters defines what updates a client wants
type SubscriptionFilters struct {
	ProjectIDs     []string `json:"project_ids,omitempty"`
	IntegrationIDs []string `json:"integration_ids,omitempty"`
	EventTypes     []string `json:"event_types,omitempty"` // job_created, job_completed, etc.
}

// Message represents a WebSocket message
type Message struct {
	Type      MessageType    `json:"type"`
	Event     string         `json:"event"`
	Data      any            `json:"data"`
	Timestamp time.Time      `json:"timestamp"`
	Meta      map[string]any `json:"meta,omitempty"`
}

// MessageType defines message types
type MessageType string

const (
	MessageTypeQueueUpdate   MessageType = "queue_update"
	MessageTypeJobCreated    MessageType = "job_created"
	MessageTypeJobStarted    MessageType = "job_started"
	MessageTypeJobCompleted  MessageType = "job_completed"
	MessageTypeJobFailed     MessageType = "job_failed"
	MessageTypeReviewCreated MessageType = "review_created"
	MessageTypeWorkerStatus  MessageType = "worker_status"
	MessageTypeStats         MessageType = "stats"
	MessageTypePing          MessageType = "ping"
	MessageTypePong          MessageType = "pong"
)

// NewHub creates a new WebSocket hub
func NewHub(logger *logrus.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())

	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Configure appropriately for production
			},
		},
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.ctx.Done():
			h.closeAllClients()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.WithField("clients", len(h.clients)).Debug("Client registered")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.WithField("clients", len(h.clients)).Debug("Client unregistered")

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-ticker.C:
			// Send periodic ping
			h.broadcastMessage(&Message{
				Type:      MessageTypePing,
				Timestamp: time.Now(),
			})
		}
	}
}

// Stop stops the hub
func (h *Hub) Stop() {
	h.cancel()
}

// broadcastMessage sends a message to all matching clients
func (h *Hub) broadcastMessage(message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal message")
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.matchesFilters(message) {
			select {
			case client.send <- data:
			default:
				// Client buffer full, skip
			}
		}
	}
}

func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		close(client.send)
		delete(h.clients, client)
	}
}

// BroadcastJobCreated broadcasts a job created event
func (h *Hub) BroadcastJobCreated(job *JobEvent) {
	h.broadcast <- &Message{
		Type:      MessageTypeJobCreated,
		Event:     "job_created",
		Data:      job,
		Timestamp: time.Now(),
	}
}

// BroadcastJobStarted broadcasts a job started event
func (h *Hub) BroadcastJobStarted(job *JobEvent) {
	h.broadcast <- &Message{
		Type:      MessageTypeJobStarted,
		Event:     "job_started",
		Data:      job,
		Timestamp: time.Now(),
	}
}

// BroadcastJobCompleted broadcasts a job completed event
func (h *Hub) BroadcastJobCompleted(job *JobEvent) {
	h.broadcast <- &Message{
		Type:      MessageTypeJobCompleted,
		Event:     "job_completed",
		Data:      job,
		Timestamp: time.Now(),
	}
}

// BroadcastJobFailed broadcasts a job failed event
func (h *Hub) BroadcastJobFailed(job *JobEvent, err string) {
	h.broadcast <- &Message{
		Type:      MessageTypeJobFailed,
		Event:     "job_failed",
		Data:      job,
		Timestamp: time.Now(),
		Meta:      map[string]any{"error": err},
	}
}

// BroadcastQueueStats broadcasts queue statistics
func (h *Hub) BroadcastQueueStats(stats *QueueStats) {
	h.broadcast <- &Message{
		Type:      MessageTypeStats,
		Event:     "queue_stats",
		Data:      stats,
		Timestamp: time.Now(),
	}
}

// BroadcastWorkerStatus broadcasts worker status
func (h *Hub) BroadcastWorkerStatus(status *WorkerStatus) {
	h.broadcast <- &Message{
		Type:      MessageTypeWorkerStatus,
		Event:     "worker_status",
		Data:      status,
		Timestamp: time.Now(),
	}
}

// JobEvent represents a job event for broadcasting
type JobEvent struct {
	JobID         string    `json:"job_id"`
	ReviewID      string    `json:"review_id"`
	ProjectID     string    `json:"project_id"`
	ProjectName   string    `json:"project_name"`
	IntegrationID string    `json:"integration_id"`
	MRIID         int       `json:"mr_iid"`
	MRTitle       string    `json:"mr_title"`
	Status        string    `json:"status"`
	Priority      int       `json:"priority"`
	WorkerID      string    `json:"worker_id,omitempty"`
	StartedAt     time.Time `json:"started_at"`
	CompletedAt   time.Time `json:"completed_at"`
	Duration      int64     `json:"duration_ms,omitempty"`
	TokensUsed    int       `json:"tokens_used,omitempty"`
	IssuesFound   int       `json:"issues_found,omitempty"`
	Score         int       `json:"score,omitempty"`
}

// QueueStats represents queue statistics
type QueueStats struct {
	Pending    int         `json:"pending"`
	Processing int         `json:"processing"`
	Completed  int         `json:"completed"`
	Failed     int         `json:"failed"`
	ByPriority map[int]int `json:"by_priority,omitempty"`
	AvgWaitMs  int64       `json:"avg_wait_ms"`
	AvgProcMs  int64       `json:"avg_proc_ms"`
}

// WorkerStatus represents worker status
type WorkerStatus struct {
	WorkerID      string    `json:"worker_id"`
	Status        string    `json:"status"` // idle, busy, stopped
	CurrentJob    string    `json:"current_job,omitempty"`
	StartedAt     time.Time `json:"started_at"`
	JobsProcessed int       `json:"jobs_processed"`
}

// ServeWS handles WebSocket connection requests
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.WithError(err).Error("WebSocket upgrade failed")
		return
	}

	// Get user ID from context (from auth middleware)
	userID := r.Context().Value("user_id")
	userIDStr := ""
	if userID != nil {
		userIDStr = userID.(string)
	}

	client := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userIDStr,
	}

	h.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	// Send initial connection message
	client.send <- mustJSON(&Message{
		Type:      MessageTypeStats,
		Event:     "connected",
		Timestamp: time.Now(),
	})
}

// matchesFilters checks if a message matches client's subscription filters
func (c *Client) matchesFilters(msg *Message) bool {
	// No filters = receive everything
	if len(c.filters.EventTypes) == 0 && len(c.filters.ProjectIDs) == 0 && len(c.filters.IntegrationIDs) == 0 {
		return true
	}

	// Check event type filter
	if len(c.filters.EventTypes) > 0 {
		eventType := string(msg.Type)
		matched := false
		for _, et := range c.filters.EventTypes {
			if et == eventType || et == msg.Event {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check project filter
	if len(c.filters.ProjectIDs) > 0 {
		if job, ok := msg.Data.(*JobEvent); ok {
			matched := slices.Contains(c.filters.ProjectIDs, job.ProjectID)
			if !matched {
				return false
			}
		}
	}

	return true
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.WithError(err).Error("WebSocket read error")
			}
			break
		}

		// Parse client message
		var clientMsg ClientMessage
		if err := json.Unmarshal(message, &clientMsg); err != nil {
			continue
		}

		// Handle client message
		c.handleMessage(&clientMsg)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Write queued messages
			n := len(c.send)
			for range n {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ClientMessage represents a message from the client
type ClientMessage struct {
	Type    string               `json:"type"`
	Filters *SubscriptionFilters `json:"filters,omitempty"`
}

func (c *Client) handleMessage(msg *ClientMessage) {
	switch msg.Type {
	case "subscribe":
		if msg.Filters != nil {
			c.filters = *msg.Filters
		}
	case "unsubscribe":
		c.filters = SubscriptionFilters{}
	case "pong":
		// Client responded to ping
	}
}

func mustJSON(v any) []byte {
	data, _ := json.Marshal(v)
	return data
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
