# Миграция API Keys Page

## Приоритет: 🟡 Высокий (Phase 2)

Управление API ключами - критический функционал для интеграций.

## Текущие файлы

- `web/api-keys.html` - HTML разметка
- `web/js/apikeys.js` - Логика управления ключами

## Функционал для миграции

### 1. Tabs (Personal / Organization)

**Текущий функционал:**
- Personal Keys tab
- Organization Keys tab с tenant selector
- Persistent tab selection

```svelte
<script>
  import * as Tabs from '$lib/components/ui/tabs';
  import * as Select from '$lib/components/ui/select';
  import { tenantsApi } from '$lib/api';
  
  let activeTab = $state('personal');
  let selectedTenantId = $state<string | null>(null);
  let userTenants = $state([]);
  
  $effect(() => {
    tenantsApi.listUserTenants().then(data => {
      userTenants = data.tenants || [];
      if (userTenants.length > 0 && !selectedTenantId) {
        selectedTenantId = userTenants[0].id;
      }
    });
  });
</script>

<Tabs.Root bind:value={activeTab}>
  <Tabs.List>
    <Tabs.Trigger value="personal">Personal Keys</Tabs.Trigger>
    <Tabs.Trigger value="tenant">Organization Keys</Tabs.Trigger>
  </Tabs.List>
  
  <Tabs.Content value="personal">
    <PersonalKeysTable />
  </Tabs.Content>
  
  <Tabs.Content value="tenant">
    <div class="tenant-selector">
      <Select.Root bind:value={selectedTenantId}>
        <Select.Trigger>
          <Select.Value placeholder="Select organization" />
        </Select.Trigger>
        <Select.Content>
          {#each userTenants as tenant}
            <Select.Item value={tenant.id}>{tenant.name}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>
    
    {#if selectedTenantId}
      <TenantKeysTable tenantId={selectedTenantId} />
    {/if}
  </Tabs.Content>
</Tabs.Root>
```

### 2. API Keys Table

**Текущий функционал:**
- Name, Description, Key Prefix
- Rate Limits (RPM, RPH)
- Models access (All or specific)
- Status badge
- Created/Last Used dates
- Actions (view, delete)

```svelte
<script>
  import * as Table from '$lib/components/ui/table';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import { Eye, Trash2, MoreVertical } from 'lucide-svelte';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
  import { formatDate } from '$lib/utils/date';
  
  let { keys = [], onView, onDelete, loading = false } = $props();
  
  function getStatusVariant(status: string) {
    return status === 'active' ? 'success' : 'secondary';
  }
</script>

<Table.Root>
  <Table.Header>
    <Table.Row>
      <Table.Head>Name</Table.Head>
      <Table.Head>Key</Table.Head>
      <Table.Head>Rate Limits</Table.Head>
      <Table.Head>Models</Table.Head>
      <Table.Head>Status</Table.Head>
      <Table.Head>Last Used</Table.Head>
      <Table.Head class="w-[50px]"></Table.Head>
    </Table.Row>
  </Table.Header>
  <Table.Body>
    {#if loading}
      {#each Array(3) as _}
        <Table.Row>
          <Table.Cell colspan={7}>
            <div class="h-12 animate-pulse bg-muted rounded" />
          </Table.Cell>
        </Table.Row>
      {/each}
    {:else if keys.length === 0}
      <Table.Row>
        <Table.Cell colspan={7} class="text-center py-8 text-muted-foreground">
          No API keys found
        </Table.Cell>
      </Table.Row>
    {:else}
      {#each keys as key}
        <Table.Row>
          <Table.Cell>
            <div>
              <p class="font-medium">{key.name}</p>
              {#if key.description}
                <p class="text-sm text-muted-foreground">{key.description}</p>
              {/if}
            </div>
          </Table.Cell>
          <Table.Cell>
            <code class="text-sm">{key.key_prefix}...</code>
          </Table.Cell>
          <Table.Cell>
            <div class="text-sm">
              <p>{key.rate_limit_rpm || '∞'} RPM</p>
              <p>{key.rate_limit_rph || '∞'} RPH</p>
            </div>
          </Table.Cell>
          <Table.Cell>
            {#if key.models === null || key.models?.length === 0}
              <Badge variant="outline">All Models</Badge>
            {:else}
              <Badge variant="secondary">{key.models.length} models</Badge>
            {/if}
          </Table.Cell>
          <Table.Cell>
            <Badge variant={getStatusVariant(key.status)}>
              {key.status}
            </Badge>
          </Table.Cell>
          <Table.Cell>
            {#if key.last_used_at}
              {formatDate(key.last_used_at)}
            {:else}
              <span class="text-muted-foreground">Never</span>
            {/if}
          </Table.Cell>
          <Table.Cell>
            <DropdownMenu.Root>
              <DropdownMenu.Trigger asChild let:builder>
                <Button variant="ghost" size="icon" builders={[builder]}>
                  <MoreVertical class="w-4 h-4" />
                </Button>
              </DropdownMenu.Trigger>
              <DropdownMenu.Content>
                <DropdownMenu.Item onclick={() => onView?.(key)}>
                  <Eye class="w-4 h-4 mr-2" /> View Details
                </DropdownMenu.Item>
                <DropdownMenu.Separator />
                <DropdownMenu.Item 
                  class="text-destructive"
                  onclick={() => onDelete?.(key)}
                >
                  <Trash2 class="w-4 h-4 mr-2" /> Delete
                </DropdownMenu.Item>
              </DropdownMenu.Content>
            </DropdownMenu.Root>
          </Table.Cell>
        </Table.Row>
      {/each}
    {/if}
  </Table.Body>
</Table.Root>
```

