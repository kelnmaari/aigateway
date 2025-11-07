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
	"aigateway/internal/services/agent"
	"aigateway/internal/services/agent/tools"
	"aigateway/internal/storage"
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
	ChatCompletionStream(ctx context.Context, req *ollama.ChatRequest) (<-chan *ollama.ChatResponse, <-chan error)
}

// ChatHandler обрабатывает chat requests через WebSocket
type ChatHandler struct {
	config          *config.Config
	logger          *logrus.Logger
	ollamaClient    OllamaClientInterface
	converter       *converter.SimpleConverter
	streamConverter *converter.StreamConverter
	hub             *Hub
	
	// Agent support (v2.5.1+: Conversational Agent)
	db               storage.Database  // For conversation context persistence
	toolRegistry     *tools.Registry
	conversationalAgent *agent.ConversationalAgent
	
	// Tool RPC (v2.5.4+: Client-side tool execution)
	toolRPCClients map[string]*agent.ToolRPCClient // clientID → RPC client
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
		toolRPCClients:  make(map[string]*agent.ToolRPCClient), // v2.5.4+
	}
}

// SetAgentSupport enables agent functionality for WebSocket chat (v2.5.1+)
func (h *ChatHandler) SetAgentSupport(db storage.Database, toolRegistry *tools.Registry) {
	h.db = db
	h.toolRegistry = toolRegistry
	
	// Get agent config from main config (v2.5.1+)
	planModel := h.config.Agent.PlanModel
	if planModel == "" {
		planModel = "llama3.2:3b" // Safe default if config not set
	}
	
	maxIterations := h.config.Agent.Reasoning.MaxIterations
	if maxIterations == 0 {
		maxIterations = 10 // Default
	}
	
	// Initialize conversational agent with config (v2.5.1+)
	h.conversationalAgent = agent.NewConversationalAgent(h.logger, toolRegistry, agent.ConversationalAgentConfig{
		OllamaURL:     h.config.Ollama.URL,
		Model:         planModel, // From config.agent.plan_model
		MaxIterations: maxIterations, // From config.agent.reasoning.max_iterations
	})
	
	h.logger.WithFields(logrus.Fields{
		"plan_model":     planModel,
		"max_iterations": maxIterations,
	}).Info("Agent support enabled for WebSocket chat")
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
		"agent_mode": req.Payload.AgentMode, // v2.5.1+
	}).Info("Processing WebSocket chat request")

	// Process request asynchronously
	go h.processChatRequest(client, &req)
}

