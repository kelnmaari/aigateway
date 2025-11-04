
INSERT INTO changelogs (version, release_date, content) VALUES
('2.3.0', '2025-10-28', '## [2.3.0] - 2025-10-28

### Added

- **Model Registry & Multi-Provider Foundation** (REGISTRY-01, REGISTRY-03): Универсальная система управления моделями от разных провайдеров
  - **Model Registry Core**:
    - Централизованный реестр всех доступных моделей с метаданными
    - Поддержка множества провайдеров: Ollama, vLLM, OpenAI, Anthropic, Custom
    - Отслеживание capabilities моделей: chat, embeddings, vision, function-calling
    - Health monitoring для провайдеров и моделей с историей статусов
    - Auto-discovery механизм для автоматического обнаружения новых моделей
    - Performance метрики: latency, throughput, total requests
  - **Model Registry WebUI** (admin-registry.html):
    - Dashboard с provider status cards (active/inactive/error states)
    - Табы "Model Providers" и "Registered Models" для раздельного управления
    - Real-time health indicators с автообновлением каждые 30 секунд
    - Модальные формы для создания/редактирования провайдеров и моделей
    - Фильтрация по provider, status, health, keyword
    - Кнопка "Discover Models" для запуска автообнаружения моделей
    - Statistics cards: total models, active models, providers count, healthy models
    - Integration с основной admin панелью через вкладку "Model Registry"

### Technical

- **Backend (Go)**: Новые модели ModelRegistry, ModelProvider, ModelCapabilities, ProviderManager
- **Storage**: CRUD методы для model_registry и model_providers таблиц
- **API**: RegistryHandler с endpoints для CRUD, health checks, discovery
- **Frontend**: admin-registry.html, admin-registry.js с real-time updates
- **Миграции**: v62 (create tables), v63 (seed Ollama provider)

### Notes

- vLLM провайдер настроен, но отключен (inactive)
- Auto-discovery работает в фоновом режиме
- Health checks обновляют статусы автоматически')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	