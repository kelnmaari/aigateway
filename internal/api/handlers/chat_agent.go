// Package handlers provides HTTP handlers for agent chat integration
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/services/agent"
	"aigateway/internal/services/agent/tools"
)

// handleAgentChatCompletion handles chat completion in agent mode (v2.5.1+)
func (h *ChatHandler) handleAgentChatCompletion(c *gin.Context, req *models.ChatCompletionRequest) {
	ctx := c.Request.Context()

	h.logger.WithFields(logrus.Fields{
		"agent_mode":      req.AgentMode,
		"agent_max_iter":  req.AgentMaxIter,
		"agent_model":     req.AgentModel,
		"conversation_id": req.ConversationID,
	}).Info("Agent mode chat completion")

	// Extract user task from last message
	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.Error{
				Message: "At least one message is required for agent mode",
				Type:    "invalid_request_error",
				Code:    "no_messages",
			},
		})
		return
	}

	lastMessage := req.Messages[len(req.Messages)-1]
	task := ""
	switch v := lastMessage.Content.(type) {
	case string:
		task = v
	default:
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.Error{
				Message: "Agent mode requires simple string message content",
				Type:    "invalid_request_error",
				Code:    "invalid_content_type",
			},
		})
		return
	}

	// Load or create conversation
	var conversation *models.Conversation
	var agentContext *models.AgentContext
	if req.ConversationID != "" && h.db != nil {
		// Load existing conversation
		conv, err := h.db.GetConversation(ctx, req.ConversationID)
		if err != nil {
			h.logger.WithError(err).Warn("Failed to load conversation, creating new")
		} else {
			conversation = conv
			agentContext = conv.AgentContext
		}
	}

	// Create conversation if not loaded
	if conversation == nil && h.db != nil {
		userID, err := getUserIDFromContext(c)
		if err != nil {
			userID = "system"
		}
		tenantID := getTenantIDFromContext(c)

		conversation = &models.Conversation{
			ID:           uuid.New().String(),
			Title:        truncateString(task, 50),
			UserID:       userID,
			TenantID:     tenantID,
			Model:        req.Model,
			Temperature:  req.Temperature,
			Status:       models.ConversationStatusActive,
			AgentMode:    true, // Mark as agent mode
			AgentContext: models.NewAgentContext(task),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if err := h.db.CreateConversation(ctx, conversation); err != nil {
			h.logger.WithError(err).Error("Failed to create conversation")
			// Continue without persistence
		} else {
			agentContext = conversation.AgentContext
		}
	}

	// Initialize conversational agent
	toolRegistry := tools.NewRegistry(h.logger)
	
	// Register tools
	if err := tools.RegisterFileTools(toolRegistry, "."); err != nil {
		h.logger.WithError(err).Warn("Failed to register file tools")
	}
	if err := tools.RegisterTerminalTool(toolRegistry, "."); err != nil {
		h.logger.WithError(err).Warn("Failed to register terminal tool")
	}

	agentModel := req.AgentModel
	if agentModel == "" {
		agentModel = "deepseek-r1:1.5b" // Default reasoning model
	}

	maxIter := req.AgentMaxIter
	if maxIter == 0 {
		maxIter = 10 // Default
	}

	conversationalAgent := agent.NewConversationalAgent(h.logger, toolRegistry, agent.ConversationalAgentConfig{
		OllamaURL:     h.config.Ollama.URL,
		Model:         agentModel,
		MaxIterations: maxIter,
	})

	// Handle streaming vs non-streaming
	if req.Stream {
		h.handleAgentStreamingCompletion(c, conversation, conversationalAgent, task, agentContext)
	} else {
		h.handleAgentNonStreamingCompletion(c, conversation, conversationalAgent, task, agentContext)
	}
}

