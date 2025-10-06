# Архитектура Ollama-OpenAI Proxy

## Обзор проекта

Ollama-OpenAI Proxy - это серверное приложение на Go, которое предоставляет OpenAI-совместимый API для работы с локальными моделями Ollama. Приложение служит мостом между клиентами, ожидающими OpenAI API, и локальным сервером Ollama.

## Архитектурная диаграма

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Web UI (Chat)  │    │  Terminal UI    │    │  Web UI (Admin) │
│  (Vanilla JS)   │    │  (Bubbletea)    │    │  (Dashboard)    │
└────────┬────────┘    └────────┬────────┘    └────────┬────────┘
         │                      │                       │
         │                      ▼                       │
         │               ┌─────────────────┐           │
         └──────────────▶│  HTTP Server    │◀──────────┘
┌─────────────────┐      │  (Gin Router)   │
│   Client Apps   │─────▶│  + Middleware   │
│  (OpenAI API)   │      └────────┬────────┘
└─────────────────┘               │
                                  ▼
                    ┌──────────────────────────┐
                    │   Ollama-OpenAI Proxy    │
                    │   ┌──────────────────┐   │
                    │   │ Authentication   │   │    ┌──────────────────┐
                    │   │ & Authorization  │   │───▶│   Database       │
                    │   └──────────────────┘   │    │  (SQLite/PG)     │
                    │   ┌──────────────────┐   │    │  • Users         │
                    │   │ Request/Response │   │    │  • Tenants       │
                    │   │    Converter     │   │    │  • API Keys      │
                    │   └──────────────────┘   │    │  • Conversations │
                    │   ┌──────────────────┐   │    │  • Messages      │
                    │   │ Ollama Client    │   │    └──────────────────┘
                    │   └──────────────────┘   │
                    └──────────┬───────────────┘
                               │
                               ▼
                    ┌──────────────────┐
                    │   Ollama Server  │
                    │   (Local LLMs)   │
                    └──────────────────┘
