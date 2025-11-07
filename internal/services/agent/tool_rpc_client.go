package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
)

// ToolRPCClient handles RPC communication with the desktop client for tool execution
// Server → Client: Send tool execution requests
// Client → Server: Receive tool execution responses
type ToolRPCClient struct {
	logger *logrus.Logger

	// Pending requests (waiting for response from client)
	pendingMu       sync.RWMutex
	pendingRequests map[string]*pendingToolRequest // requestID → request

	// Response sender (WebSocket client sender)
	sendFunc func(messageType string, payload interface{}) error

	// Configuration
	defaultTimeout time.Duration // Default timeout for tool execution
}

// pendingToolRequest represents a tool execution request waiting for response
type pendingToolRequest struct {
	RequestID   string
	Tool        string
	Parameters  map[string]interface{}
	StartTime   time.Time
	Timeout     time.Duration
	ResponseCh  chan *models.ToolExecutionResponse
	CancelFunc  context.CancelFunc
	TimerCancel context.CancelFunc // For cleanup timeout timer
}

// NewToolRPCClient creates a new RPC client for tool execution
func NewToolRPCClient(logger *logrus.Logger, sendFunc func(string, interface{}) error) *ToolRPCClient {
	return &ToolRPCClient{
		logger:          logger,
		pendingRequests: make(map[string]*pendingToolRequest),
		sendFunc:        sendFunc,
		defaultTimeout:  60 * time.Second, // Default 60s timeout
	}
}

// ExecuteTool executes a tool on the client via RPC
// This is called by the ConversationalAgent during the ReAct loop
func (c *ToolRPCClient) ExecuteTool(ctx context.Context, tool string, parameters map[string]interface{}, requiresApproval bool) (*models.ToolExecutionResponse, error) {
	// Generate unique request ID
	requestID := fmt.Sprintf("tool_rpc_%d_%s", time.Now().UnixNano(), randString(8))

	c.logger.WithFields(logrus.Fields{
		"request_id":        requestID,
		"tool":              tool,
		"parameters":        parameters,
		"requires_approval": requiresApproval,
	}).Info("Sending tool execution request to client")

	// Create request
	request := &models.ToolExecutionRequest{
		RequestID:        requestID,
		Tool:             tool,
		Parameters:       parameters,
		Description:      getToolDescription(tool, parameters),
		Timestamp:        time.Now(),
		RequiresApproval: requiresApproval,
		IsDangerous:      isDangerousTool(tool),
	}

	// Create response channel
	responseCh := make(chan *models.ToolExecutionResponse, 1)

	// Create cancellable context for timeout
	requestCtx, cancel := context.WithTimeout(ctx, c.defaultTimeout)
	defer cancel()

	// Register pending request
	pending := &pendingToolRequest{
		RequestID:  requestID,
		Tool:       tool,
		Parameters: parameters,
		StartTime:  time.Now(),
		Timeout:    c.defaultTimeout,
		ResponseCh: responseCh,
		CancelFunc: cancel,
	}

	c.pendingMu.Lock()
	c.pendingRequests[requestID] = pending
	c.pendingMu.Unlock()

	// Cleanup on exit
	defer func() {
		c.pendingMu.Lock()
		delete(c.pendingRequests, requestID)
		c.pendingMu.Unlock()
		close(responseCh)
	}()

	// Send request to client via WebSocket
	if err := c.sendFunc(models.WSMessageTypeToolExecutionRequest, request); err != nil {
		c.logger.WithError(err).WithField("request_id", requestID).Error("Failed to send tool execution request")
		return nil, fmt.Errorf("failed to send tool execution request: %w", err)
	}

	c.logger.WithField("request_id", requestID).Debug("Tool execution request sent, waiting for response...")

	// Wait for response or timeout
	select {
	case response := <-responseCh:
		duration := time.Since(pending.StartTime)
		c.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"success":    response.Success,
			"duration":   duration,
		}).Info("Received tool execution response")

		return response, nil

	case <-requestCtx.Done():
		// Timeout or cancellation
		if requestCtx.Err() == context.DeadlineExceeded {
			c.logger.WithField("request_id", requestID).Warn("Tool execution request timed out")

			// Send cancellation message to client
			_ = c.sendFunc(models.WSMessageTypeToolExecutionCancel, map[string]string{
				"request_id": requestID,
				"reason":     "timeout",
			})

			return &models.ToolExecutionResponse{
				RequestID:   requestID,
				Success:     false,
				Error:       fmt.Sprintf("tool execution timed out after %v", c.defaultTimeout),
				Duration:    int64(c.defaultTimeout.Milliseconds()),
				CompletedAt: time.Now(),
			}, nil
		}

		return nil, fmt.Errorf("tool execution cancelled: %w", requestCtx.Err())
	}
}

