# Inference Providers How-To Guide

Руководство по настройке и использованию провайдеров инференса.

## Общие требования

- Docker с NVIDIA Container Toolkit
- GPU с поддержкой CUDA (RTX 4090, A100, H100, etc.)
- Директории кеша:
  - `/data/models` — HuggingFace модели
  - `/data/gguf` — GGUF файлы
  - `/data/engines/trt` — TensorRT engines (опционально)

## Provider A: vLLM

**Формат**: HuggingFace (transformers)
**Контейнер**: `vllm/vllm-openai:latest`
**Типы моделей**: Текстовые LLM (Llama, Mistral, Qwen, etc.)

### Пример загрузки

```bash
curl -X POST http://localhost:8080/api/system/inference/load \
  -H "Content-Type: application/json" \
  -d '{
    "alias": "llama3-8b",
    "provider": "vllm",
    "format": "hf",
    "hf_repo": "meta-llama/Llama-3.1-8B-Instruct",
    "capabilities": ["chat"],
    "vllm_tensor_parallel": 1,
    "vllm_max_model_len": 8192,
    "vllm_gpu_utilization": 0.9
  }'
```

### Параметры

| Параметр | Описание | Default |
|----------|----------|---------|
| `vllm_tensor_parallel` | Количество GPU для tensor parallelism | 1 |
| `vllm_max_model_len` | Максимальная длина контекста | auto |
| `vllm_gpu_utilization` | Использование VRAM (0.0-1.0) | 0.9 |

### Multi-GPU (2x RTX 4090)

```json
{
  "alias": "llama3-70b",
  "provider": "vllm",
  "format": "hf",
  "hf_repo": "meta-llama/Llama-3.1-70B-Instruct",
  "vllm_tensor_parallel": 2,
  "vllm_max_model_len": 4096,
  "vllm_gpu_utilization": 0.85
}
```

---

## Provider B: SGLang

**Формат**: HuggingFace (transformers)
**Контейнер**: `arrichm/sglang:latest`
**Типы моделей**: Текстовые + Vision (Qwen2-VL, LLaVA, etc.)

### Пример загрузки (текстовая модель)

```bash
curl -X POST http://localhost:8080/api/system/inference/load \
  -H "Content-Type: application/json" \
  -d '{
    "alias": "qwen2-7b",
    "provider": "sglang",
    "format": "hf",
    "hf_repo": "Qwen/Qwen2-7B-Instruct",
    "capabilities": ["chat"],
    "sglang_tensor_parallel": 1,
    "sglang_mem_fraction": 0.9
  }'
```

### Пример загрузки (VLM — Vision Language Model)

```bash
curl -X POST http://localhost:8080/api/system/inference/load \
  -H "Content-Type: application/json" \
  -d '{
    "alias": "qwen2-vl",
    "provider": "sglang",
    "format": "hf",
    "hf_repo": "Qwen/Qwen2-VL-7B-Instruct",
    "capabilities": ["chat", "vision"],
    "sglang_tensor_parallel": 1,
    "sglang_context_len": 32768,
    "sglang_chunked_prefill": true
  }'
```

### Параметры

| Параметр | Описание | Default |
|----------|----------|---------|
| `sglang_tensor_parallel` | Количество GPU (--tp) | 1 |
| `sglang_data_parallel` | Data parallelism (--dp) | 1 |
| `sglang_mem_fraction` | Использование VRAM | 0.9 |
| `sglang_context_len` | Длина контекста | auto |
| `sglang_chunked_prefill` | Chunked prefill для длинного контекста | false |

---

## Provider C: TGI (Text Generation Inference)

**Формат**: HuggingFace (transformers)
**Контейнер**: `ghcr.io/huggingface/text-generation-inference:latest`
**Типы моделей**: Текстовые LLM

### Пример загрузки

```bash
curl -X POST http://localhost:8080/api/system/inference/load \
  -H "Content-Type: application/json" \
  -d '{
    "alias": "mistral-7b",
    "provider": "tgi",
    "format": "hf",
    "hf_repo": "mistralai/Mistral-7B-Instruct-v0.3",
    "capabilities": ["chat"],
    "tgi_num_shard": 1,
    "tgi_max_concurrent_reqs": 128
  }'
```

