// Package router provides HTTP routing setup for Ollama-OpenAI Proxy
package router

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/api/handlers"
	"ollama-openai-proxy/internal/api/middleware"
	"ollama-openai-proxy/internal/auth/apikey"
	"ollama-openai-proxy/internal/auth/jwt"
	authMiddleware "ollama-openai-proxy/internal/auth/middleware"
	"ollama-openai-proxy/internal/auth/ratelimit"
	authService "ollama-openai-proxy/internal/auth/service"
	"ollama-openai-proxy/internal/client/ollama"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/metrics"
	"ollama-openai-proxy/internal/request"
	"ollama-openai-proxy/internal/storage"
	"ollama-openai-proxy/internal/websocket"
)

// Router представляет HTTP роутер приложения с опциональным API Key Management
type Router struct {
	config       *config.Config
	logger       *logrus.Logger
	engine       *gin.Engine
	ollamaClient *ollama.ClientWithCircuitBreaker
	version      string // Версия сервера

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
	userHandler           *handlers.UserHandler         // User management
	tenantHandler         *handlers.TenantHandler       // Tenant management
	conversationHandler   *handlers.ConversationHandler // Conversation management (Version 1.3.0)
	adminUserHandler      *handlers.AdminUserHandler    // Admin User Management (Version 1.3.0)
	usageHandler          *handlers.UsageHandler        // Usage Statistics (Version 1.3.0)
	modelsHandler         *handlers.ModelsHandler
	chatHandler           *handlers.ChatHandler
	embeddingsHandler     *handlers.EmbeddingsHandler
	completionsHandler    *handlers.CompletionsHandler
	adminHandler          *handlers.AdminHandler
	statsHandler          *handlers.StatsHandler          // Handler для TUI статистики
	configHandler         *handlers.ConfigHandler         // Handler для конфигурации
	logsHandler           *handlers.LogsHandler           // Handler для логов
	metricsHistoryHandler *handlers.MetricsHistoryHandler // Handler для historical metrics
	requestsHandler       *handlers.RequestsHandler       // Handler для request monitoring (TUI-04)

	// WebSocket components
	wsHub              *websocket.Hub
	wsHandler          *websocket.Handler
	eventBroadcaster   *websocket.EventBroadcaster
	metricsBroadcaster *websocket.MetricsBroadcaster

	// Metrics storage
	metricsStorage *metrics.MetricsStorage

	// Request tracking
	requestStorage *request.Storage
}

// NewOptions содержит опции для создания роутера
type NewOptions struct {
	Config     *config.Config
	Logger     *logrus.Logger
	Version    string
	Database   storage.Database // Опциональная база данных для user auth
	JWTManager *jwt.Manager     // Опциональный JWT manager
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
		config:       opts.Config,
		logger:       opts.Logger,
		ollamaClient: ollamaClient,
		version:      opts.Version,
		db:           opts.Database,
		jwtManager:   opts.JWTManager,
	}

	// Setup Auth Service если есть database и JWT manager
	if r.db != nil && r.jwtManager != nil {
		r.authService = authService.NewAuthService(r.db, r.jwtManager, opts.Logger)
		r.bootstrapService = authService.NewBootstrapService(r.db, opts.Config.Auth.AdminKey, opts.Logger)
		opts.Logger.Info("User Authentication and Bootstrap services initialized")
	}

	// Если включена аутентификация, настраиваем API Key Management
	if opts.Config.Auth.Enabled {
		if err := r.setupAPIKeyManagement(opts.Config, opts.Logger); err != nil {
			return nil, fmt.Errorf("failed to setup API Key Management: %w", err)
		}
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
	// Error handling middleware (должен быть первым)
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

	// API Key middleware применяется только к защищенным группам
}

