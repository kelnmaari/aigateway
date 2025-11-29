# Инвентарь функциональности WebUI

## Модуль: Аутентификация

### login.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Форма входа | Username/Email + Password | `POST /api/auth/login` |
| Remember Me | Сессия 24 часа | - |
| Init Status Check | Проверка bootstrap | `GET /api/system/init-status` |
| System Info | Версия, модели, RAG статус | `GET /api/system/info`, `GET /api/system/models` |
| Token Storage | access_token, refresh_token в localStorage | - |

### register.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Форма регистрации | Username, Email, Password, Display Name | `POST /api/auth/register` |
| Invitation Token | Поддержка invite-only режима | `GET /api/invitations/{token}/validate` |
| Password Validation | Мин 8 символов, uppercase, lowercase, digit, special | - |

### bootstrap.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Bootstrap Token | Токен из конфигурации | - |
| Admin Creation | Создание первого администратора | `POST /api/system/bootstrap` |
| Password Validation | Те же требования что и register | - |

---

## Модуль: Dashboard

### dashboard.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Stats Cards | Conversations, Tenants, API Keys, Requests | `GET /api/dashboard/stats` |
| Available Models | Список загруженных моделей | `GET /api/models` |
| Quick Actions | Ссылки на основные функции | - |
| Recent Conversations | Последние 5 диалогов | `GET /api/conversations?limit=5` |
| Your Tenants | Список организаций пользователя | `GET /api/users/me/tenants` |

---

## Модуль: Chat

### chat.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Sidebar | Список conversations, new chat, logout | `GET /api/conversations` |
| Model Selector | Выбор модели из yzma | `GET /api/models` |
| Parameters Panel | Temperature, Top P, Max Tokens, Context Window | - |
| Presets | Creative, Balanced, Precise, Coding | - |
| RAG Section | Включение RAG, выбор sources, TopK, MinScore | `GET /api/rag/sources` |
| Messages | Рендеринг markdown, code highlight | - |
| Streaming | SSE для потоковых ответов | `POST /v1/chat/completions` |
| File Attachment | Прикрепление файлов | `POST /api/files/upload` |
| Image Attachment | VLM поддержка | - |
| Export/Import | JSON, Markdown, Text форматы | - |
| Context Indicator | Подсчёт токенов | - |
| Formatting Toolbar | Bold, Italic, Code, Link, Lists | - |

---

## Модуль: API Keys

### api-keys.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Personal Keys Tab | Личные API ключи | `GET /api/users/me/api-keys` |
| Tenant Keys Tab | Ключи организации | `GET /api/tenants/{id}/api-keys` |
| Create Key Modal | Name, Description, Rate Limits, Models | `POST /api/users/me/api-keys`, `POST /api/tenants/{id}/api-keys` |
| Key Created Modal | Показ ключа (один раз) | - |
| Delete Key | Подтверждение удаления | `DELETE /api/users/me/api-keys/{id}` |
| Tenant Selector | Выбор организации для tenant keys | `GET /api/users/me/tenants` |

---

## Модуль: Tenants

### tenants.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Tenants Grid | Карточки организаций | `GET /api/users/me/tenants` |
| Create Tenant Modal | Name, Slug, Description | `POST /api/tenants` |
| Tenant Detail Modal | Info, Members, Actions | `GET /api/tenants/{id}` |
| Members Table | User, Role, Joined, Actions | `GET /api/tenants/{id}/members` |
| Add Member Modal | Search User + Role Select | `GET /api/tenants/{id}/search-users`, `POST /api/tenants/{id}/members` |
| Edit Tenant | Изменение данных | `PUT /api/tenants/{id}` |
| Delete Tenant | Подтверждение удаления | `DELETE /api/tenants/{id}` |

---

## Модуль: Profile

### profile.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Profile Info Form | Username (readonly), Email, Full Name | `GET /api/users/me`, `PUT /api/users/me` |
| Change Password | Current + New + Confirm | `POST /api/users/me/password` |
| Account Info | Status, Member Since, Last Login | `GET /api/users/me` |
| Danger Zone | Delete Account | `DELETE /api/users/me` |