```

## Компоненты системы

### 1. HTTP Server Layer

- **Назначение**: Обработка входящих HTTP запросов с OpenAI API эндпоинтами
- **Технологии**: Gin Web Framework
- **Ответственность**:
  - Роутинг запросов
  - Валидация входящих данных
  - Аутентификация (API keys)
  - CORS обработка
  - Middleware для логирования и метрик

### 2. OpenAI API Compatibility Layer

- **Назначение**: Эмуляция OpenAI API эндпоинтов
- **Поддерживаемые эндпоинты**:
  - `POST /v1/chat/completions` - основной эндпоинт для chat completions
  - `GET /v1/models` - список доступных моделей
  - `POST /v1/completions` - text completions (legacy)
  - `POST /v1/embeddings` - генерация embedding'ов
- **Ответственность**:
  - Парсинг OpenAI формата запросов
  - Валидация параметров
  - Формирование OpenAI-совместимых ответов

### 3. Request/Response Converter

- **Назначение**: Конвертация между форматами OpenAI и Ollama
- **Ответственность**:
  - Трансформация структур данных
  - Маппинг параметров запросов
  - Конвертация streaming responses
  - Обработка специфичных для OpenAI параметров
  - **🔧 КРИТИЧНО: Tool Calling (Function Calling) Support**:
    - Ollama возвращает `tool_calls` в **ДВУХ местах**: `message.tool_calls` и `response.tool_calls`
    - OpenAI streaming требует `index` поле в каждом `tool_call` (без `omitempty`!)
    - StreamConverter **ОБЯЗАН** запоминать состояние (`hadToolCalls`) между chunks
    - `finish_reason` ДОЛЖЕН быть `"tool_calls"` если инструменты были вызваны в **ЛЮБОМ** chunk
    - Вызов `Reset()` на StreamConverter **перед каждым новым streaming запросом**

### 4. Ollama Client

- **Назначение**: Взаимодействие с Ollama API
- **Ответственность**:
  - HTTP клиент для общения с Ollama
  - Connection pooling и retry логика
  - Обработка streaming responses
  - Управление таймаутами

### 5. Model Manager

- **Назначение**: Управление доступными моделями и их маппинг
- **Ответственность**:
  - Получение списка моделей из Ollama
  - Маппинг имен моделей (OpenAI ↔ Ollama)
  - Кеширование списка моделей
  - Проверка доступности моделей

### 6. Configuration Manager

- **Назначение**: Управление конфигурацией приложения
- **Источники конфигурации**:
  - Environment variables
  - YAML/JSON config файлы
  - Command line arguments
- **Параметры**:
  - Ollama server URL
  - Proxy server port
  - Model mappings
  - API keys
  - Logging settings
  - Timeouts и retry settings

### 7. Logging & Monitoring

- **Назначение**: Логирование и мониторинг системы
- **Компоненты**:
  - Structured logging (JSON format)
  - Request/response logging
  - Error tracking
  - Performance metrics
  - Health check endpoints

### 8. API Key Manager

- **Назначение**: Управление API ключами и авторизацией доступа
- **Ответственность**:
  - Создание, обновление и удаление API ключей
  - Валидация API ключей в запросах
  - Управление правами доступа к моделям
  - Rate limiting для каждого API ключа
  - Логирование использования ключей
  - Ротация и истечение ключей
- **Хранение данных**:
  - JSON файлы для простого развертывания
  - SQLite для production (опционально)
  - In-memory cache для быстрой валидации
- **Структура API ключа**:
  - Уникальный идентификатор ключа
  - Хеш ключа (bcrypt/argon2)
  - Список разрешенных моделей (или "*" для всех)
  - Rate limit settings (requests per minute/hour)
  - Metadata (название, описание, создатель)
  - Временные метки (created_at, expires_at)
  - Статус (active/disabled/expired)

### 9. Terminal User Interface (TUI)

- **Назначение**: Интерактивный терминальный интерфейс для администрирования и мониторинга
- **Технологии**: Bubbletea ecosystem (bubbletea, lipgloss, bubbles)
- **Ответственность**:
  - Real-time мониторинг метрик и статистики
  - Отображение активных запросов и их статусов
  - Управление конфигурацией сервера
  - Просмотр логов в реальном времени
  - Мониторинг состояния Ollama моделей
  - Интерактивное управление сервером (start/stop/restart)
  - **Управление API ключами через TUI**
- **Компоненты TUI**:
  - Dashboard экран с общей статистикой
  - Request monitor с таблицей активных запросов
  - Model manager для управления моделями
  - **API Keys manager для CRUD операций с ключами**
  - Configuration viewer/editor
  - Logs viewer с фильтрацией и поиском
  - Performance metrics graphs (ASCII charts)

### 10. Database Layer (Version 1.6.0+)

- **Назначение**: Абстрактный слой для работы с реляционными БД
- **Поддерживаемые БД**:
  - SQLite (по умолчанию, embedded)
  - PostgreSQL (для production)
- **Ответственность**:
  - Единый интерфейс для всех CRUD операций
  - Автоматические миграции при старте
  - Connection pooling и prepared statements
  - Transaction support
  - Type-safe queries
- **Хранимые данные**:
  - Users (аутентификация)
  - Tenants (multi-tenancy)
  - Tenant Members (роли и permissions)
  - API Keys (с scoping: personal/tenant)
  - Conversations (чаты)
  - Messages (история сообщений)
  - Usage Statistics (метрики использования)

### 11. User Authentication System (Version 1.6.0+)

- **Назначение**: Управление пользователями и аутентификацией
- **Технологии**: JWT tokens, bcrypt
- **Ответственность**:
  - User registration и login
  - Password hashing (bcrypt cost 12)
  - JWT token generation/validation
  - Session management
  - Brute force protection
  - Password reset flow (future)
- **Особенности**:
  - Short-lived access tokens (15 min)
  - Refresh tokens (7 days)
  - Token rotation при refresh
  - Blacklist для revoked tokens

### 12. Multi-Tenancy System (Version 1.6.0+)

- **Назначение**: Управление workspace'ами и командами
- **Гибридная модель**:
  - **Personal Workspace**: Auto-created для каждого user
  - **Organization Tenants**: Создаются вручную, multi-user
- **Role-Based Access Control**:
  - **Owner**: Full control над tenant
  - **Admin**: Manage members и API keys
  - **Member**: Use resources (chat, API keys)
  - **Viewer**: Read-only access
- **API Key Scoping**:
  - **Personal Keys**: Принадлежат user, видны только ему
  - **Tenant Keys**: Принадлежат organization, доступны всем members

### 13. Chat UI (Version 1.6.0+)

- **Назначение**: ChatGPT-like интерфейс для взаимодействия с моделями
- **Технологии**: Vanilla JavaScript (zero NPM deps)
- **Ответственность**:
  - Real-time streaming responses (Server-Sent Events)
  - Markdown rendering (marked.js)
  - Code syntax highlighting (highlight.js)
  - LaTeX math rendering (katex, optional)
  - Conversation management (save/load/export)
  - Model selector
  - Advanced settings (temperature, top_p, system prompt)
- **Особенности**:
  - Single binary deployment (embed в Go)
  - CDN libraries для offline работы
  - Web Components для модульности
  - Responsive design
  - Dark/Light themes

### 14. User Dashboard (Version 1.6.0+)

- **Назначение**: Личный кабинет пользователя
- **Компоненты**:
  - **Profile Management**: Редактирование профиля, смена пароля, avatar
  - **Personal API Keys**: CRUD для личных ключей
  - **Usage Statistics**: Charts и графики использования
  - **Tenant Switching**: Переключение между workspace'ами
  - **Preferences**: Theme, language, notifications
  - **Conversations History**: Список всех чатов
- **Charts & Analytics**:
  - Requests over time
  - Tokens usage
  - Models distribution
  - Response latency
  - Per-key breakdown

## Паттерны проектирования

### 1. Adapter Pattern

- Используется в Request/Response Converter для адаптации форматов

### 2. Strategy Pattern

- Для разных стратегий обработки различных типов запросов

### 3. Factory Pattern

- Для создания соответствующих конвертеров и обработчиков

### 4. Singleton Pattern

- Для Configuration Manager и Ollama Client

## Технологический стек

- **Язык**: Go 1.25+
- **HTTP Framework**: Gin
- **HTTP Client**: net/http с custom pooling
- **Configuration**: Viper
- **Logging**: logrus или zap
- **Testing**: testify
- **Authentication**:
  - `bcrypt` - хеширование паролей (cost 12)
  - `golang.org/x/crypto/argon2` - альтернативное хеширование
  - `github.com/golang-jwt/jwt/v5` - JWT tokens (Version 1.6.0+)
  - `github.com/google/uuid` - UUID generation
- **Storage**:
  - JSON файлы для простого развертывания (deprecated в 1.6.0+)
  - `modernc.org/sqlite` - embedded SQLite (default в 1.6.0+)
  - `github.com/lib/pq` - PostgreSQL driver (1.6.0+)
- **Rate Limiting**:
  - `golang.org/x/time/rate` - token bucket rate limiter
  - Redis для distributed rate limiting (опционально)
- **Terminal UI**: Bubbletea ecosystem
  - `bubbletea` - основная TUI библиотека
  - `lipgloss` - стилизация и layout
  - `bubbles` - готовые UI компоненты
- **Web UI (Version 1.6.0+)**:
  - **Frontend**: Vanilla JavaScript (ES6+)
  - **Markdown**: marked.js (~20KB)
  - **Syntax Highlighting**: highlight.js (~100KB)
  - **Math Rendering**: katex (~200KB, optional)
  - **Charts**: Chart.js или lightweight alternative
  - **Icons**: Bootstrap Icons или Feather Icons
  - **Deployment**: Embedded в Go binary (`embed.FS`)
- **Build**: Docker multi-stage build, Makefile

## Структура проекта

```
ollama-openai-proxy/
├── cmd/
│   ├── server/
│   │   └── main.go              # HTTP сервер (основной)
│   ├── tui/
│   │   └── main.go              # TUI приложение
│   ├── webui/
│   │   └── main.go              # WebUI server (dashboard + chat)
│   └── migrate/                 # (1.6.0+)
│       └── main.go              # Database migration tool
├── internal/
│   ├── api/
│   │   ├── handlers/            # HTTP handlers
│   │   ├── middleware/          # HTTP middleware
│   │   └── router/              # Роутинг
│   ├── auth/                    # Authentication & Authorization
│   │   ├── apikey/              # API Key manager
│   │   ├── jwt/                 # JWT token manager (1.6.0+)
│   │   ├── middleware/          # Auth middleware
│   │   └── ratelimit/           # Rate limiting
│   ├── client/
│   │   └── ollama/              # Ollama client
│   ├── converter/               # Request/Response converters
│   ├── models/                  # Data models
│   ├── config/                  # Configuration management
│   ├── manager/
│   │   └── model/               # Model manager
│   ├── metrics/                 # Metrics collection & storage
│   ├── storage/                 # Data storage layer (1.6.0+)
│   │   ├── interfaces.go        # Database interface
│   │   ├── factory.go           # Factory для создания DB
│   │   ├── sqlite/              # SQLite implementation
│   │   │   └── sqlite.go
│   │   ├── postgres/            # PostgreSQL implementation
│   │   │   └── postgres.go
│   │   ├── migrations/          # Database migrations
│   │   │   └── migrations.go
│   │   └── json_storage.go      # Legacy JSON storage (deprecated)
│   ├── tui/
│   │   ├── components/          # TUI компоненты
│   │   ├── models/              # TUI модели состояния
│   │   ├── styles/              # Lipgloss стили
│   │   └── views/               # TUI экраны/представления
│   └── web/                     # Web UI (1.6.0+)
│       ├── static/
│       │   ├── css/
│       │   │   ├── style.css
│       │   │   ├── chat.css
│       │   │   └── dashboard.css
│       │   ├── js/
│       │   │   ├── chat/
│       │   │   │   ├── chat-app.js
│       │   │   │   ├── markdown-renderer.js
│       │   │   │   └── conversation-manager.js
│       │   │   ├── dashboard/
│       │   │   │   ├── overview.js
│       │   │   │   ├── api-keys.js
│       │   │   │   ├── usage-stats.js
│       │   │   │   └── tenant-selector.js
│       │   │   ├── auth.js
│       │   │   └── utils.js
│       │   ├── vendor/          # CDN libraries (offline mode)
│       │   │   ├── marked.min.js
│       │   │   ├── highlight.min.js
│       │   │   └── katex.min.js
│       │   ├── index.html       # Main dashboard
│       │   └── chat.html        # Chat interface
│       └── embed.go             # Embed static files
├── pkg/
│   └── types/                   # Общие типы данных
├── configs/                     # Конфигурационные файлы
│   ├── dev.yaml
│   └── production.yaml.example
├── data/                        # Runtime data (1.6.0+)
│   ├── proxy.db                 # SQLite database
│   └── api_keys.json.backup     # Legacy backup
├── docs/                        # Documentation
│   ├── API.md
│   ├── Authentication.md
│   └── Multi-Tenancy.md
├── docker/
│   └── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

