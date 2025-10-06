// Package router provides Phase 8 HTTP routing with API Key Management
package router

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/api/handlers"
	"ollama-openai-proxy/internal/api/middleware"
	"ollama-openai-proxy/internal/auth/apikey"
	authMiddleware "ollama-openai-proxy/internal/auth/middleware"
	"ollama-openai-proxy/internal/auth/ratelimit"
	"ollama-openai-proxy/internal/client/ollama"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/storage"
)

// RouterPhase8 представляет HTTP роутер с API Key Management
type RouterPhase8 struct {
	config       *config.Config
	logger       *logrus.Logger
	engine       *gin.Engine
	ollamaClient *ollama.ClientWithCircuitBreaker

	// API Key Management компоненты
	storage       storage.APIKeyStorage
	keyManager    *apikey.Manager
	authenticator *authMiddleware.APIKeyAuthenticator
	rateLimiter   *ratelimit.Limiter

	// Handlers
	healthHandler *handlers.HealthHandler
	modelsHandler *handlers.ModelsHandler
	chatHandler   *handlers.ChatHandler
	adminHandler  *handlers.AdminHandler
}

// NewPhase8Router создает новый роутер с API Key Management
func NewPhase8Router(cfg *config.Config, logger *logrus.Logger) (*RouterPhase8, error) {
	// Создаем Ollama клиент с circuit breaker
	ollamaClient, err := ollama.NewClientWithCircuitBreaker(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ollama client: %w", err)
	}

	// Создаем API Key Storage
	var stor storage.APIKeyStorage
	switch cfg.Auth.StorageType {
	case "json":
		stor = storage.NewJSONStorage(cfg.Auth.StoragePath, logger)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Auth.StorageType)
	}

	// Создаем API Key Manager
	keyManager := apikey.NewManager(cfg, logger, stor)

	// Создаем Rate Limiter
	rateLimiter := ratelimit.NewLimiter(cfg, logger)

	// Создаем Authenticator
	authenticator := authMiddleware.NewAPIKeyAuthenticator(cfg, logger, keyManager)

	r := &RouterPhase8{
		config:        cfg,
		logger:        logger,
		ollamaClient:  ollamaClient,
		storage:       stor,
		keyManager:    keyManager,
		authenticator: authenticator,
		rateLimiter:   rateLimiter,
	}

	// Инициализация handlers с API Key Manager
	r.healthHandler = handlers.NewHealthHandler(cfg, logger, ollamaClient)
	r.modelsHandler = handlers.NewModelsHandler(cfg, logger, ollamaClient)
	r.chatHandler = handlers.NewChatHandler(cfg, logger, ollamaClient)
	r.adminHandler = handlers.NewAdminHandler(cfg, logger, keyManager)

	r.setupEngine()
	r.setupRoutes()

	return r, nil
}

// Initialize инициализирует все компоненты Phase 8
func (r *RouterPhase8) Initialize() error {
	r.logger.Info("Initializing Phase 8 router with API Key Management")

	// Инициализируем API Key Manager
	ctx := context.Background()
	if err := r.keyManager.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize API Key Manager: %w", err)
	}

	r.logger.Info("Phase 8 router initialized successfully")
	return nil
}

// Engine возвращает Gin engine
func (r *RouterPhase8) Engine() *gin.Engine {
	return r.engine
}