### profile-devices.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Filters | Status, Sort By, Order | - |
| Stats | Total Devices, Active Devices, Current Device | `GET /api/users/me/devices` |
| Devices Grid | Device cards с информацией | `GET /api/users/me/devices` |
| Device Details Modal | Полная информация об устройстве | - |
| Revoke Device | Отзыв доступа устройства | `DELETE /api/users/me/devices/{id}` |
| Bulk Actions | Массовый отзыв устройств | - |

---

## Модуль: Usage

### usage.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Time Period Selector | Today, 7 days, 30 days, 90 days, All | - |
| Scope Selector | Personal / Organization | - |
| Summary Stats | Total Requests, Avg Response Time, Tokens Used, Error Rate | `GET /api/usage/stats` |
| Usage by Model | Table с метриками по моделям | `GET /api/usage/by-model` |
| Usage by API Key | Table с метриками по ключам | `GET /api/usage/by-key` |
| Recent Requests | Последние запросы | `GET /api/usage/requests` |

---

## Модуль: RAG

### rag-sources.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Sources Table | Name, Type, Status, Last Sync, Chunks, Tokens | `GET /api/rag/sources` |
| Create Source Modal | Name, Description, Type + Config | `POST /api/rag/sources` |
| API Config | URL, Method, Auth (Bearer/API Key/Basic) | - |
| Database Config | Host, Port, DB Name, Credentials, SQL Query | - |
| File Config | File upload (TXT, PDF, DOC, DOCX, MD) | - |
| Web Config | URL, Crawl Depth | - |
| Indexing Settings | Chunk Size, Chunk Overlap | - |
| Test Connection | Проверка подключения | `POST /api/rag/sources/test-connection` |
| Sync Source | Запуск синхронизации | `POST /api/rag/sources/{id}/sync` |
| Delete Source | Удаление источника | `DELETE /api/rag/sources/{id}` |

---

## Модуль: Files

### files.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Upload Area | Drag & drop / Click to upload | `POST /api/files/upload` |
| Filters | Search, Type, Status | - |
| Files Grid | File cards с preview | `GET /api/files` |
| Text Preview Modal | Просмотр извлечённого текста | `GET /api/files/{id}/text` |
| Delete File | Удаление файла | `DELETE /api/files/{id}` |
| Pagination | Постраничная навигация | - |

---

## Модуль: MCP

### mcp.html / mcp-catalog.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Search | Поиск по имени, описанию, тегам | - |
| Filters | Category, Sort By | - |
| Servers Grid | Карточки MCP серверов | `GET /api/mcp/servers` |
| Server Details Modal | Полная информация о сервере | `GET /api/mcp/servers/{id}` |
| Pagination | Постраничная навигация | - |

---

## Модуль: Monitor

### monitor.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| GPU Metrics | HTMX polling каждые 5 секунд | `GET /ui/monitor/gpu-metrics` |
| System Resources | CPU, Memory, Active Requests, Uptime | `GET /api/system/stats` |
| Request Rate | RPS, Avg Response Time, Success Rate, Error Rate | - |
| Live Indicator | Индикатор real-time обновления | - |

---

## Модуль: Admin (admin.html - главная админ-панель)

### Dashboard Tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Stats Cards | Users, API Keys, Models, Requests (30d) | `GET /api/ui/dashboard/stats/*` |
| System Health Table | Status, Active Keys, Rate Limits | `GET /api/ui/dashboard/system-health` |

### Users & RBAC Tab

