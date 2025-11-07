// Package agent provides conversational agent with ReAct loop
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

	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
	"aigateway/internal/services/agent/tools"
)

// ConversationalAgent implements ReAct loop for agent conversations
// ReAct: Reasoning + Acting - iterative loop where agent thinks, acts, observes, and repeats
type ConversationalAgent struct {
	logger        *logrus.Logger
	toolRegistry  *tools.Registry
	ollamaURL     string
	model         string // Model to use for reasoning
	maxIterations int    // Max ReAct loop iterations

	// RPC Tool Client (v2.5.4+: Client-side tool execution)
	rpcClient *ToolRPCClient // RPC client for executing tools on desktop client

	// Event streaming callback
	onThought     func(*models.AgentThought)
	onAction      func(*models.AgentActionRecord)
	onObservation func(string)
	onComplete    func(bool, string) // success, result
}

// ConversationalAgentConfig configures the conversational agent
type ConversationalAgentConfig struct {
	OllamaURL     string
	Model         string // Model for reasoning (e.g., "qwen2.5:14b", "deepseek-r1:1.5b")
	MaxIterations int    // Max ReAct loop iterations (default: 10)
}

// NewConversationalAgent creates a new conversational agent
func NewConversationalAgent(logger *logrus.Logger, toolRegistry *tools.Registry, cfg ConversationalAgentConfig) *ConversationalAgent {
	if cfg.MaxIterations == 0 {
		cfg.MaxIterations = 10 // Default
	}
	if cfg.Model == "" {
		cfg.Model = "deepseek-r1:1.5b" // Default reasoning model
	}
	if cfg.OllamaURL == "" {
		cfg.OllamaURL = "http://localhost:11434"
	}

	return &ConversationalAgent{
		logger:        logger,
		toolRegistry:  toolRegistry,
		ollamaURL:     cfg.OllamaURL,
		model:         cfg.Model,
		maxIterations: cfg.MaxIterations,
	}
}

// SetStreamingCallbacks sets the callback functions for streaming events (v2.5.1+)
func (ca *ConversationalAgent) SetStreamingCallbacks(
	onThought func(*models.AgentThought),
	onAction func(*models.AgentActionRecord),
	onObservation func(string),
	onComplete func(bool, string),
) {
	ca.onThought = onThought
	ca.onAction = onAction
	ca.onObservation = onObservation
	ca.onComplete = onComplete
}

// SetRPCClient sets the RPC client for tool execution (v2.5.4+)
func (ca *ConversationalAgent) SetRPCClient(rpcClient *ToolRPCClient) {
	ca.rpcClient = rpcClient
	ca.logger.Info("RPC client set for conversational agent - tools will execute on desktop client")
}

// SetEventCallbacks sets callbacks for streaming agent events
func (ca *ConversationalAgent) SetEventCallbacks(
	onThought func(*models.AgentThought),
	onAction func(*models.AgentActionRecord),
	onObservation func(string),
	onComplete func(bool, string),
) {
	ca.onThought = onThought
	ca.onAction = onAction
	ca.onObservation = onObservation
	ca.onComplete = onComplete
}

