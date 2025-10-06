# 🌐 WebUI Guide - Ollama-OpenAI Proxy

## Версия: 1.0.0

---

## 📖 Содержание

1. [Введение](#введение)
2. [Запуск WebUI](#запуск-webui)
3. [Аутентификация](#аутентификация)
4. [Dashboard](#dashboard)
5. [API Keys Management](#api-keys-management)
6. [Models](#models)
7. [Configuration](#configuration)
8. [Logs Viewer](#logs-viewer)
9. [Export Functions](#export-functions)
10. [Темы и настройки](#темы-и-настройки)
11. [Troubleshooting](#troubleshooting)

---

## 🎯 Введение

**WebUI** — это веб-интерфейс для управления Ollama-OpenAI Proxy. Он предоставляет удобный графический интерфейс для:

- 📊 Мониторинга метрик сервера в real-time
- 🔑 Управления API ключами (создание, просмотр, удаление)
- 🤖 Просмотра информации о моделях
- ⚙️ Просмотра конфигурации
- 📝 Real-time просмотра логов
- 📤 Экспорта данных (CSV, JSON)

### Архитектура

```
┌─────────────────┐
│   Browser       │
│  (localhost:    │
│     8081)       │
└────────┬────────┘
         │ HTTP
         ↓
┌─────────────────┐
│  WebUI Server   │
│  (cmd/webui)    │
└────────┬────────┘
         │ Proxy API
         ↓
┌─────────────────┐
│  Main Server    │
│  (localhost:    │
│     8080)       │
└─────────────────┘
```

WebUI работает как **отдельный сервис**, проксирующий запросы к основному API серверу.

---

## 🚀 Запуск WebUI

### Предварительные требования

1. **Main Server** должен быть запущен:

   ```bash
   ./bin/server.exe
   # или через air
   air
   ```

2. **Ollama** должен быть доступен:

   ```bash
   ollama serve
   ```

### Запуск

#### Windows

```bash
.\bin\webui.exe
```

#### Linux/MacOS

```bash
./bin/webui
```

### Параметры запуска

```bash
.\bin\webui.exe --help

Flags:
  --port int           Port to run WebUI on (default: 8081)
  --host string        Host to bind to (default: "0.0.0.0")
  --server-url string  Main server URL (default: "http://localhost:8080")
```

### Примеры

**Запуск на другом порту:**

```bash
.\bin\webui.exe --port 9090
```

**Подключение к удаленному серверу:**

```bash
.\bin\webui.exe --server-url http://192.168.1.100:8080
```

### Доступ

После запуска откройте браузер:

```
http://localhost:8081
```

---

## 🔐 Аутентификация

### Login Modal

При первом входе появится **Login Modal**:

```
┌──────────────────────────────────┐
│ 🔐 Admin Authentication         │
├──────────────────────────────────┤
│ Please enter your admin API key │
│ to access management features.   │
│                                  │
│ Admin API Key:                   │
│ [sk-admin-........................] │
│                                  │
│      [🔓 Login]                  │
└──────────────────────────────────┘
```

### Получение Admin Key

Admin key берется из конфигурации сервера:

**configs/dev.yaml:**

```yaml
auth:
  enabled: true
  admin_key: "sk-admin-dev-key-12345"  # ← Этот ключ
```

### Session Management

- ✅ После успешного логина ключ сохраняется в **SessionStorage**
- ✅ Ключ действует до закрытия вкладки/браузера
- ✅ При новой сессии нужно логиниться заново
- ✅ **Logout** кнопка в правом верхнем углу

### Security

- 🔒 Admin key передается через `Authorization: Bearer` header
- 🔒 Key не логируется в console
- 🔒 Key хранится только в SessionStorage (не LocalStorage)
- 🔒 HTTPS рекомендуется для production

---

## 📊 Dashboard

### Обзор

Dashboard — главный экран WebUI с real-time метриками:

```
┌─────────────────────────────────────────────────────┐
│ Dashboard                    [📊 CSV] [📄 JSON]     │
├─────────────────────────────────────────────────────┤
│ ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐            │
│ │ 📈   │  │ ✅   │  │ ❌   │  │ 🔑   │            │
│ │ 1,234│  │  950 │  │   12 │  │   5  │            │
│ │Total │  │Success│  │Errors│  │Keys │            │
│ └──────┘  └──────┘  └──────┘  └──────┘            │
│                                                     │
│ ┌─────────────────────┐  ┌────────────────────┐   │
│ │ 🖥️ Server Info     │  │ 🤖 Ollama Status   │   │
│ │ Status: Running    │  │ Connected: Yes     │   │
│ │ Uptime: 2h 15m     │  │ Models: 12         │   │
│ │ Address: :8080     │  │ URL: localhost:... │   │
│ └─────────────────────┘  └────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

### Метрики

**Stat Cards:**

- 📈 **Total Requests** - всего запросов с момента запуска
- ✅ **Success** - успешные запросы
- ❌ **Errors** - запросы с ошибками
- 🔑 **API Keys** - количество активных ключей

**Server Info:**

- **Status**: Running / Stopped
- **Uptime**: время работы сервера
- **Address**: адрес HTTP сервера

**Ollama Status:**

- **Connected**: статус подключения
- **Models**: количество доступных моделей
- **URL**: адрес Ollama сервера

### Auto-Refresh

Dashboard обновляется **каждые 5 секунд** автоматически.

**Ручное обновление:**

```
[🔄 Refresh] кнопка в header
```

### Export Data

**📊 Export CSV** - экспорт метрик в CSV:

```csv
Metric,Value
Server Uptime,2h 15m 30s
Total Requests,1234
Success Requests,950
Error Requests,12
Ollama Connected,Yes
Ollama Models Count,12
Total API Keys,5
Active API Keys,4
Disabled API Keys,1
```

**📄 Export JSON** - экспорт всех данных в JSON:

```json
{
  "stats": {
    "uptime": 8130,
    "total_requests": 1234,
    "success_requests": 950,
    "error_requests": 12
  },
  "ollama": {
    "connected": true,
    "models_count": 12
  },
  "api_keys": {
    "total": 5,
    "active": 4,
    "disabled": 1
  }
}
```

---

## 🔑 API Keys Management

### Просмотр ключей

```
┌──────────────────────────────────────────────────────┐
│ API Keys Management      [📊 CSV] [➕ Create New]   │
├──────────────────────────────────────────────────────┤
│ Name     │ Status │ Rate Limit │ Models │ Last Used │
├──────────┼────────┼────────────┼────────┼───────────┤
│ DevKey1  │ 🟢     │ 60/min    │ *      │ 2m ago    │
│ ProdKey  │ 🟢     │ 1000/min  │ gpt-4  │ 5s ago    │
│ TestKey  │ 🔴     │ 10/min    │ llama3 │ Never     │
└──────────┴────────┴────────────┴────────┴───────────┘
```

**Столбцы:**

- **Name**: имя ключа
- **Status**: 🟢 Active / 🔴 Disabled
- **Rate Limit**: лимит запросов в минуту
- **Models**: разрешенные модели (* = все)
- **Last Used**: последнее использование

### Создание ключа

**1. Нажать [➕ Create New Key]**

**2. Заполнить форму:**

```
┌────────────────────────────────┐
│ ➕ Create New API Key         │
├────────────────────────────────┤
│ Key Name:                      │
│ [My Application Key...........]│
│                                │
│ Rate Limit (req/min):          │
│ [60...........................]│
│                                │
│ Allowed Models:                │
│ [*............................]│
│ (* for all, comma-separated)   │
│                                │
│  [Cancel]   [Create Key]       │
└────────────────────────────────┘
```

**3. Получить plaintext key:**

```
┌────────────────────────────────┐
│ ✅ API Key Created!           │
├────────────────────────────────┤
│ ⚠️ Save this key now - you    │
│ won't be able to see it again! │
│                                │
│ ┌────────────────────────────┐ │
│ │ sk-1234567890abcdef...     │ │
│ │ [📋 Copy]                  │ │
│ └────────────────────────────┘ │
│                                │
│           [Done]               │
└────────────────────────────────┘
```

**4. Копировать ключ:**

- Нажать **[📋 Copy]**
- Ключ скопируется в буфер обмена
- Toast уведомление: ✅ "API key copied to clipboard!"

### Удаление ключа

**Кнопка [🗑️ Delete]** в строке ключа:

```javascript
Confirm: Are you sure you want to delete "DevKey1"?
[Cancel] [Delete]
```

После удаления:

- ✅ Toast: "API key deleted successfully"
- Таблица обновляется автоматически

### Export Keys

**📊 Export CSV** - экспорт всех ключей:

```csv
Key ID,Name,Status,Rate Limit,Allowed Models,Created,Last Used
ak_123,DevKey1,active,60,*,2025-10-01 10:00,2m ago
ak_456,ProdKey,active,1000,gpt-4,2025-10-02 15:30,5s ago
```

---

## 🤖 Models

### Extended Model Cards

```
┌──────────────────────────────────────┐
│ qwen2.5-coder:7b           4.7 GB   │
├──────────────────────────────────────┤
│ Parameters:     7B                   │
│ Family:         qwen2                │
│ Quantization:   Q4_K_M               │
│ Format:         GGUF                 │
│ Modified:       04.10.2025           │
├──────────────────────────────────────┤
│ Digest: sha256:3c7c4d6a7b8e...      │
└──────────────────────────────────────┘
```

### Информация о моделях

**Каждая карточка содержит:**

**Header:**

- **Model Name**: имя модели в Ollama
- **Size (GB)**: размер модели на диске

**Details:**

- **Parameters**: количество параметров (7B, 30B, etc.)
- **Family**: семейство модели (llama, qwen, mistral, etc.)
- **Quantization**: уровень квантизации (Q4_K_M, Q8_0, etc.)
- **Format**: формат файла (GGUF, GGML)
- **Modified**: дата последнего изменения

**Footer:**

- **Digest**: SHA256 hash модели (сокращенный)

### Hover эффекты

При наведении на карточку:

- ⬆️ Карточка поднимается (translateY)
- 🔵 Border становится синим
- ✨ Появляется box-shadow

### Responsive Design

Карточки автоматически адаптируются к ширине окна:

- **Wide screen**: 3-4 карточки в ряд
- **Medium**: 2 карточки в ряд
- **Mobile**: 1 карточка в ряд

---

## ⚙️ Configuration

### Просмотр конфигурации

```
┌────────────────────────────────────────┐
│ Configuration                          │
├────────────────────────────────────────┤
│ {                                      │
│   "server": {                          │
│     "host": "0.0.0.0",                 │
│     "port": 8080,                      │
│     "timeout": {                       │
│       "read": 30000000000,             │
│       "write": 30000000000             │
│     }                                  │
│   },                                   │
│   "ollama": {                          │
│     "url": "http://localhost:11434",   │
│     "timeout": 300000000000            │
│   },                                   │
│   ...                                  │
│ }                                      │
└────────────────────────────────────────┘
```

### Секции конфигурации

**Отображаются:**

- ✅ Server settings (host, port, timeouts)
- ✅ Ollama settings (URL, timeout, retries)
- ✅ Auth settings (enabled, storage type)
- ✅ Logging settings (level, format, file)
- ✅ Models (mappings, cache)
- ✅ Tools (force usage, fallback model)
- ✅ Metrics (enabled, prometheus)
- ✅ TUI (enabled, refresh rate)

**Скрыты (для безопасности):**

- 🔒 admin_key
- 🔒 jwt_secret
- 🔒 API keys

### JSON Formatting

Конфигурация отображается в **pretty-printed JSON** с:

- 🎨 Syntax highlighting
- 🔢 Line numbers
- 📏 2-space indentation
- 📜 Scrollable view

---

## 📝 Logs Viewer

### Real-time Logs

```
┌──────────────────────────────────────────────┐
│ Recent Logs  [All][Errors][Warnings][Info]  │
│              [Debug]              [🔄]       │
├──────────────────────────────────────────────┤
│ 🔵 time="..." level=info msg="Server..."    │
│ 🔵 time="..." level=info msg="Request..."   │
│ 🟠 time="..." level=warning msg="Slow..."   │
│ 🔴 time="..." level=error msg="Failed..."   │
│ ⬛ time="..." level=debug msg="Details..."  │
└──────────────────────────────────────────────┘
```

### Фильтры

**5 кнопок-фильтров:**

1. **All** - все логи (по умолчанию)
2. **Errors** - только ошибки 🔴
3. **Warnings** - только предупреждения 🟠
4. **Info** - только информационные 🔵
5. **Debug** - только отладочные ⬛

**Active state:**

- Активная кнопка подсвечивается цветом уровня
- Остальные кнопки серые

### Color Coding

**По уровням:**

- 🔴 **ERROR** - красный фон, красная граница
- 🟠 **WARNING** - оранжевый фон, оранжевая граница
- 🔵 **INFO** - синий фон, обычный текст
- ⬛ **DEBUG** - серый фон, серый текст

### Функции

**Auto-scroll:**

- При загрузке логов автоматически прокручивается к последним
- `container.scrollTop = container.scrollHeight`

**Refresh:**

- Кнопка **[🔄 Refresh]** обновляет логи
- Limit: 200 последних строк

**API:**

```
GET /api/logs?limit=200&level=error
```

---

## 📤 Export Functions

### Dashboard Export

**Export CSV:**

```csv
Metric,Value
Server Uptime,2h 15m 30s
Total Requests,1234
Success Requests,950
...
```

**Export JSON:**

```json
{
  "stats": {...},
  "ollama": {...},
  "api_keys": {...}
}
```

### API Keys Export

**Export CSV:**

```csv
Key ID,Name,Status,Rate Limit,Allowed Models,Created,Last Used
ak_123,DevKey1,active,60,*,2025-10-01 10:00:00,2m ago
...
```

### Именование файлов

Файлы автоматически именуются с датой:

```
ollama-proxy-stats-2025-10-05.csv
ollama-proxy-stats-2025-10-05.json
ollama-proxy-api-keys-2025-10-05.csv
```

### Toast Notifications

После экспорта появляется уведомление:

```
✅ Stats exported to CSV
```

---

## 🎨 Темы и настройки

### Dark/Light Theme Toggle

**Кнопка переключения тем:**

```
[🌙] - Dark mode (по умолчанию)
[☀️] - Light mode
```

**Расположение:** правый нижний угол

### Сохранение темы

Тема сохраняется в **localStorage**:

```javascript
localStorage.setItem('theme', 'dark');
localStorage.setItem('theme', 'light');
```

### Цветовые схемы

**Dark Theme:**

```css
--bg-main: #1a1b26
--bg-secondary: #24283b
--bg-tertiary: #2f3549
--text-primary: #c0caf5
--primary: #7aa2f7
```

**Light Theme:**

```css
--bg-main: #ffffff
--bg-secondary: #f5f5f5
--bg-tertiary: #e0e0e0
--text-primary: #333333
--primary: #2196f3
```

### Toast Notifications

**4 типа уведомлений:**

1. **Success** ✅
   - Зеленый фон
   - Галочка иконка
   - Auto-dismiss через 3 секунды

2. **Error** ❌
   - Красный фон
   - Крестик иконка
   - Auto-dismiss через 5 секунд

3. **Warning** ⚠️
   - Оранжевый фон
   - Предупреждение иконка
   - Auto-dismiss через 4 секунды

4. **Info** ℹ️
   - Синий фон
   - Info иконка
   - Auto-dismiss через 3 секунды

**Manual Close:**

- Кнопка **[×]** для ручного закрытия

---

## 🔧 Troubleshooting

### Проблема: Login Modal не появляется

**Причина:** SessionStorage уже содержит ключ

**Решение:**

1. Открыть DevTools (F12)
2. Console → `sessionStorage.clear()`
3. Перезагрузить страницу (F5)

---

### Проблема: "Failed to load stats"

**Причина:** Main Server недоступен

**Решение:**

1. Проверить что server запущен:

   ```bash
   curl http://localhost:8080/health
   ```

2. Проверить `--server-url` в WebUI:

   ```bash
   .\bin\webui.exe --server-url http://localhost:8080
   ```

---

### Проблема: API Keys не загружаются

**Причина:** Неверный admin key

**Решение:**

1. Logout из WebUI
2. Проверить admin_key в `configs/dev.yaml`
3. Login с правильным ключом

---

### Проблема: Models не отображаются

**Причина:** Ollama недоступен

**Решение:**

1. Запустить Ollama:

   ```bash
   ollama serve
   ```

2. Проверить подключение:

   ```bash
   curl http://localhost:11434/api/tags
   ```

---

### Проблема: Logs не загружаются

**Причина:** Файл логов не существует

**Решение:**

1. Проверить `configs/dev.yaml`:

   ```yaml
   logging:
     file_path: "logs/proxy-dev.log"
   ```

2. Убедиться что директория `logs/` существует
3. Перезапустить server для создания файла

---

### Проблема: Export не работает

**Причина:** Popup blocker в браузере

**Решение:**

1. Разрешить popups для `localhost:8081`
2. Попробовать снова

---

### Проблема: Theme не сохраняется

**Причина:** LocalStorage заблокирован

**Решение:**

1. Проверить настройки браузера
2. Разрешить localStorage для сайта
3. Проверить в DevTools:

   ```javascript
   localStorage.getItem('theme')
   ```

---

## 📚 Дополнительные ресурсы

- [README.md](../README.md) - Общая информация о проекте
- [TUI_GUIDE.md](TUI_GUIDE.md) - Руководство по Terminal UI
- [API_DOCUMENTATION.md](API_DOCUMENTATION.md) - API документация
- [CONFIGURATION.md](CONFIGURATION.md) - Конфигурация
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Решение проблем
- [ARCHITECTURE.md](ARCHITECTURE.md) - Архитектура проекта

---

## 🎯 Best Practices

### Security

1. **Never share your admin key publicly**
2. **Use HTTPS in production**
3. **Enable CORS only for trusted origins**
4. **Regularly rotate admin keys**
5. **Monitor API key usage**

### Performance

1. **Use reasonable refresh intervals** (default 5s)
2. **Limit log viewer to 200 lines**
3. **Export large datasets incrementally**
4. **Close unused browser tabs**

### UX

1. **Enable Dark mode for night work** 🌙
2. **Use keyboard shortcuts** (Tab, Esc, Enter)
3. **Check Toast notifications** for important messages
4. **Export data regularly** for backup

---

## 🚀 Roadmap

### Planned Features

- [ ] WebSocket для real-time updates (Фаза 12.2)
- [ ] Live request monitor с таблицей
- [ ] Usage statistics графики
- [ ] PDF export для отчетов
- [ ] Scheduled exports
- [ ] Multi-language support
- [ ] Mobile app (React Native)

---

**Version**: 1.0.0  
**Last Updated**: 2025-10-05  
**Author**: Ollama-OpenAI Proxy Team
