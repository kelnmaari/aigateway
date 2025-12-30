// Package handlers provides HTTP handlers for GitLab test generation API.
package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"aigateway/internal/gitlab/storage"
	"aigateway/internal/gitlab/testgen"
	"aigateway/internal/rag/vector"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GitLabTestGenHandler handles test generation API requests.
type GitLabTestGenHandler struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	logger      *logrus.Logger
}

// NewGitLabTestGenHandler creates a new test generation handler.
func NewGitLabTestGenHandler(store storage.Store, vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *GitLabTestGenHandler {
	return &GitLabTestGenHandler{
		store:       store,
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
}

// ScanTestableRequest is the request body for scanning testable functions.
type ScanTestableRequest struct {
	MaxFiles int    `json:"max_files,omitempty"`
	Language string `json:"language,omitempty"`
}

// GenerateTestsRequest is the request body for generating tests.
type GenerateTestsRequest struct {
	MaxFunctions int    `json:"max_functions,omitempty"`
	Language     string `json:"language,omitempty"`
	Framework    string `json:"framework,omitempty"`
}

// ScanTestable POST /api/admin/gitlab/projects/:id/scan-tests
func (h *GitLabTestGenHandler) ScanTestable(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req ScanTestableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = ScanTestableRequest{}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	// Get project
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Check if project is indexed
	collectionName := project.GetCollectionName()
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no indexed collection. Please index the repository first."})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"collection": collectionName,
	}).Info("Starting test scan")

	// Create generator and scan
	generator := testgen.NewGenerator(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)
	result, err := generator.Scan(ctx, testgen.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		MaxFiles:       req.MaxFiles,
		Language:       req.Language,
	})
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Test scan failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GenerateTests POST /api/admin/gitlab/projects/:id/generate-tests
func (h *GitLabTestGenHandler) GenerateTests(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req GenerateTestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = GenerateTestsRequest{}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Minute)
	defer cancel()

	// Get project
	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Failed to get project")
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	// Check if project has analysis model
	if project.AnalysisModelID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no analysis model configured"})
		return
	}

	// Check if project is indexed
	collectionName := project.GetCollectionName()
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no indexed collection. Please index the repository first."})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"collection": collectionName,
		"model":      project.AnalysisModelID,
	}).Info("Starting test generation")

	// First, scan for testable functions
	generator := testgen.NewGenerator(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)
	scanResult, err := generator.Scan(ctx, testgen.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		MaxFiles:       50,
		Language:       req.Language,
	})
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Test scan failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan failed: " + err.Error()})
		return
	}

	if len(scanResult.Functions) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":     "No testable functions found",
			"status":      "completed",
			"tests":       []interface{}{},
			"tokens_used": 0,
		})
		return
	}

	// Generate tests
	result, err := generator.Generate(ctx, testgen.GenerateRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		ModelID:        project.AnalysisModelID,
		MaxFunctions:   req.MaxFunctions,
		Language:       req.Language,
		Framework:      testgen.TestFramework(req.Framework),
	}, scanResult.Functions)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Test generation failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Generation failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DownloadTestsRequest is the request body for downloading tests.
type DownloadTestsRequest struct {
	Tests    []testgen.GeneratedTest `json:"tests" binding:"required"`
	Format   string                  `json:"format,omitempty"` // "single" or "zip", default "single"
	Filename string                  `json:"filename,omitempty"`
}

// DownloadTests POST /api/admin/gitlab/projects/:id/download-tests
// Downloads generated tests as a file.
func (h *GitLabTestGenHandler) DownloadTests(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req DownloadTestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if len(req.Tests) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No tests to download"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id":  projectID,
		"tests_count": len(req.Tests),
		"format":      req.Format,
	}).Info("Downloading tests")

	format := req.Format
	if format == "" {
		format = "single"
	}

	if format == "zip" {
		// Create a ZIP file with all tests
		h.downloadAsZip(c, projectID, req.Tests)
		return
	}

	// Single file: combine all tests
	h.downloadAsSingleFile(c, projectID, req.Tests, req.Filename)
}

func (h *GitLabTestGenHandler) downloadAsSingleFile(c *gin.Context, projectID string, tests []testgen.GeneratedTest, customFilename string) {
	if len(tests) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No tests to download"})
		return
	}

	// Determine language and extension
	language := tests[0].Language
	ext := getTestFileExtension(language)

	// Build combined test content
	var content strings.Builder
	content.WriteString(getTestFileHeader(language, projectID))
	content.WriteString("\n\n")

	for i, test := range tests {
		content.WriteString(fmt.Sprintf("// =============================================================================\n"))
		content.WriteString(fmt.Sprintf("// Tests for: %s\n", test.Function.Name))
		content.WriteString(fmt.Sprintf("// File: %s\n", test.Function.FilePath))
		content.WriteString(fmt.Sprintf("// =============================================================================\n\n"))
		content.WriteString(test.TestCode)
		if i < len(tests)-1 {
			content.WriteString("\n\n")
		}
	}

	filename := customFilename
	if filename == "" {
		filename = fmt.Sprintf("generated_tests_%s%s", time.Now().Format("20060102_150405"), ext)
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", getContentType(language))
	c.String(http.StatusOK, content.String())
}

