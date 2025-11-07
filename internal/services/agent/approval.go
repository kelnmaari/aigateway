// Package agent provides approval management for dangerous agent operations (v2.5.0+)
package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
)

// ApprovalManager handles approval requests for dangerous operations
type ApprovalManager struct {
	approvals map[string]*models.ApprovalRequest // approval ID -> request
	mu        sync.RWMutex
	logger    *logrus.Logger
	
	// Configuration
	approvalTimeout time.Duration // Time before approval expires
}

// ApprovalConfig holds approval manager configuration
type ApprovalConfig struct {
	ApprovalTimeout time.Duration // Default: 5 minutes
}

// DefaultApprovalConfig returns default configuration
func DefaultApprovalConfig() ApprovalConfig {
	return ApprovalConfig{
		ApprovalTimeout: 5 * time.Minute,
	}
}

// NewApprovalManager creates a new approval manager
func NewApprovalManager(logger *logrus.Logger, cfg ApprovalConfig) *ApprovalManager {
	if cfg.ApprovalTimeout == 0 {
		cfg.ApprovalTimeout = 5 * time.Minute
	}

	return &ApprovalManager{
		approvals:       make(map[string]*models.ApprovalRequest),
		logger:          logger,
		approvalTimeout: cfg.ApprovalTimeout,
	}
}

// CreateApprovalRequest creates a new approval request
func (am *ApprovalManager) CreateApprovalRequest(
	sessionID string,
	stepNumber int,
	description string,
	action models.AgentAction,
	tool string,
	params map[string]interface{},
	reason string,
) (*models.ApprovalRequest, error) {
	
	approvalID := uuid.New().String()
	
	request := &models.ApprovalRequest{
		ID:          approvalID,
		SessionID:   sessionID,
		StepNumber:  stepNumber,
		Description: description,
		Action:      action,
		Tool:        tool,
		Parameters:  params,
		Reason:      reason,
		Status:      models.ApprovalStatusPending,
		CreatedAt:   time.Now(),
	}
	
	am.mu.Lock()
	am.approvals[approvalID] = request
	am.mu.Unlock()
	
	am.logger.WithFields(logrus.Fields{
		"approval_id": approvalID,
		"session_id":  sessionID,
		"step":        stepNumber,
		"tool":        tool,
		"reason":      reason,
	}).Info("Approval request created")
	
	return request, nil
}

// GetApprovalRequest retrieves approval request by ID
func (am *ApprovalManager) GetApprovalRequest(approvalID string) (*models.ApprovalRequest, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	request, exists := am.approvals[approvalID]
	if !exists {
		return nil, fmt.Errorf("approval request not found: %s", approvalID)
	}
	
	// Check if expired
	if request.Status == models.ApprovalStatusPending {
		if time.Since(request.CreatedAt) > am.approvalTimeout {
			request.Status = models.ApprovalStatusExpired
		}
	}
	
	return request, nil
}

// GetPendingApprovals returns all pending approvals for a session
func (am *ApprovalManager) GetPendingApprovals(sessionID string) []*models.ApprovalRequest {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var pending []*models.ApprovalRequest
	for _, request := range am.approvals {
		if request.SessionID == sessionID && request.Status == models.ApprovalStatusPending {
			// Check if expired
			if time.Since(request.CreatedAt) > am.approvalTimeout {
				request.Status = models.ApprovalStatusExpired
			} else {
				pending = append(pending, request)
			}
		}
	}
	
	return pending
}

// ApproveRequest approves an approval request
func (am *ApprovalManager) ApproveRequest(approvalID string, decision models.ApprovalDecision) error {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	request, exists := am.approvals[approvalID]
	if !exists {
		return fmt.Errorf("approval request not found: %s", approvalID)
	}
	
	// Check if already responded
	if request.Status != models.ApprovalStatusPending {
		return fmt.Errorf("approval request already %s", request.Status)
	}
	
	// Check if expired
	if time.Since(request.CreatedAt) > am.approvalTimeout {
		request.Status = models.ApprovalStatusExpired
		return fmt.Errorf("approval request expired")
	}
	
	// Update status
	now := time.Now()
	request.RespondedAt = &now
	request.Decision = &decision
	
	if decision.Approved {
		request.Status = models.ApprovalStatusApproved
	} else {
		request.Status = models.ApprovalStatusRejected
	}
	
	am.logger.WithFields(logrus.Fields{
		"approval_id": approvalID,
		"session_id":  request.SessionID,
		"approved":    decision.Approved,
		"comment":     decision.Comment,
	}).Info("Approval decision recorded")
	
	return nil
}

// WaitForApproval blocks until approval is granted or rejected
func (am *ApprovalManager) WaitForApproval(ctx context.Context, approvalID string) (bool, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	
	timeout := time.After(am.approvalTimeout)
	
	for {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-timeout:
			// Mark as expired
			am.mu.Lock()
			if request, exists := am.approvals[approvalID]; exists {
				request.Status = models.ApprovalStatusExpired
			}
			am.mu.Unlock()
			return false, fmt.Errorf("approval timeout")
		case <-ticker.C:
			// Check status
			request, err := am.GetApprovalRequest(approvalID)
			if err != nil {
				return false, err
			}
			
			switch request.Status {
			case models.ApprovalStatusApproved:
				return true, nil
			case models.ApprovalStatusRejected:
				return false, nil
			case models.ApprovalStatusExpired:
				return false, fmt.Errorf("approval expired")
			case models.ApprovalStatusPending:
				// Continue waiting
				continue
			}
		}
	}
}

// CleanupExpiredApprovals removes expired approval requests
func (am *ApprovalManager) CleanupExpiredApprovals() {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	now := time.Now()
	for id, request := range am.approvals {
		// Remove if completed and older than 1 hour
		if request.Status != models.ApprovalStatusPending {
			if request.RespondedAt != nil && now.Sub(*request.RespondedAt) > time.Hour {
				delete(am.approvals, id)
				am.logger.WithField("approval_id", id).Debug("Cleaned up old approval request")
			}
		} else {
			// Mark as expired if timeout passed
			if now.Sub(request.CreatedAt) > am.approvalTimeout {
				request.Status = models.ApprovalStatusExpired
			}
		}
	}
}

// Count returns the number of approval requests
func (am *ApprovalManager) Count() int {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return len(am.approvals)
}

