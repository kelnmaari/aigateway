# Миграция Auth Pages

## Приоритет: 🔴 Критический (Phase 1)

Authentication страницы - точка входа в приложение.

## Текущие файлы

- `web/login.html` - Страница входа
- `web/register.html` - Страница регистрации
- `web/bootstrap.html` - Первичная настройка системы
- `web/js/auth-guard.js` - Защита страниц

## 1. Login Page

### Текущий функционал

- Форма логина (username/email + password)
- Remember me checkbox
- Редирект после логина на return_url
- Отображение системной информации (статус, модели, RAG)
- Проверка init-status (редирект на bootstrap если нужно)

### Svelte реализация

**Route:** `routes/(auth)/login/+page.svelte`

```svelte
<script>
  import { goto } from '$app/navigation';
  import { browser } from '$app/environment';
  import { auth } from '$lib/stores/auth.svelte';
  import { systemApi } from '$lib/api';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Label } from '$lib/components/ui/label';
  import { Checkbox } from '$lib/components/ui/checkbox';
  import * as Card from '$lib/components/ui/card';
  import { toast } from 'svelte-sonner';
  
  let username = $state('');
  let password = $state('');
  let rememberMe = $state(false);
  let loading = $state(false);
  let systemInfo = $state<{ models: string[]; rag_enabled: boolean } | null>(null);
  
  // Check if already authenticated
  $effect(() => {
    if (browser && auth.isAuthenticated) {
      const returnUrl = localStorage.getItem('return_url') || '/dashboard';
      localStorage.removeItem('return_url');
      goto(returnUrl);
    }
  });
  
  // Load system info
  $effect(() => {
    systemApi.getSystemInfo().then(info => {
      systemInfo = info;
    }).catch(() => {});
  });
  
  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    if (!username.trim() || !password) return;
    
    try {
      loading = true;
      await auth.login(username, password, rememberMe);
      
      const returnUrl = localStorage.getItem('return_url') || '/dashboard';
      localStorage.removeItem('return_url');
      goto(returnUrl);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Login failed');
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-page">
  <div class="login-container">
    <Card.Root class="login-card">
      <Card.Header>
        <Card.Title>
          <span class="logo">🤖</span> AIGateway
        </Card.Title>
        <Card.Description>Sign in to your account</Card.Description>
      </Card.Header>
      
      <Card.Content>
        <form onsubmit={handleSubmit}>
          <div class="space-y-4">
            <div class="space-y-2">
              <Label for="username">Username or Email</Label>
              <Input
                id="username"
                type="text"
                bind:value={username}
                placeholder="Enter username or email"
                required
              />
            </div>
            
            <div class="space-y-2">
              <Label for="password">Password</Label>
              <Input
                id="password"
                type="password"
                bind:value={password}
                placeholder="Enter password"
                required
              />
            </div>
            
            <div class="flex items-center space-x-2">
              <Checkbox id="remember" bind:checked={rememberMe} />
              <Label for="remember" class="text-sm">Remember me</Label>
            </div>
            
            <Button type="submit" class="w-full" disabled={loading}>
              {loading ? 'Signing in...' : 'Sign In'}
            </Button>
          </div>
        </form>
      </Card.Content>
      
      <Card.Footer>
        <p class="text-sm text-muted-foreground">
          Don't have an account? <a href="/register" class="text-primary">Register</a>
        </p>
      </Card.Footer>
    </Card.Root>
    
    {#if systemInfo}
      <Card.Root class="info-card">
        <Card.Header>
          <Card.Title class="text-sm">System Status</Card.Title>
        </Card.Header>
        <Card.Content>
          <div class="info-grid">
            <div>
              <span class="label">Models:</span>
              <span class="value">{systemInfo.models?.length || 0}</span>
            </div>
            <div>
              <span class="label">RAG:</span>
              <span class="value">{systemInfo.rag_enabled ? 'Enabled' : 'Disabled'}</span>
            </div>
          </div>
        </Card.Content>
      </Card.Root>
    {/if}
  </div>
</div>

<style>
  .login-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--background);
  }
  
  .login-container {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    width: 100%;
    max-width: 400px;
    padding: 1rem;
  }
  
  .logo {
    font-size: 1.5rem;
    margin-right: 0.5rem;
  }
</style>
```

### Page Load (+page.ts)

```typescript
// routes/(auth)/login/+page.ts
import type { PageLoad } from './$types';
import { systemApi } from '$lib/api';
import { redirect } from '@sveltejs/kit';
import { browser } from '$app/environment';

export const load: PageLoad = async () => {
  if (!browser) return {};
  
  // Check if bootstrap is required
  try {
    const status = await systemApi.checkInitStatus();
    if (status.requires_bootstrap) {
      throw redirect(307, '/bootstrap');
    }
  } catch (e) {
    if (e instanceof Response && e.status === 307) throw e;
    // Ignore other errors
  }
  
  return {};
};
```

---

## 2. Register Page

### Текущий функционал

- Форма регистрации (username, email, display_name, password, confirm)
- Поддержка invitation token из URL
- Валидация invitation token
- Auto-fill email из invitation