### 3. Create API Key Modal

**Текущий функционал:**
- Name input
- Description textarea
- Rate limit inputs (RPM, RPH)
- Models selection (all or specific)
- Show created key value ONCE

```svelte
<script>
  import * as Dialog from '$lib/components/ui/dialog';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Label } from '$lib/components/ui/label';
  import { Textarea } from '$lib/components/ui/textarea';
  import { Switch } from '$lib/components/ui/switch';
  import * as Checkbox from '$lib/components/ui/checkbox';
  import { modelsStore } from '$lib/stores/models.svelte';
  import { apiKeysApi } from '$lib/api';
  import { toast } from 'svelte-sonner';
  import { Copy, Check } from 'lucide-svelte';
  
  let { open = $bindable(false), tenantId = null, onCreated } = $props();
  
  let name = $state('');
  let description = $state('');
  let rateLimitRpm = $state<number | undefined>();
  let rateLimitRph = $state<number | undefined>();
  let allModels = $state(true);
  let selectedModels = $state<string[]>([]);
  
  let createdKey = $state<string | null>(null);
  let copied = $state(false);
  let loading = $state(false);
  
  async function handleCreate() {
    if (!name.trim()) {
      toast.error('Name is required');
      return;
    }
    
    try {
      loading = true;
      const data = {
        name: name.trim(),
        description: description.trim() || undefined,
        rate_limit_rpm: rateLimitRpm,
        rate_limit_rph: rateLimitRph,
        models: allModels ? undefined : selectedModels,
      };
      
      let response;
      if (tenantId) {
        response = await apiKeysApi.createForTenant(tenantId, data);
      } else {
        response = await apiKeysApi.create(data);
      }
      
      createdKey = response.key;
      toast.success('API key created');
      onCreated?.();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to create key');
    } finally {
      loading = false;
    }
  }
  
  function copyKey() {
    if (createdKey) {
      navigator.clipboard.writeText(createdKey);
      copied = true;
      setTimeout(() => copied = false, 2000);
    }
  }
  
  function handleClose() {
    open = false;
    // Reset form
    name = '';
    description = '';
    rateLimitRpm = undefined;
    rateLimitRph = undefined;
    allModels = true;
    selectedModels = [];
    createdKey = null;
  }
</script>

<Dialog.Root bind:open onOpenChange={(o) => !o && handleClose()}>
  <Dialog.Content class="max-w-lg">
    <Dialog.Header>
      <Dialog.Title>
        {createdKey ? 'API Key Created' : 'Create API Key'}
      </Dialog.Title>
      <Dialog.Description>
        {createdKey 
          ? 'Save this key now. You won\'t be able to see it again.'
          : 'Create a new API key for authentication.'
        }
      </Dialog.Description>
    </Dialog.Header>
    
    {#if createdKey}
      <!-- Show created key -->
      <div class="space-y-4">
        <div class="p-4 bg-muted rounded-lg">
          <code class="text-sm break-all">{createdKey}</code>
        </div>
        <Button class="w-full" onclick={copyKey}>
          {#if copied}
            <Check class="w-4 h-4 mr-2" /> Copied!
          {:else}
            <Copy class="w-4 h-4 mr-2" /> Copy to Clipboard
          {/if}
        </Button>
      </div>
    {:else}
      <!-- Create form -->
      <form onsubmit|preventDefault={handleCreate} class="space-y-4">
        <div class="space-y-2">
          <Label for="name">Name *</Label>
          <Input id="name" bind:value={name} placeholder="My API Key" required />
        </div>
        
        <div class="space-y-2">
          <Label for="description">Description</Label>
          <Textarea 
            id="description" 
            bind:value={description} 
            placeholder="What this key is used for"
            rows={2}
          />
        </div>
        
        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-2">
            <Label for="rpm">Rate Limit (RPM)</Label>
            <Input 
              id="rpm" 
              type="number" 
              bind:value={rateLimitRpm} 
              placeholder="Unlimited"
              min={1}
            />
          </div>
          <div class="space-y-2">
            <Label for="rph">Rate Limit (RPH)</Label>
            <Input 
              id="rph" 
              type="number" 
              bind:value={rateLimitRph} 
              placeholder="Unlimited"
              min={1}
            />
          </div>
        </div>
        
        <div class="space-y-2">
          <div class="flex items-center space-x-2">
            <Switch id="all-models" bind:checked={allModels} />
            <Label for="all-models">Access to all models</Label>
          </div>
        </div>
        
        {#if !allModels}
          <div class="space-y-2">
            <Label>Select Models</Label>
            <div class="max-h-40 overflow-y-auto border rounded p-2 space-y-2">
              {#each modelsStore.models as model}
                <div class="flex items-center space-x-2">
                  <Checkbox.Root
                    id={model.id}
                    checked={selectedModels.includes(model.id)}
                    onCheckedChange={(checked) => {
                      if (checked) {
                        selectedModels = [...selectedModels, model.id];
                      } else {
                        selectedModels = selectedModels.filter(m => m !== model.id);
                      }
                    }}
                  >
                    <Checkbox.Indicator />
                  </Checkbox.Root>
                  <Label for={model.id} class="text-sm">{model.name}</Label>
                </div>
              {/each}
            </div>
          </div>
        {/if}
        
        <Dialog.Footer>
          <Button variant="outline" type="button" onclick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={loading}>
            {loading ? 'Creating...' : 'Create Key'}
          </Button>
        </Dialog.Footer>
      </form>
    {/if}
  </Dialog.Content>
</Dialog.Root>
```

