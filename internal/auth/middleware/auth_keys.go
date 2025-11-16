// Package middleware provides authentication middleware for API keys
package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/auth/apikey"
	"aigateway/internal/config"
	"aigateway/internal/metrics"
	"aigateway/internal/models"
)

// APIKeyAuthenticator обрабатывает аутентификацию через API ключи
// APIKeyWorker interface for Redis cache-through pattern (v3.0.6+)
type APIKeyWorker interface {
	GetAPIKey(ctx context.Context, keyID string) (interface{}, error)
}

type APIKeyAuthenticator struct {
	config       *config.Config
	logger       *logrus.Logger
	keyManager   *apikey.Manager
	apiKeyWorker APIKeyWorker  // v3.0.6+: Redis cache-through
	enabled      bool
}

// NewAPIKeyAuthenticator создает новый аутентификатор
func NewAPIKeyAuthenticator(cfg *config.Config, logger *logrus.Logger, keyManager *apikey.Manager) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{
		config:     cfg,
		logger:     logger,
		keyManager: keyManager,
		enabled:    cfg.Auth.Enabled,
	}
}

// SetAPIKeyWorker sets the API key worker for Redis caching (v3.0.6+)
func (a *APIKeyAuthenticator) SetAPIKeyWorker(worker APIKeyWorker) {
	a.apiKeyWorker = worker
	a.logger.Info("✅ API Key cache-through enabled (Redis)")
}

// AuthenticationMiddleware создает middleware для аутентификации API ключей
func (a *APIKeyAuthenticator) AuthenticationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Если аутентификация отключена, пропускаем
		if !a.enabled {
			a.logger.Debug("Authentication disabled, skipping API key validation")
			c.Next()
			return
		}

	// Извлекаем API ключ из заголовков
	apiKey := a.extractAPIKey(c)
	if apiKey == "" {
		a.handleAuthenticationError(c, "missing_api_key", "API key is required")
		return
	}

	// Валидируем API ключ
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// v3.0.6+: Try cache-through pattern via APIKeyWorker if available
	var validation *models.APIKeyValidationResult
	var err error

	if a.apiKeyWorker != nil {
		// Cache-through: Redis first → DB fallback
		validation, err = a.validateAPIKeyWithCache(ctx, apiKey)
	} else {
		// Fallback: Direct validation (slow - checks all keys!)
		validation, err = a.keyManager.ValidateAPIKey(ctx, apiKey)
	}

	if err != nil {
		a.logger.WithError(err).Error("Failed to validate API key")
		a.handleAuthenticationError(c, "validation_error", "Failed to validate API key")
		return
	}

	if !validation.Valid {
		a.logger.WithField("error", validation.Error).Warn("API key validation failed")
		a.handleAuthenticationError(c, "invalid_api_key", validation.Error)
		return
	}

	// Записываем информацию об аутентификации в контекст
	c.Set("authenticated", true)
	c.Set("api_key_id", validation.APIKey.ID)
	c.Set("api_key_info", validation.APIKey)

	a.logger.WithFields(logrus.Fields{
		"key_id":   validation.APIKey.ID,
		"key_name": validation.APIKey.Name,
		"endpoint": c.Request.URL.Path,
	}).Debug("API key authentication successful")

	c.Next()
	}
}

// ModelAuthorizationMiddleware проверяет доступ к моделям
func (a *APIKeyAuthenticator) ModelAuthorizationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Если аутентификация отключена, пропускаем
		if !a.enabled {
			c.Next()
			return
		}

		// Проверяем что пользователь аутентифицирован
		if !a.isAuthenticated(c) {
			c.Next()
			return
		}

		// Извлекаем информацию о модели из запроса
		model := a.extractModelFromRequest(c)
		if model == "" {
			// Если модель не указана, пропускаем проверку
			c.Next()
			return
		}

		// Получаем информацию о ключе
		apiKeyInfo, exists := c.Get("api_key_info")
		if !exists {
			a.handleAuthorizationError(c, "missing_key_info", "API key information not found")
			return
		}

		apiKey := apiKeyInfo.(*models.APIKeyPublic)

		// Проверяем доступ к модели
		if !a.hasModelAccess(apiKey, model) {
			a.logger.WithFields(logrus.Fields{
				"key_id":         apiKey.ID,
				"key_name":       apiKey.Name,
				"model":          model,
				"allowed_models": apiKey.Models,
			}).Warn("API key does not have access to requested model")

			a.handleAuthorizationError(c, "model_access_denied",
				fmt.Sprintf("API key does not have access to model: %s", model))
			return
		}

		a.logger.WithFields(logrus.Fields{
			"key_id": apiKey.ID,
			"model":  model,
		}).Debug("Model access authorized")

		c.Set("authorized_model", model)
		c.Next()
	}
}

