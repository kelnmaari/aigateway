# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.9.3] - 2025-10-14

### Added

- **MoniGo Performance Dashboard**: Real-time performance monitoring интеграция
  - MoniGo запускается на отдельном порту 9091 для изоляции
  - Quick Stats Cards с автоматическим обновлением каждые 5 секунд
  - Real-time метрики: CPU Usage, Memory Usage, Goroutines, System Health
  - Visual indicators (success/warning/critical) для метрик
  - API proxy для `/admin/performance/monigo/api/v1/metrics`
  - Direct link на Advanced Dashboard для полных возможностей MoniGo

- **NVIDIA GPU Monitoring**: Мониторинг GPU метрик через nvidia-smi
  - **БЕЗ CGO зависимостей** - использует `nvidia-smi` CLI напрямую
  - **БЕЗ NVML headers** - работает на любой системе с nvidia-smi
  - Поддержка нескольких GPU (multi-GPU configurations)
  - Единый компактный блок для всех GPU с gradient top border
  - Real-time метрики: Temperature, Power, GPU Load, VRAM, Clock, Fan Speed
  - Автообновление каждые 5 секунд
  - Hover эффект с подсветкой для каждой GPU строки
  - Цветовые индикаторы: Green (<70°C), Orange (70-80°C), Red (>80°C)
  - Graceful degradation если GPU не обнаружены или nvidia-smi недоступен
  - Platform-specific builds: Linux/macOS (full GPU support), Windows (stub)

- **Admin Panel Reorganization**: Улучшенная структура навигации
  - Новая вкладка "Models" с Available Models списком
  - Refresh Models кнопка для обновления списка моделей
  - System Tab переработан для Performance & GPU Monitoring
  - Logs Tab отделен от System (уже существовал)
  - Models Tab отделен от System

### Changed

- **System Tab**: Переработан полностью под мониторинг
  - Убраны "Available Models" (перенесены в Models Tab)
  - Убраны "System Logs" (остались в Logs Tab)
  - MoniGo Dashboard link вместо embedded iframe
  - Quick Stats Cards для основных метрик
  - NVIDIA GPU Metrics секция (если GPU доступны)

- **Models Tab**: Новая навигационная структура
  - Перенесены Available Models из System Tab
  - Добавлена кнопка Refresh для обновления списка
  - Section header с красивым дизайном
  - Fix: Модели корректно загружаются при первом заходе

- **GPU Monitoring Architecture**:
  - Отказ от `go-nvml` (CGO зависимость) в пользу nvidia-smi CLI
  - Build tags для platform-specific реализаций
  - Windows: stub версия (GPU monitoring disabled)
  - Linux/macOS: полная функциональность через nvidia-smi

### Fixed

- **Models Tab Loading**: Исправлен баг с загрузкой моделей при первом заходе
- **MoniGo Integration**: Убран iframe, решены проблемы с CORS и static files
- **GPU Monitoring**: Убраны CGO compilation errors на Windows/Linux

### Technical

- **Backend (Go)**:
  - `internal/metrics/gpu_monitor_smi.go` - GPU monitoring через nvidia-smi (Linux/macOS)
  - `internal/metrics/gpu_monitor_windows.go` - Stub для Windows
  - `internal/metrics/custom_monigo.go` для custom MoniGo metrics
  - `internal/api/handlers/gpu.go` - REST API handler для `/api/gpu/metrics`
  - MoniGo запускается на порту 9091 в отдельном HTTP сервере
  - GPU Monitor инициализация в `cmd/server/main.go`
  - Reverse proxy для MoniGo API endpoints в `internal/api/router/router.go`
  - Зависимость: `github.com/iyashjayesh/monigo v1.1.0`

- **Frontend (JavaScript)**:
  - `web/js/performance.js` - Real-time performance metrics от MoniGo
  - `web/js/gpu-monitor.js` - NVIDIA GPU metrics визуализация
  - Unified GPU card дизайн с табличным layout для нескольких GPU
  - Gradient top border на карточке (цвет зависит от max температуры)
  - Grid layout: Name, Temp, Power, Clock, Fan | GPU Load & VRAM bars
  - Auto-refresh каждые 5 секунд для актуальных данных
  - CSS animations и hover эффекты

- **Database Migration**:
  - Migration v24: `add_changelog_v1_9_3` для системной истории изменений
  - Автоматическое применение при старте сервера

### Security

- MoniGo dashboard доступен только через JWT authentication Admin Panel
- API proxy для метрик защищен Bearer token authentication
- GPU metrics endpoint требует аутентификацию

### Notes

- MoniGo собирает метрики автоматически через Gin middleware
- Custom metrics (Ollama latency, API keys) подготовлены для future versions
- GPU monitoring работает только в Linux/macOS, Windows использует stub
- Dashboard доступен только для admin users с валидным JWT
