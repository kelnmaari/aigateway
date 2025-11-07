// Package agent provides Agentic AI orchestration service (v2.5.0+)
package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/storage"
	"aigateway/internal/services/agent/tools"
)

// Service handles agent session management and task orchestration
type Service struct {
	db       storage.Database
	logger   *logrus.Logger
	sessions map[string]*models.AgentSession // In-memory session cache
	mu       sync.RWMutex
	
	// Tool execution (v2.5.0+)
	toolRegistry  *tools.Registry
	
	// Approval management (v2.5.0+)
	approvalManager *ApprovalManager
	
	// LLM Integration for planning (v2.5.0+)
	ollamaURL  string // Ollama server URL
	planModel  string // Model to use for task planning
	
	// Configuration
	maxSteps      int
	maxSessions   int
	sessionTTL    time.Duration
	toolTimeout   time.Duration
	
	// Event callbacks for WebSocket streaming
	onEvent func(*models.AgentEvent)
}

// Config holds agent service configuration
type Config struct {
	MaxSteps      int           // Maximum steps per plan (default: 50)
	MaxSessions   int           // Maximum concurrent sessions (default: 100)
	SessionTTL    time.Duration // Session time-to-live (default: 5 minutes)
	ToolTimeout   time.Duration // Tool execution timeout (default: 60 seconds)
	OllamaURL     string        // Ollama server URL for LLM planning (default: http://localhost:11434)
	PlanModel     string        // Model to use for planning (default: qwen2.5:14b)
}

// DefaultConfig returns default agent service configuration
func DefaultConfig() Config {
	return Config{
		MaxSteps:    50,
		MaxSessions: 100,
		SessionTTL:  5 * time.Minute,
		ToolTimeout: 60 * time.Second,
		OllamaURL:   "http://localhost:11434",
		PlanModel:   "deepseek-r1:1.5b", // Available model for task planning
	}
}

// NewService creates a new agent service
func NewService(db storage.Database, logger *logrus.Logger, cfg Config) *Service {
	if cfg.MaxSteps == 0 {
		cfg.MaxSteps = 50
	}
	if cfg.MaxSessions == 0 {
		cfg.MaxSessions = 100
	}
	if cfg.SessionTTL == 0 {
		cfg.SessionTTL = 5 * time.Minute
	}
	if cfg.ToolTimeout == 0 {
		cfg.ToolTimeout = 60 * time.Second
	}

	// Initialize tool registry
	toolRegistry := tools.NewRegistry(logger)
	
	// Register file tools (for current working directory)
	if err := tools.RegisterFileTools(toolRegistry, "."); err != nil {
		logger.WithError(err).Warn("Failed to register file tools")
	}
	
	// Register terminal tool
	if err := tools.RegisterTerminalTool(toolRegistry, "."); err != nil {
		logger.WithError(err).Warn("Failed to register terminal tool")
	}
	
	logger.WithField("tool_count", toolRegistry.Count()).Info("Tool registry initialized")
	
	// Initialize approval manager
	approvalCfg := DefaultApprovalConfig()
	approvalManager := NewApprovalManager(logger, approvalCfg)
	logger.Info("Approval manager initialized")

	return &Service{
		db:              db,
		logger:          logger,
		sessions:        make(map[string]*models.AgentSession),
		toolRegistry:    toolRegistry,
		approvalManager: approvalManager,
		ollamaURL:       cfg.OllamaURL,
		planModel:       cfg.PlanModel,
		maxSteps:        cfg.MaxSteps,
		maxSessions:     cfg.MaxSessions,
		sessionTTL:      cfg.SessionTTL,
		toolTimeout:     cfg.ToolTimeout,
	}
}

// SetEventCallback sets the callback function for agent events (WebSocket streaming)
func (s *Service) SetEventCallback(callback func(*models.AgentEvent)) {
	s.onEvent = callback
}

// GetToolRegistry returns the tool registry (v2.5.1+: for WebSocket agent support)
func (s *Service) GetToolRegistry() *tools.Registry {
	return s.toolRegistry
}

