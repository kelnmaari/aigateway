# AIGateway Platform v5

> **Enterprise AI Infrastructure with Multi-Provider Inference, Distributed Workers & External Providers**
> OpenAI-Compatible API | Docker-Based Inference | Agent Mode (Remote Workers) | vLLM/SGLang/TGI/TEI/llama.cpp | RAG | GitLab Code Review | Web Search

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-5.4.9-brightgreen.svg)](VERSION)

---

## What is AIGateway?

AIGateway is a production-ready AI infrastructure platform that provides a unified **OpenAI-compatible API** for managing LLM inference across local Docker containers, remote GPU/CPU workers, and external providers (OpenAI, Anthropic, Gemini, DeepSeek).

### Key Features

- **Docker-Based Inference**: vLLM, SGLang, TGI, TEI, llama.cpp, TensorRT-LLM via Docker containers
- **Agent Mode (v5.4+)**: Distribute inference across remote GPU/CPU workers with centralized management
- **External Providers**: OpenAI, Anthropic, Gemini, DeepSeek via Model Registry
- **Multi-GPU**: Tensor parallelism, per-GPU assignment, VRAM monitoring
- **HuggingFace Browser**: Search and load models from HF Hub with size/provider filters
- **Format Support**: HF models (BF16/FP16/AWQ/GPTQ/FP8), GGUF, TensorRT engines
- **RAG System**: Vector search with Qdrant/PgVector, document processing pipeline
- **GitLab Integration**: AI-powered MR code reviews with per-file analysis
- **Web Search**: Tavily API for real-time internet search in Chat UI
- **Enterprise**: Multi-tenancy, RBAC, OIDC/LDAP, audit logging, quotas
- **Modern WebUI**: SvelteKit 5 + Tailwind v4, dark mode, i18n (EN/RU)
- **RPM Packages**: One-line install for Rocky Linux / RHEL 9

---

## Architecture

```
                              ┌──────────────────────────┐
                              │   External Providers     │
                              │ OpenAI, Anthropic, etc.  │
                              └──────────┬───────────────┘
                                         │
┌────────────────────────────────────────────────────────────────────┐
│ AIGateway Main Server (Port 8085)                                  │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ OpenAI-Compatible API Layer                                  │  │
│  │ /v1/chat/completions  /v1/models  /v1/embeddings             │  │
│  │ Anthropic→OpenAI message conversion (tool_use/tool_result)   │  │
│  │ SSE streaming with reasoning_content merge                   │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                           │                                        │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ Inference Router & Orchestrator                              │  │
│  │  ├─ Local: Docker containers on this server                 │  │
│  │  ├─ Remote: Agent workers (GPU/CPU machines)                │  │
│  │  └─ External: Model Registry (OpenAI, Anthropic, etc.)     │  │
│  └──────────────────────────────────────────────────────────────┘  │
│           │                              │                         │
│  ┌────────┴─────────┐     ┌──────────────┴──────────────────┐     │
│  │ Local Docker      │     │ Agent Manager                   │     │
│  │  ├─ vLLM          │     │  ├─ Health checks (30s)        │     │
│  │  ├─ SGLang        │     │  ├─ GPU/CPU monitoring         │     │
│  │  ├─ TGI / TEI     │     │  ├─ Model sync                │     │
│  │  ├─ llama.cpp     │     │  └─ Image push (docker save)   │     │
│  │  └─ TensorRT-LLM  │     └──────────────┬──────────────────┘     │
│  └──────────────────┘                      │                       │
│                                            │ HTTPS + API Key       │
│  ┌──────────────────────────────────────┐  │                       │
│  │ Integrations                         │  │                       │
│  │  ├─ GitLab MR Reviews               │  │                       │
│  │  ├─ RAG (Qdrant, document pipeline)  │  │                       │
│  │  ├─ Web Search (Tavily)              │  │                       │
│  │  └─ MCP Tool Servers                │  │                       │
│  └──────────────────────────────────────┘  │                       │
│                                            │                       │
│  ┌──────────────────────────────────────┐  │                       │
│  │ Admin UI (SvelteKit 5)               │  │                       │
│  │  Dashboard, Users, Models, Providers │  │                       │
│  │  Workers, Downloads, GitLab, Logs    │  │                       │
│  └──────────────────────────────────────┘  │                       │
└────────────────────────────────────────────┼───────────────────────┘
                                             │
                    ┌────────────────────────┼────────────────────────┐
                    │                        │                        │
          ┌─────────┴──────────┐  ┌──────────┴─────────┐  ┌──────────┴─────────┐
          │ GPU Worker 1       │  │ GPU Worker 2       │  │ CPU Worker         │
          │ aigateway-agent    │  │ aigateway-agent    │  │ aigateway-agent    │
          │  ├─ DockerRuntime  │  │  ├─ DockerRuntime  │  │  ├─ DockerRuntime  │
          │  ├─ vLLM (RTX4090) │  │  ├─ SGLang (A100)  │  │  ├─ llama.cpp     │
          │  └─ TEI (embed)    │  │  └─ vLLM (A100)    │  │  └─ TEI CPU       │
          └────────────────────┘  └────────────────────┘  └────────────────────┘
```

