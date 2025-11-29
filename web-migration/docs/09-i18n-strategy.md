# Стратегия интернационализации (i18n)

## Требования

- **Языки:** Russian (ru), English (en)
- **Хранение выбора:** localStorage браузера
- **Формат файлов:** Удобный для редактирования и просмотра
- **TypeScript:** Полная типизация ключей

---

## Анализ форматов i18n файлов

| Формат | Плюсы | Минусы | Оценка |
|--------|-------|--------|--------|
| **JSON** | Простой, стандартный, IDE поддержка | Нет комментариев | ⭐⭐⭐⭐⭐ |
| **YAML** | Читаемый, комментарии | Чувствителен к отступам | ⭐⭐⭐⭐ |
| **TypeScript** | Типизация из коробки | Сложнее для не-разработчиков | ⭐⭐⭐ |
| **PO/POT** | Стандарт индустрии, Poedit | Избыточен для 2 языков | ⭐⭐⭐ |
| **TOML** | Читаемый, секции | Менее популярен | ⭐⭐⭐ |

**Рекомендация:** JSON с вложенной структурой по модулям.

---

## Анализ библиотек для Svelte

### 1. typesafe-i18n
- **Репутация:** High
- **Формат:** TypeScript объекты
- **Типизация:** Полная, генерируется
- **Минус:** Файлы в TS, сложнее редактировать

### 2. Paraglide JS (Inlang) ⭐ РЕКОМЕНДУЕТСЯ
- **Репутация:** High
- **Формат:** JSON файлы
- **Типизация:** Полная, генерируется
- **Плюсы:**
  - Tree-shakable (только используемые переводы в бандле)
  - SSG support встроен
  - VSCode extension (i18n-ally совместим)
  - Современный подход
  - Хорошая документация для SvelteKit

### 3. sveltekit-i18n
- **Репутация:** Medium
- **Формат:** JSON
- **Типизация:** Частичная

---

## Выбор: Paraglide JS

### Установка

```bash
npm install @inlang/paraglide-js
```

### Структура файлов

```
web-svelte/
├── project.inlang/
│   └── settings.json
├── messages/
│   ├── en.json          # English translations
│   └── ru.json          # Russian translations
└── src/
    └── lib/
        └── paraglide/   # Generated (DO NOT EDIT)
            ├── messages.js
            ├── runtime.js
            └── server.js
```

### settings.json

```json
{
  "$schema": "https://inlang.com/schema/project-settings",
  "baseLocale": "en",
  "locales": ["en", "ru"],
  "modules": [
    "https://cdn.jsdelivr.net/npm/@inlang/message-lint-rule-empty-pattern@latest/dist/index.js",
    "https://cdn.jsdelivr.net/npm/@inlang/message-lint-rule-missing-translation@latest/dist/index.js",
    "https://cdn.jsdelivr.net/npm/@inlang/plugin-message-format@latest/dist/index.js",
    "https://cdn.jsdelivr.net/npm/@inlang/plugin-m-function-matcher@latest/dist/index.js"
  ],
  "plugin.inlang.messageFormat": {
    "pathPattern": "./messages/{locale}.json"
  }
}
```

### vite.config.ts

```typescript
import { sveltekit } from '@sveltejs/kit/vite';
import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [
    sveltekit(),
    paraglideVitePlugin({
      project: './project.inlang',
      outdir: './src/lib/paraglide',
    }),
  ],
});
```

---

## Формат JSON файлов

### messages/en.json

