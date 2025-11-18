## План миграции WebUI на Svelte с встраиванием в Go-бинарь

### Цель
- Заменить текущий MPA WebUI на SPA, построенный на Svelte (Vite).
- Собрать SPA в статический дистрибутив и внедрить его в Go-бинарь через `embed.FS`.
- Сохранить возможность запускать backend как единый binary, при этом подготовить опцию упаковки SPA в Docker.

### Ограничения и допущения
- Backend остаётся на Go 1.25, API контракты фиксируются в OpenAPI.
- Auth продолжает использовать текущие JWT cookies; CORS не требуется, т.к. SPA будет сервиться тем же origin.
- Build-пайплайн расширяется существующим `build.ps1`, без внедрения CI/CD пока.
- Нужна полная функциональная эквивалентность текущего WebUI (dashboard, admin, settings, tenants, models, auth).

### Оценка трудозатрат
- Человеко-часы указаны для всей команды (1 фронт, 1 бек). 
- Колонка «Я» — сколько займёт при выполнении лично мной.

| Этап | Описание | Часы команды | Часы меня |
|------|----------|--------------|-----------|
|0|Аналитика и дизайн|32|24|
|1|Инфраструктура Svelte/Vite|24|18|
|2|Embed и Go-интеграция|20|14|
|3|Базовый каркас SPA (routing, store, auth)|36|28|
|4|Миграция Dashboard+Admin|80|60|
|5|Миграция Settings/Models/Tenants|96|72|
|6|UX/QA/Accessibility|32|24|
|7|Документация и релиз|16|12|

### Детальный план

#### Этап 0. Аналитика и дизайн (32ч / 24ч)
1. Инвентаризация текущих страниц, компонентов, API-вызовов, состояний загрузки.
2. Формирование OpenAPI-спецификации или TypeScript SDK для строгой типизации.
3. Решение по state-management (Svelte store, возможно svelte-query).
4. Дизайн схемы маршрутов и guard’ов (auth, admin-only).

#### Этап 1. Инфраструктура Svelte/Vite (24ч / 18ч)
1. Создание `web/svelte` с Vite + Svelte + TypeScript + ESLint/Prettier.
2. Настройка alias’ов, postcss, dark theme, глобальных переменных стиля.
3. Добавление библиотек (svelte-routing, form libs, toast, skeleton).
4. Настройка `package.json` скриптов (`dev`, `build`, `preview`).

#### Этап 2. Embed и Go-интеграция (20ч / 14ч)
1. Расширение `build.ps1`: шаг `npm ci && npm run build`, копирование `dist` в `internal/web/svelte_dist`.
2. Добавление `//go:embed internal/web/svelte_dist/*` + HTTP handler с cache-control и gzip/brotli.
3. Интеграция с существующим framework handler, синхронизация маршрутов `/`, `/dashboard`, `/admin`.
4. Проверка работы auth middleware + SPA history fallback.

#### Этап 3. Каркас SPA (36ч / 28ч)
1. Создание layout’ов (AuthLayout, AppLayout) с navbar/sidebar.
2. Реализация глобальных сторов: user, settings, notifications, loading.
3. Имплементация auth guard (token refresh, redirect на login).
4. Базовые UI-компоненты: кнопки, карточки, формы, таблицы, skeleton.

#### Этап 4. Миграция Dashboard и Admin (80ч / 60ч)
1. Dashboard: stats widgets, models, recent conversations, tenants, skeleton + error boundary.
2. Admin panel: summary, user/tenant/api-key списки, действия, модальные окна.
3. Перенос всех AJAX эндпоинтов, caching, polling.
4. Тестирование на feature parity и производительность.

#### Этап 5. Settings / Models / Tenants / Chat (96ч / 72ч)
1. Settings: категории, inline-редактирование, optimistic updates, diff view.
2. Models: загрузка/удаление, status badges, интеграция с backend hooks.
3. Tenants: карточки организаций, управление членами, ролями, приглашениями.
4. Chat / Conversations: история, live updates, streaming, attachments.

#### Этап 6. UX, QA, Accessibility (32ч / 24ч)
1. Полный регрессионный прогон, сквозные сценарии.
2. Lighthouse/axe accessibility, responsive layout, keyboard navigation.
3. Финальный ESLint/TypeCheck, e2e тесты (Playwright).

#### Этап 7. Документация и релиз (16ч / 12ч)
1. Обновление `README`, `FRAMEWORK_OPTIMIZATION.md`, новых инструкций деплоя.
2. Сценарии отката, релизные заметки, changelog + миграции.
3. Финальная сборка, smoke тест, передача артефактов.

### Риски и смягчающие меры
- **Scope creep**: зафиксировать MVP: dashboard, admin, settings, chat. Доп. страницы — последующие релизы.
- **Auth/CORS проблемы**: оставить SPA на том же домене, проверять cookie и CSRF в dev.
- **Перфоманс**: включить code-splitting, prefetch, использовать已有 cache layer API.
- **Команда/время**: оценка ≈ 336 ч команды (2–3 месяца одиночной работы). Приоритетно выделить фронт-инженера.

### Следующие шаги
1. Утвердить бюджет и расписание.
2. Создать `web/svelte` scaffold, подключить к build.ps1.
3. Импортировать OpenAPI и начать реализацию Этапа 0–1.

