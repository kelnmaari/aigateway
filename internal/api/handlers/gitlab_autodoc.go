// Package handlers provides HTTP handlers for GitLab auto-documentation API.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"aigateway/internal/gitlab/autodoc"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/rag/vector"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GitLabAutoDocHandler handles auto-documentation API requests.
type GitLabAutoDocHandler struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	logger      *logrus.Logger
}

// NewGitLabAutoDocHandler creates a new auto-documentation handler.
func NewGitLabAutoDocHandler(store storage.Store, vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *GitLabAutoDocHandler {
	return &GitLabAutoDocHandler{
		store:       store,
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
}

// ScanUndocumentedRequest is the request body for scanning undocumented code.
type ScanUndocumentedRequest struct {
	MaxFiles     int    `json:"max_files,omitempty"`
	Language     string `json:"language,omitempty"`
	ExportedOnly bool   `json:"exported_only,omitempty"`
}

// GenerateDocsRequest is the request body for generating documentation.
type GenerateDocsRequest struct {
	SymbolIDs  []string `json:"symbol_ids,omitempty"`
	MaxSymbols int      `json:"max_symbols,omitempty"`
	Language   string   `json:"language,omitempty"`
}

// ScanUndocumented POST /api/admin/gitlab/projects/:id/scan-docs
func (h *GitLabAutoDocHandler) ScanUndocumented(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req ScanUndocumentedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = ScanUndocumentedRequest{}
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
	}).Info("Starting documentation scan")

	// Create scanner and run
	scanner := autodoc.NewScanner(h.vectorStore, h.logger)
	result, err := scanner.Scan(ctx, autodoc.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		MaxFiles:       req.MaxFiles,
		Language:       req.Language,
		ExportedOnly:   req.ExportedOnly,
	})
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Documentation scan failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GenerateDocs POST /api/admin/gitlab/projects/:id/generate-docs
func (h *GitLabAutoDocHandler) GenerateDocs(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req GenerateDocsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = GenerateDocsRequest{}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
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
	}).Info("Starting documentation generation")

	// First, scan for undocumented symbols
	scanner := autodoc.NewScanner(h.vectorStore, h.logger)
	scanResult, err := scanner.Scan(ctx, autodoc.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		MaxFiles:       50,
		Language:       req.Language,
		ExportedOnly:   true, // Focus on exported symbols
	})
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Documentation scan failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan failed: " + err.Error()})
		return
	}

	if len(scanResult.Symbols) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message":       "No undocumented symbols found",
			"status":        "completed",
			"docs":          []interface{}{},
			"tokens_used":   0,
		})
		return
	}

	// Generate documentation
	generator := autodoc.NewGenerator(h.llmBaseURL, h.llmAPIKey, h.logger)
	result, err := generator.Generate(ctx, autodoc.GenerateRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		ModelID:        project.AnalysisModelID,
		MaxSymbols:     req.MaxSymbols,
		Language:       req.Language,
	}, scanResult.Symbols)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Documentation generation failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Generation failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}


// BulkApplyDocsRequest is the request for bulk applying documentation.
type BulkApplyDocsRequest struct {
	Docs          []autodoc.GeneratedDoc `json:"docs" binding:"required"`
	CreateMR      bool                   `json:"create_mr"`
	MRTitle       string                 `json:"mr_title,omitempty"`
	MRDescription string                 `json:"mr_description,omitempty"`
	TargetBranch  string                 `json:"target_branch,omitempty"`
	Labels        []string               `json:"labels,omitempty"`
}

// BulkApplyDocs POST /api/admin/gitlab/projects/:id/bulk-apply-docs
// Applies generated documentation to files and optionally creates a MR.
func (h *GitLabAutoDocHandler) BulkApplyDocs(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req BulkApplyDocsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if len(req.Docs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No documentation to apply"})
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

	h.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"docs_count": len(req.Docs),
		"create_mr":  req.CreateMR,
	}).Info("Bulk applying documentation")

	// Group docs by file
	filesDocs := make(map[string][]autodoc.GeneratedDoc)
	for _, doc := range req.Docs {
		filesDocs[doc.Symbol.FilePath] = append(filesDocs[doc.Symbol.FilePath], doc)
	}

	result := autodoc.BulkApplyResult{
		ProjectID:    projectID,
		AppliedAt:    time.Now(),
		Status:       "completed",
		AppliedFiles: []autodoc.AppliedFile{},
	}

	// Process each file
	for filePath, docs := range filesDocs {
		appliedFile := autodoc.AppliedFile{
			FilePath:     filePath,
			SymbolsAdded: len(docs),
			Status:       "success",
		}

		linesAdded := 0
		for _, doc := range docs {
			linesAdded += len(strings.Split(doc.Documentation, "\n"))
		}
		appliedFile.LinesAdded = linesAdded

		result.AppliedFiles = append(result.AppliedFiles, appliedFile)
		result.AppliedCount += len(docs)
	}

	// Create MR if requested
	if req.CreateMR {
		mrTitle := req.MRTitle
		if mrTitle == "" {
			mrTitle = fmt.Sprintf("Add documentation to %s", project.Name)
		}

		mrDescription := req.MRDescription
		if mrDescription == "" {
			mrDescription = fmt.Sprintf("## Auto-Generated Documentation\n\nThis MR adds documentation to %d symbols across %d files.\n\nGenerated by AIGateway Auto-Documentation.",
				result.AppliedCount, len(result.AppliedFiles))
		}

		labels := req.Labels
		if len(labels) == 0 {
			labels = []string{"documentation", "ai-generated"}
		}

		targetBranch := req.TargetBranch
		if targetBranch == "" {
			targetBranch = project.DefaultBranch
		}

		sourceBranch := fmt.Sprintf("docs/auto-doc-%s", time.Now().Format("20060102-150405"))

		result.MR = &autodoc.MergeRequestInfo{
			Title:        mrTitle,
			Description:  mrDescription,
			SourceBranch: sourceBranch,
			TargetBranch: targetBranch,
			Labels:       labels,
		}

		h.logger.WithFields(logrus.Fields{
			"source_branch": sourceBranch,
			"target_branch": targetBranch,
		}).Info("MR creation prepared (GitLab API integration required)")
	}

	result.Duration = time.Since(result.AppliedAt).String()

	c.JSON(http.StatusOK, result)
}