### 4. Delete Confirmation

```svelte
<script>
  import * as AlertDialog from '$lib/components/ui/alert-dialog';
  import { Button } from '$lib/components/ui/button';
  import { apiKeysApi } from '$lib/api';
  import { toast } from 'svelte-sonner';
  
  let { open = $bindable(false), apiKey, tenantId, onDeleted } = $props();
  let loading = $state(false);
  
  async function handleDelete() {
    if (!apiKey) return;
    
    try {
      loading = true;
      if (tenantId) {
        await apiKeysApi.deleteForTenant(tenantId, apiKey.id);
      } else {
        await apiKeysApi.delete(apiKey.id);
      }
      toast.success('API key deleted');
      onDeleted?.();
      open = false;
    } catch (e) {
      toast.error(e instanceof Error ? e.message : 'Failed to delete');
    } finally {
      loading = false;
    }
  }
</script>

<AlertDialog.Root bind:open>
  <AlertDialog.Content>
    <AlertDialog.Header>
      <AlertDialog.Title>Delete API Key</AlertDialog.Title>
      <AlertDialog.Description>
        Are you sure you want to delete "{apiKey?.name}"? 
        This action cannot be undone and any applications using this key will stop working.
      </AlertDialog.Description>
    </AlertDialog.Header>
    <AlertDialog.Footer>
      <AlertDialog.Cancel>Cancel</AlertDialog.Cancel>
      <AlertDialog.Action asChild let:builder>
        <Button 
          variant="destructive" 
          builders={[builder]}
          onclick={handleDelete}
          disabled={loading}
        >
          {loading ? 'Deleting...' : 'Delete'}
        </Button>
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
```

## Полная страница

