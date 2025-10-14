package metrics

import (
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/storage"
)

// CustomMonigoMetrics управляет сбором custom метрик для MoniGo dashboard
// Примечание: В текущей версии MoniGo (v1.1.0) custom metrics собираются
// автоматически через middleware. Этот тип оставлен для будущих расширений.
type CustomMonigoMetrics struct {
	monigo interface{}
	db     storage.Database
	config *config.Config
	logger *logrus.Logger
}

// NewCustomMonigoMetrics создает новый collector для custom метрик
func NewCustomMonigoMetrics(m interface{}, db storage.Database, cfg *config.Config, logger *logrus.Logger) *CustomMonigoMetrics {
	return &CustomMonigoMetrics{
		monigo: m,
		db:     db,
		config: cfg,
		logger: logger,
	}
}

// Start запускает сбор всех custom метрик в фоновом режиме
// В текущей версии это заглушка, так как MoniGo собирает метрики автоматически
func (c *CustomMonigoMetrics) Start() {
	c.logger.Info("Custom MoniGo metrics collector started (passive mode)")

	// TODO (v1.9.4+): Добавить сбор специфичных для Ollama метрик
	// - Ollama latency (avg, p50, p95, p99)
	// - Active API keys count
	// - Active users (24h)
	// - Usage statistics (requests, tokens)
	// - Error rates
	//
	// Эти метрики будут собираться через фоновые goroutines
	// и передаваться в MoniGo через API (если поддерживается в будущих версиях)
}

// Stop останавливает сбор метрик
func (c *CustomMonigoMetrics) Stop() {
	c.logger.Info("Custom MoniGo metrics collector stopped")
}
