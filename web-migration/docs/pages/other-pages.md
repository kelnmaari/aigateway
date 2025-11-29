# Миграция остальных страниц

## Приоритет: 🟢 Средний - 🔵 Низкий (Phase 3-4)

---

## 1. Tenants Page

**Текущие файлы:** `web/tenants.html`, `web/js/tenants.js`

### Функционал

- Create organization modal
- Organizations grid
- Tenant detail modal:
  - Info tab
  - Members tab (list, add, remove, change role)
  - Edit/Delete actions
- Search/filter tenants

### Svelte компоненты

```
routes/(app)/tenants/
├── +page.svelte
└── +page.ts

lib/components/tenants/
├── TenantCard.svelte
├── CreateTenantModal.svelte
├── TenantDetailModal.svelte
├── MembersList.svelte
└── AddMemberForm.svelte
```

### API Endpoints

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/users/me/tenants` | GET | User's tenants |
| `/api/tenants` | POST | Create tenant |
| `/api/tenants/:id` | GET/PUT/DELETE | CRUD |
| `/api/tenants/:id/members` | GET | List members |
| `/api/tenants/:id/members` | POST | Add member |
| `/api/tenants/:id/members/:userId` | PUT/DELETE | Update/Remove |
| `/api/tenants/:id/search-users` | GET | Search users for adding |

---

## 2. Profile Page

**Текущие файлы:** `web/profile.html`, `web/js/profile.js`

### Функционал

- Update profile (email, display name)
- Change password (current + new + confirm)
- Account info (status, member since, last login)
- Danger zone: delete account

### Svelte компоненты

```
routes/(app)/profile/
├── +page.svelte
├── +page.ts
└── devices/
    └── +page.svelte

lib/components/profile/
├── ProfileForm.svelte
├── PasswordForm.svelte
├── AccountInfo.svelte
└── DangerZone.svelte
```

---

## 3. Profile Devices Page

**Текущие файлы:** `web/profile-devices.html`, `web/js/profile-devices.js`

### Функционал

- Filters (status, sort, order)
- Stats (total, active, current device)
- Devices list with details (OS, browser, IP, last seen)
- Actions: view, revoke single, revoke selected
- Bulk actions bar

### API Endpoints

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/users/me/devices` | GET | List devices |
| `/api/users/me/devices/:id` | DELETE | Revoke device |
| `/api/users/me/devices/revoke-all` | POST | Revoke all except current |

---

## 4. Usage Page

**Текущие файлы:** `web/usage.html`, `web/js/usage.js`

### Функционал

- Time period selector (7d, 30d, 90d, custom)
- Scope selector (personal/organization)
- Summary stats (requests, avg response, tokens, error rate)
- Usage by model table
- Usage by API key table
- Recent requests table

### Svelte компоненты

```
routes/(app)/usage/
├── +page.svelte
└── +page.ts

lib/components/usage/
├── UsageStats.svelte
├── UsageByModel.svelte
├── UsageByKey.svelte
└── RecentRequests.svelte
```

---

## 5. Files Page

**Текущие файлы:** `web/files.html`, `web/js/files.js`

### Функционал

- Drag & drop upload area
- Filters (search, type, status)
- Files grid with details:
  - Name, size, date
  - Chunks count, tokens count
  - Extraction status
- Actions: view content, delete
- Text preview modal

### Svelte компоненты

```
routes/(app)/files/
├── +page.svelte
└── +page.ts

lib/components/files/
├── UploadZone.svelte
├── FileCard.svelte
├── FileFilters.svelte
└── TextPreviewModal.svelte
```

### API Endpoints

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/files` | GET | List files |
| `/api/files` | POST | Upload file |
| `/api/files/:id` | GET | Get file info |
| `/api/files/:id` | DELETE | Delete file |
| `/api/files/:id/content` | GET | Get extracted text |

---

## 6. RAG Sources Page (User)

**Текущие файлы:** `web/rag-sources.html`, `web/js/rag-sources.js`

### Функционал

- Stats (total, active, error, chunks)
- Filters (type, status)
- Sources grid with details
- Create source modal (different types):
  - API source
  - Database source
  - File source
  - Web source
- Sync/Delete actions
- Test connection

### Svelte компоненты

```
routes/(app)/rag-sources/
├── +page.svelte
└── +page.ts

lib/components/rag/
├── RAGSourceCard.svelte
├── CreateSourceModal.svelte
├── SourceConfigForm/
│   ├── APIConfig.svelte
│   ├── DatabaseConfig.svelte
│   ├── FileConfig.svelte
│   └── WebConfig.svelte
└── SyncStatusBadge.svelte
```

---

## 7. MCP Catalog Page

**Текущие файлы:** `web/mcp-catalog.html`, `web/mcp.html`, `web/js/mcp.js`

### Функционал

- Search MCP servers
- Category filter
- Sort options
- Servers grid
- Server detail modal

### Svelte компоненты

```
routes/(app)/mcp/
├── +page.svelte
└── +page.ts

