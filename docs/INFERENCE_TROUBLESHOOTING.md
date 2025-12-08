# Inference Troubleshooting Guide

## Download Issues

### Ошибка: "download failed after 3 retries"

**Причина**: Сетевые проблемы, HuggingFace rate limiting, неверный URL.

**Решение**:
1. Проверьте интернет-соединение
2. Проверьте HF_TOKEN в конфиге (для gated моделей)
3. Проверьте URL модели
4. Попробуйте скачать вручную:
   ```bash
   huggingface-cli download meta-llama/Llama-3.1-8B-Instruct
   ```

### Ошибка: "sha mismatch: expected X got Y"

**Причина**: Файл повреждён при скачивании или неверный expected_sha.

**Решение**:
1. Удалите частично скачанный файл:
   ```bash
   rm /data/models/*.part
   rm /data/gguf/*.part
   ```
2. Повторите загрузку
3. Если используете expected_sha, проверьте его корректность

### Ошибка: "prepare gguf cache: permission denied"

**Причина**: Недостаточно прав на директорию кеша.

**Решение**:
```bash
sudo chown -R $USER:$USER /data/gguf /data/models
chmod -R 755 /data/gguf /data/models
```

---

## Container Startup Issues

### Ошибка: "docker run failed: nvidia-container-cli: initialization error"

**Причина**: NVIDIA Container Toolkit не установлен или не настроен.

**Решение**:
```bash
# Проверка nvidia-smi
nvidia-smi

# Установка NVIDIA Container Toolkit (Rocky 9)
curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | \
  sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo

sudo dnf install -y nvidia-container-toolkit
sudo nvidia-ctk runtime configure --runtime=docker
sudo systemctl restart docker

# Тест
docker run --rm --gpus all nvidia/cuda:12.0-base nvidia-smi
```

### Ошибка: "health check failed: context deadline exceeded"

**Причина**: Контейнер запускается слишком долго (модель загружается в VRAM).

**Решение**:
1. Увеличьте `health_check_timeout` в конфиге:
   ```yaml
   inference:
     health_check_timeout: 120s  # default 60s
     startup_timeout: 10m        # default 5m
   ```
2. Проверьте достаточно ли VRAM:
   ```bash
   nvidia-smi
   ```
3. Используйте меньшую модель или quantized версию

### Ошибка: "OOM killed" / "CUDA out of memory"

**Причина**: Недостаточно VRAM для модели.

**Решение**:
1. Уменьшите `vllm_gpu_utilization` или `sglang_mem_fraction`:
   ```json
   {"vllm_gpu_utilization": 0.8}
   ```
2. Уменьшите `max_model_len`:
   ```json
   {"vllm_max_model_len": 4096}
   ```
3. Используйте quantized GGUF вместо HF:
   ```json
   {"provider": "llama.cpp", "llama_n_gpu_layers": 40}
   ```
4. Включите tensor parallelism (multi-GPU):
   ```json
   {"vllm_tensor_parallel": 2}
   ```

---

## llama.cpp Specific Issues

### Ошибка: "model file not found" / "failed to load model"

**Причина**: Неверный путь к GGUF или файл не скачан.

**Решение**:
1. Проверьте наличие файла:
   ```bash
   ls -la /data/gguf/
   ```
2. Проверьте gguf_url в запросе

### Ошибка: "GPU offload failed" / "cuBLAS error"

**Причина**: Несовместимость CUDA версии или неподдерживаемый квант.

**Решение**:
1. Проверьте CUDA версию:
   ```bash
   nvcc --version
   nvidia-smi
   ```
2. Используйте совместимый образ llama.cpp
3. Уменьшите `llama_n_gpu_layers` для частичного offload

### Модель работает на CPU вместо GPU

**Причина**: `llama_n_gpu_layers` = 0 или CUDA не обнаружен.

**Решение**:
```json
{
  "llama_n_gpu_layers": 99
}
```
Значение 99 означает "все слои на GPU".

---

## vLLM Specific Issues

### Ошибка: "ValueError: Model architecture not supported"

**Причина**: vLLM не поддерживает данную архитектуру модели.

**Решение**: Используйте SGLang или TGI для неподдерживаемых архитектур.

### Ошибка: "Cannot use tensor parallelism with X GPUs"

**Причина**: Количество слоёв модели не делится на tensor_parallel.

**Решение**: Используйте tensor_parallel = 1, 2, 4, 8 (степени двойки).

---

## SGLang Specific Issues

### Ошибка: "Failed to load vision model"

**Причина**: Неподдерживаемая vision модель или неверные параметры.

**Решение**:
1. Используйте поддерживаемые VLM: Qwen2-VL, LLaVA
2. Убедитесь что HF_TOKEN имеет доступ к модели

---

## TGI Specific Issues

### Ошибка: "Model requires sharding but num_shard=1"

**Причина**: Модель слишком большая для одного GPU.

**Решение**:
```json
{"tgi_num_shard": 2}
```

---

## TensorRT-LLM Conversion (INF-042)

> **Примечание**: TRT-LLM конверсия пока не реализована. Контейнер стартует только если engine уже существует.

### Workaround: Ручная конверсия

```bash
# Конвертация с помощью trtllm-build
docker run --gpus all -v /data/models:/models -v /data/engines/trt:/engines \
  nvcr.io/nvidia/tensorrt-llm:latest \
  trtllm-build --model_dir /models/meta-llama/Llama-3.1-8B-Instruct \
               --output_dir /engines/llama3-8b \
               --dtype float16
```

---

## Logs & Diagnostics

### Просмотр логов контейнера

```bash
# Через API
curl "http://localhost:8080/api/system/inference/logs?alias=mymodel&tail=100"

# Напрямую через Docker
docker logs $(docker ps -q --filter name=inf-mymodel)
```

### Prometheus метрики

```bash
# Метрики провайдера
curl "http://localhost:8080/api/system/inference/metrics?alias=mymodel"

# Системные метрики inference
curl http://localhost:8080/metrics | grep inference_
```

### Проверка GPU

```bash
# Использование VRAM
nvidia-smi

# Процессы на GPU
nvidia-smi pmon -s u -d 1

# Детальная информация
nvidia-smi -q
```

---

## Common Patterns

### Освобождение VRAM

```bash
# Остановить все модели
for alias in $(curl -s http://localhost:8080/api/system/inference/models | jq -r '.[].alias'); do
  curl -X POST "http://localhost:8080/api/system/inference/stop?alias=$alias"
done
```

### Очистка кеша

```bash
# Очистить до 50GB
curl -X POST "http://localhost:8080/api/system/inference/evict-cache?limit_bytes=53687091200"

# Полная очистка (осторожно!)
rm -rf /data/models/* /data/gguf/*
```

### Перезапуск модели

```bash
alias=mymodel
curl -X POST "http://localhost:8080/api/system/inference/stop?alias=$alias"
sleep 2
curl -X POST "http://localhost:8080/api/system/inference/load" \
  -H "Content-Type: application/json" \
  -d '{"alias": "'$alias'", ...}'
```

