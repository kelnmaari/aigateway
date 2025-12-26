# Варианты архитектуры инференса (GPU, авто-загрузка моделей)

## Условные обозначения
- UI — WebUI/CLI пользователя.
- Go Proxy — текущий сервер (OpenAI-совместимый фасад).
- Model Orchestrator — код в Go, управляющий жизненным циклом контейнера и моделей (download/start/stop/status).
- Inference Service — контейнер с выбранным движком (vLLM/SGLang/TGI/llama.cpp).
- HF Hub — huggingface.co (или корпоративный mirror).
- Cache — локальный том с моделями /data/models (HF форматы) или /data/gguf (GGUF).
- GPU — RTX 4090 (NVIDIA runtime).

## Вариант A: vLLM (текст, высокая пропускная, OpenAI API)
```
UI ──HTTP(S)──> Go Proxy ──RPC──> Model Orchestrator
                            │
                            │ start container (Docker, --gpus all)
                            │ mount: /data/models -> ~/.cache/huggingface
                            ▼
                    Inference Service (vLLM, OpenAI API)
                            │
                            │ on first load: download HF weights
                            │   - HF Hub -> /data/models (cached)
                            │   - supports resume/etag
                            ▼
                          GPU
                            │
                            ▼
                        Responses -> Go Proxy -> UI
```
Автозагрузка: при LoadModel Go-прокси проверяет наличие в /data/models; если нет — скачивает через HF токен, затем стартует контейнер с `--model <repo>` или локальным путём.

## Вариант B: SGLang (мультимодальность: текст+изображение/видео, OpenAI API)
```
UI ──HTTP──> Go Proxy ──RPC──> Model Orchestrator
                         │
                         │ start container (Docker, --gpus all, HF_TOKEN)
                         │ mount: /data/models
                         ▼
                 Inference Service (SGLang)
                         │
                         │ download HF weights if missing
                         ▼
                       GPU
                         │
                         ▼
                     Responses
```
Автозагрузка: аналогично vLLM; SGLang может работать с vision моделями (Qwen2-VL, LLaVA и др.).

## Вариант C: TGI + backend (vLLM или TensorRT-LLM)
```
UI ──HTTP──> Go Proxy ──RPC──> Model Orchestrator
                         │
                         │ start TGI container (Docker, --gpus all)
                         │ flags: --model-id <repo_or_path>
                         │ mount: /data/models
                         ▼
                TGI (text-generation-inference)
                         │
                         │ Backend:
                         │   - vLLM: быстрота, без конверсии
                         │   - TensorRT-LLM: максимум скорость, нужна конверсия и кеш eng
                         │
                         │ download HF weights if missing
                         ▼
                       GPU
                         │
                         ▼
                     Responses
```
Автозагрузка: как vLLM, плюс опция подготовки TRT-энджинов и кеширования их рядом.

## Вариант D: llama.cpp server (GGUF, минимальная зависимость)
```
UI ──HTTP──> Go Proxy ──RPC──> Model Orchestrator
                         │
                         │ start llama.cpp server container (--gpus all)
                         │ mount: /data/gguf
                         ▼
               Inference Service (llama.cpp server)
                         │
                         │ download GGUF (HTTP/S3/HF mirror) if missing
                         ▼
                       GPU (если квант поддерживает offload)
                         │
                         ▼
                     Responses
```
Автозагрузка: при LoadModel скачиваем GGUF в /data/gguf (как уже реализовано для yzma), затем стартуем/перезапускаем контейнер с нужным путём к файлу.

## Поток операций (общий для A/B/C; для D меняется формат и движок)
1) UI: запрос на загрузку модели (alias, repo_id, optional_revision, quant/preset).
2) Go Proxy -> Orchestrator: create/update record, resolve локальный путь.
3) Orchestrator:
   - Проверка наличия в Cache.
   - Если нет — скачивание из HF (с токеном), в atomic-папку с checksum; затем mv в целевой путь.
4) Orchestrator: запускает/рестартует контейнер Inference Service с:
   - `--model` (HF id) или `--model-id` (локальный путь).
   - `--gpus all`, `--ipc=host`, монтирование Cache.
   - Health endpoint пробрасывается наружу для статуса.
5) Go Proxy: маршрутизирует OpenAI-совместимые запросы к сервису; собирает метрики.
6) UI: показывает статус загрузки, кеш-хит/промах, использование GPU.

## Хранилище моделей
- HF форматы: `/data/models/<org>/<name>/...` (shared volume между Go Proxy и контейнерами).
- GGUF: `/data/gguf/<vendor>/<model>.gguf`.
- Опционально: `/data/engines/trt/<model>/...` для TensorRT-LLM.