// handleAgentStreamingCompletion handles streaming agent responses via SSE
func (h *ChatHandler) handleAgentStreamingCompletion(c *gin.Context, conversation *models.Conversation, conversationalAgent *agent.ConversationalAgent, task string, agentContext *models.AgentContext) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		h.logger.Error("Streaming not supported")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	ctx := c.Request.Context()
	requestID := fmt.Sprintf("chatcmpl-%s", uuid.New().String())

	// Set up event callbacks for streaming
	conversationalAgent.SetEventCallbacks(
		// onThought
		func(thought *models.AgentThought) {
			chunk := models.ChatCompletionChunk{
				ID:      requestID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   conversation.Model,
				Choices: []models.ChatCompletionChunkChoice{
					{
						Index: 0,
						Delta: models.ChatMessage{
							Role:    string(models.MessageRoleAgentThinking),
							Content: thought.Thought,
						},
						FinishReason: nil,
					},
				},
			}
			h.streamChunk(c, chunk, flusher)
		},
		// onAction
		func(action *models.AgentActionRecord) {
			actionDesc := fmt.Sprintf("Executing %s with %v", action.Tool, action.Parameters)
			chunk := models.ChatCompletionChunk{
				ID:      requestID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   conversation.Model,
				Choices: []models.ChatCompletionChunkChoice{
					{
						Index: 0,
						Delta: models.ChatMessage{
							Role:    string(models.MessageRoleAgentAction),
							Content: actionDesc,
						},
						FinishReason: nil,
					},
				},
			}
			h.streamChunk(c, chunk, flusher)

			// Stream result
			if action.Success {
				resultChunk := models.ChatCompletionChunk{
					ID:      requestID,
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   conversation.Model,
					Choices: []models.ChatCompletionChunkChoice{
						{
							Index: 0,
							Delta: models.ChatMessage{
								Role:    string(models.MessageRoleAgentObservation),
								Content: fmt.Sprintf("Result: %s", truncateString(action.Result, 200)),
							},
							FinishReason: nil,
						},
					},
				}
				h.streamChunk(c, resultChunk, flusher)
			}
		},
		// onObservation
		func(observation string) {
			chunk := models.ChatCompletionChunk{
				ID:      requestID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   conversation.Model,
				Choices: []models.ChatCompletionChunkChoice{
					{
						Index: 0,
						Delta: models.ChatMessage{
							Role:    string(models.MessageRoleAgentObservation),
							Content: observation,
						},
						FinishReason: nil,
					},
				},
			}
			h.streamChunk(c, chunk, flusher)
		},
		// onComplete
		func(success bool, result string) {
			finishReason := "stop"
			if !success {
				finishReason = "error"
			}

			// Final assistant message with result
			finalChunk := models.ChatCompletionChunk{
				ID:      requestID,
				Object:  "chat.completion.chunk",
				Created: time.Now().Unix(),
				Model:   conversation.Model,
				Choices: []models.ChatCompletionChunkChoice{
					{
						Index: 0,
						Delta: models.ChatMessage{
							Role:    "assistant",
							Content: result,
						},
						FinishReason: &finishReason,
					},
				},
			}
			h.streamChunk(c, finalChunk, flusher)

			// Send [DONE]
			fmt.Fprintf(c.Writer, "data: [DONE]\n\n")
			flusher.Flush()
		},
	)

	// Execute agent task
	result, updatedContext, err := conversationalAgent.ProcessTask(ctx, task, agentContext)
	if err != nil {
		h.logger.WithError(err).Error("Agent task failed")
		// Error already streamed via onComplete callback
	}

	// Save updated context to conversation
	if conversation != nil && h.db != nil {
		conversation.AgentContext = updatedContext
		conversation.UpdatedAt = time.Now()
		if err := h.db.UpdateConversation(ctx, conversation); err != nil {
			h.logger.WithError(err).Error("Failed to update conversation context")
		}
	}

	h.logger.WithField("result", truncateString(result, 100)).Info("Agent task completed")
}

// handleAgentNonStreamingCompletion handles non-streaming agent responses
func (h *ChatHandler) handleAgentNonStreamingCompletion(c *gin.Context, conversation *models.Conversation, conversationalAgent *agent.ConversationalAgent, task string, agentContext *models.AgentContext) {
	ctx := c.Request.Context()

	// Execute agent task
	result, updatedContext, err := conversationalAgent.ProcessTask(ctx, task, agentContext)
	if err != nil {
		h.logger.WithError(err).Error("Agent task failed")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.Error{
				Message: fmt.Sprintf("Agent task failed: %v", err),
				Type:    "agent_error",
				Code:    "task_failed",
			},
		})
		return
	}

	// Save updated context to conversation
	if conversation != nil && h.db != nil {
		conversation.AgentContext = updatedContext
		conversation.UpdatedAt = time.Now()
		if err := h.db.UpdateConversation(ctx, conversation); err != nil {
			h.logger.WithError(err).Error("Failed to update conversation context")
		}
	}

	// Build response with all agent messages (thoughts, actions, observations)
	messages := []map[string]interface{}{}
	
	// Add thoughts
	for _, thought := range updatedContext.ThoughtHistory {
		messages = append(messages, map[string]interface{}{
			"role":       models.MessageRoleAgentThinking,
			"content":    thought.Thought,
			"confidence": thought.Confidence,
			"step":       thought.StepNumber,
		})
	}

	// Add actions and observations interleaved
	for _, action := range updatedContext.ActionHistory {
		messages = append(messages, map[string]interface{}{
			"role":    models.MessageRoleAgentAction,
			"content": fmt.Sprintf("Executing %s", action.Tool),
			"tool":    action.Tool,
			"params":  action.Parameters,
			"step":    action.StepNumber,
		})
		
		obsContent := ""
		if action.Success {
			obsContent = fmt.Sprintf("Success: %s", truncateString(action.Result, 200))
		} else {
			obsContent = fmt.Sprintf("Failed: %s", action.Error)
		}
		
		messages = append(messages, map[string]interface{}{
			"role":    models.MessageRoleAgentObservation,
			"content": obsContent,
			"step":    action.StepNumber,
		})
	}

	// Add final assistant response
	messages = append(messages, map[string]interface{}{
		"role":    "assistant",
		"content": result,
	})

	// Create OpenAI-compatible response
	response := models.ChatCompletionResponse{
		ID:      fmt.Sprintf("chatcmpl-%s", uuid.New().String()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   conversation.Model,
		Choices: []models.ChatCompletionChoice{
			{
				Index: 0,
				Message: models.ChatMessage{
					Role:    "assistant",
					Content: result,
				},
				FinishReason: "stop",
			},
		},
		Usage: models.Usage{
			PromptTokens:     len(task) / 4,     // Rough estimate
			CompletionTokens: len(result) / 4,   // Rough estimate
			TotalTokens:      (len(task) + len(result)) / 4,
		},
		AgentMessages: messages, // Custom field for agent steps
	}

	c.JSON(http.StatusOK, response)
}

// streamChunk sends a chunk via SSE
func (h *ChatHandler) streamChunk(c *gin.Context, chunk models.ChatCompletionChunk, flusher http.Flusher) {
	chunkJSON, err := json.Marshal(chunk)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal chunk")
		return
	}

	fmt.Fprintf(c.Writer, "data: %s\n\n", string(chunkJSON))
	flusher.Flush()
}

// truncateString truncates string to maxLen
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

