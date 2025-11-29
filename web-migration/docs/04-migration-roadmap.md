# Migration Roadmap

## Обзор фаз миграции

| Phase | Название | Длительность | Статус |
|-------|----------|--------------|--------|
| 0 | Подготовка инфраструктуры | 1-2 дня | ⏳ Pending |
| 1 | Критический функционал | 3-5 дней | ⏳ Pending |
| 2 | Основной функционал | 3-4 дня | ⏳ Pending |
| 3 | Административный функционал | 2-3 дня | ⏳ Pending |
| 4 | Дополнительный функционал | 2-3 дня | ⏳ Pending |
| 5 | Финализация | 1-2 дня | ⏳ Pending |

**Общая оценка: 12-19 дней**

---

## Phase 0: Подготовка инфраструктуры

### Задачи

1. **Инициализация SvelteKit проекта**
   - [ ] Создать `web-svelte/` директорию
   - [ ] Инициализировать SvelteKit с TypeScript
   - [ ] Настроить `adapter-static`
   - [ ] Настроить Vite для proxy в dev режиме

2. **Установка зависимостей**
   - [ ] Svelte 5 + SvelteKit
   - [ ] shadcn-svelte (UI компоненты)
   - [ ] lucide-svelte (иконки)
   - [ ] svelte-sonner (уведомления)
   - [ ] TailwindCSS 4
   - [ ] marked + highlight.js (для чата)

3. **Базовая структура**
   - [ ] Настроить layouts
   - [ ] Настроить routing
   - [ ] Создать base API client
   - [ ] Создать auth store

4. **Интеграция с Go backend**
   - [ ] Обновить `internal/web/embed.go`
   - [ ] Настроить SPA handler
   - [ ] Настроить build pipeline

### Документация

- [Technology Stack](03-technology-stack.md)
- [Embed Integration](08-embed-integration.md)

---

## Phase 1: Критический функционал 🔴

### 1.1 Authentication

**Файлы:** `login.html`, `register.html`, `bootstrap.html`, `auth-guard.js`

**Задачи:**
- [ ] Auth store с Svelte 5 runes
- [ ] Login page
- [ ] Register page с invitation support
- [ ] Bootstrap page
- [ ] AuthGuard component
- [ ] Token refresh logic

**Документация:** [Auth Pages](pages/auth.md)

### 1.2 Chat Interface

**Файлы:** `chat.html`, `chat.js`, `model-panel.js`, `context-manager.js`, `vlm.js`

**Задачи:**
- [ ] Chat store
- [ ] Conversations sidebar
- [ ] Message list с markdown
- [ ] Chat input с streaming
- [ ] Model control panel
- [ ] RAG integration panel
- [ ] VLM image support
- [ ] Export/Import conversations

**Документация:** [Chat Page](pages/chat.md)

### 1.3 Core Components

**Задачи:**
- [ ] Navbar component
- [ ] API client с full coverage
- [ ] Notifications (sonner)
- [ ] Theme system

**Документация:**
- [Component Mapping](05-component-mapping.md)
- [API Layer](06-api-layer.md)
- [State Management](07-state-management.md)

---

## Phase 2: Основной функционал 🟡

### 2.1 Dashboard

**Файлы:** `dashboard.html`, `dashboard.js`

**Задачи:**
- [ ] Stats cards
- [ ] Quick actions
- [ ] Recent conversations
- [ ] User tenants
- [ ] Models grid

**Документация:** [Dashboard Page](pages/dashboard.md)

### 2.2 API Keys Management

**Файлы:** `api-keys.html`, `apikeys.js`

**Задачи:**
- [ ] Personal/Tenant tabs
- [ ] Keys table
- [ ] Create key modal
- [ ] Key value display (one-time)
- [ ] Delete confirmation

**Документация:** [API Keys Page](pages/api-keys.md)

### 2.3 Tenants Management

**Файлы:** `tenants.html`, `tenants.js`

**Задачи:**
- [ ] Tenants grid
- [ ] Create tenant modal
- [ ] Tenant detail modal
- [ ] Members management