---

## Inference Providers

| Provider | GPU | CPU | Quantization | Use Case |
|----------|:---:|:---:|-------------|----------|
| **vLLM** | Yes | — | BF16, FP16, AWQ, GPTQ, FP8 | High-throughput production |
| **SGLang** | Yes | — | BF16, FP16, AWQ, GPTQ, FP8 | Fast inference, tool calling |
| **TGI** | Yes | — | BF16, FP16, GPTQ, BnB | HuggingFace native |
| **TEI** | Yes | Yes | — | Text embeddings & reranking |
| **llama.cpp** | Yes | Yes | GGUF (Q2-Q8, K-quants) | CPU inference, low VRAM |
| **TensorRT-LLM** | Yes | — | FP16, INT8, INT4 | Maximum throughput (NVIDIA) |

---

## Quick Start

### Prerequisites

- **Go 1.25+** (build)
- **Docker** with NVIDIA Container Toolkit (GPU inference)
- **PostgreSQL 15+** (data storage, migrations)
- **Node.js 20+** (WebUI build, optional)

### Option 1: RPM Package (Rocky Linux / RHEL 9)

```bash
curl -fsSL https://gitlab.alexue4.dev/api/v4/projects/146/packages/generic/ollama-openai-proxy/latest/install.sh | sudo bash
sudo systemctl enable --now oop
```

### Option 2: Build from Source

```bash
git clone https://gitlab.alexue4.dev/KelnMaari/ollama-openai-proxy.git
cd ollama-openai-proxy

# Build WebUI
cd web-svelte && npm install && npm run build && cd ..
cp -r web-svelte/build/* internal/web/svelte-build/

# Build server
go build -o bin/aigateway cmd/server/main.go

# Build agent (for remote workers)
go build -o bin/aigateway-agent cmd/agent/main.go

# Run
./bin/aigateway -config configs/dev.yaml
```

### First Run