// HandleResponse handles a tool execution response from the client
// This is called by the WebSocket handler when a response is received
func (c *ToolRPCClient) HandleResponse(responseData []byte) error {
	var response models.ToolExecutionResponse
	if err := json.Unmarshal(responseData, &response); err != nil {
		c.logger.WithError(err).Error("Failed to unmarshal tool execution response")
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"request_id": response.RequestID,
		"success":    response.Success,
		"duration":   response.Duration,
	}).Debug("Received tool execution response")

	// Find pending request
	c.pendingMu.RLock()
	pending, exists := c.pendingRequests[response.RequestID]
	c.pendingMu.RUnlock()

	if !exists {
		c.logger.WithField("request_id", response.RequestID).Warn("Received response for unknown request ID")
		return fmt.Errorf("unknown request ID: %s", response.RequestID)
	}

	// Send response to waiting goroutine
	select {
	case pending.ResponseCh <- &response:
		c.logger.WithField("request_id", response.RequestID).Debug("Response delivered to waiting goroutine")
	case <-time.After(1 * time.Second):
		c.logger.WithField("request_id", response.RequestID).Warn("Response channel blocked, discarding response")
	}

	return nil
}

// CancelPendingRequest cancels a pending tool execution request
func (c *ToolRPCClient) CancelPendingRequest(requestID string) error {
	c.pendingMu.Lock()
	pending, exists := c.pendingRequests[requestID]
	if exists {
		delete(c.pendingRequests, requestID)
	}
	c.pendingMu.Unlock()

	if !exists {
		return fmt.Errorf("request not found: %s", requestID)
	}

	c.logger.WithField("request_id", requestID).Info("Cancelling tool execution request")

	// Cancel context
	if pending.CancelFunc != nil {
		pending.CancelFunc()
	}

	// Send cancellation message to client
	return c.sendFunc(models.WSMessageTypeToolExecutionCancel, map[string]string{
		"request_id": requestID,
		"reason":     "cancelled",
	})
}

// GetPendingRequestsCount returns the number of pending requests
func (c *ToolRPCClient) GetPendingRequestsCount() int {
	c.pendingMu.RLock()
	defer c.pendingMu.RUnlock()
	return len(c.pendingRequests)
}

// Helper functions

// getToolDescription generates a human-readable description for a tool execution
func getToolDescription(tool string, params map[string]interface{}) string {
	switch tool {
	case "file.read":
		if path, ok := params["path"].(string); ok {
			return fmt.Sprintf("Read file: %s", path)
		}
		return "Read file"

	case "file.list":
		if path, ok := params["path"].(string); ok {
			return fmt.Sprintf("List files in: %s", path)
		}
		return "List files"

	case "file.write":
		if path, ok := params["path"].(string); ok {
			return fmt.Sprintf("Write file: %s", path)
		}
		return "Write file"

	case "file.delete":
		if path, ok := params["path"].(string); ok {
			return fmt.Sprintf("Delete file: %s", path)
		}
		return "Delete file"

	case "terminal.execute":
		if cmd, ok := params["command"].(string); ok {
			return fmt.Sprintf("Execute command: %s", cmd)
		}
		return "Execute terminal command"

	default:
		return fmt.Sprintf("Execute tool: %s", tool)
	}
}

// isDangerousTool determines if a tool is dangerous (requires approval)
func isDangerousTool(tool string) bool {
	dangerousTools := map[string]bool{
		"file.write":       true,
		"file.delete":      true,
		"terminal.execute": true,
	}
	return dangerousTools[tool]
}

// randString generates a random string of given length
func randString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

