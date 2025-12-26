INSERT INTO changelogs (version, release_date, content) VALUES
('3.3.0', '2025-12-08', '## [3.3.0] - 2025-12-08

### Added
- **Multi-Provider Inference System (v4)**: Полная система инференса с поддержкой нескольких провайдеров
  - vLLM: высокопроизводительный инференс для HuggingFace моделей
  - SGLang: поддержка Vision Language Models (Qwen2-VL, LLaVA)
  - TGI: Text Generation Inference от HuggingFace
  - llama.cpp: GGUF модели с GPU offload
  - TensorRT-LLM: конверсия и кеширование TRT engines

- **Model Orchestrator**: управление жизненным циклом контейнеров
  - Автоматическое скачивание моделей (HF/GGUF/HTTP)
  - Resume downloads с exponential backoff retry
  - SHA256 validation для GGUF файлов
  - Health check с таймаутами
  - Pin/unpin для защиты моделей от auto-evict

- **TensorRT-LLM Converter**: конверсия HF моделей в TRT engines
  - Кеширование engines в /data/engines/trt
  - Валидация совместимости CUDA/TRT/Driver/SM версий
  - Автоматическая реконверсия при несовпадении версий

- **API Endpoints** (`/api/system/inference/*`):
  - load, prepare, stop, evict, pin/unpin
  - delete-artifacts, evict-cache, cache
  - health, models, logs, metrics
  - convert-trt, trt-engines, delete-trt-engine

- **OpenAI Proxy** (`/v1/inference/*`):
  - chat/completions с streaming
  - Routing по alias и capability (text/vision)

- **Prometheus Metrics**:
  - inference_startup_failures_total
  - inference_health_failures_total
  - inference_containers_started_total

### Technical
- internal/inference/ package с 10+ модулями
- MockContainerRuntime для интеграционных тестов
- 28 unit/integration tests
- Benchmark tests: 4700+ req/s throughput

### Documentation
- docs/LLM_INFRA_ARCH.md - архитектура системы
- docs/INFERENCE_HOWTO.md - руководство по провайдерам
- docs/INFERENCE_TROUBLESHOOTING.md - диагностика проблем')
ON CONFLICT (version) DO UPDATE SET
  release_date = EXCLUDED.release_date,
  content = EXCLUDED.content;