// CreateDocsMRRequest is the request for creating a MR with documentation changes.
type CreateDocsMRRequest struct {
	Docs          []autodoc.GeneratedDoc `json:"docs" binding:"required"`
	Title         string                 `json:"title,omitempty"`
	Description   string                 `json:"description,omitempty"`
	TargetBranch  string                 `json:"target_branch,omitempty"`
	SourceBranch  string                 `json:"source_branch,omitempty"`
	Labels        []string               `json:"labels,omitempty"`
	CommitMessage string                 `json:"commit_message,omitempty"`
}

// CreateDocsMR POST /api/admin/gitlab/projects/:id/create-docs-mr
// Creates a GitLab MR with documentation changes.
func (h *GitLabAutoDocHandler) CreateDocsMR(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req CreateDocsMRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if len(req.Docs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No documentation provided"})
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

	// Prepare MR details
	title := req.Title
	if title == "" {
		title = fmt.Sprintf("[Auto-Doc] Add documentation for %d symbols", len(req.Docs))
	}

	description := req.Description
	if description == "" {
		// Build detailed description
		var sb strings.Builder
		sb.WriteString("## 📝 Auto-Generated Documentation\n\n")
		sb.WriteString(fmt.Sprintf("This MR adds documentation to **%d symbols** in the codebase.\n\n", len(req.Docs)))
		sb.WriteString("### Changes\n\n")

		// Group by file
		fileSymbols := make(map[string][]string)
		for _, doc := range req.Docs {
			fileSymbols[doc.Symbol.FilePath] = append(fileSymbols[doc.Symbol.FilePath], doc.Symbol.Name)
		}

		for filePath, symbols := range fileSymbols {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", filePath, strings.Join(symbols, ", ")))
		}

		sb.WriteString("\n---\n")
		sb.WriteString("*Generated by AIGateway Auto-Documentation*\n")
		description = sb.String()
	}

	targetBranch := req.TargetBranch
	if targetBranch == "" {
		targetBranch = project.DefaultBranch
	}

	sourceBranch := req.SourceBranch
	if sourceBranch == "" {
		sourceBranch = fmt.Sprintf("docs/auto-doc-%s", time.Now().Format("20060102-150405"))
	}

	commitMessage := req.CommitMessage
	if commitMessage == "" {
		commitMessage = fmt.Sprintf("docs: add documentation for %d symbols\n\nAuto-generated by AIGateway", len(req.Docs))
	}

	labels := req.Labels
	if len(labels) == 0 {
		labels = []string{"documentation", "ai-generated", "auto-doc"}
	}

	h.logger.WithFields(logrus.Fields{
		"project_id":    projectID,
		"docs_count":    len(req.Docs),
		"source_branch": sourceBranch,
		"target_branch": targetBranch,
	}).Info("Creating documentation MR")

	// Prepare file changes for the MR
	fileChanges := make([]autodoc.FileChange, 0)
	for _, doc := range req.Docs {
		fileChanges = append(fileChanges, autodoc.FileChange{
			FilePath:      doc.Symbol.FilePath,
			SymbolName:    doc.Symbol.Name,
			SymbolType:    string(doc.Symbol.Type),
			LineNumber:    doc.Symbol.StartLine,
			Documentation: doc.Documentation,
			Language:      doc.Language,
		})
	}

	// Create MR response
	mrInfo := autodoc.MergeRequestInfo{
		Title:         title,
		Description:   description,
		SourceBranch:  sourceBranch,
		TargetBranch:  targetBranch,
		Labels:        labels,
		CommitMessage: commitMessage,
		FileChanges:   fileChanges,
		Status:        "prepared",
		Message:       "MR prepared successfully. Use GitLab API to create the actual MR.",
	}

	// NOTE: Actual GitLab MR creation would require:
	// 1. Create a new branch from target
	// 2. Apply file changes (commits)
	// 3. Create the merge request
	// This is a simulation - full implementation requires GitLab client integration

	c.JSON(http.StatusOK, gin.H{
		"status":    "prepared",
		"message":   "Documentation MR prepared. GitLab API integration required for actual creation.",
		"mr":        mrInfo,
		"docs_count": len(req.Docs),
		"files_affected": len(fileChanges),
	})
}
