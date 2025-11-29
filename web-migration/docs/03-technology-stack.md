# Технологический стек Svelte миграции

## Core Framework

### Svelte 5.37.0 (Latest Stable)
- **Runes API** — Новая система реактивности
- **$state** — Реактивные переменные
- **$derived** — Вычисляемые значения
- **$effect** — Побочные эффекты
- **$props** — Пропсы компонентов
- **Snippets** — Переиспользуемые фрагменты шаблонов

```svelte
<script>
  let count = $state(0);
  const doubled = $derived(count * 2);
  
  $effect(() => {
    console.log('Count changed:', count);
  });
</script>
```

### SvelteKit 2.x
- **Static Adapter** — Статическая сборка для embed.FS
- **File-based routing** — Роутинг на основе файловой структуры
- **Prerendering** — Pre-render всех страниц
- **Hash-based routing** — Для SPA без сервера

```bash
npm create svelte@latest web-svelte
```

## UI Components

### shadcn-svelte 1.0.0-next.19
- **Svelte 5 совместимость** — Использует новый Runes API
- **Bits UI** — Headless UI примитивы
- **Tailwind CSS** — Стилизация
- **TypeScript** — Полная типизация

**Установка:**
```bash
npx shadcn-svelte@next init
```

**Доступные компоненты:**
| Компонент | Использование |
|-----------|---------------|
| Button | Кнопки по всему приложению |
| Card | Dashboard cards, stat cards |
| Dialog | Модальные окна |
| Input | Формы |
| Select | Dropdowns |
| Table | Data tables |
| Tabs | Tab navigation |
| Toast | Уведомления |
| Badge | Status badges |
| Dropdown Menu | User menu, actions |
| Command | Command palette |
| Alert | Alerts, warnings |
| Avatar | User avatars |
| Checkbox | Form checkboxes |
| Form | Form validation |
| Label | Form labels |
| Textarea | Multiline inputs |
| Popover | Tooltips, popovers |
| Sheet | Sidebars, panels |
| Skeleton | Loading states |
| Progress | Progress bars |
| Slider | Range inputs |
| Switch | Toggle switches |
| Tooltip | Hover tooltips |
| Separator | Dividers |

### Lucide Svelte (Icons)
```bash
npm install lucide-svelte
```

```svelte
<script>
  import { Home, Settings, User } from 'lucide-svelte';
</script>

<Home class="w-4 h-4" />
```

## Internationalization (i18n)

### Paraglide JS (Inlang)
- **Tree-shakable** — Только используемые переводы в бандле
- **Type-safe** — Полная типизация ключей
- **JSON формат** — Удобное редактирование
- **SSG support** — Статическая генерация

```bash
npm install @inlang/paraglide-js
```

**Структура:**
```
project.inlang/settings.json
messages/
  en.json
  ru.json
```

**Использование:**
```svelte
<script>
  import * as m from '$lib/paraglide/messages';
</script>

<h1>{m.dashboard_title()}</h1>
<p>{m.greeting({ name: user.name })}</p>
```

**Подробнее:** [09-i18n-strategy.md](09-i18n-strategy.md)

## Styling

### Tailwind CSS 4.x
- **CSS Variables** — Интеграция с dark theme
- **JIT** — Just-in-Time compilation
- **Container Queries** — Responsive компоненты

```javascript
// tailwind.config.js
export default {
  darkMode: 'class',
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      colors: {
        background: 'var(--background)',
        foreground: 'var(--foreground)',
        primary: {
          DEFAULT: 'var(--primary)',
          foreground: 'var(--primary-foreground)',
        },
        // ... shadcn theme tokens
      },
    },
  },
};
```

### CSS Variables Mapping
```css
/* Текущие -> Новые (shadcn-compatible) */
--bg-primary     -> --background
--bg-secondary   -> --card
--bg-tertiary    -> --muted
--text-primary   -> --foreground
--text-secondary -> --muted-foreground
--accent-primary -> --primary
--border-color   -> --border
--error-color    -> --destructive
--success-color  -> --success (custom)
```

## State Management

### Svelte 5 Runes
Основной подход — использование `$state` и `$derived` из Svelte 5 Runes API.

```typescript
// stores/auth.svelte.ts
import { browser } from '$app/environment';

class AuthStore {
  #accessToken = $state<string | null>(null);
  #user = $state<User | null>(null);
  
  get isAuthenticated() {
    return this.#accessToken !== null;
  }
  
  get user() {
    return this.#user;
  }
  
  login(tokens: TokenPair, user: User) {
    this.#accessToken = tokens.access_token;
    this.#user = user;
    if (browser) {
      localStorage.setItem('access_token', tokens.access_token);
      localStorage.setItem('refresh_token', tokens.refresh_token);
    }
  }
  
  logout() {
    this.#accessToken = null;
    this.#user = null;
    if (browser) {
      localStorage.removeItem('access_token');
      localStorage.removeItem('refresh_token');
    }
  }
}

export const auth = new AuthStore();
```

### Context API для локального состояния
```svelte
<!-- Parent.svelte -->
<script>
  import { setContext } from 'svelte';
  
  const chatState = $state({
    messages: [],
    isLoading: false,
  });
  
  setContext('chat', chatState);
</script>

<!-- Child.svelte -->
<script>
  import { getContext } from 'svelte';
  const chat = getContext('chat');
</script>
```