// CreateSession creates a new agent session and generates a task plan
func (s *Service) CreateSession(ctx context.Context, userID string, tenantID *string, req models.CreateAgentSessionRequest) (*models.AgentSession, *models.AgentPlan, error) {
	s.mu.Lock()
	if len(s.sessions) >= s.maxSessions {
		s.mu.Unlock()
		return nil, nil, fmt.Errorf("maximum concurrent sessions reached (%d)", s.maxSessions)
	}
	s.mu.Unlock()

	// Create session
	sessionID := uuid.New().String()
	now := time.Now()

	session := &models.AgentSession{
		ID:          sessionID,
		UserID:      userID,
		TenantID:    tenantID,
		Task:        req.Task,
		Context:     req.Context,
		Status:      models.AgentStatusPlanning,
		CurrentStep: 0,
		TotalSteps:  0,
		StartedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Store in memory cache
	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	s.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"user_id":    userID,
		"task":       req.Task,
	}).Info("Agent session created")

	// Emit thinking event
	s.emitEvent(&models.AgentEvent{
		Type:      models.AgentEventThinking,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": "Analyzing task and creating plan...",
		},
	})

	// Generate plan (AI planning logic would go here)
	plan, err := s.generatePlan(ctx, session, req)
	if err != nil {
		session.Status = models.AgentStatusFailed
		errMsg := err.Error()
		session.Error = &errMsg
		s.updateSession(session)
		return nil, nil, fmt.Errorf("failed to generate plan: %w", err)
	}

	// Update session with plan
	session.Plan = plan
	session.TotalSteps = len(plan.Steps)
	session.Status = models.AgentStatusPending
	session.UpdatedAt = time.Now()
	s.updateSession(session)

	// Emit plan created event
	s.emitEvent(&models.AgentEvent{
		Type:      models.AgentEventPlanCreated,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"plan":        plan,
			"total_steps": len(plan.Steps),
		},
	})

	s.logger.WithFields(logrus.Fields{
		"session_id": sessionID,
		"steps":      len(plan.Steps),
	}).Info("Agent plan created")

	// AUTO-START: Execute plan automatically in background (AGENT-02, v2.5.0+)
	go func() {
		s.logger.WithField("session_id", sessionID).Info("Auto-starting agent execution")
		if err := s.executeSessionSteps(context.Background(), session); err != nil {
			s.logger.WithError(err).WithField("session_id", sessionID).Error("Auto-execution failed")
		}
	}()

	return session, plan, nil
}

// executeSessionSteps executes all steps in an agent session sequentially
func (s *Service) executeSessionSteps(ctx context.Context, session *models.AgentSession) error {
	s.logger.WithFields(logrus.Fields{
		"session_id":  session.ID,
		"total_steps": session.TotalSteps,
	}).Info("Starting session execution")

	// Update status to executing
	session.Status = models.AgentStatusExecuting
	session.UpdatedAt = time.Now()
	s.updateSession(session)

	// Emit execution started event
	s.emitEvent(&models.AgentEvent{
		Type:      models.AgentEventExecutionStarted,
		SessionID: session.ID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"total_steps": session.TotalSteps,
		},
	})

	// Execute each step sequentially
	for i, step := range session.Plan.Steps {
		// Check if session was cancelled
		s.mu.RLock()
		currentSession, exists := s.sessions[session.ID]
		s.mu.RUnlock()
		
		if !exists || currentSession.Status == models.AgentStatusCancelled {
			s.logger.WithField("session_id", session.ID).Info("Session cancelled, stopping execution")
			return fmt.Errorf("session cancelled")
		}

		// Update current step
		session.CurrentStep = i + 1
		session.UpdatedAt = time.Now()
		s.updateSession(session)

		// Execute step (using Iteration 2 ExecuteStep)
		_, err := s.ExecuteStep(ctx, session.ID, step.StepNumber)
		if err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"session_id": session.ID,
				"step_id":    step.ID,
				"step_num":   step.StepNumber,
			}).Error("Step execution failed")
			
			// Mark session as failed
			session.Status = models.AgentStatusFailed
			errMsg := fmt.Sprintf("Step %d failed: %v", step.StepNumber, err)
			session.Error = &errMsg
			session.UpdatedAt = time.Now()
			s.updateSession(session)
			
			return err
		}
	}

	// All steps completed successfully
	session.Status = models.AgentStatusCompleted
	now := time.Now()
	session.CompletedAt = &now
	session.UpdatedAt = now
	s.updateSession(session)

	// Emit session completed event
	s.emitEvent(&models.AgentEvent{
		Type:      models.AgentEventSessionCompleted,
		SessionID: session.ID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"total_steps":     session.TotalSteps,
			"completed_steps": session.CurrentStep,
		},
	})

	s.logger.WithField("session_id", session.ID).Info("Session execution completed successfully")
	return nil
}

