# Стратегия интернационализации (i18n)

## Выбранное решение: Paraglide JS v2.5.0

### Почему Paraglide JS

1. **Компиляция в функции** — переводы становятся tree-shakable функциями
2. **Типизация** — полная поддержка TypeScript с автодополнением
3. **Нет runtime overhead** — нет асинхронных загрузок
4. **Интеграция с Vite** — плагин с HMR поддержкой
5. **Малый размер бандла** — только используемые переводы

### Установка

```bash
npm install @inlang/paraglide-js@latest
```

### Конфигурация

#### vite.config.ts

```typescript
import { sveltekit } from '@sveltejs/kit/vite';
import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [
    sveltekit(),
    paraglideVitePlugin({
      project: './project.inlang',
      outdir: './src/lib/paraglide'
    })
  ]
});
```

#### project.inlang/settings.json

```json
{
  "baseLocale": "en",
  "locales": ["en", "ru"],
  "modules": [
    "https://cdn.jsdelivr.net/npm/@inlang/plugin-message-format@latest/dist/index.js"
  ],
  "plugin.inlang.messageFormat": {
    "pathPattern": "./messages/{locale}.json"
  }
}
```

## Структура файлов переводов

```
web-svelte/
├── messages/
│   ├── en.json          # Английские переводы
│   └── ru.json          # Русские переводы
├── project.inlang/
│   └── settings.json    # Конфигурация Paraglide
└── src/lib/paraglide/   # Генерируемые файлы (не редактировать!)
    ├── messages.js
    ├── runtime.js
    └── server.js
```

## Формат файлов переводов

### messages/en.json

```json
{
  "common_loading": "Loading...",
  "common_save": "Save",
  "auth_login": "Login",
  "auth_welcomeBack": "Welcome back",
  "nav_dashboard": "Dashboard",
  "error_invalidCredentials": "Invalid email or password"
}
```

### messages/ru.json

```json
{
  "common_loading": "Загрузка...",
  "common_save": "Сохранить",
  "auth_login": "Войти",
  "auth_welcomeBack": "С возвращением",
  "nav_dashboard": "Панель управления",
  "error_invalidCredentials": "Неверный email или пароль"
}
```

### Правила именования ключей

- **Формат**: `category_keyName` (snake_case + camelCase)
- **Категории**: `common`, `auth`, `nav`, `dashboard`, `chat`, `admin`, `error`, `success`, `theme`, `language`
- **Примеры**:
  - `common_save` — общие элементы
  - `auth_login` — авторизация
  - `nav_dashboard` — навигация
  - `error_notFound` — ошибки

### Параметры в сообщениях

```json
{
  "greeting": "Hello, {name}!",
  "items_count": "You have {count} items"
}
```

```typescript
import * as m from '$lib/paraglide/messages';

m.greeting({ name: 'John' }); // "Hello, John!"
m.items_count({ count: 5 });  // "You have 5 items"
```

## Использование в компонентах

### Импорт сообщений

```svelte
<script lang="ts">
  import * as m from '$lib/paraglide/messages';
</script>

<h1>{m.auth_welcomeBack()}</h1>
<button>{m.common_save()}</button>
```

### Управление локалью

```typescript
import { locales, getLocale, setLocale } from '$lib/paraglide/runtime';

// Получить текущую локаль
const current = getLocale(); // "en" | "ru"

// Установить локаль
setLocale('ru');

// Список доступных локалей
console.log(locales); // ["en", "ru"]
```

### Сохранение локали в localStorage

```typescript
const STORAGE_KEY = 'PARAGLIDE_LOCALE';

function toggleLocale() {
  const current = getLocale();
  const currentIndex = locales.indexOf(current);
  const nextIndex = (currentIndex + 1) % locales.length;
  const newLocale = locales[nextIndex];
  
  setLocale(newLocale);
  localStorage.setItem(STORAGE_KEY, newLocale);
  
  // Перезагрузка для применения новой локали
  window.location.reload();
}
```

## Генерация кода

При сборке Paraglide генерирует файлы в `src/lib/paraglide/`:

```
src/lib/paraglide/
├── messages/
│   ├── _index.js              # Экспорт всех сообщений
│   ├── auth_login.js          # Функция для auth_login
│   ├── common_save.js         # Функция для common_save
│   └── ...
├── messages.js                # Публичный API
├── runtime.js                 # Runtime функции (getLocale, setLocale, etc.)
├── registry.js                # Внутренний регистр
└── server.js                  # Server-side функции
```

### Пример сгенерированной функции

```javascript
// messages/auth_login.js
const en_auth_login = () => `Login`;
const ru_auth_login = () => `Войти`;

export const auth_login = (inputs = {}, options = {}) => {
  const locale = options.locale ?? getLocale();
  if (locale === "en") return en_auth_login(inputs);
  return ru_auth_login(inputs);
};
```

## Добавление новых переводов

1. **Добавьте ключ в `messages/en.json`**:
   ```json
   {
     "new_feature_title": "New Feature"
   }
   ```

2. **Добавьте перевод в `messages/ru.json`**:
   ```json
   {
     "new_feature_title": "Новая функция"
   }
   ```

3. **Перезапустите dev server** — Paraglide автоматически сгенерирует новые функции

4. **Используйте в коде**:
   ```svelte
   <h2>{m.new_feature_title()}</h2>
   ```

## Добавление новой локали

1. **Обновите `project.inlang/settings.json`**:
   ```json
   {
     "locales": ["en", "ru", "de"]
   }
   ```

2. **Создайте файл `messages/de.json`** с переводами

3. **Перезапустите сборку**

## Best Practices

### DO ✅

- Используйте осмысленные ключи (`auth_loginButton`, не `btn1`)
- Группируйте по категориям (`nav_`, `auth_`, `error_`)
- Храните все тексты в файлах переводов, не хардкодьте в компонентах
- Используйте параметры для динамических значений

### DON'T ❌

- Не редактируйте файлы в `src/lib/paraglide/` — они генерируются автоматически
- Не используйте пробелы в ключах
- Не смешивайте языки в одном файле переводов
- Не используйте HTML в переводах (используйте компоненты)

## Интеграция с VS Code

Установите расширение [Sherlock](https://marketplace.visualstudio.com/items?itemName=inlang.vs-code-extension) для:

- Автодополнения ключей переводов
- Подсветки отсутствующих переводов
- Inline редактирования переводов

## Миграция с других решений

### С svelte-i18n

```typescript
// Было (svelte-i18n)
import { t } from 'svelte-i18n';
$t('auth.login')

// Стало (Paraglide)
import * as m from '$lib/paraglide/messages';
m.auth_login()
```

### С i18next

```typescript
// Было (i18next)
i18next.t('auth:login')

// Стало (Paraglide)
import * as m from '$lib/paraglide/messages';
m.auth_login()
```

## Troubleshooting

### Ошибка "Module has no exported member"

- Убедитесь что ключ существует в файлах переводов
- Перезапустите dev server для перегенерации

### Переводы не обновляются

- Проверьте что vite plugin настроен правильно
- Очистите `.svelte-kit` директорию и пересоберите

### Локаль не сохраняется

- Проверьте что `localStorage` доступен
- Убедитесь что `PARAGLIDE_LOCALE` ключ используется
