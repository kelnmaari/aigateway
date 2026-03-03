// Package router provides HTTP routing setup for AIGateway
package router

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/pprof"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"aigateway/internal/api/handlers"
	handlersUI "aigateway/internal/api/handlers/ui"
	"aigateway/internal/api/middleware"
	"aigateway/internal/auth/apikey"
	"aigateway/internal/auth/jwt"
	ldapauth "aigateway/internal/auth/ldap"
	authMiddleware "aigateway/internal/auth/middleware"
	oidcauth "aigateway/internal/auth/oidc"
	"aigateway/internal/auth/ratelimit"
	authService "aigateway/internal/auth/service"

	// "aigateway/internal/client/ollama" // Removed: v3.0.5+
	"aigateway/internal/cache/redis"
	"aigateway/internal/config"
	"aigateway/internal/extractors"
	"aigateway/internal/filestorage"
	filestorageBackend "aigateway/internal/filestorage/storage"
	"aigateway/internal/gitlab/dependencies/schedule"
	gitlabIndexer "aigateway/internal/gitlab/indexer"
	gitlabJobs "aigateway/internal/gitlab/jobs"
	gitlabProcessor "aigateway/internal/gitlab/processor"
	gitlabRAG "aigateway/internal/gitlab/rag"
	gitlabStorage "aigateway/internal/gitlab/storage"
	gitlabWebhook "aigateway/internal/gitlab/webhook"
	gitlabWorker "aigateway/internal/gitlab/worker"
	"aigateway/internal/health"
	"aigateway/internal/huggingface"
	"aigateway/internal/inference"
	internalLogger "aigateway/internal/logger"
	"aigateway/internal/metrics"
	"aigateway/internal/models"
	"aigateway/internal/observability"
	"aigateway/internal/providers"
	ragorchestrator "aigateway/internal/rag/orchestrator"
	"aigateway/internal/rag/vector"
	"aigateway/internal/request"
	agentService "aigateway/internal/services/agent"
	"aigateway/internal/services/audit"
	"aigateway/internal/services/quota"
	ragservice "aigateway/internal/services/rag"
	"aigateway/internal/services/rbac"
	"aigateway/internal/settings"
	"aigateway/internal/storage"
	"aigateway/internal/tools"
	"aigateway/internal/web"
	"aigateway/internal/web/framework"
	"aigateway/internal/web/templates"
	"aigateway/internal/websocket"

	"go.opentelemetry.io/otel/trace"
)

// apiKeyDatabaseAdapter adapts storage.Database to redis.APIKeyProvider interface (v3.0.6+)
type apiKeyDatabaseAdapter struct {
	db storage.Database
}

// GetAPIKey implements redis.APIKeyProvider by wrapping storage.Database
func (a *apiKeyDatabaseAdapter) GetAPIKey(ctx context.Context, keyID string) (interface{}, error) {
	apiKey, err := a.db.GetAPIKey(ctx, keyID)
	if err != nil {
		return nil, err
	}
	return apiKey, nil
}

// Router представляет HTTP роутер приложения с опциональным API Key Management
type Router struct {
	config        *config.Config
	logger        *logrus.Logger
	httpLogger    *logrus.Logger // Отдельный логгер для детальных HTTP логов (в http.log)
	metricsLogger *logrus.Logger // Отдельный логгер для GPU/performance метрик (в metrics.log)
	engine        *gin.Engine
	version       string       // Версия сервера
	tracer        trace.Tracer // OpenTelemetry tracer (v1.6.0+)

	// API Key Management компоненты (опциональные)
	storage       storage.APIKeyStorage
	keyManager    *apikey.Manager
	authenticator *authMiddleware.APIKeyAuthenticator
	rateLimiter   *ratelimit.Limiter

	// User Authentication (Version 1.3.0+)
	db               storage.Database              // Database для user/tenant management
	jwtManager       *jwt.Manager                  // JWT manager для токенов
	authService      *authService.AuthService      // Auth service
	bootstrapService *authService.BootstrapService // Bootstrap service для первичной настройки

	// Redis (v3.0.6+: Distributed caching & session management)
	redisManager *redis.Manager // Redis manager для всех сервисов

	// Handlers
	systemHandler             *handlers.SystemHandler             // System endpoints (bootstrap, init-status)
	authHandler               *handlers.AuthHandler               // Auth endpoints (login, register, etc)
	deviceHandler             *handlers.DeviceHandler             // Device management (Version 2.4.0+)
	userHandler               *handlers.UserHandler               // User management
	tenantHandler             *handlers.TenantHandler             // Tenant management
	conversationHandler       *handlers.ConversationHandler       // Conversation management (Version 1.3.0)
	conversationExportHandler *handlers.ConversationExportHandler // Conversation export/import (Version 1.12.3+)

	// Health probes (v3.0.8+: Kubernetes support)
	healthChecker    *health.HealthChecker
	livenessProbe    *health.LivenessProbe
	readinessProbe   *health.ReadinessProbe
	adminUserHandler *handlers.AdminUserHandler // Admin User Management (Version 1.3.0)
	usageHandler     *handlers.UsageHandler     // Usage Statistics (Version 1.3.0)
	// Removed: modelsHandler, chatHandler, embeddingsHandler, completionsHandler (Ollama-based, v3.0.5+)
	adminHandler          *handlers.AdminHandler
	adminFilesHandler     *handlers.AdminFilesHandler     // Admin files management (v1.10.0)
	statsHandler          *handlers.StatsHandler          // Handler для TUI статистики
	logsHandler           *handlers.LogsHandler           // Handler для логов
	metricsHistoryHandler *handlers.MetricsHistoryHandler // Handler для historical metrics
	requestsHandler       *handlers.RequestsHandler       // Handler для request monitoring (TUI-04)
	mcpHandler            *handlers.MCPHandler            // Handler для MCP servers catalog (v1.4.5)
	changelogHandler      *handlers.ChangelogHandler      // Handler для changelog (v1.4.11)
	backupHandler         *handlers.BackupHandler         // Handler для backup/restore (v1.5.14)
	performanceHandler    *handlers.PerformanceHandler    // Handler для performance monitoring (v1.6.2)
	fileHandler           *handlers.FileHandler           // Handler для file operations (v1.10.0)
	oidcHandler           *handlers.OIDCHandler           // Handler для OIDC/Keycloak SSO (v1.11.1)
	ldapHandler           *handlers.LDAPHandler           // Handler для LDAP/AD authentication (v1.11.3)
	auditHandler          *handlers.AuditHandler          // Handler для audit logging (v1.11.4)
	rbacHandler           *handlers.RBACHandler           // Handler для RBAC management (v1.11.5)
	quotaHandler          *handlers.QuotaHandler          // Handler для quota management (v1.11.7)
	invitationHandler     *handlers.InvitationHandler     // Handler для invitation system (AUTH-03, v2.2.0)
	agentHandler          *handlers.AgentHandler          // Handler для agent API (v2.5.0+, v3.0.6+)
	ragDataSourcesHandler *handlers.RAGDataSourcesHandler // Handler для RAG data sources (v1.13.1)
	ragStatsHandler       *handlers.RAGStatsHandler       // Handler для RAG statistics (v3.2.0)
	registryHandler       *handlers.RegistryHandler       // Handler для model registry (REGISTRY-01, v2.3.0)
	dashboardHandler      *handlers.DashboardHandler      // Handler для batch dashboard API (v3.1.0, AJAX-01)

	// Provider Management (Version 2.3.0+: REGISTRY-01)
	providerManager *providers.ProviderManager // Model providers manager

	// WebSocket components
	wsHub              *websocket.Hub
	wsHandler          *websocket.Handler
	eventBroadcaster   *websocket.EventBroadcaster
	metricsBroadcaster *websocket.MetricsBroadcaster

	// Metrics storage
	metricsStorage *metrics.MetricsStorage

	// Request tracking
	requestStorage *request.Storage

	// Performance monitoring (v1.6.2)
	performanceMonitor *observability.PerformanceMonitor
	leakDetector       *observability.LeakDetector

	// MoniGo performance dashboard (v1.9.3)
	monigoPort int // Порт на котором запущен MoniGo (0 если отключен)

	// GPU Monitoring (v1.9.3)
	gpuMonitor *metrics.GPUMonitor
	gpuHandler *handlers.GPUHandler

	// Audit Logging (v1.11.4)
	auditLogger          *audit.AuditLogger
	auditRetentionPolicy *audit.RetentionPolicy

	// RBAC (Version 1.11.5+: Custom Roles & Permissions)
	rbacService    *rbac.Service
	rbacMiddleware *middleware.RBACMiddleware

	// Quotas (Version 1.11.7+: Usage Quotas System)
	quotaService    *quota.Service
	quotaMiddleware *middleware.QuotaMiddleware

	// Metrics (Version 1.11.6+: Prometheus Metrics Export)
	metricsCollector *metrics.MetricsCollector

	// RAG System (Version 1.13.0+: RAG System)
	ragDataSourceService *ragservice.DataSourceService

	// RAG Orchestrator
	ragOrchestrator *ragorchestrator.RAGOrchestrator

	// Vector Store for RAG (v3.2.0+)
	vectorStore vector.VectorStore

	// HTMX UI Handlers (Version 2.6.0+: HTMX-01)
	templateRenderer   *templates.Renderer
	registryUIHandler  *handlersUI.RegistryUIHandler
	apiKeysUIHandler   *handlersUI.APIKeysUIHandler
	tenantsUIHandler   *handlersUI.TenantsUIHandler
	monitorUIHandler   *handlersUI.MonitorUIHandler   // Monitor UI (GPU, Audit, Usage) - HTMX-02
	usersUIHandler     *handlersUI.UsersUIHandler     // Users UI (HTMX-03)
	rbacUIHandler      *handlersUI.RBACUIHandler      // RBAC UI (HTMX-03)
	dashboardUIHandler *handlersUI.DashboardUIHandler // Dashboard UI (HTMX-03)
	settingsUIHandler  *handlersUI.SettingsHandler    // Settings UI (v3.0.9: Config in DB)

	// Hugging Face Integration (Version 3.0.0+: HF-01)
	hfClient     *huggingface.Client
	hfDownloader *huggingface.Downloader          // Model downloader
	hfUIHandler  *handlersUI.HuggingFaceUIHandler // Hugging Face Model Browser

	// Inference v4 (multi-provider)
	inferenceSvc          *inference.Service
	inferenceMgr          *inference.Manager
	inferenceRouter       *inference.Router
	inferenceHandler      *handlers.InferenceHandler
	inferenceProxyHandler *handlers.InferenceProxyHandler
	chatToolsHandler      *handlers.ChatToolsHandler // Chat with tools support (v4.0.3+)
	externalProxyHandler  *handlers.ExternalProxyHandler // External provider proxy (v4.11.0+)
	inferenceModelStore   *inference.ModelStore

	// Agent Service (v2.5.0+)
	agentService *agentService.AgentService // Conversational agent with tools

	// UI Framework (v3.1.0: Optimized JS+CSS bundling)
	frameworkHandler *framework.Handler // Framework asset handler

	// GitLab Integration (v3.1.0+)
	gitlabHandler             *handlers.GitLabAdminHandler        // GitLab admin handler
	gitlabWebhookHandler      *handlers.GitLabWebhookHandler      // GitLab webhook handler
	gitlabWorkerPool          *gitlabWorker.Pool                  // GitLab worker pool for MR analysis
	gitlabIndexerHandler      *handlers.GitLabIndexerHandler      // GitLab indexer handler (admin)
	gitlabUserIndexerHandler  *handlers.GitLabUserIndexerHandler  // GitLab indexer handler (user-level)
	gitlabIndexer             *gitlabIndexer.Indexer              // GitLab repository indexer
	gitlabDependenciesHandler *handlers.GitLabDependenciesHandler // GitLab dependencies scanner handler
	gitlabScheduleHandler     *handlers.GitLabScheduleHandler     // GitLab scheduled scans handler
	gitlabScheduler           *schedule.Scheduler                 // GitLab dependency scan scheduler
	gitlabStore               gitlabStorage.Store                 // GitLab storage (v4.1.0+)
	gitlabAPIKey              string                              // Internal API key for GitLab LLM calls (v4.1.0+)
	gitlabJobService          *gitlabJobs.Service                 // GitLab background jobs service (v4.8.9+)
}

// NewOptions содержит опции для создания роутера
type NewOptions struct {
	Config               *config.Config
	Logger               *logrus.Logger
	HTTPLogger           *logrus.Logger // Опциональный логгер для HTTP запросов (отдельный файл)
	MetricsLogger        *logrus.Logger // Опциональный логгер для GPU/performance метрик (отдельный файл)
	Version              string
	Database             storage.Database                  // Опциональная база данных для user auth
	JWTManager           *jwt.Manager                      // Опциональный JWT manager
	TracerProvider       *observability.TracerProvider     // Опциональный OpenTelemetry tracer (v1.6.0+)
	PerformanceMonitor   *observability.PerformanceMonitor // Опциональный performance monitor (v1.6.2+)
	LeakDetector         *observability.LeakDetector       // Опциональный leak detector (v1.6.2+)
	MonigoPort           int                               // Порт MoniGo dashboard (0 если отключен) (v1.9.3+)
	GPUMonitor           *metrics.GPUMonitor               // Опциональный GPU monitor (v1.9.3+)
	RAGDataSourceService *ragservice.DataSourceService     // Опциональный RAG Data Source Service (v1.13.1+)
	RAGOrchestrator      *ragorchestrator.RAGOrchestrator  // Опциональный RAG Orchestrator (v1.13.1+)
	VectorStore          vector.VectorStore                // Опциональный Vector Store для RAG (v3.2.0+)
	InferenceRouter      *inference.Router                 // Опциональный Inference Router для Docker-based providers (v3.3.0+)
}

// New создает новый экземпляр роутера с опциональным API Key Management
func New(cfg *config.Config, logger *logrus.Logger, version string) (*Router, error) {
	return NewWithOptions(NewOptions{
		Config:  cfg,
		Logger:  logger,
		Version: version,
	})
}

// NewWithOptions создает роутер с расширенными опциями (v3.0.5+: Ollama removed)
func NewWithOptions(opts NewOptions) (*Router, error) {
	r := &Router{
		config:               opts.Config,
		logger:               opts.Logger,
		httpLogger:           opts.HTTPLogger,
		metricsLogger:        opts.MetricsLogger,
		version:              opts.Version,
		db:                   opts.Database,
		jwtManager:           opts.JWTManager,
		performanceMonitor:   opts.PerformanceMonitor,
		leakDetector:         opts.LeakDetector,
		monigoPort:           opts.MonigoPort,
		gpuMonitor:           opts.GPUMonitor,
		ragDataSourceService: opts.RAGDataSourceService,
		ragOrchestrator:      opts.RAGOrchestrator,
		vectorStore:          opts.VectorStore,
	}

	// Setup tracer if provided
	if opts.TracerProvider != nil {
		r.tracer = opts.TracerProvider.Tracer()
		opts.Logger.Info("OpenTelemetry tracer initialized")
	}

	// Setup Redis (v3.0.6+: Distributed caching & session management)
	if err := r.setupRedis(opts.Config, opts.Logger); err != nil {
		opts.Logger.WithError(err).Warn("Redis initialization failed, will use in-memory fallback")
	}

	// Setup Auth Service если есть database и JWT manager
	if r.db != nil && r.jwtManager != nil {
		r.authService = authService.NewAuthService(r.db, r.jwtManager, opts.Config, opts.Logger)
		r.bootstrapService = authService.NewBootstrapService(r.db, opts.Config.Auth.AdminKey, opts.Logger)
		opts.Logger.Info("User Authentication and Bootstrap services initialized")
	}

	// Если включена аутентификация, настраиваем API Key Management
	if opts.Config.Auth.Enabled {
		if err := r.setupAPIKeyManagement(opts.Config, opts.Logger); err != nil {
			return nil, fmt.Errorf("failed to setup API Key Management: %w", err)
		}
	}

	// Setup Model Registry (Version 2.3.0+: REGISTRY-01)
	if r.db != nil && opts.Config.ModelRegistry.Enabled {
		if err := r.setupModelRegistry(opts.Config, opts.Logger); err != nil {
			return nil, fmt.Errorf("failed to setup Model Registry: %w", err)
		}
		opts.Logger.Info("Model Registry initialized successfully")
	}

	// Инициализация handlers (v3.0.5+: removed ollamaClient)
	r.setupHandlers(opts.Config, opts.Logger)

	r.setupEngine()
	r.setupRoutes() // includes setupInferenceRoutes()

	return r, nil
}

// Initialize инициализирует роутер и все компоненты
func (r *Router) Initialize() error {
	r.logger.Info("Initializing router")

	// Инициализируем API Key Manager если включен
	if r.keyManager != nil {
		ctx := context.Background()
		if err := r.keyManager.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize API Key Manager: %w", err)
		}
	}

	// Auto-start saved models with auto_start=true
	if r.inferenceModelStore != nil && r.inferenceRouter != nil {
		go r.autoStartSavedModels()
	}

	r.logger.Info("Router initialized successfully")
	return nil
}

// autoStartSavedModels loads models marked with auto_start=true
func (r *Router) autoStartSavedModels() {
	autoStart := r.inferenceModelStore.ListAutoStart()
	if len(autoStart) == 0 {
		return
	}

	r.logger.WithField("count", len(autoStart)).Info("Auto-starting saved models")
	ctx := context.Background()

	for _, saved := range autoStart {
		spec := saved.ToSpec()
		r.logger.WithField("alias", spec.Alias).Info("Auto-starting model")

		_, err := r.inferenceRouter.EnsureBySpec(ctx, spec)
		if err != nil {
			r.logger.WithError(err).WithField("alias", spec.Alias).Error("Failed to auto-start model")
		}
	}
}

// Close закрывает роутер и освобождает ресурсы
func (r *Router) Close() error {
	r.logger.Info("Closing router")

	// Останавливаем WebSocket и Metrics компоненты
	ctx := context.Background()
	if err := r.Shutdown(ctx); err != nil {
		r.logger.WithError(err).Error("Error shutting down WebSocket/Metrics")
	}

	if r.rateLimiter != nil {
		r.rateLimiter.Stop()
	}

	if r.keyManager != nil {
		r.keyManager.Close()
	}

	// Shutdown GitLab indexer (v4.1.0+: invalidate in-progress indexations)
	if r.gitlabIndexer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := r.gitlabIndexer.Shutdown(shutdownCtx); err != nil {
			r.logger.WithError(err).Error("Failed to shutdown GitLab indexer")
		}
		cancel()
	}

	// Close Redis connections (v3.0.6+)
	if r.redisManager != nil {
		if err := r.redisManager.Close(); err != nil {
			r.logger.WithError(err).Error("Failed to close Redis connections")
		}
	}

	return nil
}

// Engine возвращает Gin engine для использования в HTTP сервере
func (r *Router) Engine() *gin.Engine {
	return r.engine
}

// setupEngine настраивает Gin engine
func (r *Router) setupEngine() {
	// Настройка режима Gin
	if r.config.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r.engine = gin.New()

	// Базовое middleware
	r.setupMiddleware()
}