// GetSession retrieves an agent session by ID
func (s *Service) GetSession(ctx context.Context, sessionID string) (*models.AgentSession, error) {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// CancelSession cancels a running agent session
func (s *Service) CancelSession(ctx context.Context, sessionID string) error {
	s.mu.Lock()
	session, exists := s.sessions[sessionID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Update status
	session.Status = models.AgentStatusCancelled
	now := time.Now()
	session.CompletedAt = &now
	session.UpdatedAt = now
	s.mu.Unlock()

	// Emit cancelled event
	s.emitEvent(&models.AgentEvent{
		Type:      models.AgentEventCancelled,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"message": "Session cancelled by user",
		},
	})

	s.logger.WithField("session_id", sessionID).Info("Agent session cancelled")

	return nil
}

// Note: generatePlan and simpleTaskDecomposition moved to planner.go (v2.5.0+)

// updateSession updates session in cache
func (s *Service) updateSession(session *models.AgentSession) {
	s.mu.Lock()
	s.sessions[session.ID] = session
	s.mu.Unlock()
}

// emitEvent emits an agent event for WebSocket streaming
func (s *Service) emitEvent(event *models.AgentEvent) {
	if s.onEvent != nil {
		s.onEvent(event)
	}
}

// CleanupExpiredSessions removes expired sessions from cache
func (s *Service) CleanupExpiredSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, session := range s.sessions {
		if session.Status == models.AgentStatusCompleted ||
			session.Status == models.AgentStatusFailed ||
			session.Status == models.AgentStatusCancelled {
			if session.CompletedAt != nil && now.Sub(*session.CompletedAt) > s.sessionTTL {
				delete(s.sessions, id)
				s.logger.WithField("session_id", id).Debug("Cleaned up expired session")
			}
		}
	}
}

// ========================================
// Tool Execution (v2.5.0+)
// ========================================

