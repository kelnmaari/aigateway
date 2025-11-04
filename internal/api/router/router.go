// Package router provides HTTP routing setup for Ollama-OpenAI Proxy
package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"aigateway/internal/api/handlers"
	"aigateway/internal/api/middleware"
	"aigateway/internal/auth/apikey"
	"aigateway/internal/auth/jwt"
	ldapauth "aigateway/internal/auth/ldap"
	authMiddleware "aigateway/internal/auth/middleware"
	oidcauth "aigateway/internal/auth/oidc"
	"aigateway/internal/auth/ratelimit"
	authService "aigateway/internal/auth/service"
	"aigateway/internal/client/ollama"
	"aigateway/internal/config"
	"aigateway/internal/extractors"
	"aigateway/internal/filestorage"
	filestorageBackend "aigateway/internal/filestorage/storage"
	"aigateway/internal/metrics"
	"aigateway/internal/observability"
	"aigateway/internal/request"
	"aigateway/internal/services/audit"
	"aigateway/internal/services/model"
	"aigateway/internal/services/quota"
	ragservice "aigateway/internal/services/rag"
	ragorchestrator "aigateway/internal/rag/orchestrator"
	"aigateway/internal/services/rbac"
	"aigateway/internal/storage"
	"aigateway/internal/websocket"
	"aigateway/internal/providers"

	"go.opentelemetry.io/otel/trace"
)

// Router представляет HTTP роутер приложения с опциональным API Key Management
type Router struct {
	config       *config.Config
	logger       *logrus.Logger
	engine       *gin.Engine
	ollamaClient *ollama.ClientWithCircuitBreaker
	version      string       // Версия сервера
	tracer       trace.Tracer // OpenTelemetry tracer (v1.6.0+)

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

	// Handlers
	healthHandler         *handlers.HealthHandler
	systemHandler         *handlers.SystemHandler       // System endpoints (bootstrap, init-status)
	authHandler           *handlers.AuthHandler         // Auth endpoints (login, register, etc)
	deviceHandler         *handlers.DeviceHandler       // Device management (Version 2.4.0+)
	userHandler           *handlers.UserHandler         // User management
	tenantHandler         *handlers.TenantHandler       // Tenant management
	conversationHandler       *handlers.ConversationHandler       // Conversation management (Version 1.3.0)
	conversationExportHandler *handlers.ConversationExportHandler // Conversation export/import (Version 1.12.3+)
	adminUserHandler          *handlers.AdminUserHandler          // Admin User Management (Version 1.3.0)
	usageHandler              *handlers.UsageHandler              // Usage Statistics (Version 1.3.0)
	modelsHandler         *handlers.ModelsHandler
	modelPreloadHandler   *handlers.ModelPreloadHandler // Model Preload Management (Version 1.12.1+)
	chatHandler           *handlers.ChatHandler
	embeddingsHandler     *handlers.EmbeddingsHandler
	completionsHandler    *handlers.CompletionsHandler
	adminHandler          *handlers.AdminHandler
	adminFilesHandler     *handlers.AdminFilesHandler     // Admin files management (v1.10.0)
	statsHandler          *handlers.StatsHandler          // Handler для TUI статистики
	configHandler         *handlers.ConfigHandler         // Handler для конфигурации
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
	ragDataSourcesHandler *handlers.RAGDataSourcesHandler // Handler для RAG data sources (v1.13.1)
	registryHandler       *handlers.RegistryHandler       // Handler для model registry (REGISTRY-01, v2.3.0)

	// Provider Management (Version 2.3.0+: REGISTRY-01)
	providerManager *providers.ProviderManager // Model providers manager

	// WebSocket components
	wsHub              *websocket.Hub
	wsHandler          *websocket.Handler
	wsChatHandler      *websocket.ChatHandler // Chat handler for desktop (DESKTOP-03)
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

	// Model Preloading (Version 1.12.1+: Model Preloading & Warming)
	modelPreloader *model.ModelPreloader
	
	// RAG System (Version 1.13.0+: RAG System)
	ragDataSourceService *ragservice.DataSourceService
	ragOrchestrator      *ragorchestrator.RAGOrchestrator
}

