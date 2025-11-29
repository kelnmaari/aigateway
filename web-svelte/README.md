# AIGateway WebUI (Svelte)

Новый веб-интерфейс AIGateway на Svelte 5 + SvelteKit.

## Технологии

- **Svelte 5** с Runes API
- **SvelteKit** со static adapter для SSG
- **Tailwind CSS** для стилизации
- **Paraglide JS** для i18n (en/ru)
- **TypeScript** для типизации

## Разработка

```bash
# Установка зависимостей
npm install

# Запуск dev-сервера
npm run dev

# Открыть http://localhost:5173
```

## Сборка

```bash
# Сборка для production
npm run build

# Файлы будут в ./build/
```

## Структура проекта

```
web-svelte/
├── messages/           # i18n translations
│   ├── en.json
│   └── ru.json
├── project.inlang/     # Paraglide config
├── src/
│   ├── app.css         # Global styles + theme
│   ├── app.html        # HTML template
│   ├── lib/
│   │   ├── api/        # API client
│   │   ├── components/ # UI components
│   │   ├── stores/     # Svelte stores
│   │   └── utils.ts    # Utilities
│   └── routes/         # SvelteKit routes
│       ├── login/      # Login page
│       └── ...
├── static/             # Static assets
└── package.json
```

## Фичи

- 🌓 **Темы**: Light / Dark (сохраняется в localStorage)
- 🌐 **i18n**: English / Русский
- 🔒 **Auth**: JWT токены
- 📱 **Responsive**: Mobile-first дизайн

## Интеграция с Go backend

Dev-сервер проксирует API запросы на `http://localhost:8080`:

- `/api/*` → Go backend
- `/v1/*` → Go backend (OpenAI API)

Для production сборка встраивается в Go binary через `embed.FS`.

