// Package indexer provides repository indexing for RAG-powered code review
package indexer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"aigateway/internal/gitlab/chunker"
	"aigateway/internal/gitlab/client"
	"aigateway/internal/gitlab/rag"
)

// IndexStatus represents the current state of repository indexing
type IndexStatus string

const (
	IndexStatusPending    IndexStatus = "pending"
	IndexStatusInProgress IndexStatus = "in_progress"
	IndexStatusCompleted  IndexStatus = "completed"
	IndexStatusFailed     IndexStatus = "failed"
)

// IndexInfo holds information about an indexed branch
type IndexInfo struct {
	ProjectID    string      `json:"project_id"`
	Branch       string      `json:"branch"`
	Status       IndexStatus `json:"status"`
	FilesIndexed int         `json:"files_indexed"`
	ChunksTotal  int         `json:"chunks_total"`
	LastIndexed  *time.Time  `json:"last_indexed,omitempty"`
	Error        string      `json:"error,omitempty"`
	StartedAt    *time.Time  `json:"started_at,omitempty"`
	CompletedAt  *time.Time  `json:"completed_at,omitempty"`
}

// IndexerConfig holds configuration for the indexer
type IndexerConfig struct {
	MaxFileSizeKB     int           // Max file size to index (default: 500KB)
	MaxFilesPerRepo   int           // Max files to index per repo (default: 5000)
	ConcurrentWorkers int           // Number of concurrent indexing workers (default: 5)
	SupportedExts     []string      // File extensions to index
	ExcludeDirs       []string      // Directories to exclude
	BatchSize         int           // Batch size for embedding generation (default: 10)
	IndexTimeout      time.Duration // Overall timeout for indexing operation (default: 30 min)
}

// DefaultConfig returns default indexer configuration
func DefaultConfig() IndexerConfig {
	return IndexerConfig{
		MaxFileSizeKB:     500,
		MaxFilesPerRepo:   5000,
		ConcurrentWorkers: 5,
		BatchSize:         10,
		IndexTimeout:      30 * time.Minute, // 30 minutes for large repos
		SupportedExts: []string{
			".go", ".py", ".js", ".ts", ".jsx", ".tsx",
			".java", ".kt", ".scala", ".rs", ".rb", ".php",
			".c", ".cpp", ".h", ".hpp", ".cs", ".swift",
			".vue", ".svelte", ".html", ".css", ".scss",
			".sql", ".sh", ".bash", ".yaml", ".yml", ".json",
			".md", ".txt", ".dockerfile", ".tf", ".hcl",
		},
		ExcludeDirs: []string{
			"node_modules", "vendor", ".git", "__pycache__",
			"dist", "build", "target", ".idea", ".vscode",
			"bin", "obj", "packages", ".next", ".nuxt",
		},
	}
}

// Indexer handles repository indexing for RAG
type Indexer struct {
	config     IndexerConfig
	ragService *rag.RAGService
	chunker    *chunker.CodeChunker
	logger     *logrus.Logger

	// Track indexing status per project+branch
	statusMu sync.RWMutex
	status   map[string]*IndexInfo // key: "projectID:branch"

	// GitLab clients cache
	clientsMu sync.RWMutex
	clients   map[string]*client.Client // key: integrationID
}

// NewIndexer creates a new repository indexer with dedicated log file
func NewIndexer(ragService *rag.RAGService, logger *logrus.Logger) *Indexer {
	return &Indexer{
		config:     DefaultConfig(),
		ragService: ragService,
		chunker:    chunker.NewCodeChunker(chunker.DefaultConfig()),
		logger:     logger, // Will be replaced by dedicated logger in NewIndexerWithLogger
		status:     make(map[string]*IndexInfo),
		clients:    make(map[string]*client.Client),
	}
}

// NewIndexerWithConfig creates indexer with custom config
func NewIndexerWithConfig(ragService *rag.RAGService, config IndexerConfig, logger *logrus.Logger) *Indexer {
	idx := NewIndexer(ragService, logger)
	idx.config = config
	return idx
}

// RegisterClient registers a GitLab client for an integration
func (i *Indexer) RegisterClient(integrationID string, c *client.Client) {
	i.clientsMu.Lock()
	defer i.clientsMu.Unlock()
	i.clients[integrationID] = c
}

// GetClient returns the GitLab client for an integration
func (i *Indexer) GetClient(integrationID string) (*client.Client, bool) {
	i.clientsMu.RLock()
	defer i.clientsMu.RUnlock()
	c, ok := i.clients[integrationID]
	return c, ok
}

// statusKey generates a key for status map
func statusKey(projectID, branch string) string {
	return fmt.Sprintf("%s:%s", projectID, branch)
}