// NewOptions содержит опции для создания роутера
type NewOptions struct {
	Config               *config.Config
	Logger               *logrus.Logger
	Version              string
	Database             storage.Database                  // Опциональная база данных для user auth
	JWTManager           *jwt.Manager                      // Опциональный JWT manager
	TracerProvider       *observability.TracerProvider     // Опциональный OpenTelemetry tracer (v1.6.0+)
	PerformanceMonitor   *observability.PerformanceMonitor // Опциональный performance monitor (v1.6.2+)
	LeakDetector         *observability.LeakDetector       // Опциональный leak detector (v1.6.2+)
	MonigoPort           int                                  // Порт MoniGo dashboard (0 если отключен) (v1.9.3+)
	GPUMonitor           *metrics.GPUMonitor                  // Опциональный GPU monitor (v1.9.3+)
	ModelPreloader       *model.ModelPreloader                // Опциональный model preloader (v1.12.1+)
	RAGDataSourceService *ragservice.DataSourceService        // Опциональный RAG Data Source Service (v1.13.1+)
	RAGOrchestrator      *ragorchestrator.RAGOrchestrator // Опциональный RAG Orchestrator (v1.13.1+)
}

// New создает новый экземпляр роутера с опциональным API Key Management
func New(cfg *config.Config, logger *logrus.Logger, version string) (*Router, error) {
	return NewWithOptions(NewOptions{
		Config:  cfg,
		Logger:  logger,
		Version: version,
	})
}

// NewWithOptions создает роутер с расширенными опциями
func NewWithOptions(opts NewOptions) (*Router, error) {
	// Создаем Ollama клиент с circuit breaker
	ollamaClient, err := ollama.NewClientWithCircuitBreaker(opts.Config, opts.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	r := &Router{
		config:               opts.Config,
		logger:               opts.Logger,
		ollamaClient:         ollamaClient,
		version:              opts.Version,
		db:                   opts.Database,
		jwtManager:           opts.JWTManager,
		performanceMonitor:   opts.PerformanceMonitor,
		leakDetector:         opts.LeakDetector,
		monigoPort:           opts.MonigoPort,
		gpuMonitor:           opts.GPUMonitor,
		modelPreloader:       opts.ModelPreloader,
		ragDataSourceService: opts.RAGDataSourceService,
		ragOrchestrator:      opts.RAGOrchestrator,
	}

	// Setup tracer if provided
	if opts.TracerProvider != nil {
		r.tracer = opts.TracerProvider.Tracer()
		opts.Logger.Info("OpenTelemetry tracer initialized")
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

	// Инициализация handlers
	r.setupHandlers(opts.Config, opts.Logger, ollamaClient)

	r.setupEngine()
	r.setupRoutes()

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

	r.logger.Info("Router initialized successfully")
	return nil
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

	// Stop model preloader (Version 1.12.1+)
	if r.modelPreloader != nil {
		r.modelPreloader.Stop()
	}

	if r.ollamaClient != nil {
		r.ollamaClient.Close()
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
		sessionSecret = []byte("ollama-proxy-session-secret-change-this-in-production!")
		r.logger.Warn("Using default session secret - please configure a secure JWT secret")
	}
	store := cookie.NewStore(sessionSecret)
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   3600,    // 1 hour
		HttpOnly: true,    // Protect against XSS
		Secure:   false,   // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
	})
	r.engine.Use(sessions.Sessions("ollama_session", store))
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
		r.engine.Use(middleware.MetricsCollector(r.metricsStorage, r.logger))
	}

	// Request Tracker middleware для monitoring (TUI-04)
	if r.requestStorage != nil && r.eventBroadcaster != nil {
		r.engine.Use(middleware.RequestTracker(r.requestStorage, r.eventBroadcaster, r.logger))
	}

	// Structured logging middleware
	loggingConfig := middleware.LoggingConfig{
		Logger:    r.logger,
		SkipPaths: []string{"/health", "/healthz", "/ready", "/api/stats", "/api/config", r.config.Metrics.PrometheusPath}, // Пропускаем health checks, stats, config и metrics
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
	r.setupHealthRoutes()
	r.setupMetricsRoutes()
	r.setupMonigoRoutes()         // MoniGo Performance Dashboard (v1.9.3+)
	r.setupStatsRoutes()          // Для TUI
	r.setupConfigRoutes()         // Для TUI Configuration Viewer
	r.setupRAGRoutes()            // RAG System (v1.13.1+)
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
	r.setupInvitationsRoutes()    // Invitation System (AUTH-03, v2.2.0)
	r.setupMCPRoutes()  // MCP Servers Catalog (v1.4.5)
	r.setupGPURoutes()  // GPU Monitoring (v1.9.3)
	r.setupFileRoutes() // File Storage & Processing (v1.10.0)
}

