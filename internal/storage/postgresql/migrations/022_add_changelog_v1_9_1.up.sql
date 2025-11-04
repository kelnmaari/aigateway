
INSERT INTO changelogs (version, release_date, content) VALUES
('1.9.1', '2025-10-13', '## [1.9.1] - 2025-10-13

### Added
- **Dynamic Model Parameters Configuration**: Красивая панель управления параметрами модели
  - Chat UI - интуитивная панель над строкой ввода с collapsible design
  - Model Selector с автоматической загрузкой доступных моделей из API
  - Real-time параметры: Temperature, Top P, Max Tokens, Context Window
  - Sliders с синхронизацией с number inputs для точной настройки
  - Tooltips с объяснениями для каждого параметра

- **Quick Presets**: 4 готовых пресета для разных сценариев
  - 🎨 Creative (temp 1.2) - для творческой генерации
  - ⚖️ Balanced (temp 0.7) - универсальный режим
  - 🎯 Precise (temp 0.3) - для точных ответов
  - 💻 Coding (temp 0.2) - оптимизирован для программирования

- **localStorage Persistence**: Автоматическое сохранение настроек
  - Запоминание выбранной модели между сессиями
  - Сохранение параметров в браузере
  - Автовосстановление при перезагрузке страницы

- **Backend Model Configuration System**:
  - ModelConfig и ModelParameters models на основе официального Ollama API
  - Database schema с поддержкой global/tenant/user scopes
  - Priority-based config resolution (user → tenant → global → defaults)
  - Automatic effective config application в chat handler

### Changed
- **Chat Interface**: Переработан UI чата
  - Model selector перемещен из header над строку ввода
  - Добавлена expandable parameters panel
  - Улучшена визуальная иерархия элементов
  - Responsive design для мобильных устройств

- **API Integration**: Обновлен формат запросов
  - api.streamChatMessage теперь принимает объект с параметрами
  - Backward compatibility с old string format
  - Support для Ollama-specific options (num_ctx)

- **Database Interface**: Новые методы для model configs
  - CreateModelConfig, GetModelConfig, UpdateModelConfig, DeleteModelConfig
  - GetModelConfigByScope для получения config по scope
  - GetEffectiveModelConfig с автоматическим priority resolution

### Technical
- **Backend (Go)**:
  - Новый файл internal/models/model_config.go с полными типами параметров
  - Миграция v21: таблица model_configs с indexes и triggers
  - SQLite реализация CRUD для model configs в internal/storage/sqlite/model_configs.go
  - PostgreSQL stubs в internal/storage/postgresql/stubs.go
  - Интеграция в internal/api/handlers/chat.go с type-safe конвертацией

- **Frontend (JavaScript)**:
  - Новый контроллер web/js/model-panel.js для управления панелью
  - CSS стили в web/css/model-panel.css с dark/light mode support
  - Обновлен web/js/chat.js для использования modelPanel
  - Обновлен web/js/api.js с поддержкой параметров в requests

- **Параметры основаны на официальном Ollama API**:
  - Predict options: Temperature, TopP, TopK, NumPredict, RepeatPenalty и др.
  - Runner options: NumCtx, NumBatch, NumGPU, MainGPU, UseMMap, NumThread
  - Полная совместимость с ollama-lib/api/types.go')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	