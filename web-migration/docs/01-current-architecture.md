# Текущая архитектура WebUI

## Обзор файловой структуры

```
web/
├── css/
│   ├── dashboard.css      # Стили dashboard и общие компоненты
│   ├── model-panel.css    # Панель выбора модели в чате
│   ├── monitor.css        # Стили для системного мониторинга
│   ├── notifications.css  # Toast уведомления
│   ├── settings.css       # Страница настроек
│   ├── style.css          # Основные глобальные стили (~2700 строк)
│   └── theme.css          # CSS переменные темы
│
├── js/
│   ├── components/
│   │   ├── navbar.js      # Навигационная панель
│   │   └── navigation.js  # Маршрутизация (не используется?)
│   │
│   ├── utils/
│   │   ├── clipboard.js   # Копирование в буфер
│   │   ├── error-boundary.js # Обработка ошибок
│   │   └── notifications.js  # Toast система
│   │
│   ├── api.js             # API клиент (~860 строк)
│   ├── apikeys.js         # Логика API ключей
│   ├── app.js             # Инициализация приложения
│   ├── auth-guard.js      # Защита маршрутов
│   ├── chat.js            # Chat логика (основной файл)
│   ├── context-manager.js # Управление контекстом чата
│   ├── dashboard.js       # Dashboard логика
│   ├── files.js           # Работа с файлами
│   ├── gpu-monitor.js     # GPU мониторинг
│   ├── logs.js            # Логирование
│   ├── mcp-catalog.js     # MCP каталог
│   ├── mcp.js             # MCP интеграция
│   ├── model-panel.js     # Панель параметров модели
│   ├── monitor.js         # Системный мониторинг
│   ├── performance.js     # Метрики производительности
│   ├── profile-devices.js # Управление устройствами
│   ├── profile.js         # Профиль пользователя
│   ├── rag-sources.js     # RAG источники
│   ├── tenants.js         # Управление организациями
│   ├── usage.js           # Статистика использования
│   └── vlm.js             # Vision Language Model
│
├── static/
│   └── index.html         # Статический fallback
│
└── [26 HTML файлов]       # Страницы приложения
```

## HTML Страницы (26)

### Публичные (без аутентификации)
| Файл | Назначение |
|------|------------|
| `login.html` | Страница входа |
| `register.html` | Регистрация |
| `bootstrap.html` | Первоначальная настройка системы |

### Защищённые (требуют JWT)
| Файл | Назначение |
|------|------------|
| `dashboard.html` | Главная панель |
| `chat.html` | Chat интерфейс |
| `api-keys.html` | Управление API ключами |
| `tenants.html` | Управление организациями |
| `profile.html` | Профиль пользователя |
| `profile-devices.html` | Управление устройствами |
| `usage.html` | Статистика использования |
| `files.html` | Файловый менеджер |
| `rag-sources.html` | RAG источники данных |
| `mcp.html` | MCP интеграция |
| `mcp-catalog.html` | Каталог MCP серверов |
| `about.html` | О системе (changelog) |
| `monitor.html` | Системный мониторинг |
| `downloads.html` | Активные загрузки |
| `huggingface.html` | HuggingFace браузер |
| `yzma.html` | Локальный инференс |

### Админ панель
| Файл | Назначение |
|------|------------|
| `admin.html` | Главная админ панель (~2500 строк!) |
| `admin-audit.html` | Аудит лог |
| `admin-invitations.html` | Управление приглашениями |
| `admin-rag.html` | RAG управление (админ) |
| `admin-rbac.html` | RBAC управление |
| `admin-registry.html` | Реестр моделей |

## Текущий UI Framework (v3.1.0)

### Структура
```
internal/web/framework/
├── assets/
│   ├── components/
│   │   ├── http.js      # HTTP клиент
│   │   ├── router.js    # SPA роутер
│   │   └── state.js     # Глобальное состояние
│   ├── framework.css    # Базовые стили фреймворка
│   └── framework.js     # Точка входа
├── builder.go           # Сборщик фреймворка
├── handler.go           # HTTP обработчик
└── minifier.go          # Минификация
```

### Функциональность framework.js
- `modal` — Модальные окна (confirm, danger, info)
- `toast` — Toast уведомления
- `clipboard` — Копирование в буфер
- HTTP клиент с auth headers
- Простой SPA роутер

## CSS Архитектура

### CSS Variables (theme.css)
```css
:root {
    --bg-primary: #212121;
    --bg-secondary: #2d2d2d;
    --bg-tertiary: #3a3a3a;
    --bg-hover: #404040;
    --text-primary: #ececec;
    --text-secondary: #b3b3b3;
    --accent-primary: #10a37f;
    --accent-hover: #0d8c6d;
    --border-color: #4a4a4a;
    --error-color: #ef4444;
    --success-color: #10b981;
    --code-bg: #1e1e1e;
}
```

### Основные компоненты в CSS
- `.auth-page`, `.auth-box` — Страницы аутентификации
- `.dashboard-page`, `.dashboard-container` — Dashboard layout
- `.chat-page`, `.chat-main`, `.sidebar` — Chat layout
- `.btn`, `.btn-primary`, `.btn-secondary` — Кнопки
- `.form-group`, `.form-control` — Формы
- `.modal`, `.modal-content` — Модальные окна
- `.card`, `.stat-card` — Карточки
- `.table-container`, `.data-table` — Таблицы
- `.tabs`, `.tab-btn`, `.tab-content` — Табы

## API Client (api.js)

