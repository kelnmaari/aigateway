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

