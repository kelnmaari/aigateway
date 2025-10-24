# 🦙 Ollama-OpenAI Proxy

> **Enterprise-grade OpenAI-compatible API for local Ollama models**  
> Multi-tenancy • ChatGPT-like UI • JWT Auth • Performance Monitoring • GPU Metrics

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-1.9.3-brightgreen.svg)](VERSION)

---

## 📖 Что это?

**Ollama-OpenAI Proxy** — production-ready Go приложение, предоставляющее **OpenAI-совместимый API** для локальных Ollama моделей. Идеальное решение для интеграции LLM в корпоративную инфраструктуру.

### 🎯 Ключевые возможности

#### 🚀 Основные

- ✅ **Full OpenAI API** - `/v1/chat/completions`, `/v1/models`, `/v1/embeddings`, `/v1/completions`
- 🔥 **Real-time Streaming** - Server-Sent Events (SSE) для живых ответов
- 🛠️ **Function Calling** - Инструменты в стиле OpenAI с автомаршрутизацией моделей
- 💬 **ChatGPT-like WebUI** - Полноценный чат-интерфейс с историей разговоров

#### 🔐 Enterprise Security

- 🔒 **JWT Authentication** - Полноценная система аутентификации пользователей
- 👥 **Multi-Tenancy** - Организации с членством и RBAC (Owner/Admin/Member/Viewer)
- 🔑 **API Key Management** - Personal & Tenant API keys с гранулярными правами
- ⏱️ **Rate Limiting** - Настраиваемые лимиты per-key/per-tenant

#### 📊 Мониторинг & Observability

- 📈 **MoniGo Dashboard** - Real-time performance monitoring (CPU, Memory, Goroutines)
- 🎮 **NVIDIA GPU Metrics** - Multi-GPU мониторинг через `nvidia-smi` (БЕЗ CGO!)
- 📉 **Prometheus Metrics** - Полная интеграция для Grafana
- 🔍 **OpenTelemetry** - Distributed tracing (Jaeger/Zipkin)
- 📋 **Enhanced Logs** - Structured logging с SSE real-time streaming

#### ⚡ Performance & UX

- 🎨 **Dynamic Model Parameters** - Настройка температуры, top_p, context window в UI
- 📊 **Context Tracking** - Real-time отслеживание использования контекста
- 🔄 **Auto-Summarization** - Автоматическое сжатие при переполнении контекста
- 💾 **Backup & Restore** - Автоматическое резервное копирование БД
- 🌐 **WebUI + TUI** - Два интерфейса управления на выбор

---

## 🚀 Быстрый старт

### Требования