// setupMiddleware настраивает базовые middleware
func (r *Router) setupMiddleware() {
	// OpenTelemetry tracing middleware (должен быть первым для полной трассировки)
	if r.tracer != nil {
		r.engine.Use(middleware.TracingMiddleware(r.tracer))
		r.logger.Info("OpenTelemetry tracing middleware enabled")
	}

	// Prometheus metrics middleware (v1.11.6+)
	if r.config.Metrics.Enabled {
		r.engine.Use(middleware.PrometheusMiddleware())
		r.logger.Info("Prometheus metrics middleware enabled")
	}

	// Session middleware для OIDC authentication (Version 1.11.1+)
	// Используем cookie-based session store
	sessionSecret := []byte(r.config.Auth.JWT.Secret) // Используем JWT secret для session encryption
	if len(sessionSecret) < 32 {
		// Ensure session secret is at least 32 bytes for security
		sessionSecret = []byte("aigateway-session-secret-change-this-in-production!")
		r.logger.Warn("Using default session secret - please configure a secure JWT secret")
	}
	store := cookie.NewStore(sessionSecret)
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600,  // 1 hour
		HttpOnly: true,  // Protect against XSS
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})
	r.engine.Use(sessions.Sessions("aigateway_session", store))
	r.logger.Info("Session middleware enabled for OIDC authentication")

	// Slow request logging middleware (v1.6.2) - после tracing
	if r.config.Observability.Performance.Enabled {
		threshold, err := time.ParseDuration(r.config.Observability.Performance.SlowRequestThreshold)
		if err != nil {
			threshold = 5 * time.Second
		}
		r.engine.Use(middleware.SlowRequestLogger(r.logger, threshold))
		r.logger.WithField("threshold", threshold).Info("Slow request logging middleware enabled")
	}

	// Error handling middleware
	r.engine.Use(middleware.ErrorHandling(r.logger))

	// Panic recovery с error handling
	r.engine.Use(middleware.PanicRecovery(r.logger))

	// Prometheus metrics middleware (если включено)
	if r.config.Metrics.Enabled {
		r.engine.Use(middleware.PrometheusMetrics())
	}

	// Stats middleware для TUI (подсчет запросов)
	r.engine.Use(middleware.StatsMiddleware())

	// Metrics Collector middleware для historical metrics (Phase 12.1)
	if r.metricsStorage != nil {
		r.engine.Use(middleware.MetricsCollectorWithConfig(middleware.MetricsCollectorConfig{
			Storage:        r.metricsStorage,
			Logger:         r.logger,
			DetailedLogger: r.httpLogger, // HTTP метрики в отдельный файл
		}))
	}

	// Request Tracker middleware для monitoring (TUI-04)
	if r.requestStorage != nil && r.eventBroadcaster != nil {
		r.engine.Use(middleware.RequestTracker(r.requestStorage, r.eventBroadcaster, r.logger))
	}

	// Structured logging middleware
	loggingConfig := middleware.LoggingConfig{
		Logger:         r.logger,
		DetailedLogger: r.httpLogger,                                                                                            // Детальные HTTP логи в отдельный файл
		SkipPaths:      []string{"/health", "/healthz", "/ready", "/api/stats", "/api/config", r.config.Metrics.PrometheusPath}, // Пропускаем health checks, stats, config и metrics
	}
	r.engine.Use(middleware.RequestLogging(loggingConfig))

	// CORS middleware
	r.engine.Use(middleware.CORS(r.config))

	// CSRF Protection middleware (v2.4.9+) - Go 1.25 CrossOriginProtection
	// Защищает от Cross-Site Request Forgery атак
	if r.config.Server.CORS.Enabled {
		csrfConfig := middleware.CrossOriginProtectionConfig{
			Enabled:             true,
			TrustedOrigins:      r.config.Server.CORS.AllowedOrigins,
			RequireOriginHeader: false, // Не требуем Origin для non-browser clients (API)
		}
		r.engine.Use(middleware.CrossOriginProtection(csrfConfig, r.logger))
		r.logger.WithField("trusted_origins", csrfConfig.TrustedOrigins).Info("CSRF protection middleware enabled (Go 1.25)")
	}

	// API Key middleware применяется только к защищенным группам
}

// setupRoutes настраивает все маршруты приложения
func (r *Router) setupRoutes() {
	r.setupHealthRoutes() // Health check endpoint (for desktop client)
	r.setupMetricsRoutes()
	r.setupMonigoRoutes()         // MoniGo Performance Dashboard (v1.9.3+)
	r.setupStatsRoutes()          // Для TUI
	r.setupConfigRoutes()         // Для TUI Configuration Viewer
	r.setupRAGRoutes()            // RAG System (v1.13.1+)
	r.setupDashboardRoutes()      // Dashboard batch API (v3.1.0, AJAX-01)
	r.setupFrameworkRoutes()      // UI Framework assets (v3.1.0)
	r.setupMetricsHistoryRoutes() // Для historical metrics (Phase 12.1)
	r.setupRequestsRoutes()       // Для request monitoring (TUI-04)
	r.setupWebSocketRoutes()      // Для real-time updates (Phase 12.2)
	r.setupSystemRoutes()         // System endpoints (bootstrap) (Version 1.3.0+)
	r.setupAuthRoutes()           // User authentication (Version 1.3.0+)
	r.setupConversationsRoutes()  // Conversations API (Version 1.3.0)
	r.setupUsageRoutes()          // Usage Statistics API (Version 1.3.0)
	r.setupWebUIRoutes()          // WebUI static files (Version 1.3.0+)
	r.setupOpenAIRoutes()
	r.setupAdminRoutes()
	r.setupInvitationsRoutes() // Invitation System (AUTH-03, v2.2.0)
	r.setupMCPRoutes()         // MCP Servers Catalog (v1.4.5)
	r.setupGPURoutes()         // GPU Monitoring (v1.9.3)
	r.setupFileRoutes()        // File Storage & Processing (v1.10.0)
	r.setupUIRoutes()          // HTMX UI Routes (v2.6.0)
	r.setupGitLabRoutes()      // GitLab Integration routes (v3.1.0)
	r.setupInferenceRoutes()   // Inference v4 system routes
}

// setupInferenceRoutes registers minimal inference v4 endpoints (system).
func (r *Router) setupInferenceRoutes() {
	if r.inferenceHandler == nil || r.engine == nil {
		return
	}
	group := r.engine.Group("/api/system/inference")

	// Use JWT authentication for admin UI access (like other admin routes)
	if r.jwtManager != nil && r.db != nil {
		r.logger.Info("Inference routes: Using JWT authentication with admin role check")
		group.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		group.Use(middleware.RequireAdmin(r.db, r.logger))
	}
	{
		group.POST("/load", r.inferenceHandler.PostLoad)
		group.POST("/prepare", r.inferenceHandler.PostPrepare)
		group.POST("/stop", r.inferenceHandler.PostStop)
		group.POST("/evict", r.inferenceHandler.PostEvict)
		group.POST("/pin", r.inferenceHandler.PostPin)
		group.POST("/unpin", r.inferenceHandler.PostUnpin)
		group.POST("/delete-artifacts", r.inferenceHandler.PostDeleteArtifacts)
		group.POST("/evict-cache", r.inferenceHandler.PostEvictCache)
		group.GET("/cache", r.inferenceHandler.GetCache)
		group.POST("/cache/clear", r.inferenceHandler.PostClearCache)
		group.GET("/health", r.inferenceHandler.GetHealth)
		group.GET("/models", r.inferenceHandler.GetModels)
		group.GET("/logs", r.inferenceHandler.GetLogs)
		group.GET("/metrics", r.inferenceHandler.GetMetrics)
		group.GET("/trt-engines", r.inferenceHandler.ListTRTEngines)
		group.POST("/convert-trt", r.inferenceHandler.ConvertTRT)
		group.POST("/delete-trt-engine", r.inferenceHandler.DeleteTRTEngine)
		// Saved models (persist config between restarts)
		group.GET("/saved", r.inferenceHandler.GetSavedModels)
		group.POST("/save", r.inferenceHandler.PostSaveModel)
		group.POST("/delete-saved", r.inferenceHandler.PostDeleteSaved)
		group.POST("/auto-start", r.inferenceHandler.PostSetAutoStart)
		group.POST("/update-saved", r.inferenceHandler.PostUpdateSaved)
		group.POST("/create-saved", r.inferenceHandler.PostCreateSaved)
		// Docker image management
		group.GET("/docker-images", r.inferenceHandler.GetDockerImages)
		group.POST("/docker-images/pull", r.inferenceHandler.PostPullDockerImage)
		// Repository download (v3.3.x+) - download all model files locally
		group.POST("/download-repo", r.inferenceHandler.PostDownloadRepository)
		group.GET("/repo-downloads", r.inferenceHandler.GetRepoDownloads)
		group.POST("/repo-downloads/status", r.inferenceHandler.GetRepoDownloadStatus) // POST because model_id contains /
		group.POST("/repo-downloads/cancel", r.inferenceHandler.CancelRepoDownload)
		group.POST("/repo-downloads/remove", r.inferenceHandler.RemoveRepoDownload)
		// Refresh saved model (re-download missing/corrupted files)
		group.POST("/refresh-saved", r.inferenceHandler.PostRefreshSaved)
	}
	r.logger.Info("Inference v4 routes configured")

	// OpenAI-compatible proxy routes for inference v4 providers
	r.setupInferenceProxyRoutes()

	// HuggingFace JSON API for model browser (v3.3.0+)
	r.setupHuggingFaceAPIRoutes()
}