func (h *GitLabTestGenHandler) downloadAsZip(c *gin.Context, projectID string, tests []testgen.GeneratedTest) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Group tests by file
	fileTests := make(map[string][]testgen.GeneratedTest)
	for _, test := range tests {
		baseName := filepath.Base(test.Function.FilePath)
		ext := filepath.Ext(baseName)
		name := strings.TrimSuffix(baseName, ext)
		testFileName := getTestFileName(name, test.Language)
		fileTests[testFileName] = append(fileTests[testFileName], test)
	}

	for fileName, testsGroup := range fileTests {
		var content strings.Builder
		language := testsGroup[0].Language
		content.WriteString(getTestFileHeader(language, projectID))
		content.WriteString("\n\n")

		for i, test := range testsGroup {
			content.WriteString(fmt.Sprintf("// Tests for: %s\n", test.Function.Name))
			content.WriteString(test.TestCode)
			if i < len(testsGroup)-1 {
				content.WriteString("\n\n")
			}
		}

		writer, err := zipWriter.Create(fileName)
		if err != nil {
			h.logger.WithError(err).Error("Failed to create file in zip")
			continue
		}
		_, _ = writer.Write([]byte(content.String()))
	}

	if err := zipWriter.Close(); err != nil {
		h.logger.WithError(err).Error("Failed to close zip writer")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create zip file"})
		return
	}

	filename := fmt.Sprintf("tests_%s_%s.zip", projectID, time.Now().Format("20060102_150405"))
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/zip")
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
}

func getTestFileExtension(language string) string {
	switch strings.ToLower(language) {
	case "go":
		return "_test.go"
	case "typescript", "ts":
		return ".test.ts"
	case "javascript", "js":
		return ".test.js"
	case "python", "py":
		return "_test.py"
	case "rust", "rs":
		return "_test.rs"
	case "java":
		return "Test.java"
	default:
		return "_test.txt"
	}
}

func getTestFileName(baseName, language string) string {
	switch strings.ToLower(language) {
	case "go":
		return baseName + "_test.go"
	case "typescript", "ts":
		return baseName + ".test.ts"
	case "javascript", "js":
		return baseName + ".test.js"
	case "python", "py":
		return "test_" + baseName + ".py"
	case "rust", "rs":
		return baseName + "_test.rs"
	case "java":
		return baseName + "Test.java"
	default:
		return baseName + "_test.txt"
	}
}

func getTestFileHeader(language, projectID string) string {
	switch strings.ToLower(language) {
	case "go":
		return fmt.Sprintf("// Auto-generated tests for project %s\n// Generated by AIGateway Test Generator\n// Generated at: %s\n\npackage tests", projectID, time.Now().Format(time.RFC3339))
	case "typescript", "ts":
		return fmt.Sprintf("/**\n * Auto-generated tests for project %s\n * Generated by AIGateway Test Generator\n * Generated at: %s\n */", projectID, time.Now().Format(time.RFC3339))
	case "javascript", "js":
		return fmt.Sprintf("/**\n * Auto-generated tests for project %s\n * Generated by AIGateway Test Generator\n * Generated at: %s\n */", projectID, time.Now().Format(time.RFC3339))
	case "python", "py":
		return fmt.Sprintf(`"""
Auto-generated tests for project %s
Generated by AIGateway Test Generator
Generated at: %s
"""

import pytest`, projectID, time.Now().Format(time.RFC3339))
	case "rust", "rs":
		return fmt.Sprintf("// Auto-generated tests for project %s\n// Generated by AIGateway Test Generator\n// Generated at: %s\n\n#[cfg(test)]\nmod tests {", projectID, time.Now().Format(time.RFC3339))
	default:
		return fmt.Sprintf("// Auto-generated tests for project %s\n// Generated by AIGateway Test Generator\n// Generated at: %s", projectID, time.Now().Format(time.RFC3339))
	}
}

func getContentType(language string) string {
	switch strings.ToLower(language) {
	case "go":
		return "text/x-go"
	case "typescript", "ts":
		return "text/typescript"
	case "javascript", "js":
		return "application/javascript"
	case "python", "py":
		return "text/x-python"
	case "rust", "rs":
		return "text/x-rust"
	case "java":
		return "text/x-java"
	default:
		return "text/plain"
	}
}

