// Package tools provides file operation tools for Agent (v2.5.0+)
package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"aigateway/internal/models"
)

// ========================================
// File Read Tool
// ========================================

// FileReadTool reads file contents
type FileReadTool struct {
	baseDir string // Base directory for security (optional)
}

// NewFileReadTool creates a new file read tool
func NewFileReadTool(baseDir string) *FileReadTool {
	// Convert baseDir to absolute path for security checks
	if baseDir != "" {
		absBaseDir, err := filepath.Abs(baseDir)
		if err == nil {
			baseDir = absBaseDir
		}
	}
	
	return &FileReadTool{
		baseDir: baseDir,
	}
}

func (t *FileReadTool) GetInfo() models.AgentTool {
	return models.AgentTool{
		ID:          "file-read",
		Name:        "file.read",
		Category:    models.AgentToolCategoryFile,
		Description: "Read contents of a file",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to read",
				},
			},
			"required": []string{"path"},
		},
		Dangerous: false,
		Available: true,
		Version:   "1.0.0",
	}
}

func (t *FileReadTool) Validate(params map[string]interface{}) error {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("'path' parameter is required and must be a string")
	}
	return nil
}

func (t *FileReadTool) Execute(ctx context.Context, params map[string]interface{}) (*models.AgentStepResult, error) {
	path := params["path"].(string)
	
	// Security: validate path if baseDir is set
	if t.baseDir != "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			errMsg := fmt.Sprintf("invalid path: %v", err)
			return &models.AgentStepResult{
				Success: false,
				Error:   &errMsg,
			}, nil
		}
		
		if !strings.HasPrefix(absPath, t.baseDir) {
			errMsg := fmt.Sprintf("access denied: path outside allowed directory")
			return &models.AgentStepResult{
				Success: false,
				Error:   &errMsg,
			}, nil
		}
	}
	
	// Read file
	content, err := os.ReadFile(path)
	if err != nil {
		errMsg := fmt.Sprintf("failed to read file: %v", err)
		return &models.AgentStepResult{
			Success: false,
			Error:   &errMsg,
		}, nil
	}
	
	return &models.AgentStepResult{
		Success: true,
		Output: map[string]interface{}{
			"path":    path,
			"content": string(content),
			"size":    len(content),
		},
	}, nil
}

// ========================================
// File Write Tool
// ========================================

// FileWriteTool writes content to a file
type FileWriteTool struct {
	baseDir string
}

func NewFileWriteTool(baseDir string) *FileWriteTool {
	// Convert baseDir to absolute path for security checks
	if baseDir != "" {
		absBaseDir, err := filepath.Abs(baseDir)
		if err == nil {
			baseDir = absBaseDir
		}
	}
	
	return &FileWriteTool{
		baseDir: baseDir,
	}
}

func (t *FileWriteTool) GetInfo() models.AgentTool {
	return models.AgentTool{
		ID:          "file-write",
		Name:        "file.write",
		Category:    models.AgentToolCategoryFile,
		Description: "Write content to a file (overwrites existing)",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file to write",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to write",
				},
			},
			"required": []string{"path", "content"},
		},
		Dangerous: true, // Writing files is dangerous
		Available: true,
		Version:   "1.0.0",
	}
}

func (t *FileWriteTool) Validate(params map[string]interface{}) error {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("'path' parameter is required")
	}
	
	_, ok = params["content"].(string)
	if !ok {
		return fmt.Errorf("'content' parameter is required and must be a string")
	}
	
	return nil
}

func (t *FileWriteTool) Execute(ctx context.Context, params map[string]interface{}) (*models.AgentStepResult, error) {
	path := params["path"].(string)
	content := params["content"].(string)
	
	// Security check
	if t.baseDir != "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			errMsg := fmt.Sprintf("invalid path: %v", err)
			return &models.AgentStepResult{
				Success: false,
				Error:   &errMsg,
			}, nil
		}
		
		if !strings.HasPrefix(absPath, t.baseDir) {
			errMsg := "access denied: path outside allowed directory"
			return &models.AgentStepResult{
				Success: false,
				Error:   &errMsg,
			}, nil
		}
	}
	
	// Create parent directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		errMsg := fmt.Sprintf("failed to create directory: %v", err)
		return &models.AgentStepResult{
			Success: false,
			Error:   &errMsg,
		}, nil
	}
	
	// Write file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		errMsg := fmt.Sprintf("failed to write file: %v", err)
		return &models.AgentStepResult{
			Success: false,
			Error:   &errMsg,
		}, nil
	}
	
	return &models.AgentStepResult{
		Success: true,
		Output: map[string]interface{}{
			"path":         path,
			"bytes_written": len(content),
		},
		RequiresApproval: false, // Already approved by virtue of execution
	}, nil
}

// ========================================
// File List Tool
// ========================================

// FileListTool lists files in a directory
type FileListTool struct {
	baseDir string
}

func NewFileListTool(baseDir string) *FileListTool {
	// Convert baseDir to absolute path for security checks
	if baseDir != "" {
		absBaseDir, err := filepath.Abs(baseDir)
		if err == nil {
			baseDir = absBaseDir
		}
	}
	
	return &FileListTool{
		baseDir: baseDir,
	}
}