// GetStatus returns the current indexing status for a project+branch
func (i *Indexer) GetStatus(projectID, branch string) *IndexInfo {
	i.statusMu.RLock()
	defer i.statusMu.RUnlock()

	key := statusKey(projectID, branch)
	if info, ok := i.status[key]; ok {
		return info
	}

	return &IndexInfo{
		ProjectID: projectID,
		Branch:    branch,
		Status:    IndexStatusPending,
	}
}

// setStatus updates indexing status
func (i *Indexer) setStatus(projectID, branch string, info *IndexInfo) {
	i.statusMu.Lock()
	defer i.statusMu.Unlock()
	i.status[statusKey(projectID, branch)] = info
}

// IndexRequest holds parameters for indexing
type IndexRequest struct {
	IntegrationID    string // Integration UUID
	ProjectID        string // Internal project UUID
	GitLabProjectID  int64  // GitLab project ID (numeric)
	Branch           string // Branch to index (target branch)
	EmbeddingModelID string // Embedding model alias (optional, uses project default)
	CollectionName   string // Qdrant collection name (optional, uses project default or generated)
	Force            bool   // Force reindex even if already indexed
}

// IndexResult holds the result of indexing operation
type IndexResult struct {
	ProjectID    string      `json:"project_id"`
	Branch       string      `json:"branch"`
	Status       IndexStatus `json:"status"`
	FilesIndexed int         `json:"files_indexed"`
	ChunksTotal  int         `json:"chunks_total"`
	Duration     string      `json:"duration"`
	Error        string      `json:"error,omitempty"`
}

// IndexBranch indexes a branch of a repository
// This is an async operation - returns immediately and runs in background
func (i *Indexer) IndexBranch(ctx context.Context, req IndexRequest) error {
	// Check if already indexing
	status := i.GetStatus(req.ProjectID, req.Branch)
	if status.Status == IndexStatusInProgress {
		return fmt.Errorf("indexing already in progress for %s:%s", req.ProjectID, req.Branch)
	}

	// Get GitLab client
	gitlabClient, ok := i.GetClient(req.IntegrationID)
	if !ok {
		return fmt.Errorf("no GitLab client registered for integration %s", req.IntegrationID)
	}

	// Start async indexing with detached background context
	// HTTP request context gets cancelled when response is sent,
	// so we use background context with timeout for long-running indexing
	// This allows indexing to continue even if user closes browser or refreshes page
	_ = ctx // original HTTP context ignored to prevent cancellation on page refresh

	go func() {
		// Recover from panics in background goroutine
		defer func() {
			if r := recover(); r != nil {
				i.logger.WithFields(logrus.Fields{
					"project_id": req.ProjectID,
					"branch":     req.Branch,
					"panic":      r,
				}).Error("PANIC in repository indexing goroutine")
				i.updateStatusFailed(req.ProjectID, req.Branch, fmt.Sprintf("panic: %v", r))
			}
		}()

		// Create background context with generous timeout for large repos
		indexCtx, cancel := context.WithTimeout(context.Background(), i.config.IndexTimeout)
		defer cancel()
		i.runIndexing(indexCtx, gitlabClient, req)
	}()

	return nil
}

// IndexBranchSync indexes a branch synchronously (blocking)
func (i *Indexer) IndexBranchSync(ctx context.Context, req IndexRequest) (*IndexResult, error) {
	// Get GitLab client
	gitlabClient, ok := i.GetClient(req.IntegrationID)
	if !ok {
		return nil, fmt.Errorf("no GitLab client registered for integration %s", req.IntegrationID)
	}

	return i.runIndexing(ctx, gitlabClient, req), nil
}