// PermissionMiddleware проверяет разрешения для эндпоинтов
func (a *APIKeyAuthenticator) PermissionMiddleware(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Если аутентификация отключена, пропускаем
		if !a.enabled {
			c.Next()
			return
		}

		// Проверяем что пользователь аутентифицирован
		if !a.isAuthenticated(c) {
			a.handleAuthorizationError(c, "not_authenticated", "Authentication required")
			return
		}

		// Получаем информацию о ключе
		apiKeyInfo, exists := c.Get("api_key_info")
		if !exists {
			a.handleAuthorizationError(c, "missing_key_info", "API key information not found")
			return
		}

		apiKey := apiKeyInfo.(*models.APIKeyPublic)

		// Проверяем разрешение
		if !apiKey.HasPermission(requiredPermission) && !apiKey.HasPermission("*") {
			a.logger.WithFields(logrus.Fields{
				"key_id":              apiKey.ID,
				"required_permission": requiredPermission,
				"key_permissions":     apiKey.Permissions,
			}).Warn("API key does not have required permission")

			a.handleAuthorizationError(c, "insufficient_permissions",
				fmt.Sprintf("API key does not have required permission: %s", requiredPermission))
			return
		}

		c.Next()
	}
}

// extractAPIKey извлекает API ключ из HTTP заголовков
func (a *APIKeyAuthenticator) extractAPIKey(c *gin.Context) string {
	// 1. Authorization header (Bearer token)
	auth := c.GetHeader("Authorization")
	if auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
	}

	// 2. X-API-Key header
	apiKey := c.GetHeader("X-API-Key")
	if apiKey != "" {
		return apiKey
	}

	// 3. Query parameter (не рекомендуется для production)
	queryKey := c.Query("api_key")
	if queryKey != "" {
		a.logger.Warn("API key passed in query parameter (not recommended)")
		return queryKey
	}

	return ""
}

// extractModelFromRequest извлекает имя модели из запроса
func (a *APIKeyAuthenticator) extractModelFromRequest(c *gin.Context) string {
	// Для POST запросов пытаемся извлечь из JSON body
	if c.Request.Method == "POST" {
		var requestBody struct {
			Model string `json:"model"`
		}

		// Читаем body
		bodyBytes, err := c.GetRawData()
		if err != nil {
			return ""
		}

		// Восстанавливаем body для последующего чтения
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Парсим JSON
		if err := json.Unmarshal(bodyBytes, &requestBody); err == nil {
			return requestBody.Model
		}
	}

	// Для GET запросов из query параметров
	return c.Query("model")
}

// isAuthenticated проверяет аутентифицирован ли пользователь
func (a *APIKeyAuthenticator) isAuthenticated(c *gin.Context) bool {
	authenticated, exists := c.Get("authenticated")
	return exists && authenticated.(bool)
}

// hasModelAccess проверяет доступ к модели
func (a *APIKeyAuthenticator) hasModelAccess(apiKey *models.APIKeyPublic, model string) bool {
	for _, allowedModel := range apiKey.Models {
		if allowedModel == "*" || allowedModel == model {
			return true
		}
	}
	return false
}