#### Users Sub-tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Users Table | Username, Email, Auth Provider, RBAC Roles, Tenants, Admin, Status, Created, Actions | `GET /api/ui/users` (HTMX) |
| Auth Provider Badge | Local, OIDC, LDAP | - |
| RBAC Roles Badges | Role badges с tenant indicator | - |
| Tenants Badges | Tenant name + role (owner/member) | - |
| Create User Modal | Username, Email, Password, Full Name, Is Admin | `POST /api/admin/users` |
| Edit User Modal | Email, Full Name, Is Admin, Status | `PUT /api/admin/users/{id}` |
| Reset Password Modal | New Password, Confirm | `POST /api/admin/users/{id}/reset-password` |
| Disable/Enable User | Изменение статуса | `PATCH /api/admin/users/{id}/disable`, `/enable` |
| Delete User | Удаление пользователя | `DELETE /api/admin/users/{id}` |

#### Invitations Sub-tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Quick Stats | Active, Pending, Used, Expired | `GET /api/admin/invitations/stats` |
| Recent Invitations Table | Token, Email, Status, Usage, Expires, Created | `GET /api/admin/invitations?limit=10` |
| Link to Full Page | Переход на admin-invitations.html | - |

#### RBAC Sub-tab (Roles)
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Roles Table | Role, Scope, Permissions, Users, Created, Actions | `GET /api/ui/rbac/roles` (HTMX) |
| Create Role Modal | Name, Description, Permissions | `POST /api/admin/rbac/roles` |

#### RBAC Sub-tab (Permissions)
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Permissions Table | Permission, Resource, Action, Created | `GET /api/ui/rbac/permissions` (HTMX) |

### API Keys Tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Keys Table | Name, Owner, Organization, Key Preview, Models, Rate Limit, Last Used, Actions | `GET /api/admin/keys` |
| Create API Key Modal | Name, Description, Models, Permissions, Rate Limit | `POST /api/admin/keys` |
| Delete Key | Удаление ключа | `DELETE /api/admin/keys/{id}` |

### Models Tab

#### Local Models (yzma) Sub-tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| yzma Statistics | Loaded models, Memory, Context (polling 5s) | `GET /api/ui/yzma/stats` (HTMX) |
| Currently Loaded Models | Model name, Size, Context, Unload button (polling 3s) | `GET /api/ui/yzma/loaded` (HTMX) |
| Available GGUF Models | Model list with Load button | `GET /api/ui/yzma/models` (HTMX) |
| Load Model | Загрузка модели в память | `POST /api/ui/yzma/load` |
| Unload Model | Выгрузка модели из памяти | `POST /api/ui/yzma/unload` |

#### Hugging Face Browser Sub-tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Search Form | Search, Author, Sort, Limit | `GET /api/ui/huggingface/search` (HTMX) |
| Quick Filters | Llama, Mistral, Phi, Gemma, Vision | - |
| Repeat Last Search | Повтор последнего поиска (localStorage) | - |
| Search Results | Grid моделей | - |
| Download Modal | Выбор GGUF файла и скачивание | `POST /api/ui/huggingface/download` |

### Downloads Tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Downloads List | Active downloads (polling 2s) | `GET /api/ui/huggingface/downloads` (HTMX) |
| Progress Bar | Индикатор прогресса | - |
| Cancel Download | Отмена загрузки | `POST /api/ui/huggingface/downloads/{id}/cancel` |

### Files Tab (Admin)
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Stats Cards | Total Files, Total Size, Extracted, Pending | `GET /api/admin/files/stats` |
| Filters | Search, Type (TXT/PDF/CSV/DOCX), Status | - |
| Files Table | Filename, Owner, Type, Size, Status, Words, Created, Actions | `GET /api/admin/files` |
| Delete File | Удаление файла | `DELETE /api/admin/files/{id}` |
| Pagination | Page navigation | - |

### RAG Sources Tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Info Card | RAG system overview | - |
| Link to Full Page | Переход на admin-rag.html | - |

### MCP Tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| MCP Servers Table | Name, Category, Description, Tags, Status, Actions | `GET /api/admin/mcp/servers` |
| Category Filter | Dropdown filter | `GET /api/mcp/categories` |
| Link to Public Catalog | Переход на mcp.html | - |
| Create/Edit MCP Modal | Name, Category, Description, Installation, URLs, Tags, Active | `POST/PUT /api/admin/mcp/servers` |
| Delete Server | Удаление сервера | `DELETE /api/admin/mcp/servers/{id}` |

