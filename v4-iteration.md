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

## 📅 Фазы

### Phase 1: Foundations (Provider Interface + Orchestrator)
- [ ] INF-001: Определить интерфейс Provider (load/start/stop/health/metrics/proxy).
- [ ] INF-002: Реализовать Orchestrator: управление Docker (nvidia runtime), монтирование кешей, порты/health.
- [ ] INF-003: Общий загрузчик моделей: HF download (с токеном), resume/etag, atomic move; HTTP/GGUF download.
- [ ] INF-004: Хранилище метаданных моделей: статус, провайдер, локальный путь, размер, checksum, backend.
- [ ] INF-005: Конфиг: выбор провайдера по модели/alias; общие лимиты (memory, gpus, ports).

### Phase 2: Provider A (vLLM)
- [ ] INF-010: Адаптер vLLM (контейнер vllm/vllm-openai).
- [ ] INF-011: Параметры запуска: --model (HF id или локальный путь), --tensor-parallel, --max-model-len, --gpu-memory-util.
- [ ] INF-012: Health/metrics проброс, логирование stdout/stderr.
- [ ] INF-013: Тест: текстовая модель (Llama-3.x), стриминг, multi-GPU.

### Phase 3: Provider D (llama.cpp server, GGUF)
- [ ] INF-020: Адаптер llama.cpp server (OpenAI mode).
- [ ] INF-021: Переработка скачивания GGUF: единый загрузчик (HTTP/HF mirror), resume, checksum, atomic move; кеш /data/gguf.
- [ ] INF-022: Параметры: n_gpu_layers, tensor_split, main_gpu; health.
- [ ] INF-023: Тест: GGUF модель q4_K_M, оффлоад на 4090.

### Phase 4: Provider B (SGLang, мультимодальность)
- [ ] INF-030: Адаптер SGLang (контейнер arrichm/sglang или офиц.).
- [ ] INF-031: Поддержка vision моделей (Qwen2-VL, LLaVA) + текстовые.
- [ ] INF-032: Настройки: impl=transformers, tp/pp/gpu flags; health/metrics.
- [ ] INF-033: Тест: VLM (картинка + текст), стриминг.

### Phase 5: Provider C (TGI + backend vLLM / TensorRT-LLM)
- [ ] INF-040: Адаптер TGI (ghcr.io/huggingface/text-generation-inference).
- [ ] INF-041: Backend=vLLM: простая ветка без конверсии.
- [ ] INF-042: Backend=TensorRT-LLM: пайплайн конверсии (trt-llm-converter), кеш /data/engines/trt, валидация совместимости GPU/SM.
- [ ] INF-043: Параметры: --model-id, --num-shard, --max-concurrent-requests; health/metrics.
- [ ] INF-044: Тест: текстовая модель HF → TRT конверсия → инференс.

### Phase 6: API & Routing
- [ ] INF-050: Единый OpenAI фасад: /v1/chat/completions, /v1/completions, /v1/models → маршрутизация к провайдеру по модели/alias.
- [ ] INF-051: Admin API: операции load/unload, статус, логи, выбор провайдера, просмотр кешей.
- [ ] INF-052: Поддержка отмены загрузки/старта контейнера, тайм-ауты.

### Phase 7: UI (Admin) — рефактор после yzma
- [ ] INF-060: Удалить/скрыть yzma-специфичные блоки (GPU offload, tensor_split, cancel load) из моделей/статуса; заменить на универсальные провайдеры и параметры per engine.
- [ ] INF-061: UI: выбор провайдера при загрузке модели; отображение статуса/health/metrics per provider.
- [ ] INF-062: UI: управление контейнером (start/stop/restart), просмотр логов tail.
- [ ] INF-063: UI: кеши (HF/GGUF/TRT), размер, очистка/GC (ручная), прогресс скачивания.
- [ ] INF-064: UI: отображение формата/пути модели (HF/GGUF/TRT), провайдер в списке моделей.

### Phase 8: Metrics & Observability
- [ ] INF-070: Prometheus метрики per provider: load time, errors, GPU util (если доступно), memory, RPS.
- [ ] INF-071: Логи: агрегирование stdout/stderr контейнеров в Go логгер (structured).
- [ ] INF-072: Алёрты: тайм-аут запуска, отказ health.

### Phase 9: Reliability & GC
- [ ] INF-080: Политики авто-стопа неиспользуемых контейнеров (idle timeout).
- [ ] INF-081: Политики очистки кеша: LRU по размеру/времени; защита pinned моделей.
- [ ] INF-082: Ретраи скачивания (backoff), проверка checksum/etag.

### Phase 10: Security & Networking
- [ ] INF-090: HF_TOKEN только в этапе download; опционально запрет исходящего трафика контейнера после старта.
- [ ] INF-091: Изоляция портов, firewall (localhost bind), auth на admin API.

### Phase 11: Тестирование
- [ ] INF-100: Интеграционные тесты провайдеров (mock download + real container happy-path).
- [ ] INF-101: Негативные тесты: нет GPU, ошибка скачивания, нет места, health fail.
- [ ] INF-102: Нагрузочные: параллельные загрузки/инференс (smoke, не полный бенч).

### Phase 12: Документация
- [ ] INF-110: Обновить `docs/LLM_INFRA_ARCH.md` ссылками на провайдеры и флаги.
- [ ] INF-111: How-to для каждого провайдера (флаги запуска, требования, типы моделей).
- [ ] INF-112: Troubleshooting (download fail, TRT конверсия, GGUF offload).

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

