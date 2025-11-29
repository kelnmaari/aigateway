# Миграция Dashboard Page

## Приоритет: 🟡 Высокий (Phase 2)

Dashboard - главная страница после логина, overview пользовательских данных.

## Текущие файлы

- `web/dashboard.html` - HTML разметка
- `web/js/dashboard.js` - Логика загрузки данных
- `web/css/dashboard.css` - Стили

## Функционал для миграции

### 1. Statistics Cards

**Текущий функционал:**
- Total Conversations count
- Tenants count
- API Keys count
- API Requests (30 days)

**Svelte компонент:** `StatCard.svelte`

```svelte
<script>
  import * as Card from '$lib/components/ui/card';
  import type { LucideIcon } from 'lucide-svelte';
  
  let { 
    title, 
    value, 
    icon: Icon, 
    trend = null,
    loading = false 
  } = $props<{
    title: string;
    value: string | number;
    icon: LucideIcon;
    trend?: { value: number; positive: boolean } | null;
    loading?: boolean;
  }>();
</script>

<Card.Root>
  <Card.Content class="pt-6">
    <div class="flex items-center justify-between">
      <div>
        <p class="text-sm font-medium text-muted-foreground">{title}</p>
        {#if loading}
          <div class="h-8 w-20 animate-pulse bg-muted rounded" />
        {:else}
          <p class="text-2xl font-bold">{value}</p>
        {/if}
        {#if trend}
          <p class="text-xs {trend.positive ? 'text-green-500' : 'text-red-500'}">
            {trend.positive ? '+' : ''}{trend.value}% from last month
          </p>
        {/if}
      </div>
      <div class="p-3 bg-primary/10 rounded-full">
        <Icon class="w-6 h-6 text-primary" />
      </div>
    </div>
  </Card.Content>
</Card.Root>
```

### 2. Available Models

**Текущий функционал:**
- Список доступных моделей
- Quick access to chat с выбранной моделью

**Svelte компонент:** Часть `DashboardPage.svelte`

```svelte
<script>
  import { modelsStore } from '$lib/stores/models.svelte';
  import { goto } from '$app/navigation';
  import * as Card from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { MessageSquare } from 'lucide-svelte';
  
  function startChat(model: string) {
    goto(`/chat?model=${encodeURIComponent(model)}`);
  }
</script>

<Card.Root>
  <Card.Header>
    <Card.Title>Available Models</Card.Title>
  </Card.Header>
  <Card.Content>
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
      {#each modelsStore.models as model}
        <button 
          class="model-card"
          onclick={() => startChat(model.id)}
        >
          <span class="model-name">{model.name}</span>
          <MessageSquare class="w-4 h-4" />
        </button>
      {/each}
    </div>
  </Card.Content>
</Card.Root>
```

### 3. Quick Actions

**Текущий функционал:**
- New Chat button
- Manage API Keys link
- View Usage link
- Manage RAG Sources link

**Svelte компонент:** `QuickActions.svelte`

```svelte
<script>
  import { goto } from '$app/navigation';
  import * as Card from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { MessageSquare, Key, BarChart3, Database } from 'lucide-svelte';
  
  const actions = [
    { label: 'New Chat', icon: MessageSquare, href: '/chat', variant: 'default' },
    { label: 'API Keys', icon: Key, href: '/api-keys', variant: 'outline' },
    { label: 'Usage', icon: BarChart3, href: '/usage', variant: 'outline' },
    { label: 'RAG Sources', icon: Database, href: '/rag-sources', variant: 'outline' },
  ];
</script>

<Card.Root>
  <Card.Header>
    <Card.Title>Quick Actions</Card.Title>
  </Card.Header>
  <Card.Content>
    <div class="flex flex-wrap gap-2">
      {#each actions as action}
        <Button 
          variant={action.variant} 
          onclick={() => goto(action.href)}
        >
          <action.icon class="w-4 h-4 mr-2" />
          {action.label}
        </Button>
      {/each}
    </div>
  </Card.Content>
</Card.Root>
```

### 4. Recent Conversations

**Текущий функционал:**
- Последние 5 conversations
- Название, модель, дата
- Quick access to conversation

**Svelte компонент:** `RecentConversations.svelte`

