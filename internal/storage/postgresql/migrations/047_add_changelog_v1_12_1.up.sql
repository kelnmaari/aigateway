
INSERT INTO changelogs (version, release_date, content) VALUES
('1.12.1', '2025-10-26', '## [1.12.1] - 2025-10-26

### Added
- **Model Preloading & Warming**: Механизм предзагрузки моделей для устранения cold start задержки
  - Preload моделей при старте сервера (настраиваемый список в конфигурации)
  - Health check loop для поддержания моделей в горячем состоянии
  - Автоматическая выгрузка неиспользуемых моделей через настраиваемый timeout
  - Track model usage для оптимизации preloading
  - Admin API endpoints:
    - GET /api/admin/models/loaded - список загруженных моделей с статусом
    - POST /api/admin/models/:name/preload - ручная загрузка модели

### Changed
- **Chat Handler**: Автоматический tracking использования моделей при каждом запросе
- **Configuration**: Добавлена секция models.preload с полной настройкой preloading

### Technical
- Новый сервис internal/services/model/preloader.go:
  - ModelPreloader с async startup и health check loops
  - Thread-safe tracking загруженных моделей
  - Graceful shutdown при остановке сервера
- Новый handler internal/api/handlers/model_preload.go для Admin API
- Integration в Router через NewOptions.ModelPreloader
- Integration в ChatHandler через ModelPreloader interface
- Конфигурация:
  - models.preload.enabled - включить/выключить preloading
  - models.preload.on_startup - загружать при старте
  - models.preload.keep_warm - поддерживать в горячем состоянии
  - models.preload.health_check_interval - интервал проверки (default: 5m)
  - models.preload.warm_up_prompt - тестовый промпт (default: "Hello")
  - models.preload.max_loaded_models - лимит одновременно загруженных (0 = unlimited)
  - models.preload.unload_after - timeout выгрузки (0 = never)

### Performance
- **First Request Latency**: Сокращение времени первого ответа с 5-30s до <1s для preloaded моделей
- **Memory Management**: LRU eviction через Ollama при достижении лимита памяти
- **Non-blocking**: Async preload не блокирует startup сервера

### Documentation
- Updated configs/dev.yaml с примером конфигурации preloading
- API documentation для Admin endpoints в Roadmap')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
    