### Параметры

| Параметр | Описание | Default |
|----------|----------|---------|
| `tgi_num_shard` | Количество GPU shards | 1 |
| `tgi_max_concurrent_reqs` | Макс. параллельных запросов | 128 |
| `tgi_max_input_len` | Макс. длина входа | 4096 |
| `tgi_max_total_tokens` | Макс. токенов (вход+выход) | 8192 |

---

## Provider D: llama.cpp server

**Формат**: GGUF
**Контейнер**: `ghcr.io/ggerganov/llama.cpp:server`
**Типы моделей**: Quantized GGUF (Q4_K_M, Q5_K_M, Q8_0, etc.)

### Пример загрузки

```bash
curl -X POST http://localhost:8080/api/system/inference/load \
  -H "Content-Type: application/json" \
  -d '{
    "alias": "llama3-q4",
    "provider": "llama.cpp",
    "format": "gguf",
    "gguf_url": "https://huggingface.co/QuantFactory/Meta-Llama-3.1-8B-Instruct-GGUF/resolve/main/Meta-Llama-3.1-8B-Instruct.Q4_K_M.gguf",
    "capabilities": ["chat"],
    "llama_n_gpu_layers": 99
  }'
```

### Параметры

| Параметр | Описание | Default |
|----------|----------|---------|
| `llama_n_gpu_layers` | Слои для GPU offload (99 = все) | 0 |
| `llama_main_gpu` | Основной GPU (0, 1, ...) | 0 |
| `llama_tensor_split` | Разделение между GPU ("0.5,0.5") | — |

### Multi-GPU Split

```json
{
  "alias": "llama3-70b-q4",
  "provider": "llama.cpp",
  "format": "gguf",
  "gguf_url": "https://huggingface.co/.../llama-3.1-70b.Q4_K_M.gguf",
  "llama_n_gpu_layers": 99,
  "llama_tensor_split": "0.5,0.5"
}
```

---

## Использование OpenAI API

После загрузки модели используйте стандартный OpenAI API:

```bash
# Chat completions
curl http://localhost:8080/v1/inference/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama3-8b",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'

# List models
curl http://localhost:8080/v1/inference/models
```

---

## Управление моделями

```bash
# Статус моделей
curl http://localhost:8080/api/system/inference/models

# Health check
curl "http://localhost:8080/api/system/inference/health?alias=llama3-8b"

# Остановить модель
curl -X POST "http://localhost:8080/api/system/inference/stop?alias=llama3-8b"

# Логи контейнера
curl "http://localhost:8080/api/system/inference/logs?alias=llama3-8b&tail=50"

# Метрики провайдера
curl "http://localhost:8080/api/system/inference/metrics?alias=llama3-8b"

# Закрепить модель (skip auto-stop)
curl -X POST "http://localhost:8080/api/system/inference/pin?alias=llama3-8b"

# Удалить артефакты (модель должна быть остановлена)
curl -X POST "http://localhost:8080/api/system/inference/delete-artifacts?alias=llama3-8b"
```

---

## Выбор провайдера

| Сценарий | Рекомендация |
|----------|--------------|
| Текст, максимум throughput | vLLM |
| Vision + текст (мультимодальность) | SGLang |
| HuggingFace ecosystem | TGI |
| GGUF, минимум зависимостей | llama.cpp |
| Ограниченный VRAM | llama.cpp (quantized) |
| Multi-GPU (70B+) | vLLM или SGLang с tensor_parallel |

---

## Конфигурация сервера

В `configs/dev.yaml`:

```yaml
inference:
  hf_token: "hf_xxx..."  # HuggingFace токен
  hf_cache_dir: "/data/models"
  gguf_cache_dir: "/data/gguf"
  max_running_models: 2  # LRU eviction
  cache_max_bytes: 107374182400  # 100GB
  health_check_timeout: 60s
  startup_timeout: 5m
```

