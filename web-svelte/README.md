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
- 🔒 **Auth**: JWT токены с refresh
- 📱 **Responsive**: Mobile-first дизайн
- 💬 **Chat**: Streaming, Markdown, Code highlighting
- 🔑 **API Keys**: Personal и Tenant ключи
- 👥 **Tenants**: Управление организациями
- ⚙️ **Admin Panel**: Полное администрирование системы

## Страницы

### Public
- `/login` - Авторизация
- `/register` - Регистрация (с инвайтом)
- `/bootstrap` - Первоначальная настройка

### Protected
- `/dashboard` - Главная страница
- `/chat` - AI Чат с streaming
- `/api-keys` - Управление API ключами
- `/tenants` - Управление организациями
- `/profile` - Профиль и безопасность
- `/settings` - Личные настройки
- `/files` - Файловый менеджер
- `/rag` - RAG источники
- `/mcp` - MCP конфигурация
- `/downloads` - Загрузки моделей
- `/usage` - Статистика использования
- `/monitor` - Системный монитор
- `/about` - О системе

### Admin (`/admin/*`)
- `/admin` - Dashboard администратора
- `/admin/users` - Управление пользователями
- `/admin/invitations` - Инвайты
- `/admin/api-keys` - Все API ключи
- `/admin/models` - Модели
- `/admin/settings` - Настройки системы
- `/admin/backups` - Бэкапы
- `/admin/logs` - Логи

## Bundle Size

| Метрика | Значение |
|---------|----------|
| Файлов | 147 |
| Размер | 563 KB |

## Интеграция с Go backend

Dev-сервер проксирует API запросы на `http://localhost:8080`:

- `/api/*` → Go backend
- `/v1/*` → Go backend (OpenAI API)

Production build встраивается в Go binary через `embed.FS`.

## Переключение UI версий

В `configs/dev.yaml`:

```yaml
server:
  webui:
    enabled: true
    version: "svelte"  # или "legacy"
```

Или через PowerShell:

```powershell
# Собрать только Svelte UI
.\build.ps1 -WebUI svelte

# Собрать обе версии
.\build.ps1 -WebUI both
```

