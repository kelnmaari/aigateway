# 🚀 vLLM Installation & Configuration Guide

**Target:** Linux Server with 2x NVIDIA GPU (e.g., 2x RTX 3080)  
**Version:** vLLM 0.3.0+  
**AIGateway Version:** 2.2.0+

---

## 📋 System Requirements

###Hardware
- **GPU:** 2x NVIDIA GPU с CUDA support
  - Tested: RTX 3080 (10GB each), RTX 4090, A100
  - Minimum: 2x 8GB VRAM
  - Recommended: 2x 10GB+ VRAM
- **RAM:** 32GB+ system RAM
- **Storage:** 100GB+ для models cache

### Software
- **OS:** Ubuntu 20.04+, Debian 11+, or RHEL 8+
- **Python:** 3.8 - 3.11 (3.11 recommended)
- **CUDA:** 11.8+ или 12.1+ (12.1 recommended)
- **NVIDIA Driver:** 525.60.13+ (latest recommended)

---

## 🔍 Step 0: Verify Prerequisites

### Check NVIDIA Drivers

```bash
# Should show both GPUs
nvidia-smi

# Expected output:
# +-----------------------------------------------------------------------------+
# | NVIDIA-SMI 535.129.03   Driver Version: 535.129.03   CUDA Version: 12.2    |
# |-------------------------------+----------------------+----------------------+
# | GPU  Name        Persistence-M| Bus-Id        Disp.A | Volatile Uncorr. ECC |
# | Fan  Temp  Perf  Pwr:Usage/Cap|         Memory-Usage | GPU-Util  Compute M. |
# |                               |                      |               MIG M. |
# |===============================+======================+======================|
# |   0  NVIDIA GeForce ... On   | 00000000:01:00.0 Off |                  N/A |
# | 30%   45C    P8    15W / 320W |      0MiB / 10240MiB |      0%      Default |
# +-------------------------------+----------------------+----------------------+
# |   1  NVIDIA GeForce ... On   | 00000000:02:00.0 Off |                  N/A |
# | 30%   43C    P8    14W / 320W |      0MiB / 10240MiB |      0%      Default |
# +-------------------------------+----------------------+----------------------+
```

### Check CUDA Installation

```bash
# Verify CUDA version
nvcc --version

# Expected output:
# nvcc: NVIDIA (R) Cuda compiler driver
# ...
# Cuda compilation tools, release 12.1, V12.1.105
```

### If CUDA Not Installed

```bash
# Ubuntu/Debian
wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.1-1_all.deb
sudo dpkg -i cuda-keyring_1.1-1_all.deb
sudo apt-get update
sudo apt-get -y install cuda-toolkit-12-1

# Add to PATH
echo 'export PATH=/usr/local/cuda-12.1/bin:$PATH' >> ~/.bashrc
echo 'export LD_LIBRARY_PATH=/usr/local/cuda-12.1/lib64:$LD_LIBRARY_PATH' >> ~/.bashrc
source ~/.bashrc
```

---

## 🐍 Step 1: Python Environment Setup

### Install Python 3.11 (if needed)

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y python3.11 python3.11-venv python3.11-dev

# Verify installation
python3.11 --version  # Should show Python 3.11.x
```

### Create Virtual Environment

```bash
# Create dedicated environment для vLLM
mkdir -p ~/vllm-server
cd ~/vllm-server

python3.11 -m venv venv
source venv/bin/activate

# Upgrade pip
pip install --upgrade pip setuptools wheel
```

---

## 📦 Step 2: Install vLLM

### Install vLLM with CUDA 12.1

```bash
# Activate venv if not already
source ~/vllm-server/venv/bin/activate

# Install vLLM (this will take 5-10 minutes)
pip install vllm

# Verify installation
python -c "import vllm; print(f'vLLM version: {vllm.__version__}')"

# Test CUDA availability
python -c "import torch; print(f'CUDA available: {torch.cuda.is_available()}'); print(f'GPU count: {torch.cuda.device_count()}')"

# Expected output:
# vLLM version: 0.3.1
# CUDA available: True
# GPU count: 2
```

### Install Additional Dependencies

```bash
# For HuggingFace models
pip install transformers accelerate

# For model management
pip install huggingface-hub
```

---

## 🤗 Step 3: HuggingFace Setup (Optional but Recommended)

### Login to HuggingFace Hub

```bash
# Required для gated models (Llama-2, Mistral, etc)
huggingface-cli login

# Paste your HF token from https://huggingface.co/settings/tokens
# Token will be saved in ~/.cache/huggingface/token
```

### Pre-download Models (Optional)

```bash
# Option 1: Using Python
python -c "
from transformers import AutoModelForCausalLM, AutoTokenizer

# Download model (will cache in ~/.cache/huggingface/)
model_name = 'mistralai/Mistral-7B-Instruct-v0.2'
tokenizer = AutoTokenizer.from_pretrained(model_name)
model = AutoModelForCausalLM.from_pretrained(model_name)
print(f'Downloaded {model_name}')
"

