# AUTH-02: Authentication Middleware

## Описание
Создание middleware для валидации API ключей и авторизации доступа к моделям.

## Задачи
- Создать auth middleware в `internal/auth/middleware/`
- Реализовать валидацию API ключей из заголовков
- Добавить model-based authorization
- Интегрировать middleware в HTTP router
- Обработка различных auth header форматов (Bearer, API-Key)
- Логирование auth событий

## Цель
Обеспечить надежную аутентификацию и авторизацию всех API запросов.

## Приоритет
**Критический** - необходимо для безопасности

## Фаза
8 - API Key Management System

## Оценка времени
4-5 часов

