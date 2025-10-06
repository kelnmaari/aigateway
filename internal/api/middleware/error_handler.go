// Package middleware provides error handling middleware
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/errors"
)

// ErrorHandling создает middleware для централизованной обработки ошибок
func ErrorHandling(logger *logrus.Logger) gin.HandlerFunc {
	errorHandler := errors.NewErrorHandler(logger)

	return func(c *gin.Context) {
		// Выполняем следующие middleware и handlers
		c.Next()

		// Если есть ошибки, обрабатываем их
		if len(c.Errors) > 0 {
			// Берем последнюю ошибку (обычно самую важную)
			err := c.Errors.Last().Err

			// Если ответ уже отправлен, просто логируем
			if c.Writer.Written() {
				logger.WithError(err).WithFields(logrus.Fields{
					"path":        c.Request.URL.Path,
					"method":      c.Request.Method,
					"status_code": c.Writer.Status(),
				}).Error("Error occurred after response was sent")
				return
			}

			// Обрабатываем ошибку через централизованный handler
			errorHandler.HandleError(c, err)
		}
	}
}

// PanicRecovery создает middleware для обработки panic с error handler
func PanicRecovery(logger *logrus.Logger) gin.HandlerFunc {
	errorHandler := errors.NewErrorHandler(logger)

	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Создаем ошибку из panic
		var err error
		if e, ok := recovered.(error); ok {
			err = e
		} else {
			err = errors.NewApplicationError(
				errors.ErrorTypeInternal,
				"Internal server error due to panic",
				"panic_recovered",
				500,
				map[string]interface{}{
					"panic_value": recovered,
				},
				nil,
			)
		}

		logger.WithFields(logrus.Fields{
			"panic":        recovered,
			"request_path": c.Request.URL.Path,
			"method":       c.Request.Method,
		}).Error("Panic recovered")

		// Используем error handler для отправки ответа
		errorHandler.HandleError(c, err)
	})
}

// AddError добавляет ошибку к контексту Gin
func AddError(c *gin.Context, err error) {
	c.Error(err)
}

// AddApplicationError добавляет ApplicationError к контексту
func AddApplicationError(c *gin.Context, errType errors.ErrorType, message, code string, statusCode int, metadata map[string]interface{}, cause error) {
	appErr := errors.NewApplicationError(errType, message, code, statusCode, metadata, cause)
	c.Error(appErr)
}