# Option 2: Using huggingface-cli
huggingface-cli download mistralai/Mistral-7B-Instruct-v0.2
```

---

## 🚀 Step 4: Launch vLLM Server

### Basic Launch (2x GPU, Tensor Parallelism)

```bash
# Terminal 1: Start vLLM Server
cd ~/vllm-server
source venv/bin/activate

# Set both GPUs visible
export CUDA_VISIBLE_DEVICES=0,1

# Launch vLLM with OpenAI-compatible API
python -m vllm.entrypoints.openai.api_server \
    --model mistralai/Mistral-7B-Instruct-v0.2 \
    --tensor-parallel-size 2 \
    --gpu-memory-utilization 0.9 \
    --host 0.0.0.0 \
    --port 8000 \
    --served-model-name mistral-7b-instruct

# Expected logs:
# INFO:     Started server process
# INFO:     Waiting for application startup.
# INFO:     Initializing model with tensor parallelism size 2
# INFO:     Loading model weights from mistralai/Mistral-7B-Instruct-v0.2
# INFO:     Using 2 GPUs for tensor parallelism
# INFO:     Model loaded successfully
# INFO:     Application startup complete.
# INFO:     Uvicorn running on http://0.0.0.0:8000
```

### Advanced Launch Options

```bash
# For better performance
python -m vllm.entrypoints.openai.api_server \
    --model mistralai/Mistral-7B-Instruct-v0.2 \
    --tensor-parallel-size 2 \
    --gpu-memory-utilization 0.9 \
    --max-model-len 4096 \
    --max-num-seqs 256 \
    --trust-remote-code \
    --host 0.0.0.0 \
    --port 8000
```

### Explanation of Parameters

| Parameter | Description | Recommendation |
|-----------|-------------|----------------|
| `--tensor-parallel-size 2` | Split model across 2 GPUs | Use 2 для 2x GPU |
| `--gpu-memory-utilization 0.9` | Use 90% of VRAM | 0.85-0.95 |
| `--max-model-len 4096` | Max context length | Model dependent |
| `--max-num-seqs 256` | Max concurrent requests | Start with 256 |
| `--trust-remote-code` | Allow custom model code | For some HF models |

---

## ✅ Step 5: Test vLLM Server

### Test 1: List Available Models

```bash
# Terminal 2 (new terminal)
curl http://localhost:8000/v1/models

# Expected response:
# {
#   "object": "list",
#   "data": [
#     {
#       "id": "mistral-7b-instruct",
#       "object": "model",
#       "created": 1234567890,
#       "owned_by": "vllm"
#     }
#   ]
# }
```

### Test 2: Chat Completion

```bash
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mistral-7b-instruct",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant."},
      {"role": "user", "content": "What is tensor parallelism?"}
    ],
    "temperature": 0.7,
    "max_tokens": 100
  }'
```

### Test 3: Streaming Response

```bash
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mistral-7b-instruct",
    "messages": [{"role": "user", "content": "Count to 10"}],
    "stream": true
  }'
```

---

## 🔧 Step 6: Configure AIGateway Integration

### Update AIGateway Configuration

```yaml
# configs/dev.yaml

# Model Providers
providers:
  # Existing Ollama
  ollama:
    enabled: true
    base_url: "http://localhost:11434"
    timeout: 120s
  
  # NEW: vLLM Provider
  vllm:
    enabled: true
    base_url: "http://localhost:8000"
    api_key: ""  # vLLM doesn't require API key by default
    timeout: 120s
    
    # Performance tuning
    gpu_memory_utilization: 0.9
    max_model_len: 4096
    tensor_parallel_size: 2
```

### Restart AIGateway

```bash
# Stop AIGateway if running
pkill aigateway

# Start with new config
./bin/server -config configs/dev.yaml
```

---

## 📊 Step 7: Verify Integration

### Test via AIGateway

```bash
# Should now route to vLLM provider
curl http://localhost:8080/api/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "model": "mistral-7b-instruct",
    "messages": [{"role": "user", "content": "Hello from AIGateway!"}]
  }'
```

### Check Model Registry

```bash
# List all models (should include vLLM models)
curl http://localhost:8080/api/models/registry \
  -H "Authorization: Bearer YOUR_API_KEY"

