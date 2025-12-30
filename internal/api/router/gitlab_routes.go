// Package router provides GitLab integration routes
package router

import (
	"context"

	"aigateway/internal/api/handlers"
	"aigateway/internal/api/middleware"
	authMiddleware "aigateway/internal/auth/middleware"
	"aigateway/internal/gitlab/analytics"
	"aigateway/internal/gitlab/dependencies/schedule"
	"aigateway/internal/gitlab/storage"
	"aigateway/internal/rag/vector"
)

// SetupGitLabUserRoutes registers user-level GitLab routes (for regular users)
func (r *Router) SetupGitLabUserRoutes(store storage.Store) {
	if store == nil {
		r.logger.Warn("GitLab user routes: Store is nil, skipping setup")
		return
	}

	r.logger.Info("Setting up GitLab user routes")

	userHandler := handlers.NewGitLabUserHandler(store, r.logger)

	// User-level GitLab Routes (authenticated users, not admin)
	gitlab := r.engine.Group("/api/gitlab")

	// Apply user authentication (not admin)
	if r.jwtManager != nil && r.db != nil {
		gitlab.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		// No RequireAdmin - regular authenticated users can access
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		gitlab.Use(r.authenticator.AuthenticationMiddleware())
	}

	// ============================================================================
	// User's Integrations
	// ============================================================================
	gitlab.GET("/integrations", userHandler.ListMyIntegrations)
	gitlab.POST("/integrations", userHandler.CreateMyIntegration)
	gitlab.GET("/integrations/:id", userHandler.GetMyIntegration)
	gitlab.PUT("/integrations/:id", userHandler.UpdateMyIntegration)
	gitlab.DELETE("/integrations/:id", userHandler.DeleteMyIntegration)

	// ============================================================================
	// User's Projects (within their integrations)
	// ============================================================================
	gitlab.GET("/integrations/:id/projects", userHandler.ListMyProjects)
	gitlab.POST("/integrations/:id/projects", userHandler.AddMyProject)
	gitlab.GET("/projects/:project_id", userHandler.GetMyProject)
	gitlab.PUT("/projects/:project_id", userHandler.UpdateMyProject)
	gitlab.DELETE("/projects/:project_id", userHandler.DeleteMyProject)

	// ============================================================================
	// User's Reviews
	// ============================================================================
	gitlab.GET("/reviews", userHandler.ListMyReviews)
	gitlab.GET("/reviews/:id", userHandler.GetMyReview)

	r.logger.Info("GitLab user routes configured: /api/gitlab/*")
}