lib/components/mcp/
├── MCPServerCard.svelte
├── MCPFilters.svelte
└── MCPDetailModal.svelte
```

---

## 8. Monitor Page

**Текущие файлы:** `web/monitor.html`, `web/js/monitor.js`

### Функционал

- GPU metrics (HTMX polling, 5s)
- System resources (CPU, memory, requests, uptime)
- Request rate stats
- Real-time updates

### Svelte компоненты

```
routes/(app)/monitor/
├── +page.svelte
└── +page.ts

lib/components/monitor/
├── GPUMetrics.svelte
├── SystemResources.svelte
├── RequestRate.svelte
└── ProgressBar.svelte
```

### Особенности

- Polling через `setInterval` или SSE
- Progress bars с animation
- Real-time updates через `$effect`

---

## 9. About Page

**Текущие файлы:** `web/about.html`, `web/js/about.js`

### Функционал

- System version info
- Git commit, build date, Go version
- Changelog accordion

### Svelte компоненты

```
routes/(app)/about/
├── +page.svelte
└── +page.ts

lib/components/about/
├── VersionInfo.svelte
└── ChangelogAccordion.svelte
```

---

## 10. Downloads Page (Hugging Face)

**Текущие файлы:** `web/downloads.html`

### Функционал

- Active downloads list (HTMX, 2s polling)
- Progress indicators
- Cancel download modal

### Svelte компоненты

```
routes/(app)/downloads/
├── +page.svelte
└── +page.ts

lib/components/downloads/
├── DownloadItem.svelte
├── DownloadProgress.svelte
└── CancelModal.svelte
```

---

## 11. Hugging Face Browser

**Текущие файлы:** `web/huggingface.html`

### Функционал

- Search GGUF models
- Filters (author, sort, limit)
- Quick category filters
- Model results grid (HTMX)
- Model detail modal
- Download modal

### Svelte компоненты

```
routes/(app)/huggingface/
├── +page.svelte
└── +page.ts

lib/components/huggingface/
├── SearchBar.svelte
├── QuickFilters.svelte
├── ModelCard.svelte
├── ModelDetailModal.svelte
└── DownloadModal.svelte
```

---

## 12. Yzma Page (Local Inference)

**Текущие файлы:** `web/yzma.html`

### Функционал

- Yzma stats (HTMX polling)
- Available GGUF models list
- Loaded models list
- Load/Unload model actions
- Delete model with confirmation

### Svelte компоненты

```
routes/(app)/yzma/
├── +page.svelte
└── +page.ts

lib/components/yzma/
├── YzmaStats.svelte
├── GGUFModelList.svelte
├── LoadedModelList.svelte
└── DeleteModelModal.svelte
```

---

## Общие паттерны для всех страниц

### Page Loading Pattern

```svelte
<script>
  import { onMount } from 'svelte';
  
  let data = $state([]);
  let loading = $state(true);
  let error = $state(null);
  
  onMount(async () => {
    try {
      data = await api.loadData();
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  });
</script>

{#if loading}
  <LoadingState />
{:else if error}
  <ErrorState {error} onRetry={loadData} />
{:else if data.length === 0}
  <EmptyState />
{:else}
  <DataList {data} />
{/if}
```

### Modal Pattern

```svelte
<script>
  let open = $state(false);
  let loading = $state(false);
  
  async function handleSubmit() {
    loading = true;
    try {
      await api.action();
      open = false;
      toast.success('Success');
    } catch (e) {
      toast.error(e.message);
    } finally {
      loading = false;
    }
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Trigger asChild let:builder>
    <Button builders={[builder]}>Open</Button>
  </Dialog.Trigger>
  <Dialog.Content>
    <!-- Form -->
  </Dialog.Content>
</Dialog.Root>
```

### Real-time Polling Pattern

```svelte
<script>
  import { onMount, onDestroy } from 'svelte';
  
  let data = $state(null);
  let interval: ReturnType<typeof setInterval>;
  
  async function fetchData() {
    data = await api.getData();
  }
  
  onMount(() => {
    fetchData();
    interval = setInterval(fetchData, 5000);
  });
  
  onDestroy(() => {
    clearInterval(interval);
  });
</script>
```

### Filter Pattern

```svelte
<script>
  let filters = $state({
    search: '',
    type: '',
    status: '',
  });
  
  let page = $state(1);
  
  // Reset page when filters change
  $effect(() => {
    const _ = filters;
    page = 1;
  });
  
  // Load data when filters or page change
  $effect(() => {
    loadData(filters, page);
  });
</script>
```

