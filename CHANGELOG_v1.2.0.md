# 📋 CHANGELOG - Version 1.2.0

**Release Date:** 5 октября 2025  
**Тип релиза:** Feature Release  
**Фокус:** Enhanced Monitoring & Management

---

## 🎯 Обзор

Version 1.2.0 значительно улучшает возможности мониторинга и управления Ollama-OpenAI Proxy:

- 🔍 **Request Monitor в TUI** - полный мониторинг всех API запросов
- 📊 **Metrics Visualization в WebUI** - интерактивные графики с Chart.js
- 📋 **Advanced Logs** - расширенные возможности работы с логами
- 🔑 **Enhanced API Key Management** - улучшенное управление ключами

---

## ✨ Новые функции

### 🔍 TUI-04: Request Monitor

Добавлен полноценный Request Monitor в Terminal UI для отслеживания всех запросов к API.

**Функциональность:**

- ✅ Live таблица активных запросов с адаптивной шириной колонок
- ✅ Сортировка по полям: time, duration, status (ascending/descending)
- ✅ Фильтрация по статусу: all, success, error, pending
- ✅ Pagination (клавиши 'n' и 'p' для навигации)
- ✅ Детальный просмотр запроса (Enter)
- ✅ Export в CSV и JSON (клавиша 'e')
- ✅ Keyboard navigation (↑↓, j/k)
- ✅ Real-time обновление (клавиша 'r')

**Новые файлы:**

- `internal/request/types.go` - типы данных для запросов
- `internal/request/storage.go` - ring buffer хранилище (1000 запросов)
- `internal/api/middleware/request_tracker.go` - middleware для отслеживания
- `internal/api/handlers/requests.go` - API endpoints для доступа к данным
- Обновлен `cmd/tui/main.go` - добавлен экран Request Monitor

**API Endpoints:**

```
GET /api/requests?page=1&per_page=50&sort=time&order=desc&status=success
GET /api/requests/:id
GET /api/requests/stats
DELETE /api/requests (admin only)
```

---

### 📊 WEBUI-01: Metrics Visualization

Интегрированы интерактивные графики для визуализации метрик в WebUI.

**Функциональность:**

- ✅ **Latency Chart** (Line) - P50, P95, P99 перцентили
- ✅ **Requests Chart** (Bar) - количество запросов за период
- ✅ **Status Distribution** (Pie) - распределение по HTTP статусам
- ✅ **Active Requests** (Area) - количество одновременных запросов
- ✅ Time range selector: 1h, 6h, 24h, 7d
- ✅ Auto-refresh toggle (вкл/выкл автообновления)
- ✅ Manual refresh button
- ✅ Real-time updates через WebSocket

**Технологии:**

- Chart.js 4.4.0 (загружается с CDN)
- WebSocket для live updates
- Responsive design (адаптивные графики)

**Новые файлы:**

- Обновлен `internal/web/static/index.html` - добавлены canvas элементы
- Обновлен `internal/web/static/js/app.js` - логика графиков
- Обновлен `internal/web/static/css/style.css` - стили для графиков

---

### 📋 WEBUI-02: Advanced Logs Features

Расширенные возможности работы с логами в WebUI.

**Функциональность:**

- ✅ Client-side фильтрация по уровню (ERROR, WARN, INFO, DEBUG, TRACE)
- ✅ Real-time поиск по тексту
- ✅ Export логов в TXT и JSON
- ✅ Auto-refresh toggle
- ✅ Color coding:
  - 🔴 ERROR - красный
  - 🟡 WARN - желтый
  - 🔵 INFO - синий
  - ⚪ DEBUG - серый
  - ⚪ TRACE - светло-серый
- ✅ Statistics: показ количества логов по каждому уровню
- ✅ Pause/Resume для изучения логов без автоскролла

**Новые файлы:**

- Обновлен `internal/web/static/js/app.js` - добавлена логика фильтрации и экспорта
- Обновлен `internal/web/static/css/style.css` - цветовая схема логов

---

### 🔑 AUTH-04: Enhanced API Key Management

Расширенные возможности управления API ключами.

**Функциональность:**

- ✅ **Edit API Keys:**
  - Name, Description
  - Models (allowed models list)
  - Permissions (chat, models, embeddings, admin)
  - Rate Limits (requests per minute/hour)
- ✅ **Revoke API Key** - деактивация с причиной
- ✅ **Enable API Key** - повторная активация отозванного ключа
- ✅ **Extend Expiration** - продление срока действия
- ✅ **Update Permissions** - изменение прав доступа
- ✅ Реализовано в WebUI и доступно через API

**Новые API Endpoints:**

