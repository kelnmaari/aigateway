# 🚀 AIGateway Platform v4.0

> **Enterprise AI Infrastructure with Multi-Provider Inference & Web Search**  
> OpenAI-Compatible API • Docker-Based Inference • vLLM/SGLang/TGI/llama.cpp • RAG System • GitLab Code Review • Tavily Web Search

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-4.5.0-brightgreen.svg)](VERSION)

---

## 📖 What is AIGateway?

**AIGateway v4.0** is a production-ready, enterprise-grade AI infrastructure platform with **multi-provider Docker-based inference**. Provides unified **OpenAI-compatible API** for models from Hugging Face with support for vLLM, SGLang, TGI, and llama.cpp backends.

### 🎯 Why AIGateway v4.0?

- **🐳 Docker-Based Inference**: vLLM, SGLang, TGI, llama.cpp — managed via Docker containers
- **🔍 Web Search (NEW!)**: Tavily API integration for real-time internet search in ChatUI
- **🎮 Multi-GPU Support**: Tensor parallelism, automatic GPU selection, memory recommendations
- **🔍 HuggingFace Browser**: Search and load models directly from HF Hub with size/provider filters
- **📦 Format Support**: HF models (BF16/FP16/INT8/INT4), GGUF quantized models
- **🧠 RAG Built-in**: Retrieval-Augmented Generation with vector search, multimodal documents
- **🦊 GitLab Integration**: AI-powered MR code reviews with per-file analysis
- **🏢 Enterprise-Ready**: Multi-tenancy, RBAC, OIDC/LDAP, audit logging, quotas
- **🌐 Modern WebUI**: ChatGPT-like interface with model management, GPU monitoring
- **📦 RPM Packages**: One-line installation for Rocky Linux / RHEL

---

## ✨ What's New in v4.0

### 🔍 Web Search in ChatUI
- **Tavily API Integration**: Real-time web search during chat
- **Visual Tool Events**: See "Searching..." → "Searched (1237ms)" like Cursor IDE
- **Toggle Control**: Enable/disable web search per session

### 🦊 GitLab MR Reviews
- **Per-File Analysis**: Each file reviewed separately with tool calling
- **Code Context**: Automatic retrieval of related functions from Qdrant
- **Multi-Language**: Review comments in Russian or English
- **Configurable Tokens**: Set `MaxReviewTokens` per project

### 🎯 Extended Model Capabilities
- **New Capabilities**: `autocomplete`, `edit`, `apply`, `rerank`
- **Auto-Expansion**: Select `chat` → adds `autocomplete`, `edit`, `apply`
- **Edit Saved Models**: Modify capabilities for existing saved models

### 📊 Container Logs UI
- **Real-time Logs**: View container output in large modal (80% screen)
- **Log Coloring**: Colorized output based on log level
- **Auto-refresh**: Continuous updates every 2 seconds

### 📦 RPM Packaging
- **One-line Install**: `curl -fsSL .../install.sh | sudo bash`
- **SystemD Service**: Auto-start, restart on failure
- **YUM Repository**: GitLab Package Registry integration

---

## 🤖 Inference Providers

| Provider | GPU | Quantization | Use Case |
|----------|-----|--------------|----------|
| **vLLM** | ✅ | BF16, FP16, AWQ, GPTQ | High-throughput production |
| **SGLang** | ✅ | BF16, FP16 | Research, embeddings |
| **TGI** | ✅ | BF16, FP16, GPTQ | HuggingFace native |
| **llama.cpp** | ✅ | GGUF (Q4-Q8) | CPU/Low VRAM inference |
| **TEI** | ✅ | — | Text embeddings |

