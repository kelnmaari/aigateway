# Система тем (Light / Dark)

## Требования

- **Две темы:** Light и Dark
- **Сохранение выбора:** localStorage браузера
- **Автоопределение:** Учитывать prefers-color-scheme
- **Мгновенное переключение:** Без перезагрузки страницы

---

## Реализация с Tailwind CSS 4

### tailwind.config.ts

```typescript
import type { Config } from 'tailwindcss';

export default {
  darkMode: 'class', // Используем class-based dark mode
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      colors: {
        // Semantic colors
        background: 'var(--color-background)',
        foreground: 'var(--color-foreground)',
        card: 'var(--color-card)',
        'card-foreground': 'var(--color-card-foreground)',
        primary: 'var(--color-primary)',
        'primary-foreground': 'var(--color-primary-foreground)',
        secondary: 'var(--color-secondary)',
        'secondary-foreground': 'var(--color-secondary-foreground)',
        muted: 'var(--color-muted)',
        'muted-foreground': 'var(--color-muted-foreground)',
        accent: 'var(--color-accent)',
        'accent-foreground': 'var(--color-accent-foreground)',
        destructive: 'var(--color-destructive)',
        border: 'var(--color-border)',
        input: 'var(--color-input)',
        ring: 'var(--color-ring)',
      },
    },
  },
} satisfies Config;
```

### src/app.css

```css
@import 'tailwindcss';

:root {
  /* Light theme (default) */
  --color-background: 255 255 255;       /* #ffffff */
  --color-foreground: 15 23 42;          /* slate-900 */
  --color-card: 255 255 255;
  --color-card-foreground: 15 23 42;
  --color-primary: 16 163 127;           /* emerald-500 */
  --color-primary-foreground: 255 255 255;
  --color-secondary: 241 245 249;        /* slate-100 */
  --color-secondary-foreground: 15 23 42;
  --color-muted: 241 245 249;
  --color-muted-foreground: 100 116 139; /* slate-500 */
  --color-accent: 241 245 249;
  --color-accent-foreground: 15 23 42;
  --color-destructive: 239 68 68;        /* red-500 */
  --color-border: 226 232 240;           /* slate-200 */
  --color-input: 226 232 240;
  --color-ring: 16 163 127;
  
  /* Radius */
  --radius: 0.5rem;
}

.dark {
  /* Dark theme */
  --color-background: 15 23 42;          /* slate-900 */
  --color-foreground: 248 250 252;       /* slate-50 */
  --color-card: 30 41 59;                /* slate-800 */
  --color-card-foreground: 248 250 252;
  --color-primary: 16 163 127;           /* emerald-500 */
  --color-primary-foreground: 255 255 255;
  --color-secondary: 51 65 85;           /* slate-700 */
  --color-secondary-foreground: 248 250 252;
  --color-muted: 51 65 85;
  --color-muted-foreground: 148 163 184; /* slate-400 */
  --color-accent: 51 65 85;
  --color-accent-foreground: 248 250 252;
  --color-destructive: 239 68 68;
  --color-border: 51 65 85;
  --color-input: 51 65 85;
  --color-ring: 16 163 127;
}

/* Smooth transition */
* {
  transition: background-color 0.2s ease, border-color 0.2s ease, color 0.1s ease;
}

body {
  background-color: rgb(var(--color-background));
  color: rgb(var(--color-foreground));
}
```

---

## Theme Store

### src/lib/stores/theme.ts

```typescript
import { browser } from '$app/environment';

type Theme = 'light' | 'dark' | 'system';

const STORAGE_KEY = 'aigateway_theme';

class ThemeStore {
  private _theme = $state<Theme>('system');
  private _resolved = $state<'light' | 'dark'>('dark');
  
  constructor() {
    if (browser) {
      this.init();
    }
  }
  
  private init() {
    // Load saved preference
    const saved = localStorage.getItem(STORAGE_KEY) as Theme | null;
    if (saved && ['light', 'dark', 'system'].includes(saved)) {
      this._theme = saved;
    }
    
    // Resolve actual theme
    this.resolve();
    
    // Listen for system preference changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if (this._theme === 'system') {
        this.resolve();
      }
    });
  }
  
  private resolve() {
    let resolved: 'light' | 'dark';
    
    if (this._theme === 'system') {
      resolved = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    } else {
      resolved = this._theme;
    }
    
    this._resolved = resolved;
    this.applyToDOM(resolved);
  }
  
  private applyToDOM(theme: 'light' | 'dark') {
    const root = document.documentElement;
    
    if (theme === 'dark') {
      root.classList.add('dark');
    } else {
      root.classList.remove('dark');
    }
    
    // Update meta theme-color for mobile browsers
    const metaThemeColor = document.querySelector('meta[name="theme-color"]');
    if (metaThemeColor) {
      metaThemeColor.setAttribute('content', theme === 'dark' ? '#0f172a' : '#ffffff');
    }
  }
  
  get theme() {
    return this._theme;
  }
  
  get resolved() {
    return this._resolved;
  }
  
  get isDark() {
    return this._resolved === 'dark';
  }
  
  set(theme: Theme) {
    this._theme = theme;
    
    if (browser) {
      localStorage.setItem(STORAGE_KEY, theme);
      this.resolve();
    }
  }
  
  toggle() {
    // Simple toggle between light and dark
    this.set(this._resolved === 'dark' ? 'light' : 'dark');
  }
}

export const themeStore = new ThemeStore();
```

---

## Theme Toggle Component

### src/lib/components/ThemeToggle.svelte