// runIndexing performs the actual indexing work
func (i *Indexer) runIndexing(ctx context.Context, gitlabClient *client.Client, req IndexRequest) *IndexResult {
	startTime := time.Now()
	now := startTime

	// Update status to in-progress
	info := &IndexInfo{
		ProjectID: req.ProjectID,
		Branch:    req.Branch,
		Status:    IndexStatusInProgress,
		StartedAt: &now,
	}
	i.setStatus(req.ProjectID, req.Branch, info)

	i.logger.WithFields(logrus.Fields{
		"project_id":        req.ProjectID,
		"gitlab_project_id": req.GitLabProjectID,
		"branch":            req.Branch,
	}).Info("Starting repository indexing")

	result := &IndexResult{
		ProjectID: req.ProjectID,
		Branch:    req.Branch,
	}

	// Step 1: Delete existing index if force reindex
	if req.Force {
		if err := i.ragService.DeleteProjectBranchIndex(ctx, req.ProjectID, req.Branch); err != nil {
			i.logger.WithError(err).Warn("Failed to delete existing index, continuing anyway")
		} else {
			i.logger.Debug("Deleted existing index for reindexing")
		}
	}

	// Step 2: Get repository tree with branch fallback
	i.logger.Info("Fetching repository file tree from GitLab...")

	// Try multiple branches if the specified one doesn't exist
	branchesToTry := []string{req.Branch}
	if req.Branch != "" {
		// Add fallback branches if primary fails
		fallbackBranches := []string{"main", "master", "develop", "dev"}
		for _, fb := range fallbackBranches {
			if fb != req.Branch {
				branchesToTry = append(branchesToTry, fb)
			}
		}
	}

	var files []TreeEntry
	var err error
	var usedBranch string

	for _, branch := range branchesToTry {
		i.logger.WithField("branch", branch).Info("Trying to fetch repository tree...")
		files, err = i.getRepositoryFiles(ctx, gitlabClient, req.GitLabProjectID, branch)
		if err == nil {
			usedBranch = branch
			if branch != req.Branch {
				i.logger.WithFields(logrus.Fields{
					"original_branch": req.Branch,
					"used_branch":     branch,
				}).Info("Fallback branch used successfully")
			}
			break
		}

		// If 404, try next branch
		if strings.Contains(err.Error(), "404") {
			i.logger.WithFields(logrus.Fields{
				"branch": branch,
			}).Debug("Branch not found, trying next fallback...")
			continue
		}

		// For other errors, fail immediately
		break
	}

	if err != nil {
		i.logger.WithError(err).WithFields(logrus.Fields{
			"project_id":        req.ProjectID,
			"gitlab_project_id": req.GitLabProjectID,
			"branches_tried":    branchesToTry,
		}).Error("Failed to get repository files from GitLab (all branches failed)")
		result.Status = IndexStatusFailed
		result.Error = fmt.Sprintf("failed to get repository files: %v", err)
		i.updateStatusFailed(req.ProjectID, req.Branch, result.Error)
		return result
	}

	// Update branch in request to the one that worked
	req.Branch = usedBranch

	i.logger.WithFields(logrus.Fields{
		"files_count": len(files),
		"branch":      usedBranch,
	}).Info("Retrieved repository file list")

	// Step 3: Process files in batches
	i.logger.WithField("files_count", len(files)).Debug("Processing files and chunking...")
	chunks, err := i.processFiles(ctx, gitlabClient, req, files)
	if err != nil {
		i.logger.WithError(err).WithFields(logrus.Fields{
			"project_id": req.ProjectID,
			"branch":     req.Branch,
		}).Error("Failed to process files")
		result.Status = IndexStatusFailed
		result.Error = fmt.Sprintf("failed to process files: %v", err)
		i.updateStatusFailed(req.ProjectID, req.Branch, result.Error)
		return result
	}

	i.logger.WithField("chunks_count", len(chunks)).Info("Files chunked for indexing")

	// Step 4: Generate embeddings and index chunks
	i.logger.WithField("chunks_count", len(chunks)).Debug("Generating embeddings and indexing chunks...")
	if err := i.indexChunks(ctx, req, chunks); err != nil {
		i.logger.WithError(err).WithFields(logrus.Fields{
			"project_id":   req.ProjectID,
			"branch":       req.Branch,
			"chunks_count": len(chunks),
		}).Error("Failed to index chunks (generate embeddings)")
		result.Status = IndexStatusFailed
		result.Error = fmt.Sprintf("failed to index chunks: %v", err)
		i.updateStatusFailed(req.ProjectID, req.Branch, result.Error)
		return result
	}

	// Success
	completedAt := time.Now()
	result.Status = IndexStatusCompleted
	result.FilesIndexed = len(files)
	result.ChunksTotal = len(chunks)
	result.Duration = completedAt.Sub(startTime).String()

	// Update final status
	finalInfo := &IndexInfo{
		ProjectID:    req.ProjectID,
		Branch:       req.Branch,
		Status:       IndexStatusCompleted,
		FilesIndexed: len(files),
		ChunksTotal:  len(chunks),
		LastIndexed:  &completedAt,
		StartedAt:    &now,
		CompletedAt:  &completedAt,
	}
	i.setStatus(req.ProjectID, req.Branch, finalInfo)

	// Log completion with summary
	i.logger.WithFields(logrus.Fields{
		"project_id":    req.ProjectID,
		"branch":        req.Branch,
		"files_indexed": len(files),
		"chunks_total":  len(chunks),
		"duration":      result.Duration,
		"collection":    req.CollectionName,
	}).Info("Repository indexing completed")

	// Log detailed summary to INFO level for visibility
	i.logger.Infof("📊 INDEX SUMMARY: %d files → %d chunks in %s (collection: %s)",
		len(files), len(chunks), result.Duration, req.CollectionName)

	return result
}

