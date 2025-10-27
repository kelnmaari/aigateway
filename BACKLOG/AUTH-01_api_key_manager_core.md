# AUTH-01: API Key Manager Core

## Описание
Создание основного менеджера API ключей с CRUD операциями и системой безопасности.

## Задачи
- Создать API Key Manager в `internal/auth/apikey/`
- Реализовать функции CRUD для API ключей:
  - CreateAPIKey с secure key generation
  - GetAPIKey с validation
  - UpdateAPIKey для изменения прав доступа
  - DeleteAPIKey и RevokeAPIKey
  - ListAPIKeys с фильтрацией
- Добавить хеширование ключей (bcrypt/argon2)
- Реализовать key expiration и rotation логику

## Цель
Создать безопасный и функциональный core для управления API ключами.

## Приоритет
**Критический** - основа всей системы аутентификации

## Фаза
8 - API Key Management System

## Оценка времени
6-8 часов

