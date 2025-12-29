# 🚀 Quick Start Guide — AIGateway Platform v4.0

Начните работу за 5 минут!

---

## 📋 Требования

- **Go 1.25+** (для сборки из исходников)
- **Docker** с NVIDIA Container Toolkit (для GPU inference)
- **NVIDIA GPU** (для vLLM/SGLang/TGI)
- **PostgreSQL 13+** с pgvector (опционально, для RAG)

---

## 🐳 Pre-pull Docker Images

```bash
# vLLM (рекомендуется для production)
docker pull vllm/vllm-openai:latest

# SGLang (research, embeddings)
docker pull lmsysorg/sglang:latest

# TGI (HuggingFace native)
docker pull ghcr.io/huggingface/text-generation-inference:latest

# llama.cpp (GGUF модели, CPU/GPU)
docker pull ghcr.io/ggml-org/llama.cpp:server-cuda
```

---

## 🔧 Установка

### Option 1: RPM Package (Rocky Linux / RHEL)

```bash
# Установка одной командой
curl -fsSL https://your-gitlab.com/api/v4/projects/XXX/packages/generic/aigateway/latest/install.sh | sudo bash

# Запуск
sudo systemctl start oop
sudo systemctl enable oop
```

### Option 2: Build from Source

```bash
# Clone
git clone https://gitlab.alexue4.dev/KelnMaari/ollama-openai-proxy.git
cd ollama-openai-proxy

# Build (Windows)
.\build-new.ps1 all -WebUI svelte

# Build (Linux/macOS)
./build.sh all

# Запуск
./bin/server -config configs/dev.yaml
```

---

## 🌐 Первый запуск

### 1. Откройте WebUI

```
http://localhost:8080/login
```

### 2. Войдите под admin

- **Email:** `admin@localhost`
- **Password:** `admin`

### 3. Загрузите модель

1. Перейдите в **Admin → Models**
2. Откройте вкладку **HuggingFace**
3. Найдите модель (например, `Qwen2.5-7B-Instruct`)
4. Выберите провайдер (vLLM, SGLang, TGI, llama.cpp)
5. Нажмите **Use**, настройте GPU
6. Нажмите **Load & Start**

### 4. Начните чат

1. Перейдите в **Chat**
2. Выберите загруженную модель
3. Включите **Web Search** (опционально)
4. Начните диалог!

---

## ⚙️ Базовая конфигурация

Файл: `configs/dev.yaml`

```yaml
server:
  host: "0.0.0.0"
  port: 8080

inference:
  enabled: true
  hf_cache_dir: "data/models/hf"
  gguf_cache_dir: "data/models/gguf"
  docker:
    api: true
    socket: "unix:///var/run/docker.sock"  # Linux
    # socket: "npipe:////./pipe/docker_engine"  # Windows

# HuggingFace token (для gated моделей)
huggingface:
  token: "hf_xxx..."

# Web Search (опционально)
tools:
  tavily_api_key: "tvly-xxx..."  # https://tavily.com
```

---

## 🔧 GPU настройки

| Размер модели | GPU | Tensor Parallel | GPU Util | Max Model Len |
|---------------|-----|-----------------|----------|---------------|
| 7B BF16       | 1   | 1               | 0.9      | 32768         |
| 14B BF16      | 1   | 1               | 0.95     | 16384         |
| 30B INT8      | 2   | 2               | 0.9      | 32768         |
| 70B INT4      | 2   | 2               | 0.95     | 8192          |

---

## 🔍 Проверка работы

```bash
# Список моделей
curl http://localhost:8080/v1/models

# Chat completion
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "your-model-alias",
    "messages": [{"role": "user", "content": "Привет!"}],
    "stream": true
  }'
```

---

## 🆘 Troubleshooting

### Модель не загружается

```bash
# Проверьте Docker
docker ps

# Логи контейнера
docker logs aigw-your-model-alias
```

### OOM Error

Уменьшите `gpu_memory_utilization` или `max_model_len` в настройках модели.

### Нет GPU

Проверьте NVIDIA Container Toolkit:
```bash
nvidia-smi
docker run --rm --gpus all nvidia/cuda:11.0-base nvidia-smi
```

---

## 📚 Следующие шаги

- [Configuration Guide](CONFIGURATION.md) — полная настройка
- [Inference HowTo](INFERENCE_HOWTO.md) — работа с моделями
- [GitLab Integration](gitlab-readme.md) — AI code review
- [RAG Config Guide](RAG_CONFIG_GUIDE.md) — настройка RAG

---

**Version:** 4.0.2  
**Last Updated:** 2025-12-29