// ProcessTask executes a task using ReAct loop
// Returns: final result, agent context, error
func (ca *ConversationalAgent) ProcessTask(ctx context.Context, task string, initialContext *models.AgentContext) (string, *models.AgentContext, error) {
	if initialContext == nil {
		initialContext = models.NewAgentContext(task)
	} else {
		initialContext.CurrentTask = task
		initialContext.StepNumber = 0
		initialContext.IsComplete = false
	}

	ca.logger.WithFields(logrus.Fields{
		"task":           task,
		"model":          ca.model,
		"max_iterations": ca.maxIterations,
	}).Info("Starting ReAct loop")

	// ReAct loop with loop detection
	var lastAction string
	var sameActionCount int

	for i := 0; i < ca.maxIterations; i++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return "", initialContext, ctx.Err()
		default:
		}

		ca.logger.WithField("iteration", i+1).Debug("ReAct iteration")

		// STEP 1: THINK - Generate reasoning about what to do next
		thought, err := ca.think(ctx, task, initialContext)
		if err != nil {
			ca.logger.WithError(err).Error("Think step failed - agent cannot proceed")
			if ca.onComplete != nil {
				ca.onComplete(false, fmt.Sprintf("❌ Agent reasoning failed: %v", err))
			}
			return "", initialContext, err
		}

		// Detect action loop (same action repeated)
		currentAction := fmt.Sprintf("%s:%v", thought.Action.Tool, thought.Action.Parameters)
		if currentAction == lastAction {
			sameActionCount++
			if sameActionCount >= 2 {
				// Same action 3 times in a row - force completion
				ca.logger.Warn("Loop detected: same action repeated 3 times, forcing completion")

				// Use formatted result instead of raw JSON
				summary := "Задача выполнена на основе доступной информации."
				if len(initialContext.ActionHistory) > 0 {
					lastActionResult := initialContext.ActionHistory[len(initialContext.ActionHistory)-1]
					if lastActionResult.Success && len(lastActionResult.Result) > 0 {
						// Use formatToolResult to get user-friendly format (not raw JSON)
						formattedResult := ca.formatToolResult(lastActionResult.Tool, lastActionResult.Result)

						// Limit to reasonable length
						if len(formattedResult) > 800 {
							formattedResult = formattedResult[:800] + "... (truncated)"
						}

						// Detect if task is in Russian
						isRussian := strings.Contains(task, "Покажи") || strings.Contains(task, "покажи") ||
							strings.Contains(task, "Прочитай") || strings.Contains(task, "расскажи")

						if isRussian {
							summary = fmt.Sprintf("Вот результат:\n\n%s", formattedResult)
						} else {
							summary = fmt.Sprintf("Here is the result:\n\n%s", formattedResult)
						}
					}
				}

				initialContext.MarkComplete()
				if ca.onComplete != nil {
					ca.onComplete(true, summary)
				}
				return summary, initialContext, nil
			}
		} else {
			lastAction = currentAction
			sameActionCount = 0
		}

		// Add thought to context
		thoughtRecord := models.AgentThought{
			Timestamp:  time.Now(),
			Thought:    thought.Reasoning,
			Reasoning:  thought.Details,
			Confidence: thought.Confidence,
			StepNumber: initialContext.StepNumber,
		}
		initialContext.AddThought(thoughtRecord)

		// Stream thought event
		if ca.onThought != nil {
			ca.onThought(&thoughtRecord)
		}

		// Check if task is complete
		if thought.IsComplete {
			ca.logger.Info("Agent decided task is complete")
			initialContext.MarkComplete()
			if ca.onComplete != nil {
				ca.onComplete(true, thought.FinalAnswer)
			}
			return thought.FinalAnswer, initialContext, nil
		}

		// STEP 2: ACT - Execute the planned action
		actionRecord, err := ca.act(ctx, thought.Action, initialContext)
		if err != nil {
			ca.logger.WithError(err).Error("Act step failed")
			// Continue loop - agent can recover from action failures
		}

		// Add action to context
		if actionRecord != nil {
			initialContext.AddAction(*actionRecord)

			// Stream action event
			if ca.onAction != nil {
				ca.onAction(actionRecord)
			}
		}

		// STEP 3: OBSERVE - Analyze action result
		observation, err := ca.observe(ctx, thought, actionRecord, initialContext)
		if err != nil {
			ca.logger.WithError(err).Warn("Observe step failed, continuing")
		}

		// Stream observation event
		if ca.onObservation != nil {
			ca.onObservation(observation)
		}

		ca.logger.WithFields(logrus.Fields{
			"iteration":   i + 1,
			"thought":     thought.Reasoning,
			"action":      thought.Action.Tool,
			"observation": observation,
		}).Debug("ReAct iteration completed")
	}

	// Max iterations reached
	ca.logger.Warn("Max ReAct iterations reached without completion")
	if ca.onComplete != nil {
		ca.onComplete(false, "Maximum iterations reached without task completion")
	}
	return "", initialContext, fmt.Errorf("max iterations (%d) reached without completion", ca.maxIterations)
}