## Управление контейнерами
- Docker (nvidia-container-runtime), либо containerd.
- Политика: один контейнер на модель или пул (по задаче). Для MVP — один контейнер на модель, стоп при unload.
- Health/metrics: HTTP endpoints контейнера, проксируются в Go Proxy.

## Безопасность и сети
- HF токен в env контейнера только на время загрузки; кешировать модель локально, затем можно запускать без токена.
- Ограничить исходящий интернет контейнера после загрузки (опционально) — тогда download делает только Orchestrator.

## Что выбрать
- Текст, максимальная пропускная: Вариант A (vLLM).
- Мультимодальность (картинки/видео): Вариант B (SGLang) или C с подходящим backend.
- Максимум скорость на NVIDIA, ок с конверсией: Вариант C + TensorRT-LLM backend.
- Нужен GGUF: Вариант D (llama.cpp server).

## Реализация v4 (Inference System)

### Компоненты
- `internal/inference/service.go` — ServiceConfig, Service (wires orchestrator, downloader, runtime)
- `internal/inference/orchestrator.go` — PrepareModel, StartModel, StopModel, health wait
- `internal/inference/manager.go` — LRU eviction, pin/unpin, idle reaper, cache GC
- `internal/inference/router.go` — EnsureByAlias, EnsureByCapability, routing logic
- `internal/inference/runtime_docker.go` — DockerRuntime (API + CLI), GPU support, logs
- `internal/inference/downloader.go` — EnsureHFFile, EnsureGGUF, retry with backoff
- `internal/inference/provider_builders.go` — BuildVLLMRequest, BuildSGLangRequest, BuildTGIRequest, BuildLlamaCPPRequest, BuildTRTLLMRequest
- `internal/inference/trt_converter.go` — TRTConverter, Convert(), GetCachedEngine(), metadata validation

### API Endpoints

**Admin API** (`/api/system/inference/*`):
| Endpoint | Method | Описание |
|----------|--------|----------|
| `/load` | POST | Загрузить и запустить модель |
| `/prepare` | POST | Только скачать артефакты |
| `/stop` | POST | Остановить контейнер |
| `/evict` | POST | Остановить и забыть модель |
| `/pin` | POST | Закрепить (skip auto-stop) |
| `/unpin` | POST | Снять закрепление |
| `/delete-artifacts` | POST | Удалить локальные файлы |
| `/evict-cache` | POST | Очистить кеш до лимита |
| `/cache` | GET | Список кешированных файлов |
| `/health` | GET | Проверка health модели |
| `/models` | GET | Список моделей |
| `/logs` | GET | Логи контейнера |
| `/metrics` | GET | Prometheus метрики провайдера |
| `/convert-trt` | POST | Конвертировать HF модель в TensorRT engine |
| `/trt-engines` | GET | Список TRT engines |
| `/delete-trt-engine` | POST | Удалить TRT engine |

**OpenAI Proxy** (`/v1/inference/*`):
| Endpoint | Method | Описание |
|----------|--------|----------|
| `/chat/completions` | POST | Чат (проксирует к провайдеру) |
| `/completions` | POST | Legacy completions |
| `/models` | GET | Список доступных моделей |

### Параметры провайдеров (LoadRequest)

**vLLM**:
- `vllm_tensor_parallel` — --tensor-parallel-size
- `vllm_max_model_len` — --max-model-len
- `vllm_gpu_utilization` — --gpu-memory-utilization (0..1)

**SGLang**:
- `sglang_tensor_parallel` — --tp
- `sglang_data_parallel` — --dp
- `sglang_mem_fraction` — --mem-fraction-static
- `sglang_context_len` — --context-length
- `sglang_chunked_prefill` — --chunked-prefill-size

**TGI**:
- `tgi_num_shard` — --num-shard
- `tgi_max_concurrent_reqs` — --max-concurrent-requests
- `tgi_max_input_len` — --max-input-length
- `tgi_max_total_tokens` — --max-total-tokens

**llama.cpp**:
- `llama_n_gpu_layers` — --n-gpu-layers
- `llama_main_gpu` — --main-gpu
- `llama_tensor_split` — --tensor-split

### Конфигурация (ServiceConfig)
- `HFToken` — токен Hugging Face
- `HFCacheDir` — директория кеша HF (/data/models)
- `GGUFCacheDir` — директория GGUF (/data/gguf)
- `MaxRunningModels` — макс. запущенных контейнеров (LRU eviction)
- `CacheMaxBytes` — лимит кеша (auto-evict)
- `HealthCheckTimeout` — таймаут health check (default 60s)
- `StartupTimeout` — общий таймаут старта (default 5m)

