# Маппинг HTML → Svelte компоненты

## Глобальная структура

### HTML шаблон (текущий)
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Page - AIGateway</title>
    <link rel="stylesheet" href="/css/theme.css">
    <link rel="stylesheet" href="/css/style.css">
    <link rel="stylesheet" href="/css/dashboard.css">
    <link rel="stylesheet" href="/css/notifications.css">
</head>
<body class="dashboard-page">
    <div id="navigation-placeholder"></div>
    <main class="dashboard-main">
        <div class="dashboard-container">
            <!-- Content -->
        </div>
    </main>
    <script src="/js/auth-guard.js"></script>
    <script src="/js/api.js"></script>
    <script src="/js/components/navbar.js"></script>
    <script src="/js/page-specific.js"></script>
    <script src="/framework/framework.js"></script>
</body>
</html>
```

### Svelte эквивалент
```svelte
<!-- routes/(app)/+layout.svelte -->
<script>
  import Navbar from '$lib/components/layout/Navbar.svelte';
  import { Toaster } from '$lib/components/ui/sonner';
  import AuthGuard from '$lib/components/AuthGuard.svelte';
</script>

<AuthGuard>
  <div class="dashboard-page">
    <Navbar />
    <main class="dashboard-main">
      <div class="dashboard-container">
        <slot />
      </div>
    </main>
  </div>
</AuthGuard>
<Toaster />
```

---

## Маппинг страниц

### Публичные страницы

| HTML файл | Svelte route | Layout |
|-----------|--------------|--------|
| `login.html` | `routes/(auth)/login/+page.svelte` | `(auth)/+layout.svelte` (без navbar) |
| `register.html` | `routes/(auth)/register/+page.svelte` | `(auth)/+layout.svelte` |
| `bootstrap.html` | `routes/(auth)/bootstrap/+page.svelte` | `(auth)/+layout.svelte` |

### Защищённые страницы (app)

| HTML файл | Svelte route | Layout |
|-----------|--------------|--------|
| `dashboard.html` | `routes/(app)/dashboard/+page.svelte` | `(app)/+layout.svelte` |
| `chat.html` | `routes/(app)/chat/+page.svelte` | `(app)/chat/+layout.svelte` (свой layout) |
| `api-keys.html` | `routes/(app)/api-keys/+page.svelte` | `(app)/+layout.svelte` |
| `tenants.html` | `routes/(app)/tenants/+page.svelte` | `(app)/+layout.svelte` |
| `profile.html` | `routes/(app)/profile/+page.svelte` | `(app)/+layout.svelte` |
| `profile-devices.html` | `routes/(app)/profile/devices/+page.svelte` | `(app)/+layout.svelte` |
| `usage.html` | `routes/(app)/usage/+page.svelte` | `(app)/+layout.svelte` |
| `files.html` | `routes/(app)/files/+page.svelte` | `(app)/+layout.svelte` |
| `rag-sources.html` | `routes/(app)/rag-sources/+page.svelte` | `(app)/+layout.svelte` |
| `mcp.html` | `routes/(app)/mcp/+page.svelte` | `(app)/+layout.svelte` |
| `about.html` | `routes/(app)/about/+page.svelte` | `(app)/+layout.svelte` |
| `monitor.html` | `routes/(app)/monitor/+page.svelte` | `(app)/+layout.svelte` |
| `downloads.html` | `routes/(app)/downloads/+page.svelte` | `(app)/+layout.svelte` |
| `huggingface.html` | `routes/(app)/huggingface/+page.svelte` | `(app)/+layout.svelte` |
| `yzma.html` | `routes/(app)/yzma/+page.svelte` | `(app)/+layout.svelte` |

### Админ страницы

| HTML файл | Svelte route | Layout |
|-----------|--------------|--------|
| `admin.html` | `routes/(admin)/admin/+page.svelte` | `(admin)/+layout.svelte` |
| `admin-audit.html` | `routes/(admin)/admin/audit/+page.svelte` | `(admin)/+layout.svelte` |
| `admin-invitations.html` | `routes/(admin)/admin/invitations/+page.svelte` | `(admin)/+layout.svelte` |
| `admin-rag.html` | `routes/(admin)/admin/rag/+page.svelte` | `(admin)/+layout.svelte` |
| `admin-rbac.html` | `routes/(admin)/admin/rbac/+page.svelte` | `(admin)/+layout.svelte` |
| `admin-registry.html` | `routes/(admin)/admin/registry/+page.svelte` | `(admin)/+layout.svelte` |

---

## Маппинг UI компонентов

### Кнопки

| CSS класс | shadcn-svelte компонент |
|-----------|------------------------|
| `.btn.btn-primary` | `<Button>` |
| `.btn.btn-secondary` | `<Button variant="secondary">` |
| `.btn.btn-danger` | `<Button variant="destructive">` |
| `.btn.btn-icon` | `<Button variant="ghost" size="icon">` |
| `.btn.btn-block` | `<Button class="w-full">` |
| `.btn.btn-sm` | `<Button size="sm">` |

```svelte
<script>
  import { Button } from '$lib/components/ui/button';