### Settings Tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Settings Categories | Accordion по категориям | `GET /api/admin/settings` |
| Category Icons | Server 🖥️, Inference 🤖, Auth 🔐, Database 💾, etc. | - |
| Settings Table | Key, Value, Type, Description | - |
| Storage Badge | DB/YAML indicator | - |
| Restart Badge | Restart Required / Live Reload | - |
| Inline Editing | Edit + Confirmation modal | `PUT /api/admin/settings/{id}` |

### Backups Tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Create Backup | Создание резервной копии | `POST /api/admin/backup` |
| Backups Table | Filename, Created, Size, Actions | `GET /api/admin/backups` |
| Download Backup | Скачивание | `GET /api/admin/backup/{filename}` |
| Restore Backup | Восстановление (с предупреждением) | `POST /api/admin/restore/{filename}` |
| Delete Backup | Удаление | `DELETE /api/admin/backup/{filename}` |

### System & Logs Tab

#### Performance Sub-tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Quick Stats | CPU Usage, Memory Usage, Goroutines, System Health | `GET /api/admin/stats` |
| GPU Metrics | NVIDIA GPU cards (utilization, memory, temp) | - |
| MoniGo Link | Ссылка на advanced dashboard | - |

#### Audit Sub-tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Quick Stats (24h) | Total Events, Critical, Warnings, Failed Logins | `GET /api/admin/audit/stats` |
| Recent Events Table | Timestamp, Event, Severity, Actor, Action, Status | `GET /api/admin/audit?limit=20` |
| Link to Full Page | Переход на admin-audit.html | - |

#### Logs Sub-tab
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Log File Selector | Dropdown со списком файлов | `GET /api/admin/logs` |
| Level Filters | All, DEBUG, INFO, WARN, ERROR | - |
| Logs Viewer | Formatted log entries | `GET /api/admin/logs/{filename}` |
| Real-time Streaming | SSE для live логов | `GET /api/admin/logs/stream?token=&file=` |
| Download Log | Скачивание лог-файла | `GET /api/admin/logs/{filename}/download` |

---

## Модуль: Admin (Отдельные страницы)

### admin-audit.html (Полный Audit Log)
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Stats | Critical, Warning, Info, Failed Logins | `GET /api/admin/audit/stats` |
| Filters | Event Type, Severity, Resource, Status, Date Range, Actor ID | - |
| Events Table | Timestamp, Event Type, Severity, Actor, Action, Resource, Status, IP | `GET /api/admin/audit` |
| Pagination | Prev/Next | - |
| Export CSV | Выгрузка в CSV | `GET /api/admin/audit/export` |

### admin-invitations.html (Полное управление приглашениями)
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Stats | Active, Pending, Expired, Revoked | `GET /api/admin/invitations/stats` |
| Filters | Status, Email, Created By | - |
| Invitations Table | ID, Token, Email, Status, Usage, Expires At, Created At, Actions | `GET /api/admin/invitations` |
| Create Invitation Modal | Email (optional), Max Uses, Expiration | `POST /api/admin/invitations` |
| View Invitation Modal | Details + Copy Link | `GET /api/admin/invitations/{id}` |
| Revoke Invitation Modal | Reason | `DELETE /api/admin/invitations/{id}` |

### admin-rag.html (RAG Sources Admin)
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Stats Cards | Total, Active, Error, Total Chunks | - |
| Filters | Source Type, Status, User | - |
| Sources Table | ID, Name, User, Type, Status, Last Sync, Chunks, Tokens, Shared, Actions | `GET /api/admin/rag/sources` |
| Details Modal | Полная информация | `GET /api/admin/rag/sources/{id}` |
| Delete Modal | Подтверждение | `DELETE /api/admin/rag/sources/{id}` |
| Force Sync | Принудительная синхронизация | `POST /api/admin/rag/sources/{id}/sync` |
| Pagination | Prev/Next | - |