### Svelte реализация

**Route:** `routes/(auth)/register/+page.svelte`

```svelte
<script>
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { auth } from '$lib/stores/auth.svelte';
  import { authApi } from '$lib/api';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Label } from '$lib/components/ui/label';
  import * as Card from '$lib/components/ui/card';
  import * as Alert from '$lib/components/ui/alert';
  import { toast } from 'svelte-sonner';
  
  let username = $state('');
  let email = $state('');
  let displayName = $state('');
  let password = $state('');
  let confirmPassword = $state('');
  let loading = $state(false);
  
  // Invitation handling
  let invitationToken = $derived($page.url.searchParams.get('invite'));
  let invitationValid = $state<boolean | null>(null);
  let invitationError = $state<string | null>(null);
  
  // Validate invitation on load
  $effect(() => {
    if (invitationToken) {
      validateInvitation();
    }
  });
  
  async function validateInvitation() {
    try {
      const result = await authApi.validateInvitation(invitationToken!);
      if (result.valid) {
        invitationValid = true;
        if (result.invitation?.email) {
          email = result.invitation.email;
        }
      } else {
        invitationValid = false;
        invitationError = result.error || 'Invalid invitation';
      }
    } catch {
      invitationValid = false;
      invitationError = 'Failed to validate invitation';
    }
  }
  
  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    
    if (password !== confirmPassword) {
      toast.error('Passwords do not match');
      return;
    }
    
    try {
      loading = true;
      await auth.register({
        username,
        email,
        password,
        display_name: displayName || undefined,
        invitation_token: invitationToken || undefined,
      });
      
      goto('/dashboard');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Registration failed');
    } finally {
      loading = false;
    }
  }
</script>

<div class="register-page">
  <Card.Root class="register-card">
    <Card.Header>
      <Card.Title>
        <span class="logo">🤖</span> Create Account
      </Card.Title>
      <Card.Description>Register for AIGateway</Card.Description>
    </Card.Header>
    
    <Card.Content>
      {#if invitationToken && invitationValid === false}
        <Alert.Root variant="destructive" class="mb-4">
          <Alert.Title>Invalid Invitation</Alert.Title>
          <Alert.Description>{invitationError}</Alert.Description>
        </Alert.Root>
      {/if}
      
      <form onsubmit={handleSubmit}>
        <div class="space-y-4">
          <div class="space-y-2">
            <Label for="username">Username</Label>
            <Input
              id="username"
              bind:value={username}
              placeholder="Choose a username"
              required
            />
          </div>
          
          <div class="space-y-2">
            <Label for="email">Email</Label>
            <Input
              id="email"
              type="email"
              bind:value={email}
              placeholder="your@email.com"
              required
              disabled={invitationToken && invitationValid && !!email}
            />
          </div>
          
          <div class="space-y-2">
            <Label for="displayName">Display Name (optional)</Label>
            <Input
              id="displayName"
              bind:value={displayName}
              placeholder="How you want to be called"
            />
          </div>
          
          <div class="space-y-2">
            <Label for="password">Password</Label>
            <Input
              id="password"
              type="password"
              bind:value={password}
              placeholder="Create a password"
              required
              minlength={8}
            />
          </div>
          
          <div class="space-y-2">
            <Label for="confirmPassword">Confirm Password</Label>
            <Input
              id="confirmPassword"
              type="password"
              bind:value={confirmPassword}
              placeholder="Confirm your password"
              required
            />
          </div>
          
          <Button type="submit" class="w-full" disabled={loading}>
            {loading ? 'Creating account...' : 'Create Account'}
          </Button>
        </div>
      </form>
    </Card.Content>
    
    <Card.Footer>
      <p class="text-sm text-muted-foreground">
        Already have an account? <a href="/login" class="text-primary">Sign in</a>
      </p>
    </Card.Footer>
  </Card.Root>
</div>
```

---

## 3. Bootstrap Page

### Текущий функционал

- Первичная настройка системы
- Создание первого администратора
- Требует admin_token из конфигурации сервера
- Одноразовое использование

### Svelte реализация

**Route:** `routes/(auth)/bootstrap/+page.svelte`

