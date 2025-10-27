# AUTH-03: Rate Limiting per API Key

## Описание
Реализация индивидуального rate limiting для каждого API ключа.

## Задачи
- Создать rate limiter в `internal/auth/ratelimit/`
- Реализовать token bucket algorithm per API key
- Интеграция с API Key Manager
- Конфигурируемые лимиты для каждого ключа
- Graceful handling rate limit exceeded
- Metrics для rate limiting

## Цель
Предоставить гранулярное управление нагрузкой на уровне отдельных API ключей.

## Приоритет
**Высокий** - важно для production deployment

## Фаза
8 - API Key Management System

## Оценка времени
5-6 часов

