// Package router provides GitLab integration routes
package router

import (
	"aigateway/internal/api/handlers"
	"aigateway/internal/api/middleware"
	authMiddleware "aigateway/internal/auth/middleware"
	"aigateway/internal/gitlab/storage"
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
	gitlab.GET("/secrets/patterns", secretsHandler.GetPatterns)

	r.logger.Info("GitLab secrets scanning routes configured: POST /api/admin/gitlab/projects/:id/scan-secrets, POST /api/admin/gitlab/projects/:id/deep-scan-secrets")
}

