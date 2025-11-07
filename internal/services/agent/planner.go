// Package agent provides LLM-based task planning (v2.5.0+)
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
)

// generatePlan generates a task decomposition plan using LLM
func (s *Service) generatePlan(ctx context.Context, session *models.AgentSession, req models.CreateAgentSessionRequest) (*models.AgentPlan, error) {
	maxSteps := s.maxSteps
	if req.MaxSteps != nil && *req.MaxSteps > 0 && *req.MaxSteps < s.maxSteps {
		maxSteps = *req.MaxSteps
	}

	// Determine which model to use for planning
	planModel := s.planModel // Default from config
	if req.Model != "" {
		planModel = req.Model // User-specified model overrides default
	}

	// Get available tools from registry
	availableTools := s.toolRegistry.List()

	// Generate plan using LLM
	steps, err := s.llmBasedPlanning(ctx, session.Task, req.Context, availableTools, maxSteps, planModel)
	if err != nil {
		s.logger.WithError(err).Error("LLM planning failed, falling back to simple planning")
		// Fallback to simple rule-based planning
		steps = s.simpleTaskDecomposition(session.Task, maxSteps)
	}

	plan := &models.AgentPlan{
		ID:        uuid.New().String(),
		SessionID: session.ID,
		Steps:     steps,
		CreatedAt: time.Now(),
	}

	// Estimate time (rough estimate: 15 seconds per step)
	estimatedTime := len(steps) * 15
	plan.EstimatedTime = &estimatedTime

	return plan, nil
}

// llmBasedPlanning uses LLM to generate intelligent task plan
func (s *Service) llmBasedPlanning(ctx context.Context, task string, taskContext map[string]interface{}, availableTools []models.AgentTool, maxSteps int, model string) ([]models.AgentStep, error) {
	// Build tools description for prompt
	toolsJSON, err := json.Marshal(availableTools)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tools: %w", err)
	}

	// Build context string
	contextStr := ""
	if len(taskContext) > 0 {
		contextJSON, _ := json.Marshal(taskContext)
		contextStr = fmt.Sprintf("\n\nAdditional Context:\n%s", string(contextJSON))
	}

	// Construct planning prompt
	prompt := fmt.Sprintf(`You are an AI task planning assistant. Your job is to decompose a user task into executable steps using available tools.

**Available Tools:**
%s

**User Task:**
%s%s

**Instructions:**
1. **Carefully read the task** and create a MINIMAL, EFFICIENT plan (aim for 1-3 steps)
2. **DO NOT** create redundant or repetitive steps
3. **PREFER file tools over terminal** - use file.read instead of terminal commands like cat/type
4. For each step:
   - Use ONLY tools from the available tools list above (exact names!)
   - Provide clear, concise description
   - Specify exact tool name from "name" field (e.g., "file.read", "file.write")
   - Include CORRECT parameters matching the tool's parameter schema
5. **CRITICAL - File paths**:
   - Use RELATIVE paths (e.g., "README.md", "src/main.go")
   - DO NOT use absolute paths starting with "/" (e.g., "/README.md" is WRONG)
   - Case-sensitive filenames (README.md not README.MD)
6. **Tool preference**:
   - For reading files: use "file.read" (not terminal.execute)
   - For listing files: use "file.list" (not terminal.execute)
   - Only use terminal.execute for actual shell operations
7. Dangerous tools (file.delete, file.write, terminal.execute) require user approval

**Output Format (MUST be valid JSON with "steps" array, optionally wrapped in code block):**
{
  "steps": [
    {
      "step_number": 1,
      "description": "Read README.md file",
      "tool": "file.read",
      "parameters": {"path": "README.md"}
    }
  ]
}

IMPORTANT: Response MUST be an object with "steps" key, NOT a direct array!

Generate the plan now:`, string(toolsJSON), task, contextStr, maxSteps)

	// Call Ollama API with specified model
	planResponse, err := s.callOllamaForPlanning(ctx, prompt, model)
	if err != nil {
		return nil, fmt.Errorf("Ollama API call failed: %w", err)
	}

	// Parse response
	steps, err := s.parsePlanResponse(planResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plan response: %w", err)
	}

	return steps, nil
}