// updateStatusFailed updates status to failed with error message
func (i *Indexer) updateStatusFailed(projectID, branch, errMsg string) {
	now := time.Now()
	info := &IndexInfo{
		ProjectID:   projectID,
		Branch:      branch,
		Status:      IndexStatusFailed,
		Error:       errMsg,
		CompletedAt: &now,
	}
	i.setStatus(projectID, branch, info)
}

// TreeEntry represents a file in the repository tree
type TreeEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // "blob" or "tree"
	Path string `json:"path"`
	Mode string `json:"mode"`
}

// getRepositoryFiles retrieves the list of files to index from GitLab
func (i *Indexer) getRepositoryFiles(ctx context.Context, c *client.Client, projectID int64, branch string) ([]TreeEntry, error) {
	var allFiles []TreeEntry

	// GitLab API: GET /projects/:id/repository/tree
	// We need to paginate and filter
	page := 1
	perPage := 100

	for {
		entries, hasMore, err := i.getTreePage(ctx, c, projectID, branch, page, perPage)
		if err != nil {
			return nil, err
		}

		for _, entry := range entries {
			// Only include files (blobs), not directories
			if entry.Type != "blob" {
				continue
			}

			// Check if file should be indexed
			if i.shouldIndexFile(entry.Path) {
				allFiles = append(allFiles, entry)
			}
		}

		if !hasMore || len(allFiles) >= i.config.MaxFilesPerRepo {
			break
		}
		page++
	}

	// Limit to max files
	if len(allFiles) > i.config.MaxFilesPerRepo {
		allFiles = allFiles[:i.config.MaxFilesPerRepo]
	}

	return allFiles, nil
}

// getTreePage retrieves a page of the repository tree
func (i *Indexer) getTreePage(ctx context.Context, c *client.Client, projectID int64, branch string, page, perPage int) ([]TreeEntry, bool, error) {
	// Use GitLab API to get tree with recursive flag
	path := fmt.Sprintf("/projects/%d/repository/tree?ref=%s&recursive=true&per_page=%d&page=%d",
		projectID, branch, perPage, page)

	i.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"branch":     branch,
		"page":       page,
	}).Info("Fetching repository tree page from GitLab API")

	resp, err := c.DoRequest(ctx, "GET", path, nil)
	if err != nil {
		i.logger.WithError(err).WithField("path", path).Error("GitLab API request failed")
		return nil, false, fmt.Errorf("get repository tree: %w", err)
	}

	var entries []TreeEntry
	if err := c.ParseResponse(resp, &entries); err != nil {
		i.logger.WithError(err).Error("Failed to parse GitLab tree response")
		return nil, false, fmt.Errorf("parse tree response: %w", err)
	}

	i.logger.WithFields(logrus.Fields{
		"entries_count": len(entries),
		"page":          page,
	}).Debug("Received tree entries from GitLab")

	// Check if there are more pages
	hasMore := len(entries) == perPage

	return entries, hasMore, nil
}