# Expected to see both Ollama and vLLM models
```

---

## 🎯 Recommended Models for 2x RTX 3080 (10GB each)

### Small Models (Fast, Development)

| Model | Size | VRAM | Speed | Best For |
|-------|------|------|-------|----------|
| **Mistral-7B-Instruct-v0.2** ⭐ | 7B | ~8GB | ⚡⚡⚡⚡⚡ | General chat, coding |
| TinyLlama-1.1B-Chat | 1B | ~2GB | ⚡⚡⚡⚡⚡ | Testing, demos |
| Phi-2 | 2.7B | ~4GB | ⚡⚡⚡⚡⚡ | Reasoning, coding |

### Medium Models (Production)

| Model | Size | VRAM | Speed | Best For |
|-------|------|------|-------|----------|
| **Llama-2-7B-Chat** ⭐ | 7B | ~8GB | ⚡⚡⚡⚡ | General purpose |
| **CodeLlama-7B-Instruct** ⭐ | 7B | ~8GB | ⚡⚡⚡⚡ | Code generation |
| Llama-2-13B-Chat | 13B | ~16GB | ⚡⚡⚡ | Better quality |

### Large Models (Quality)

| Model | Size | VRAM | Speed | Best For |
|-------|------|------|-------|----------|
| Mixtral-8x7B-Instruct | 47B (MoE) | ~19GB | ⚡⚡ | Best quality |

⭐ = Recommended для начала

---

## 🐳 Step 8: Docker Compose Setup (Optional)

### Create docker-compose.yml

```yaml
version: '3.8'

services:
  # AIGateway
  aigateway:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./configs:/app/configs
      - ./data:/app/data
    environment:
      - CONFIG_PATH=/app/configs/dev.yaml
    depends_on:
      - ollama
      - vllm

  # Ollama
  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama-data:/root/.ollama
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]

  # vLLM
  vllm:
    image: vllm/vllm-openai:latest
    ports:
      - "8000:8000"
    volumes:
      - vllm-cache:/root/.cache/huggingface
    environment:
      - CUDA_VISIBLE_DEVICES=0,1
      - HF_TOKEN=${HF_TOKEN}  # Optional, для gated models
    command: >
      --model mistralai/Mistral-7B-Instruct-v0.2
      --tensor-parallel-size 2
      --gpu-memory-utilization 0.9
      --host 0.0.0.0
      --port 8000
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 2
              capabilities: [gpu]
    shm_size: '10gb'  # Important для large models

volumes:
  ollama-data:
  vllm-cache:
```

### Launch Stack

```bash
# Set HuggingFace token (if needed)
export HF_TOKEN=your_token_here

# Start all services
docker-compose up -d

# Check logs
docker-compose logs -f vllm

# Stop services
docker-compose down
```

---

## 🔍 Troubleshooting

### Issue: OutOfMemoryError

```bash
# Solution 1: Reduce GPU memory utilization
--gpu-memory-utilization 0.85  # instead of 0.9

# Solution 2: Reduce max model length
--max-model-len 2048  # instead of 4096

# Solution 3: Use smaller model
# Try Mistral-7B instead of Llama-13B
```

### Issue: CUDA initialization failed

```bash
# Check NVIDIA drivers
nvidia-smi

# Reinstall CUDA toolkit if needed
sudo apt install --reinstall nvidia-cuda-toolkit

# Verify PyTorch CUDA
python -c "import torch; print(torch.cuda.is_available())"
```

### Issue: Model not found / Download fails

```bash
# Check HuggingFace cache
ls -lh ~/.cache/huggingface/hub/

# Manual download
huggingface-cli download mistralai/Mistral-7B-Instruct-v0.2

# Check disk space
df -h ~/.cache
```

### Issue: Port already in use

```bash
# Check what's using port 8000
sudo lsof -i :8000

# Kill process if needed
sudo kill -9 <PID>

# Or use different port
--port 8001
```

### Issue: Slow inference

```bash
# Check GPU utilization
nvidia-smi -l 1

# Should see high GPU utilization (>80%) during inference
# If low, check:
# 1. Tensor parallelism configured correctly
# 2. Model fully loaded on GPUs
# 3. No CPU fallback happening
```

---

## 📈 Performance Benchmarks

### 2x RTX 3080 (10GB each)

| Model | Batch Size | Tokens/sec | Latency (ms) |
|-------|------------|------------|--------------|
| Mistral-7B | 1 | 45-55 | ~50 |
| Mistral-7B | 8 | 280-320 | ~80 |
| Llama-2-7B | 1 | 40-50 | ~55 |
| Llama-2-13B | 1 | 25-35 | ~85 |

*Benchmarks для context_len=2048, max_tokens=512*

---

## 🎓 Next Steps

1. ✅ Test basic inference via vLLM
2. ✅ Configure AIGateway integration
3. ✅ Load multiple models (Ollama + vLLM)
4. 📊 Monitor GPU utilization (`nvidia-smi`)
5. 🚀 Test production workload
6. 📈 Optimize parameters для your use case

---

## 📚 Additional Resources

- [vLLM Official Docs](https://docs.vllm.ai/)
- [HuggingFace Models](https://huggingface.co/models)
- [CUDA Installation Guide](https://docs.nvidia.com/cuda/cuda-installation-guide-linux/)
- [AIGateway Documentation](../README.md)

---

**Questions or Issues?** Open an issue on GitHub or check Discord community.