```svelte
<script>
  import { goto } from '$app/navigation';
  import { auth } from '$lib/stores/auth.svelte';
  import { systemApi } from '$lib/api';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Label } from '$lib/components/ui/label';
  import * as Card from '$lib/components/ui/card';
  import * as Alert from '$lib/components/ui/alert';
  import { toast } from 'svelte-sonner';
  
  let adminToken = $state('');
  let username = $state('');
  let email = $state('');
  let displayName = $state('');
  let password = $state('');
  let confirmPassword = $state('');
  let loading = $state(false);
  
  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    
    if (password !== confirmPassword) {
      toast.error('Passwords do not match');
      return;
    }
    
    try {
      loading = true;
      const response = await systemApi.bootstrap({
        admin_token: adminToken,
        username,
        email,
        password,
        display_name: displayName || undefined,
      });
      
      // Set tokens and user
      localStorage.setItem('access_token', response.token.access_token);
      localStorage.setItem('refresh_token', response.token.refresh_token);
      localStorage.setItem('user', JSON.stringify(response.user));
      
      toast.success('System initialized successfully!');
      goto('/dashboard');
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Bootstrap failed');
    } finally {
      loading = false;
    }
  }
</script>

<div class="bootstrap-page">
  <Card.Root class="bootstrap-card">
    <Card.Header>
      <Card.Title>
        <span class="logo">🤖</span> System Setup
      </Card.Title>
      <Card.Description>
        Create the first administrator account
      </Card.Description>
    </Card.Header>
    
    <Card.Content>
      <Alert.Root class="mb-4">
        <Alert.Title>First Time Setup</Alert.Title>
        <Alert.Description>
          Enter the admin token from your server configuration to create the first admin account.
        </Alert.Description>
      </Alert.Root>
      
      <form onsubmit={handleSubmit}>
        <div class="space-y-4">
          <div class="space-y-2">
            <Label for="adminToken">Admin Token</Label>
            <Input
              id="adminToken"
              type="password"
              bind:value={adminToken}
              placeholder="Enter admin token from config"
              required
            />
          </div>
          
          <hr class="my-4" />
          
          <div class="space-y-2">
            <Label for="username">Admin Username</Label>
            <Input
              id="username"
              bind:value={username}
              placeholder="Choose admin username"
              required
            />
          </div>
          
          <div class="space-y-2">
            <Label for="email">Admin Email</Label>
            <Input
              id="email"
              type="email"
              bind:value={email}
              placeholder="admin@example.com"
              required
            />
          </div>
          
          <div class="space-y-2">
            <Label for="displayName">Display Name</Label>
            <Input
              id="displayName"
              bind:value={displayName}
              placeholder="Administrator"
            />
          </div>
          
          <div class="space-y-2">
            <Label for="password">Password</Label>
            <Input
              id="password"
              type="password"
              bind:value={password}
              placeholder="Create a strong password"
              required
              minlength={8}
            />
          </div>
          
          <div class="space-y-2">
            <Label for="confirmPassword">Confirm Password</Label>
            <Input
              id="confirmPassword"
              type="password"
              bind:value={confirmPassword}
              placeholder="Confirm password"
              required
            />
          </div>
          
          <Button type="submit" class="w-full" disabled={loading}>
            {loading ? 'Initializing...' : 'Initialize System'}
          </Button>
        </div>
      </form>
    </Card.Content>
  </Card.Root>
</div>
```

---

## 4. Auth Guard (Component)

### Текущий функционал

- Проверка access_token в localStorage
- Редирект на login если не аутентифицирован
- Сохранение return_url
- Периодическая проверка токена

### Svelte реализация

**Component:** `lib/components/AuthGuard.svelte`

```svelte
<script>
  import { browser } from '$app/environment';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { auth } from '$lib/stores/auth.svelte';
  import { onMount } from 'svelte';
  
  let { children } = $props();
  let checking = $state(true);
  
  const PUBLIC_PATHS = ['/login', '/register', '/bootstrap'];
  
  onMount(() => {
    auth.init();
    checkAuth();
    
    // Periodic auth check
    const interval = setInterval(checkAuth, 60000);
    return () => clearInterval(interval);
  });
  
  function checkAuth() {
    if (!browser) return;
    
    const currentPath = $page.url.pathname;
    const isPublicPath = PUBLIC_PATHS.some(p => currentPath.startsWith(p));
    
    if (isPublicPath) {
      checking = false;
      return;
    }
    
    if (!auth.isAuthenticated) {
      localStorage.setItem('return_url', window.location.href);
      goto('/login');
      return;
    }
    
    checking = false;
  }
</script>

{#if checking}
  <div class="auth-loading">
    <div class="spinner"></div>
    <p>Loading...</p>
  </div>
{:else}
  {@render children?.()}
{/if}

<style>
  .auth-loading {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1rem;
  }
  
  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid var(--border);
    border-top-color: var(--primary);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }
  
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
```

---

## Структура файлов

```
routes/(auth)/
├── +layout.svelte        # Auth layout (без navbar)
├── login/
│   ├── +page.svelte
│   └── +page.ts
├── register/
│   ├── +page.svelte
│   └── +page.ts
└── bootstrap/
    ├── +page.svelte
    └── +page.ts

lib/components/
└── AuthGuard.svelte
```

## Auth Layout

```svelte
<!-- routes/(auth)/+layout.svelte -->
<script>
  import '../app.css';
</script>

<div class="auth-layout">
  <slot />
</div>

<style>
  .auth-layout {
    min-height: 100vh;
    background: linear-gradient(135deg, var(--background) 0%, hsl(var(--muted)) 100%);
  }
</style>
```

## Тестирование

- [ ] Login с валидными credentials
- [ ] Login с невалидными credentials
- [ ] Remember me функционал
- [ ] Редирект на return_url после логина
- [ ] Register новый аккаунт
- [ ] Register с invitation token
- [ ] Invalid invitation token handling
- [ ] Bootstrap при первом запуске
- [ ] Auth guard редирект
- [ ] Token refresh автоматический
- [ ] Logout очищает state