</script>

<Button>Primary</Button>
<Button variant="secondary">Secondary</Button>
<Button variant="destructive">Danger</Button>
```

### Формы

| HTML элемент | shadcn-svelte компонент |
|--------------|------------------------|
| `.form-group` | Form field wrapper |
| `.form-group label` | `<Label>` |
| `.form-group input` | `<Input>` |
| `.form-group select` | `<Select>` |
| `.form-group textarea` | `<Textarea>` |
| `.form-group small` | Description text |

```svelte
<script>
  import { Label } from '$lib/components/ui/label';
  import { Input } from '$lib/components/ui/input';
</script>

<div class="space-y-2">
  <Label for="email">Email</Label>
  <Input id="email" type="email" placeholder="your@email.com" />
  <p class="text-sm text-muted-foreground">We'll never share your email.</p>
</div>
```

### Карточки

| CSS класс | shadcn-svelte компонент |
|-----------|------------------------|
| `.card` | `<Card>` |
| `.card-header` | `<CardHeader>` |
| `.card-body` | `<CardContent>` |
| `.card-footer` | `<CardFooter>` |
| `.card-title` | `<CardTitle>` |
| `.stat-card` | Custom `<StatCard>` |

```svelte
<script>
  import * as Card from '$lib/components/ui/card';
</script>

<Card.Root>
  <Card.Header>
    <Card.Title>Title</Card.Title>
    <Card.Description>Description</Card.Description>
  </Card.Header>
  <Card.Content>
    <p>Content</p>
  </Card.Content>
  <Card.Footer>
    <Button>Action</Button>
  </Card.Footer>
</Card.Root>
```

### Таблицы

| CSS класс | shadcn-svelte компонент |
|-----------|------------------------|
| `.table-container` | Wrapper div |
| `.data-table` | `<Table>` |
| `thead` | `<TableHeader>` |
| `tbody` | `<TableBody>` |
| `tr` | `<TableRow>` |
| `th` | `<TableHead>` |
| `td` | `<TableCell>` |

```svelte
<script>
  import * as Table from '$lib/components/ui/table';
</script>