// setupHealthRoutes настраивает эндпоинты проверки здоровья
func (r *Router) setupHealthRoutes() {
	r.engine.GET("/health", r.healthHandler.Health)
	r.engine.GET("/ready", r.healthHandler.Ready)
	r.engine.GET("/healthz", r.healthHandler.Live)
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
	r.engine.GET("/api/stats", r.statsHandler.GetStats)
}

// setupConfigRoutes настраивает эндпоинты конфигурации для TUI
func (r *Router) setupConfigRoutes() {
	// Эндпоинт для TUI (без аутентификации, только для локального использования)
	r.engine.GET("/api/config", r.configHandler.GetConfig)

	// Публичные эндпоинты для списка моделей (для WebUI/TUI/Login page)
	r.engine.GET("/api/models", r.modelsHandler.List)
	r.engine.GET("/api/v1/models", r.modelsHandler.List) // OpenAI-compatible path
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
	}

	r.logger.Info("System API endpoints configured")
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
			conversations.GET("/:id/export", r.conversationExportHandler.ExportConversation)           // Export conversation
			conversations.POST("/import", r.conversationExportHandler.ImportConversation)              // Import conversation
			conversations.POST("/bulk-export", r.conversationExportHandler.BulkExportConversations)  // Bulk export
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

// setupWebUIRoutes настраивает static file serving для WebUI (Version 1.3.0+)
func (r *Router) setupWebUIRoutes() {
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
	r.engine.StaticFile("/files.html", "./web/files.html") // Files Management (v1.10.0)
	r.engine.StaticFile("/rag-sources.html", "./web/rag-sources.html") // RAG Data Sources (v1.13.0)
	r.engine.StaticFile("/usage.html", "./web/usage.html")
	r.engine.StaticFile("/mcp.html", "./web/mcp.html")     // MCP Catalog (v1.4.5)
	r.engine.StaticFile("/about.html", "./web/about.html") // About System (v1.4.11)
	r.engine.StaticFile("/admin.html", "./web/admin.html") // Admin Panel (Version 1.3.0)
	r.engine.StaticFile("/admin-invitations.html", "./web/admin-invitations.html") // Invitations Management (AUTH-03, v2.2.0)
	r.engine.StaticFile("/admin-rbac.html", "./web/admin-rbac.html") // RBAC Management (v1.11.5)
	r.engine.StaticFile("/admin-audit.html", "./web/admin-audit.html") // Audit Log (v1.11.4)
	r.engine.StaticFile("/admin-rag.html", "./web/admin-rag.html") // RAG Management (v1.13.0)
	r.engine.StaticFile("/admin-registry.html", "./web/admin-registry.html") // Model Registry (REGISTRY-03, v2.3.0)

	// Serve CSS and JS directories
	r.engine.Static("/css", "./web/css")
	r.engine.Static("/js", "./web/js")
	r.engine.Static("/assets", "./web/assets")
	r.engine.Static("/images", "./web/images")

	// Legacy /web/* routes for backward compatibility
	r.engine.Static("/web", "./web")

	r.logger.Info("WebUI static file serving configured at root path")
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

		v1.GET("/models", r.modelsHandler.List)
		v1.POST("/chat/completions", r.chatHandler.Completion)

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

		v1.GET("/models", r.authenticator.PermissionMiddleware("models"), r.modelsHandler.List)
		v1.POST("/chat/completions", r.authenticator.PermissionMiddleware("chat"), r.chatHandler.Completion)
	} else {
		// Открытые эндпоинты (MVP mode без аутентификации)
		r.logger.Info("Using NO authentication for /v1 endpoints (MVP mode)")

		// Usage tracking даже без auth для мониторинга (Version 1.4.6+)
		if r.db != nil {
			v1.Use(middleware.UsageTracking(r.db, r.logger))
			r.logger.Info("Usage tracking enabled for /v1 endpoints (No auth mode)")
		}

		v1.GET("/models", r.modelsHandler.List)
		v1.POST("/chat/completions", r.chatHandler.Completion)
	}

	// Embeddings endpoint
	if r.jwtManager != nil && r.authenticator != nil && r.config.Auth.Enabled {
		// Hybrid auth уже применен к v1 группе
		v1.POST("/embeddings", r.embeddingsHandler.HandleEmbeddings)
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		v1.POST("/embeddings", r.authenticator.PermissionMiddleware("embeddings"), r.embeddingsHandler.HandleEmbeddings)
	} else {
		v1.POST("/embeddings", r.embeddingsHandler.HandleEmbeddings)
	}

	// Legacy text completions
	if r.jwtManager != nil && r.authenticator != nil && r.config.Auth.Enabled {
		// Hybrid auth уже применен к v1 группе
		v1.POST("/completions", r.completionsHandler.HandleCompletions)
	} else if r.config.Auth.Enabled && r.authenticator != nil {
		v1.POST("/completions", r.authenticator.PermissionMiddleware("completions"), r.completionsHandler.HandleCompletions)
	} else {
		v1.POST("/completions", r.completionsHandler.HandleCompletions)
	}
}