- **Go 1.25+** (для сборки из исходников)
- **Ollama** server ([Скачать Ollama](https://ollama.ai/))
- Хотя бы одна модель: `ollama pull llama3.2`

### Установка

#### Docker Compose (рекомендуется)

```bash
git clone https://github.com/yourusername/ollama-openai-proxy.git
cd ollama-openai-proxy

# Build & Start
docker compose --profile build up --build

# Сервер: http://localhost:8080
# WebUI: http://localhost:8080/login
```

#### Сборка из исходников

```bash
git clone https://github.com/yourusername/ollama-openai-proxy.git
cd ollama-openai-proxy

# Install dependencies
go mod tidy

# Build all binaries
./build.sh all

# Или через Docker
docker compose --profile build up --build

# Binaries в папке dist/
```

### Первый запуск

1. **Запустите Ollama**:

```bash
ollama serve
ollama pull llama3.2  # Или любую другую модель
```

2. **Запустите Proxy Server** (WSL/Linux рекомендуется для GPU monitoring):

```bash
# Linux/macOS
./dist/ollama-proxy-linux-amd64 -config configs/dev.yaml

# Windows (без GPU monitoring)
dist\ollama-proxy-windows-amd64.exe -config configs/dev.yaml

# Server: http://localhost:8080
```

3. **Bootstrap Admin User**:

При первом запуске система предложит создать admin пользователя через специальный токен. Следуйте инструкциям в логах.

4. **Откройте WebUI**:

```
http://localhost:8080/login
```

Зарегистрируйтесь и начните использовать!

---

## 💬 ChatGPT-like WebUI

### Основные возможности

- 🎨 **Modern Dark Theme** - Красивый, отзывчивый интерфейс
- 💬 **Real-time Chat** - Streaming ответы с поддержкой Markdown
- 📚 **Conversations History** - Сохранение и восстановление диалогов
- 🎛️ **Dynamic Parameters** - Настройка model parameters прямо в чате
- 📊 **Context Tracking** - Визуальный индикатор использования контекста
- 🔄 **Auto-Summarization** - Автоматическое сжатие при заполнении

### Страницы WebUI

| Страница | Описание | Доступ |
|----------|----------|--------|
| **Chat** | ChatGPT-подобный интерфейс | Все пользователи |
| **Dashboard** | Статистика использования, быстрые действия | Все пользователи |
| **Profile** | Управление профилем, смена пароля | Все пользователи |
| **API Keys** | Personal & Tenant API keys management | Все пользователи |
| **Tenants** | Управление организациями и участниками | Owner/Admin |
| **Usage** | Детальная аналитика использования | Все пользователи |
| **Admin** | Models, System, Logs, Performance, GPU | Admin only |
| **MCP** | Catalog MCP серверов (справочник) | Все пользователи |
| **About** | Changelog и информация о системе | Все пользователи |

### Dynamic Model Parameters

В чате доступна панель настройки параметров:

- **Temperature** (0.0 - 2.0) - Креативность ответов
- **Top P** (0.0 - 1.0) - Nucleus sampling
- **Top K** (0 - 100) - Ограничение словаря
- **Context Window** (512 - 128000) - Размер контекста
- **Max Tokens** - Максимум токенов в ответе

**Quick Presets:**

- 🎨 Creative (temp: 1.2, top_p: 0.95)
- ⚖️ Balanced (temp: 0.7, top_p: 0.9)
- 🎯 Precise (temp: 0.3, top_p: 0.5)
- 💻 Coding (temp: 0.2, top_p: 0.1)

Настройки сохраняются в `localStorage` браузера.

---

## 🔐 Multi-Tenancy & Authentication

### User Roles

| Role | Описание | Права |
|------|----------|-------|
| **Owner** | Создатель организации | Полный доступ, добавление админов |
| **Admin** | Администратор | Управление участниками, API keys |
| **Member** | Участник | Доступ к tenant resources |
| **Viewer** | Наблюдатель | Только чтение |

### Организации (Tenants)

- **Personal Workspace** - Автоматически для каждого пользователя
- **Organization Tenants** - Создаются вручную для команд
- **API Keys Scoping** - Personal (user-scoped) + Tenant (org-scoped)
- **Members Management** - Добавление/удаление участников с ролями

### API Keys Management

**Personal API Keys** - привязаны к пользователю:

   ```bash
# Создать в WebUI: API Keys → Personal Keys → Create New
# Использование:
curl -H "Authorization: Bearer sk-your-personal-key" \
  http://localhost:8080/v1/chat/completions
```

**Tenant API Keys** - привязаны к организации:

```bash
# Создать в WebUI: API Keys → Tenant Keys → Create New
# Использование аналогично
```

Каждый ключ имеет:

- ✅ Список разрешенных моделей
- ⏱️ Rate limits (requests/min, requests/hour)
- 📅 Expiration date
- 🔒 Enable/Disable toggle

---

## 📊 Performance Monitoring

### MoniGo Dashboard

Доступен на отдельном порту **:9091** для админов:

```
http://localhost:9091
```

**Quick Stats Cards** (интегрированы в Admin → System):

- 💻 **CPU Usage** - Real-time загрузка процессора
- 🧠 **Memory Usage** - Использование RAM
- 🔄 **Goroutines** - Активные горутины
- ✅ **System Health** - Общее состояние

### NVIDIA GPU Monitoring

**Multi-GPU поддержка** через `nvidia-smi` (Linux/macOS only):

Метрики на каждую GPU:

- 🌡️ **Temperature** (с цветовыми индикаторами)
- ⚡ **Power Usage** (W / % от лимита)
- 📊 **GPU Load** (utilization %)
- 💾 **VRAM Usage** (used / total GB)
- 🔧 **Clock Speed** (MHz)
- 💨 **Fan Speed** (%)

Автообновление каждые 5 секунд. Unified card дизайн для всех GPU.

**Примечание:** Windows использует stub версию (GPU monitoring disabled).

### OpenTelemetry

Distributed tracing для production мониторинга:

```yaml
observability:
  enabled: true
  tracing:
    enabled: true
    exporter: "jaeger"
    jaeger_endpoint: "http://localhost:14268/api/traces"
    sampling_rate: 0.1  # 10% запросов
   ```

---

## 📚 API Documentation

### Supported Endpoints

| Endpoint | Method | Description | Auth Required |
|----------|--------|-------------|---------------|
| `/v1/chat/completions` | POST | Chat completions с streaming | ✅ Yes |
| `/v1/models` | GET | Список доступных моделей | ✅ Yes |
| `/v1/embeddings` | POST | Генерация embeddings | ✅ Yes |
| `/v1/completions` | POST | Legacy text completions | ✅ Yes |
| `/health` | GET | Health check | ❌ No |
| `/api/system/changelogs` | GET | Changelog истории | ✅ Yes |
| `/api/gpu/metrics` | GET | NVIDIA GPU метрики | ✅ Admin |
| `/metrics` | GET | Prometheus metrics | ❌ No |

### Chat Completions Example

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "model": "llama3.2",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "Explain quantum computing"}
    ],
    "temperature": 0.7,
    "max_tokens": 500,
    "stream": true
  }'
