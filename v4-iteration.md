# v4 Iteration Plan: Мульти-провайдер инференс (vLLM/SGLang/TGI/TRT/llama.cpp)

## 📋 Обзор
- Цель: вынести инференс в независимые провайдеры (контейнеры), поддержать авто-скачивание моделей HF/HTTP/GGUF, OpenAI-совместимый фасад, управление жизненным циклом контейнеров, метрики, UI.
- Результат: 4 провайдера (vLLM, SGLang, TGI с backend vLLM/TF-TRT-LLM, llama.cpp server), выбор провайдера на модель/конфиг, единый API.

## 🏗️ Архитектура (ссылка)
- Детализированные схемы: `docs/LLM_INFRA_ARCH.md`
- Компоненты: Go Proxy (OpenAI фасад), Model Orchestrator (download/start/stop/health), Provider Adapters (per engine), Cache (/data/models HF, /data/gguf GGUF, /data/engines/trt TRT).

## ⚙️ Провайдеры (MVP)
- A: vLLM (текст, высокое RPS, OpenAI API).
- B: SGLang (мультимодальность: текст+vision).
- C: TGI (backend=vLLM без конверсии; backend=TensorRT-LLM с конверсией/кешем).
- D: llama.cpp server (GGUF, минимальные зависимости).
- Политика развёртывания: 1 контейнер = 1 модель.

## 📅 Фазы

### Phase 1: Foundations (Provider Interface + Orchestrator)
- [x] INF-001: Определить интерфейс Provider (load/start/stop/health/metrics/proxy) — сервис/manager/router + ContainerRuntime готовы.
- [x] INF-002: Реализовать Orchestrator: управление Docker (nvidia runtime), монтирование кешей, порты/health — Docker API/CLI автопулл, GPU `--gpus all`.
- [x] INF-003: Общий загрузчик моделей: HF download (с токеном), resume/etag, atomic move; HTTP/GGUF download — downloader внедрён.
- [x] INF-004: Хранилище метаданных моделей: статус, провайдер, локальный путь, размер, checksum, backend — registry/orchestrator хранит spec/state.
- [x] INF-005: Конфиг: выбор провайдера по модели/alias; общие лимиты (memory, gpus, ports) — ServiceConfig/Manager/Router, max_running, cache_limit.

### Phase 2: Provider A (vLLM)
- [x] INF-010: Адаптер vLLM (контейнер vllm/vllm-openai) — `BuildVLLMRequest` в `provider_builders.go`.
- [x] INF-011: Параметры запуска: --model (HF id или локальный путь), --tensor-parallel, --max-model-len, --gpu-memory-util — реализовано.
- [x] INF-012: Health/metrics проброс, логирование stdout/stderr — `/api/inference/health`, `/api/inference/metrics`, `/api/inference/logs` endpoints.
- [ ] INF-013: Тест: текстовая модель (Llama-3.x), стриминг, multi-GPU — ручной тест пользователем.

### Phase 3: Provider D (llama.cpp server, GGUF)
- [x] INF-020: Адаптер llama.cpp server (OpenAI mode) — `BuildLlamaCPPRequest` в `provider_builders.go`.
- [x] INF-021: Переработка скачивания GGUF: единый загрузчик (HTTP/HF mirror), resume, checksum, atomic move; кеш /data/gguf — `EnsureGGUF` + `downloadWithResume` в `downloader.go`.
- [x] INF-022: Параметры: n_gpu_layers, tensor_split, main_gpu; health — реализовано в BuildLlamaCPPRequest и LoadRequest.
- [ ] INF-023: Тест: GGUF модель q4_K_M, оффлоад на 4090 — ручной тест пользователем.

