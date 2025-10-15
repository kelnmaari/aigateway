# 🌐 WebUI Guide - Ollama-OpenAI Proxy v1.9.3

> **Enterprise-grade ChatGPT-like interface for local Ollama models**

---

## 📖 Содержание

1. [Введение](#введение)
2. [Первый запуск](#первый-запуск)
3. [Authentication & Multi-Tenancy](#authentication--multi-tenancy)
4. [Chat Interface](#chat-interface)
5. [Dashboard](#dashboard)
6. [Profile Management](#profile-management)
7. [API Keys](#api-keys)
8. [Tenants & Teams](#tenants--teams)
9. [Usage Statistics](#usage-statistics)
10. [Admin Panel](#admin-panel)
11. [MCP Catalog](#mcp-catalog)
12. [About & Changelog](#about--changelog)

---

## 🎯 Введение

**WebUI v1.9.3** — полноценный ChatGPT-подобный интерфейс с enterprise возможностями:

### Ключевые возможности

#### 💬 Chat Features
- **Real-time streaming** responses
- **Markdown rendering** с code highlighting
- **Conversation history** с автосохранением
- **Dynamic model parameters** в UI
- **Context tracking** с автоматической суммаризацией
- **Quick presets** для model parameters

#### 🔐 Enterprise Features
- **JWT Authentication** - полноценная система пользователей
- **Multi-Tenancy** - организации с членством и RBAC
- **API Keys Management** - Personal & Tenant keys
- **Rate Limiting** - настраиваемые лимиты per-key
- **Usage Analytics** - детальная статистика

#### 📊 Monitoring (Admin only)
- **MoniGo Dashboard** - real-time performance
- **NVIDIA GPU Metrics** - multi-GPU monitoring
- **Models Management** - список и обновление
- **System Logs** - real-time streaming через SSE

---

## 🚀 Первый запуск

### Требования

1. **Ollama server** запущен:
   ```bash
   ollama serve
   ollama pull llama3.2  # Или любая другая модель
   ```

2. **Proxy server** запущен:
   ```bash
   # Linux/macOS
   ./dist/ollama-proxy-linux-amd64 -config configs/dev.yaml
   
   # Windows
   dist\ollama-proxy-windows-amd64.exe -config configs/dev.yaml
   ```

### Bootstrap Admin User

При первом запуске система предложит создать admin пользователя:

```bash
# В логах сервера увидите:
🔐 Bootstrap Token: http://localhost:8080/bootstrap?token=abc123def456...

# Откройте эту ссылку в браузере
# Заполните форму регистрации первого admin пользователя
```

**Важно:** Bootstrap token действует **10 минут** и **одноразовый**!

### Доступ к WebUI

```
http://localhost:8080/login
```

---

## 🔐 Authentication & Multi-Tenancy

### Регистрация

1. Откройте `/register`
2. Заполните форму:
   - **Username** (3-50 символов)
   - **Email** (валидный email)
   - **Password** (минимум 8 символов)
3. Нажмите **Register**
4. Автоматический вход после регистрации

### Login

1. Откройте `/login`
2. Введите **Email** и **Password**
3. JWT token сохраняется в `localStorage`
4. Автоматический redirect на `/dashboard`

### Logout

- Кнопка **Logout** в правом верхнем углу
- Очистка JWT token из `localStorage`
- Redirect на `/login`

### User Roles

| Role | Description | Permissions |
|------|-------------|-------------|
| **Owner** | Создатель организации | Полный доступ, добавление админов |
| **Admin** | Администратор | Управление участниками, API keys |
| **Member** | Участник | Доступ к tenant resources |
| **Viewer** | Наблюдатель | Только чтение |

### Personal Workspace

Каждый пользователь автоматически получает **Personal Tenant**:
- Имя: "username's Personal Workspace"
- Role: Owner
- Используется для personal API keys

---

## 💬 Chat Interface

### Основная страница

`/chat` - ChatGPT-подобный интерфейс

```
┌─────────────────────────────────────────────────────────┐
│ Sidebar          │ Main Chat Area                       │
│                  │                                       │
│ Conversations:   │ ┌─────────────────────────────────┐ │
│ • New Chat       │ │ Model Selection & Parameters    │ │
│ • Chat 1         │ │ ▼ llama3.2  [⚙️ Parameters]     │ │
│ • Chat 2         │ ├─────────────────────────────────┤ │
│                  │ │ Chat Messages                   │ │
│                  │ │ [User message...]               │ │
│                  │ │ [AI response...]                │ │
│                  │ │                                 │ │
│                  │ ├─────────────────────────────────┤ │
│                  │ │ 📊 Context: 1.2K / 32K (3.8%)   │ │
│                  │ └─────────────────────────────────┘ │
│                  │ [Type your message...] [Send]       │
└─────────────────────────────────────────────────────────┘
```

### Model Selection

**Dropdown над input field:**
- Список всех доступных моделей из Ollama
- Сохранение последней выбранной модели в `localStorage`
- Автоматическая загрузка при входе в чат

### Dynamic Parameters Panel

**Кнопка [⚙️ Parameters]** открывает collapsible панель:

```
┌─────────────────────────────────────────────────┐
│ Model Parameters                                │
├─────────────────────────────────────────────────┤
│ Quick Presets:                                  │
│ [🎨 Creative] [⚖️ Balanced] [🎯 Precise] [💻 Coding]│
├─────────────────────────────────────────────────┤
│ Temperature: 0.7      [━━━━━━━━━━]              │
│ Top P: 0.9           [━━━━━━━━━━]              │
│ Top K: 40            [━━━━━━━━━━]              │
│ Context Window: 32000 [━━━━━━━━━━]              │
│ Max Tokens: 2048     [━━━━━━━━━━]              │
└─────────────────────────────────────────────────┘
```

**Quick Presets:**
- 🎨 **Creative**: temp=1.2, top_p=0.95, top_k=60
- ⚖️ **Balanced**: temp=0.7, top_p=0.9, top_k=40 (default)
- 🎯 **Precise**: temp=0.3, top_p=0.5, top_k=20
- 💻 **Coding**: temp=0.2, top_p=0.1, top_k=10

**Параметры сохраняются:**
- В `localStorage` браузера
- Отдельно для каждого чата
- Восстанавливаются при возврате к чату

### Context Tracking

**Real-time индикатор под messages:**

```
📊 Context: 1,234 / 32,000 tokens (3.8%)
```

**Цветовые индикаторы:**
- 🟢 Green: < 70% (нормально)
- 🟡 Yellow: 70-90% (предупреждение)
- 🔴 Red: > 90% (критично)

**Auto-Summarization:**
При достижении 90% контекста:
1. Автоматический вызов `/api/chat/summarize`
2. Сжатие старых сообщений
3. Уведомление пользователя
4. Продолжение чата с освобожденным контекстом

### Conversation Management

**Sidebar:**
- ➕ **New Chat** - создать новый разговор
- 📝 Список всех conversations
- 🗑️ **Delete** при hover на conversation
- 🔍 **Search** для фильтрации (coming soon)

**Auto-save:**
- Каждое сообщение сохраняется автоматически
- ID conversation в URL: `/chat?id=conv_123`
- Восстановление при перезагрузке страницы

---

## 📊 Dashboard

`/dashboard` - Обзор использования и быстрые действия

### Quick Stats

```
┌─────────────────────────────────────────────────┐
│ Total Requests  │ This Month │ API Keys │ Usage │
│     1,234       │    456     │    5     │ 12.5GB│
└─────────────────────────────────────────────────┘
```

### Quick Actions

- ➕ **New Chat** → `/chat`
- 🔑 **Create API Key** → `/api-keys`
- 👥 **Invite Member** → `/tenants`
- 📊 **View Usage** → `/usage`

### Recent Activity

Последние 10 events:
- Chat conversations created
- API keys generated
- Tenant members added
- Usage milestones

---

## 👤 Profile Management

`/profile` - Управление профилем пользователя

### Profile Information

- **Username** (read-only after registration)
- **Email** (editable)
- **Created At** (read-only)
- **Last Login** (read-only)

### Change Password

```
┌────────────────────────────────┐
│ Change Password                │
├────────────────────────────────┤
│ Current Password:              │
│ [••••••••••]                   │
│                                │
│ New Password:                  │
│ [••••••••••]                   │
│                                │
│ Confirm Password:              │
│ [••••••••••]                   │
│                                │
│ [Change Password]              │
└────────────────────────────────┘
```

**Требования:**
- Минимум 8 символов
- Текущий пароль обязателен
- New password != current password

### Delete Account

⚠️ **Danger Zone:**
- Удаление всех conversations
- Удаление всех personal API keys
- Выход из всех tenants (кроме owned)
- Необратимое действие!

---

## 🔑 API Keys

`/api-keys` - Управление API ключами

### Personal Keys Tab

**Привязаны к пользователю:**

```
┌──────────────────────────────────────────────────┐
│ Personal API Keys         [➕ Create New]        │
├────────┬────────┬────────────┬────────┬─────────┤
│ Name   │ Status │ Rate Limit │ Models │ Actions │
├────────┼────────┼────────────┼────────┼─────────┤
│ DevKey │ 🟢     │ 60/min    │ *      │ [🗑️]   │
│ TestKey│ 🔴     │ 10/min    │ llama3 │ [🗑️]   │
└────────┴────────┴────────────┴────────┴─────────┘
```

### Tenant Keys Tab

**Привязаны к организации:**

Требуется роль **Owner** или **Admin** в tenant.

```
┌──────────────────────────────────────────────────┐
│ Select Tenant: [My Organization ▼]               │
├────────┬────────┬────────────┬────────┬─────────┤
│ Name   │ Status │ Rate Limit │ Models │ Actions │
├────────┼────────┼────────────┼────────┼─────────┤
│ ProdKey│ 🟢     │ 1000/min  │ gpt-4  │ [🗑️]   │
└────────┴────────┴────────────┴────────┴─────────┘
```

### Create API Key

```
┌────────────────────────────────┐
│ Create New API Key             │
├────────────────────────────────┤
│ Key Name:                      │
│ [Production Key...............]│
│                                │
│ Rate Limits:                   │
│ • Per Minute: [60...........]  │
│ • Per Hour:   [1000.........]  │
│                                │
│ Allowed Models:                │
│ [*] (* for all, comma-sep)     │
│                                │
│ Expires In (days):             │
│ [30..........................]  │
│                                │
│ [Cancel] [Create]              │
└────────────────────────────────┘
```

**После создания:**

```
✅ API Key Created!

⚠️ Save this key now - you won't see it again!

┌────────────────────────────────────┐
│ sk-1234567890abcdef1234567890abcd │
│          [📋 Copy to Clipboard]    │
└────────────────────────────────────┘

[Done]
```

---

## 👥 Tenants & Teams

`/tenants` - Управление организациями

### My Tenants List

```
┌─────────────────────────────────────────────────┐
│ My Tenants                  [➕ Create Tenant]  │
├──────────────┬──────┬────────┬─────────────────┤
│ Name         │ Role │ Members│ Actions         │
├──────────────┼──────┼────────┼─────────────────┤
│ Personal WS  │ Owner│   1    │ [👁️]           │
│ My Company   │ Owner│   12   │ [👁️] [✏️] [🗑️]│
│ Client Team  │ Admin│   5    │ [👁️] [✏️]      │
└──────────────┴──────┴────────┴─────────────────┘
```

### Create Tenant

```
┌────────────────────────────────┐
│ Create Organization            │
├────────────────────────────────┤
│ Organization Name:             │
│ [Acme Corporation.............]│
│                                │
│ Description (optional):        │
│ [Our main development team...]│
│                                │
│ [Cancel] [Create]              │
└────────────────────────────────┘
```

### Manage Members

**View Tenant → Members Tab:**

```
┌─────────────────────────────────────────────────┐
│ Members                     [➕ Add Member]     │
├───────────┬──────────────┬──────┬──────────────┤
│ Username  │ Email        │ Role │ Actions      │
├───────────┼──────────────┼──────┼──────────────┤
│ john_doe  │ john@co.com  │ Owner│ -            │
│ jane_smith│ jane@co.com  │ Admin│ [✏️] [🗑️]   │
│ bob_dev   │ bob@co.com   │ Member│[✏️] [🗑️]   │
└───────────┴──────────────┴──────┴──────────────┘
```

**Add Member Modal:**

```
┌────────────────────────────────┐
│ Add Member to Tenant           │
├────────────────────────────────┤
│ Search User:                   │
│ [john@........................]│
│                                │
│ Search Results:                │
│ ┌────────────────────────────┐ │
│ │ john_doe (john@example.com)│ │
│ │ johnny (johnny@corp.com)   │ │
│ └────────────────────────────┘ │
│                                │
│ Assign Role:                   │
│ • ( ) Admin                    │
│ • (•) Member                   │
│ • ( ) Viewer                   │
│                                │
│ [Cancel] [Add]                 │
└────────────────────────────────┘
```

**Live Search:**
- Поиск по username или email
- Автоматический поиск при вводе
- Только users не в текущем tenant

**Change Role:**
- Кнопка [✏️] открывает modal
- Выбор новой роли (Admin/Member/Viewer)
- Owner role не изменяется

**Remove Member:**
- Кнопка [🗑️] с confirmation
- Удаление из tenant membership
- Revoke доступа к tenant resources

---

## 📈 Usage Statistics

`/usage` - Детальная аналитика

### Filters

```
┌─────────────────────────────────────────┐
│ Date Range: [Last 7 Days ▼]            │
│ Tenant:     [All Tenants ▼]            │
│ Model:      [All Models ▼]             │
│                      [Apply Filters]    │
└─────────────────────────────────────────┘
```

### Usage Cards

```
┌──────────────────────────────────────────────────┐
│ Total Requests │ Total Tokens │ Avg Latency     │
│    1,234       │   456.7K     │    2.3s         │
└──────────────────────────────────────────────────┘
```

### Usage Table

```
┌────────────┬───────┬────────┬─────────┬─────────┐
│ Date       │ Model │ Tokens │ Requests│ Latency │
├────────────┼───────┼────────┼─────────┼─────────┤
│ 2025-10-14 │ llama3│ 12.5K  │   45    │  2.1s   │
│ 2025-10-13 │ qwen  │ 8.3K   │   32    │  1.9s   │
└────────────┴───────┴────────┴─────────┴─────────┘
```

### Export

- **📊 Export CSV** - вся таблица usage
- **📄 Export JSON** - raw данные для анализа

---

## 🛡️ Admin Panel

`/admin` - Только для admin users (JWT проверка)

### Navigation Tabs

```
┌─────────────────────────────────────────┐
│ [Models] [System] [Logs]                │
└─────────────────────────────────────────┘
```

### Models Tab

**Available Models:**

```
┌─────────────────────────────────────────┐
│ Available Models      [🔄 Refresh]      │
├─────────────────────────────────────────┤
│ ▶ llama3.2:latest             3.2 GB   │
│   Family: llama • Params: 3B            │
│   Modified: 2025-10-12                  │
│                                         │
│ ▶ qwen2.5-coder:7b            4.7 GB   │
│   Family: qwen2 • Params: 7B            │
│   Modified: 2025-10-10                  │
└─────────────────────────────────────────┘
```

### System Tab

**Performance Monitoring:**

```
┌─────────────────────────────────────────┐
│ Quick Stats Cards                       │
├─────────────────────────────────────────┤
│ CPU Usage  │ Memory   │ Goroutines     │
│   12.5%    │  1.2 GB  │     45         │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│ NVIDIA GPU Metrics                      │
├─────────────────────────────────────────┤
│ GPU 0: RTX 3080                         │
│ • Temp: 47°C  • Power: 104W             │
│ • GPU Load: 3%  • VRAM: 3.6/10 GB      │
│ • Clock: 1890 MHz  • Fan: 61%          │
│                                         │
│ GPU 1: RTX 3080                         │
│ • Temp: 45°C  • Power: 98W              │
│ • GPU Load: 2%  • VRAM: 2.1/10 GB      │
│ • Clock: 1875 MHz  • Fan: 58%          │
└─────────────────────────────────────────┘

[Open Advanced Dashboard →] (MoniGo :9091)
```

**Auto-refresh:** каждые 5 секунд

**MoniGo Dashboard:**
- Открывается на порту **9091**
- Полные возможности real-time monitoring
- CPU, Memory, Network, Disk, HTTP metrics

**GPU Monitoring:**
- Только Linux/macOS (Windows = stub)
- Multi-GPU support (unified card)
- Real-time через `nvidia-smi` CLI
- Цветовые индикаторы температуры

### Logs Tab

**Real-time Log Streaming:**

```
┌─────────────────────────────────────────┐
│ System Logs                [🔄 Refresh] │
├─────────────────────────────────────────┤
│ File: [proxy-dev.log ▼]                 │
│ Level: [All ▼] [Error] [Warn] [Info]   │
├─────────────────────────────────────────┤
│ 🔵 14:32:15 INFO  Server started :8080  │
│ 🔵 14:32:16 INFO  Request processed     │
│ 🟠 14:32:20 WARN  Slow request 5.2s     │
│ 🔴 14:32:25 ERROR Failed to connect     │
└─────────────────────────────────────────┘
```

**Features:**
- **SSE Streaming** - real-time через Server-Sent Events
- **Syntax Highlighting** - цветовые уровни
- **Filtering** - по level (All/Error/Warn/Info/Debug)
- **File Selection** - выбор лог-файла
- **Auto-scroll** - к последним записям

---

## 📚 MCP Catalog

`/mcp` - Справочник MCP серверов

### Catalog View

```
┌─────────────────────────────────────────────────┐
│ MCP Servers Catalog         [Search...........]│
├─────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────┐ │
│ │ 🔧 Weather Service                          │ │
│ │ Category: Utilities                         │ │
│ │ Provides weather information via API       │ │
│ │ [View Details]                              │ │
│ └─────────────────────────────────────────────┘ │
│                                                 │
│ ┌─────────────────────────────────────────────┐ │
│ │ 📊 Database Tools                           │ │
│ │ Category: Development                       │ │
│ │ SQL query execution and management         │ │
│ │ [View Details]                              │ │
│ └─────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────┘
```

**Admin only:**
- ➕ **Add MCP Server**
- ✏️ **Edit** existing servers
- 🗑️ **Delete** servers

**All users:**
- 👁️ **View** catalog
- 🔍 **Search** servers
- 📋 **Copy** connection details

---

## 📖 About & Changelog

`/about` - Информация о системе

### System Info

```
┌─────────────────────────────────────────┐
│ Ollama-OpenAI Proxy                     │
│ Version: 1.9.3                          │
│ Build Date: 2025-10-14                  │
│ Go Version: 1.25+                       │
└─────────────────────────────────────────┘
```

### Changelog History

**Accordion UI:**

```
┌─────────────────────────────────────────┐
│ ▼ Version 1.9.3 - 2025-10-14            │
├─────────────────────────────────────────┤
│ ### Added                               │
│ - MoniGo Performance Dashboard          │
│ - NVIDIA GPU Monitoring                 │
│ - Admin Panel Reorganization            │
│                                         │
│ ### Technical                           │
│ - MoniGo integration on port 9091       │
│ - GPU metrics through nvidia-smi        │
│ └─────────────────────────────────────┘ │
│                                         │
│ ▶ Version 1.9.2 - 2025-10-13            │
│ ▶ Version 1.9.1 - 2025-10-13            │
│ ▶ Version 1.6.3 - 2025-10-12            │
└─────────────────────────────────────────┘
```

**Features:**
- Markdown rendering в каждой версии
- Collapsible sections
- Автоматическая загрузка из БД
- API: `/api/system/changelogs`

---

## 🔧 Troubleshooting

### Chat не отправляет сообщения

**Причина:** Модель не выбрана

**Решение:**
1. Выберите модель из dropdown
2. Проверьте что Ollama запущен
3. Проверьте `/admin` → Models

### Context tracking показывает 0%

**Причина:** Token estimation не работает

**Решение:**
1. Проверьте логи сервера
2. Обновите страницу
3. Проверьте API `/api/chat/estimate`

### API Keys не создаются

**Причина:** Недостаточные права

**Решение:**
1. Для Tenant keys нужна роль Owner/Admin
2. Проверьте роль в `/tenants`
3. Используйте Personal keys вместо Tenant

### GPU Metrics не отображаются

**Причина:** Windows или nvidia-smi недоступен

**Решение:**
1. GPU monitoring работает только в Linux/macOS
2. Проверьте `nvidia-smi` в терминале
3. Windows использует stub версию (disabled)

### MoniGo Dashboard 404

**Причина:** MoniGo не запущен

**Решение:**
1. Проверьте `configs/dev.yaml`:
   ```yaml
   performance:
     monigo:
       enabled: true
       port: 9091
   ```
2. Restart server
3. Откройте `http://localhost:9091`

---

## 🎯 Best Practices

### Security

1. **Используйте сложные пароли** (минимум 8 символов)
2. **Регулярно меняйте API keys** (особенно tenant keys)
3. **Проверяйте members** организаций
4. **Используйте HTTPS** в production
5. **Не делитесь JWT tokens**

### Performance

1. **Используйте context tracking** для оптимизации
2. **Включайте auto-summarization** при 90%
3. **Выбирайте подходящие presets** для задач
4. **Мониторьте GPU** при больших нагрузках
5. **Настройте rate limits** для API keys

### UX

1. **Сохраняйте важные conversations** (export)
2. **Используйте Markdown** в сообщениях
3. **Настраивайте parameters** под задачу
4. **Проверяйте usage statistics** регулярно
5. **Используйте Quick Presets** для быстрой настройки

---

**Version:** 1.9.3  
**Last Updated:** 2025-10-14  
**Author:** Ollama-OpenAI Proxy Team
