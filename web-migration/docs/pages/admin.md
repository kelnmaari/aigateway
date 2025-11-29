# Миграция Admin Pages

## Приоритет: 🟢 Средний (Phase 3)

Административные страницы для управления системой. Это самая большая и сложная часть интерфейса.

## Текущие файлы

### Основные
- `web/admin.html` - Главная админ-панель (~2450 строк)
- `web/js/admin.js` - Основная логика админки
- `web/js/admin-files.js` - Управление файлами
- `web/js/admin-settings.js` - Настройки системы
- `web/js/logs.js` - Просмотр логов
- `web/js/performance.js` - Мониторинг производительности
- `web/js/gpu-monitor.js` - Мониторинг GPU

### Отдельные страницы
- `web/admin-audit.html` - Полный Audit Log
- `web/admin-invitations.html` - Управление приглашениями
- `web/admin-rag.html` - RAG управление (admin)
- `web/admin-rbac.html` - RBAC управление (полное)
- `web/admin-registry.html` - Model Registry

---

## Полная структура табов Admin Panel

```
Admin Panel
├── Dashboard (tab)
│   ├── Stats Cards (users, keys, models, requests)
│   └── System Health Table
│
├── Users & RBAC (tab)
│   ├── Users (sub-tab) ← User Management
│   ├── Invitations (sub-tab) ← Quick preview + link to full page
│   └── RBAC (sub-tab)
│       ├── Roles (nested sub-tab)
│       └── Permissions (nested sub-tab)
│
├── API Keys (tab) ← Global API Keys Management
│
├── Models (tab)
│   ├── Local Models / yzma (sub-tab)
│   │   ├── yzma Statistics
│   │   ├── Currently Loaded Models
│   │   └── Available GGUF Models
│   └── Hugging Face Browser (sub-tab)
│       ├── Search Form
│       ├── Quick Filters
│       └── Search Results
│
├── Downloads (tab) ← Active HF Downloads (HTMX polling)
│
├── Files (tab) ← Admin Files Management
│   ├── Stats Cards
│   ├── Filters
│   └── Files Table + Pagination
│
├── RAG Sources (tab) ← Link to admin-rag.html
│
├── MCP (tab) ← MCP Servers Catalog
│
├── Settings (tab) ← System Settings (editable)
│   └── Category Accordions
│
├── Backups (tab) ← Database Backups
│
└── System & Logs (tab)
    ├── Performance (sub-tab)
    │   ├── Quick Stats (CPU, Memory, Goroutines, Health)
    │   ├── GPU Metrics Section
    │   └── MoniGo Link
    ├── Audit (sub-tab)
    │   ├── Quick Stats
    │   └── Recent Events Preview
    └── Logs (sub-tab)
        ├── File Selector
        ├── Level Filters
        └── Logs Viewer (with real-time SSE)
```

---

## 1. Admin Dashboard

### Функционал
- Stats cards с HTMX polling:
  - Total Users
  - API Keys
  - Models
  - API Requests (30d)
- System Health table

### Svelte реализация

```svelte
<script>
  import { onMount } from 'svelte';
  import { adminApi } from '$lib/api';
  import StatCard from '$lib/components/admin/StatCard.svelte';
  import { Users, Key, Cpu, Activity } from 'lucide-svelte';
  
  let stats = $state({
    users: 0,
    apiKeys: 0,
    models: 0,
    requests: 0,
  });
  let systemHealth = $state([]);
  let loading = $state(true);
  
  onMount(async () => {
    const [usersData, keysData, modelsData, healthData] = await Promise.all([
      adminApi.getUsersCount(),
      adminApi.getKeysCount(),
      adminApi.getModelsCount(),
      adminApi.getSystemHealth(),
    ]);
    
    stats = {
      users: usersData.total,
      apiKeys: keysData.total,
      models: modelsData.total,
      requests: healthData.requests_30d,
    };
    systemHealth = healthData.health;
    loading = false;
  });
</script>
```

---

## 2. Users & RBAC Tab

### 2.1 Users Management (sub-tab)

**Функционал:**
- Users table с колонками:
  - Username
  - Email
  - Auth Provider (local/OIDC/LDAP)
  - RBAC Roles (badges)
  - Tenants (badges)
  - Admin flag
  - Status
  - Created
  - Actions
- Create User modal
- Edit User modal
- Reset Password modal
- Enable/Disable user
- Delete user

### 2.2 Invitations (sub-tab)

**Функционал:**
- Quick stats (active, pending, used, expired)
- Recent invitations preview table
- Link to full `/admin-invitations.html`

### 2.3 RBAC (sub-tab)