### llama.cpp Parameters (v4.0+)
- `ctx_size`: Context window size (default: model's max)
- `n_parallel`: Concurrent request slots
- `flash_attn`: Enable Flash Attention for faster inference

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ AIGateway Platform (Port 8080)                                  │
│                                                                 │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ OpenAI-Compatible API Layer                                 │ │
│ │ /v1/chat/completions, /v1/models, /v1/embeddings            │ │
│ │ /api/chat/completions (with web search tools)               │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                         │                                       │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ Inference Router & Orchestrator                             │ │
│ │  ├─ Model Registry (specs, status, health)                 │ │
│ │  ├─ Container Manager (start, stop, logs)                  │ │
│ │  ├─ HuggingFace Downloader (cache management)              │ │
│ │  └─ Tools Registry (Tavily web search)                     │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                         │                                       │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ Docker Runtime                                              │ │
│ │  ├─ vLLM Container (GPU, Tensor Parallel)                  │ │
│ │  ├─ SGLang Container (GPU, Embeddings)                     │ │
│ │  ├─ TGI Container (GPU, Sharding)                          │ │
│ │  ├─ llama.cpp Container (GGUF, CPU/GPU)                    │ │
│ │  └─ TEI Container (Embeddings)                             │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ Integrations                                                │ │
│ │  ├─ GitLab MR Reviews (per-file analysis)                  │ │
│ │  ├─ RAG System (PgVector, Qdrant)                          │ │
│ │  └─ Web Search (Tavily API)                                │ │
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

### Installation

#### Option 1: RPM Package (Rocky Linux / RHEL)

```bash
# One-line install
curl -fsSL https://gitlab.example.com/api/v4/projects/XXX/packages/generic/aigateway/latest/install.sh | sudo bash

# Start service
sudo systemctl start oop
sudo systemctl enable oop
```

#### Option 2: Build from Source

```bash
git clone https://gitlab.alexue4.dev/KelnMaari/ollama-openai-proxy.git
cd ollama-openai-proxy

# Build with WebUI
./build-new.ps1 all -WebUI svelte  # Windows
./build.sh all                      # Linux

# Run
./bin/server -config configs/dev.yaml
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
   - Enable **Web Search** toggle for internet access
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

# Tools Configuration (v4.0+)
tools:
  # Tavily Web Search API key
  # Get your key at https://tavily.com
  tavily_api_key: "tvly-xxx..."

# RAG System
rag:
  enabled: true
  vector_store:
    type: "pgvector"
    connection_string: "postgresql://user:pass@localhost:5432/aigateway"

# GitLab Integration
gitlab:
  enabled: true
  default_model: "qwen-7b"
  embedding_model: "nomic-embed-text"
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

# Chat with web search (v4.0+)
curl http://localhost:8080/api/chat/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "qwen-7b",
    "messages": [{"role": "user", "content": "What is the latest news about AI?"}],
    "use_tools": true,
    "stream": true
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
    "capabilities": ["chat", "autocomplete", "edit", "apply"],
    "vllm_tensor_parallel": 1,
    "vllm_gpu_utilization": 0.9,
    "vllm_max_model_len": 32768
  }'

# List running models
curl http://localhost:8080/api/system/inference/models

# Get container logs
curl "http://localhost:8080/api/system/inference/logs?alias=qwen-7b&tail=100"

# Update saved model capabilities
curl -X POST "http://localhost:8080/api/system/inference/update-saved?alias=qwen-7b" \
  -H "Content-Type: application/json" \
  -d '{"capabilities": ["chat", "autocomplete", "edit", "apply"]}'
```

---

## 🔍 Web Search Integration

### How It Works

1. User enables "Use web search" toggle in ChatUI
2. Request goes to `/api/chat/completions` with `use_tools: true`
3. Model can call `web_search` tool when it needs current information
4. Tavily API returns search results
5. Results are injected into context, model generates final response

### Tool Events in UI

During streaming, you'll see Cursor-like tool events:
- 🔄 **Searching** "query..." (while searching)
- ✅ **Searched** "query" (1237ms) • Found 5 results

### Configuration

```yaml
tools:
  tavily_api_key: "tvly-xxx..."  # Get at https://tavily.com
```

---

## 🦊 GitLab Integration

### Features

- **MR Code Review**: AI-powered analysis of merge requests
- **Per-File Review**: Each file analyzed separately
- **Tool Calling**: Model can search codebase for context
- **Multi-Language**: Comments in Russian or English

### Setup

1. Add GitLab integration in Admin → GitLab
2. Configure project with analysis and embedding models
3. Index repository (creates Qdrant collection)
4. Reviews trigger automatically on MR creation

---

## 📊 Troubleshooting

### Common Issues

**1. OOM Error (Out of Memory)**
```
ValueError: No available memory for the cache blocks
```
Solution: Reduce `gpu_memory_utilization` or `max_model_len`

**2. Web Search Not Working**
```
web search not configured
```
Solution: Add `tavily_api_key` to `tools` section in config

**3. Context Canceled During Load**
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

We welcome contributions! 

```bash
# Development setup
git clone https://gitlab.alexue4.dev/KelnMaari/ollama-openai-proxy.git
cd ollama-openai-proxy
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
- [Tavily](https://tavily.com/) - Web search API
- [Gin](https://gin-gonic.com/) - HTTP framework
- [pgvector](https://github.com/pgvector/pgvector) - Vector similarity search

---

**Built with ❤️ for the open-source AI community**
