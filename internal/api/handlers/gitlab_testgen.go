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

	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/gitlab/testgen"
	"aigateway/internal/models"
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
	h := &GitLabTestGenHandler{
		store:       store,
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
	h.logger.Debug("GitLabTestGenHandler initialized")
	return h
}

// getUserID extracts user ID from context
func (h *GitLabTestGenHandler) getUserID(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(string); ok {
			return id
		}
	}
	return ""
}

// canAccessProject checks if user can access the project (via integration ownership)
func (h *GitLabTestGenHandler) canAccessProject(c *gin.Context, project *models.GitLabProject) bool {
	userID := h.getUserID(c)
	if userID == "" {
		return false
	}

	// Admins can access everything (if is_admin is set by middleware)
	if isAdmin, exists := c.Get("is_admin"); exists {
		if a, ok := isAdmin.(bool); ok && a {
			return true
		}
	}

	// Get integration to check ownership
	ctx := c.Request.Context()
	integration, err := h.store.GetIntegration(ctx, project.IntegrationID)
	if err != nil || integration == nil {
		return false
	}

	// Check ownership
	if integration.OwnerID == userID {
		return true
	}

	// Check tenant membership
	if tids, exists := c.Get("tenant_ids"); exists {
		if ids, ok := tids.([]string); ok {
			for _, tid := range ids {
				if integration.TenantID == tid {
					return true
				}
			}
		}
	}

	return false
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

// ScanMyTestable handles user-level test scan
func (h *GitLabTestGenHandler) ScanMyTestable(c *gin.Context) {
	projectID := c.Param("id")
	ctx := c.Request.Context()

	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if !h.canAccessProject(c, project) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	h.ScanTestable(c)
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

// GenerateMyTests handles user-level test generation
func (h *GitLabTestGenHandler) GenerateMyTests(c *gin.Context) {
	projectID := c.Param("id")
	ctx := c.Request.Context()

	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if !h.canAccessProject(c, project) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	h.GenerateTests(c)
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

// CreateTestsMRRequest is the request body for creating a MR with tests.
type CreateTestsMRRequest struct {
	Tests         []testgen.GeneratedTest `json:"tests" binding:"required"`
	Title         string                  `json:"title,omitempty"`
	Description   string                  `json:"description,omitempty"`
	TargetBranch  string                  `json:"target_branch,omitempty"`
	Labels        []string                `json:"labels,omitempty"`
	CommitMessage string                  `json:"commit_message,omitempty"`
	TestDir       string                  `json:"test_dir,omitempty"` // Directory for test files, e.g. "tests/"
}

// CreateTestsMR POST /api/admin/gitlab/projects/:id/create-tests-mr
// Creates a GitLab MR with generated tests.
func (h *GitLabTestGenHandler) CreateTestsMR(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req CreateTestsMRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if len(req.Tests) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No tests provided"})
		return
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

	// Get integration for GitLab API access
	integration, err := h.store.GetIntegration(ctx, project.IntegrationID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get integration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get GitLab integration"})
		return
	}

	// Prepare MR details
	title := req.Title
	if title == "" {
		title = fmt.Sprintf("[Auto-Test] Add %d unit tests", len(req.Tests))
	}

	description := req.Description
	if description == "" {
		var sb strings.Builder
		sb.WriteString("## 🧪 Auto-Generated Unit Tests\n\n")
		sb.WriteString("⚠️ **REVIEW REQUIRED**: This MR contains AI-generated code that needs manual review.\n\n")
		sb.WriteString(fmt.Sprintf("This MR adds **%d unit tests** to the codebase.\n\n", len(req.Tests)))

		sb.WriteString("### ⚠️ Before Merge - Please Check\n\n")
		sb.WriteString("1. **Import paths** - AI may use placeholder paths like `yourapp/...` that need to be replaced with actual module paths\n")
		sb.WriteString("2. **Test logic** - Verify assertions and test cases are correct for your business logic\n")
		sb.WriteString("3. **Dependencies** - Ensure all imported packages are available in go.mod/package.json\n")
		sb.WriteString("4. **Run tests** - Execute `go test` / `npm test` to verify tests pass\n\n")

		sb.WriteString("### Tests Added\n\n")

		// Group by source file
		fileTests := make(map[string][]string)
		for _, t := range req.Tests {
			fileTests[t.Function.FilePath] = append(fileTests[t.Function.FilePath], t.TestName)
		}

		for filePath, tests := range fileTests {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", filePath, strings.Join(tests, ", ")))
		}

		sb.WriteString("\n---\n")
		sb.WriteString("*Generated by AIGateway Test Generator*\n")
		description = sb.String()
	}

	targetBranch := req.TargetBranch
	if targetBranch == "" {
		targetBranch = project.DefaultBranch
	}
	if targetBranch == "" {
		targetBranch = "main" // Fallback if not set
	}

	commitMessage := req.CommitMessage
	if commitMessage == "" {
		commitMessage = fmt.Sprintf("test: add %d auto-generated unit tests\n\nGenerated by AIGateway", len(req.Tests))
	}

	labels := req.Labels
	if len(labels) == 0 {
		labels = []string{"tests", "ai-generated", "auto-test"}
	}

	testDir := req.TestDir
	if testDir == "" {
		testDir = "" // Put test files alongside source files
	}

	h.logger.WithFields(logrus.Fields{
		"project_id":    projectID,
		"tests_count":   len(req.Tests),
		"target_branch": targetBranch,
	}).Info("Creating tests MR")

	// Create GitLab client
	gitlabClient := client.NewClient(client.ClientConfig{
		BaseURL:     integration.BaseURL,
		AccessToken: integration.AccessToken,
		Timeout:     2 * time.Minute,
	})

	// Group tests by output file
	fileContents := make(map[string][]string)
	for _, t := range req.Tests {
		// Determine test file path
		testFilePath := getTestFilePath(t.Function.FilePath, t.Language, testDir)
		fileContents[testFilePath] = append(fileContents[testFilePath], t.TestCode)
	}

	// Create commit actions for each test file
	var actions []client.CommitAction
	for testFilePath, codes := range fileContents {
		// Check if file already exists in the target branch
		action := "create"
		_, err := gitlabClient.GetFile(ctx, project.GitLabProjectID, testFilePath, targetBranch)
		if err == nil {
			// File exists, use update action
			action = "update"
			h.logger.WithField("file", testFilePath).Debug("Test file exists, will update")
		}

		// Get language from first test for this file
		var lang string
		for _, t := range req.Tests {
			if getTestFilePath(t.Function.FilePath, t.Language, testDir) == testFilePath {
				lang = t.Language
				break
			}
		}

		// Build file content with header
		var content strings.Builder
		content.WriteString(getTestFileHeader(lang, projectID))
		content.WriteString("\n\n")
		for i, code := range codes {
			if i > 0 {
				content.WriteString("\n\n")
			}
			content.WriteString(code)
		}
		// Close rust test module if needed
		if strings.ToLower(lang) == "rust" || strings.ToLower(lang) == "rs" {
			content.WriteString("\n}")
		}

		actions = append(actions, client.CommitAction{
			Action:   action,
			FilePath: testFilePath,
			Content:  content.String(),
		})
	}

	if len(actions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No test files to create"})
		return
	}

	// Create MR with test files
	mr, err := gitlabClient.CreateMRWithChanges(ctx, project.GitLabProjectID, client.CreateMRWithChangesOptions{
		BranchPrefix:  "tests/auto-test",
		TargetBranch:  targetBranch,
		CommitMessage: commitMessage,
		Actions:       actions,
		AuthorName:    "AIGateway Test Generator",
		AuthorEmail:   "aigateway@localhost",
		MRTitle:       title,
		MRDescription: description,
		Labels:        strings.Join(labels, ","),
	})
	if err != nil {
		h.logger.WithError(err).Error("Failed to create tests MR")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create merge request: " + err.Error()})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"mr_iid":     mr.IID,
		"mr_url":     mr.WebURL,
	}).Info("Tests MR created successfully")

	c.JSON(http.StatusOK, gin.H{
		"status":        "created",
		"message":       "Merge request with tests created successfully",
		"mr_id":         mr.ID,
		"mr_iid":        mr.IID,
		"mr_url":        mr.WebURL,
		"tests_count":   len(req.Tests),
		"files_created": len(actions),
	})
}

// CreateMyTestsMR handles user-level MR creation for tests
func (h *GitLabTestGenHandler) CreateMyTestsMR(c *gin.Context) {
	projectID := c.Param("id")
	ctx := c.Request.Context()

	project, err := h.store.GetProject(ctx, projectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}

	if !h.canAccessProject(c, project) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	h.CreateTestsMR(c)
}

// getTestFilePath returns the path for a test file based on source file
func getTestFilePath(sourceFilePath, language, testDir string) string {
	dir := filepath.Dir(sourceFilePath)
	base := filepath.Base(sourceFilePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	if testDir != "" {
		dir = testDir
	}

	switch strings.ToLower(language) {
	case "go":
		return filepath.Join(dir, name+"_test.go")
	case "typescript", "ts":
		return filepath.Join(dir, name+".test.ts")
	case "javascript", "js":
		return filepath.Join(dir, name+".test.js")
	case "python", "py":
		return filepath.Join(dir, "test_"+name+".py")
	case "rust", "rs":
		return filepath.Join(dir, name+"_test.rs")
	case "java":
		return filepath.Join(dir, name+"Test.java")
	default:
		return filepath.Join(dir, name+"_test.txt")
	}
}