#### Roles (nested sub-tab)
- Roles table:
  - Role name
  - Scope (system/tenant)
  - Permissions count
  - Users count
  - Created
  - Actions (edit, delete)
- Create Role modal

#### Permissions (nested sub-tab)
- Permissions directory table:
  - Permission name
  - Resource
  - Action
  - Created

---

## 3. API Keys Tab (Global)

### Функционал
- Admin can see ALL API keys in system
- Table columns:
  - Name
  - Owner (username)
  - Organization (tenant)
  - Key Preview
  - Models count
  - Rate Limit
  - Last Used
  - Actions
- Create API Key modal (admin-level)
- Delete API Key

---

## 4. Models Tab

### 4.1 Local Models / yzma (sub-tab)

**Функционал:**
- yzma Statistics cards (HTMX polling 5s)
- Currently Loaded Models section (HTMX polling 3s)
  - Model name
  - Size
  - Context
  - Unload button
- Available GGUF Models section
  - Model list with Load button
  - Refresh button

### 4.2 Hugging Face Browser (sub-tab)

**Функционал:**
- Search form:
  - Search query
  - Author filter
  - Sort by (downloads, trending, updated)
  - Results limit
  - Search button
  - Repeat last search button
- Quick filters (Llama, Mistral, Phi, Gemma, Vision)
- Search results grid
- Download modal

---

## 5. Downloads Tab

### Функционал
- Active downloads list (HTMX polling 2s)
- Progress indicators
- Cancel download

```svelte
<script>
  import { onMount, onDestroy } from 'svelte';
  import { huggingfaceApi } from '$lib/api';
  
  let downloads = $state([]);
  let interval: ReturnType<typeof setInterval>;
  
  async function loadDownloads() {
    downloads = await huggingfaceApi.getActiveDownloads();
  }
  
  onMount(() => {
    loadDownloads();
    interval = setInterval(loadDownloads, 2000);
  });
  
  onDestroy(() => {
    clearInterval(interval);
  });
</script>

{#each downloads as download}
  <DownloadItem {download} onCancel={handleCancel} />
{/each}
```

---

## 6. Files Tab (Admin)

### Функционал
- Stats cards:
  - Total Files
  - Total Size
  - Extracted
  - Pending
- Filters:
  - Search by filename/owner
  - Type filter (TXT, PDF, CSV, DOCX)
  - Status filter (completed, pending, failed)
- Files table:
  - Filename
  - Owner (email/username)
  - Type
  - Size
  - Status
  - Words count
  - Created
  - Actions (delete)
- Pagination

---

## 7. RAG Sources Tab

### Функционал
- Info card with RAG overview
- Link to `/admin-rag.html` for full management

---

## 8. MCP Servers Tab

### Функционал
- MCP servers table:
  - Name
  - Category
  - Description
  - Tags
  - Status (Active/Inactive)
  - Actions (edit, delete)
- Category filter dropdown
- Link to public catalog
- Create/Edit MCP Server modal:
  - Name
  - Category (development, productivity, database, cloud, ai, other)
  - Description
  - Installation Guide
  - Website URL
  - GitHub URL
  - Tags
  - Active checkbox

---

## 9. Settings Tab

### Функционал
- System settings по категориям:
  - Server Configuration
  - Inference Engine
  - Authentication & Security
  - Database Settings
  - Logging Configuration
  - Metrics & Monitoring
  - TLS/SSL Configuration
  - RAG System
  - Hugging Face Integration
- Category accordions (expandable)
- Settings table per category:
  - Setting key
  - Value (editable inline)
  - Type
  - Description
  - Storage badge (DB/YAML)
  - Live reload / Restart required badge
- Inline editing with confirmation modal

```svelte
<script>
  import { settingsApi } from '$lib/api';
  import * as Accordion from '$lib/components/ui/accordion';
  
  let settings = $state<Record<string, Setting[]>>({});
  let editingId = $state<string | null>(null);
  
  const categoryMeta = {
    server: { icon: '🖥️', label: 'Server Configuration' },
    inference: { icon: '🤖', label: 'Inference Engine' },
    auth: { icon: '🔐', label: 'Authentication & Security' },
    database: { icon: '💾', label: 'Database Settings' },
    logging: { icon: '📝', label: 'Logging Configuration' },
    metrics: { icon: '📊', label: 'Metrics & Monitoring' },
    tls: { icon: '🔒', label: 'TLS/SSL Configuration' },
    rag: { icon: '🧠', label: 'RAG System' },
    huggingface: { icon: '🤗', label: 'Hugging Face Integration' },
  };
</script>

<Accordion.Root type="multiple">
  {#each Object.entries(settings) as [category, items]}
    <Accordion.Item value={category}>
      <Accordion.Trigger>
        <span>{categoryMeta[category]?.icon}</span>
        <span>{categoryMeta[category]?.label || category}</span>
        <span class="badge">{items.length} settings</span>
      </Accordion.Trigger>
      <Accordion.Content>
        <SettingsTable {items} onEdit={handleEdit} />
      </Accordion.Content>
    </Accordion.Item>
  {/each}
</Accordion.Root>
```