// ExecuteStep executes a single agent step using tools
func (s *Service) ExecuteStep(ctx context.Context, sessionID string, stepNumber int) (*models.AgentStepResult, error) {
	session, err := s.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session.Plan == nil || len(session.Plan.Steps) == 0 {
		return nil, fmt.Errorf("session has no plan")
	}

	if stepNumber < 1 || stepNumber > len(session.Plan.Steps) {
		return nil, fmt.Errorf("invalid step number: %d", stepNumber)
	}

	step := &session.Plan.Steps[stepNumber-1]

	// Check if tool is dangerous and requires approval
	tool, err := s.toolRegistry.Get(step.Tool)
	if err != nil {
		errMsg := err.Error()
		step.Status = models.AgentStepStatusFailed
		step.Error = &errMsg
		s.updateSession(session)
		return nil, err
	}

	toolInfo := tool.GetInfo()
	if toolInfo.Dangerous {
		// Create approval request
		approvalReq, err := s.approvalManager.CreateApprovalRequest(
			sessionID,
			stepNumber,
			step.Description,
			step.Action,
			step.Tool,
			step.Parameters,
			fmt.Sprintf("Tool '%s' requires approval (dangerous operation)", step.Tool),
		)
		if err != nil {
			errMsg := fmt.Sprintf("failed to create approval request: %v", err)
			step.Status = models.AgentStepStatusFailed
			step.Error = &errMsg
			s.updateSession(session)
			return nil, err
		}

		// Emit approval needed event
		s.emitEvent(&models.AgentEvent{
			Type:      models.AgentEventApprovalNeeded,
			SessionID: sessionID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"approval_id": approvalReq.ID,
				"step_number": stepNumber,
				"tool":        step.Tool,
				"reason":      approvalReq.Reason,
			},
		})

		s.logger.WithFields(logrus.Fields{
			"session_id":  sessionID,
			"step_number": stepNumber,
			"tool":        step.Tool,
			"approval_id": approvalReq.ID,
		}).Info("Waiting for approval")

		// Wait for approval
		approved, err := s.approvalManager.WaitForApproval(ctx, approvalReq.ID)
		if err != nil {
			errMsg := fmt.Sprintf("approval failed: %v", err)
			step.Status = models.AgentStepStatusFailed
			step.Error = &errMsg
			s.updateSession(session)
			return nil, err
		}

		if !approved {
			// Step rejected
			step.Status = models.AgentStepStatusSkipped
			errMsg := "step rejected by user"
			step.Error = &errMsg
			s.updateSession(session)

			return &models.AgentStepResult{
				Success: false,
				Error:   &errMsg,
			}, nil
		}

		s.logger.WithFields(logrus.Fields{
			"session_id":  sessionID,
			"step_number": stepNumber,
			"approval_id": approvalReq.ID,
		}).Info("Step approved, executing")
	}

	// Update step status
	step.Status = models.AgentStepStatusRunning
	now := time.Now()
	step.StartedAt = &now
	s.updateSession(session)

	// Emit step start event
	s.emitEvent(&models.AgentEvent{
		Type:      models.AgentEventStepStart,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"step_number": stepNumber,
			"description": step.Description,
			"action":      step.Action,
		},
	})

	// Execute step using tool
	result, err := s.toolRegistry.Execute(ctx, step.Tool, step.Parameters, s.toolTimeout)
	if err != nil {
		// Tool execution failed
		errMsg := err.Error()
		step.Status = models.AgentStepStatusFailed
		step.Error = &errMsg
		completed := time.Now()
		step.CompletedAt = &completed
		s.updateSession(session)

		s.emitEvent(&models.AgentEvent{
			Type:      models.AgentEventStepFailed,
			SessionID: sessionID,
			Timestamp: time.Now(),
			Data: map[string]interface{}{
				"step_number": stepNumber,
				"error":       err.Error(),
			},
		})

		return result, err
	}

	// Update step with result
	step.Result = result
	completed := time.Now()
	step.CompletedAt = &completed

	if result.Success {
		step.Status = models.AgentStepStatusCompleted
	} else {
		step.Status = models.AgentStepStatusFailed
	}

	s.updateSession(session)

	// Emit step complete event
	s.emitEvent(&models.AgentEvent{
		Type:      models.AgentEventStepComplete,
		SessionID: sessionID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"step_number": stepNumber,
			"success":     result.Success,
			"output":      result.Output,
		},
	})

	s.logger.WithFields(logrus.Fields{
		"session_id":  sessionID,
		"step_number": stepNumber,
		"tool":        step.Tool,
		"success":     result.Success,
	}).Info("Step executed")

	return result, nil
}

// GetAvailableTools returns list of available tools
func (s *Service) GetAvailableTools() []models.AgentTool {
	return s.toolRegistry.List()
}

// GetToolsByCategory returns tools filtered by category
func (s *Service) GetToolsByCategory(category models.AgentToolCategory) []models.AgentTool {
	return s.toolRegistry.ListByCategory(category)
}

// ========================================
// Approval Management (v2.5.0+)
// ========================================

// GetPendingApprovals returns all pending approvals for a session
func (s *Service) GetPendingApprovals(sessionID string) []*models.ApprovalRequest {
	return s.approvalManager.GetPendingApprovals(sessionID)
}

// ApproveStep approves or rejects a pending step
func (s *Service) ApproveStep(approvalID string, decision models.ApprovalDecision) error {
	return s.approvalManager.ApproveRequest(approvalID, decision)
}