// validateAPIKeyWithCache validates API key using Redis cache-through pattern (v3.0.6+)
func (a *APIKeyAuthenticator) validateAPIKeyWithCache(ctx context.Context, plainKey string) (*models.APIKeyValidationResult, error) {
	// Проверка формата ключа
	if !models.IsValidAPIKeyFormat(plainKey) {
		return &models.APIKeyValidationResult{
			Valid: false,
			Error: "invalid API key format",
		}, nil
	}

	// Извлекаем key_id из plain key (формат: sk-proj-<keyid>-<random>)
	keyID := extractKeyIDFromPlainKey(plainKey)
	if keyID == "" {
		return &models.APIKeyValidationResult{
			Valid: false,
			Error: "invalid API key format - cannot extract key ID",
		}, nil
	}

	// Cache-through: Redis first → DB fallback
	cachedData, err := a.apiKeyWorker.GetAPIKey(ctx, keyID)
	if err != nil {
		a.logger.WithError(err).Warn("Failed to get API key from cache, falling back to direct validation")
		// Fallback to direct validation
		return a.keyManager.ValidateAPIKey(ctx, plainKey)
	}

	// Преобразуем interface{} обратно в *models.APIKey
	apiKey, ok := cachedData.(*models.APIKey)
	if !ok {
		a.logger.Error("Invalid API key type from cache")
		return &models.APIKeyValidationResult{
			Valid: false,
			Error: "internal error - invalid cached key format",
		}, nil
	}

	// Verify bcrypt hash
	if !apiKey.VerifyKey(plainKey) {
		return &models.APIKeyValidationResult{
			Valid: false,
			Error: "invalid API key",
		}, nil
	}

	// Проверка статуса
	if apiKey.Status != models.APIKeyStatusActive {
		return &models.APIKeyValidationResult{
			Valid: false,
			Error: "API key is not active",
		}, nil
	}

	// Проверка истечения срока
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return &models.APIKeyValidationResult{
			Valid: false,
			Error: "API key has expired",
		}, nil
	}

	// Update last used timestamp
	apiKey.UpdateLastUsed()

	// Convert to APIKeyPublic
	publicKey := &models.APIKeyPublic{
		ID:          apiKey.ID,
		Name:        apiKey.Name,
		Description: apiKey.Description,
		Status:      apiKey.Status,
		Models:      apiKey.Models,
		Permissions: apiKey.Permissions,
		CreatedAt:   apiKey.CreatedAt,
		ExpiresAt:   apiKey.ExpiresAt,
		LastUsedAt:  apiKey.LastUsedAt,
	}

	return &models.APIKeyValidationResult{
		Valid:  true,
		APIKey: publicKey,
	}, nil
}

// extractKeyIDFromPlainKey извлекает key_id из plain API key
// Поддерживаемые форматы:
//   - sk-proj-<keyid>-<random>      (новый формат)
//   - sk-existing-<keyid>           (device API key, без random)
//   - sk-<keyid>-<random>           (старый формат с random)
func extractKeyIDFromPlainKey(plainKey string) string {
	if !strings.HasPrefix(plainKey, "sk-") {
		return ""
	}
	
	parts := strings.Split(plainKey, "-")
	if len(parts) < 2 {
		return ""
	}
	
	// Новый формат: sk-proj-<keyid>-<random>
	// parts[0] = "sk", parts[1] = "proj", parts[2] = keyid, parts[3+] = random
	if parts[1] == "proj" && len(parts) >= 4 {
		return parts[2]
	}
	
	// Device API key: sk-existing-<keyid>
	// parts[0] = "sk", parts[1] = "existing", parts[2] = keyid (ak_xxx)
	if parts[1] == "existing" && len(parts) == 3 {
		return parts[2]
	}
	
	// Старый формат с random: sk-<keyid>-<random>
	// parts[0] = "sk", parts[1] = keyid, parts[2+] = random
	if len(parts) >= 3 {
		// Key ID может содержать underscores (например, ak_1762963462_a44e9a32)
		// но НЕ содержит дефисы (поэтому это parts[1])
		return parts[1]
	}
	
	// Fallback: просто второй элемент
	if len(parts) >= 2 {
		return parts[1]
	}
	
	return ""
}

