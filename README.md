# 🚀 AIGateway Platform v3.1

> **Enterprise AI Infrastructure with Multi-Provider Inference**  
> OpenAI-Compatible API • Docker-Based Inference • vLLM/SGLang/TGI • RAG System • Enterprise Security

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-3.1.0-brightgreen.svg)](VERSION)

---

## 📖 What is AIGateway?

**AIGateway v3.1** is a production-ready, enterprise-grade AI infrastructure platform with **multi-provider Docker-based inference**. Provides unified **OpenAI-compatible API** for models from Hugging Face with support for vLLM, SGLang, TGI, and llama.cpp backends.

### 🎯 Why AIGateway v3.1?

- **🐳 Docker-Based Inference**: vLLM, SGLang, TGI, llama.cpp — managed via Docker containers
- **🎮 Multi-GPU Support**: Tensor parallelism, automatic GPU selection, memory recommendations
- **🔍 HuggingFace Browser**: Search and load models directly from HF Hub with size/provider filters
- **📦 Format Support**: HF models (BF16/FP16/INT8/INT4), GGUF quantized models
- **🧠 RAG Built-in**: Retrieval-Augmented Generation with vector search, multimodal documents
- **🏢 Enterprise-Ready**: Multi-tenancy, RBAC, OIDC/LDAP, audit logging, quotas
- **🌐 Modern WebUI**: ChatGPT-like interface with model management, GPU monitoring
- **📊 Observability**: Prometheus metrics, container logs, health checks

---

## ✨ Key Features

### 🤖 Inference Providers (v3.1+)

| Provider | GPU | Quantization | Use Case |
|----------|-----|--------------|----------|
| **vLLM** | ✅ | BF16, FP16, AWQ, GPTQ | High-throughput production |
| **SGLang** | ✅ | BF16, FP16 | Research, embeddings |
| **TGI** | ✅ | BF16, FP16, GPTQ | HuggingFace native |
| **llama.cpp** | ✅ | GGUF (Q4-Q8) | CPU/Low VRAM inference |

### 🔍 HuggingFace Browser

- **Search Models**: Filter by provider (vLLM, SGLang, TGI, llama.cpp, Embeddings)
- **Size Filters**: < 3B, 3-7B, 7-14B, 14-30B, 30-70B, 70B+
- **Memory Recommendations**: Automatic GPU utilization calculation
- **One-Click Load**: Pre-fill form, configure, and start

### 🎮 GPU Management

- **Multi-GPU Selection**: Checkboxes with GPU name and memory info
- **Auto Tensor Parallel**: Automatically set based on selected GPUs
- **Memory Recommendations**: Calculate optimal `gpu_memory_utilization` and `max_model_len`
- **Container Logs**: Real-time streaming to `logs/containers/{alias}.log`

### 🧠 RAG System (v2.0.0)

- 📄 **Multi-Format Documents** - PDF, DOCX, CSV, TXT, Images с OCR
- 🔌 **External Data Sources** - REST APIs, PostgreSQL databases, Web scraping
- 🔍 **Vector Search** - PgVector с HNSW indexing
- 💬 **Chat Integration** - RAG toggle, source selector, Top-K controls

### 🔐 Enterprise Security

- 🔒 **JWT Authentication** - Access & Refresh tokens с rotation
- 👥 **Multi-Tenancy** - Organizations с membership и RBAC
- 🔑 **API Key Management** - Personal & Tenant keys с model-level permissions
- 🛡️ **OIDC/LDAP** - Enterprise SSO (Keycloak, Active Directory)

### 📊 Monitoring & Observability

- 📈 **Real-time Dashboard** - Backend status, Docker images, GPU metrics
- 🐳 **Container Management** - Start, stop, logs, health checks
- 📉 **Prometheus Integration** - Full metrics export
- 📋 **Provider Logs** - Detailed launch commands in `logs/providers/{alias}.log`

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ AIGateway Platform (Port 8080)                                  │
│                                                                 │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ OpenAI-Compatible API Layer                                 │ │
│ │ /v1/chat/completions, /v1/models, /v1/embeddings            │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                         │                                       │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ Inference Router & Orchestrator                             │ │
│ │  ├─ Model Registry (specs, status, health)                 │ │
│ │  ├─ Container Manager (start, stop, logs)                  │ │
│ │  └─ HuggingFace Downloader (cache management)              │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                         │                                       │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ Docker Runtime                                              │ │
│ │  ├─ vLLM Container (GPU, Tensor Parallel)                  │ │
│ │  ├─ SGLang Container (GPU, Embeddings)                     │ │
│ │  ├─ TGI Container (GPU, Sharding)                          │ │
│ │  └─ llama.cpp Container (GGUF, CPU/GPU)                    │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ RAG System                                                  │ │
│ │  ├─ Document Processing Pipeline                           │ │
│ │  ├─ Vector Store (PgVector)                                │ │
│ │  └─ Embeddings (Ollama/SGLang)                             │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ Enterprise Features                                         │ │
│ │  ├─ Multi-Tenancy & RBAC                                   │ │
│ │  ├─ OIDC/LDAP Integration                                  │ │
│ │  └─ Audit Logging & Rate Limiting                          │ │
│ └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.25+** (для сборки)
- **Docker** with NVIDIA Container Toolkit (для GPU inference)
- **PostgreSQL 13+** с pgvector extension (для RAG)
- **NVIDIA GPU** (для vLLM/SGLang/TGI)

### Pre-pull Docker Images