<Table.Root>
  <Table.Header>
    <Table.Row>
      <Table.Head>Name</Table.Head>
      <Table.Head>Status</Table.Head>
    </Table.Row>
  </Table.Header>
  <Table.Body>
    {#each items as item}
      <Table.Row>
        <Table.Cell>{item.name}</Table.Cell>
        <Table.Cell>{item.status}</Table.Cell>
      </Table.Row>
    {/each}
  </Table.Body>
</Table.Root>
```

### Модальные окна

| CSS класс | shadcn-svelte компонент |
|-----------|------------------------|
| `.modal` | `<Dialog>` |
| `.modal-content` | `<DialogContent>` |
| `.modal-header` | `<DialogHeader>` |
| `.modal-body` | Content area |
| `.modal-footer` | `<DialogFooter>` |
| `.modal-close` | `<DialogClose>` |

```svelte
<script>
  import * as Dialog from '$lib/components/ui/dialog';
  let open = $state(false);
</script>

<Dialog.Root bind:open>
  <Dialog.Trigger asChild let:builder>
    <Button builders={[builder]}>Open</Button>
  </Dialog.Trigger>
  <Dialog.Content>
    <Dialog.Header>
      <Dialog.Title>Title</Dialog.Title>
      <Dialog.Description>Description</Dialog.Description>
    </Dialog.Header>
    <div class="py-4">
      <!-- Content -->
    </div>
    <Dialog.Footer>
      <Button variant="secondary" onclick={() => open = false}>Cancel</Button>
      <Button>Save</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
```

### Табы

| CSS класс | shadcn-svelte компонент |
|-----------|------------------------|
| `.tabs` | `<Tabs>` |
| `.tab-btn` | `<TabsTrigger>` |
| `.tab-content` | `<TabsContent>` |

```svelte
<script>
  import * as Tabs from '$lib/components/ui/tabs';
</script>

<Tabs.Root value="personal">
  <Tabs.List>
    <Tabs.Trigger value="personal">Personal Keys</Tabs.Trigger>
    <Tabs.Trigger value="tenant">Organization Keys</Tabs.Trigger>
  </Tabs.List>
  <Tabs.Content value="personal">
    <!-- Personal keys content -->
  </Tabs.Content>
  <Tabs.Content value="tenant">
    <!-- Tenant keys content -->
  </Tabs.Content>
</Tabs.Root>
```

### Badges

| CSS класс | shadcn-svelte компонент |
|-----------|------------------------|
| `.badge` | `<Badge>` |
| `.badge-success` | `<Badge variant="success">` |
| `.badge-warning` | `<Badge variant="warning">` |
| `.badge-danger` | `<Badge variant="destructive">` |
| `.badge-info` | `<Badge variant="secondary">` |

```svelte
<script>
  import { Badge } from '$lib/components/ui/badge';
</script>

<Badge>Default</Badge>
<Badge variant="secondary">Secondary</Badge>
<Badge variant="destructive">Error</Badge>
```

### Уведомления

| Текущее | shadcn-svelte |
|---------|---------------|
| `toast.success()` | `toast.success()` (sonner) |
| `toast.error()` | `toast.error()` |
| `toast.info()` | `toast.info()` |
| `toast.warning()` | `toast.warning()` |

```svelte
<script>
  import { toast } from 'svelte-sonner';
  
  function handleClick() {
    toast.success('Operation completed!');
  }
</script>
```

### Select/Dropdown

| CSS класс | shadcn-svelte компонент |
|-----------|------------------------|
| `select.form-control` | `<Select>` |
| Dropdown menu | `<DropdownMenu>` |

```svelte
<script>
  import * as Select from '$lib/components/ui/select';
  let value = $state('');
</script>

<Select.Root bind:value>
  <Select.Trigger>
    <Select.Value placeholder="Select option" />
  </Select.Trigger>
  <Select.Content>
    <Select.Item value="option1">Option 1</Select.Item>
    <Select.Item value="option2">Option 2</Select.Item>
  </Select.Content>
</Select.Root>
```

---

## Маппинг JS функционала

### API Client (api.js → lib/api/client.ts)

```typescript
// lib/api/client.ts
class ApiClient {
  async get<T>(path: string): Promise<T>
  async post<T>(path: string, data: unknown): Promise<T>
  async put<T>(path: string, data: unknown): Promise<T>
  async delete(path: string): Promise<void>
  
  // Auth
  async login(credentials: LoginRequest): Promise<AuthResponse>
  async register(data: RegisterRequest): Promise<AuthResponse>
  async refreshToken(): Promise<boolean>
  async logout(): Promise<void>
  
  // ... other methods from api.js
}

export const api = new ApiClient();
```

### Auth Guard (auth-guard.js → lib/components/AuthGuard.svelte)

```svelte
<script>
  import { browser } from '$app/environment';
  import { goto } from '$app/navigation';
  import { auth } from '$lib/stores/auth.svelte';
  import { onMount } from 'svelte';
  
  let { children } = $props();
  let checking = $state(true);
  
  onMount(async () => {
    if (!auth.isAuthenticated) {
      const returnUrl = window.location.href;
      localStorage.setItem('return_url', returnUrl);
      goto('/login');
      return;
    }
    checking = false;
  });
</script>

{#if checking}
  <div class="loading">Loading...</div>
{:else}
  {@render children?.()}
{/if}
```

### Navbar (navbar.js → lib/components/layout/Navbar.svelte)

```svelte
<script>
  import { auth } from '$lib/stores/auth.svelte';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
  import { Home, MessageSquare, Key, Building2, User, LogOut } from 'lucide-svelte';
</script>

<nav class="navbar">
  <a href="/dashboard" class="logo">
    <span>🤖</span> AIGateway
  </a>
  
  <div class="nav-links">
    <a href="/chat"><MessageSquare class="w-4 h-4" /> Chat</a>
    <a href="/dashboard"><Home class="w-4 h-4" /> Dashboard</a>
    <a href="/api-keys"><Key class="w-4 h-4" /> API Keys</a>
    <a href="/tenants"><Building2 class="w-4 h-4" /> Tenants</a>
    
    {#if auth.user?.is_admin}
      <a href="/admin" class="admin-link">Admin</a>
    {/if}
  </div>
  
  <DropdownMenu.Root>
    <DropdownMenu.Trigger>
      <Button variant="ghost">
        <User class="w-4 h-4" />
        {auth.user?.username}
      </Button>
    </DropdownMenu.Trigger>
    <DropdownMenu.Content>
      <DropdownMenu.Item href="/profile">Profile</DropdownMenu.Item>
      <DropdownMenu.Item href="/profile/devices">Devices</DropdownMenu.Item>
      <DropdownMenu.Separator />
      <DropdownMenu.Item onclick={() => auth.logout()}>
        <LogOut class="w-4 h-4 mr-2" /> Logout
      </DropdownMenu.Item>
    </DropdownMenu.Content>
  </DropdownMenu.Root>
</nav>
```

---

## CSS миграция

### CSS Variables маппинг

```css
/* app.css */
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  :root {
    --background: 33 33 33;          /* #212121 */
    --foreground: 236 236 236;       /* #ececec */
    --card: 45 45 45;                /* #2d2d2d */
    --card-foreground: 236 236 236;
    --popover: 45 45 45;
    --popover-foreground: 236 236 236;
    --primary: 16 163 127;           /* #10a37f */
    --primary-foreground: 255 255 255;
    --secondary: 58 58 58;           /* #3a3a3a */
    --secondary-foreground: 236 236 236;
    --muted: 58 58 58;
    --muted-foreground: 179 179 179; /* #b3b3b3 */
    --accent: 16 163 127;
    --accent-foreground: 255 255 255;
    --destructive: 239 68 68;        /* #ef4444 */
    --destructive-foreground: 255 255 255;
    --border: 74 74 74;              /* #4a4a4a */
    --input: 74 74 74;
    --ring: 16 163 127;
    --radius: 0.5rem;
  }
}
```

### Tailwind классы эквиваленты

| CSS класс | Tailwind класс |
|-----------|----------------|
| `.dashboard-page` | `min-h-screen bg-background` |
| `.dashboard-main` | `container mx-auto p-6` |
| `.page-header` | `flex items-center justify-between mb-6` |
| `.page-description` | `text-muted-foreground` |
| `.stats-grid` | `grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4` |
| `.table-container` | `rounded-lg border` |
| `.loading` | `animate-spin` |

