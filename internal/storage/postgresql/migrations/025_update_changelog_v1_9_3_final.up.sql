
UPDATE changelogs 
SET content = '## [1.9.3] - 2025-10-14

### Added
- **MoniGo Performance Dashboard**: Real-time performance monitoring интеграция
  - MoniGo запускается на отдельном порту 9091 для изоляции
  - Quick Stats Cards с автоматическим обновлением каждые 5 секунд
  - Real-time метрики: CPU Usage, Memory Usage, Goroutines, System Health
  - Visual indicators (success/warning/critical) для метрик
  - API proxy для /admin/performance/monigo/api/v1/metrics
  - Direct link на Advanced Dashboard для полных возможностей MoniGo

- **NVIDIA GPU Monitoring**: Мониторинг GPU метрик через nvidia-smi
  - БЕЗ CGO зависимостей - использует nvidia-smi CLI напрямую
  - БЕЗ NVML headers - работает на любой системе с nvidia-smi
  - Поддержка нескольких GPU (multi-GPU configurations)
  - Единый компактный блок для всех GPU с gradient top border
  - Real-time метрики: Temperature, Power, GPU Load, VRAM, Clock, Fan Speed
  - Автообновление каждые 5 секунд
  - Hover эффект с подсветкой для каждой GPU строки
  - Цветовые индикаторы: Green (<70°C), Orange (70-80°C), -- Red (>80°C)
  - Graceful degradation если GPU не обнаружены или nvidia-smi недоступен
  - Platform-specific builds: Linux/macOS (full GPU support), -- Windows (stub)

- **Admin Panel Reorganization**: Улучшенная структура навигации
  - Новая вкладка Models с Available Models списком
  - Refresh Models кнопка для обновления списка моделей
  - System Tab переработан для Performance & GPU Monitoring

### Changed
- **System Tab**: Переработан полностью под мониторинг
  - Убраны Available Models (перенесены в Models Tab)
  - MoniGo Dashboard link вместо embedded iframe
  - Quick Stats Cards для основных метрик
  - NVIDIA GPU Metrics секция (если GPU доступны)

- **Models Tab**: Новая навигационная структура
  - Fix: Модели корректно загружаются при первом заходе

- **GPU Monitoring Architecture**:
  - Отказ от go-nvml (CGO зависимость) в пользу nvidia-smi CLI
  - Build tags для platform-specific реализаций
  - Windows: stub версия (GPU monitoring disabled)
  - Linux/macOS: полная функциональность через nvidia-smi

### Fixed
- **Models Tab Loading**: Исправлен баг с загрузкой моделей при первом заходе
- **MoniGo Integration**: Убран iframe, решены проблемы с CORS и static files
- **GPU Monitoring**: Убраны CGO compilation errors на Windows/Linux

### Technical
- **Backend**: internal/metrics/gpu_monitor_smi.go (Linux/macOS), gpu_monitor_windows.go (stub), internal/api/handlers/gpu.go для REST API
- **Frontend**: web/js/gpu-monitor.js - NVIDIA GPU metrics, unified GPU card дизайн, gradient top border, grid layout
- **Migration**: v25 для полного обновления changelog v1.9.3

### Security
- MoniGo dashboard доступен только через JWT authentication
- API proxy защищен Bearer token authentication
- GPU metrics endpoint требует аутентификацию'
WHERE version = '1.9.3';
	