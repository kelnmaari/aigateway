# CORE-02: Request/Response конвертеры

## Описание
Создание системы конвертации запросов и ответов между форматами OpenAI и Ollama API.

## Задачи
- Создать конвертеры в `internal/converter/`:
  - OpenAI → Ollama request converter
  - Ollama → OpenAI response converter
  - Streaming response converter
- Реализовать маппинг параметров:
  - model names
  - temperature, top_p и другие параметры
  - system/user messages
- Обработка специфичных OpenAI параметров

## Цель
Обеспечить seamless конвертацию между различными API форматами.

## Приоритет
**Высокий** - ключевая логика прокси

## Фаза
4 - Конвертеры запросов и ответов

## Оценка времени
6-8 часов
