# API-05: API Key Management API

## Описание
Создание admin API для управления API ключами через HTTP endpoints.

## Задачи
- Создать admin API endpoints для управления ключами:
  - `POST /admin/api-keys` - создание ключа
  - `GET /admin/api-keys` - список ключей
  - `GET /admin/api-keys/{id}` - детали ключа
  - `PUT /admin/api-keys/{id}` - обновление ключа
  - `DELETE /admin/api-keys/{id}` - удаление ключа
- Admin authentication (отдельный admin API key)
- Валидация входных данных
- Error handling и OpenAPI spec

## Цель
Предоставить REST API для программного управления API ключами.

## Приоритет
**Средний** - полезно для автоматизации

## Фаза
8 - API Key Management System

## Оценка времени
4-5 часов