## Безопасность

### Authentication & Authorization (Version 1.6.0+)

- **User Authentication**:
  - JWT-based authentication (access + refresh tokens)
  - Password hashing (bcrypt cost 12)
  - Brute force protection (rate limiting на login)
  - Token rotation при refresh
  - Blacklist для revoked tokens
  
- **Multi-Tenancy Security**:
  - RBAC (Role-Based Access Control)
  - Tenant isolation на уровне БД
  - Permission checks для каждого действия
  - Audit log всех tenant операций

- **API Key аутентификация**: Поддержка OpenAI-совместимых API ключей
  - Хеширование ключей с использованием bcrypt/argon2
  - Безопасное хранение в SQLite/PostgreSQL
  - Валидация ключей через middleware
  - API Key Scoping (personal vs tenant keys)
  
- **Model-based Authorization**: Ограничение доступа к моделям по API ключу
  - Whitelist моделей для каждого ключа
  - Динамическая проверка прав доступа
  - Логирование попыток несанкционированного доступа
  
- **Rate Limiting**: Ограничение запросов на IP/API key
  - Индивидуальные лимиты для каждого API ключа
  - Token bucket algorithm
  - Graceful handling превышения лимитов
  - Bypass rate limits для admin keys
  
- **Input Validation**: Валидация всех входящих данных
  - SQL injection prevention (prepared statements)
  - XSS protection (output sanitization)
  - CSRF protection (для Web UI)
  - Content-Type validation
  
- **CORS**: Настройка CORS политик
- **TLS**: Поддержка HTTPS (reverse proxy recommended)
  
- **API Key Management Security**:
  - Secure key generation (crypto/rand)
  - Key expiration и rotation
  - Audit logging всех операций с ключами
  - Protection от timing attacks
  - Plain key показывается ТОЛЬКО ОДИН РАЗ при создании

## Производительность

- **Connection Pooling**: Переиспользование соединений с Ollama
- **Caching**: Кеширование списка моделей
- **Streaming**: Поддержка streaming responses
- **Graceful Shutdown**: Корректное завершение работы
- **Resource Management**: Управление ресурсами и memory leaks

## Мониторинг и наблюдаемость

- **Health Checks**: `/health` эндпоинт для проверки состояния
- **Metrics**: Prometheus metrics для мониторинга
- **Distributed Tracing**: OpenTelemetry integration
- **Error Tracking**: Централизованная обработка ошибок
- **Terminal UI**: Интерактивный мониторинг через Bubbletea TUI
  - Real-time dashboard с метриками
  - Live view активных запросов
  - Визуализация производительности (ASCII charts)
  - Интерактивное управление конфигурацией
