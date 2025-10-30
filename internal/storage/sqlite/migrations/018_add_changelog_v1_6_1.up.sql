
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.6.1', '2025-10-12', '## [1.6.1] - 2025-10-12

### Added
- **OpenTelemetry Distributed Tracing**: Полная интеграция OpenTelemetry для distributed tracing
  - Поддержка Jaeger и Zipkin экспортеров
  - Автоматическая трассировка всех HTTP запросов
  - W3C Trace Context propagation
  - Настраиваемый sampling rate (0.0 - 1.0)
  - Span annotations с HTTP metadata (method, URL, status code)
  - Error tracking для запросов со status code >= 400

### Changed
- **Configuration**: Добавлена секция observability.tracing в конфигурацию
  - enabled: включение/отключение tracing
  - provider: выбор между "jaeger" или "zipkin"
  - service_name: название сервиса в traces
  - sampling_rate: процент трассируемых запросов
  - jaeger.endpoint и zipkin.endpoint: настройка экспортеров

### Technical
- Новый пакет internal/observability с TracerProvider
- TracingMiddleware для автоматической трассировки Gin запросов
- Интеграция в router и main.go с graceful shutdown
- Зависимости: go.opentelemetry.io/otel v1.38.0
- Тесты: 100% покрытие для observability и tracing middleware
- Tracing middleware применяется первым в цепочке для полной трассировки');
	