// setupHuggingFaceAPIRoutes registers JSON API endpoints for HuggingFace model browser.
func (r *Router) setupHuggingFaceAPIRoutes() {
	if r.hfClient == nil || r.engine == nil {
		return
	}

	hfGroup := r.engine.Group("/api/huggingface")
	if r.jwtManager != nil && r.db != nil {
		hfGroup.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
	}
	{
		// Search models
		hfGroup.GET("/search", func(c *gin.Context) {
			query := c.Query("q")
			author := c.Query("author")
			tag := c.Query("tag")
			limitStr := c.DefaultQuery("limit", "20")
			limit := 20
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
				limit = l
			}

			filters := huggingface.ModelFilters{
				Search: query,
				Author: author,
				Limit:  limit,
				Sort:   "downloads",
			}
			if tag != "" {
				filters.Tags = []string{tag}
			}

			models, err := r.hfClient.SearchModels(c.Request.Context(), filters)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"models": models})
		})

		// Get model info with files
		hfGroup.GET("/models/:repo/*subpath", func(c *gin.Context) {
			repo := c.Param("repo")
			subpath := c.Param("subpath")
			if subpath != "" && subpath != "/" {
				repo = repo + subpath
			}

			info, err := r.hfClient.GetModelInfo(c.Request.Context(), repo)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, info)
		})

		// Get popular GGUF models
		hfGroup.GET("/popular", func(c *gin.Context) {
			filters := huggingface.ModelFilters{
				Tags:  []string{"gguf"},
				Sort:  "downloads",
				Limit: 50,
			}

			models, err := r.hfClient.SearchModels(c.Request.Context(), filters)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"models": models})
		})

		// Start download
		hfGroup.POST("/download", func(c *gin.Context) {
			if r.hfDownloader == nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Downloader not configured"})
				return
			}

			var req struct {
				ModelID   string `json:"model_id"`
				Filename  string `json:"filename"`
				TotalSize int64  `json:"total_size"`
				SHA256    string `json:"sha256"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			download, err := r.hfDownloader.StartDownload(req.ModelID, req.Filename, req.TotalSize, req.SHA256)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"download_id": download.ID, "message": "Download started"})
		})
	}
	r.logger.Info("HuggingFace JSON API routes configured: /api/huggingface/*")
}

// setupInferenceProxyRoutes registers OpenAI-compatible proxy routes for inference v4.
func (r *Router) setupInferenceProxyRoutes() {
	if r.inferenceProxyHandler == nil || r.engine == nil {
		return
	}

	// /v1/inference/* routes - proxy to running provider containers
	v1inf := r.engine.Group("/v1/inference")
	if r.db != nil {
		v1inf.Use(middleware.APIKeyDBAuth(r.config, r.db, r.logger))
	}
	{
		v1inf.POST("/chat/completions", r.inferenceProxyHandler.HandleChatCompletions)
		v1inf.POST("/completions", r.inferenceProxyHandler.HandleCompletions)
		v1inf.GET("/models", r.inferenceProxyHandler.HandleModels)
	}
	r.logger.Info("Inference v4 OpenAI proxy routes configured: /v1/inference/*")

	// /api/chat/completions - Chat with tools support (web search etc.)
	// Supports both inference and external provider models
	{
		apiChat := r.engine.Group("/api/chat")
		if r.jwtManager != nil && r.db != nil {
			apiChat.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		}
		apiChat.POST("/completions", r.unifiedChatCompletions())
		r.logger.Info("Chat with tools route configured: /api/chat/completions")
	}
}

// unifiedChatCompletions returns a handler that routes chat completions
// to inference (Docker) or external providers (Model Registry) based on model source.
func (r *Router) unifiedChatCompletions() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read body and preserve it for downstream handlers
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"type": "invalid_request_error", "message": "failed to read request body"},
			})
			return
		}

		// Parse model from request
		var peek struct {
			Model string `json:"model"`
		}
		if err := json.Unmarshal(bodyBytes, &peek); err != nil || peek.Model == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"type": "invalid_request_error", "message": "model is required"},
			})
			return
		}

		// Check inference first (Docker-based models)
		if r.inferenceRouter != nil {
			if _, running := r.inferenceRouter.GetModel(peek.Model); running {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				if r.chatToolsHandler != nil {
					r.chatToolsHandler.HandleChatWithTools(c)
				} else {
					r.inferenceProxyHandler.HandleChatCompletions(c)
				}
				return
			}
		}

		// Check external providers (Model Registry)
		if r.externalProxyHandler != nil && r.db != nil {
			if _, err := r.db.GetModelRegistryByModelID(c.Request.Context(), peek.Model); err == nil {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				r.externalProxyHandler.HandleChatCompletions(c)
				return
			}
		}

		// Model not found anywhere
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"type":    "model_not_found",
				"message": fmt.Sprintf("model '%s' not found in inference or model registry", peek.Model),
			},
		})
	}
}

// unifiedPassthrough returns a generic handler that routes any OpenAI-compatible request
// to inference (Docker) or external providers (Model Registry) based on model source.
// Supports both JSON and multipart/form-data requests (embeddings, rerank, audio, images, moderations).
func (r *Router) unifiedPassthrough(endpointPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"type": "invalid_request_error", "message": "failed to read request body"},
			})
			return
		}

		model := handlers.ExtractModelFromRequest(bodyBytes, c.GetHeader("Content-Type"))

		r.logger.WithFields(logrus.Fields{
			"path":  endpointPath,
			"model": model,
		}).Debug("unifiedPassthrough: incoming request")

		if model == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"type": "invalid_request_error", "message": "model is required"},
			})
			return
		}

		// Check inference first (Docker-based models)
		if r.inferenceRouter != nil {
			if _, running := r.inferenceRouter.GetModel(model); running {
				r.logger.WithFields(logrus.Fields{
					"path":  endpointPath,
					"model": model,
				}).Debug("unifiedPassthrough: routing to inference")
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				r.inferenceProxyHandler.HandlePassthrough(c, endpointPath)
				return
			}
		}

		// Check external providers (Model Registry)
		if r.externalProxyHandler != nil && r.db != nil {
			modelEntry, err := r.db.GetModelRegistryByModelID(c.Request.Context(), model)
			if err == nil {
				provider, err := r.db.GetModelProvider(c.Request.Context(), modelEntry.ProviderID)
				if err == nil && provider.Enabled {
					r.logger.WithFields(logrus.Fields{
						"path":     endpointPath,
						"model":    model,
						"provider": provider.Name,
					}).Debug("unifiedPassthrough: routing to external provider")
					c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
					r.externalProxyHandler.HandlePassthrough(c, provider, endpointPath)
					return
				}
			}
		}

		r.logger.WithFields(logrus.Fields{
			"path":  endpointPath,
			"model": model,
		}).Warn("unifiedPassthrough: model not found in inference or model registry")

		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"type":    "model_not_found",
				"message": fmt.Sprintf("model '%s' not found in inference or model registry", model),
			},
		})
	}
}

// handleGetModel returns a handler for GET /v1/models/:model.
// Returns model details from inference or Model Registry.
func (r *Router) handleGetModel() gin.HandlerFunc {
	return func(c *gin.Context) {
		modelID := c.Param("model")

		// Check inference first
		if r.inferenceRouter != nil {
			for _, m := range r.inferenceRouter.ListModels() {
				if m.Spec.Alias == modelID {
					c.JSON(http.StatusOK, gin.H{
						"id":       m.Spec.Alias,
						"object":   "model",
						"created":  m.LastUsed.Unix(),
						"owned_by": string(m.Spec.Provider),
					})
					return
				}
			}
		}

		// Check Model Registry
		if r.db != nil {
			entry, err := r.db.GetModelRegistryByModelID(c.Request.Context(), modelID)
			if err == nil {
				provider, err := r.db.GetModelProvider(c.Request.Context(), entry.ProviderID)
				if err == nil && provider.Enabled {
					c.JSON(http.StatusOK, gin.H{
						"id":       entry.ModelID,
						"object":   "model",
						"created":  entry.CreatedAt.Unix(),
						"owned_by": string(provider.ProviderType) + ":" + provider.Name,
					})
					return
				}
			}
		}

		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"type":    "model_not_found",
				"message": fmt.Sprintf("model '%s' not found", modelID),
			},
		})
	}
}

// setupHealthRoutes настраивает health check endpoint для desktop client
func (r *Router) setupHealthRoutes() {
	// Health endpoint without authentication (needed for API key verification)
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"version": r.version,
		})
	})
	r.logger.Info("Health check endpoint configured: GET /health")
}

// setupMetricsRoutes настраивает эндпоинты метрик
func (r *Router) setupMetricsRoutes() {
	if r.config.Metrics.Enabled {
		// Prometheus metrics endpoint
		r.engine.GET(r.config.Metrics.PrometheusPath, gin.WrapH(promhttp.Handler()))
		r.logger.WithField("path", r.config.Metrics.PrometheusPath).Info("Prometheus metrics endpoint enabled")
	}
}

// setupMonigoRoutes настраивает MoniGo Performance Dashboard (v1.9.3+)
// MoniGo работает на отдельном порту (9090)
// Reverse proxy только для API endpoints (для карточек метрик в admin panel)
func (r *Router) setupMonigoRoutes() {
	if r.monigoPort == 0 {
		r.logger.Info("MoniGo disabled (port = 0)")
		return
	}

	monigoURL := fmt.Sprintf("http://localhost:%d", r.monigoPort)

	// Reverse proxy ТОЛЬКО для API эндпоинтов (для карточек метрик)
	r.engine.GET("/admin/performance/monigo/api/v1/metrics", r.createMonigoAPIProxy(monigoURL))

	r.logger.WithFields(logrus.Fields{
		"monigo_port":   r.monigoPort,
		"monigo_url":    monigoURL,
		"api_proxy":     "/admin/performance/monigo/api/v1/metrics",
		"dashboard_url": monigoURL,
	}).Info("✅ MoniGo Performance Dashboard running on separate port")
}

// createMonigoAPIProxy создает простой reverse proxy для MoniGo API
func (r *Router) createMonigoAPIProxy(targetURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Проксируем на /monigo/api/v1/metrics
		proxyURL := targetURL + "/monigo/api/v1/metrics"

		client := &http.Client{Timeout: 30 * time.Second}

		resp, err := client.Get(proxyURL)
		if err != nil {
			r.logger.WithError(err).Error("Failed to proxy MoniGo API request")
			c.JSON(http.StatusBadGateway, gin.H{"error": "MoniGo unavailable"})
			return
		}
		defer resp.Body.Close()

		// Копируем заголовки ответа
		for key, values := range resp.Header {
			for _, value := range values {
				c.Writer.Header().Add(key, value)
			}
		}

		// Копируем статус и тело
		c.Status(resp.StatusCode)
		c.Writer.Write(mustReadAll(resp.Body))
	}
}

// mustReadAll reads all data from reader
func mustReadAll(r interface{ Read([]byte) (int, error) }) []byte {
	buf := make([]byte, 0, 512)
	for {
		if len(buf) == cap(buf) {
			newBuf := make([]byte, len(buf), 2*cap(buf)+1)
			copy(newBuf, buf)
			buf = newBuf
		}
		n, err := r.Read(buf[len(buf):cap(buf)])
		buf = buf[:len(buf)+n]
		if err != nil {
			break
		}
	}
	return buf
}

// setupGPURoutes настраивает эндпоинты GPU мониторинга (v1.9.3+)
func (r *Router) setupGPURoutes() {
	if r.gpuHandler == nil {
		r.logger.Info("GPU monitoring disabled - no NVIDIA GPUs detected")
		return
	}

	// Публичный endpoint для GPU метрик (требует аутентификации)
	api := r.engine.Group("/api/gpu")
	{
		api.GET("/metrics", r.gpuHandler.GetGPUMetrics)
		api.GET("/list", r.gpuHandler.GetGPUList) // v3.3.x: GPU list for model deployment
	}

	r.logger.Info("✅ GPU monitoring routes registered")
}

// setupFileRoutes настраивает эндпоинты для работы с файлами (v1.10.0)
func (r *Router) setupFileRoutes() {
	if r.fileHandler == nil {
		r.logger.Info("File storage disabled - handler not initialized")
		return
	}

	// Файловые операции требуют аутентификации
	api := r.engine.Group("/api/files")
	if r.jwtManager != nil {
		api.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
	}
	{
		api.POST("/upload", r.fileHandler.UploadFile)
		api.GET("", r.fileHandler.ListFiles)
		api.GET("/:id", r.fileHandler.GetFile)
		api.GET("/:id/download", r.fileHandler.DownloadFile)
		api.GET("/:id/text", r.fileHandler.GetFileText)
		api.DELETE("/:id", r.fileHandler.DeleteFile)
		api.POST("/search", r.fileHandler.SearchFiles)
	}
}

// setupUIRoutes настраивает HTMX UI routes (v2.6.0+: HTMX-01)
func (r *Router) setupUIRoutes() {
	// UI routes требуют JWT аутентификации
	ui := r.engine.Group("/api/ui")
	if r.jwtManager != nil {
		ui.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
	}

	// Model Registry UI (conditional)
	if r.registryUIHandler != nil {
		registry := ui.Group("/registry")
		{
			registry.GET("/models", r.registryUIHandler.GetModelsTable)
			registry.GET("/models/search", r.registryUIHandler.SearchModels)
			registry.GET("/models/new-form", r.registryUIHandler.GetNewModelForm)
			registry.GET("/models/:id/edit", r.registryUIHandler.GetEditModelForm)
			registry.GET("/providers/new-form", r.registryUIHandler.GetNewProviderForm)
		}
		r.logger.Info("✅ Model Registry UI routes registered")
	}

	// API Keys UI (conditional)
	if r.apiKeysUIHandler != nil {
		apikeys := ui.Group("/api-keys")
		{
			apikeys.GET("/personal", r.apiKeysUIHandler.GetAPIKeysList)
			apikeys.GET("/tenant/:id", r.apiKeysUIHandler.GetAPIKeysForTenant)
			apikeys.GET("/create-form", r.apiKeysUIHandler.GetCreateAPIKeyForm)
			apikeys.GET("/:id", r.apiKeysUIHandler.GetAPIKeyCard)
			apikeys.GET("/:id/edit", r.apiKeysUIHandler.GetEditAPIKeyForm)
		}
		r.logger.Info("✅ API Keys UI routes registered")
	}

	// Tenants UI (conditional)
	if r.tenantsUIHandler != nil {
		tenants := ui.Group("/tenants")
		{
			tenants.GET("", r.tenantsUIHandler.GetTenantsGrid)
			tenants.GET("/create-form", r.tenantsUIHandler.GetCreateTenantForm)
			tenants.GET("/:id", r.tenantsUIHandler.GetTenantDetails)
			tenants.GET("/:id/edit", r.tenantsUIHandler.GetEditTenantForm)
			tenants.GET("/:id/members", r.tenantsUIHandler.GetTenantMembers)
		}
		r.logger.Info("✅ Tenants UI routes registered")
	}

	// Monitor UI (HTMX-02: Live Updates & Real-time Features)
	if r.monitorUIHandler != nil {
		monitor := ui.Group("/monitor")
		{
			monitor.GET("/gpu-metrics", r.monitorUIHandler.RenderGPUMetrics)
			monitor.GET("/audit-logs", r.monitorUIHandler.RenderAuditLogRows)
			monitor.GET("/usage-stats", r.monitorUIHandler.RenderUsageStats)
		}
		r.logger.Info("✅ Monitor UI routes registered")
	}

	// Users UI (HTMX-03: Advanced Forms & Search)
	if r.usersUIHandler != nil {
		users := ui.Group("/users")
		{
			users.GET("", r.usersUIHandler.GetUsersList)
			users.GET("/create-form", r.usersUIHandler.GetCreateUserForm)
			users.GET("/:id/edit", r.usersUIHandler.GetEditUserForm)
			users.GET("/:id/roles", r.usersUIHandler.GetUserRolesForm)
		}
		r.logger.Info("✅ Users UI routes registered")
	}

	// RBAC UI (HTMX-03: Advanced Forms & Search)
	if r.rbacUIHandler != nil {
		rbac := ui.Group("/rbac")
		{
			// Roles
			rbac.GET("/roles", r.rbacUIHandler.GetRolesList)
			rbac.GET("/roles/create-form", r.rbacUIHandler.GetCreateRoleForm)
			rbac.GET("/roles/:id/edit", r.rbacUIHandler.GetEditRoleForm)
			rbac.GET("/roles/:id/permissions", r.rbacUIHandler.GetRolePermissionsForm)

			// Permissions
			rbac.GET("/permissions", r.rbacUIHandler.GetPermissionsList)
		}
		r.logger.Info("✅ RBAC UI routes registered")
	}

	// Dashboard UI (HTMX-03: Advanced Forms & Search)
	if r.dashboardUIHandler != nil {
		dashboard := ui.Group("/dashboard")
		{
			// Stat cards
			dashboard.GET("/stats/users", r.dashboardUIHandler.GetUsersStatCard)
			dashboard.GET("/stats/api-keys", r.dashboardUIHandler.GetAPIKeysStatCard)
			dashboard.GET("/stats/models", r.dashboardUIHandler.GetModelsStatCard)
			dashboard.GET("/stats/requests", r.dashboardUIHandler.GetRequestsStatCard)

			// System health
			dashboard.GET("/system-health", r.dashboardUIHandler.GetSystemHealthTable)
		}
		r.logger.Info("✅ Dashboard UI routes registered")
	}

	// Hugging Face Model Browser (v3.0.0+: HF-UI-01)
	if r.hfUIHandler != nil {
		hf := ui.Group("/huggingface")
		{
			// Search and browse
			hf.GET("/search", r.hfUIHandler.GetModelsSearch)
			hf.GET("/popular", r.hfUIHandler.GetPopularModels)

			// Model details (support both /models/ and /model/ paths)
			hf.GET("/models/*model_id", r.hfUIHandler.GetModelDetails)
			hf.GET("/model/*model_id", r.hfUIHandler.GetModelDetails)
			hf.GET("/gguf-files/*model_id", r.hfUIHandler.GetGGUFFilesList)

			// Downloads (HF-02: Download Manager)
			hf.POST("/download", r.hfUIHandler.PostDownloadModel)
			hf.GET("/downloads", r.hfUIHandler.GetDownloadsList)
			hf.GET("/downloads/:download_id/progress", r.hfUIHandler.GetDownloadProgress)
			hf.POST("/downloads/:download_id/pause", r.hfUIHandler.PostPauseDownload)
			hf.POST("/downloads/:download_id/cancel", r.hfUIHandler.PostCancelDownload)
			hf.POST("/downloads/clear-completed", r.hfUIHandler.PostClearCompleted)
		}
		r.logger.Info("✅ Hugging Face UI routes registered")
	}

	r.logger.Info("✅ HTMX UI routes setup completed")
}

// setupRAGRoutes настраивает RAG System routes (v1.13.1+)
func (r *Router) setupRAGRoutes() {
	if r.ragDataSourcesHandler == nil {
		r.logger.Info("RAG System disabled - handler not initialized")
		return
	}

	// RAG operations требуют аутентификации
	api := r.engine.Group("/api/rag")
	if r.jwtManager != nil {
		api.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
	}
	{
		// Data Sources Management
		sources := api.Group("/sources")
		{
			sources.POST("", r.ragDataSourcesHandler.CreateDataSource)
			sources.GET("", r.ragDataSourcesHandler.ListDataSources)
			sources.GET("/:id", r.ragDataSourcesHandler.GetDataSource)
			sources.PUT("/:id", r.ragDataSourcesHandler.UpdateDataSource)
			sources.DELETE("/:id", r.ragDataSourcesHandler.DeleteDataSource)
			sources.POST("/test-connection", r.ragDataSourcesHandler.TestConnection)
			sources.POST("/:id/sync", r.ragDataSourcesHandler.SyncSource)
		}
	}

	r.logger.Info("RAG System routes configured successfully")
}

// setupStatsRoutes настраивает эндпоинты статистики для TUI
func (r *Router) setupStatsRoutes() {
	// Эндпоинт для TUI (без аутентификации, только для локального использования)
	if r.statsHandler != nil {
		r.engine.GET("/api/stats", r.statsHandler.GetStats)
		r.logger.Info("✅ Stats endpoint registered: /api/stats")
	}
}

// setupConfigRoutes настраивает эндпоинты конфигурации для TUI
func (r *Router) setupConfigRoutes() {
}

// setupMetricsHistoryRoutes настраивает эндпоинты для historical metrics (Phase 12.1)
func (r *Router) setupMetricsHistoryRoutes() {
	metricsAPI := r.engine.Group("/api/metrics")
	{
		metricsAPI.GET("/history", r.metricsHistoryHandler.GetHistory)
		metricsAPI.GET("/stats", r.metricsHistoryHandler.GetStats)
		metricsAPI.GET("/stats/all", r.metricsHistoryHandler.GetAllStats)
		metricsAPI.GET("/buffers", r.metricsHistoryHandler.GetBufferInfo)
		metricsAPI.GET("/recent", r.metricsHistoryHandler.GetRecentData)
		metricsAPI.GET("/types", r.metricsHistoryHandler.GetMetricTypes)
		metricsAPI.POST("/clear", r.metricsHistoryHandler.ClearMetrics) // Admin only
	}

	r.logger.Info("Metrics history API endpoints configured")

	// User Quota API (v1.11.7)
	if r.quotaHandler != nil && r.authenticator != nil {
		quotaAPI := r.engine.Group("/api/quota")
		quotaAPI.Use(r.authenticator.AuthenticationMiddleware())
		{
			quotaAPI.GET("/me", r.quotaHandler.GetMyQuota) // Current user's quota stats
		}
		r.logger.Info("User quota API endpoints configured")
	}
}

// setupRequestsRoutes настраивает эндпоинты для request monitoring (TUI-04)
func (r *Router) setupRequestsRoutes() {
	requestsAPI := r.engine.Group("/api/requests")
	{
		requestsAPI.GET("", r.requestsHandler.ListRequests)          // Список запросов с фильтрацией
		requestsAPI.GET("/stats", r.requestsHandler.GetRequestStats) // Статистика
		requestsAPI.GET("/:id", r.requestsHandler.GetRequest)        // Детали запроса
		requestsAPI.DELETE("", r.requestsHandler.ClearRequests)      // Очистка истории (admin only)
	}

	r.logger.Info("Request monitoring API endpoints configured")
}

// setupWebSocketRoutes настраивает WebSocket endpoint для real-time updates (Phase 12.2)
func (r *Router) setupWebSocketRoutes() {
	// WebSocket endpoint (metrics, без auth)
	r.engine.GET("/ws", r.wsHandler.HandleWebSocket)

	// WebSocket endpoint для chat с API key auth (DESKTOP-03 v2.4.3)
	r.engine.GET("/ws/chat", r.wsHandler.HandleChatWebSocket)

	r.logger.WithFields(logrus.Fields{
		"endpoint":       "/ws",
		"chat_endpoint":  "/ws/chat",
		"clients_active": r.wsHub.GetActiveClientsCount(),
	}).Info("WebSocket endpoints configured")
}

// setupSystemRoutes настраивает System endpoints (bootstrap, init-status) (Version 1.3.0+)
func (r *Router) setupSystemRoutes() {
	// Пропускаем если system handler не инициализирован
	if r.systemHandler == nil {
		return
	}

	// Public system endpoints (no authentication required)
	system := r.engine.Group("/api/system")
	{
		system.GET("/init-status", r.systemHandler.GetInitStatus)
		system.POST("/bootstrap", r.systemHandler.Bootstrap)

		// Changelog endpoints (v1.4.11)
		if r.changelogHandler != nil {
			system.GET("/info", r.changelogHandler.GetSystemInfo)
			system.GET("/changelogs", r.changelogHandler.GetChangelogs)
			system.GET("/changelogs/:version", r.changelogHandler.GetChangelog)
		}

		// Public models endpoint for login page
		system.GET("/models", func(c *gin.Context) {
			// Return models from inference v4 if available
			if r.inferenceRouter != nil {
				models := r.inferenceRouter.ListModels()
				data := make([]gin.H, 0, len(models))
				for _, m := range models {
					data = append(data, gin.H{
						"id":       m.Spec.Alias,
						"object":   "model",
						"owned_by": string(m.Spec.Provider),
						"created":  0,
					})
				}
				c.JSON(http.StatusOK, gin.H{"object": "list", "data": data})
				return
			}
			c.JSON(http.StatusOK, gin.H{"object": "list", "data": []interface{}{}})
		})

		// Backend status endpoint (v3.3.x) - shows which inference backend is active
		system.GET("/backend", func(c *gin.Context) {
			backendType := r.config.Inference.Backend
			if backendType == "" {
				backendType = "docker" // default
			}

			resp := gin.H{
				"backend": backendType,
				"ready":   false,
			}

			if backendType == "docker" && r.inferenceRouter != nil {
				models := r.inferenceRouter.ListModels()
				runningCount := 0
				for _, m := range models {
					if m.Status == "running" {
						runningCount++
					}
				}
				resp["ready"] = true
				resp["loaded_models"] = len(models)
				resp["running_models"] = runningCount
				resp["max_running_models"] = r.config.Inference.Docker.MaxRunningModels
				resp["docker_enabled"] = r.config.Inference.Docker.Enabled
			}

			c.JSON(http.StatusOK, resp)
		})
	}

	r.logger.Info("System API endpoints configured")
}

// setupDashboardRoutes настраивает Dashboard batch API endpoints (v3.1.0: AJAX-01)
func (r *Router) setupDashboardRoutes() {
	if r.dashboardHandler == nil {
		r.logger.Info("Dashboard handler not initialized, skipping dashboard batch routes")
		return
	}

	// Dashboard routes требуют JWT аутентификации
	dashboard := r.engine.Group("/api/dashboard")
	if r.jwtManager != nil {
		dashboard.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
	}
	{
		dashboard.GET("/stats", r.dashboardHandler.GetDashboardStats)
	}

	r.logger.Info("✅ Dashboard batch API endpoints configured")
}

// setupFrameworkRoutes настраивает UI Framework asset routes (v3.1.0)
func (r *Router) setupFrameworkRoutes() {
	if r.frameworkHandler == nil {
		r.logger.Info("Framework handler not initialized, skipping framework routes")
		return
	}

	// Register framework routes
	r.frameworkHandler.RegisterRoutes(r.engine)

	r.logger.Info("✅ UI Framework routes configured")
}

// setupAuthRoutes настраивает User Authentication endpoints (Version 1.3.0+)
func (r *Router) setupAuthRoutes() {
	// Если handlers не инициализированы, пропускаем
	if r.authHandler == nil || r.userHandler == nil || r.tenantHandler == nil {
		return
	}

	// Public authentication endpoints (no middleware)
	authPublic := r.engine.Group("/api/auth")
	{
		authPublic.POST("/register", r.authHandler.Register)
		authPublic.POST("/login", r.authHandler.Login)
		authPublic.POST("/refresh", r.authHandler.RefreshToken)

		// Auth providers status (for UI to show SSO buttons)
		authPublic.GET("/providers", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"oidc_enabled": r.oidcHandler != nil,
				"ldap_enabled": r.ldapHandler != nil,
			})
		})
	}

	// OIDC authentication endpoints (Version 1.11.1+: Keycloak SSO Integration)
	if r.oidcHandler != nil {
		oidcPublic := r.engine.Group("/api/auth/oidc")
		{
			oidcPublic.GET("/login", r.oidcHandler.HandleLogin)
			oidcPublic.GET("/callback", r.oidcHandler.HandleCallback)
			oidcPublic.POST("/logout", r.oidcHandler.HandleLogout)
		}
		r.logger.Info("OIDC authentication routes registered")
	}

	// LDAP/Active Directory authentication endpoints (Version 1.11.3+: LDAP Integration)
	if r.ldapHandler != nil {
		ldapPublic := r.engine.Group("/api/auth/ldap")
		{
			ldapPublic.POST("/login", r.ldapHandler.HandleLogin)
		}

		// Admin endpoints (requires admin authentication)
		if r.jwtManager != nil && r.db != nil {
			ldapAdmin := r.engine.Group("/api/auth/ldap")
			ldapAdmin.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
			ldapAdmin.Use(middleware.RequireAdmin(r.db, r.logger))
			{
				ldapAdmin.GET("/test", r.ldapHandler.HandleTestConnection)
			}
		}

		r.logger.Info("LDAP authentication routes registered")
	}

	// Protected authentication endpoints (require JWT)
	if r.jwtManager != nil {
		authProtected := r.engine.Group("/api/auth")
		authProtected.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		{
			authProtected.POST("/logout", r.authHandler.Logout)
			authProtected.GET("/me", r.authHandler.Me)

			// Device registration endpoint (Version 2.4.0+: Desktop Client Support)
			if r.deviceHandler != nil {
				authProtected.POST("/devices/register", r.deviceHandler.RegisterDevice)
			}
		}

		// Device management endpoints (Version 2.4.2+: Device Management API)
		if r.deviceHandler != nil && r.jwtManager != nil {
			devices := r.engine.Group("/api/auth/devices")
			devices.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
			{
				devices.GET("", r.deviceHandler.ListDevices)
				devices.GET("/:id", r.deviceHandler.GetDevice)
				devices.DELETE("/:id", r.deviceHandler.DeleteDevice)
				devices.PATCH("/:id", r.deviceHandler.UpdateDeviceName)
			}
		}

		// User management endpoints
		users := r.engine.Group("/api/users/me")
		users.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		{
			users.GET("", r.userHandler.GetProfile)
			users.PUT("", r.userHandler.UpdateProfile)
			users.DELETE("", r.userHandler.DeleteAccount)
			users.GET("/tenants", r.userHandler.ListTenants)
			users.GET("/api-keys", r.userHandler.ListPersonalAPIKeys)
			users.POST("/api-keys", r.userHandler.CreatePersonalAPIKey)
			users.DELETE("/api-keys/:id", r.userHandler.DeletePersonalAPIKey)
			users.POST("/password", r.userHandler.ChangePassword)
		}

		// Tenant management endpoints
		tenants := r.engine.Group("/api/tenants")
		tenants.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		{
			tenants.POST("", r.tenantHandler.CreateTenant)
			tenants.GET("/:id", r.tenantHandler.GetTenant)
			tenants.PUT("/:id", r.tenantHandler.UpdateTenant)
			tenants.DELETE("/:id", r.tenantHandler.DeleteTenant)

			// Tenant members
			tenants.GET("/:id/search-users", r.tenantHandler.SearchUsers)
			tenants.GET("/:id/members", r.tenantHandler.ListMembers)
			tenants.POST("/:id/members", r.tenantHandler.AddMember)
			tenants.PUT("/:id/members/:user_id", r.tenantHandler.UpdateMemberRole)
			tenants.DELETE("/:id/members/:user_id", r.tenantHandler.RemoveMember)

			// Tenant API keys (v1.5.11)
			tenants.GET("/:id/api-keys", r.tenantHandler.ListTenantAPIKeys)
			tenants.POST("/:id/api-keys", r.tenantHandler.CreateTenantAPIKey)
			tenants.DELETE("/:id/api-keys/:key_id", r.tenantHandler.DeleteTenantAPIKey)
		}
	}

	r.logger.Info("User authentication API endpoints configured")
}

// setupConversationsRoutes настраивает Conversations API endpoints (Version 1.3.0)
func (r *Router) setupConversationsRoutes() {
	// Если handler не инициализирован, пропускаем
	if r.conversationHandler == nil || r.jwtManager == nil {
		return
	}

	// Conversations endpoints (require JWT authentication)
	conversations := r.engine.Group("/api/conversations")
	conversations.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
	{
		conversations.POST("", r.conversationHandler.CreateConversation)       // Создать беседу
		conversations.GET("", r.conversationHandler.ListConversations)         // Список бесед
		conversations.GET("/:id", r.conversationHandler.GetConversation)       // Получить беседу с сообщениями
		conversations.PUT("/:id", r.conversationHandler.UpdateConversation)    // Обновить беседу
		conversations.DELETE("/:id", r.conversationHandler.DeleteConversation) // Удалить беседу

		// Messages sub-routes
		conversations.POST("/:id/messages", r.conversationHandler.CreateMessage) // Добавить сообщение
		conversations.GET("/:id/messages", r.conversationHandler.ListMessages)   // Получить сообщения

		// Export/Import endpoints (Version 1.12.3+, v2.0.0 UI)
		if r.conversationExportHandler != nil {
			conversations.GET("/:id/export", r.conversationExportHandler.ExportConversation)        // Export conversation
			conversations.POST("/import", r.conversationExportHandler.ImportConversation)           // Import conversation
			conversations.POST("/bulk-export", r.conversationExportHandler.BulkExportConversations) // Bulk export
		}
	}

	r.logger.Info("Conversations API endpoints configured")
}

// setupUsageRoutes настраивает Usage Statistics API endpoints (Version 1.3.0)
func (r *Router) setupUsageRoutes() {
	// Если handler не инициализирован, пропускаем
	if r.usageHandler == nil || r.jwtManager == nil {
		return
	}

	usage := r.engine.Group("/api/usage")
	usage.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
	{
		usage.GET("/personal", r.usageHandler.GetUserUsage)
		usage.GET("/tenant/:tenant_id", r.usageHandler.GetTenantUsage)
	}

	r.logger.Info("Usage Statistics API endpoints configured")
}

// setupWebUIRoutes настраивает static file serving для WebUI (Version 1.3.0+, v3.2.0: Svelte support)
func (r *Router) setupWebUIRoutes() {
	webUIVersion := r.config.Server.WebUI.Version
	r.logger.WithField("version", webUIVersion).Info("Setting up WebUI routes")

	// Check which UI version to use
	if webUIVersion == "svelte" {
		r.setupSvelteUIRoutes()
	} else {
		r.setupLegacyUIRoutes()
	}
}

// setupSvelteUIRoutes serves the Svelte SPA from embedded files (v3.2.0+)
func (r *Router) setupSvelteUIRoutes() {
	staticFS, err := web.GetStaticFS("svelte")
	if err != nil {
		r.logger.WithError(err).Error("Failed to get Svelte static FS, falling back to legacy")
		r.setupLegacyUIRoutes()
		return
	}

	// Check if Svelte UI is available
	if !web.HasSvelteUI() {
		r.logger.Warn("Svelte UI not available, falling back to legacy")
		r.setupLegacyUIRoutes()
		return
	}

	// Read index.html content once for SPA fallback
	indexHTML, err := fs.ReadFile(staticFS, "index.html")
	if err != nil {
		r.logger.WithError(err).Error("Failed to read index.html, falling back to legacy")
		r.setupLegacyUIRoutes()
		return
	}

	// Create file server for static assets
	fileServer := http.FileServer(http.FS(staticFS))

	// Helper to serve index.html for SPA routes
	serveIndex := func(c *gin.Context) {
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	}

	// Helper to serve static files
	serveStatic := func(c *gin.Context) {
		// Check if file exists
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := staticFS.Open(path)
		if err != nil {
			// File not found - serve index.html for SPA routing
			serveIndex(c)
			return
		}
		f.Close()

		// Serve actual file
		fileServer.ServeHTTP(c.Writer, c.Request)
	}

	// Serve static assets (JS, CSS, images, etc.)
	r.engine.GET("/_app/*filepath", serveStatic)

	// Serve favicon
	r.engine.GET("/favicon.png", serveStatic)

	// Root path - serve index.html
	r.engine.GET("/", serveIndex)

	// All SvelteKit routes (must be registered explicitly for Gin)
	svelteRoutes := []string{
		"/login",
		"/register",
		"/bootstrap",
		"/dashboard",
		"/chat",
		"/api-keys",
		"/tenants",
		"/profile",
		"/settings",
		"/files",
		"/rag",
		"/mcp",
		"/downloads",
		"/usage",
		"/monitor",
		"/about",
		"/admin",
		"/admin/users",
		"/admin/invitations",
		"/admin/api-keys",
		"/admin/models",
		"/admin/settings",
		"/admin/backups",
		"/admin/logs",
		"/admin/providers",
	}

	for _, route := range svelteRoutes {
		r.engine.GET(route, serveIndex)
	}

	// Catch-all for any other paths (SPA fallback)
	r.engine.NoRoute(func(c *gin.Context) {
		// Don't intercept API routes
		if strings.HasPrefix(c.Request.URL.Path, "/api/") ||
			strings.HasPrefix(c.Request.URL.Path, "/v1/") ||
			strings.HasPrefix(c.Request.URL.Path, "/framework/") ||
			strings.HasPrefix(c.Request.URL.Path, "/metrics") ||
			strings.HasPrefix(c.Request.URL.Path, "/health") ||
			strings.HasPrefix(c.Request.URL.Path, "/debug/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
			return
		}
		serveStatic(c)
	})

	r.logger.Info("Svelte WebUI configured (SPA mode)")
}

// setupLegacyUIRoutes serves the legacy HTML/JS frontend (v1.3.0+)
func (r *Router) setupLegacyUIRoutes() {
	// Smart root handler - redirect based on auth status
	r.engine.GET("/", func(c *gin.Context) {
		// Check for JWT token in Authorization header or cookie
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header - redirect to login
			c.Redirect(http.StatusFound, "/login.html")
			return
		}
		// Has auth header - redirect to dashboard
		c.Redirect(http.StatusFound, "/dashboard.html")
	})

	// Serve specific HTML files at root level
	r.engine.StaticFile("/login.html", "./web/login.html")
	r.engine.StaticFile("/register.html", "./web/register.html")
	r.engine.StaticFile("/bootstrap.html", "./web/bootstrap.html")
	r.engine.StaticFile("/chat.html", "./web/chat.html") // Chat interface (renamed from index.html)
	r.engine.StaticFile("/dashboard.html", "./web/dashboard.html")
	r.engine.StaticFile("/profile.html", "./web/profile.html")
	r.engine.StaticFile("/profile-devices.html", "./web/profile-devices.html") // Device Management (DESKTOP-02, v2.4.2)
	r.engine.StaticFile("/tenants.html", "./web/tenants.html")
	r.engine.StaticFile("/api-keys.html", "./web/api-keys.html")
	r.engine.StaticFile("/files.html", "./web/files.html")             // Files Management (v1.10.0)
	r.engine.StaticFile("/rag-sources.html", "./web/rag-sources.html") // RAG Data Sources (v1.13.0)
	r.engine.StaticFile("/usage.html", "./web/usage.html")
	r.engine.StaticFile("/mcp.html", "./web/mcp.html")                             // MCP Catalog (v1.4.5)
	r.engine.StaticFile("/about.html", "./web/about.html")                         // About System (v1.4.11)
	r.engine.StaticFile("/admin.html", "./web/admin.html")                         // Admin Panel (Version 1.3.0)
	r.engine.StaticFile("/admin-invitations.html", "./web/admin-invitations.html") // Invitations Management (AUTH-03, v2.2.0)
	r.engine.StaticFile("/admin-rbac.html", "./web/admin-rbac.html")               // RBAC Management (v1.11.5)
	r.engine.StaticFile("/admin-audit.html", "./web/admin-audit.html")             // Audit Log (v1.11.4)
	r.engine.StaticFile("/admin-rag.html", "./web/admin-rag.html")                 // RAG Management (v1.13.0)
	r.engine.StaticFile("/admin-registry.html", "./web/admin-registry.html")       // Model Registry (REGISTRY-03, v2.3.0)
	r.engine.StaticFile("/huggingface.html", "./web/huggingface.html")             // Hugging Face Model Browser (HF-UI-01, v3.0.0)
	r.engine.StaticFile("/downloads.html", "./web/downloads.html")                 // Active Downloads Window (HF-UI-02, v3.0.5)

	// Serve CSS and JS directories
	r.engine.Static("/css", "./web/css")
	r.engine.Static("/js", "./web/js")
	r.engine.Static("/assets", "./web/assets")
	r.engine.Static("/images", "./web/images")

	// Legacy /web/* routes for backward compatibility
	r.engine.Static("/web", "./web")

	r.logger.Info("Legacy WebUI static file serving configured at root path")
}

// setupOpenAIRoutes настраивает OpenAI-совместимые API эндпоинты
func (r *Router) setupOpenAIRoutes() {
	v1 := r.engine.Group("/v1")

	// Если есть и JWT и API Key auth - используем hybrid middleware (принимает оба типа)
	if r.jwtManager != nil && r.authenticator != nil && r.config.Auth.Enabled {
		r.logger.Info("Using hybrid authentication (JWT + API Key) for /v1 endpoints")

		// Hybrid auth принимает либо JWT либо API Key (database-backed + bootstrap admin)
		v1.Use(middleware.HybridAuth(r.jwtManager, r.config, r.db, r.logger))

		// Usage tracking для аналитики (Version 1.3.0+)
		if r.db != nil {
			v1.Use(middleware.UsageTracking(r.db, r.logger))
			r.logger.Info("Usage tracking enabled for /v1 endpoints")
		}

		// Rate limiting только для API Keys (JWT users не ограничены per-key лимитами)
		// Model authorization остается

	} else if r.config.Auth.Enabled && r.authenticator != nil {
		// Только API Key auth (legacy mode)
		r.logger.Info("Using API Key authentication for /v1 endpoints")
		v1.Use(r.authenticator.AuthenticationMiddleware())
		v1.Use(r.authenticator.ModelAuthorizationMiddleware())
		if r.rateLimiter != nil {
			v1.Use(r.rateLimiter.RateLimitMiddleware())
		}

		// Usage tracking для аналитики (Version 1.4.6+)
		if r.db != nil {
			v1.Use(middleware.UsageTracking(r.db, r.logger))
			r.logger.Info("Usage tracking enabled for /v1 endpoints (API Key mode)")
		} else {
			// Fallback на старый метод если нет DB
			v1.Use(r.authenticator.RecordUsageMiddleware())
		}

	} else {
		// Открытые эндпоинты (MVP mode без аутентификации)
		r.logger.Info("Using NO authentication for /v1 endpoints (MVP mode)")

		// Usage tracking даже без auth для мониторинга (Version 1.4.6+)
		if r.db != nil {
			v1.Use(middleware.UsageTracking(r.db, r.logger))
			r.logger.Info("Usage tracking enabled for /v1 endpoints (No auth mode)")
		}

	}


	// /v1/models endpoint - returns models from inference manager + model registry
	v1.GET("/models", func(c *gin.Context) {
		var data []gin.H

		// 1. Inference models (Docker-based)
		if r.inferenceRouter != nil {
			inferenceList := r.inferenceRouter.ListModels()
			for _, m := range inferenceList {
				data = append(data, gin.H{
					"id":       m.Spec.Alias,
					"object":   "model",
					"owned_by": string(m.Spec.Provider),
					"created":  0,
				})
			}
		}

		// 2. Model Registry models (external providers: OpenAI, DeepSeek, Anthropic, Gemini, etc.)
		if r.db != nil && r.registryHandler != nil {
			filter := &models.ModelRegistryFilter{
				Status: models.ModelStatusActive,
			}
			registryModels, err := r.db.ListModelRegistry(c.Request.Context(), filter)
			if err == nil {
				for _, m := range registryModels {
					// Skip models whose provider is missing or disabled
					provider, err := r.db.GetModelProvider(c.Request.Context(), m.ProviderID)
					if err != nil || !provider.Enabled {
						continue
					}
					data = append(data, gin.H{
						"id":       m.ModelID,
						"object":   "model",
						"owned_by": string(provider.ProviderType) + ":" + provider.Name,
						"created":  0,
					})
				}
			}
		}

		if data == nil {
			data = []gin.H{}
		}
		c.JSON(http.StatusOK, gin.H{"object": "list", "data": data})
	})

	// /v1/chat/completions — unified handler: inference models + external providers
	v1.POST("/chat/completions", r.unifiedChatCompletions())
	if r.inferenceProxyHandler != nil {
		v1.POST("/completions", r.inferenceProxyHandler.HandleCompletions)
	}

	// /v1/models/:model — retrieve model details (v4.13.0+)
	v1.GET("/models/:model", r.handleGetModel())

	// Unified passthrough endpoints (v4.13.0+)
	// Routes to inference (Docker) or external providers (Model Registry) based on model.
	// Supports both JSON and multipart/form-data bodies.
	v1.POST("/embeddings", r.unifiedPassthrough("/v1/embeddings"))
	v1.POST("/rerank", r.unifiedPassthrough("/v1/rerank"))
	v1.POST("/audio/transcriptions", r.unifiedPassthrough("/v1/audio/transcriptions"))
	v1.POST("/audio/translations", r.unifiedPassthrough("/v1/audio/translations"))
	v1.POST("/audio/speech", r.unifiedPassthrough("/v1/audio/speech"))
	v1.POST("/images/generations", r.unifiedPassthrough("/v1/images/generations"))
	v1.POST("/images/edits", r.unifiedPassthrough("/v1/images/edits"))
	v1.POST("/images/variations", r.unifiedPassthrough("/v1/images/variations"))
	v1.POST("/moderations", r.unifiedPassthrough("/v1/moderations"))
	r.logger.Info("OpenAI-compatible passthrough routes configured: embeddings, rerank, audio/*, images/*, moderations")

	// /v1/external/chat/completions — dedicated external provider route (v4.11.0+)
	if r.externalProxyHandler != nil {
		v1ext := v1.Group("/external")
		v1ext.POST("/chat/completions", r.externalProxyHandler.HandleChatCompletions)
	}

	// Agent API endpoints (v2.5.0+)
	if r.agentHandler != nil {
		r.logger.Info("Setting up Agent API endpoints")

		if r.jwtManager != nil && r.authenticator != nil && r.config.Auth.Enabled {
			// Hybrid auth
			v1.GET("/agent/tools", r.agentHandler.HandleListTools)
			v1.GET("/agent/tools/:category", r.agentHandler.HandleListToolsByCategory)
		} else if r.config.Auth.Enabled && r.authenticator != nil {
			// API Key auth
			v1.GET("/agent/tools", r.authenticator.PermissionMiddleware("chat"), r.agentHandler.HandleListTools)
			v1.GET("/agent/tools/:category", r.authenticator.PermissionMiddleware("chat"), r.agentHandler.HandleListToolsByCategory)
		} else {
			// No auth
			v1.GET("/agent/tools", r.agentHandler.HandleListTools)
			v1.GET("/agent/tools/:category", r.agentHandler.HandleListToolsByCategory)
		}

		r.logger.Info("Agent API routes: /v1/agent/tools, /v1/agent/tools/:category")
	}
}

// setupAdminRoutes настраивает административные API эндпоинты
func (r *Router) setupAdminRoutes() {
	r.logger.Info("Setting up admin routes")

	// Agent API (AGENT-01, v2.5.0+) - hybrid auth (API Key or JWT Session)
	r.logger.Info("Setting up Agent API routes (hybrid auth)")

	admin := r.engine.Group("/api/admin")

	// Admin routes require JWT authentication + admin role check (Version 1.3.0+)
	if r.jwtManager != nil && r.db != nil {
		r.logger.Info("Admin routes: Using JWT authentication with admin role check")
		admin.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		admin.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		// Fallback to API Key auth (legacy mode)
		r.logger.Info("Admin routes: Using API Key authentication (legacy)")
		admin.Use(r.authenticator.AuthenticationMiddleware())
		admin.Use(r.authenticator.PermissionMiddleware("admin"))
	} else {
		r.logger.Warn("Admin routes: No authentication enabled")
	}

	// Admin Dashboard Summary (moved from setupDashboardRoutes)
	if r.dashboardHandler != nil {
		admin.GET("/summary", r.dashboardHandler.GetAdminSummary)
		r.logger.Info("Admin routes: Registered /summary endpoint")
	}

	// RAG Statistics (v3.2.0+)
	if r.ragStatsHandler != nil {
		admin.GET("/rag/stats", r.ragStatsHandler.GetRAGStats)
		r.logger.Info("Admin routes: Registered /rag/stats endpoint")
	}

	// User Management endpoints (Version 1.3.0+)
	if r.adminUserHandler != nil {
		r.logger.Info("Admin routes: Registering User Management endpoints")
		admin.GET("/users", r.adminUserHandler.ListUsers)
		admin.POST("/users", r.adminUserHandler.CreateUser)
		admin.GET("/users/:id", r.adminUserHandler.GetUser)
		admin.PUT("/users/:id", r.adminUserHandler.UpdateUser)
		admin.DELETE("/users/:id", r.adminUserHandler.DeleteUser)

		// Additional user management actions
		admin.POST("/users/:id/reset-password", r.adminUserHandler.ResetUserPassword)
		admin.PATCH("/users/:id/disable", r.adminUserHandler.DisableUser)
		admin.PATCH("/users/:id/enable", r.adminUserHandler.EnableUser)
	}

	// Tenant Management endpoints (for RBAC/admin)
	if r.tenantHandler != nil {
		r.logger.Info("Admin routes: Registering Tenant Management endpoints")
		admin.GET("/tenants", r.tenantHandler.ListAllTenants)
	}

	// API Keys management (if available)
	if r.adminHandler != nil {
		r.logger.Info("Admin routes: Registering API Keys Management endpoints")

		// API Keys management (OpenAI-style paths)
		admin.GET("/keys", r.adminHandler.ListAPIKeys)
		admin.POST("/keys", r.adminHandler.CreateAPIKey)

		// Enhanced Key Management (AUTH-04) - СНАЧАЛА более специфичные маршруты
		admin.PATCH("/keys/:id/revoke", r.adminHandler.RevokeAPIKey)
		admin.PATCH("/keys/:id/enable", r.adminHandler.EnableAPIKey)
		admin.POST("/keys/:id/extend", r.adminHandler.ExtendAPIKeyExpiration)
		admin.PATCH("/keys/:id/permissions", r.adminHandler.UpdateAPIKeyPermissions)
		admin.GET("/keys/:id/usage", r.adminHandler.GetAPIKeyUsage)

		// Общие маршруты с :id - ПОТОМ
		admin.GET("/keys/:id", r.adminHandler.GetAPIKey)
		admin.PUT("/keys/:id", r.adminHandler.UpdateAPIKey)
		admin.DELETE("/keys/:id", r.adminHandler.DeleteAPIKey)
	}

	// Files management (v1.10.0) - Admin can manage all files
	if r.adminFilesHandler != nil {
		r.logger.Info("Admin routes: Registering Files Management endpoints")
		admin.GET("/files", r.adminFilesHandler.ListAllFiles)
		admin.GET("/files/stats", r.adminFilesHandler.GetFileStats)
		admin.DELETE("/files/:id", r.adminFilesHandler.DeleteFile)
	}

	// Backup & Restore endpoints (v1.5.14)
	if r.backupHandler != nil {
		r.logger.Info("Admin routes: Registering Backup & Restore endpoints")
		admin.POST("/backup", r.backupHandler.CreateBackup)
		admin.GET("/backups", r.backupHandler.ListBackups)
		admin.GET("/backup/:filename", r.backupHandler.DownloadBackup)
		admin.POST("/restore/:filename", r.backupHandler.RestoreBackup)
		admin.DELETE("/backup/:filename", r.backupHandler.DeleteBackup)
	}

	// Audit Logging endpoints (v1.11.4)
	if r.auditHandler != nil {
		r.logger.Info("Admin routes: Registering Audit Logging endpoints")
		admin.GET("/audit", r.auditHandler.GetAuditEvents)
		admin.GET("/audit/stats", r.auditHandler.GetAuditStats)
		admin.GET("/audit/export", r.auditHandler.ExportAuditEvents)
	}

	// RBAC endpoints (v1.11.5)
	if r.rbacHandler != nil {
		r.logger.Info("Admin routes: Registering RBAC Management endpoints")

		// Permissions
		admin.GET("/rbac/permissions", r.rbacHandler.ListPermissions)

		// Roles
		admin.GET("/rbac/roles", r.rbacHandler.ListRoles)
		admin.POST("/rbac/roles", r.rbacHandler.CreateRole)
		admin.GET("/rbac/roles/:id", r.rbacHandler.GetRole)
		admin.PUT("/rbac/roles/:id", r.rbacHandler.UpdateRole)
		admin.DELETE("/rbac/roles/:id", r.rbacHandler.DeleteRole)

		// Role-Permission mapping
		admin.GET("/rbac/roles/:id/permissions", r.rbacHandler.GetRolePermissions)
		admin.POST("/rbac/roles/:id/permissions", r.rbacHandler.AssignPermissionToRole)
		admin.DELETE("/rbac/roles/:id/permissions/:permission_id", r.rbacHandler.RemovePermissionFromRole)

		// User-Role assignments
		admin.GET("/rbac/users/:id/roles", r.rbacHandler.GetUserRoles)
		admin.POST("/rbac/users/:id/roles", r.rbacHandler.AssignRoleToUser)
		admin.DELETE("/rbac/users/:id/roles/:role_id", r.rbacHandler.RemoveRoleFromUser)

		// User permissions (computed)
		admin.GET("/rbac/users/:id/permissions", r.rbacHandler.GetUserPermissions)
	}

	// Settings endpoints (v3.0.9: Configuration in DB)
	if r.settingsUIHandler != nil {
		r.logger.Info("Admin routes: Registering Settings Management endpoints")
		admin.GET("/settings", r.settingsUIHandler.GetSettings)
		admin.GET("/settings/:category", r.settingsUIHandler.GetSettingsByCategory)
		admin.PUT("/settings/:id", r.settingsUIHandler.UpdateSetting) // Phase 3: Editing (v3.0.9)
	}

	// Quota endpoints (v1.11.7)
	if r.quotaHandler != nil {
		r.logger.Info("Admin routes: Registering Quota Management endpoints")

		// Quota CRUD
		admin.GET("/quotas", r.quotaHandler.ListQuotas)
		admin.POST("/quotas", r.quotaHandler.CreateQuota)
		admin.GET("/quotas/:id", r.quotaHandler.GetQuota)
		admin.PUT("/quotas/:id", r.quotaHandler.UpdateQuota)
		admin.DELETE("/quotas/:id", r.quotaHandler.DeleteQuota)

		// Quota usage
		admin.GET("/quotas/:id/usage", r.quotaHandler.GetQuotaUsage)
	}

	// System endpoints
	admin.GET("/stats", func(c *gin.Context) {
		if r.keyManager != nil {
			stats, _ := r.keyManager.GetStats(context.Background())
			c.JSON(200, gin.H{"manager_stats": stats})
		} else {
			c.JSON(200, gin.H{"message": "API Key Management disabled"})
		}
	})

	admin.GET("/rate-limits", func(c *gin.Context) {
		if r.rateLimiter != nil {
			stats := r.rateLimiter.GetLimiterStats()
			c.JSON(200, gin.H{"rate_limit_stats": stats})
		} else {
			c.JSON(200, gin.H{"message": "Rate limiting disabled"})
		}
	})

	// Logs viewer (v1.5.1 - Enhanced Logs System)
	if r.logsHandler != nil {
		r.logger.Info("Admin routes: Registering Logs Management endpoints")
		admin.GET("/logs", r.logsHandler.ListLogFiles)                       // Список всех лог-файлов
		admin.GET("/logs/:filename", r.logsHandler.GetLogFile)               // Содержимое файла
		admin.GET("/logs/:filename/download", r.logsHandler.DownloadLogFile) // Скачать файл
	}

	// Performance Monitoring endpoints (v1.6.2)
	if r.performanceHandler != nil {
		r.logger.Info("Admin routes: Registering Performance Monitoring endpoints")
		admin.GET("/performance/metrics", r.performanceHandler.GetMetrics)             // Текущие метрики
		admin.GET("/performance/leaks", r.performanceHandler.GetLeakStatus)            // Leak detection status
		admin.POST("/performance/reset-baseline", r.performanceHandler.ResetBaseline)  // Reset baseline
		admin.POST("/performance/reset-leaks", r.performanceHandler.ResetLeakDetector) // Reset leak detector
	}

	// pprof endpoints (v1.6.2) - Admin only, if enabled
	if r.config.Observability.Performance.PprofEnabled {
		r.logger.Info("Admin routes: Registering pprof endpoints")
		pprofGroup := admin.Group("/pprof")
		{
			pprofGroup.GET("/", gin.WrapF(pprof.Index))
			pprofGroup.GET("/cmdline", gin.WrapF(pprof.Cmdline))
			pprofGroup.GET("/profile", gin.WrapF(pprof.Profile))
			pprofGroup.GET("/symbol", gin.WrapF(pprof.Symbol))
			pprofGroup.GET("/trace", gin.WrapF(pprof.Trace))
			pprofGroup.GET("/allocs", gin.WrapH(pprof.Handler("allocs")))
			pprofGroup.GET("/block", gin.WrapH(pprof.Handler("block")))
			pprofGroup.GET("/goroutine", gin.WrapH(pprof.Handler("goroutine")))
			pprofGroup.GET("/heap", gin.WrapH(pprof.Handler("heap")))
			pprofGroup.GET("/mutex", gin.WrapH(pprof.Handler("mutex")))
			pprofGroup.GET("/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
		}
	}

	// Model Registry endpoints (v2.3.0+: REGISTRY-01)
	if r.registryHandler != nil {
		r.logger.Info("Admin routes: Registering Model Registry endpoints")

		// Provider Management
		admin.GET("/registry/providers", r.registryHandler.ListProviders)
		admin.POST("/registry/providers", r.registryHandler.CreateProvider)
		admin.GET("/registry/providers/health", r.registryHandler.HealthCheckProviders)
		admin.GET("/registry/providers/:id", r.registryHandler.GetProvider)
		admin.PUT("/registry/providers/:id", r.registryHandler.UpdateProvider)
		admin.DELETE("/registry/providers/:id", r.registryHandler.DeleteProvider)
		admin.POST("/registry/providers/:id/health", r.registryHandler.HealthCheckProvider)
		admin.POST("/registry/providers/:id/discover", r.registryHandler.DiscoverModelsProvider)

		// Model Registry Management
		admin.GET("/registry/models", r.registryHandler.ListModels)
		admin.POST("/registry/models", r.registryHandler.RegisterModel)
		admin.GET("/registry/models/:id", r.registryHandler.GetModel)
		admin.PUT("/registry/models/:id", r.registryHandler.UpdateModel)

		admin.DELETE("/registry/models/:id", r.registryHandler.DeleteModel)

		// Model Discovery
		admin.POST("/registry/discover", r.registryHandler.DiscoverModels)

		// Registry Stats
		admin.GET("/registry/stats", r.registryHandler.GetStats)
	}

	r.logger.Info("Admin routes configured successfully")

	// SSE stream вне admin group (SSE не поддерживает Authorization header)
	// Используем отдельный middleware который читает token из query параметра
	if r.logsHandler != nil && r.jwtManager != nil && r.db != nil {
		sseAuth := middleware.SSEAuthMiddleware(r.jwtManager, r.logger)
		requireAdmin := middleware.RequireAdmin(r.db, r.logger)
		r.engine.GET("/api/admin/logs/stream", sseAuth, requireAdmin, r.logsHandler.StreamLogs)
		r.logger.Info("SSE logs stream registered with query token auth")
	}
}

// setupAPIKeyManagement настраивает API Key Management компоненты
func (r *Router) setupAPIKeyManagement(cfg *config.Config, logger *logrus.Logger) error {
	logger.Info("Setting up API Key Management")

	// Создаем API Key Storage
	var stor storage.APIKeyStorage
	switch cfg.Auth.StorageType {
	case "json":
		stor = storage.NewJSONStorage(cfg.Auth.StoragePath, logger)
	default:
		return fmt.Errorf("unsupported storage type: %s", cfg.Auth.StorageType)
	}

	// Создаем API Key Manager
	keyManager := apikey.NewManager(cfg, logger, stor)

	// Создаем Rate Limiter
	rateLimiter := ratelimit.NewLimiter(cfg, logger)

	// Connect Redis to rate limiter if available (v3.0.6+)
	if r.redisManager != nil && r.redisManager.RateLimit != nil {
		rateLimiter.SetRedisService(r.redisManager.RateLimit)
		logger.Info("Rate limiter connected to Redis (distributed mode)")
	} else {
		logger.Info("Rate limiter using in-memory mode (no Redis)")
	}

	// Создаем Authenticator
	authenticator := authMiddleware.NewAPIKeyAuthenticator(cfg, logger, keyManager)

	// Connect API Key Worker to authenticator if Redis is available (v3.0.6+)
	if r.redisManager != nil && r.redisManager.BackgroundSync != nil {
		// Create API Key Worker if not already initialized
		apiKeyWorker := r.redisManager.BackgroundSync.APIKeyWorker
		if apiKeyWorker == nil {
			// Initialize API Key Worker with Database adapter
			dbProvider := &apiKeyDatabaseAdapter{db: r.db}
			apiKeyWorker = redis.NewAPIKeyWorker(
				r.redisManager,
				logger,
				dbProvider,     // Database adapter for cache-through
				15*time.Minute, // Cache TTL: 15 минут
			)
			r.redisManager.BackgroundSync.APIKeyWorker = apiKeyWorker
		}

		// Connect worker to authenticator for cache-through
		authenticator.SetAPIKeyWorker(apiKeyWorker)

		// Connect worker to key manager for cache invalidation
		keyManager.SetCacheInvalidator(apiKeyWorker)

		logger.Info("✅ API Key cache-through + invalidation connected (Redis)")
	} else {
		logger.Debug("API Key cache-through disabled (Redis not available)")
	}

	// Сохраняем компоненты
	r.storage = stor
	r.keyManager = keyManager
	r.rateLimiter = rateLimiter
	r.authenticator = authenticator

	logger.Info("API Key Management setup completed")
	return nil
}

// setupRedis настраивает Redis для distributed caching и session management (v3.0.6+)
func (r *Router) setupRedis(cfg *config.Config, logger *logrus.Logger) error {
	// Check if Redis is enabled
	if !cfg.Auth.RateLimiting.Redis.Enabled {
		logger.Info("Redis is disabled (using in-memory fallback)")
		return nil
	}

	logger.WithFields(logrus.Fields{
		"url":        cfg.Auth.RateLimiting.Redis.URL,
		"key_prefix": cfg.Auth.RateLimiting.Redis.KeyPrefix,
	}).Info("Initializing Redis...")

	// Create Redis manager
	redisManager, err := redis.NewManager(redis.ManagerConfig{
		URL:        cfg.Auth.RateLimiting.Redis.URL,
		KeyPrefix:  cfg.Auth.RateLimiting.Redis.KeyPrefix,
		DB:         0, // Default database
		MaxRetries: 3,
		PoolSize:   10,
		Enabled:    true,
	}, logger)
	if err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisManager.Ping(ctx); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Get initial stats
	stats, err := redisManager.GetStats(ctx)
	if err == nil {
		logger.WithFields(logrus.Fields{
			"db_size": stats["db_size"],
		}).Info("Redis connected successfully")
	}

	r.redisManager = redisManager

	logger.Info("✅ Redis initialized successfully")
	return nil
}

// setupModelRegistry инициализирует Model Registry систему (Version 2.3.0+: REGISTRY-01)
func (r *Router) setupModelRegistry(cfg *config.Config, logger *logrus.Logger) error {
	logger.Info("Setting up Model Registry")

	// Создаем Provider Manager
	r.providerManager = providers.NewProviderManager(r.db, logger)

	// Загружаем providers из БД
	ctx := context.Background()
	if err := r.providerManager.LoadProvidersFromDB(ctx); err != nil {
		return fmt.Errorf("failed to load providers from database: %w", err)
	}

	// Initial model discovery если включен
	if cfg.ModelRegistry.AutoDiscovery.Enabled {
		logger.Info("Running initial model discovery...")
		discovered, err := r.providerManager.DiscoverModels(ctx)
		if err != nil {
			logger.WithError(err).Warn("Initial model discovery failed, will retry later")
		} else {
			logger.Infof("Initial model discovery complete: %d new models registered", discovered)
		}
	}

	// Initial health check для всех providers
	if cfg.ModelRegistry.HealthCheck.Enabled {
		logger.Info("Running initial provider health check...")
		r.providerManager.HealthCheckAll(ctx)
		logger.Info("Initial health check complete")
	}

	// Запускаем background loops в горутинах
	if cfg.ModelRegistry.AutoDiscovery.Enabled && cfg.ModelRegistry.AutoDiscovery.Interval > 0 {
		go r.providerManager.RunDiscoveryLoop(ctx, cfg.ModelRegistry.AutoDiscovery.Interval)
		logger.Infof("Auto-discovery loop started with interval: %s", cfg.ModelRegistry.AutoDiscovery.Interval)
	}

	if cfg.ModelRegistry.HealthCheck.Enabled && cfg.ModelRegistry.HealthCheck.Interval > 0 {
		go r.providerManager.RunHealthCheckLoop(ctx, cfg.ModelRegistry.HealthCheck.Interval)
		logger.Infof("Health check loop started with interval: %s", cfg.ModelRegistry.HealthCheck.Interval)
	}

	logger.Info("Model Registry setup completed")
	return nil
}

// setupInvitationsRoutes настраивает Invitation System endpoints (AUTH-03, v2.2.0)
func (r *Router) setupInvitationsRoutes() {
	if r.invitationHandler == nil {
		r.logger.Warn("Invitation handler not initialized, skipping invitation routes")
		return
	}

	r.logger.Info("Setting up invitation routes")

	// Public route for invitation validation (no auth required)
	publicInvitations := r.engine.Group("/api/invitations")
	{
		publicInvitations.GET("/:token/validate", r.invitationHandler.ValidateInvitation)
	}

	// Admin routes (requires admin auth)
	adminInvitations := r.engine.Group("/api/admin/invitations")

	// Use JWT authentication for admin routes
	if r.jwtManager != nil && r.db != nil {
		r.logger.Info("Invitation admin routes: Using JWT authentication with admin role check")
		adminInvitations.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		adminInvitations.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		// Fallback to API Key auth (legacy mode)
		r.logger.Info("Invitation admin routes: Using API Key authentication (legacy)")
		adminInvitations.Use(middleware.APIKeyDBAuth(r.config, r.db, r.logger))
	} else {
		r.logger.Warn("Invitation admin routes: No authentication configured!")
	}

	{
		adminInvitations.POST("", r.invitationHandler.CreateInvitation)        // Create invitation
		adminInvitations.GET("", r.invitationHandler.ListInvitations)          // List invitations
		adminInvitations.GET("/stats", r.invitationHandler.GetInvitationStats) // Get statistics
		adminInvitations.GET("/:id", r.invitationHandler.GetInvitationDetails) // Get invitation details with user info
		adminInvitations.DELETE("/:id", r.invitationHandler.RevokeInvitation)  // Revoke invitation
	}

	r.logger.Info("Invitation routes configured successfully")
}

// setupHandlers инициализирует все handlers
func (r *Router) setupHandlers(cfg *config.Config, logger *logrus.Logger) {
	logger.WithField("db_is_nil", r.db == nil).Info("DEBUG: setupHandlers called")

	// RAG Data Sources handler (v1.13.1+)
	if r.ragDataSourceService != nil {
		r.ragDataSourcesHandler = handlers.NewRAGDataSourcesHandler(r.ragDataSourceService, logger)
	}

	// RAG Stats handler (v3.2.0+)
	r.ragStatsHandler = handlers.NewRAGStatsHandler(r.vectorStore, logger)

	// Logs handler (v1.5.1) - извлекаем директорию из Logging.FilePath
	logsDir := "./logs" // По умолчанию
	if cfg.Logging.FilePath != "" {
		// Извлекаем директорию из пути к файлу
		lastSlash := strings.LastIndex(cfg.Logging.FilePath, "/")
		lastBackslash := strings.LastIndex(cfg.Logging.FilePath, "\\")
		if lastBackslash > lastSlash {
			lastSlash = lastBackslash
		}
		if lastSlash > 0 {
			logsDir = cfg.Logging.FilePath[:lastSlash]
		}
	}
	r.logsHandler = handlers.NewLogsHandler(logsDir, logger)

	// System and User Authentication handlers (Version 1.3.0+)
	if r.bootstrapService != nil {
		r.systemHandler = handlers.NewSystemHandler(cfg, logger, r.bootstrapService)
		logger.Info("System handler initialized")
	}

	if r.authService != nil && r.db != nil {
		r.authHandler = handlers.NewAuthHandler(r.authService, logger, r.auditLogger)

		// Connect Redis to auth handler for JWT session storage (v3.0.6+)
		if r.redisManager != nil {
			r.authHandler.SetRedisManager(r.redisManager)
			logger.Debug("Redis manager connected to auth handler")
		}

		r.deviceHandler = handlers.NewDeviceHandler(r.db, logger) // Version 2.4.0+: Device management
		r.userHandler = handlers.NewUserHandler(r.db, logger, r.auditLogger)
		r.tenantHandler = handlers.NewTenantHandler(r.db, logger, r.auditLogger)
		r.conversationHandler = handlers.NewConversationHandler(r.db)
		r.conversationExportHandler = handlers.NewConversationExportHandler(r.db, logger) // v1.12.3+ Export/Import
		r.adminUserHandler = handlers.NewAdminUserHandler(r.db, logger, r.auditLogger)
		r.usageHandler = handlers.NewUsageHandler(r.db, logger)
		logger.Info("User authentication handlers initialized")
	}

	// OIDC/Keycloak SSO handler (Version 1.11.1+: Enterprise Suite)
	if cfg.Auth.OIDC.Enabled && r.db != nil && r.jwtManager != nil {
		logger.Info("Initializing OIDC provider...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		oidcProvider, err := oidcauth.NewOIDCProvider(ctx, &cfg.Auth.OIDC, logger)
		if err != nil {
			logger.WithError(err).Error("Failed to initialize OIDC provider")
			logger.Warn("OIDC authentication will be unavailable")
		} else {
			r.oidcHandler = handlers.NewOIDCHandler(cfg, oidcProvider, r.db, r.jwtManager, logger)
			logger.WithFields(logrus.Fields{
				"provider": cfg.Auth.OIDC.Provider,
				"issuer":   cfg.Auth.OIDC.Issuer,
			}).Info("OIDC handler initialized successfully")
		}
	} else if cfg.Auth.OIDC.Enabled {
		logger.Warn("OIDC is enabled but required components (database or JWT manager) are not available")
	}

	// LDAP/Active Directory handler (Version 1.11.3+: Enterprise Suite)
	if cfg.Auth.LDAP.Enabled && r.db != nil && r.jwtManager != nil {
		logger.Info("Initializing LDAP client...")

		ldapClient, err := ldapauth.NewClient(&cfg.Auth.LDAP, logger)
		if err != nil {
			logger.WithError(err).Error("Failed to initialize LDAP client")
			logger.Warn("LDAP authentication will be unavailable")
		} else {
			r.ldapHandler = handlers.NewLDAPHandler(cfg, ldapClient, r.db, r.jwtManager, logger)
			logger.WithFields(logrus.Fields{
				"url":          cfg.Auth.LDAP.URL,
				"user_base_dn": cfg.Auth.LDAP.UserBaseDN,
			}).Info("LDAP handler initialized successfully")
		}
	} else if cfg.Auth.LDAP.Enabled {
		logger.Warn("LDAP is enabled but required components (database or JWT manager) are not available")
	}

	// Audit Logger initialization (Version 1.11.4+: Enhanced Audit Logging)
	if r.db != nil {
		logger.Info("Initializing Audit logger...")
		r.auditLogger = audit.NewAuditLogger(r.db, logger)
		logger.Info("Audit logger initialized successfully")

		// Audit Handler initialization
		r.auditHandler = handlers.NewAuditHandler(cfg, r.db, logger)
		logger.Info("Audit handler initialized successfully")

		// RBAC Service initialization (Version 1.11.5+: Custom Roles & Permissions)
		logger.Info("Initializing RBAC service...")
		r.rbacService = rbac.NewService(r.db, logger)
		logger.Info("RBAC service initialized successfully")

		// RBAC Handler initialization
		r.rbacHandler = handlers.NewRBACHandler(r.db, r.rbacService, logger)
		logger.Info("RBAC handler initialized successfully")

		// RBAC Middleware initialization
		r.rbacMiddleware = middleware.NewRBACMiddleware(r.rbacService, r.db, logger)
		logger.Info("RBAC middleware initialized successfully")

		// Quota Service initialization (Version 1.11.7+: Usage Quotas System)
		logger.Info("Initializing Quota service...")
		r.quotaService = quota.NewService(r.db, logger)
		logger.Info("Quota service initialized successfully")

		// Quota Handler initialization
		r.quotaHandler = handlers.NewQuotaHandler(r.db, r.quotaService, logger)
		logger.Info("Quota handler initialized successfully")

		// Invitation Handler initialization (AUTH-03, v2.2.0)
		r.invitationHandler = handlers.NewInvitationHandler(r.db, cfg, logger)
		logger.Info("Invitation handler initialized successfully")

		// Model Registry Handler initialization (REGISTRY-03, v2.3.0)
		if r.providerManager != nil {
			r.registryHandler = handlers.NewRegistryHandler(r.db, r.providerManager, logger)
			logger.Info("Model Registry handler initialized successfully")
		}

		// Dashboard Handler initialization (v3.1.0: AJAX-01)
		r.dashboardHandler = handlers.NewDashboardHandler(r.db, logger)
		logger.Info("Dashboard batch handler initialized successfully")

		// UI Framework initialization (v3.1.0: Framework)
		// Skip for Svelte UI - legacy framework is not needed
		if cfg.Server.WebUI.Version != "svelte" {
			devMode := os.Getenv("ENV") == "development" || os.Getenv("ENV") == "dev"
			frameworkBuilder := framework.NewBuilder(logger, !devMode) // minify in production
			if err := frameworkBuilder.Build(); err != nil {
				logger.WithError(err).Warn("Failed to build UI framework, continuing without it")
			} else {
				r.frameworkHandler = framework.NewHandler(frameworkBuilder, logger, devMode)
				logger.WithFields(logrus.Fields{
					"dev_mode": devMode,
					"minified": !devMode,
				}).Info("UI Framework initialized successfully")
			}
		} else {
			logger.Info("UI Framework skipped (Svelte UI mode)")
		}

		// Quota Middleware initialization
		r.quotaMiddleware = middleware.NewQuotaMiddleware(r.quotaService, logger)
		logger.Info("Quota middleware initialized successfully")

		// Metrics Collector initialization (Version 1.11.6+: Prometheus Metrics Export)
		if cfg.Metrics.Enabled {
			logger.Info("Initializing Prometheus metrics collector...")
			r.metricsCollector = metrics.NewMetricsCollector(r.db, logger, 30*time.Second)
			logger.Info("Prometheus metrics collector initialized successfully")
		}

		// Audit Retention Policy initialization
		r.auditRetentionPolicy = audit.NewRetentionPolicy(r.db, logger).
			WithRetentionPeriod(90 * 24 * time.Hour). // 90 days retention
			WithCleanupInterval(24 * time.Hour)       // Daily cleanup
		r.auditRetentionPolicy.Start()
		logger.Info("Audit retention policy started (90 days retention, daily cleanup)")
	} else {
		logger.Warn("Audit logging unavailable (database not configured)")
	}

	logger.Info("DEBUG: BEFORE AdminHandler creation block")

	// Admin handler с API Key Manager если доступен (+ Database для Version 1.3.0+)
	if r.keyManager != nil {
		logger.Info("DEBUG: Creating AdminHandler WITH keyManager and database")
		logger.Infof("DEBUG: has_db=%v, has_keyManager=%v", r.db != nil, true)
		r.adminHandler = handlers.NewAdminHandler(cfg, logger, r.keyManager, r.db)
	} else {
		logger.Info("DEBUG: Creating AdminHandler WITHOUT keyManager but WITH database")
		logger.Infof("DEBUG: has_db=%v, has_keyManager=%v", r.db != nil, false)
		// БД всегда передается, даже если keyManager отсутствует
		r.adminHandler = handlers.NewAdminHandlerWithoutKeys(cfg, logger, r.db)
	}

	// v3.0.5+: Ollama client removed
	// if r.adminHandler != nil && r.ollamaClient != nil {
	// 	r.adminHandler.SetOllamaClient(r.ollamaClient)
	// 	logger.Info("Ollama client set for AdminHandler")
	// }

	logger.Info("DEBUG: AdminHandler created successfully")

	// MCP Handler (v1.4.5)
	if r.db != nil {
		r.mcpHandler = handlers.NewMCPHandler(r.db, logger)
		logger.Info("MCP handler initialized")
	} else {
		logger.Warn("MCP handler NOT initialized: database is nil")
	}

	// Changelog Handler (v1.4.11)
	if r.db != nil {
		r.changelogHandler = handlers.NewChangelogHandler(r.config, r.db, logger)
		logger.Info("Changelog handler initialized")
	} else {
		logger.Warn("Changelog handler NOT initialized: database is nil")
	}

	// Backup Handler (v1.5.14)
	r.backupHandler = handlers.NewBackupHandler(r.config, logger, r.db, r.auditLogger)
	logger.Info("Backup handler initialized")

	// Performance Handler (v1.6.2)
	r.performanceHandler = handlers.NewPerformanceHandler(r.performanceMonitor, r.leakDetector)
	logger.Info("Performance handler initialized")

	// GPU Handler (v1.9.3)
	if r.gpuMonitor != nil {
		r.gpuHandler = handlers.NewGPUHandler(logger, r.gpuMonitor)
		logger.Info("GPU handler initialized")
	}

	// File Storage & Handler (v1.10.0)
	if r.db != nil {
		r.setupFileStorage(cfg, logger)

		// Admin files handler (v1.10.0) - после setupFileStorage
		if r.fileHandler != nil {
			storageBackend, err := filestorageBackend.NewStorageBackend(cfg.FileStorage)
			if err == nil {
				fileService := filestorage.NewService(storageBackend, cfg.FileStorage, logger)
				r.adminFilesHandler = handlers.NewAdminFilesHandler(fileService, r.db, logger)
				logger.Info("Admin files handler initialized")
			}
		}
	} else {
		logger.Warn("File storage NOT initialized: database is nil")
	}

	// Metrics Storage (Phase 12.1)
	r.metricsStorage = metrics.NewMetricsStorage(metrics.DefaultStorageConfig(), logger)
	r.metricsStorage.Start()

	// Request Storage (TUI-04)
	r.requestStorage = request.NewStorage(request.DefaultStorageConfig(), logger)
	r.requestStorage.Start()

	// Stats Handler с metrics storage для latency данных и Database (Version 1.3.0+)
	r.statsHandler = handlers.NewStatsHandler(cfg, logger, r.keyManager, r.version, r.metricsStorage, r.db)

	// Metrics History Handler
	r.metricsHistoryHandler = handlers.NewMetricsHistoryHandler(cfg, logger, r.metricsStorage)

	// Requests Handler (TUI-04)
	r.requestsHandler = handlers.NewRequestsHandler(cfg, logger, r.requestStorage)

	// WebSocket Hub (Phase 12.2)
	r.wsHub = websocket.NewHub(logger)
	go r.wsHub.Run() // Запускаем hub в фоне

	// WebSocket Handler
	r.wsHandler = websocket.NewHandler(r.wsHub, logger)

	// Set database for API key validation (DESKTOP-03)
	r.wsHandler.SetDatabase(r.db)

	// v3.0.5+: WebSocket chat handler temporarily disabled
	// r.wsChatHandler = websocket.NewChatHandler(cfg, logger, r.ollamaClient, r.wsHub)
	// r.wsHandler.SetChatHandler(r.wsChatHandler)
	// if r.agentService != nil && r.db != nil {
	// 	toolRegistry := r.agentService.GetToolRegistry()
	// 	if toolRegistry != nil {
	// 		r.wsChatHandler.SetAgentSupport(r.db, toolRegistry)
	// 		logger.Info("Agent support enabled for WebSocket chat handler")
	// 	}
	// }

	// Event Broadcaster
	r.eventBroadcaster = websocket.NewEventBroadcaster(r.wsHub)

	// Metrics Broadcaster (автоматически отправляет метрики каждые 5 секунд)
	r.metricsBroadcaster = websocket.NewMetricsBroadcaster(
		r.metricsStorage,
		r.eventBroadcaster,
		logger,
		5*time.Second,
	)
	r.metricsBroadcaster.Start()

	logger.Info("WebSocket Hub, Chat Handler and Metrics Storage initialized")

	// HTMX UI Handlers initialization (Version 2.6.0+: HTMX-01)
	if r.db != nil {
		// Initialize template renderer
		var err error
		devMode := cfg.Development.HotReload // Enable hot reload for templates in dev mode
		r.templateRenderer, err = templates.NewRenderer(devMode)
		if err != nil {
			logger.WithError(err).Error("Failed to initialize template renderer")
		} else {
			// Initialize UI handlers
			r.registryUIHandler = handlersUI.NewRegistryUIHandler(r.db, r.templateRenderer, logger)
			r.apiKeysUIHandler = handlersUI.NewAPIKeysUIHandler(r.db, r.templateRenderer, logger)
			r.tenantsUIHandler = handlersUI.NewTenantsUIHandler(r.db, r.templateRenderer, logger)
			r.monitorUIHandler = handlersUI.NewMonitorUIHandler(logger, r.gpuMonitor, r.db)   // HTMX-02: Monitor UI
			r.usersUIHandler = handlersUI.NewUsersUIHandler(r.db, r.templateRenderer, logger) // HTMX-03: Users UI
			r.rbacUIHandler = handlersUI.NewRBACUIHandler(r.db, r.templateRenderer, logger)   // HTMX-03: RBAC UI
			r.dashboardUIHandler = handlersUI.NewDashboardUIHandler(r.db, logger)             // HTMX-03: Dashboard UI

			// Settings UI (v3.0.9: Configuration in DB)
			// Temporary: Use fallback storage (empty settings until Phase 2)
			// TODO: Implement proper DB adapter in Phase 2
			settingsStorage := settings.NewSQLStorageAdapter(r.db, logger)
			settingsManager := settings.NewManager(settingsStorage, logger)
			r.settingsUIHandler = handlersUI.NewSettingsHandler(settingsManager, logger)

			logger.Info("✅ HTMX UI handlers initialized successfully")
		}
	}

	// Hugging Face Integration (Version 3.0.0+: HF-01)
	// Create separate logger for Hugging Face with dedicated log file
	hfLogger := internalLogger.NewFileLogger("logs/huggingface.log", cfg.Logging.Level)
	hfLogger.Info("🤗 Hugging Face logger initialized with separate log file")

	hfAPIToken := cfg.HuggingFace.APIToken
	r.hfClient = huggingface.NewClient(hfAPIToken, hfLogger)

	// Initialize downloader (HF-02)
	// Create separate logger for downloads with dedicated log file
	downloadLogger := internalLogger.NewFileLogger("logs/downloads.log", cfg.Logging.Level)
	downloadLogger.Info("📥 Downloads logger initialized with separate log file")

	downloadsDir := cfg.HuggingFace.ModelsDir
	if downloadsDir == "" {
		downloadsDir = "./data/models"
	}
	maxConcurrent := cfg.HuggingFace.MaxConcurrentDownloads
	if maxConcurrent <= 0 {
		maxConcurrent = 2
	}
	autoResume := cfg.HuggingFace.AutoResume

	var err error
	r.hfDownloader, err = huggingface.NewDownloader(r.hfClient, downloadsDir, maxConcurrent, autoResume, downloadLogger)
	if err != nil {
		downloadLogger.WithError(err).Error("Failed to initialize Hugging Face downloader")
	} else {
		downloadLogger.WithFields(logrus.Fields{
			"downloads_dir":  downloadsDir,
			"max_concurrent": maxConcurrent,
			"auto_resume":    autoResume,
		}).Info("✅ Hugging Face downloader initialized")
	}

	// Initialize UI handler
	if r.templateRenderer != nil && r.hfDownloader != nil {
		r.hfUIHandler = handlersUI.NewHuggingFaceUIHandler(r.hfClient, r.hfDownloader, r.templateRenderer, hfLogger)
		hfLogger.Info("✅ Hugging Face browser initialized")
	}

	// Inference v4 (multi-provider) bootstrap
	// Use cfg.Inference.Docker if backend is "docker", otherwise fallback to legacy config
	dockerCfg := cfg.Inference.Docker
	useDockerInference := cfg.Inference.Backend == "docker" && dockerCfg.Enabled

	var infHFCache, infGGUFCache, infTRTDir string
	var infMaxConcurrent int
	var infAutoResume bool
	var infHTTPTimeout, infHealthTimeout, infStartupTimeout time.Duration
	var infMaxRunning int
	var infCacheMax int64
	var infDockerBin string

	if useDockerInference {
		// Use new Docker inference config (v3.3.0+)
		infHFCache = dockerCfg.HFCacheDir
		if infHFCache == "" {
			infHFCache = "./data/models/hf"
		}
		infGGUFCache = dockerCfg.GGUFDir
		if infGGUFCache == "" {
			infGGUFCache = "./data/models/gguf"
		}
		infTRTDir = dockerCfg.TRTEnginesDir
		if infTRTDir == "" {
			infTRTDir = "./data/engines/trt"
		}
		infMaxConcurrent = dockerCfg.MaxConcurrentDownloads
		if infMaxConcurrent <= 0 {
			infMaxConcurrent = 2
		}
		infAutoResume = dockerCfg.AutoResume
		infHTTPTimeout = cfg.HuggingFace.DefaultDownloadTimeout
		if infHTTPTimeout == 0 {
			infHTTPTimeout = 5 * time.Minute
		}
		infHealthTimeout = dockerCfg.HealthCheckTimeout
		if infHealthTimeout == 0 {
			infHealthTimeout = 60 * time.Second
		}
		infStartupTimeout = dockerCfg.StartupTimeout
		if infStartupTimeout == 0 {
			infStartupTimeout = 5 * time.Minute
		}
		infMaxRunning = dockerCfg.MaxRunningModels
		if infMaxRunning <= 0 {
			infMaxRunning = 2
		}
		infCacheMax = dockerCfg.CacheMaxBytes
		infDockerBin = dockerCfg.DockerBin
		logger.WithFields(logrus.Fields{
			"hf_cache":         infHFCache,
			"gguf_cache":       infGGUFCache,
			"trt_engines":      infTRTDir,
			"max_running":      infMaxRunning,
			"default_provider": dockerCfg.DefaultProvider,
		}).Info("Using Docker-based inference (v3.3.0+)")
	} else {
		// Legacy fallback for non-docker mode
		infHFCache = downloadsDir
		infGGUFCache = infHFCache
		infHTTPTimeout = cfg.HuggingFace.DefaultDownloadTimeout
		if infHTTPTimeout == 0 {
			infHTTPTimeout = 5 * time.Minute
		}
		infMaxConcurrent = maxConcurrent
		infAutoResume = cfg.HuggingFace.AutoResume
		infMaxRunning = cfg.Models.Preload.MaxLoadedModels
		infCacheMax = cfg.HuggingFace.CacheMaxBytes
		infHealthTimeout = 60 * time.Second
		infStartupTimeout = 5 * time.Minute
	}

	infSvc, err := inference.NewService(inference.ServiceConfig{
		HFToken:               hfAPIToken,
		HFCacheDir:            infHFCache,
		GGUFCacheDir:          infGGUFCache,
		TRTEnginesDir:         infTRTDir,
		ContainerLogsDir:      "logs/containers", // Container logs directory
		MaxConcurrentDownload: infMaxConcurrent,
		AutoResume:            infAutoResume,
		HTTPTimeout:           infHTTPTimeout,
		DockerBin:             infDockerBin,
		Logger:                logger,
		MaxRunningModels:      infMaxRunning,
		CacheMaxBytes:         infCacheMax,
		HealthCheckTimeout:    infHealthTimeout,
		StartupTimeout:        infStartupTimeout,
	})
	if err != nil {
		logger.WithError(err).Warn("Inference v4 service init failed")
	} else {
		r.inferenceSvc = infSvc
		r.inferenceMgr = inference.NewManager(infSvc)
		r.inferenceRouter = inference.NewRouter(r.inferenceMgr)
		r.inferenceHandler = handlers.NewInferenceHandler(r.inferenceRouter, logger)
		r.inferenceProxyHandler = handlers.NewInferenceProxyHandler(r.inferenceRouter, logger)

		// Initialize chat tools handler with Tavily web search (v4.0.3+)
		if cfg.Tools.TavilyAPIKey != "" {
			toolsReg := tools.NewRegistry(cfg.Tools.TavilyAPIKey)
			r.chatToolsHandler = handlers.NewChatToolsHandler(r.inferenceRouter, toolsReg, logger)
			logger.Info("Chat tools handler initialized with Tavily web search")
		} else {
			// No Tavily key - handler without tools
			r.chatToolsHandler = handlers.NewChatToolsHandler(r.inferenceRouter, nil, logger)
			logger.Info("Chat tools handler initialized (no Tavily key configured)")
		}

		// Initialize model store for persistence (store in data/ directory)
		modelStore, storeErr := inference.NewModelStore("./data", logger)
		if storeErr != nil {
			logger.WithError(storeErr).Warn("Failed to create model store")
		} else {
			r.inferenceModelStore = modelStore
			r.inferenceHandler.SetModelStore(modelStore)
			logger.Info("Inference model store initialized")

			// Register saved specs to update recovered instances with full specs (capabilities, etc.)
			savedModels := modelStore.List()
			if len(savedModels) > 0 {
				specs := make([]inference.ModelSpec, 0, len(savedModels))
				for _, sm := range savedModels {
					specs = append(specs, sm.ToSpec())
				}
				infSvc.RegisterSavedSpecs(specs)
			}
		}
		// Idle stop using legacy preload.unload_after if set
		idleAfter := cfg.Models.Preload.UnloadAfter
		if idleAfter > 0 {
			checkEvery := idleAfter / 2
			if checkEvery <= 0 {
				checkEvery = idleAfter
			}
			r.inferenceMgr.StartIdleReaper(context.Background(), idleAfter, checkEvery)
			logger.WithFields(logrus.Fields{
				"idle_after": idleAfter,
				"interval":   checkEvery,
			}).Info("Inference idle reaper started")
		}
		// Periodic cache metrics refresh
		r.inferenceMgr.StartCacheGaugeUpdater(context.Background(), time.Minute)
		logger.WithFields(logrus.Fields{
			"hf_cache":     infHFCache,
			"gguf_cache":   infGGUFCache,
			"max_download": maxConcurrent,
			"use_docker":   true,
		}).Info("Inference v4 service initialized")
	}

	// External Provider Proxy (v4.11.0+)
	if r.db != nil {
		r.externalProxyHandler = handlers.NewExternalProxyHandler(r.db, logger)
		logger.Info("External provider proxy handler initialized")
	}

	// GitLab Integration (v3.1.0+)
	// Note: r.engine is nil here (setupHandlers called before setupEngine)
	// Webhook route is registered in setupGitLabRoutes
	if cfg.GitLab.Enabled && r.db != nil {
		// Create separate logger for GitLab with dedicated log file
		gitlabLogger := internalLogger.NewFileLogger("logs/gitlab.log", cfg.Logging.Level)
		if gitlabLogger == nil {
			logger.Error("Failed to create GitLab logger")
			return
		}
		gitlabLogger.Info("🦊 GitLab Integration logger initialized with separate log file")

		// Create GitLab storage using main database
		// Try to get underlying *sql.DB via type assertion
		type sqlDBGetter interface {
			GetDB() interface{}
		}
		if getter, ok := r.db.(sqlDBGetter); ok {
			dbInterface := getter.GetDB()
			if dbInterface == nil {
				logger.Warn("GitLab Integration disabled: GetDB() returned nil")
				return
			}
			if sqlDB, ok := dbInterface.(*sql.DB); ok && sqlDB != nil {
				glStore := gitlabStorage.NewPostgresStore(sqlDB)
				if glStore == nil {
					logger.Error("Failed to create GitLab PostgresStore")
					return
				}
				r.gitlabStore = glStore // Save for use in setupGitLabRoutes

				glHandler := handlers.NewGitLabAdminHandler(glStore, gitlabLogger)
				if glHandler == nil {
					logger.Error("Failed to create GitLabAdminHandler")
					return
				}
				glHandler.SetMainDB(r.db)
				r.gitlabHandler = glHandler

				// Create webhook handler
				webhookService := gitlabWebhook.NewHandler(glStore, gitlabLogger, nil)
				if webhookService == nil {
					logger.Error("Failed to create GitLab webhook service")
					return
				}
				r.gitlabWebhookHandler = handlers.NewGitLabWebhookHandler(webhookService, "", "", gitlabLogger)
				if r.gitlabWebhookHandler == nil {
					logger.Error("Failed to create GitLabWebhookHandler")
					return
				}

				// Get or create internal API key for GitLab workers
				r.gitlabAPIKey = r.getOrCreateGitLabAPIKey(context.Background(), gitlabLogger)
				if r.gitlabAPIKey == "" {
					gitlabLogger.Error("⚠️ GitLab API key not created - workers will fail with 401. Check migration 093 (system user).")
				} else {
					gitlabLogger.WithField("key_prefix", r.gitlabAPIKey[:20]+"...").Info("✅ GitLab API key ready for workers")
				}

				// Initialize RAG service if enabled
				// Uses main RAG config (config.RAG) for Qdrant settings
				// Uses GitLab-specific embedding_model_alias for dynamic embedding resolution (optional - auto-detect if empty)
				var ragService *gitlabRAG.RAGService
				if cfg.GitLab.EnableRAG {
					// Check if Qdrant is configured in main RAG config
					qdrantURL := cfg.RAG.VectorStore.Qdrant.URL
					if qdrantURL == "" && cfg.RAG.VectorStore.Type == "qdrant" {
						qdrantURL = cfg.RAG.VectorStore.ConnectionString
					}

					if qdrantURL == "" {
						gitlabLogger.Warn("RAG enabled but Qdrant not configured in main rag.vector_store section")
					} else {
						gitlabLogger.Info("🔍 Initializing RAG for GitLab code review...")

						// Create dynamic embedding provider (resolves URL from inference registry at runtime)
						// If EmbeddingModelAlias is empty, it will auto-detect any running embedding model
						embeddingCfg := gitlabRAG.DynamicEmbeddingConfig{
							ModelAlias: cfg.GitLab.RAG.EmbeddingModelAlias, // Empty = auto-detect
							Timeout:    60 * time.Second,
						}
						// Use inference router as model instance provider
						var embedder gitlabRAG.EmbeddingProvider
						if r.inferenceRouter != nil {
							embedder = gitlabRAG.NewDynamicEmbeddingProvider(embeddingCfg, r.inferenceRouter, gitlabLogger)
							if cfg.GitLab.RAG.EmbeddingModelAlias != "" {
								gitlabLogger.WithField("model_alias", cfg.GitLab.RAG.EmbeddingModelAlias).Info("Using dynamic embedding provider (explicit model)")
							} else {
								gitlabLogger.Info("Using dynamic embedding provider (auto-detect any running embedding model)")
							}
						} else {
							gitlabLogger.Warn("Inference router not available, RAG embedding will not work")
						}

						// Get vector dimensions from main RAG config
						vectorSize := cfg.RAG.VectorStore.Dimensions
						if vectorSize == 0 {
							vectorSize = cfg.RAG.Embeddings.Dimensions
						}
						if vectorSize == 0 {
							vectorSize = 768 // Default for most embedding models (BERT-based)
						}

						// Collection name: GitLab-specific or default
						collection := cfg.GitLab.RAG.CollectionName
						if collection == "" {
							collection = "gitlab_code_embeddings"
						}

						ragCfg := gitlabRAG.RAGConfig{
							Enabled: true,
							Qdrant: gitlabRAG.QdrantConfig{
								URL:        qdrantURL,
								APIKey:     "", // Use main RAG config if needed
								Collection: collection,
								VectorSize: vectorSize,
								Timeout:    30 * time.Second,
								Enabled:    true,
							},
						}

						if embedder != nil {
							ragService = gitlabRAG.NewRAGService(ragCfg, embedder, gitlabLogger)
							if err := ragService.Initialize(context.Background()); err != nil {
								gitlabLogger.WithError(err).Warn("Failed to initialize RAG, continuing without it")
								ragService = nil
							} else {
								gitlabLogger.WithFields(logrus.Fields{
									"qdrant_url":  qdrantURL,
									"collection":  collection,
									"vector_size": vectorSize,
								}).Info("✅ RAG service initialized for GitLab")
							}
						}
					}
				}

				// Create processor and worker pool
				processorCfg := gitlabProcessor.ProcessorConfig{
					LLMBaseURL: fmt.Sprintf("http://localhost:%d", cfg.Server.Port), // Use self as LLM endpoint
					LLMAPIKey:  r.gitlabAPIKey,                                      // Auto-generated API key
					Timeout:    10 * time.Minute,
				}
				processor := gitlabProcessor.NewProcessor(glStore, ragService, processorCfg, gitlabLogger)

				poolCfg := gitlabWorker.DefaultPoolConfig()
				if cfg.GitLab.Workers > 0 {
					poolCfg.WorkerCount = cfg.GitLab.Workers
				}
				r.gitlabWorkerPool = gitlabWorker.NewPool(glStore, processor, gitlabLogger, poolCfg)

				// Connect worker pool to handler for stats
				r.gitlabHandler.SetWorkerPool(r.gitlabWorkerPool)

				// Start worker pool
				r.gitlabWorkerPool.Start()
				gitlabLogger.WithField("workers", poolCfg.WorkerCount).Info("✅ GitLab worker pool started")

				// Initialize repository indexer for RAG with dedicated log file
				if ragService != nil {
					// Connect RAG service to handler for index statistics
					r.gitlabHandler.SetQdrantStats(ragService)

					// Create dedicated logger for indexer with log rotation
					indexerLogger := internalLogger.NewFileLogger("logs/gitlab-indexer.log", cfg.Logging.Level)
					if indexerLogger == nil {
						indexerLogger = gitlabLogger // Fallback to gitlab logger
						gitlabLogger.Warn("Failed to create indexer logger, using gitlab logger")
					}

					r.gitlabIndexer = gitlabIndexer.NewIndexer(ragService, indexerLogger)
					r.gitlabIndexer.SetStore(glStore) // Enable DB persistence for index status

					// Enable Redis for fast index status updates (v4.1.0+)
					if r.redisManager != nil && r.redisManager.Client != nil {
						redisStatusStore := gitlabIndexer.NewRedisStatusStore(r.redisManager.Client, indexerLogger)
						r.gitlabIndexer.SetRedisStore(redisStatusStore)
					}
					r.gitlabIndexerHandler = handlers.NewGitLabIndexerHandler(r.gitlabIndexer, glStore, indexerLogger)
					r.gitlabUserIndexerHandler = handlers.NewGitLabUserIndexerHandler(r.gitlabIndexer, glStore, indexerLogger)

					// Set onMerge callback for webhook handler to trigger reindexing
					if r.gitlabWebhookHandler != nil {
						r.gitlabWebhookHandler.SetOnMergeCallback(r.gitlabIndexerHandler.HandleMergeEvent)
					}

					gitlabLogger.Info("✅ GitLab repository indexer initialized for RAG (logs: logs/gitlab-indexer.log)")

					// Initialize dependencies handler with vector store for changelog analysis
					if qdrantStore, ok := r.vectorStore.(*vector.QdrantStore); ok && qdrantStore != nil {
						llmURL := fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
						r.gitlabDependenciesHandler = handlers.NewGitLabDependenciesHandler(
							glStore,
							qdrantStore,
							llmURL,         // Self-reference to internal API
							r.gitlabAPIKey, // Use same API key as MR Review workers
							gitlabLogger,
						)
						gitlabLogger.Info("✅ GitLab dependencies handler initialized")

						// Initialize scheduled scans handler and scheduler
						// PostgresStore implements schedule.ScheduleStore interface
						var scheduleStore schedule.ScheduleStore = glStore
						r.gitlabScheduler = schedule.NewScheduler(
							glStore,
							scheduleStore,
							qdrantStore,
							llmURL,         // Self-reference to internal API (reuse from dependencies handler)
							r.gitlabAPIKey, // Use same API key as MR Review workers
							gitlabLogger,
						)
						if r.gitlabScheduler != nil {
							r.gitlabScheduleHandler = handlers.NewGitLabScheduleHandler(
								r.gitlabScheduler,
								scheduleStore,
								gitlabLogger,
							)
							gitlabLogger.Info("✅ GitLab scheduled scans handler initialized")

							// Start scheduler in background
							go func() {
								ctx := context.Background()
								if err := r.gitlabScheduler.Start(ctx); err != nil {
									gitlabLogger.WithError(err).Error("Failed to start dependency scan scheduler")
								} else {
									gitlabLogger.Info("✅ GitLab dependency scan scheduler started")
								}
							}()
						}

						// Initialize background jobs service (v4.8.9+)
						gitlabLogger.Info("Initializing GitLab background jobs service...")
						r.gitlabJobService = gitlabJobs.NewService(glStore, gitlabLogger, gitlabJobs.DefaultConfig())

						// Register executors for all scan types
						r.gitlabJobService.RegisterExecutor(
							models.JobTypeDeepSecretsScan,
							gitlabJobs.NewDeepScanExecutor(glStore, qdrantStore, llmURL, r.gitlabAPIKey, gitlabLogger),
						)
						r.gitlabJobService.RegisterExecutor(
							models.JobTypeSecretsScn,
							gitlabJobs.NewSecretsScanExecutor(glStore, qdrantStore, gitlabLogger),
						)
						r.gitlabJobService.RegisterExecutor(
							models.JobTypeSASTScan,
							gitlabJobs.NewSASTScanExecutor(glStore, qdrantStore, gitlabLogger),
						)
						r.gitlabJobService.RegisterExecutor(
							models.JobTypeQualityScan,
							gitlabJobs.NewQualityScanExecutor(glStore, qdrantStore, llmURL, r.gitlabAPIKey, gitlabLogger),
						)
						r.gitlabJobService.RegisterExecutor(
							models.JobTypeDependencyScan,
							gitlabJobs.NewDependencyScanExecutor(glStore, qdrantStore, gitlabLogger),
						)
						r.gitlabJobService.RegisterExecutor(
							models.JobTypeDeadCodeScan,
							gitlabJobs.NewDeadCodeScanExecutor(glStore, qdrantStore, llmURL, r.gitlabAPIKey, gitlabLogger),
						)

						// Start job service in background
						go func() {
							r.gitlabJobService.Start()
							gitlabLogger.Info("✅ GitLab background jobs service started")
						}()

						gitlabLogger.Info("✅ GitLab background jobs service initialized with executors")
					}
				}

				// Note: Webhook route registered in setupGitLabRoutes (after engine is created)
				gitlabLogger.Info("✅ GitLab Integration handler initialized")
				logger.Info("✅ GitLab Integration handler initialized (logs: logs/gitlab.log)")
			} else {
				logger.Warn("GitLab Integration disabled: cannot access SQL DB")
			}
		} else {
			logger.Warn("GitLab Integration disabled: database does not support GetDB()")
		}
	} else if cfg.GitLab.Enabled {
		logger.Warn("GitLab Integration enabled but database not available")
	}
}

// setupFileStorage инициализирует file storage и extractors (v1.10.0)
func (r *Router) setupFileStorage(cfg *config.Config, logger *logrus.Logger) {
	// Создаем storage backend
	storageBackend, err := filestorageBackend.NewStorageBackend(cfg.FileStorage)
	if err != nil {
		logger.WithError(err).Warn("Failed to create file storage backend")
		return
	}

	// Создаем extractor registry
	extractorRegistry, err := extractors.NewExtractorRegistry(cfg.Extractors, logger)
	if err != nil {
		logger.WithError(err).Warn("Failed to create extractor registry")
		return
	}

	// Создаем file service
	fileService := filestorage.NewService(storageBackend, cfg.FileStorage, logger)

	// Создаем file handler
	r.fileHandler = handlers.NewFileHandler(fileService, extractorRegistry, r.db, logger)

	logger.WithFields(logrus.Fields{
		"backend":       cfg.FileStorage.Backend,
		"extractors":    len(extractorRegistry.SupportedTypes()),
		"allowed_types": len(cfg.FileStorage.Local.AllowedExts),
	}).Info("File storage initialized successfully")
}

// Shutdown gracefully останавливает все компоненты роутера
// GetMetricsCollector возвращает Prometheus metrics collector
func (r *Router) GetMetricsCollector() *metrics.MetricsCollector {
	return r.metricsCollector
}

func (r *Router) Shutdown(ctx context.Context) error {
	r.logger.Info("Shutting down router components")

	// Останавливаем Prometheus Metrics Collector (v1.11.6+)
	if r.metricsCollector != nil {
		r.metricsCollector.Stop()
		r.logger.Info("Prometheus metrics collector stopped")
	}

	// Останавливаем Metrics Broadcaster
	if r.metricsBroadcaster != nil {
		r.metricsBroadcaster.Stop()
	}

	// Останавливаем WebSocket Hub
	if r.wsHub != nil {
		r.wsHub.Stop()
	}

	// Останавливаем Metrics Storage
	if r.metricsStorage != nil {
		r.metricsStorage.Stop()
	}

	// Останавливаем Request Storage
	if r.requestStorage != nil {
		r.requestStorage.Stop()
	}

	if r.inferenceSvc != nil {
		r.inferenceSvc.Shutdown()
		r.logger.Info("Inference service stopped")
	}

	// Stop GitLab worker pool
	if r.gitlabWorkerPool != nil {
		r.gitlabWorkerPool.Stop(30 * time.Second)
		r.logger.Info("GitLab worker pool stopped")
	}

	// Stop GitLab background jobs service (v4.8.9+)
	if r.gitlabJobService != nil {
		r.gitlabJobService.Stop()
		r.logger.Info("GitLab background jobs service stopped")
	}

	r.logger.Info("All router components stopped")
	return nil
}

// getOrCreateGitLabAPIKey returns existing or creates new API key for GitLab workers
func (r *Router) getOrCreateGitLabAPIKey(parentCtx context.Context, logger *logrus.Logger) string {
	const gitlabKeyName = "GitlabJobsApiKey"

	logger.Info("🔑 Starting GitLab API key creation/retrieval...")

	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	if r.db == nil {
		logger.Warn("Database not available, cannot create GitLab API key")
		return ""
	}

	// List all API keys and find by name
	keys, err := r.db.ListAPIKeys(ctx)
	if err != nil {
		logger.WithError(err).Error("Failed to list API keys")
		return ""
	}

	logger.WithField("total_keys", len(keys)).Debug("Listed existing API keys")

	// Find existing key - we need to regenerate since we can't recover plain key from hash
	for _, key := range keys {
		if key.Name == gitlabKeyName {
			// Delete and recreate to get a new plain key
			logger.WithField("key_id", key.ID).Info("Found existing GitLab API key, regenerating")
			if err := r.db.DeleteAPIKey(ctx, key.ID); err != nil {
				logger.WithError(err).Warn("Failed to delete old GitLab API key")
			}
			break
		}
	}

	// Generate new key
	plainKey := generateSecureAPIKey()
	keyHash, err := models.HashAPIKey(plainKey)
	if err != nil {
		logger.WithError(err).Error("Failed to hash API key")
		return ""
	}

	// Create new key owned by system user
	systemUserID := "system" // Created by migration 093
	apiKey := &models.APIKey{
		ID:          "gitlab", // Must match extracted keyID from sk-proj-gitlab-<random>
		Name:        gitlabKeyName,
		Description: "Internal API key for GitLab MR analysis workers (auto-generated)",
		KeyHash:     keyHash,
		KeyPrefix:   plainKey[:12] + "...",
		UserID:      &systemUserID,                            // Owned by system user
		TenantID:    nil,                                      // No tenant binding
		Scope:       models.APIKeyScopePersonal,               // Personal key of system user
		Status:      models.APIKeyStatusActive,                // Must be active!
		Models:      []string{"*"},                            // Access to all models
		Permissions: []string{"chat", "models", "embeddings"}, // Required permissions
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := r.db.CreateAPIKey(ctx, apiKey); err != nil {
		logger.WithError(err).Error("❌ Failed to create GitLab API key - check if system user exists (migration 093)")
		// Also log to main logger
		r.logger.WithError(err).Error("❌ Failed to create GitLab API key - check if system user exists (migration 093)")
		return ""
	}

	logger.WithFields(logrus.Fields{
		"key_id":     apiKey.ID,
		"key_name":   apiKey.Name,
		"key_prefix": apiKey.KeyPrefix,
	}).Info("✅ Created GitLab API key for workers")
	r.logger.WithField("key_id", apiKey.ID).Info("✅ Created GitLab API key for workers")

	return plainKey
}

// generateSecureAPIKey generates a secure random API key
// Format: sk-proj-<keyid>-<random> where keyid=gitlab and random is hex
// bcrypt has a 72 byte limit, so we use 16 random bytes = 32 hex chars
// Total: "sk-proj-gitlab-" (15) + 32 = 47 bytes (well under 72)
func generateSecureAPIKey() string {
	b := make([]byte, 16) // 16 bytes = 32 hex characters
	if _, err := rand.Read(b); err != nil {
		// Fallback to less secure but working method
		return fmt.Sprintf("sk-proj-gitlab-%d", time.Now().UnixNano())
	}
	return "sk-proj-gitlab-" + hex.EncodeToString(b)
}

// setupMCPRoutes настраивает MCP servers catalog endpoints (v1.4.5)
func (r *Router) setupMCPRoutes() {
	if r.mcpHandler == nil {
		r.logger.Warn("MCP handler not initialized, skipping MCP routes")
		return
	}

	r.logger.Info("Setting up MCP routes")

	// Public routes (no auth required)
	publicMCP := r.engine.Group("/api/mcp")
	{
		publicMCP.GET("/servers", r.mcpHandler.ListMCPServers)
		publicMCP.GET("/servers/:id", r.mcpHandler.GetMCPServer)
		publicMCP.GET("/categories", r.mcpHandler.GetMCPCategories)
	}

	// Admin routes (requires admin auth)
	adminMCP := r.engine.Group("/api/admin/mcp")

	// Use JWT authentication for admin routes if available
	if r.jwtManager != nil && r.db != nil {
		r.logger.Info("MCP admin routes: Using JWT authentication with admin role check")
		adminMCP.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		adminMCP.Use(middleware.RequireAdmin(r.db, r.logger))
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		// Fallback to API Key auth (legacy mode)
		r.logger.Info("MCP admin routes: Using API Key authentication (legacy)")
		adminMCP.Use(middleware.APIKeyDBAuth(r.config, r.db, r.logger))
	} else {
		r.logger.Warn("MCP admin routes: No authentication configured!")
	}

	{
		adminMCP.GET("/servers", r.mcpHandler.ListMCPServers)   // Admin can see ALL servers (including inactive)
		adminMCP.GET("/servers/:id", r.mcpHandler.GetMCPServer) // Get single server details
		adminMCP.POST("/servers", r.mcpHandler.CreateMCPServer)
		adminMCP.PUT("/servers/:id", r.mcpHandler.UpdateMCPServer)
		adminMCP.DELETE("/servers/:id", r.mcpHandler.DeleteMCPServer)
	}

	r.logger.Info("MCP routes configured successfully")
}

// ========================================
// Helper Functions
// ========================================

// mapAgentEventToWebSocketType maps AgentEventType to websocket.EventType (AGENT-05, v2.5.0+)
func mapAgentEventToWebSocketType(agentEventType models.AgentEventType) websocket.EventType {
	switch agentEventType {
	case models.AgentEventSessionCreated:
		return websocket.EventTypeAgentSessionCreated
	case models.AgentEventPlanningStarted:
		return websocket.EventTypeAgentPlanningStarted
	case models.AgentEventPlanningCompleted:
		return websocket.EventTypeAgentPlanningCompleted
	case models.AgentEventExecutionStarted:
		return websocket.EventTypeAgentExecutionStarted
	case models.AgentEventStepStarted:
		return websocket.EventTypeAgentStepStarted
	case models.AgentEventStepCompleted:
		return websocket.EventTypeAgentStepCompleted
	case models.AgentEventStepFailed:
		return websocket.EventTypeAgentStepFailed
	case models.AgentEventApprovalNeeded:
		return websocket.EventTypeAgentApprovalNeeded
	case models.AgentEventApprovalResponded:
		return websocket.EventTypeAgentApprovalResponded
	case models.AgentEventSessionCompleted:
		return websocket.EventTypeAgentSessionCompleted
	case models.AgentEventSessionFailed:
		return websocket.EventTypeAgentSessionFailed
	case models.AgentEventSessionCancelled:
		return websocket.EventTypeAgentSessionCancelled
	case models.AgentEventProgressUpdate:
		return websocket.EventTypeAgentProgressUpdate
	default:
		// Unknown event type - не транслируем
		return ""
	}
}

// setupK8sProbes настраивает Kubernetes liveness/readiness probes (v3.0.8+)
func (r *Router) setupK8sProbes() {
	// Initialize health checker
	r.healthChecker = health.NewHealthChecker(r.logger)

	// Register probes for critical dependencies

	// Database probe
	if r.db != nil {
		r.healthChecker.RegisterProbe("database", func(ctx context.Context) error {
			// Check DB availability with simple query
			_, err := r.db.GetUser(ctx, "health-check-probe")
			// Ignore "not found" error - DB is responsive
			if err != nil && !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "no rows") {
				return err
			}
			return nil
		})
	}

	// Redis probe
	if r.redisManager != nil {
		r.healthChecker.RegisterProbe("redis", func(ctx context.Context) error {
			// Simple ping via cache set/get
			testKey := "health:check"
			err := r.redisManager.Cache.SetJSON(ctx, testKey, "ok", 1*time.Second)
			if err != nil {
				return err
			}
			var result string
			return r.redisManager.Cache.GetJSON(ctx, testKey, &result)
		})
	}

	// Initialize probes
	r.livenessProbe = health.NewLivenessProbe()
	r.readinessProbe = health.NewReadinessProbe(r.healthChecker)

	// K8s liveness probe: /healthz/live
	r.engine.GET("/healthz/live", func(c *gin.Context) {
		if err := r.livenessProbe.Check(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"uptime": r.livenessProbe.GetUptime().String(),
		})
	})

	// K8s readiness probe: /healthz/ready
	r.engine.GET("/healthz/ready", func(c *gin.Context) {
		if err := r.readinessProbe.Check(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not_ready",
				"error":  err.Error(),
			})
			return
		}

		results := r.healthChecker.CheckAll(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{
			"status":  "ready",
			"probes":  results,
			"version": r.version,
		})
	})

	// Detailed health status: /healthz/status
	r.engine.GET("/healthz/status", func(c *gin.Context) {
		results := r.healthChecker.CheckAll(c.Request.Context())

		allHealthy := true
		for _, result := range results {
			if result.Status != "healthy" {
				allHealthy = false
				break
			}
		}

		statusCode := http.StatusOK
		if !allHealthy {
			statusCode = http.StatusServiceUnavailable
		}

		c.JSON(statusCode, gin.H{
			"status":      gin.H{"overall": allHealthy, "uptime": r.livenessProbe.GetUptime().String()},
			"probes":      results,
			"check_count": r.healthChecker.GetCheckCount(),
			"version":     r.version,
		})
	})

	r.logger.Info("✅ Kubernetes health probes configured: /healthz/live, /healthz/ready, /healthz/status")
}

// setupGitLabRoutes настраивает routes для GitLab Integration
// Использует реальный handler если gitlab.enabled=true, иначе stub routes
func (r *Router) setupGitLabRoutes() {
	// Admin GitLab routes
	adminGitlab := r.engine.Group("/api/admin/gitlab")
	if r.jwtManager != nil && r.db != nil {
		adminGitlab.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		adminGitlab.Use(middleware.RequireAdmin(r.db, r.logger))
	}

	// If gitlabHandler is initialized, use real handlers
	if r.gitlabHandler != nil {
		// Integrations
		adminGitlab.GET("/integrations", r.gitlabHandler.ListIntegrations)
		adminGitlab.POST("/integrations", r.gitlabHandler.CreateIntegration)
		adminGitlab.GET("/integrations/:id", r.gitlabHandler.GetIntegration)
		adminGitlab.PUT("/integrations/:id", r.gitlabHandler.UpdateIntegration)
		adminGitlab.DELETE("/integrations/:id", r.gitlabHandler.DeleteIntegration)
		adminGitlab.POST("/integrations/:id/test", r.gitlabHandler.TestIntegration)

		// Projects
		adminGitlab.GET("/integrations/:id/projects", r.gitlabHandler.ListProjects)
		adminGitlab.POST("/integrations/:id/projects", r.gitlabHandler.AddProject)
		adminGitlab.GET("/projects/:id", r.gitlabHandler.GetProject)
		adminGitlab.PUT("/projects/:id", r.gitlabHandler.UpdateProject)
		adminGitlab.DELETE("/projects/:id", r.gitlabHandler.DeleteProject)
		adminGitlab.POST("/projects/:id/webhook", r.gitlabHandler.SetupWebhook)

		// Project Indexing (RAG)
		if r.gitlabIndexerHandler != nil {
			adminGitlab.POST("/projects/:id/index", r.gitlabIndexerHandler.IndexProject)
			adminGitlab.GET("/projects/:id/index/status", r.gitlabIndexerHandler.GetIndexStatus)
			adminGitlab.DELETE("/projects/:id/index", r.gitlabIndexerHandler.DeleteIndex)
		}

		// Dependencies scanning and changelog analysis (handlers use c.Param("id"))
		if r.gitlabDependenciesHandler != nil {
			adminGitlab.POST("/projects/:id/check-dependencies", r.gitlabDependenciesHandler.CheckDependencies)
			adminGitlab.POST("/projects/:id/create-dependency-issue", r.gitlabDependenciesHandler.CreateDependencyIssue)
			adminGitlab.POST("/projects/:id/analyze-changelog", r.gitlabDependenciesHandler.AnalyzeChangelog)
			adminGitlab.POST("/projects/:id/analyze-changelogs", r.gitlabDependenciesHandler.AnalyzeDependenciesChangelogs)
			r.logger.Info("✅ GitLab dependencies routes registered")
		}

		// Scheduled scans management
		if r.gitlabScheduleHandler != nil {
			adminGitlab.POST("/schedules", r.gitlabScheduleHandler.CreateSchedule)
			adminGitlab.GET("/schedules", r.gitlabScheduleHandler.ListSchedules)
			adminGitlab.GET("/schedules/status", r.gitlabScheduleHandler.GetSchedulerStatus)
			adminGitlab.GET("/schedules/:id", r.gitlabScheduleHandler.GetSchedule)
			adminGitlab.PUT("/schedules/:id", r.gitlabScheduleHandler.UpdateSchedule)
			adminGitlab.DELETE("/schedules/:id", r.gitlabScheduleHandler.DeleteSchedule)
			adminGitlab.POST("/schedules/:id/trigger", r.gitlabScheduleHandler.TriggerSchedule)
			adminGitlab.GET("/schedules/:id/history", r.gitlabScheduleHandler.GetScheduleHistory)
			r.logger.Info("✅ GitLab scheduled scans routes registered")
		}

		// Security, Quality, Dead Code, Auto-Doc, Test Gen routes (v4.1.0+)
		// These require Qdrant vector store and GitLab store
		if qdrantStore, ok := r.vectorStore.(*vector.QdrantStore); ok && qdrantStore != nil && r.gitlabStore != nil {
			glStore := r.gitlabStore
			llmBaseURL := fmt.Sprintf("http://localhost:%d", r.config.Server.Port)
			llmAPIKey := r.gitlabAPIKey // Use same API key as MR Review workers

			// Validate LLM configuration
			if llmAPIKey == "" {
				r.logger.Warn("⚠️ GitLab LLM API key not configured - deep scan and LLM analysis will fail")
			}
			if r.config.Server.Port == 0 {
				r.logger.Warn("⚠️ Server port not configured - LLM requests may fail")
			}

			// Secrets scanning (handlers use c.Param("id"))
			secretsHandler := handlers.NewGitLabSecretsHandler(glStore, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
			adminGitlab.POST("/projects/:id/scan-secrets", secretsHandler.ScanSecrets)
			adminGitlab.POST("/projects/:id/deep-scan-secrets", secretsHandler.DeepScanSecrets)
			adminGitlab.POST("/projects/:id/sast-scan", secretsHandler.SASTScan)
			adminGitlab.GET("/secrets/patterns", secretsHandler.GetPatterns)
			adminGitlab.POST("/projects/:id/secrets/create-issue", secretsHandler.CreateSecretsIssue)
			r.logger.Info("✅ GitLab secrets scanning routes registered")

			// Code quality analysis (handlers use c.Param("id"))
			qualityHandler := handlers.NewGitLabQualityHandler(glStore, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
			adminGitlab.POST("/projects/:id/quality-score", qualityHandler.AnalyzeQuality)
			adminGitlab.POST("/projects/:id/detect-duplication", qualityHandler.DetectDuplication)
			adminGitlab.POST("/projects/:id/quality/create-issue", qualityHandler.CreateQualityIssue)
			r.logger.Info("✅ GitLab quality analysis routes registered")

			// Dead code detection (handlers use c.Param("id"))
			deadCodeHandler := handlers.NewGitLabDeadCodeHandler(glStore, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
			adminGitlab.POST("/projects/:id/dead-code", deadCodeHandler.DetectDeadCode)
			adminGitlab.POST("/projects/:id/unreachable-code", deadCodeHandler.DetectUnreachable)
			adminGitlab.POST("/projects/:id/commented-code", deadCodeHandler.DetectCommentedCode)
			adminGitlab.POST("/projects/:id/dead-code-issue", deadCodeHandler.CreateDeadCodeIssue)
			r.logger.Info("✅ GitLab dead code detection routes registered")

			// Auto-documentation (handlers use c.Param("id"))
			autoDocHandler := handlers.NewGitLabAutoDocHandler(glStore, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
			adminGitlab.POST("/projects/:id/scan-undocumented", autoDocHandler.ScanUndocumented)
			adminGitlab.POST("/projects/:id/scan-docs", autoDocHandler.ScanUndocumented) // Alias for frontend
			adminGitlab.POST("/projects/:id/generate-docs", autoDocHandler.GenerateDocs)
			adminGitlab.POST("/projects/:id/bulk-apply-docs", autoDocHandler.BulkApplyDocs)
			adminGitlab.POST("/projects/:id/create-docs-mr", autoDocHandler.CreateDocsMR)
			r.logger.Info("✅ GitLab auto-documentation routes registered")

			// Test generation (handlers use c.Param("id"))
			testGenHandler := handlers.NewGitLabTestGenHandler(glStore, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
			adminGitlab.POST("/projects/:id/scan-testable", testGenHandler.ScanTestable)
			adminGitlab.POST("/projects/:id/scan-tests", testGenHandler.ScanTestable) // Alias for frontend
			adminGitlab.POST("/projects/:id/generate-tests", testGenHandler.GenerateTests)
			adminGitlab.POST("/projects/:id/download-tests", testGenHandler.DownloadTests)
			adminGitlab.POST("/projects/:id/create-tests-mr", testGenHandler.CreateTestsMR)
			r.logger.Info("✅ GitLab test generation routes registered")

			// Architecture diagrams (handlers use c.Param("id"))
			architectureHandler := handlers.NewGitLabArchitectureHandler(glStore, qdrantStore, llmBaseURL, llmAPIKey, r.logger)
			adminGitlab.POST("/projects/:id/scan-architecture", architectureHandler.ScanArchitecture)
			adminGitlab.POST("/projects/:id/generate-diagram", architectureHandler.GenerateDiagram)
			adminGitlab.GET("/projects/:id/architecture", architectureHandler.GetArchitecture)
			r.logger.Info("✅ GitLab architecture diagram routes registered")
		}

		// Reviews
		adminGitlab.GET("/reviews", r.gitlabHandler.ListReviews)
		adminGitlab.GET("/reviews/:id", r.gitlabHandler.GetReview)
		adminGitlab.POST("/reviews/:id/retry", r.gitlabHandler.RetryReview)

		// Queue
		adminGitlab.GET("/queue/status", r.gitlabHandler.GetQueueStatus)
		adminGitlab.GET("/queue/jobs", r.gitlabHandler.ListJobs)
		adminGitlab.POST("/queue/jobs/:id/cancel", r.gitlabHandler.CancelJob)
		adminGitlab.POST("/queue/jobs/:id/retry", r.gitlabHandler.RetryJob)

		// Models
		adminGitlab.GET("/models", r.gitlabHandler.ListActiveModels)
		adminGitlab.GET("/models/analysis", r.gitlabHandler.ListAnalysisModels)
		adminGitlab.GET("/models/embedding", r.gitlabHandler.ListEmbeddingModels)

		// Settings, Analytics, Feedback (GITLAB-UI)
		adminGitlab.GET("/settings", r.gitlabHandler.GetSettings)
		adminGitlab.PUT("/settings", r.gitlabHandler.UpdateSettings)
		adminGitlab.GET("/analytics", r.gitlabHandler.GetAnalytics)
		adminGitlab.GET("/feedback", r.gitlabHandler.ListFeedback)
		adminGitlab.POST("/feedback", r.gitlabHandler.SubmitFeedback)

		// Scan History (v4.1.2+) - view all scan results with historical data
		scanHistoryHandler := handlers.NewGitLabScanHistoryHandler(r.gitlabStore, r.logger)
		adminGitlab.GET("/scan-history", scanHistoryHandler.ListScanResults)
		adminGitlab.GET("/scan-history/types", scanHistoryHandler.GetScanTypes)
		adminGitlab.GET("/scan-history/:id", scanHistoryHandler.GetScanResult)
		r.logger.Info("✅ GitLab scan history routes registered")

		// Available GitLab projects for selection (GITLAB-AUTO)
		adminGitlab.GET("/integrations/:id/available-projects", r.gitlabHandler.ListAvailableProjects)

		// Webhook route - NO authentication, uses webhook secret verification
		if r.gitlabWebhookHandler != nil {
			r.engine.POST("/api/gitlab/webhook/:integration_id", r.gitlabWebhookHandler.HandleWebhook)
			r.logger.Info("✅ GitLab webhook route registered: POST /api/gitlab/webhook/:integration_id")
		}

		r.logger.Info("✅ GitLab Integration routes configured with real handlers")
	} else {
		// Stub routes when GitLab not configured
		adminGitlab.GET("/integrations", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
		})
		adminGitlab.POST("/integrations", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "GitLab storage not initialized. Configure gitlab section in config."})
		})
		adminGitlab.GET("/integrations/:id", func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Integration not found"})
		})
		adminGitlab.PUT("/integrations/:id", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "GitLab storage not initialized"})
		})
		adminGitlab.DELETE("/integrations/:id", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "GitLab storage not initialized"})
		})
		adminGitlab.POST("/integrations/:id/test", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "GitLab storage not initialized"})
		})

		// Projects stub
		adminGitlab.GET("/integrations/:id/projects", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
		})
		adminGitlab.POST("/integrations/:id/projects", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "GitLab storage not initialized"})
		})
		adminGitlab.GET("/projects/:id", func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		})
		adminGitlab.PUT("/projects/:id", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "GitLab storage not initialized"})
		})
		adminGitlab.DELETE("/projects/:id", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "GitLab storage not initialized"})
		})
		adminGitlab.POST("/projects/:id/webhook", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "GitLab storage not initialized"})
		})

		// Reviews stub
		adminGitlab.GET("/reviews", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
		})
		adminGitlab.GET("/reviews/:id", func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Review not found"})
		})
		adminGitlab.POST("/reviews/:id/retry", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "GitLab storage not initialized"})
		})

		// Queue stub
		adminGitlab.GET("/queue/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"pending":    0,
				"processing": 0,
				"completed":  0,
				"failed":     0,
				"workers":    gin.H{"total": 0, "active": 0, "idle": 0},
			})
		})
		adminGitlab.GET("/queue/jobs", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
		})
		adminGitlab.POST("/queue/jobs/:id/cancel", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "GitLab storage not initialized"})
		})
		adminGitlab.POST("/queue/jobs/:id/retry", func(c *gin.Context) {
			c.JSON(http.StatusNotImplemented, gin.H{"error": "GitLab storage not initialized"})
		})

		// Models stub
		adminGitlab.GET("/models", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
		})
		adminGitlab.GET("/models/analysis", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
		})
		adminGitlab.GET("/models/embedding", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
		})

		// Settings, Analytics, Feedback stubs
		adminGitlab.GET("/settings", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"auto_review_enabled":     true,
				"default_analysis_model":  "",
				"default_embedding_model": "",
				"max_files_per_mr":        50,
				"max_lines_per_file":      1000,
				"webhook_secret_rotation": false,
				"notification_email":      "",
			})
		})
		adminGitlab.PUT("/settings", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Settings updated"})
		})
		adminGitlab.GET("/analytics", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"total_reviews":        0,
				"avg_processing_time":  0,
				"issues_found":         0,
				"reviews_by_day":       []interface{}{},
				"reviews_by_project":   []interface{}{},
				"top_issue_categories": []interface{}{},
			})
		})
		adminGitlab.GET("/feedback", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
		})
		adminGitlab.POST("/feedback", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "Feedback submitted"})
		})
		adminGitlab.GET("/integrations/:id/available-projects", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"projects": []interface{}{}, "total": 0})
		})

		r.logger.Info("✅ GitLab Integration stub routes configured (gitlab.enabled=false)")
	}

	// User-level GitLab routes
	if r.gitlabStore != nil {
		r.SetupGitLabUserRoutes(r.gitlabStore)

		// User background jobs routes (v4.8.9+)
		if r.gitlabJobService != nil {
			r.SetupGitLabUserJobsRoutes(r.gitlabStore, r.gitlabJobService)
		}
	}
}
