// Package handlers provides HTTP handlers for GitLab architecture API.
package handlers

import (
	"context"
	"net/http"
	"time"

	"aigateway/internal/gitlab/architecture"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/rag/vector"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GitLabArchitectureHandler handles architecture diagram API requests.
type GitLabArchitectureHandler struct {
	store       storage.Store
	vectorStore *vector.QdrantStore
	llmBaseURL  string
	llmAPIKey   string
	logger      *logrus.Logger
}

// NewGitLabArchitectureHandler creates a new architecture handler.
func NewGitLabArchitectureHandler(store storage.Store, vectorStore *vector.QdrantStore, llmBaseURL, llmAPIKey string, logger *logrus.Logger) *GitLabArchitectureHandler {
	return &GitLabArchitectureHandler{
		store:       store,
		vectorStore: vectorStore,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		logger:      logger,
	}
}

// ScanArchitectureRequest is the request body for scanning architecture.
type ScanArchitectureRequest struct {
	MaxFiles int    `json:"max_files,omitempty"`
	Language string `json:"language,omitempty"`
}

// GenerateDiagramRequest is the request body for generating diagrams.
type GenerateDiagramRequest struct {
	Type     string `json:"type,omitempty"`     // "module_dependency", "call_graph", "data_flow", "package_structure"
	Format   string `json:"format,omitempty"`   // "mermaid", "svg", "png", "d3_json"
	Scope    string `json:"scope,omitempty"`    // Path prefix filter
	MaxDepth int    `json:"max_depth,omitempty"`
}

// ScanArchitecture POST /api/admin/gitlab/projects/:id/scan-architecture
func (h *GitLabArchitectureHandler) ScanArchitecture(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req ScanArchitectureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = ScanArchitectureRequest{}
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
	}).Info("Starting architecture scan")

	// Create generator and scan
	generator := architecture.NewGenerator(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)
	result, err := generator.Scan(ctx, architecture.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		MaxFiles:       req.MaxFiles,
		Language:       req.Language,
	})
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Architecture scan failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GenerateDiagram POST /api/admin/gitlab/projects/:id/generate-diagram
func (h *GitLabArchitectureHandler) GenerateDiagram(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	var req GenerateDiagramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req = GenerateDiagramRequest{}
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

	// Check if project is indexed
	collectionName := project.GetCollectionName()
	if collectionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project has no indexed collection. Please index the repository first."})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"project_id":   projectID,
		"diagram_type": req.Type,
		"format":       req.Format,
	}).Info("Starting diagram generation")

	generator := architecture.NewGenerator(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)

	// First scan
	scanResult, err := generator.Scan(ctx, architecture.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
	})
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Architecture scan failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan failed: " + err.Error()})
		return
	}

	// Parse diagram type
	diagramType := architecture.DiagramModuleDependency
	switch req.Type {
	case "module_dependency":
		diagramType = architecture.DiagramModuleDependency
	case "call_graph":
		diagramType = architecture.DiagramCallGraph
	case "data_flow":
		diagramType = architecture.DiagramDataFlow
	case "package_structure":
		diagramType = architecture.DiagramPackageStructure
	}

	// Parse format
	format := architecture.FormatMermaid
	switch req.Format {
	case "svg":
		format = architecture.FormatSVG
	case "png":
		format = architecture.FormatPNG
	case "d3_json":
		format = architecture.FormatD3JSON
	}

	// Generate diagram
	result, err := generator.Generate(ctx, architecture.GenerateRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		Type:           diagramType,
		Format:         format,
		ModelID:        project.AnalysisModelID,
		Scope:          req.Scope,
		MaxDepth:       req.MaxDepth,
	}, scanResult)
	if err != nil {
		h.logger.WithError(err).WithField("project_id", projectID).Error("Diagram generation failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Generation failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetArchitecture GET /api/admin/gitlab/projects/:id/architecture
// Returns the cached architecture data (or generates it).
func (h *GitLabArchitectureHandler) GetArchitecture(c *gin.Context) {
	projectID := c.Param("id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project ID is required"})
		return
	}

	diagramType := c.Query("type")
	if diagramType == "" {
		diagramType = "module_dependency"
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

	generator := architecture.NewGenerator(h.vectorStore, h.llmBaseURL, h.llmAPIKey, h.logger)

	// Scan
	scanResult, err := generator.Scan(ctx, architecture.ScanRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Scan failed: " + err.Error()})
		return
	}

	// Parse diagram type
	dtype := architecture.DiagramModuleDependency
	switch diagramType {
	case "call_graph":
		dtype = architecture.DiagramCallGraph
	case "data_flow":
		dtype = architecture.DiagramDataFlow
	case "package_structure":
		dtype = architecture.DiagramPackageStructure
	}

	// Generate
	result, err := generator.Generate(ctx, architecture.GenerateRequest{
		ProjectID:      projectID,
		CollectionName: collectionName,
		Type:           dtype,
		Format:         architecture.FormatMermaid,
		ModelID:        project.AnalysisModelID,
	}, scanResult)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Generation failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