---

## 10. Backups Tab

### Функционал
- Info card about backup management
- Create Backup button
- Backups table:
  - Filename
  - Created
  - Size
  - Actions (Download, Restore, Delete)
- Restore confirmation with warning
- Delete confirmation

---

## 11. System & Logs Tab

### 11.1 Performance (sub-tab)

**Функционал:**
- Quick stats cards:
  - CPU Usage
  - Memory Usage
  - Goroutines
  - System Health
- GPU Metrics section (NVIDIA):
  - GPU cards with utilization, memory, temperature
  - Updates every 5s
- Link to MoniGo advanced dashboard

### 11.2 Audit (sub-tab)

**Функционал:**
- Quick stats (24h):
  - Total events
  - Critical
  - Warnings
  - Failed logins
- Recent events table (last 20)
- Link to full `/admin-audit.html`

### 11.3 Logs (sub-tab)

**Функционал:**
- Log file selector
- Level filters (All, DEBUG, INFO, WARN, ERROR)
- Download button
- Refresh button
- Real-time toggle (SSE streaming)
- Logs viewer:
  - Timestamp
  - Level badge
  - Message with context highlighting
  - Auto-scroll

```svelte
<script>
  import { logsApi } from '$lib/api';
  
  let logFiles = $state([]);
  let currentFile = $state('');
  let entries = $state([]);
  let levelFilter = $state('');
  let isRealtime = $state(false);
  let eventSource: EventSource | null = null;
  
  function startRealtime() {
    const token = localStorage.getItem('access_token');
    eventSource = new EventSource(
      `/api/admin/logs/stream?token=${token}&file=${currentFile}`
    );
    
    eventSource.addEventListener('log', (e) => {
      const entry = JSON.parse(e.data);
      entries = [...entries, entry].slice(-1000); // Keep last 1000
    });
    
    isRealtime = true;
  }
  
  function stopRealtime() {
    eventSource?.close();
    eventSource = null;
    isRealtime = false;
  }
</script>
```

---

## Отдельные Admin страницы

### admin-audit.html
- Full audit log with:
  - Filters (event type, severity, resource, status, date range, actor)
  - Pagination
  - CSV Export
  - Detailed view

### admin-invitations.html
- Full invitations management:
  - Stats cards
  - Filters
  - Create invitation modal
  - Invitations table with all actions
  - Copy link functionality
  - Revoke invitation

### admin-rag.html
- Full RAG sources management:
  - Stats cards
  - Filters (type, status, user)
  - Sources table
  - Admin actions (disable, delete, force sync)

### admin-rbac.html
- Full RBAC management (Bootstrap-based):
  - Roles tab
  - User Roles tab
  - Permissions Reference tab
  - Full CRUD for roles
  - Role-user assignments

### admin-registry.html
- Model Registry:
  - Providers management
  - Registered models
  - Health checks

---

## Modals (admin.html)

1. **Create User Modal**
   - Username, Email, Password, Full Name, Is Admin

2. **Create API Key Modal**
   - Name, Description, Models, Permissions, Rate Limit

3. **Edit User Modal**
   - Username (readonly), Email, Full Name, Is Admin, Status

4. **Reset Password Modal**
   - User display, New Password, Confirm Password

5. **MCP Server Modal**
   - Name, Category, Description, Installation Guide
   - Website URL, GitHub URL, Tags, Active

---

## Svelte Route Structure

