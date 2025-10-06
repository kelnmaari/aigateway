# 🧪 Final Testing Checklist

## Дата: 2025-10-05

## Версия: 1.0.0-rc1 (Release Candidate)

---

## ✅ 1. SERVER CORE

### 1.1 Запуск и Health Check

- [ ] Server запускается без ошибок
- [ ] Graceful shutdown работает (Ctrl+C)
- [ ] `/health` endpoint отвечает 200 OK
- [ ] Логи пишутся в файл с rotation
- [ ] Prometheus metrics доступны на `/metrics`

### 1.2 OpenAI API Endpoints

- [ ] `GET /v1/models` - возвращает список моделей
- [ ] `POST /v1/chat/completions` - обычный chat
- [ ] `POST /v1/chat/completions` - streaming (SSE)
- [ ] `POST /v1/chat/completions` - с tools (function calling)
- [ ] `POST /v1/embeddings` - embeddings API
- [ ] `POST /v1/completions` - legacy completions

### 1.3 Authentication & Authorization

- [ ] Запрос без API key → 401 Unauthorized
- [ ] Запрос с невалидным ключом → 401 Unauthorized
- [ ] Запрос с валидным ключом → 200 OK
- [ ] Model-based authorization работает
- [ ] Admin key имеет полные права

### 1.4 Rate Limiting

- [ ] Rate limit per key работает
- [ ] Превышение лимита → 429 Too Many Requests
- [ ] Admin key не лимитируется
- [ ] Лимиты восстанавливаются через время

### 1.5 Tools (Function Calling)

- [ ] Запрос с tools возвращает tool_calls
- [ ] `tool_choice: "auto"` работает
- [ ] `tool_choice: "required"` форсирует вызов
- [ ] `force_usage: true` в конфиге работает
- [ ] Tool-aware routing на правильную модель
- [ ] Streaming с tools работает (index поле)

---

## ✅ 2. TUI (Terminal User Interface)

### 2.1 Запуск и Навигация

- [ ] TUI запускается без ошибок (`./bin/tui.exe`)
- [ ] Навигация между экранами работает (1-7, Tab)
- [ ] Выход работает (q, Ctrl+C)
- [ ] Auto-refresh каждую секунду

### 2.2 Dashboard Screen (1)

- [ ] Отображает server stats (uptime, requests)
- [ ] Показывает Ollama status
- [ ] Prometheus metrics обновляются
- [ ] Нет mock данных

### 2.3 Models Screen (2)

- [ ] Список моделей из Ollama
- [ ] Количество моделей корректное

### 2.4 API Keys Screen (3)

- [ ] Список всех API ключей
- [ ] Создание нового ключа (n)
- [ ] Plaintext key показывается один раз
- [ ] Копирование ключа (c) в clipboard
- [ ] Адаптивная таблица к ширине терминала
- [ ] Статистика использования ключей

### 2.5 Config Screen (4)

- [ ] Отображение конфигурации
- [ ] Секреты скрыты
- [ ] Прокрутка работает (↑↓, PgUp/PgDn)

### 2.6 Logs Screen (5)

- [ ] Последние 100 строк логов
- [ ] Color coding (error=red, warn=yellow, info=blue, debug=gray)
- [ ] Прокрутка работает
- [ ] Статистика по уровням

### 2.7 Control Screen (6)

- [ ] Server status (address, uptime)
- [ ] Ollama connection status
- [ ] Request statistics
- [ ] Прокрутка работает

---

## ✅ 3. WEBUI (Web User Interface)

### 3.1 Запуск и Доступ

- [ ] WebUI запускается (`./bin/webui.exe`)
- [ ] Доступен на `http://localhost:8081`
- [ ] Login modal появляется
- [ ] Login с admin key работает
- [ ] SessionStorage сохраняет ключ
- [ ] Logout работает

### 3.2 Dashboard

- [ ] Метрики обновляются каждые 5 секунд
- [ ] Stat cards показывают данные
- [ ] Export CSV работает
- [ ] Export JSON работает
- [ ] Скачивание файлов с датой

### 3.3 API Keys Management

- [ ] Таблица ключей загружается
- [ ] Create New Key modal работает
- [ ] Plaintext key показывается после создания
- [ ] Copy to clipboard работает
- [ ] Delete key работает
- [ ] Export Keys to CSV работает

### 3.4 Models Screen

- [ ] Extended model cards отображаются
- [ ] Размер модели (GB) показывается
- [ ] Parameters, Family, Quantization
- [ ] Format, Modified date, Digest
- [ ] Hover эффекты работают

### 3.5 Configuration Screen

- [ ] Конфигурация загружается
- [ ] JSON форматирование корректное
- [ ] Секреты скрыты

### 3.6 Logs Screen

- [ ] Логи загружаются
- [ ] Фильтры работают (All, Error, Warning, Info, Debug)
- [ ] Color coding по уровням
- [ ] Refresh работает
- [ ] Auto-scroll к последним логам

### 3.7 UI/UX Features

- [ ] Toast notifications (success, error, warning, info)
- [ ] Dark/Light theme toggle работает
- [ ] Theme сохраняется в localStorage
- [ ] Responsive design (разные размеры окна)
- [ ] Все кнопки работают
- [ ] Hover эффекты везде

---

## ✅ 4. INTEGRATION TESTS

### 4.1 Server ↔ Ollama