// SetupGitLabRoutes registers GitLab admin API routes
// Call this method from main.go after creating Router if GitLab is enabled
func (r *Router) SetupGitLabRoutes(store storage.Store) {
	if store == nil {
		r.logger.Warn("GitLab routes: Store is nil, skipping setup")
		return
	}

	r.logger.Info("Setting up GitLab admin routes")

	gitlabHandler := handlers.NewGitLabAdminHandler(store, r.logger)
	
	// Set main DB for model access
	if r.db != nil {
		gitlabHandler.SetMainDB(r.db)
	}

	// GitLab Admin Routes
	gitlab := r.engine.Group("/api/admin/gitlab")

	// Apply authentication
	if r.jwtManager != nil && r.db != nil {
		r.logger.Info("GitLab routes: Using JWT authentication with admin role check")
		gitlab.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		gitlab.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		r.logger.Info("GitLab routes: Using API Key authentication (legacy)")
		gitlab.Use(r.authenticator.AuthenticationMiddleware())
		gitlab.Use(r.authenticator.PermissionMiddleware("admin"))
	} else {
		r.logger.Warn("GitLab routes: No authentication enabled")
	}

	// ============================================================================
	// Integration Management
	// ============================================================================
	gitlab.GET("/integrations", gitlabHandler.ListIntegrations)
	gitlab.POST("/integrations", gitlabHandler.CreateIntegration)
	gitlab.GET("/integrations/:id", gitlabHandler.GetIntegration)
	gitlab.PUT("/integrations/:id", gitlabHandler.UpdateIntegration)
	gitlab.DELETE("/integrations/:id", gitlabHandler.DeleteIntegration)
	gitlab.POST("/integrations/:id/test", gitlabHandler.TestIntegration)

	// ============================================================================
	// Project Management (within integration)
	// ============================================================================
	gitlab.GET("/integrations/:id/projects", gitlabHandler.ListProjects)
	gitlab.POST("/integrations/:id/projects", gitlabHandler.AddProject)

	// ============================================================================
	// Project Management (direct)
	// ============================================================================
	gitlab.GET("/projects/:project_id", gitlabHandler.GetProject)
	gitlab.PUT("/projects/:project_id", gitlabHandler.UpdateProject)
	gitlab.DELETE("/projects/:project_id", gitlabHandler.DeleteProject)
	gitlab.POST("/projects/:project_id/webhook", gitlabHandler.SetupWebhook)

	// ============================================================================
	// Secrets Scanning (v4.0+)
	// ============================================================================
	// Note: Secrets handler is registered separately via SetupGitLabSecretsRoutes

	// ============================================================================
	// Review Management
	// ============================================================================
	gitlab.GET("/reviews", gitlabHandler.ListReviews)
	gitlab.GET("/reviews/:id", gitlabHandler.GetReview)
	gitlab.POST("/reviews/:id/retry", gitlabHandler.RetryReview)

	// ============================================================================
	// Queue Management
	// ============================================================================
	gitlab.GET("/queue/status", gitlabHandler.GetQueueStatus)
	gitlab.GET("/queue/jobs", gitlabHandler.ListJobs)
	gitlab.POST("/queue/jobs/:id/cancel", gitlabHandler.CancelJob)
	gitlab.POST("/queue/jobs/:id/retry", gitlabHandler.RetryJob)

	// ============================================================================
	// Model Selection (for GitLab project configuration)
	// ============================================================================
	gitlab.GET("/models", gitlabHandler.ListActiveModels)
	gitlab.GET("/models/analysis", gitlabHandler.ListAnalysisModels)
	gitlab.GET("/models/embedding", gitlabHandler.ListEmbeddingModels)

	r.logger.Info("GitLab admin routes configured successfully")

	// ============================================================================
	// Model Usage (extends existing admin routes)
	// ============================================================================
	// This should be called from admin group, add model usage endpoint
	admin := r.engine.Group("/api/admin")
	if r.jwtManager != nil && r.db != nil {
		admin.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		admin.Use(middleware.RequireAdmin(r.db, r.logger))
	}
	
	admin.GET("/models/:id/gitlab-usage", gitlabHandler.GetModelUsage)
	r.logger.Info("Model GitLab usage endpoint registered: GET /api/admin/models/:id/gitlab-usage")
}

// SetupGitLabWebhookRoute registers GitLab webhook endpoint
// This endpoint should not require authentication (uses webhook secret)
func (r *Router) SetupGitLabWebhookRoute(webhookHandler handlers.WebhookHandler) {
	if webhookHandler == nil {
		r.logger.Warn("GitLab webhook: Handler is nil, skipping setup")
		return
	}

	r.logger.Info("Setting up GitLab webhook routes")

	// Webhook endpoints - NO authentication, uses webhook secret verification
	// Legacy route
	r.engine.POST("/webhook/gitlab", webhookHandler.HandleWebhook)
	// New route with integration ID in path (used by frontend)
	r.engine.POST("/api/gitlab/webhook/:integration_id", webhookHandler.HandleWebhook)
	
	r.logger.Info("GitLab webhook routes configured: POST /webhook/gitlab, POST /api/gitlab/webhook/:integration_id")
}

// WebhookHandler interface for GitLab webhooks
// Defined in handlers package, re-exported here for convenience
type WebhookHandlerInterface interface {
	HandleWebhook(c interface{})
}