```svelte
<script lang="ts">
  import { themeStore } from '$lib/stores/theme';
  import { Sun, Moon, Monitor } from 'lucide-svelte';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
  import { Button } from '$lib/components/ui/button';
  
  const themes = [
    { value: 'light', label: 'Light', icon: Sun },
    { value: 'dark', label: 'Dark', icon: Moon },
    { value: 'system', label: 'System', icon: Monitor },
  ] as const;
  
  const CurrentIcon = $derived(
    themeStore.resolved === 'dark' ? Moon : Sun
  );
</script>

<DropdownMenu.Root>
  <DropdownMenu.Trigger>
    {#snippet child({ props })}
      <Button variant="ghost" size="icon" {...props}>
        <CurrentIcon class="h-5 w-5" />
        <span class="sr-only">Toggle theme</span>
      </Button>
    {/snippet}
  </DropdownMenu.Trigger>
  <DropdownMenu.Content align="end">
    {#each themes as { value, label, icon: Icon }}
      <DropdownMenu.Item 
        onclick={() => themeStore.set(value)}
        class:bg-accent={themeStore.theme === value}
      >
        <Icon class="mr-2 h-4 w-4" />
        <span>{label}</span>
      </DropdownMenu.Item>
    {/each}
  </DropdownMenu.Content>
</DropdownMenu.Root>
```

### Простой Toggle (Light/Dark only)

```svelte
<script lang="ts">
  import { themeStore } from '$lib/stores/theme';
  import { Sun, Moon } from 'lucide-svelte';
  import { Button } from '$lib/components/ui/button';
</script>

<Button 
  variant="ghost" 
  size="icon"
  onclick={() => themeStore.toggle()}
>
  {#if themeStore.isDark}
    <Sun class="h-5 w-5" />
  {:else}
    <Moon class="h-5 w-5" />
  {/if}
  <span class="sr-only">Toggle theme</span>
</Button>
```

---

## Prevent Flash of Wrong Theme

### src/app.html

```html
<!doctype html>
<html lang="%lang%">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="theme-color" content="#0f172a" />
    
    <!-- Prevent flash of wrong theme -->
    <script>
      (function() {
        const theme = localStorage.getItem('aigateway_theme') || 'system';
        let resolved;
        
        if (theme === 'system') {
          resolved = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
        } else {
          resolved = theme;
        }
        
        if (resolved === 'dark') {
          document.documentElement.classList.add('dark');
        }
        
        // Update theme-color meta
        document.querySelector('meta[name="theme-color"]')?.setAttribute(
          'content', 
          resolved === 'dark' ? '#0f172a' : '#ffffff'
        );
      })();
    </script>
    
    %sveltekit.head%
  </head>
  <body data-sveltekit-preload-data="hover">
    <div style="display: contents">%sveltekit.body%</div>
  </body>
</html>
```

---

## shadcn-svelte Integration

shadcn-svelte автоматически поддерживает dark mode через class. Убедитесь что:

1. `darkMode: 'class'` в tailwind.config.ts
2. CSS переменные определены для обоих режимов
3. `.dark` класс добавляется на `<html>` элемент

---

## Цветовая палитра

### Light Theme

| Элемент | Цвет | Hex |
|---------|------|-----|
| Background | white | #ffffff |
| Foreground | slate-900 | #0f172a |
| Primary | emerald-500 | #10a37f |
| Secondary | slate-100 | #f1f5f9 |
| Muted | slate-500 | #64748b |
| Border | slate-200 | #e2e8f0 |
| Destructive | red-500 | #ef4444 |

### Dark Theme

| Элемент | Цвет | Hex |
|---------|------|-----|
| Background | slate-900 | #0f172a |
| Foreground | slate-50 | #f8fafc |
| Primary | emerald-500 | #10a37f |
| Secondary | slate-700 | #334155 |
| Muted | slate-400 | #94a3b8 |
| Border | slate-700 | #334155 |
| Destructive | red-500 | #ef4444 |

---

## Использование в компонентах

```svelte
<!-- Автоматически адаптируется -->
<div class="bg-background text-foreground">
  <h1 class="text-foreground">Title</h1>
  <p class="text-muted-foreground">Description</p>
  <button class="bg-primary text-primary-foreground">
    Action
  </button>
</div>

<!-- Явное переопределение для dark -->
<div class="bg-white dark:bg-slate-800">
  <span class="text-black dark:text-white">Text</span>
</div>
```

---

## Тестирование

### Playwright test

```typescript
test('theme toggle works', async ({ page }) => {
  await page.goto('/');
  
  // Check initial dark theme (system default or saved)
  const html = page.locator('html');
  
  // Click toggle
  await page.click('[data-testid="theme-toggle"]');
  
  // Verify class changed
  await expect(html).toHaveClass(/dark/);
  
  // Click again
  await page.click('[data-testid="theme-toggle"]');
  await expect(html).not.toHaveClass(/dark/);
});

test('theme persists across reload', async ({ page }) => {
  await page.goto('/');
  
  // Set dark theme
  await page.click('[data-testid="theme-toggle"]');
  
  // Reload
  await page.reload();
  
  // Should still be dark
  const html = page.locator('html');
  await expect(html).toHaveClass(/dark/);
});
```

---

## Чек-лист

- [ ] Настроить tailwind.config.ts с darkMode: 'class'
- [ ] Создать CSS переменные для обеих тем
- [ ] Создать ThemeStore с Svelte 5 Runes
- [ ] Создать ThemeToggle компонент
- [ ] Добавить inline script в app.html для предотвращения flash
- [ ] Добавить ThemeToggle в Navbar
- [ ] Проверить все компоненты в обеих темах
- [ ] Написать тесты

