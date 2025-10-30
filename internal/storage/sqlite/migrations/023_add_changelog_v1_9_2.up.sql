
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.9.2', '2025-10-13', '## [1.9.2] - 2025-10-13

### Added
- **Compact UI Design (Cursor-style)**: Переработан дизайн панели моделей
  - Компактная горизонтальная панель с минималистичным дизайном
  - Model selector в виде элегантного dropdown без лишних элементов
  - Кнопка параметров в виде иконки (32x32px) для экономии места
  - Адаптивный дизайн для мобильных устройств

- **Context Window Tracking**: Умное управление контекстным окном
  - Real-time индикатор использования контекста (tokens used / max tokens)
  - Визуальные уровни предупреждений:
    - ✅ Нормальный (0-60%%): серый фон
    - ⚠️ Предупреждение (60-75%%): желтый фон
    - 🔴 Критический (75%%+): красный фон
  - Динамическое обновление при изменении num_ctx в параметрах

- **Auto-Summarization**: Автоматическая суммаризация при заполнении контекста
  - Автоматический триггер при достижении 75%% контекста
  - Умная суммаризация с сохранением последних 3 обменов сообщениями
  - Запрос к модели для создания лаконичного summary (2-3 параграфа)
  - Замена старых сообщений на system message с summary
  - Уведомления об успешной суммаризации с метриками токенов

- **Context Manager Module**: Новый модуль для управления контекстом
  - Оценка токенов в реальном времени (~1 токен = 4 символа)
  - Отслеживание всех сообщений с подсчетом токенов
  - API для получения статистики контекста
  - Автоматическая очистка при начале новой беседы

### Changed
- **Model Panel UI**: Компактный дизайн в стиле Cursor
  - Уменьшен padding с 16px до 8px для компактности
  - Model selector без label, только dropdown
  - Parameters toggle в виде иконки вместо текстовой кнопки
  - Уменьшены размеры шрифтов для экономии места
  - Sliders уменьшены с 18px до 14px thumb size

- **Request Parameters**: Ollama-specific options теперь применяются
  - Добавлено поле Options в ChatCompletionRequest model
  - Converter обрабатывает req.Options (num_ctx, top_k, repeat_penalty)
  - Полная интеграция с Ollama API types

- **Chat Flow**: Интеграция Context Manager
  - Автоматическое добавление сообщений в context tracker
  - Обновление context window size при смене параметров
  - Очистка контекста при начале нового чата

### Technical
- **Frontend (JavaScript)**:
  - Новый модуль web/js/context-manager.js с ContextManager class
  - Интеграция в chat.js для tracking user/assistant messages
  - Автоматическая суммаризация через /v1/chat/completions API
  - Notification system для уведомлений о суммаризации

- **Backend (Go)**:
  - Добавлено поле Options map[string]interface{} в models.ChatCompletionRequest
  - Converter применяет num_ctx, top_k, repeat_penalty из req.Options
  - Helper functions intPtr(), float64Ptr() для конвертации типов

- **CSS Updates**:
  - Полная переработка web/css/model-panel.css для compact design
  - Responsive breakpoints для мобильных устройств
  - Dark/Light mode совместимость с новыми стилями');
	