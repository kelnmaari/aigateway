
INSERT OR REPLACE INTO changelogs (version, release_date, content) VALUES
('1.10.5', '2025-10-25', '## [1.10.5] - 2025-10-25

### Changed
- **WEB-FETCH-01: Full Content Processing** 🚀
  - **BREAKING CHANGE**: Web fetcher теперь передает полное содержимое страницы модели без обрезания
  - Удален hardcoded truncation до 3000 символов
  - Добавлен параметр TruncateLength в ProcessMessageOptions для гибкого контроля
  - Default: TruncateLength: 0 (без ограничений) - оптимально для моделей с большим контекстом (128K+)
  - Добавлены поля WordCount и Language в WebPage для статистики
  - Логирование truncation когда применяется

### Technical
- Структуры данных: WebPage добавлены WordCount int и Language string, ParsedContent добавлено Language string, ProcessMessageOptions добавлено TruncateLength int (0 = без ограничений)
- Поведение по умолчанию: Chat Integration TruncateLength: 0 - полный контент для LLM, Старое поведение можно вернуть: TruncateLength: 3000
- Улучшения: Показ статистики для больших страниц (>10K chars), Детальное логирование при truncation, Language detection из HTML metadata');
	