- [ ] Подключение к Ollama успешное
- [ ] Получение моделей работает
- [ ] Chat completion работает
- [ ] Streaming работает
- [ ] Embeddings работает
- [ ] Error handling работает

### 4.2 Server ↔ TUI

- [ ] `/api/stats` endpoint работает
- [ ] `/api/config` endpoint работает
- [ ] `/api/models` endpoint работает
- [ ] `/api/admin/keys` CRUD работает
- [ ] TUI получает real-time данные

### 4.3 Server ↔ WebUI

- [ ] Все proxy endpoints работают
- [ ] CORS настроен правильно
- [ ] Authentication flow работает
- [ ] Export functions работают
- [ ] Logs API работает

### 4.4 TUI ↔ Clipboard

- [ ] Копирование API ключа работает
- [ ] Работает на Windows

---

## ✅ 5. ERROR HANDLING

### 5.1 Server Errors

- [ ] Ollama недоступен → корректная ошибка
- [ ] Invalid request → 400 Bad Request
- [ ] Auth failed → 401 Unauthorized
- [ ] Rate limit → 429 Too Many Requests
- [ ] Internal error → 500 Internal Server Error
- [ ] Все ошибки логируются

### 5.2 TUI Errors

- [ ] Server недоступен → сообщение об ошибке
- [ ] Network timeout → graceful handling
- [ ] Invalid response → error message

### 5.3 WebUI Errors

- [ ] Server недоступен → toast error
- [ ] Auth failed → redirect to login
- [ ] Network timeout → toast error
- [ ] Invalid response → toast error

---

## ✅ 6. PERFORMANCE

### 6.1 Latency

- [ ] Chat completion < 100ms (без учета Ollama)
- [ ] Models endpoint < 50ms
- [ ] Health check < 10ms
- [ ] Metrics endpoint < 50ms

### 6.2 Throughput

- [ ] Поддерживает 100+ concurrent requests
- [ ] Rate limiting работает корректно
- [ ] Memory usage стабильный

### 6.3 Resource Usage

- [ ] Server: < 50MB RAM (idle)
- [ ] TUI: < 20MB RAM
- [ ] WebUI: < 10MB RAM
- [ ] CPU usage < 5% (idle)

---

## ✅ 7. SECURITY

### 7.1 API Keys

- [ ] Ключи хешируются (bcrypt)
- [ ] Plaintext key не сохраняется
- [ ] Ключи не логируются
- [ ] Timing attack protected

### 7.2 Admin Access

- [ ] Admin endpoints требуют admin key
- [ ] Admin key не в plaintext в config
- [ ] Rate limiting bypass для admin

### 7.3 Input Validation

- [ ] SQL injection protected
- [ ] XSS protected
- [ ] Path traversal protected
- [ ] Large payload rejected

---

## ✅ 8. CONFIGURATION

### 8.1 Development Config

- [ ] `configs/dev.yaml` валидный
- [ ] Все параметры документированы
- [ ] Defaults разумные

### 8.2 Production Config

- [ ] `configs/production.yaml.example` обновлен
- [ ] Secure defaults (auth enabled, rate limits)
- [ ] Logging в файл с rotation
- [ ] Metrics enabled

---

## ✅ 9. DOCUMENTATION

### 9.1 Main Docs

- [ ] README.md актуальный
- [ ] WebUI упоминается
- [ ] Quick start работает
- [ ] Examples валидные

### 9.2 Detailed Docs

- [ ] `docs/TUI_GUIDE.md` актуальный
- [ ] `docs/API_DOCUMENTATION.md` актуальный
- [ ] `docs/CONFIGURATION.md` актуальный
- [ ] `docs/TROUBLESHOOTING.md` актуальный
- [ ] `docs/ARCHITECTURE.md` актуальный
- [ ] `docs/PERFORMANCE.md` актуальный
- [ ] **`docs/WEBUI_GUIDE.md`** ← **НУЖНО СОЗДАТЬ**

---

## ✅ 10. BUILD & DEPLOYMENT

### 10.1 Build

- [ ] `go build cmd/server/main.go` успешен
- [ ] `go build cmd/tui/main.go` успешен
- [ ] `go build cmd/webui/main.go` успешен
- [ ] Все зависимости в `go.mod`

### 10.2 Binaries

- [ ] `bin/server.exe` работает
- [ ] `bin/tui.exe` работает
- [ ] `bin/webui.exe` работает
- [ ] Версия в бинарниках корректная

### 10.3 Deployment

- [ ] systemd service файлы готовы (опционально)
- [ ] Инструкции по деплою в README
- [ ] Production checklist в docs

---

## 📊 РЕЗУЛЬТАТЫ ТЕСТИРОВАНИЯ

### Дата завершения: _____________

### Тестировщик: _____________

### Критичные проблемы

- [ ] Нет

### Некритичные проблемы

- [ ] Нет

### Статус

- [ ] ✅ Готов к релизу
- [ ] ⚠️ Требуются исправления
- [ ] ❌ Не готов

---

## 🚀 NEXT STEPS

После прохождения всех тестов:

1. ✅ Обновить MVP.MD
2. ✅ Создать Release Notes
3. ✅ Обновить документацию
4. ✅ Проверить production config
5. ✅ Tag релиза в Git
6. ✅ Deploy на production

---

**Примечание**: Этот чеклист должен быть пройден полностью перед релизом версии 1.0.0.
