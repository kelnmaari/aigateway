
INSERT INTO changelogs (version, release_date, content) VALUES
('2.2.0', '2025-10-28', '## [2.2.0] - 2025-10-28

### Added

- **Invitation-Only Registration System (AUTH-03)**: Полная система управления приглашениями для контролируемой регистрации пользователей
  - Database schema с таблицей invitations (migration v57 для SQLite, v4 для PostgreSQL)
  - REST API endpoints для создания, просмотра, отзыва и валидации приглашений
  - Admin WebUI: /admin-invitations.html - страница управления приглашениями с фильтрацией и статистикой
  - Поддержка ограничений: по email, сроку действия, количеству использований
  - Детальная информация о пользователях: кто создал, кто использовал, кто отозвал приглашение
  - Регистрация по приглашению: обновлен /register.html с поддержкой invitation tokens

- **Invitation Management Features**:
  - Создание приглашений с настраиваемыми параметрами (email restriction, expiry, max uses)
  - Автоматическая генерация уникальных invitation links
  - Статистика приглашений (Active, Pending, Used, Expired, Revoked)
  - Фильтрация по статусу, email, создателю
  - View modal с полной информацией включая user details через LEFT JOIN
  - One-click копирование invitation links и tokens

- **Configuration Options**: Новые настройки в configs/dev.yaml
  - auth.registration.mode: "open" | "invitation_only" | "disabled"
  - auth.invitations.enabled: включение системы приглашений
  - auth.invitations.default_expiry_days: срок действия по умолчанию
  - auth.invitations.max_uses_default: количество использований
  - Rate limits для создания и валидации приглашений

### Changed

- **WebUI Improvements**:
  - Улучшен контраст текста в статистических карточках (белый текст на цветных градиентах)
  - Markdown форматирование для пользовательских сообщений в ChatUI
  - WYSIWYG-подобная панель форматирования текста при выделении (bold, italic, code, lists)
  - Сохранение переносов строк в сообщениях чата (white-space: pre-wrap)

- **Registration Flow**: Обновлен процесс регистрации с проверкой invitation tokens
  - Валидация токена перед показом формы регистрации
  - Автоматическое использование приглашения после успешной регистрации
  - Email restriction check для приглашений привязанных к конкретному email

### Fixed

- **Database Schema**: Исправлена ошибка с колонкой display_name → full_name в запросах с JOIN к таблице users
- **Invitation Links**: Исправлена генерация ссылок - добавлено .html расширение (/register.html?invite=...)

### Technical

- **Backend (Go)**:
  - Новые модели: Invitation, InvitationWithUsers, UserInfo, InvitationStatus
  - Storage layer: полная реализация CRUD операций для SQLite и PostgreSQL
  - Handler: InvitationHandler с 6 endpoint''ами (create, list, stats, details, revoke, validate)
  - AuthService: интеграция invitation token validation в процесс регистрации
  - Transaction delegation: добавлены методы в sqliteTx и postgresqlTx

- **Database Migrations**:
  - SQLite migration v57: создание таблицы invitations с индексами
  - PostgreSQL migration v4: аналогичная схема для PostgreSQL
  - LEFT JOIN queries для получения информации о пользователях

- **API Endpoints**:
  - POST /api/admin/invitations - создание приглашения
  - GET /api/admin/invitations - список приглашений с фильтрацией
  - GET /api/admin/invitations/stats - статистика
  - GET /api/admin/invitations/:id - детальная информация с user info
  - DELETE /api/admin/invitations/:id - отзыв приглашения
  - GET /api/invitations/:token/validate - публичная валидация токена

- **Frontend**:
  - web/admin-invitations.html (623 строки) - полнофункциональная админ-панель
  - web/js/api.js - 6 новых методов для работы с invitations API
  - web/register.html - поддержка ?invite= query parameter
  - Formatting toolbar для ChatUI с keyboard shortcuts (Ctrl+B, Ctrl+I, Ctrl+K, Ctrl+L)

### Security

- **Access Control**: Все admin endpoints защищены JWT authentication + RequireAdmin middleware
- **Rate Limiting**: Настраиваемые лимиты для создания приглашений и валидации токенов
- **Token Security**: UUID v4 tokens для приглашений, проверка валидности перед использованием
- **Email Verification**: Опциональная привязка приглашения к конкретному email')
ON CONFLICT (version) DO UPDATE SET
    release_date = EXCLUDED.release_date,
    content = EXCLUDED.content;
	