// Package agent provides Agent service for orchestrating ReAct loops (v2.5.0+)
package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/services/agent/tools"
)

// AgentService orchestrates conversational agent with ReAct loop
// Tools are NOT executed on server - only orchestration and reasoning
type AgentService struct {
	toolRegistry *tools.Registry
	logger       *logrus.Logger
	mu           sync.RWMutex

	// Active agent sessions (sessionID -> AgentSession)
	sessions map[string]*models.AgentContext
}

// NewAgentService creates a new agent service
func NewAgentService(logger *logrus.Logger) *AgentService {
	toolRegistry := tools.NewRegistry(logger)

	// Register default tools (descriptor-only for client execution)
	registerDefaultTools(toolRegistry)

	return &AgentService{
		toolRegistry: toolRegistry,
		logger:       logger,
		sessions:     make(map[string]*models.AgentContext),
	}
}

// registerDefaultTools registers built-in tools (v3.0.6+: client-side execution only)
func registerDefaultTools(registry *tools.Registry) {
	// System tools
	registry.Register(tools.NewThinkTool()) // Pseudo-tool for non-reasoning models to pause and reflect
	
	// File tools
	registry.Register(tools.NewFileReadTool(""))    // Empty baseDir = no restriction
	registry.Register(tools.NewFileWriteTool(""))
	registry.Register(tools.NewFileDeleteTool(""))
	registry.Register(tools.NewFileListTool(""))

	// Terminal tool
	registry.Register(tools.NewTerminalTool("")) // Empty workDir = current directory

	// MCP tools will be registered dynamically when MCP servers are configured
}

// GetToolRegistry returns the tool registry
func (s *AgentService) GetToolRegistry() *tools.Registry {
	return s.toolRegistry
}

// ListTools returns all available tools for client
func (s *AgentService) ListTools() []models.AgentTool {
	return s.toolRegistry.List()
}

// GetToolsByCategory returns tools filtered by category
func (s *AgentService) GetToolsByCategory(category models.AgentToolCategory) []models.AgentTool {
	return s.toolRegistry.ListByCategory(category)
}

// CreateAgentSession creates a new agent session context
func (s *AgentService) CreateAgentSession(ctx context.Context, conversationID string, task string) (*models.AgentContext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	agentCtx := models.NewAgentContext(task)
	s.sessions[conversationID] = agentCtx

	s.logger.WithFields(logrus.Fields{
		"conversation_id": conversationID,
		"task":            task,
	}).Info("Agent session created")

	return agentCtx, nil
}

// GetAgentSession retrieves agent session context
func (s *AgentService) GetAgentSession(conversationID string) (*models.AgentContext, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agentCtx, exists := s.sessions[conversationID]
	if !exists {
		return nil, fmt.Errorf("agent session not found: %s", conversationID)
	}

	return agentCtx, nil
}

// UpdateAgentSession updates agent session context
func (s *AgentService) UpdateAgentSession(conversationID string, agentCtx *models.AgentContext) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[conversationID]; !exists {
		return fmt.Errorf("agent session not found: %s", conversationID)
	}

	s.sessions[conversationID] = agentCtx

	s.logger.WithFields(logrus.Fields{
		"conversation_id": conversationID,
		"step":            agentCtx.StepNumber,
		"complete":        agentCtx.IsComplete,
	}).Debug("Agent session updated")

	return nil
}

// CloseAgentSession closes and removes agent session
func (s *AgentService) CloseAgentSession(conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[conversationID]; !exists {
		return fmt.Errorf("agent session not found: %s", conversationID)
	}

	delete(s.sessions, conversationID)

	s.logger.WithField("conversation_id", conversationID).Info("Agent session closed")

	return nil
}

// GenerateSystemPrompt generates ReAct system prompt with available tools
func (s *AgentService) GenerateSystemPrompt(workingDir string) string {
	availableTools := s.toolRegistry.List()

	prompt := `You are an AI agent operating in ReAct (Reasoning + Acting) mode.

## Your Task
Analyze user requests, break them into steps, and use available tools to accomplish tasks.

## Available Tools
You have access to the following tools (executed on client-side):

`

	for _, tool := range availableTools {
		dangerous := ""
		if tool.Dangerous {
			dangerous = " [⚠️ REQUIRES APPROVAL]"
		}

		prompt += fmt.Sprintf("- **%s**%s: %s\n", tool.Name, dangerous, tool.Description)
	}

	prompt += `
## ReAct Loop
Follow this pattern:

1. **Thought**: Analyze what needs to be done
2. **Action**: Choose a tool and provide parameters in JSON format
3. **Observation**: Receive tool execution result from client
4. Repeat until task is complete

## Output Format

When you need to use a tool:
` + "```json" + `
{
  "thought": "Your reasoning about what to do next",
  "action": "tool.name",
  "parameters": {
    "param1": "value1",
    "param2": "value2"
  }
}
` + "```" + `

When task is complete:
` + "```json" + `
{
  "thought": "Task completed successfully",
  "final_answer": "Your response to the user"
}
` + "```" + `

## Important Rules
- Always explain your reasoning in "thought"
- Use ONE tool at a time
- Wait for observation before next action
- Dangerous tools (file.write, file.delete, terminal.execute) require user approval
- Be concise and focused

## 💡 Using the "think" Tool
The **think** tool is special - it doesn't execute any action. Instead, it helps you:
- **Pause and reflect** on complex problems before acting
- **Reconsider your approach** if you feel uncertain
- **Break down** multi-step problems into clear sub-tasks
- **Avoid hasty decisions** by forcing yourself to articulate your reasoning

Use think() when:
- The problem seems complex or ambiguous
- You've tried something that didn't work and need to rethink
- You're about to use a dangerous tool (write/delete/execute)
- You need to organize your thoughts before multiple operations

Example:
` + "```json" + `
{
  "thought": "This task requires multiple file operations. Let me think through the sequence.",
  "action": "think",
  "parameters": {
    "reflection": "I need to: 1) List files to see what exists, 2) Read specific files to check content, 3) Decide which files to modify. I should NOT rush into deleting anything until I understand the full context."
  }
}
` + "```" + `
`

	if workingDir != "" {
		prompt += fmt.Sprintf("\n## Working Directory\nYou are operating in: `%s`\n", workingDir)
	}

	return prompt
}