// OllamaChatRequest represents Ollama API request
type OllamaChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
	Options  Options   `json:"options,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Options struct {
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
}

// OllamaChatResponse represents Ollama API response
type OllamaChatResponse struct {
	Message Message `json:"message"`
}

// callOllamaForPlanning calls Ollama API to generate plan
func (s *Service) callOllamaForPlanning(ctx context.Context, prompt string, model string) (string, error) {
	reqBody := OllamaChatRequest{
		Model: model,
		Messages: []Message{
			{
				Role:    "system",
				Content: "You are an expert task planning assistant. Create MINIMAL, EFFICIENT plans. Respond with valid JSON (optionally wrapped in code block). Be concise and practical.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
		Options: Options{
			Temperature: 0.1, // Very low temperature for consistent, focused planning
			TopP:        0.8,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/chat", s.ollamaURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Set timeout
	client := &http.Client{Timeout: 60 * time.Second}

	s.logger.WithFields(logrus.Fields{
		"url":   url,
		"model": model,
	}).Debug("Calling Ollama for task planning")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var ollamaResp OllamaChatResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return ollamaResp.Message.Content, nil
}

// PlanResponseFormat represents LLM's plan output
type PlanResponseFormat struct {
	Steps []struct {
		StepNumber  int                    `json:"step_number"`
		Description string                 `json:"description"`
		Tool        string                 `json:"tool"`
		Parameters  map[string]interface{} `json:"parameters"`
	} `json:"steps"`
}

// parsePlanResponse parses LLM response into AgentStep array
func (s *Service) parsePlanResponse(response string) ([]models.AgentStep, error) {
	// Try to extract JSON if wrapped in markdown code blocks
	cleanedResponse := response
	responseBytes := []byte(response)

	// Method 1: Extract from ```json block
	if jsonStart := bytes.Index(responseBytes, []byte("```json")); jsonStart >= 0 {
		jsonStart += 7 // Skip "```json"
		// Skip any whitespace/newlines after ```json
		for jsonStart < len(responseBytes) && (responseBytes[jsonStart] == '\n' || responseBytes[jsonStart] == '\r' || responseBytes[jsonStart] == ' ') {
			jsonStart++
		}

		if jsonEnd := bytes.Index(responseBytes[jsonStart:], []byte("```")); jsonEnd > 0 {
			cleanedResponse = string(responseBytes[jsonStart : jsonStart+jsonEnd])
		}
	} else if codeStart := bytes.Index(responseBytes, []byte("```")); codeStart >= 0 {
		// Method 2: Extract from generic ``` block
		codeStart += 3
		// Skip any whitespace/newlines
		for codeStart < len(responseBytes) && (responseBytes[codeStart] == '\n' || responseBytes[codeStart] == '\r' || responseBytes[codeStart] == ' ') {
			codeStart++
		}

		if codeEnd := bytes.Index(responseBytes[codeStart:], []byte("```")); codeEnd > 0 {
			cleanedResponse = string(responseBytes[codeStart : codeStart+codeEnd])
		}
	}

	// Trim any remaining whitespace
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	var planResp PlanResponseFormat

	// Try parsing as object with "steps" field first
	if err := json.Unmarshal([]byte(cleanedResponse), &planResp); err != nil {
		// Fallback: Try parsing as direct array of steps
		var stepsArray []struct {
			StepNumber  int                    `json:"step_number"`
			Description string                 `json:"description"`
			Tool        string                 `json:"tool"`
			Parameters  map[string]interface{} `json:"parameters"`
		}

		if err2 := json.Unmarshal([]byte(cleanedResponse), &stepsArray); err2 != nil {
			s.logger.WithError(err).WithField("response", response).Error("Failed to parse LLM plan response")
			return nil, fmt.Errorf("invalid JSON response from LLM: %w", err)
		}

		// Convert array to PlanResponseFormat
		planResp.Steps = stepsArray
		s.logger.Debug("LLM returned steps as direct array, converted to object format")
	}

	if len(planResp.Steps) == 0 {
		return nil, fmt.Errorf("LLM returned empty plan")
	}

	// Convert to AgentStep format
	steps := make([]models.AgentStep, 0, len(planResp.Steps))
	for _, stepData := range planResp.Steps {
		// Validate tool exists
		_, err := s.toolRegistry.Get(stepData.Tool)
		if err != nil {
			s.logger.WithFields(logrus.Fields{
				"step_num": stepData.StepNumber,
				"tool":     stepData.Tool,
			}).Warn("LLM specified non-existent tool, skipping step")
			continue
		}

		// Determine action based on tool name (tools use dot notation: file.read, terminal.execute)
		action := models.AgentActionThinking
		switch stepData.Tool {
		case "file.read":
			action = models.AgentActionFileRead
		case "file.write", "file.create":
			action = models.AgentActionFileWrite
		case "file.list":
			action = models.AgentActionFileList
		case "file.delete":
			action = models.AgentActionFileDelete
		case "terminal.execute":
			action = models.AgentActionTerminalExecute
		case "mcp.invoke":
			action = models.AgentActionMCPInvoke
		}

		step := models.AgentStep{
			ID:          uuid.New().String(),
			StepNumber:  stepData.StepNumber,
			Description: stepData.Description,
			Action:      action,
			Tool:        stepData.Tool,
			Parameters:  stepData.Parameters,
			Status:      models.AgentStepStatusPending,
		}

		steps = append(steps, step)
	}

	if len(steps) == 0 {
		return nil, fmt.Errorf("no valid steps generated by LLM")
	}

	s.logger.WithField("steps_count", len(steps)).Info("LLM plan generated successfully")
	return steps, nil
}

// simpleTaskDecomposition performs simple rule-based task decomposition (fallback)
func (s *Service) simpleTaskDecomposition(task string, maxSteps int) []models.AgentStep {
	// MVP: Very basic decomposition using real tools (with correct dot notation and parameters)
	steps := []models.AgentStep{
		{
			ID:          uuid.New().String(),
			StepNumber:  1,
			Description: fmt.Sprintf("List files to understand project structure"),
			Action:      models.AgentActionFileList,
			Tool:        "file.list",                                             // Correct tool name with dot
			Parameters:  map[string]interface{}{"path": ".", "recursive": false}, // Correct parameter names
			Status:      models.AgentStepStatusPending,
		},
	}

	// Limit to maxSteps
	if len(steps) > maxSteps {
		steps = steps[:maxSteps]
	}

	return steps
}