### admin-rbac.html (Полное RBAC управление)
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Roles Tab | List, Create, Edit, Delete roles | `GET /api/admin/rbac/roles` |
| Role Details | Name, Display Name, Description, Scope, Permissions | `GET /api/admin/rbac/roles/{id}` |
| Role Modal | Create/Edit role form | `POST /api/admin/rbac/roles`, `PUT /api/admin/rbac/roles/{id}` |
| Permissions Grid | Checkboxes для permissions | `GET /api/admin/rbac/permissions` |
| User Roles Tab | User selector, current roles, assign role | `GET /api/admin/rbac/users/{id}/roles` |
| Permissions Reference Tab | System permissions list | `GET /api/admin/rbac/permissions` |

### admin-registry.html (Model Registry)
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Stats | Total Models, Active Models, Providers, Healthy Providers | - |
| Providers Tab | Grid карточек провайдеров | `GET /api/admin/registry/providers` |
| Create Provider Modal | Name, Type, Base URL, API Key, Priority, Enabled | `POST /api/admin/registry/providers` |
| Health Check | Проверка доступности провайдера | `GET /api/admin/registry/providers/{id}/health` |
| Models Tab | Table с фильтрами | `GET /api/admin/registry/models` |
| Create Model Modal | Model ID, Name, Provider, Description, Capabilities, Tags | `POST /api/admin/registry/models` |
| Discover Models | Auto-discovery моделей | `POST /api/admin/registry/discover` |

---

## Модуль: HuggingFace

### huggingface.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Search Form | Search, Author, Sort, Limit | `GET /api/ui/huggingface/search` |
| Quick Filters | All Popular, Llama, Mistral, Phi, Gemma, Vision | `GET /api/ui/huggingface/popular` |
| Models Results | HTMX динамическая загрузка | - |
| Model Details Modal | Информация о модели | - |
| Download Modal | Скачивание GGUF файлов | `POST /api/ui/huggingface/download` |

### downloads.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Downloads List | HTMX polling каждые 2 секунды | `GET /api/ui/huggingface/downloads` |
| Progress Bar | Индикатор прогресса | - |
| Cancel Download | Отмена загрузки | `POST /api/ui/huggingface/downloads/{id}/cancel` |

---

## Модуль: Yzma

### yzma.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| Stats | HTMX polling каждые 5 секунд | `GET /api/ui/yzma/stats` |
| Available Models | Grid GGUF моделей | `GET /api/ui/yzma/models` |
| Loaded Models | Модели в памяти | `GET /api/ui/yzma/loaded` |
| Load Model | Загрузка модели | `POST /api/ui/yzma/load` |
| Unload Model | Выгрузка модели | `POST /api/ui/yzma/unload` |
| Delete Model | Удаление файла модели | `POST /api/ui/yzma/delete` |

---

## Модуль: About

### about.html
| Функция | Описание | API Endpoint |
|---------|----------|--------------|
| System Info | Version, Git Commit, Build Date, Go Version | `GET /api/system/info` |
| Changelog Accordion | История изменений | `GET /api/system/changelogs` |

---

## Общие компоненты

### Navbar (navbar.js)
| Элемент | Описание |
|---------|----------|
| Logo + Home Link | Ссылка на dashboard |
| Main Navigation | Chat, Dashboard, API Keys, Tenants, Usage |
| Admin Link | Только для admin роли |
| User Menu | Profile, Devices, Logout |
| Theme Toggle | Dark/Light (если есть) |

### Toast Notifications (notifications.js)
| Тип | Описание |
|-----|----------|
| success | Зелёный, позитивное сообщение |
| error | Красный, ошибка |
| info | Синий, информационное |
| warning | Жёлтый, предупреждение |

### Modal System (framework.js)
| Метод | Описание |
|-------|----------|
| `modal.confirm(message, title)` | Подтверждение (OK/Cancel) |
| `modal.danger(message, title, details)` | Опасное действие (красный) |
| `modal.info(message, title)` | Информационное |

