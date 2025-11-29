# AIGateway WebUI Migration to Svelte 5

## 🎉 Миграция завершена! Версия 3.2.0

Веб-интерфейс AIGateway полностью мигрирован с HTML + Native JS на Svelte 5.

## Результат миграции

| Метрика | Legacy | Svelte |
|---------|--------|--------|
| Страниц | 26 HTML | 22 routes |
| JavaScript | 34 файлов | TypeScript modules |
| CSS | 7 файлов | Tailwind CSS |
| Bundle size | ~1.2 MB | **563 KB** |
| Типизация | Нет | Полная (TS) |
| i18n | Нет | ru/en |
| Темы | Частично | Light/Dark |

## Исходное состояние (legacy)

- **26 HTML страниц** 
- **34 JavaScript файлов**
- **7 CSS файлов**
- **Framework v3.1.0** (custom components: http, router, state)
- **Внешние зависимости**: HTMX, Marked.js, Highlight.js, Bootstrap (частично)

## Целевой стек

| Компонент | Версия | Назначение |
|-----------|--------|------------|
| **Svelte** | 5.37.0 | UI Framework (Runes API) |
| **SvelteKit** | 2.x | App Framework + Static Adapter |
| **@sveltejs/adapter-static** | 3.x | Статическая сборка |
| **shadcn-svelte** | 1.0.0-next.19 | UI компоненты |
| **Tailwind CSS** | 4.x | Стилизация |
| **TypeScript** | 5.x | Типизация |
| **bits-ui** | latest | Примитивы для shadcn |
| **lucide-svelte** | latest | Иконки |
| **@inlang/paraglide-js** | latest | i18n (ru/en) |

## Дополнительные фичи

- 🌓 **Темы:** Light / Dark (localStorage)
- 🌐 **i18n:** Russian / English (JSON формат)
- 🔄 **Version Switch:** Legacy / Svelte UI (флаг запуска)

## Структура документации

### Основные документы

1. **[01-current-architecture.md](docs/01-current-architecture.md)** — Анализ текущей архитектуры
2. **[02-feature-inventory.md](docs/02-feature-inventory.md)** — Полный инвентарь функциональности  
3. **[03-technology-stack.md](docs/03-technology-stack.md)** — Технологический стек
4. **[04-migration-roadmap.md](docs/04-migration-roadmap.md)** — Детальный план миграции
5. **[05-component-mapping.md](docs/05-component-mapping.md)** — Маппинг HTML → Svelte
6. **[06-api-layer.md](docs/06-api-layer.md)** — API интеграция
7. **[07-state-management.md](docs/07-state-management.md)** — Управление состоянием
8. **[08-embed-integration.md](docs/08-embed-integration.md)** — Интеграция с Go embed.FS
9. **[09-i18n-strategy.md](docs/09-i18n-strategy.md)** — Интернационализация (ru/en)
10. **[10-version-switching.md](docs/10-version-switching.md)** — Переключение Legacy/Svelte UI
11. **[11-theming.md](docs/11-theming.md)** — Система тем (Light/Dark)

### Документы по модулям

Каждый модуль/страница имеет отдельный файл с описанием функциональности:

- **[pages/auth.md](docs/pages/auth.md)** — Login, Register, Bootstrap
- **[pages/dashboard.md](docs/pages/dashboard.md)** — Главная панель
- **[pages/chat.md](docs/pages/chat.md)** — Chat интерфейс
- **[pages/admin.md](docs/pages/admin.md)** — Админ-панель
- **[pages/api-keys.md](docs/pages/api-keys.md)** — Управление API ключами
- **[pages/tenants.md](docs/pages/tenants.md)** — Организации
- **[pages/rag.md](docs/pages/rag.md)** — RAG система
- **[pages/mcp.md](docs/pages/mcp.md)** — MCP каталог
- **[pages/profile.md](docs/pages/profile.md)** — Профиль пользователя
- **[pages/monitor.md](docs/pages/monitor.md)** — Мониторинг системы

## Критические требования

### 1. Статическая сборка
- Вся сборка должна быть статической (SSG, не SSR)
- Выходные файлы размещаются в `/web` для embed.FS
- Клиентская маршрутизация через hash-based router

### 2. Совместимость с Go embed.FS
```go
//go:embed all:web/build
var StaticFiles embed.FS
```

### 3. Сохранение API контракта
- Все текущие API endpoints остаются неизменными
- JWT токены в localStorage
- Refresh token логика

## Roadmap (завершён)

| Фаза | Название | Статус |
|------|----------|--------|
| 0 | Подготовка инфраструктуры | ✅ Completed |
| 1 | Критический функционал (Auth, Dashboard, Chat) | ✅ Completed |
| 2 | Основной функционал (API Keys, Tenants, Profile) | ✅ Completed |
| 3 | Административный функционал (Admin Panel) | ✅ Completed |
| 4 | Дополнительный функционал (Files, RAG, MCP) | ✅ Completed |
| 5 | Финализация (Docs, Versioning) | ✅ Completed |

**Все фазы завершены!**

## Быстрый старт

### Разработка

```bash
cd web-svelte
npm install
npm run dev
# http://localhost:5173 (proxy → :8080)
```

### Production сборка

```powershell
# Из корня проекта
.\build.ps1 -WebUI svelte    # Только Svelte
.\build.ps1 -WebUI both      # Legacy + Svelte
.\build.ps1 frontend         # Только фронтенд без Go
```

### Переключение UI

В `configs/dev.yaml`:
```yaml
server:
  webui:
    version: "svelte"  # или "legacy"
```

## Решения по архитектуре

| Вопрос | Решение |
|--------|---------|
| Темы | Light + Dark, сохранение в localStorage |
| Компоненты | shadcn-svelte полностью |
| i18n | Paraglide JS (ru/en), JSON формат |
| PWA | Только для mobile (add to homescreen) |
| Version switch | Флаг `--webui-version=legacy\|svelte` |

## Контакты

- Документация поддерживается в каталоге `web-migration/`
- Задачи трекаются в отдельных markdown файлах