// SetupGitLabSecretsRoutes registers GitLab secrets scanning routes
func (r *Router) SetupGitLabSecretsRoutes(store storage.Store, vectorStore interface{}) {
	if store == nil {
		r.logger.Warn("GitLab secrets routes: Store is nil, skipping setup")
		return
	}

	// Check if vectorStore is a QdrantStore
	qdrantStore, ok := vectorStore.(*handlers.QdrantStoreInterface)
	if !ok || qdrantStore == nil {
		// Try direct type assertion
		if vs, ok := vectorStore.(handlers.QdrantStoreForSecrets); ok {
			r.logger.Info("Setting up GitLab secrets scanning routes")
			secretsHandler := handlers.NewGitLabSecretsHandlerWithInterface(store, vs, r.logger)
			r.registerSecretsRoutes(secretsHandler)
			return
		}
		r.logger.Warn("GitLab secrets routes: VectorStore is not available, skipping setup")
		return
	}

	r.logger.Info("Setting up GitLab secrets scanning routes")
	
	secretsHandler := handlers.NewGitLabSecretsHandlerWithInterface(store, *qdrantStore, r.logger)
	r.registerSecretsRoutes(secretsHandler)
}

func (r *Router) registerSecretsRoutes(secretsHandler *handlers.GitLabSecretsHandler) {
	gitlab := r.engine.Group("/api/admin/gitlab")

	// Apply authentication
	if r.jwtManager != nil && r.db != nil {
		gitlab.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		gitlab.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		gitlab.Use(r.authenticator.AuthenticationMiddleware())
		gitlab.Use(r.authenticator.PermissionMiddleware("admin"))
	}

	// Secrets scanning endpoints
	gitlab.POST("/projects/:id/scan-secrets", secretsHandler.ScanSecrets)
	gitlab.POST("/projects/:id/deep-scan-secrets", secretsHandler.DeepScanSecrets)
	gitlab.POST("/projects/:id/sast-scan", secretsHandler.SASTScan)
	gitlab.GET("/secrets/patterns", secretsHandler.GetPatterns)

	r.logger.Info("GitLab security scanning routes configured: /scan-secrets, /deep-scan-secrets, /sast-scan")
}