**Документация:** [Other Pages](pages/other-pages.md#1-tenants-page)

### 2.4 Profile & Security

**Файлы:** `profile.html`, `profile-devices.html`, `profile.js`, `profile-devices.js`

**Задачи:**
- [ ] Profile edit form
- [ ] Password change form
- [ ] Account info
- [ ] Devices list
- [ ] Revoke devices

**Документация:** [Other Pages](pages/other-pages.md#2-profile-page)

---

## Phase 3: Административный функционал 🟢

### 3.1 Admin Core (admin.html)

**Файлы:** `admin.html`, `admin.js`

**Задачи:**
- [ ] Admin layout с TabGroup
- [ ] Admin route protection (isAdmin check)
- [ ] Tab persistence (localStorage)

**Документация:** [Admin Pages](pages/admin.md)

### 3.2 Admin Dashboard Tab

**Задачи:**
- [ ] Stats cards (users, keys, models, requests)
- [ ] System Health table

### 3.3 Users & RBAC Tab

**Задачи:**
- [ ] SubTabs component
- [ ] Users table с HTMX → Svelte polling
- [ ] Create/Edit/Delete User modals
- [ ] Reset Password modal
- [ ] Enable/Disable user actions
- [ ] Auth provider badges (local/OIDC/LDAP)
- [ ] RBAC Roles badges
- [ ] Tenants badges
- [ ] Invitations preview (link to full page)
- [ ] Roles management (nested sub-tab)
- [ ] Permissions directory (nested sub-tab)

### 3.4 Admin API Keys Tab

**Задачи:**
- [ ] Global API keys table
- [ ] Create API Key modal (admin)
- [ ] Delete key action

### 3.5 Models Tab (yzma + HuggingFace)

**Задачи:**
- [ ] yzma Statistics (polling 5s)
- [ ] Loaded Models section (polling 3s)
- [ ] Available GGUF Models list
- [ ] Load/Unload model actions
- [ ] HuggingFace Browser search form
- [ ] Quick filters (Llama, Mistral, Phi, etc.)
- [ ] Search results grid
- [ ] Download modal
- [ ] Repeat last search

### 3.6 Downloads Tab

**Задачи:**
- [ ] Active downloads list (polling 2s)
- [ ] Progress indicators
- [ ] Cancel download modal

### 3.7 Files Tab (Admin)

**Файлы:** `admin-files.js`

**Задачи:**
- [ ] Stats cards
- [ ] Filters (search, type, status)
- [ ] Files table with owner info
- [ ] Pagination
- [ ] Delete file action

### 3.8 MCP Servers Tab

**Задачи:**
- [ ] MCP servers table
- [ ] Category filter
- [ ] Create/Edit MCP Server modal
- [ ] Delete server action

### 3.9 Settings Tab

**Файлы:** `admin-settings.js`

**Задачи:**
- [ ] Settings by category accordion
- [ ] Category icons and labels
- [ ] Settings table per category
- [ ] Inline editing
- [ ] Save confirmation modal
- [ ] Storage badge (DB/YAML)
- [ ] Restart required indicator

### 3.10 Backups Tab

**Задачи:**
- [ ] Backups list
- [ ] Create backup
- [ ] Download backup
- [ ] Restore backup (with warning)
- [ ] Delete backup

### 3.11 System & Logs Tab

**Файлы:** `logs.js`, `performance.js`, `gpu-monitor.js`

**Задачи:**
- [ ] Performance sub-tab:
  - [ ] Quick stats (CPU, Memory, Goroutines, Health)
  - [ ] GPU Metrics section
  - [ ] MoniGo link
- [ ] Audit sub-tab:
  - [ ] Quick stats (24h)
  - [ ] Recent events preview
  - [ ] Link to full audit page
- [ ] Logs sub-tab:
  - [ ] Log file selector
  - [ ] Level filters
  - [ ] Logs viewer
  - [ ] Real-time SSE streaming
  - [ ] Download log

### 3.12 Отдельные Admin страницы

**admin-audit.html:**
- [ ] Full audit table with filters
- [ ] Pagination
- [ ] CSV export

**admin-invitations.html:**
- [ ] Full invitations management
- [ ] Create invitation
- [ ] Copy link
- [ ] Revoke

**admin-rbac.html:**
- [ ] Full RBAC management
- [ ] Roles CRUD
- [ ] User role assignments

**admin-registry.html:**
- [ ] Providers management
- [ ] Models registry
- [ ] Health checks

**admin-rag.html:**
- [ ] All RAG sources
- [ ] Admin actions (disable, delete, sync)

---

## Phase 4: Дополнительный функционал 🔵

### 4.1 Usage Statistics

**Файлы:** `usage.html`, `usage.js`

**Задачи:**
- [ ] Time period selector
- [ ] Usage tables
- [ ] Stats summary

**Документация:** [Other Pages](pages/other-pages.md#4-usage-page)

### 4.2 Files Management

**Файлы:** `files.html`, `files.js`

**Задачи:**
- [ ] Drag & drop upload
- [ ] Files grid
- [ ] Content preview

**Документация:** [Other Pages](pages/other-pages.md#5-files-page)

### 4.3 RAG Sources (User)

**Файлы:** `rag-sources.html`, `rag-sources.js`

**Задачи:**
- [ ] Sources grid
- [ ] Create source forms
- [ ] Sync/Test actions

**Документация:** [Other Pages](pages/other-pages.md#6-rag-sources-page-user)

### 4.4 MCP Catalog

**Файлы:** `mcp-catalog.html`, `mcp.html`, `mcp.js`

**Задачи:**
- [ ] Search/filter
- [ ] Servers grid
- [ ] Detail modal

**Документация:** [Other Pages](pages/other-pages.md#7-mcp-catalog-page)

### 4.5 System Monitor

**Файлы:** `monitor.html`, `monitor.js`

**Задачи:**
- [ ] GPU metrics polling
- [ ] System resources
- [ ] Request rate

**Документация:** [Other Pages](pages/other-pages.md#8-monitor-page)

### 4.6 About Page

**Файлы:** `about.html`, `about.js`

**Задачи:**
- [ ] Version info
- [ ] Changelog accordion

**Документация:** [Other Pages](pages/other-pages.md#9-about-page)

### 4.7 Hugging Face Integration

**Файлы:** `huggingface.html`, `downloads.html`

**Задачи:**
- [ ] Model browser
- [ ] Downloads list
- [ ] Download/Cancel

**Документация:** [Other Pages](pages/other-pages.md#10-downloads-page-hugging-face)

### 4.8 Yzma (Local Inference)

**Файлы:** `yzma.html`

**Задачи:**
- [ ] Stats polling
- [ ] Models management

**Документация:** [Other Pages](pages/other-pages.md#12-yzma-page-local-inference)

---

## Phase 5: Финализация

### 5.1 Testing

- [ ] Unit tests для stores
- [ ] Component tests
- [ ] E2E tests для critical paths
- [ ] Mobile responsiveness testing

### 5.2 Optimization

- [ ] Bundle size analysis
- [ ] Code splitting review
- [ ] Lazy loading optimization
- [ ] Performance profiling

### 5.3 Cleanup

- [ ] Удалить старые HTML/JS файлы
- [ ] Обновить документацию
- [ ] Обновить README

### 5.4 Release

- [ ] Обновить VERSION
- [ ] Обновить CHANGELOG
- [ ] Создать migration SQL

---

## Зависимости между задачами

```
Phase 0: Infrastructure
    │
    ├─► Phase 1.1: Auth (required for all protected pages)
    │       │
    │       └─► Phase 1.2: Chat (depends on auth + models)
    │               │
    │               └─► Phase 1.3: Core Components
    │
    └─► Phase 2: Main Features (parallel, depends on Phase 1)
            │
            ├─► 2.1 Dashboard
            ├─► 2.2 API Keys
            ├─► 2.3 Tenants
            └─► 2.4 Profile
                    │
                    └─► Phase 3: Admin (depends on core components)
                            │
                            └─► Phase 4: Additional (can be parallel)
                                    │
                                    └─► Phase 5: Finalization
```

---

## Критерии готовности (Definition of Done)

### Для каждой страницы

- [ ] Все функции из оригинальной страницы реализованы
- [ ] TypeScript типы для всех данных
- [ ] Обработка ошибок с уведомлениями
- [ ] Loading states
- [ ] Empty states
- [ ] Responsive design (mobile + desktop)
- [ ] Keyboard navigation где применимо

### Для всего проекта

- [ ] Все страницы мигрированы
- [ ] API client полностью типизирован
- [ ] Stores с Svelte 5 runes
- [ ] shadcn-svelte компоненты
- [ ] Темная тема по умолчанию
- [ ] Production build работает через embed.FS
- [ ] Старые файлы удалены

---

## Документация

### Архитектура
- [Current Architecture](01-current-architecture.md)
- [Feature Inventory](02-feature-inventory.md)
- [Technology Stack](03-technology-stack.md)

### Техническая реализация
- [Component Mapping](05-component-mapping.md)
- [API Layer](06-api-layer.md)
- [State Management](07-state-management.md)
- [Embed Integration](08-embed-integration.md)

### Страницы
- [Auth Pages](pages/auth.md)
- [Chat Page](pages/chat.md)
- [Dashboard Page](pages/dashboard.md)
- [API Keys Page](pages/api-keys.md)
- [Admin Pages](pages/admin.md)
- [Other Pages](pages/other-pages.md)