// setupAdminRoutes настраивает административные API эндпоинты
func (r *Router) setupAdminRoutes() {
	r.logger.Info("Setting up admin routes")
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

		// Models management (v1.4.4)
		admin.GET("/models/:name/details", r.adminHandler.GetModelDetails)
	}

	// Model Preloading endpoints (v1.12.1+)
	if r.modelPreloadHandler != nil {
		r.logger.Info("Admin routes: Registering Model Preloading endpoints")
		admin.GET("/models/loaded", r.modelPreloadHandler.GetLoadedModels)
		admin.POST("/models/:name/preload", r.modelPreloadHandler.PreloadModel)
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

	// Models list for admin
	admin.GET("/models", r.modelsHandler.List)

	// Config viewer
	admin.GET("/config", r.configHandler.GetConfig)

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

	// Создаем Authenticator
	authenticator := authMiddleware.NewAPIKeyAuthenticator(cfg, logger, keyManager)

	// Сохраняем компоненты
	r.storage = stor
	r.keyManager = keyManager
	r.rateLimiter = rateLimiter
	r.authenticator = authenticator

	logger.Info("API Key Management setup completed")
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
		adminInvitations.POST("", r.invitationHandler.CreateInvitation)            // Create invitation
		adminInvitations.GET("", r.invitationHandler.ListInvitations)              // List invitations
		adminInvitations.GET("/stats", r.invitationHandler.GetInvitationStats)     // Get statistics
		adminInvitations.GET("/:id", r.invitationHandler.GetInvitationDetails)     // Get invitation details with user info
		adminInvitations.DELETE("/:id", r.invitationHandler.RevokeInvitation)      // Revoke invitation
	}

	r.logger.Info("Invitation routes configured successfully")
}

