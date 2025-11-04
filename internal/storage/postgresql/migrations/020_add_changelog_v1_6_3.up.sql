
INSERT INTO changelogs (version, release_date, content) VALUES
('1.6.3', '2025-10-12', '## [1.6.3] - 2025-10-12

### Added
- **Scheduled Reports**: Автоматическая генерация и отправка отчетов по email
  - Cron-based scheduler с поддержкой гибких расписаний
  - Report Generator для Usage, Performance и System Health отчетов
  - Email Mailer с SMTP отправкой (поддержка TLS/SSL)
  - HTML email templates с современным дизайном
  - Поддержка множественных recipients

- **Report Types**: Три типа отчетов
  - **Usage Report**: Total requests, tokens, top models, top users
  - **Performance Report**: Latency metrics (P50/P95/P99), slow requests, error rates
  - **System Health Report**: Uptime, memory, goroutines, database stats

### Changed
- **Configuration**: Добавлена секция reports
  - enabled: включение/отключение scheduler
  - smtp: SMTP конфигурация (host, port, username, password, TLS)
  - schedules: массив scheduled reports с cron expressions
  - Поддержка переменных окружения для паролей

- **Database Interface**: Добавлены методы для reports statistics
  - GetUsageStats(): агрегированная статистика использования
  - GetPerformanceStats(): performance metrics за период
  - CountActiveUsers(), CountTotalUsers(), -- CountActiveAPIKeys()

### Technical
- Новый пакет internal/reports с полной реализацией
  - scheduler.go: Cron scheduler с github.com/robfig/cron/v3
  - generator.go: Report generation с embedded HTML templates
  - mailer.go: SMTP email delivery с gopkg.in/gomail.v2
  - types.go: Report models и data structures
  - templates/*.html: Beautiful HTML email templates

- SQLite реализация reports statistics методов
  - Percentile calculations для latency metrics
  - JOIN queries для user/model aggregations
  - Optimized queries для больших datasets

- PostgreSQL stub реализация (для будущей поддержки)
- Зависимости: github.com/robfig/cron/v3, gopkg.in/gomail.v2
- Graceful shutdown для scheduler')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	