```svelte
<script>
  import { goto } from '$app/navigation';
  import * as Card from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { formatRelativeTime } from '$lib/utils/date';
  import { MessageSquare, ChevronRight } from 'lucide-svelte';
  
  let { conversations = [], loading = false } = $props<{
    conversations: Array<{
      id: string;
      title: string;
      model: string;
      updated_at: string;
    }>;
    loading?: boolean;
  }>();
</script>

<Card.Root>
  <Card.Header class="flex-row items-center justify-between">
    <Card.Title>Recent Conversations</Card.Title>
    <Button variant="ghost" size="sm" onclick={() => goto('/chat')}>
      View All <ChevronRight class="w-4 h-4 ml-1" />
    </Button>
  </Card.Header>
  <Card.Content>
    {#if loading}
      <div class="space-y-2">
        {#each Array(3) as _}
          <div class="h-16 animate-pulse bg-muted rounded" />
        {/each}
      </div>
    {:else if conversations.length === 0}
      <div class="text-center py-8 text-muted-foreground">
        <MessageSquare class="w-12 h-12 mx-auto mb-2 opacity-50" />
        <p>No conversations yet</p>
        <Button variant="outline" size="sm" class="mt-2" onclick={() => goto('/chat')}>
          Start a Chat
        </Button>
      </div>
    {:else}
      <div class="space-y-2">
        {#each conversations as conv}
          <button 
            class="conversation-item"
            onclick={() => goto(`/chat?id=${conv.id}`)}
          >
            <div class="flex-1 min-w-0">
              <p class="font-medium truncate">{conv.title}</p>
              <p class="text-sm text-muted-foreground">
                {conv.model} • {formatRelativeTime(conv.updated_at)}
              </p>
            </div>
            <ChevronRight class="w-4 h-4 text-muted-foreground" />
          </button>
        {/each}
      </div>
    {/if}
  </Card.Content>
</Card.Root>

<style>
  .conversation-item {
    display: flex;
    align-items: center;
    width: 100%;
    padding: 0.75rem;
    border-radius: var(--radius);
    background: hsl(var(--muted) / 0.5);
    transition: background 0.2s;
  }
  
  .conversation-item:hover {
    background: hsl(var(--muted));
  }
</style>
```

### 5. User Tenants

**Текущий функционал:**
- Список organizations пользователя
- Role в каждой organization
- Quick access to tenant

**Svelte компонент:** `UserTenants.svelte`

```svelte
<script>
  import { goto } from '$app/navigation';
  import * as Card from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Building2, ChevronRight } from 'lucide-svelte';
  
  let { tenants = [], loading = false } = $props<{
    tenants: Array<{
      id: string;
      name: string;
      role: string;
    }>;
    loading?: boolean;
  }>();
  
  function getRoleBadgeVariant(role: string) {
    switch (role) {
      case 'owner': return 'default';
      case 'admin': return 'secondary';
      default: return 'outline';
    }
  }
</script>

<Card.Root>
  <Card.Header class="flex-row items-center justify-between">
    <Card.Title>Your Organizations</Card.Title>
    <Button variant="ghost" size="sm" onclick={() => goto('/tenants')}>
      Manage <ChevronRight class="w-4 h-4 ml-1" />
    </Button>
  </Card.Header>
  <Card.Content>
    {#if loading}
      <div class="space-y-2">
        {#each Array(2) as _}
          <div class="h-12 animate-pulse bg-muted rounded" />
        {/each}
      </div>
    {:else if tenants.length === 0}
      <div class="text-center py-8 text-muted-foreground">
        <Building2 class="w-12 h-12 mx-auto mb-2 opacity-50" />
        <p>No organizations</p>
        <Button variant="outline" size="sm" class="mt-2" onclick={() => goto('/tenants')}>
          Create Organization
        </Button>
      </div>
    {:else}
      <div class="space-y-2">
        {#each tenants as tenant}
          <button 
            class="tenant-item"
            onclick={() => goto(`/tenants?id=${tenant.id}`)}
          >
            <Building2 class="w-5 h-5 text-muted-foreground" />
            <span class="flex-1 text-left font-medium">{tenant.name}</span>
            <Badge variant={getRoleBadgeVariant(tenant.role)}>
              {tenant.role}
            </Badge>
          </button>
        {/each}
      </div>
    {/if}
  </Card.Content>
</Card.Root>

<style>
  .tenant-item {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    padding: 0.75rem;
    border-radius: var(--radius);
    background: hsl(var(--muted) / 0.5);
    transition: background 0.2s;
  }
  
  .tenant-item:hover {
    background: hsl(var(--muted));
  }
</style>
```