func (t *FileListTool) GetInfo() models.AgentTool {
	return models.AgentTool{
		ID:          "file-list",
		Name:        "file.list",
		Category:    models.AgentToolCategoryFile,
		Description: "List files in a directory",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Directory path to list",
					"default":     ".",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "List recursively",
					"default":     false,
				},
			},
		},
		Dangerous: false,
		Available: true,
		Version:   "1.0.0",
	}
}

func (t *FileListTool) Validate(params map[string]interface{}) error {
	if path, ok := params["path"].(string); ok && path == "" {
		return fmt.Errorf("'path' cannot be empty")
	}
	return nil
}

func (t *FileListTool) Execute(ctx context.Context, params map[string]interface{}) (*models.AgentStepResult, error) {
	path := "."
	if p, ok := params["path"].(string); ok && p != "" {
		path = p
	}
	
	recursive := false
	if r, ok := params["recursive"].(bool); ok {
		recursive = r
	}
	
	// Security check
	if t.baseDir != "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			errMsg := fmt.Sprintf("invalid path: %v", err)
			return &models.AgentStepResult{
				Success: false,
				Error:   &errMsg,
			}, nil
		}
		
		if !strings.HasPrefix(absPath, t.baseDir) {
			errMsg := "access denied: path outside allowed directory"
			return &models.AgentStepResult{
				Success: false,
				Error:   &errMsg,
			}, nil
		}
	}
	
	var files []map[string]interface{}
	
	if recursive {
		// Walk directory tree
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			relPath, _ := filepath.Rel(path, filePath)
			files = append(files, map[string]interface{}{
				"name":  info.Name(),
				"path":  relPath,
				"size":  info.Size(),
				"is_dir": info.IsDir(),
				"mode":  info.Mode().String(),
			})
			
			return nil
		})
		
		if err != nil {
			errMsg := fmt.Sprintf("failed to walk directory: %v", err)
			return &models.AgentStepResult{
				Success: false,
				Error:   &errMsg,
			}, nil
		}
	} else {
		// List single directory
		entries, err := os.ReadDir(path)
		if err != nil {
			errMsg := fmt.Sprintf("failed to read directory: %v", err)
			return &models.AgentStepResult{
				Success: false,
				Error:   &errMsg,
			}, nil
		}
		
		for _, entry := range entries {
			info, _ := entry.Info()
			files = append(files, map[string]interface{}{
				"name":  entry.Name(),
				"path":  entry.Name(),
				"size":  info.Size(),
				"is_dir": entry.IsDir(),
				"mode":  info.Mode().String(),
			})
		}
	}
	
	return &models.AgentStepResult{
		Success: true,
		Output: map[string]interface{}{
			"path":      path,
			"files":     files,
			"count":     len(files),
			"recursive": recursive,
		},
	}, nil
}

// ========================================
// File Delete Tool
// ========================================

// FileDeleteTool deletes a file
type FileDeleteTool struct {
	baseDir string
}

func NewFileDeleteTool(baseDir string) *FileDeleteTool {
	// Convert baseDir to absolute path for security checks
	if baseDir != "" {
		absBaseDir, err := filepath.Abs(baseDir)
		if err == nil {
			baseDir = absBaseDir
		}
	}
	
	return &FileDeleteTool{
		baseDir: baseDir,
	}
}

func (t *FileDeleteTool) GetInfo() models.AgentTool {
	return models.AgentTool{
		ID:          "file-delete",
		Name:        "file.delete",
		Category:    models.AgentToolCategoryFile,
		Description: "Delete a file or directory",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to delete",
				},
			},
			"required": []string{"path"},
		},
		Dangerous: true, // Deletion is dangerous
		Available: true,
		Version:   "1.0.0",
	}
}

func (t *FileDeleteTool) Validate(params map[string]interface{}) error {
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("'path' parameter is required")
	}
	return nil
}

func (t *FileDeleteTool) Execute(ctx context.Context, params map[string]interface{}) (*models.AgentStepResult, error) {
	path := params["path"].(string)
	
	// Security check
	if t.baseDir != "" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			errMsg := fmt.Sprintf("invalid path: %v", err)
			return &models.AgentStepResult{
				Success: false,
				Error:   &errMsg,
			}, nil
		}
		
		if !strings.HasPrefix(absPath, t.baseDir) {
			errMsg := "access denied: path outside allowed directory"
			return &models.AgentStepResult{
				Success: false,
				Error:   &errMsg,
			}, nil
		}
	}
	
	// Delete file/directory
	if err := os.RemoveAll(path); err != nil {
		errMsg := fmt.Sprintf("failed to delete: %v", err)
		return &models.AgentStepResult{
			Success: false,
			Error:   &errMsg,
		}, nil
	}
	
	return &models.AgentStepResult{
		Success: true,
		Output: map[string]interface{}{
			"path":    path,
			"deleted": true,
		},
	}, nil
}

// ========================================
// Helper Functions
// ========================================

// RegisterFileTools registers all file operation tools
func RegisterFileTools(registry *Registry, baseDir string) error {
	tools := []Tool{
		NewFileReadTool(baseDir),
		NewFileWriteTool(baseDir),
		NewFileListTool(baseDir),
		NewFileDeleteTool(baseDir),
	}
	
	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return err
		}
	}
	
	return nil
}

