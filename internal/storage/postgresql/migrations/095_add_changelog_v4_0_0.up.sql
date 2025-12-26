INSERT INTO changelogs (version, release_date, content) VALUES
('4.0.0', '2025-12-26', '## [4.0.0] - 2025-12-26

### Added

- **🚀 Model Management Enhancements**:
  - **Save Config**: сохранение конфигурации модели без загрузки/скачивания
  - **Download & Save**: скачивание модели с автоматическим сохранением конфига
  - **Container Logs Modal**: просмотр логов контейнера в реальном времени
    - Автообновление каждые 2 секунды
    - Цветовая подсветка логов (ERROR=красный, WARN=жёлтый, INFO=синий)
    - Авто-скролл к последним записям
  - **llama.cpp Parameters**: ctx_size, n_parallel, flash_attn для настройки контекста

- **🔍 GitLab Code Review System**:
  - **Per-File Review Mode**: ревью каждого файла отдельным запросом к LLM
  - **Tool Calling**: инструменты search_codebase, get_function_definition, get_type_definition
  - **Review Language Setting**: настройка языка ревью (русский/английский) в проекте
  - **Max Review Tokens**: настраиваемый лимит токенов для ответа LLM
  - **File Path Enforcement**: обязательное указание файла и строки в issues

- **📦 RPM Packaging & Distribution**:
  - **GitLab CI/CD Pipeline**: автоматическая сборка RPM пакетов
  - **One-liner Install**: curl -fsSL .../install.sh | sudo bash
  - **Systemd Integration**: автоматический restart сервиса при обновлении
  - **Version Management**: семантическое версионирование для веток и тегов

- **👤 User Management**:
  - **Make Admin / Remove Admin**: управление ролью администратора из UI
  - **Version Update Banner**: уведомление о новой версии на Dashboard

- **🤗 HuggingFace Integration**:
  - **GGUF Filter**: фильтрация моделей по провайдеру (GGUF/vLLM/SGLang)
  - **Pagination**: загрузка дополнительных моделей при прокрутке
  - **Download Cancellation**: отмена и удаление загрузок
  - **Single File Download**: скачивание конкретного GGUF файла

### Fixed

- **Variable Shadowing**: исправлен конфликт имён переменных цикла с Paraglide messages
- **Log Coloring**: inline styles вместо Tailwind для динамически генерируемого HTML
- **Model Auto-Start Race Condition**: per-alias mutex для предотвращения дублирования контейнеров
- **API Routing for Model IDs**: корректная обработка / в идентификаторах моделей
- **RPM Binary Compatibility**: glibc вместо musl для Rocky Linux

### Technical

- POST /api/system/inference/create-saved - создание saved модели из формы
- GET /api/system/inference/logs/:alias - получение логов контейнера
- GET /api/system/info - информация о версии приложения
- Per-file review с tool calling для GitLab MR анализа
- RPM spec file с systemd service unit
- GitLab Package Registry для дистрибуции')
ON CONFLICT (version) DO UPDATE SET 
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;