// Close закрывает роутер и освобождает ресурсы
func (r *RouterPhase8) Close() error {
	r.logger.Info("Closing Phase 8 router")

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

// setupEngine настраивает Gin engine
func (r *RouterPhase8) setupEngine() {
	// Настройка режима Gin
	if r.config.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r.engine = gin.New()

	// Настройка middleware
	r.setupMiddleware()
}

// setupMiddleware настраивает базовые middleware
func (r *RouterPhase8) setupMiddleware() {
	// Error handling middleware (должен быть первым)
	r.engine.Use(middleware.ErrorHandling(r.logger))

	// Panic recovery с error handling
	r.engine.Use(middleware.PanicRecovery(r.logger))

	// Structured logging middleware
	loggingConfig := middleware.LoggingConfig{
		Logger:    r.logger,
		SkipPaths: []string{"/health", "/healthz", "/ready"}, // Пропускаем health checks
	}
	r.engine.Use(middleware.RequestLogging(loggingConfig))

	// CORS middleware
	r.engine.Use(middleware.CORS(r.config))

	// API Key middleware НЕ применяется глобально!
	// Будет применяться только к защищенным группам маршрутов
}

// setupRoutes настраивает все маршруты с API Key защитой
func (r *RouterPhase8) setupRoutes() {
	r.setupHealthRoutes()
	r.setupMetricsRoutes()
	r.setupOpenAIRoutes()
	r.setupAdminRoutes()
}

// setupHealthRoutes настраивает health check эндпоинты (без аутентификации)
func (r *RouterPhase8) setupHealthRoutes() {
	// Health endpoints БЕЗ middleware (доступны всем)
	healthGroup := r.engine.Group("/")
	healthGroup.Use() // Пустой middleware список

	healthGroup.GET("/health", r.healthHandler.Health)
	healthGroup.GET("/ready", r.healthHandler.Ready)
	healthGroup.GET("/healthz", r.healthHandler.Live)
}

// setupMetricsRoutes настраивает metrics эндпоинты
func (r *RouterPhase8) setupMetricsRoutes() {
	if r.config.Metrics.Enabled {
		// Metrics без аутентификации (можно добавить отдельную auth)
		r.engine.GET(r.config.Metrics.PrometheusPath, func(c *gin.Context) {
			// TODO: Интеграция с Prometheus
			c.String(200, "# HELP http_requests_total The total number of HTTP requests.\n# TYPE http_requests_total counter\nhttp_requests_total 0\n")
		})
	}
}

// setupOpenAIRoutes настраивает OpenAI API эндпоинты с аутентификацией
func (r *RouterPhase8) setupOpenAIRoutes() {
	v1 := r.engine.Group("/v1")

	// Применяем API Key middleware к защищенным эндпоинтам
	v1.Use(r.authenticator.AuthenticationMiddleware())
	v1.Use(r.authenticator.ModelAuthorizationMiddleware())
	v1.Use(r.rateLimiter.RateLimitMiddleware())
	v1.Use(r.authenticator.RecordUsageMiddleware())

	// Модели доступны всем аутентифицированным пользователям
	v1.GET("/models", r.authenticator.PermissionMiddleware("models"), r.modelsHandler.List)

	// Chat completions требуют chat разрешение
	v1.POST("/chat/completions", r.authenticator.PermissionMiddleware("chat"), r.chatHandler.Completion)

	// Embeddings и completions (заглушки с аутентификацией)
	v1.POST("/embeddings", r.authenticator.PermissionMiddleware("chat"), func(c *gin.Context) {
		c.JSON(501, gin.H{
			"error": gin.H{
				"message": "Embeddings endpoint not implemented yet",
				"type":    "not_implemented_error",
				"code":    "not_implemented",
			},
		})
	})

	v1.POST("/completions", r.authenticator.PermissionMiddleware("chat"), func(c *gin.Context) {
		c.JSON(501, gin.H{
			"error": gin.H{
				"message": "Legacy completions endpoint not implemented yet",
				"type":    "not_implemented_error",
				"code":    "not_implemented",
			},
		})
	})
}

// setupAdminRoutes настраивает административные API эндпоинты
func (r *RouterPhase8) setupAdminRoutes() {
	admin := r.engine.Group("/admin")

	// Применяем API Key middleware к admin эндпоинтам
	admin.Use(r.authenticator.AuthenticationMiddleware())
	admin.Use(r.authenticator.ModelAuthorizationMiddleware())
	admin.Use(r.rateLimiter.RateLimitMiddleware())
	admin.Use(r.authenticator.RecordUsageMiddleware())

	// Все admin эндпоинты требуют admin разрешение
	admin.Use(r.authenticator.PermissionMiddleware("admin"))

	// API Keys management
	admin.GET("/api-keys", r.adminHandler.ListAPIKeys)
	admin.POST("/api-keys", r.adminHandler.CreateAPIKey)
	admin.GET("/api-keys/:id", r.adminHandler.GetAPIKey)
	admin.PUT("/api-keys/:id", r.adminHandler.UpdateAPIKey)
	admin.DELETE("/api-keys/:id", r.adminHandler.DeleteAPIKey)
	admin.POST("/api-keys/:id/revoke", r.adminHandler.RevokeAPIKey)
	admin.GET("/api-keys/:id/usage", r.adminHandler.GetAPIKeyUsage)

	// System endpoints
	admin.GET("/stats", func(c *gin.Context) {
		// TODO: Системная статистика
		c.JSON(200, gin.H{
			"message": "System stats endpoint",
		})
	})

	admin.GET("/rate-limits", func(c *gin.Context) {
		// TODO: Статистика rate limiting
		stats := r.rateLimiter.GetLimiterStats()
		c.JSON(200, gin.H{
			"rate_limit_stats": stats,
		})
	})
}