```bash
# vLLM (recommended for production)
docker pull vllm/vllm-openai:latest

# SGLang (research, embeddings)
docker pull lmsysorg/sglang:latest

# TGI (HuggingFace native)
docker pull ghcr.io/huggingface/text-generation-inference:latest

# llama.cpp (GGUF models)
docker pull ghcr.io/ggml-org/llama.cpp:server-cuda
```

### Installation

#### Option 1: Docker Compose (Recommended)

```bash
git clone https://github.com/yourusername/aigateway.git
cd aigateway

# Start infrastructure (PostgreSQL, Redis, etc.)
docker compose -f infra/docker-compose.yml up -d

# Build & run server
go build -o bin/server cmd/server/main.go
./bin/server -config configs/dev.yaml
```

#### Option 2: Build from Source

```bash
git clone https://github.com/yourusername/aigateway.git
cd aigateway

# Install dependencies
go mod tidy

# Build
./build.ps1 all  # Windows
./build.sh all   # Linux/macOS

# Run
./dist/server -config configs/production.yaml
```

### First Run

1. **Access WebUI**: [http://localhost:8080/login](http://localhost:8080/login)

2. **Default credentials**:
   - Email: `admin@localhost`
   - Password: `admin`

3. **Load a model**:
   - Go to **Admin → Models → HuggingFace**
   - Search for a model (e.g., "Qwen2.5-7B")
   - Select provider filter (vLLM, SGLang, etc.)
   - Click **Use**, configure GPU settings, click **Load & Start**

4. **Chat with model**:
   - Go to **Chat**
   - Select loaded model from dropdown
   - Start chatting!

---

## 🔧 Configuration

### Basic Configuration

```yaml
# configs/production.yaml
server:
  host: "0.0.0.0"
  port: 8080

# Docker-based inference
inference:
  enabled: true
  hf_cache_dir: "data/models/hf"
  gguf_cache_dir: "data/models/gguf"
  health_check_timeout: 60m
  startup_timeout: 60m
  docker:
    api: true
    socket: "unix:///var/run/docker.sock"

# HuggingFace token (for gated models)
huggingface:
  token: "hf_xxx..."

# RAG System
rag:
  enabled: true
  vector_store:
    type: "pgvector"
    connection_string: "postgresql://user:pass@localhost:5432/aigateway"

# OIDC (Keycloak example)
oidc:
  enabled: true
  issuer_url: "http://keycloak:8180/realms/master"
  client_id: "aigateway"
  client_secret: "xxx"
```

### GPU Settings for Large Models

| Model Size | GPUs | Tensor Parallel | GPU Utilization | Max Model Len |
|------------|------|-----------------|-----------------|---------------|
| 7B BF16    | 1    | 1               | 0.9             | 32768         |
| 14B BF16   | 1    | 1               | 0.95            | 16384         |
| 30B INT8   | 2    | 2               | 0.9             | 32768         |
| 70B INT4   | 2    | 2               | 0.95            | 8192          |

---

## 📚 API Reference

### OpenAI-Compatible Endpoints

```bash
# List models
curl http://localhost:8080/v1/models \
  -H "Authorization: Bearer sk-xxx"

# Chat completion
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen2-5-7b-instruct",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'

# Embeddings
curl http://localhost:8080/v1/embeddings \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mxbai-embed-large",
    "input": "Hello world"
  }'
```

### Model Management Endpoints

```bash
# Load model
curl -X POST http://localhost:8080/api/system/inference/load \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "alias": "qwen-7b",
    "provider": "vllm",
    "format": "hf",
    "hf_repo": "Qwen/Qwen2.5-7B-Instruct",
    "capabilities": ["chat"],
    "vllm_tensor_parallel": 1,
    "vllm_gpu_utilization": 0.9,
    "vllm_max_model_len": 32768
  }'

# List running models
curl http://localhost:8080/api/system/inference/models

# Stop model
curl -X POST http://localhost:8080/api/system/inference/stop/qwen-7b

# Get container logs
curl http://localhost:8080/api/system/inference/logs/qwen-7b
```

---

## 🧪 Testing

```bash
# Unit tests
go test ./...

# With race detector
go test -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 📊 Troubleshooting

### Common Issues

**1. OOM Error (Out of Memory)**
```
ValueError: No available memory for the cache blocks
```
Solution: Reduce `gpu_memory_utilization` or `max_model_len`

**2. NCCL Error (Multi-GPU)**
```
NCCL error: unhandled system error
```
Solution: Ensure Docker uses `--ipc=host`, `--shm-size=16g`

**3. Model not found in chat**
```
model does not exist
```
Solution: Check that model alias matches, or use full HFRepo name

**4. Context canceled during load**
```
context canceled
```
Solution: Don't refresh page during model loading (takes 2-10 min for large models)

### Useful Commands

```bash
# Check GPU memory
nvidia-smi

# Kill all inference containers
docker ps -a | grep aigw- | awk '{print $1}' | xargs -r docker rm -f

# View container logs
docker logs -f <container_id>

# Check server logs
tail -f logs/proxy-dev.log
```

---

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

```bash
# Development setup
git clone https://github.com/yourusername/aigateway.git
cd aigateway
go mod tidy
go run cmd/server/main.go -config configs/dev.yaml
```

---

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- [vLLM](https://vllm.ai/) - Fast inference engine
- [SGLang](https://github.com/sgl-project/sglang) - Research inference
- [TGI](https://github.com/huggingface/text-generation-inference) - HuggingFace inference
- [llama.cpp](https://github.com/ggerganov/llama.cpp) - GGUF inference
- [Gin](https://gin-gonic.com/) - HTTP framework
- [pgvector](https://github.com/pgvector/pgvector) - Vector similarity search

---

**Built with ❤️ for the open-source AI community**
