# Установка yzma (llama.cpp) с GPU на Rocky Linux 9

> **Целевое оборудование:** NVIDIA GeForce RTX 4090 (24 GB VRAM)  
> **ОС:** Rocky Linux 9.5  
> **Дата актуализации:** Декабрь 2025

## Содержание

1. [Требования к системе](#требования-к-системе)
2. [Актуальные версии компонентов NVIDIA](#актуальные-версии-компонентов-nvidia)
3. [Установка драйверов NVIDIA](#установка-драйверов-nvidia)
4. [Установка CUDA Toolkit](#установка-cuda-toolkit)
5. [Установка зависимостей для сборки](#установка-зависимостей-для-сборки)
6. [Компиляция llama.cpp с CUDA](#компиляция-llamacpp-с-cuda)
7. [Проверка работы GPU](#проверка-работы-gpu)
8. [Конфигурация AIGateway](#конфигурация-aigateway)
9. [Диагностика проблем](#диагностика-проблем)
10. [Оптимизация для RTX 4090](#оптимизация-для-rtx-4090)

---

## Требования к системе

### Минимальные требования
- Rocky Linux 9.4+ (рекомендуется 9.5)
- NVIDIA GeForce RTX 4090 (или другой GPU с Compute Capability ≥ 8.9)
- 24 GB VRAM (RTX 4090)
- 64 GB RAM (рекомендуется для больших моделей)
- Root доступ или sudo права

### Характеристики RTX 4090

| Параметр | Значение |
|----------|----------|
| Архитектура | Ada Lovelace |
| Compute Capability | 8.9 |
| CMAKE_CUDA_ARCHITECTURES | **89** |
| CUDA Cores | 16384 |
| VRAM | 24 GB GDDR6X |
| Memory Bandwidth | 1008 GB/s |
| TDP | 450W |
| Рекомендуемый БП | 850W+ |

### Таблица архитектур GPU (для справки)

| GPU серия | Архитектура | Compute Capability | CMAKE_CUDA_ARCHITECTURES |
|-----------|-------------|-------------------|--------------------------|
| GTX 10xx | Pascal | 6.1 | 61 |
| RTX 20xx | Turing | 7.5 | 75 |
| RTX 30xx | Ampere | 8.6 | 86 |
| **RTX 40xx** | **Ada Lovelace** | **8.9** | **89** |
| RTX 50xx | Blackwell | 10.0 | 100 |
| A100 | Ampere | 8.0 | 80 |
| H100 | Hopper | 9.0 | 90 |
| H200 | Hopper | 9.0 | 90 |

---

## Актуальные версии компонентов NVIDIA

> Данные на декабрь 2025

| Компонент | Версия | Примечание |
|-----------|--------|------------|
| **NVIDIA Driver** | 591.39+ (Latest) | Production Branch |
| **NVIDIA Driver** | R580.x | Long Term Support (до 08/2028) |
| **CUDA Toolkit** | 13.1 | Latest Release |
| **CUDA Toolkit** | 13.0.1 | Stable Release |
| **CUDA Toolkit** | 12.9.x | LTS (последний с поддержкой Pascal/Volta) |
| **cuDNN** | 9.6.x | Для Deep Learning |
| **TensorRT** | 10.8.x | Для inference |
| **NCCL** | 2.25.x | Multi-GPU |
| **Nsight Compute** | 2025.3.1 | GPU Profiler |
| **Nsight Systems** | 2025.3.2 | System Profiler |

### Совместимость версий

```
Driver R590.x → CUDA 13.1, 13.0, 12.9, 12.8, ...
Driver R580.x → CUDA 13.0, 12.9, 12.8, ... (LTS до 08/2028)
Driver R570.x → CUDA 12.8, 12.7, 12.6, ...
```

### Важно: Прекращение поддержки старых архитектур

Начиная с CUDA 13.0, NVIDIA прекратила поддержку:
- ❌ Maxwell (GTX 9xx, Compute 5.x)
- ❌ Pascal (GTX 10xx, Compute 6.x)
- ❌ Volta (V100, Compute 7.0)

**RTX 4090 (Ada Lovelace, Compute 8.9) полностью поддерживается!**

**Рекомендация для RTX 4090:**
- Driver: **R590.x** (latest) или **R580.x** (LTS)
- CUDA Toolkit: **13.1** (для новых фич) или **13.0.1** (для стабильности)

---

## Установка драйверов NVIDIA

### Шаг 1: Подготовка системы

```bash
# Обновление системы
sudo dnf update -y
sudo dnf upgrade --refresh -y

# Установка EPEL репозитория
sudo dnf install -y epel-release

# Установка базовых инструментов
sudo dnf groupinstall -y "Development Tools"
sudo dnf install -y kernel-devel-$(uname -r) kernel-headers-$(uname -r) dkms

# Отключение nouveau драйвера (обязательно для RTX 4090!)
sudo bash -c 'cat > /etc/modprobe.d/blacklist-nouveau.conf << EOF
blacklist nouveau
options nouveau modeset=0
EOF'

# Альтернативный метод через grubby
sudo grubby --args="nouveau.modeset=0 rd.driver.blacklist=nouveau" --update-kernel=ALL

# Пересоздание initramfs
sudo dracut --force

# Перезагрузка
sudo reboot
```

### Шаг 2: Установка драйвера NVIDIA

```bash
# Добавление официального NVIDIA репозитория для RHEL9/Rocky9
sudo dnf config-manager --add-repo https://developer.download.nvidia.com/compute/cuda/repos/rhel9/x86_64/cuda-rhel9.repo

# Очистка кэша
sudo dnf clean all
sudo dnf makecache

# Установка драйвера (выберите один из вариантов):

# Вариант A: Последний stable драйвер с DKMS (рекомендуется)
sudo dnf module install -y nvidia-driver:latest-dkms

# Вариант B: Конкретная версия драйвера
# sudo dnf install -y nvidia-driver-565.57.01 nvidia-driver-cuda-libs

# Перезагрузка
sudo reboot
```

### Шаг 3: Проверка драйвера

```bash
# Проверка загрузки модуля
lsmod | grep nvidia

# Ожидаемый вывод:
# nvidia_drm            xxx  x
# nvidia_modeset        xxx  x nvidia_drm
# nvidia_uvm            xxx  x
# nvidia              xxxxx  x nvidia_uvm,nvidia_modeset

# Проверка nvidia-smi
nvidia-smi
```

**Ожидаемый вывод для RTX 4090:**
```
+-----------------------------------------------------------------------------------------+
| NVIDIA-SMI 591.39                Driver Version: 591.39        CUDA Version: 13.1       |
|-----------------------------------------+------------------------+----------------------+
| GPU  Name                 Persistence-M | Bus-Id          Disp.A | Volatile Uncorr. ECC |
| Fan  Temp   Perf          Pwr:Usage/Cap |           Memory-Usage | GPU-Util  Compute M. |
|=========================================+========================+======================|
|   0  NVIDIA GeForce RTX 4090        Off | 00000000:13:00.0 Off   |                  Off |
|  0%   30C    P8              15W / 450W |       1MiB / 24564MiB  |      0%      Default |
+-----------------------------------------+------------------------+----------------------+
```

---

## Установка CUDA Toolkit

### Шаг 1: Установка CUDA

```bash
# Установка CUDA Toolkit 13.1 (последняя версия, декабрь 2025)
sudo dnf install -y cuda-toolkit-13-1

# ИЛИ установка CUDA 13.0 (stable)
# sudo dnf install -y cuda-toolkit-13-0

# ИЛИ установка CUDA 12.9 (если нужна совместимость со старым кодом)
# sudo dnf install -y cuda-toolkit-12-9

# ИЛИ установка полного пакета (драйвер + toolkit)
# sudo dnf install -y cuda
```

### Шаг 2: Настройка переменных окружения

```bash
# Создание файла профиля для CUDA
sudo tee /etc/profile.d/cuda.sh << 'EOF'
# CUDA Configuration
export CUDA_HOME=/usr/local/cuda
export PATH=$CUDA_HOME/bin:$PATH
export LD_LIBRARY_PATH=$CUDA_HOME/lib64:$LD_LIBRARY_PATH
EOF

# Применение изменений
source /etc/profile.d/cuda.sh

# Также добавить в .bashrc для текущего пользователя
echo 'source /etc/profile.d/cuda.sh' >> ~/.bashrc
```

### Шаг 3: Проверка CUDA

```bash
# Проверка версии nvcc
nvcc --version

# Ожидаемый вывод:
# nvcc: NVIDIA (R) Cuda compiler driver
# Copyright (c) 2005-2025 NVIDIA Corporation
# Built on ...
# Cuda compilation tools, release 13.1, V13.1.xxx

# Проверка CUDA samples (опционально)
/usr/local/cuda/extras/demo_suite/deviceQuery
```

**Ожидаемый вывод deviceQuery для RTX 4090:**
```
Device 0: "NVIDIA GeForce RTX 4090"
  CUDA Driver Version / Runtime Version          13.1 / 13.1
  CUDA Capability Major/Minor version number:    8.9
  Total amount of global memory:                 24564 MBytes
  (128) Multiprocessors, (128) CUDA Cores/MP:    16384 CUDA Cores
  GPU Max Clock rate:                            2520 MHz
  Memory Clock rate:                             10501 Mhz
  Memory Bus Width:                              384-bit
  ...
Result = PASS
```

---

## Установка зависимостей для сборки

### Базовые инструменты

```bash
# Установка Development Tools
sudo dnf groupinstall -y "Development Tools"

# Установка CMake (нужна версия >= 3.18)
sudo dnf install -y cmake

# Проверка версии CMake
cmake --version
# Должно быть >= 3.18
```

### Установка современного GCC (для libstdc++ с GLIBCXX_3.4.30+)

```bash
# Установка GCC Toolset 13
sudo dnf install -y gcc-toolset-13

# Активация GCC Toolset
scl enable gcc-toolset-13 bash

# Или добавить в .bashrc для постоянной активации
echo 'source /opt/rh/gcc-toolset-13/enable' >> ~/.bashrc
source ~/.bashrc

# Проверка версии GCC
gcc --version
# Должно быть >= 13.x
```

### Установка дополнительных зависимостей

```bash
# CURL для загрузки моделей (опционально)
sudo dnf install -y libcurl-devel

# ccache для ускорения повторной компиляции (опционально)
sudo dnf install -y ccache
```

---

## Компиляция llama.cpp с CUDA

### Шаг 1: Клонирование репозитория

```bash
cd /opt
sudo git clone https://github.com/ggerganov/llama.cpp.git
sudo chown -R $USER:$USER llama.cpp
cd llama.cpp

# Проверка последней версии
git describe --tags
```

### Шаг 2: Подтверждение архитектуры GPU

```bash
# Узнать модель GPU
lspci | grep -i nvidia

# Ожидаемый вывод для RTX 4090:
# 13:00.0 VGA compatible controller: NVIDIA Corporation AD102 [GeForce RTX 4090] (rev a1)
# → RTX 4090 = Ada Lovelace = Compute Capability 8.9 = CMAKE_CUDA_ARCHITECTURES=89
```

### Шаг 3: Компиляция для RTX 4090

```bash
# Активация GCC Toolset 13 (ОБЯЗАТЕЛЬНО!)
source /opt/rh/gcc-toolset-13/enable

# Проверка версии GCC
gcc --version  # Должно быть 13.x

# Очистка предыдущей сборки (если была)
rm -rf build

# Создание build директории
mkdir -p build && cd build

# Конфигурация CMake для RTX 4090 (двойная 4090 тоже подходит)
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DGGML_CUDA=ON \
  -DCMAKE_CUDA_ARCHITECTURES=89 \
  -DCMAKE_CUDA_COMPILER=/usr/local/cuda/bin/nvcc \
  -DGGML_CUDA_F16=ON \
  -DGGML_CUDA_FA_ALL_QUANTS=ON \
  -DGGML_CUDA_FA_CUBLAS=ON \
  -DBUILD_SHARED_LIBS=ON \
  -DCMAKE_INSTALL_PREFIX=/usr/local/llama \
  -DCMAKE_INSTALL_RPATH=/usr/local/llama \
  -DLLAMA_BUILD_TESTS=OFF \
  -DLLAMA_BUILD_EXAMPLES=OFF

# Объяснение флагов:
# -DGGML_CUDA=ON                  - Включить CUDA backend
# -DCMAKE_CUDA_ARCHITECTURES=89   - Архитектура RTX 4090 (Ada Lovelace)
# -DGGML_CUDA_F16=ON              - FP16 ускорение (поддерживается RTX 4090)
# -DGGML_CUDA_FA_ALL_QUANTS=ON    - Flash Attention для всех квантизаций (нужно для q4_k_m/q4_0)
# -DGGML_CUDA_FA_CUBLAS=ON        - FA через cuBLAS для стабильности
# -DBUILD_SHARED_LIBS=ON          - собрать .so для dynamic linking
# -DCMAKE_INSTALL_PREFIX=/usr/local/llama - куда ставить бинарники и .so
# -DCMAKE_INSTALL_RPATH=/usr/local/llama  - чтобы .so находились без LD_LIBRARY_PATH
# -DLLAMA_BUILD_TESTS=OFF/EXAMPLES=OFF    - быстрее сборка, меньше мусора

# Опционально для производительности (проверять на своей сборке):
# -DGGML_CUDA_FORCE_MMQ=ON                 - Mixed-Memory Queues, иногда быстрее на Ada
# -DGGML_CUDA_PEER_MAX_BATCH_SIZE=128      - для multi-GPU без NVLink можно снизить до 64
# -DLLAMA_CURL=OFF                         - если не нужен HTTP в llama-cli (меньше зависимостей)

# Компиляция (RTX 4090 рекомендуется на системе с 16+ ядрами CPU)
cmake --build . --config Release -j $(nproc)

# Установка (ставит бинари и .so в /usr/local/llama)
sudo cmake --install .

# Если без install (не рекомендуется):
# sudo mkdir -p /usr/local/llama
# sudo cp bin/llama-* /usr/local/llama/
# sudo cp build/lib/*.so* /usr/local/llama/
# echo "/usr/local/llama" | sudo tee /etc/ld.so.conf.d/llama.conf
# sudo ldconfig
```

### Шаг 4: Проверка сборки

```bash
# Проверка наличия CUDA в бинарнике
strings bin/llama-cli | grep -i cuda | head -5

# Ожидаемый вывод:
# CUDA
# cuda_pool_malloc
# ggml_cuda_init
# ...

# Проверка версии
./bin/llama-cli --version
```

### Шаг 5: Установка

```bash
# Создание директории
sudo mkdir -p /usr/local/llama

# Копирование бинарников
sudo cp bin/llama-* /usr/local/llama/

# Создание симлинков в /usr/local/bin
sudo ln -sf /usr/local/llama/llama-server /usr/local/bin/llama-server
sudo ln -sf /usr/local/llama/llama-cli /usr/local/bin/llama-cli

# Проверка
which llama-server
llama-server --version
```

---

## Проверка работы GPU

### Тест 1: Проверка бинарника

```bash
# Запуск с тестовой моделью
/opt/llama.cpp/build/bin/llama-cli \
  -m /path/to/model.gguf \
  -p "Hello" \
  -n 10 \
  --n-gpu-layers 99

# В логах должно быть:
# llm_load_tensors: offloading XX layers to GPU
# llm_load_tensors: CUDA buffer size = XXXX MiB
```

### Тест 2: Мониторинг GPU

```bash
# В отдельном терминале запустите мониторинг
watch -n 1 nvidia-smi

# При загрузке модели GPU-Util должен увеличиться
# Memory-Usage должен показать использование VRAM
```

### Тест 3: Проверка offloading

При успешной загрузке на GPU вы увидите в логах:

```
llm_load_tensors: offloading 32 repeating layers to GPU
llm_load_tensors: offloaded 33/33 layers to GPU
llm_load_tensors:   CUDA0 buffer size =  5765.62 MiB
```

Если видите:
```
llm_load_tensors: offloaded 0/33 layers to GPU
llm_load_tensors:   CPU_Mapped model buffer size = XXXX MiB
```
→ GPU не используется, см. раздел диагностики.

---

## Конфигурация AIGateway

### Настройка yzma в конфиге для одной RTX 4090

```yaml
# configs/production.yaml

inference:
  gpu_layers: -1            # -1 = все слои на GPU (auto)
  
  yzma:
    enabled: true
    models_dir: "/data/models"
    
    # Безопасные настройки для GPU offload (Q4_K_M)
    context_size: 8192       # Начните с 8K, потом повышайте
    batch_size: 1024
    ubatch_size: 256
    
    # GPU настройки (v3.2.1+)
    main_gpu: 0             # Индекс основного GPU (0 = первый)
    flash_attention: true   # ОБЯЗАТЕЛЬНО для RTX 4090!
    
    # CPU threads для препроцессинга
    threads: 8              # 0 = auto (все ядра CPU)
    threads_batch: 8        # 0 = same as threads
```

### Настройка для DUAL RTX 4090 (48GB VRAM)

```yaml
# configs/production.yaml

inference:
  gpu_layers: -1            # Все слои на GPU
  
  yzma:
    enabled: true
    models_dir: "/data/models"
    
    # Шаг 1: стабильный старт (Q4_K_M)
    context_size: 16384      # 16K для стабильности
    batch_size: 2048
    ubatch_size: 512

    # Шаг 2: если всё ок — повышать
    # context_size: 32768
    # batch_size: 4096
    # ubatch_size: 1024
    
    # Multi-GPU настройки (v3.2.1+)
    main_gpu: 0             # Scratch buffers на первом GPU
    tensor_split: "0.5,0.5" # 50/50 распределение весов между GPU
    flash_attention: true
    
    threads: 16
    threads_batch: 16
```

### Параметры GPU (v3.2.1+)

| Параметр | Описание | Рекомендация |
|----------|----------|--------------|
| `inference.gpu_layers` | Слои на GPU | -1 (все) |
| `yzma.main_gpu` | Индекс основного GPU | 0 |
| `yzma.tensor_split` | Распределение между GPU | `"0.5,0.5"` для 2 GPU |
| `yzma.flash_attention` | Flash Attention v2 | **true** |
| `yzma.threads` | CPU threads | 0 (auto) или 8-16 |
| `yzma.threads_batch` | CPU threads для batch | 0 (= threads) |
| `yzma.context_size` | Размер контекста | 32768 (1 GPU) / 65536 (2 GPU) |
| `quant` (реком.) | Формат GGUF | **q4_K_M** (q8_0 часто остаётся на CPU) |

### Как работает `tensor_split`

```
tensor_split: ""          → Вся модель на main_gpu (GPU 0)
tensor_split: "0.5,0.5"   → 50% весов на GPU 0, 50% на GPU 1
tensor_split: "0.7,0.3"   → 70% на GPU 0, 30% на GPU 1
tensor_split: "0.33,0.33,0.34" → Равномерно на 3 GPU
```

**Важно:** `main_gpu` без `tensor_split` = вторая карта простаивает!

### Размеры моделей на RTX 4090

#### Одна RTX 4090 (24GB VRAM)

| Модель | Квантизация | VRAM | Context 4K | Context 32K |
|--------|-------------|------|------------|-------------|
| 7B | Q4_K_M | ~4.5 GB | ✅ | ✅ |
| 7B | Q8_0 | ~7.5 GB | ✅ | ✅ |
| 13B | Q4_K_M | ~8 GB | ✅ | ✅ |
| 22B | Q4_K_M | ~13 GB | ✅ | ✅ |
| 34B | Q4_K_M | ~20 GB | ✅ | ⚠️ (16K max) |
| 70B | Q4_K_M | ~40 GB | ❌ | ❌ |

#### Две RTX 4090 (48GB VRAM с tensor_split)

| Модель | Квантизация | VRAM | Context 4K | Context 64K |
|--------|-------------|------|------------|-------------|
| 70B | Q4_K_M | ~40 GB | ✅ | ✅ |
| 70B | Q5_K_M | ~48 GB | ✅ | ⚠️ (32K max) |
| 70B | Q8_0 | ~70 GB | ❌ | ❌ |
| 34B | Q8_0 | ~36 GB | ✅ | ✅ |

---

## Диагностика проблем

### Проблема: `nvidia-smi` не работает

```bash
# Проверка загрузки модуля
lsmod | grep nvidia

# Если пусто - драйвер не загружен
# Попробуйте:
sudo modprobe nvidia

# Если ошибка - переустановите драйвер
sudo dnf reinstall -y nvidia-driver nvidia-driver-cuda
sudo reboot
```

### Проблема: `Driver/library version mismatch`

```bash
# Это означает что ядро загружено со старым драйвером
# Решение: перезагрузка
sudo reboot

# Если не помогло - переустановка драйвера
sudo dnf remove -y nvidia-driver*
sudo dnf install -y nvidia-driver nvidia-driver-cuda
sudo reboot
```

### Проблема: `CMAKE_CUDA_COMPILER not found`

```bash
# Проверьте путь к nvcc
which nvcc
# Должно быть: /usr/local/cuda/bin/nvcc

# Если не найден - добавьте в PATH
export PATH=/usr/local/cuda/bin:$PATH

# Или явно укажите при cmake
cmake .. -DCMAKE_CUDA_COMPILER=/usr/local/cuda/bin/nvcc
```

### Проблема: `CUDA_ARCHITECTURES is set to "native", but no GPU was detected`

```bash
# Проверьте nvidia-smi
nvidia-smi

# Если ошибка - см. выше про Driver/library mismatch

# Если nvidia-smi работает но cmake не видит GPU:
# Явно укажите архитектуру
cmake .. -DCMAKE_CUDA_ARCHITECTURES=89  # Замените на вашу
```

### Проблема: `libstdc++.so.6: version GLIBCXX_3.4.30 not found`

```bash
# Это значит нужен более новый libstdc++

# Вариант 1: Использовать GCC Toolset
source /opt/rh/gcc-toolset-13/enable
export LD_LIBRARY_PATH=/opt/rh/gcc-toolset-13/root/usr/lib64:$LD_LIBRARY_PATH

# Вариант 2: Пересобрать llama.cpp с активированным GCC Toolset
source /opt/rh/gcc-toolset-13/enable
cd /opt/llama.cpp/build
rm -rf *
cmake .. -DGGML_CUDA=ON -DCMAKE_CUDA_ARCHITECTURES=89
cmake --build . --config Release -j $(nproc)

# Вариант 3: Статическая линковка (при сборке)
cmake .. -DGGML_CUDA=ON -DCMAKE_CUDA_ARCHITECTURES=89 \
  -DCMAKE_EXE_LINKER_FLAGS="-static-libstdc++ -static-libgcc"
```

### Проблема: GPU не используется (0 layers offloaded)

```bash
# 1. Проверьте что бинарник собран с CUDA
strings /opt/llama.cpp/build/bin/llama-cli | grep -i cuda
# Должны быть строки с CUDA

# 2. Проверьте что указан gpu_layers
# В конфиге или командной строке: --n-gpu-layers 99

# 3. Проверьте доступную VRAM
nvidia-smi
# Убедитесь что достаточно памяти для модели

# 4. Пересоберите с правильной архитектурой
cd /opt/llama.cpp/build
rm -rf *
cmake .. -DGGML_CUDA=ON -DCMAKE_CUDA_ARCHITECTURES=89
cmake --build . --config Release -j $(nproc)
```

### Проблема: Медленная генерация несмотря на GPU

```bash
# 1. Проверьте что все слои на GPU
# В логах должно быть: offloaded XX/XX layers to GPU

# 2. Включите Flash Attention (для Ampere и новее)
# В конфиге: flash_attention: true
# Или: --flash-attn

# 3. Увеличьте batch_size
# В конфиге: batch_size: 1024
# Или: --batch-size 1024

# 4. Проверьте Power Limit
nvidia-smi -q -d POWER
# Если ограничен - увеличьте:
sudo nvidia-smi -pl 450  # Для RTX 4090
```

---

## Полный скрипт установки

```bash
#!/bin/bash
# Rocky Linux 9 - llama.cpp с CUDA
# Запуск: sudo bash install_llamacpp_cuda.sh

set -e

GPU_ARCH="${1:-89}"  # По умолчанию RTX 40xx

echo "=== Установка зависимостей ==="
dnf update -y
dnf install -y epel-release
dnf groupinstall -y "Development Tools"
dnf install -y cmake git kernel-devel kernel-headers

echo "=== Установка NVIDIA драйвера ==="
dnf config-manager --add-repo https://developer.download.nvidia.com/compute/cuda/repos/rhel9/x86_64/cuda-rhel9.repo
dnf clean all
dnf install -y nvidia-driver nvidia-driver-cuda cuda-toolkit-12-4

echo "=== Установка GCC Toolset 13 ==="
dnf install -y gcc-toolset-13

echo "=== Настройка окружения ==="
cat >> /etc/profile.d/cuda.sh << 'EOF'
export CUDA_HOME=/usr/local/cuda
export PATH=$CUDA_HOME/bin:$PATH
export LD_LIBRARY_PATH=$CUDA_HOME/lib64:$LD_LIBRARY_PATH
EOF

source /etc/profile.d/cuda.sh
source /opt/rh/gcc-toolset-13/enable

echo "=== Клонирование llama.cpp ==="
cd /opt
git clone https://github.com/ggerganov/llama.cpp.git
cd llama.cpp

echo "=== Компиляция с CUDA (архитектура: $GPU_ARCH) ==="
mkdir -p build && cd build
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DGGML_CUDA=ON \
  -DCMAKE_CUDA_ARCHITECTURES=$GPU_ARCH \
  -DCMAKE_CUDA_COMPILER=/usr/local/cuda/bin/nvcc

cmake --build . --config Release -j $(nproc)

echo "=== Установка ==="
mkdir -p /usr/local/bin/llama
cp bin/* /usr/local/bin/llama/

echo "=== Готово! ==="
echo "После перезагрузки проверьте: nvidia-smi"
echo "Тест: /usr/local/bin/llama/llama-cli -m model.gguf -p 'Hello' -n 10 --n-gpu-layers 99"
```

---

## Полезные команды

```bash
# Мониторинг GPU в реальном времени
watch -n 1 nvidia-smi

# Детальная информация о GPU
nvidia-smi -q

# Проверка CUDA устройств
/usr/local/cuda/extras/demo_suite/deviceQuery

# Проверка версий библиотек
ldconfig -p | grep -E "cuda|nvidia"

# Логи загрузки модели с подробностями
llama-cli -m model.gguf -p "test" -n 1 --n-gpu-layers 99 --verbose
```

---

---

## Оптимизация для RTX 4090

### Настройка Power Limit

RTX 4090 по умолчанию имеет TDP 450W. Для серверного использования можно настроить:

```bash
# Проверка текущих настроек
nvidia-smi -q -d POWER

# Установка power limit (300W для тихой работы, 450W для максимума)
sudo nvidia-smi -pl 450

# Сделать постоянным (создать systemd service)
sudo tee /etc/systemd/system/nvidia-power.service << 'EOF'
[Unit]
Description=Set NVIDIA Power Limit
After=nvidia-persistenced.service

[Service]
Type=oneshot
ExecStart=/usr/bin/nvidia-smi -pl 450
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable nvidia-power.service
```

### Включение Persistence Mode

```bash
# Включить persistence mode (уменьшает латентность первого запроса)
sudo nvidia-smi -pm 1

# Сделать постоянным
sudo systemctl enable nvidia-persistenced
sudo systemctl start nvidia-persistenced
```

### Оптимальные параметры запуска llama-server для RTX 4090

```bash
# Запуск с оптимальными параметрами для RTX 4090
llama-server \
  --model /data/models/llama-3.1-70b-q4_k_m.gguf \
  --host 0.0.0.0 \
  --port 8080 \
  --n-gpu-layers 99 \
  --ctx-size 32768 \
  --batch-size 2048 \
  --ubatch-size 512 \
  --flash-attn \
  --threads 8 \
  --threads-batch 8 \
  --parallel 4 \
  --cont-batching
```

### Параметры для максимальной производительности

| Параметр | Значение | Описание |
|----------|----------|----------|
| `--n-gpu-layers` | 99 | Все слои на GPU |
| `--ctx-size` | 32768 | Размер контекста |
| `--batch-size` | 2048-4096 | Размер batch |
| `--ubatch-size` | 512 | Размер micro-batch |
| `--flash-attn` | enabled | Flash Attention v2 |
| `--parallel` | 4-8 | Параллельные запросы |
| `--cont-batching` | enabled | Continuous batching |

### Мониторинг в реальном времени

```bash
# Терминал 1: Мониторинг GPU
watch -n 0.5 nvidia-smi

# Терминал 2: Детальный мониторинг
nvidia-smi dmon -s pucvmet -d 1

# Терминал 3: Мониторинг температуры
nvidia-smi --query-gpu=temperature.gpu,utilization.gpu,utilization.memory,power.draw --format=csv -l 1
```

### Рекомендуемые модели для RTX 4090

| Модель | Размер | Квантизация | Описание |
|--------|--------|-------------|----------|
| Llama 3.1 8B | 8B | Q8_0 | Лучшее качество для размера |
| Llama 3.1 70B | 70B | Q4_K_M | С частичным offload на CPU |
| Qwen 2.5 32B | 32B | Q4_K_M | Полностью в VRAM |
| DeepSeek-V2-Lite | 16B | Q4_K_M | Отличный для code |
| Codestral 22B | 22B | Q4_K_M | Code completion |
| Mistral 7B | 7B | Q8_0 | Быстрый и качественный |

### Dual RTX 4090 Setup

Если у вас две RTX 4090:

```bash
# Проверка обеих GPU
nvidia-smi -L
# Ожидаемый вывод:
# GPU 0: NVIDIA GeForce RTX 4090 (UUID: ...)
# GPU 1: NVIDIA GeForce RTX 4090 (UUID: ...)
```

**Через конфиг AIGateway (рекомендуется):**

```yaml
inference:
  gpu_layers: -1
  yzma:
    tensor_split: "0.5,0.5"  # 50/50 между GPU
    main_gpu: 0
    flash_attention: true
    context_size: 65536      # 64K с двумя картами!
```

**Через CLI llama-server (для тестирования):**

```bash
llama-server \
  --model /data/models/llama-3.1-70b-q4_k_m.gguf \
  --n-gpu-layers 99 \
  --tensor-split 0.5,0.5 \
  --main-gpu 0 \
  --flash-attn \
  --ctx-size 65536
```

**При успешном запуске в логах:**
```
llm_load_tensors: tensor split: 0.500, 0.500
llm_load_tensors: offloading 80 repeating layers to GPU
llm_load_tensors: offloaded 81/81 layers to GPU
llm_load_tensors:   CUDA0 buffer size = 19200.00 MiB
llm_load_tensors:   CUDA1 buffer size = 19200.00 MiB
```

---

## Полный скрипт установки для RTX 4090

```bash
#!/bin/bash
# Rocky Linux 9 + RTX 4090 - llama.cpp с CUDA
# Запуск: sudo bash install_llamacpp_rtx4090.sh

set -e

echo "=== Установка для NVIDIA RTX 4090 на Rocky Linux 9 ==="
echo "=== CMAKE_CUDA_ARCHITECTURES=89 (Ada Lovelace) ==="

echo "=== Шаг 1: Обновление системы ==="
dnf update -y
dnf upgrade --refresh -y

echo "=== Шаг 2: Установка зависимостей ==="
dnf install -y epel-release
dnf groupinstall -y "Development Tools"
dnf install -y kernel-devel-$(uname -r) kernel-headers-$(uname -r) dkms cmake git

echo "=== Шаг 3: Отключение Nouveau ==="
cat > /etc/modprobe.d/blacklist-nouveau.conf << 'EOF'
blacklist nouveau
options nouveau modeset=0
EOF
grubby --args="nouveau.modeset=0 rd.driver.blacklist=nouveau" --update-kernel=ALL
dracut --force

echo "=== Шаг 4: Добавление NVIDIA репозитория ==="
dnf config-manager --add-repo https://developer.download.nvidia.com/compute/cuda/repos/rhel9/x86_64/cuda-rhel9.repo
dnf clean all
dnf makecache

echo "=== Шаг 5: Установка NVIDIA драйвера ==="
dnf module install -y nvidia-driver:latest-dkms

echo "=== Шаг 6: Установка CUDA Toolkit 13.1 ==="
dnf install -y cuda-toolkit-13-1

echo "=== Шаг 7: Настройка окружения ==="
cat > /etc/profile.d/cuda.sh << 'EOF'
export CUDA_HOME=/usr/local/cuda
export PATH=$CUDA_HOME/bin:$PATH
export LD_LIBRARY_PATH=$CUDA_HOME/lib64:$LD_LIBRARY_PATH
EOF

echo "=== Шаг 8: Установка GCC Toolset 13 ==="
dnf install -y gcc-toolset-13

echo "=== Шаг 9: Клонирование llama.cpp ==="
cd /opt
git clone https://github.com/ggerganov/llama.cpp.git
cd llama.cpp

echo "=== Шаг 10: Компиляция с CUDA для RTX 4090 ==="
source /etc/profile.d/cuda.sh
source /opt/rh/gcc-toolset-13/enable

mkdir -p build && cd build
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DGGML_CUDA=ON \
  -DCMAKE_CUDA_ARCHITECTURES=89 \
  -DCMAKE_CUDA_COMPILER=/usr/local/cuda/bin/nvcc \
  -DGGML_CUDA_F16=ON \
  -DGGML_CUDA_FA_ALL_QUANTS=ON

cmake --build . --config Release -j $(nproc)

echo "=== Шаг 11: Установка бинарников ==="
mkdir -p /usr/local/llama
cp bin/llama-* /usr/local/llama/
ln -sf /usr/local/llama/llama-server /usr/local/bin/llama-server
ln -sf /usr/local/llama/llama-cli /usr/local/bin/llama-cli

echo "=== Шаг 12: Настройка NVIDIA Persistence ==="
systemctl enable nvidia-persistenced
cat > /etc/systemd/system/nvidia-power.service << 'EOF'
[Unit]
Description=Set NVIDIA Power Limit
After=nvidia-persistenced.service

[Service]
Type=oneshot
ExecStart=/usr/bin/nvidia-smi -pm 1
ExecStart=/usr/bin/nvidia-smi -pl 450
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF
systemctl enable nvidia-power.service

echo ""
echo "=============================================="
echo "  УСТАНОВКА ЗАВЕРШЕНА!"
echo "=============================================="
echo ""
echo "  ТРЕБУЕТСЯ ПЕРЕЗАГРУЗКА: sudo reboot"
echo ""
echo "  После перезагрузки проверьте:"
echo "    nvidia-smi"
echo "    nvcc --version"
echo "    llama-server --version"
echo ""
echo "  Тест GPU:"
echo "    llama-cli -m model.gguf -p 'Hello' -n 10 --n-gpu-layers 99"
echo ""
echo "=============================================="
```

---

## Ссылки

- [llama.cpp GitHub](https://github.com/ggerganov/llama.cpp)
- [NVIDIA CUDA Toolkit](https://developer.nvidia.com/cuda-toolkit)
- [NVIDIA Driver Downloads](https://www.nvidia.com/Download/index.aspx)
- [Rocky Linux Documentation](https://docs.rockylinux.org/)
- [CUDA Compute Capability](https://developer.nvidia.com/cuda-gpus)
- [RTX 4090 Specifications](https://www.nvidia.com/en-us/geforce/graphics-cards/40-series/rtx-4090/)