### Структура класса API
```javascript
class API {
    baseURL
    accessToken
    refreshToken
    cache         // Request кэш (5 минут)
    cacheable     // Список кэшируемых endpoints
    
    // Auth
    refreshAccessToken()
    logout()
    getCurrentUser()
    
    // HTTP shortcuts
    get(path, useCache)
    post(path, data)
    put(path, data)
    delete(path)
    
    // Models
    getModels()
    
    // Chat
    sendChatMessage(messages, modelParams, stream)
    *streamChatMessage(messages, modelParams)
    
    // Conversations
    getConversations()
    getConversation(id)
    createConversation(title, model)
    updateConversation(id, updates)
    deleteConversation(id)
    
    // Messages
    getMessages(conversationId)
    createMessage(conversationId, role, content, model, fileIds)
    
    // Profile
    updateUserProfile(data)
    changePassword(currentPassword, newPassword)
    deleteAccount()
    
    // Tenants
    getUserTenants()
    getTenant(tenantId)
    createTenant(data)
    updateTenant(tenantId, data)
    deleteTenant(tenantId)
    getTenantMembers(tenantId)
    searchTenantUsers(tenantId, query)
    addTenantMember(tenantId, userInfo, role)
    updateTenantMember(tenantId, userId, role)
    removeTenantMember(tenantId, userId)
    
    // API Keys
    getPersonalAPIKeys()
    createPersonalAPIKey(data)
    deletePersonalAPIKey(keyId)
    getTenantAPIKeys(tenantId)
    createTenantAPIKey(tenantId, data)
    deleteTenantAPIKey(tenantId, keyId)
    
    // RAG
    getRAGSources(filters)
    getRAGSource(sourceId)
    createRAGSource(data)
    updateRAGSource(sourceId, data)
    deleteRAGSource(sourceId)
    testRAGConnection(data)
    syncRAGSource(sourceId)
    
    // RBAC
    getRBACPermissions()
    getRBACRoles(includePermissions)
    createRBACRole(data)
    updateRBACRole(roleId, data)
    deleteRBACRole(roleId)
    addRolePermission(roleId, permissionId)
    removeRolePermission(roleId, permissionId)
    getUserRoles(userId, includeDetails)
    assignUserRole(userId, data)
    removeUserRole(userId, roleId, tenantId)
    getAdminUsers()
    getAdminTenants()
    
    // Audit
    getAuditStats()
    getAuditLogs(filters)
    exportAuditLogs(filters)
    
    // Invitations
    createInvitation(data)
    listInvitations(queryString)
    getInvitationStats()
    getInvitationDetails(id)
    revokeInvitation(id, data)
    validateInvitation(token, email)
    
    // System
    getSystemInfo()
    getDashboardStats()
    getAdminSummary()
    isRAGEnabled()
}
```

## Auth Guard (auth-guard.js)

### Публичные страницы
```javascript
const PUBLIC_PAGES = [
    '/',
    '/login.html',
    '/register.html',
    '/bootstrap.html',
    '/web/login.html',
    '/web/register.html',
    '/web/bootstrap.html'
];
```

### Логика
1. Проверка на публичную страницу
2. Проверка наличия `access_token` в localStorage
3. Редирект на `/login.html` если не авторизован
4. Сохранение `return_url` для редиректа после логина
5. Периодическая проверка токена (60 секунд)
6. Проверка при смене вкладки (visibilitychange)

## Внешние зависимости

### CDN
- **HTMX** `1.9.10` / `2.0.4` — Декларативный AJAX
- **Marked.js** `11.1.1` — Markdown рендеринг
- **Highlight.js** `11.9.0` — Подсветка кода
- **Bootstrap** `5.3.0` — Только для RBAC страницы (частично)
- **jQuery** `3.7.0` — Только для RBAC страницы
- **Font Awesome** `6.4.0` — Иконки (используется редко)

## Embed интеграция (internal/web/embed.go)

```go
package web

import (
    "embed"
)

//go:embed all:static
var StaticFiles embed.FS
```

### Текущая структура embedded файлов
```
internal/web/static/
├── css/
│   ├── style.css
│   └── themes.css
├── js/
│   ├── app.js
│   ├── htmx.min.js
│   └── websocket.js
└── index.html
```

## Проблемы текущей архитектуры

### 1. Дублирование кода
- Каждая HTML страница содержит схожий boilerplate
- CSS классы повторяются в разных файлах
- Одинаковые компоненты (модальные окна, таблицы) описаны в каждом файле

### 2. Отсутствие компонентной архитектуры
- Нет переиспользуемых компонентов
- navbar.js просто вставляет HTML строку
- Сложно поддерживать консистентность UI

### 3. Смешение логики и представления
- HTML содержит inline JavaScript в `<script>` тегах
- Обработчики событий через `onclick` атрибуты
- Состояние хранится в глобальных переменных

### 4. Монолитные файлы
- `admin.html` — 2500+ строк
- `style.css` — 2700+ строк
- Сложно найти нужный код

### 5. Несовместимые подходы
- Часть страниц использует HTMX
- Часть использует fetch API напрямую
- Часть использует Bootstrap, часть — кастомные стили

### 6. Отсутствие типизации
- JavaScript без типов
- Легко допустить ошибку в именах полей API
- Нет автодополнения в IDE

## Рекомендации для миграции

1. **Один подход** — Svelte 5 + shadcn-svelte везде
2. **Компонентная архитектура** — Выделить общие компоненты
3. **TypeScript** — Типизация API и состояния
4. **Store/Runes** — Централизованное управление состоянием
5. **SPA роутинг** — Один entry point, клиентская навигация
6. **Static build** — Сохранить embed.FS совместимость