## API Layer

### Typed Fetch Client
```typescript
// lib/api/client.ts
import { auth } from '$lib/stores/auth.svelte';
import { goto } from '$app/navigation';

class ApiClient {
  private baseUrl = '';
  
  private async request<T>(
    path: string, 
    options: RequestInit = {}
  ): Promise<T> {
    const headers = new Headers(options.headers);
    headers.set('Content-Type', 'application/json');
    
    if (auth.accessToken) {
      headers.set('Authorization', `Bearer ${auth.accessToken}`);
    }
    
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...options,
      headers,
    });
    
    if (response.status === 401) {
      const refreshed = await this.refreshToken();
      if (!refreshed) {
        auth.logout();
        goto('/login');
        throw new Error('Session expired');
      }
      return this.request(path, options);
    }
    
    if (!response.ok) {
      const error = await response.json().catch(() => ({}));
      throw new Error(error.error || `Request failed: ${response.status}`);
    }
    
    return response.json();
  }
  
  get<T>(path: string): Promise<T> {
    return this.request(path);
  }
  
  post<T>(path: string, data: unknown): Promise<T> {
    return this.request(path, {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }
  
  // ... put, delete, etc.
}

export const api = new ApiClient();
```

## Form Validation

### Superforms + Zod
```bash
npm install sveltekit-superforms zod
```

```typescript
// routes/login/+page.server.ts
import { z } from 'zod';
import { superValidate } from 'sveltekit-superforms/server';

const loginSchema = z.object({
  username: z.string().min(1, 'Username is required'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
  rememberMe: z.boolean().default(false),
});

export const load = async () => {
  const form = await superValidate(loginSchema);
  return { form };
};
```

```svelte
<!-- +page.svelte -->
<script>
  import { superForm } from 'sveltekit-superforms/client';
  
  export let data;
  const { form, errors, enhance } = superForm(data.form);
</script>

<form method="POST" use:enhance>
  <input name="username" bind:value={$form.username} />
  {#if $errors.username}
    <span class="error">{$errors.username}</span>
  {/if}
</form>
```

## Markdown Rendering

### marked + highlight.js
```bash
npm install marked highlight.js
```

```typescript
// lib/utils/markdown.ts
import { marked } from 'marked';
import hljs from 'highlight.js';

marked.setOptions({
  highlight: (code, lang) => {
    if (lang && hljs.getLanguage(lang)) {
      return hljs.highlight(code, { language: lang }).value;
    }
    return hljs.highlightAuto(code).value;
  },
});

export function renderMarkdown(content: string): string {
  return marked.parse(content);
}
```

## SSE Streaming

### Native EventSource
```typescript
// lib/api/stream.ts
export async function* streamChat(
  messages: Message[],
  params: ChatParams
): AsyncGenerator<string> {
  const response = await fetch('/v1/chat/completions', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${auth.accessToken}`,
    },
    body: JSON.stringify({
      ...params,
      messages,
      stream: true,
    }),
  });
  
  const reader = response.body?.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  
  while (reader) {
    const { done, value } = await reader.read();
    if (done) break;
    
    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';
    
    for (const line of lines) {
      const trimmed = line.trim();
      if (trimmed.startsWith('data: ') && trimmed !== 'data: [DONE]') {
        const data = JSON.parse(trimmed.slice(6));
        const content = data.choices?.[0]?.delta?.content;
        if (content) yield content;
      }
    }
  }
}
```

## Build Configuration

### vite.config.ts
```typescript
import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  build: {
    target: 'esnext',
    minify: 'esbuild',
  },
});
```

### svelte.config.js
```javascript
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

export default {
  preprocess: vitePreprocess(),
  kit: {
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: 'index.html',
      precompress: false,
      strict: true,
    }),
    paths: {
      base: '',
    },
    prerender: {
      entries: ['*'],
    },
  },
};
```

## Package Versions Summary

```json
{
  "dependencies": {
    "svelte": "^5.37.0",
    "bits-ui": "^1.0.0-next.x",
    "lucide-svelte": "^0.460.0",
    "clsx": "^2.1.1",
    "tailwind-merge": "^2.5.5",
    "marked": "^15.0.0",
    "highlight.js": "^11.10.0",
    "zod": "^3.24.0"
  },
  "devDependencies": {
    "@sveltejs/adapter-static": "^3.0.8",
    "@sveltejs/kit": "^2.15.0",
    "@sveltejs/vite-plugin-svelte": "^5.0.0",
    "@tailwindcss/typography": "^0.5.15",
    "autoprefixer": "^10.4.20",
    "postcss": "^8.4.49",
    "sveltekit-superforms": "^2.22.0",
    "tailwindcss": "^4.0.0",
    "typescript": "^5.7.0",
    "vite": "^6.0.0"
  }
}
```

## Security Considerations

1. **XSS Prevention** — Svelte автоматически экранирует HTML
2. **CSRF** — Все мутации через POST с JSON body
3. **Token Storage** — localStorage (как сейчас)
4. **Sensitive Data** — Никогда не логировать токены
5. **Input Validation** — Zod на клиенте, повторная на сервере

## Browser Support

- Chrome 90+
- Firefox 90+
- Safari 15+
- Edge 90+

(Соответствует `target: 'esnext'` в Vite config)

