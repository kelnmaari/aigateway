package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/client/ollama"
	"aigateway/internal/config"
	"aigateway/internal/converter"
	"aigateway/internal/models"
)

// Message types для chat streaming (DESKTOP-03)
const (
	MessageTypeChatChunk = "chat_chunk"
	MessageTypeChatDone  = "chat_done"
	MessageTypeChatError = "chat_error"
)

// ChatRequestMessage запрос от desktop client
type ChatRequestMessage struct {
	Type      string                        `json:"type"` // "chat_request"
	RequestID string                        `json:"request_id"`
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

// OllamaClientInterface интерфейс для Ollama клиента
type OllamaClientInterface interface {
	ChatCompletion(ctx context.Context, req *ollama.ChatRequest) (*ollama.ChatResponse, error)
}

// ChatHandler обрабатывает chat requests через WebSocket
type ChatHandler struct {
	config          *config.Config
	logger          *logrus.Logger
	ollamaClient    OllamaClientInterface
	converter       *converter.SimpleConverter
	streamConverter *converter.StreamConverter
	hub             *Hub
}

// NewChatHandler создает новый chat handler для WebSocket
func NewChatHandler(cfg *config.Config, logger *logrus.Logger, ollamaClient OllamaClientInterface, hub *Hub) *ChatHandler {
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

	// Enable streaming
	ollamaReq.Stream = true

	// Create streaming client
	baseClient, ok := h.ollamaClient.(*ollama.Client)
	if !ok {
		h.sendError(client, req.RequestID, "Invalid Ollama client type", "internal_error")
		return
	}

	streamingClient := ollama.NewStreamingClient(baseClient)
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
				h.logger.WithFields(logrus.Fields{
					"request_id":       requestID,
					"chunks_processed": chunksProcessed,
					"total_tokens":     totalTokens,
				}).Debug("WebSocket chat streaming completed")

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
			// Approximate token count
			totalTokens += len(ollamaChunk.Message.Content) / 4

			h.logger.WithFields(logrus.Fields{
				"request_id":   requestID,
				"chunk_index":  chunksProcessed,
				"chunk_tokens": len(ollamaChunk.Message.Content) / 4,
			}).Debug("Sent WebSocket chat chunk")

		case err, ok := <-errorChan:
			if !ok {
				return
			}
			h.logger.WithError(err).WithField("request_id", requestID).Error("Streaming error from Ollama")
			h.sendError(client, requestID, err.Error(), "streaming_error")
			return
		}
	}
}

// sendMessage отправляет JSON message через WebSocket
func (h *ChatHandler) sendMessage(client *Client, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
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

	if err := h.sendMessage(client, &errMsg); err != nil {
		h.logger.WithError(err).Error("Failed to send error message to WebSocket client")
	}
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

	if err := h.sendMessage(client, &doneMsg); err != nil {
		h.logger.WithError(err).Error("Failed to send done message to WebSocket client")
	}
}