// ThoughtResult represents the output of the think step
type ThoughtResult struct {
	Reasoning   string        // Agent's reasoning
	Details     string        // Additional details
	Confidence  float64       // 0.0-1.0
	IsComplete  bool          // Is task complete?
	FinalAnswer string        // Final answer if complete
	Action      PlannedAction // Next action to take
}

// PlannedAction represents an action the agent wants to take
type PlannedAction struct {
	Tool        string                 // Tool name (e.g., "file.read")
	Parameters  map[string]interface{} // Tool parameters
	Description string                 // Human-readable description
}

// think generates reasoning about what to do next using LLM
func (ca *ConversationalAgent) think(ctx context.Context, task string, agentContext *models.AgentContext) (*ThoughtResult, error) {
	// Build prompt with task, context, available tools
	availableTools := ca.toolRegistry.List()
	toolsJSON, _ := json.Marshal(availableTools)

	// Build history summary
	historyStr := ca.buildHistorySummary(agentContext)

	// Check if history is too long (might cause empty responses)
	historyLength := len(historyStr)
	if historyLength > 2000 {
		// Truncate history to last 1500 chars
		historyStr = "...(history truncated)...\n" + historyStr[historyLength-1500:]
		ca.logger.WithField("truncated_from", historyLength).Warn("History too long, truncated")
	}

	prompt := fmt.Sprintf(`You are a coding agent assistant helping users with file operations and code analysis.

# CURRENT TASK
%s

# AVAILABLE TOOLS
%s

# PREVIOUS ACTIONS
%s

# CORE PRINCIPLES

1. **Understand the Intent**: Break down what the user actually wants, not just what they said
2. **Review History First**: Check what's already been done - never repeat successful actions
3. **Think Before Acting**: Choose the right tool for the task
4. **Analyze Before Completing**: Raw data (file contents, lists) must be analyzed and explained
5. **Respond in User's Language**: Match the language of the user's request
6. **NEVER GUESS FILE CONTENTS**: If user asks to read a file, ALWAYS use file.read - don't assume or guess based on filename

# DECISION FRAMEWORK

Ask yourself these questions IN ORDER:

1. **Is there DATA in history from previous actions?**
   - If YES: Analyze it and provide final answer (don't repeat the action!)
   - If NO: Choose appropriate tool to get data

2. **What is the user's end goal?**
   - Read file → file.read → analyze content → complete
   - List files → file.list → summarize list → complete
   - Explain code → file.read → analyze code → complete

3. **What information do I already have from history?**
   - Check EVERY previous action result carefully
   - Don't repeat successful actions

4. **Is the task complete?**
   - ✅ YES if you have analyzed data and provided human-readable answer
   - ❌ NO if you only have raw data without analysis

# TOOL USAGE RULES

**file.list**: List files/directories
- Use when: "show files", "list directory", "what's here"
- Path: "." for current dir, "src/" for subdirectory

**file.read**: Read file contents (full or partial)
- Use when: "read file", "show contents", "what's in X"
- Parameters:
  - path: relative path like "README.md" or "src/main.go" (required)
  - start_line: first line to read (optional, 1-based, default: 1)
  - end_line: last line to read (optional, 1-based, default: end of file)
- Examples:
  - {"path": "README.md"} - read entire file
  - {"path": "main.go", "start_line": 10, "end_line": 50} - read lines 10-50
  - {"path": "logs/server.log", "start_line": 1000} - read from line 1000 to end
  - {"path": "big_file.txt", "end_line": 100} - read first 100 lines

**file.write**: Create or update file
- Use when: "create file", "write to X", "save content"
- Requires: path and content

**file.delete**: Remove file
- Use when: "delete X", "remove file"
- Be careful - this is destructive!

**terminal.execute**: Run shell commands
- Use when: "run command", "execute script", "check version"
- Be careful with destructive commands

# COMPLETION CRITERIA

✅ **Complete when you have:**
- Human-readable summary or answer
- Analyzed data (not just raw output)
- Addressed the user's actual intent
- Actually READ the file if user asked to read it (not guessed)

❌ **NOT complete when:**
- You only have raw data (file content, file list)
- Action succeeded but no analysis done
- User's question not fully answered
- User asked "прочитай X" / "read X" but you haven't used file.read yet
- You're GUESSING what a file contains instead of reading it

# RESPONSE FORMAT (JSON)

{
  "reasoning": "Why this is the right next step OR why task is complete",
  "confidence": 0.9,
  "is_complete": false,
  "final_answer": "",
  "next_action": {
    "tool": "file.read",
    "parameters": {"path": "README.md"},
    "description": "Read README.md file"
  }
}

# EXAMPLES

**Example 1: Simple file read**
Task: "Show me package.json"
History: (empty)
Response:
{
  "reasoning": "User wants to see package.json. Need to read it first.",
  "confidence": 0.9,
  "is_complete": false,
  "final_answer": "",
  "next_action": {
    "tool": "file.read",
    "parameters": {"path": "package.json"},
    "description": "Read package.json"
  }
}

**Example 2: Analyze after reading**
Task: "What dependencies does this project use?"
History: "file.read succeeded on package.json: {\"dependencies\": {\"react\": \"^18.0.0\", \"express\": \"^4.18.0\"}}"
Response:
{
  "reasoning": "Package.json already read. I can see dependencies: React 18 and Express 4.18. Task complete.",
  "confidence": 1.0,
  "is_complete": true,
  "final_answer": "This project uses 2 main dependencies: React 18.0.0 for frontend and Express 4.18.0 for backend server.",
  "next_action": null
}

**Example 3: Multi-step reasoning**
Task: "Прочитай README.md и скажи о чем проект"
History: "file.read succeeded on README.md: # MyApp\nA web server for API management..."
Response:
{
  "reasoning": "README прочитан. Содержит описание проекта MyApp - веб-сервер для управления API. Могу дать ответ.",
  "confidence": 1.0,
  "is_complete": true,
  "final_answer": "Проект MyApp - это веб-сервер для управления API. Основное назначение: предоставление REST endpoints для работы с API.",
  "next_action": null
}

**Example 4: Don't repeat successful actions**
Task: "List files again"
History: "file.list succeeded: found 15 files including src/, package.json, README.md"
Response:
{
  "reasoning": "Files already listed in history. No need to repeat. Providing summary.",
  "confidence": 1.0,
  "is_complete": true,
  "final_answer": "Already listed: 15 files total including src/ directory, package.json, and README.md",
  "next_action": null
}

**Example 5: Partial file reading**
Task: "Покажи мне функцию main в main.go (она примерно в строках 20-50)"
History: (empty)
Response:
{
  "reasoning": "Пользователь хочет увидеть конкретную функцию в определенном диапазоне строк. Используем file.read с start_line и end_line",
  "confidence": 0.9,
  "is_complete": false,
  "final_answer": "",
  "next_action": {
    "tool": "file.read",
    "parameters": {"path": "main.go", "start_line": 20, "end_line": 50},
    "description": "Read lines 20-50 from main.go"
  }
}

**Example 6: Read beginning of large file**
Task: "What's in the first 100 lines of server.log?"
History: (empty)
Response:
{
  "reasoning": "User wants first 100 lines of log file. Using file.read with end_line to avoid loading entire large file",
  "confidence": 0.9,
  "is_complete": false,
  "final_answer": "",
  "next_action": {
    "tool": "file.read",
    "parameters": {"path": "logs/server.log", "end_line": 100},
    "description": "Read first 100 lines of server.log"
  }
}

**Example 7: WRONG - Guessing file contents (DON'T DO THIS!)**
Task: "Прочитай README.md и скажи о чем он"
History: (empty)
❌ WRONG Response:
{
  "reasoning": "README.md обычно содержит информацию о проекте",
  "confidence": 0.8,
  "is_complete": true,
  "final_answer": "README.md - это документ о проекте",
  "next_action": null
}
✅ CORRECT Response:
{
  "reasoning": "Пользователь просит ПРОЧИТАТЬ файл. Нужно использовать file.read чтобы увидеть реальное содержимое",
  "confidence": 0.9,
  "is_complete": false,
  "final_answer": "",
  "next_action": {
    "tool": "file.read",
    "parameters": {"path": "README.md"},
    "description": "Read README.md to see actual content"
  }
}

YOUR RESPONSE (MUST BE VALID JSON):`, task, string(toolsJSON), historyStr)

	// Call LLM
	response, err := ca.callOllamaForReasoning(ctx, prompt)
	if err != nil {
		ca.logger.WithError(err).Error("LLM call failed")
		return nil, fmt.Errorf("LLM call failed: %w. Please check if Ollama is running and the model '%s' is available", err, ca.model)
	}

	// Parse response
	thought, err := ca.parseThoughtResponse(response)
	if err != nil {
		ca.logger.WithError(err).WithField("response_preview", response[:min(len(response), 200)]).Error("Failed to parse LLM response")
		return nil, fmt.Errorf("LLM returned invalid response: %w. The model '%s' may not be suitable for agent reasoning. Consider using: qwen2.5:7b, llama3.2:3b, or mistral:7b", err, ca.model)
	}

	return thought, nil
}

