# API-01: HTTP сервер

## Описание
Создание основного HTTP сервера с базовыми middleware и graceful shutdown.

## Задачи
- Создать основной сервер в `cmd/server/main.go`
- Реализовать роутер в `internal/api/router/`
- Добавить базовый middleware (логирование, CORS, recovery)
- Реализовать graceful shutdown
- Добавить health check эндпоинт

## Цель
Запустить базовый HTTP сервер, готовый для добавления API эндпоинтов.

## Приоритет
**Высокий** - основа для всех API endpoints

## Фаза
2 - HTTP сервер и базовый роутинг

## Оценка времени
3-4 часа