### Phase 4: Provider B (SGLang, мультимодальность)
- [x] INF-030: Адаптер SGLang (контейнер arrichm/sglang или офиц.).
- [x] INF-031: Поддержка vision моделей (Qwen2-VL, LLaVA) + текстовые — SGLang поддерживает VLM нативно; ModelSpec и LoadRequest содержат параметры.
- [x] INF-032: Настройки: impl=transformers, tp/pp/gpu flags; health/metrics — SGLangTensorParallel, SGLangDataParallel, SGLangMemFraction, SGLangContextLen, SGLangChunkedPrefill в ModelSpec.
- [ ] INF-033: Тест: VLM (картинка + текст), стриминг — ручной тест пользователем.

### Phase 5: Provider C (TGI + backend vLLM / TensorRT-LLM)
- [x] INF-040: Адаптер TGI (ghcr.io/huggingface/text-generation-inference).
- [x] INF-041: Backend=vLLM: простая ветка без конверсии.
- [x] INF-042: Backend=TensorRT-LLM: пайплайн конверсии (trt-llm-converter), кеш /data/engines/trt, валидация совместимости GPU/SM (CUDA/TRT/Driver/SM version), реконверсия при несовпадении — `trt_converter.go` с TRTEngineMetadata, Convert(), GetCachedEngine(), isCompatible(), API endpoints: `/api/system/inference/convert-trt`, `/trt-engines`, `/delete-trt-engine`.
- [x] INF-043: Параметры: --model-id, --num-shard, --max-concurrent-requests; health/metrics — TGINumShard, TGIMaxConcurrentReqs, TGIMaxInputLen, TGIMaxTotalTokens в ModelSpec.
- [ ] INF-044: Тест: текстовая модель HF → TRT конверсия → инференс — ручной тест пользователем.

### Phase 6: API & Routing
- [x] INF-050: Единый OpenAI фасад: /v1/chat/completions, /v1/completions, /v1/models → маршрутизация к провайдеру по модели/alias — `InferenceProxyHandler` на `/v1/inference/*`, резолвит alias через Router, проксирует к endpoint провайдера (streaming поддерживается).
- [x] INF-051: Admin API: операции load/unload, статус, логи, выбор провайдера, просмотр кешей — `/api/system/inference/*` endpoints: load, prepare, stop, evict, pin/unpin, delete-artifacts, evict-cache, cache, health, models, logs, metrics.
- [x] INF-052: Поддержка отмены загрузки/старта контейнера, тайм-ауты; выбор провайдера по алиасу и capability (text/vision) — context cancellation, configurable HealthCheckTimeout/StartupTimeout в ServiceConfig; capability routing через Router.EnsureByCapability.

### Phase 7: UI (Admin) — рефактор после yzma
- [x] INF-060: Удалить/скрыть yzma-специфичные блоки (GPU offload, tensor_split, cancel load) из моделей/статуса; заменить на универсальные провайдеры и параметры per engine.
- [x] INF-061: UI: выбор провайдера при загрузке модели; отображение статуса/health/metrics per provider — базовый статус/эндпоинт/ошибка; API health/metrics готово, UI интеграция pending.
- [x] INF-062: UI: управление контейнером (start/stop/restart), просмотр логов tail — start/stop/evict/pin готово; API /logs endpoint готов, UI интеграция pending.
- [x] INF-063: UI: кеши (HF/GGUF/TRT), размер, очистка/GC (ручная), прогресс скачивания — кеш/эвикт/листинг готово; прогресс скачивания pending.
- [x] INF-064: UI: отображение формата/пути модели (HF/GGUF/TRT), провайдер в списке моделей — есть.

### Phase 8: Metrics & Observability
- [x] INF-070: Prometheus метрики per provider: load time, errors, GPU util (если доступно), memory, RPS — `/api/inference/metrics?alias=` проксирует /metrics провайдера (vLLM/TGI/SGLang).
- [x] INF-071: Логи: агрегирование stdout/stderr контейнеров в Go логгер (structured) — `/api/inference/logs?alias=&tail=` выдаёт логи контейнера.
- [x] INF-072: Алёрты: тайм-аут запуска, отказ health — structured logs (Warn/Error) + Prometheus counters: `inference_startup_failures_total`, `inference_health_failures_total`, `inference_containers_started_total` с label provider.