// act executes the planned action using tools
func (ca *ConversationalAgent) act(ctx context.Context, action PlannedAction, agentContext *models.AgentContext) (*models.AgentActionRecord, error) {
	startTime := time.Now()

	ca.logger.WithFields(logrus.Fields{
		"tool":       action.Tool,
		"parameters": action.Parameters,
	}).Info("Executing action")

	// v2.5.4+: Use RPC client if available (client-side tool execution)
	if ca.rpcClient != nil {
		ca.logger.Debug("Using RPC client for tool execution (client-side)")

		// Determine if tool requires approval
		requiresApproval := isDangerousTool(action.Tool)

		// Execute tool via RPC (sends request to desktop client)
		response, err := ca.rpcClient.ExecuteTool(ctx, action.Tool, action.Parameters, requiresApproval)
		if err != nil {
			return &models.AgentActionRecord{
				Timestamp:  startTime,
				Action:     action.Description,
				Tool:       action.Tool,
				Parameters: action.Parameters,
				Success:    false,
				Error:      fmt.Sprintf("RPC tool execution failed: %v", err),
				Duration:   time.Since(startTime).Milliseconds(),
				StepNumber: agentContext.StepNumber,
			}, err
		}

		// Convert result to string
		resultStr := ""
		if response.Result != nil {
			if str, ok := response.Result.(string); ok {
				resultStr = str
			} else {
				// Convert to JSON string if not string
				if jsonBytes, err := json.Marshal(response.Result); err == nil {
					resultStr = string(jsonBytes)
				}
			}
		}

		// Create action record from RPC response
		actionRecord := &models.AgentActionRecord{
			Timestamp:  startTime,
			Action:     action.Description,
			Tool:       action.Tool,
			Parameters: action.Parameters,
			Result:     resultStr,
			Success:    response.Success,
			Error:      response.Error,
			Duration:   response.Duration,
			StepNumber: agentContext.StepNumber,
		}

		if !response.Success {
			ca.logger.WithField("error", response.Error).Warn("RPC tool execution failed")
		}

		return actionRecord, nil
	}

	// Fallback: Use local tool registry (server-side execution - deprecated)
	ca.logger.Warn("Using local tool execution (server-side) - RPC client not set. Tools may fail if they require client filesystem access.")

	// Get tool from registry
	tool, err := ca.toolRegistry.Get(action.Tool)
	if err != nil {
		return &models.AgentActionRecord{
			Timestamp:  startTime,
			Action:     action.Description,
			Tool:       action.Tool,
			Parameters: action.Parameters,
			Success:    false,
			Error:      fmt.Sprintf("Tool not found: %v", err),
			Duration:   time.Since(startTime).Milliseconds(),
			StepNumber: agentContext.StepNumber,
		}, err
	}

	// Execute tool locally
	result, execErr := tool.Execute(ctx, action.Parameters)
	if execErr != nil {
		return &models.AgentActionRecord{
			Timestamp:  startTime,
			Action:     action.Description,
			Tool:       action.Tool,
			Parameters: action.Parameters,
			Success:    false,
			Error:      execErr.Error(),
			Duration:   time.Since(startTime).Milliseconds(),
			StepNumber: agentContext.StepNumber,
		}, execErr
	}

	// Convert output to string
	outputStr := ""
	if result.Output != nil {
		if str, ok := result.Output.(string); ok {
			outputStr = str
		} else {
			// Convert to JSON string if not string
			if jsonBytes, err := json.Marshal(result.Output); err == nil {
				outputStr = string(jsonBytes)
			}
		}
	}

	// Convert error pointer to string
	errorStr := ""
	if result.Error != nil {
		errorStr = *result.Error
	}

	// Create action record
	actionRecord := &models.AgentActionRecord{
		Timestamp:  startTime,
		Action:     action.Description,
		Tool:       action.Tool,
		Parameters: action.Parameters,
		Result:     outputStr,
		Success:    result.Success,
		Error:      errorStr,
		Duration:   time.Since(startTime).Milliseconds(),
		StepNumber: agentContext.StepNumber,
	}

	if !result.Success {
		ca.logger.WithField("error", result.Error).Warn("Action execution failed")
	}

	return actionRecord, nil
}