// SetupGitLabDependenciesRoutes registers GitLab dependencies scanning routes
func (r *Router) SetupGitLabDependenciesRoutes(store storage.Store, vectorStore interface{}, llmBaseURL, llmAPIKey string) {
	if store == nil {
		r.logger.Warn("GitLab dependencies routes: Store is nil, skipping setup")
		return
	}

	// Check if vectorStore is a QdrantStore
	qdrantStore, ok := vectorStore.(*vector.QdrantStore)
	if !ok || qdrantStore == nil {
		r.logger.Warn("GitLab dependencies routes: VectorStore is not available, skipping setup")
		return
	}

	r.logger.Info("Setting up GitLab dependencies scanning routes")

	gitlab := r.engine.Group("/api/admin/gitlab")

	// Apply authentication
	if r.jwtManager != nil && r.db != nil {
		gitlab.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		gitlab.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		gitlab.Use(r.authenticator.AuthenticationMiddleware())
		gitlab.Use(r.authenticator.PermissionMiddleware("admin"))
	}

	dependenciesHandler := handlers.NewGitLabDependenciesHandler(store, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
	gitlab.POST("/projects/:id/check-dependencies", dependenciesHandler.CheckDependencies)
	gitlab.POST("/projects/:id/create-dependency-issue", dependenciesHandler.CreateDependencyIssue)
	gitlab.POST("/projects/:id/analyze-changelog", dependenciesHandler.AnalyzeChangelog)
	gitlab.POST("/projects/:id/analyze-changelogs", dependenciesHandler.AnalyzeDependenciesChangelogs)

	r.logger.Info("GitLab dependencies routes configured: POST /api/admin/gitlab/projects/:id/check-dependencies, analyze-changelog, analyze-changelogs")
}

// SetupGitLabQualityRoutes registers GitLab code quality analysis routes
func (r *Router) SetupGitLabQualityRoutes(store storage.Store, vectorStore interface{}, llmBaseURL, llmAPIKey string) {
	if store == nil {
		r.logger.Warn("GitLab quality routes: Store is nil, skipping setup")
		return
	}

	// Check if vectorStore is a QdrantStore
	qdrantStore, ok := vectorStore.(*vector.QdrantStore)
	if !ok || qdrantStore == nil {
		r.logger.Warn("GitLab quality routes: VectorStore is not available, skipping setup")
		return
	}

	r.logger.Info("Setting up GitLab code quality routes")

	gitlab := r.engine.Group("/api/admin/gitlab")

	// Apply authentication
	if r.jwtManager != nil && r.db != nil {
		gitlab.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		gitlab.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		gitlab.Use(r.authenticator.AuthenticationMiddleware())
		gitlab.Use(r.authenticator.PermissionMiddleware("admin"))
	}

	qualityHandler := handlers.NewGitLabQualityHandler(store, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
	gitlab.POST("/projects/:id/quality-score", qualityHandler.AnalyzeQuality)
	gitlab.POST("/projects/:id/detect-duplication", qualityHandler.DetectDuplication)

	r.logger.Info("GitLab quality routes configured: POST /api/admin/gitlab/projects/:id/quality-score, detect-duplication")
}

// SetupGitLabDeadCodeRoutes registers GitLab dead code detection routes
func (r *Router) SetupGitLabDeadCodeRoutes(store storage.Store, vectorStore interface{}, llmBaseURL, llmAPIKey string) {
	if store == nil {
		r.logger.Warn("GitLab dead code routes: Store is nil, skipping setup")
		return
	}

	// Check if vectorStore is a QdrantStore
	qdrantStore, ok := vectorStore.(*vector.QdrantStore)
	if !ok || qdrantStore == nil {
		r.logger.Warn("GitLab dead code routes: VectorStore is not available, skipping setup")
		return
	}

	r.logger.Info("Setting up GitLab dead code detection routes")

	gitlab := r.engine.Group("/api/admin/gitlab")

	// Apply authentication
	if r.jwtManager != nil && r.db != nil {
		gitlab.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		gitlab.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		gitlab.Use(r.authenticator.AuthenticationMiddleware())
		gitlab.Use(r.authenticator.PermissionMiddleware("admin"))
	}

	deadCodeHandler := handlers.NewGitLabDeadCodeHandler(store, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
	gitlab.POST("/projects/:id/dead-code", deadCodeHandler.DetectDeadCode)
	gitlab.POST("/projects/:id/detect-unreachable", deadCodeHandler.DetectUnreachable)
	gitlab.POST("/projects/:id/detect-commented-code", deadCodeHandler.DetectCommentedCode)
	gitlab.POST("/projects/:id/dead-code/create-issue", deadCodeHandler.CreateDeadCodeIssue)

	r.logger.Info("GitLab dead code routes configured: POST /api/admin/gitlab/projects/:id/dead-code, detect-unreachable, detect-commented-code, create-issue")
}

// SetupGitLabAutoDocRoutes registers GitLab auto-documentation routes
func (r *Router) SetupGitLabAutoDocRoutes(store storage.Store, vectorStore interface{}, llmBaseURL, llmAPIKey string) {
	if store == nil {
		r.logger.Warn("GitLab autodoc routes: Store is nil, skipping setup")
		return
	}

	// Check if vectorStore is a QdrantStore
	qdrantStore, ok := vectorStore.(*vector.QdrantStore)
	if !ok || qdrantStore == nil {
		r.logger.Warn("GitLab autodoc routes: VectorStore is not available, skipping setup")
		return
	}

	r.logger.Info("Setting up GitLab auto-documentation routes")

	gitlab := r.engine.Group("/api/admin/gitlab")

	// Apply authentication
	if r.jwtManager != nil && r.db != nil {
		gitlab.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		gitlab.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		gitlab.Use(r.authenticator.AuthenticationMiddleware())
		gitlab.Use(r.authenticator.PermissionMiddleware("admin"))
	}

	autoDocHandler := handlers.NewGitLabAutoDocHandler(store, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
	gitlab.POST("/projects/:id/scan-docs", autoDocHandler.ScanUndocumented)
	gitlab.POST("/projects/:id/generate-docs", autoDocHandler.GenerateDocs)
	gitlab.POST("/projects/:id/bulk-apply-docs", autoDocHandler.BulkApplyDocs)
	gitlab.POST("/projects/:id/create-docs-mr", autoDocHandler.CreateDocsMR)

	r.logger.Info("GitLab autodoc routes configured: POST /api/admin/gitlab/projects/:id/scan-docs, /generate-docs, /bulk-apply-docs, /create-docs-mr")
}

// SetupGitLabTestGenRoutes registers GitLab test generation routes
func (r *Router) SetupGitLabTestGenRoutes(store storage.Store, vectorStore interface{}, llmBaseURL, llmAPIKey string) {
	if store == nil {
		r.logger.Warn("GitLab testgen routes: Store is nil, skipping setup")
		return
	}

	// Check if vectorStore is a QdrantStore
	qdrantStore, ok := vectorStore.(*vector.QdrantStore)
	if !ok || qdrantStore == nil {
		r.logger.Warn("GitLab testgen routes: VectorStore is not available, skipping setup")
		return
	}

	r.logger.Info("Setting up GitLab test generation routes")

	gitlab := r.engine.Group("/api/admin/gitlab")

	// Apply authentication
	if r.jwtManager != nil && r.db != nil {
		gitlab.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		gitlab.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		gitlab.Use(r.authenticator.AuthenticationMiddleware())
		gitlab.Use(r.authenticator.PermissionMiddleware("admin"))
	}

	testGenHandler := handlers.NewGitLabTestGenHandler(store, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
	gitlab.POST("/projects/:id/scan-tests", testGenHandler.ScanTestable)
	gitlab.POST("/projects/:id/generate-tests", testGenHandler.GenerateTests)
	gitlab.POST("/projects/:id/download-tests", testGenHandler.DownloadTests)

	r.logger.Info("GitLab testgen routes configured: POST /api/admin/gitlab/projects/:id/scan-tests, /generate-tests, /download-tests")
}

// SetupGitLabArchitectureRoutes registers GitLab architecture diagram routes
func (r *Router) SetupGitLabArchitectureRoutes(store storage.Store, vectorStore interface{}, llmBaseURL, llmAPIKey string) {
	if store == nil {
		r.logger.Warn("GitLab architecture routes: Store is nil, skipping setup")
		return
	}

	// Check if vectorStore is a QdrantStore
	qdrantStore, ok := vectorStore.(*vector.QdrantStore)
	if !ok || qdrantStore == nil {
		r.logger.Warn("GitLab architecture routes: VectorStore is not available, skipping setup")
		return
	}

	r.logger.Info("Setting up GitLab architecture diagram routes")

	gitlab := r.engine.Group("/api/admin/gitlab")

	// Apply authentication
	if r.jwtManager != nil && r.db != nil {
		gitlab.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		gitlab.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		gitlab.Use(r.authenticator.AuthenticationMiddleware())
		gitlab.Use(r.authenticator.PermissionMiddleware("admin"))
	}

	archHandler := handlers.NewGitLabArchitectureHandler(store, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
	gitlab.POST("/projects/:id/scan-architecture", archHandler.ScanArchitecture)
	gitlab.POST("/projects/:id/generate-diagram", archHandler.GenerateDiagram)
	gitlab.GET("/projects/:id/architecture", archHandler.GetArchitecture)

	r.logger.Info("GitLab architecture routes configured: /scan-architecture, /generate-diagram, /architecture")
}

// SetupGitLabScheduleRoutes registers GitLab scheduled scan routes
func (r *Router) SetupGitLabScheduleRoutes(store storage.Store, vectorStore interface{}, llmBaseURL, llmAPIKey string) {
	if store == nil {
		r.logger.Warn("GitLab schedule routes: Store is nil, skipping setup")
		return
	}

	// Check if vectorStore is a QdrantStore
	qdrantStore, ok := vectorStore.(*vector.QdrantStore)
	if !ok || qdrantStore == nil {
		r.logger.Warn("GitLab schedule routes: VectorStore is not available, skipping setup")
		return
	}

	r.logger.Info("Setting up GitLab scheduled scan routes")

	gitlab := r.engine.Group("/api/admin/gitlab")

	// Apply authentication
	if r.jwtManager != nil && r.db != nil {
		gitlab.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		gitlab.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		gitlab.Use(r.authenticator.AuthenticationMiddleware())
		gitlab.Use(r.authenticator.PermissionMiddleware("admin"))
	}

	// Cast store to PostgresStore to get ScheduleStore interface
	scheduleStore, ok := store.(schedule.ScheduleStore)
	if !ok {
		r.logger.Warn("GitLab schedule routes: Store does not implement ScheduleStore, skipping setup")
		return
	}

	// Create scheduler
	sched := schedule.NewScheduler(store, scheduleStore, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
	
	// Start scheduler in background (uses background context, lives until process terminates)
	go func() {
		ctx := context.Background()
		if err := sched.Start(ctx); err != nil {
			r.logger.WithError(err).Error("Failed to start dependency scan scheduler")
		}
	}()

	scheduleHandler := handlers.NewGitLabScheduleHandler(sched, scheduleStore, r.logger)
	
	// Schedule management
	gitlab.POST("/schedules", scheduleHandler.CreateSchedule)
	gitlab.GET("/schedules", scheduleHandler.ListSchedules)
	gitlab.GET("/schedules/status", scheduleHandler.GetSchedulerStatus)
	gitlab.GET("/schedules/:id", scheduleHandler.GetSchedule)
	gitlab.PUT("/schedules/:id", scheduleHandler.UpdateSchedule)
	gitlab.DELETE("/schedules/:id", scheduleHandler.DeleteSchedule)
	gitlab.POST("/schedules/:id/trigger", scheduleHandler.TriggerSchedule)
	gitlab.GET("/schedules/:id/history", scheduleHandler.GetScheduleHistory)

	r.logger.Info("GitLab schedule routes configured: /api/admin/gitlab/schedules/*")
}

// SetupGitLabAnalyticsRoutes registers GitLab analytics routes
func (r *Router) SetupGitLabAnalyticsRoutes(store storage.Store, analyticsStore interface{}) {
	if store == nil {
		r.logger.Warn("GitLab analytics routes: Store is nil, skipping setup")
		return
	}

	r.logger.Info("Setting up GitLab analytics routes")

	gitlab := r.engine.Group("/api/admin/gitlab")

	// Apply authentication
	if r.jwtManager != nil && r.db != nil {
		gitlab.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		gitlab.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		gitlab.Use(r.authenticator.AuthenticationMiddleware())
		gitlab.Use(r.authenticator.PermissionMiddleware("admin"))
	}

	// Check if analyticsStore implements the analytics.Store interface
	analyticsSt, ok := analyticsStore.(analytics.Store)
	if !ok {
		r.logger.Warn("GitLab analytics routes: Analytics store type assertion failed, skipping")
		return
	}

	analyticsHandler := handlers.NewGitLabAnalyticsHandler(store, analyticsSt, r.logger)
	gitlab.GET("/analytics/dashboard", analyticsHandler.GetDashboard)
	gitlab.GET("/analytics/models", analyticsHandler.GetModelComparison)
	gitlab.GET("/analytics/security", analyticsHandler.GetSecurityOverview)
	gitlab.GET("/analytics/dependencies", analyticsHandler.GetDependencyHealth)
	gitlab.GET("/analytics/export", analyticsHandler.ExportReport)
	gitlab.GET("/projects/:id/analytics", analyticsHandler.GetProjectAnalytics)

	r.logger.Info("GitLab analytics routes configured: /api/admin/gitlab/analytics/*")
}

