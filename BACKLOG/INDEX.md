# Индекс задач AIGateway Platform

📋 **Связанный документ**: [Plan.md](../Plan.md) - детальный план разработки со ссылками на все задачи BACKLOG

## Структура BACKLOG

Все задачи организованы по категориям с уникальными идентификаторами.

### Коды категорий:
- **CMD** - Command-line interface, основные команды
- **CONFIG** - Конфигурация и настройки
- **API** - HTTP API endpoints и handlers  
- **CLIENT** - Клиенты (Ollama client)
- **CORE** - Основная логика (конвертеры, model manager, error handling)
- **AUTH** - Аутентификация и авторизация
- **STORAGE** - Хранение данных
- **TUI** - Terminal User Interface (Bubbletea)
- **MONITORING** - Мониторинг, метрики и производительность
- **TEST** - Тестирование и безопасность
- **DEPLOY** - Деплой и контейнеризация
- **DOCS** - Документация

## Список всех задач

### Фаза 1-2: Базовая инфраструктура
- **CMD-01** - Создание структуры проекта *(1-2ч)*
- **CONFIG-01** - Конфигурация и зависимости *(2-3ч)*
- **API-01** - HTTP сервер *(3-4ч)*
- **API-02** - Базовые эндпоинты *(4-5ч)*

### Фаза 3: Ollama Integration
- **CLIENT-01** - HTTP клиент для Ollama *(5-6ч)*
- **TEST-01** - Интеграционные тесты с Ollama *(3-4ч)*

### Фаза 4: Core Logic
- **CORE-01** - Модели данных *(3-4ч)*
- **CORE-02** - Request/Response конвертеры *(6-8ч)*
- **TEST-02** - Тестирование конвертеров *(4-5ч)*

### Фаза 5: Model Management
- **CORE-03** - Model Manager *(4-5ч)*
- **CONFIG-02** - Конфигурируемый маппинг моделей *(3-4ч)*

### Фаза 6: Integration
- **API-03** - Полная интеграция chat completions *(4-6ч)* **[КРИТИЧЕСКИЙ]**
- **TEST-04** - 100% Test Coverage для MVP *(8-12ч)* **[КРИТИЧЕСКИЙ]**
- **API-04** - Реализация /v1/models эндпоинта *(2-3ч)*

### Фаза 7: Reliability
- **CORE-04** - Централизованная обработка ошибок *(3-4ч)*
- **CORE-05** - Retry логика и circuit breaker *(4-5ч)*

### Фаза 8: Security & Auth
- **STORAGE-01** - Storage и Data Models для API Keys *(4-5ч)*
- **AUTH-01** - API Key Manager Core *(6-8ч)* **[КРИТИЧЕСКИЙ]**
- **AUTH-02** - Authentication Middleware *(4-5ч)* **[КРИТИЧЕСКИЙ]** 
- **AUTH-03** - Rate Limiting per API Key *(5-6ч)*
- **API-05** - API Key Management API *(4-5ч)*
- **TUI-01** - TUI Integration для API Keys *(5-6ч)*
- **TEST-03** - Testing и Security *(6-8ч)* **[КРИТИЧЕСКИЙ]**

### Фаза 9: Extended API
- **API-06** - Поддержка embeddings *(4-5ч)*
- **API-07** - Legacy completions API *(3-4ч)*

### Фаза 10: Performance & Monitoring
- **MONITORING-01** - Performance optimization *(4-6ч)*
- **MONITORING-02** - Metrics и мониторинг *(5-6ч)*

### Фаза 11: Terminal UI
- **TUI-02** - Базовая TUI инфраструктура *(4-5ч)*
- **TUI-03** - Dashboard экран *(6-8ч)*
- **TUI-04** - Request Monitor *(5-6ч)*
- **TUI-05** - Model Manager экран *(4-5ч)*
- **TUI-06** - API Keys Manager экран *(6-7ч)* **[КРИТИЧЕСКИЙ]**
- **TUI-07** - Configuration Viewer *(3-4ч)*
- **TUI-08** - Logs Viewer *(4-5ч)*
- **TUI-09** - Server Control *(3-4ч)*

### Фаза 12: TUI Integration
- **MONITORING-03** - Metrics Collection *(5-6ч)*
- **TUI-10** - Real-time Communication *(4-5ч)*
- **TUI-11** - Advanced TUI Features *(3-4ч)*

### Фаза 13: Production
- **DEPLOY-01** - Docker и деплой *(3-4ч)*
- **DOCS-01** - Документация *(4-5ч)*

## Критические задачи (обязательно для MVP)
- **API-03** - Полная интеграция chat completions
- **TEST-04** - 100% Test Coverage для MVP
- **AUTH-01** - API Key Manager Core  
- **AUTH-02** - Authentication Middleware
- **TEST-03** - Testing и Security
- **TUI-06** - API Keys Manager экран

## Общие оценки времени
- **MVP с 100% test coverage (Фазы 1-6)**: 40-55 часов
- **MVP + API Keys (Фазы 1-8)**: 75-95 часов  
- **Production-ready с TUI (Фазы 1-12)**: 130-155 часов
- **Full-featured (Все фазы)**: 145-170+ часов