// observe analyzes the action result and provides feedback
func (ca *ConversationalAgent) observe(ctx context.Context, thought *ThoughtResult, action *models.AgentActionRecord, agentContext *models.AgentContext) (string, error) {
	// Simple observation: success/failure + result summary
	if action == nil {
		return "No action was taken", nil
	}

	if action.Success {
		// Format result based on tool type (NO TRUNCATION - user needs full data)
		result := ca.formatToolResult(action.Tool, action.Result)
		return fmt.Sprintf("Action succeeded. Result: %s", result), nil
	}

	return fmt.Sprintf("Action failed: %s", action.Error), nil
}

// formatToolResult formats tool result for better readability
func (ca *ConversationalAgent) formatToolResult(tool string, rawResult string) string {
	switch tool {
	case "file.list":
		return ca.formatFileListResult(rawResult)
	case "file.read":
		return ca.formatFileReadResult(rawResult)
	default:
		return rawResult
	}
}

// formatFileListResult formats file.list JSON result into readable list
func (ca *ConversationalAgent) formatFileListResult(rawResult string) string {
	var result struct {
		Count int `json:"count"`
		Files []struct {
			Name  string `json:"name"`
			IsDir bool   `json:"is_dir"`
			Size  int64  `json:"size"`
		} `json:"files"`
	}

	if err := json.Unmarshal([]byte(rawResult), &result); err != nil {
		return rawResult // Return raw if parse fails
	}

	// Format as readable list
	var formatted strings.Builder
	formatted.WriteString(fmt.Sprintf("Found %d items:\n", result.Count))

	// Separate dirs and files
	var dirs, files []string
	for _, f := range result.Files {
		if f.IsDir {
			dirs = append(dirs, fmt.Sprintf("📁 %s/", f.Name))
		} else {
			// Format file size
			sizeStr := formatFileSize(f.Size)
			files = append(files, fmt.Sprintf("📄 %s (%s)", f.Name, sizeStr))
		}
	}

	// Show directories first
	if len(dirs) > 0 {
		formatted.WriteString("\nDirectories:\n")
		for _, d := range dirs {
			formatted.WriteString("  " + d + "\n")
		}
	}

	// Then files (SHOW ALL - no truncation)
	if len(files) > 0 {
		formatted.WriteString("\nFiles:\n")
		for _, f := range files {
			formatted.WriteString("  " + f + "\n")
		}
	}

	return formatted.String()
}