1. Open [http://localhost:8085/bootstrap](http://localhost:8085/bootstrap) to create admin account
2. Go to **Admin > Models > HuggingFace** — search and load a model
3. Go to **Chat** — select model, start chatting

---

## Agent Mode (Remote Workers)

Distribute inference across multiple machines. The agent binary (`aigateway-agent`) runs on GPU/CPU workers and is controlled by the main server.

### Setup

1. **Main server** — enable workers in `config.yaml`:

```yaml
workers:
  enabled: true
  health_check_interval: "30s"
  tls:
    skip_verify: true  # for dev
```

2. **Admin UI** — go to **Admin > Workers > Generate Config**:
   - Enter worker name, address (`http://gpu-server:9090`), node type (GPU/CPU)
   - Copy generated `agent.yaml` and API key

3. **Worker machine** — install and start agent:

```bash
# Install agent RPM
curl -fsSL https://gitlab.alexue4.dev/api/v4/projects/146/packages/generic/aigateway-agent/latest/install.sh | sudo bash

# Copy generated config
sudo cp agent.yaml /opt/aigateway-agent/configs/agent.yaml

# Start
sudo systemctl enable --now aigateway-agent
```

4. Worker appears as **online** in Admin > Workers with GPU metrics

### Loading Models on Workers

- **Admin > Models** — "Target Node" dropdown selects Local or remote worker
- **Admin > Workers** — "Load Model" button on each worker card
- Models on workers appear in `/v1/models` and Chat UI automatically

### Docker Image Transfer

Push custom Docker images from main server to workers:

```
POST /api/admin/workers/:id/images/push
{"image": "vllm-custom:patched"}
```

Main server runs `docker save` > streams to agent > `docker load`.

### Private Docker Registries

Agent config supports registry authentication:

```yaml
agent:
  docker_registries:
    - registry: "registry.gitlab.com"
      username: "deploy-token"
      password: "glpat-xxxxxxxxxxxx"
```

---

## External Providers

Route requests to external APIs alongside local/remote inference:

| Provider | Streaming | Tool Calling | Format Conversion |
|----------|:---------:|:------------:|:-----------------:|
| OpenAI | Yes | Yes | — |
| Anthropic | Yes | Yes | Anthropic > OpenAI |
| Gemini | Yes | Yes | Gemini > OpenAI |
| DeepSeek | Yes | Yes | — |
| Custom (OpenAI-compat) | Yes | — | — |

Configure in **Admin > Providers** — add endpoint URL and API key.

---

## API Reference

### OpenAI-Compatible Endpoints

```bash
# List models (local + agent + external)
curl http://localhost:8085/v1/models \
  -H "Authorization: Bearer sk-xxx"

# Chat completion (streaming)
curl http://localhost:8085/v1/chat/completions \
  -H "Authorization: Bearer sk-xxx" \
  -d '{"model": "qwen3-5-35b", "messages": [{"role": "user", "content": "Hello"}], "stream": true}'

# Embeddings
curl http://localhost:8085/v1/embeddings \
  -H "Authorization: Bearer sk-xxx" \
  -d '{"model": "nomic-embed", "input": "Hello world"}'
```

### Model Management

```bash
# Load model (local)
curl -X POST http://localhost:8085/api/system/inference/load \
  -d '{"alias": "qwen-7b", "provider": "vllm", "format": "hf", "hf_repo": "Qwen/Qwen2.5-7B-Instruct"}'

# Load model on remote worker
curl -X POST http://localhost:8085/api/system/inference/load \
  -d '{"alias": "llama3", "provider": "vllm", "format": "hf", "hf_repo": "meta-llama/Llama-3-8B", "target_node": "worker-uuid"}'

# Stop model
curl -X POST http://localhost:8085/api/system/inference/stop -d '{"alias": "qwen-7b"}'
```

### Worker Management

```bash
# List workers
curl http://localhost:8085/api/admin/workers

# Register worker
curl -X POST http://localhost:8085/api/admin/workers \
  -d '{"name": "gpu-01", "address": "http://192.168.1.50:9090", "node_type": "gpu"}'

# GPU metrics from worker
curl http://localhost:8085/api/admin/workers/:id/gpu

# Load model on specific worker
curl -X POST http://localhost:8085/api/admin/workers/:id/models/load \
  -d '{"alias": "llama3", "provider": "vllm", "hf_repo": "meta-llama/Llama-3-8B"}'
```

---

## Configuration

### Main Server (`configs/dev.yaml`)

```yaml
server:
  host: "0.0.0.0"
  port: 8085

inference:
  backend: "docker"
  merge_reasoning_content: true  # merge reasoning_content into <think> tags
  docker:
    enabled: true
    hf_cache_dir: "./data/models/hf"
    gguf_cache_dir: "./data/models/gguf"
    max_running_models: 2
    default_provider: "vllm"

workers:
  enabled: true
  health_check_interval: "30s"

database:
  type: "postgresql"
  postgresql:
    host: "localhost"
    port: 5432
    database: "aigateway"

huggingface:
  token: "hf_xxx..."

tools:
  tavily_api_key: "tvly-xxx..."
```

### Agent (`configs/agent.yaml`)

```yaml
agent:
  listen_addr: "0.0.0.0:9090"
  api_key: "agent-xxx..."
  node_name: "gpu-worker-1"
  node_type: "gpu"              # gpu | cpu | mixed
  hf_cache_dir: "/data/models/hf"
  gguf_cache_dir: "/data/models/gguf"
  default_provider: "vllm"
  max_running_models: 2
```

### vLLM-Specific Options

| Option | Description |
|--------|-------------|
| Tensor Parallel | Number of GPUs for tensor parallelism |
| Max Model Len | Maximum context length |
| GPU Utilization | VRAM usage fraction (0-1) |
| Quantization | AWQ, GPTQ, FP8, squeezellm |
| KV Cache Dtype | auto, fp8 |
| Enforce Eager | Disable CUDA graphs |
| Prefix Caching | Reuse KV cache for shared prefixes |
| Chunked Prefill | Process long prompts in chunks |
| Allow Long Context | Allow max_model_len > max_position_embeddings |
| Disable Reasoning | Keep `<think>` tags in content (don't extract to reasoning_content) |
| Auto Tool Choice | Enable autonomous tool calling |
| Tool Call Parser | hermes, mistral, llama3_json, deepseek_v3, qwen25 |

---

## Security

### CI/CD Security Scanning

Two parallel security scanners run on every push:

- **Trivy** (Aqua Security): Filesystem scan of Go + npm dependencies, SARIF output to GitLab Security Dashboard
- **govulncheck** (Go team): Callgraph-aware CVE analysis — only reports vulnerabilities in actually reachable code paths

### Authentication

- JWT with configurable issuer and expiry
- LDAP integration
- OIDC (Keycloak, etc.)
- API Key authentication (Bearer token)

### Agent Communication

- TLS between main server and agents
- API Key authentication on every request
- Per-node keys generated during worker registration

---

## Project Structure

```
cmd/
  server/main.go              # Main HTTP/WebSocket server
  agent/main.go               # Remote inference agent
  tui/main.go                 # Terminal UI for monitoring
internal/
  agent/                      # Agent Mode (remote workers)
  api/handlers/               # HTTP handlers
  api/middleware/              # Auth, rate limiting
  api/router/                 # Route registration & wiring
  auth/                       # JWT, LDAP, OIDC
  config/                     # Configuration system
  inference/                  # Docker inference orchestration
  providers/                  # External provider adapters
  storage/postgresql/         # PostgreSQL + migrations
  gitlab/                     # GitLab MR review integration
  rag/                        # RAG pipeline
  models/                     # Domain models
  metrics/                    # Prometheus metrics
  web/                        # Embedded WebUI
web-svelte/                   # SvelteKit 5 frontend
  src/routes/(protected)/admin/
    models/                   # Model management UI
    workers/                  # Worker management UI
    providers/                # External providers UI
configs/
  dev.yaml                    # Development config
  agent.yaml                  # Agent sample config
packaging/
  rpm/                        # Main server RPM spec + service
  agent/                      # Agent RPM spec + service + install script
```

---

## Troubleshooting

### Common Errors

**OOM (Out of Memory)**
```
ValueError: No available memory for the cache blocks
```
Reduce `gpu_memory_utilization` or `max_model_len`.

**Max Model Len Exceeded**
```
max_model_len (400000) is greater than max_position_embeddings (262144)
```
Set `max_model_len` to model's max or enable "Allow Long Context" checkbox.

**Reasoning Content Not Visible**
```
</think> shown without <think> in client
```
Either enable "Disable Reasoning" per-model, or set `merge_reasoning_content: true` in server config.

**Agent Offline**
```
Worker status: offline
```
Check agent service: `sudo systemctl status aigateway-agent`. Verify network connectivity and API key match.

### Useful Commands

```bash
# GPU status
nvidia-smi

# Kill all inference containers
docker ps -a | grep aigw- | awk '{print $1}' | xargs -r docker rm -f

# Server logs
journalctl -u oop -f

# Agent logs
journalctl -u aigateway-agent -f
```

---

## License

MIT License. See [LICENSE](LICENSE).

---

## Acknowledgments

- [vLLM](https://vllm.ai/) — High-throughput inference
- [SGLang](https://github.com/sgl-project/sglang) — Fast inference with tool support
- [TGI](https://github.com/huggingface/text-generation-inference) — HuggingFace inference server
- [TEI](https://github.com/huggingface/text-embeddings-inference) — Text embeddings
- [llama.cpp](https://github.com/ggerganov/llama.cpp) — GGUF/CPU inference
- [Gin](https://gin-gonic.com/) — HTTP framework
- [SvelteKit](https://svelte.dev/) — Frontend framework
- [Tailwind CSS](https://tailwindcss.com/) — Styling
- [Tavily](https://tavily.com/) — Web search API
- [Qdrant](https://qdrant.tech/) — Vector database
- [Trivy](https://trivy.dev/) — Security scanning