```
PUT /api/admin/keys/:id - Update key
PATCH /api/admin/keys/:id/revoke - Revoke key
PATCH /api/admin/keys/:id/enable - Enable key
POST /api/admin/keys/:id/extend - Extend expiration
PATCH /api/admin/keys/:id/permissions - Update permissions
```

**Обновленные файлы:**

- `internal/api/handlers/admin.go` - новые handler методы
- `internal/auth/apikey/manager.go` - расширенные методы управления
- `internal/storage/interfaces.go` - обновлен интерфейс
- `internal/storage/json_storage.go` - реализация для JSON storage
- `internal/web/static/index.html` - модальные окна для редактирования
- `internal/web/static/js/app.js` - логика редактирования

---

## 🔧 Улучшения

### Backend

- **Request Tracking Middleware:** Автоматическое отслеживание всех входящих запросов
- **Metrics Storage:** Ring buffer для хранения последних 10,000 метрик
- **WebSocket Events:** Новые типы событий для Request Monitor
- **API Router:** Добавлены routes для request monitoring

### TUI (Terminal UI)

- **Новая вкладка "Requests"** - доступна по клавише '2'
- **Mouse support** - клик по табам для переключения view
- **Адаптивные колонки** - автоматическая подстройка под ширину терминала
- **Color coding** - визуальное различие статусов (success=green, error=red)

### WebUI (Web Interface)

- **Dashboard:** Добавлена кнопка Analytics (переход к графикам)
- **Navigation:** Улучшенная навигация между секциями
- **Session Management:** Сохранение admin token в sessionStorage
- **Toast Notifications:** Уведомления об успешных операциях

---

## 🐛 Исправления

- **WebUI 404 ошибки:** Исправлены отсутствующие proxy endpoints для `/api/admin/keys/:id`
- **Permission Middleware:** Восстановлена проверка прав для admin routes
- **Rate Limiting:** Исправлена логика bypass для admin ключей
- **TUI Navigation:** Исправлена нумерация табов после добавления Requests

---

## 📚 Документация

- **Roadmap.MD:** Обновлен статус Version 1.2.0 как завершенной
- **Coverage.MD:** Добавлены тесты для новых handlers
- **CHANGELOG_v1.2.0.md:** Создан данный файл

---

## 🧪 Тестирование

### Покрытие тестами

**Handlers:**

- ✅ Health Handler: 6/6 тестов (90%+ coverage)
- ✅ Chat Handler: 5/5 тестов (60%+ coverage)
- ✅ Models Handler: 3/3 тестов (70%+ coverage)
- ✅ Embeddings Handler: 4/4 тестов (70%+ coverage)
- ✅ Completions Handler: 4/4 тестов (70%+ coverage)
- ✅ Admin Handler: 12/12 тестов (80%+ coverage)

**Текущее покрытие:** 34.9% (handlers), стремимся к 95%+

---

## 🚀 Обновление

### Для обновления с Version 1.1.0

1. **Обновите бинарные файлы:**

   ```bash
   go build -o bin/server.exe ./cmd/server
   go build -o bin/tui.exe ./cmd/tui
   go build -o bin/webui.exe ./cmd/webui
   ```

2. **Обновите конфигурацию** (опционально):
   - Новых конфигурационных параметров не добавлено
   - Все изменения обратно совместимы

3. **Перезапустите сервисы:**

   ```bash
   # Остановите старые процессы
   # Запустите новые бинарники
   .\bin\server.exe
   .\bin\tui.exe
   .\bin\webui.exe
   ```

### Breaking Changes

**НЕТ** breaking changes в данном релизе. Полная обратная совместимость с Version 1.1.0.

---

## 📦 Зависимости

### Новые зависимости

Нет новых Go зависимостей.

### Frontend (CDN)

- **Chart.js 4.4.0** - для визуализации метрик (загружается с CDN)

---

## 🎯 Следующие шаги

### Version 1.6.0 - User Experience & Multi-Tenancy (HIGH Priority)

**Планируемые функции:**

- 💾 **DB-01:** Database Abstraction Layer (SQLite + PostgreSQL)
- 🔐 **AUTH-05:** User Authentication & Multi-Tenancy
- 💬 **WEBUI-03:** Interactive Chat Interface (ChatGPT-like UI)
- 📊 **WEBUI-04:** User Dashboard

**Оценка:** 45-57 часов работы

---

## 🙏 Благодарности

Спасибо за использование Ollama-OpenAI Proxy!

Если у вас есть вопросы или предложения, создайте issue в репозитории.

---

**Команда Ollama-OpenAI Proxy**  
*Turning local LLMs into production-ready APIs* 🦙✨