// formatFileReadResult formats file.read JSON result
func (ca *ConversationalAgent) formatFileReadResult(rawResult string) string {
	var result struct {
		Content string `json:"content"`
	}

	if err := json.Unmarshal([]byte(rawResult), &result); err != nil {
		return rawResult // Return raw if parse fails
	}

	// Return FULL content (no truncation - user needs complete file)
	return fmt.Sprintf("File content:\n%s", result.Content)
}

// formatFileSize formats bytes to human-readable size
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// extractSummary extracts first N characters from content, cleaning up formatting
func (ca *ConversationalAgent) extractSummary(content string, maxLen int) string {
	// Remove excessive newlines and escape sequences
	content = strings.ReplaceAll(content, "\\r\\n", "\n")
	content = strings.ReplaceAll(content, "\\n", "\n")
	content = strings.ReplaceAll(content, "\\u003e", ">")

	// Limit length
	if len(content) > maxLen {
		content = content[:maxLen] + "..."
	}

	return strings.TrimSpace(content)
}

// buildHistorySummary creates a summary of past thoughts and actions
func (ca *ConversationalAgent) buildHistorySummary(agentContext *models.AgentContext) string {
	if len(agentContext.ThoughtHistory) == 0 && len(agentContext.ActionHistory) == 0 {
		return "No previous history."
	}

	var history strings.Builder
	history.WriteString("Previous steps:\n")

	// Interleave thoughts and actions by step number
	for i := 0; i < len(agentContext.ThoughtHistory) || i < len(agentContext.ActionHistory); i++ {
		if i < len(agentContext.ThoughtHistory) {
			t := agentContext.ThoughtHistory[i]
			history.WriteString(fmt.Sprintf("- Thought %d: %s\n", t.StepNumber, t.Thought))
		}
		if i < len(agentContext.ActionHistory) {
			a := agentContext.ActionHistory[i]
			if !a.Success {
				history.WriteString(fmt.Sprintf("- Action %d: %s using %s → FAILED: %s\n", a.StepNumber, a.Action, a.Tool, a.Error))
			} else {
				// Format result for history (user-friendly format, not raw JSON)
				formattedResult := ca.formatToolResult(a.Tool, a.Result)

				// Intelligent truncation based on tool type
				maxLen := 500
				if a.Tool == "file.read" {
					// For file.read, show more content so LLM knows file was fully read
					maxLen = 3000 // ~50 lines of code or 3-4 paragraphs
				} else if a.Tool == "file.list" {
					maxLen = 1000 // Show more files in list
				}

				if len(formattedResult) > maxLen {
					formattedResult = formattedResult[:maxLen] + fmt.Sprintf("... (truncated, showing first %d/%d chars)", maxLen, len(formattedResult))
				}

				history.WriteString(fmt.Sprintf("- Action %d: %s using %s → SUCCESS.\n%s\n", a.StepNumber, a.Action, a.Tool, formattedResult))
			}
		}
	}

	return history.String()
}