## Полная страница Dashboard

```svelte
<!-- routes/(app)/dashboard/+page.svelte -->
<script>
  import { onMount } from 'svelte';
  import { dashboardApi, conversationsApi, tenantsApi } from '$lib/api';
  import { modelsStore } from '$lib/stores/models.svelte';
  import { auth } from '$lib/stores/auth.svelte';
  import StatCard from '$lib/components/dashboard/StatCard.svelte';
  import QuickActions from '$lib/components/dashboard/QuickActions.svelte';
  import RecentConversations from '$lib/components/dashboard/RecentConversations.svelte';
  import UserTenants from '$lib/components/dashboard/UserTenants.svelte';
  import { MessageSquare, Building2, Key, Activity } from 'lucide-svelte';
  
  let stats = $state({
    conversations: 0,
    tenants: 0,
    apiKeys: 0,
    requests30d: 0,
  });
  let recentConversations = $state([]);
  let userTenants = $state([]);
  let loading = $state(true);
  
  onMount(async () => {
    try {
      // Batch fetch dashboard data
      const [statsData, convData, tenantsData] = await Promise.all([
        dashboardApi.getStats(),
        conversationsApi.list(),
        tenantsApi.listUserTenants(),
        modelsStore.fetch(),
      ]);
      
      stats = {
        conversations: statsData.conversations_count,
        tenants: statsData.tenants_count,
        apiKeys: statsData.api_keys_count,
        requests30d: statsData.requests_30d,
      };
      
      recentConversations = (convData.conversations || []).slice(0, 5);
      userTenants = tenantsData.tenants || [];
    } finally {
      loading = false;
    }
  });
</script>

<svelte:head>
  <title>Dashboard - AIGateway</title>
</svelte:head>

<div class="dashboard">
  <header class="page-header">
    <div>
      <h1 class="text-2xl font-bold">Dashboard</h1>
      <p class="text-muted-foreground">
        Welcome back, {auth.user?.display_name || auth.user?.username}
      </p>
    </div>
  </header>
  
  <!-- Stats Grid -->
  <div class="stats-grid">
    <StatCard 
      title="Conversations" 
      value={stats.conversations} 
      icon={MessageSquare}
      {loading}
    />
    <StatCard 
      title="Organizations" 
      value={stats.tenants} 
      icon={Building2}
      {loading}
    />
    <StatCard 
      title="API Keys" 
      value={stats.apiKeys} 
      icon={Key}
      {loading}
    />
    <StatCard 
      title="API Requests (30d)" 
      value={stats.requests30d.toLocaleString()} 
      icon={Activity}
      {loading}
    />
  </div>
  
  <!-- Quick Actions -->
  <QuickActions />
  
  <!-- Two Column Layout -->
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
    <RecentConversations conversations={recentConversations} {loading} />
    <UserTenants tenants={userTenants} {loading} />
  </div>
  
  <!-- Available Models -->
  <div class="models-section">
    <!-- Models grid -->
  </div>
</div>

<style>
  .dashboard {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }
  
  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(1, 1fr);
    gap: 1rem;
  }
  
  @media (min-width: 640px) {
    .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }
  
  @media (min-width: 1024px) {
    .stats-grid {
      grid-template-columns: repeat(4, 1fr);
    }
  }
</style>
```

## Структура файлов

```
routes/(app)/dashboard/
├── +page.svelte
└── +page.ts

lib/components/dashboard/
├── StatCard.svelte
├── QuickActions.svelte
├── RecentConversations.svelte
├── UserTenants.svelte
└── ModelsGrid.svelte
```

## API Endpoints

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/api/dashboard/batch` | GET | Batch stats |
| `/api/conversations` | GET | Recent conversations |
| `/api/users/me/tenants` | GET | User tenants |
| `/api/models` | GET | Available models |

## Тестирование

- [ ] Stats загружаются корректно
- [ ] Loading states отображаются
- [ ] Empty states для conversations/tenants
- [ ] Quick actions работают
- [ ] Recent conversations кликабельны
- [ ] Tenants с правильными badges
- [ ] Models grid функционален
- [ ] Responsive layout на всех размерах

