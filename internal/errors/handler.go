// Package errors provides centralized error handling for AIGateway
package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
)

// ErrorHandler обрабатывает централизованную обработку ошибок
type ErrorHandler struct {
	logger *logrus.Logger
}

// NewErrorHandler создает новый обработчик ошибок
func NewErrorHandler(logger *logrus.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

// ErrorType представляет тип ошибки
type ErrorType string

const (
	ErrorTypeValidation         ErrorType = "validation"
	ErrorTypeAuthentication     ErrorType = "authentication"
	ErrorTypeAuthorization      ErrorType = "authorization"
	ErrorTypeNotFound           ErrorType = "not_found"
	ErrorTypeRateLimit          ErrorType = "rate_limit"
	ErrorTypeServiceUnavailable ErrorType = "service_unavailable"
	ErrorTypeInternal           ErrorType = "internal"
	ErrorTypeTimeout            ErrorType = "timeout"
	ErrorTypeConversion         ErrorType = "conversion"
	ErrorTypeModelNotFound      ErrorType = "model_not_found"
	ErrorTypeUpstreamError        ErrorType = "upstream_error"
)

// ApplicationError представляет ошибку приложения
type ApplicationError struct {
	Type       ErrorType              `json:"type"`
	Message    string                 `json:"message"`
	Code       string                 `json:"code"`
	StatusCode int                    `json:"status_code"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Cause      error                  `json:"-"`
}

// Error реализует интерфейс error
func (e *ApplicationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap возвращает underlying ошибку
func (e *ApplicationError) Unwrap() error {
	return e.Cause
}

// HandleError обрабатывает ошибку и отправляет соответствующий HTTP ответ
func (h *ErrorHandler) HandleError(c *gin.Context, err error) {
	var appErr *ApplicationError

	if errors.As(err, &appErr) {
		// Это наша ApplicationError
		h.handleApplicationError(c, appErr)
	} else {
		// Это обычная ошибка, оборачиваем её
		h.handleGenericError(c, err)
	}
}

// handleApplicationError обрабатывает ApplicationError
func (h *ErrorHandler) handleApplicationError(c *gin.Context, appErr *ApplicationError) {
	// Логирование в зависимости от типа ошибки
	logFields := logrus.Fields{
		"error_type":     appErr.Type,
		"error_code":     appErr.Code,
		"status_code":    appErr.StatusCode,
		"request_path":   c.Request.URL.Path,
		"request_method": c.Request.Method,
	}

	if appErr.Metadata != nil {
		for k, v := range appErr.Metadata {
			logFields[k] = v
		}
	}

	// Выбираем уровень логирования
	switch appErr.StatusCode {
	case http.StatusInternalServerError, http.StatusServiceUnavailable:
		h.logger.WithFields(logFields).WithError(appErr.Cause).Error("Application error")
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden:
		h.logger.WithFields(logFields).Warn("Client error")
	default:
		h.logger.WithFields(logFields).Info("Application error")
	}

	// Формируем OpenAI-совместимый ответ
	errorResponse := &models.ErrorResponse{
		Error: models.Error{
			Message: appErr.Message,
			Type:    string(appErr.Type) + "_error",
			Code:    appErr.Code,
		},
	}

	c.JSON(appErr.StatusCode, errorResponse)
}

// handleGenericError обрабатывает обычные ошибки
func (h *ErrorHandler) handleGenericError(c *gin.Context, err error) {
	h.logger.WithFields(logrus.Fields{
		"request_path":   c.Request.URL.Path,
		"request_method": c.Request.Method,
	}).WithError(err).Error("Unhandled error")

	// Создаем generic ApplicationError
	appErr := &ApplicationError{
		Type:       ErrorTypeInternal,
		Message:    "Internal server error",
		Code:       "internal_error",
		StatusCode: http.StatusInternalServerError,
		Cause:      err,
	}

	h.handleApplicationError(c, appErr)
}

// Создатели ошибок для разных типов

// NewValidationError создает ошибку валидации
func NewValidationError(message, code string, metadata map[string]interface{}) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeValidation,
		Message:    message,
		Code:       code,
		StatusCode: http.StatusBadRequest,
		Metadata:   metadata,
	}
}

// NewAuthenticationError создает ошибку аутентификации
func NewAuthenticationError(message, code string) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeAuthentication,
		Message:    message,
		Code:       code,
		StatusCode: http.StatusUnauthorized,
	}
}

// NewAuthorizationError создает ошибку авторизации
func NewAuthorizationError(message, code string) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeAuthorization,
		Message:    message,
		Code:       code,
		StatusCode: http.StatusForbidden,
	}
}

// NewModelNotFoundError создает ошибку отсутствия модели
func NewModelNotFoundError(model string) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeModelNotFound,
		Message:    fmt.Sprintf("Model '%s' not found", model),
		Code:       "model_not_found",
		StatusCode: http.StatusNotFound,
		Metadata: map[string]interface{}{
			"model": model,
		},
	}
}

// NewServiceUnavailableError создает ошибку недоступности сервиса
func NewServiceUnavailableError(service string, cause error) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeServiceUnavailable,
		Message:    fmt.Sprintf("Service '%s' is currently unavailable", service),
		Code:       "service_unavailable",
		StatusCode: http.StatusServiceUnavailable,
		Cause:      cause,
		Metadata: map[string]interface{}{
			"service": service,
		},
	}
}

// NewRateLimitError создает ошибку превышения лимита запросов
func NewRateLimitError(limit int, resetTime int64) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeRateLimit,
		Message:    "Rate limit exceeded",
		Code:       "rate_limit_exceeded",
		StatusCode: http.StatusTooManyRequests,
		Metadata: map[string]interface{}{
			"limit":      limit,
			"reset_time": resetTime,
		},
	}
}

// NewTimeoutError создает ошибку таймаута
func NewTimeoutError(operation string, timeout string) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeTimeout,
		Message:    fmt.Sprintf("Operation '%s' timed out after %s", operation, timeout),
		Code:       "timeout",
		StatusCode: http.StatusRequestTimeout,
		Metadata: map[string]interface{}{
			"operation": operation,
			"timeout":   timeout,
		},
	}
}

// NewConversionError создает ошибку конвертации
func NewConversionError(direction string, cause error) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeConversion,
		Message:    fmt.Sprintf("Failed to convert %s", direction),
		Code:       "conversion_error",
		StatusCode: http.StatusBadRequest,
		Cause:      cause,
		Metadata: map[string]interface{}{
			"conversion_direction": direction,
		},
	}
}

// NewUpstreamError creates an error from an upstream provider
func NewUpstreamError(msg string, cause error) *ApplicationError {
	return &ApplicationError{
		Type:       ErrorTypeUpstreamError,
		Message:    fmt.Sprintf("upstream error: %s", msg),
		Code:       "upstream_error",
		StatusCode: http.StatusServiceUnavailable,
		Cause:      cause,
		Metadata: map[string]interface{}{
			"upstream_message": msg,
		},
	}
}

// IsRetriableError проверяет можно ли повторить операцию
func IsRetriableError(err error) bool {
	var appErr *ApplicationError
	if !errors.As(err, &appErr) {
		return false
	}

	// Retriable error types
	retriableTypes := map[ErrorType]bool{
		ErrorTypeServiceUnavailable: true,
		ErrorTypeTimeout:            true,
		ErrorTypeUpstreamError:        true,
	}

	return retriableTypes[appErr.Type]
}

// IsClientError проверяет является ли ошибка клиентской (4xx)
func IsClientError(err error) bool {
	var appErr *ApplicationError
	if !errors.As(err, &appErr) {
		return false
	}

	return appErr.StatusCode >= 400 && appErr.StatusCode < 500
}

// IsServerError проверяет является ли ошибка серверной (5xx)
func IsServerError(err error) bool {
	var appErr *ApplicationError
	if !errors.As(err, &appErr) {
		return false
	}

	return appErr.StatusCode >= 500
}

// GetErrorCode извлекает код ошибки
func GetErrorCode(err error) string {
	var appErr *ApplicationError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return "unknown_error"
}

// GetErrorMetadata извлекает метаданные ошибки
func GetErrorMetadata(err error) map[string]interface{} {
	var appErr *ApplicationError
	if errors.As(err, &appErr) {
		return appErr.Metadata
	}
	return nil
}

// NewApplicationError создает новую ApplicationError
func NewApplicationError(errType ErrorType, message, code string, statusCode int, metadata map[string]interface{}, cause error) *ApplicationError {
	return &ApplicationError{
		Type:       errType,
		Message:    message,
		Code:       code,
		StatusCode: statusCode,
		Metadata:   metadata,
		Cause:      cause,
	}
}