// setupHandlers инициализирует все handlers
func (r *Router) setupHandlers(cfg *config.Config, logger *logrus.Logger, ollamaClient *ollama.ClientWithCircuitBreaker) {
	logger.WithField("db_is_nil", r.db == nil).Info("DEBUG: setupHandlers called")

	r.healthHandler = handlers.NewHealthHandler(cfg, logger, ollamaClient)
	r.modelsHandler = handlers.NewModelsHandler(cfg, logger, ollamaClient)

	// Model Preload handler (v1.12.1+)
	if r.modelPreloader != nil {
		r.modelPreloadHandler = handlers.NewModelPreloadHandler(logger, r.modelPreloader)
	}
	
	// RAG Data Sources handler (v1.13.1+)
	if r.ragDataSourceService != nil {
		r.ragDataSourcesHandler = handlers.NewRAGDataSourcesHandler(r.ragDataSourceService, logger)
	}

	// Chat handler with database for model configs (v1.9.1+)
	if r.db != nil {
		r.chatHandler = handlers.NewChatHandlerWithDB(cfg, logger, ollamaClient, r.db)
	} else {
		r.chatHandler = handlers.NewChatHandler(cfg, logger, ollamaClient)
	}

	// Setup model preloader для tracking (v1.12.1+)
	if r.modelPreloader != nil {
		r.chatHandler.SetModelPreloader(r.modelPreloader)
		logger.Info("Model preloader attached to chat handler")
	}

	// Setup RAG orchestrator для chat retrieval (v1.13.1+)
	if r.ragOrchestrator != nil {
		r.chatHandler.SetRAGOrchestrator(r.ragOrchestrator)
		logger.Info("RAG Orchestrator attached to chat handler")
	}

	r.embeddingsHandler = handlers.NewEmbeddingsHandler(cfg, logger, ollamaClient)
	r.completionsHandler = handlers.NewCompletionsHandler(cfg, logger, ollamaClient)
	r.configHandler = handlers.NewConfigHandler(cfg, logger) // Для TUI configuration viewer

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
			WithRetentionPeriod(90 * 24 * time.Hour).  // 90 days retention
			WithCleanupInterval(24 * time.Hour)         // Daily cleanup
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

	// Set Ollama client for admin handler (v1.4.4)
	if r.adminHandler != nil && r.ollamaClient != nil {
		r.adminHandler.SetOllamaClient(r.ollamaClient)
		logger.Info("Ollama client set for AdminHandler")
	}

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
	r.statsHandler = handlers.NewStatsHandler(cfg, logger, ollamaClient, r.keyManager, r.version, r.metricsStorage, r.db)

	// Metrics History Handler
	r.metricsHistoryHandler = handlers.NewMetricsHistoryHandler(cfg, logger, r.metricsStorage)

	// Requests Handler (TUI-04)
	r.requestsHandler = handlers.NewRequestsHandler(cfg, logger, r.requestStorage)

	// WebSocket Hub (Phase 12.2)
	r.wsHub = websocket.NewHub(logger)
	go r.wsHub.Run() // Запускаем hub в фоне

	// WebSocket Handler
	r.wsHandler = websocket.NewHandler(r.wsHub, logger)

	// Chat Handler для WebSocket (DESKTOP-03 v2.4.3)
	r.wsChatHandler = websocket.NewChatHandler(cfg, logger, r.ollamaClient, r.wsHub)

	// Настраиваем WebSocket handler (DESKTOP-03)
	r.wsHandler.SetDatabase(r.db)
	r.wsHandler.SetChatHandler(r.wsChatHandler)

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

	r.logger.Info("All router components stopped")
	return nil
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