// shouldIndexFile checks if a file should be indexed based on config
func (i *Indexer) shouldIndexFile(filePath string) bool {
	// Check excluded directories
	for _, excludeDir := range i.config.ExcludeDirs {
		if strings.Contains(filePath, excludeDir+"/") || strings.HasPrefix(filePath, excludeDir+"/") {
			return false
		}
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, supportedExt := range i.config.SupportedExts {
		if ext == supportedExt {
			return true
		}
	}

	// Also index files without extension that are common (Dockerfile, Makefile, etc.)
	baseName := strings.ToLower(filepath.Base(filePath))
	noExtFiles := []string{"dockerfile", "makefile", "rakefile", "gemfile", "procfile", "brewfile"}
	for _, name := range noExtFiles {
		if baseName == name {
			return true
		}
	}

	return false
}

// FileContent holds file content for processing
type FileContent struct {
	Path     string
	Content  string
	Language string
}

// processFiles downloads and chunks files
func (i *Indexer) processFiles(ctx context.Context, c *client.Client, req IndexRequest, files []TreeEntry) ([]rag.CodeChunk, error) {
	var allChunks []rag.CodeChunk
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Semaphore for concurrency control
	sem := make(chan struct{}, i.config.ConcurrentWorkers)

	for _, file := range files {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		wg.Add(1)
		sem <- struct{}{} // acquire

		go func(f TreeEntry) {
			defer wg.Done()
			defer func() { <-sem }() // release

			// Get file content
			content, err := c.GetFileRaw(ctx, req.GitLabProjectID, f.Path, req.Branch)
			if err != nil {
				i.logger.WithError(err).WithField("path", f.Path).Debug("Failed to get file content, skipping")
				return
			}

			// Check file size
			if len(content) > i.config.MaxFileSizeKB*1024 {
				i.logger.WithField("path", f.Path).Debug("File too large, skipping")
				return
			}

			// Detect language
			language := detectLanguage(f.Path)

			// Chunk the file
			fileChunks, chunkErr := i.chunker.Chunk(f.Path, string(content), "add")
			if chunkErr != nil {
				i.logger.WithError(chunkErr).WithField("path", f.Path).Debug("Failed to chunk file, skipping")
				return
			}

			// Convert to RAG chunks
			mu.Lock()
			for idx, chunk := range fileChunks {
				ragChunk := rag.CodeChunk{
					ProjectID:   req.ProjectID,
					FilePath:    f.Path,
					ChunkIndex:  idx,
					Language:    language,
					Content:     chunk.Content,
					StartLine:   chunk.LineStart,
					EndLine:     chunk.LineEnd,
					BranchName:  req.Branch,
					LastUpdated: time.Now(),
				}
				allChunks = append(allChunks, ragChunk)
			}
			mu.Unlock()

		}(file)
	}

	wg.Wait()

	return allChunks, nil
}

// indexChunks generates embeddings and stores chunks in Qdrant
func (i *Indexer) indexChunks(ctx context.Context, req IndexRequest, chunks []rag.CodeChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	// Ensure collection exists if specified
	if req.CollectionName != "" {
		if err := i.ragService.EnsureCollectionNamed(ctx, req.CollectionName); err != nil {
			return fmt.Errorf("ensure collection %s: %w", req.CollectionName, err)
		}
	}

	// Process in batches
	batchSize := i.config.BatchSize
	for start := 0; start < len(chunks); start += batchSize {
		end := start + batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[start:end]

		// Index batch (use collection name if specified)
		if err := i.ragService.IndexCodeChunksToCollection(ctx, req.CollectionName, batch, req.EmbeddingModelID); err != nil {
			return fmt.Errorf("index batch %d-%d: %w", start, end, err)
		}

		i.logger.WithFields(logrus.Fields{
			"batch":      fmt.Sprintf("%d-%d", start, end),
			"total":      len(chunks),
			"collection": req.CollectionName,
			"progress":   fmt.Sprintf("%.1f%%", float64(end)/float64(len(chunks))*100),
		}).Debug("Indexed batch")
	}

	return nil
}

// DeleteIndex removes the index for a project+branch
func (i *Indexer) DeleteIndex(ctx context.Context, projectID, branch string) error {
	if err := i.ragService.DeleteProjectBranchIndex(ctx, projectID, branch); err != nil {
		return fmt.Errorf("delete index: %w", err)
	}

	// Clear status
	i.statusMu.Lock()
	delete(i.status, statusKey(projectID, branch))
	i.statusMu.Unlock()

	i.logger.WithFields(logrus.Fields{
		"project_id": projectID,
		"branch":     branch,
	}).Info("Deleted repository index")

	return nil
}

// OnMergeRequest handles MR merge event - triggers reindex of target branch
func (i *Indexer) OnMergeRequest(ctx context.Context, req IndexRequest) error {
	i.logger.WithFields(logrus.Fields{
		"project_id":    req.ProjectID,
		"target_branch": req.Branch,
	}).Info("MR merged, triggering target branch reindex")

	// Force reindex the target branch
	req.Force = true
	return i.IndexBranch(ctx, req)
}

// detectLanguage detects programming language from file path
func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	baseName := strings.ToLower(filepath.Base(filePath))

	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".jsx":
		return "jsx"
	case ".tsx":
		return "tsx"
	case ".java":
		return "java"
	case ".kt", ".kts":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".rs":
		return "rust"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".h", ".hpp":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".swift":
		return "swift"
	case ".vue":
		return "vue"
	case ".svelte":
		return "svelte"
	case ".html":
		return "html"
	case ".css":
		return "css"
	case ".scss", ".sass":
		return "scss"
	case ".sql":
		return "sql"
	case ".sh", ".bash":
		return "bash"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".md":
		return "markdown"
	case ".tf", ".hcl":
		return "terraform"
	}

	// Check special files
	switch baseName {
	case "dockerfile":
		return "dockerfile"
	case "makefile":
		return "makefile"
	}

	return "text"
}