```json
{
  "common": {
    "loading": "Loading...",
    "save": "Save",
    "cancel": "Cancel",
    "delete": "Delete",
    "confirm": "Confirm",
    "error": "Error",
    "success": "Success"
  },
  "auth": {
    "login": "Login",
    "logout": "Logout",
    "register": "Register",
    "email": "Email",
    "password": "Password",
    "username": "Username",
    "rememberMe": "Remember me",
    "forgotPassword": "Forgot password?",
    "noAccount": "Don't have an account?",
    "hasAccount": "Already have an account?"
  },
  "nav": {
    "dashboard": "Dashboard",
    "chat": "Chat",
    "apiKeys": "API Keys",
    "tenants": "Organizations",
    "profile": "Profile",
    "admin": "Admin",
    "settings": "Settings"
  },
  "dashboard": {
    "title": "Dashboard",
    "conversations": "Conversations",
    "apiKeys": "API Keys",
    "models": "Models",
    "requests": "Requests"
  },
  "chat": {
    "newChat": "New Chat",
    "sendMessage": "Send message",
    "typeMessage": "Type your message...",
    "model": "Model",
    "parameters": "Parameters",
    "temperature": "Temperature",
    "maxTokens": "Max Tokens"
  },
  "admin": {
    "title": "Administration",
    "users": "Users",
    "rbac": "RBAC",
    "settings": "Settings",
    "backups": "Backups",
    "logs": "Logs"
  }
}
```

### messages/ru.json

```json
{
  "common": {
    "loading": "Загрузка...",
    "save": "Сохранить",
    "cancel": "Отмена",
    "delete": "Удалить",
    "confirm": "Подтвердить",
    "error": "Ошибка",
    "success": "Успешно"
  },
  "auth": {
    "login": "Войти",
    "logout": "Выйти",
    "register": "Регистрация",
    "email": "Email",
    "password": "Пароль",
    "username": "Имя пользователя",
    "rememberMe": "Запомнить меня",
    "forgotPassword": "Забыли пароль?",
    "noAccount": "Нет аккаунта?",
    "hasAccount": "Уже есть аккаунт?"
  },
  "nav": {
    "dashboard": "Главная",
    "chat": "Чат",
    "apiKeys": "API Ключи",
    "tenants": "Организации",
    "profile": "Профиль",
    "admin": "Администрирование",
    "settings": "Настройки"
  },
  "dashboard": {
    "title": "Главная панель",
    "conversations": "Диалоги",
    "apiKeys": "API Ключи",
    "models": "Модели",
    "requests": "Запросы"
  },
  "chat": {
    "newChat": "Новый чат",
    "sendMessage": "Отправить",
    "typeMessage": "Введите сообщение...",
    "model": "Модель",
    "parameters": "Параметры",
    "temperature": "Температура",
    "maxTokens": "Макс. токенов"
  },
  "admin": {
    "title": "Администрирование",
    "users": "Пользователи",
    "rbac": "Роли и права",
    "settings": "Настройки",
    "backups": "Резервные копии",
    "logs": "Логи"
  }
}
```

---

## Использование в компонентах

### Импорт сообщений

```svelte
<script lang="ts">
  import * as m from '$lib/paraglide/messages';
</script>

<h1>{m.dashboard_title()}</h1>
<button>{m.common_save()}</button>
```

### С параметрами

```json
{
  "greeting": "Hello, {name}!",
  "itemCount": "{count, plural, =0 {No items} =1 {1 item} other {# items}}"
}
```

```svelte
<p>{m.greeting({ name: user.name })}</p>
<p>{m.itemCount({ count: items.length })}</p>
```

---

## Переключение языка

### Svelte store для языка

```typescript
// src/lib/stores/locale.ts
import { browser } from '$app/environment';
import { setLocale, locale as paraglideLocale } from '$lib/paraglide/runtime';

const STORAGE_KEY = 'aigateway_locale';
const DEFAULT_LOCALE = 'en';

export function initLocale() {
  if (browser) {
    const saved = localStorage.getItem(STORAGE_KEY);
    const locale = saved || navigator.language.split('-')[0] || DEFAULT_LOCALE;
    
    if (['en', 'ru'].includes(locale)) {
      setLocale(locale as 'en' | 'ru');
    } else {
      setLocale(DEFAULT_LOCALE);
    }
  }
}

export function changeLocale(newLocale: 'en' | 'ru') {
  setLocale(newLocale);
  if (browser) {
    localStorage.setItem(STORAGE_KEY, newLocale);
  }
}

export { paraglideLocale as locale };
```