```
routes/(admin)/
├── +layout.svelte           # Admin layout with permission check
├── +layout.ts               # Auth guard for admin
├── admin/
│   ├── +page.svelte         # Main admin panel with tabs
│   ├── audit/
│   │   └── +page.svelte     # Full audit log
│   ├── invitations/
│   │   └── +page.svelte     # Full invitations
│   ├── rag/
│   │   └── +page.svelte     # Full RAG management
│   ├── rbac/
│   │   └── +page.svelte     # Full RBAC management
│   └── registry/
│       └── +page.svelte     # Model registry

lib/components/admin/
├── AdminTabs.svelte
├── AdminSubTabs.svelte
├── dashboard/
│   ├── StatsGrid.svelte
│   └── SystemHealth.svelte
├── users/
│   ├── UsersTable.svelte
│   ├── CreateUserModal.svelte
│   ├── EditUserModal.svelte
│   └── ResetPasswordModal.svelte
├── rbac/
│   ├── RolesTable.svelte
│   ├── PermissionsTable.svelte
│   └── RoleModal.svelte
├── apikeys/
│   ├── AdminKeysTable.svelte
│   └── CreateKeyModal.svelte
├── models/
│   ├── YzmaStats.svelte
│   ├── LoadedModels.svelte
│   ├── AvailableModels.svelte
│   └── HuggingFaceBrowser.svelte
├── downloads/
│   ├── DownloadsList.svelte
│   └── DownloadItem.svelte
├── files/
│   ├── AdminFilesStats.svelte
│   ├── AdminFilesFilters.svelte
│   └── AdminFilesTable.svelte
├── mcp/
│   ├── MCPTable.svelte
│   └── MCPServerModal.svelte
├── settings/
│   ├── SettingsAccordion.svelte
│   ├── SettingsTable.svelte
│   └── EditSettingDialog.svelte
├── backups/
│   ├── BackupsList.svelte
│   └── RestoreDialog.svelte
└── system/
    ├── PerformanceStats.svelte
    ├── GPUMetrics.svelte
    ├── AuditPreview.svelte
    └── LogsViewer.svelte
```

---

## API Endpoints (Admin)

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/admin/users` | GET/POST | Users list / Create user |
| `/api/admin/users/:id` | GET/PUT/DELETE | User CRUD |
| `/api/admin/users/:id/disable` | PATCH | Disable user |
| `/api/admin/users/:id/enable` | PATCH | Enable user |
| `/api/admin/users/:id/reset-password` | POST | Reset password |
| `/api/admin/keys` | GET/POST | API keys list / Create |
| `/api/admin/keys/:id` | DELETE | Delete key |
| `/api/admin/rbac/roles` | GET/POST | Roles |
| `/api/admin/rbac/permissions` | GET | Permissions |
| `/api/admin/mcp/servers` | GET/POST | MCP servers |
| `/api/admin/mcp/servers/:id` | PUT/DELETE | MCP CRUD |
| `/api/admin/settings` | GET | Get all settings |
| `/api/admin/settings/:id` | PUT | Update setting |
| `/api/admin/backups` | GET | List backups |
| `/api/admin/backup` | POST | Create backup |
| `/api/admin/backup/:file` | GET/DELETE | Download/Delete |
| `/api/admin/restore/:file` | POST | Restore backup |
| `/api/admin/files` | GET | List all files |
| `/api/admin/files/stats` | GET | File stats |
| `/api/admin/files/:id` | DELETE | Delete file |
| `/api/admin/logs` | GET | List log files |
| `/api/admin/logs/:file` | GET | Get log entries |
| `/api/admin/logs/stream` | SSE | Real-time logs |
| `/api/admin/audit` | GET | Audit events |
| `/api/admin/audit/stats` | GET | Audit stats |
| `/api/admin/stats` | GET | System stats |
| `/api/ui/dashboard/stats/*` | GET | HTMX stats endpoints |
| `/api/ui/yzma/*` | GET/POST | yzma endpoints |
| `/api/ui/huggingface/*` | GET/POST | HuggingFace endpoints |

---

## Тестирование

### Dashboard
- [ ] Stats загружаются
- [ ] System health отображается

### Users & RBAC
- [ ] Users table с фильтрацией
- [ ] Create/Edit/Delete user
- [ ] Reset password
- [ ] Enable/Disable user
- [ ] Auth provider badges
- [ ] RBAC roles badges
- [ ] Tenants badges
- [ ] Roles CRUD
- [ ] Permissions directory

### API Keys
- [ ] Admin keys table
- [ ] Create key modal
- [ ] Delete key

### Models
- [ ] yzma stats polling
- [ ] Loaded models display
- [ ] Load/Unload model
- [ ] HuggingFace search
- [ ] Quick filters
- [ ] Download model

### Downloads
- [ ] Active downloads polling
- [ ] Progress display
- [ ] Cancel download

### Files
- [ ] Stats display
- [ ] Filters work
- [ ] Pagination
- [ ] Delete file

### MCP
- [ ] MCP table
- [ ] Category filter
- [ ] Create/Edit server
- [ ] Delete server

### Settings
- [ ] Categories load
- [ ] Accordion expand/collapse
- [ ] Inline editing
- [ ] Save confirmation

### Backups
- [ ] List backups
- [ ] Create backup
- [ ] Download backup
- [ ] Restore with warning
- [ ] Delete backup

### System & Logs
- [ ] Performance stats
- [ ] GPU metrics (if NVIDIA)
- [ ] Audit preview
- [ ] Logs file selector
- [ ] Level filtering
- [ ] Real-time streaming
- [ ] Download log
