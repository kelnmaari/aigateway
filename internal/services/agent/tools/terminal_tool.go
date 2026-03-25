// Package tools provides terminal execution tool for Agent (v2.5.0+)
package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"slices"
	"strings"

	"aigateway/internal/models"
)

// TerminalTool executes shell commands
type TerminalTool struct {
	workDir       string   // Working directory
	allowedShells []string // Allowed shell executables
}

// NewTerminalTool creates a new terminal execution tool
func NewTerminalTool(workDir string) *TerminalTool {
	// Default allowed shells based on OS
	var allowedShells []string
	if runtime.GOOS == "windows" {
		allowedShells = []string{"powershell.exe", "pwsh.exe", "cmd.exe"}
	} else {
		allowedShells = []string{"/bin/bash", "/bin/sh", "/usr/bin/bash"}
	}

	return &TerminalTool{
		workDir:       workDir,
		allowedShells: allowedShells,
	}
}

func (t *TerminalTool) GetInfo() models.AgentTool {
	return models.AgentTool{
		ID:          "terminal-execute",
		Name:        "terminal.execute",
		Category:    models.AgentToolCategoryTerminal,
		Description: "Execute shell commands",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "Command to execute",
				},
				"shell": map[string]any{
					"type":        "string",
					"description": fmt.Sprintf("Shell to use (allowed: %v)", t.allowedShells),
					"enum":        t.allowedShells,
				},
				"work_dir": map[string]any{
					"type":        "string",
					"description": "Working directory",
				},
			},
			"required": []string{"command"},
		},
		Dangerous: true, // Terminal execution is very dangerous
		Available: true,
		Version:   "1.0.0",
	}
}

func (t *TerminalTool) Validate(params map[string]any) error {
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return fmt.Errorf("'command' parameter is required")
	}

	// Validate shell if provided
	if shell, ok := params["shell"].(string); ok {
		allowed := slices.Contains(t.allowedShells, shell)
		if !allowed {
			return fmt.Errorf("shell '%s' is not allowed. Allowed shells: %v", shell, t.allowedShells)
		}
	}

	return nil
}

func (t *TerminalTool) Execute(ctx context.Context, params map[string]any) (*models.AgentStepResult, error) {
	command := params["command"].(string)

	// Get shell (default to first allowed shell)
	shell := t.allowedShells[0]
	if s, ok := params["shell"].(string); ok && s != "" {
		shell = s
	}

	// Get working directory
	workDir := t.workDir
	if wd, ok := params["work_dir"].(string); ok && wd != "" {
		workDir = wd
	}

	// Prepare command execution
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Windows: use -Command for PowerShell, /c for cmd
		if strings.Contains(shell, "powershell") || strings.Contains(shell, "pwsh") {
			cmd = exec.CommandContext(ctx, shell, "-Command", command)
		} else {
			cmd = exec.CommandContext(ctx, shell, "/c", command)
		}
	} else {
		// Unix: use -c for bash/sh
		cmd = exec.CommandContext(ctx, shell, "-c", command)
	}

	// Set working directory
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start
			errMsg := fmt.Sprintf("failed to execute command: %v", err)
			return &models.AgentStepResult{
				Success: false,
				Error:   &errMsg,
			}, nil
		}
	}

	// Success if exit code is 0
	success := exitCode == 0

	output := map[string]any{
		"command":   command,
		"shell":     shell,
		"work_dir":  workDir,
		"exit_code": exitCode,
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
	}

	result := &models.AgentStepResult{
		Success: success,
		Output:  output,
	}

	// Add error if non-zero exit code
	if !success {
		errMsg := fmt.Sprintf("command exited with code %d", exitCode)
		result.Error = &errMsg
	}

	return result, nil
}

// RegisterTerminalTool registers terminal tool to registry
func RegisterTerminalTool(registry *Registry, workDir string) error {
	tool := NewTerminalTool(workDir)
	return registry.Register(tool)
}