### Компонент переключателя

```svelte
<script lang="ts">
  import { locale, changeLocale } from '$lib/stores/locale';
  import { Globe } from 'lucide-svelte';
  import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
  
  const languages = [
    { code: 'en', label: 'English', flag: '🇬🇧' },
    { code: 'ru', label: 'Русский', flag: '🇷🇺' },
  ] as const;
</script>

<DropdownMenu.Root>
  <DropdownMenu.Trigger class="btn btn-ghost">
    <Globe class="h-5 w-5" />
  </DropdownMenu.Trigger>
  <DropdownMenu.Content>
    {#each languages as lang}
      <DropdownMenu.Item 
        onclick={() => changeLocale(lang.code)}
        class:active={$locale === lang.code}
      >
        <span>{lang.flag}</span>
        <span>{lang.label}</span>
      </DropdownMenu.Item>
    {/each}
  </DropdownMenu.Content>
</DropdownMenu.Root>
```

---

## Инструменты для редактирования

### VSCode Extensions

1. **i18n Ally** - Визуализация переводов прямо в коде
   - Показывает переводы inline
   - Подсветка отсутствующих ключей
   - Автодополнение

2. **JSON Tools** - Форматирование JSON

### Online редакторы

- **Inlang Fink** - Веб-редактор от создателей Paraglide
- **POEditor** - Профессиональный редактор (поддерживает JSON)
- **Localize** - Онлайн платформа

### CLI инструменты

```bash
# Проверка отсутствующих переводов
npx @inlang/cli lint

# Машинный перевод (опционально)
npx @inlang/cli machine translate --from en --to ru
```

---

## Миграция существующих текстов

### Шаги

1. **Извлечь все тексты** из HTML/JS файлов
2. **Структурировать** по модулям (auth, nav, dashboard, chat, admin, etc.)
3. **Создать en.json** с базовыми ключами
4. **Перевести ru.json**
5. **Заменить** хардкод в Svelte компонентах на вызовы m.*()

### Пример миграции

**До (HTML):**
```html
<button class="btn btn-primary">Create API Key</button>
<span class="text-muted">No API keys found</span>
```

**После (Svelte):**
```svelte
<script>
  import * as m from '$lib/paraglide/messages';
</script>

<Button>{m.apiKeys_create()}</Button>
<span class="text-muted">{m.apiKeys_empty()}</span>
```

---

## Структура ключей по модулям

| Модуль | Префикс | Пример |
|--------|---------|--------|
| Общие | `common_` | `common_save`, `common_cancel` |
| Навигация | `nav_` | `nav_dashboard`, `nav_chat` |
| Авторизация | `auth_` | `auth_login`, `auth_password` |
| Dashboard | `dashboard_` | `dashboard_title` |
| Chat | `chat_` | `chat_newChat`, `chat_send` |
| API Keys | `apiKeys_` | `apiKeys_create` |
| Tenants | `tenants_` | `tenants_members` |
| Admin | `admin_` | `admin_users`, `admin_rbac` |
| Profile | `profile_` | `profile_settings` |
| Errors | `error_` | `error_notFound` |
| Success | `success_` | `success_saved` |

---

## Чек-лист i18n

- [ ] Установить Paraglide JS
- [ ] Создать project.inlang/settings.json
- [ ] Создать messages/en.json с базовой структурой
- [ ] Создать messages/ru.json с переводами
- [ ] Настроить vite.config.ts
- [ ] Создать locale store
- [ ] Добавить LanguageSwitcher компонент
- [ ] Заменить все хардкод тексты на m.*()
- [ ] Установить i18n Ally в VSCode