// callOllamaForReasoning calls Ollama API for reasoning
func (ca *ConversationalAgent) callOllamaForReasoning(ctx context.Context, prompt string) (string, error) {
	reqBody := map[string]interface{}{
		"model": ca.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are an intelligent agent. Respond with valid JSON only. Be precise and deterministic.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.3, // Low temperature for focused, valid JSON (0.0 too strict for reasoning models)
			"top_p":       0.9,
			"num_predict": 512, // Limit response length for faster iterations
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/chat", ca.ollamaURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	ca.logger.WithFields(logrus.Fields{
		"url":          url,
		"model":        ca.model,
		"prompt_chars": len(prompt),
	}).Debug("Calling Ollama for reasoning")

	client := &http.Client{Timeout: 120 * time.Second} // Longer timeout for reasoning
	resp, err := client.Do(req)
	if err != nil {
		ca.logger.WithError(err).Error("Failed to call Ollama")
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		ca.logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(bodyBytes),
		}).Error("Ollama returned non-200 status")
		return "", fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	ca.logger.WithField("response_bytes", len(bodyBytes)).Debug("Received response from Ollama")

	var ollamaResp struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.Unmarshal(bodyBytes, &ollamaResp); err != nil {
		ca.logger.WithError(err).WithField("raw_response", string(bodyBytes[:min(len(bodyBytes), 500)])).Error("Failed to unmarshal Ollama response")
		return "", fmt.Errorf("failed to unmarshal Ollama response: %w", err)
	}

	content := ollamaResp.Message.Content

	ca.logger.WithField("content_chars", len(content)).Debug("Extracted content from Ollama response")

	// Check for empty response
	if len(content) == 0 {
		ca.logger.Warn("Ollama returned empty content")
		return "", fmt.Errorf("Ollama returned empty response")
	}

	return content, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// parseThoughtResponse parses LLM response into ThoughtResult
