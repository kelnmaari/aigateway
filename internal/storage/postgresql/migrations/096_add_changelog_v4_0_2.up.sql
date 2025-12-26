INSERT INTO changelogs (version, release_date, content) VALUES
('4.0.2', '2025-12-26', '## [4.0.2] - 2025-12-26

### Changed

- **About Page Layout**: увеличена ширина страницы до 1600px
- **Repository Links**: ссылки обновлены на GitLab (https://gitlab.alexue4.dev/KelnMaari/ollama-openai-proxy)

### Fixed

- **LLM Review Suggestions**: промпты обновлены для запроса примеров кода в предложениях по исправлению')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;