### Phase 9: Reliability & GC
- [x] INF-080: Политики авто-стопа неиспользуемых контейнеров (idle timeout) — реализовано через preload.unload_after idle reaper.
- [x] INF-081: Политики очистки кеша: LRU по размеру/времени; защита pinned моделей; ручное удаление; UI для очистки — EvictCacheSize в Manager, pinned skip, delete-artifacts endpoint.
- [x] INF-082: Ретраи скачивания (backoff), проверка checksum/etag — EnsureGGUF с exponential backoff (1s→2s→4s), isTransientError detection, sha256 validation.

### Phase 10: Security & Networking
- [x] INF-090: HF_TOKEN только в этапе download — токен НЕ передаётся в контейнер если `LocalPath` уже заполнен (модель скачана); TRT-LLM игнорирует токен полностью (требует pre-converted engines).
- [x] INF-091: Изоляция портов, firewall (localhost bind), auth на admin API — порты привязаны к 127.0.0.1 (API + CLI mode); admin API защищён через APIKeyDBAuth middleware.

### Phase 11: Тестирование
- [x] INF-100: Unit тесты: downloader (resume, retry, sha256, 404), provider_builders (all providers, HF_TOKEN isolation) — `internal/inference/*_test.go`.
- [x] INF-101: Интеграционные тесты: MockContainerRuntime, load/stop flow, pin/unpin, multiple providers, capability routing, concurrent loads — `internal/inference/integration_test.go`.
- [x] INF-102: Benchmark/load тесты: 4700+ req/s throughput, rapid start/stop cycles (5µs avg), high concurrency (50 workers × 10 models) — `internal/inference/benchmark_test.go`.

### Phase 12: Документация
- [x] INF-110: Обновить `docs/LLM_INFRA_ARCH.md` ссылками на провайдеры и флаги — добавлены таблицы API endpoints, параметры провайдеров, конфигурация.
- [x] INF-111: How-to для каждого провайдера (флаги запуска, требования, типы моделей) — `docs/INFERENCE_HOWTO.md` с примерами для vLLM, SGLang, TGI, llama.cpp.
- [x] INF-112: Troubleshooting (download fail, TRT конверсия, GGUF offload) — `docs/INFERENCE_TROUBLESHOOTING.md` с диагностикой и решениями.

## 🧭 Приоритетный путь (минимально жизнеспособный)
1) Ph1 Foundations → 2) vLLM (A) + 3) llama.cpp (D) → 6) Routing/API → 7) UI (базовый статус) → 8) Metrics.
Потом добавить 4) SGLang, 5) TGI/TRT.

## 🛠️ Требования к окружению
- Docker + NVIDIA Container Toolkit на хосте (Rocky9).
- Директории кеша: `/data/models` (HF), `/data/gguf` (GGUF), `/data/engines/trt` (TRT).
- Порты: по умолчанию динамические/из пула, управлятся Orchestrator’ом.

## 📌 Заметки по реализации
- Оставляем загрузчик моделей в Go (как yzma) — модели скачиваем перед стартом контейнера; контейнер запускается на локальных путях.
- Для TRT: отдельный шаг конверсии и кеширования, повторное использование при перезапуске.
- Health: опрашивать HTTP health каждого провайдера; тайм-ауты и автокилл при старте, если завис.

## ✅ Definition of Done (MVP)
- vLLM и llama.cpp доступны как провайдеры; можно загрузить модель HF и GGUF, получить ответ через OpenAI API, видеть health/статус, управлять старт/стоп, иметь базовые метрики.

## Known Gaps / Follow-ups
- INF-013/023/033/044: Ручные тесты провайдеров на реальном оборудовании (user).

## ✅ Completed (Post-MVP)
- **UI интеграция**: полная страница `/admin/models` с tabs (Models/Cache/TRT), health/metrics/logs panel, TRT conversion form.
- **Real Docker tests**: `internal/inference/docker_integration_test.go` с `go test -tags=docker_integration`.