```

**Streaming response:**

```
data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1699999999,"model":"llama3.2","choices":[{"index":0,"delta":{"role":"assistant","content":"Quantum"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1699999999,"model":"llama3.2","choices":[{"index":0,"delta":{"content":" computing"},"finish_reason":null}]}

...

data: [DONE]
```

Полная документация: [docs/API_DOCUMENTATION.md](docs/API_DOCUMENTATION.md)

---

## ⚙️ Конфигурация

### Основная конфигурация

`configs/dev.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "30s"
  write_timeout: "180s"

ollama:
  url: "http://localhost:11434"
  timeout: "180s"
  retry_attempts: 3

database:
  type: "sqlite"  # или "postgresql"
  sqlite:
    path: "data/proxy.db"

auth:
  jwt_secret: "change-me-in-production"
  token_expiration: "24h"
  refresh_expiration: "7d"

observability:
  enabled: true
  tracing:
    enabled: true
    exporter: "jaeger"
    jaeger_endpoint: "http://localhost:14268/api/traces"

performance:
  monigo:
    enabled: true
    port: 9091
  gpu_monitoring:
    enabled: true
    refresh_interval: "5s"

logging:
  level: "info"
  format: "text"
  output: "both"
  file_path: "logs/proxy-dev.log"
```

### Environment Variables

```bash
export PROXY_SERVER_PORT=9000
export PROXY_OLLAMA_URL="http://remote-server:11434"
export PROXY_JWT_SECRET="super-secret-key"
export PROXY_DATABASE_TYPE="postgresql"
```

Полная документация: [docs/CONFIGURATION.md](docs/CONFIGURATION.md)

---

## 🏗️ Архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                    CLIENT APPLICATIONS                       │
│   (Zed, Continue.dev, VSCode, Custom Apps, WebUI Chat)      │
└───────────────────────────┬─────────────────────────────────┘
                            │ OpenAI API + JWT Auth
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                 OLLAMA-OPENAI PROXY (v1.9.3)                │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  HTTP Server (Gin) + WebUI (Embedded)                │  │
│  │  • /v1/chat/completions  • /v1/models                │  │
│  │  • /login  • /chat  • /dashboard  • /admin           │  │
│  └──────────────────────────────────────────────────────┘  │
│                            │                                 │
│  ┌─────────────────────────┴────────────────────────────┐  │
│  │  Middleware Pipeline                                  │  │
│  │  JWT → RBAC → Rate Limit → Tracing → Metrics         │  │
│  └────────────────────────────────────────────────────────┘ │
│                            │                                 │
│  ┌─────────────────────────┴────────────────────────────┐  │
│  │  Business Logic                                       │  │
│  │  • User/Tenant Management  • Conversations            │  │
│  │  • API Keys (Personal/Tenant)  • Context Tracking    │  │
│  │  • Dynamic Parameters  • Auto-Summarization          │  │
│  └────────────────────────────────────────────────────────┘ │
│                            │                                 │
│  ┌─────────────────────────┴────────────────────────────┐  │
│  │  Database (SQLite/PostgreSQL)                        │  │
│  │  • Users  • Tenants  • Members  • API Keys           │  │
│  │  • Conversations  • Messages  • Usage  • Changelogs  │  │
│  └────────────────────────────────────────────────────────┘ │
└───────────────────────────┬─────────────────────────────────┘
                            │ Ollama API Format
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                      OLLAMA SERVER                           │
│            (llama3.2, qwen2.5-coder, etc.)                  │
└─────────────────────────────────────────────────────────────┘

       ┌──────────────────┐        ┌──────────────────┐
       │  MoniGo :9091    │        │  TUI (optional)  │
       │  Performance     │        │  Monitoring      │
       │  Monitoring      │        │  Management      │
       └──────────────────┘        └──────────────────┘

       ┌──────────────────┐
       │  nvidia-smi      │
       │  GPU Metrics     │
       │  (Linux/macOS)   │
       └──────────────────┘
```

Подробная архитектура: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)

---

## 🧪 Тестирование

Проект имеет **высокое покрытие** тестами критических компонентов.

```bash
# Все тесты
go test ./...

# С покрытием
go test -cover ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Race detection
go test -race ./...
```

**Категории тестов:**

- ✅ Unit Tests (auth, converter, client)
- ✅ Integration Tests (API handlers, middleware)
- ✅ Security Tests (timing attacks, brute force)
- ✅ Performance Tests (rate limiting, concurrency)

---

## 🚀 Deployment

### Systemd Service (Linux)

```bash
# Скопируйте пример
sudo cp ollama-openai-proxy.service /etc/systemd/system/

# Отредактируйте пути и настройки
sudo nano /etc/systemd/system/ollama-openai-proxy.service

# Включите и запустите
sudo systemctl enable ollama-openai-proxy
sudo systemctl start ollama-openai-proxy
sudo systemctl status ollama-openai-proxy
```

Подробная инструкция: [SYSTEMD_INSTALL.md](SYSTEMD_INSTALL.md)

### Docker Production

```bash
# Build production image
docker compose -f docker-compose.yml up -d

# Или через build скрипт
./docker/build.sh
```

---

## 🔧 Troubleshooting

### "Connection refused" to Ollama

```bash
# Проверьте Ollama
curl http://localhost:11434/api/tags

# Запустите если не работает
ollama serve
```

### API key not working

- Проверьте header: `Authorization: Bearer sk-xxx`
- Проверьте права доступа к модели
- Проверьте rate limits

### GPU monitoring не работает

**Linux/macOS:**

```bash
# Проверьте nvidia-smi
nvidia-smi

# Должен вывести список GPU
```

**Windows:** GPU monitoring не поддерживается (используется stub).

Полное руководство: [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)

---

## 🤖 AI Code Review (GitLab CI/CD)

Интегрируйте AI code review в ваш GitLab CI/CD pipeline используя Ollama-OpenAI Proxy!

### ✨ Возможности

- 🔍 **Inline Review** - Построчные комментарии к проблемным местам
- 📝 **Summary Review** - Общий обзор изменений MR
- 🏗️ **Context Review** - Архитектурный анализ
- 💬 **Reply Mode** - Ответы на существующие комментарии
- 🔐 **Self-Hosted** - AI работает через ваш proxy, данные не утекают

### 🚀 Быстрый старт

1. Создайте API ключ в WebUI вашего proxy
2. Добавьте ключ в GitLab CI/CD Variables
3. Настройте `.gitlab-ci.yml`

**Пример:**

```yaml
ai-review:
  stage: review
  image: nikitafilonov/ai-review:latest
  when: manual
  script:
    - ai-review run
  variables:
    # Ваш Ollama-OpenAI Proxy
    LLM__PROVIDER: "OPENAI"
    LLM__HTTP_CLIENT__API_URL: "http://your-proxy:8080/v1"
    LLM__HTTP_CLIENT__API_TOKEN: "$OLLAMA_PROXY_API_KEY"
    LLM__META__MODEL: "qwen2.5-coder:7b"
    
    # GitLab VCS
    VCS__PROVIDER: "GITLAB"
    VCS__PIPELINE__PROJECT_ID: "$CI_PROJECT_ID"
    VCS__PIPELINE__MERGE_REQUEST_ID: "$CI_MERGE_REQUEST_IID"
    VCS__HTTP_CLIENT__API_URL: "$CI_SERVER_URL"
    VCS__HTTP_CLIENT__API_TOKEN: "$CI_JOB_TOKEN"
```

### 📖 Документация

- 📘 [AI Review Quick Start](docs/AI_REVIEW_QUICKSTART.md) - 3 шага до первого review
- 📖 [AI Review Setup](docs/AI_REVIEW_SETUP.md) - Полная инструкция
- 🎨 [AI Review Go Prompts](docs/AI_REVIEW_GO_PROMPTS.md) - Кастомные промпты

### 🎯 Рекомендуемые модели

| Модель | JSON Support | Context | AI Review |
|--------|--------------|---------|-----------|
| **deepseek-coder:6.7b** | ✅ Отличный | 16K | ⭐⭐⭐⭐⭐ |
| **qwen2.5-coder:7b** | ✅ Хороший | 32K | ⭐⭐⭐⭐ |
| **qwen2.5-coder:14b** | ✅ Отличный | 32K | ⭐⭐⭐⭐⭐ |
| llama3.2:70b | ⚠️ Средний | 128K | ⭐⭐⭐ |

**⚠️ Важно:** Используйте модели с хорошей JSON поддержкой. AI-review требует строгий JSON формат без markdown wrapper.

**Инструмент:** [github.com/Nikita-Filonov/ai-review](https://github.com/Nikita-Filonov/ai-review)

---

## 📦 Структура проекта

```
ollama-openai-proxy/
├── cmd/
│   ├── server/main.go          # HTTP server + WebUI
│   └── tui/                    # Terminal UI (legacy)
├── internal/
│   ├── api/
│   │   ├── handlers/           # REST API handlers
│   │   ├── middleware/         # JWT, RBAC, Rate Limit
│   │   └── router/             # Route configuration
│   ├── auth/                   # Authentication & Authorization
│   ├── storage/                # Database (SQLite/PostgreSQL)
│   ├── metrics/                # MoniGo, GPU, Prometheus
│   ├── converter/              # OpenAI ↔ Ollama
│   └── models/                 # Data models
├── web/                        # WebUI (HTML/CSS/JS)
│   ├── *.html                  # Pages
│   ├── js/                     # JavaScript
│   └── css/                    # Styles
├── configs/
│   ├── dev.yaml                # Development config
│   └── production.yaml.example # Production template
├── data/                       # Runtime data (SQLite DB)
├── logs/                       # Log files
├── docs/                       # Documentation
├── BACKLOG/                    # Task specifications (85 files)
├── go.mod
├── Roadmap.MD                  # Development roadmap
└── README.md
```

---

## 🗺️ Roadmap

### ✅ Завершено (v1.9.3)

- ✅ **Full OpenAI API** compatibility
- ✅ **Multi-Tenancy** с RBAC
- ✅ **JWT Authentication** + Bootstrap system
- ✅ **ChatGPT-like WebUI** с conversations
- ✅ **Dynamic Model Parameters** в чате
- ✅ **Context Tracking** + Auto-Summarization
- ✅ **MoniGo Performance Monitoring**
- ✅ **NVIDIA GPU Monitoring** (multi-GPU)
- ✅ **OpenTelemetry** distributed tracing
- ✅ **Backup & Restore** system

### 📋 Запланировано

**v1.10.0 - Smart Chat & Content**

- Vision OCR (Multimodal Chat)
- Web Content Fetcher & Summarization
- File Upload (PDF, DOCX, TXT)
- Conversation Export/Import
- WebSocket Real-time Updates

**v1.11.0 - Enterprise Auth**

- Keycloak SSO Integration
- LDAP/Active Directory
- Enhanced Audit Logging
- Custom Roles & Permissions

**v1.12.0 - Model Management Pro II**

- Usage Quotas System
- Model Preloading & Warming
- Prometheus Metrics Export
- Advanced Rate Limiting

Полный roadmap: [Roadmap.MD](Roadmap.MD)

---

## 🤝 Contributing

Приветствуются contributions!

1. Fork репозитория
2. Создайте feature branch (`git checkout -b feature/amazing`)
3. Напишите тесты
4. Commit (`git commit -m 'Add amazing feature'`)
5. Push (`git push origin feature/amazing`)
6. Откройте Pull Request

**Code Standards:**

- Go 1.25+ idioms
- 100% coverage критических компонентов
- Structured logging (logrus)
- Explicit error handling

---

## 📄 License

MIT License - см. [LICENSE](LICENSE)

---

## 🙏 Acknowledgments

- **Ollama** - Замечательная платформа для локальных LLM
- **OpenAI** - API стандарт
- **Go Libraries:**
  - [Gin](https://github.com/gin-gonic/gin) - HTTP framework
  - [GORM](https://gorm.io/) - ORM для SQLite/PostgreSQL
  - [JWT-Go](https://github.com/golang-jwt/jwt) - JWT tokens
  - [Viper](https://github.com/spf13/viper) - Configuration
  - [Logrus](https://github.com/sirupsen/logrus) - Logging
  - [MoniGo](https://github.com/iyashjayesh/monigo) - Performance monitoring
  - [OpenTelemetry](https://opentelemetry.io/) - Tracing

---

## 🐛 Критические исправления

### SSE Content-Type для Spring AI (v1.10.1)

- **[BUGFIX: SSE Content-Type Header](docs/BUGFIX_SSE_CONTENT_TYPE.md)** - Исправлен `Content-Type` для Server-Sent Events
  - **Проблема:** Spring AI плагины (JetBrains IDE) падали с `JsonParseException`
  - **Решение:** Изменен заголовок с `text/plain` на `text/event-stream`
  - **Impact:** ✅ Spring AI работает, ✅ обратная совместимость сохранена

### AI Review совместимость (v1.6.1)

- **[BUGFIX: AI Review Compatibility](docs/BUGFIX_AI_REVIEW_COMPATIBILITY.md)** - OpenAI API совместимость
  - Исправлен `completion_tokens` в `usage` объекте
  - Исправлен streaming bug с `omitempty` на `Stream` field

### Streaming omitempty (v1.6.0)

- **[BUGFIX: Streaming omitempty](docs/BUGFIX_STREAMING_OMITEMPTY.md)** - Критический bug в streaming
  - Ollama возвращал только первое слово вместо полного ответа
  - Убран `omitempty` с `Stream` field в `ChatRequest`

---

## 📚 Документация

- **[API Documentation](docs/API_DOCUMENTATION.md)** - Полный API reference
- **[Configuration](docs/CONFIGURATION.md)** - Все параметры конфигурации
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Решение проблем
- **[Architecture](docs/ARCHITECTURE.md)** - Архитектура системы
- **[Performance](docs/PERFORMANCE.md)** - Performance tuning
- **[TUI Guide](docs/TUI_GUIDE.md)** - Terminal UI документация
- **[WebUI Guide](docs/WEBUI_GUIDE.md)** - Web UI документация
- **[Roadmap](Roadmap.MD)** - План развития
- **[Build Guide](BUILD.md)** - Инструкции по сборке

---

## 📞 Support

- **Issues:** [GitHub Issues](https://github.com/yourusername/ollama-openai-proxy/issues)
- **Discussions:** [GitHub Discussions](https://github.com/yourusername/ollama-openai-proxy/discussions)

---

**Made with ❤️ for the Open Source Community**

⭐ Star this repo if you find it useful!

**Current Version:** 1.9.3 | **Status:** Active Development 🚀