// handleAuthenticationError обрабатывает ошибки аутентификации
func (a *APIKeyAuthenticator) handleAuthenticationError(c *gin.Context, code, message string) {
	a.logger.WithFields(logrus.Fields{
		"code":      code,
		"message":   message,
		"endpoint":  c.Request.URL.Path,
		"method":    c.Request.Method,
		"client_ip": c.ClientIP(),
	}).Warn("Authentication failed")

	c.JSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"message": message,
			"type":    "authentication_error",
			"code":    code,
		},
	})
	c.Abort()
}

// handleAuthorizationError обрабатывает ошибки авторизации
func (a *APIKeyAuthenticator) handleAuthorizationError(c *gin.Context, code, message string) {
	a.logger.WithFields(logrus.Fields{
		"code":     code,
		"message":  message,
		"endpoint": c.Request.URL.Path,
		"method":   c.Request.Method,
	}).Warn("Authorization failed")

	c.JSON(http.StatusForbidden, gin.H{
		"error": gin.H{
			"message": message,
			"type":    "authorization_error",
			"code":    code,
		},
	})
	c.Abort()
}

// RecordUsageMiddleware записывает использование API ключа
func (a *APIKeyAuthenticator) RecordUsageMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Если аутентификация отключена, пропускаем
		if !a.enabled {
			c.Next()
			return
		}

		// Проверяем что пользователь аутентифицирован
		if !a.isAuthenticated(c) {
			c.Next()
			return
		}

		// Записываем время начала запроса
		start := time.Now()

		// Обрабатываем запрос
		c.Next()

		// Записываем использование после обработки запроса
		go a.recordUsageAsync(c, start)
	}
}

// recordUsageAsync записывает использование в фоне
func (a *APIKeyAuthenticator) recordUsageAsync(c *gin.Context, start time.Time) {
	defer func() {
		if r := recover(); r != nil {
			a.logger.WithField("panic", r).Error("Panic in usage recording")
		}
	}()

	keyID, exists := c.Get("api_key_id")
	if !exists {
		return
	}

	model := a.extractModelFromRequest(c)
	endpoint := c.Request.URL.Path
	success := c.Writer.Status() < 400

	// Примерная оценка токенов (для точного подсчета нужен доступ к response body)
	var tokens int64 = 0
	if success && strings.Contains(endpoint, "chat/completions") {
		// Приблизительная оценка на основе размера response
		responseSize := c.Writer.Size()
		tokens = int64(responseSize / 4) // ~4 символа = 1 токен
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	keyIDStr := keyID.(string)
	if err := a.keyManager.RecordUsage(ctx, keyIDStr, model, endpoint, tokens, success); err != nil {
		a.logger.WithError(err).WithField("key_id", keyID).Warn("Failed to record API key usage")
	}

	// Record tokens usage metric
	if metrics.DefaultMetrics != nil && tokens > 0 {
		metrics.DefaultMetrics.RecordAPIKeyTokensUsed(keyIDStr, model, int(tokens))
	}
}

// GetAuthenticatedKeyID получает ID аутентифицированного ключа
func GetAuthenticatedKeyID(c *gin.Context) (string, bool) {
	keyID, exists := c.Get("api_key_id")
	if !exists {
		return "", false
	}
	return keyID.(string), true
}

// GetAuthenticatedKeyInfo получает информацию об аутентифицированном ключе
func GetAuthenticatedKeyInfo(c *gin.Context) (*models.APIKeyPublic, bool) {
	keyInfo, exists := c.Get("api_key_info")
	if !exists {
		return nil, false
	}
	return keyInfo.(*models.APIKeyPublic), true
}

// RequireAuthentication проверяет что пользователь аутентифицирован
func RequireAuthentication() gin.HandlerFunc {
	return func(c *gin.Context) {
		authenticated, exists := c.Get("authenticated")
		if !exists || !authenticated.(bool) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"message": "Authentication required",
					"type":    "authentication_error",
					"code":    "authentication_required",
				},
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