// processChatRequest обрабатывает chat request и отправляет streaming ответ
func (h *ChatHandler) processChatRequest(client *Client, req *ChatRequestMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Agent mode (v2.5.1+: Conversational Agent)
	if req.Payload.AgentMode {
		h.logger.WithField("request_id", req.RequestID).Info("Using agent mode for WebSocket chat")
		h.processAgentRequest(ctx, client, req)
		return
	}

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

	// Start streaming directly via interface (DESKTOP-03 fix)
	responseChan, errorChan := h.ollamaClient.ChatCompletionStream(ctx, ollamaReq)

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

				// Try to send done message, but ignore errors (client may be disconnected)
				_ = h.sendMessage(client, &ChatDoneMessage{
					Type:      MessageTypeChatDone,
					RequestID: requestID,
					Payload: struct {
						MessageID    string `json:"message_id"`
						TotalTokens  int    `json:"total_tokens"`
						FinishReason string `json:"finish_reason"`
					}{
						MessageID:    requestID,
						TotalTokens:  totalTokens,
						FinishReason: "stop",
					},
					Timestamp: time.Now(),
				})
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
				// Client disconnected or send failed - stop streaming gracefully
				h.logger.WithFields(logrus.Fields{
					"error":       err,
					"request_id":  requestID,
					"client_id":   client.ID,
					"chunks_sent": chunksProcessed,
				}).Warn("Client disconnected during streaming, stopping gracefully")
				
				// Try to send done message before exiting (client might still be connected)
				// Ignore errors - if this fails too, client is definitely gone
				_ = h.sendMessage(client, &ChatDoneMessage{
					Type:      MessageTypeChatDone,
					RequestID: requestID,
					Payload: struct {
						MessageID    string `json:"message_id"`
						TotalTokens  int    `json:"total_tokens"`
						FinishReason string `json:"finish_reason"`
					}{
						MessageID:    requestID,
						TotalTokens:  totalTokens,
						FinishReason: "stop",
					},
					Timestamp: time.Now(),
				})
				
				return // Exit goroutine - don't try to send more chunks
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

// processAgentRequest handles agent mode request via WebSocket (v2.5.1+)
func (h *ChatHandler) processAgentRequest(ctx context.Context, client *Client, req *ChatRequestMessage) {
	if h.conversationalAgent == nil {
		h.sendError(client, req.RequestID, "Agent mode not initialized", "agent_not_available")
		return
	}

	// Extract task from last message
	if len(req.Payload.Messages) == 0 {
		h.sendError(client, req.RequestID, "No messages provided for agent mode", "invalid_request")
		return
	}
	
	lastMessage := req.Payload.Messages[len(req.Payload.Messages)-1]
	task := ""
	switch content := lastMessage.Content.(type) {
	case string:
		task = content
	default:
		h.sendError(client, req.RequestID, "Invalid message content type", "invalid_request")
		return
	}

	// Get or create agent context
	var agentContext *models.AgentContext
	if req.Payload.ConversationID != "" && h.db != nil {
		conv, err := h.db.GetConversation(ctx, req.Payload.ConversationID)
		if err == nil && conv != nil {
			agentContext = conv.AgentContext
		}
	}
	if agentContext == nil {
		agentContext = models.NewAgentContext(task)
	}

	// Determine working directory for file operations (v2.5.1+)
	workingDir := req.Payload.AgentWorkingDirectory
	if workingDir == "" {
		workingDir = "." // Default to current directory
	}
	
	h.logger.WithFields(logrus.Fields{
		"request_id":    req.RequestID,
		"working_dir":   workingDir,
	}).Info("Agent working directory set")
	
	// Create tool registry with custom working directory
	customToolRegistry := tools.NewRegistry(h.logger)
	if err := tools.RegisterFileTools(customToolRegistry, workingDir); err != nil {
		h.logger.WithError(err).Warn("Failed to register file tools with custom working dir")
		h.sendError(client, req.RequestID, fmt.Sprintf("Failed to initialize file tools: %v", err), "tool_init_error")
		return
	}
	if err := tools.RegisterTerminalTool(customToolRegistry, workingDir); err != nil {
		h.logger.WithError(err).Warn("Failed to register terminal tool with custom working dir")
	}
	
	// Create conversational agent with custom tool registry
	agentMaxIter := req.Payload.AgentMaxIter
	if agentMaxIter == 0 {
		agentMaxIter = h.config.Agent.Reasoning.MaxIterations
		if agentMaxIter == 0 {
			agentMaxIter = 10 // Default
		}
	}
	
	agentModel := req.Payload.AgentModel
	if agentModel == "" {
		agentModel = h.config.Agent.PlanModel
		if agentModel == "" {
			agentModel = "llama3.2:3b" // Safe default
		}
	}
	
	customAgent := agent.NewConversationalAgent(h.logger, customToolRegistry, agent.ConversationalAgentConfig{
		OllamaURL:     h.config.Ollama.URL,
		Model:         agentModel,
		MaxIterations: agentMaxIter,
	})
	
	// v2.5.4+: Setup Tool RPC Client for client-side tool execution
	rpcClient := h.getOrCreateToolRPCClient(client)
	customAgent.SetRPCClient(rpcClient)
	
	h.logger.WithFields(logrus.Fields{
		"client_id":   client.ID,
		"agent_mode":  "rpc_tools",
	}).Info("ToolRPCClient configured for agent - tools will execute on desktop client")

	// Setup streaming callbacks for custom agent
	customAgent.SetStreamingCallbacks(
		// onThought
		func(thought *models.AgentThought) {
			h.sendMessage(client, &ChatChunkMessage{
				Type:      MessageTypeChatChunk,
				RequestID: req.RequestID,
				Payload: struct {
					Content string `json:"content"`
					Role    string `json:"role"`
					Done    bool   `json:"done"`
				}{
					Content: thought.Thought,
					Role:    string(models.MessageRoleAgentThinking),
					Done:    false,
				},
				Timestamp: time.Now(),
			})
		},
		// onAction
		func(action *models.AgentActionRecord) {
			// Show action description (NO result here - result will be shown in onObservation)
			actionDesc := fmt.Sprintf("🔧 **Tool**: `%s`\n📝 **Action**: %s\n⏱️ **Duration**: %dms\n✅ **Status**: %s",
				action.Tool,
				action.Action,
				action.Duration,
				map[bool]string{true: "Success", false: "Failed"}[action.Success])
			
			if !action.Success && action.Error != "" {
				actionDesc += fmt.Sprintf("\n❌ **Error**: %s", action.Error)
			}
			
			h.sendMessage(client, &ChatChunkMessage{
				Type:      MessageTypeChatChunk,
				RequestID: req.RequestID,
				Payload: struct {
					Content string `json:"content"`
					Role    string `json:"role"`
					Done    bool   `json:"done"`
				}{
					Content: actionDesc,
					Role:    string(models.MessageRoleAgentAction),
					Done:    false,
				},
				Timestamp: time.Now(),
			})
			
			// Result is NOT shown here - it will be formatted and shown in onObservation callback
		},
		// onObservation
		func(observation string) {
			h.sendMessage(client, &ChatChunkMessage{
				Type:      MessageTypeChatChunk,
				RequestID: req.RequestID,
				Payload: struct {
					Content string `json:"content"`
					Role    string `json:"role"`
					Done    bool   `json:"done"`
				}{
					Content: observation,
					Role:    string(models.MessageRoleAgentObservation),
					Done:    false,
				},
				Timestamp: time.Now(),
			})
		},
		// onComplete
		func(success bool, result string) {
			// Send final assistant message
			h.sendMessage(client, &ChatChunkMessage{
				Type:      MessageTypeChatChunk,
				RequestID: req.RequestID,
				Payload: struct {
					Content string `json:"content"`
					Role    string `json:"role"`
					Done    bool   `json:"done"`
				}{
					Content: result,
					Role:    "assistant",
					Done:    true,
				},
				Timestamp: time.Now(),
			})
			
			// Send done message
			finishReason := "stop"
			if !success {
				finishReason = "error"
			}
			h.sendMessage(client, &ChatDoneMessage{
				Type:      MessageTypeChatDone,
				RequestID: req.RequestID,
				Payload: struct {
					MessageID    string `json:"message_id"`
					TotalTokens  int    `json:"total_tokens"`
					FinishReason string `json:"finish_reason"`
				}{
					MessageID:    req.RequestID,
					TotalTokens:  0, // Not tracked in agent mode
					FinishReason: finishReason,
				},
				Timestamp: time.Now(),
			})
		},
	)

	// Start agent task with custom agent (uses custom working directory)
	_, _, err := customAgent.ProcessTask(ctx, task, agentContext)
	if err != nil {
		h.logger.WithError(err).WithField("request_id", req.RequestID).Error("Agent task failed")
		// onComplete already called with error
	}
}

// sendMessage отправляет JSON message через WebSocket
// Gracefully handles closed channels (CLIENT-016 fix)
func (h *ChatHandler) sendMessage(client *Client, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Protect against sending to closed channel
	defer func() {
		if r := recover(); r != nil {
			h.logger.WithFields(logrus.Fields{
				"client_id": client.ID,
				"panic":     r,
			}).Warn("Recovered from panic when sending message (client disconnected)")
		}
	}()

	select {
	case client.Send <- data:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("send timeout: client may be disconnected")
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

// getOrCreateToolRPCClient получает или создает ToolRPCClient для клиента (v2.5.4+)
func (h *ChatHandler) getOrCreateToolRPCClient(client *Client) *agent.ToolRPCClient {
	// Check if client already has RPC client
	if rpcClient, exists := h.toolRPCClients[client.ID]; exists {
		return rpcClient
	}
	
	// Create new RPC client with send function
	sendFunc := func(messageType string, payload interface{}) error {
		// Send message to WebSocket client
		message := map[string]interface{}{
			"type":    messageType,
			"payload": payload,
		}
		
		if err := h.sendMessage(client, message); err != nil {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"client_id":    client.ID,
				"message_type": messageType,
			}).Error("Failed to send tool RPC message to client")
			return err
		}
		
		return nil
	}
	
	rpcClient := agent.NewToolRPCClient(h.logger, sendFunc)
	h.toolRPCClients[client.ID] = rpcClient
	
	h.logger.WithField("client_id", client.ID).Info("Created new ToolRPCClient for WebSocket client")
	
	return rpcClient
}

// HandleToolExecutionResponse handles tool_execution_response message from client (v2.5.4+)
func (h *ChatHandler) HandleToolExecutionResponse(client *Client, message []byte) {
	h.logger.WithField("client_id", client.ID).Debug("Received tool execution response")
	
	// Get RPC client for this WebSocket client
	rpcClient, exists := h.toolRPCClients[client.ID]
	if !exists {
		h.logger.WithField("client_id", client.ID).Warn("Received tool response for unknown client, no RPC client found")
		return
	}
	
	// Forward response to RPC client
	if err := rpcClient.HandleResponse(message); err != nil {
		h.logger.WithError(err).WithField("client_id", client.ID).Error("Failed to handle tool execution response")
	}
}

// CleanupToolRPCClient removes ToolRPCClient when client disconnects (v2.5.4+)
func (h *ChatHandler) CleanupToolRPCClient(clientID string) {
	if rpcClient, exists := h.toolRPCClients[clientID]; exists {
		// Cancel all pending requests for this client
		h.logger.WithField("client_id", clientID).Info("Cleaning up ToolRPCClient")
		
		// Remove from map
		delete(h.toolRPCClients, clientID)
		
		// Note: RPC client will be garbage collected, pending requests will timeout
		_ = rpcClient // Prevent unused warning
	}
}