// setupRoutes настраивает все маршруты приложения
func (r *Router) setupRoutes() {
	r.setupHealthRoutes()
	r.setupMetricsRoutes()
	r.setupStatsRoutes()          // Для TUI
	r.setupConfigRoutes()         // Для TUI Configuration Viewer
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

// setupStatsRoutes настраивает эндпоинты статистики для TUI
func (r *Router) setupStatsRoutes() {
	// Эндпоинт для TUI (без аутентификации, только для локального использования)
	r.engine.GET("/api/stats", r.statsHandler.GetStats)
}

// setupConfigRoutes настраивает эндпоинты конфигурации для TUI
func (r *Router) setupConfigRoutes() {
	// Эндпоинт для TUI (без аутентификации, только для локального использования)
	r.engine.GET("/api/config", r.configHandler.GetConfig)

	// Публичный эндпоинт для списка моделей (для WebUI/TUI)
	r.engine.GET("/api/models", r.modelsHandler.List)

	// Эндпоинт для логов (для WebUI)
	r.engine.GET("/api/logs", r.logsHandler.GetLogs)
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
	// WebSocket endpoint
	r.engine.GET("/ws", r.wsHandler.HandleWebSocket)

	r.logger.WithFields(logrus.Fields{
		"endpoint":       "/ws",
		"clients_active": r.wsHub.GetActiveClientsCount(),
	}).Info("WebSocket endpoint configured")
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

	// Protected authentication endpoints (require JWT)
	if r.jwtManager != nil {
		authProtected := r.engine.Group("/api/auth")
		authProtected.Use(authMiddleware.JWTAuth(r.jwtManager, r.logger))
		{
			authProtected.POST("/logout", r.authHandler.Logout)
			authProtected.GET("/me", r.authHandler.Me)
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
			tenants.GET("/:id/members", r.tenantHandler.ListMembers)
			tenants.POST("/:id/members", r.tenantHandler.AddMember)
			tenants.PUT("/:id/members/:user_id", r.tenantHandler.UpdateMemberRole)
			tenants.DELETE("/:id/members/:user_id", r.tenantHandler.RemoveMember)
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
	r.engine.StaticFile("/tenants.html", "./web/tenants.html")
	r.engine.StaticFile("/api-keys.html", "./web/api-keys.html")
	r.engine.StaticFile("/usage.html", "./web/usage.html")
	r.engine.StaticFile("/admin.html", "./web/admin.html") // Admin Panel (Version 1.3.0)

	// Serve CSS and JS directories
	r.engine.Static("/css", "./web/css")
	r.engine.Static("/js", "./web/js")
	r.engine.Static("/assets", "./web/assets")

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

		// Hybrid auth принимает либо JWT либо API Key (database-backed)
		v1.Use(middleware.HybridAuth(r.jwtManager, r.db, r.logger))

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
		v1.Use(r.authenticator.RecordUsageMiddleware())

		v1.GET("/models", r.authenticator.PermissionMiddleware("models"), r.modelsHandler.List)
		v1.POST("/chat/completions", r.authenticator.PermissionMiddleware("chat"), r.chatHandler.Completion)
	} else {
		// Открытые эндпоинты (MVP mode без аутентификации)
		r.logger.Info("Using NO authentication for /v1 endpoints (MVP mode)")
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

	// Logs viewer
	admin.GET("/logs", r.logsHandler.GetLogs)

	r.logger.Info("Admin routes configured successfully")
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

// setupHandlers инициализирует все handlers
func (r *Router) setupHandlers(cfg *config.Config, logger *logrus.Logger, ollamaClient *ollama.ClientWithCircuitBreaker) {
	r.healthHandler = handlers.NewHealthHandler(cfg, logger, ollamaClient)
	r.modelsHandler = handlers.NewModelsHandler(cfg, logger, ollamaClient)
	r.chatHandler = handlers.NewChatHandler(cfg, logger, ollamaClient)
	r.embeddingsHandler = handlers.NewEmbeddingsHandler(cfg, logger, ollamaClient)
	r.completionsHandler = handlers.NewCompletionsHandler(cfg, logger, ollamaClient)
	r.configHandler = handlers.NewConfigHandler(cfg, logger) // Для TUI configuration viewer
	r.logsHandler = handlers.NewLogsHandler(cfg, logger)     // Для просмотра логов

	// System and User Authentication handlers (Version 1.3.0+)
	if r.bootstrapService != nil {
		r.systemHandler = handlers.NewSystemHandler(cfg, logger, r.bootstrapService)
		logger.Info("System handler initialized")
	}

	if r.authService != nil && r.db != nil {
		r.authHandler = handlers.NewAuthHandler(r.authService, logger)
		r.userHandler = handlers.NewUserHandler(r.db, logger)
		r.tenantHandler = handlers.NewTenantHandler(r.db, logger)
		r.conversationHandler = handlers.NewConversationHandler(r.db)
		r.adminUserHandler = handlers.NewAdminUserHandler(r.db, logger)
		r.usageHandler = handlers.NewUsageHandler(r.db, logger)
		logger.Info("User authentication handlers initialized")
	}

	// Admin handler с API Key Manager если доступен (+ Database для Version 1.3.0+)
	if r.keyManager != nil {
		r.adminHandler = handlers.NewAdminHandler(cfg, logger, r.keyManager, r.db)
	} else {
		r.adminHandler = handlers.NewAdminHandlerWithoutKeys(cfg, logger)
	}

	// Metrics Storage (Phase 12.1)
	r.metricsStorage = metrics.NewMetricsStorage(metrics.DefaultStorageConfig(), logger)
	r.metricsStorage.Start()

	// Request Storage (TUI-04)
	r.requestStorage = request.NewStorage(request.DefaultStorageConfig(), logger)
	r.requestStorage.Start()

	// Stats Handler с metrics storage для latency данных
	r.statsHandler = handlers.NewStatsHandler(cfg, logger, ollamaClient, r.keyManager, r.version, r.metricsStorage)

	// Metrics History Handler
	r.metricsHistoryHandler = handlers.NewMetricsHistoryHandler(cfg, logger, r.metricsStorage)

	// Requests Handler (TUI-04)
	r.requestsHandler = handlers.NewRequestsHandler(cfg, logger, r.requestStorage)

	// WebSocket Hub (Phase 12.2)
	r.wsHub = websocket.NewHub(logger)
	go r.wsHub.Run() // Запускаем hub в фоне

	// WebSocket Handler
	r.wsHandler = websocket.NewHandler(r.wsHub, logger)

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

	logger.Info("WebSocket Hub and Metrics Storage initialized")
}

// Shutdown gracefully останавливает все компоненты роутера
func (r *Router) Shutdown(ctx context.Context) error {
	r.logger.Info("Shutting down router components")

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