func (ca *ConversationalAgent) parseThoughtResponse(response string) (*ThoughtResult, error) {
	// Extract JSON from markdown blocks if present
	cleanedResponse := response
	responseBytes := []byte(response)

	// Try ```json block
	if jsonStart := bytes.Index(responseBytes, []byte("```json")); jsonStart >= 0 {
		jsonStart += 7
		for jsonStart < len(responseBytes) && (responseBytes[jsonStart] == '\n' || responseBytes[jsonStart] == '\r' || responseBytes[jsonStart] == ' ') {
			jsonStart++
		}
		if jsonEnd := bytes.Index(responseBytes[jsonStart:], []byte("```")); jsonEnd > 0 {
			cleanedResponse = string(responseBytes[jsonStart : jsonStart+jsonEnd])
		}
	} else if codeStart := bytes.Index(responseBytes, []byte("```")); codeStart >= 0 {
		// Try generic ``` block
		codeStart += 3
		for codeStart < len(responseBytes) && (responseBytes[codeStart] == '\n' || responseBytes[codeStart] == '\r' || responseBytes[codeStart] == ' ') {
			codeStart++
		}
		if codeEnd := bytes.Index(responseBytes[codeStart:], []byte("```")); codeEnd > 0 {
			cleanedResponse = string(responseBytes[codeStart : codeStart+codeEnd])
		}
	}

	cleanedResponse = strings.TrimSpace(cleanedResponse)

	// Parse JSON
	var parsed struct {
		Reasoning   string  `json:"reasoning"`
		Confidence  float64 `json:"confidence"`
		IsComplete  bool    `json:"is_complete"`
		FinalAnswer string  `json:"final_answer"`
		NextAction  *struct {
			Tool        string                 `json:"tool"`
			Parameters  map[string]interface{} `json:"parameters"`
			Description string                 `json:"description"`
		} `json:"next_action"`
	}

	if err := json.Unmarshal([]byte(cleanedResponse), &parsed); err != nil {
		ca.logger.WithError(err).WithField("response", response).Error("Failed to parse thought response")
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	result := &ThoughtResult{
		Reasoning:   parsed.Reasoning,
		Confidence:  parsed.Confidence,
		IsComplete:  parsed.IsComplete,
		FinalAnswer: parsed.FinalAnswer,
	}

	// Parse next action if provided
	if parsed.NextAction != nil {
		result.Action = PlannedAction{
			Tool:        parsed.NextAction.Tool,
			Parameters:  parsed.NextAction.Parameters,
			Description: parsed.NextAction.Description,
		}
	}

	return result, nil
}