```svelte
<!-- routes/(app)/api-keys/+page.svelte -->
<script>
  import { onMount } from 'svelte';
  import { apiKeysApi, tenantsApi } from '$lib/api';
  import { modelsStore } from '$lib/stores/models.svelte';
  import * as Tabs from '$lib/components/ui/tabs';
  import * as Select from '$lib/components/ui/select';
  import { Button } from '$lib/components/ui/button';
  import { Plus, Key } from 'lucide-svelte';
  import ApiKeysTable from '$lib/components/api-keys/ApiKeysTable.svelte';
  import CreateKeyModal from '$lib/components/api-keys/CreateKeyModal.svelte';
  import DeleteKeyDialog from '$lib/components/api-keys/DeleteKeyDialog.svelte';
  
  let activeTab = $state('personal');
  let selectedTenantId = $state<string | null>(null);
  let userTenants = $state([]);
  
  let personalKeys = $state([]);
  let tenantKeys = $state([]);
  let loading = $state(true);
  
  let createModalOpen = $state(false);
  let deleteDialogOpen = $state(false);
  let selectedKey = $state(null);
  
  onMount(async () => {
    await Promise.all([
      loadPersonalKeys(),
      loadTenants(),
      modelsStore.fetch(),
    ]);
  });
  
  async function loadPersonalKeys() {
    try {
      loading = true;
      const data = await apiKeysApi.list();
      personalKeys = data.api_keys || [];
    } finally {
      loading = false;
    }
  }
  
  async function loadTenants() {
    const data = await tenantsApi.listUserTenants();
    userTenants = data.tenants || [];
    if (userTenants.length > 0) {
      selectedTenantId = userTenants[0].id;
      await loadTenantKeys();
    }
  }
  
  async function loadTenantKeys() {
    if (!selectedTenantId) return;
    try {
      const data = await apiKeysApi.listForTenant(selectedTenantId);
      tenantKeys = data.api_keys || [];
    } catch {
      tenantKeys = [];
    }
  }
  
  $effect(() => {
    if (selectedTenantId) {
      loadTenantKeys();
    }
  });
  
  function handleKeyCreated() {
    if (activeTab === 'personal') {
      loadPersonalKeys();
    } else {
      loadTenantKeys();
    }
    createModalOpen = false;
  }
  
  function handleKeyDeleted() {
    if (activeTab === 'personal') {
      loadPersonalKeys();
    } else {
      loadTenantKeys();
    }
  }
</script>

<svelte:head>
  <title>API Keys - AIGateway</title>
</svelte:head>

<div class="api-keys-page">
  <header class="page-header">
    <div>
      <h1 class="text-2xl font-bold flex items-center gap-2">
        <Key class="w-6 h-6" /> API Keys
      </h1>
      <p class="text-muted-foreground">
        Manage your API keys for authentication
      </p>
    </div>
    <Button onclick={() => createModalOpen = true}>
      <Plus class="w-4 h-4 mr-2" /> Create Key
    </Button>
  </header>
  
  <Tabs.Root bind:value={activeTab}>
    <Tabs.List>
      <Tabs.Trigger value="personal">Personal Keys</Tabs.Trigger>
      <Tabs.Trigger value="tenant">Organization Keys</Tabs.Trigger>
    </Tabs.List>
    
    <Tabs.Content value="personal">
      <ApiKeysTable 
        keys={personalKeys} 
        {loading}
        onView={(key) => { selectedKey = key; }}
        onDelete={(key) => { selectedKey = key; deleteDialogOpen = true; }}
      />
    </Tabs.Content>
    
    <Tabs.Content value="tenant">
      {#if userTenants.length > 0}
        <div class="mb-4">
          <Select.Root bind:value={selectedTenantId}>
            <Select.Trigger class="w-64">
              <Select.Value placeholder="Select organization" />
            </Select.Trigger>
            <Select.Content>
              {#each userTenants as tenant}
                <Select.Item value={tenant.id}>{tenant.name}</Select.Item>
              {/each}
            </Select.Content>
          </Select.Root>
        </div>
        
        <ApiKeysTable 
          keys={tenantKeys}
          onView={(key) => { selectedKey = key; }}
          onDelete={(key) => { selectedKey = key; deleteDialogOpen = true; }}
        />
      {:else}
        <div class="text-center py-12 text-muted-foreground">
          <p>You are not a member of any organization.</p>
          <Button variant="outline" class="mt-4" href="/tenants">
            Create Organization
          </Button>
        </div>
      {/if}
    </Tabs.Content>
  </Tabs.Root>
</div>

<CreateKeyModal 
  bind:open={createModalOpen}
  tenantId={activeTab === 'tenant' ? selectedTenantId : null}
  onCreated={handleKeyCreated}
/>

<DeleteKeyDialog
  bind:open={deleteDialogOpen}
  apiKey={selectedKey}
  tenantId={activeTab === 'tenant' ? selectedTenantId : null}
  onDeleted={handleKeyDeleted}
/>
```

## Структура файлов

```
routes/(app)/api-keys/
├── +page.svelte
└── +page.ts

lib/components/api-keys/
├── ApiKeysTable.svelte
├── CreateKeyModal.svelte
├── DeleteKeyDialog.svelte
└── KeyDetailsModal.svelte
```

## API Endpoints

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/apikeys` | GET | List personal keys |
| `/api/apikeys` | POST | Create personal key |
| `/api/apikeys/:id` | GET | Get key details |
| `/api/apikeys/:id` | DELETE | Delete key |
| `/api/tenants/:id/apikeys` | GET | List tenant keys |
| `/api/tenants/:id/apikeys` | POST | Create tenant key |
| `/api/tenants/:id/apikeys/:keyId` | DELETE | Delete tenant key |

## Тестирование

- [ ] List personal keys
- [ ] Create personal key with all options
- [ ] Created key shown and copyable
- [ ] Delete personal key
- [ ] Switch to tenant tab
- [ ] Tenant selector works
- [ ] List tenant keys
- [ ] Create tenant key
- [ ] Delete tenant key
- [ ] Empty states display correctly
- [ ] Rate limits display correctly
- [ ] Models access badges work

